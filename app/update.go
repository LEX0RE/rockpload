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

	"lexore/rockpload/app/tools/logger"

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
    url := fmt.Sprintf(uploaderUpdateURL + "?os=%s", runtime.GOOS)

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

    logger.Rlogger.Debug("Current version", slog.Any("current", currentVersion), slog.Any("latest", info.Version))
    
    if info.Version == "" {
        return false, fmt.Errorf("no update available")
    }

    if info.Version != currentVersion {
        logger.Rlogger.Info("New version available: " + info.Version)
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