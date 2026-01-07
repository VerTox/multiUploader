package providers

import (
	"context"
	"io"
)

// Provider interface для всех провайдеров файлового хостинга
type Provider interface {
	// Name возвращает название провайдера
	Name() string

	// Upload загружает файл на хостинг
	// ctx - контекст для отмены операции
	// file - ReadSeeker для чтения файла с поддержкой перемещения
	// filename - имя файла
	// fileSize - размер файла в байтах
	// progress - канал для отправки информации о прогрессе
	Upload(ctx context.Context, file io.ReadSeeker, filename string, fileSize int64, progress chan<- UploadProgress) (*UploadResult, error)

	// RequiresAuth возвращает true, если провайдер требует API ключ
	RequiresAuth() bool

	// ValidateAPIKey проверяет корректность API ключа
	ValidateAPIKey(apiKey string) error
}

// UploadResult содержит результат загрузки файла
type UploadResult struct {
	// URL для просмотра файла
	URL string

	// DownloadURL прямая ссылка для скачивания
	DownloadURL string

	// DeleteURL ссылка для удаления файла (если доступна)
	DeleteURL string

	// FileID уникальный идентификатор файла
	FileID string

	// Message дополнительное сообщение от провайдера
	Message string
}
