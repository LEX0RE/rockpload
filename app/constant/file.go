package constant

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
)

const (
	rockploadCacheDir = "rockpload"
)

type AppPaths struct {
	CacheDir  string
	ConfigDir string
	HomeDir   string

	UploadedCache  string
	BrowserSession string
	AppLog         string

	AppLock    string
	TokensPath string

	SettingsFolder       string
	AccountSettingsFile  string
	StorageSettingsFile  string
	BehaviorSettingsFile string
	SecretSettingsFile   string
}

var Paths *AppPaths

func InitPaths(force bool) {
	// No logger as constant are used in logger
	if Paths != nil && force != true {
		return
	}

	cacheDir := getCachePath()
	configDir := getConfigPath()
	homeDir := getHomePath()

	settingsDir := filepath.Join(configDir, "settings")
	os.MkdirAll(settingsDir, 0700)

	Paths = &AppPaths{
		CacheDir:  cacheDir,
		ConfigDir: configDir,
		HomeDir:   homeDir,

		UploadedCache:  filepath.Join(cacheDir, ".uploaded"),
		BrowserSession: filepath.Join(cacheDir, ".browser_session"),
		AppLog:         filepath.Join(cacheDir, "rockpload.log"),

		AppLock:    filepath.Join(configDir, "rockpload.lock"),
		TokensPath: filepath.Join(configDir, ".tokens"),

		SettingsFolder:       settingsDir,
		AccountSettingsFile:  filepath.Join(settingsDir, "account.json"),
		StorageSettingsFile:  filepath.Join(settingsDir, "storage.json"),
		BehaviorSettingsFile: filepath.Join(settingsDir, "behavior.json"),
		SecretSettingsFile:   filepath.Join(settingsDir, ".secrets.json"),
	}
}

func getCachePath() string {
	// No logger as constant are used in logger
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}

	dir = filepath.Join(dir, rockploadCacheDir)
	os.MkdirAll(dir, 0700)

	os.Chmod(dir, 0700)

	return dir
}

func getConfigPath() string {
	// No logger as constant are used in logger
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

func getHomePath() string {
	// No logger as constant are used in logger
	if runtime.GOOS == "android" {
		log.Println("Android detected: redirecting HomePath to internal ConfigPath")
		return getConfigPath()
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		dir = getConfigPath()
	}

	return dir
}
