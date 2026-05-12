//go:build unix

package logger

import (
	"os"
	"syscall"
)

func redirectStderr(f *os.File) error {
	return syscall.Dup2(int(f.Fd()), 2)
}
