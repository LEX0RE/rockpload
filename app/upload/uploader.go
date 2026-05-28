package upload

import (
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

	uploadSleep             = time.Second // Ballchasing PATCH is 2 req/sec max, so don't go below that
	autoUploadTickerTime    = time.Minute * 45
	autoUploadJitterMinTime = 0
	autoUploadJitterMaxTime = time.Minute * 15
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
	Ping() error
	GetConfig() *config.StorageConfig
}

type Uploader struct {
	*rtime.Looper

	lockInUpload     sync.Mutex
	lockInAutoUpload sync.Mutex
	lockInRunRLAPI   sync.Mutex

	appConfig      *config.AppConfig
	accountManager *manager.AccountManager

	EventManager *tools.EventManager
}

func NewUploader(appConfig *config.AppConfig, accountManager *manager.AccountManager) *Uploader {
	logger.FuncDebug()

	u := &Uploader{appConfig: appConfig, EventManager: tools.NewEventManager(), accountManager: accountManager}

	if appConfig.BehaviorConfig.UploadOnLaunch.Get() {
		u.Looper = rtime.NewLooper(autoUploadTickerTime, u.Run, u.Run, nil, autoUploadJitterMinTime, autoUploadJitterMaxTime)
	} else {
		u.Looper = rtime.NewLooper(autoUploadTickerTime, u.Run, nil, nil, autoUploadJitterMinTime, autoUploadJitterMaxTime)
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

			logger.Rlogger.Debug("Downloading file for the first time in this loop", slog.Any("url", replay.ReplayUrl))
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

	u.EventManager.Notify(EventUploadProgress, currentProgress/maxProgress)

	if !storage.GetConfig().SendReplay {
		return
	}

	uploadCache := LoadUploadedCache(storage.GetConfig().Name, len(u.appConfig.AccountSettings.Get()))
	if err := uploadCache.Touch(); err != nil {
		logger.Rlogger.Error("Error while updating the upload cache file:", slog.Any("err", err))
		return
	}

	if os.Getenv("FAKE_UPLOAD") == "true" {
		logger.Rlogger.Debug("FAKE UPLOAD - ", slog.Any("Account", ac.AccountName()), slog.Any("Storage", storage.GetConfig().Name), slog.Any("matchGUID", match.Match.MatchGUID))
		ac.AddToMatchHistory(match.Match.MatchGUID)
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

		err = storage.UploadReplay(filePath, ReplayUpload{
			PlayerName: ac.Player.PlayerName,
			PlayerID:   playerID,
			Replay:     match,
		})
		if err != nil {
			logger.Rlogger.Error("Upload error:", slog.Any("err", err))
		} else {
			uploadCache.Add(match.Match.MatchGUID)
			ac.AddToMatchHistory(match.Match.MatchGUID)
		}

		time.Sleep(uploadSleep)
	} else {
		logger.Rlogger.Debug("Skipping replay as it was already uploaded", slog.Any("Storage", storage.GetConfig().Name))
		ac.AddToMatchHistory(match.Match.MatchGUID)
	}

	uploadCache.Save()
}

func (u *Uploader) getStorages() []UploadStorage {
	logger.FuncDebug()

	var storages []UploadStorage

	for _, storageConfig := range u.appConfig.StorageSettings.Get() {
		if !storageConfig.SendReplay {
			continue
		}

		var backend UploadStorage

		switch storageConfig.StorageType {
		case config.FileSystemConfig:
			backend = NewFileSystem(storageConfig)
		default:
			fallthrough
		case config.WebsiteConfig:
			backend = NewWebsite(storageConfig)
		}

		storages = append(storages, backend)
	}

	return storages
}
