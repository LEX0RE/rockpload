package ui

import (
	"fmt"
	"net/url"
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

func parseHelpURL(raw string) (*url.URL, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, false
	}

	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, false
	}

	return parsed, true
}

type StorageInfoContainer struct {
	titleLabel        *widget.Label
	descContainer     *fyne.Container
	urlEntry          *widget.Entry
	uploadStyleSelect *widget.Select
	pingStyleSelect   *widget.Select
	uriParamsEntry    *widget.Entry
	tokenStyleSelect  *widget.Select
	tokenEntry        *widget.Entry
	pingPathEntry     *widget.Entry
	pingProbeIDEntry  *widget.Entry
	replayPathEntry   *widget.Entry
	templateNameEntry *widget.Entry
	liveStyleSelect   *widget.Select
	livePathEntry     *widget.Entry
	helpLink          *widget.Hyperlink

	urlForm          *widget.FormItem
	uploadStyleForm  *widget.FormItem
	pingStyleForm    *widget.FormItem
	uriParamsForm    *widget.FormItem
	tokenStyleForm   *widget.FormItem
	tokenForm        *widget.FormItem
	pingPathForm     *widget.FormItem
	pingProbeIDForm  *widget.FormItem
	replayPathForm   *widget.FormItem
	templateNameForm *widget.FormItem
	liveStyleForm    *widget.FormItem
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

	wsp.btnSave = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), wsp.onSaveBtn)
	wsp.btnSave.Importance = widget.HighImportance
	wsp.btnSave.Disable()

	wsp.btnDelete = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), wsp.onDeleteStorageBtn)
	wsp.btnDelete.Importance = widget.DangerImportance
	wsp.btnDelete.Disable()

	wsp.editForm = wsp.createInfoContainer()
	scrollableForm := container.NewVScroll(wsp.editForm)
	buttonsBox := container.NewHBox(layout.NewSpacer(), wsp.btnSave, wsp.btnDelete)

	wsp.detailPanel = container.NewBorder(
		container.NewVBox(wsp.infoContainer.titleLabel, wsp.infoContainer.descContainer, widget.NewSeparator()),
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
			if site.UploadStyle == config.UploadDisabled {
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

	btnAdd := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), wsp.onAddStorageBtn)

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

	wsp.infoContainer.uploadStyleSelect = widget.NewSelect(config.UploadStyleLabels[:], func(v string) {
		wsp.currentWebsite.UploadStyle = config.UploadStyleFromLabel(v)
		wsp.reload()
	})

	wsp.infoContainer.pingStyleSelect = widget.NewSelect(config.PingStyleLabels[:], func(v string) {
		wsp.currentWebsite.PingStyle = config.PingStyleFromLabel(v)
		wsp.reload()
	})

	wsp.infoContainer.uriParamsEntry = widget.NewMultiLineEntry()
	wsp.infoContainer.uriParamsEntry.SetPlaceHolder("")

	wsp.infoContainer.tokenStyleSelect = widget.NewSelect(config.TokenStyleLabels[:], func(v string) {
		wsp.currentWebsite.TokenStyle = config.TokenStyleFromLabel(v)
		wsp.reload()
	})

	wsp.infoContainer.tokenEntry = widget.NewPasswordEntry()
	wsp.infoContainer.tokenEntry.SetPlaceHolder("")

	wsp.infoContainer.helpLink = widget.NewHyperlink("", nil)
	wsp.infoContainer.helpLink.OnTapped = func() {
		parsed, ok := parseHelpURL(wsp.currentWebsite.HelpURL)
		if !ok {
			return
		}

		dialog.ShowConfirm("Open external link?", "This will open the following link in your browser:\n\n"+parsed.String(),
			func(confirmed bool) {
				if confirmed {
					fyne.CurrentApp().OpenURL(parsed)
				}
			}, wsp.parentWindow)
	}

	wsp.infoContainer.descContainer = container.NewCenter(wsp.infoContainer.helpLink)
	wsp.infoContainer.descContainer.Hide()

	wsp.infoContainer.pingPathEntry = widget.NewEntry()
	wsp.infoContainer.pingProbeIDEntry = widget.NewEntry()

	wsp.infoContainer.liveStyleSelect = widget.NewSelect(config.LiveStyleLabels[:], func(v string) {
		wsp.currentWebsite.LiveStyle = config.LiveStyleFromLabel(v)
		wsp.reload()
	})
	wsp.infoContainer.livePathEntry = widget.NewEntry()

	wsp.infoContainer.replayPathEntry = widget.NewEntry()
	wsp.infoContainer.templateNameEntry = widget.NewEntry()

	wsp.infoContainer.urlForm = widget.NewFormItem("URL", wsp.infoContainer.urlEntry)
	wsp.infoContainer.uploadStyleForm = widget.NewFormItem("Upload Style", wsp.infoContainer.uploadStyleSelect)
	wsp.infoContainer.pingStyleForm = widget.NewFormItem("Ping Style", wsp.infoContainer.pingStyleSelect)
	wsp.infoContainer.uriParamsForm = widget.NewFormItem("URI Params", wsp.infoContainer.uriParamsEntry)
	wsp.infoContainer.tokenStyleForm = widget.NewFormItem("Token Style", wsp.infoContainer.tokenStyleSelect)
	wsp.infoContainer.tokenForm = widget.NewFormItem("Token", wsp.infoContainer.tokenEntry)
	wsp.infoContainer.pingPathForm = widget.NewFormItem("Ping Path", wsp.infoContainer.pingPathEntry)
	wsp.infoContainer.pingProbeIDForm = widget.NewFormItem("Ping Probe ID", wsp.infoContainer.pingProbeIDEntry)
	wsp.infoContainer.replayPathForm = widget.NewFormItem("Replay Path", wsp.infoContainer.replayPathEntry)
	wsp.infoContainer.templateNameForm = widget.NewFormItem("Name Template", wsp.infoContainer.templateNameEntry)
	wsp.infoContainer.liveStyleForm = widget.NewFormItem("Live Style", wsp.infoContainer.liveStyleSelect)
	wsp.infoContainer.livePathForm = widget.NewFormItem("Live Path", wsp.infoContainer.livePathEntry)

	return widget.NewForm(
		wsp.infoContainer.uploadStyleForm,
		wsp.infoContainer.replayPathForm,
		wsp.infoContainer.templateNameForm,
		wsp.infoContainer.urlForm,
		wsp.infoContainer.tokenStyleForm,
		wsp.infoContainer.tokenForm,
		wsp.infoContainer.pingStyleForm,
		wsp.infoContainer.pingPathForm,
		wsp.infoContainer.pingProbeIDForm,
		wsp.infoContainer.uriParamsForm,
		wsp.infoContainer.liveStyleForm,
		wsp.infoContainer.livePathForm,
	)
}

