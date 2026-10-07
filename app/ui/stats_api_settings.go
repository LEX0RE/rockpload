package ui

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/LEX0RE/rockpload/app/rlgame"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	installNotFoundText = "Rocket League installation not found.\nLaunch the game once so Rockpload can detect it, or choose the installation folder with \"Browse\"."
	restartGameText     = "Saved! Restart Rocket League to apply the changes."
)

type StatsAPISettingPopup struct {
	*Popup

	locator *rlgame.Locator

	installs      []rlgame.Install
	installSelect *widget.Select
	forgetBtn     *widget.Button
	pathLabel     *widget.Label
	statusLabel   *widget.Label

	rawCheck   *widget.Check
	editorBox  *fyne.Container
	formBox    *fyne.Container
	rawEntry   *widget.Entry
	actionsBox *fyne.Container
	saveBtn    *widget.Button

	file        *rlgame.StatsAPIFile
	formEntries map[string]*widget.Entry
	// Avoid the raw check callback when the check is changed by the code
	updatingRawCheck bool
}

func NewStatsAPISettingPopup(p *Popup, locator *rlgame.Locator) *StatsAPISettingPopup {
	logger.FuncDebug()

	sp := &StatsAPISettingPopup{
		Popup:       p,
		locator:     locator,
		formEntries: make(map[string]*widget.Entry),
	}

	sp.installSelect = widget.NewSelect([]string{}, func(string) { sp.loadFile() })

	browseBtn := widget.NewButton("Browse", sp.browseInstall)
	sp.forgetBtn = widget.NewButton("Forget manual folder", func() {
		sp.locator.SetManualDir("")
		sp.reload()
	})
	sp.forgetBtn.Hide()

	sp.pathLabel = widget.NewLabel("")
	sp.pathLabel.Wrapping = fyne.TextWrapBreak
	sp.pathLabel.TextStyle = fyne.TextStyle{Italic: true}

	sp.statusLabel = widget.NewLabel("")
	sp.statusLabel.Wrapping = fyne.TextWrapWord
	sp.statusLabel.Hide()

	sp.rawCheck = widget.NewCheck("Edit raw file", sp.onRawToggle)

	sp.formBox = container.NewVBox()

	sp.rawEntry = widget.NewMultiLineEntry()
	sp.rawEntry.TextStyle = fyne.TextStyle{Monospace: true}
	sp.rawEntry.Wrapping = fyne.TextWrapOff
	sp.rawEntry.SetMinRowsVisible(10)

	openEditorBtn := widget.NewButton("Open in text editor", func() {
		sp.openInEditor(false)
	})
	showFolderBtn := widget.NewButton("Show in folder", func() {
		if err := tools.ShowInFolder(sp.currentPath()); err != nil {
			dialog.ShowError(err, sp.parentWindow)
		}
	})
	reloadBtn := widget.NewButton("Reload", sp.loadFile)

	sp.saveBtn = widget.NewButton("Save", sp.save)
	sp.saveBtn.Importance = widget.HighImportance

	sp.actionsBox = container.NewHBox(sp.rawCheck, layout.NewSpacer(), reloadBtn, showFolderBtn, openEditorBtn)

	installBox := container.NewBorder(nil, nil, nil, container.NewHBox(browseBtn, sp.forgetBtn), sp.installSelect)
	sp.editorBox = container.NewVBox(sp.pathLabel, sp.statusLabel, sp.formBox, sp.rawEntry)
	editorScroll := container.NewVScroll(sp.editorBox)
	bottom := container.NewVBox(widget.NewSeparator(), sp.actionsBox, sp.saveBtn)

	sp.SetContent(container.NewBorder(installBox, bottom, nil, nil, editorScroll))
	sp.popup.Resize(fyne.NewSize(620, 440))

	return sp
}

func (sp *StatsAPISettingPopup) Show() {
	logger.FuncDebug()

	sp.reload()

	sp.Popup.Show()
}

// reload detects the installs again, keeping the selected one when it still exists.
func (sp *StatsAPISettingPopup) reload() {
	logger.FuncDebug()

	previous := sp.installSelect.Selected

	sp.installs = sp.locator.Refresh()

	labels := make([]string, 0, len(sp.installs))
	for _, install := range sp.installs {
		labels = append(labels, install.Label())
	}

	sp.installSelect.SetOptions(labels)

	if sp.locator.ManualDir() != "" {
		sp.forgetBtn.Show()
	} else {
		sp.forgetBtn.Hide()
	}

	if len(labels) == 0 {
		sp.installSelect.ClearSelected()
		return
	}

	selected := labels[0]
	for _, label := range labels {
		if label == previous {
			selected = label
		}
	}

	// Always calls loadFile, even when the selection did not change
	sp.installSelect.SetSelected(selected)
}

func (sp *StatsAPISettingPopup) selectedInstall() (rlgame.Install, bool) {
	index := sp.installSelect.SelectedIndex()
	if index < 0 || index >= len(sp.installs) {
		return rlgame.Install{}, false
	}

	return sp.installs[index], true
}

