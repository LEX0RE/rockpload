package config

import (
	"encoding/json"
	"errors"
	"log/slog"

	"fyne.io/fyne/v2"
	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	EVENT_SELECTED_ACCOUNT_CHANGED = "selected_account_changed"
)

type AppConfig struct {
	BehaviorConfig  BehaviorConfig
	StorageSettings Setting[storageListConfig]
	AccountSettings Setting[map[int]*AccountConfig]

	EventManager *tools.EventManager
}

func NewAppConfig() *AppConfig {
	logger.FuncDebug()

	cfg := &AppConfig{EventManager: tools.NewEventManager()}

	saveHook := func() {
		cfg.Save()
	}

	cfg.BehaviorConfig.AutoUpload = NewSetting(true, saveHook)
	cfg.BehaviorConfig.ExitInTray = NewSetting(true, saveHook)
	cfg.BehaviorConfig.NoUploadConnected = NewSetting(true, saveHook)

	cfg.BehaviorConfig.AutoStart = NewSetting(false, saveHook)
	cfg.BehaviorConfig.StartInTray = NewSetting(false, saveHook)
	cfg.BehaviorConfig.UploadOnLaunch = NewSetting(false, saveHook)

	cfg.StorageSettings = NewSetting(storageListConfig{}, saveHook)
	cfg.AccountSettings = NewSetting(make(map[int]*AccountConfig), saveHook)

	return cfg
}

func (ac *AppConfig) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	type Alias AppConfig

	aux := (*Alias)(ac)

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	ac.EventManager = tools.NewEventManager()

	return nil
}

// TODO prefs parameters is deprecated, will be removed in future version
func (a *AppConfig) Load(prefs fyne.Preferences) error {
	logger.FuncDebug()

	tools.WaitFileBoot(constant.BehaviorSettingsFile)

	if err := tools.LoadJSONFilePath(constant.BehaviorSettingsFile, &a.BehaviorConfig, false); err != nil {
		return err
	}

	if err := tools.LoadJSONFilePath(constant.StorageSettingsFile, &a.StorageSettings.value, false); err != nil {
		return err
	}

	if err := tools.LoadJSONFilePath(constant.AccountSettingsFile, &a.AccountSettings.value, false); err != nil {
		return err
	}

	// TODO Deprecated, Fyne Pref will be removed in future version
	a.importFynePreferences(prefs)

	a.AuthAllAccounts()

	return nil
}

func (a *AppConfig) Save() error {
	logger.FuncDebug()

	originalWebsites := a.StorageSettings.value

	filteredWebsites := storageListConfig{}
	for _, website := range originalWebsites {
		if !website.IsPrimary {
			filteredWebsites = append(filteredWebsites, website)
		}
	}

	a.StorageSettings.value = filteredWebsites

	accountErr := tools.SaveJSONFilePath(constant.AccountSettingsFile, a.AccountSettings.Get())
	storageErr := tools.SaveJSONFilePath(constant.StorageSettingsFile, a.StorageSettings.Get())
	behaviorErr := tools.SaveJSONFilePath(constant.BehaviorSettingsFile, a.BehaviorConfig)

	a.StorageSettings.value = originalWebsites

	if accountErr == nil && storageErr == nil && behaviorErr == nil {
		return nil
	}

	return errors.New("Failed to save settings: " + errors.Join(accountErr, storageErr, behaviorErr).Error())
}

// TODO Deprecated, Fyne Pref will be removed in future version
func (a *AppConfig) importFynePreferences(prefs fyne.Preferences) {
	logger.FuncDebug()

	savedAndRemove := func(key string) {
		if a.Save() == nil {
			prefs.RemoveValue(key)
		}
	}

	a.BehaviorConfig.AutoUpload.value = prefs.BoolWithFallback("rockpload_autoUpload", a.BehaviorConfig.AutoUpload.value)
	savedAndRemove("rockpload_autoUpload")
	a.BehaviorConfig.ExitInTray.value = prefs.BoolWithFallback("rockpload_exitInTray", a.BehaviorConfig.ExitInTray.value)
	savedAndRemove("rockpload_exitInTray")

	a.BehaviorConfig.AutoStart.value = prefs.BoolWithFallback("rockpload_autoStart", a.BehaviorConfig.AutoStart.value)
	savedAndRemove("rockpload_autoStart")
	a.BehaviorConfig.StartInTray.value = prefs.BoolWithFallback("rockpload_startInTray", a.BehaviorConfig.StartInTray.value)
	savedAndRemove("rockpload_startInTray")
	a.BehaviorConfig.UploadOnLaunch.value = prefs.BoolWithFallback("rockpload_uploadOnLaunch", a.BehaviorConfig.UploadOnLaunch.value)
	savedAndRemove("rockpload_uploadOnLaunch")

	jsonData := prefs.StringWithFallback("rockpload_websiteSettings", "[]")
	var websites []*StorageConfig
	json.Unmarshal([]byte(jsonData), &websites)

	for _, importedSite := range websites {
		found := false

		for _, configSite := range a.StorageSettings.Get() {
			if configSite.Name == importedSite.Name {
				found = true
				break
			}
		}

		if !found {
			a.StorageSettings.value = append(a.StorageSettings.value, importedSite)
		} else if importedSite.IsPredefined && !importedSite.IsPrimary {
			for i, configSite := range a.StorageSettings.Get() {
				if configSite.Name == importedSite.Name {
					a.StorageSettings.value[i] = importedSite
				}
			}
		}
	}

	savedAndRemove("rockpload_websiteSettings")
}

func (a *AppConfig) SelectedAccount() *AccountConfig {
	logger.FuncDebug()

	if ac, ok := a.AccountSettings.Get()[a.BehaviorConfig.SelectedAccountId.Get()]; ok {
		return ac
	} else if ac, ok := a.AccountSettings.Get()[0]; ok {
		a.BehaviorConfig.SelectedAccountId.Set(0)
		return ac
	} else {
		a.BehaviorConfig.SelectedAccountId.Set(0)
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
		a.BehaviorConfig.SelectedAccountId.Set(accountId)
	} else {
		a.BehaviorConfig.SelectedAccountId.Set(0)
	}

	a.EventManager.Notify(EVENT_SELECTED_ACCOUNT_CHANGED, a.BehaviorConfig.SelectedAccountId.Get())
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

		if a.BehaviorConfig.SelectedAccountId.Get() == accountId {
			a.SetSelectedAccount(0)
		}
	}
}

func (a *AppConfig) AuthAllAccounts() {
	logger.FuncDebug()

	for _, ac := range a.AccountSettings.Get() {
		ac.Player.Auth.Authenticate()
		if err := ac.Player.Auth.Authenticate(); err != nil {
			logger.Rlogger.Error("Failed to authenticate:", slog.Any("err", err))
		}
	}
}
