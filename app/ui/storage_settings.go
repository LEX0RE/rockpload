package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/LEX0RE/rockpload/app/upload"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	testPingResultDisplayDuration = 3 * time.Second
	playlistSearchMaxResults      = 5
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

type playlistFilterBox struct {
	widget.DisableableWidget

	content     fyne.CanvasObject
	searchEntry *widget.Entry
}

func newPlaylistFilterBox(content fyne.CanvasObject, searchEntry *widget.Entry) *playlistFilterBox {
	b := &playlistFilterBox{content: content, searchEntry: searchEntry}
	b.ExtendBaseWidget(b)

	return b
}

func (b *playlistFilterBox) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.content)
}

func (b *playlistFilterBox) Enable() {
	b.DisableableWidget.Enable()
	b.searchEntry.Enable()
}

func (b *playlistFilterBox) Disable() {
	b.DisableableWidget.Disable()
	b.searchEntry.Disable()
}

type StorageInfoContainer struct {
	titleLabel                *widget.Label
	descContainer             *fyne.Container
	urlEntry                  *widget.Entry
	uploadStyleSelect         *widget.Select
	pingStyleSelect           *widget.Select
	uriParamsEntry            *widget.Entry
	tokenStyleSelect          *widget.Select
	tokenEntry                *widget.Entry
	pingPathEntry             *widget.Entry
	pingProbeIDEntry          *widget.Entry
	replayPathEntry           *widget.Entry
	templateNameEntry         *widget.Entry
	liveStyleSelect           *widget.Select
	livePathEntry             *widget.Entry
	renameStyleSelect         *widget.Select
	renamePathEntry           *widget.Entry
	helpLink                  *widget.Hyperlink
	testPingBtn               *widget.Button
	playlistFilterStyleSelect *widget.Select
	playlistListBox           *fyne.Container
	playlistSearchEntry       *widget.Entry
	playlistSearchResults     *fyne.Container
	playlistFilterContainer   *playlistFilterBox

	urlForm                 *widget.FormItem
	uploadStyleForm         *widget.FormItem
	pingStyleForm           *widget.FormItem
	uriParamsForm           *widget.FormItem
	tokenStyleForm          *widget.FormItem
	tokenForm               *widget.FormItem
	pingPathForm            *widget.FormItem
	pingProbeIDForm         *widget.FormItem
	replayPathForm          *widget.FormItem
	templateNameForm        *widget.FormItem
	liveStyleForm           *widget.FormItem
	livePathForm            *widget.FormItem
	renameStyleForm         *widget.FormItem
	renamePathForm          *widget.FormItem
	playlistFilterStyleForm *widget.FormItem
	playlistFilterForm      *widget.FormItem
}

type StorageSettingsPopup struct {
	*Popup

	version           string
	currentWebsite    config.StorageConfig
	testPingGen       int
	playlistAvailable []config.PlaylistFilterEntry
	selectedPlaylists []config.PlaylistFilterEntry

	infoContainer StorageInfoContainer
	btnDelete     *widget.Button
	btnSave       *widget.Button
	btnExport     *widget.Button
	list          *widget.List
	split         *container.Split
	leftPanel     *fyne.Container
	detailPanel   *fyne.Container
	editForm      *widget.Form
}

