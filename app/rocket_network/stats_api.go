package rocket_network

import (
	"encoding/json"
	"log/slog"
	"net"
	"time"

	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"
	"github.com/dank/rlapi"
)

const (
	EventFirstUpdateState tools.EventType = "first_update_state"
)

func assignPtr[T any](m map[string]any, key string, target **T, assign func(map[string]any, string, *T)) {
	if _, ok := m[key]; !ok {
		return
	}

	v := new(T)
	assign(m, key, v)
	*target = v
}

type UpdateState struct {
	MatchGuid string
	Players   []*UpdateStatePlayer
	Game      *UpdateStateGame
}

type UpdateStatePlayer struct {
	Name          string
	PrimaryId     string
	Shortcut      int
	TeamNum       int
	Score         int
	Goals         int
	Shots         int
	Assists       int
	Saves         int
	Touches       int
	CarTouches    int
	Demos         int
	Loadout       []string
	PickupClass   string
	bHasCar       *bool
	Speed         *float64
	Boost         *int
	bBoosting     *bool
	bOnGround     *bool
	bOnWall       *bool
	bPowersliding *bool
	bDemolished   *bool
	bSupersonic   *bool
	Attacker      *UpdateStateTarget
}

type UpdateStateTarget struct {
	Name     string
	Shortcut int
	TeamNum  int
}

type UpdateStateGame struct {
	Teams       []*UpdateStateGameTeam
	PlaylistId  int
	TimeSeconds int
	bOvertime   bool
	Ball        *UpdateStateGameBall
	bReplay     bool
	bHasWinner  bool
	Winner      string
	Arena       string
	Frame       *int
	Elapsed     *float64
	bHasTarget  bool
	Target      *UpdateStateTarget
}

type UpdateStateGameBall struct {
	Speed   float64
	TeamNum int
}

type UpdateStateGameTeam struct {
	Name           string
	TeamNum        int
	Score          int
	ColorPrimary   string
	ColorSecondary string
}

type LiveStats struct {
	State  *UpdateState
	Skills map[rlapi.PlayerID][]rlapi.Skill
}

type RLEvent struct {
	Event tools.EventType `json:"Event"`
	Data  json.RawMessage `json:"Data"`
}

type StatsAPI struct {
	port          string
	lastStateTime int
	isFirstSent   bool

	LastInfo     *LiveStats
	EventManager *tools.EventManager
}

const (
	defaultPort        = "49123"
	dialTimeout        = 3 * time.Second
	listenerLoop       = 2 * time.Second
	listenerErrorSleep = 5 * time.Second
)

