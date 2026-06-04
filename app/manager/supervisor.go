package manager

import (
	"bufio"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/LEX0RE/rockpload/app/tools/rtime"

	"github.com/fsnotify/fsnotify"
	ps "github.com/mitchellh/go-ps"
)

const (
	EVENT_ON_RL_DETECTED        tools.EventType = "rl_detected"
	EVENT_ON_RL_PLAYER_DETECTED tools.EventType = "rl_player_detected"
	EVENT_ON_RL_CLOSED          tools.EventType = "rl_closed"

	ProcessSupervisorCheckTime = time.Second * 5
	MinFlickerSameState        = 3
)

type RLSupervisor struct {
	*rtime.Looper

	LastRLRunningState bool

	EventManager *tools.EventManager
	RLLogInfo    *RLLogInfo

	appConfig      *config.AppConfig
	accountManager *AccountManager

	rlLogsFolder     []string
	rlLogReader      map[string]*rlLogReader
	positionLogMutex sync.Mutex

	stateFlickerCount int
}

type RLLogInfo struct {
	GameVersion  string
	AccountFound *rocket_network.Account
}

type rlLogReader struct {
	lastReadPosition int64
	isInit           bool
}

type RLPlayerLogInfo struct {
	Name      string
	ID        string
	IsPrimary bool
}

func NewRLSupervisor(appConfig *config.AppConfig, accountManager *AccountManager) *RLSupervisor {
	logger.FuncDebug()

	rls := &RLSupervisor{
		LastRLRunningState: false,
		EventManager:       tools.NewEventManager(),
		RLLogInfo:          &RLLogInfo{},
		appConfig:          appConfig,
		accountManager:     accountManager,
		rlLogsFolder:       []string{},
		rlLogReader:        make(map[string]*rlLogReader),
		positionLogMutex:   sync.Mutex{},
	}

	rls.Looper = rtime.NewLooper(ProcessSupervisorCheckTime, rls.Supervise, rls.Supervise, nil, 0, 0)

	go rls.superviseLog()

	return rls
}

func (rls *RLSupervisor) isRLRunning() bool {
	logger.FuncDebug()

	processes, err := ps.Processes()
	if err != nil {
		return false
	}

	for _, p := range processes {
		rawName := p.Executable()

		if len(rawName) == 0 || (rawName[0] != 'R' && rawName[0] != 'r') {
			continue
		}

		ext := filepath.Ext(rawName)
		nameWithoutExt := strings.TrimSuffix(rawName, ext)
		cleanName := strings.ToLower(nameWithoutExt)

		if cleanName == "rocketleague" || cleanName == "rocket league" {
			return true
		}
	}

	return false
}

func (rls *RLSupervisor) Supervise() {
	logger.FuncDebug()

	running := rls.isRLRunning()

	if running != rls.LastRLRunningState {
		rls.stateFlickerCount++

		if rls.stateFlickerCount >= MinFlickerSameState {
			rls.LastRLRunningState = running
			rls.stateFlickerCount = 0

			if running {
				rls.onRLDetected()
			} else {
				rls.onRLClose()
			}
		}
	} else {
		rls.stateFlickerCount = 0
	}

	if rls.LastRLRunningState {
		rls.checkLogsFallback()
	}
}

func (rls *RLSupervisor) checkLogsFallback() {
	if len(rls.rlLogsFolder) == 0 {
		return
	}

	for _, logFolder := range rls.rlLogsFolder {
		filePath := filepath.Join(logFolder, "Launch.log")
		stat, err := os.Stat(filePath)

		if err == nil {
			rls.positionLogMutex.Lock()
			lastPos := rls.rlLogReader[filePath].lastReadPosition
			rls.positionLogMutex.Unlock()

			if stat.Size() != lastPos {
				logger.Rlogger.Debug("Fallback triggered: Missed log write detected")
				rls.processLogFile(filePath)
			}
		}
	}
}

