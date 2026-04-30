package upload

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	rtime "github.com/LEX0RE/rockpload/app/tools/time"
	"github.com/dank/rlapi"

	"fyne.io/fyne/v2"
)

const (
	uploadSleep             = time.Second
	autoUploadTickerTime    = time.Minute * 45
	autoUploadJitterMinTime = 0
	autoUploadJitterMaxTime = time.Minute * 15
)

type Uploader struct {
	lockInUpload     sync.Mutex
	lockInAutoUpload sync.Mutex
	lockInRunRLAPI   sync.Mutex

	Player    *rocket_network.Player
	appConfig *config.AppConfig
	websites  []*Website

	updateGUI            func()
	updateUploadProgress func(float64)

	autoTicker    *rtime.Ticker
	HistorySended []string
}

func NewUploader(player *rocket_network.Player, appConfig *config.AppConfig, updateGUI func(), updateUploadProgress func(float64)) *Uploader {
	logger.FuncDebug()

	u := &Uploader{
		Player:               player,
		updateGUI:            updateGUI,
		updateUploadProgress: updateUploadProgress,
		appConfig:            appConfig,
		websites:             []*Website{},
	}

	for _, websiteConfig := range appConfig.GetWebsiteAppConfig() {
		website := NewWebsite(websiteConfig)
		u.websites = append(u.websites, website)
	}

	if appConfig.GetAppConfig(config.UploadOnLaunch) {
		u.autoTicker = rtime.NewTicker(autoUploadTickerTime, u.Run, u.Run, nil, autoUploadJitterMinTime, autoUploadJitterMaxTime)
	} else {
		u.autoTicker = rtime.NewTicker(autoUploadTickerTime, u.Run, nil, nil, autoUploadJitterMinTime, autoUploadJitterMaxTime)
	}

	return u
}

func (u *Uploader) UpdateWebsite() {
	logger.FuncDebug()
	u.websites = []*Website{}

	for _, websiteConfig := range u.appConfig.GetWebsiteAppConfig() {
		website := NewWebsite(websiteConfig)
		u.websites = append(u.websites, website)
	}
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

	err := u.Player.GetInfo()
	if err != nil {
		return
	}

	u.updateGUI()
	go u.upload(u.Player)
	u.updateGUI()

	logger.Rlogger.Info("Upload complete")
}

func (u *Uploader) upload(p *rocket_network.Player) {
	logger.FuncDebug()

	if !u.lockInUpload.TryLock() {
		logger.Rlogger.Debug("Duplicate upload request at the same time, skipping")
		return
	}

	defer u.lockInUpload.Unlock()

	u.HistorySended = []string{}

	addToHistory := func(replay rlapi.MatchEntry) {
		alreadyInHistory := false
		for _, matchGUID := range u.HistorySended {
			if matchGUID == replay.Match.MatchGUID {
				alreadyInHistory = true
			}
		}

		if !alreadyInHistory {
			u.HistorySended = append(u.HistorySended, replay.Match.MatchGUID)
		}
	}

	for i, replay := range p.MatchHistory {
		filePath, err := downloadFile(replay.ReplayUrl)
		if err != nil {
			logger.Rlogger.Error("Download error:", slog.Any("err", err))
			continue
		}
		defer os.Remove(filePath)

		for wi, website := range u.websites {
			u.updateUploadProgress(float64((i*len(u.websites))+wi) / float64(len(p.MatchHistory)*len(u.websites)))

			if !website.config.SendReplay {
				continue
			}

			uploadCache := LoadUploadedCache(website.config.Name)

			if !uploadCache.index[replay.Match.MatchGUID] {
				logger.Rlogger.Debug("Uploading replay", slog.Any("matchGUID", replay.Match.MatchGUID), slog.Any("filePath", filePath))

				err = website.UploadReplay(filePath)
				if err != nil {
					logger.Rlogger.Error("Upload error:", slog.Any("err", err))
				} else {
					uploadCache.Add(replay.Match.MatchGUID)
					addToHistory(replay)
				}

				time.Sleep(uploadSleep)
			} else {
				logger.Rlogger.Debug("Skipping replay as it was already uploaded")
				addToHistory(replay)
			}

			uploadCache.Save()
		}

		os.Remove(filePath)
		u.updateGUI()
	}

	logger.Rlogger.Debug("Upload complete")

	u.updateUploadProgress(float64(-1))
}
