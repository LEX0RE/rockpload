package upload

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/dank/rlapi"
)

const DefaultReplayNameTemplate = "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}"

var replayUploadSequenceIndex = -1

type ReplayUpload struct {
	PlayerName string
	PlayerID   string
	Replay     rlapi.MatchEntry
}

func replayUploadFileName(filePath string, template string, replayUpload ReplayUpload, knownPlaylists []config.PlaylistFilterEntry) string {
	if strings.TrimSpace(template) == "" {
		return filepath.Base(filePath)
	}

	replayUploadSequenceIndex += 1

	replayTime := time.Now()
	if replayUpload.Replay.Match.RecordStartTimestamp > 0 {
		replayTime = timeFromReplayTimestamp(replayUpload.Replay.Match.RecordStartTimestamp)
	}

	win := replayUpload.PlayerWon()
	winLoss := ""
	wl := ""

	switch win {
	case 1:
		winLoss = "Win"
		wl = "W"
	case 0:
		winLoss = "Loss"
		wl = "L"
	}

	replacements := map[string]string{
		"{PLAYER}":  replayUpload.PlayerName,
		"{MODE}":    config.PlaylistDisplayName(replayUpload.Replay.Match.Playlist, knownPlaylists),
		"{NUM}":     strconv.Itoa(replayUploadSequenceIndex),
		"{YEAR}":    strconv.Itoa(replayTime.Year()),
		"{MONTH}":   fmt.Sprintf("%02d", replayTime.Month()),
		"{DAY}":     fmt.Sprintf("%02d", replayTime.Day()),
		"{HOUR}":    fmt.Sprintf("%02d", replayTime.Hour()),
		"{MIN}":     fmt.Sprintf("%02d", replayTime.Minute()),
		"{WL}":      wl,
		"{WINLOSS}": winLoss,
		"{GUID}":    replayUpload.Replay.Match.MatchGUID,
	}

	fileName := template
	for token, value := range replacements {
		fileName = strings.ReplaceAll(fileName, token, value)
	}

	fileName = sanitizeFileName(fileName)
	ext := filepath.Ext(filePath)
	if ext == "" {
		ext = ".replay"
	}
	if !strings.EqualFold(filepath.Ext(fileName), ext) {
		fileName += ext
	}

	return fileName
}

func timeFromReplayTimestamp(timestamp int64) time.Time {
	switch {
	case timestamp > 1_000_000_000_000_000:
		return time.Unix(0, timestamp)
	case timestamp > 1_000_000_000_000:
		return time.UnixMilli(timestamp)
	default:
		return time.Unix(timestamp, 0)
	}
}

func (ru ReplayUpload) PlayerWon() int {
	if ru.Replay.Match.WinningTeam == -1 {
		return -1
	}

	for _, player := range ru.Replay.Match.Players {
		if player.PlayerID == ru.PlayerID || player.PlayerName == ru.PlayerName {
			if player.LastTeam == ru.Replay.Match.WinningTeam {
				return 1
			} else {
				return 0
			}
		}
	}

	return -1
}

func sanitizeFileName(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	fileName = strings.ReplaceAll(fileName, "/", "-")
	fileName = strings.ReplaceAll(fileName, "\\", "-")
	fileName = regexp.MustCompile(`[<>:"|?*\x00-\x1F]`).ReplaceAllString(fileName, "-")
	fileName = regexp.MustCompile(`\s+`).ReplaceAllString(fileName, " ")
	fileName = strings.Trim(fileName, ". ")

	if fileName == "" {
		return "replay"
	}

	return fileName
}