func (rls *RLSupervisor) onRLDetected() {
	logger.FuncDebug()

	logger.Rlogger.Debug("Rocket League program detected")
	rls.EventManager.Notify(EVENT_ON_RL_DETECTED, nil)
}

func (rls *RLSupervisor) verifyRLPlayerDetected(logInfo *RLPlayerLogInfo) {
	logger.FuncDebug()

	if byIDFound := rls.accountManager.GetByPlayerID(logInfo.ID); byIDFound != nil {
		if rls.RLLogInfo.AccountFound != nil && rls.RLLogInfo.AccountFound.Id() != byIDFound.Id() {
			rls.RLLogInfo.AccountFound.Player.LastCheckOnline = false
		}

		rls.RLLogInfo.AccountFound = byIDFound
	}

	if rls.RLLogInfo.AccountFound == nil || !rls.LastRLRunningState {
		return
	}

	if !rls.RLLogInfo.AccountFound.Player.LastCheckOnline {
		rls.RLLogInfo.AccountFound.Player.LastCheckOnline = true

		logger.Rlogger.Debug("Rocket League Player Account Online detected", slog.Any("Player", rls.RLLogInfo.AccountFound.AccountName()))
		rls.EventManager.Notify(EVENT_ON_RL_PLAYER_DETECTED, rls.RLLogInfo.AccountFound)
	}
}

func (rls *RLSupervisor) onRLClose() {
	logger.FuncDebug()

	if rls.RLLogInfo.AccountFound != nil {
		rls.RLLogInfo.AccountFound.Player.LastCheckOnline = false
	}

	rls.resetMemory()
	rls.EventManager.Notify(EVENT_ON_RL_CLOSED, nil)
}

func (rls *RLSupervisor) superviseLog() {
	logger.FuncDebug()

	rls.resetMemory()
	rls.updateRLLogsFolder()

	if len(rls.rlLogsFolder) == 0 {
		return
	}

	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()

	for _, logFolder := range rls.rlLogsFolder {
		watcher.Add(logFolder)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if !rls.LastRLRunningState {
				continue
			}

			if strings.Contains(event.Name, "Launch.log") {
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Chmod) != 0 {
					rls.processLogFile(event.Name)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}

			logger.Rlogger.Debug("Error got while watching RL logs file", slog.Any("err", err))
			return
		}
	}
}

func (rls *RLSupervisor) resetMemory() {
	logger.FuncDebug()

	for key := range rls.rlLogReader {
		rls.rlLogReader[key].lastReadPosition = -1
		rls.rlLogReader[key].isInit = false
	}

	rls.RLLogInfo = &RLLogInfo{}
}

