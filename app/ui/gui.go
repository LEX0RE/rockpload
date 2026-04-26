package ui

import (
	"log/slog"
	"net/url"
	"regexp"
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

var authCodePattern = regexp.MustCompile(`(?i)\b[a-z0-9]{32}\b`)

type GUI struct {
	window    fyne.Window
	uploader  *upload.Uploader
	appConfig *config.AppConfig

	LoginBox  *fyne.Container
	PlayerBox *fyne.Container
	OptionBox *fyne.Container

	UploadDialog dialog.Dialog
	UploadBar    *widget.ProgressBar

	TokenEntry       *widget.Entry
	ConnectBtn       *widget.Button
	LoginBtn         *widget.Button
	LoginStatus      *widget.Label
	PlayerName       *widget.Label
	SwitchAccountBtn *widget.Button
	MatchHistoryData []string
	MatchHistoryList *widget.List
	optionAccordion  *widget.AccordionItem
}

func NewGUI(window fyne.Window, version string, appConfig *config.AppConfig, uploader *upload.Uploader) (g *GUI, err error) {
	logger.FuncDebug()
	g = &GUI{window: window, appConfig: appConfig, uploader: uploader}

	infoBox := container.NewVBox(
		container.NewCenter(widget.NewLabel("Welcome to Rockpload! ("+version+")")),
		widget.NewLabel("This program will fetch your Rocket League match history and upload replays to the Rocky server."),
		widget.NewSeparator(),
	)

	g.createLoginUI()
	g.createPlayerUI()
	g.createOptionsUI()

	contentBox := container.NewBorder(g.LoginBox, nil, nil, nil, g.PlayerBox)

	g.window.SetContent(
		container.NewBorder(
			infoBox,
			nil,
			nil,
			nil,
			container.NewBorder(g.OptionBox, nil, nil, nil, contentBox),
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
		g.SwitchAccountBtn.Show()

		if g.uploader.Player.Auth.Auth.DisplayName != "" {
			g.PlayerName.SetText("Connected Account: " + g.uploader.Player.Auth.Auth.DisplayName)
			g.PlayerName.Show()
		} else {
			g.PlayerName.Hide()
		}
	} else {
		g.LoginBox.Show()
		g.PlayerBox.Hide()
		g.SwitchAccountBtn.Hide()
	}

	if g.uploader.Player.MatchHistory != nil {
		g.MatchHistoryData = []string{}
		for _, match := range g.uploader.Player.MatchHistory {
			matchDate := time.Unix(match.Match.RecordStartTimestamp, 0).Format("2006-01-02 15:04:05")
			matchScore := strconv.Itoa(match.Match.Team0Score) + " - " + strconv.Itoa(match.Match.Team1Score)
			g.MatchHistoryData = append(g.MatchHistoryData, matchDate+" : "+matchScore+" ("+match.Match.MatchGUID+")")
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
			g.LoginStatus.SetText("Authentication failed. Paste a fresh code and try again.")
			return
		}

		g.LoginStatus.SetText("Authenticated.")
		g.UpdateState()
	})

	g.ConnectBtn.Disable()

	g.TokenEntry = widget.NewPasswordEntry()
	g.TokenEntry.SetPlaceHolder("Paste auth code if automatic sign-in fails")

	g.TokenEntry.OnChanged = func(s string) {
		if strings.TrimSpace(s) != "" {
			g.ConnectBtn.Enable()
		} else {
			g.ConnectBtn.Disable()
		}
	}

	g.LoginStatus = widget.NewLabel("")

	g.LoginBtn = widget.NewButton("Sign in with Epic", func() {
		g.LoginStatus.SetText("Opening Epic sign-in window...")
		go g.tryAutomaticBrowserSignIn(false)
	})

	g.LoginBox = container.NewVBox(
		container.NewBorder(nil, nil, g.LoginBtn, nil, g.TokenEntry),
		g.LoginStatus,
		g.ConnectBtn,
	)
}

func (g *GUI) tryAutomaticBrowserSignIn(forceAccountPicker bool) {
	logger.FuncDebug()

	fyne.Do(func() {
		if forceAccountPicker {
			g.LoginStatus.SetText("Opening Epic logout before account switch...")
		} else {
			g.LoginStatus.SetText("Opening Epic sign-in window...")
		}
	})

	var err error
	if config.EpicAuthMode() == "custom" {
		err = g.uploader.Player.Auth.AuthenticateWithConfiguredOAuth(3 * time.Minute)
	} else {
		err = g.uploader.Player.Auth.AuthenticateWithLauncherClient(3*time.Minute, forceAccountPicker)
	}
	if err != nil {
		logger.Rlogger.Error("Automatic browser authentication failed:", slog.Any("err", err))
		fyne.Do(func() {
			g.LoginStatus.SetText("Automatic browser sign-in failed: " + err.Error())
		})
		return
	}

	fyne.Do(func() {
		g.LoginStatus.SetText("Authenticated.")
		g.UpdateState()
	})
}

func (g *GUI) watchClipboardForAuthCode(timeout time.Duration) bool {
	logger.FuncDebug()

	baseline := strings.TrimSpace(g.window.Clipboard().Content())
	attempted := map[string]struct{}{}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		current := strings.TrimSpace(g.window.Clipboard().Content())
		if current == "" || current == baseline {
			time.Sleep(1 * time.Second)
			continue
		}

		authCode := extractAuthCode(current)
		if authCode == "" {
			time.Sleep(1 * time.Second)
			continue
		}

		if _, exists := attempted[authCode]; exists {
			time.Sleep(1 * time.Second)
			continue
		}
		attempted[authCode] = struct{}{}

		fyne.Do(func() {
			g.TokenEntry.SetText(authCode)
			g.LoginStatus.SetText("Auth code detected. Connecting...")
		})

		err := g.uploader.Player.Auth.AuthenticateWithCode(authCode)
		if err == nil {
			fyne.Do(func() {
				g.LoginStatus.SetText("Authenticated.")
			})
			return true
		}

		logger.Rlogger.Error("Automatic authentication failed:", slog.Any("err", err))
		fyne.Do(func() {
			g.LoginStatus.SetText("Auth code detected but rejected. Copy a fresh code and try again.")
		})

		time.Sleep(1 * time.Second)
	}

	return false
}

