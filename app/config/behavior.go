package config

import (
	"fmt"
	"time"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type BehaviorConfig struct {
	AutoUpload           Setting[bool]          `json:"auto_upload"`
	AutoUploadTime       Setting[time.Duration] `json:"auto_upload_time"`
	AutoUploadRandomTime Setting[time.Duration] `json:"auto_upload_random_time"`
	UploadOnLaunch       Setting[bool]          `json:"upload_on_launch"`
	UploadOnRLClose      Setting[bool]          `json:"upload_on_rl_close"`
	NoUploadOnline       Setting[bool]          `json:"no_upload_online"`
	AutoStart            Setting[bool]          `json:"auto_start"`
	ExitInTray           Setting[bool]          `json:"exit_in_tray"`
	StartInTray          Setting[bool]          `json:"start_in_tray"`
	UploadOlderFirst     Setting[bool]          `json:"upload_older_first"`
	SendLiveStat         Setting[bool]          `json:"send_live_stat"`

	SelectedAccountId Setting[int] `json:"selected_account_id"`
	SelectedStorageId Setting[int] `json:"selected_storage_id"`
}

type BehaviorSettingType int

const (
	AutoUpload BehaviorSettingType = iota
	AutoUploadTime
	AutoUploadRandomTime
	UploadOnLaunch
	UploadOnRLClose
	NoUploadOnline
	AutoStart
	ExitInTray
	StartInTray
	UploadOlderFirst
	SendLiveStat
)

type BehaviorSettingVisualDependency struct {
	Name     string
	Children []BehaviorSettingType
	Position int
	Kind     SettingKind
	IsMobile bool
	Setting  AnySetting
	Validate func(any) error
}

func minimumDuration(minimum time.Duration) func(any) error {
	return func(value any) error {
		duration, ok := value.(time.Duration)
		if !ok {
			return fmt.Errorf("expected a duration, got %T", value)
		}
		if duration < minimum {
			return fmt.Errorf("duration must be at least %s", minimum)
		}
		return nil
	}
}

func (bc *BehaviorConfig) GetSettingsMap() map[BehaviorSettingType]BehaviorSettingVisualDependency {
	logger.FuncDebug()

	return map[BehaviorSettingType]BehaviorSettingVisualDependency{
		AutoUpload: {
			Name:     "Auto Upload Replays",
			Children: []BehaviorSettingType{AutoUploadTime, AutoUploadRandomTime},
			Position: int(AutoUpload),
			Kind:     SettingKindBool,
			IsMobile: true,
			Setting:  &bc.AutoUpload,
		},
		AutoUploadTime: {
			Name:     "Auto Upload Refresh Time",
			Children: []BehaviorSettingType{},
			Position: int(AutoUploadTime),
			Kind:     SettingKindDuration,
			IsMobile: true,
			Setting:  &bc.AutoUploadTime,
			Validate: minimumDuration(time.Minute),
		},
		AutoUploadRandomTime: {
			Name:     "Auto Upload Random Time",
			Children: []BehaviorSettingType{},
			Position: int(AutoUploadRandomTime),
			Kind:     SettingKindDuration,
			IsMobile: true,
			Setting:  &bc.AutoUploadRandomTime,
			Validate: minimumDuration(0),
		},
		UploadOnLaunch: {
			Name:     "Upload Replays on launch",
			Children: []BehaviorSettingType{},
			Position: int(UploadOnLaunch),
			Kind:     SettingKindBool,
			IsMobile: true,
			Setting:  &bc.UploadOnLaunch,
		},
		UploadOnRLClose: {
			Name:     "Upload when RL is closed",
			Children: []BehaviorSettingType{},
			Position: int(UploadOnRLClose),
			Kind:     SettingKindBool,
			IsMobile: false,
			Setting:  &bc.UploadOnRLClose,
		},
		NoUploadOnline: {
			Name:     "No Upload if player is online",
			Children: []BehaviorSettingType{},
			Position: int(NoUploadOnline),
			Kind:     SettingKindBool,
			IsMobile: true,
			Setting:  &bc.NoUploadOnline,
		},
		AutoStart: {
			Name:     "Start with system",
			Children: []BehaviorSettingType{},
			Position: int(AutoStart),
			Kind:     SettingKindBool,
			IsMobile: false,
			Setting:  &bc.AutoStart,
		},
		ExitInTray: {
			Name:     "Exit in System Tray",
			Children: []BehaviorSettingType{StartInTray},
			Position: int(ExitInTray),
			Kind:     SettingKindBool,
			IsMobile: false,
			Setting:  &bc.ExitInTray,
		},
		StartInTray: {
			Name:     "Start in Tray",
			Children: []BehaviorSettingType{},
			Position: int(StartInTray),
			Kind:     SettingKindBool,
			IsMobile: false,
			Setting:  &bc.StartInTray,
		},
		UploadOlderFirst: {
			Name:     "Upload older replay first",
			Children: []BehaviorSettingType{},
			Position: int(UploadOlderFirst),
			Kind:     SettingKindBool,
			IsMobile: true,
			Setting:  &bc.UploadOlderFirst,
		},
		SendLiveStat: {
			Name:     "Send Live Stats (with StatsAPI)",
			Children: []BehaviorSettingType{},
			Position: int(SendLiveStat),
			Kind:     SettingKindBool,
			IsMobile: false,
			Setting:  &bc.SendLiveStat,
		},
	}
}
