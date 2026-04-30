package ui

import (
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type AnnouncementPopup struct {
	*Popup
}

func NewAnnouncementPopup(p *Popup, message string) *AnnouncementPopup {
	logger.FuncDebug()

	ap := &AnnouncementPopup{Popup: p}

	description := widget.NewLabel(message)
	description.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(description)

	ap.popup.Resize(fyne.NewSize(450, 150))

	ap.SetContent(content)

	return ap
}