func (sp *StatsAPISettingPopup) currentPath() string {
	install, _ := sp.selectedInstall()
	return install.StatsAPIConfigPath()
}

func (sp *StatsAPISettingPopup) loadFile() {
	logger.FuncDebug()

	sp.file = nil
	sp.formBox.RemoveAll()
	sp.formEntries = make(map[string]*widget.Entry)

	install, ok := sp.selectedInstall()
	if !ok {
		sp.showUnavailable(installNotFoundText, "")
		return
	}

	path := install.StatsAPIConfigPath()
	file, err := rlgame.LoadStatsAPIFile(path)
	if err != nil {
		logger.Rlogger.Error("Failed to load StatsAPI file", slog.String("Path", path), slog.Any("err", err))

		message := "Unable to read the StatsAPI file: " + err.Error()
		if errors.Is(err, fs.ErrNotExist) {
			message = "The StatsAPI file does not exist in this installation. Update Rocket League or choose another installation."
		}

		sp.showUnavailable(message, path)
		return
	}

	sp.file = file
	sp.pathLabel.SetText(path)
	sp.actionsBox.Show()

	if file.CanUseForm() {
		sp.setRawCheck(false, true)
		sp.showForm()
	} else {
		sp.setRawCheck(true, false)
		sp.showRaw()
		sp.setStatus("This file cannot be edited with the form: " + file.Problem)
	}
}

func (sp *StatsAPISettingPopup) showUnavailable(message string, path string) {
	logger.FuncDebug()

	sp.pathLabel.SetText(path)
	sp.setStatus(message)
	sp.setRawCheck(false, false)
	sp.rawEntry.Hide()
	sp.formBox.Hide()
	sp.saveBtn.Disable()

	if path != "" {
		sp.actionsBox.Show()
	} else {
		sp.actionsBox.Hide()
	}
}

func (sp *StatsAPISettingPopup) setStatus(status string) {
	sp.statusLabel.SetText(status)

	if status == "" {
		sp.statusLabel.Hide()
	} else {
		sp.statusLabel.Show()
	}

	if sp.editorBox != nil {
		sp.editorBox.Refresh()
	}
}

func (sp *StatsAPISettingPopup) setRawCheck(checked bool, enabled bool) {
	sp.updatingRawCheck = true
	sp.rawCheck.SetChecked(checked)
	sp.updatingRawCheck = false

	if enabled {
		sp.rawCheck.Enable()
	} else {
		sp.rawCheck.Disable()
	}
}

