package rocket_network

import (
	"log/slog"
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

	p := &Player{PlayerName: "Unknown Player Name"}
	playerAuth, err := NewAuth(profileId)
	if err != nil {
		logger.Rlogger.Error("Unable to create Auth:", slog.Any("err", err))
	}
	p.Auth = playerAuth

	return p
}

func (p *Player) GetBaseInfo() (err error) {
	logger.FuncDebug()

	if !p.mu.TryLock() {
		logger.Rlogger.Debug("Duplicate GetInfo at the same time, skipping")
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
	} else {
		p.PlayerName = playerData[0].PlayerName
	}

	return nil
}

func (p *Player) GetInfo() (err error) {
	logger.FuncDebug()

	if !p.mu.TryLock() {
		logger.Rlogger.Debug("Duplicate GetInfo at the same time, skipping")
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
	} else {
		p.PlayerName = playerData[0].PlayerName
	}

	p.MatchHistory, err = GetReplays(rpc)
	if err != nil {
		logger.Rlogger.Error("Failed to get replays:", slog.Any("err", err))
		return err
	}

	return nil
}

func (p *Player) Reset() {
	logger.FuncDebug()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.PlayerName = "Unknown Player Name"
	p.Auth.ClearToken()
	p.PlayerID = nil
	p.MatchHistory = nil
}
