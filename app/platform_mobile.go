//go:build android || ios

package app

import "github.com/LEX0RE/rockpload/app/tools/logger"

func (a *App) configureSystemTray() {}

func (a *App) canRunInTray() bool {
	return false
}

func (a *App) supportsLocalStatsAPI() bool {
	return false
}

func (a *App) supportsSelfUpdate() bool {
	return false
}

func (a *App) startPlatformManagers() {}

func (a *App) createDuplicateLock() bool {
	return true
}

func (a *App) SetAutoStart(value bool) {
	if value {
		logger.Rlogger.Info("Autostart is not supported on mobile")
	}
}

func (a *App) restart() {
	logger.Rlogger.Info("Restart after update is not supported on mobile")
	a.Close()
}
