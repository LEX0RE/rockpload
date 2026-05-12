//go:build windows

package logger

import (
	"os"
	"syscall"
)

func redirectStderr(f *os.File) error {
	err := syscall.SetStdHandle(syscall.STD_ERROR_HANDLE, syscall.Handle(f.Fd()))
	if err != nil {
		return err
	}

	os.Stderr = f
	return nil
}
