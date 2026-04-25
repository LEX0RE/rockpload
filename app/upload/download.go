package upload

import (
	"fmt"
	"io"
	"lexore/rockpload/app/tools/logger"
	"net/http"
	"os"
)

func downloadFile(url string) (string, error) {
	logger.FuncDebug()
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to download: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "*.replay")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}