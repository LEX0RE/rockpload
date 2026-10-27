package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/LEX0RE/rockpload/app/rlgame"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/LEX0RE/rockpload/app/ui"
)

func (a *App) loadReplayInGame(request ui.LoadReplayRequest) (string, error) {
	logger.FuncDebug()

	var command rocket_network.StatsAPICommand
	var err error

	switch request.Mode {
	case ui.LoadReplayFileName:
		demosPath, err := rlgame.CopyToDemos(request.FilePath)
		if err != nil {
			return "", fmt.Errorf("copy to Demos failed: %w", err)
		}

		fileName := filepath.Base(demosPath)
		command, err = rocket_network.NewLoadReplayCommand(strings.TrimSuffix(fileName, filepath.Ext(fileName)), "")
		if err != nil {
			return "", err
		}

	default:
		replayPath := request.FilePath

		if request.Mode == ui.LoadReplayTempPath {
			replayPath, err = rlgame.CopyToTemp(request.FilePath)
			if err != nil {
				return "", fmt.Errorf("copy to temp folder failed: %w", err)
			}
		}

		command, err = rocket_network.NewLoadReplayCommand("", rlgame.GamePath(replayPath))
		if err != nil {
			return "", err
		}
	}

	payload, err := json.Marshal(command)
	if err != nil {
		return "", err
	}

	sent := string(payload)
	if transport := a.statsApi.Transport(); transport != "" {
		sent += " (" + string(transport) + ")"
	}

	return sent, a.statsApi.SendCommand(command)
}
