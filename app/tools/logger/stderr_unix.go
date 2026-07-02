//go:build !windows

package logger

import (
	"os"

	"golang.org/x/sys/unix"
)

func redirectStderr(f *os.File) error {
	return unix.Dup3(int(f.Fd()), 2, 0)
}
