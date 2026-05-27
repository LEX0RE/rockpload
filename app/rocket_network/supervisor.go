package rocket_network

import (
	"strings"
	"time"

	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/LEX0RE/rockpload/app/tools/rtime"

	ps "github.com/mitchellh/go-ps"
)

const (
	EventOnRLDetected = "rl_detected"
	EventOnRLClosed   = "rl_closed"

	PROCESS_SUPERVISOR_CHECH_TIME = time.Second * 10
)

type RLSupervisor struct {
	*rtime.Looper

	LastRLRunningState bool

	EventManager *tools.EventManager
}

func NewRLSupervisor() *RLSupervisor {
	logger.FuncDebug()

	rls := &RLSupervisor{LastRLRunningState: false, EventManager: tools.NewEventManager()}
	rls.Looper = rtime.NewLooper(PROCESS_SUPERVISOR_CHECH_TIME, rls.Supervise, nil, nil, 0, 0)

	return rls
}

func (rls *RLSupervisor) isRLRunning() bool {
	logger.FuncDebug()

	processes, _ := ps.Processes()

	for _, p := range processes {
		name := strings.ToLower(p.Executable())

		if name == "rocketleague.exe" || name == "rocket league.exe" {
			return true
		}
	}
	return false
}

func (rls *RLSupervisor) Supervise() {
	logger.FuncDebug()

	running := rls.isRLRunning()

	if running && !rls.LastRLRunningState {
		rls.EventManager.Notify(EventOnRLDetected, nil)
	}

	if !running && rls.LastRLRunningState {
		rls.EventManager.Notify(EventOnRLClosed, nil)
	}

	rls.LastRLRunningState = running
}
