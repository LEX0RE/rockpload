package ui

import (
	"image/color"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/manager"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/LEX0RE/rockpload/app/upload"
	"github.com/dank/rlapi"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	EVENT_CLICK_UPLOAD        tools.EventType = "click_upload"
	EVENT_CLICK_UPLOAD_FILE   tools.EventType = "click_upload_file"
	EVENT_CLICK_FETCH_HISTORY tools.EventType = "click_fetch_history"
	EVENT_CLICK_UPLOAD_MATCH  tools.EventType = "click_upload_match"

	waitBeforeOpenBrowser = 5 * time.Second
)

type matchHistoryRow struct {
	label     string
	matchGUID string
	uploaded  bool
}

type GUI struct {
	window         fyne.Window
	appConfig      *config.AppConfig
	accountManager *manager.AccountManager
	rlSupervisor   *manager.RLSupervisor

	loginBox           *fyne.Container
	retryConnectionBtn *widget.Button
	playerBox          *fyne.Container

	tokenEntry         *widget.Entry
	connectedLabel     *widget.Label
	matchHistoryData   []matchHistoryRow
	matchHistoryList   *widget.List
	uploadStatus       *widget.Label
	uploadProgress     *widget.ProgressBar
	rlDetectedLabel    *fyne.Container
	rlConnectedWarning *fyne.Container

	EventManager *tools.EventManager

	Clipboard func() fyne.Clipboard
}

func NewGUI(window fyne.Window, version string, appConfig *config.AppConfig, accountManager *manager.AccountManager, rlSupervisor *manager.RLSupervisor, clipboard func() fyne.Clipboard) (g *GUI, err error) {
	logger.FuncDebug()
	g = &GUI{
		window:         window,
		appConfig:      appConfig,
		accountManager: accountManager,
		rlSupervisor:   rlSupervisor,
		EventManager:   tools.NewEventManager(),
		Clipboard:      clipboard,
	}

	centeredLabel := container.NewCenter(widget.NewLabelWithStyle("Rockpload ("+version+")", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

	accountPopup := NewAccountSettingsPopup(NewPopup("Account Settings", g.window, appConfig, accountManager))
	accountBtn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() { accountPopup.Show() })
	accountBtn.Importance = widget.LowImportance

	StorageSettingsPopup := NewStorageSettingsPopup(NewPopup("Storage Settings", g.window, appConfig, accountManager), version)
	storageSettingsBtn := widget.NewButtonWithIcon("", theme.StorageIcon(), func() { StorageSettingsPopup.Show() })
	storageSettingsBtn.Importance = widget.LowImportance

	behaviorSettingsPopup := NewBehaviorSettingPopup(NewPopup("Behavior Settings", g.window, appConfig, accountManager))
	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() { behaviorSettingsPopup.Show() })
	settingsBtn.Importance = widget.LowImportance

	warningRLDetectedLabel := container.NewCenter(widget.NewLabel("⚠️ Rocket League instance detected"))
	warningBackground := canvas.NewRectangle(color.RGBA{R: 200, G: 100, B: 0, A: 255})
	g.rlDetectedLabel = container.NewStack(warningBackground, warningRLDetectedLabel)
	g.rlDetectedLabel.Hide()

	rightAlignedBtn := container.NewHBox(layout.NewSpacer(), accountBtn, storageSettingsBtn, settingsBtn)

	header := container.NewStack(centeredLabel, rightAlignedBtn)

	descriptionLabel := widget.NewLabel("This program will fetch your Rocket League match history and upload replays to any website you want.")
	descriptionLabel.Wrapping = fyne.TextWrapWord

	infoBox := container.NewVBox(
		header,
		descriptionLabel,
		widget.NewSeparator(),
		g.rlDetectedLabel,
	)

	g.createLoginUI()
	g.createPlayerUI()

	contentBox := container.NewBorder(g.loginBox, nil, nil, nil, g.playerBox)

	g.window.SetContent(container.NewBorder(infoBox, nil, nil, nil, contentBox))
	g.UpdateState()

	return g, nil
}

