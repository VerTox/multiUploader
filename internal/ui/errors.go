package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
)

// FriendlyError представляет понятное пользователю сообщение об ошибке
type FriendlyError struct {
	Title   string // Краткое описание проблемы
	Message string // Подробное описание
	Hint    string // Подсказка как исправить
}

// ErrorType представляет категорию ошибки
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeNetwork
	ErrorTypeAuth
	ErrorTypeFile
	ErrorTypeServer
	ErrorTypeValidation
	ErrorTypeCancelled
)

// MakeFriendly конвертирует техническую ошибку в понятное сообщение
func MakeFriendly(err error) *FriendlyError {
	if err == nil {
		return nil
	}

	// Определяем тип ошибки и создаем дружественное сообщение
	errType := classifyError(err)

	switch errType {
	case ErrorTypeNetwork:
		return makeNetworkError(err)
	case ErrorTypeAuth:
		return makeAuthError(err)
	case ErrorTypeFile:
		return makeFileError(err)
	case ErrorTypeServer:
		return makeServerError(err)
	case ErrorTypeValidation:
		return makeValidationError(err)
	case ErrorTypeCancelled:
		return &FriendlyError{
			Title:   "Upload Cancelled",
			Message: "The upload was cancelled by user.",
			Hint:    "",
		}
	default:
		return makeUnknownError(err)
	}
}

// classifyError определяет тип ошибки
func classifyError(err error) ErrorType {
	errMsg := strings.ToLower(err.Error())

	// Context cancellation
	if errors.Is(err, context.Canceled) || strings.Contains(errMsg, "cancelled") || strings.Contains(errMsg, "canceled") {
		return ErrorTypeCancelled
	}

	// Network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return ErrorTypeNetwork
	}

	// DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return ErrorTypeNetwork
	}

	// Connection errors
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return ErrorTypeNetwork
	}

	// Syscall errors (connection refused, etc.)
	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) {
		if syscallErr.Err == syscall.ECONNREFUSED {
			return ErrorTypeNetwork
		}
	}

	// File errors
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return ErrorTypeFile
	}

	// Check error message for common patterns
	if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "dial") || strings.Contains(errMsg, "network") {
		return ErrorTypeNetwork
	}

	if strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "api key") ||
		strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "authentication") {
		return ErrorTypeAuth
	}

	if strings.Contains(errMsg, "file") || strings.Contains(errMsg, "permission denied") ||
		strings.Contains(errMsg, "no such file") {
		return ErrorTypeFile
	}

	if strings.Contains(errMsg, "status") {
		// Проверяем HTTP статусы в тексте ошибки
		if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "403") {
			return ErrorTypeAuth
		}
		if strings.Contains(errMsg, "400") || strings.Contains(errMsg, "413") {
			return ErrorTypeValidation
		}
		if strings.Contains(errMsg, "500") || strings.Contains(errMsg, "502") ||
			strings.Contains(errMsg, "503") || strings.Contains(errMsg, "504") {
			return ErrorTypeServer
		}
		return ErrorTypeServer
	}

	if strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "too large") {
		return ErrorTypeValidation
	}

	return ErrorTypeUnknown
}

// makeNetworkError создает дружественное сообщение для сетевых ошибок
func makeNetworkError(err error) *FriendlyError {
	errMsg := strings.ToLower(err.Error())

	// Timeout
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &FriendlyError{
			Title:   "Connection Timeout",
			Message: "The connection to the server timed out.",
			Hint:    "Please check your internet connection and try again. If the problem persists, the server may be experiencing issues.",
		}
	}

	// DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &FriendlyError{
			Title:   "DNS Lookup Failed",
			Message: "Could not resolve the server address.",
			Hint:    "Please check your internet connection and DNS settings. Try again in a few moments.",
		}
	}

	// Connection refused
	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "econnrefused") {
		return &FriendlyError{
			Title:   "Connection Refused",
			Message: "The server refused the connection.",
			Hint:    "The service may be temporarily unavailable. Please try again later.",
		}
	}

	// Generic network error
	return &FriendlyError{
		Title:   "Network Error",
		Message: "A network error occurred while communicating with the server.",
		Hint:    "Please check your internet connection and try again.",
	}
}

