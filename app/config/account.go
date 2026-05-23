package config

import (
	"encoding/json"

	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type AccountMapConfig map[int]*rocket_network.Account

func (amc *AccountMapConfig) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	var temp map[int]*rocket_network.Account
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if len(temp) == 0 {
		temp = map[int]*rocket_network.Account{
			0: rocket_network.NewAccount(0),
		}
	}

	*amc = temp

	return nil
}