func extractAuthCode(raw string) string {
	trimmed := strings.Trim(strings.TrimSpace(raw), "\"")

	if parsedURL, err := url.Parse(trimmed); err == nil {
		if code := strings.TrimSpace(parsedURL.Query().Get("code")); code != "" {
			return code
		}
	}

	if match := authCodePattern.FindString(trimmed); match != "" {
		return strings.TrimSpace(match)
	}

	return ""
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

	g.SwitchAccountBtn = widget.NewButton("Switch Epic Account", func() {
		g.uploader.Player.Auth.ClearToken()
		g.UpdateState()
		g.LoginStatus.SetText("Opening Epic account switch...")
		go g.tryAutomaticBrowserSignIn(true)
	})
	g.SwitchAccountBtn.Hide()

	matchHistoryAccordion := widget.NewAccordionItem("Match History", g.MatchHistoryList)
	matchHistoryAccordion.Open = false
	actionBar := container.NewHBox(disconnectBtn, g.SwitchAccountBtn, uploadBtn)

	g.PlayerBox = container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			g.PlayerName,
			actionBar,
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
	if !g.appConfig.GetAppConfig(config.ExitInTray) {
		startInTrayCheck.Hide()
	}

	exitInTrayCheck := widget.NewCheck("Exit in System Tray", func(value bool) {
		g.appConfig.SetAppConfig(config.ExitInTray, value)
		if value {
			startInTrayCheck.Show()
			if g.optionAccordion != nil {
				g.optionAccordion.Detail.Refresh()
				g.OptionBox.Refresh()
			}
		} else {
			startInTrayCheck.Hide()
			if g.optionAccordion != nil {
				g.optionAccordion.Detail.Refresh()
				g.OptionBox.Refresh()
			}
		}
	})
	exitInTrayCheck.SetChecked(g.appConfig.GetAppConfig(config.ExitInTray))

	g.optionAccordion = widget.NewAccordionItem("Option", container.NewVBox(autoStartCheck, autoUploadCheck, exitInTrayCheck, startInTrayCheck))
	g.optionAccordion.Open = false

	g.OptionBox = container.NewBorder(nil, nil, nil, nil, widget.NewAccordion(g.optionAccordion))
}
