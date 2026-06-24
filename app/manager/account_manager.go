package manager

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/dank/rlapi"
)

const (
	EVENT_SELECT_ACCOUNT tools.EventType = "select_account"
	EVENT_UNUSED_ACCOUNT tools.EventType = "unused_account"
	EVENT_ADD_ACCOUNT    tools.EventType = "add_account"
	EVENT_DELETE_ACCOUNT tools.EventType = "delete_account"
)

type AccountManager struct {
	appConfig  *config.AppConfig
	psyNetList []*rlapi.PsyNet

	EventManager *tools.EventManager
}

func NewAccountManager(appConfig *config.AppConfig) *AccountManager {
	logger.FuncDebug()

	am := &AccountManager{
		appConfig:    appConfig,
		EventManager: tools.NewEventManager(),
		psyNetList:   []*rlapi.PsyNet{rlapi.NewPsyNet()},
	}
	am.RefreshProfile()

	return am
}

func (am *AccountManager) GetByPlayerName(playerName string) *rocket_network.Account {
	logger.FuncDebug()

	allAccount := am.GetAll()
	for _, a := range allAccount {
		if a != nil && a.Player != nil && a.Player.PlayerName == playerName {
			return a
		}
	}

	return nil
}

func (am *AccountManager) GetByPlayerID(playerID string) *rocket_network.Account {
	logger.FuncDebug()

	allAccount := am.GetAll()
	for _, a := range allAccount {
		if a == nil || a.Player == nil || a.Player.PlayerID == nil {
			continue
		}

		if a.Player.EqualStringID(playerID) {
			return a
		}
	}

	return nil
}

