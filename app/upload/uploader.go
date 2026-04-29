package upload

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	rtime "github.com/LEX0RE/rockpload/app/tools/time"

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

	Player   *rocket_network.Player
	websites []Website

	updateGUI            func()
	updateUploadProgress func(float64)

	autoTicker *rtime.Ticker
}

func NewUploader(player *rocket_network.Player, updateGUI func(), updateUploadProgress func(float64)) *Uploader {
	logger.FuncDebug()

	rockyWebsite := NewRockyWebsite()

	u := &Uploader{
		Player:               player,
		updateGUI:            updateGUI,
		updateUploadProgress: updateUploadProgress,
		websites:             []Website{rockyWebsite},
	}

	u.autoTicker = rtime.NewTicker(autoUploadTickerTime, u.Run, u.Run, nil, autoUploadJitterMinTime, autoUploadJitterMaxTime)

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

	uploadCache := LoadUploadedCache()

	for i, replay := range p.MatchHistory {
		u.updateUploadProgress(float64(i) / float64(len(p.MatchHistory)))

		if !uploadCache.index[replay.Match.MatchGUID] {
			filePath, err := downloadFile(replay.ReplayUrl)
			if err != nil {
				logger.Rlogger.Error("Download error:", slog.Any("err", err))
				continue
			}

			logger.Rlogger.Debug("Uploading replay", slog.Any("matchGUID", replay.Match.MatchGUID), slog.Any("filePath", filePath))

			for _, website := range u.websites {
				err = website.UploadReplay(filePath)
				if err != nil {
					logger.Rlogger.Error("Upload error:", slog.Any("err", err))
				} else {
					uploadCache.Add(replay.Match.MatchGUID)
				}
			}

			os.Remove(filePath)

			time.Sleep(uploadSleep)
		} else {
			logger.Rlogger.Debug("Skipping replay as it was already uploaded")
		}
	}

	logger.Rlogger.Debug("Upload complete")

	u.updateUploadProgress(float64(-1))

	uploadCache.Save()
}
