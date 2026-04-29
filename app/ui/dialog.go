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

func (d *Dialog) NewUploadProgress(progress float64) {
	logger.FuncDebug()

	var uploadDialog *dialog.CustomDialog
	var uploadBar *widget.ProgressBar

	fyne.Do(func() {
		if progress == -1 {
			if uploadDialog != nil {
				uploadDialog.Hide()
				uploadDialog = nil
			}
			return
		}

		if uploadDialog == nil {
			uploadBar = widget.NewProgressBar()
			uploadBar.SetValue(0)

			uploadDialog = dialog.NewCustomWithoutButtons("Uploading Replays...", uploadBar, d.window)
			uploadDialog.Show()
		}

		value := float64(max(0, min(progress, 1)))
		uploadBar.SetValue(value)
	})
}
