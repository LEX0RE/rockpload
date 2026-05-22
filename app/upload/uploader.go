package upload

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	rtime "github.com/LEX0RE/rockpload/app/tools/time"

	"fyne.io/fyne/v2"
)

const (
	EventUploadStarted         = "upload_started"
	EventUploadCompleted       = "upload_completed"
	EventUploadPlayerCompleted = "upload_player_completed"
	EventUploadProgress        = "upload_progress"
	EventReplayUploaded        = "replay_uploaded"

	uploadSleep             = time.Second
	autoUploadTickerTime    = time.Minute * 45
	autoUploadJitterMinTime = 0
	autoUploadJitterMaxTime = time.Minute * 15
)

type UploadStorage interface {
	UploadReplay(filePath string) error
	Ping() error
	GetConfig() *config.StorageConfig
}

type Uploader struct {
	lockInUpload     sync.Mutex
	lockInAutoUpload sync.Mutex
	lockInRunRLAPI   sync.Mutex

	appConfig *config.AppConfig

	autoTicker *rtime.Ticker

	EventManager *tools.EventManager
}

func NewUploader(appConfig *config.AppConfig) *Uploader {
	logger.FuncDebug()

	u := &Uploader{appConfig: appConfig, EventManager: tools.NewEventManager()}

	if appConfig.BehaviorConfig.UploadOnLaunch.Get() {
		u.autoTicker = rtime.NewTicker(autoUploadTickerTime, u.Run, u.Run, nil, autoUploadJitterMinTime, autoUploadJitterMaxTime)
	} else {
		u.autoTicker = rtime.NewTicker(autoUploadTickerTime, u.Run, nil, nil, autoUploadJitterMinTime, autoUploadJitterMaxTime)
	}

	return u
}

func (u *Uploader) Toggle(value bool) {
	logger.FuncDebug()
	if value {
		u.Start()
	} else {
		u.Stop()
	}
}

func (u *Uploader) Start() {
	logger.FuncDebug()

	fyne.Do(func() {
		if !u.lockInAutoUpload.TryLock() {
			logger.Rlogger.Debug("Duplicate auto upload at the same time, skipping")
			return
		}
		defer u.lockInAutoUpload.Unlock()

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic:", r)
			}
		}()

		u.autoTicker.Start()
	})
}

func (u *Uploader) Stop() {
	logger.FuncDebug()
	u.autoTicker.Stop()
}

func (u *Uploader) Run() {
	logger.FuncDebug()

	if !u.lockInRunRLAPI.TryLock() {
		logger.Rlogger.Debug("Duplicate Run at the same time, skipping")
		return
	}
	defer u.lockInRunRLAPI.Unlock()

	u.EventManager.Notify(EventUploadStarted, nil)

	go func() {
		for _, ac := range u.appConfig.AccountSettings.Get() {
			u.upload(ac)
		}
		u.EventManager.Notify(EventUploadCompleted, nil)
	}()
}

func (u *Uploader) upload(ac *config.AccountConfig) {
	logger.FuncDebug()

	if !u.lockInUpload.TryLock() {
		logger.Rlogger.Debug("Duplicate upload request at the same time, skipping")
		return
	}

	defer u.lockInUpload.Unlock()

	storages := u.getStorages()

	for i, replay := range ac.Player.MatchHistory {
		var filePath string
		var isDownloaded bool

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

		for wi := range storages {
			u.singleUpload(i, wi, storages, ac, lazyDownload)
		}

		if isDownloaded && filePath != "" {
			os.Remove(filePath)
		}

		u.EventManager.Notify(EventReplayUploaded, nil)
	}

	logger.Rlogger.Info("Upload complete", slog.Any("Player", ac.Player.PlayerName))

	u.EventManager.Notify(EventUploadProgress, float64(-1))
	u.EventManager.Notify(EventUploadPlayerCompleted, nil)
}

func (u *Uploader) singleUpload(replayIndex int, storageIndex int, storages []UploadStorage, ac *config.AccountConfig, getFilePath func() (string, error)) {
	logger.FuncDebug()

	storage := storages[storageIndex]
	replay := ac.Player.MatchHistory[replayIndex]

	u.EventManager.Notify(EventUploadProgress, float64((replayIndex*len(storages))+storageIndex)/float64(len(ac.Player.MatchHistory)*len(storages)))

	if !storage.GetConfig().SendReplay {
		return
	}

	if os.Getenv("FAKE_UPLOAD") == "true" {
		logger.Rlogger.Debug("FAKE UPLOAD - ", slog.Any("Account", ac.Player.PlayerName), slog.Any("matchGUID", replay.Match.MatchGUID))
		ac.AddToMatchHistory(replay.Match.MatchGUID)
		return
	}

	uploadCache := LoadUploadedCache(storage.GetConfig().Name, len(u.appConfig.AccountSettings.Get()))

	if !uploadCache.index[replay.Match.MatchGUID] {
		filePath, err := getFilePath()
		if err != nil {
			logger.Rlogger.Error("Download error before upload:", slog.Any("err", err))
			return
		}

		logger.Rlogger.Debug("Uploading replay", slog.Any("matchGUID", replay.Match.MatchGUID), slog.Any("filePath", filePath))

		err = storage.UploadReplay(filePath)
		if err != nil {
			logger.Rlogger.Error("Upload error:", slog.Any("err", err))
		} else {
			uploadCache.Add(replay.Match.MatchGUID)
			ac.AddToMatchHistory(replay.Match.MatchGUID)
		}

		time.Sleep(uploadSleep)
	} else {
		logger.Rlogger.Debug("Skipping replay as it was already uploaded")
		ac.AddToMatchHistory(replay.Match.MatchGUID)
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
		case config.WebsiteConfig:
			backend = NewWebsite(storageConfig)
		default:
			backend = NewWebsite(storageConfig)
		}

		storages = append(storages, backend)
	}

	return storages
}
