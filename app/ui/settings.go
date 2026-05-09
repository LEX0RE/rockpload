package ui

import (
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type SettingPopup struct {
	*Popup
	OptionBox *fyne.Container

	autoStartCheck      *widget.Check
	autoUploadCheck     *widget.Check
	uploadOnLaunchCheck *widget.Check
	startInTrayCheck    *widget.Check
	exitInTrayCheck     *widget.Check
}

func NewSettingPopup(p *Popup) *SettingPopup {
	logger.FuncDebug()

	sp := &SettingPopup{Popup: p}

	sp.autoStartCheck = widget.NewCheck("Start with system", func(value bool) {})
	sp.autoStartCheck.SetChecked(sp.appConfig.AutoStart.Get())

	sp.autoUploadCheck = widget.NewCheck("Auto Upload Replays", func(value bool) {})
	sp.autoUploadCheck.SetChecked(sp.appConfig.AutoUpload.Get())

	sp.uploadOnLaunchCheck = widget.NewCheck("Upload Replays on launch", func(value bool) {})
	sp.uploadOnLaunchCheck.SetChecked(sp.appConfig.UploadOnLaunch.Get())

	sp.startInTrayCheck = widget.NewCheck("Start in Tray", func(value bool) {})
	sp.startInTrayCheck.SetChecked(sp.appConfig.StartInTray.Get())
	if !p.appConfig.ExitInTray.Get() {
		sp.startInTrayCheck.Hide()
	}

	sp.exitInTrayCheck = widget.NewCheck("Exit in System Tray", func(value bool) {
		if value {
			sp.startInTrayCheck.Show()
			if sp.OptionBox != nil {
				sp.OptionBox.Refresh()
			}
		} else {
			sp.startInTrayCheck.Hide()
			if sp.OptionBox != nil {
				sp.OptionBox.Refresh()
			}
		}
	})
	sp.exitInTrayCheck.SetChecked(p.appConfig.ExitInTray.Get())

	sp.OptionBox = container.NewVBox(sp.autoStartCheck, sp.autoUploadCheck, sp.uploadOnLaunchCheck, sp.exitInTrayCheck, sp.startInTrayCheck)

	saveBtn := widget.NewButton("Save", func() {
		currentAutoSave := sp.appConfig.AutoStart.Get()
		if sp.autoStartCheck.Checked != currentAutoSave {
			sp.appConfig.AutoStart.Set(sp.autoStartCheck.Checked)
		}

		currentAutoUpload := sp.appConfig.AutoUpload.Get()
		if sp.autoUploadCheck.Checked != currentAutoUpload {
			sp.appConfig.AutoUpload.Set(sp.autoUploadCheck.Checked)
		}

		currentUploadOnLaunch := sp.appConfig.UploadOnLaunch.Get()
		if sp.uploadOnLaunchCheck.Checked != currentUploadOnLaunch {
			sp.appConfig.UploadOnLaunch.Set(sp.uploadOnLaunchCheck.Checked)
		}

		currentExitInTray := sp.appConfig.ExitInTray.Get()
		if sp.exitInTrayCheck.Checked != currentExitInTray {
			sp.appConfig.ExitInTray.Set(sp.exitInTrayCheck.Checked)
		}

		currentStartInTray := sp.appConfig.StartInTray.Get()
		if sp.startInTrayCheck.Checked != currentStartInTray {
			sp.appConfig.StartInTray.Set(sp.startInTrayCheck.Checked)
		}

		sp.popup.Hide()
	})
	saveBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		sp.OptionBox,
		layout.NewSpacer(),
		saveBtn,
	)

	sp.SetContent(content)

	return sp
}

func (sp *SettingPopup) Show() {
	logger.FuncDebug()

	sp.autoStartCheck.SetChecked(sp.appConfig.AutoStart.Get())
	sp.autoUploadCheck.SetChecked(sp.appConfig.AutoUpload.Get())
	sp.uploadOnLaunchCheck.SetChecked(sp.appConfig.UploadOnLaunch.Get())
	sp.startInTrayCheck.SetChecked(sp.appConfig.StartInTray.Get())
	sp.exitInTrayCheck.SetChecked(sp.appConfig.ExitInTray.Get())

	sp.Popup.Show()
}
