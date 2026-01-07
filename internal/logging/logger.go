package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const (
	maxLogSize     = 5 * 1024 * 1024 // 5 MB
	logFileName    = "app.log"
	logFileNameOld = "app.old.log"
)

var (
	logger   *slog.Logger
	logFile  *os.File
	logDir   string
	logMutex sync.Mutex
	initOnce sync.Once
)

// Init инициализирует логгер (вызывается один раз при старте приложения)
func Init() error {
	var initErr error
	initOnce.Do(func() {
		// Получаем кроссплатформенный путь для логов
		dir, err := getLogDir()
		if err != nil {
			initErr = fmt.Errorf("failed to get log directory: %w", err)
			return
		}
		logDir = dir

		// Создаем директорию если не существует
		if err := os.MkdirAll(logDir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create log directory: %w", err)
			return
		}

		// Открываем лог файл
		logPath := filepath.Join(logDir, logFileName)
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			initErr = fmt.Errorf("failed to open log file: %w", err)
			return
		}
		logFile = file

		// Создаем slog логгер (только ERROR уровень)
		handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level:     slog.LevelError,
			AddSource: true, // Добавляем информацию о месте вызова
		})
		logger = slog.New(handler)
	})
	return initErr
}

// getLogDir возвращает кроссплатформенный путь для логов
func getLogDir() (string, error) {
	var baseDir string
	var err error

	switch runtime.GOOS {
	case "darwin": // macOS
		// ~/Library/Logs/multiUploader
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(homeDir, "Library", "Logs", "multiUploader")

	case "windows":
		// %LOCALAPPDATA%\multiUploader\logs
		baseDir, err = os.UserCacheDir() // Returns %LOCALAPPDATA% on Windows
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(baseDir, "multiUploader", "logs")

	default: // Linux and others
		// ~/.local/share/multiUploader/logs
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(homeDir, ".local", "share", "multiUploader", "logs")
	}

	return baseDir, nil
}

// Error логирует ошибку с контекстом
func Error(msg string, args ...any) {
	if logger == nil {
		return
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	// Проверяем размер файла перед записью
	checkAndRotate()

	logger.Error(msg, args...)
}

// ErrorWithError логирует ошибку с объектом error
func ErrorWithError(msg string, err error, args ...any) {
	if logger == nil {
		return
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	checkAndRotate()

	// Добавляем error к аргументам
	allArgs := append([]any{"error", err.Error()}, args...)
	logger.Error(msg, allArgs...)
}

// checkAndRotate проверяет размер файла и делает ротацию если нужно
// ВАЖНО: должен вызываться с залоченным logMutex!
func checkAndRotate() {
	if logFile == nil {
		return
	}

	// Получаем информацию о файле
	info, err := logFile.Stat()
	if err != nil {
		return
	}

	// Если файл меньше лимита, ничего не делаем
	if info.Size() < maxLogSize {
		return
	}

	// Ротация: закрываем текущий файл
	logFile.Close()

	// Удаляем старый backup если существует
	oldPath := filepath.Join(logDir, logFileNameOld)
	os.Remove(oldPath) // Игнорируем ошибку если файл не существует

	// Переименовываем текущий файл в backup
	currentPath := filepath.Join(logDir, logFileName)
	if err := os.Rename(currentPath, oldPath); err != nil {
		// Если не получилось переименовать, просто удаляем
		os.Remove(currentPath)
	}

	// Создаем новый файл
	file, err := os.OpenFile(currentPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}

	logFile = file

	// Обновляем handler
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level:     slog.LevelError,
		AddSource: true,
	})
	logger = slog.New(handler)
}

// GetLogDir возвращает путь к директории с логами
func GetLogDir() string {
	if logDir == "" {
		// Если еще не инициализировано, получаем путь
		dir, err := getLogDir()
		if err != nil {
			return ""
		}
		return dir
	}
	return logDir
}

// Close закрывает лог файл (вызывается при выходе из приложения)
func Close() error {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logFile != nil {
		return logFile.Close()
	}

	return nil
}

// GetWriter возвращает io.Writer для использования в других местах
// (например, для перенаправления stderr)
func GetWriter() io.Writer {
	if logFile == nil {
		return io.Discard
	}
	return logFile
}
