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
}

type StorageSettingsPopup struct {
	*Popup

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

	wsp.btnDelete = widget.NewButtonWithIcon("Delete this website", theme.DeleteIcon(), wsp.onDeleteWebsiteBtn)
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

	btnAdd := widget.NewButtonWithIcon("Add a website", theme.ContentAddIcon(), wsp.onAddWebsiteBtn)

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

	wsp.infoContainer.titleLabel = widget.NewLabelWithStyle("Select a website", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	wsp.infoContainer.urlEntry = widget.NewEntry()

	wsp.infoContainer.storageTypeSelect = widget.NewSelect([]string{"Website", "File System"}, func(val string) {
		wsp.reload()
	})

	wsp.infoContainer.uriParamsEntry = widget.NewMultiLineEntry()
	wsp.infoContainer.uriParamsEntry.SetPlaceHolder("")

	wsp.infoContainer.needTokenCheck = widget.NewCheck("", func(bool) { wsp.reload() })
	wsp.infoContainer.tokenEntry = widget.NewPasswordEntry()
	wsp.infoContainer.tokenEntry.SetPlaceHolder("")

	wsp.infoContainer.sendPingCheck = widget.NewCheck("", func(bool) { wsp.reload() })
	wsp.infoContainer.pingPathEntry = widget.NewEntry()

	wsp.infoContainer.sendReplayCheck = widget.NewCheck("", func(bool) { wsp.reload() })
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
	)
}

func (wsp *StorageSettingsPopup) onSelected(id widget.ListItemID) {
	logger.FuncDebug()

	wsp.appConfig.BehaviorConfig.SelectedStorageId.Set(id)

	websites := wsp.appConfig.StorageSettings.Get()
	site := websites[id]

	wsp.list.Select(id)

	wsp.infoContainer.titleLabel.SetText(site.Name)
	wsp.infoContainer.urlEntry.SetText(site.URL)

	switch site.StorageType {
	case config.FileSystemConfig:
		wsp.infoContainer.storageTypeSelect.SetSelected("File System")
	default:
		fallthrough
	case config.WebsiteConfig:
		wsp.infoContainer.storageTypeSelect.SetSelected("Website")
	}

	wsp.infoContainer.needTokenCheck.SetChecked(site.NeedToken)
	wsp.infoContainer.tokenEntry.SetText(site.Token)
	wsp.infoContainer.sendPingCheck.SetChecked(site.SendPing)
	wsp.infoContainer.pingPathEntry.SetText(site.PingPath)
	wsp.infoContainer.sendReplayCheck.SetChecked(site.SendReplay)
	wsp.infoContainer.replayPathEntry.SetText(site.ReplayPath)
	wsp.infoContainer.templateNameEntry.SetText(site.TemplateName)

	if wsp.infoContainer.needTokenCheck.Checked {
		wsp.infoContainer.tokenEntry.Enable()
	} else {
		wsp.infoContainer.tokenEntry.Disable()
	}

	var paramsStr string
	for k, v := range site.URIParams {
		paramsStr += fmt.Sprintf("%s=%s\n", k, v)
	}
	wsp.infoContainer.uriParamsEntry.SetText(strings.TrimSpace(paramsStr))

	wsp.reload()
}

func (wsp *StorageSettingsPopup) reload() {
	logger.FuncDebug()

	websites := wsp.appConfig.StorageSettings.Get()

	selectedIndex := wsp.appConfig.BehaviorConfig.SelectedStorageId.Get()
	if selectedIndex < len(websites) && selectedIndex >= 0 {
		site := websites[selectedIndex]

		if site.IsPredefined {
			wsp.btnDelete.Disable()

			wsp.infoContainer.urlEntry.Disable()
			wsp.infoContainer.storageTypeSelect.Disable()
			wsp.infoContainer.needTokenCheck.Disable()
			wsp.infoContainer.sendPingCheck.Disable()
			wsp.infoContainer.pingPathEntry.Disable()
			wsp.infoContainer.replayPathEntry.Disable()
			wsp.infoContainer.templateNameEntry.Disable()

			if site.IsPrimary {
				wsp.btnSave.Disable()
				wsp.infoContainer.sendReplayCheck.Disable()
				wsp.infoContainer.uriParamsEntry.Disable()
			} else {
				wsp.btnSave.Enable()
				wsp.infoContainer.sendReplayCheck.Enable()
				wsp.infoContainer.templateNameEntry.Enable()

				switch site.StorageType {
				case config.FileSystemConfig:
					wsp.infoContainer.replayPathEntry.Enable()
					wsp.infoContainer.uriParamsEntry.Disable()
				default:
					fallthrough
				case config.WebsiteConfig:
					wsp.infoContainer.replayPathEntry.Disable()
					wsp.infoContainer.uriParamsEntry.Enable()
				}
			}
		} else {
			wsp.btnDelete.Enable()
			wsp.btnSave.Enable()

			wsp.infoContainer.urlEntry.Enable()
			wsp.infoContainer.storageTypeSelect.Enable()
			wsp.infoContainer.uriParamsEntry.Enable()
			wsp.infoContainer.needTokenCheck.Enable()
			wsp.infoContainer.sendPingCheck.Enable()
			wsp.infoContainer.pingPathEntry.Enable()
			wsp.infoContainer.sendReplayCheck.Enable()
			wsp.infoContainer.replayPathEntry.Enable()
			wsp.infoContainer.templateNameEntry.Enable()
		}
	}

	if wsp.infoContainer.needTokenCheck.Checked {
		wsp.infoContainer.tokenForm.Widget.Show()
	} else {
		wsp.infoContainer.tokenForm.Widget.Hide()
	}

	if wsp.infoContainer.sendPingCheck.Checked {
		wsp.infoContainer.pingPathForm.Widget.Show()
	} else {
		wsp.infoContainer.pingPathForm.Widget.Hide()
	}

	if wsp.infoContainer.sendReplayCheck.Checked {
		wsp.infoContainer.replayPathForm.Widget.Show()
		wsp.infoContainer.templateNameForm.Widget.Show()
	} else {
		wsp.infoContainer.replayPathForm.Widget.Hide()
		wsp.infoContainer.templateNameForm.Widget.Hide()
	}

	wsp.editForm.Refresh()
}

