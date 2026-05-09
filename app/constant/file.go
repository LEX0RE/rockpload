package constant

import (
	"os"
	"path/filepath"
)

const (
	rockploadCacheDir = "rockpload"
)

var (
	UploadedCache  = filepath.Join(GetCachePath(), ".uploaded")
	BrowserSession = filepath.Join(GetCachePath(), ".browser_session")

	AppLock      = filepath.Join(GetConfigPath(), "rockpload.lock")
	SettingsFile = filepath.Join(GetConfigPath(), "settings.json")
	TokensPath   = filepath.Join(GetConfigPath(), ".tokens")
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
