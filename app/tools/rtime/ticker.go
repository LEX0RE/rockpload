package rtime

import (
	"math/rand"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type ticker struct {
	channel   chan struct{}
	duration  time.Duration
	onTick    func()
	onStart   func()
	onStop    func()
	jitterMin time.Duration
	jitterMax time.Duration
	stopMu    sync.Mutex
	startMu   sync.Mutex
}

func newTicker(duration time.Duration, onTick func(), onStart func(), onStop func(), jMin time.Duration, jMax time.Duration) *ticker {
	logger.FuncDebug()

	if jMin < -duration {
		jMin = 0
	}

	return &ticker{
		duration:  duration,
		onTick:    onTick,
		onStart:   onStart,
		onStop:    onStop,
		jitterMin: jMin,
		jitterMax: jMax,
	}
}

func (t *ticker) Stop() {
	logger.FuncDebug()

	t.stopMu.Lock()
	defer t.stopMu.Unlock()

	if t.channel == nil {
		return
	}

	close(t.channel)
	t.channel = nil
}

func (t *ticker) Start() {
	logger.FuncDebug()

	t.startMu.Lock()
	defer t.startMu.Unlock()

	if t.channel != nil {
		return
	}

	t.channel = make(chan struct{})
	go t.run()
}

func (t *ticker) Set(value bool) {
	logger.FuncDebug()

	if value {
		t.Start()
	} else {
		t.Stop()
	}
}

func (t *ticker) run() {
	logger.FuncDebug()

	ticker := time.NewTimer(t.fuzzyDuration())
	defer ticker.Stop()

	if t.onStart != nil {
		t.onStart()
	}

	if t.onTick == nil {
		t.onStop()
		return
	}

	for {
		select {
		case <-ticker.C:
			if t.onTick != nil {
				t.onTick()
			}

			delay := t.fuzzyDuration()
			ticker.Reset(delay)

		case <-t.channel:
			if t.onStop != nil {
				t.onStop()
			}
			return
		}
	}
}

func (t *ticker) fuzzyDuration() time.Duration {
	logger.FuncDebug()
	delay := t.duration

	if t.jitterMax > t.jitterMin {
		jitter := t.jitterMin + time.Duration(rand.Int63n(int64(t.jitterMax-t.jitterMin)))
		delay += jitter
	}

	return delay
}
