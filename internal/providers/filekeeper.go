package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

const (
	filekeeperBaseURL = "https://filekeeper.net"
)

// FileKeeperProvider провайдер для FileKeeper.net
type FileKeeperProvider struct {
	apiKey string
}

// NewFileKeeperProvider создает новый провайдер FileKeeper.net
func NewFileKeeperProvider(apiKey string) *FileKeeperProvider {
	return &FileKeeperProvider{apiKey: apiKey}
}

func (f *FileKeeperProvider) Name() string {
	return "FileKeeper"
}

func (f *FileKeeperProvider) RequiresAuth() bool {
	return true
}

func (f *FileKeeperProvider) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// serverResponse структура ответа от /api/upload/server
type filekeeperServerResponse struct {
	Msg    string `json:"msg"`
	Result string `json:"result"`
	SessID string `json:"sess_id"`
	Status int    `json:"status"`
}

// Upload загружает файл на FileKeeper.net
func (f *FileKeeperProvider) Upload(ctx context.Context, file io.ReadSeeker, filename string, fileSize int64, progress chan<- UploadProgress) (*UploadResult, error) {
	// 1. Получаем URL сервера для загрузки
	serverData, err := f.getUploadServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload server: %w", err)
	}

	// 2. Загружаем файл
	fileCode, err := f.uploadFile(ctx, serverData, file, filename, fileSize, progress)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// 3. Формируем URL файла
	fileURL := fmt.Sprintf("%s/%s", filekeeperBaseURL, fileCode)

	return &UploadResult{
		URL:    fileURL,
		FileID: fileCode,
	}, nil
}

// getUploadServer получает URL сервера для загрузки
func (f *FileKeeperProvider) getUploadServer(ctx context.Context) (*filekeeperServerResponse, error) {
	u, err := url.Parse(filekeeperBaseURL + "/api/upload/server")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("key", f.apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get upload server failed with status %d", resp.StatusCode)
	}

	var result filekeeperServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != 200 {
		return nil, fmt.Errorf("server returned error: %s", result.Msg)
	}

	return &result, nil
}

// filekeeperUploadResponse структура ответа от сервера загрузки
type filekeeperUploadResponse struct {
	FileCode   string `json:"file_code"`
	FileStatus string `json:"file_status"`
}

// uploadFile загружает файл на сервер
func (f *FileKeeperProvider) uploadFile(ctx context.Context, serverData *filekeeperServerResponse, file io.ReadSeeker, filename string, fileSize int64, progress chan<- UploadProgress) (string, error) {
	pipeR, pipeW := io.Pipe()
	mw := multipart.NewWriter(pipeW)

	var fileSent ByteCounter

	// Горутина для записи multipart данных в pipe
	go func() {
		defer func() {
			_ = mw.Close()
			_ = pipeW.Close()
		}()

		// Поле sess_id
		if err := mw.WriteField("sess_id", serverData.SessID); err != nil {
			_ = pipeW.CloseWithError(err)
			return
		}

		// Файл
		part, err := mw.CreateFormFile("file", filename)
		if err != nil {
			_ = pipeW.CloseWithError(err)
			return
		}

		// Считаем байты файла при чтении
		cr := CountingReader{
			r: file,
			cb: func(n int64) {
				fileSent.Add(n)
			},
		}

		_, err = io.Copy(part, cr)
		if err != nil {
			_ = pipeW.CloseWithError(err)
			return
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverData.Result, pipeR)
	if err != nil {
		_ = pipeR.Close()
		_ = pipeW.Close()
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	stopProgress := make(chan struct{})
	defer close(stopProgress)

	// Горутина для отслеживания прогресса
	go func() {
		ticker := time.NewTicker(ProgressUpdateInterval)
		defer ticker.Stop()

		start := time.Now()
		var lastFile int64
		var lastT = start
		var speed float64

		for {
			select {
			case <-ctx.Done():
				return
			case <-stopProgress:
				return
			case <-ticker.C:
				now := time.Now()
				fs := fileSent.N()

				dt := now.Sub(lastT).Seconds()
				df := fs - lastFile
				if dt > 0 && df > 0 {
					speed = float64(df) / dt // bytes/sec
				}

				var pct float64
				if fileSize > 0 {
					pct = (float64(fs) / float64(fileSize)) * 100.0
					if pct > 100 {
						pct = 100
					}
				}

				upd := UploadProgress{
					BytesUploaded: fs,
					TotalBytes:    fileSize,
					Speed:         speed,
					Percentage:    int(pct),
				}

				select {
				case <-ctx.Done():
					return
				case <-stopProgress:
					return
				case progress <- upd:
				}

				lastFile = fs
				lastT = now
			}
		}
	}()

	client := &http.Client{Timeout: 0}

	resp, reqErr := client.Do(req)
	_ = pipeR.Close()
	if reqErr != nil {
		if errors.Is(reqErr, context.Canceled) {
			return "", fmt.Errorf("upload cancelled")
		}
		return "", reqErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	// Парсим ответ - ожидаем массив с одним элементом
	var uploadResp []filekeeperUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", err
	}

	if len(uploadResp) == 0 {
		return "", fmt.Errorf("FileKeeper returned empty response")
	}

	return uploadResp[0].FileCode, nil
}
