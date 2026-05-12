//go:build windows

package logger

import (
	"os"

	"golang.org/x/sys/windows"
)

func redirectStderr(f *os.File) error {
	err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd()))
	if err != nil {
		return err
	}

	os.Stderr = f
	return nil
}