func (g *GUI) UpdateState() {
	logger.FuncDebug()

	selectedAccount := g.accountManager.GetSelected()

	g.RefreshConnectedAccount()

	if g.rlDetectedLabel != nil && g.rlSupervisor.LastRLRunningState {
		g.rlDetectedLabel.Show()
	} else {
		g.rlDetectedLabel.Hide()
	}

	if selectedAccount != nil && selectedAccount.IsConnected() {
		g.loginBox.Hide()
		g.playerBox.Show()

		if selectedAccount.Player.LastCheckOnline && g.appConfig.BehaviorConfig.NoUploadOnline.Get() {
			g.rlConnectedWarning.Show()
		} else {
			g.rlConnectedWarning.Hide()
		}

		if len(selectedAccount.Player.CachedMatchHistory) > 0 {
			g.matchHistoryData = []matchHistoryRow{}

			cacheSet := upload.LoadActiveUploadedCaches(g.appConfig.StorageSettings.Get(), len(g.appConfig.AccountSettings.Get()))

			for _, match := range selectedAccount.Player.CachedMatchHistory {
				matchLabel := "[" + match.MatchGUID[:2] + "..." + match.MatchGUID[len(match.MatchGUID)-2:] + "] "
				matchLabel += upload.PlaylistName(match.Playlist) + " • "

				matchDate := time.Unix(match.RecordStartTimestamp, 0).Format("2006-01-02 15:04")
				matchScore := strconv.Itoa(match.Team0Score) + " - " + strconv.Itoa(match.Team1Score)
				matchLabel += matchDate + " • " + matchScore

				g.matchHistoryData = append(g.matchHistoryData, matchHistoryRow{
					label:     matchLabel,
					matchGUID: match.MatchGUID,
					uploaded:  cacheSet.IsUploadedEverywhere(match.MatchGUID),
				})
			}
			g.matchHistoryList.Refresh()
		} else {
			g.matchHistoryData = []matchHistoryRow{}
			g.matchHistoryList.Refresh()
		}
	} else {
		g.loginBox.Show()
		g.playerBox.Hide()

		if g.rlConnectedWarning != nil {
			g.rlConnectedWarning.Hide()
		}

		if selectedAccount.Player.Auth.HasTokens() {
			g.retryConnectionBtn.Enable()
			g.retryConnectionBtn.Show()
		} else {
			g.retryConnectionBtn.Disable()
			g.retryConnectionBtn.Hide()
		}

		g.matchHistoryData = []matchHistoryRow{}
		g.matchHistoryList.Refresh()
	}
}

func (g *GUI) createLoginUI() {
	logger.FuncDebug()

	// Auth with device code
	deviceCodeEntry := widget.NewEntry()
	deviceCodeEntry.Disable()
	deviceCodeEntry.Hide()

	var deviceCodeText *rlapi.DeviceAuthResponse
	var err error

	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if deviceCodeText != nil && deviceCodeText.UserCode != "" {
			g.Clipboard().SetContent(deviceCodeText.UserCode)
		}
	})
	copyBtn.Hide()

	AuthWithDeviceCodeBtn := widget.NewButton("Temporary Authenticate", func() {
		selectedAccount := g.accountManager.GetSelected()
		deviceCodeText, err = selectedAccount.Player.Auth.GetDeviceCode()
		if err != nil {
			logger.Rlogger.Error("Failed to get device code:", slog.Any("err", err))
			return
		}

		deviceCodeEntry.SetText("Copy this code to the browser that will open: " + deviceCodeText.UserCode)
		deviceCodeEntry.Show()
		copyBtn.Show()

		go func() {
			time.Sleep(waitBeforeOpenBrowser)
			selectedAccount := g.accountManager.GetSelected()
			selectedAccount.Player.Auth.OpenDeviceAuth()

			if err = selectedAccount.Player.Auth.AuthenticateWithDeviceCode(); err != nil {
				logger.Rlogger.Error("Authentication failed:", slog.Any("err", err))
			}
		}()
	})
	deviceAuthBorder := container.NewBorder(nil, nil, AuthWithDeviceCodeBtn, copyBtn, deviceCodeEntry)

	// Auth with Auth Code
	ConnectBtn := widget.NewButton("Connect", func() {
		selectedAccount := g.accountManager.GetSelected()
		if err := selectedAccount.Player.Auth.AuthenticateWithAuthCode(g.tokenEntry.Text); err != nil {
			logger.Rlogger.Error("Authentication failed:", slog.Any("err", err))
		}
	})

	ConnectBtn.Disable()

	g.tokenEntry = widget.NewPasswordEntry()
	g.tokenEntry.SetPlaceHolder("Paste your OAuth token here if needed")

	g.tokenEntry.OnChanged = func(s string) {
		if strings.TrimSpace(s) != "" {
			ConnectBtn.Enable()
		} else {
			ConnectBtn.Disable()
		}
	}

	LoginBtn := widget.NewButton("Authenticate", func() {
		selectedAccount := g.accountManager.GetSelected()
		selectedAccount.Player.Auth.OpenAuth()
	})

	var actionButtonBorder *fyne.Container
	if runtime.GOOS != "android" && runtime.GOOS != "ios" {
		resetBrowserBtn := widget.NewButton("Reset Browser", func() {
			tools.ClearBrowserSession()
		})
		actionButtonBorder = container.NewBorder(nil, nil, LoginBtn, nil, resetBrowserBtn)
	} else {
		actionButtonBorder = container.NewBorder(nil, nil, LoginBtn, nil)
	}

	authContainer := container.NewBorder(nil, nil, actionButtonBorder, nil, g.tokenEntry)

	g.retryConnectionBtn = widget.NewButton("Retry Last Connection", func() {
		if g.accountManager.GetSelected().IsConnected() {
			logger.Rlogger.Error("Retry last connection failed:")
		}
	})

	g.loginBox = container.NewVBox(deviceAuthBorder, authContainer, ConnectBtn, g.retryConnectionBtn)
}

