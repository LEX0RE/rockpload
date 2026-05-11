package config

import (
	"encoding/json"
	"os"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type storageListConfig []*StorageConfig

func (wls *storageListConfig) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	var temp []*StorageConfig
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	temp = append([]*StorageConfig{ROCKY_WEBSITE}, temp...)

	if len(temp) <= 1 {
		temp = append(temp, BALLCHASING_WEBSITE)
	}

	if os.Getenv("ADD_LOCALHOST") == "true" {
		temp = append(temp, LOCAL_WEBSITE)
	}

	*wls = temp

	return nil
}

type StorageConfig struct {
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	IsPrimary    bool              `json:"is_primary"`
	IsPredefined bool              `json:"is_predefined"`
	URIParams    map[string]string `json:"uri_params"`
	NeedToken    bool              `json:"need_token"`
	Token        string            `json:"token"`
	SendPing     bool              `json:"send_ping"`
	PingPath     string            `json:"ping_path"`
	SendReplay   bool              `json:"send_replay"`
	ReplayPath   string            `json:"replay_path"`
	// SendLive   bool // TODO Not implemented yet
	// LivePath   string // TODO Not implemented yet
}

var ROCKY_WEBSITE = &StorageConfig{
	Name:         "Rocky",
	URL:          "https://lexore.ca/rocky/api",
	IsPrimary:    true,
	IsPredefined: true,
	URIParams:    map[string]string{},
	NeedToken:    false,
	Token:        "",
	SendPing:     false,
	PingPath:     "/",
	SendReplay:   true,
	ReplayPath:   "/upload",
	// SendLive:   false, // TODO Not implemented yet
	// LivePath:   "", // TODO Not implemented yet
}

var BALLCHASING_WEBSITE = &StorageConfig{
	Name:         "Ballchasing",
	URL:          "https://ballchasing.com/api",
	IsPrimary:    false,
	IsPredefined: true,
	URIParams:    map[string]string{"vibilitity": "public"},
	NeedToken:    true,
	Token:        "",
	SendPing:     true,
	PingPath:     "/",
	SendReplay:   true,
	ReplayPath:   "/v2/upload",
	// SendLive:   false, // TODO Not implemented yet
	// LivePath:   "", // TODO Not implemented yet
}

var LOCAL_WEBSITE = &StorageConfig{
	Name:         "Localhost",
	URL:          "http://localhost:3000",
	IsPrimary:    true,
	IsPredefined: true,
	URIParams:    map[string]string{},
	NeedToken:    false,
	Token:        "",
	SendPing:     false,
	PingPath:     "/",
	SendReplay:   true,
	ReplayPath:   "/upload",
	// SendLive:   false, // TODO Not implemented yet
	// LivePath:   "", // TODO Not implemented yet
}
