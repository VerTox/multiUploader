package providers

import (
	"testing"
	"time"
)

// TestFormatSize проверяет форматирование размера файла
func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Zero bytes", 0, "0 B"},
		{"Bytes", 500, "500 B"},
		{"Kilobytes", 1024, "1.0 KB"},
		{"Megabytes", 1024 * 1024, "1.00 MB"},
		{"Gigabytes", 5 * 1024 * 1024 * 1024, "5.00 GB"},
		{"Mixed KB", 1536, "1.5 KB"},               // 1.5 KB
		{"Mixed MB", 10 * 1024 * 1024, "10.00 MB"}, // 10 MB
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

// TestFormatSpeed проверяет форматирование скорости
func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		name     string
		speed    float64
		expected string
	}{
		{"Zero speed", 0, "0 B/s"},
		{"Bytes per second", 500, "500 B/s"},
		{"Kilobytes per second", 1024, "1.0 KB/s"},
		{"Megabytes per second", 1024 * 1024, "1.00 MB/s"},
		{"Fast speed", 10 * 1024 * 1024, "10.00 MB/s"},
		{"Very slow", 100, "100 B/s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSpeed(tt.speed)
			if result != tt.expected {
				t.Errorf("FormatSpeed(%f) = %s, want %s", tt.speed, result, tt.expected)
			}
		})
	}
}

// TestCalculateETA проверяет расчёт оставшегося времени
func TestCalculateETA(t *testing.T) {
	tests := []struct {
		name      string
		remaining int64
		speed     float64
		expected  string
	}{
		{"Zero speed", 1024, 0, "calculating..."},
		{"Negative speed", 1024, -1, "calculating..."},
		{"Seconds only", 1024, 1024, "~1s"},                       // 1 секунда
		{"Minutes and seconds", 1024 * 90, 1024, "~1m 30s"},       // 1.5 минуты = 1m 30s
		{"Hours", 1024 * 3600 * 2, 1024, "~2h 0m"},                // 2 часа
		{"Large file", 1024 * 1024 * 100, 1024 * 1024, "~1m 40s"}, // 100 секунд
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateETA(tt.remaining, tt.speed)
			if result != tt.expected {
				t.Errorf("CalculateETA(%d, %f) = %s, want %s", tt.remaining, tt.speed, result, tt.expected)
			}
		})
	}
}

// TestSpeedCalculator проверяет работу калькулятора скорости
func TestSpeedCalculator(t *testing.T) {
	t.Run("Initial state", func(t *testing.T) {
		calc := NewSpeedCalculator()
		if calc == nil {
			t.Fatal("NewSpeedCalculator() returned nil")
		}

		// Первое обновление должно вернуть 0 (нет данных для расчёта)
		speed := calc.Update(0)
		if speed != 0 {
			t.Errorf("Initial Update(0) = %f, want 0", speed)
		}
	})

	t.Run("Speed calculation", func(t *testing.T) {
		calc := NewSpeedCalculator()

		// Подождём немного и обновим прогресс
		time.Sleep(100 * time.Millisecond)
		speed := calc.Update(1024) // 1KB за 100ms

		// Скорость должна быть больше 0
		if speed <= 0 {
			t.Errorf("Update(1024) after delay = %f, want > 0", speed)
		}

		// Примерная скорость: 1024 bytes / 0.1 sec = ~10240 bytes/sec
		// Допускаем погрешность из-за таймингов
		if speed < 5000 || speed > 20000 {
			t.Logf("Warning: speed %f B/s seems unusual (expected ~10240 B/s)", speed)
		}
	})

	t.Run("Speed smoothing", func(t *testing.T) {
		calc := NewSpeedCalculator()

		// Несколько обновлений для проверки сглаживания
		speeds := make([]float64, 0)
		for i := 0; i < 5; i++ {
			time.Sleep(50 * time.Millisecond)
			speed := calc.Update(int64((i + 1) * 512))
			speeds = append(speeds, speed)
		}

		// После нескольких обновлений должна быть рассчитана скорость
		lastSpeed := speeds[len(speeds)-1]
		if lastSpeed <= 0 {
			t.Errorf("Speed after multiple updates = %f, want > 0", lastSpeed)
		}
	})

	t.Run("Reset", func(t *testing.T) {
		calc := NewSpeedCalculator()

		// Обновляем
		time.Sleep(100 * time.Millisecond)
		calc.Update(1024)

		// Сбрасываем
		calc.Reset()

		// После сброса первое обновление должно вернуть 0
		speed := calc.Update(0)
		if speed != 0 {
			t.Errorf("Update(0) after Reset() = %f, want 0", speed)
		}
	})
}

// TestUploadProgress проверяет структуру UploadProgress
func TestUploadProgress(t *testing.T) {
	progress := UploadProgress{
		BytesUploaded: 1024,
		TotalBytes:    2048,
		Speed:         1024,
		Percentage:    50,
	}

	if progress.BytesUploaded != 1024 {
		t.Errorf("BytesUploaded = %d, want 1024", progress.BytesUploaded)
	}

	if progress.TotalBytes != 2048 {
		t.Errorf("TotalBytes = %d, want 2048", progress.TotalBytes)
	}

	if progress.Speed != 1024 {
		t.Errorf("Speed = %f, want 1024", progress.Speed)
	}

	if progress.Percentage != 50 {
		t.Errorf("Percentage = %d, want 50", progress.Percentage)
	}
}
