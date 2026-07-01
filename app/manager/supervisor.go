package manager

import (
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools"
)

const (
	EVENT_ON_RL_DETECTED        tools.EventType = "rl_detected"
	EVENT_ON_RL_PLAYER_DETECTED tools.EventType = "rl_player_detected"
	EVENT_ON_RL_CLOSED          tools.EventType = "rl_closed"
)

type RLVersionInfo struct {
	GameVersion string
	FeatureSet  string
}

type RLLogInfo struct {
	VersionInfo  *RLVersionInfo
	AccountFound *rocket_network.Account
}
