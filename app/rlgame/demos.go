package rlgame

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	DemosFolderName      = "Demos"
	TempReplayFolderName = "Rockpload"
	MaxTempReplays       = 1

	replayExtension  = ".replay"
	partialExtension = ".part"
)

var ErrLogsFolderNotFound = errors.New("rocket league logs folder not found, launch the game once")

func activeLogsFolder() (string, error) {
	logger.FuncDebug()

	latestLogs := ""
	var latestTime time.Time

	for _, logsFolder := range LogsFolders() {
		info, err := os.Stat(filepath.Join(logsFolder, LaunchLogName))
		if err != nil {
			continue
		}

		if latestLogs == "" || info.ModTime().After(latestTime) {
			latestLogs = logsFolder
			latestTime = info.ModTime()
		}
	}

	if latestLogs == "" {
		return "", ErrLogsFolderNotFound
	}

	return latestLogs, nil
}

func ActiveDemosFolder() (string, error) {
	logger.FuncDebug()

	logsFolder, err := activeLogsFolder()
	if err != nil {
		return "", err
	}

	demosFolder := filepath.Join(filepath.Dir(logsFolder), DemosFolderName)
	if err := os.MkdirAll(demosFolder, 0o755); err != nil {
		return "", err
	}

	return demosFolder, nil
}

func TempReplayFolder() (string, error) {
	logger.FuncDebug()

	tempFolder := filepath.Join(os.TempDir(), TempReplayFolderName)

	if runtime.GOOS != "windows" {
		logsFolder, err := activeLogsFolder()
		if err != nil {
			return "", err
		}

		if userFolder, ok := wineUserFolder(logsFolder); ok {
			tempFolder = filepath.Join(userFolder, "AppData", "Local", "Temp", TempReplayFolderName)
		}
	}

	if err := os.MkdirAll(tempFolder, 0o755); err != nil {
		return "", err
	}

	return tempFolder, nil
}

func wineUserFolder(path string) (string, bool) {
	prefix, ok := winePrefix(path)
	if !ok {
		return "", false
	}

	usersFolder := filepath.Join(prefix, wineDriveFolder, "users")

	relative, err := filepath.Rel(usersFolder, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return "", false
	}

	user := strings.Split(relative, string(filepath.Separator))[0]
	if user == "" || user == "." {
		return "", false
	}

	return filepath.Join(usersFolder, user), true
}

func CopyToDemos(filePath string) (string, error) {
	logger.FuncDebug()

	demosFolder, err := ActiveDemosFolder()
	if err != nil {
		return "", err
	}

	return copyReplay(filePath, demosFolder)
}

func CopyToTemp(filePath string) (string, error) {
	logger.FuncDebug()

	source, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer source.Close()

	return SaveTempReplay(filepath.Base(filePath), source)
}

func SaveTempReplay(fileName string, content io.Reader) (string, error) {
	logger.FuncDebug()

	tempFolder, err := TempReplayFolder()
	if err != nil {
		return "", err
	}

	destination := filepath.Join(tempFolder, filepath.Base(fileName))
	if err := writeReplay(destination, content); err != nil {
		return "", err
	}

	pruneTempReplays(tempFolder, MaxTempReplays)

	return destination, nil
}

func ClearTempReplays() {
	logger.FuncDebug()

	tempFolder, err := TempReplayFolder()
	if err != nil {
		return
	}

	pruneTempReplays(tempFolder, 0)
}

func pruneTempReplays(folder string, keep int) {
	logger.FuncDebug()

	entries, err := os.ReadDir(folder)
	if err != nil {
		return
	}

	type replayFile struct {
		path    string
		modTime time.Time
	}

	replays := []replayFile{}
	leftovers := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(folder, entry.Name())

		if !strings.EqualFold(filepath.Ext(entry.Name()), replayExtension) {
			leftovers = append(leftovers, path)
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		replays = append(replays, replayFile{path: path, modTime: info.ModTime()})
	}

	sort.Slice(replays, func(i, j int) bool {
		return replays[i].modTime.After(replays[j].modTime)
	})

	for index := keep; index < len(replays); index++ {
		leftovers = append(leftovers, replays[index].path)
	}

	for _, path := range leftovers {
		if err := os.Remove(path); err != nil {
			logger.Rlogger.Debug("Unable to remove temporary replay, retrying on next cleanup", slog.String("Path", path), slog.Any("err", err))
		}
	}
}

func copyReplay(filePath string, folder string) (string, error) {
	logger.FuncDebug()

	destination := filepath.Join(folder, filepath.Base(filePath))

	if sourceInfo, err := os.Stat(filePath); err != nil {
		return "", err
	} else if destinationInfo, err := os.Stat(destination); err == nil && os.SameFile(sourceInfo, destinationInfo) {
		return destination, nil
	}

	source, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer source.Close()

	return destination, writeReplay(destination, source)
}

func writeReplay(destination string, content io.Reader) error {
	logger.FuncDebug()

	partial := destination + partialExtension

	target, err := os.Create(partial)
	if err != nil {
		return err
	}

	if _, err := io.Copy(target, content); err != nil {
		target.Close()
		os.Remove(partial)
		return err
	}

	if err := target.Close(); err != nil {
		os.Remove(partial)
		return err
	}

	return os.Rename(partial, destination)
}

func GamePath(path string) string {
	path = filepath.ToSlash(path)

	if runtime.GOOS == "windows" {
		return path
	}

	driveMarker := "/" + wineDriveFolder + "/"
	if index := strings.Index(path, driveMarker); index != -1 {
		return "C:/" + path[index+len(driveMarker):]
	}

	return "Z:" + path
}
