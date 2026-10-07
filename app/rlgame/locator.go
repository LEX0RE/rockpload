package rlgame

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	LaunchLogName = "Launch.log"

	baseDirectoryMarker = "Base directory:"
	maxLogHeaderLines   = 200

	wineDriveFolder = "drive_c"
)

type Install struct {
	Dir        string
	Source     string
	LastLaunch time.Time
}

func (i Install) Label() string {
	return i.Source + " - " + i.Dir
}

// StatsAPIConfigPath returns the file read by the game: TAStatsAPI.ini when it exists, DefaultStatsAPI.ini otherwise.
func (i Install) StatsAPIConfigPath() string {
	configDir := filepath.Join(i.Dir, "TAGame", "Config")

	userPath := filepath.Join(configDir, StatsAPIUserFileName)
	if _, err := os.Stat(userPath); err == nil {
		return userPath
	}

	return filepath.Join(configDir, StatsAPIFileName)
}

// Locator keeps the Rocket League installations found on the system.
// The manual directory is only kept in memory and always takes priority over detected ones.
type Locator struct {
	mu        sync.RWMutex
	installs  []Install
	manualDir string
}

func NewLocator() *Locator {
	logger.FuncDebug()

	return &Locator{}
}

func (l *Locator) SetManualDir(dir string) {
	logger.FuncDebug()

	l.mu.Lock()
	l.manualDir = dir
	l.mu.Unlock()
}

func (l *Locator) ManualDir() string {
	logger.FuncDebug()

	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.manualDir
}

func (l *Locator) Refresh() []Install {
	logger.FuncDebug()

	installs := []Install{}

	if dir := l.ManualDir(); dir != "" && IsInstallDir(dir) {
		installs = append(installs, Install{Dir: filepath.Clean(dir), Source: "Manual"})
	}

	for _, install := range DetectInstalls() {
		if !containsInstall(installs, install.Dir) {
			installs = append(installs, install)
		}
	}

	l.mu.Lock()
	l.installs = installs
	l.mu.Unlock()

	return installs
}

func (l *Locator) Installs() []Install {
	logger.FuncDebug()

	l.mu.RLock()
	defer l.mu.RUnlock()

	return append([]Install{}, l.installs...)
}

func (l *Locator) Active() (Install, bool) {
	logger.FuncDebug()

	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.installs) == 0 {
		return Install{}, false
	}

	return l.installs[0], true
}

// StatsAPIPorts returns the TCP and WebSocket ports set in the active install StatsAPI file.
func (l *Locator) StatsAPIPorts() (StatsAPIPorts, error) {
	logger.FuncDebug()

	install, ok := l.Active()
	if !ok {
		return StatsAPIPorts{}, ErrInstallNotFound
	}

	file, err := LoadStatsAPIFile(install.StatsAPIConfigPath())
	if err != nil {
		return StatsAPIPorts{}, err
	}

	return file.Ports()
}

func containsInstall(installs []Install, dir string) bool {
	for _, install := range installs {
		if install.Dir == dir {
			return true
		}
	}

	return false
}

// LogsFolders returns every existing Rocket League "TAGame/Logs" folder on the system.
func LogsFolders() []string {
	logger.FuncDebug()

	homeDir, _ := os.UserHomeDir()

	wineLogs := filepath.Join("users", "*", "Documents", "My Games", "Rocket League", "TAGame", "Logs")
	heroicPrefixes := filepath.Join(homeDir, "Games", "Heroic", "Prefixes")

	patterns := []string{
		// Base Windows
		filepath.Join(homeDir, "Documents", "My Games", "Rocket League", "TAGame", "Logs"),
		// Windows OneDrive
		filepath.Join(homeDir, "OneDrive", "Documents", "My Games", "Rocket League", "TAGame", "Logs"),
		// Linux Steam Proton
		filepath.Join(homeDir, ".local", "share", "Steam", "steamapps", "compatdata", "252950", "pfx", wineDriveFolder, wineLogs),
		// Linux Heroic Launcher (Wine or Proton prefix, with or without the "default" folder)
		filepath.Join(heroicPrefixes, "*", wineDriveFolder, wineLogs),
		filepath.Join(heroicPrefixes, "*", "pfx", wineDriveFolder, wineLogs),
		filepath.Join(heroicPrefixes, "*", "*", wineDriveFolder, wineLogs),
		filepath.Join(heroicPrefixes, "*", "*", "pfx", wineDriveFolder, wineLogs),
		// MacOS
		filepath.Join(homeDir, "Library", "Application Support", "Rocket League", "TAGame", "Logs"),
	}

	folders := []string{}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err == nil && info.IsDir() && !containsString(folders, match) {
				folders = append(folders, match)
			}
		}
	}

	return folders
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}

	return false
}

