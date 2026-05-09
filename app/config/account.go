package config

import (
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

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
