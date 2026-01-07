package providers

import (
	"fmt"
	"time"
)

// UploadProgress содержит информацию о прогрессе загрузки
type UploadProgress struct {
	// BytesUploaded количество загруженных байт
	BytesUploaded int64

	// TotalBytes общий размер файла в байтах
	TotalBytes int64

	// Speed текущая скорость загрузки в байтах/сек
	Speed float64

	// Percentage процент выполнения (0-100)
	Percentage int
}

// SpeedCalculator отслеживает и вычисляет скорость загрузки
type SpeedCalculator struct {
	startTime         time.Time
	lastUpdateTime    time.Time
	lastBytesUploaded int64
	smoothingWindow   []float64
	maxWindowSize     int
}

// NewSpeedCalculator создает новый калькулятор скорости
func NewSpeedCalculator() *SpeedCalculator {
	now := time.Now()
	return &SpeedCalculator{
		startTime:       now,
		lastUpdateTime:  now,
		smoothingWindow: make([]float64, 0, 5),
		maxWindowSize:   5,
	}
}

// Update обновляет информацию о загруженных байтах и возвращает сглаженную скорость
func (s *SpeedCalculator) Update(bytesUploaded int64) float64 {
	now := time.Now()
	duration := now.Sub(s.lastUpdateTime).Seconds()

	if duration > 0 {
		bytesDelta := bytesUploaded - s.lastBytesUploaded
		currentSpeed := float64(bytesDelta) / duration

		// Добавляем в окно сглаживания
		s.smoothingWindow = append(s.smoothingWindow, currentSpeed)
		if len(s.smoothingWindow) > s.maxWindowSize {
			s.smoothingWindow = s.smoothingWindow[1:]
		}

		// Усредняем скорость
		avgSpeed := 0.0
		for _, speed := range s.smoothingWindow {
			avgSpeed += speed
		}
		avgSpeed /= float64(len(s.smoothingWindow))

		s.lastUpdateTime = now
		s.lastBytesUploaded = bytesUploaded

		return avgSpeed
	}

	return 0
}

// Reset сбрасывает калькулятор
func (s *SpeedCalculator) Reset() {
	now := time.Now()
	s.startTime = now
	s.lastUpdateTime = now
	s.lastBytesUploaded = 0
	s.smoothingWindow = s.smoothingWindow[:0]
}

// FormatSpeed форматирует скорость для отображения
func FormatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	} else if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	} else {
		return fmt.Sprintf("%.2f MB/s", bytesPerSec/(1024*1024))
	}
}

// FormatSize форматирует размер в байтах для отображения
func FormatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
	}
}

// CalculateETA вычисляет оставшееся время на основе оставшихся байт и скорости
func CalculateETA(bytesRemaining int64, speed float64) string {
	if speed <= 0 {
		return "calculating..."
	}

	seconds := float64(bytesRemaining) / speed
	duration := time.Duration(seconds) * time.Second

	if duration < time.Minute {
		return fmt.Sprintf("~%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		secs := int(duration.Seconds()) % 60
		return fmt.Sprintf("~%dm %ds", minutes, secs)
	} else {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		return fmt.Sprintf("~%dh %dm", hours, minutes)
	}
}
