package upload

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/manager"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/LEX0RE/rockpload/app/tools/rtime"
	"github.com/dank/rlapi"
)

const (
	EventUploadStarted         tools.EventType = "upload_started"
	EventUploadCompleted       tools.EventType = "upload_completed"
	EventUploadPlayerCompleted tools.EventType = "upload_player_completed"
	EventUploadProgress        tools.EventType = "upload_progress"
	EventReplayUploaded        tools.EventType = "replay_uploaded"
	EventSingleUploadCompleted tools.EventType = "single_upload_completed"
	EventMatchUploadCompleted  tools.EventType = "match_upload_completed"

	uploadSleep = time.Second // Ballchasing PATCH is 2 req/sec max, so don't go below that
)

type uploadCtx struct {
	matchIndex            int
	matchList             []rlapi.MatchEntry
	accountIndex          int
	accountList           []*rocket_network.Account
	storageIndex          int
	storageList           []UploadStorage
	currentAllReplayIndex int
	allReplaysLength      int
}

type UploadStorage interface {
	UploadReplay(filePath string, replayUpload ReplayUpload) error
	UploadLive(liveStats *rocket_network.LiveStats) error
	Ping() error
	GetConfig() *config.StorageConfig
}

type Uploader struct {
	*rtime.Looper

	lockInUpload       sync.Mutex
	lockInLiveUpload   sync.Mutex
	lockInAutoUpload   sync.Mutex
	lockInRunRLAPI     sync.Mutex
	lockInSingleUpload sync.Mutex

	version        string
	appConfig      *config.AppConfig
	accountManager *manager.AccountManager

	EventManager *tools.EventManager
}

func NewUploader(version string, appConfig *config.AppConfig, accountManager *manager.AccountManager) *Uploader {
	logger.FuncDebug()

	u := &Uploader{version: version, appConfig: appConfig, EventManager: tools.NewEventManager(), accountManager: accountManager}

	autoUploadTime := appConfig.BehaviorConfig.AutoUploadTime.Get()
	autoUploadRandomTime := appConfig.BehaviorConfig.AutoUploadRandomTime.Get()

	if appConfig.BehaviorConfig.UploadOnLaunch.Get() {
		u.Looper = rtime.NewLooper(autoUploadTime, u.Run, u.Run, nil, 0, autoUploadRandomTime)
	} else {
		u.Looper = rtime.NewLooper(autoUploadTime, u.Run, nil, nil, 0, autoUploadRandomTime)
	}

	return u
}

func (u *Uploader) Run() {
	logger.FuncDebug()

	if !u.lockInRunRLAPI.TryLock() {
		logger.Rlogger.Debug("Duplicate Run at the same time, skipping")
		return
	}
	defer u.lockInRunRLAPI.Unlock()

	logger.Rlogger.Info("Start Upload Replays Process")

	u.EventManager.Notify(EventUploadStarted, nil)
	u.EventManager.Notify(EventUploadProgress, 0)

	go func() {
		accountList := u.accountManager.GetAll()

		allReplaysLength := 0
		for _, ac := range accountList {
			allReplaysLength += len(ac.Player.MatchHistory)
		}

		uploadCtx := &uploadCtx{
			matchIndex:            -1,
			matchList:             nil,
			accountIndex:          -1,
			accountList:           accountList,
			currentAllReplayIndex: 0,
			allReplaysLength:      allReplaysLength,
		}

		for i := range accountList {
			uploadCtx.accountIndex = i
			u.upload(uploadCtx)
		}

		u.EventManager.Notify(EventUploadCompleted, nil)
	}()
}

func (u *Uploader) UploadLiveStats(liveStats *rocket_network.LiveStats) {
	logger.FuncDebug()

	if !u.lockInLiveUpload.TryLock() {
		logger.Rlogger.Debug("Duplicate live stat upload request at the same time, skipping")
		return
	}

	defer u.lockInLiveUpload.Unlock()

	logger.Rlogger.Info("Start Upload Live Stats Process")

	if !u.appConfig.BehaviorConfig.SendLiveStat.Get() {
		return
	}

	for _, storage := range u.getStorages() {
		if storage.GetConfig().LiveStyle != config.LiveDisabled {
			storage.UploadLive(liveStats)
		}
	}
}

