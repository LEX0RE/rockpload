package rocket_network

import (
	"log/slog"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"github.com/dank/rlapi"
)

type PlayerList []*Player

func (pl PlayerList) ToPlayerIDs() []rlapi.PlayerID {
	logger.FuncDebug()

	playerIds := make([]rlapi.PlayerID, 0, len(pl))
	for _, player := range pl {
		if player.PlayerID != nil {
			playerIds = append(playerIds, *player.PlayerID)
		}
	}

	return playerIds
}

type CachedMatchEntry struct {
	MatchGUID            string `json:"match_guid"`
	RecordStartTimestamp int64  `json:"record_start_timestamp"`
	Playlist             int    `json:"playlist"`
	Team0Score           int    `json:"team0_score"`
	Team1Score           int    `json:"team1_score"`
}

type Player struct {
	PlayerName         string             `json:"player_name"`
	Auth               *Auth              `json:"auth"`
	PlayerID           *rlapi.PlayerID    `json:"player_id,omitempty"`
	MatchHistory       []rlapi.MatchEntry `json:"-"`
	CachedMatchHistory []CachedMatchEntry `json:"cached_match_history,omitempty"`
	LastCheckOnline    bool               `json:"-"`

	rpcMu  sync.Mutex `json:"-"`
	authMu sync.Mutex `json:"-"`
}

func NewPlayer(profileId int) *Player {
	logger.FuncDebug()

	p := &Player{PlayerName: "Player", Auth: NewAuth(profileId)}

	return p
}

func (p *Player) SetProfile(profile rlapi.PlayerData) {
	p.PlayerName = profile.PlayerName
}

func (p *Player) Reset() {
	logger.FuncDebug()

	p.PlayerName = "Player"
	p.PlayerID = nil
	p.MatchHistory = nil
	p.CachedMatchHistory = nil

	p.Auth.clearToken()
}

func (p *Player) GetInfo(psyNet *rlapi.PsyNet) (err error) {
	logger.FuncDebug()

	if !p.rpcMu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return
	}
	defer p.rpcMu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, playerId, err := GetRPC(psyNet, p)
	if err != nil {
		return err
	}
	p.PlayerID = playerId

	defer rpc.Close()

	playerData, err := GetProfiles(rpc, []rlapi.PlayerID{*p.PlayerID})
	if err != nil {
		logger.Rlogger.Error("Failed to get profiles:", slog.Any("err", err))
		return err
	}

	p.SetProfile(playerData[0])

	p.MatchHistory, err = GetReplays(rpc)
	if err != nil {
		logger.Rlogger.Error("Failed to get replays:", slog.Any("err", err))
		return err
	}

	sort.Slice(p.MatchHistory, func(i, j int) bool {
		return p.MatchHistory[i].Match.RecordStartTimestamp > p.MatchHistory[j].Match.RecordStartTimestamp
	})

	p.CachedMatchHistory = make([]CachedMatchEntry, 0, len(p.MatchHistory))
	for _, entry := range p.MatchHistory {
		p.CachedMatchHistory = append(p.CachedMatchHistory, CachedMatchEntry{
			MatchGUID:            entry.Match.MatchGUID,
			RecordStartTimestamp: entry.Match.RecordStartTimestamp,
			Playlist:             entry.Match.Playlist,
			Team0Score:           entry.Match.Team0Score,
			Team1Score:           entry.Match.Team1Score,
		})
	}

	return nil
}

func (p *Player) UpdateProfile(psyNet *rlapi.PsyNet) (err error) {
	logger.FuncDebug()

	if !p.rpcMu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return
	}
	defer p.rpcMu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, p.PlayerID, err = GetRPC(psyNet, p)
	if err != nil {
		return err
	}

	defer rpc.Close()

	playerData, err := GetProfiles(rpc, []rlapi.PlayerID{*p.PlayerID})
	if err != nil {
		logger.Rlogger.Error("Failed to get profiles:", slog.Any("err", err))
		return err
	}

	p.SetProfile(playerData[0])
	return nil
}

func (p *Player) GetProfiles(psyNet *rlapi.PsyNet, playersIds []rlapi.PlayerID) ([]rlapi.PlayerData, error) {
	logger.FuncDebug()

	if !p.rpcMu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return nil, nil
	}
	defer p.rpcMu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, _, err := GetRPC(psyNet, p)
	if err != nil {
		return nil, err
	}

	defer rpc.Close()
	profiles, err := GetProfiles(rpc, playersIds)
	if err != nil {
		logger.Rlogger.Error("Failed to get online statuses:", slog.Any("err", err))
		return nil, err
	}

	return profiles, nil
}

func (p *Player) GetRanks(psyNet *rlapi.PsyNet, playersIds []rlapi.PlayerID) ([]rlapi.PlayerWithSkills, error) {
	logger.FuncDebug()

	if !p.rpcMu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return nil, nil
	}
	defer p.rpcMu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, _, err := GetRPC(psyNet, p)
	if err != nil {
		return nil, err
	}

	defer rpc.Close()

	platersSkills, err := GetPlayersSkills(rpc, playersIds)
	if err != nil {
		logger.Rlogger.Error("Failed to get online statuses:", slog.Any("err", err))
		return nil, err
	}

	return platersSkills, nil
}

func (p *Player) IsAuthenticated() bool {
	logger.FuncDebug()

	p.authMu.Lock()
	defer p.authMu.Unlock()

	if p.Auth != nil && p.Auth.isAuthenticated() {
		return true
	}

	return false
}

func (p *Player) Equal(otherPlayer *Player) bool {
	logger.FuncDebug()

	if otherPlayer == nil {
		return false
	}

	return p.EqualID(otherPlayer.PlayerID)
}

func (p *Player) EqualID(otherPlayerID *rlapi.PlayerID) bool {
	logger.FuncDebug()

	if otherPlayerID == nil {
		return false
	}

	return p.EqualStringID(otherPlayerID.String())
}

func (p *Player) EqualStringID(otherPlayerStringID string) bool {
	logger.FuncDebug()

	if p.PlayerID == nil {
		return false
	}

	return strings.EqualFold(p.PlayerID.String(), otherPlayerStringID)
}

func (p *Player) GetMatchHistoryIndex(matchGUID string) int {
	logger.FuncDebug()

	return slices.IndexFunc(p.MatchHistory, func(m rlapi.MatchEntry) bool {
		return m.Match.MatchGUID == matchGUID
	})
}
