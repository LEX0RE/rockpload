//go:build !android && !ios

package app

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/inconshreveable/go-update"
)

func (u *Updater) ApplyUpdate(onProgress UpdateProgress) error {
	logger.FuncDebug()

	resp, err := updateDownloadClient.Get(u.UpdateInfo.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(&progressReader{reader: resp.Body, total: resp.ContentLength, onProgress: onProgress})
	if err != nil {
		return err
	}

	if !verifyChecksum(data, u.UpdateInfo.Checksum) {
		return fmt.Errorf("invalid checksum")
	}

	return update.Apply(bytes.NewReader(data), update.Options{})
}