func NewStatsAPI() *StatsAPI {
	logger.FuncDebug()

	return &StatsAPI{
		port:          defaultPort,
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

		switch rlEvent.Event {
		case "MatchCreated", "MatchDestroyed":
			s.resetState()
		case "UpdateState":
			updateStateData := ExtractUpdateState(dynamicData)

			if updateStateData != nil {
				s.LastInfo.State = updateStateData

				if s.lastStateTime == -1 {
					s.lastStateTime = s.LastInfo.State.Game.TimeSeconds
				} else if s.lastStateTime != s.LastInfo.State.Game.TimeSeconds {
					s.lastStateTime = s.LastInfo.State.Game.TimeSeconds

					if !s.isFirstSent && len(s.LastInfo.State.Players) > 0 {
						s.isFirstSent = true
						s.EventManager.Notify(EventFirstUpdateState, nil)
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
}

func ExtractUpdateState(dynamicData map[string]any) *UpdateState {
	logger.FuncDebug()

	state := &UpdateState{}

	assignStr := func(m map[string]any, key string, target *string) {
		if v, ok := m[key].(string); ok {
			*target = v
		}
	}

	assignInt := func(m map[string]any, key string, target *int) {
		if v, ok := m[key].(float64); ok {
			*target = int(v)
		}
	}

	assignFloat := func(m map[string]any, key string, target *float64) {
		if v, ok := m[key].(float64); ok {
			*target = v
		}
	}

	assignBool := func(m map[string]any, key string, target *bool) {
		if v, ok := m[key].(bool); ok {
			*target = v
		}
	}

	assignStrSlice := func(m map[string]any, key string, target *[]string) {
		arr, ok := m[key].([]any)
		if !ok {
			return
		}

		items := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}

		*target = items
	}

	extractTarget := func(m map[string]any, key string) *UpdateStateTarget {
		obj, ok := m[key].(map[string]any)
		if !ok {
			return nil
		}

		target := &UpdateStateTarget{}
		assignStr(obj, "Name", &target.Name)
		assignInt(obj, "Shortcut", &target.Shortcut)
		assignInt(obj, "TeamNum", &target.TeamNum)

		return target
	}

	assignStr(dynamicData, "MatchGuid", &state.MatchGuid)

	if playersArr, ok := dynamicData["Players"].([]any); ok {
		for _, playerAny := range playersArr {
			if playerObj, ok := playerAny.(map[string]any); ok {
				var newPlayer UpdateStatePlayer

				assignStr(playerObj, "Name", &newPlayer.Name)
				assignStr(playerObj, "PrimaryId", &newPlayer.PrimaryId)
				assignInt(playerObj, "Shortcut", &newPlayer.Shortcut)
				assignInt(playerObj, "TeamNum", &newPlayer.TeamNum)
				assignInt(playerObj, "Score", &newPlayer.Score)
				assignInt(playerObj, "Goals", &newPlayer.Goals)
				assignInt(playerObj, "Shots", &newPlayer.Shots)
				assignInt(playerObj, "Assists", &newPlayer.Assists)
				assignInt(playerObj, "Saves", &newPlayer.Saves)
				assignInt(playerObj, "Touches", &newPlayer.Touches)
				assignInt(playerObj, "CarTouches", &newPlayer.CarTouches)
				assignInt(playerObj, "Demos", &newPlayer.Demos)
				assignStrSlice(playerObj, "Loadout", &newPlayer.Loadout)
				assignStr(playerObj, "PickupClass", &newPlayer.PickupClass)
				assignPtr(playerObj, "bHasCar", &newPlayer.bHasCar, assignBool)
				assignPtr(playerObj, "Speed", &newPlayer.Speed, assignFloat)
				assignPtr(playerObj, "Boost", &newPlayer.Boost, assignInt)
				assignPtr(playerObj, "bBoosting", &newPlayer.bBoosting, assignBool)
				assignPtr(playerObj, "bOnGround", &newPlayer.bOnGround, assignBool)
				assignPtr(playerObj, "bOnWall", &newPlayer.bOnWall, assignBool)
				assignPtr(playerObj, "bPowersliding", &newPlayer.bPowersliding, assignBool)
				assignPtr(playerObj, "bDemolished", &newPlayer.bDemolished, assignBool)
				assignPtr(playerObj, "bSupersonic", &newPlayer.bSupersonic, assignBool)
				newPlayer.Attacker = extractTarget(playerObj, "Attacker")

				state.Players = append(state.Players, &newPlayer)
			}
		}
	}

	if gameObj, ok := dynamicData["Game"].(map[string]any); ok {
		if state.Game == nil {
			state.Game = &UpdateStateGame{}
		}

		assignInt(gameObj, "PlaylistId", &state.Game.PlaylistId)
		assignInt(gameObj, "TimeSeconds", &state.Game.TimeSeconds)
		assignBool(gameObj, "bOvertime", &state.Game.bOvertime)
		assignBool(gameObj, "bReplay", &state.Game.bReplay)
		assignPtr(gameObj, "Frame", &state.Game.Frame, assignInt)
		assignPtr(gameObj, "Elapsed", &state.Game.Elapsed, assignFloat)
		assignBool(gameObj, "bHasWinner", &state.Game.bHasWinner)
		assignStr(gameObj, "Winner", &state.Game.Winner)
		assignStr(gameObj, "Arena", &state.Game.Arena)
		assignBool(gameObj, "bHasTarget", &state.Game.bHasTarget)
		state.Game.Target = extractTarget(gameObj, "Target")

		if ballObj, ok := gameObj["Ball"].(map[string]any); ok {
			ball := &UpdateStateGameBall{}
			assignFloat(ballObj, "Speed", &ball.Speed)
			assignInt(ballObj, "TeamNum", &ball.TeamNum)
			state.Game.Ball = ball
		}

		if teamsArr, ok := gameObj["Teams"].([]any); ok {
			for _, teamAny := range teamsArr {
				if teamObj, ok := teamAny.(map[string]any); ok {
					var newTeam UpdateStateGameTeam

					assignStr(teamObj, "Name", &newTeam.Name)
					assignStr(teamObj, "ColorPrimary", &newTeam.ColorPrimary)
					assignStr(teamObj, "ColorSecondary", &newTeam.ColorSecondary)
					assignInt(teamObj, "TeamNum", &newTeam.TeamNum)
					assignInt(teamObj, "Score", &newTeam.Score)

					state.Game.Teams = append(state.Game.Teams, &newTeam)
				}
			}
		}
	}

	return state
}
