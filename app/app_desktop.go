//go:build !android && !ios

package app

import (
	"log/slog"
	"os"
	"os/exec"

	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/emersion/go-autostart"
	"github.com/gofrs/flock"
)

func (a *App) configureSystemTray() {
	logger.FuncDebug()

	if desk, ok := a.app.(desktop.App); ok {
		menu := fyne.NewMenu("Rockpload",
			fyne.NewMenuItem("Open", func() {
				a.window.Show()
			}),
		)

		desk.SetSystemTrayMenu(menu)
	}
}

func (a *App) configureApp() {
	logger.FuncDebug()
}

func (a *App) canRunInTray() bool {
	logger.FuncDebug()

	return true
}

func (a *App) supportsLocalStatsAPI() bool {
	logger.FuncDebug()

	return true
}

func (a *App) startPlatformManagers() {
	logger.FuncDebug()

	a.rlSupervisor.Start()
}

func (a *App) createDuplicateLock() bool {
	logger.FuncDebug()

	lock := flock.New(constant.Paths.AppLock)
	gotLocked, err := lock.TryLock()
	if err != nil {
		panic(err)
	}

	a.duplicateLock = lock
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
