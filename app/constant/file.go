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
	AppLog         = filepath.Join(GetCachePath(), "rockpload.log")

	AppLock    = filepath.Join(GetConfigPath(), "rockpload.lock")
	TokensPath = filepath.Join(GetConfigPath(), ".tokens")

	SettingsFolder       = filepath.Join(GetConfigPath(), "settings")
	AccountSettingsFile  = filepath.Join(SettingsFolder, "account.json")
	StorageSettingsFile  = filepath.Join(SettingsFolder, "storage.json")
	BehaviorSettingsFile = filepath.Join(SettingsFolder, "behavior.json")
	SecretSettingsFile   = filepath.Join(SettingsFolder, ".secrets.json")
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
		dir, err = os.UserCacheDir()
		if err != nil {
			dir = os.TempDir()
		}
	}

	dir = filepath.Join(dir, rockploadCacheDir)
	os.MkdirAll(dir, 0700)

	os.Chmod(dir, 0700)

	return dir
}

func GetHomePath() string {
	// No logger as some global var use it
	dir, err := os.UserHomeDir()
	if err != nil {
		dir = GetConfigPath()
	}

	return dir
}
