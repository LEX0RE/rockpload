package ui

import (
	"fmt"
	"strings"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type StorageInfoContainer struct {
	titleLabel        *widget.Label
	urlEntry          *widget.Entry
	storageTypeSelect *widget.Select
	uriParamsEntry    *widget.Entry
	needTokenCheck    *widget.Check
	tokenEntry        *widget.Entry
	sendPingCheck     *widget.Check
	pingPathEntry     *widget.Entry
	sendReplayCheck   *widget.Check
	replayPathEntry   *widget.Entry
	templateNameEntry *widget.Entry
	sendLiveCheck     *widget.Check
	livePathEntry     *widget.Entry

	urlForm          *widget.FormItem
	storageTypeForm  *widget.FormItem
	uriParamsForm    *widget.FormItem
	needTokenForm    *widget.FormItem
	tokenForm        *widget.FormItem
	sendPingForm     *widget.FormItem
	pingPathForm     *widget.FormItem
	sendReplayForm   *widget.FormItem
	replayPathForm   *widget.FormItem
	templateNameForm *widget.FormItem
	sendLiveForm     *widget.FormItem
	livePathForm     *widget.FormItem
}

type StorageSettingsPopup struct {
	*Popup

	currentWebsite config.StorageConfig

	infoContainer StorageInfoContainer
	btnDelete     *widget.Button
	btnSave       *widget.Button
	list          *widget.List
	split         *container.Split
	leftPanel     *fyne.Container
	detailPanel   *fyne.Container
	editForm      *widget.Form
}

func NewStorageSettingsPopup(p *Popup) *StorageSettingsPopup {
	logger.FuncDebug()

	wsp := &StorageSettingsPopup{Popup: p, infoContainer: StorageInfoContainer{}}

	wsp.btnSave = widget.NewButtonWithIcon("Save Changes", theme.DocumentSaveIcon(), wsp.onSaveBtn)
	wsp.btnSave.Importance = widget.HighImportance
	wsp.btnSave.Disable()

	wsp.btnDelete = widget.NewButtonWithIcon("Delete this storage", theme.DeleteIcon(), wsp.onDeleteStorageBtn)
	wsp.btnDelete.Importance = widget.DangerImportance
	wsp.btnDelete.Disable()

	wsp.editForm = wsp.createInfoContainer()
	scrollableForm := container.NewVScroll(wsp.editForm)
	buttonsBox := container.NewHBox(layout.NewSpacer(), wsp.btnSave, wsp.btnDelete)

	wsp.detailPanel = container.NewBorder(
		container.NewVBox(wsp.infoContainer.titleLabel, widget.NewSeparator()),
		buttonsBox,
		nil, nil,
		scrollableForm,
	)

	wsp.list = widget.NewList(
		func() int { return len(wsp.appConfig.StorageSettings.Get()) },
		func() fyne.CanvasObject {
			return widget.NewRichTextWithText("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			rt := o.(*widget.RichText)
			site := wsp.appConfig.StorageSettings.Get()[i]

			style := widget.RichTextStyleInline
			if !site.SendReplay {
				style.ColorName = theme.ColorNameError
			} else {
				style.ColorName = theme.ColorNameForeground
			}

			rt.Segments = []widget.RichTextSegment{
				&widget.TextSegment{
					Text:  site.Name,
					Style: style,
				},
			}
			rt.Refresh()
		},
	)

	wsp.list.OnSelected = wsp.onSelected
	wsp.list.OnUnselected = wsp.onUnselected

	btnAdd := widget.NewButtonWithIcon("Add a storage", theme.ContentAddIcon(), wsp.onAddStorageBtn)

	wsp.leftPanel = container.NewBorder(nil, btnAdd, nil, nil, wsp.list)
	wsp.split = container.NewHSplit(wsp.leftPanel, wsp.detailPanel)
	wsp.split.Offset = 0.3

	wsp.SetContent(wsp.split)

	wsp.popup.Resize(fyne.NewSize(650, 350))

	wsp.onSelected(wsp.appConfig.BehaviorConfig.SelectedStorageId.Get())

	return wsp
}

func (wsp *StorageSettingsPopup) Show() {
	logger.FuncDebug()

	wsp.list.Refresh()
	wsp.reload()

	wsp.Popup.Show()
}

func (wsp *StorageSettingsPopup) createInfoContainer() *widget.Form {
	logger.FuncDebug()

	wsp.infoContainer.titleLabel = widget.NewLabelWithStyle("Select a storage", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	wsp.infoContainer.urlEntry = widget.NewEntry()

	wsp.infoContainer.storageTypeSelect = widget.NewSelect([]string{"Website", "File System"}, func(v string) {
		switch v {
		case "File System":
			wsp.currentWebsite.StorageType = config.FileSystemConfig
		default:
			fallthrough
		case "Website":
			wsp.currentWebsite.StorageType = config.WebsiteConfig
		}

		wsp.reload()
	})

	wsp.infoContainer.uriParamsEntry = widget.NewMultiLineEntry()
	wsp.infoContainer.uriParamsEntry.SetPlaceHolder("")

	wsp.infoContainer.needTokenCheck = widget.NewCheck("", func(v bool) {
		wsp.currentWebsite.NeedToken = v
		wsp.reload()
	})

	wsp.infoContainer.tokenEntry = widget.NewPasswordEntry()
	wsp.infoContainer.tokenEntry.SetPlaceHolder("")

	wsp.infoContainer.sendPingCheck = widget.NewCheck("", func(v bool) {
		wsp.currentWebsite.SendPing = v
		wsp.reload()
	})
	wsp.infoContainer.pingPathEntry = widget.NewEntry()

	wsp.infoContainer.sendReplayCheck = widget.NewCheck("", func(v bool) {
		wsp.currentWebsite.SendReplay = v
		wsp.reload()
	})

	wsp.infoContainer.sendLiveCheck = widget.NewCheck("", func(v bool) {
		wsp.currentWebsite.SendLive = v
		wsp.reload()
	})
	wsp.infoContainer.livePathEntry = widget.NewEntry()

	wsp.infoContainer.replayPathEntry = widget.NewEntry()
	wsp.infoContainer.templateNameEntry = widget.NewEntry()

	wsp.infoContainer.urlForm = widget.NewFormItem("URL", wsp.infoContainer.urlEntry)
	wsp.infoContainer.storageTypeForm = widget.NewFormItem("Storage Type", wsp.infoContainer.storageTypeSelect)
	wsp.infoContainer.uriParamsForm = widget.NewFormItem("URI Params", wsp.infoContainer.uriParamsEntry)
	wsp.infoContainer.needTokenForm = widget.NewFormItem("Need Token", wsp.infoContainer.needTokenCheck)
	wsp.infoContainer.tokenForm = widget.NewFormItem("Token", wsp.infoContainer.tokenEntry)
	wsp.infoContainer.sendPingForm = widget.NewFormItem("Send Ping", wsp.infoContainer.sendPingCheck)
	wsp.infoContainer.pingPathForm = widget.NewFormItem("Ping Path", wsp.infoContainer.pingPathEntry)
	wsp.infoContainer.sendReplayForm = widget.NewFormItem("Send Replay", wsp.infoContainer.sendReplayCheck)
	wsp.infoContainer.replayPathForm = widget.NewFormItem("Replay Path", wsp.infoContainer.replayPathEntry)
	wsp.infoContainer.templateNameForm = widget.NewFormItem("Replay Name Template", wsp.infoContainer.templateNameEntry)
	wsp.infoContainer.sendLiveForm = widget.NewFormItem("Send Live", wsp.infoContainer.sendLiveCheck)
	wsp.infoContainer.livePathForm = widget.NewFormItem("Live Path", wsp.infoContainer.livePathEntry)

	return widget.NewForm(
		wsp.infoContainer.sendReplayForm,
		wsp.infoContainer.replayPathForm,
		wsp.infoContainer.templateNameForm,
		wsp.infoContainer.urlForm,
		wsp.infoContainer.storageTypeForm,
		wsp.infoContainer.needTokenForm,
		wsp.infoContainer.tokenForm,
		wsp.infoContainer.sendPingForm,
		wsp.infoContainer.pingPathForm,
		wsp.infoContainer.uriParamsForm,
		wsp.infoContainer.sendLiveForm,
		wsp.infoContainer.livePathForm,
	)
}

func (wsp *StorageSettingsPopup) onSelected(id widget.ListItemID) {
	logger.FuncDebug()

	if id < 0 {
		wsp.onUnselected(-1)
		return
	}

	storages := wsp.appConfig.StorageSettings.Get()
	formatedId := min(id, len(storages)-1)

	wsp.appConfig.BehaviorConfig.SelectedStorageId.Set(formatedId)
	wsp.currentWebsite = *storages[formatedId]

	wsp.list.Select(formatedId)

	wsp.infoContainer.titleLabel.SetText(wsp.currentWebsite.Name)
	wsp.infoContainer.urlEntry.SetText(wsp.currentWebsite.URL)

	switch wsp.currentWebsite.StorageType {
	case config.FileSystemConfig:
		wsp.infoContainer.storageTypeSelect.SetSelected("File System")
	default:
		fallthrough
	case config.WebsiteConfig:
		wsp.infoContainer.storageTypeSelect.SetSelected("Website")
	}

	wsp.infoContainer.needTokenCheck.SetChecked(wsp.currentWebsite.NeedToken)
	wsp.infoContainer.tokenEntry.SetText(wsp.currentWebsite.Token)
	wsp.infoContainer.sendPingCheck.SetChecked(wsp.currentWebsite.SendPing)
	wsp.infoContainer.pingPathEntry.SetText(wsp.currentWebsite.PingPath)
	wsp.infoContainer.sendReplayCheck.SetChecked(wsp.currentWebsite.SendReplay)
	wsp.infoContainer.replayPathEntry.SetText(wsp.currentWebsite.ReplayPath)
	wsp.infoContainer.templateNameEntry.SetText(wsp.currentWebsite.TemplateName)
	wsp.infoContainer.sendLiveCheck.SetChecked(wsp.appConfig.BehaviorConfig.SendLiveStat.Get() && wsp.currentWebsite.SendLive)
	wsp.infoContainer.livePathEntry.SetText(wsp.currentWebsite.LivePath)

	var paramsStr strings.Builder
	for k, v := range wsp.currentWebsite.URIParams {
		fmt.Fprintf(&paramsStr, "%s=%s\n", k, v)
	}
	wsp.infoContainer.uriParamsEntry.SetText(strings.TrimSpace(paramsStr.String()))

	wsp.reload()
}

func (wsp *StorageSettingsPopup) reload() {
	logger.FuncDebug()

	wsp.reloadShow()
	wsp.reloadEnable()

	wsp.editForm.Refresh()
}

func (wsp *StorageSettingsPopup) reloadShow() {
	logger.FuncDebug()

	wsp.infoContainer.urlForm.Widget.Show()
	wsp.infoContainer.storageTypeForm.Widget.Show()
	wsp.infoContainer.uriParamsForm.Widget.Show()
	wsp.infoContainer.needTokenForm.Widget.Show()
	wsp.infoContainer.tokenForm.Widget.Show()
	wsp.infoContainer.sendPingForm.Widget.Show()
	wsp.infoContainer.pingPathForm.Widget.Show()
	wsp.infoContainer.sendReplayForm.Widget.Show()
	wsp.infoContainer.replayPathForm.Widget.Show()
	wsp.infoContainer.templateNameForm.Widget.Show()
	wsp.infoContainer.sendLiveForm.Widget.Show()
	wsp.infoContainer.livePathForm.Widget.Show()

	if !wsp.infoContainer.needTokenCheck.Checked {
		wsp.infoContainer.tokenForm.Widget.Hide()
	}

	if !wsp.infoContainer.sendPingCheck.Checked {
		wsp.infoContainer.pingPathForm.Widget.Hide()
	}

	if !wsp.infoContainer.sendLiveCheck.Checked {
		wsp.infoContainer.livePathForm.Widget.Hide()
	}

	if !wsp.appConfig.BehaviorConfig.SendLiveStat.Get() {
		wsp.infoContainer.sendLiveForm.Widget.Hide()
		wsp.infoContainer.livePathForm.Widget.Hide()
	}

	if !wsp.infoContainer.sendReplayCheck.Checked {
		wsp.infoContainer.replayPathForm.Widget.Hide()
		wsp.infoContainer.templateNameForm.Widget.Hide()
	}

	switch wsp.currentWebsite.StorageType {
	case config.FileSystemConfig:
		wsp.infoContainer.urlForm.Widget.Hide()
		wsp.infoContainer.uriParamsForm.Widget.Hide()
		wsp.infoContainer.needTokenForm.Widget.Hide()
		wsp.infoContainer.tokenForm.Widget.Hide()
		wsp.infoContainer.sendPingForm.Widget.Hide()
		wsp.infoContainer.pingPathForm.Widget.Hide()
		wsp.infoContainer.sendLiveForm.Widget.Hide()
		wsp.infoContainer.livePathForm.Widget.Hide()
	default:
		fallthrough
	case config.WebsiteConfig:
	}

	if wsp.currentWebsite.IsPredefined {
		if wsp.currentWebsite.IsPrimary {
			wsp.infoContainer.templateNameForm.Widget.Hide()
		}
	}
}

func (wsp *StorageSettingsPopup) reloadEnable() {
	logger.FuncDebug()

	wsp.btnSave.Enable()
	wsp.btnDelete.Enable()

	wsp.infoContainer.urlEntry.Enable()
	wsp.infoContainer.storageTypeSelect.Enable()
	wsp.infoContainer.uriParamsEntry.Enable()
	wsp.infoContainer.needTokenCheck.Enable()
	wsp.infoContainer.tokenEntry.Enable()
	wsp.infoContainer.sendPingCheck.Enable()
	wsp.infoContainer.pingPathEntry.Enable()
	wsp.infoContainer.sendReplayCheck.Enable()
	wsp.infoContainer.replayPathEntry.Enable()
	wsp.infoContainer.templateNameEntry.Enable()
	wsp.infoContainer.sendLiveCheck.Enable()
	wsp.infoContainer.livePathEntry.Enable()

	if !wsp.appConfig.BehaviorConfig.SendLiveStat.Get() {
		wsp.infoContainer.sendLiveCheck.Disable()
		wsp.infoContainer.livePathEntry.Disable()
	}

	switch wsp.currentWebsite.StorageType {
	case config.FileSystemConfig:
		wsp.infoContainer.urlEntry.Disable()
		wsp.infoContainer.uriParamsEntry.Disable()
		wsp.infoContainer.needTokenCheck.Disable()
		wsp.infoContainer.tokenEntry.Disable()
		wsp.infoContainer.sendPingCheck.Disable()
		wsp.infoContainer.pingPathEntry.Disable()
		wsp.infoContainer.sendLiveCheck.Disable()
		wsp.infoContainer.livePathEntry.Disable()
	default:
		fallthrough
	case config.WebsiteConfig:
	}

	if wsp.currentWebsite.IsPredefined {
		wsp.btnDelete.Disable()

		wsp.infoContainer.urlEntry.Disable()
		wsp.infoContainer.storageTypeSelect.Disable()
		wsp.infoContainer.needTokenCheck.Disable()
		wsp.infoContainer.sendPingCheck.Disable()
		wsp.infoContainer.pingPathEntry.Disable()
		wsp.infoContainer.livePathEntry.Disable()
		wsp.infoContainer.sendLiveCheck.Disable()

		if wsp.currentWebsite.IsPrimary {
			wsp.btnSave.Disable()
			wsp.infoContainer.sendReplayCheck.Disable()
			wsp.infoContainer.uriParamsEntry.Disable()
			wsp.infoContainer.tokenEntry.Disable()
			wsp.infoContainer.templateNameEntry.Disable()
			wsp.infoContainer.replayPathEntry.Disable()
		} else {
			switch wsp.currentWebsite.StorageType {
			default:
				fallthrough
			case config.WebsiteConfig:
				wsp.infoContainer.replayPathEntry.Disable()
			}
		}
	}
}

func (wsp *StorageSettingsPopup) onUnselected(id widget.ListItemID) {
	logger.FuncDebug()

	wsp.appConfig.BehaviorConfig.SelectedStorageId.Set(-1)
	wsp.infoContainer.titleLabel.SetText("Select a storage")

	wsp.currentWebsite = config.StorageConfig{}

	wsp.infoContainer.urlEntry.SetText("")
	wsp.infoContainer.storageTypeSelect.ClearSelected()
	wsp.infoContainer.needTokenCheck.SetChecked(false)
	wsp.infoContainer.tokenEntry.SetText("")
	wsp.infoContainer.sendPingCheck.SetChecked(false)
	wsp.infoContainer.pingPathEntry.SetText("")
	wsp.infoContainer.sendReplayCheck.SetChecked(false)
	wsp.infoContainer.replayPathEntry.SetText("")
	wsp.infoContainer.templateNameEntry.SetText("")
	wsp.infoContainer.uriParamsEntry.SetText("")
	wsp.infoContainer.sendLiveCheck.SetChecked(false)
	wsp.infoContainer.livePathEntry.SetText("")

	wsp.btnDelete.Disable()
	wsp.btnSave.Disable()
}

func (wsp *StorageSettingsPopup) onSaveBtn() {
	logger.FuncDebug()

	selectedIndex := wsp.appConfig.BehaviorConfig.SelectedStorageId.Get()
	if selectedIndex < 0 {
		return
	}

	storages := wsp.appConfig.StorageSettings.Get()
	selectedIndex = min(selectedIndex, len(storages)-1)
	site := storages[selectedIndex]

	site.URL = wsp.infoContainer.urlEntry.Text

	switch wsp.infoContainer.storageTypeSelect.Selected {
	case "File System":
		site.StorageType = config.FileSystemConfig
	default:
		fallthrough
	case "Website":
		site.StorageType = config.WebsiteConfig
	}

	site.NeedToken = wsp.infoContainer.needTokenCheck.Checked
	site.Token = wsp.infoContainer.tokenEntry.Text
	site.SendPing = wsp.infoContainer.sendPingCheck.Checked
	site.PingPath = wsp.infoContainer.pingPathEntry.Text
	site.SendReplay = wsp.infoContainer.sendReplayCheck.Checked
	site.ReplayPath = wsp.infoContainer.replayPathEntry.Text
	site.TemplateName = wsp.infoContainer.templateNameEntry.Text

	if wsp.appConfig.BehaviorConfig.SendLiveStat.Get() {
		site.SendLive = wsp.infoContainer.sendLiveCheck.Checked
		site.LivePath = wsp.infoContainer.livePathEntry.Text
	}

	newParams := make(map[string]string)
	lines := strings.Split(wsp.infoContainer.uriParamsEntry.Text, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if k != "" {
				newParams[k] = v
			}
		}
	}
	site.URIParams = newParams

	wsp.appConfig.StorageSettings.Set(storages)
	wsp.currentWebsite = *wsp.appConfig.StorageSettings.Get()[selectedIndex]

	wsp.reload()
	wsp.list.Refresh()

	dialog.ShowInformation("Success", "Settings saved successfully!", wsp.parentWindow)
}

func (wsp *StorageSettingsPopup) onAddStorageBtn() {
	logger.FuncDebug()

	nameEntry := widget.NewEntry()
	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
	}

	dialog.ShowForm("New Storage", "Add", "Cancel", items, func(valide bool) {
		if valide && nameEntry.Text != "" {
			newSite := &config.StorageConfig{
				Name:         nameEntry.Text,
				URL:          "",
				IsPrimary:    false,
				IsPredefined: false,
				StorageType:  config.WebsiteConfig,
				NeedToken:    false,
				SendPing:     false,
				SendReplay:   false,
				URIParams:    make(map[string]string),
			}

			storages := wsp.appConfig.StorageSettings.Get()

			for _, StorageConfig := range storages {
				if StorageConfig.Name == newSite.Name {
					return
				}
			}

			storages = append(storages, newSite)
			wsp.appConfig.StorageSettings.Set(storages)

			wsp.onSelected(len(storages) - 1)
			wsp.list.Refresh()
		}
	}, wsp.parentWindow)
}

func (wsp *StorageSettingsPopup) onDeleteStorageBtn() {
	logger.FuncDebug()

	storages := wsp.appConfig.StorageSettings.Get()

	selectedIndex := wsp.appConfig.BehaviorConfig.SelectedStorageId.Get()
	if selectedIndex < 0 || selectedIndex >= len(storages) {
		return
	}

	storageToDelete := storages[selectedIndex]

	dialog.ShowConfirm("Delete", "Are you sure you want to delete the storage '"+storageToDelete.Name+"' ?", func(confirmed bool) {
		if confirmed {
			storages = append(storages[:selectedIndex], storages[selectedIndex+1:]...)

			wsp.appConfig.StorageSettings.Set(storages)

			wsp.onSelected(len(storages) - 1)
			wsp.list.Refresh()
		}
	}, wsp.parentWindow)
}
