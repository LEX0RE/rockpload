package manager

import (
	"log/slog"

	"github.com/LEX0RE/rockpload/app/config"
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

	am.AuthenticateAll()

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

	am.EventManager.Notify(EVENT_UNUSED_ACCOUNT, am.GetUnused().Id())
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
		deletedId := temp[accountId].Id()

		delete(temp, accountId)
		am.appConfig.AccountSettings.Set(temp)

		am.GetSelected()

		am.EventManager.Notify(EVENT_DELETE_ACCOUNT, deletedId)
	}
}

func (am *AccountManager) AuthenticateAll() {
	logger.FuncDebug()

	for _, ac := range am.appConfig.AccountSettings.Get() {
		ac.Player.Auth.Authenticate()
		if err := ac.Player.Auth.Authenticate(); err != nil {
			logger.Rlogger.Error("Failed to authenticate:", slog.Any("err", err))
		}
	}
}

func (am *AccountManager) RefreshInfo() {
	logger.FuncDebug()

	playerIds := []rlapi.PlayerID{}
	uploadActiveMap := make(map[int]bool)
	playerIdMap := make(map[int]rlapi.PlayerID)

	for _, ac := range am.GetActives() {
		playerIds = append(playerIds, *ac.Player.PlayerID)
		uploadActiveMap[ac.Id()] = true
		playerIdMap[ac.Id()] = *ac.Player.PlayerID
	}

	unusedAccount := am.GetUnused()
	if unusedAccount != nil && am.appConfig.BehaviorConfig.NoUploadConnected.Get() {
		onlineStatus := unusedAccount.Player.CheckOnline(playerIds)

		for pid := range uploadActiveMap {
			if onlineStatus[playerIdMap[pid]] {
				uploadActiveMap[pid] = false
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