func (g *GUI) createPlayerUI() {
	logger.FuncDebug()
	g.connectedLabel = widget.NewLabel("")
	g.RefreshConnectedAccount()

	g.uploadStatus = widget.NewLabel(g.lastUploadStatusText())
	g.uploadProgress = widget.NewProgressBar()
	g.uploadProgress.Hide()

	warningRLConnectedLabel := container.NewCenter(widget.NewLabel("⚠️ The player could be connected or unused during the last check.\nNo refresh will be done for this player while 'No Upload if Online' is checked."))
	warningBackground := canvas.NewRectangle(color.RGBA{R: 200, G: 100, B: 0, A: 255})
	g.rlConnectedWarning = container.NewStack(warningBackground, warningRLConnectedLabel)
	g.rlConnectedWarning.Hide()

	g.matchHistoryData = []matchHistoryRow{}
	g.matchHistoryList = widget.NewList(
		func() int {
			return len(g.matchHistoryData)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")

			uploadedIcon := widget.NewIcon(theme.ConfirmIcon())
			uploadedIcon.Hide()

			rowUploadBtn := widget.NewButtonWithIcon("", theme.UploadIcon(), nil)
			rowUploadBtn.Importance = widget.LowImportance

			rightBox := container.NewHBox(uploadedIcon, rowUploadBtn)
			return container.NewBorder(nil, nil, nil, rightBox, label)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			row := o.(*fyne.Container)
			label := row.Objects[0].(*widget.Label)
			rightBox := row.Objects[1].(*fyne.Container)
			uploadedIcon := rightBox.Objects[0].(*widget.Icon)
			rowUploadBtn := rightBox.Objects[1].(*widget.Button)

			data := g.matchHistoryData[i]
			label.SetText(data.label)

			if data.uploaded {
				uploadedIcon.Show()
			} else {
				uploadedIcon.Hide()
			}

			rowUploadBtn.OnTapped = func() {
				g.EventManager.Notify(EVENT_CLICK_UPLOAD_MATCH, data.matchGUID)
			}
		},
	)

	fetchHistoryBtn := widget.NewButton("Fetch Match History", func() {
		g.EventManager.Notify(EVENT_CLICK_FETCH_HISTORY, nil)
	})

	uploadBtn := widget.NewButton("Upload Now", func() {
		g.EventManager.Notify(EVENT_CLICK_UPLOAD, nil)
	})

	uploadFileBtn := widget.NewButton("Upload File", func() {
		g.showUploadReplayFileDialog()
	})

	disconnectBtn := widget.NewButton("Disconnect", func() {
		dialog.ShowConfirm("Disconnect", "Are you sure you want to disconnect this account ?", func(confirmed bool) {
			if confirmed {
				selectedAccount := g.accountManager.GetSelected()
				selectedAccount.Player.Reset()
				g.UpdateState()
			}
		}, g.window)
	})

	var connectionContainer *fyne.Container
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		buttonContainer := container.New(NewCenterWrapLayout(), disconnectBtn, uploadFileBtn, uploadBtn)
		connectionContainer = container.NewBorder(nil, buttonContainer, g.connectedLabel, nil, nil)
	} else {
		rightButtons := container.NewHBox(uploadFileBtn, uploadBtn)
		buttonContainer := container.NewBorder(nil, nil, disconnectBtn, rightButtons)
		connectionContainer = container.NewBorder(nil, nil, g.connectedLabel, buttonContainer, nil)
	}

	matchHistoryBox := container.NewBorder(fetchHistoryBtn, nil, nil, nil, g.matchHistoryList)
	matchHistoryAccordion := widget.NewAccordionItem("Match History", matchHistoryBox)
	matchHistoryAccordion.Open = false

	uploadProgressBox := container.NewBorder(nil, nil, g.uploadStatus, nil, g.uploadProgress)
	centerTopBox := container.NewVBox(g.rlConnectedWarning, uploadProgressBox)

	g.playerBox = container.NewBorder(
		connectionContainer,
		nil,
		nil,
		nil,
		container.NewBorder(centerTopBox, nil, nil, nil, widget.NewAccordion(matchHistoryAccordion)),
	)
}

