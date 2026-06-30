//go:build android

package logger

import "os"

func redirectStderr(f *os.File) error {
	return nil
}
