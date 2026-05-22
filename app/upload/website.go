package upload

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type Website struct {
	config *config.StorageConfig
}

func NewWebsite(config *config.StorageConfig) *Website {
	return &Website{config: config}
}

func (w *Website) UploadReplay(filePath string) error {
	logger.FuncDebug()

	if !w.config.SendReplay {
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Failed to open file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", w.config.URL+w.config.ReplayPath, &body)
	if err != nil {
		return err
	}

	// Send Ping
	if w.config.SendPing {
		if err := w.Ping(); err != nil {
			return fmt.Errorf("Website ping failed: %w", err)
		}
	}

	// Token
	if w.config.NeedToken && w.config.Token != "" {
		req.Header.Set("Authorization", w.config.Token)
	}

	// URI Params
	q := req.URL.Query()
	for key, value := range w.config.URIParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case 201:
		logger.Rlogger.Debug("Upload successful")
	case 409:
		logger.Rlogger.Debug("Duplicate upload")
	default:
		return fmt.Errorf("Upload failed: %s\n%s", resp.Status, string(respBody))
	}

	return nil
}

func (w *Website) Ping() error {
	if !w.config.SendPing {
		return nil
	}

	req, err := http.NewRequest("GET", w.config.URL+w.config.PingPath, nil)
	if err != nil {
		return err
	}

	if w.config.NeedToken && w.config.Token != "" {
		req.Header.Set("Authorization", w.config.Token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("Token is invalid: %s\n%s", resp.Status, string(body))
}

func (w *Website) GetConfig() *config.StorageConfig {
	return w.config
}
