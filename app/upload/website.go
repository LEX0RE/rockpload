package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type Website struct {
	config *config.StorageConfig
}

func NewWebsite(config *config.StorageConfig) *Website {
	return &Website{config: config}
}

func (w *Website) UploadReplay(filePath string, replayUpload ReplayUpload) error {
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

	fileName := replayUploadFileName(filePath, w.config.TemplateName, replayUpload)
	part, err := writer.CreateFormFile("file", fileName)
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

		w.handleBallchasingRenaming(fileName, respBody)
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

func (w *Website) handleBallchasingRenaming(fileName string, respBody []byte) {
	if w.GetConfig().Name == config.BALLCHASING_STORAGE.Name {
		var data struct {
			ID string `json:"id"`
		}

		if err := json.Unmarshal(respBody, &data); err != nil {
			logger.Rlogger.Debug("Upload successful but failed to parse response id", "err", err)
			return
		}

		title := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		ballchasingPatchPath := w.config.URL + "/replays/" + data.ID

		patchBody, _ := json.Marshal(map[string]string{"title": title})
		patchReq, err := http.NewRequest("PATCH", ballchasingPatchPath, bytes.NewReader(patchBody))
		if err != nil {
			logger.Rlogger.Debug("Failed to create patch request", "err", err)
			return
		}

		patchReq.Header.Set("Content-Type", "application/json")
		if w.config.NeedToken && w.config.Token != "" {
			patchReq.Header.Set("Authorization", w.config.Token)
		}

		patchClient := &http.Client{}
		patchResp, err := patchClient.Do(patchReq)
		if err != nil {
			logger.Rlogger.Debug("Failed to patch replay title", "err", err)
			return
		}
		defer patchResp.Body.Close()

		if patchResp.StatusCode >= 200 && patchResp.StatusCode < 300 {
			logger.Rlogger.Debug("Upload successful and title updated")
			return
		}

		b, _ := io.ReadAll(patchResp.Body)
		logger.Rlogger.Debug("Upload successful but patch failed", "status", patchResp.Status, "body", string(b))
	}
}
