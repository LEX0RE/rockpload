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
	BLAST_NAME       = "BLAST.tv"
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
		fresh := *BALLCHASING_STORAGE
		fresh.UploadStyle = UploadDisabled
		temp = append(temp, &fresh)
	} else {
		reconcilePreset(temp[ballchasingIndex], BALLCHASING_STORAGE)
	}

	blastIndex := slices.IndexFunc(temp, func(c *StorageConfig) bool { return c.Name == BLAST_NAME })
	if blastIndex == -1 {
		fresh := *BLAST_STORAGE
		fresh.UploadStyle = UploadDisabled
		temp = append(temp, &fresh)
	} else {
		reconcilePreset(temp[blastIndex], BLAST_STORAGE)
	}

	fileSystemIndex := slices.IndexFunc(temp, func(c *StorageConfig) bool { return c.Name == FILE_SYSTEM_NAME })
	if fileSystemIndex == -1 {
		fresh := *FILE_SYSTEM_STORAGE
		fresh.UploadStyle = UploadDisabled
		fresh.ReplayPath = constant.Paths.HomeDir
		temp = append(temp, &fresh)
	} else {
		reconcilePreset(temp[fileSystemIndex], FILE_SYSTEM_STORAGE)
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

func reconcilePreset(sc *StorageConfig, preset *StorageConfig) {
	logger.FuncDebug()

	sc.IsPrimary = preset.IsPrimary
	sc.IsPredefined = preset.IsPredefined
	sc.IsTemporary = preset.IsTemporary
	sc.TokenStyle = preset.TokenStyle
	sc.LiveStyle = preset.LiveStyle
	sc.HelpText = preset.HelpText
	sc.HelpURL = preset.HelpURL

	if sc.UploadStyle != UploadDisabled && sc.UploadStyle != preset.UploadStyle {
		sc.UploadStyle = preset.UploadStyle
	}

	if sc.PingStyle != PingDisabled && sc.PingStyle != preset.PingStyle {
		sc.PingStyle = preset.PingStyle
	}
}

type UploadStyleType int

const (
	UploadDisabled UploadStyleType = iota
	LocalFileCopy
	MultipartUpload
	PresignedSessionUpload
)

var UploadStyleLabels = [...]string{
	UploadDisabled:         "Upload Disabled",
	LocalFileCopy:          "Local File Copy",
	MultipartUpload:        "Multipart Form POST",
	PresignedSessionUpload: "Presigned Session (URL + status poll)",
}

func (s UploadStyleType) Label() string {
	logger.FuncDebug()

	if int(s) < 0 || int(s) >= len(UploadStyleLabels) {
		return UploadStyleLabels[UploadDisabled]
	}

	return UploadStyleLabels[s]
}

func UploadStyleFromLabel(label string) UploadStyleType {
	logger.FuncDebug()

	for style, l := range UploadStyleLabels {
		if l == label {
			return UploadStyleType(style)
		}
	}

	return UploadDisabled
}

type PingStyleType int

const (
	PingDisabled PingStyleType = iota
	PingRequiresOK
	PingNotFoundIsValid
)

var PingStyleLabels = [...]string{
	PingDisabled:        "Ping Disabled",
	PingRequiresOK:      "Requires 200 OK",
	PingNotFoundIsValid: "404 counts as valid",
}

func (s PingStyleType) Label() string {
	logger.FuncDebug()

	if int(s) < 0 || int(s) >= len(PingStyleLabels) {
		return PingStyleLabels[PingDisabled]
	}

	return PingStyleLabels[s]
}

func PingStyleFromLabel(label string) PingStyleType {
	logger.FuncDebug()

	for style, l := range PingStyleLabels {
		if l == label {
			return PingStyleType(style)
		}
	}

	return PingDisabled
}

type TokenStyleType int

const (
	NoToken TokenStyleType = iota
	RawToken
	BearerToken
)

var TokenStyleLabels = [...]string{
	NoToken:     "No Token",
	RawToken:    "Raw Token",
	BearerToken: "Bearer Token",
}

func (s TokenStyleType) Label() string {
	logger.FuncDebug()

	if int(s) < 0 || int(s) >= len(TokenStyleLabels) {
		return TokenStyleLabels[NoToken]
	}

	return TokenStyleLabels[s]
}

func TokenStyleFromLabel(label string) TokenStyleType {
	logger.FuncDebug()

	for style, l := range TokenStyleLabels {
		if l == label {
			return TokenStyleType(style)
		}
	}

	return NoToken
}

type LiveStyleType int

const (
	LiveDisabled LiveStyleType = iota
	LiveEnabled
)

var LiveStyleLabels = [...]string{
	LiveDisabled: "Live Disabled",
	LiveEnabled:  "Live Enabled",
}

func (s LiveStyleType) Label() string {
	logger.FuncDebug()

	if int(s) < 0 || int(s) >= len(LiveStyleLabels) {
		return LiveStyleLabels[LiveDisabled]
	}

	return LiveStyleLabels[s]
}

func LiveStyleFromLabel(label string) LiveStyleType {
	logger.FuncDebug()

	for style, l := range LiveStyleLabels {
		if l == label {
			return LiveStyleType(style)
		}
	}

	return LiveDisabled
}

type StorageConfig struct {
	Name         string            `json:"name" secret_id:"true"`
	HelpText     string            `json:"help_text"`
	HelpURL      string            `json:"help_url"`
	UploadStyle  UploadStyleType   `json:"upload_style"`
	ReplayPath   string            `json:"replay_path"`
	TemplateName string            `json:"template_file"`
	URL          string            `json:"url"`
	IsPrimary    bool              `json:"-"`
	IsPredefined bool              `json:"-"`
	IsTemporary  bool              `json:"-"`
	URIParams    map[string]string `json:"uri_params"`
	TokenStyle   TokenStyleType    `json:"token_style"`
	Token        string            `json:"token,omitempty" secret:"true"`
	PingStyle    PingStyleType     `json:"ping_style"`
	PingPath     string            `json:"ping_path"`
	PingProbeID  string            `json:"ping_probe_id"`
	LiveStyle    LiveStyleType     `json:"live_style"`
	LivePath     string            `json:"live_path"`
}

var STORAGE_PRESET = map[string]*StorageConfig{
	ROCKY_STORAGE.Name: ROCKY_STORAGE,
	BALLCHASING_NAME:   BALLCHASING_STORAGE,
	BLAST_NAME:         BLAST_STORAGE,
	FILE_SYSTEM_NAME:   FILE_SYSTEM_STORAGE,
}

var ROCKY_STORAGE = &StorageConfig{
	Name:         "Rocky",
	HelpText:     "Want to see the website? Click here",
	HelpURL:      "https://lexore.ca/rocky",
	URL:          "https://lexore.ca/rocky/api",
	IsPrimary:    true,
	IsPredefined: true,
	IsTemporary:  false,
	UploadStyle:  MultipartUpload,
	URIParams:    map[string]string{},
	TokenStyle:   NoToken,
	Token:        "",
	PingStyle:    PingDisabled,
	PingPath:     "/",
	ReplayPath:   "/upload",
	TemplateName: "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	LiveStyle:    LiveEnabled,
	LivePath:     "/upload/live",
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
