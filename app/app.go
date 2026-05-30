package app

import (
	_ "embed"
	"log/slog"
	"os"
	"os/exec"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/manager"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools"
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

	uploader       *upload.Uploader
	accountManager *manager.AccountManager
	rlSupervisor   *manager.RLSupervisor
}

func NewApp(version string) *App {
	logger.FuncDebug()

	a := &App{version: version}

	a.app = app.NewWithID("com.lexore.rockpload")

	// TODO Check for update before load config to be sure that if the config crash, we can still update it
	a.appConfig = config.NewAppConfig()
	err := a.appConfig.Load(a.app.Preferences())
	if err != nil {
		logger.Rlogger.Error("Failed to load settings:", slog.Any("err", err))
	}

	a.appConfig.BehaviorConfig.AutoStart.Bind(a.SetAutoStart)
	a.appConfig.BehaviorConfig.AutoUpload.Bind(a.SetAutoUpload)
	a.appConfig.BehaviorConfig.UploadOnRLClose.Bind(a.SetUploadOnRLClose)

	// Needed in Popup (so update popup will take it), so we need to init before all
	a.accountManager = manager.NewAccountManager(a.appConfig)

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
		if a.appConfig.BehaviorConfig.ExitInTray.Get() {
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

	a.initManager()
	a.initEvents()
	a.startManager()

	a.window.Resize(fyne.NewSize(450, 400))

	if a.appConfig.BehaviorConfig.ExitInTray.Get() && a.appConfig.BehaviorConfig.StartInTray.Get() {
		a.app.Run()
	} else {
		a.window.ShowAndRun()
	}
}

func (a *App) initManager() {
	logger.FuncDebug()

	a.uploader = upload.NewUploader(a.appConfig, a.accountManager)
	a.rlSupervisor = manager.NewRLSupervisor(a.appConfig, a.accountManager)

	var err error
	a.gui, err = ui.NewGUI(a.window, a.version, a.appConfig, a.accountManager, a.rlSupervisor, a.app.Clipboard)
	if err != nil {
		logger.Rlogger.Error("Failed to initialize GUI:", slog.Any("err", err))
	}
}

func (a *App) initEvents() {
	logger.FuncDebug()

	onUpdateState := func() {
		fyne.Do(a.gui.UpdateState)
	}

	a.uploader.EventManager.Subscribe(upload.EventUploadProgress, tools.Listener{IsSync: false, Callback: func(data any) {
		if progress, ok := data.(float64); ok && a.gui != nil {
			a.gui.UpdateUploadProgress(progress)
		}
	}})

	guiUploaderEventList := []tools.EventType{upload.EventReplayUploaded, upload.EventUploadCompleted, upload.EventReplayUploaded}
	a.uploader.EventManager.MultiSubscribe(guiUploaderEventList, tools.Listener{IsSync: false, Callback: func(data any) {
		onUpdateState()
	}})

	a.uploader.EventManager.Subscribe(upload.EventUploadStarted, tools.Listener{IsSync: true, Callback: func(data any) {
		a.accountManager.RefreshInfo()
		a.appConfig.Save()
	}})

	a.gui.EventManager.Subscribe(ui.EVENT_CLICK_UPLOAD, tools.Listener{IsSync: false, Callback: func(data any) {
		a.uploader.Run()
	}})

	guiAccountManagerEventList := []tools.EventType{manager.EVENT_SELECT_ACCOUNT, manager.EVENT_ADD_ACCOUNT, manager.EVENT_DELETE_ACCOUNT}
	a.accountManager.EventManager.MultiSubscribe(guiAccountManagerEventList, tools.Listener{IsSync: false, Callback: func(data any) {
		a.gui.UpdateState()
	}})

	refreshSubscription := func(ac *rocket_network.Account) {
		if ac.Player.Auth != nil {
			ac.Player.Auth.EventManager.UnsubscribeAll(rocket_network.EventUserAuthenticated)
			ac.Player.Auth.EventManager.Subscribe(rocket_network.EventUserAuthenticated, tools.Listener{IsSync: false, Callback: func(data any) {
				a.accountManager.RefreshProfile()

				if a.appConfig.BehaviorConfig.AutoUpload.Get() {
					a.uploader.Stop()
					a.uploader.Start()
				}

				onUpdateState()
				a.appConfig.Save()
			}})
		}
	}

	a.appConfig.AccountSettings.Bind(func(config.AccountMapConfig) {
		for _, ac := range a.appConfig.AccountSettings.Get() {
			refreshSubscription(ac)
		}
	})

	for _, ac := range a.appConfig.AccountSettings.Get() {
		refreshSubscription(ac)
	}

	updateGUIFromSupervisorEvent := []tools.EventType{manager.EVENT_ON_RL_DETECTED, manager.EVENT_ON_RL_PLAYER_DETECTED, manager.EVENT_ON_RL_CLOSED}
	a.rlSupervisor.EventManager.MultiSubscribe(updateGUIFromSupervisorEvent, tools.Listener{IsSync: false, Callback: func(data any) {
		onUpdateState()
	}})

	a.rlSupervisor.EventManager.Subscribe(manager.EVENT_ON_RL_CLOSED, tools.Listener{IsSync: false, Callback: func(data any) {
		if a.appConfig.BehaviorConfig.UploadOnRLClose.Get() {
			a.uploader.Run()
		}
	}})
}

func (a *App) startManager() {
	logger.FuncDebug()

	if a.appConfig.BehaviorConfig.AutoUpload.Get() {
		a.uploader.Start()
	}

	a.rlSupervisor.Start()
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
		updatePopup := ui.NewUpdatePopup(ui.NewPopup("New Update!", a.window, a.appConfig, a.accountManager), a.updateInfo.Version, func() {
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

	a.duplicateLock = flock.New(constant.AppLock)
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

func (a *App) SetUploadOnRLClose(value bool) {
	logger.FuncDebug()

	if a.rlSupervisor != nil {
		a.rlSupervisor.Toggle(value)
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
