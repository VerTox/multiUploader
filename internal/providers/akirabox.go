package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	akiraboxBaseURL = "https://akirabox.com"
)

// AkiraBoxProvider провайдер для AkiraBox.com
type AkiraBoxProvider struct {
	apiToken string
}

// NewAkiraBoxProvider создает новый провайдер AkiraBox.com
func NewAkiraBoxProvider(apiToken string) *AkiraBoxProvider {
	return &AkiraBoxProvider{apiToken: apiToken}
}

func (a *AkiraBoxProvider) Name() string {
	return "AkiraBox"
}

func (a *AkiraBoxProvider) RequiresAuth() bool {
	return true
}

func (a *AkiraBoxProvider) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API token is required")
	}
	return nil
}

// Upload загружает файл на AkiraBox.com
func (a *AkiraBoxProvider) Upload(ctx context.Context, file io.ReadSeeker, filename string, fileSize int64, progress chan<- UploadProgress) (*UploadResult, error) {
	// 1. Инициализация upload
	startData, err := a.startUpload(ctx, filename, fileSize)
	if err != nil {
		return nil, fmt.Errorf("start upload failed: %w", err)
	}

	// 2. Загружаем части
	parts, err := a.uploadParts(ctx, file, fileSize, startData, progress)
	if err != nil {
		return nil, fmt.Errorf("upload parts failed: %w", err)
	}

	// 3. Завершаем upload
	downloadLink, err := a.completeUpload(ctx, startData, parts)
	if err != nil {
		return nil, fmt.Errorf("complete upload failed: %w", err)
	}

	return &UploadResult{
		URL:         downloadLink,
		DownloadURL: downloadLink,
	}, nil
}

// startUploadResponse структура ответа от /api/upload/start
type startUploadResponse struct {
	UploadID    string `json:"uploadId"`
	Key         string `json:"key"`
	ProviderID  int64  `json:"providerId"`
	ChunkSize   int64  `json:"chunkSize"`
	TotalChunks int    `json:"totalChunks"`
	Metadata    string `json:"metadata"`
}

// startUpload инициализирует загрузку
func (a *AkiraBoxProvider) startUpload(ctx context.Context, filename string, fileSize int64) (*startUploadResponse, error) {
	u, err := url.Parse(akiraboxBaseURL + "/api/upload/start")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("api_token", a.apiToken)
	q.Set("file", filename)
	q.Set("fileSize", fmt.Sprintf("%d", fileSize))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("start upload failed with status %d", resp.StatusCode)
	}

	var result startUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// chunkURLResponse структура ответа от /api/upload/chunk-url
type chunkURLResponse struct {
	URL string `json:"url"`
}

// getChunkURL получает presigned URL для загрузки чанка
func (a *AkiraBoxProvider) getChunkURL(ctx context.Context, startData *startUploadResponse, partNumber int) (string, error) {
	u, err := url.Parse(akiraboxBaseURL + "/api/upload/chunk-url")
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("api_token", a.apiToken)
	q.Set("uploadId", startData.UploadID)
	q.Set("part-number", fmt.Sprintf("%d", partNumber))
	q.Set("key", startData.Key)
	q.Set("providerId", strconv.FormatInt(startData.ProviderID, 10))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get chunk URL failed with status %d", resp.StatusCode)
	}

	var result chunkURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.URL, nil
}

// uploadParts загружает все части файла
func (a *AkiraBoxProvider) uploadParts(ctx context.Context, file io.ReadSeeker, fileSize int64, startData *startUploadResponse, progress chan<- UploadProgress) ([]map[string]interface{}, error) {
	speedCalc := NewSpeedCalculator()
	uploadedParts := make([]map[string]interface{}, startData.TotalChunks)
	var totalUploaded int64

	chunkSize := startData.ChunkSize

	// Загружаем части последовательно
	for partNum := 1; partNum <= startData.TotalChunks; partNum++ {
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

		// Получаем URL для загрузки
		uploadURL, err := a.getChunkURL(ctx, startData, partNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get URL for part %d: %w", partNum, err)
		}

		// Загружаем часть с отслеживанием прогресса
		etag, err := a.uploadPartWithProgress(ctx, uploadURL, limitedReader, partSize, &totalUploaded, fileSize, speedCalc, progress)
		if err != nil {
			return nil, fmt.Errorf("failed to upload part %d: %w", partNum, err)
		}

		// Сохраняем информацию о части
		uploadedParts[partNum-1] = map[string]interface{}{
			"PartNumber": partNum,
			"ETag":       etag,
		}
	}

	return uploadedParts, nil
}

// uploadPartWithProgress загружает часть файла с отслеживанием прогресса
func (a *AkiraBoxProvider) uploadPartWithProgress(ctx context.Context, uploadURL string, reader io.Reader, partSize int64, totalUploaded *int64, fileSize int64, speedCalc *SpeedCalculator, progress chan<- UploadProgress) (string, error) {
	// Создаем reader с отслеживанием прогресса
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, progressReader)
	if err != nil {
		return "", err
	}

	req.ContentLength = partSize
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
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

// completeUpload завершает загрузку
func (a *AkiraBoxProvider) completeUpload(ctx context.Context, startData *startUploadResponse, parts []map[string]interface{}) (string, error) {
	u, err := url.Parse(akiraboxBaseURL + "/api/upload/complete")
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("api_token", a.apiToken)
	q.Set("key", startData.Key)
	q.Set("providerId", strconv.FormatInt(startData.ProviderID, 10))
	u.RawQuery = q.Encode()

	body := map[string]interface{}{
		"UploadId": startData.UploadID,
		"MultipartUpload": map[string]interface{}{
			"Parts": parts,
		},
		"metadata": startData.Metadata,
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("complete upload failed with status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	downloadLink, ok := result["download_link"].(string)
	if !ok {
		return "", fmt.Errorf("download_link not found in response")
	}

	return downloadLink, nil
}
