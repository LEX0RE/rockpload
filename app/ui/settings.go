package ui

import (
	"lexore/rockpload/app/config"
	"lexore/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type SettingPopup struct {
	*Popup
	OptionBox 			*fyne.Container

	autoStartCheck 		*widget.Check
	autoUploadCheck 	*widget.Check
	startInTrayCheck 	*widget.Check
	exitInTrayCheck 	*widget.Check
}

func NewSettingPopup(p *Popup) *SettingPopup {
	logger.FuncDebug()

	sp := &SettingPopup{Popup: p}
		
	sp.autoStartCheck = widget.NewCheck("Start with system", func(value bool) {})
	sp.autoStartCheck.SetChecked(sp.appConfig.GetAppConfig(config.AutoStart))

	sp.autoUploadCheck = widget.NewCheck("Auto Upload Replays", func(value bool) {})
	sp.autoUploadCheck.SetChecked(sp.appConfig.GetAppConfig(config.AutoUpload))

	sp.startInTrayCheck = widget.NewCheck("Start in Tray", func(value bool) {})
	sp.startInTrayCheck.SetChecked(sp.appConfig.GetAppConfig(config.StartInTray))
	if (!p.appConfig.GetAppConfig(config.ExitInTray)) {
		sp.startInTrayCheck.Hide()
	}

	sp.exitInTrayCheck = widget.NewCheck("Exit in System Tray", func(value bool) {
		if (value) {
			sp.startInTrayCheck.Show()
			if (sp.OptionBox != nil) {
				sp.OptionBox.Refresh()
			}
		} else {
			sp.startInTrayCheck.Hide()
			if (sp.OptionBox != nil) {
				sp.OptionBox.Refresh()
			}
		}
	})
	sp.exitInTrayCheck.SetChecked(p.appConfig.GetAppConfig(config.ExitInTray))

	sp.OptionBox = container.NewVBox(sp.autoStartCheck, sp.autoUploadCheck, sp.exitInTrayCheck, sp.startInTrayCheck)

    saveBtn := widget.NewButton("Save", func() {
		currentAutoSave := sp.appConfig.GetAppConfig(config.AutoStart)
		if sp.autoStartCheck.Checked != currentAutoSave {
			sp.appConfig.SetAppConfig(config.AutoStart, sp.autoStartCheck.Checked)
		}

		currentAutoUpload := sp.appConfig.GetAppConfig(config.AutoUpload)
		if sp.autoUploadCheck.Checked != currentAutoUpload {
			sp.appConfig.SetAppConfig(config.AutoUpload, sp.autoUploadCheck.Checked)
		}

		currentExitInTray := sp.appConfig.GetAppConfig(config.ExitInTray)
		if sp.exitInTrayCheck.Checked != currentExitInTray {
			sp.appConfig.SetAppConfig(config.ExitInTray, sp.exitInTrayCheck.Checked)
		}

		currentStartInTray := sp.appConfig.GetAppConfig(config.StartInTray)
		if sp.startInTrayCheck.Checked != currentStartInTray {
			sp.appConfig.SetAppConfig(config.StartInTray, sp.startInTrayCheck.Checked)
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

	sp.autoStartCheck.SetChecked(sp.appConfig.GetAppConfig(config.AutoStart))
	sp.autoUploadCheck.SetChecked(sp.appConfig.GetAppConfig(config.AutoUpload))
	sp.startInTrayCheck.SetChecked(sp.appConfig.GetAppConfig(config.StartInTray))
	sp.exitInTrayCheck.SetChecked(sp.appConfig.GetAppConfig(config.ExitInTray))

	sp.Popup.Show()
}