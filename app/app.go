package app

import (
	_ "embed"
	"log/slog"
	"os"
	"sync"

	"github.com/LEX0RE/rockpload/app/bridge"
	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/manager"
	"github.com/LEX0RE/rockpload/app/rlgame"
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

	profileRefreshMu sync.Mutex
	startOnce        sync.Once
	updateOffered    bool
	windowHidden     bool

	uploader       *upload.Uploader
	statsApi       *rocket_network.StatsAPI
	accountManager *manager.AccountManager
	rlSupervisor   *manager.RLSupervisor
	rlLocator      *rlgame.Locator
	bridge         *bridge.Bridge
}

func NewApp(version string, app fyne.App) *App {
	logger.FuncDebug()
	logger.Rlogger.Info("Metadata", "version", version)

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

	icon := fyne.NewStaticResource("logo.png", logoBytes)
	a.app.SetIcon(icon)

	a.window = a.app.NewWindow("Rockpload")

	a.configureSystemTray()

	a.window.SetCloseIntercept(func() {
		if a.canRunInTray() && a.appConfig != nil && a.appConfig.BehaviorConfig.ExitInTray.Get() {
			a.hideWindow()
		} else {
			a.app.Quit()
		}
	})

	return a
}

func (a *App) Close() {
	logger.FuncDebug()
	if a.bridge != nil {
		a.bridge.Stop()
	}
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

	go func() {
		updater := a.findUpdate()

		fyne.Do(func() {
			if updater == nil {
				a.start()
				return
			}

			a.offerUpdate(updater)
		})
	}()

	a.window.SetContent(ui.NewStartupScreen("Checking for updates..."))
	a.window.Resize(fyne.NewSize(700, 450))

	if a.canRunInTray() && config.PeekStartInTray() {
		a.windowHidden = true
		a.app.Run()
	} else {
		a.window.ShowAndRun()
	}
}

func (a *App) start() {
	logger.FuncDebug()

	a.startOnce.Do(func() {
		a.window.SetContent(ui.NewStartupScreen("Loading..."))

		go func() {
			appConfig := config.NewAppConfig()
			if err := appConfig.Load(a.app.Preferences()); err != nil {
				logger.Rlogger.Error("Failed to load settings:", slog.Any("err", err))
			}

			appConfig.BehaviorConfig.AutoStart.Bind(a.SetAutoStart)
			appConfig.BehaviorConfig.AutoUpload.Bind(a.SetAutoUpload)
			appConfig.BehaviorConfig.UploadOnRLClose.Bind(a.SetUploadOnRLClose)

			accountManager := manager.NewAccountManager(appConfig)

			fyne.Do(func() {
				a.appConfig = appConfig
				a.accountManager = accountManager

				a.initManager()
				a.initEvents()
				a.startManager()

				startInTray := a.canRunInTray() && a.appConfig.BehaviorConfig.ExitInTray.Get() && a.appConfig.BehaviorConfig.StartInTray.Get()
				if startInTray && !a.updateOffered {
					a.hideWindow()
				} else {
					a.showWindow()
				}
			})
		}()
	})
}

func (a *App) initManager() {
	logger.FuncDebug()

	a.uploader = upload.NewUploader(a.version, a.appConfig, a.accountManager)
	a.appConfig.BehaviorConfig.AutoUploadTime.Bind(a.uploader.SetDuration)
	a.appConfig.BehaviorConfig.AutoUploadRandomTime.Bind(a.uploader.SetMaxJitter)

	a.rlSupervisor = manager.NewRLSupervisor(a.appConfig, a.accountManager)

	a.rlLocator = rlgame.NewLocator()
	if a.supportsLocalStatsAPI() {
		a.rlLocator.Refresh()
	}

	var err error
	a.gui, err = ui.NewGUI(a.window, a.version, a.appConfig, a.accountManager, a.rlSupervisor, a.rlLocator, a.bridgeStatus, a.bridgePort, a.app.Clipboard)
	if err != nil {
		logger.Rlogger.Error("Failed to initialize GUI:", slog.Any("err", err))
	}
}

