package ui

import (
	"lexore/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type UpdatePopup struct {
	*Popup
}

func NewUpdatePopup(p *Popup, newVersion string, onAccept func()) *UpdatePopup {
	logger.FuncDebug()

	versionInfo := ""
	if newVersion != "" {
		versionInfo = " (" + newVersion + ")"
	}

	up := &UpdatePopup{Popup: p}
	
	description := widget.NewLabel("A new update " + versionInfo + " is ready to be install")
	description.Wrapping = fyne.TextWrapWord

	btnLater := widget.NewButton("Later", func() {
		up.Hide()
	})

	btnUpdate := widget.NewButtonWithIcon("Update now", theme.ConfirmIcon(), func() {
		onAccept()
		up.Hide()
	})
	btnUpdate.Importance = widget.HighImportance

	buttonGroup := container.NewHBox(btnLater, btnUpdate)
	buttonsCentered := container.NewCenter(buttonGroup)

	content := container.NewVBox(
		description,
		layout.NewSpacer(),
		buttonsCentered,
	)

	up.popup.Resize(fyne.NewSize(450, 150))

	up.SetContent(content)

	return up
}