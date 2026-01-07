package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"multiUploader/internal/httpclient"
)

const (
	rootzBaseURL            = "https://www.rootz.so"
	rootzMultipartThreshold = 4 * 1024 * 1024 // 4MB
)

// RootzProvider провайдер для Rootz.so
type RootzProvider struct {
	apiKey string
}

// NewRootzProvider создает новый провайдер Rootz.so
func NewRootzProvider(apiKey string) *RootzProvider {
	return &RootzProvider{apiKey: apiKey}
}

func (r *RootzProvider) Name() string {
	return "Rootz"
}

func (r *RootzProvider) RequiresAuth() bool {
	return true
}

func (r *RootzProvider) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// Upload загружает файл на Rootz.so
func (r *RootzProvider) Upload(ctx context.Context, file io.ReadSeeker, filename string, fileSize int64, progress chan<- UploadProgress) (*UploadResult, error) {
	// Выбираем метод загрузки в зависимости от размера файла
	if fileSize < rootzMultipartThreshold {
		return r.uploadSmallFile(ctx, file, filename, fileSize, progress)
	}
	return r.uploadLargeFile(ctx, file, filename, fileSize, progress)
}

// uploadSmallFile загружает маленький файл (<4MB) напрямую
func (r *RootzProvider) uploadSmallFile(ctx context.Context, file io.Reader, filename string, fileSize int64, progress chan<- UploadProgress) (*UploadResult, error) {
	// Читаем весь файл в память (он маленький)
	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Создаем multipart form
	body := &bytes.Buffer{}
	writer := NewMultipartWriter(body)

	if err := writer.WriteFile("file", filename, fileData); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Создаем запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rootzBaseURL+"/api/files/upload", body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if r.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiKey)
	}

	// Отправляем запрос
	resp, err := httpclient.Default().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	// Парсим ответ
	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Data    struct {
			ShortID string `json:"shortId"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("upload failed: %s", result.Error)
	}

	// Отправляем финальный прогресс
	progress <- UploadProgress{
		BytesUploaded: fileSize,
		TotalBytes:    fileSize,
		Speed:         0,
		Percentage:    100,
	}

	return &UploadResult{
		URL:    fmt.Sprintf("%s/d/%s", rootzBaseURL, result.Data.ShortID),
		FileID: result.Data.ShortID,
	}, nil
}

// uploadLargeFile загружает большой файл (≥4MB) через multipart upload
func (r *RootzProvider) uploadLargeFile(ctx context.Context, file io.ReadSeeker, filename string, fileSize int64, progress chan<- UploadProgress) (*UploadResult, error) {
	// 1. Инициализация multipart upload
	initReq := map[string]interface{}{
		"fileName": filename,
		"fileSize": fileSize,
		"fileType": "application/octet-stream",
	}

	initResp, err := r.makeJSONRequest(ctx, http.MethodPost, "/api/files/multipart/init", initReq)
	if err != nil {
		return nil, fmt.Errorf("init failed: %w", err)
	}

	uploadID := initResp["uploadId"].(string)
	key := initResp["key"].(string)
	serverChunkSize := int64(initResp["chunkSize"].(float64))
	totalParts := int(initResp["totalParts"].(float64))

	// 2. Получаем presigned URLs для всех частей
	urlsReq := map[string]interface{}{
		"key":        key,
		"uploadId":   uploadID,
		"totalParts": totalParts,
	}

	urlsResp, err := r.makeJSONRequestNoAuth(ctx, http.MethodPost, "/api/files/multipart/batch-urls", urlsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get URLs: %w", err)
	}

	if !urlsResp["success"].(bool) {
		errMsg := "unknown error"
		if e, ok := urlsResp["error"].(string); ok {
			errMsg = e
		}
		return nil, fmt.Errorf("failed to get URLs: %s", errMsg)
	}

	urls := urlsResp["urls"].(map[string]interface{})

	// 3. Загружаем части
	uploadedParts, err := r.uploadParts(ctx, file, fileSize, serverChunkSize, totalParts, urls, progress)
	if err != nil {
		return nil, fmt.Errorf("upload parts failed: %w", err)
	}

	// 4. Завершаем upload
	completeReq := map[string]interface{}{
		"key":         key,
		"uploadId":    uploadID,
		"parts":       uploadedParts,
		"fileName":    filename,
		"fileSize":    fileSize,
		"contentType": "application/octet-stream",
	}

	completeResp, err := r.makeJSONRequest(ctx, http.MethodPost, "/api/files/multipart/complete", completeReq)
	if err != nil {
		return nil, fmt.Errorf("complete failed: %w", err)
	}

	if !completeResp["success"].(bool) {
		errMsg := "unknown error"
		if e, ok := completeResp["error"].(string); ok {
			errMsg = e
		}
		return nil, fmt.Errorf("complete failed: %s", errMsg)
	}

	fileData := completeResp["file"].(map[string]interface{})
	shortID := fileData["shortId"].(string)

	return &UploadResult{
		URL:    fmt.Sprintf("%s/d/%s", rootzBaseURL, shortID),
		FileID: shortID,
	}, nil
}

// uploadParts загружает части файла используя Seek для эффективной работы с большими файлами
func (r *RootzProvider) uploadParts(ctx context.Context, file io.ReadSeeker, fileSize int64, chunkSize int64, totalParts int, urls map[string]interface{}, progress chan<- UploadProgress) ([]map[string]interface{}, error) {
	speedCalc := NewSpeedCalculator()
	uploadedParts := make([]map[string]interface{}, totalParts)
	var totalUploaded int64

	// Загружаем части последовательно
	for partNum := 1; partNum <= totalParts; partNum++ {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("upload cancelled")
		default:
		}

		// Вычисляем границы части
		start := int64(partNum-1) * chunkSize
		partSize := chunkSize
		if start+partSize > fileSize {
			partSize = fileSize - start
		}

		// Перемещаемся к началу части
		_, err := file.Seek(start, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to part %d: %w", partNum, err)
		}

		// Создаем LimitReader для чтения только текущего чанка
		limitedReader := io.LimitReader(file, partSize)

		url := urls[fmt.Sprintf("%d", partNum)].(string)

		// Загружаем часть с отслеживанием прогресса
		etag, err := r.uploadPartWithProgress(ctx, url, limitedReader, partSize, &totalUploaded, fileSize, speedCalc, progress)
		if err != nil {
			return nil, fmt.Errorf("failed to upload part %d: %w", partNum, err)
		}

		// Сохраняем информацию о части
		uploadedParts[partNum-1] = map[string]interface{}{
			"partNumber": partNum,
			"etag":       etag,
		}
	}

	return uploadedParts, nil
}

// uploadPartWithProgress загружает часть файла с отслеживанием прогресса в реальном времени
func (r *RootzProvider) uploadPartWithProgress(ctx context.Context, url string, reader io.Reader, partSize int64, totalUploaded *int64, fileSize int64, speedCalc *SpeedCalculator, progress chan<- UploadProgress) (string, error) {
	// Создаем reader с отслеживанием прогресса
	// Обновляем прогресс каждые 512KB для плавного отображения
	const progressChunkSize = 512 * 1024 // 512KB
	var lastProgressUpdate int64

	progressReader := &progressReader{
		reader: reader,
		onProgress: func(n int64) {
			*totalUploaded += n

			// Обновляем прогресс не чаще чем каждые 512KB
			if *totalUploaded-lastProgressUpdate >= progressChunkSize || *totalUploaded == fileSize {
				lastProgressUpdate = *totalUploaded
				speed := speedCalc.Update(*totalUploaded)
				percentage := int(float64(*totalUploaded) / float64(fileSize) * 100)

				select {
				case progress <- UploadProgress{
					BytesUploaded: *totalUploaded,
					TotalBytes:    fileSize,
					Speed:         speed,
					Percentage:    percentage,
				}:
				default:
					// Канал прогресса заполнен, пропускаем обновление
				}
			}
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, progressReader)
	if err != nil {
		return "", err
	}

	req.ContentLength = partSize

	resp, err := httpclient.LongLived().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	// Получаем ETag и убираем кавычки
	etag := resp.Header.Get("ETag")
	etag = strings.Trim(etag, "\"")

	return etag, nil
}

// progressReader оборачивает io.Reader и вызывает callback при каждом чтении
type progressReader struct {
	reader     io.Reader
	onProgress func(n int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 && pr.onProgress != nil {
		pr.onProgress(int64(n))
	}
	return n, err
}

// makeJSONRequest выполняет JSON запрос с авторизацией
func (r *RootzProvider) makeJSONRequest(ctx context.Context, method, path string, data interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, rootzBaseURL+path, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if r.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiKey)
	}

	resp, err := httpclient.Default().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// makeJSONRequestNoAuth выполняет JSON запрос без авторизации
func (r *RootzProvider) makeJSONRequestNoAuth(ctx context.Context, method, path string, data interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, rootzBaseURL+path, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpclient.Default().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
