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

func (u *Updater) ApplyUpdate() error {
	logger.FuncDebug()

	resp, err := http.Get(u.UpdateInfo.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if !verifyChecksum(data, u.UpdateInfo.Checksum) {
		return fmt.Errorf("invalid checksum")
	}

	return update.Apply(bytes.NewReader(data), update.Options{})
}