func (am *AccountManager) GetSelected() *rocket_network.Account {
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

func (am *AccountManager) GetUnused() *rocket_network.Account {
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
		if ac.IsUnused {
			ac.Player.LastCheckOnline = false
		}

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

func (am *AccountManager) GetActives() []*rocket_network.Account {
	logger.FuncDebug()

	actives := make([]*rocket_network.Account, 0, len(am.appConfig.AccountSettings.Get()))
	for _, ac := range am.appConfig.AccountSettings.Get() {
		if ac.IsUnused {
			continue
		}

		actives = append(actives, ac)
	}

	return actives
}

func (am *AccountManager) GetConnectedAccount() []*rocket_network.Account {
	logger.FuncDebug()

	connected := make([]*rocket_network.Account, 0, len(am.appConfig.AccountSettings.Get()))
	for _, ac := range am.appConfig.AccountSettings.Get() {
		if ac.IsConnected() {
			connected = append(connected, ac)
		}
	}

	return connected
}

func (am *AccountManager) GetAll() []*rocket_network.Account {
	logger.FuncDebug()

	all := make([]*rocket_network.Account, 0, len(am.appConfig.AccountSettings.Get()))
	for _, ac := range am.appConfig.AccountSettings.Get() {
		all = append(all, ac)
	}

	slices.SortFunc(all, func(a, b *rocket_network.Account) int {
		return cmp.Compare(a.Id(), b.Id())
	})

	return all
}

func (am *AccountManager) Add() *rocket_network.Account {
	logger.FuncDebug()

	for i := 0; ; i++ {
		if _, ok := am.appConfig.AccountSettings.Get()[i]; !ok {
			temp := am.appConfig.AccountSettings.Get()
			temp[i] = rocket_network.NewAccount(i)
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

func (am *AccountManager) RefreshInfo() {
	logger.FuncDebug()

	uploadActiveMap := am.uploadStatusMap()
	for _, ac := range am.appConfig.AccountSettings.Get() {
		if !uploadActiveMap[ac.Id()] {
			continue
		}

		ac.Player.GetInfo(am.currentPsyNet())
	}
}

func (am *AccountManager) RefreshProfile() {
	logger.FuncDebug()

	unusedAccount := am.GetUnused()
	if unusedAccount != nil || am.appConfig.BehaviorConfig.NoUploadOnline.Get() {
		if unusedAccount != nil {
			am.onlineStatusFromAccount(unusedAccount)
		}
	} else {
		for _, ac := range am.appConfig.AccountSettings.Get() {
			ac.Player.UpdateProfile(am.currentPsyNet())
		}
	}
}

func (am *AccountManager) AddPsyNetVersion(rlVersionInfo RLVersionInfo) {
	logger.FuncDebug()

	if rlVersionInfo.GameVersion == "" || rlVersionInfo.FeatureSet == "" {
		return
	}

	duplicate := false
	for _, psyNet := range am.psyNetList {
		gameVersion, featureSet := psyNet.GetVersion()

		if gameVersion == rlVersionInfo.GameVersion && featureSet == rlVersionInfo.FeatureSet {
			duplicate = true
			break
		}
	}

	if !duplicate {
		newPsyNet := rlapi.NewPsyNet()
		newPsyNet.SetVersion(rlVersionInfo.GameVersion, rlVersionInfo.FeatureSet)
		am.psyNetList = append(am.psyNetList, newPsyNet)
	}
}

func (am *AccountManager) uploadStatusMap() map[int]bool {
	logger.FuncDebug()

	uploadActiveMap := make(map[int]bool)
	accountMap := make(map[int]*rocket_network.Account)

	for _, ac := range am.GetActives() {
		uploadActiveMap[ac.Id()] = true
		accountMap[ac.Id()] = ac
	}

	if am.appConfig.BehaviorConfig.NoUploadOnline.Get() {
		unusedAccount := am.GetUnused()
		if unusedAccount != nil {
			am.onlineStatusFromAccount(unusedAccount)
		}

		for pid := range uploadActiveMap {
			currentAccount := accountMap[pid]

			if currentAccount.Player.PlayerID != nil && currentAccount.Player.LastCheckOnline {
				logger.Rlogger.Debug("Account could be connected, upload is skipped", slog.Any("Account", currentAccount.AccountName()))
				uploadActiveMap[pid] = false
			}
		}
	}

	return uploadActiveMap
}

func (am *AccountManager) onlineStatusFromAccount(account *rocket_network.Account) map[rlapi.PlayerID]bool {
	logger.FuncDebug()

	onlineStatus := make(map[rlapi.PlayerID]bool)
	if account == nil || account.Player == nil {
		logger.Rlogger.Debug("Invalid Account for getting Online Status", slog.Any("Account", account))
		return onlineStatus
	}

	logger.Rlogger.Debug("Account is being used to get Online Status", slog.Any("Account", account.AccountName()))

	playerList := []*rocket_network.Player{}
	for _, ac := range am.GetAll() {
		playerList = append(playerList, ac.Player)
	}

	getProfile := func(psyNet *rlapi.PsyNet) ([]rlapi.PlayerData, error) {
		return account.Player.GetProfiles(psyNet, rocket_network.PlayerList(playerList).ToPlayerIDs())
	}

	profiles, _ := tryRotatingPsyNet(am, getProfile)
	for _, player := range playerList {
		if player.PlayerID == nil {
			continue
		}

		for _, profile := range profiles {
			if player.EqualStringID(profile.PlayerID) {
				player.SetProfile(profile)
				player.LastCheckOnline = profile.PresenceState == "Online"
				onlineStatus[*player.PlayerID] = player.LastCheckOnline
				break
			}
		}
	}

	return onlineStatus
}

func (am *AccountManager) currentPsyNet() *rlapi.PsyNet {
	logger.FuncDebug()

	return am.psyNetList[0]
}

func tryRotatingPsyNet[T any](am *AccountManager, request func(*rlapi.PsyNet) (T, error)) (T, error) {
	logger.FuncDebug()

	defaultGameVersion, defaultFeatureSet := am.psyNetList[0].GetVersion()
	var result T
	var err error

	for {
		currPsyNet := am.psyNetList[0]

		result, err = request(currPsyNet)
		if err == nil {
			return result, nil
		}

		if len(am.psyNetList) < 2 {
			break
		}

		logger.Rlogger.Info("PsyNet version info seems not good, trying with another one")
		current := am.psyNetList[0]
		am.psyNetList = append(am.psyNetList[1:], current)

		tempGameVersion, tempFeatureSet := am.psyNetList[0].GetVersion()
		if tempGameVersion == defaultGameVersion && tempFeatureSet == defaultFeatureSet {
			break
		}
	}

	logger.Rlogger.Info("Error with current PsyNet and no other PsyNet version found")
	return result, err
}
