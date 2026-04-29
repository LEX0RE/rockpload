package upload

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type RockyWebsite struct {
	apiURL string
}

func NewRockyWebsite() *RockyWebsite {
	logger.FuncDebug()
	return &RockyWebsite{apiURL: "https://lexore.ca/rocky/api"}
}

func (w *RockyWebsite) UploadReplay(filePath string) error {
	logger.FuncDebug()

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

	req, err := http.NewRequest("POST", w.apiURL+"/upload", &body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 201 {
		logger.Rlogger.Debug("Upload successful ✅")
	} else if resp.StatusCode == 409 {
		logger.Rlogger.Debug("Upload duplicate ✅")
	} else {
		return fmt.Errorf("Upload failed: %s\n%s", resp.Status, string(respBody))
	}

	logger.Rlogger.Debug("Upload response:", slog.Any("body", string(respBody)))

	return nil
}