func (g *GUI) showUploadReplayFileDialog() {
	logger.FuncDebug()

	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, g.window)
			return
		}

		if reader == nil {
			return
		}
		defer reader.Close()

		tmpFile, err := os.CreateTemp("", "*.replay")
		if err != nil {
			dialog.ShowError(err, g.window)
			return
		}
		tmpPath := tmpFile.Name()

		_, err = io.Copy(tmpFile, reader)
		tmpFile.Close()
		if err != nil {
			os.Remove(tmpPath)
			dialog.ShowError(err, g.window)
			return
		}

		g.EventManager.Notify(EVENT_CLICK_UPLOAD_FILE, tmpPath)
	}, g.window)

	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".replay"}))
	fileDialog.Show()
}

func (g *GUI) UpdateUploadProgress(progress float64) {
	logger.FuncDebug()

	if g.uploadStatus == nil || g.uploadProgress == nil {
		return
	}

	if progress == -1 {
		g.uploadProgress.Hide()
		g.uploadStatus.SetText("Last upload: " + time.Now().Format("2006-01-02 15:04:05"))
		return
	}

	value := float64(max(0, min(progress, 1)))
	g.uploadProgress.SetValue(value)
	g.uploadProgress.Show()
	g.uploadStatus.SetText("Uploading replays...")
}

func (g *GUI) lastUploadStatusText() string {
	lastUploadAt, ok := g.lastUploadAt()
	if !ok {
		return "Last upload: Never"
	}

	return "Last upload: " + lastUploadAt.Format("2006-01-02 15:04:05")
}

func (g *GUI) lastUploadAt() (time.Time, bool) {
	var lastUploadAt time.Time

	for _, storage := range g.appConfig.StorageSettings.Get() {
		info, err := os.Stat(constant.Paths.UploadedCache + "_" + storage.Name)
		if err != nil {
			continue
		}

		if info.ModTime().After(lastUploadAt) {
			lastUploadAt = info.ModTime()
		}
	}

	return lastUploadAt, !lastUploadAt.IsZero()
}

func (g *GUI) RefreshConnectedAccount() {
	logger.FuncDebug()

	allAccount := len(g.appConfig.AccountSettings.Get())
	authenticatedAccount := len(g.accountManager.GetConnectedAccount())
	connectedText := "Connected (" + strconv.Itoa(authenticatedAccount) + "/" + strconv.Itoa(allAccount) + ")"

	selectedAccount := g.accountManager.GetSelected()
	if selectedAccount != nil && selectedAccount.IsConnected() {
		g.connectedLabel.SetText(connectedText + ": " + selectedAccount.AccountName())
	} else {
		g.connectedLabel.SetText(connectedText)
	}
}
