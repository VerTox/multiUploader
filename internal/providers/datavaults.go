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

	"multiUploader/internal/httpclient"
)

const (
	baseURL             = "https://datavaults.co/"
	selectServerPostfix = "api/upload/server"
)

type fileUploadResponse struct {
	FileCode   string `json:"file_code"`
	FileStatus string `json:"file_status"`
}

type serverSelectionResponse struct {
	Status     int    `json:"status"`
	SessId     string `json:"sess_id"`
	Result     string `json:"result"`
	Msg        string `json:"msg"`
	ServerTime string `json:"server_time"`
}

func NewDataVaultsProvider(apiKey string) *DataVaults {
	return &DataVaults{ApiKey: apiKey}
}

type DataVaults struct {
	ApiKey string
}

func (d DataVaults) Name() string {
	return "DataVaults"
}

func (d DataVaults) Upload(ctx context.Context, file io.ReadSeeker, filename string, fileSize int64, progress chan<- UploadProgress) (*UploadResult, error) {
	curl, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	curl.Path = selectServerPostfix
	curl.RawQuery = url.Values{"key": []string{d.ApiKey}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, curl.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpclient.Default().Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	response := serverSelectionResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Status != 200 {
		return nil, fmt.Errorf("DataVaults server returned error: %s", response.Msg)
	}

	pipeR, pipeW := io.Pipe()
	mw := multipart.NewWriter(pipeW)

	var fileSent ByteCounter

	// Горутина для записи multipart данных в pipe
	go func() {
		defer func() {
			_ = mw.Close()
			_ = pipeW.Close()
		}()

		// Текстовые поля
		if err := mw.WriteField("sess_id", response.SessId); err != nil {
			_ = pipeW.CloseWithError(err)
			return
		}
		if err := mw.WriteField("utype", "prem"); err != nil {
			_ = pipeW.CloseWithError(err)
			return
		}

		// Файл
		part, err := mw.CreateFormFile("file_0", filename)
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

	req2, err := http.NewRequestWithContext(ctx, http.MethodPost, response.Result, pipeR)
	if err != nil {
		_ = pipeR.Close()
		_ = pipeW.Close()
		return nil, err
	}
	req2.Header.Set("Content-Type", mw.FormDataContentType())

	stopProgress := make(chan struct{})
	defer close(stopProgress)

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
					speed = float64(df) / dt // bytes/sec по файлу
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

	resp, reqErr := httpclient.LongLived().Do(req2)
	_ = pipeR.Close()
	if reqErr != nil {
		if errors.Is(reqErr, context.Canceled) {
			return nil, fmt.Errorf("upload cancelled")
		}
		return nil, reqErr
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DataVaults server returned error: %s", resp.Status)
	}

	uploadResp := make([]fileUploadResponse, 0)
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, err
	}

	if len(uploadResp) == 0 {
		return nil, fmt.Errorf("DataVaults returned empty response")
	}

	return &UploadResult{
		URL: baseURL + uploadResp[0].FileCode,
	}, nil
}

func (d DataVaults) RequiresAuth() bool {
	return true
}

func (d DataVaults) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}
