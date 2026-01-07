package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Client обертка над http.Client с retry логикой
type Client struct {
	httpClient *http.Client
	maxRetries int
	maxElapsed time.Duration
}

// ClientConfig конфигурация для HTTP клиента
type ClientConfig struct {
	Timeout    time.Duration // Таймаут для запроса
	MaxRetries int           // Максимальное количество попыток
	MaxElapsed time.Duration // Максимальное время на все попытки
}

// DefaultConfig возвращает стандартную конфигурацию
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		MaxElapsed: 5 * time.Minute,
	}
}

// LongLivedConfig конфигурация для длительных операций (uploads)
func LongLivedConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:    10 * time.Minute,
		MaxRetries: 3,
		MaxElapsed: 30 * time.Minute,
	}
}

// NewClient создает новый HTTP клиент с retry логикой
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	// Настраиваем Transport для connection pooling
	transport := &http.Transport{
		// Connection pooling settings
		MaxIdleConns:        100,              // Максимум idle connections
		MaxIdleConnsPerHost: 10,               // Максимум idle connections на хост
		IdleConnTimeout:     90 * time.Second, // Время жизни idle connection
		// Таймауты для установки соединения
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// Таймауты для TLS
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   config.Timeout,
		},
		maxRetries: config.MaxRetries,
		maxElapsed: config.MaxElapsed,
	}
}

// Do выполняет HTTP запрос с retry логикой
// Retry применяется только для идемпотентных методов (GET, PUT) и временных ошибок
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// Проверяем, нужен ли retry для этого метода
	if !isIdempotent(req.Method) {
		// Для неидемпотентных методов (POST, PATCH, DELETE) не делаем retry
		return c.httpClient.Do(req)
	}

	// Создаем exponential backoff
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = c.maxElapsed
	b.InitialInterval = 500 * time.Millisecond
	b.MaxInterval = 30 * time.Second
	b.Multiplier = 2.0

	// Ограничиваем количество попыток
	backoffWithRetry := backoff.WithMaxRetries(b, uint64(c.maxRetries))

	var resp *http.Response
	var lastErr error

	// Retry operation
	operation := func() error {
		// Проверяем контекст перед каждой попыткой
		if req.Context().Err() != nil {
			return backoff.Permanent(req.Context().Err())
		}

		// Клонируем запрос для безопасности (body может быть прочитан только один раз)
		reqClone := cloneRequest(req)

		r, err := c.httpClient.Do(reqClone)
		if err != nil {
			// Проверяем, является ли ошибка временной
			if isTemporaryError(err) {
				lastErr = err
				return err // Retry
			}
			// Постоянная ошибка - не retry
			return backoff.Permanent(err)
		}

		// Проверяем статус код
		if isRetriableStatusCode(r.StatusCode) {
			// Читаем и закрываем body для переиспользования connection
			io.Copy(io.Discard, r.Body)
			r.Body.Close()
			lastErr = fmt.Errorf("retriable status code: %d", r.StatusCode)
			return lastErr // Retry
		}

		resp = r
		return nil
	}

	err := backoff.Retry(operation, backoffWithRetry)
	if err != nil {
		if lastErr != nil {
			return nil, fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
		}
		return nil, err
	}

	return resp, nil
}

// isIdempotent проверяет, является ли HTTP метод идемпотентным
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// isTemporaryError проверяет, является ли ошибка временной (стоит retry)
func isTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation - не retry
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Network timeout
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Connection refused
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Syscall errors (connection refused, etc.)
	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) {
		if syscallErr.Err == syscall.ECONNREFUSED || syscallErr.Err == syscall.ECONNRESET {
			return true
		}
	}

	// EOF errors (connection closed unexpectedly)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	return false
}

// isRetriableStatusCode проверяет, стоит ли делать retry для данного HTTP статуса
func isRetriableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, // 408
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// cloneRequest создает копию http.Request для retry
// ВАЖНО: body может быть прочитан только один раз, поэтому клонируем
func cloneRequest(req *http.Request) *http.Request {
	// Клонируем запрос
	reqClone := req.Clone(req.Context())

	// Body нужно обработать особым образом
	// Если есть GetBody, используем его (это безопасно для retry)
	if req.Body != nil && req.GetBody != nil {
		body, err := req.GetBody()
		if err == nil {
			reqClone.Body = body
		}
	}

	return reqClone
}
