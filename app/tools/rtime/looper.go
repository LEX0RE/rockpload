package rtime

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type Looper struct {
	Ticker *ticker

	tickerMu sync.Mutex
}

func NewLooper(duration time.Duration, onTick func(), onStart func(), onStop func(), jMin time.Duration, jMax time.Duration) *Looper {
	logger.FuncDebug()

	return &Looper{Ticker: newTicker(duration, onTick, onStart, onStop, jMin, jMax), tickerMu: sync.Mutex{}}
}

func (l *Looper) Toggle(value bool) {
	logger.FuncDebug()

	if value {
		l.Start()
	} else {
		l.Stop()
	}
}

func (l *Looper) Start() {
	logger.FuncDebug()

	fyne.Do(func() {
		if !l.tickerMu.TryLock() {
			logger.Rlogger.Debug("Duplicate Start ticker at the same time, skipping")
			return
		}
		defer l.tickerMu.Unlock()

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic:", r)
			}
		}()

		l.Ticker.Start()
	})
}

func (l *Looper) Stop() {
	logger.FuncDebug()

	l.Ticker.Stop()
}
