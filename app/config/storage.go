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
		temp = append(temp, &fresh)
	} else {
		reconcilePreset(temp[ballchasingIndex], BALLCHASING_STORAGE)
	}

	blastIndex := slices.IndexFunc(temp, func(c *StorageConfig) bool { return c.Name == BLAST_NAME })
	if blastIndex == -1 {
		fresh := *BLAST_STORAGE
		temp = append(temp, &fresh)
	} else {
		reconcilePreset(temp[blastIndex], BLAST_STORAGE)
	}

	fileSystemIndex := slices.IndexFunc(temp, func(c *StorageConfig) bool { return c.Name == FILE_SYSTEM_NAME })
	if fileSystemIndex == -1 {
		fresh := *FILE_SYSTEM_STORAGE
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

	sc.UploadStyle = preset.UploadStyle

	if sc.PingStyle != PingDisabled && sc.PingStyle != preset.PingStyle {
		sc.PingStyle = preset.PingStyle
	}

	if sc.RenameStyle != RenameDisabled && sc.RenameStyle != preset.RenameStyle {
		sc.RenameStyle = preset.RenameStyle
	}
}

type UploadStyleType int

const (
	LocalFileCopy UploadStyleType = iota
	MultipartUpload
	PresignedSessionUpload
)

var UploadStyleLabels = [...]string{
	LocalFileCopy:          "Local File Copy",
	MultipartUpload:        "Multipart Form POST",
	PresignedSessionUpload: "Presigned Session (URL + status poll)",
}

