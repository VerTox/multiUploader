package providers

import (
	"fmt"
	"io"
	"mime/multipart"
	"sync/atomic"
)

type ByteCounter struct {
	n atomic.Int64
}

func (c *ByteCounter) Add(x int64) { c.n.Add(x) }
func (c *ByteCounter) N() int64    { return c.n.Load() }

type CountingReader struct {
	r  io.Reader
	cb func(int64)
}

func (cr CountingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	if n > 0 && cr.cb != nil {
		cr.cb(int64(n))
	}
	return n, err
}

type countingWriter struct {
	w  *io.PipeWriter
	cb func(int64)
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	if n > 0 && cw.cb != nil {
		cw.cb(int64(n))
	}
	return n, err
}

func (cw *countingWriter) Close() error {
	return cw.w.Close()
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	value := float64(n) / float64(div)
	suffix := []string{"KB", "MB", "GB", "TB"}[exp]
	return fmt.Sprintf("%.2f %s", value, suffix)
}

// MultipartWriter wrapper для multipart.Writer с удобными методами
type MultipartWriter struct {
	writer *multipart.Writer
}

// NewMultipartWriter создает новый MultipartWriter
func NewMultipartWriter(w io.Writer) *MultipartWriter {
	return &MultipartWriter{
		writer: multipart.NewWriter(w),
	}
}

// WriteFile записывает файл в multipart form
func (mw *MultipartWriter) WriteFile(fieldName, filename string, data []byte) error {
	part, err := mw.writer.CreateFormFile(fieldName, filename)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}

// WriteField записывает текстовое поле в multipart form
func (mw *MultipartWriter) WriteField(fieldName, value string) error {
	return mw.writer.WriteField(fieldName, value)
}

// Close закрывает multipart writer
func (mw *MultipartWriter) Close() error {
	return mw.writer.Close()
}

// FormDataContentType возвращает Content-Type для multipart/form-data
func (mw *MultipartWriter) FormDataContentType() string {
	return mw.writer.FormDataContentType()
}
