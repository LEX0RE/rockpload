package config

type BehaviorConfig struct {
	AutoUpload        Setting[bool] `json:"auto_upload"`
	ExitInTray        Setting[bool] `json:"exit_in_tray"`
	AutoStart         Setting[bool] `json:"auto_start"`
	StartInTray       Setting[bool] `json:"start_in_tray"`
	UploadOnLaunch    Setting[bool] `json:"upload_on_launch"`
	NoUploadConnected Setting[bool] `json:"no_upload_connected"`
	UploadOnRLClose   Setting[bool] `json:"upload_on_rl_close"`

	SelectedAccountId Setting[int] `json:"selected_account_id"`
	SelectedStorageId Setting[int] `json:"selected_storage_id"`
}

type BehaviorSettingType int

const (
	AutoUpload BehaviorSettingType = iota
	ExitInTray
	AutoStart
	StartInTray
	UploadOnLaunch
	NoUploadConnected
	UploadOnRLClose
)

type BehaviorSettingVisualDependancy struct {
	Name     string
	Children []BehaviorSettingType
	Position int
}

var BehaviorSettingVisualMapping = map[BehaviorSettingType]BehaviorSettingVisualDependancy{
	AutoUpload:        {Name: "Auto Upload Replays", Children: []BehaviorSettingType{}, Position: 0},
	ExitInTray:        {Name: "Exit in System Tray", Children: []BehaviorSettingType{StartInTray}, Position: 5},
	AutoStart:         {Name: "Start with system", Children: []BehaviorSettingType{}, Position: 4},
	StartInTray:       {Name: "Start in Tray", Children: []BehaviorSettingType{}, Position: 6},
	UploadOnLaunch:    {Name: "Upload Replays on launch", Children: []BehaviorSettingType{}, Position: 1},
	NoUploadConnected: {Name: "No Upload if connected (Only when Unused Account is set)", Children: []BehaviorSettingType{}, Position: 3},
	UploadOnRLClose:   {Name: "Upload when RL is closed", Children: []BehaviorSettingType{}, Position: 2},
}

func (c *BehaviorConfig) GetBoolSettingsMap() map[BehaviorSettingType]*Setting[bool] {
	return map[BehaviorSettingType]*Setting[bool]{
		AutoUpload:        &c.AutoUpload,
		ExitInTray:        &c.ExitInTray,
		AutoStart:         &c.AutoStart,
		StartInTray:       &c.StartInTray,
		UploadOnLaunch:    &c.UploadOnLaunch,
		NoUploadConnected: &c.NoUploadConnected,
		UploadOnRLClose:   &c.UploadOnRLClose,
	}
}
