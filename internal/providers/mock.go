package providers

import (
	"context"
	"fmt"
	"io"
	"time"
)

// MockProvider симулирует работу реального провайдера для тестирования UI
type MockProvider struct {
	name                   string
	uploadSpeedBytesPerSec int64
	simulateError          bool
}

// NewMockProvider создает новый мок провайдер
// uploadSpeedMBPerSec - скорость загрузки в МБ/сек (например, 2 для симуляции медленной загрузки)
func NewMockProvider(name string, uploadSpeedMBPerSec int) *MockProvider {
	return &MockProvider{
		name:                   name,
		uploadSpeedBytesPerSec: int64(uploadSpeedMBPerSec) * 1024 * 1024,
		simulateError:          false,
	}
}

// NewMockProviderWithError создает мок провайдер который симулирует ошибку
func NewMockProviderWithError(name string) *MockProvider {
	return &MockProvider{
		name:                   name,
		uploadSpeedBytesPerSec: 2 * 1024 * 1024, // 2 MB/s
		simulateError:          true,
	}
}

// Name возвращает название провайдера
func (m *MockProvider) Name() string {
	return m.name
}

// Upload симулирует загрузку файла
func (m *MockProvider) Upload(ctx context.Context, file io.ReadSeeker, filename string, fileSize int64, progress chan<- UploadProgress) (*UploadResult, error) {
	// Создаем калькулятор скорости
	speedCalc := NewSpeedCalculator()

	// Начальное время
	startTime := time.Now()
	var totalUploaded int64 = 0

	// Вычисляем сколько байт загружать за каждый интервал обновления
	bytesPerUpdate := int64(float64(m.uploadSpeedBytesPerSec) * ProgressUpdateInterval.Seconds())
	if bytesPerUpdate <= 0 {
		bytesPerUpdate = 1024 // минимум 1 KB за обновление
	}

	ticker := time.NewTicker(ProgressUpdateInterval)
	defer ticker.Stop()

	for totalUploaded < fileSize {
		select {
		case <-ctx.Done():
			// Загрузка отменена
			return nil, fmt.Errorf("upload cancelled")
		case <-ticker.C:
			// Вычисляем сколько должно быть загружено к текущему моменту
			elapsed := time.Since(startTime).Seconds()
			expectedUploaded := int64(float64(m.uploadSpeedBytesPerSec) * elapsed)

			// Не превышаем размер файла
			if expectedUploaded > fileSize {
				expectedUploaded = fileSize
			}

			totalUploaded = expectedUploaded

			// Вычисляем скорость
			speed := speedCalc.Update(totalUploaded)

			// Вычисляем процент
			percentage := int((float64(totalUploaded) / float64(fileSize)) * 100)
			if percentage > 100 {
				percentage = 100
			}

			// Отправляем прогресс
			progress <- UploadProgress{
				BytesUploaded: totalUploaded,
				TotalBytes:    fileSize,
				Speed:         speed,
				Percentage:    percentage,
			}

			// Симулируем ошибку на 50%
			if m.simulateError && percentage >= 50 {
				return nil, fmt.Errorf("simulated upload error at 50%%")
			}
		}
	}

	// Отправляем финальный прогресс (100%)
	progress <- UploadProgress{
		BytesUploaded: fileSize,
		TotalBytes:    fileSize,
		Speed:         speedCalc.Update(fileSize),
		Percentage:    100,
	}

	// Возвращаем результат
	result := &UploadResult{
		URL:         fmt.Sprintf("https://mock.provider/%s/%s", m.name, filename),
		DownloadURL: fmt.Sprintf("https://mock.provider/download/%s", filename),
		DeleteURL:   fmt.Sprintf("https://mock.provider/delete/%s", filename),
		FileID:      fmt.Sprintf("mock-%d", time.Now().Unix()),
		Message:     fmt.Sprintf("File uploaded successfully to %s (mock)", m.name),
	}

	return result, nil
}

// RequiresAuth возвращает true для тестирования настроек API ключа
func (m *MockProvider) RequiresAuth() bool {
	return true
}

// ValidateAPIKey симулирует валидацию API ключа
func (m *MockProvider) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	if len(apiKey) < 10 {
		return fmt.Errorf("API key is too short (minimum 10 characters)")
	}
	return nil
}
