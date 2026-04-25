package time

import (
	"math/rand"
	"sync"
	"time"

	"lexore/rockpload/app/tools/logger"
)

type Ticker struct {
    channel chan struct{}
    duration time.Duration
    onTick func()
    onStart func()
    onStop func()
	jitterMin time.Duration
	jitterMax time.Duration
	stopMu sync.Mutex
	startMu sync.Mutex
}

func NewTicker(duration time.Duration, onTick func(), onStart func(), onStop func(), jMin time.Duration, jMax time.Duration) *Ticker {
	logger.FuncDebug()

	if jMin < -duration {
		jMin = 0
	}

	return &Ticker{
		duration: duration,
		onTick: onTick,
		onStart: onStart,
		onStop: onStop,
		jitterMin: jMin,
		jitterMax: jMax,
	}
}

func (t *Ticker) Stop() {
	logger.FuncDebug()

	t.stopMu.Lock()
	defer t.stopMu.Unlock()

	if t.channel == nil {
		return
	}

	close(t.channel)
	t.channel = nil
}

func (t *Ticker) Start() {
	logger.FuncDebug()

	t.startMu.Lock()
	defer t.startMu.Unlock()

	if (t.channel != nil) {
		return
	}

	t.channel = make(chan struct{})
	go t.run()
}

func (t *Ticker) Set(value bool) {
	logger.FuncDebug()

	if (value) {
		t.Start()
	} else {
		t.Stop()
	}
}

func (t *Ticker) run() {
	logger.FuncDebug()

	ticker := time.NewTimer(t.fuzzyDuration())
	defer ticker.Stop()

	if (t.onStart != nil) {
		t.onStart()
	}

    for {
        select {
        case <-ticker.C:
			if (t.onTick != nil) {
				t.onTick()
			}

			delay := t.fuzzyDuration()
			ticker.Reset(delay)

        case <-t.channel:
			if (t.onStop != nil) {
				t.onStop()
			}
            return
        }
    }
}

func (t *Ticker) fuzzyDuration() time.Duration {
	logger.FuncDebug()
	delay := t.duration

	if t.jitterMax > t.jitterMin {
		jitter := t.jitterMin + time.Duration(rand.Int63n(int64(t.jitterMax-t.jitterMin)))
		delay += jitter
	}

	return delay
}