package config

type BehaviorConfig struct {
	AutoUpload       Setting[bool] `json:"auto_upload"`
	ExitInTray       Setting[bool] `json:"exit_in_tray"`
	AutoStart        Setting[bool] `json:"auto_start"`
	StartInTray      Setting[bool] `json:"start_in_tray"`
	UploadOnLaunch   Setting[bool] `json:"upload_on_launch"`
	NoUploadOnline   Setting[bool] `json:"no_upload_online"`
	UploadOnRLClose  Setting[bool] `json:"upload_on_rl_close"`
	UploadOlderFirst Setting[bool] `json:"upload_older_first"`
	SendLiveStat     Setting[bool] `json:"send_live_stat"`

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
	NoUploadOnline
	UploadOnRLClose
	UploadOlderFirst
	SendLiveStat
)

type BehaviorSettingVisualDependancy struct {
	Name     string
	Children []BehaviorSettingType
	Position int
}

var BehaviorSettingVisualMapping = map[BehaviorSettingType]BehaviorSettingVisualDependancy{
	AutoUpload:       {Name: "Auto Upload Replays", Children: []BehaviorSettingType{}, Position: 0},
	ExitInTray:       {Name: "Exit in System Tray", Children: []BehaviorSettingType{StartInTray}, Position: 5},
	AutoStart:        {Name: "Start with system", Children: []BehaviorSettingType{}, Position: 4},
	StartInTray:      {Name: "Start in Tray", Children: []BehaviorSettingType{}, Position: 6},
	UploadOnLaunch:   {Name: "Upload Replays on launch", Children: []BehaviorSettingType{}, Position: 1},
	NoUploadOnline:   {Name: "No Upload if player is online", Children: []BehaviorSettingType{}, Position: 3},
	UploadOnRLClose:  {Name: "Upload when RL is closed", Children: []BehaviorSettingType{}, Position: 2},
	UploadOlderFirst: {Name: "Upload older replay first", Children: []BehaviorSettingType{}, Position: 7},
	SendLiveStat:     {Name: "Send Live Stats (with StatsAPI)", Children: []BehaviorSettingType{}, Position: 8},
}

func (bc *BehaviorConfig) GetBoolSettingsMap() map[BehaviorSettingType]*Setting[bool] {
	return map[BehaviorSettingType]*Setting[bool]{
		AutoUpload:       &bc.AutoUpload,
		ExitInTray:       &bc.ExitInTray,
		AutoStart:        &bc.AutoStart,
		StartInTray:      &bc.StartInTray,
		UploadOnLaunch:   &bc.UploadOnLaunch,
		NoUploadOnline:   &bc.NoUploadOnline,
		UploadOnRLClose:  &bc.UploadOnRLClose,
		UploadOlderFirst: &bc.UploadOlderFirst,
		SendLiveStat:     &bc.SendLiveStat,
	}
}
