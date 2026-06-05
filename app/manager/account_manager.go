package manager

import (
	"cmp"
	"log/slog"
	"slices"
	"time"

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

type MatchPlaylistRanking struct {
	Data       map[rlapi.PlayerID]*rlapi.Skill `json:"Data"`
	LastUpdate time.Time                       `json:"LastUpdate"`
}

type MatchRanking struct {
	data       map[rlapi.PlayerID][]rlapi.Skill
	lastUpdate time.Time
}

type AccountManager struct {
	appConfig    *config.AppConfig
	skillMatches map[string]*MatchRanking

	EventManager *tools.EventManager
}

func NewAccountManager(appConfig *config.AppConfig) *AccountManager {
	logger.FuncDebug()

	am := &AccountManager{appConfig: appConfig, EventManager: tools.NewEventManager(), skillMatches: make(map[string]*MatchRanking)}
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

		ac.Player.GetInfo()
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
			ac.Player.UpdateProfile()
		}
	}
}

func (am *AccountManager) AddPlayersSkills(liveStats *rocket_network.LiveStats) {
	logger.FuncDebug()

	logger.Rlogger.Debug("Getting players skills", slog.Any("Player", liveStats.State.Players))

	// We won't allow querying skills if we don't have an unused account as it would disconnect the account that is currently playing
	unusedAccount := am.GetUnused()
	if unusedAccount == nil || unusedAccount.Player == nil {
		logger.Rlogger.Debug("Invalid Unused Account for getting Players Skills", slog.Any("Unused Account", unusedAccount))
		return
	}

	logger.Rlogger.Debug("Account is being used to get Players Skills", slog.Any("Account", unusedAccount.AccountName()))

	playerIDList := []rlapi.PlayerID{}
	for _, playerState := range liveStats.State.Players {
		if playerState.PrimaryId == "" {
			continue
		}

		if playerSkills, ok := liveStats.Skills[rlapi.PlayerID(playerState.PrimaryId)]; !ok || len(playerSkills) == 0 {
			playerIDList = append(playerIDList, rlapi.PlayerID(playerState.PrimaryId))
		}
	}

	ranks := unusedAccount.Player.GetRanks(playerIDList)
	if liveStats.State.MatchGuid != "" {
		if _, ok := am.skillMatches[liveStats.State.MatchGuid]; !ok {
			am.skillMatches[liveStats.State.MatchGuid] = &MatchRanking{
				data:       make(map[rlapi.PlayerID][]rlapi.Skill),
				lastUpdate: time.Now(),
			}
		}

		for _, playerRank := range ranks {
			am.skillMatches[liveStats.State.MatchGuid].data[playerRank.PlayerID] = playerRank.Skills
			am.skillMatches[liveStats.State.MatchGuid].lastUpdate = time.Now()

			liveStats.Skills[playerRank.PlayerID] = playerRank.Skills

		}
	}
}

func (am *AccountManager) GetSkillMatch(match rlapi.MatchEntry) *MatchPlaylistRanking {
	logger.FuncDebug()

	originalRanking, ok := am.skillMatches[match.Match.MatchGUID]
	if !ok {
		return nil
	}

	filteredRanking := &MatchPlaylistRanking{
		Data:       make(map[rlapi.PlayerID]*rlapi.Skill),
		LastUpdate: originalRanking.lastUpdate,
	}

	for playerID, skills := range originalRanking.data {
		for _, skill := range skills {
			if skill.Playlist == match.Match.Playlist {
				filteredRanking.Data[playerID] = &skill
				break
			}
		}
	}

	return filteredRanking
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

	profiles := account.Player.GetProfiles(rocket_network.PlayerList(playerList).ToPlayerIDs())

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
