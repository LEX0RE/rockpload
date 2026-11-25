package ui

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const updateProgressRefreshInterval = 100 * time.Millisecond

type UpdateApplier func(onProgress func(downloaded int64, total int64)) error

type UpdatePopup struct {
	*Popup

	apply     UpdateApplier
	onApplied func()
	onLater   func()

	busy bool

	status           *widget.Label
	progress         *widget.ProgressBar
	progressInfinite *widget.ProgressBarInfinite
	buttons          *fyne.Container
	btnUpdate        *widget.Button
}

func NewUpdatePopup(p *Popup, newVersion string, source string, apply UpdateApplier, onApplied func(), onLater func()) *UpdatePopup {
	logger.FuncDebug()

	versionInfo := ""
	if newVersion != "" {
		versionInfo = " (v" + newVersion + ")"
	}

	sourceInfo := ""
	if source != "" {
		sourceInfo = " from " + source
	}

	up := &UpdatePopup{Popup: p, apply: apply, onApplied: onApplied, onLater: onLater}

	up.onClosed = func() {
		if !up.busy {
			up.later()
		}
	}

	description := widget.NewLabel("A new update" + versionInfo + " is ready to be installed" + sourceInfo)
	description.Wrapping = fyne.TextWrapWord

	up.status = widget.NewLabel("")
	up.status.Wrapping = fyne.TextWrapWord
	up.status.Hide()

	up.progress = widget.NewProgressBar()
	up.progress.Hide()

	up.progressInfinite = widget.NewProgressBarInfinite()
	up.progressInfinite.Stop()
	up.progressInfinite.Hide()

	btnLater := widget.NewButton("Later", up.later)

	up.btnUpdate = widget.NewButtonWithIcon("Update now", theme.ConfirmIcon(), up.startUpdate)
	up.btnUpdate.Importance = widget.HighImportance

	up.buttons = container.NewCenter(container.NewHBox(btnLater, up.btnUpdate))

	content := container.NewVBox(
		description,
		up.status,
		container.NewStack(up.progress, up.progressInfinite),
		layout.NewSpacer(),
		up.buttons,
	)

	up.popup.Resize(fyne.NewSize(450, 200))

	up.SetContent(content)

	return up
}

func (up *UpdatePopup) later() {
	logger.FuncDebug()

	up.Hide()

	if up.onLater != nil {
		up.onLater()
	}
}

func (up *UpdatePopup) startUpdate() {
	logger.FuncDebug()

	if up.busy {
		return
	}

	up.busy = true
	up.buttons.Hide()
	up.setStatus("Downloading update...", widget.MediumImportance)
	up.showProgress(-1, -1)

	go func() {
		err := up.safeApply()

		fyne.Do(func() {
			if err != nil {
				logger.Rlogger.Error("Update failed", slog.Any("err", err))
				up.showFailure(err)
				return
			}

			up.setStatus("Update installed, restarting...", widget.SuccessImportance)
			up.showProgress(-1, -1)
			up.onApplied()
		})
	}()
}

func (up *UpdatePopup) safeApply() (err error) {
	logger.FuncDebug()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unexpected error: %v", r)
		}
	}()

	return up.apply(up.throttledProgress())
}

func (up *UpdatePopup) throttledProgress() func(downloaded int64, total int64) {
	logger.FuncDebug()

	var mu sync.Mutex
	var lastRefresh time.Time

	return func(downloaded int64, total int64) {
		mu.Lock()
		done := total > 0 && downloaded >= total
		if !done && time.Since(lastRefresh) < updateProgressRefreshInterval {
			mu.Unlock()
			return
		}
		lastRefresh = time.Now()
		mu.Unlock()

		fyne.Do(func() {
			if !up.busy {
				return
			}

			if done {
				up.setStatus("Installing update...", widget.MediumImportance)
				up.showProgress(-1, -1)
				return
			}

			if total > 0 {
				up.setStatus(fmt.Sprintf("Downloading update... %s / %s", formatMegabytes(downloaded), formatMegabytes(total)), widget.MediumImportance)
			} else {
				up.setStatus("Downloading update... "+formatMegabytes(downloaded), widget.MediumImportance)
			}

			up.showProgress(downloaded, total)
		})
	}
}

func (up *UpdatePopup) showFailure(err error) {
	logger.FuncDebug()

	up.busy = false
	up.hideProgress()
	up.setStatus("Update failed: "+err.Error(), widget.DangerImportance)

	up.btnUpdate.SetText("Retry")
	up.btnUpdate.SetIcon(theme.ViewRefreshIcon())
	up.buttons.Show()
}

func (up *UpdatePopup) setStatus(text string, importance widget.Importance) {
	logger.FuncDebug()

	logger.FuncDebug()

	up.status.Importance = importance
	up.status.SetText(text)
	up.status.Show()
}

func (up *UpdatePopup) showProgress(downloaded int64, total int64) {
	logger.FuncDebug()

	logger.FuncDebug()

	if total <= 0 {
		up.progress.Hide()
		if !up.progressInfinite.Running() {
			up.progressInfinite.Start()
		}
		up.progressInfinite.Show()
		return
	}

	up.progressInfinite.Stop()
	up.progressInfinite.Hide()
	up.progress.SetValue(float64(downloaded) / float64(total))
	up.progress.Show()
}

func (up *UpdatePopup) hideProgress() {
	logger.FuncDebug()

	logger.FuncDebug()

	up.progressInfinite.Stop()
	up.progressInfinite.Hide()
	up.progress.Hide()
}

func formatMegabytes(bytes int64) string {
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