func (a *App) initEvents() {
	logger.FuncDebug()

	a.statsApi = rocket_network.NewStatsAPI()
	a.statsApi.PortsProvider = a.rlLocator.StatsAPIPorts
	uploadLiveStatsEvent := []tools.EventType{"CountdownBegin", "PodiumStart", rocket_network.EventOvertimeStarted}
	a.statsApi.EventManager.MultiSubscribe(uploadLiveStatsEvent, tools.Listener{Mode: tools.ListenerBackground, Callback: func(data any) {
		a.uploader.UploadLiveStats(a.statsApi.LastInfo)
	}})

	if a.supportsLocalStatsAPI() {
		a.statsApi.StartListener()

		a.bridge = bridge.New(a.version, a.statsApi, a.appConfig.BehaviorConfig.AllowWebsiteReplay.Get, a.appConfig.OpenReplayTargets)
		a.bridge.Start(a.appConfig.BehaviorConfig.WebsiteBridgePort.Get())
		a.appConfig.BehaviorConfig.WebsiteBridgePort.Bind(func(port int) {
			go a.bridge.Start(port)
		})
	}

	onUpdateState := func() {
		fyne.Do(a.gui.UpdateState)
	}

	a.uploader.EventManager.Subscribe(upload.EventUploadPreparing, tools.Listener{Mode: tools.ListenerUI, Callback: func(data any) {
		a.gui.SetUploadPreparing()
	}})

	a.uploader.EventManager.Subscribe(upload.EventUploadProgress, tools.Listener{Mode: tools.ListenerUI, Callback: func(data any) {
		if progress, ok := data.(float64); ok && a.gui != nil {
			a.gui.UpdateUploadProgress(progress)
		}
	}})

	guiUploaderEventList := []tools.EventType{upload.EventReplayUploaded, upload.EventUploadCompleted, upload.EventReplayUploaded}
	a.uploader.EventManager.MultiSubscribe(guiUploaderEventList, tools.Listener{Mode: tools.ListenerUI, Callback: func(data any) {
		a.gui.UpdateState()
	}})

	a.uploader.EventManager.Subscribe(upload.EventUploadStarted, tools.Listener{Mode: tools.ListenerSync, Callback: func(data any) {
		a.accountManager.RefreshInfo()
		a.appConfig.Save()
	}})

	a.gui.EventManager.Subscribe(ui.EVENT_CLICK_UPLOAD, tools.Listener{Mode: tools.ListenerBackground, Callback: func(data any) {
		a.uploader.Run()
	}})

	a.gui.EventManager.Subscribe(ui.EVENT_CLICK_UPLOAD_FILE, tools.Listener{Mode: tools.ListenerBackground, Callback: func(data any) {
		filePath, ok := data.(string)
		if !ok {
			return
		}

		a.uploader.UploadLocalFile(filePath)
	}})

	a.gui.EventManager.Subscribe(ui.EVENT_CLICK_LOAD_REPLAY_IN_GAME, tools.Listener{Mode: tools.ListenerBackground, Callback: func(data any) {
		request, ok := data.(ui.LoadReplayRequest)
		if !ok {
			return
		}

		sent, err := a.loadReplayInGame(request)
		if err != nil {
			logger.Rlogger.Error("Failed to load replay in game", slog.String("Mode", string(request.Mode)), slog.Any("err", err))
		}
		request.OnResult(sent, err)
	}})

	a.uploader.EventManager.Subscribe(upload.EventSingleUploadCompleted, tools.Listener{Mode: tools.ListenerUI, Callback: func(data any) {
		if data == nil {
			dialog.ShowInformation("Upload Complete", "The replay was uploaded successfully.", a.window)
			return
		}

		if err, ok := data.(error); ok {
			dialog.ShowError(err, a.window)
		}
	}})

	a.uploader.EventManager.Subscribe(upload.EventMatchUploadCompleted, tools.Listener{Mode: tools.ListenerUI, Callback: func(data any) {
		if err, ok := data.(error); ok && err != nil {
			dialog.ShowError(err, a.window)
			return
		}

		a.gui.UpdateState()
	}})

	a.gui.EventManager.Subscribe(ui.EVENT_CLICK_FETCH_HISTORY, tools.Listener{Mode: tools.ListenerUI, Callback: func(data any) {
		gen := a.gui.SetFetchHistoryBusy()

		go func() {
			selectedAccount := a.accountManager.GetSelected()
			err := a.accountManager.RefreshMatchHistory(selectedAccount)

			if err == nil {
				a.appConfig.Save()
			}

			fyne.Do(func() {
				a.gui.SetFetchHistoryResult(gen, err)

				if err != nil {
					dialog.ShowError(err, a.window)
					return
				}

				a.gui.UpdateState()
			})
		}()
	}})

	a.gui.EventManager.Subscribe(ui.EVENT_CLICK_UPLOAD_MATCH, tools.Listener{Mode: tools.ListenerUI, Callback: func(data any) {
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
	a.accountManager.EventManager.MultiSubscribe(guiAccountManagerEventList, tools.Listener{Mode: tools.ListenerUI, Callback: func(data any) {
		a.gui.UpdateState()
	}})

	if rocket_network.AuthEventManager == nil {
		rocket_network.AuthEventManager = tools.NewEventManager()
	}

	rocket_network.AuthEventManager.Subscribe(rocket_network.EventUserAuthenticated, tools.Listener{Mode: tools.ListenerBackground, Callback: func(data any) {
		a.profileRefreshMu.Lock()
		defer a.profileRefreshMu.Unlock()

		a.accountManager.RefreshProfile()

		if a.appConfig.BehaviorConfig.AutoUpload.Get() {
			a.uploader.Stop()
			a.uploader.Start()
		}

		onUpdateState()
		a.appConfig.Save()
	}})

	updateGUIFromSupervisorEvent := []tools.EventType{manager.EVENT_ON_RL_DETECTED, manager.EVENT_ON_RL_PLAYER_DETECTED, manager.EVENT_ON_RL_CLOSED}
	a.rlSupervisor.EventManager.MultiSubscribe(updateGUIFromSupervisorEvent, tools.Listener{Mode: tools.ListenerUI, Callback: func(data any) {
		a.gui.UpdateState()
	}})

	// A new Launch.log is written on each launch, the game could have been moved or installed elsewhere
	a.rlSupervisor.EventManager.Subscribe(manager.EVENT_ON_RL_DETECTED, tools.Listener{Mode: tools.ListenerBackground, Callback: func(data any) {
		a.rlLocator.Refresh()
	}})

	a.rlSupervisor.EventManager.Subscribe(manager.EVENT_ON_RL_CLOSED, tools.Listener{Mode: tools.ListenerBackground, Callback: func(data any) {
		if a.appConfig.BehaviorConfig.UploadOnRLClose.Get() {
			a.uploader.Run()
		}

		rlgame.ClearTempReplays()
	}})
}

func (a *App) startManager() {
	logger.FuncDebug()

	if a.appConfig.BehaviorConfig.AutoUpload.Get() {
		a.uploader.Start()
	}

	if a.supportsLocalStatsAPI() {
		go rlgame.ClearTempReplays()
	}

	go func() {
		a.profileRefreshMu.Lock()
		a.accountManager.RefreshProfile()
		a.profileRefreshMu.Unlock()

		fyne.Do(a.gui.UpdateState)
	}()

	a.startPlatformManagers()
}

func (a *App) findUpdate() (updater *Updater) {
	logger.FuncDebug()

	defer func() {
		if r := recover(); r != nil {
			logger.Rlogger.Error("Update check crashed", slog.Any("panic", r))
			updater = nil
		}
	}()

	if os.Getenv("SKIP_UPDATE") == "true" {
		return nil
	}

	updater = NewUpdater()

	needUpdate, err := updater.CheckForUpdate(a.version)
	if err != nil {
		logger.Rlogger.Error("Failed to check for update:", slog.Any("err", err))
		return nil
	}

	if !needUpdate {
		return nil
	}

	return updater
}

func (a *App) offerUpdate(updater *Updater) {
	logger.FuncDebug()

	a.updateInfo = updater.UpdateInfo
	a.updateOffered = true

	updatePopup := ui.NewUpdatePopup(ui.NewPopup("New Update!", a.window, nil, nil), a.updateInfo.Version, string(a.updateInfo.Source), updater.ApplyUpdate, a.restart, a.start)

	a.showWindow()
	updatePopup.Show()
}

func (a *App) showWindow() {
	logger.FuncDebug()

	if !a.windowHidden {
		return
	}

	a.windowHidden = false
	a.window.Show()
}

func (a *App) hideWindow() {
	logger.FuncDebug()

	a.windowHidden = true
	a.window.Hide()
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

func (a *App) bridgePort() int {
	if a.bridge != nil {
		if port := a.bridge.Port(); port != 0 {
			return port
		}
	}

	return a.appConfig.BehaviorConfig.WebsiteBridgePort.Get()
}

func (a *App) bridgeStatus() (string, bool) {
	if a.bridge == nil {
		return "", false
	}

	return a.bridge.StatusText()
}
