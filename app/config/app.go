package config

import (
	"encoding/json"
	"os"

	"fyne.io/fyne/v2"
	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	EVENT_SELECTED_ACCOUNT_CHANGED = "selected_account_changed"
)

type AppConfig struct {
	AutoUpload      Setting[bool]                   `json:"auto_upload"`
	ExitInTray      Setting[bool]                   `json:"exit_in_tray"`
	AutoStart       Setting[bool]                   `json:"auto_start"`
	StartInTray     Setting[bool]                   `json:"start_in_tray"`
	UploadOnLaunch  Setting[bool]                   `json:"upload_on_launch"`
	WebsiteSettings Setting[[]*WebsiteConfig]       `json:"website_settings"`
	AccountSettings Setting[map[int]*AccountConfig] `json:"account_settings"`

	SelectedAccountId int `json:"selected_account_id"`

	EventManager *tools.EventManager `json:"-"`
}

func NewAppConfig() *AppConfig {
	logger.FuncDebug()

	cfg := &AppConfig{EventManager: tools.NewEventManager()}

	saveHook := func() {
		cfg.Save()
	}

	cfg.AutoUpload = NewSetting(true, saveHook)
	cfg.ExitInTray = NewSetting(true, saveHook)

	cfg.AutoStart = NewSetting(false, saveHook)
	cfg.StartInTray = NewSetting(false, saveHook)
	cfg.UploadOnLaunch = NewSetting(false, saveHook)

	cfg.WebsiteSettings = NewSetting([]*WebsiteConfig{}, saveHook)
	cfg.AccountSettings = NewSetting(make(map[int]*AccountConfig), saveHook)

	return cfg
}

// TODO prefs parameters is deprecated, will be removed in future version
func (a *AppConfig) Load(prefs fyne.Preferences) error {
	logger.FuncDebug()

	WaitFileBoot(constant.SettingsFile)

	err := LoadFilePath(constant.SettingsFile, a, false)
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

	if len(a.AccountSettings.Get()) == 0 {
		a.AccountSettings.value = map[int]*AccountConfig{
			0: NewAccountConfig(0),
		}
	}

	// TODO Deprecated, Fyne Pref will be removed in future version
	a.importFynePreferences(prefs)

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

	err := SaveFilePath(constant.SettingsFile, a)

	a.WebsiteSettings.value = originalWebsites

	return err
}

// TODO Deprecated, Fyne Pref will be removed in future version
func (a *AppConfig) importFynePreferences(prefs fyne.Preferences) {
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

func (a *AppConfig) SelectedAccount() *AccountConfig {
	logger.FuncDebug()

	if ac, ok := a.AccountSettings.Get()[a.SelectedAccountId]; ok {
		return ac
	} else if ac, ok := a.AccountSettings.Get()[0]; ok {
		a.SelectedAccountId = 0
		return ac
	} else {
		a.SelectedAccountId = 0
		return a.AddAccount()
	}
}

func (a *AppConfig) UnusedAccount() *AccountConfig {
	logger.FuncDebug()

	var currentAlt *AccountConfig

	for _, ac := range a.AccountSettings.Get() {
		if ac.IsUnused {
			if currentAlt != nil {
				currentAlt.IsUnused = false
				ac.IsUnused = true
			}

			currentAlt = ac
		}
	}

	return currentAlt
}

func (a *AppConfig) SetSelectedAccount(accountId int) {
	logger.FuncDebug()

	if _, ok := a.AccountSettings.Get()[accountId]; ok {
		a.SelectedAccountId = accountId
	} else {
		a.SelectedAccountId = 0
	}

	a.EventManager.Notify(EVENT_SELECTED_ACCOUNT_CHANGED, a.SelectedAccountId)
}

func (a *AppConfig) SetUnusedAccount(accountId int) {
	logger.FuncDebug()

	if _, ok := a.AccountSettings.Get()[accountId]; ok {
		for _, ac := range a.AccountSettings.Get() {
			ac.IsUnused = false
		}

		a.AccountSettings.Get()[accountId].IsUnused = true
	} else {
		for _, ac := range a.AccountSettings.Get() {
			ac.IsUnused = false
		}
	}

	a.Save()
}

func (a *AppConfig) AddAccount() *AccountConfig {
	logger.FuncDebug()

	for i := 0; ; i++ {
		if _, ok := a.AccountSettings.Get()[i]; !ok {
			currentAccountSettings := a.AccountSettings.Get()
			currentAccountSettings[i] = NewAccountConfig(i)
			a.AccountSettings.Set(currentAccountSettings)
			return currentAccountSettings[i]
		}
	}
}

func (a *AppConfig) DeleteAccount(accountId int) {
	logger.FuncDebug()

	if len(a.AccountSettings.Get()) <= 1 {
		return
	}

	if _, ok := a.AccountSettings.Get()[accountId]; ok {
		currentAccountSettings := a.AccountSettings.Get()
		delete(currentAccountSettings, accountId)
		a.AccountSettings.Set(currentAccountSettings)

		if a.SelectedAccountId == accountId {
			a.SetSelectedAccount(0)
		}
	}
}