func (u *Uploader) UploadLocalFile(filePath string) {
	logger.FuncDebug()

	if !u.lockInSingleUpload.TryLock() {
		logger.Rlogger.Debug("Duplicate single replay upload request at the same time, skipping")
		u.EventManager.Notify(EventSingleUploadCompleted, fmt.Errorf("an upload is already in progress"))
		return
	}

	go func() {
		defer u.lockInSingleUpload.Unlock()
		defer os.Remove(filePath)

		logger.Rlogger.Info("Start Single Replay Upload", slog.Any("filePath", filePath))

		storages := u.getStorages()
		if len(storages) == 0 {
			u.EventManager.Notify(EventSingleUploadCompleted, fmt.Errorf("no storage is configured to receive replays"))
			return
		}

		var uploadErrors []error
		for _, storage := range storages {
			if err := storage.UploadReplay(filePath, ReplayUpload{}); err != nil {
				logger.Rlogger.Error("Single replay upload error:", slog.Any("storage", storage.GetConfig().Name), slog.Any("err", err))
				uploadErrors = append(uploadErrors, fmt.Errorf("%s: %w", storage.GetConfig().Name, err))
			}
		}

		u.EventManager.Notify(EventSingleUploadCompleted, errors.Join(uploadErrors...))
	}()
}

func (u *Uploader) UploadMatchEntry(ac *rocket_network.Account, match rlapi.MatchEntry) {
	logger.FuncDebug()

	if !u.lockInSingleUpload.TryLock() {
		logger.Rlogger.Debug("Duplicate single replay upload request at the same time, skipping")
		u.EventManager.Notify(EventMatchUploadCompleted, fmt.Errorf("an upload is already in progress"))
		return
	}

	go func() {
		defer u.lockInSingleUpload.Unlock()

		logger.Rlogger.Info("Start Single Match Upload", slog.Any("matchGUID", match.Match.MatchGUID))

		storages := u.getStorages()
		if len(storages) == 0 {
			u.EventManager.Notify(EventMatchUploadCompleted, fmt.Errorf("no storage is configured to receive replays"))
			return
		}

		filePath, err := downloadFile(match.ReplayUrl)
		if err != nil {
			u.EventManager.Notify(EventMatchUploadCompleted, fmt.Errorf("failed to download replay: %w", err))
			return
		}
		defer os.Remove(filePath)

		playerID := ""
		if ac.Player.PlayerID != nil {
			playerID = ac.Player.PlayerID.String()
		}

		replayUpload := ReplayUpload{PlayerName: ac.Player.PlayerName, PlayerID: playerID, Replay: match}

		var uploadErrors []error

		for _, storage := range storages {
			if err := storage.UploadReplay(filePath, replayUpload); err != nil {
				logger.Rlogger.Error("Single match upload error:", slog.Any("storage", storage.GetConfig().Name), slog.Any("err", err))
				uploadErrors = append(uploadErrors, fmt.Errorf("%s: %w", storage.GetConfig().Name, err))
				continue
			}

			LoadUploadedCache(storage.GetConfig().Name, len(u.appConfig.AccountSettings.Get())).Add(match.Match.MatchGUID)
		}

		u.EventManager.Notify(EventMatchUploadCompleted, errors.Join(uploadErrors...))
	}()
}

