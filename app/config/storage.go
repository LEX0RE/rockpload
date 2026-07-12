package config

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"

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
	} else {
		temp[ballchasingIndex].IsPrimary = BALLCHASING_STORAGE.IsPrimary
		temp[ballchasingIndex].IsPredefined = BALLCHASING_STORAGE.IsPredefined
		temp[ballchasingIndex].IsTemporary = BALLCHASING_STORAGE.IsTemporary
		temp[ballchasingIndex].StorageType = BALLCHASING_STORAGE.StorageType
	}

	fileSystemIndex := slices.IndexFunc(temp, func(c *StorageConfig) bool { return c.Name == FILE_SYSTEM_NAME })
	if fileSystemIndex == -1 {
		temp = append(temp, FILE_SYSTEM_STORAGE)
		temp[len(temp)-1].ReplayPath = constant.Paths.HomeDir
	} else {
		temp[fileSystemIndex].IsPrimary = FILE_SYSTEM_STORAGE.IsPrimary
		temp[fileSystemIndex].IsPredefined = FILE_SYSTEM_STORAGE.IsPredefined
		temp[fileSystemIndex].IsTemporary = BALLCHASING_STORAGE.IsTemporary
		temp[fileSystemIndex].StorageType = FILE_SYSTEM_STORAGE.StorageType
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
	Name         string            `json:"name" secret_id:"true"`
	SendReplay   bool              `json:"send_replay"`
	ReplayPath   string            `json:"replay_path"`
	TemplateName string            `json:"template_file"`
	URL          string            `json:"url"`
	IsPrimary    bool              `json:"-"`
	IsPredefined bool              `json:"-"`
	IsTemporary  bool              `json:"-"`
	StorageType  StorageConfigType `json:"storage_type"`
	URIParams    map[string]string `json:"uri_params"`
	NeedToken    bool              `json:"need_token"`
	Token        string            `json:"token,omitempty" secret:"true"`
	SendPing     bool              `json:"send_ping"`
	PingPath     string            `json:"ping_path"`
	SendLive     bool              `json:"send_live"`
	LivePath     string            `json:"live_path"`
}

var STORAGE_PRESET = map[string]*StorageConfig{
	ROCKY_STORAGE.Name: ROCKY_STORAGE,
	BALLCHASING_NAME:   BALLCHASING_STORAGE,
	FILE_SYSTEM_NAME:   FILE_SYSTEM_STORAGE,
}

var ROCKY_STORAGE = &StorageConfig{
	Name:         "Rocky",
	URL:          "https://lexore.ca/rocky/api",
	IsPrimary:    true,
	IsPredefined: true,
	IsTemporary:  false,
	StorageType:  WebsiteConfig,
	URIParams:    map[string]string{},
	NeedToken:    false,
	Token:        "",
	SendPing:     false,
	PingPath:     "/",
	SendReplay:   true,
	ReplayPath:   "/upload",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	SendLive:     true,
	LivePath:     "/upload/live",
}

var BALLCHASING_STORAGE = &StorageConfig{
	Name:         BALLCHASING_NAME,
	URL:          "https://ballchasing.com/api",
	IsPrimary:    false,
	IsPredefined: true,
	IsTemporary:  false,
	StorageType:  WebsiteConfig,
	URIParams:    map[string]string{"visibility": "public"},
	NeedToken:    true,
	Token:        "",
	SendPing:     true,
	PingPath:     "/",
	SendReplay:   false,
	ReplayPath:   "/v2/upload",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	SendLive:     false,
	LivePath:     "",
}

var FILE_SYSTEM_STORAGE = &StorageConfig{
	Name:         FILE_SYSTEM_NAME,
	URL:          "",
	IsPrimary:    false,
	IsPredefined: true,
	IsTemporary:  false,
	StorageType:  FileSystemConfig,
	URIParams:    map[string]string{},
	NeedToken:    false,
	Token:        "",
	SendPing:     false,
	PingPath:     "/",
	SendReplay:   false,
	ReplayPath:   "",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	SendLive:     false,
	LivePath:     "",
}

// Dev testing storage only
var LOCALHOST_STORAGE = &StorageConfig{
	Name:         LOCALHOST_NAME,
	URL:          "http://localhost:3000",
	IsPrimary:    false,
	IsPredefined: false,
	IsTemporary:  true,
	StorageType:  WebsiteConfig,
	URIParams:    map[string]string{},
	NeedToken:    false,
	Token:        "",
	SendPing:     false,
	PingPath:     "/",
	SendReplay:   true,
	ReplayPath:   "/upload",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	SendLive:     false,
	LivePath:     "",
}

func (sc *StorageConfig) UnmarshalJSON(data []byte) error {
	var jsonMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return err
	}

	var name string
	if nameRaw, ok := jsonMap["name"]; ok {
		_ = json.Unmarshal(nameRaw, &name)
	}

	type Alias StorageConfig
	if err := json.Unmarshal(data, (*Alias)(sc)); err != nil {
		return err
	}

	if preset, ok := STORAGE_PRESET[name]; ok {
		valSC := reflect.ValueOf(sc).Elem()
		valPreset := reflect.ValueOf(preset).Elem()
		typeSC := valSC.Type()

		for i := 0; i < valSC.NumField(); i++ {
			field := typeSC.Field(i)

			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}
			jsonKey := strings.Split(jsonTag, ",")[0]

			if _, present := jsonMap[jsonKey]; !present {
				valSC.Field(i).Set(valPreset.Field(i))
			}
		}

		// Force specific field to be sure the storage are in a good state
		sc.IsPrimary = preset.IsPrimary
		sc.IsPredefined = preset.IsPredefined
		sc.IsTemporary = preset.IsTemporary
		sc.StorageType = preset.StorageType
	}

	return nil
}
