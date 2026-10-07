//go:build android || ios

package tools

import (
	"errors"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

var errEditorNotSupported = errors.New("opening files in an external editor is not supported on mobile")

func CanElevateEditor() bool {
	return false
}

func OpenInTextEditor(path string, elevated bool) error {
	logger.FuncDebug()

	return errEditorNotSupported
}

func ShowInFolder(path string) error {
	logger.FuncDebug()

	return errEditorNotSupported
}
