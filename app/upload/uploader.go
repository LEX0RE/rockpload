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
	EventUploadStarted   = "upload_started"
	EventUploadCompleted = "upload_completed"
	EventUploadProgress  = "upload_progress"
	EventReplayUploaded  = "replay_uploaded"

	uploadSleep             = time.Second
	autoUploadTickerTime    = time.Minute * 45
	autoUploadJitterMinTime = 0
	autoUploadJitterMaxTime = time.Minute * 15
)

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
	}()
}

func (u *Uploader) upload(ac *config.AccountConfig) {
	logger.FuncDebug()

	if !u.lockInUpload.TryLock() {
		logger.Rlogger.Debug("Duplicate upload request at the same time, skipping")
		return
	}

	defer u.lockInUpload.Unlock()

	websites := u.getWebsites()

	for i, replay := range ac.Player.MatchHistory {
		filePath, err := downloadFile(replay.ReplayUrl)
		if err != nil {
			logger.Rlogger.Error("Download error:", slog.Any("err", err))
			continue
		}
		defer os.Remove(filePath)

		for wi := range websites {
			u.singleUpload(i, wi, websites, ac, filePath)
		}

		os.Remove(filePath)
		u.EventManager.Notify(EventReplayUploaded, nil)
	}

	logger.Rlogger.Info("Upload complete")

	u.EventManager.Notify(EventUploadProgress, float64(-1))
	u.EventManager.Notify(EventUploadCompleted, nil)
}

func (u *Uploader) singleUpload(replayIndex int, websiteIndex int, websites []*Website, ac *config.AccountConfig, filePath string) {
	logger.FuncDebug()

	website := websites[websiteIndex]
	replay := ac.Player.MatchHistory[replayIndex]

	u.EventManager.Notify(EventUploadProgress, float64((replayIndex*len(websites))+websiteIndex)/float64(len(ac.Player.MatchHistory)*len(websites)))

	if !website.config.SendReplay {
		return
	}

	if os.Getenv("FAKE_UPLOAD") == "true" {
		logger.Rlogger.Debug("FAKE UPLOAD - ", slog.Any("Account", ac.Player.PlayerName), slog.Any("matchGUID", replay.Match.MatchGUID), slog.Any("filePath", filePath))
		ac.AddToMatchHistory(replay.Match.MatchGUID)
		return
	}

	uploadCache := LoadUploadedCache(website.config.Name)

	if !uploadCache.index[replay.Match.MatchGUID] {
		logger.Rlogger.Debug("Uploading replay", slog.Any("matchGUID", replay.Match.MatchGUID), slog.Any("filePath", filePath))

		err := website.UploadReplay(filePath)
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

func (u *Uploader) getWebsites() []*Website {
	logger.FuncDebug()

	websites := []*Website{}
	for _, StorageConfig := range u.appConfig.StorageSettings.Get() {
		if !StorageConfig.SendReplay {
			continue
		}

		website := NewWebsite(StorageConfig)
		websites = append(websites, website)
	}

	return websites
}
