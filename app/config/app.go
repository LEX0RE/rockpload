package config

import (
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
)

type AppConfigType int
type OnAppConfigChange map[AppConfigType]func(bool)

const (
	AutoStart AppConfigType = iota
	AutoUpload
	ExitInTray
	StartInTray
)

var AppConfigMap = map[AppConfigType]string{
	AutoStart:   "rockpload_autoStart",
	AutoUpload:  "rockpload_autoUpload",
	ExitInTray:  "rockpload_exitInTray",
	StartInTray: "rockpload_startInTray",
}

type AppConfig struct {
	prefs             fyne.Preferences
	onAppConfigChange OnAppConfigChange
}

func NewAppConfig(prefs fyne.Preferences, onAppConfigChange OnAppConfigChange) *AppConfig {
	logger.FuncDebug()
	return &AppConfig{prefs: prefs, onAppConfigChange: onAppConfigChange}
}

func (a *AppConfig) GetAppConfig(configName AppConfigType) bool {
	logger.FuncDebug()

	switch configName {
	case AutoUpload, ExitInTray:
		return a.prefs.BoolWithFallback(AppConfigMap[configName], true)
	case AutoStart, StartInTray:
		return a.prefs.BoolWithFallback(AppConfigMap[configName], false)
	default:
		return a.prefs.BoolWithFallback(AppConfigMap[configName], false)
	}
}

func (a *AppConfig) SetAppConfig(configName AppConfigType, value bool) {
	logger.FuncDebug()
	a.prefs.SetBool(AppConfigMap[configName], value)

	if a.onAppConfigChange != nil {
		if f, ok := a.onAppConfigChange[configName]; ok {
			f(value)
		}
	}
}