func NewStorageSettingsPopup(p *Popup, version string) *StorageSettingsPopup {
	logger.FuncDebug()

	wsp := &StorageSettingsPopup{Popup: p, version: version, infoContainer: StorageInfoContainer{}}

	wsp.btnSave = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), wsp.onSaveBtn)
	wsp.btnSave.Importance = widget.HighImportance
	wsp.btnSave.Disable()

	wsp.btnDelete = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), wsp.onDeleteStorageBtn)
	wsp.btnDelete.Importance = widget.DangerImportance
	wsp.btnDelete.Disable()

	wsp.btnExport = widget.NewButtonWithIcon("Export", theme.UploadIcon(), wsp.onExportStorageBtn)
	wsp.btnExport.Disable()

	wsp.editForm = wsp.createInfoContainer()
	scrollableForm := container.NewVScroll(wsp.editForm)
	buttonsBox := container.NewHBox(wsp.btnExport, layout.NewSpacer(), wsp.btnSave, wsp.btnDelete)
	titleBar := container.NewBorder(nil, nil, nil, wsp.infoContainer.testPingBtn, wsp.infoContainer.titleLabel)

	wsp.detailPanel = container.NewBorder(
		container.NewVBox(titleBar, wsp.infoContainer.descContainer, widget.NewSeparator()),
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
	btnImport := widget.NewButtonWithIcon("Import", theme.DownloadIcon(), wsp.onImportStorageBtn)

	leftButtons := container.NewGridWithColumns(2, btnAdd, btnImport)

	wsp.leftPanel = container.NewBorder(nil, leftButtons, nil, nil, wsp.list)
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

	wsp.infoContainer.testPingBtn = widget.NewButtonWithIcon("Test Ping", theme.ViewRefreshIcon(), wsp.onTestPingBtn)

	wsp.infoContainer.playlistFilterStyleSelect = widget.NewSelect(config.PlaylistFilterStyleLabels[:], func(v string) {
		wsp.currentWebsite.PlaylistFilterStyle = config.PlaylistFilterStyleFromLabel(v)
		wsp.reload()
	})

	wsp.infoContainer.playlistListBox = container.NewVBox()

	wsp.infoContainer.playlistSearchResults = container.NewVBox()

	wsp.infoContainer.playlistSearchEntry = widget.NewEntry()
	wsp.infoContainer.playlistSearchEntry.SetPlaceHolder("Search playlists to add...")
	wsp.infoContainer.playlistSearchEntry.OnChanged = func(_ string) {
		wsp.refreshPlaylistSearchResults()
	}

	playlistContent := container.NewVBox(
		wsp.infoContainer.playlistListBox,
		wsp.infoContainer.playlistSearchEntry,
		wsp.infoContainer.playlistSearchResults,
	)
	wsp.infoContainer.playlistFilterContainer = newPlaylistFilterBox(playlistContent, wsp.infoContainer.playlistSearchEntry)

	wsp.infoContainer.liveStyleSelect = widget.NewSelect(config.LiveStyleLabels[:], func(v string) {
		wsp.currentWebsite.LiveStyle = config.LiveStyleFromLabel(v)
		wsp.reload()
	})
	wsp.infoContainer.livePathEntry = widget.NewEntry()

	wsp.infoContainer.replayPathEntry = widget.NewEntry()
	wsp.infoContainer.templateNameEntry = widget.NewEntry()

	wsp.infoContainer.renameStyleSelect = widget.NewSelect(config.RenameStyleLabels[:], func(v string) {
		wsp.currentWebsite.RenameStyle = config.RenameStyleFromLabel(v)
		wsp.reload()
	})
	wsp.infoContainer.renamePathEntry = widget.NewEntry()

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
	wsp.infoContainer.renameStyleForm = widget.NewFormItem("Rename Style", wsp.infoContainer.renameStyleSelect)
	wsp.infoContainer.renamePathForm = widget.NewFormItem("Rename Path", wsp.infoContainer.renamePathEntry)
	wsp.infoContainer.playlistFilterStyleForm = widget.NewFormItem("Playlist Filter", wsp.infoContainer.playlistFilterStyleSelect)
	wsp.infoContainer.playlistFilterForm = widget.NewFormItem("Playlists", wsp.infoContainer.playlistFilterContainer)

	return widget.NewForm(
		wsp.infoContainer.uploadStyleForm,
		wsp.infoContainer.replayPathForm,
		wsp.infoContainer.templateNameForm,
		wsp.infoContainer.renameStyleForm,
		wsp.infoContainer.renamePathForm,
		wsp.infoContainer.urlForm,
		wsp.infoContainer.tokenStyleForm,
		wsp.infoContainer.tokenForm,
		wsp.infoContainer.pingStyleForm,
		wsp.infoContainer.pingPathForm,
		wsp.infoContainer.pingProbeIDForm,
		wsp.infoContainer.uriParamsForm,
		wsp.infoContainer.liveStyleForm,
		wsp.infoContainer.livePathForm,
		wsp.infoContainer.playlistFilterStyleForm,
		wsp.infoContainer.playlistFilterForm,
	)
}

