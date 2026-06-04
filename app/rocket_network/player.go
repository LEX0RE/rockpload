package rocket_network

import (
	"log/slog"
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

type Player struct {
	PlayerName      string             `json:"player_name"`
	Auth            *Auth              `json:"auth"`
	PlayerID        *rlapi.PlayerID    `json:"player_id,omitempty"`
	MatchHistory    []rlapi.MatchEntry `json:"-"`
	LastCheckOnline bool               `json:"-"`

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

	p.Auth.clearToken()
}

func (p *Player) GetInfo() (err error) {
	logger.FuncDebug()

	if !p.rpcMu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return
	}
	defer p.rpcMu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, playerId, err := GetRPC(p)
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

	return nil
}

func (p *Player) UpdateProfile() (err error) {
	logger.FuncDebug()

	if !p.rpcMu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return
	}
	defer p.rpcMu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, p.PlayerID, err = GetRPC(p)
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

func (p *Player) GetProfiles(playersIds []rlapi.PlayerID) []rlapi.PlayerData {
	logger.FuncDebug()

	if !p.rpcMu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return nil
	}
	defer p.rpcMu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, _, err := GetRPC(p)
	if err != nil {
		return nil
	}

	defer rpc.Close()
	profiles, err := GetProfiles(rpc, playersIds)
	if err != nil {
		logger.Rlogger.Error("Failed to get online statuses:", slog.Any("err", err))
		return nil
	}

	return profiles
}

func (p *Player) GetRanks(playersIds []rlapi.PlayerID) []rlapi.PlayerWithSkills {
	logger.FuncDebug()

	if !p.rpcMu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return nil
	}
	defer p.rpcMu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, _, err := GetRPC(p)
	if err != nil {
		return nil
	}

	defer rpc.Close()

	platersSkills, err := GetPlayersSkills(rpc, playersIds)
	if err != nil {
		logger.Rlogger.Error("Failed to get online statuses:", slog.Any("err", err))
		return nil
	}

	return platersSkills
}

func (p *Player) IsAuthenticated() bool {
	logger.FuncDebug()

	p.authMu.Lock()
	defer p.authMu.Unlock()

	if p.Auth != nil && p.Auth.isAuthenticate() {
		return true
	}

	p.Reset()
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
