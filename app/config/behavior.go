package config

type BehaviorConfig struct {
	AutoUpload        Setting[bool] `json:"auto_upload"`
	ExitInTray        Setting[bool] `json:"exit_in_tray"`
	AutoStart         Setting[bool] `json:"auto_start"`
	StartInTray       Setting[bool] `json:"start_in_tray"`
	UploadOnLaunch    Setting[bool] `json:"upload_on_launch"`
	NoUploadConnected Setting[bool] `json:"no_upload_connected"`

	SelectedAccountId Setting[int] `json:"selected_account_id"`
	SelectedStorageId Setting[int] `json:"selected_storage_id"`
}
