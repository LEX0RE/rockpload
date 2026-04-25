package config

import (
	"os"
	"path/filepath"
)

const (
	rockploadCacheDir = "rockpload"
)

var (
	RLToken = filepath.Join(GetCachePath(), ".rltoken")
	AppLock = filepath.Join(GetCachePath(), "rockpload.lock")
	UploadedCache = filepath.Join(GetCachePath(), ".uploaded")
	BrowserSession = filepath.Join(GetCachePath(), ".browser_session")
)

func GetCachePath() string {
	// No logger as some global var use it
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}

	dir = filepath.Join(dir, rockploadCacheDir)
	os.MkdirAll(dir, 0700)

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

	return dir
}