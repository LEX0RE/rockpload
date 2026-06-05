package rocket_network

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"github.com/dank/rlapi"
)

func GetRPC(p *Player) (*rlapi.PsyNetRPC, *rlapi.PlayerID, error) {
	logger.FuncDebug()

	logger.Rlogger.Info("Log in to player account", slog.Any("PlayerName", p.PlayerName), slog.Any("PlayerID", p.PlayerID), slog.Any("Online", p.LastCheckOnline))

	if !p.IsAuthenticated() {
		logger.Rlogger.Error("No valid authentication token found. Please retrieve a new token.")
		p.Reset()

		return nil, nil, fmt.Errorf("no valid authentication token")
	}

	psyNet := rlapi.NewPsyNet()
	eosToken := p.Auth.getValidToken()
	if eosToken == nil {
		return nil, nil, fmt.Errorf("failed to get any valid token")
	}

	rpc, err := psyNet.AuthPlayer(eosToken.AccessToken, eosToken.AccountID, "")
	if err != nil {
		logger.Rlogger.Error("Failed to authenticate player:", slog.Any("err", err))
		return nil, nil, err
	}

	pid := rlapi.NewPlayerID(rlapi.PlatformEpic, eosToken.AccountID)
	return rpc, &pid, nil
}

func GetReplays(rpc *rlapi.PsyNetRPC) (matchHistory []rlapi.MatchEntry, err error) {
	logger.FuncDebug()

	apiCtx, apiCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer apiCancel()

	matchHistory, err = rpc.GetMatchHistory(apiCtx)
	if err != nil {
		logger.Rlogger.Error("Failed to get match history", slog.Any("error", err))
		return nil, err
	}

	return matchHistory, nil
}

func GetProfiles(rpc *rlapi.PsyNetRPC, playerIDs []rlapi.PlayerID) (profiles []rlapi.PlayerData, err error) {
	logger.FuncDebug()

	apiCtx, apiCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer apiCancel()

	profiles, err = rpc.GetProfiles(apiCtx, playerIDs)
	if err != nil {
		logger.Rlogger.Error("Failed to get profiles:", slog.Any("err", err))
		return nil, err
	}

	return profiles, nil
}

func GetShops(rpc *rlapi.PsyNetRPC) (shops *rlapi.GetStandardShopsResponse, err error) {
	logger.FuncDebug()

	apiCtx, apiCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer apiCancel()

	shops, err = rpc.GetStandardShops(apiCtx)
	if err != nil {
		logger.Rlogger.Error("Failed to get shops", slog.Any("error", err))
		return nil, err
	}

	return shops, nil
}

func GetPlayersSkills(rpc *rlapi.PsyNetRPC, playerIDs []rlapi.PlayerID) (playersSkills []rlapi.PlayerWithSkills, err error) {
	logger.FuncDebug()

	apiCtx, apiCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer apiCancel()

	playersSkills, err = rpc.GetPlayersSkills(apiCtx, playerIDs)
	if err != nil {
		logger.Rlogger.Error("Failed to get shops", slog.Any("error", err))
		return nil, err
	}

	return playersSkills, nil
}
