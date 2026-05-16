package manager

import (
	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/dank/rlapi"
)

const (
	EVENT_SELECT_ACCOUNT = "select_account"
	EVENT_UNUSED_ACCOUNT = "unused_account"
	EVENT_ADD_ACCOUNT    = "add_account"
	EVENT_DELETE_ACCOUNT = "delete_account"
)

type AccountManager struct {
	appConfig *config.AppConfig

	EventManager *tools.EventManager
}

func NewAccountManager(appConfig *config.AppConfig) *AccountManager {
	logger.FuncDebug()

	am := &AccountManager{appConfig: appConfig, EventManager: tools.NewEventManager()}

	am.ConnectAll()

	return am
}

func (am *AccountManager) GetSelected() *config.AccountConfig {
	logger.FuncDebug()

	if ac, ok := am.appConfig.AccountSettings.Get()[am.appConfig.BehaviorConfig.SelectedAccountId.Get()]; ok {
		return ac
	} else {
		am.appConfig.BehaviorConfig.SelectedAccountId.Set(0)
		am.EventManager.Notify(EVENT_SELECT_ACCOUNT, 0)

		if ac, ok := am.appConfig.AccountSettings.Get()[0]; ok {
			return ac
		} else {
			return am.Add()
		}
	}
}

func (am *AccountManager) GetUnused() *config.AccountConfig {
	logger.FuncDebug()

	for _, ac := range am.appConfig.AccountSettings.Get() {
		if ac.IsUnused {
			return ac
		}
	}

	return nil
}

func (am *AccountManager) SetSelected(accountId int) {
	logger.FuncDebug()

	if _, ok := am.appConfig.AccountSettings.Get()[accountId]; ok {
		am.appConfig.BehaviorConfig.SelectedAccountId.Set(accountId)
	} else {
		am.appConfig.BehaviorConfig.SelectedAccountId.Set(0)
	}

	am.EventManager.Notify(EVENT_SELECT_ACCOUNT, am.GetSelected().Id())
	am.appConfig.Save()
}

func (am *AccountManager) SetUnused(accountId int) {
	logger.FuncDebug()

	for _, ac := range am.appConfig.AccountSettings.Get() {
		ac.IsUnused = false
	}

	if _, ok := am.appConfig.AccountSettings.Get()[accountId]; ok {
		am.appConfig.AccountSettings.Get()[accountId].IsUnused = true
	}

	nextUnused := am.GetUnused()
	nextId := -1
	if nextUnused != nil {
		nextId = nextUnused.Id()
	}

	am.EventManager.Notify(EVENT_UNUSED_ACCOUNT, nextId)
	am.appConfig.Save()
}

func (am *AccountManager) GetActives() []*config.AccountConfig {
	logger.FuncDebug()

	actives := make([]*config.AccountConfig, 0, len(am.appConfig.AccountSettings.Get()))
	for _, ac := range am.appConfig.AccountSettings.Get() {
		if ac.IsUnused {
			continue
		}

		actives = append(actives, ac)
	}

	return actives
}

func (am *AccountManager) Add() *config.AccountConfig {
	logger.FuncDebug()

	for i := 0; ; i++ {
		if _, ok := am.appConfig.AccountSettings.Get()[i]; !ok {
			temp := am.appConfig.AccountSettings.Get()
			temp[i] = config.NewAccountConfig(i)
			am.appConfig.AccountSettings.Set(temp)

			am.EventManager.Notify(EVENT_ADD_ACCOUNT, temp[i].Id())

			return temp[i]
		}
	}
}

func (am *AccountManager) Delete(accountId int) {
	logger.FuncDebug()

	if len(am.appConfig.AccountSettings.Get()) <= 1 {
		return
	}

	if _, ok := am.appConfig.AccountSettings.Get()[accountId]; ok {
		temp := am.appConfig.AccountSettings.Get()
		accountDeleted := temp[accountId]
		accountDeleted.Player.Reset()

		deletedId := accountDeleted.Id()

		delete(temp, accountId)
		am.appConfig.AccountSettings.Set(temp)

		am.GetSelected()

		am.EventManager.Notify(EVENT_DELETE_ACCOUNT, deletedId)
	}
}

func (am *AccountManager) ConnectAll() {
	logger.FuncDebug()

	for _, ac := range am.appConfig.AccountSettings.Get() {
		ac.Player.Connect()
	}
}

func (am *AccountManager) RefreshInfo() {
	logger.FuncDebug()

	playerList := []*rocket_network.Player{}
	uploadActiveMap := make(map[int]bool)
	playersMap := make(map[int]*rocket_network.Player)

	for _, ac := range am.GetActives() {
		playerList = append(playerList, ac.Player)
		uploadActiveMap[ac.Id()] = true
		playersMap[ac.Id()] = ac.Player
	}

	unusedAccount := am.GetUnused()
	if unusedAccount != nil && am.appConfig.BehaviorConfig.NoUploadConnected.Get() {
		profiles := unusedAccount.Player.GetProfiles(playerList)

		onlineStatus := make(map[rlapi.PlayerID]bool)
		for _, player := range playerList {
			if player.PlayerID == nil {
				continue
			}

			for _, profile := range profiles {
				if profile.PlayerID == player.PlayerID.String() {
					player.SetProfile(profile)

					onlineStatus[*player.PlayerID] = profile.PresenceState != "Online"
					break
				}
			}
		}

		for pid := range uploadActiveMap {
			if playersMap[pid].PlayerID != nil {
				if value, ok := onlineStatus[*playersMap[pid].PlayerID]; ok && !value {
					uploadActiveMap[pid] = false
				}
			}
		}
	}

	for _, ac := range am.appConfig.AccountSettings.Get() {
		if !uploadActiveMap[ac.Id()] {
			continue
		}

		ac.Player.GetInfo()
	}
}

func (am *AccountManager) RefreshProfile() {
	logger.FuncDebug()

	playerList := []*rocket_network.Player{}
	for _, ac := range am.GetActives() {
		playerList = append(playerList, ac.Player)
	}

	unusedAccount := am.GetUnused()
	if am.appConfig.BehaviorConfig.NoUploadConnected.Get() || unusedAccount != nil {
		if unusedAccount != nil {
			profiles := unusedAccount.Player.GetProfiles(playerList)

			for _, player := range playerList {
				if player.PlayerID == nil {
					continue
				}

				for _, profile := range profiles {
					if profile.PlayerID == player.PlayerID.String() {
						player.SetProfile(profile)
						break
					}
				}
			}
		}
	} else {
		for _, ac := range am.appConfig.AccountSettings.Get() {
			ac.Player.UpdateProfile()
		}
	}
}
