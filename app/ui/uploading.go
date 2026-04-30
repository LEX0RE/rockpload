package ui

import (
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type UploadingPopup struct {
	*Popup

	uploadBar *widget.ProgressBar
	isClose   bool
}

func NewUploadingPopup(p *Popup, progress float64) *UploadingPopup {
	logger.FuncDebug()

	up := &UploadingPopup{Popup: p}

	up.onClosed = func() {
		up.isClose = true
		up.Hide()
	}

	up.uploadBar = widget.NewProgressBar()
	up.uploadBar.SetValue(progress)

	content := container.NewVBox(up.uploadBar)

	up.popup.Resize(fyne.NewSize(400, 75))

	up.SetContent(content)

	return up
}

func (up *UploadingPopup) UpdateProgress(progress float64) {
	fyne.Do(func() {
		if progress == -1 {
			up.Hide()
			up.isClose = false
			return
		}

		value := float64(max(0, min(progress, 1)))
		up.uploadBar.SetValue(value)

		if !up.isClose {
			up.Show()
		}
	})
}