func (wsp *StorageSettingsPopup) currentPreset() *config.StorageConfig {
	logger.FuncDebug()

	if !wsp.currentWebsite.IsPredefined {
		return nil
	}

	return config.STORAGE_PRESET[wsp.currentWebsite.Name]
}

func (wsp *StorageSettingsPopup) reloadOptions() {
	logger.FuncDebug()

	preset := wsp.currentPreset()
	if preset == nil {
		wsp.infoContainer.uploadStyleSelect.SetOptions(config.UploadStyleLabels[:])
		wsp.infoContainer.pingStyleSelect.SetOptions(config.PingStyleLabels[:])
		wsp.infoContainer.tokenStyleSelect.SetOptions(config.TokenStyleLabels[:])
		wsp.infoContainer.liveStyleSelect.SetOptions(config.LiveStyleLabels[:])
		return
	}

	wsp.infoContainer.uploadStyleSelect.SetOptions([]string{
		config.UploadDisabled.Label(),
		preset.UploadStyle.Label(),
	})
	wsp.infoContainer.pingStyleSelect.SetOptions([]string{
		config.PingDisabled.Label(),
		preset.PingStyle.Label(),
	})
	wsp.infoContainer.tokenStyleSelect.SetOptions([]string{
		preset.TokenStyle.Label(),
	})
	wsp.infoContainer.liveStyleSelect.SetOptions([]string{
		preset.LiveStyle.Label(),
	})
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
	wsp.reloadOptions()

	wsp.list.Select(formatedId)

	wsp.infoContainer.titleLabel.SetText(wsp.currentWebsite.Name)
	wsp.infoContainer.urlEntry.SetText(wsp.currentWebsite.URL)
	wsp.infoContainer.uploadStyleSelect.SetSelected(wsp.currentWebsite.UploadStyle.Label())
	wsp.infoContainer.pingStyleSelect.SetSelected(wsp.currentWebsite.PingStyle.Label())

	wsp.infoContainer.tokenStyleSelect.SetSelected(wsp.currentWebsite.TokenStyle.Label())
	wsp.infoContainer.tokenEntry.SetText(wsp.currentWebsite.Token)
	wsp.infoContainer.pingPathEntry.SetText(wsp.currentWebsite.PingPath)
	wsp.infoContainer.pingProbeIDEntry.SetText(wsp.currentWebsite.PingProbeID)
	wsp.infoContainer.replayPathEntry.SetText(wsp.currentWebsite.ReplayPath)
	wsp.infoContainer.templateNameEntry.SetText(wsp.currentWebsite.TemplateName)
	wsp.infoContainer.liveStyleSelect.SetSelected(wsp.currentWebsite.LiveStyle.Label())
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

	wsp.reloadOptions()
	wsp.reloadShow()
	wsp.reloadEnable()

	wsp.editForm.Refresh()
	wsp.detailPanel.Refresh()
}

