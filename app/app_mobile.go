//go:build android || ios

package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type mobileDuplicateLock struct{}

func (m mobileDuplicateLock) Unlock() error {
	logger.FuncDebug()
	return nil
}

type mobileScaledTheme struct {
	fyne.Theme
}

func (m *mobileScaledTheme) Size(name fyne.ThemeSizeName) float32 {
	logger.FuncDebug()

	baseSize := m.Theme.Size(name)
	return baseSize * 0.55
}

func (a *App) configureApp() {
	logger.FuncDebug()

	a.app.Settings().SetTheme(&mobileScaledTheme{Theme: theme.DefaultTheme()})
}

func (a *App) configureSystemTray() {
	logger.FuncDebug()
}

func (a *App) canRunInTray() bool {
	logger.FuncDebug()

	return false
}

func (a *App) supportsLocalStatsAPI() bool {
	logger.FuncDebug()

	return false
}

func (a *App) startPlatformManagers() {
	logger.FuncDebug()
}

func (a *App) createDuplicateLock() bool {
	logger.FuncDebug()

	a.duplicateLock = mobileDuplicateLock{}
	return true
}

func (a *App) SetAutoStart(value bool) {
	logger.FuncDebug()

	if value {
		logger.Rlogger.Info("Autostart is not supported on mobile")
	}
}

func (a *App) restart() {
	logger.FuncDebug()
	a.Close()
}
