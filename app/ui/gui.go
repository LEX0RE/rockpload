package ui

import (
	"log/slog"
	"os"
	"slices"
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
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	EVENT_CLICK_UPLOAD       = "click_upload"
	WAIT_BEFORE_OPEN_BROWSER = 5 * time.Second
)

type GUI struct {
	window         fyne.Window
	appConfig      *config.AppConfig
	accountManager *manager.AccountManager

	LoginBox  *fyne.Container
	PlayerBox *fyne.Container

	TokenEntry       *widget.Entry
	ConnectedLabel   *widget.Label
	MatchHistoryData []string
	MatchHistoryList *widget.List
	UploadStatus     *widget.Label
	UploadProgress   *widget.ProgressBar

	EventManager *tools.EventManager

	Clipboard func() fyne.Clipboard
}

func NewGUI(window fyne.Window, version string, appConfig *config.AppConfig, accountManager *manager.AccountManager, clipboard func() fyne.Clipboard) (g *GUI, err error) {
	logger.FuncDebug()
	g = &GUI{window: window, appConfig: appConfig, accountManager: accountManager, EventManager: tools.NewEventManager(), Clipboard: clipboard}

	centeredLabel := container.NewCenter(widget.NewLabelWithStyle("Welcome to Rockpload! ("+version+")", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

	accountPopup := NewAccountSettingsPopup(NewPopup("Account Settings", g.window, appConfig, accountManager))
	accountBtn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() { accountPopup.Show() })
	accountBtn.Importance = widget.LowImportance

	StorageSettingsPopup := NewStorageSettingsPopup(NewPopup("Storage Settings", g.window, appConfig, accountManager))
	storageSettingsBtn := widget.NewButtonWithIcon("", theme.StorageIcon(), func() { StorageSettingsPopup.Show() })
	storageSettingsBtn.Importance = widget.LowImportance

	behaviorSettingsPopup := NewBehaviorSettingPopup(NewPopup("Behavior Settings", g.window, appConfig, accountManager))
	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() { behaviorSettingsPopup.Show() })
	settingsBtn.Importance = widget.LowImportance

	rightAlignedBtn := container.NewHBox(layout.NewSpacer(), accountBtn, storageSettingsBtn, settingsBtn)

	header := container.NewStack(centeredLabel, rightAlignedBtn)

	infoBox := container.NewVBox(
		header,
		widget.NewLabel("This program will fetch your Rocket League match history and upload replays to any website you want."),
		widget.NewSeparator(),
	)

	g.createLoginUI()
	g.createPlayerUI()

	contentBox := container.NewBorder(g.LoginBox, nil, nil, nil, g.PlayerBox)

	g.window.SetContent(container.NewBorder(infoBox, nil, nil, nil, contentBox))
	g.window.Resize(fyne.NewSize(450, 400))

	g.UpdateState()

	return g, nil
}

func (g *GUI) UpdateState() {
	logger.FuncDebug()

	selectedAccount := g.accountManager.GetSelected()

	g.RefreshConnectedAccount()

	if selectedAccount != nil && selectedAccount.IsConnected() {
		g.LoginBox.Hide()
		g.PlayerBox.Show()

		if selectedAccount.Player.MatchHistory != nil {
			g.MatchHistoryData = []string{}

			for _, match := range selectedAccount.Player.MatchHistory {
				matchUploaded := ""
				if slices.Contains(selectedAccount.HistorySended, match.Match.MatchGUID) {
					matchUploaded = "   (Uploaded)"
				}

				matchDate := time.Unix(match.Match.RecordStartTimestamp, 0).Format("2006-01-02 15:04:05")
				matchScore := strconv.Itoa(match.Match.Team0Score) + " - " + strconv.Itoa(match.Match.Team1Score)
				g.MatchHistoryData = append(g.MatchHistoryData, matchDate+" : "+matchScore+" ("+match.Match.MatchGUID+")"+matchUploaded)
			}
			g.MatchHistoryList.Refresh()
		} else {
			g.MatchHistoryData = []string{}
			g.MatchHistoryList.Refresh()
		}
	} else {
		g.LoginBox.Show()
		g.PlayerBox.Hide()

		g.MatchHistoryData = []string{}
		g.MatchHistoryList.Refresh()
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
			time.Sleep(WAIT_BEFORE_OPEN_BROWSER)
			selectedAccount := g.accountManager.GetSelected()
			selectedAccount.Player.Auth.OpenDeviceAuth()

			err = selectedAccount.Player.Auth.AuthenticateWithDeviceCode()
			if err != nil {
				logger.Rlogger.Error("Authentication failed:", slog.Any("err", err))
			}
		}()
	})
	deviceAuthBorder := container.NewBorder(nil, nil, AuthWithDeviceCodeBtn, copyBtn, deviceCodeEntry)

	// Auth with Auth Code
	ConnectBtn := widget.NewButton("Connect", func() {
		selectedAccount := g.accountManager.GetSelected()
		err := selectedAccount.Player.Auth.AuthenticateWithAuthCode(g.TokenEntry.Text)
		if err != nil {
			logger.Rlogger.Error("Authentication failed:", slog.Any("err", err))
		}
	})

	ConnectBtn.Disable()

	g.TokenEntry = widget.NewPasswordEntry()
	g.TokenEntry.SetPlaceHolder("Paste your OAuth token here if needed")

	g.TokenEntry.OnChanged = func(s string) {
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

	ResetBrowserBtn := widget.NewButton("Reset Browser", func() {
		tools.ClearBrowserSession()
	})

	actionButtonBorder := container.NewBorder(nil, nil, LoginBtn, nil, ResetBrowserBtn)

	g.LoginBox = container.NewVBox(
		deviceAuthBorder,
		container.NewBorder(nil, nil, actionButtonBorder, nil, g.TokenEntry),
		ConnectBtn,
	)
}