func (sp *StatsAPISettingPopup) showForm() {
	logger.FuncDebug()

	sp.formBox.RemoveAll()
	sp.formEntries = make(map[string]*widget.Entry)

	for _, settingEntry := range sp.file.Entries {
		key := settingEntry.Key

		entry := widget.NewEntry()
		entry.SetText(settingEntry.Value)
		entry.Validator = func(text string) error {
			return rlgame.ValidateStatsAPIValue(key, text)
		}
		entry.OnChanged = func(string) { sp.refreshStatus() }

		keyLabel := widget.NewLabelWithStyle(key, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		sp.formBox.Add(keyLabel)

		// Comments are shown as written in the file so new keys are explained by Rocket League itself
		if settingEntry.Comment != "" {
			hintLabel := widget.NewLabel(settingEntry.Comment)
			hintLabel.Wrapping = fyne.TextWrapWord
			hintLabel.SizeName = theme.SizeNameCaptionText
			sp.formBox.Add(hintLabel)
		}

		sp.formEntries[key] = entry
		sp.formBox.Add(entry)
	}

	sp.formBox.Show()
	sp.rawEntry.Hide()
	sp.saveBtn.Enable()
	sp.refreshStatus()
}

func (sp *StatsAPISettingPopup) showRaw() {
	logger.FuncDebug()

	sp.formBox.Hide()
	sp.rawEntry.SetText(sp.file.RawText())
	sp.rawEntry.Show()

	if sp.file.RawEditable {
		sp.rawEntry.Enable()
		sp.saveBtn.Enable()
	} else {
		sp.rawEntry.Disable()
		sp.saveBtn.Disable()
	}

	sp.setStatus("")
}

func (sp *StatsAPISettingPopup) onRawToggle(raw bool) {
	logger.FuncDebug()

	if sp.updatingRawCheck || sp.file == nil {
		return
	}

	if raw {
		if err := sp.applyForm(); err != nil {
			dialog.ShowError(err, sp.parentWindow)
			sp.setRawCheck(false, true)
			return
		}

		sp.showRaw()
		return
	}

	if err := sp.file.SetRawText(sp.rawEntry.Text); err != nil {
		dialog.ShowError(err, sp.parentWindow)
		sp.setRawCheck(true, true)
		return
	}

	if !sp.file.CanUseForm() {
		dialog.ShowInformation("Raw edit needed", "This content cannot be edited with the form:\n"+sp.file.Problem, sp.parentWindow)
		sp.setRawCheck(true, true)
		return
	}

	sp.showForm()
}

// refreshStatus warns when the form values will prevent Rockpload from receiving live stats.
func (sp *StatsAPISettingPopup) refreshStatus() {
	logger.FuncDebug()

	sp.setStatus(statsAPIWarning(func(key string) (string, bool) {
		entry, ok := sp.formEntries[key]
		if !ok {
			return "", false
		}
		return entry.Text, true
	}))
}

func (sp *StatsAPISettingPopup) applyForm() error {
	logger.FuncDebug()

	for _, settingEntry := range sp.file.Entries {
		entry, ok := sp.formEntries[settingEntry.Key]
		if !ok {
			continue
		}

		if err := entry.Validate(); err != nil {
			return err
		}

		if err := sp.file.Set(settingEntry.Key, entry.Text); err != nil {
			return err
		}
	}

	return sp.file.Validate()
}

func (sp *StatsAPISettingPopup) save() {
	logger.FuncDebug()

	if sp.file == nil {
		return
	}

	var err error
	if sp.rawCheck.Checked {
		err = sp.file.SetRawText(sp.rawEntry.Text)
		if err == nil && sp.file.CanUseForm() {
			err = sp.file.Validate()
		}
	} else {
		err = sp.applyForm()
	}

	if err != nil {
		dialog.ShowError(err, sp.parentWindow)
		return
	}

	err = sp.file.Save()

	switch {
	case err == nil:
		sp.setStatus(restartGameText)

	case errors.Is(err, rlgame.ErrFileChanged):
		dialog.ShowInformation("File changed", "The StatsAPI file was modified by another program (a game update or a text editor).\nIt will be reloaded, apply your changes again.", sp.parentWindow)
		sp.loadFile()

	case errors.Is(err, fs.ErrPermission):
		logger.Rlogger.Warn("No permission to write StatsAPI file", slog.String("Path", sp.file.Path), slog.Any("err", err))

		message := "Rockpload is not allowed to modify this file.\nDo you want to open it in a text editor to change it yourself?"
		if tools.CanElevateEditor() {
			message = "Rockpload is not allowed to modify this file.\nDo you want to open it in Notepad as administrator to change it yourself?"
		}

		dialog.ShowConfirm("Permission denied", message, func(confirmed bool) {
			if confirmed {
				sp.openInEditor(tools.CanElevateEditor())
			}
		}, sp.parentWindow)

	default:
		logger.Rlogger.Error("Failed to save StatsAPI file", slog.Any("err", err))
		dialog.ShowError(err, sp.parentWindow)
	}
}

func (sp *StatsAPISettingPopup) openInEditor(elevated bool) {
	logger.FuncDebug()

	if err := tools.OpenInTextEditor(sp.currentPath(), elevated); err != nil {
		logger.Rlogger.Error("Failed to open StatsAPI file in editor", slog.Any("err", err))
		dialog.ShowError(err, sp.parentWindow)
		return
	}

	sp.setStatus("Press \"Reload\" after saving the file in the text editor.")
}

func (sp *StatsAPISettingPopup) browseInstall() {
	logger.FuncDebug()

	folderDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, sp.parentWindow)
			return
		}

		if uri == nil {
			return
		}

		dir := uri.Path()
		if !rlgame.IsInstallDir(dir) {
			dialog.ShowInformation("Invalid folder", "This folder is not a Rocket League installation.\nChoose the folder containing \"TAGame\" (usually named \"rocketleague\").", sp.parentWindow)
			return
		}

		sp.locator.SetManualDir(dir)
		sp.installSelect.ClearSelected()
		sp.reload()
	}, sp.parentWindow)

	folderDialog.Show()
}

// StatsAPIStatus returns a warning about the active install StatsAPI config, empty when everything is fine.
func StatsAPIStatus(locator *rlgame.Locator) string {
	logger.FuncDebug()

	install, ok := locator.Active()
	if !ok {
		return "Rocket League installation not found"
	}

	file, err := rlgame.LoadStatsAPIFile(install.StatsAPIConfigPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "StatsAPI file not found"
		}
		return "Unable to read the StatsAPI file"
	}

	return statsAPIWarning(file.Get)
}

func statsAPIWarning(get func(key string) (string, bool)) string {
	if value, ok := get(rlgame.KeyPacketSendRate); ok && rlgame.IsSendRateDisabled(value) {
		return "StatsAPI is disabled in Rocket League (PacketSendRate=0). Set it to " + strconv.Itoa(rlgame.RecommendedSendRate) + " to send live stats."
	}

	// A missing key uses the game default, which is enabled
	isDisabled := func(key string) bool {
		value, ok := get(key)
		return ok && strings.TrimSpace(value) == "0"
	}

	if isDisabled(rlgame.KeyPort) && isDisabled(rlgame.KeyWebPort) {
		return "Both StatsAPI sockets are disabled in Rocket League (Port=0 and WebPort=0)."
	}

	return ""
}
