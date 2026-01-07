package providers

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// TestByteCounter проверяет работу счётчика байтов
func TestByteCounter(t *testing.T) {
	t.Run("Initial state", func(t *testing.T) {
		counter := &ByteCounter{}
		if counter.N() != 0 {
			t.Errorf("Initial N() = %d, want 0", counter.N())
		}
	})

	t.Run("Add bytes", func(t *testing.T) {
		counter := &ByteCounter{}
		counter.Add(100)

		if counter.N() != 100 {
			t.Errorf("After Add(100), N() = %d, want 100", counter.N())
		}

		counter.Add(50)
		if counter.N() != 150 {
			t.Errorf("After Add(50), N() = %d, want 150", counter.N())
		}
	})

	t.Run("Thread safety", func(t *testing.T) {
		counter := &ByteCounter{}

		// Параллельно добавляем байты
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					counter.Add(1)
				}
				done <- true
			}()
		}

		// Ждём завершения
		for i := 0; i < 10; i++ {
			<-done
		}

		// Должно быть 10 * 100 = 1000
		if counter.N() != 1000 {
			t.Errorf("After concurrent Add(), N() = %d, want 1000", counter.N())
		}
	})
}

// TestCountingReader проверяет читатель с подсчётом байтов
func TestCountingReader(t *testing.T) {
	t.Run("Read with callback", func(t *testing.T) {
		data := "Hello, World!"
		reader := strings.NewReader(data)

		var totalBytes int64
		countingReader := CountingReader{
			r: reader,
			cb: func(n int64) {
				totalBytes += n
			},
		}

		// Читаем все данные
		buf := make([]byte, 1024)
		n, err := countingReader.Read(buf)
		if err != nil && err != io.EOF {
			t.Fatalf("Read() error = %v", err)
		}

		if n != len(data) {
			t.Errorf("Read() = %d bytes, want %d", n, len(data))
		}

		if totalBytes != int64(len(data)) {
			t.Errorf("Callback received %d bytes, want %d", totalBytes, len(data))
		}

		if string(buf[:n]) != data {
			t.Errorf("Read() data = %s, want %s", string(buf[:n]), data)
		}
	})

	t.Run("Read without callback", func(t *testing.T) {
		data := "Test data"
		reader := strings.NewReader(data)

		countingReader := CountingReader{
			r:  reader,
			cb: nil, // Нет коллбека
		}

		buf := make([]byte, 1024)
		n, err := countingReader.Read(buf)
		if err != nil && err != io.EOF {
			t.Fatalf("Read() error = %v", err)
		}

		if n != len(data) {
			t.Errorf("Read() = %d bytes, want %d", n, len(data))
		}
	})

	t.Run("Multiple reads", func(t *testing.T) {
		data := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		reader := strings.NewReader(data)

		var totalBytes int64
		countingReader := CountingReader{
			r: reader,
			cb: func(n int64) {
				totalBytes += n
			},
		}

		// Читаем по 5 байт за раз
		buf := make([]byte, 5)
		var totalRead int
		for {
			n, err := countingReader.Read(buf)
			totalRead += n
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
		}

		if totalRead != len(data) {
			t.Errorf("Total read = %d bytes, want %d", totalRead, len(data))
		}

		if totalBytes != int64(len(data)) {
			t.Errorf("Callback received %d bytes, want %d", totalBytes, len(data))
		}
	})
}

// TestHumanBytes проверяет форматирование (косвенно через FormatSize из progress.go)
func TestHumanBytes(t *testing.T) {
	// Проверяем что humanBytes работает правильно через FormatSize
	tests := []struct {
		bytes    int64
		contains string // Что должно содержаться в результате
	}{
		{0, "B"},
		{100, "B"},
		{1024, "KB"},
		{1024 * 1024, "MB"},
		{1024 * 1024 * 1024, "GB"},
	}

	for _, tt := range tests {
		result := FormatSize(tt.bytes)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("FormatSize(%d) = %s, should contain %s", tt.bytes, result, tt.contains)
		}
	}
}

// TestMultipartWriter проверяет wrapper для multipart.Writer
func TestMultipartWriter(t *testing.T) {
	t.Run("Create and write field", func(t *testing.T) {
		var buf bytes.Buffer
		mw := NewMultipartWriter(&buf)

		err := mw.WriteField("name", "test")
		if err != nil {
			t.Fatalf("WriteField() error = %v", err)
		}

		err = mw.Close()
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		// Проверяем что данные записаны
		if buf.Len() == 0 {
			t.Error("Buffer is empty after WriteField()")
		}

		// Проверяем что в буфере есть наше поле
		content := buf.String()
		if !strings.Contains(content, "name") || !strings.Contains(content, "test") {
			t.Errorf("Buffer content doesn't contain expected field: %s", content)
		}
	})

	t.Run("FormDataContentType", func(t *testing.T) {
		var buf bytes.Buffer
		mw := NewMultipartWriter(&buf)

		contentType := mw.FormDataContentType()
		if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
			t.Errorf("FormDataContentType() = %s, should start with 'multipart/form-data; boundary='", contentType)
		}
	})

	t.Run("WriteFile", func(t *testing.T) {
		var buf bytes.Buffer
		mw := NewMultipartWriter(&buf)

		fileData := []byte("file content")
		err := mw.WriteFile("upload", "test.txt", fileData)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		err = mw.Close()
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		// Проверяем что файл записан
		content := buf.String()
		if !strings.Contains(content, "upload") || !strings.Contains(content, "test.txt") {
			t.Errorf("Buffer doesn't contain file metadata: %s", content)
		}
		if !strings.Contains(content, "file content") {
			t.Errorf("Buffer doesn't contain file data: %s", content)
		}
	})
}
