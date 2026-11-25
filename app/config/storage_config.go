package config

import (
	"encoding/json"
	"log/slog"
	"net/url"
	"reflect"
	"slices"
	"strings"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

var GLOBAL_LOCKED_FIELDS = []string{
	"upload_style",
	"rename_path",
	"url",
	"token_style",
	"ping_path",
	"ping_probe_id",
	"live_path",
	"open_replay_origin",
	"open_replay_path",
}

type StorageConfig struct {
	Name                string                  `json:"name" secret_id:"true"`
	HelpText            string                  `json:"help_text"`
	HelpURL             string                  `json:"help_url"`
	UploadStyle         UploadStyleType         `json:"upload_style"`
	ReplayPath          string                  `json:"replay_path"`
	TemplateName        string                  `json:"template_file"`
	URL                 string                  `json:"url"`
	IsPredefined        bool                    `json:"-"`
	IsTemporary         bool                    `json:"-"`
	PresetLockedFields  []string                `json:"-"`
	PresetAllowedValues map[string][]int        `json:"-"`
	PresetFieldLinks    map[string]FieldLink    `json:"-"`
	LockedFields        []string                `json:"locked_fields,omitempty"`
	Enabled             bool                    `json:"enabled"`
	URIParams           map[string]string       `json:"uri_params"`
	TokenStyle          TokenStyleType          `json:"token_style"`
	Token               string                  `json:"token,omitempty" secret:"true"`
	PingStyle           PingStyleType           `json:"ping_style"`
	PingPath            string                  `json:"ping_path"`
	PingProbeID         string                  `json:"ping_probe_id"`
	LiveStyle           LiveStyleType           `json:"live_style"`
	LivePath            string                  `json:"live_path"`
	OpenReplayStyle     OpenReplayStyleType     `json:"open_replay_style"`
	OpenReplayOrigin    string                  `json:"open_replay_origin"`
	OpenReplayPath      string                  `json:"open_replay_path"`
	RenameStyle         RenameStyleType         `json:"rename_style"`
	RenamePath          string                  `json:"rename_path"`
	PlaylistFilterStyle PlaylistFilterStyleType `json:"playlist_filter_style"`
	FilteredPlaylists   []PlaylistFilterEntry   `json:"filtered_playlists,omitempty"`
}

var storageFieldByJSON = func() map[string]int {
	fields := map[string]int{}
	t := reflect.TypeFor[StorageConfig]()

	for i := range t.NumField() {
		if key := jsonFieldName(t.Field(i)); key != "-" {
			fields[key] = i
		}
	}

	return fields
}()

var STYLE_LABELS = map[string][]string{
	"upload_style":          UploadStyleLabels[:],
	"token_style":           TokenStyleLabels[:],
	"ping_style":            PingStyleLabels[:],
	"live_style":            LiveStyleLabels[:],
	"open_replay_style":     OpenReplayStyleLabels[:],
	"rename_style":          RenameStyleLabels[:],
	"playlist_filter_style": PlaylistFilterStyleLabels[:],
}

func allowed[T ~int](values ...T) []int {
	ints := make([]int, len(values))

	for i, value := range values {
		ints[i] = int(value)
	}

	return ints
}

func (sc *StorageConfig) AllowedValues(styleKey string) []int {
	if values, ok := sc.PresetAllowedValues[styleKey]; ok {
		return values
	}

	values := make([]int, len(STYLE_LABELS[styleKey]))
	for i := range values {
		values[i] = i
	}

	return values
}

type FieldLink struct {
	Text string
	URL  string
}

func (l FieldLink) Resolve(values map[string]string) (*url.URL, bool) {
	raw := l.URL
	for placeholder, value := range values {
		raw = strings.ReplaceAll(raw, placeholder, url.QueryEscape(value))
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || !strings.EqualFold(parsed.Scheme, "https") {
		return nil, false
	}

	return parsed, true
}

func (sc *StorageConfig) FieldLink(jsonKey string) (FieldLink, bool) {
	link, ok := sc.PresetFieldLinks[jsonKey]

	return link, ok && link.Text != "" && link.URL != ""
}

var FIELD_LINK_KEYS = func() []string {
	keys := []string{}

	for _, preset := range STORAGE_PRESETS {
		for key := range preset.PresetFieldLinks {
			if !slices.Contains(keys, key) {
				keys = append(keys, key)
			}
		}
	}

	slices.Sort(keys)

	return keys
}()

func (sc *StorageConfig) styleValue(styleKey string) int {
	return int(reflect.ValueOf(sc).Elem().Field(storageFieldByJSON[styleKey]).Int())
}

var STYLE_DEPENDENT_FIELDS = map[string][]string{
	"token_style":       {"token"},
	"ping_style":        {"ping_path", "ping_probe_id"},
	"live_style":        {"live_path"},
	"open_replay_style": {"open_replay_origin", "open_replay_path"},
	"rename_style":      {"rename_path"},
}

func (sc *StorageConfig) IsLocked(jsonKey string) bool {
	return slices.Contains(sc.PresetLockedFields, jsonKey) || slices.Contains(sc.LockedFields, jsonKey)
}

func (sc *StorageConfig) IsStyleOff(styleKey string) bool {
	return sc.styleValue(styleKey) == 0
}

func (sc *StorageConfig) copyFields(src *StorageConfig, jsonKeys []string) {
	dst, from := reflect.ValueOf(sc).Elem(), reflect.ValueOf(src).Elem()

	for _, key := range jsonKeys {
		if i, ok := storageFieldByJSON[key]; ok {
			dst.Field(i).Set(from.Field(i))
		}
	}
}

var STORAGE_PRESETS = []*StorageConfig{
	ROCKY_STORAGE,
	BALLCHASING_STORAGE,
	BLAST_STORAGE,
	REPLAYLABS_STORAGE,
	FILE_SYSTEM_STORAGE,
}

var STORAGE_PRESET = func() map[string]*StorageConfig {
	presets := make(map[string]*StorageConfig, len(STORAGE_PRESETS))

	for _, preset := range STORAGE_PRESETS {
		presets[preset.Name] = preset
	}

	return presets
}()

var ROCKY_STORAGE = &StorageConfig{
	Name:                ROCKY_NAME,
	HelpText:            "Want to see the website? Click here",
	HelpURL:             "https://lexore.ca/rocky",
	URL:                 "https://lexore.ca/rocky/api",
	IsPredefined:        true,
	IsTemporary:         false,
	PresetLockedFields:  slices.Concat(GLOBAL_LOCKED_FIELDS, []string{"replay_path", "template_file", "rename_style", "uri_params", "ping_style"}),
	Enabled:             true,
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
	OpenReplayStyle:     OpenReplayEnabled,
	OpenReplayOrigin:    "https://lexore.ca",
	OpenReplayPath:      "/replays/download/" + OpenReplayIDPlaceholder,
	RenameStyle:         RenameDisabled,
	PlaylistFilterStyle: PlaylistFilterFollowAny,
}

var BALLCHASING_STORAGE = &StorageConfig{
	Name:               BALLCHASING_NAME,
	UploadStyle:        MultipartUpload,
	ReplayPath:         "/v2/upload",
	URL:                "https://ballchasing.com/api",
	IsPredefined:       true,
	IsTemporary:        false,
	PresetLockedFields: slices.Concat(GLOBAL_LOCKED_FIELDS, []string{"replay_path", "live_style"}),
	PresetAllowedValues: map[string][]int{
		"ping_style":   allowed(PingDisabled, PingRequiresOK),
		"rename_style": allowed(RenameDisabled, RenameTitleField),
	},
	PresetFieldLinks: map[string]FieldLink{
		"open_replay_style": {Text: "Turn on in ballchasing.com", URL: "https://ballchasing.com/play-in-rocket-league?port=" + WebsiteBridgePortPlaceholder},
	},
	Enabled:          false,
	URIParams:        map[string]string{"visibility": "public"},
	TokenStyle:       RawToken,
	Token:            "",
	PingStyle:        PingRequiresOK,
	PingPath:         "/",
	TemplateName:     "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	LiveStyle:        LiveDisabled,
	LivePath:         "",
	OpenReplayStyle:  OpenReplayEnabled,
	OpenReplayOrigin: "https://ballchasing.com",
	OpenReplayPath:   "/replays/" + OpenReplayIDPlaceholder + "/file",
	RenameStyle:      RenameTitleField,
	RenamePath:       "",
}

var BLAST_STORAGE = &StorageConfig{
	Name:                BLAST_NAME,
	HelpText:            "Need a token? Click here",
	HelpURL:             "https://blast.tv/profile?utm_source=rockpload&utm_medium=desktop-app&utm_campaign=upload-token",
	URL:                 "https://api.blast.tv/v1/community-stats/rl/replays",
	ReplayPath:          "/upload-session",
	IsPredefined:        true,
	IsTemporary:         false,
	PresetLockedFields:  slices.Concat(GLOBAL_LOCKED_FIELDS, []string{"replay_path", "live_style", "open_replay_style", "rename_style"}),
	PresetAllowedValues: map[string][]int{"ping_style": allowed(PingDisabled, PingNotFoundIsValid)},
	Enabled:             false,
	UploadStyle:         PresignedSessionUpload,
	PingStyle:           PingNotFoundIsValid,
	TokenStyle:          BearerToken,
	URIParams:           map[string]string{},
	Token:               "",
	PingPath:            "/",
	PingProbeID:         "00000000-0000-0000-0000-000000000000",
	TemplateName:        "",
	LiveStyle:           LiveDisabled,
	LivePath:            "",
	RenameStyle:         RenameDisabled,
}

var REPLAYLABS_STORAGE = &StorageConfig{
	Name:                REPLAYLABS_NAME,
	HelpText:            "Need a token? Click here",
	HelpURL:             "https://replaylabs.app/upload",
	UploadStyle:         MultipartUpload,
	ReplayPath:          "/replays",
	URL:                 "https://api.replaylabs.app/api/uploader/v1",
	IsPredefined:        true,
	IsTemporary:         false,
	PresetLockedFields:  slices.Concat(GLOBAL_LOCKED_FIELDS, []string{"replay_path", "live_style", "open_replay_style", "rename_style"}),
	PresetAllowedValues: map[string][]int{"ping_style": allowed(PingDisabled, PingRequiresOK)},
	Enabled:             false,
	URIParams:           map[string]string{},
	TokenStyle:          BearerToken,
	Token:               "",
	PingStyle:           PingRequiresOK,
	PingPath:            "/ping",
	TemplateName:        "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	LiveStyle:           LiveDisabled,
	LivePath:            "",
	RenameStyle:         RenameDisabled,
	RenamePath:          "",
}

var FILE_SYSTEM_STORAGE = &StorageConfig{
	Name:               FILE_SYSTEM_NAME,
	UploadStyle:        LocalFileCopy,
	ReplayPath:         "",
	TemplateName:       "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	URL:                "",
	IsPredefined:       true,
	IsTemporary:        false,
	PresetLockedFields: slices.Concat(GLOBAL_LOCKED_FIELDS, []string{"live_style", "open_replay_style", "rename_style"}),
	Enabled:            false,
	URIParams:          map[string]string{},
	TokenStyle:         NoToken,
	Token:              "",
	PingStyle:          PingDisabled,
	PingPath:           "/",
	LiveStyle:          LiveDisabled,
	LivePath:           "",
	RenameStyle:        RenameDisabled,
}

// Dev testing storage only
var LOCALHOST_STORAGE = &StorageConfig{
	Name:             LOCALHOST_NAME,
	UploadStyle:      MultipartUpload,
	ReplayPath:       "/upload",
	TemplateName:     "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}",
	URL:              "http://localhost:4200/api",
	IsPredefined:     false,
	IsTemporary:      true,
	Enabled:          false,
	URIParams:        map[string]string{},
	TokenStyle:       NoToken,
	Token:            "",
	PingStyle:        PingDisabled,
	PingPath:         "/",
	LiveStyle:        LiveDisabled,
	LivePath:         "",
	OpenReplayStyle:  OpenReplayEnabled,
	OpenReplayOrigin: "http://localhost:4200",
	OpenReplayPath:   "/replays/download/" + OpenReplayIDPlaceholder,
	RenameStyle:      RenameDisabled,
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

	for _, key := range sc.LockedFields {
		if _, ok := storageFieldByJSON[key]; !ok {
			logger.Rlogger.Warn("Unknown locked field in storage settings", slog.String("storage", name), slog.String("field", key))
		}
	}

	if preset, ok := STORAGE_PRESET[name]; ok {
		var missing []string
		for key := range storageFieldByJSON {
			if _, present := jsonMap[key]; !present {
				missing = append(missing, key)
			}
		}
		sc.copyFields(preset, missing)

		reconcilePreset(sc, preset)
	}

	return nil
}
