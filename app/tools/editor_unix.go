//go:build !windows && !android && !ios

package tools

import (
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

// CanElevateEditor tells if OpenInTextEditor can ask for administrator rights.
func CanElevateEditor() bool {
	return false
}

// OpenInTextEditor opens the file with the default text editor of the system.
func OpenInTextEditor(path string, elevated bool) error {
	logger.FuncDebug()

	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-t", path).Start()
	}

	return exec.Command("xdg-open", path).Start()
}

// ShowInFolder opens the file manager on the folder containing the file.
func ShowInFolder(path string) error {
	logger.FuncDebug()

	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-R", path).Start()
	}

	return exec.Command("xdg-open", filepath.Dir(path)).Start()
}