// DetectInstalls finds the installs written in each Launch.log, the most recently launched first.
func DetectInstalls() []Install {
	logger.FuncDebug()

	installs := []Install{}

	for _, logFolder := range LogsFolders() {
		logPath := filepath.Join(logFolder, LaunchLogName)

		stat, err := os.Stat(logPath)
		if err != nil {
			continue
		}

		dir, err := InstallDirFromLog(logPath)
		if err != nil || containsInstall(installs, dir) {
			continue
		}

		installs = append(installs, Install{Dir: dir, Source: guessSource(dir), LastLaunch: stat.ModTime()})
	}

	sort.SliceStable(installs, func(i, j int) bool {
		return installs[i].LastLaunch.After(installs[j].LastLaunch)
	})

	return installs
}

// InstallDirFromLog reads the "Base directory" line of a Launch.log and resolves the install folder.
func InstallDirFromLog(logPath string) (string, error) {
	logger.FuncDebug()

	file, err := os.Open(logPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for i := 0; i < maxLogHeaderLines && scanner.Scan(); i++ {
		_, baseDir, ok := strings.Cut(scanner.Text(), baseDirectoryMarker)
		if !ok {
			continue
		}

		dir := ResolveGamePath(strings.TrimSpace(baseDir), filepath.Dir(logPath))
		// Base directory is "<install>/Binaries/Win64/"
		dir = filepath.Dir(filepath.Dir(filepath.Clean(dir)))

		if !IsInstallDir(dir) {
			return "", ErrInstallNotFound
		}

		return dir, nil
	}

	return "", ErrInstallNotFound
}

// ResolveGamePath converts a path seen by the game to a local path.
// Under Wine/Proton, "S:\..." is resolved with the prefix "dosdevices" links found from logFolder.
func ResolveGamePath(gamePath string, logFolder string) string {
	logger.FuncDebug()

	if runtime.GOOS == "windows" || !isWindowsDrivePath(gamePath) {
		return gamePath
	}

	rest := strings.ReplaceAll(gamePath[2:], `\`, "/")

	prefix, ok := winePrefix(logFolder)
	if !ok {
		return gamePath
	}

	drive := strings.ToLower(gamePath[:2])
	driveDir, err := filepath.EvalSymlinks(filepath.Join(prefix, "dosdevices", drive))
	if err != nil {
		switch drive {
		case "c:":
			driveDir = filepath.Join(prefix, wineDriveFolder)
		case "z:":
			driveDir = "/"
		default:
			return gamePath
		}
	}

	return filepath.Join(driveDir, rest)
}

func isWindowsDrivePath(path string) bool {
	if len(path) < 3 || path[1] != ':' || (path[2] != '\\' && path[2] != '/') {
		return false
	}

	letter := path[0] | 0x20
	return letter >= 'a' && letter <= 'z'
}

func winePrefix(path string) (string, bool) {
	marker := string(filepath.Separator) + wineDriveFolder + string(filepath.Separator)

	index := strings.Index(path, marker)
	if index < 0 {
		return "", false
	}

	return path[:index], true
}

func IsInstallDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "TAGame", "Config"))
	return err == nil && info.IsDir()
}

func guessSource(dir string) string {
	lower := strings.ToLower(dir)

	switch {
	case strings.Contains(lower, "steamapps"):
		return "Steam"
	case strings.Contains(lower, "heroic"):
		return "Heroic"
	case strings.Contains(lower, "epic games"):
		return "Epic Games"
	default:
		return "Detected"
	}
}
