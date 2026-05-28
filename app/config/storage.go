package config

import (
	"encoding/json"
	"errors"
	"os"
	"slices"

	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	BALLCHASING_NAME = "Ballchasing"
	LOCALHOST_NAME   = "Localhost"
	FILE_SYSTEM_NAME = "FileSystem"
)

type storageListConfig []*StorageConfig

func (wls *storageListConfig) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	var temp []*StorageConfig
	if err := json.Unmarshal(data, &temp); err != nil {
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) || syntaxErr.Offset != 0 {
			return err
		}
	}

	temp = append([]*StorageConfig{ROCKY_STORAGE}, temp...)

	ballchasingIndex := slices.IndexFunc(temp, func(c *StorageConfig) bool { return c.Name == BALLCHASING_NAME })
	if ballchasingIndex == -1 {
		temp = append(temp, BALLCHASING_STORAGE)
	}

	fileSystemIndex := slices.IndexFunc(temp, func(c *StorageConfig) bool { return c.Name == FILE_SYSTEM_NAME })
	if fileSystemIndex == -1 {
		temp = append(temp, FILE_SYSTEM_STORAGE)
	}

	if os.Getenv("ADD_LOCALHOST") == "true" {
		localhostIndex := slices.IndexFunc(temp, func(c *StorageConfig) bool { return c.Name == LOCALHOST_NAME })
		if localhostIndex == -1 {
			temp = append(temp, LOCALHOST_STORAGE)
		}
	}

	*wls = temp

	return nil
}

type StorageConfigType int

const (
	WebsiteConfig StorageConfigType = iota
	FileSystemConfig
)

type StorageConfig struct {
	Name         string            `json:"name"`
	SendReplay   bool              `json:"send_replay"`
	ReplayPath   string            `json:"replay_path"`
	TemplateName string            `json:"template_file"`
	URL          string            `json:"url"`
	IsPrimary    bool              `json:"is_primary"`
	IsPredefined bool              `json:"is_predefined"`
	StorageType  StorageConfigType `json:"storage_type"`
	URIParams    map[string]string `json:"uri_params"`
	NeedToken    bool              `json:"need_token"`
	Token        string            `json:"token"`
	SendPing     bool              `json:"send_ping"`
	PingPath     string            `json:"ping_path"`
	// SendLive   bool // TODO Not implemented yet
	// LivePath   string // TODO Not implemented yet
}

var ROCKY_STORAGE = &StorageConfig{
	Name:         "Rocky",
	URL:          "https://lexore.ca/rocky/api",
	IsPrimary:    true,
	IsPredefined: true,
	StorageType:  WebsiteConfig,
	URIParams:    map[string]string{},
	NeedToken:    false,
	Token:        "",
	SendPing:     false,
	PingPath:     "/",
	SendReplay:   true,
	ReplayPath:   "/upload",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	// SendLive:   false, // TODO Not implemented yet
	// LivePath:   "", // TODO Not implemented yet
}

var BALLCHASING_STORAGE = &StorageConfig{
	Name:         BALLCHASING_NAME,
	URL:          "https://ballchasing.com/api",
	IsPrimary:    false,
	IsPredefined: true,
	StorageType:  WebsiteConfig,
	URIParams:    map[string]string{"visibility": "public"},
	NeedToken:    true,
	Token:        "",
	SendPing:     true,
	PingPath:     "/",
	SendReplay:   false,
	ReplayPath:   "/v2/upload",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	// SendLive:   false, // TODO Not implemented yet
	// LivePath:   "", // TODO Not implemented yet
}

var LOCALHOST_STORAGE = &StorageConfig{
	Name:         LOCALHOST_NAME,
	URL:          "http://localhost:3000",
	IsPrimary:    true,
	IsPredefined: true,
	StorageType:  WebsiteConfig,
	URIParams:    map[string]string{},
	NeedToken:    false,
	Token:        "",
	SendPing:     false,
	PingPath:     "/",
	SendReplay:   true,
	ReplayPath:   "/upload",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	// SendLive:   false, // TODO Not implemented yet
	// LivePath:   "", // TODO Not implemented yet
}

var FILE_SYSTEM_STORAGE = &StorageConfig{
	Name:         FILE_SYSTEM_NAME,
	URL:          "",
	IsPrimary:    false,
	IsPredefined: true,
	StorageType:  FileSystemConfig,
	URIParams:    map[string]string{},
	NeedToken:    false,
	Token:        "",
	SendPing:     false,
	PingPath:     "/",
	SendReplay:   false,
	ReplayPath:   constant.GetHomePath(),
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	// SendLive:   false, // TODO Not implemented yet
	// LivePath:   "", // TODO Not implemented yet
}
