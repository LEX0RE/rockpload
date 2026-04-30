package config

import (
	"os"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"encoding/json"

	"fyne.io/fyne/v2"
)

type AppConfigType int
type OnAppConfigChange map[AppConfigType]func(bool)

type WebsiteConfig struct {
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	IsPrimary    bool              `json:"is_primary"`
	IsPredefined bool              `json:"is_predefined"`
	URIParams    map[string]string `json:"uri_params"`
	NeedToken    bool              `json:"need_token"`
	Token        string            `json:"token"`
	SendPing     bool              `json:"send_ping"`
	PingPath     string            `json:"ping_path"`
	SendReplay   bool              `json:"send_replay"`
	ReplayPath   string            `json:"replay_path"`
	// SendLive   bool // TODO Not implemented yet
	// LivePath   string // TODO Not implemented yet
}

const (
	AutoStart AppConfigType = iota
	AutoUpload
	UploadOnLaunch
	ExitInTray
	StartInTray
)

var AppConfigMap = map[AppConfigType]string{
	AutoStart:      "rockpload_autoStart",
	AutoUpload:     "rockpload_autoUpload",
	UploadOnLaunch: "rockpload_uploadOnLaunch",
	ExitInTray:     "rockpload_exitInTray",
	StartInTray:    "rockpload_startInTray",
}

var websiteConfigPrefs = "rockpload_websiteSettings"

type AppConfig struct {
	prefs                 fyne.Preferences
	onAppConfigChange     OnAppConfigChange
	onWebsiteConfigChance func([]*WebsiteConfig)
}

func NewAppConfig(prefs fyne.Preferences, onAppConfigChange OnAppConfigChange, onWebsiteConfigChance func([]*WebsiteConfig)) *AppConfig {
	logger.FuncDebug()
	return &AppConfig{prefs: prefs, onAppConfigChange: onAppConfigChange, onWebsiteConfigChance: onWebsiteConfigChance}
}

func (a *AppConfig) GetAppConfig(configName AppConfigType) bool {
	logger.FuncDebug()

	switch configName {
	case AutoUpload, ExitInTray:
		return a.prefs.BoolWithFallback(AppConfigMap[configName], true)
	case AutoStart, StartInTray, UploadOnLaunch:
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

func (a *AppConfig) GetWebsiteAppConfig() []*WebsiteConfig {
	logger.FuncDebug()

	jsonData := a.prefs.StringWithFallback(websiteConfigPrefs, "[]")

	var websites []*WebsiteConfig
	json.Unmarshal([]byte(jsonData), &websites)

	websites = append([]*WebsiteConfig{ROCKY_WEBSITE}, websites...)

	if len(websites) == 1 {
		predefinedWebsites := []*WebsiteConfig{BALLCHASING_WEBSITE}
		websites = append(websites, predefinedWebsites...)
	}

	addLocalhost := os.Getenv("ADD_LOCALHOST")
	if addLocalhost == "true" {
		websites = append(websites, LOCAL_WEBSITE)
	}

	return websites
}

func (a *AppConfig) SetWebsiteAppConfig(value []*WebsiteConfig) {
	logger.FuncDebug()

	filteredWebsite := []*WebsiteConfig{}
	for _, website := range value {
		if !website.IsPrimary {
			filteredWebsite = append(filteredWebsite, website)
		}
	}

	jsonData, err := json.Marshal(filteredWebsite)
	if err == nil {
		a.prefs.SetString(websiteConfigPrefs, string(jsonData))
	}

	if a.onWebsiteConfigChance != nil {
		a.onWebsiteConfigChance(value)
	}
}
