package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"

	"golang.org/x/mod/semver"

	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/inconshreveable/go-update"
)

type UpdateInfo struct {
	Version  string `json:"version"`
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
}

type Updater struct {
	UpdateInfo *UpdateInfo
}

const uploaderUpdateURL = "https://lexore.ca/rocky/api/rockpload"

func NewUpdater() *Updater {
	logger.FuncDebug()
	return &Updater{}
}

func (u *Updater) CheckForUpdate(currentVersion string) (bool, error) {
	logger.FuncDebug()
	url := fmt.Sprintf(uploaderUpdateURL+"?os=%s", runtime.GOOS)

	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var info UpdateInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		return false, err
	}

	if info.Version == "" {
		logger.Rlogger.Info("No update available")
		return false, nil
	}

	formatVersion := func(v string) string {
		if len(v) > 0 && v[0] != 'v' {
			return "v" + v
		}
		return v
	}

	if semver.Compare(formatVersion(info.Version), formatVersion(currentVersion)) == 1 {
		logger.Rlogger.Info("New version available: "+info.Version, slog.Any("current", currentVersion))
		u.UpdateInfo = &info
		return true, nil
	}

	return false, nil
}

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

func verifyChecksum(data []byte, expected string) bool {
	logger.FuncDebug()
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]) == expected
}
