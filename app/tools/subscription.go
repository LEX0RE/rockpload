package tools

import (
	"sync"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
)

type Observer func(event string)

type Subscription struct {
	observers []Observer
	lock     sync.Mutex
}

func NewSubscription() *Subscription {
	logger.FuncDebug()
	return &Subscription{
		observers: make([]Observer, 0),
	}
}

func (s *Subscription) Subscribe(obs Observer) {
	logger.FuncDebug()
	s.lock.Lock()
	defer s.lock.Unlock()
	s.observers = append(s.observers, obs)
}

func (s *Subscription) Notify(event string) {
	logger.FuncDebug()
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, obs := range s.observers {
		fyne.Do(func() {
			obs(event)
		})
	}
}