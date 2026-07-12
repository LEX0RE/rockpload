//go:build !windows

package logger

import (
	"io"
	"os"

	"golang.org/x/sys/unix"
)

func redirectStderr(logFile *os.File) error {
	originalStderr := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	os.Stderr = w

	err = unix.Dup2(int(w.Fd()), 2)
	if err != nil {
		return err
	}

	go func() {
		mw := io.MultiWriter(originalStderr, logFile)
		_, _ = io.Copy(mw, r)
	}()

	return nil
}
