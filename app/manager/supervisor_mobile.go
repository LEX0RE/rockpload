//go:build android || ios

package manager

import (
	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	EVENT_ON_RL_DETECTED        tools.EventType = "rl_detected"
	EVENT_ON_RL_PLAYER_DETECTED tools.EventType = "rl_player_detected"
	EVENT_ON_RL_CLOSED          tools.EventType = "rl_closed"
)

type RLSupervisor struct {
	LastRLRunningState bool

	EventManager *tools.EventManager
	RLLogInfo    *RLLogInfo
}

type RLVersionInfo struct {
	GameVersion string
	FeatureSet  string
}

type RLLogInfo struct {
	VersionInfo  *RLVersionInfo
	AccountFound *rocket_network.Account
}

func NewRLSupervisor(appConfig *config.AppConfig, accountManager *AccountManager) *RLSupervisor {
	logger.FuncDebug()

	return &RLSupervisor{
		LastRLRunningState: false,
		EventManager:       tools.NewEventManager(),
		RLLogInfo:          &RLLogInfo{},
	}
}

func (rls *RLSupervisor) Start() {}

func (rls *RLSupervisor) Stop() {}

func (rls *RLSupervisor) Toggle(bool) {}
