package ui

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"lexore/rockpload/app/config"
	"lexore/rockpload/app/tools/logger"
	"lexore/rockpload/app/upload"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type GUI struct {
	window 		fyne.Window
	uploader 	*upload.Uploader
	appConfig 	*config.AppConfig

	LoginBox 	*fyne.Container
	PlayerBox 	*fyne.Container
	OptionBox 	*fyne.Container

	UploadDialog  dialog.Dialog
	UploadBar     *widget.ProgressBar

	TokenEntry 	*widget.Entry
	ConnectBtn 	*widget.Button
	LoginBtn   	*widget.Button
	PlayerName 	*widget.Label
	MatchHistoryData []string
	MatchHistoryList *widget.List
	optionAccordion *widget.AccordionItem
}

func NewGUI(window fyne.Window, version string, appConfig *config.AppConfig, uploader *upload.Uploader) (g *GUI, err error) {
	logger.FuncDebug()
	g = &GUI{window: window, appConfig: appConfig,uploader: uploader}

	infoBox := container.NewVBox(
		container.NewCenter(widget.NewLabel("Welcome to Rockpload! (" + version + ")")),
		widget.NewLabel("This program will fetch your Rocket League match history and upload replays to the Rocky server."),
		widget.NewSeparator(),
	)

	g.createLoginUI()
	g.createPlayerUI()
	g.createOptionsUI()

	contentBox := container.NewBorder(g.LoginBox,nil,nil,nil,g.PlayerBox)

	g.window.SetContent(
		container.NewBorder(
			infoBox,
			nil,
			nil,
			nil,
			container.NewBorder(g.OptionBox,nil,nil,nil,contentBox),
		),
	)

	g.UpdateState()

	return g, nil
}

func (g *GUI) UpdateState() {
	logger.FuncDebug()

	if g.uploader.Player.Auth.Auth != nil {
		g.LoginBox.Hide()
		g.PlayerBox.Show()

		if (g.uploader.Player.Auth.Auth.DisplayName != "") {
			g.PlayerName.SetText("Connected Account: " + g.uploader.Player.Auth.Auth.DisplayName)
			g.PlayerName.Show()
		} else {
			g.PlayerName.Hide()
		}
	} else {
		g.LoginBox.Show()
		g.PlayerBox.Hide()
	}

	if (g.uploader.Player.MatchHistory != nil) {
		g.MatchHistoryData = []string{}
		for _, match := range g.uploader.Player.MatchHistory {
			matchDate := time.Unix(match.Match.RecordStartTimestamp, 0).Format("2006-01-02 15:04:05")
			matchScore := strconv.Itoa(match.Match.Team0Score) + " - " + strconv.Itoa(match.Match.Team1Score)
			g.MatchHistoryData = append(g.MatchHistoryData, matchDate + " : " +matchScore + " (" + match.Match.MatchGUID + ")")
		}
		g.MatchHistoryList.Refresh()
	} else {
		g.MatchHistoryData = []string{}
		g.MatchHistoryList.Refresh()
	}
}

func (g *GUI) createLoginUI() {
	logger.FuncDebug()
	g.ConnectBtn = widget.NewButton("Connect", func() {
		err := g.uploader.Player.Auth.AuthenticateWithCode(g.TokenEntry.Text)
		if err != nil {
			logger.Rlogger.Error("Authentication failed:", slog.Any("err", err))
		}
	})

	g.ConnectBtn.Disable()

	g.TokenEntry = widget.NewPasswordEntry()
	g.TokenEntry.SetPlaceHolder("Paste your OAuth token here")

	g.TokenEntry.OnChanged = func(s string) {
		if strings.TrimSpace(s) != "" {
			g.ConnectBtn.Enable()
		} else {
			g.ConnectBtn.Disable()
		}
	}

	g.LoginBtn = widget.NewButton("Retrieve OAuth Token", func() {
		g.uploader.Player.Auth.OpenAuthURL()
	})

	g.LoginBox = container.NewVBox(
		container.NewBorder(nil,nil,g.LoginBtn,nil,g.TokenEntry),
		g.ConnectBtn,
	)
}

func (g *GUI) createPlayerUI() {
	logger.FuncDebug()
	g.PlayerName = widget.NewLabel("")
	g.PlayerName.Hide()

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
		g.uploader.Player.Auth.ClearToken()
		g.UpdateState()
	})

	matchHistoryAccordion := widget.NewAccordionItem("Match History", g.MatchHistoryList)
	matchHistoryAccordion.Open = false

	g.PlayerBox = container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			g.PlayerName,
			container.NewBorder(nil,nil,disconnectBtn,nil,uploadBtn),
			nil,
		),
		nil,
		nil,
		nil,
		widget.NewAccordion(matchHistoryAccordion),
	)
}

func (g *GUI) createOptionsUI() {
	logger.FuncDebug()
	autoStartCheck := widget.NewCheck("Start with system", func(value bool) {
		g.appConfig.SetAppConfig(config.AutoStart, value)
	})
	autoStartCheck.SetChecked(g.appConfig.GetAppConfig(config.AutoStart))

	autoUploadCheck := widget.NewCheck("Auto Upload Replays", func(value bool) {
		g.appConfig.SetAppConfig(config.AutoUpload, value)
	})
	autoUploadCheck.SetChecked(g.appConfig.GetAppConfig(config.AutoUpload))

	startInTrayCheck := widget.NewCheck("Start in Tray", func(value bool) {
		g.appConfig.SetAppConfig(config.StartInTray, value)
	})
	startInTrayCheck.SetChecked(g.appConfig.GetAppConfig(config.StartInTray))
	if (!g.appConfig.GetAppConfig(config.ExitInTray)) {
		startInTrayCheck.Hide()
	}

	exitInTrayCheck := widget.NewCheck("Exit in System Tray", func(value bool) {
		g.appConfig.SetAppConfig(config.ExitInTray, value)
		if (value) {
			startInTrayCheck.Show()
			if (g.optionAccordion != nil) {
				g.optionAccordion.Detail.Refresh()
				g.OptionBox.Refresh()
			}
		} else {
			startInTrayCheck.Hide()
			if (g.optionAccordion != nil) {
				g.optionAccordion.Detail.Refresh()
				g.OptionBox.Refresh()
			}
		}
	})
	exitInTrayCheck.SetChecked(g.appConfig.GetAppConfig(config.ExitInTray))

	g.optionAccordion = widget.NewAccordionItem("Option", container.NewVBox(autoStartCheck, autoUploadCheck, exitInTrayCheck, startInTrayCheck))
	g.optionAccordion.Open = false

	g.OptionBox = container.NewBorder(nil,nil,nil,nil,widget.NewAccordion(g.optionAccordion))
}