//go:build android || ios

package manager

import (
	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type RLSupervisor struct {
	LastRLRunningState bool

	EventManager *tools.EventManager
	RLLogInfo    *RLLogInfo
}

func NewRLSupervisor(appConfig *config.AppConfig, accountManager *AccountManager) *RLSupervisor {
	logger.FuncDebug()

	return &RLSupervisor{
		LastRLRunningState: false,
		EventManager:       tools.NewEventManager(),
		RLLogInfo:          &RLLogInfo{},
	}
}

func (rls *RLSupervisor) Supervise() {
	logger.FuncDebug()
}

// Looper Inherit method
func (rls *RLSupervisor) Start() {
	logger.FuncDebug()
}

// Looper Inherit method
func (rls *RLSupervisor) Stop() {
	logger.FuncDebug()
}

// Looper Inherit method
func (rls *RLSupervisor) Toggle(bool) {
	logger.FuncDebug()
}