func (rls *RLSupervisor) updateRLLogsFolder() {
	logger.FuncDebug()

	rls.rlLogsFolder = []string{}

	homeDir, _ := os.UserHomeDir()

	paths := []string{
		// Base Windows
		filepath.Join(homeDir, "Documents", "My Games", "Rocket League", "TAGame", "Logs"),
		// Windows OneDrive
		filepath.Join(homeDir, "OneDrive", "Documents", "My Games", "Rocket League", "TAGame", "Logs"),
		// Linux Steam Proton
		filepath.Join(homeDir, ".local", "share", "Steam", "steamapps", "compatdata", "252950", "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "Rocket League", "TAGame", "Logs"),
		// Linux Heroic Launcher
		filepath.Join(homeDir, "Games", "Heroic", "Prefixes", "RocketLeague", "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "Rocket League", "TAGame", "Logs"),
		// Linux Heroic Launcher From Epic Games
		filepath.Join(homeDir, "Games", "Heroic", "Prefixes", "default", "Epic Games", "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "Rocket League", "TAGame", "Logs"),
		// MacOS
		filepath.Join(homeDir, "Library", "Application Support", "Rocket League", "TAGame", "Logs"),
	}

	for _, path := range paths {
		info, err := os.Stat(path)

		if err == nil && info.IsDir() {
			rls.rlLogsFolder = append(rls.rlLogsFolder, path)
			rls.rlLogReader[filepath.Join(path, "Launch.log")] = &rlLogReader{lastReadPosition: -1, isInit: false}
		}
	}

	if len(rls.rlLogsFolder) == 0 {
		logger.Rlogger.Debug("No logs folder found for Rocket League on the system")
	}
}

func (rls *RLSupervisor) processLogFile(filePath string) {
	logger.FuncDebug()

	rls.positionLogMutex.Lock()
	defer rls.positionLogMutex.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return
	}

	doAnalyze := true
	if rls.rlLogReader[filePath].lastReadPosition < 0 {
		rls.rlLogReader[filePath].lastReadPosition = 0
		doAnalyze = false
	}

	if stat.Size() < rls.rlLogReader[filePath].lastReadPosition {
		rls.rlLogReader[filePath].lastReadPosition = 0
	}

	if !rls.rlLogReader[filePath].isInit && doAnalyze {
		rls.rlLogReader[filePath].isInit = true
		rls.rlLogReader[filePath].lastReadPosition = 0
	}

	_, err = file.Seek(rls.rlLogReader[filePath].lastReadPosition, io.SeekStart)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if doAnalyze {
			rls.analyzeLogLine(line)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Rlogger.Error("Error while scanning RL logs file", slog.Any("err", err))
	}

	if newPos, err := file.Seek(0, io.SeekCurrent); err == nil {
		rls.rlLogReader[filePath].lastReadPosition = newPos
	}
}

func (rls *RLSupervisor) analyzeLogLine(line string) {
	logger.FuncDebug()

	if strings.Contains(line, "GPsyonixBuildID") {
		rls.updateRLStatusInfo(line)
	}

	if strings.Contains(line, "HandleLocalPlayerLoginStatusChanged") && strings.Contains(line, "LS_LoggedIn") {
		rls.updateLoginPlayerInfo(line)
	}

	if strings.Contains(line, "Connect login result") && strings.Contains(line, "EOS_Success") {
		rls.updateEpicLoginPlayerInfo(line)
	}
}

func (rls *RLSupervisor) updateEpicLoginPlayerInfo(logLine string) {
	logger.FuncDebug()

	marker := "'EOS_Success' for player '"

	_, after, ok := strings.Cut(logLine, marker)
	if !ok {
		return
	}

	playerID, _, ok := strings.Cut(after, "'")
	if !ok || playerID == "" {
		return
	}

	logInfo := &RLPlayerLogInfo{
		ID: "Epic|" + playerID + "|0",
	}

	rls.verifyRLPlayerDetected(logInfo)
}

func (rls *RLSupervisor) updateLoginPlayerInfo(logLine string) {
	logger.FuncDebug()

	if !strings.Contains(logLine, "PlayerName=") {
		return
	}

	logInfo := &RLPlayerLogInfo{}

	words := strings.FieldsSeq(logLine)
	for word := range words {
		if after, ok := strings.CutPrefix(word, "PlayerName="); ok {
			logInfo.Name = after
		}

		if after, ok := strings.CutPrefix(word, "PlayerID="); ok {
			parts := strings.Split(after, "|")

			if len(parts) >= 2 {
				logInfo.ID = "Epic|" + parts[1] + "|0"
			}
		}

		if after, ok := strings.CutPrefix(word, "IsPrimary="); ok {
			logInfo.IsPrimary = strings.ToLower(after) == "true"
		}
	}

	if logInfo.ID == "" || !strings.HasPrefix(logInfo.ID, "Epic") {
		return
	}

	rls.verifyRLPlayerDetected(logInfo)
}

func (rls *RLSupervisor) updateRLStatusInfo(logLine string) {
	logger.FuncDebug()

	splitted := strings.Split(logLine, " ")

	if len(splitted) != 3 || splitted[1] != "GPsyonixBuildID" {
		return
	}

	if rls.RLLogInfo.GameVersion != splitted[2] {
		rls.RLLogInfo.GameVersion = splitted[2]

		logger.Rlogger.Debug("Rocket League Game Version detected", slog.Any("Version", rls.RLLogInfo.GameVersion))
	}
}
