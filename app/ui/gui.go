package ui

import (
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/LEX0RE/rockpload/app/upload"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type GUI struct {
	window    fyne.Window
	uploader  *upload.Uploader
	appConfig *config.AppConfig

	LoginBox  *fyne.Container
	PlayerBox *fyne.Container

	TokenEntry       *widget.Entry
	ConnectedLabel   *widget.Label
	MatchHistoryData []string
	MatchHistoryList *widget.List
}

func NewGUI(window fyne.Window, version string, appConfig *config.AppConfig, uploader *upload.Uploader) (g *GUI, err error) {
	logger.FuncDebug()
	g = &GUI{window: window, appConfig: appConfig, uploader: uploader}

	centeredLabel := container.NewCenter(widget.NewLabelWithStyle("Welcome to Rockpload! ("+version+")", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

	accountPopup := NewAccountSettingsPopup(NewPopup("Account Settings", g.window, appConfig))
	accountBtn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() { accountPopup.Show() })
	accountBtn.Importance = widget.LowImportance

	StorageSettingsPopup := NewStorageSettingsPopup(NewPopup("Storage Settings", g.window, appConfig))
	storageSettingsBtn := widget.NewButtonWithIcon("", theme.StorageIcon(), func() { StorageSettingsPopup.Show() })
	storageSettingsBtn.Importance = widget.LowImportance

	behaviorSettingsPopup := NewBehaviorSettingPopup(NewPopup("Behavior Settings", g.window, appConfig))
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
	g.window.Resize(fyne.NewSize(450, 350))

	g.UpdateState()

	return g, nil
}

func (g *GUI) UpdateState() {
	logger.FuncDebug()

	selectedAccount := g.appConfig.SelectedAccount()

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
	ConnectBtn := widget.NewButton("Connect", func() {
		selectedAccount := g.appConfig.SelectedAccount()
		err := selectedAccount.Player.Auth.AuthenticateWithCode(g.TokenEntry.Text)
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
		selectedAccount := g.appConfig.SelectedAccount()
		selectedAccount.Player.Auth.OpenAuth()
	})

	ResetBrowserBtn := widget.NewButton("Reset Browser", func() {
		tools.ClearBrowserSession()
	})

	actionButtonBorder := container.NewBorder(nil, nil, LoginBtn, nil, ResetBrowserBtn)

	g.LoginBox = container.NewVBox(
		container.NewBorder(nil, nil, actionButtonBorder, nil, g.TokenEntry),
		ConnectBtn,
	)
}

func (g *GUI) createPlayerUI() {
	logger.FuncDebug()
	g.ConnectedLabel = widget.NewLabel("")
	g.RefreshConnectedAccount()

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
		g.uploader.Run()
	})

	disconnectBtn := widget.NewButton("Disconnect", func() {
		selectedAccount := g.appConfig.SelectedAccount()
		selectedAccount.Player.Reset()
		g.UpdateState()
	})

	matchHistoryAccordion := widget.NewAccordionItem("Match History", g.MatchHistoryList)
	matchHistoryAccordion.Open = false

	g.PlayerBox = container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			g.ConnectedLabel,
			container.NewBorder(nil, nil, disconnectBtn, nil, uploadBtn),
			nil,
		),
		nil,
		nil,
		nil,
		widget.NewAccordion(matchHistoryAccordion),
	)
}

func (g *GUI) RefreshConnectedAccount() {
	logger.FuncDebug()

	selectedAccount := g.appConfig.SelectedAccount()
	allAccount := len(g.appConfig.AccountSettings.Get())

	authenticatedAccount := 0
	for _, ac := range g.appConfig.AccountSettings.Get() {
		if ac.Player.Auth != nil && ac.Player.Auth.IsAuthenticated() {
			authenticatedAccount++
		}
	}

	connectedText := "Connected (" + strconv.Itoa(authenticatedAccount) + "/" + strconv.Itoa(allAccount) + ")"

	if selectedAccount != nil && selectedAccount.IsConnected() {
		g.ConnectedLabel.SetText(connectedText + ": " + selectedAccount.Player.PlayerName)
	} else {
		g.ConnectedLabel.SetText(connectedText)
	}
}
