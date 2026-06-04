package upload

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type FileSystem struct {
	config *config.StorageConfig
}

func NewFileSystem(config *config.StorageConfig) *FileSystem {
	return &FileSystem{config: config}
}

func (fs *FileSystem) UploadReplay(filePath string, replayUpload ReplayUpload) error {
	logger.FuncDebug()

	if !fs.config.SendReplay {
		return nil
	}

	srcFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	destFolder := fs.config.ReplayPath
	if destFolder == "" {
		return fmt.Errorf("replay path is not configured")
	}

	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination folder: %w", err)
	}

	fileName := replayUploadFileName(filePath, fs.config.TemplateName, replayUpload)
	destPath := filepath.Join(destFolder, fileName)

	if _, err := os.Stat(destPath); err == nil {
		logger.Rlogger.Debug("Duplicate file: replacing the existing one")
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file to destination: %w", err)
	}

	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync file to disk: %w", err)
	}

	logger.Rlogger.Debug("Upload (copy) successful")
	return nil
}

func (w *FileSystem) UploadLive(liveData *rocket_network.UpdateState) error {
	if !w.config.SendLive {
		return nil
	}

	return fmt.Errorf("File System can't send live data")
}

func (w *FileSystem) Ping() error {
	if !w.config.SendPing {
		return nil
	}

	destFolder := w.config.ReplayPath
	if destFolder == "" {
		return fmt.Errorf("replay path is empty, cannot ping")
	}

	info, err := os.Stat(destFolder)
	if os.IsNotExist(err) {
		err := os.MkdirAll(destFolder, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to access or create target directory: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("error accessing target directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target path is a file, not a directory: %s", destFolder)
	}

	return nil
}

func (fs *FileSystem) GetConfig() *config.StorageConfig {
	return fs.config
}