func (wsp *StorageSettingsPopup) currentPreset() *config.StorageConfig {
	logger.FuncDebug()

	if !wsp.currentWebsite.IsPredefined {
		return nil
	}

	return config.STORAGE_PRESET[wsp.currentWebsite.Name]
}

func (wsp *StorageSettingsPopup) reloadPlaylistOptions() {
	logger.FuncDebug()

	knownPlaylists := wsp.appConfig.BehaviorConfig.KnownPlaylists.Get()

	idSet := make(map[int]bool)
	for _, entry := range knownPlaylists {
		idSet[entry.ID] = true
	}
	for _, entry := range wsp.currentWebsite.FilteredPlaylists {
		idSet[entry.ID] = true
	}
	for _, entry := range wsp.selectedPlaylists {
		idSet[entry.ID] = true
	}

	selectedSet := make(map[int]bool, len(wsp.selectedPlaylists))
	for _, entry := range wsp.selectedPlaylists {
		selectedSet[entry.ID] = true
	}

	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	available := make([]config.PlaylistFilterEntry, 0, len(ids))
	for _, id := range ids {
		if selectedSet[id] {
			continue
		}

		available = append(available, config.PlaylistFilterEntry{ID: id, Name: config.PlaylistDisplayName(id, knownPlaylists)})
	}

	wsp.playlistAvailable = available

	wsp.renderSelectedPlaylists()
	wsp.refreshPlaylistSearchResults()
}

func (wsp *StorageSettingsPopup) refreshPlaylistSearchResults() {
	logger.FuncDebug()

	wsp.infoContainer.playlistSearchResults.RemoveAll()

	query := strings.TrimSpace(strings.ToLower(wsp.infoContainer.playlistSearchEntry.Text))
	if query == "" {
		wsp.infoContainer.playlistSearchResults.Refresh()
		return
	}

	matches := make([]config.PlaylistFilterEntry, 0, len(wsp.playlistAvailable))
	for _, entry := range wsp.playlistAvailable {
		if strings.Contains(strings.ToLower(entry.Name), query) {
			matches = append(matches, entry)
		}
	}

	shown := min(len(matches), playlistSearchMaxResults)
	for _, entry := range matches[:shown] {
		id := entry.ID

		addBtn := widget.NewButton(entry.Name, func() {
			wsp.addSelectedPlaylist(id)
		})
		addBtn.Alignment = widget.ButtonAlignLeading
		wsp.infoContainer.playlistSearchResults.Add(addBtn)
	}

	if len(matches) > shown {
		wsp.infoContainer.playlistSearchResults.Add(widget.NewLabel(fmt.Sprintf("+%d more, refine your search", len(matches)-shown)))
	} else if len(matches) == 0 {
		wsp.infoContainer.playlistSearchResults.Add(widget.NewLabel("No matching playlist"))
	}

	wsp.infoContainer.playlistSearchResults.Refresh()
}

func (wsp *StorageSettingsPopup) renderSelectedPlaylists() {
	logger.FuncDebug()

	knownPlaylists := wsp.appConfig.BehaviorConfig.KnownPlaylists.Get()

	sorted := slices.Clone(wsp.selectedPlaylists)
	slices.SortFunc(sorted, func(a, b config.PlaylistFilterEntry) int { return a.ID - b.ID })

	wsp.infoContainer.playlistListBox.RemoveAll()
	for _, entry := range sorted {
		id := entry.ID

		removeBtn := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() {
			wsp.removeSelectedPlaylist(id)
		})
		removeBtn.Importance = widget.LowImportance

		row := container.NewBorder(nil, nil, nil, removeBtn, widget.NewLabel(config.PlaylistDisplayName(id, knownPlaylists)))
		wsp.infoContainer.playlistListBox.Add(row)
	}

	wsp.infoContainer.playlistListBox.Refresh()
}

