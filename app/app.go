package app

import (
	_ "embed"
	"log/slog"
	"os"
	"os/exec"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/LEX0RE/rockpload/app/ui"
	"github.com/LEX0RE/rockpload/app/upload"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/emersion/go-autostart"
	"github.com/gofrs/flock"
)

//go:embed assets/logo.png
var logoBytes []byte

type App struct {
	app    fyne.App
	window fyne.Window

	appConfig *config.AppConfig
	gui       *ui.GUI

	duplicateLock *flock.Flock
	version       string
	updateInfo    *UpdateInfo

	player   *rocket_network.Player
	uploader *upload.Uploader
}

func NewApp(version string) *App {
	logger.FuncDebug()

	a := &App{version: version}

	a.app = app.NewWithID("com.lexore.rockpload")

	a.appConfig = config.NewAppConfig()
	a.appConfig.AutoStart.Bind(a.SetAutoStart)
	a.appConfig.AutoUpload.Bind(a.SetAutoUpload)
	a.appConfig.WebsiteSettings.Bind(a.OnWebsiteConfigChange)

	// TODO Deprecated, Fyne Pref will be removed in future version
	a.appConfig.ImportFynePreferences(a.app.Preferences())

	icon := fyne.NewStaticResource("logo.png", logoBytes)
	a.app.SetIcon(icon)

	a.window = a.app.NewWindow("Rockpload")

	if desk, ok := a.app.(desktop.App); ok {
		menu := fyne.NewMenu("Rockpload",
			fyne.NewMenuItem("Open", func() {
				a.window.Show()
			}),
		)

		desk.SetSystemTrayMenu(menu)
	}

	a.window.SetCloseIntercept(func() {
		if a.appConfig.ExitInTray.Get() {
			a.window.Hide()
		} else {
			a.app.Quit()
		}
	})

	return a
}

func (a *App) Close() {
	logger.FuncDebug()
	a.duplicateLock.Unlock()
	a.app.Quit()
}

func (a *App) Run() {
	logger.FuncDebug()

	gotLock := a.createDuplicateLock()
	if !gotLock {
		return
	}
	defer a.duplicateLock.Unlock()

	a.setupAppUpdate()
	a.initPlayer()

	a.window.Resize(fyne.NewSize(400, 300))

	if a.appConfig.ExitInTray.Get() && a.appConfig.StartInTray.Get() {
		a.app.Run()
	} else {
		a.window.ShowAndRun()
	}
}

func (a *App) initPlayer() {
	logger.FuncDebug()
	auth, err := rocket_network.NewAuth()
	if err != nil {
		logger.Rlogger.Error("Authentication error:", slog.Any("err", err))
	}

	a.player = rocket_network.NewPlayer(auth)

	onUpdateState := func() {
		fyne.Do(a.gui.UpdateState)
	}

	uploadPopup := ui.NewUploadingPopup(ui.NewPopup("Uploading Replays...", a.window, a.appConfig), 0)
	a.uploader = upload.NewUploader(a.player, a.appConfig, onUpdateState, uploadPopup.UpdateProgress)

	a.gui, err = ui.NewGUI(a.window, a.version, a.appConfig, a.uploader)
	if err != nil {
		logger.Rlogger.Error("Failed to initialize GUI:", slog.Any("err", err))
	}

	a.player.Auth.Sub.Subscribe(func(event string) {
		if event == rocket_network.EventUserAuthenticated {
			if a.appConfig.AutoUpload.Get() {
				a.uploader.Start()
			}

			a.gui.UpdateState()
		}
	})

	if a.player.Auth != nil {
		a.player.GetInfo()
		a.gui.UpdateState()
	}

	if a.appConfig.AutoUpload.Get() {
		a.uploader.Start()
	}
}

func (a *App) setupAppUpdate() {
	logger.FuncDebug()
	updater := NewUpdater()
	skipUpdate := os.Getenv("SKIP_UPDATE")

	needUpdate, err := updater.CheckForUpdate(a.version)
	if err != nil {
		logger.Rlogger.Error("Failed to check for update:", slog.Any("err", err))
		a.updateInfo = nil
		return
	}

	a.updateInfo = updater.UpdateInfo

	if needUpdate && skipUpdate != "true" {
		// TODO Make auto update without ask setting to not annoy people
		updatePopup := ui.NewUpdatePopup(ui.NewPopup("New Update!", a.window, a.appConfig), a.updateInfo.Version, func() {
			err := updater.ApplyUpdate()
			if err != nil {
				logger.Rlogger.Error("Update failed", slog.Any("err", err))
				return
			}

			a.restart()
		})

		updatePopup.Show()
	}
}

func (a *App) createDuplicateLock() bool {
	logger.FuncDebug()
	a.duplicateLock = flock.New(config.AppLock)
	gotLocked, err := a.duplicateLock.TryLock()
	if err != nil {
		panic(err)
	}

	return gotLocked
}

func (a *App) SetAutoStart(value bool) {
	logger.FuncDebug()
	execPath, err := os.Executable()
	if err != nil {
		logger.Rlogger.Error("Failed to get executable path for autostart:", slog.Any("err", err))
		return
	}

	asapp := &autostart.App{
		Name:        "lexore-rockpload",
		DisplayName: "Rockpload",
		Exec:        []string{execPath},
	}

	if value {
		err = asapp.Enable()
	} else {
		err = asapp.Disable()
	}

	if err != nil && !os.IsNotExist(err) {
		logger.Rlogger.Error("Failed to enable autostart:", slog.Any("err", err))
	}
}

func (a *App) SetAutoUpload(value bool) {
	logger.FuncDebug()

	if a.uploader != nil {
		a.uploader.Toggle(value)
	}
}

func (a *App) restart() {
	logger.FuncDebug()

	a.Close()

	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Start()
	if err != nil {
		logger.Rlogger.Error("restart failed", slog.Any("err", err))
	}

	os.Exit(0)
}

func (a *App) OnWebsiteConfigChange(value []*config.WebsiteConfig) {
	logger.FuncDebug()

	a.uploader.UpdateWebsite()
}