func (wsp *StorageSettingsPopup) reloadShow() {
	logger.FuncDebug()

	wsp.infoContainer.urlForm.Widget.Show()
	wsp.infoContainer.uploadStyleForm.Widget.Show()
	wsp.infoContainer.pingStyleForm.Widget.Show()
	wsp.infoContainer.uriParamsForm.Widget.Show()
	wsp.infoContainer.tokenStyleForm.Widget.Show()
	wsp.infoContainer.tokenForm.Widget.Show()
	wsp.infoContainer.pingPathForm.Widget.Show()
	wsp.infoContainer.pingProbeIDForm.Widget.Show()
	wsp.infoContainer.replayPathForm.Widget.Show()
	wsp.infoContainer.templateNameForm.Widget.Show()
	wsp.infoContainer.liveStyleForm.Widget.Show()
	wsp.infoContainer.livePathForm.Widget.Show()

	wsp.infoContainer.helpLink.Text = wsp.currentWebsite.HelpText
	wsp.infoContainer.helpLink.Refresh()

	if _, ok := parseHelpURL(wsp.currentWebsite.HelpURL); ok {
		wsp.infoContainer.descContainer.Show()
	} else {
		wsp.infoContainer.descContainer.Hide()
	}

	if wsp.currentWebsite.TokenStyle == config.NoToken {
		wsp.infoContainer.tokenForm.Widget.Hide()
	}

	if wsp.currentWebsite.PingStyle == config.PingDisabled {
		wsp.infoContainer.pingPathForm.Widget.Hide()
	}

	if wsp.currentWebsite.PingStyle != config.PingNotFoundIsValid {
		wsp.infoContainer.pingProbeIDForm.Widget.Hide()
	}

	if wsp.currentWebsite.LiveStyle == config.LiveDisabled {
		wsp.infoContainer.livePathForm.Widget.Hide()
	}

	if !wsp.appConfig.BehaviorConfig.SendLiveStat.Get() {
		wsp.infoContainer.liveStyleForm.Widget.Hide()
		wsp.infoContainer.livePathForm.Widget.Hide()
	}

	if wsp.currentWebsite.UploadStyle == config.UploadDisabled {
		wsp.infoContainer.replayPathForm.Widget.Hide()
		wsp.infoContainer.templateNameForm.Widget.Hide()
	}

	switch wsp.currentWebsite.UploadStyle {
	case config.LocalFileCopy:
		wsp.infoContainer.urlForm.Widget.Hide()
		wsp.infoContainer.uriParamsForm.Widget.Hide()
		wsp.infoContainer.tokenStyleForm.Widget.Hide()
		wsp.infoContainer.tokenForm.Widget.Hide()
		wsp.infoContainer.pingStyleForm.Widget.Hide()
		wsp.infoContainer.pingPathForm.Widget.Hide()
		wsp.infoContainer.pingProbeIDForm.Widget.Hide()
		wsp.infoContainer.liveStyleForm.Widget.Hide()
		wsp.infoContainer.livePathForm.Widget.Hide()
	case config.PresignedSessionUpload:
		wsp.infoContainer.templateNameForm.Widget.Hide()
	default:
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
	wsp.infoContainer.uploadStyleSelect.Enable()
	wsp.infoContainer.pingStyleSelect.Enable()
	wsp.infoContainer.uriParamsEntry.Enable()
	wsp.infoContainer.tokenStyleSelect.Enable()
	wsp.infoContainer.tokenEntry.Enable()
	wsp.infoContainer.pingPathEntry.Enable()
	wsp.infoContainer.pingProbeIDEntry.Enable()
	wsp.infoContainer.replayPathEntry.Enable()
	wsp.infoContainer.templateNameEntry.Enable()
	wsp.infoContainer.liveStyleSelect.Enable()
	wsp.infoContainer.livePathEntry.Enable()

	if !wsp.appConfig.BehaviorConfig.SendLiveStat.Get() {
		wsp.infoContainer.liveStyleSelect.Disable()
		wsp.infoContainer.livePathEntry.Disable()
	}

	switch wsp.currentWebsite.UploadStyle {
	case config.LocalFileCopy:
		wsp.infoContainer.urlEntry.Disable()
		wsp.infoContainer.uriParamsEntry.Disable()
		wsp.infoContainer.tokenStyleSelect.Disable()
		wsp.infoContainer.tokenEntry.Disable()
		wsp.infoContainer.pingStyleSelect.Disable()
		wsp.infoContainer.pingPathEntry.Disable()
		wsp.infoContainer.pingProbeIDEntry.Disable()
		wsp.infoContainer.liveStyleSelect.Disable()
		wsp.infoContainer.livePathEntry.Disable()
	case config.PresignedSessionUpload:
		wsp.infoContainer.templateNameEntry.Disable()
	default:
	}

	if wsp.currentWebsite.IsPredefined {
		wsp.btnDelete.Disable()

		wsp.infoContainer.urlEntry.Disable()
		wsp.infoContainer.tokenStyleSelect.Disable()
		wsp.infoContainer.pingPathEntry.Disable()
		wsp.infoContainer.pingProbeIDEntry.Disable()
		wsp.infoContainer.livePathEntry.Disable()
		wsp.infoContainer.liveStyleSelect.Disable()

		if wsp.currentWebsite.IsPrimary {
			wsp.btnSave.Disable()
			wsp.infoContainer.uploadStyleSelect.Disable()
			wsp.infoContainer.pingStyleSelect.Disable()
			wsp.infoContainer.uriParamsEntry.Disable()
			wsp.infoContainer.tokenEntry.Disable()
			wsp.infoContainer.templateNameEntry.Disable()
			wsp.infoContainer.replayPathEntry.Disable()
		} else if preset := wsp.currentPreset(); preset != nil && preset.UploadStyle != config.LocalFileCopy {
			wsp.infoContainer.replayPathEntry.Disable()
		}
	}
}

func (wsp *StorageSettingsPopup) onUnselected(id widget.ListItemID) {
	logger.FuncDebug()

	wsp.appConfig.BehaviorConfig.SelectedStorageId.Set(-1)
	wsp.infoContainer.titleLabel.SetText("Select a storage")

	wsp.currentWebsite = config.StorageConfig{}

	wsp.infoContainer.urlEntry.SetText("")
	wsp.infoContainer.uploadStyleSelect.ClearSelected()
	wsp.infoContainer.pingStyleSelect.ClearSelected()
	wsp.infoContainer.tokenStyleSelect.ClearSelected()
	wsp.infoContainer.tokenEntry.SetText("")
	wsp.infoContainer.pingPathEntry.SetText("")
	wsp.infoContainer.pingProbeIDEntry.SetText("")
	wsp.infoContainer.replayPathEntry.SetText("")
	wsp.infoContainer.templateNameEntry.SetText("")
	wsp.infoContainer.uriParamsEntry.SetText("")
	wsp.infoContainer.liveStyleSelect.ClearSelected()
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
	site.UploadStyle = config.UploadStyleFromLabel(wsp.infoContainer.uploadStyleSelect.Selected)
	site.PingStyle = config.PingStyleFromLabel(wsp.infoContainer.pingStyleSelect.Selected)

	site.TokenStyle = config.TokenStyleFromLabel(wsp.infoContainer.tokenStyleSelect.Selected)
	site.Token = wsp.infoContainer.tokenEntry.Text
	site.PingPath = wsp.infoContainer.pingPathEntry.Text
	site.PingProbeID = wsp.infoContainer.pingProbeIDEntry.Text
	site.ReplayPath = wsp.infoContainer.replayPathEntry.Text
	site.TemplateName = wsp.infoContainer.templateNameEntry.Text

	if wsp.appConfig.BehaviorConfig.SendLiveStat.Get() {
		site.LiveStyle = config.LiveStyleFromLabel(wsp.infoContainer.liveStyleSelect.Selected)
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
				UploadStyle:  config.UploadDisabled,
				PingStyle:    config.PingDisabled,
				TokenStyle:   config.NoToken,
				LiveStyle:    config.LiveDisabled,
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