func (s UploadStyleType) Label() string {
	logger.FuncDebug()

	if int(s) < 0 || int(s) >= len(UploadStyleLabels) {
		return UploadStyleLabels[LocalFileCopy]
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

	return LocalFileCopy
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

type RenameStyleType int

const (
	RenameDisabled RenameStyleType = iota
	RenamePatchTitle
)

var RenameStyleLabels = [...]string{
	RenameDisabled:   "Rename Disabled",
	RenamePatchTitle: "Patch Title After Upload",
}

func (s RenameStyleType) Label() string {
	logger.FuncDebug()

	if int(s) < 0 || int(s) >= len(RenameStyleLabels) {
		return RenameStyleLabels[RenameDisabled]
	}

	return RenameStyleLabels[s]
}

func RenameStyleFromLabel(label string) RenameStyleType {
	logger.FuncDebug()

	for style, l := range RenameStyleLabels {
		if l == label {
			return RenameStyleType(style)
		}
	}

	return RenameDisabled
}

type PlaylistFilterStyleType int

const (
	PlaylistFilterDisabled PlaylistFilterStyleType = iota
	PlaylistFilterWhitelist
	PlaylistFilterBlacklist
	PlaylistFilterFollowAny
)

var PlaylistFilterStyleLabels = [...]string{
	PlaylistFilterDisabled:  "Upload Everything",
	PlaylistFilterWhitelist: "Only Selected Playlists",
	PlaylistFilterBlacklist: "All Except Selected Playlists",
	PlaylistFilterFollowAny: "Follow Any Other Storage",
}

func (s PlaylistFilterStyleType) Label() string {
	logger.FuncDebug()

	if int(s) < 0 || int(s) >= len(PlaylistFilterStyleLabels) {
		return PlaylistFilterStyleLabels[PlaylistFilterDisabled]
	}

	return PlaylistFilterStyleLabels[s]
}

func PlaylistFilterStyleFromLabel(label string) PlaylistFilterStyleType {
	logger.FuncDebug()

	for style, l := range PlaylistFilterStyleLabels {
		if l == label {
			return PlaylistFilterStyleType(style)
		}
	}

	return PlaylistFilterDisabled
}

// TODO Deprecated, previous storage schema
type legacyStorageConfig struct {
	Name        string `json:"name"`
	SendReplay  bool   `json:"send_replay"`
	StorageType int    `json:"storage_type"`
	NeedToken   bool   `json:"need_token"`
	SendPing    bool   `json:"send_ping"`
	SendLive    bool   `json:"send_live"`
}

// The old StorageConfigType enum: WebsiteConfig = 0, FileSystemConfig = 1.
const legacyFileSystemStorageType = 1

func (a *AppConfig) migrateStorageStyles() (bool, error) {
	logger.FuncDebug()

	data, err := os.ReadFile(constant.Paths.StorageSettingsFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	var rawEntries []json.RawMessage
	if err := json.Unmarshal(data, &rawEntries); err != nil {
		return false, nil
	}

	migrated := false
	for _, rawEntry := range rawEntries {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(rawEntry, &fields); err != nil {
			continue
		}

		if _, isCurrentSchema := fields["upload_style"]; isCurrentSchema {
			continue
		}

		var legacy legacyStorageConfig
		if err := json.Unmarshal(rawEntry, &legacy); err != nil || legacy.Name == "" {
			continue
		}

		index := slices.IndexFunc(a.StorageSettings.value, func(c *StorageConfig) bool { return c.Name == legacy.Name })
		if index == -1 {
			continue
		}

		site := a.StorageSettings.value[index]

		site.Enabled = legacy.SendReplay
		if legacy.SendReplay {
			if legacy.StorageType == legacyFileSystemStorageType {
				site.UploadStyle = LocalFileCopy
			} else {
				site.UploadStyle = MultipartUpload
			}
		} else if preset, ok := STORAGE_PRESET[legacy.Name]; ok {
			site.UploadStyle = preset.UploadStyle
		} else {
			site.UploadStyle = MultipartUpload
		}

		site.TokenStyle = NoToken
		if legacy.NeedToken {
			site.TokenStyle = RawToken
		}

		site.PingStyle = PingDisabled
		if legacy.SendPing {
			site.PingStyle = PingRequiresOK
		}

		site.LiveStyle = LiveDisabled
		if legacy.SendLive {
			site.LiveStyle = LiveEnabled
		}

		migrated = true
	}

	return migrated, nil
}

// TODO Deprecated, previous storage schema had upload_style but no Enabled field
func (a *AppConfig) migrateStorageEnabled() (bool, error) {
	logger.FuncDebug()

	data, err := os.ReadFile(constant.Paths.StorageSettingsFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	var rawEntries []json.RawMessage
	if err := json.Unmarshal(data, &rawEntries); err != nil {
		return false, nil
	}

	migrated := false
	for _, rawEntry := range rawEntries {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(rawEntry, &fields); err != nil {
			continue
		}

		if _, enabledPresent := fields["enabled"]; enabledPresent {
			continue
		}

		uploadStyleRaw, hasUploadStyle := fields["upload_style"]
		if !hasUploadStyle {
			// Predates upload_style entirely; migrateStorageStyles handles this tier.
			continue
		}

		var name string
		if raw, ok := fields["name"]; ok {
			_ = json.Unmarshal(raw, &name)
		}

		index := slices.IndexFunc(a.StorageSettings.value, func(c *StorageConfig) bool { return c.Name == name })
		if index == -1 {
			continue
		}

		var legacyUploadStyle int
		_ = json.Unmarshal(uploadStyleRaw, &legacyUploadStyle)

		site := a.StorageSettings.value[index]

		site.Enabled = legacyUploadStyle != 0

		switch legacyUploadStyle {
		case 1:
			site.UploadStyle = LocalFileCopy
		case 2:
			site.UploadStyle = MultipartUpload
		case 3:
			site.UploadStyle = PresignedSessionUpload
		default:
			if preset, ok := STORAGE_PRESET[name]; ok {
				site.UploadStyle = preset.UploadStyle
			} else {
				site.UploadStyle = MultipartUpload
			}
		}

		migrated = true
	}

	return migrated, nil
}
