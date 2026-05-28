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

	ProcessSupervisorCheckTime = time.Second * 30
)

type RLSupervisor struct {
	*rtime.Looper

	LastRLRunningState bool
	EventManager       *tools.EventManager
	PlayerLogDetected  *LogPlayerInfo

	appConfig      *config.AppConfig
	accountManager *AccountManager

	rlLogsFolder        []string
	lastLogReadPosition int64
	positionLogMutex    sync.Mutex
}

type LogPlayerInfo struct {
	Name      string
	ID        string
	IsPrimary bool
}

func NewRLSupervisor(appConfig *config.AppConfig, accountManager *AccountManager) *RLSupervisor {
	logger.FuncDebug()

	rls := &RLSupervisor{
		LastRLRunningState:  false,
		EventManager:        tools.NewEventManager(),
		PlayerLogDetected:   nil,
		appConfig:           appConfig,
		accountManager:      accountManager,
		rlLogsFolder:        []string{},
		lastLogReadPosition: 0,
		positionLogMutex:    sync.Mutex{},
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

	if running {
		rls.onRLDetected()
	} else {
		rls.onRLClose()
	}

	rls.LastRLRunningState = running
}

func (rls *RLSupervisor) onRLDetected() {
	if !rls.LastRLRunningState {
		rls.EventManager.Notify(EVENT_ON_RL_DETECTED, nil)
	}
}

func (rls *RLSupervisor) onRLPlayerDetected(account *rocket_network.Account, logPlayer *LogPlayerInfo) {
	if account == nil || !rls.LastRLRunningState {
		return
	}

	rls.PlayerLogDetected = logPlayer

	if !account.Player.LastCheckOnline {
		account.Player.LastCheckOnline = true

		rls.EventManager.Notify(EVENT_ON_RL_PLAYER_DETECTED, account)
	}
}

func (rls *RLSupervisor) onRLClose() {
	if rls.LastRLRunningState {
		if rls.PlayerLogDetected != nil {
			if byIDFound := rls.accountManager.GetByPlayerID(rls.PlayerLogDetected.ID); byIDFound != nil {
				byIDFound.Player.LastCheckOnline = false
			} else if byNameFound := rls.accountManager.GetByPlayerName(rls.PlayerLogDetected.Name); byNameFound != nil {
				byNameFound.Player.LastCheckOnline = false
			}
		}

		rls.resetMemory()

		rls.EventManager.Notify(EVENT_ON_RL_CLOSED, nil)
	}
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
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
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
	rls.lastLogReadPosition = 0
	rls.PlayerLogDetected = nil
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

	if stat.Size() < rls.lastLogReadPosition {
		rls.lastLogReadPosition = 0
	}

	_, err = file.Seek(rls.lastLogReadPosition, io.SeekStart)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		rls.analyzeLogLine(line)
	}

	if err := scanner.Err(); err != nil {
		logger.Rlogger.Error("Error while scanning RL logs file", slog.Any("err", err))
	}

	if newPos, err := file.Seek(0, io.SeekCurrent); err == nil {
		rls.lastLogReadPosition = newPos
	}
}

func (rls *RLSupervisor) analyzeLogLine(line string) {
	logger.FuncDebug()

	if strings.Contains(line, "HandleLocalPlayerLoginStatusChanged") && strings.Contains(line, "LS_LoggedIn") {
		rls.updateLoginPlayerInfo(line)
		return
	}

	if strings.Contains(line, "Connect login result") && strings.Contains(line, "EOS_Success") {
		rls.updateEpicLoginPlayerInfo(line)
		return
	}
}

func (rls *RLSupervisor) updateEpicLoginPlayerInfo(logLine string) {
	marker := "'EOS_Success' for player '"

	_, after, ok := strings.Cut(logLine, marker)
	if !ok {
		return
	}

	playerID, _, ok := strings.Cut(after, "'")
	if !ok || playerID == "" {
		return
	}

	logInfo := &LogPlayerInfo{
		ID: "Epic|" + playerID + "|0",
	}

	if byIDFound := rls.accountManager.GetByPlayerID(logInfo.ID); byIDFound != nil {
		rls.onRLPlayerDetected(byIDFound, logInfo)
	}
}

func (rls *RLSupervisor) updateLoginPlayerInfo(logLine string) {
	if !strings.Contains(logLine, "PlayerName=") {
		return
	}

	logInfo := &LogPlayerInfo{}

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

	if byIDFound := rls.accountManager.GetByPlayerID(logInfo.ID); byIDFound != nil {
		rls.onRLPlayerDetected(byIDFound, logInfo)
	}
}
