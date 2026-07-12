//go:build windows

package logger

import (
	"io"
	"os"

	"golang.org/x/sys/windows"
)

func redirectStderr(logFile *os.File) error {
	originalStderr := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	os.Stderr = w

	err = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(w.Fd()))
	if err != nil {
		return err
	}

	go func() {
		mw := io.MultiWriter(originalStderr, logFile)
		_, _ = io.Copy(mw, r)
	}()

	return nil
}
