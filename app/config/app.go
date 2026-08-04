package config

import (
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"time"

	"fyne.io/fyne/v2"
	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type AppConfig struct {
	BehaviorConfig  BehaviorConfig
	StorageSettings Setting[storageListConfig]
	AccountSettings Setting[AccountMapConfig]
}

func NewAppConfig() *AppConfig {
	logger.FuncDebug()

	cfg := &AppConfig{}

	saveHook := func() {
		cfg.Save()
	}

	// TODO Find a way to make this in Unmarshal instead
	cfg.BehaviorConfig.AutoUpload = NewSetting(true, saveHook)
	cfg.BehaviorConfig.AutoUploadTime = NewSetting(45*time.Minute, saveHook)
	cfg.BehaviorConfig.AutoUploadRandomTime = NewSetting(15*time.Minute, saveHook)
	cfg.BehaviorConfig.ExitInTray = NewSetting(true, saveHook)
	cfg.BehaviorConfig.NoUploadOnline = NewSetting(true, saveHook)
	cfg.BehaviorConfig.UploadOnRLClose = NewSetting(true, saveHook)
	cfg.BehaviorConfig.SendLiveStat = NewSetting(true, saveHook)

	cfg.BehaviorConfig.AutoStart = NewSetting(false, saveHook)
	cfg.BehaviorConfig.StartInTray = NewSetting(false, saveHook)
	cfg.BehaviorConfig.UploadOnLaunch = NewSetting(false, saveHook)
	cfg.BehaviorConfig.UploadOlderFirst = NewSetting(false, saveHook)

	cfg.BehaviorConfig.SelectedAccountId = NewSetting(0, saveHook)
	cfg.BehaviorConfig.SelectedStorageId = NewSetting(0, saveHook)

	cfg.StorageSettings = NewSetting(storageListConfig{}, saveHook)
	cfg.AccountSettings = NewSetting(make(AccountMapConfig), saveHook)

	return cfg
}

// TODO prefs parameters is deprecated, will be removed in future version
func (a *AppConfig) Load(prefs fyne.Preferences) error {
	logger.FuncDebug()

	tools.WaitFileBoot([]string{constant.Paths.BehaviorSettingsFile, constant.Paths.AppLog})

	if err := tools.LoadJSONFilePath(constant.Paths.BehaviorSettingsFile, &a.BehaviorConfig, false); err != nil {
		return err
	}

	if err := tools.LoadJSONFilePath(constant.Paths.StorageSettingsFile, &a.StorageSettings.value, false); err != nil {
		return err
	}

	if len(a.StorageSettings.value) == 0 {
		a.StorageSettings.value.UnmarshalJSON([]byte{})
	}

	if err := tools.LoadJSONFilePath(constant.Paths.AccountSettingsFile, &a.AccountSettings.value, false); err != nil {
		return err
	}

	if len(a.AccountSettings.value) == 0 {
		a.AccountSettings.value.UnmarshalJSON([]byte{})
	}

	if err := a.loadSecretSettings(); err != nil {
		return err
	}

	if err := a.migrateDepreciated(prefs); err != nil {
		return err
	}

	logger.Rlogger.Debug("Application configuration loaded", slog.String("config", a.loggableConfig()))

	return nil
}

func (a *AppConfig) loggableConfig() string {
	restoreSecretSettings, err := extractSecretSettings(reflect.ValueOf(a), nil, secretSettings{})
	if err != nil {
		return "failed to prepare app config for logging: " + err.Error()
	}
	defer restoreSecretSettings()

	data, err := json.Marshal(a)
	if err != nil {
		return "failed to marshal app config: " + err.Error()
	}

	return string(data)
}

func (a *AppConfig) Save() error {
	logger.FuncDebug()

	restoreSecretSettings, secretErr := a.saveSecretSettings()
	defer restoreSecretSettings()

	accountErr, storageErr, behaviorErr := a.savePublicSettings()

	if accountErr == nil && storageErr == nil && behaviorErr == nil && secretErr == nil {
		return nil
	}

	return errors.New("Failed to save settings: " + errors.Join(accountErr, storageErr, behaviorErr, secretErr).Error())
}

func (a *AppConfig) savePublicSettings() (error, error, error) {
	originalWebsites := a.StorageSettings.value

	filteredWebsites := storageListConfig{}
	for _, website := range originalWebsites {
		if !website.IsPrimary && !website.IsTemporary {
			filteredWebsites = append(filteredWebsites, website)
		}
	}

	a.StorageSettings.value = filteredWebsites

	accountErr := tools.SaveJSONFilePath(constant.Paths.AccountSettingsFile, a.AccountSettings.Get())
	storageErr := tools.SaveJSONFilePath(constant.Paths.StorageSettingsFile, a.StorageSettings.Get())
	behaviorErr := tools.SaveJSONFilePath(constant.Paths.BehaviorSettingsFile, a.BehaviorConfig)

	a.StorageSettings.value = originalWebsites

	return accountErr, storageErr, behaviorErr
}

// TODO Deprecated section for previous setup. These will progressively be removed in future versions.
func (a *AppConfig) migrateDepreciated(prefs fyne.Preferences) error {
	// TODO Deprecated, Fyne Pref will be removed
	modified := a.importFynePreferences(prefs)

	// TODO Deprecated, previous storage settings with bool instead of enum
	storageModified, err := a.migrateStorageStyles()
	if err != nil {
		return err
	}

	if modified || storageModified {
		a.Save()
	}

	// TODO Deprecated, there won't have any migration of previous config
	if err := a.migrateSecretSettings(); err != nil {
		return err
	}

	return nil
}

// TODO Deprecated, Fyne Pref will be removed
func (a *AppConfig) importFynePreferences(prefs fyne.Preferences) bool {
	logger.FuncDebug()

	modified := false

	importBool := func(key string, target *bool) {
		newVal := prefs.BoolWithFallback(key, *target)

		if newVal != *target {
			*target = newVal
			modified = true
		}

		prefs.RemoveValue(key)
	}

	importBool("rockpload_autoUpload", &a.BehaviorConfig.AutoUpload.value)
	importBool("rockpload_exitInTray", &a.BehaviorConfig.ExitInTray.value)
	importBool("rockpload_autoStart", &a.BehaviorConfig.AutoStart.value)
	importBool("rockpload_startInTray", &a.BehaviorConfig.StartInTray.value)
	importBool("rockpload_uploadOnLaunch", &a.BehaviorConfig.UploadOnLaunch.value)

	jsonData := prefs.StringWithFallback("rockpload_websiteSettings", "")

	if jsonData != "" && jsonData != "[]" {
		var websites []*StorageConfig

		if err := json.Unmarshal([]byte(jsonData), &websites); err == nil {
			for _, importedSite := range websites {
				found := false

				for i, configSite := range a.StorageSettings.Get() {
					if configSite.Name == importedSite.Name {
						found = true

						if importedSite.IsPredefined && !importedSite.IsPrimary {
							a.StorageSettings.value[i] = importedSite
							modified = true
						}
						break
					}
				}

				if !found {
					a.StorageSettings.value = append(a.StorageSettings.value, importedSite)
					modified = true
				}
			}
		}
	}
	prefs.RemoveValue("rockpload_websiteSettings")

	return modified
}
