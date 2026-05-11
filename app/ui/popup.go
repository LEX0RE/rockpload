package ui

import (
	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/manager"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Popup struct {
	popup        *widget.PopUp
	content      *fyne.Container
	parentWindow fyne.Window
	onClosed     func()

	appConfig      *config.AppConfig
	accountManager *manager.AccountManager
}

func NewPopup(title string, parentWindow fyne.Window, appConfig *config.AppConfig, accountManager *manager.AccountManager) *Popup {
	logger.FuncDebug()

	p := &Popup{
		content:        container.NewStack(),
		parentWindow:   parentWindow,
		appConfig:      appConfig,
		accountManager: accountManager,
	}

	p.onClosed = func() {
		p.popup.Hide()
	}

	closeBtn := widget.NewButtonWithIcon("", theme.WindowCloseIcon(), func() {
		p.onClosed()
	})
	closeBtn.Importance = widget.LowImportance

	header := container.NewHBox(
		widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		closeBtn,
	)

	topSection := container.NewVBox(header, widget.NewSeparator())
	content := container.NewBorder(topSection, nil, nil, nil, p.content)

	p.popup = widget.NewModalPopUp(content, parentWindow.Canvas())

	return p
}

func (p *Popup) SetContent(content fyne.CanvasObject) {
	logger.FuncDebug()
	p.content.RemoveAll()
	p.content.Add(content)
}

func (p *Popup) Show() {
	logger.FuncDebug()
	p.popup.Show()
}

func (p *Popup) Hide() {
	logger.FuncDebug()
	p.popup.Hide()
}
