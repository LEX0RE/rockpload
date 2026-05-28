package rocket_network

import (
	"encoding/json"
	"slices"
	"strconv"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type Account struct {
	Player        *Player  `json:"player"`
	IsUnused      bool     `json:"is_unused,omitempty"`
	HistorySended []string `json:"-"`
}

// TODO Delete uploaded file when account is deleted
func NewAccount(profileId int) *Account {
	logger.FuncDebug()

	ac := &Account{IsUnused: false, Player: NewPlayer(profileId)}
	return ac
}

func (ac *Account) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	type Alias Account
	aux := (*Alias)(ac)

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if ac.HistorySended == nil {
		ac.HistorySended = make([]string, 0)
	}

	return nil
}

func (ac *Account) AccountName() string {
	if ac.Player == nil || ac.Player.Auth == nil {
		return "Unknown"
	}

	idName := ""
	if ac.Player.Auth.ProfileId >= 0 {
		idName = " (ID: " + strconv.Itoa(ac.Player.Auth.ProfileId) + ")"
	}

	if ac.Player.PlayerName == "" {
		return "Player" + idName
	}

	return ac.Player.PlayerName + " (ID: " + strconv.Itoa(ac.Player.Auth.ProfileId) + ")"
}

func (ac *Account) AddToMatchHistory(matchGUID string) {
	logger.FuncDebug()

	if !slices.Contains(ac.HistorySended, matchGUID) {
		ac.HistorySended = append(ac.HistorySended, matchGUID)
	}
}

func (ac *Account) IsConnected() bool {
	return ac.Player.IsAuthenticated()
}

func (ac *Account) Id() int {
	logger.FuncDebug()

	return ac.Player.Auth.ProfileId
}