func (wsp *StorageSettingsPopup) addSelectedPlaylist(id int) {
	logger.FuncDebug()

	if slices.ContainsFunc(wsp.selectedPlaylists, func(entry config.PlaylistFilterEntry) bool { return entry.ID == id }) {
		return
	}

	name := config.PlaylistDisplayName(id, wsp.appConfig.BehaviorConfig.KnownPlaylists.Get())
	wsp.selectedPlaylists = append(wsp.selectedPlaylists, config.PlaylistFilterEntry{ID: id, Name: name})

	wsp.infoContainer.playlistSearchEntry.SetText("")
	wsp.reloadPlaylistOptions()
}

func (wsp *StorageSettingsPopup) removeSelectedPlaylist(id int) {
	logger.FuncDebug()

	wsp.selectedPlaylists = slices.DeleteFunc(wsp.selectedPlaylists, func(entry config.PlaylistFilterEntry) bool {
		return entry.ID == id
	})
	wsp.reloadPlaylistOptions()
}

func (wsp *StorageSettingsPopup) reloadOptions() {
	logger.FuncDebug()

	preset := wsp.currentPreset()
	if preset == nil {
		wsp.reloadPlaylistOptions()

		wsp.infoContainer.uploadStyleSelect.SetOptions(config.UploadStyleLabels[:])
		wsp.infoContainer.pingStyleSelect.SetOptions(config.PingStyleLabels[:])
		wsp.infoContainer.tokenStyleSelect.SetOptions(config.TokenStyleLabels[:])
		wsp.infoContainer.liveStyleSelect.SetOptions(config.LiveStyleLabels[:])
		wsp.infoContainer.renameStyleSelect.SetOptions(config.RenameStyleLabels[:])
		wsp.infoContainer.playlistFilterStyleSelect.SetOptions(config.PlaylistFilterStyleLabels[:])
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
	wsp.infoContainer.renameStyleSelect.SetOptions([]string{
		config.RenameDisabled.Label(),
		preset.RenameStyle.Label(),
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
	wsp.selectedPlaylists = slices.Clone(wsp.currentWebsite.FilteredPlaylists)
	wsp.infoContainer.playlistSearchEntry.SetText("")
	wsp.resetTestPingBtn()
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
	wsp.infoContainer.renameStyleSelect.SetSelected(wsp.currentWebsite.RenameStyle.Label())
	wsp.infoContainer.renamePathEntry.SetText(wsp.currentWebsite.RenamePath)
	wsp.infoContainer.playlistFilterStyleSelect.SetSelected(wsp.currentWebsite.PlaylistFilterStyle.Label())

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
	wsp.infoContainer.testPingBtn.Show()
	wsp.infoContainer.playlistFilterStyleForm.Widget.Show()
	wsp.infoContainer.playlistFilterForm.Widget.Show()
	wsp.infoContainer.replayPathForm.Widget.Show()
	wsp.infoContainer.templateNameForm.Widget.Show()
	wsp.infoContainer.liveStyleForm.Widget.Show()
	wsp.infoContainer.livePathForm.Widget.Show()
	wsp.infoContainer.renameStyleForm.Widget.Show()
	wsp.infoContainer.renamePathForm.Widget.Show()

	wsp.infoContainer.helpLink.Text = wsp.currentWebsite.HelpText
	wsp.infoContainer.helpLink.Refresh()

	if _, ok := parseHelpURL(wsp.currentWebsite.HelpURL); ok {
		wsp.infoContainer.descContainer.Show()
	} else {
		wsp.infoContainer.descContainer.Hide()
	}

	switch wsp.currentWebsite.TokenStyle {
	case config.RawToken, config.BearerToken:
	default:
		fallthrough
	case config.NoToken:
		wsp.infoContainer.tokenForm.Widget.Hide()
	}

	switch wsp.currentWebsite.PingStyle {
	case config.PingNotFoundIsValid:
	case config.PingRequiresOK:
		wsp.infoContainer.pingProbeIDForm.Widget.Hide()
	default:
		fallthrough
	case config.PingDisabled:
		wsp.infoContainer.pingPathForm.Widget.Hide()
		wsp.infoContainer.pingProbeIDForm.Widget.Hide()
		wsp.infoContainer.testPingBtn.Hide()
	}

	switch wsp.currentWebsite.LiveStyle {
	case config.LiveEnabled:
	default:
		fallthrough
	case config.LiveDisabled:
		wsp.infoContainer.livePathForm.Widget.Hide()
	}

	if !wsp.appConfig.BehaviorConfig.SendLiveStat.Get() {
		wsp.infoContainer.liveStyleForm.Widget.Hide()
		wsp.infoContainer.livePathForm.Widget.Hide()
	}

	switch wsp.currentWebsite.RenameStyle {
	case config.RenamePatchTitle:
	default:
		fallthrough
	case config.RenameDisabled:
		wsp.infoContainer.renamePathForm.Widget.Hide()
	}

	switch wsp.currentWebsite.UploadStyle {
	case config.MultipartUpload:
	case config.LocalFileCopy:
		wsp.infoContainer.urlForm.Widget.Hide()
		wsp.infoContainer.uriParamsForm.Widget.Hide()
		wsp.infoContainer.tokenStyleForm.Widget.Hide()
		wsp.infoContainer.tokenForm.Widget.Hide()
		wsp.infoContainer.pingStyleForm.Widget.Hide()
		wsp.infoContainer.pingPathForm.Widget.Hide()
		wsp.infoContainer.pingProbeIDForm.Widget.Hide()
		wsp.infoContainer.testPingBtn.Hide()
		wsp.infoContainer.liveStyleForm.Widget.Hide()
		wsp.infoContainer.livePathForm.Widget.Hide()
		wsp.infoContainer.renameStyleForm.Widget.Hide()
		wsp.infoContainer.renamePathForm.Widget.Hide()
	case config.PresignedSessionUpload:
		wsp.infoContainer.templateNameForm.Widget.Hide()
		wsp.infoContainer.renameStyleForm.Widget.Hide()
		wsp.infoContainer.renamePathForm.Widget.Hide()
	default:
		fallthrough
	case config.UploadDisabled:
		wsp.infoContainer.replayPathForm.Widget.Hide()
		wsp.infoContainer.templateNameForm.Widget.Hide()
		wsp.infoContainer.renameStyleForm.Widget.Hide()
		wsp.infoContainer.renamePathForm.Widget.Hide()
	}

	if wsp.currentWebsite.IsPredefined {
		if wsp.currentWebsite.IsPrimary {
			wsp.infoContainer.templateNameForm.Widget.Hide()
			wsp.infoContainer.playlistFilterForm.Widget.Hide()
		}
	}
}

func (wsp *StorageSettingsPopup) reloadEnable() {
	logger.FuncDebug()

	wsp.btnSave.Enable()
	wsp.btnDelete.Enable()
	wsp.btnExport.Enable()

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
	wsp.infoContainer.renameStyleSelect.Enable()
	wsp.infoContainer.renamePathEntry.Enable()
	wsp.infoContainer.playlistFilterStyleSelect.Enable()
	wsp.infoContainer.playlistFilterContainer.Enable()

	if !wsp.appConfig.BehaviorConfig.SendLiveStat.Get() {
		wsp.infoContainer.liveStyleSelect.Disable()
		wsp.infoContainer.livePathEntry.Disable()
	}

	switch wsp.currentWebsite.UploadStyle {
	case config.MultipartUpload:
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
		wsp.infoContainer.renameStyleSelect.Disable()
		wsp.infoContainer.renamePathEntry.Disable()
	case config.PresignedSessionUpload:
		wsp.infoContainer.templateNameEntry.Disable()
		wsp.infoContainer.renameStyleSelect.Disable()
		wsp.infoContainer.renamePathEntry.Disable()
	default:
		fallthrough
	case config.UploadDisabled:
		wsp.infoContainer.replayPathEntry.Disable()
		wsp.infoContainer.templateNameEntry.Disable()
		wsp.infoContainer.renameStyleSelect.Disable()
		wsp.infoContainer.renamePathEntry.Disable()
	}

	if wsp.currentWebsite.IsPredefined {
		wsp.btnDelete.Disable()

		wsp.infoContainer.urlEntry.Disable()
		wsp.infoContainer.tokenStyleSelect.Disable()
		wsp.infoContainer.pingPathEntry.Disable()
		wsp.infoContainer.pingProbeIDEntry.Disable()
		wsp.infoContainer.livePathEntry.Disable()
		wsp.infoContainer.liveStyleSelect.Disable()
		wsp.infoContainer.renamePathEntry.Disable()

		if wsp.currentWebsite.IsPrimary {
			wsp.btnSave.Disable()
			wsp.btnExport.Disable()
			wsp.infoContainer.uploadStyleSelect.Disable()
			wsp.infoContainer.pingStyleSelect.Disable()
			wsp.infoContainer.uriParamsEntry.Disable()
			wsp.infoContainer.tokenEntry.Disable()
			wsp.infoContainer.templateNameEntry.Disable()
			wsp.infoContainer.replayPathEntry.Disable()
			wsp.infoContainer.renameStyleSelect.Disable()
			wsp.infoContainer.playlistFilterStyleSelect.Disable()
			wsp.infoContainer.playlistFilterContainer.Disable()
		} else if preset := wsp.currentPreset(); preset != nil && preset.UploadStyle != config.LocalFileCopy {
			wsp.infoContainer.replayPathEntry.Disable()
		}
	}

	wsp.infoContainer.testPingBtn.Enable()
	if wsp.currentWebsite.PingStyle == config.PingDisabled || wsp.currentWebsite.UploadStyle == config.LocalFileCopy {
		wsp.infoContainer.testPingBtn.Disable()
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
	wsp.infoContainer.renameStyleSelect.ClearSelected()
	wsp.infoContainer.renamePathEntry.SetText("")
	wsp.infoContainer.playlistFilterStyleSelect.ClearSelected()
	wsp.infoContainer.playlistSearchEntry.SetText("")
	wsp.selectedPlaylists = nil
	wsp.reloadPlaylistOptions()
	wsp.resetTestPingBtn()

	wsp.btnDelete.Disable()
	wsp.btnSave.Disable()
	wsp.btnExport.Disable()
	wsp.infoContainer.testPingBtn.Disable()
}

func (wsp *StorageSettingsPopup) formToConfig() config.StorageConfig {
	logger.FuncDebug()

	cfg := wsp.currentWebsite

	cfg.URL = wsp.infoContainer.urlEntry.Text
	cfg.UploadStyle = config.UploadStyleFromLabel(wsp.infoContainer.uploadStyleSelect.Selected)
	cfg.PingStyle = config.PingStyleFromLabel(wsp.infoContainer.pingStyleSelect.Selected)

	cfg.TokenStyle = config.TokenStyleFromLabel(wsp.infoContainer.tokenStyleSelect.Selected)
	cfg.Token = wsp.infoContainer.tokenEntry.Text
	cfg.PingPath = wsp.infoContainer.pingPathEntry.Text
	cfg.PingProbeID = wsp.infoContainer.pingProbeIDEntry.Text
	cfg.ReplayPath = wsp.infoContainer.replayPathEntry.Text
	cfg.TemplateName = wsp.infoContainer.templateNameEntry.Text
	cfg.RenameStyle = config.RenameStyleFromLabel(wsp.infoContainer.renameStyleSelect.Selected)
	cfg.RenamePath = wsp.infoContainer.renamePathEntry.Text

	if wsp.appConfig.BehaviorConfig.SendLiveStat.Get() {
		cfg.LiveStyle = config.LiveStyleFromLabel(wsp.infoContainer.liveStyleSelect.Selected)
		cfg.LivePath = wsp.infoContainer.livePathEntry.Text
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
	cfg.URIParams = newParams

	cfg.PlaylistFilterStyle = config.PlaylistFilterStyleFromLabel(wsp.infoContainer.playlistFilterStyleSelect.Selected)

	switch cfg.PlaylistFilterStyle {
	case config.PlaylistFilterWhitelist, config.PlaylistFilterBlacklist:
		sorted := slices.Clone(wsp.selectedPlaylists)
		slices.SortFunc(sorted, func(a, b config.PlaylistFilterEntry) int { return a.ID - b.ID })
		cfg.FilteredPlaylists = sorted
	default:
		cfg.FilteredPlaylists = nil
	}

	return cfg
}

func (wsp *StorageSettingsPopup) runPingTest(cfg config.StorageConfig, onDone func(err error)) {
	logger.FuncDebug()

	go func() {
		err := upload.NewWebsite(&cfg, wsp.version, nil).Ping()

		fyne.Do(func() {
			onDone(err)
		})
	}()
}

func (wsp *StorageSettingsPopup) resetTestPingBtn() {
	logger.FuncDebug()

	wsp.testPingGen++

	wsp.infoContainer.testPingBtn.Importance = widget.MediumImportance
	wsp.infoContainer.testPingBtn.SetIcon(theme.ViewRefreshIcon())
	wsp.infoContainer.testPingBtn.SetText("Test Ping")
}

func (wsp *StorageSettingsPopup) testPingBtnBusy() int {
	logger.FuncDebug()

	wsp.testPingGen++
	gen := wsp.testPingGen

	wsp.infoContainer.testPingBtn.Importance = widget.MediumImportance
	wsp.infoContainer.testPingBtn.SetIcon(theme.ViewRefreshIcon())
	wsp.infoContainer.testPingBtn.SetText("Testing...")
	wsp.infoContainer.testPingBtn.Disable()

	return gen
}

func (wsp *StorageSettingsPopup) testPingBtnResult(gen int, err error) {
	logger.FuncDebug()

	if gen != wsp.testPingGen {
		return
	}

	if err != nil {
		wsp.infoContainer.testPingBtn.Importance = widget.DangerImportance
		wsp.infoContainer.testPingBtn.SetIcon(theme.ErrorIcon())
		wsp.infoContainer.testPingBtn.SetText("Ping Failed")
	} else {
		wsp.infoContainer.testPingBtn.Importance = widget.SuccessImportance
		wsp.infoContainer.testPingBtn.SetIcon(theme.ConfirmIcon())
		wsp.infoContainer.testPingBtn.SetText("Ping OK")
	}

	wsp.reloadEnable()

	time.AfterFunc(testPingResultDisplayDuration, func() {
		fyne.Do(func() {
			if gen != wsp.testPingGen {
				return
			}

			wsp.resetTestPingBtn()
			wsp.reloadEnable()
		})
	})
}

func (wsp *StorageSettingsPopup) onTestPingBtn() {
	logger.FuncDebug()

	cfg := wsp.formToConfig()
	gen := wsp.testPingBtnBusy()

	wsp.runPingTest(cfg, func(err error) {
		wsp.testPingBtnResult(gen, err)

		if err != nil && gen == wsp.testPingGen {
			dialog.ShowError(err, wsp.parentWindow)
		}
	})
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

	cfg := wsp.formToConfig()

	if cfg.PingStyle == config.PingDisabled {
		*site = cfg
		wsp.persistSave(selectedIndex)
		return
	}

	wsp.btnSave.Disable()
	gen := wsp.testPingBtnBusy()

	wsp.runPingTest(cfg, func(err error) {
		wsp.testPingBtnResult(gen, err)

		if err != nil {
			if gen != wsp.testPingGen {
				return
			}

			dialog.ShowConfirm("Ping test failed",
				fmt.Sprintf("The ping test failed before saving:\n\n%s\n\nSave anyway?", err),
				func(confirmed bool) {
					if !confirmed {
						return
					}

					*site = cfg
					wsp.persistSave(selectedIndex)
				}, wsp.parentWindow)
			return
		}

		*site = cfg
		wsp.persistSave(selectedIndex)
	})
}

func (wsp *StorageSettingsPopup) persistSave(selectedIndex int) {
	logger.FuncDebug()

	storages := wsp.appConfig.StorageSettings.Get()
	wsp.appConfig.StorageSettings.Set(storages)
	wsp.currentWebsite = *storages[min(selectedIndex, len(storages)-1)]

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
				RenameStyle:  config.RenameDisabled,
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

func sanitizeFileName(name string) string {
	logger.FuncDebug()

	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_", " ", "_",
	)

	sanitized := replacer.Replace(strings.TrimSpace(name))
	if sanitized == "" {
		return "storage"
	}

	return sanitized
}

func (wsp *StorageSettingsPopup) onExportStorageBtn() {
	logger.FuncDebug()

	exported := wsp.currentWebsite
	exported.Token = ""

	data, err := json.MarshalIndent([]*config.StorageConfig{&exported}, "", "  ")
	if err != nil {
		dialog.ShowError(err, wsp.parentWindow)
		return
	}

	saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil || err != nil {
			return
		}
		defer writer.Close()

		if _, err := writer.Write(data); err != nil {
			logger.Rlogger.Error("Error writing storage settings export:", slog.Any("err", err))
			dialog.ShowInformation("Error", "Failed to export storage settings!\nError writing file", wsp.parentWindow)
			return
		}

		dialog.ShowInformation("Success", "Storage settings exported successfully!\nNote: the token is not included, it must be re-entered.", wsp.parentWindow)
	}, wsp.parentWindow)

	saveDialog.SetFileName(sanitizeFileName(wsp.currentWebsite.Name) + "_rockpload_storage_settings.json")
	saveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	saveDialog.Show()
}

func (wsp *StorageSettingsPopup) onImportStorageBtn() {
	logger.FuncDebug()

	openDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if reader == nil || err != nil {
			return
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			logger.Rlogger.Error("Error reading storage settings import:", slog.Any("err", err))
			dialog.ShowInformation("Error", "Failed to import storage settings!\nError reading file", wsp.parentWindow)
			return
		}

		var imported []*config.StorageConfig
		if err := json.Unmarshal(data, &imported); err != nil {
			logger.Rlogger.Error("Error parsing storage settings import:", slog.Any("err", err))
			dialog.ShowInformation("Error", "Failed to import storage settings!\nInvalid file format", wsp.parentWindow)
			return
		}

		storages := wsp.appConfig.StorageSettings.Get()

		added, updated := 0, 0
		for _, importedSite := range imported {
			if importedSite.IsPrimary || importedSite.Name == "" {
				continue
			}

			existingIndex := slices.IndexFunc(storages, func(c *config.StorageConfig) bool { return c.Name == importedSite.Name })
			if existingIndex == -1 {
				storages = append(storages, importedSite)
				added++
				continue
			}

			importedSite.Token = storages[existingIndex].Token
			storages[existingIndex] = importedSite
			updated++
		}

		if added == 0 && updated == 0 {
			dialog.ShowInformation("Nothing to import", "No valid storage settings were found in the file.", wsp.parentWindow)
			return
		}

		wsp.appConfig.StorageSettings.Set(storages)
		wsp.list.Refresh()
		wsp.onSelected(wsp.appConfig.BehaviorConfig.SelectedStorageId.Get())

		dialog.ShowInformation("Success", fmt.Sprintf("Storage settings imported successfully!\nAdded: %d, Updated: %d", added, updated), wsp.parentWindow)
	}, wsp.parentWindow)

	openDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	openDialog.Show()
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