func (wsp *StorageSettingsPopup) onUnselected(id widget.ListItemID) {
	logger.FuncDebug()

	wsp.appConfig.BehaviorConfig.SelectedStorageId.Set(-1)
	wsp.infoContainer.titleLabel.SetText("Select a website")

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

	wsp.btnDelete.Disable()
	wsp.btnSave.Disable()
}

func (wsp *StorageSettingsPopup) onSaveBtn() {
	logger.FuncDebug()

	selectedIndex := wsp.appConfig.BehaviorConfig.SelectedStorageId.Get()
	if selectedIndex < 0 {
		return
	}

	websites := wsp.appConfig.StorageSettings.Get()
	site := websites[selectedIndex]

	site.URL = wsp.infoContainer.urlEntry.Text

	switch site.StorageType {
	case config.FileSystemConfig:
		site.StorageType = config.FileSystemConfig
	default:
		fallthrough
	case config.WebsiteConfig:
		site.StorageType = config.WebsiteConfig
	}

	site.NeedToken = wsp.infoContainer.needTokenCheck.Checked
	site.Token = wsp.infoContainer.tokenEntry.Text
	site.SendPing = wsp.infoContainer.sendPingCheck.Checked
	site.PingPath = wsp.infoContainer.pingPathEntry.Text
	site.SendReplay = wsp.infoContainer.sendReplayCheck.Checked
	site.ReplayPath = wsp.infoContainer.replayPathEntry.Text
	site.TemplateName = wsp.infoContainer.templateNameEntry.Text

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

	wsp.appConfig.StorageSettings.Set(websites)

	wsp.reload()
	wsp.list.Refresh()

	dialog.ShowInformation("Success", "Settings saved successfully!", wsp.parentWindow)
}

func (wsp *StorageSettingsPopup) onAddWebsiteBtn() {
	logger.FuncDebug()

	nameEntry := widget.NewEntry()
	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
	}

	dialog.ShowForm("New Website", "Add", "Cancel", items, func(valide bool) {
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

			websites := wsp.appConfig.StorageSettings.Get()

			for _, StorageConfig := range websites {
				if StorageConfig.Name == newSite.Name {
					return
				}
			}

			websites = append(websites, newSite)
			wsp.appConfig.StorageSettings.Set(websites)

			wsp.onSelected(len(websites) - 1)
			wsp.list.Refresh()
		}
	}, wsp.parentWindow)
}

func (wsp *StorageSettingsPopup) onDeleteWebsiteBtn() {
	logger.FuncDebug()

	websites := wsp.appConfig.StorageSettings.Get()

	selectedIndex := wsp.appConfig.BehaviorConfig.SelectedStorageId.Get()
	if selectedIndex < 0 || selectedIndex >= len(websites) {
		return
	}

	siteToDelete := websites[selectedIndex]

	dialog.ShowConfirm("Delete", "Are you sure you want to delete the website '"+siteToDelete.Name+"' ?", func(confirmed bool) {
		if confirmed {
			websites = append(websites[:selectedIndex], websites[selectedIndex+1:]...)

			wsp.appConfig.StorageSettings.Set(websites)

			wsp.onSelected(len(websites) - 1)
			wsp.list.Refresh()
		}
	}, wsp.parentWindow)
}
