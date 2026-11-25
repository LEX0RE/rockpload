package ui

import (
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func NewStartupScreen(message string) fyne.CanvasObject {
	logger.FuncDebug()

	title := widget.NewLabelWithStyle("Rockpload", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	status := widget.NewLabelWithStyle(message, fyne.TextAlignCenter, fyne.TextStyle{})
	progress := widget.NewProgressBarInfinite()

	return container.NewCenter(container.NewVBox(title, status, progress))
}
