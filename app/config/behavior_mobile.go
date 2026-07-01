//go:build android || ios

package config

func init() {
	delete(BehaviorSettingVisualMapping, AutoStart)
	delete(BehaviorSettingVisualMapping, ExitInTray)
	delete(BehaviorSettingVisualMapping, StartInTray)
	delete(BehaviorSettingVisualMapping, UploadOnRLClose)
	delete(BehaviorSettingVisualMapping, SendLiveStat)
}
