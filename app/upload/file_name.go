package upload

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dank/rlapi"
)

const DefaultReplayNameTemplate = "{YEAR}-{MONTH}-{DAY}.{HOUR}.{MIN} {PLAYER} {MODE} {WINLOSS}"

type ReplayUpload struct {
	PlayerName string
	PlayerID   string
	Replay     rlapi.MatchEntry
	Number     int
}

func replayUploadFileName(filePath string, template string, replayUpload ReplayUpload) string {
	if strings.TrimSpace(template) == "" {
		return filepath.Base(filePath)
	}

	replayTime := time.Now()
	if replayUpload.Replay.Match.RecordStartTimestamp > 0 {
		replayTime = timeFromReplayTimestamp(replayUpload.Replay.Match.RecordStartTimestamp)
	}

	win := replayUpload.playerWon()
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
		"{MODE}":    playlistName(replayUpload.Replay.Match.Playlist),
		"{NUM}":     strconv.Itoa(replayUpload.Number),
		"{YEAR}":    strconv.Itoa(replayTime.Year()),
		"{MONTH}":   fmt.Sprintf("%02d", replayTime.Month()),
		"{DAY}":     fmt.Sprintf("%02d", replayTime.Day()),
		"{HOUR}":    fmt.Sprintf("%02d", replayTime.Hour()),
		"{MIN}":     fmt.Sprintf("%02d", replayTime.Minute()),
		"{WL}":      wl,
		"{WINLOSS}": winLoss,
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

func (ru ReplayUpload) playerWon() int {
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

// Based from https://bakkesplugins.com/wiki/bakkesmod-sdk/enums/playlistids
func playlistName(playlist int) string {
	switch playlist {
	case -1337:
		return "Unknown"
	case 0:
		return "Casual"
	case 1:
		return "Casual Duel"
	case 2:
		return "Casual Doubles"
	case 3:
		return "Casual Standard"
	case 4:
		return "Casual Chaos"
	case 6:
		return "Private"
	case 7:
		return "Season"
	case 8:
		return "OfflineSplitscreen"
	case 9:
		return "Training"
	case 10:
		return "Ranked Duel"
	case 11:
		return "Ranked Doubles"
	case 13:
		return "Ranked Standard"
	case 15:
		return "Snow Day"
	case 16:
		return "Experimental"
	case 17:
		return "Hoops"
	case 18:
		return "Rumble"
	case 19:
		return "Workshop"
	case 20:
		return "UGC Training Editor"
	case 21:
		return "UGC Training"
	case 22:
		return "Tournament"
	case 23:
		return "Dropshot"
	case 25:
		return "10th Anniversary"
	case 26:
		return "Face It"
	case 27:
		return "Ranked Hoops"
	case 28:
		return "Ranked Rumble"
	case 29:
		return "Ranked Dropshot"
	case 30:
		return "Ranked Snow Day"
	case 31:
		return "Haunted Ball"
	case 32:
		return "Beach Ball"
	case 33:
		return "Rugby"
	case 34:
		return "Auto Tournament"
	case 35:
		return "Rocket Labs"
	case 37:
		return "Rum Shot"
	case 38:
		return "God Ball"
	case 40:
		return "Coop Vs AI"
	case 41:
		return "Boomer Ball"
	case 43:
		return "God Ball Doubles"
	case 44:
		return "Special Snow Day"
	case 46:
		return "Football"
	case 47:
		return "Cubic"
	case 48:
		return "Tactical Rumble"
	case 49:
		return "Spring Loaded"
	case 50:
		return "Speed Demon"
	case 52:
		return "Rumble BM"
	case 54:
		return "Knockout"
	case 55:
		return "Thirdwheel"
	case 62:
		return "Magnus Futball"
	default:
		return "Playlist " + strconv.Itoa(playlist)
	}
}