// makeAuthError создает дружественное сообщение для ошибок авторизации
func makeAuthError(err error) *FriendlyError {
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "unauthorized") {
		return &FriendlyError{
			Title:   "Invalid API Key",
			Message: "The API key you provided is not valid.",
			Hint:    "Please check your API key in Settings and make sure it's correct.",
		}
	}

	if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "forbidden") {
		return &FriendlyError{
			Title:   "Access Denied",
			Message: "Your API key does not have permission to perform this operation.",
			Hint:    "Please check that your API key has the necessary permissions, or contact the service provider.",
		}
	}

	return &FriendlyError{
		Title:   "Authentication Error",
		Message: "There was a problem authenticating with the service.",
		Hint:    "Please check your API key in Settings.",
	}
}

// makeFileError создает дружественное сообщение для файловых ошибок
func makeFileError(err error) *FriendlyError {
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "no such file") || strings.Contains(errMsg, "not found") {
		return &FriendlyError{
			Title:   "File Not Found",
			Message: "The selected file could not be found.",
			Hint:    "The file may have been moved or deleted. Please select the file again.",
		}
	}

	if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "access is denied") {
		return &FriendlyError{
			Title:   "Permission Denied",
			Message: "You don't have permission to access this file.",
			Hint:    "Please check the file permissions or try selecting a different file.",
		}
	}

	if errors.Is(err, io.EOF) || strings.Contains(errMsg, "eof") {
		return &FriendlyError{
			Title:   "File Read Error",
			Message: "The file could not be read completely.",
			Hint:    "The file may be corrupted or locked by another program. Please try again.",
		}
	}

	return &FriendlyError{
		Title:   "File Error",
		Message: "There was a problem reading the file.",
		Hint:    "Please make sure the file is accessible and not being used by another program.",
	}
}

// makeServerError создает дружественное сообщение для серверных ошибок
func makeServerError(err error) *FriendlyError {
	errMsg := strings.ToLower(err.Error())

	// Извлекаем HTTP статус код если есть
	statusCode := extractStatusCode(errMsg)

	switch statusCode {
	case http.StatusBadRequest: // 400
		return &FriendlyError{
			Title:   "Invalid Request",
			Message: "The server could not process your request.",
			Hint:    "Please try selecting the file again. If the problem persists, the file may not be supported.",
		}

	case http.StatusNotFound: // 404
		return &FriendlyError{
			Title:   "Service Not Found",
			Message: "The upload service endpoint could not be found.",
			Hint:    "The service may be temporarily unavailable or under maintenance. Please try again later.",
		}

	case http.StatusRequestEntityTooLarge: // 413
		return &FriendlyError{
			Title:   "File Too Large",
			Message: "The file you're trying to upload is too large for this provider.",
			Hint:    "Please try a smaller file or use a different provider that supports larger files.",
		}

	case http.StatusTooManyRequests: // 429
		return &FriendlyError{
			Title:   "Rate Limit Exceeded",
			Message: "You've made too many requests in a short period.",
			Hint:    "Please wait a few minutes before trying again.",
		}

	case http.StatusInternalServerError: // 500
		return &FriendlyError{
			Title:   "Server Error",
			Message: "The server encountered an internal error.",
			Hint:    "This is a temporary server issue. Please try again in a few minutes.",
		}

	case http.StatusBadGateway: // 502
		return &FriendlyError{
			Title:   "Bad Gateway",
			Message: "The server received an invalid response from an upstream server.",
			Hint:    "This is a temporary server issue. Please try again in a few minutes.",
		}

	case http.StatusServiceUnavailable: // 503
		return &FriendlyError{
			Title:   "Service Unavailable",
			Message: "The service is temporarily unavailable.",
			Hint:    "The server may be under maintenance. Please try again later.",
		}

	case http.StatusGatewayTimeout: // 504
		return &FriendlyError{
			Title:   "Gateway Timeout",
			Message: "The server did not receive a timely response.",
			Hint:    "The service may be experiencing high load. Please try again in a few minutes.",
		}

	default:
		// Generic server error
		if statusCode >= 500 {
			return &FriendlyError{
				Title:   "Server Error",
				Message: fmt.Sprintf("The server returned an error (HTTP %d).", statusCode),
				Hint:    "This is a temporary issue. Please try again later.",
			}
		}

		// Check for provider-specific error messages
		if strings.Contains(errMsg, "server returned error") {
			// Extract error message after "server returned error:"
			parts := strings.Split(err.Error(), ":")
			if len(parts) >= 2 {
				serverMsg := strings.TrimSpace(parts[len(parts)-1])
				return &FriendlyError{
					Title:   "Upload Failed",
					Message: fmt.Sprintf("The server reported an error: %s", serverMsg),
					Hint:    "Please check your file and try again.",
				}
			}
		}

		return &FriendlyError{
			Title:   "Server Error",
			Message: "The server encountered an error while processing your request.",
			Hint:    "Please try again. If the problem persists, try a different provider.",
		}
	}
}

