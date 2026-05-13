package rocket_network

import (
	"log/slog"
	"strconv"
	"sync"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"github.com/dank/rlapi"
)

type Player struct {
	PlayerName   string             `json:"player_name"`
	Auth         *Auth              `json:"auth"`
	PlayerID     *rlapi.PlayerID    `json:"player_id,omitempty"`
	MatchHistory []rlapi.MatchEntry `json:"-"`
	mu           sync.Mutex         `json:"-"`
}

func NewPlayer(profileId int) *Player {
	logger.FuncDebug()

	p := &Player{PlayerName: "Player (ID: " + strconv.Itoa(profileId) + ")", Auth: NewAuth(profileId)}

	return p
}

func (p *Player) Connect() {
	logger.FuncDebug()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.Auth.Authenticate()

	if !p.Auth.IsAuthenticated() {
		p.Reset()
	}
}

func (p *Player) SetProfile(profile rlapi.PlayerData) {
	p.PlayerName = profile.PlayerName + " (ID: " + strconv.Itoa(p.Auth.ProfileId) + ")"
}

func (p *Player) Reset() {
	logger.FuncDebug()

	p.PlayerName = "Player (ID: " + strconv.Itoa(p.Auth.ProfileId) + ")"
	p.Auth.ClearToken()
	p.PlayerID = nil
	p.MatchHistory = nil
}

func (p *Player) GetInfo() (err error) {
	logger.FuncDebug()

	if !p.mu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return
	}
	defer p.mu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, playerId, err := GetRPC(p.Auth)
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

	return nil
}

func (p *Player) UpdateProfile() (err error) {
	logger.FuncDebug()

	if !p.mu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return
	}
	defer p.mu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, p.PlayerID, err = GetRPC(p.Auth)
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

func (p *Player) GetProfiles(players []*Player) []rlapi.PlayerData {
	logger.FuncDebug()

	if !p.mu.TryLock() {
		logger.Rlogger.Debug("Duplicate request at the same time, skipping")
		return nil
	}
	defer p.mu.Unlock()

	var rpc *rlapi.PsyNetRPC
	rpc, _, err := GetRPC(p.Auth)
	if err != nil {
		return nil
	}

	defer rpc.Close()

	playerIds := make([]rlapi.PlayerID, 0, len(players))
	for _, player := range players {
		if player.PlayerID != nil {
			playerIds = append(playerIds, *player.PlayerID)
		}
	}

	profiles, err := GetProfiles(rpc, playerIds)
	if err != nil {
		logger.Rlogger.Error("Failed to get online statuses:", slog.Any("err", err))
		return nil
	}

	return profiles
}
