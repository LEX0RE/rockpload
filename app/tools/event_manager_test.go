package tools

import (
	"testing"
	"time"
)

func TestNotifySyncRunsBeforeReturn(t *testing.T) {
	em := NewEventManager()

	called := false
	em.Subscribe("event", Listener{Mode: ListenerSync, Callback: func(data any) {
		called = data == "data"
	}})

	em.Notify("event", "data")

	if !called {
		t.Fatal("sync listener was not called before Notify returned")
	}
}

func TestNotifyBackgroundDoesNotBlock(t *testing.T) {
	em := NewEventManager()

	release := make(chan struct{})
	done := make(chan any, 1)
	em.Subscribe("event", Listener{Mode: ListenerBackground, Callback: func(data any) {
		<-release
		done <- data
	}})

	returned := make(chan struct{})
	go func() {
		em.Notify("event", "data")
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("Notify blocked on a background listener")
	}

	close(release)

	select {
	case data := <-done:
		if data != "data" {
			t.Fatalf("background listener got %v, want data", data)
		}
	case <-time.After(time.Second):
		t.Fatal("background listener was not called")
	}
}