func (u *Uploader) upload(uploadCtx *uploadCtx) {
	logger.FuncDebug()

	if !u.lockInUpload.TryLock() {
		logger.Rlogger.Debug("Duplicate upload request at the same time, skipping")
		return
	}

	defer u.lockInUpload.Unlock()

	uploadCtx.storageList = u.getStorages()

	ac := uploadCtx.accountList[uploadCtx.accountIndex]

	matchHistoryOrdered := []rlapi.MatchEntry{}
	for _, replay := range ac.Player.MatchHistory {
		matchHistoryOrdered = append(matchHistoryOrdered, replay)
	}

	sort.Slice(matchHistoryOrdered, func(i, j int) bool {
		if u.appConfig.BehaviorConfig.UploadOlderFirst.Get() {
			return matchHistoryOrdered[i].Match.RecordStartTimestamp < matchHistoryOrdered[j].Match.RecordStartTimestamp
		} else {
			return matchHistoryOrdered[i].Match.RecordStartTimestamp > matchHistoryOrdered[j].Match.RecordStartTimestamp
		}
	})

	uploadCtx.matchList = matchHistoryOrdered

	for i, replay := range uploadCtx.matchList {
		var filePath string
		var isDownloaded bool

		uploadCtx.matchIndex = i
		uploadCtx.currentAllReplayIndex += 1

		lazyDownload := func() (string, error) {
			if isDownloaded {
				return filePath, nil
			}

			logger.Rlogger.Debug("Downloading file", slog.Any("url", replay.ReplayUrl))
			path, err := downloadFile(replay.ReplayUrl)
			if err != nil {
				return "", err
			}

			filePath = path
			isDownloaded = true
			return filePath, nil
		}

		for wi := range uploadCtx.storageList {
			uploadCtx.storageIndex = wi
			u.singleUpload(uploadCtx, lazyDownload)
		}

		if isDownloaded && filePath != "" {
			os.Remove(filePath)
		}

		u.EventManager.Notify(EventReplayUploaded, nil)
	}

	logger.Rlogger.Info("Upload complete", slog.Any("Player", ac.AccountName()))

	u.EventManager.Notify(EventUploadProgress, float64(-1))
	u.EventManager.Notify(EventUploadPlayerCompleted, nil)
}

func (u *Uploader) singleUpload(uploadCtx *uploadCtx, getFilePath func() (string, error)) {
	logger.FuncDebug()

	storage := uploadCtx.storageList[uploadCtx.storageIndex]
	ac := uploadCtx.accountList[uploadCtx.accountIndex]
	match := uploadCtx.matchList[uploadCtx.matchIndex]

	currentProgress := float64((uploadCtx.currentAllReplayIndex * len(uploadCtx.storageList)) + uploadCtx.storageIndex)
	maxProgress := float64(uploadCtx.allReplaysLength * len(uploadCtx.storageList))

	u.EventManager.Notify(EventUploadProgress, max(0, currentProgress-1)/maxProgress)

	if storage.GetConfig().UploadStyle == config.UploadDisabled {
		return
	}

	uploadCache := LoadUploadedCache(storage.GetConfig().Name, len(u.appConfig.AccountSettings.Get()))
	if err := uploadCache.Touch(); err != nil {
		logger.Rlogger.Error("Error while updating the upload cache file:", slog.Any("err", err))
		return
	}

	if os.Getenv("FAKE_UPLOAD") == "true" {
		logger.Rlogger.Debug("FAKE UPLOAD - ", slog.Any("Account", ac.AccountName()), slog.Any("Storage", storage.GetConfig().Name), slog.Any("matchGUID", match.Match.MatchGUID))
		uploadCache.Add(match.Match.MatchGUID)
		return
	}

	if !uploadCache.index[match.Match.MatchGUID] {
		filePath, err := getFilePath()
		if err != nil {
			logger.Rlogger.Error("Download error before upload:", slog.Any("err", err))
			return
		}

		logger.Rlogger.Debug("Uploading replay", slog.Any("matchGUID", match.Match.MatchGUID), slog.Any("filePath", filePath))

		playerID := ""
		if ac.Player.PlayerID != nil {
			playerID = ac.Player.PlayerID.String()
		}

		err = storage.UploadReplay(filePath, ReplayUpload{PlayerName: ac.Player.PlayerName, PlayerID: playerID, Replay: match})
		if err != nil {
			logger.Rlogger.Error("Upload error:", slog.Any("err", err))
		} else {
			uploadCache.Add(match.Match.MatchGUID)
		}

		time.Sleep(uploadSleep)
	} else {
		logger.Rlogger.Debug("Skipping replay as it was already uploaded", slog.Any("Storage", storage.GetConfig().Name))
	}

	uploadCache.Save()
}

func (u *Uploader) getStorages() []UploadStorage {
	logger.FuncDebug()

	var storages []UploadStorage

	for _, storageConfig := range u.appConfig.StorageSettings.Get() {
		if storageConfig.UploadStyle == config.UploadDisabled {
			continue
		}

		var backend UploadStorage

		switch storageConfig.UploadStyle {
		case config.LocalFileCopy:
			backend = NewFileSystem(storageConfig)
		default:
			fallthrough
		case config.MultipartUpload, config.PresignedSessionUpload:
			backend = NewWebsite(storageConfig, u.version)
		}

		storages = append(storages, backend)
	}

	return storages
}
