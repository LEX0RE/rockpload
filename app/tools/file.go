package tools

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	FOUND_FILE_BOOT_LOOP = 10
	FOUND_FILE_BOOT_WAIT = 500 * time.Millisecond
)

func SaveJSONFilePath(filePath string, data any) error {
	logger.FuncDebug()

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return SaveFilePath(filePath, jsonData)
}

func SaveFilePath(filePath string, data []byte) error {
	logger.FuncDebug()

	logger.Rlogger.Debug("Saving JSON File", slog.Any("Path", filePath))

	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return err
	}

	err := os.WriteFile(filePath, data, 0600)
	if err != nil {
		return err
	}

	return os.Chmod(filePath, 0600)
}

func LoadJSONFilePath(filePath string, data any, errNotFound bool) error {
	logger.FuncDebug()

	logger.Rlogger.Info("Loading JSON File", slog.Any("Path", filePath))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if errNotFound {
			return err
		}
		return nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	return decoder.Decode(data)
}

// Wait a bit if file is not found
func WaitFileBoot(filesPath []string) {
	for range FOUND_FILE_BOOT_LOOP {
		for _, filePath := range filesPath {
			if _, err := os.Stat(filePath); err == nil {
				return
			}
		}

		time.Sleep(FOUND_FILE_BOOT_WAIT)
	}
}