func (g *GUI) createPlayerUI() {
	logger.FuncDebug()
	g.ConnectedLabel = widget.NewLabel("")
	g.RefreshConnectedAccount()

	g.UploadStatus = widget.NewLabel(g.lastUploadStatusText())
	g.UploadProgress = widget.NewProgressBar()
	g.UploadProgress.Hide()

	g.MatchHistoryData = []string{}
	g.MatchHistoryList = widget.NewList(
		func() int {
			return len(g.MatchHistoryData)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(g.MatchHistoryData[i])
		},
	)

	uploadBtn := widget.NewButton("Upload Now", func() {
		g.EventManager.Notify(EVENT_CLICK_UPLOAD, nil)
	})

	disconnectBtn := widget.NewButton("Disconnect", func() {
		dialog.ShowConfirm("Disconnect", "Are you sure you want tto disconnect this account ?", func(confirmed bool) {
			if confirmed {
				selectedAccount := g.accountManager.GetSelected()
				selectedAccount.Player.Reset()
				g.UpdateState()
			}
		}, g.window)
	})

	clearCacheBtn := widget.NewButton("Clear Match History Cache", func() {
		dialog.ShowConfirm("Clear Match History Cache", "Are you sure you want to delete match history cache ?", func(confirmed bool) {
			if confirmed {
				storage := g.appConfig.StorageSettings.Get()

				for _, website := range storage {
					uploadCache := upload.LoadUploadedCache(website.Name, 0)
					uploadCache.Clear()
				}

				if g.UploadStatus != nil {
					g.UploadStatus.SetText(g.lastUploadStatusText())
				}
			}
		}, g.window)
	})

	matchHistoryAccordion := widget.NewAccordionItem("Match History", g.MatchHistoryList)
	matchHistoryAccordion.Open = false

	uploadProgressBox := container.NewBorder(nil, nil, g.UploadStatus, nil, g.UploadProgress)

	g.PlayerBox = container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			g.ConnectedLabel,
			container.NewBorder(nil, nil, disconnectBtn, uploadBtn, clearCacheBtn),
			nil,
		),
		nil,
		nil,
		nil,
		container.NewBorder(uploadProgressBox, nil, nil, nil, widget.NewAccordion(matchHistoryAccordion)),
	)
}

func (g *GUI) UpdateUploadProgress(progress float64) {
	logger.FuncDebug()

	if g.UploadStatus == nil || g.UploadProgress == nil {
		return
	}

	if progress == -1 {
		g.UploadProgress.Hide()
		g.UploadStatus.SetText("Last upload: " + time.Now().Format("2006-01-02 15:04:05"))
		return
	}

	value := float64(max(0, min(progress, 1)))
	g.UploadProgress.SetValue(value)
	g.UploadProgress.Show()
	g.UploadStatus.SetText("Uploading replays... " + strconv.Itoa(int(value*100)) + "%")
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
		info, err := os.Stat(constant.UploadedCache + "_" + storage.Name)
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

	selectedAccount := g.accountManager.GetSelected()
	allAccount := len(g.appConfig.AccountSettings.Get())

	authenticatedAccount := 0
	for _, ac := range g.appConfig.AccountSettings.Get() {
		if ac.Player.Auth != nil && ac.Player.Auth.IsAuthenticated() {
			authenticatedAccount++
		}
	}

	connectedText := "Connected (" + strconv.Itoa(authenticatedAccount) + "/" + strconv.Itoa(allAccount) + ")"

	if selectedAccount != nil && selectedAccount.IsConnected() {
		g.ConnectedLabel.SetText(connectedText + ": " + selectedAccount.AccountName())
	} else {
		g.ConnectedLabel.SetText(connectedText)
	}
}
