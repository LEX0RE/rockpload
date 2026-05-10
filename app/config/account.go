package config

import (
	"encoding/json"

	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type accountMapConfig map[int]*AccountConfig

func (amc *accountMapConfig) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	var temp map[int]*AccountConfig
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if len(temp) == 0 {
		temp = map[int]*AccountConfig{
			0: NewAccountConfig(0),
		}
	}

	*amc = temp

	return nil
}

type AccountConfig struct {
	Player        rocket_network.Player `json:"player"`
	IsUnused      bool                  `json:"is_unused,omitempty"`
	HistorySended []string              `json:"-"`
}

// TODO Delete uploaded file when account is deleted
func NewAccountConfig(profileId int) *AccountConfig {
	logger.FuncDebug()

	ac := &AccountConfig{IsUnused: false}
	ac.Player = *rocket_network.NewPlayer(profileId)
	return ac
}

func (ac *AccountConfig) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	type Alias AccountConfig
	aux := (*Alias)(ac)

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if ac.HistorySended == nil {
		ac.HistorySended = make([]string, 0)
	}

	return nil
}

func (ac *AccountConfig) AddToMatchHistory(matchGUID string) {
	logger.FuncDebug()

	alreadyInHistory := false
	for _, guid := range ac.HistorySended {
		if guid == matchGUID {
			alreadyInHistory = true
		}
	}

	if !alreadyInHistory {
		ac.HistorySended = append(ac.HistorySended, matchGUID)
	}
}

func (ac *AccountConfig) IsConnected() bool {
	return ac.Player.Auth != nil && ac.Player.Auth.IsAuthenticated()
}

func (ac *AccountConfig) Id() int {
	logger.FuncDebug()

	return ac.Player.Auth.ProfileId
}
