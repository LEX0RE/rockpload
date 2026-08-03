package app

import (
	_ "embed"
	"log/slog"
	"os"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/manager"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/LEX0RE/rockpload/app/ui"
	"github.com/LEX0RE/rockpload/app/upload"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

//go:embed assets/logo.png
var logoBytes []byte

type duplicateLock interface {
	Unlock() error
}

type App struct {
	app    fyne.App
	window fyne.Window

	appConfig *config.AppConfig
	gui       *ui.GUI

	duplicateLock duplicateLock
	version       string
	updateInfo    *UpdateInfo

	uploader       *upload.Uploader
	statsApi       *rocket_network.StatsAPI
	accountManager *manager.AccountManager
	rlSupervisor   *manager.RLSupervisor
}

func NewApp(version string, app fyne.App) *App {
	logger.FuncDebug()

	a := &App{version: version, app: app}

	if version == "dev" {
		metadata := a.app.Metadata()
		if rockploadVersion := metadata.Custom["rockploadVersion"]; rockploadVersion != "" {
			a.version = rockploadVersion
		} else if metadata.Version != "" && metadata.Version != "0.0.1" {
			a.version = metadata.Version
		}
	}

	a.configureApp()

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

	a.configureSystemTray()

	a.window.SetCloseIntercept(func() {
		if a.canRunInTray() && a.appConfig.BehaviorConfig.ExitInTray.Get() {
			a.window.Hide()
		} else {
			a.app.Quit()
		}
	})

	return a
}

func (a *App) Close() {
	logger.FuncDebug()
	if a.duplicateLock != nil {
		a.duplicateLock.Unlock()
	}
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

	a.window.Resize(fyne.NewSize(700, 450))

	if a.canRunInTray() && a.appConfig.BehaviorConfig.ExitInTray.Get() && a.appConfig.BehaviorConfig.StartInTray.Get() {
		a.app.Run()
	} else {
		a.window.ShowAndRun()
	}
}

func (a *App) initManager() {
	logger.FuncDebug()

	a.uploader = upload.NewUploader(a.appConfig, a.accountManager)
	a.appConfig.BehaviorConfig.AutoUploadTime.Bind(a.uploader.SetDuration)
	a.appConfig.BehaviorConfig.AutoUploadRandomTime.Bind(a.uploader.SetMaxJitter)

	a.rlSupervisor = manager.NewRLSupervisor(a.appConfig, a.accountManager)

	var err error
	a.gui, err = ui.NewGUI(a.window, a.version, a.appConfig, a.accountManager, a.rlSupervisor, a.app.Clipboard)
	if err != nil {
		logger.Rlogger.Error("Failed to initialize GUI:", slog.Any("err", err))
	}
}

func (a *App) initEvents() {
	logger.FuncDebug()

	a.statsApi = rocket_network.NewStatsAPI()
	uploadLiveStatsEvent := []tools.EventType{"CountdownBegin", "PodiumStart"}
	a.statsApi.EventManager.MultiSubscribe(uploadLiveStatsEvent, tools.Listener{IsSync: false, Callback: func(data any) {
		a.uploader.UploadLiveStats(a.statsApi.LastInfo)
	}})

	if a.supportsLocalStatsAPI() {
		a.statsApi.StartListener()
	}

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

	a.gui.EventManager.Subscribe(ui.EVENT_CLICK_UPLOAD_FILE, tools.Listener{IsSync: false, Callback: func(data any) {
		filePath, ok := data.(string)
		if !ok {
			return
		}

		a.uploader.UploadLocalFile(filePath)
	}})

	a.uploader.EventManager.Subscribe(upload.EventSingleUploadCompleted, tools.Listener{IsSync: false, Callback: func(data any) {
		if data == nil {
			dialog.ShowInformation("Upload Complete", "The replay was uploaded successfully.", a.window)
			return
		}

		if err, ok := data.(error); ok {
			dialog.ShowError(err, a.window)
		}
	}})

	a.uploader.EventManager.Subscribe(upload.EventMatchUploadCompleted, tools.Listener{IsSync: false, Callback: func(data any) {
		if err, ok := data.(error); ok && err != nil {
			dialog.ShowError(err, a.window)
			return
		}

		a.gui.UpdateState()
	}})

	a.gui.EventManager.Subscribe(ui.EVENT_CLICK_FETCH_HISTORY, tools.Listener{IsSync: false, Callback: func(data any) {
		go func() {
			selectedAccount := a.accountManager.GetSelected()
			err := a.accountManager.RefreshMatchHistory(selectedAccount)

			if err == nil {
				a.appConfig.Save()
			}

			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, a.window)
					return
				}

				a.gui.UpdateState()
			})
		}()
	}})

	a.gui.EventManager.Subscribe(ui.EVENT_CLICK_UPLOAD_MATCH, tools.Listener{IsSync: false, Callback: func(data any) {
		matchGUID, ok := data.(string)
		if !ok {
			return
		}

		selectedAccount := a.accountManager.GetSelected()
		if selectedAccount == nil {
			return
		}

		matchIndex := selectedAccount.Player.GetMatchHistoryIndex(matchGUID)
		if matchIndex == -1 {
			dialog.ShowInformation("Fetch needed", "Fetch match history first to upload this replay.", a.window)
			return
		}

		a.uploader.UploadMatchEntry(selectedAccount, selectedAccount.Player.MatchHistory[matchIndex])
	}})

	guiAccountManagerEventList := []tools.EventType{manager.EVENT_SELECT_ACCOUNT, manager.EVENT_ADD_ACCOUNT, manager.EVENT_DELETE_ACCOUNT}
	a.accountManager.EventManager.MultiSubscribe(guiAccountManagerEventList, tools.Listener{IsSync: false, Callback: func(data any) {
		a.gui.UpdateState()
	}})

	if rocket_network.AuthEventManager == nil {
		rocket_network.AuthEventManager = tools.NewEventManager()
	}

	rocket_network.AuthEventManager.Subscribe(rocket_network.EventUserAuthenticated, tools.Listener{IsSync: false, Callback: func(data any) {
		a.accountManager.RefreshProfile()

		if a.appConfig.BehaviorConfig.AutoUpload.Get() {
			a.uploader.Stop()
			a.uploader.Start()
		}

		onUpdateState()
		a.appConfig.Save()
	}})

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

	a.startPlatformManagers()
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
		updatePopup := ui.NewUpdatePopup(ui.NewPopup("New Update!", a.window, a.appConfig, a.accountManager), a.updateInfo.Version, string(a.updateInfo.Source), func() {
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