// makeValidationError создает дружественное сообщение для ошибок валидации
func makeValidationError(err error) *FriendlyError {
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "too large") || strings.Contains(errMsg, "413") {
		return &FriendlyError{
			Title:   "File Too Large",
			Message: "The file exceeds the maximum size allowed by this provider.",
			Hint:    "Please try a smaller file or use a different provider.",
		}
	}

	if strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "400") {
		return &FriendlyError{
			Title:   "Invalid File",
			Message: "The file or request parameters are not valid.",
			Hint:    "Please make sure you selected a valid file and try again.",
		}
	}

	return &FriendlyError{
		Title:   "Validation Error",
		Message: "The file or request could not be validated.",
		Hint:    "Please check your file and try again.",
	}
}

// makeUnknownError создает дружественное сообщение для неизвестных ошибок
func makeUnknownError(err error) *FriendlyError {
	return &FriendlyError{
		Title:   "Unexpected Error",
		Message: "An unexpected error occurred.",
		Hint:    fmt.Sprintf("Technical details: %s", err.Error()),
	}
}

// extractStatusCode извлекает HTTP статус код из текста ошибки
func extractStatusCode(errMsg string) int {
	// Ищем паттерны типа "status 404", "status code 500", etc.
	statusPatterns := []string{
		"status ", "status code ", "http status ", "code ",
	}

	for _, pattern := range statusPatterns {
		if idx := strings.Index(errMsg, pattern); idx != -1 {
			// Извлекаем число после паттерна
			start := idx + len(pattern)
			if start < len(errMsg) {
				// Читаем цифры
				numStr := ""
				for i := start; i < len(errMsg) && len(numStr) < 3; i++ {
					if errMsg[i] >= '0' && errMsg[i] <= '9' {
						numStr += string(errMsg[i])
					} else if len(numStr) > 0 {
						break
					}
				}

				// Конвертируем в число
				var code int
				fmt.Sscanf(numStr, "%d", &code)
				if code >= 100 && code < 600 {
					return code
				}
			}
		}
	}

	return 0
}

// FormatErrorMessage форматирует FriendlyError в строку для отображения
func FormatErrorMessage(fe *FriendlyError) string {
	if fe == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fe.Title)
	sb.WriteString("\n\n")
	sb.WriteString(fe.Message)

	if fe.Hint != "" {
		sb.WriteString("\n\n")
		sb.WriteString("💡 Tip: ")
		sb.WriteString(fe.Hint)
	}

	return sb.String()
}
