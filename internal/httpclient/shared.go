package httpclient

import "sync"

var (
	// Глобальные shared клиенты (создаются один раз, используются многократно)
	defaultClient     *Client
	longLivedClient   *Client
	defaultClientOnce sync.Once
	longLivedOnce     sync.Once
)

// Default возвращает shared HTTP клиент для обычных запросов
// Timeout: 30 секунд, MaxRetries: 3, MaxElapsed: 5 минут
func Default() *Client {
	defaultClientOnce.Do(func() {
		defaultClient = NewClient(DefaultConfig())
	})
	return defaultClient
}

// LongLived возвращает shared HTTP клиент для длительных операций (uploads)
// Timeout: 10 минут, MaxRetries: 3, MaxElapsed: 30 минут
func LongLived() *Client {
	longLivedOnce.Do(func() {
		longLivedClient = NewClient(LongLivedConfig())
	})
	return longLivedClient
}
