package config

import (
	"encoding/json"
	"log/slog"
	"os"

	"fyne.io/fyne/v2"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

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

type AppConfig struct {
	AutoUpload      Setting[bool]             `json:"auto_upload"`
	ExitInTray      Setting[bool]             `json:"exit_in_tray"`
	AutoStart       Setting[bool]             `json:"auto_start"`
	StartInTray     Setting[bool]             `json:"start_in_tray"`
	UploadOnLaunch  Setting[bool]             `json:"upload_on_launch"`
	WebsiteSettings Setting[[]*WebsiteConfig] `json:"website_settings"`
}

func NewAppConfig() *AppConfig {
	logger.FuncDebug()

	cfg := &AppConfig{}

	saveHook := func() {
		cfg.Save()
	}

	cfg.AutoUpload = NewSetting(true, saveHook)
	cfg.ExitInTray = NewSetting(true, saveHook)

	cfg.AutoStart = NewSetting(false, saveHook)
	cfg.StartInTray = NewSetting(false, saveHook)
	cfg.UploadOnLaunch = NewSetting(false, saveHook)

	cfg.WebsiteSettings = NewSetting([]*WebsiteConfig{}, saveHook)

	err := cfg.Load()
	if err != nil {
		logger.Rlogger.Error("Failed to load settings:", slog.Any("err", err))
	}

	return cfg
}

func (a *AppConfig) Load() error {
	logger.FuncDebug()

	err := LoadFilePath(SettingsFile, a, false)
	if err != nil {
		return err
	}

	a.WebsiteSettings.value = append([]*WebsiteConfig{ROCKY_WEBSITE}, a.WebsiteSettings.value...)

	if len(a.WebsiteSettings.value) == 1 {
		predefinedWebsites := []*WebsiteConfig{BALLCHASING_WEBSITE}
		a.WebsiteSettings.value = append(a.WebsiteSettings.value, predefinedWebsites...)
	}

	addLocalhost := os.Getenv("ADD_LOCALHOST")
	if addLocalhost == "true" {
		a.WebsiteSettings.value = append(a.WebsiteSettings.value, LOCAL_WEBSITE)
	}

	return nil
}

func (a *AppConfig) Save() error {
	logger.FuncDebug()

	originalWebsites := a.WebsiteSettings.value

	filteredWebsites := []*WebsiteConfig{}
	for _, website := range originalWebsites {
		if !website.IsPrimary {
			filteredWebsites = append(filteredWebsites, website)
		}
	}

	a.WebsiteSettings.value = filteredWebsites

	err := SaveFilePath(SettingsFile, a)

	a.WebsiteSettings.value = originalWebsites

	return err
}

// TODO Deprecated, Fyne Pref will be removed in future version
func (a *AppConfig) ImportFynePreferences(prefs fyne.Preferences) {
	logger.FuncDebug()

	savedAndRemove := func(key string) {
		if a.Save() == nil {
			prefs.RemoveValue(key)
		}
	}

	a.AutoUpload.value = prefs.BoolWithFallback("rockpload_autoUpload", a.AutoUpload.value)
	savedAndRemove("rockpload_autoUpload")
	a.ExitInTray.value = prefs.BoolWithFallback("rockpload_exitInTray", a.ExitInTray.value)
	savedAndRemove("rockpload_exitInTray")

	a.AutoStart.value = prefs.BoolWithFallback("rockpload_autoStart", a.AutoStart.value)
	savedAndRemove("rockpload_autoStart")
	a.StartInTray.value = prefs.BoolWithFallback("rockpload_startInTray", a.StartInTray.value)
	savedAndRemove("rockpload_startInTray")
	a.UploadOnLaunch.value = prefs.BoolWithFallback("rockpload_uploadOnLaunch", a.UploadOnLaunch.value)
	savedAndRemove("rockpload_uploadOnLaunch")

	jsonData := prefs.StringWithFallback("rockpload_websiteSettings", "[]")
	var websites []*WebsiteConfig
	json.Unmarshal([]byte(jsonData), &websites)

	for _, importedSite := range websites {
		found := false

		for _, configSite := range a.WebsiteSettings.Get() {
			if configSite.Name == importedSite.Name {
				found = true
				break
			}
		}

		if !found {
			a.WebsiteSettings.value = append(a.WebsiteSettings.value, importedSite)
		} else if importedSite.IsPredefined {
			for i, configSite := range a.WebsiteSettings.Get() {
				if configSite.Name == importedSite.Name {
					a.WebsiteSettings.value[i] = importedSite
				}
			}
		}
	}

	savedAndRemove("rockpload_websiteSettings")
}
