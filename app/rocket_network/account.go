package rocket_network

import (
	"encoding/json"
	"strconv"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type Account struct {
	Player   *Player `json:"player"`
	IsUnused bool    `json:"is_unused,omitempty"`
}

func NewAccount(profileId int) *Account {
	logger.FuncDebug()

	ac := &Account{IsUnused: false, Player: NewPlayer(profileId)}
	return ac
}

func (ac *Account) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	type Alias Account
	aux := (*Alias)(ac)

	return json.Unmarshal(data, &aux)
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

func (ac *Account) IsConnected() bool {
	return ac.Player.IsAuthenticated()
}

func (ac *Account) Id() int {
	logger.FuncDebug()

	return ac.Player.Auth.ProfileId
}
