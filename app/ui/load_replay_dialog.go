package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const EVENT_CLICK_LOAD_REPLAY_IN_GAME tools.EventType = "click_load_replay_in_game"

type LoadReplayMode string

const (
	LoadReplayFileName     LoadReplayMode = "FileName: name (copied to Demos)"
	LoadReplayTempPath     LoadReplayMode = "Path: temp copy"
	LoadReplayOriginalPath LoadReplayMode = "Path: original file"
)

var loadReplayModes = []LoadReplayMode{LoadReplayTempPath, LoadReplayOriginalPath, LoadReplayFileName}

type LoadReplayRequest struct {
	FilePath string
	Mode     LoadReplayMode
	OnResult func(sent string, err error)
}

func (g *GUI) showLoadReplayInGameDialog() {
	logger.FuncDebug()

	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, g.window)
			return
		}

		if reader == nil {
			return
		}

		filePath := reader.URI().Path()
		reader.Close()

		g.showLoadReplayModeDialog(filePath)
	}, g.window)

	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".replay"}))
	fileDialog.Show()
}

func (g *GUI) showLoadReplayModeDialog(filePath string) {
	logger.FuncDebug()

	fileLabel := widget.NewLabel(filePath)
	fileLabel.Wrapping = fyne.TextWrapBreak
	fileLabel.TextStyle = fyne.TextStyle{Italic: true}

	resultLabel := widget.NewLabel("")
	resultLabel.Wrapping = fyne.TextWrapBreak

	modeLabels := make([]string, 0, len(loadReplayModes))
	for _, mode := range loadReplayModes {
		modeLabels = append(modeLabels, string(mode))
	}

	modeSelect := widget.NewSelect(modeLabels, nil)
	modeSelect.SetSelected(modeLabels[0])

	var sendBtn *widget.Button
	sendBtn = widget.NewButton("Send LoadReplay", func() {
		sendBtn.Disable()
		resultLabel.SetText("Sending...")

		g.EventManager.Notify(EVENT_CLICK_LOAD_REPLAY_IN_GAME, LoadReplayRequest{
			FilePath: filePath,
			Mode:     LoadReplayMode(modeSelect.Selected),
			OnResult: func(sent string, err error) {
				fyne.Do(func() {
					sendBtn.Enable()

					if err != nil {
						resultLabel.SetText("Failed: " + err.Error() + "\n" + sent)
						return
					}

					resultLabel.SetText("Sent: " + sent)
				})
			},
		})
	})
	sendBtn.Importance = widget.HighImportance

	content := container.NewVBox(fileLabel, modeSelect, sendBtn, resultLabel)

	modeDialog := dialog.NewCustom("Load replay in Rocket League", "Close", content, g.window)
	modeDialog.Resize(fyne.NewSize(560, 280))
	modeDialog.Show()
}
