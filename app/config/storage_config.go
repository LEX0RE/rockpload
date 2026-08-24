package config

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type StorageConfig struct {
	Name                string                  `json:"name" secret_id:"true"`
	HelpText            string                  `json:"help_text"`
	HelpURL             string                  `json:"help_url"`
	UploadStyle         UploadStyleType         `json:"upload_style"`
	ReplayPath          string                  `json:"replay_path"`
	TemplateName        string                  `json:"template_file"`
	URL                 string                  `json:"url"`
	IsPrimary           bool                    `json:"-"`
	IsPredefined        bool                    `json:"-"`
	IsTemporary         bool                    `json:"-"`
	URIParams           map[string]string       `json:"uri_params"`
	TokenStyle          TokenStyleType          `json:"token_style"`
	Token               string                  `json:"token,omitempty" secret:"true"`
	PingStyle           PingStyleType           `json:"ping_style"`
	PingPath            string                  `json:"ping_path"`
	PingProbeID         string                  `json:"ping_probe_id"`
	LiveStyle           LiveStyleType           `json:"live_style"`
	LivePath            string                  `json:"live_path"`
	RenameStyle         RenameStyleType         `json:"rename_style"`
	RenamePath          string                  `json:"rename_path"`
	PlaylistFilterStyle PlaylistFilterStyleType `json:"playlist_filter_style"`
	FilteredPlaylists   []PlaylistFilterEntry   `json:"filtered_playlists,omitempty"`
}

var STORAGE_PRESET = map[string]*StorageConfig{
	ROCKY_STORAGE.Name: ROCKY_STORAGE,
	BALLCHASING_NAME:   BALLCHASING_STORAGE,
	BLAST_NAME:         BLAST_STORAGE,
	FILE_SYSTEM_NAME:   FILE_SYSTEM_STORAGE,
}

var ROCKY_STORAGE = &StorageConfig{
	Name:                "Rocky",
	HelpText:            "Want to see the website? Click here",
	HelpURL:             "https://lexore.ca/rocky",
	URL:                 "https://lexore.ca/rocky/api",
	IsPrimary:           true,
	IsPredefined:        true,
	IsTemporary:         false,
	UploadStyle:         MultipartUpload,
	URIParams:           map[string]string{},
	TokenStyle:          NoToken,
	Token:               "",
	PingStyle:           PingDisabled,
	PingPath:            "/",
	ReplayPath:          "/upload",
	TemplateName:        "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	LiveStyle:           LiveEnabled,
	LivePath:            "/upload/live",
	RenameStyle:         RenameDisabled,
	PlaylistFilterStyle: PlaylistFilterFollowAny,
}

var BALLCHASING_STORAGE = &StorageConfig{
	Name:         BALLCHASING_NAME,
	UploadStyle:  MultipartUpload,
	ReplayPath:   "/v2/upload",
	URL:          "https://ballchasing.com/api",
	IsPrimary:    false,
	IsPredefined: true,
	IsTemporary:  false,
	URIParams:    map[string]string{"visibility": "public"},
	TokenStyle:   RawToken,
	Token:        "",
	PingStyle:    PingRequiresOK,
	PingPath:     "/",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	LiveStyle:    LiveDisabled,
	LivePath:     "",
	RenameStyle:  RenamePatchTitle,
	RenamePath:   "/replays/",
}

var BLAST_STORAGE = &StorageConfig{
	Name:         BLAST_NAME,
	HelpText:     "Need a token? Click here",
	HelpURL:      "https://blast.tv/profile?utm_source=rockpload&utm_medium=desktop-app&utm_campaign=upload-token",
	URL:          "https://api.blast.tv/v1/community-stats/rl/replays",
	ReplayPath:   "/upload-session",
	IsPrimary:    false,
	IsPredefined: true,
	IsTemporary:  false,
	UploadStyle:  PresignedSessionUpload,
	PingStyle:    PingNotFoundIsValid,
	TokenStyle:   BearerToken,
	URIParams:    map[string]string{},
	Token:        "",
	PingPath:     "/",
	PingProbeID:  "00000000-0000-0000-0000-000000000000",
	TemplateName: "",
	LiveStyle:    LiveDisabled,
	LivePath:     "",
	RenameStyle:  RenameDisabled,
}

var FILE_SYSTEM_STORAGE = &StorageConfig{
	Name:         FILE_SYSTEM_NAME,
	UploadStyle:  LocalFileCopy,
	ReplayPath:   "",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	URL:          "",
	IsPrimary:    false,
	IsPredefined: true,
	IsTemporary:  false,
	URIParams:    map[string]string{},
	TokenStyle:   NoToken,
	Token:        "",
	PingStyle:    PingDisabled,
	PingPath:     "/",
	LiveStyle:    LiveDisabled,
	LivePath:     "",
	RenameStyle:  RenameDisabled,
}

// Dev testing storage only
var LOCALHOST_STORAGE = &StorageConfig{
	Name:         LOCALHOST_NAME,
	UploadStyle:  UploadDisabled,
	ReplayPath:   "/upload",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	URL:          "http://localhost:3000",
	IsPrimary:    false,
	IsPredefined: false,
	IsTemporary:  true,
	URIParams:    map[string]string{},
	TokenStyle:   NoToken,
	Token:        "",
	PingStyle:    PingDisabled,
	PingPath:     "/",
	LiveStyle:    LiveDisabled,
	LivePath:     "",
	RenameStyle:  RenameDisabled,
}

func (sc *StorageConfig) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

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
			jsonKey, _, _ := strings.Cut(jsonTag, ",")

			if _, present := jsonMap[jsonKey]; !present {
				valSC.Field(i).Set(valPreset.Field(i))
			}
		}

		reconcilePreset(sc, preset)
	}

	return nil
}
