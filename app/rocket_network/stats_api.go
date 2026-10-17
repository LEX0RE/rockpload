package rocket_network

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/rlgame"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/dank/rlapi"
)

const (
	EventFirstUpdateState tools.EventType = "first_update_state"
	EventOvertimeStarted  tools.EventType = "overtime_started"
)

type LiveStats struct {
	State  map[string]any
	Skills map[rlapi.PlayerID][]rlapi.Skill
}

func (l *LiveStats) MatchGuid() string {
	guid, _ := l.State["MatchGuid"].(string)
	return guid
}

func (l *LiveStats) TimeSeconds() (int, bool) {
	game, _ := l.State["Game"].(map[string]any)
	seconds, ok := game["TimeSeconds"].(float64)
	return int(seconds), ok
}

func (l *LiveStats) Overtime() bool {
	game, _ := l.State["Game"].(map[string]any)
	overtime, _ := game["bOvertime"].(bool)
	return overtime
}

func (l *LiveStats) PlayerCount() int {
	players, _ := l.State["Players"].([]any)
	return len(players)
}

type RLEvent struct {
	Event tools.EventType `json:"Event"`
	Data  json.RawMessage `json:"Data"`
}

type StatsAPI struct {
	// PortsProvider returns the ports set in Rocket League StatsAPI config, the game defaults are used when it fails
	PortsProvider func() (rlgame.StatsAPIPorts, error)

	lastPorts     rlgame.StatsAPIPorts
	connMu        sync.RWMutex
	conn          statsAPIConn
	transport     StatsAPITransport
	lastStateTime int
	isFirstSent   bool
	isOvertime    bool

	LastInfo     *LiveStats
	EventManager *tools.EventManager
}

const (
	dialTimeout        = 3 * time.Second
	writeTimeout       = 3 * time.Second
	listenerLoop       = 2 * time.Second
	listenerErrorSleep = 5 * time.Second
)

func NewStatsAPI() *StatsAPI {
	logger.FuncDebug()

	return &StatsAPI{
		lastPorts:     rlgame.StatsAPIPorts{TCP: -1, Web: -1},
		EventManager:  tools.NewEventManager(),
		lastStateTime: -1,
		isFirstSent:   false,
		LastInfo: &LiveStats{
			State:  nil,
			Skills: make(map[rlapi.PlayerID][]rlapi.Skill),
		},
	}
}

func (s *StatsAPI) StartListener() {
	logger.FuncDebug()

	go s.innerStartListener()
}

func (s *StatsAPI) innerStartListener() {
	logger.FuncDebug()

	for {
		stream, transport, err := connectStatsAPI(s.currentPorts())
		if err != nil {
			time.Sleep(listenerErrorSleep)
			continue
		}

		logger.Rlogger.Info("Connected to StatsAPI", slog.String("Transport", string(transport)))
		s.setConn(stream, transport)
		s.readLoop(stream)
		s.setConn(nil, "")

		time.Sleep(listenerLoop)
	}
}

func (s *StatsAPI) setConn(conn statsAPIConn, transport StatsAPITransport) {
	s.connMu.Lock()
	defer s.connMu.Unlock()

	s.conn = conn
	s.transport = transport
}

// Transport returns the socket used to talk with the game, empty when not connected.
func (s *StatsAPI) Transport() StatsAPITransport {
	s.connMu.RLock()
	defer s.connMu.RUnlock()

	return s.transport
}

// SendCommand sends a command to the game on the connected socket (TCP or WebSocket).
func (s *StatsAPI) SendCommand(command StatsAPICommand) error {
	logger.FuncDebug()

	payload, err := json.Marshal(command)
	if err != nil {
		return err
	}

	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()

	if conn == nil {
		return ErrStatsAPINotConnected
	}

	if err := conn.Send(payload); err != nil {
		logger.Rlogger.Error("Failed to send StatsAPI command", slog.String("Command", command.Command), slog.Any("err", err))
		return err
	}

	logger.Rlogger.Debug("StatsAPI command sent", slog.String("Command", command.Command))

	return nil
}

func (s *StatsAPI) currentPorts() rlgame.StatsAPIPorts {
	logger.FuncDebug()

	ports := rlgame.DefaultStatsAPIPorts()
	if s.PortsProvider != nil {
		if configPorts, err := s.PortsProvider(); err == nil {
			ports = configPorts
		}
	}

	if ports != s.lastPorts {
		s.lastPorts = ports

		if ports.TCP == 0 && ports.Web == 0 {
			logger.Rlogger.Info("StatsAPI sockets are disabled in Rocket League config (Port=0 and WebPort=0)")
		} else {
			logger.Rlogger.Info("StatsAPI listening ports", slog.Int("Port", ports.TCP), slog.Int("WebPort", ports.Web))
		}
	}

	return ports
}

func (s *StatsAPI) readLoop(stream io.ReadCloser) {
	logger.FuncDebug()

	defer stream.Close()

	decoder := json.NewDecoder(stream)

	for {
		var rlEvent RLEvent
		err := decoder.Decode(&rlEvent)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			logger.Rlogger.Error("Reading error", slog.Any("error", err))
			return
		}

		var dynamicData map[string]any

		var jsonString string
		if err := json.Unmarshal(rlEvent.Data, &jsonString); err == nil {
			json.Unmarshal([]byte(jsonString), &dynamicData)
		} else {
			json.Unmarshal(rlEvent.Data, &dynamicData)
		}

		switch rlEvent.Event {
		case "MatchCreated", "MatchDestroyed":
			s.resetState()
		case "UpdateState":
			if dynamicData != nil {
				s.LastInfo.State = dynamicData

				if overtime := s.LastInfo.Overtime(); overtime != s.isOvertime {
					s.isOvertime = overtime
					if overtime {
						s.EventManager.Notify(EventOvertimeStarted, nil)
					}
				}

				if seconds, ok := s.LastInfo.TimeSeconds(); ok {
					if s.lastStateTime == -1 {
						s.lastStateTime = seconds
					} else if s.lastStateTime != seconds {
						s.lastStateTime = seconds

						if !s.isFirstSent && s.LastInfo.PlayerCount() > 0 {
							s.isFirstSent = true
							s.EventManager.Notify(EventFirstUpdateState, nil)
						}
					}
				}
			}
		}

		s.EventManager.Notify(rlEvent.Event, dynamicData)
	}
}

func (s *StatsAPI) resetState() {
	logger.FuncDebug()

	s.LastInfo = &LiveStats{
		State:  nil,
		Skills: make(map[rlapi.PlayerID][]rlapi.Skill),
	}
	s.lastStateTime = -1
	s.isFirstSent = false
	s.isOvertime = false
}
