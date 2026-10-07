//go:build windows

package tools

import (
	"os/exec"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"golang.org/x/sys/windows"
)

// CanElevateEditor tells if OpenInTextEditor can ask for administrator rights.
func CanElevateEditor() bool {
	return true
}

// OpenInTextEditor opens the file with Notepad, as administrator (UAC prompt) when elevated is true.
func OpenInTextEditor(path string, elevated bool) error {
	logger.FuncDebug()

	verb := "open"
	if elevated {
		verb = "runas"
	}

	verbPtr, err := windows.UTF16PtrFromString(verb)
	if err != nil {
		return err
	}

	filePtr, err := windows.UTF16PtrFromString("notepad.exe")
	if err != nil {
		return err
	}

	argsPtr, err := windows.UTF16PtrFromString(`"` + path + `"`)
	if err != nil {
		return err
	}

	return windows.ShellExecute(0, verbPtr, filePtr, argsPtr, nil, windows.SW_SHOWNORMAL)
}

// ShowInFolder opens the file explorer with the file selected.
func ShowInFolder(path string) error {
	logger.FuncDebug()

	return exec.Command("explorer.exe", "/select,", path).Start()
}
