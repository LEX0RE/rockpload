package rocket_network

import (
	"encoding/json"
	"log/slog"
	"net"
	"time"

	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type RLEvent struct {
	Event string          `json:"Event"`
	Data  json.RawMessage `json:"Data"`
}

type StatsAPI struct {
	port string

	EventManager *tools.EventManager
}

const (
	defaultPort        = "49123"
	dialTimeout        = 3 * time.Second
	listenerLoop       = 2 * time.Second
	listenerErrorSleep = 5 * time.Second
)

func NewStatsAPI() *StatsAPI {
	return &StatsAPI{port: defaultPort, EventManager: tools.NewEventManager()}
}

func (s *StatsAPI) StartListener() {
	go s.innerStartListener()
}

func (s *StatsAPI) innerStartListener() {
	logger.FuncDebug()
	address := "localhost:" + s.port

	for {
		conn, err := net.DialTimeout("tcp", address, dialTimeout)
		if err != nil {
			time.Sleep(listenerErrorSleep)
			continue
		}

		s.readLoop(conn)

		time.Sleep(listenerLoop)
	}
}

func (s *StatsAPI) readLoop(conn net.Conn) {
	logger.FuncDebug()

	defer conn.Close()

	decoder := json.NewDecoder(conn)

	for {
		var rlEvent RLEvent
		err := decoder.Decode(&rlEvent)
		if err != nil {
			if err.Error() == "EOF" {
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

		s.EventManager.Notify(rlEvent.Event, dynamicData)
	}
}
