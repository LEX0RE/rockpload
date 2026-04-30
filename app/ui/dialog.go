package ui

import (
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Dialog struct {
	window fyne.Window
}

func NewDialog(a fyne.App) *Dialog {
	logger.FuncDebug()

	window := a.NewWindow("Rockpload")

	return &Dialog{window: window}
}

func (d *Dialog) DuplicateApp() {
	logger.FuncDebug()

	d.window.SetTitle("Error")

	dialog.ShowCustomWithoutButtons("Rockpload", widget.NewLabel("The application is already running."), d.window)

	d.window.SetContent(widget.NewLabel(""))

	d.window.Resize(fyne.NewSize(300, 170))
	d.window.Show()
}
