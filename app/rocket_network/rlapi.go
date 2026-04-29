package rocket_network

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"github.com/dank/rlapi"
)

func GetRPC(ra *Auth)  (*rlapi.PsyNetRPC, *rlapi.PlayerID, error) {
	logger.FuncDebug()
	if ra.Auth == nil {
		logger.Rlogger.Error("No valid authentication token found. Please retrieve a new token.")
		return nil, nil, fmt.Errorf("no valid authentication token")
	}
	
	var authToken *rlapi.EOSTokenResponse

	code, err := ra.EGS.GetExchangeCode(ra.Auth.AccessToken)
	if err != nil {
		logger.Rlogger.Error("Failed to get exchange code:", slog.Any("err", err))
		return nil, nil, err
	}

	authToken, err = ra.EGS.ExchangeEOSToken(code)
	if err != nil {
		logger.Rlogger.Error("Failed to exchange EOS token:", slog.Any("err", err))
		return nil, nil, err
	}

	psyNet := rlapi.NewPsyNet()
	rpc, err := psyNet.AuthPlayer(authToken.AccessToken, authToken.AccountID, ra.Auth.DisplayName)
	if err != nil {
		logger.Rlogger.Error("Failed to authenticate player:", slog.Any("err", err))
		return nil, nil, err
	}

	pid := rlapi.NewPlayerID(rlapi.PlatformEpic, authToken.AccountID)
	return rpc, &pid, nil
}

func GetReplays(rpc *rlapi.PsyNetRPC) (matchHistory []rlapi.MatchEntry, err error) {
	logger.FuncDebug()
	apiCtx, apiCancel := context.WithTimeout(context.Background(), 15 * time.Second)
	defer apiCancel()

	matchHistory, err = rpc.GetMatchHistory(apiCtx)
	if err != nil {
		logger.Rlogger.Error("Failed to get match history", slog.Any("error", err))
		return nil, err
	}

	return matchHistory, nil
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

	logger.Rlogger.Debug("Retrieved standard shops with ", slog.Any("shops", shops))

	return shops, nil
}


