package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	rockploadCacheDir = "rockpload"
)

var (
	RLToken        = filepath.Join(GetCachePath(), ".rltoken")
	AppLock        = filepath.Join(GetCachePath(), "rockpload.lock")
	UploadedCache  = filepath.Join(GetCachePath(), ".uploaded")
	BrowserSession = filepath.Join(GetCachePath(), ".browser_session")
	SettingsFile   = filepath.Join(GetConfigPath(), "settings.json")
)

func GetCachePath() string {
	// No logger as some global var use it
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}

	dir = filepath.Join(dir, rockploadCacheDir)
	os.MkdirAll(dir, 0700)

	os.Chmod(dir, 0700)

	return dir
}

func GetConfigPath() string {
	// No logger as some global var use it
	dir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	dir = filepath.Join(dir, rockploadCacheDir)
	os.MkdirAll(dir, 0700)

	os.Chmod(dir, 0700)

	return dir
}

func SaveFilePath(filePath string, data any) error {
	logger.FuncDebug()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, jsonData, 0600)
	if err != nil {
		return err
	}

	return os.Chmod(filePath, 0600)
}

func LoadFilePath(filePath string, data any, errNotFound bool) error {
	logger.FuncDebug()

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
