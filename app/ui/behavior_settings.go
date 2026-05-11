package ui

import (
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type BehaviorSettingPopup struct {
	*Popup
	OptionBox *fyne.Container

	autoUploadCheck        *widget.Check
	uploadOnLaunchCheck    *widget.Check
	noUploadConnectedCheck *widget.Check
	autoStartCheck         *widget.Check
	startInTrayCheck       *widget.Check
	exitInTrayCheck        *widget.Check
}

var BehaviorSettingNameMapping = map[string]string{
	"AutoUpload":        "Auto Upload Replays",
	"UploadOnLaunch":    "Upload Replays on launch",
	"NoUploadConnected": "No Upload if connected (with Unused Account only)",
	"AutoStart":         "Start with system",
	"StartInTray":       "Start in Tray",
	"ExitInTray":        "Exit in System Tray",
}

func NewBehaviorSettingPopup(p *Popup) *BehaviorSettingPopup {
	logger.FuncDebug()

	sp := &BehaviorSettingPopup{Popup: p}

	sp.autoUploadCheck = widget.NewCheck("Auto Upload Replays", func(value bool) {})
	sp.autoUploadCheck.SetChecked(sp.appConfig.BehaviorConfig.AutoUpload.Get())

	sp.uploadOnLaunchCheck = widget.NewCheck("Upload Replays on launch", func(value bool) {})
	sp.uploadOnLaunchCheck.SetChecked(sp.appConfig.BehaviorConfig.UploadOnLaunch.Get())

	sp.noUploadConnectedCheck = widget.NewCheck("No Upload if Connected", func(value bool) {})
	sp.noUploadConnectedCheck.SetChecked(sp.appConfig.BehaviorConfig.NoUploadConnected.Get())

	sp.autoStartCheck = widget.NewCheck("Start with system", func(value bool) {})
	sp.autoStartCheck.SetChecked(sp.appConfig.BehaviorConfig.AutoStart.Get())

	sp.startInTrayCheck = widget.NewCheck("Start in Tray", func(value bool) {})
	sp.startInTrayCheck.SetChecked(sp.appConfig.BehaviorConfig.StartInTray.Get())
	if !p.appConfig.BehaviorConfig.ExitInTray.Get() {
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
	sp.exitInTrayCheck.SetChecked(p.appConfig.BehaviorConfig.ExitInTray.Get())

	sp.OptionBox = container.NewVBox(sp.autoUploadCheck, sp.uploadOnLaunchCheck, sp.noUploadConnectedCheck, sp.autoStartCheck, sp.exitInTrayCheck, sp.startInTrayCheck)

	saveBtn := widget.NewButton("Save", func() {
		currentAutoSave := sp.appConfig.BehaviorConfig.AutoStart.Get()
		if sp.autoStartCheck.Checked != currentAutoSave {
			sp.appConfig.BehaviorConfig.AutoStart.Set(sp.autoStartCheck.Checked)
		}

		currentAutoUpload := sp.appConfig.BehaviorConfig.AutoUpload.Get()
		if sp.autoUploadCheck.Checked != currentAutoUpload {
			sp.appConfig.BehaviorConfig.AutoUpload.Set(sp.autoUploadCheck.Checked)
		}

		currentUploadOnLaunch := sp.appConfig.BehaviorConfig.UploadOnLaunch.Get()
		if sp.uploadOnLaunchCheck.Checked != currentUploadOnLaunch {
			sp.appConfig.BehaviorConfig.UploadOnLaunch.Set(sp.uploadOnLaunchCheck.Checked)
		}

		currentNoUploadConnected := sp.appConfig.BehaviorConfig.NoUploadConnected.Get()
		if sp.noUploadConnectedCheck.Checked != currentNoUploadConnected {
			sp.appConfig.BehaviorConfig.NoUploadConnected.Set(sp.noUploadConnectedCheck.Checked)
		}

		currentExitInTray := sp.appConfig.BehaviorConfig.ExitInTray.Get()
		if sp.exitInTrayCheck.Checked != currentExitInTray {
			sp.appConfig.BehaviorConfig.ExitInTray.Set(sp.exitInTrayCheck.Checked)
		}

		currentStartInTray := sp.appConfig.BehaviorConfig.StartInTray.Get()
		if sp.startInTrayCheck.Checked != currentStartInTray {
			sp.appConfig.BehaviorConfig.StartInTray.Set(sp.startInTrayCheck.Checked)
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

func (sp *BehaviorSettingPopup) Show() {
	logger.FuncDebug()

	sp.autoUploadCheck.SetChecked(sp.appConfig.BehaviorConfig.AutoUpload.Get())
	sp.uploadOnLaunchCheck.SetChecked(sp.appConfig.BehaviorConfig.UploadOnLaunch.Get())
	sp.noUploadConnectedCheck.SetChecked(sp.appConfig.BehaviorConfig.NoUploadConnected.Get())
	sp.autoStartCheck.SetChecked(sp.appConfig.BehaviorConfig.AutoStart.Get())
	sp.startInTrayCheck.SetChecked(sp.appConfig.BehaviorConfig.StartInTray.Get())
	sp.exitInTrayCheck.SetChecked(sp.appConfig.BehaviorConfig.ExitInTray.Get())

	sp.Popup.Show()
}
