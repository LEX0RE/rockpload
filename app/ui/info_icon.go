package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	tooltipWidth   = 280
	tooltipPadding = 24
)

func NewInfoIcon(text string) *widget.Button {
	var btn *widget.Button
	var popUp *widget.PopUp

	btn = widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
		if popUp != nil && popUp.Visible() {
			popUp.Hide()
			return
		}

		label := widget.NewLabel(text)
		label.Wrapping = fyne.TextWrapWord
		label.Resize(fyne.NewSize(tooltipWidth, 0))
		textHeight := label.MinSize().Height

		content := container.NewPadded(label)

		c := fyne.CurrentApp().Driver().CanvasForObject(btn)
		if c == nil {
			return
		}

		popUp = widget.NewPopUp(content, c)
		popUp.Resize(fyne.NewSize(tooltipWidth+tooltipPadding, textHeight+tooltipPadding))
		popUp.ShowAtRelativePosition(fyne.NewPos(0, btn.Size().Height), btn)
	})
	btn.Importance = widget.LowImportance

	return btn
}
