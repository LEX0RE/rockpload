package rocket_network

import (
	"log/slog"
	"sync"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"github.com/dank/rlapi"
)

type Player struct {
	Auth 		 	*Auth
	PlayerID     	*rlapi.PlayerID
	MatchHistory 	[]rlapi.MatchEntry
	mu 				sync.Mutex
}

func NewPlayer(auth *Auth) *Player {
	logger.FuncDebug()
	return &Player{Auth: auth}
}

func (p *Player) GetInfo() (err error) {
	logger.FuncDebug()

	if (!p.mu.TryLock()) {
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

	p.MatchHistory, err = GetReplays(rpc)
	if err != nil {
		logger.Rlogger.Error("Failed to get replays:", slog.Any("err", err))
		return err
	}

	return nil
}