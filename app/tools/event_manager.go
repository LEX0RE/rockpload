package tools

import (
	"sync"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
)

type EventType string

type Listener struct {
	Callback func(data any)
	IsSync   bool
}

type EventManager struct {
	listeners map[EventType][]Listener
	lock      sync.Mutex
}

func NewEventManager() *EventManager {
	logger.FuncDebug()

	return &EventManager{
		listeners: make(map[EventType][]Listener),
	}
}

func (n *EventManager) Subscribe(event EventType, listener Listener) {
	logger.FuncDebug()

	n.lock.Lock()
	defer n.lock.Unlock()

	n.listeners[event] = append(n.listeners[event], listener)
}

func (n *EventManager) MultiSubscribe(events []EventType, listener Listener) {
	logger.FuncDebug()

	n.lock.Lock()
	defer n.lock.Unlock()

	for _, event := range events {
		n.listeners[event] = append(n.listeners[event], listener)
	}
}

func (n *EventManager) Notify(event EventType, data any) {
	logger.FuncDebug()

	n.lock.Lock()
	listeners, ok := n.listeners[event]
	n.lock.Unlock()

	if ok {
		for _, listener := range listeners {
			callback := listener.Callback

			if listener.IsSync {
				callback(data)
			} else {
				fyne.Do(func() {
					callback(data)
				})
			}
		}
	}
}

func (n *EventManager) UnsubscribeAll(event EventType) {
	logger.FuncDebug()

	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.listeners, event)
}
