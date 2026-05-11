package config

import (
	"encoding/json"
	"errors"

	"fyne.io/fyne/v2"
	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type AppConfig struct {
	BehaviorConfig  BehaviorConfig
	StorageSettings Setting[storageListConfig]
	AccountSettings Setting[map[int]*AccountConfig]
}

func NewAppConfig() *AppConfig {
	logger.FuncDebug()

	cfg := &AppConfig{}

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
