package rocket_network

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// StatsAPICommand is sent to the game on the connected StatsAPI socket.
type StatsAPICommand struct {
	Command string `json:"Command"`
	Data    any    `json:"Data"`
}

type POVPerspective string

const (
	PerspectiveFly            POVPerspective = "Fly"
	PerspectiveSoftAttach     POVPerspective = "SoftAttach"
	PerspectiveHardAttach     POVPerspective = "HardAttach"
	PerspectivePlayerView     POVPerspective = "PlayerView"
	PerspectiveAutoCam        POVPerspective = "AutoCam"
	PerspectiveCameraDirector POVPerspective = "Camera_Director"

	// FocusBall is the ChangePOV focus on the ball, a player is focused with its shortcut
	FocusBall = "Ball"
)

var perspectives = []POVPerspective{PerspectiveFly, PerspectiveSoftAttach, PerspectiveHardAttach, PerspectivePlayerView, PerspectiveAutoCam, PerspectiveCameraDirector}

var (
	ErrStatsAPINotConnected  = errors.New("not connected to the Rocket League StatsAPI")
	ErrCommandsNeedWebSocket = errors.New("commands can only be sent on the StatsAPI WebSocket, check WebPort in the StatsAPI config")
)

type changePOVData struct {
	Focus       string         `json:"Focus,omitempty"`
	Perspective POVPerspective `json:"Perspective,omitempty"`
}

type loadReplayData struct {
	FileName string `json:"FileName,omitempty"`
	Path     string `json:"Path,omitempty"`
}

type seekReplayData struct {
	Frame       *int     `json:"Frame,omitempty"`
	TimeSeconds *float64 `json:"TimeSeconds,omitempty"`
}

type setGameSpeedData struct {
	Speed float64 `json:"Speed"`
}

type setHUDVisibilityData struct {
	BVisible bool `json:"bVisible"`
}

type setMatchPausedData struct {
	BPaused bool `json:"bPaused"`
}

// NewChangePOVCommand changes the camera while spectating or in a replay.
// focus is FocusBall or a player shortcut, it can be empty when only the perspective changes.
func NewChangePOVCommand(focus string, perspective POVPerspective) (StatsAPICommand, error) {
	focus = strings.TrimSpace(focus)

	if focus == "" && perspective == "" {
		return StatsAPICommand{}, errors.New("ChangePOV needs a focus or a perspective")
	}

	if perspective != "" && !isKnownPerspective(perspective) {
		return StatsAPICommand{}, fmt.Errorf("unknown ChangePOV perspective %q", perspective)
	}

	return StatsAPICommand{Command: "ChangePOV", Data: changePOVData{Focus: focus, Perspective: perspective}}, nil
}

func isKnownPerspective(perspective POVPerspective) bool {
	for _, known := range perspectives {
		if known == perspective {
			return true
		}
	}

	return false
}

// NewLoadReplayCommand loads a replay by file name (used first by the game) or by path.
func NewLoadReplayCommand(fileName string, path string) (StatsAPICommand, error) {
	if fileName == "" && path == "" {
		return StatsAPICommand{}, errors.New("LoadReplay needs a file name or a path")
	}

	return StatsAPICommand{Command: "LoadReplay", Data: loadReplayData{FileName: fileName, Path: path}}, nil
}

func NewSeekReplayFrameCommand(frame int) (StatsAPICommand, error) {
	if frame < 0 {
		return StatsAPICommand{}, errors.New("SeekReplay frame cannot be negative")
	}

	return StatsAPICommand{Command: "SeekReplay", Data: seekReplayData{Frame: &frame}}, nil
}

func NewSeekReplayTimeCommand(timeSeconds float64) (StatsAPICommand, error) {
	if timeSeconds < 0 || math.IsNaN(timeSeconds) || math.IsInf(timeSeconds, 0) {
		return StatsAPICommand{}, errors.New("SeekReplay time must be a positive number")
	}

	return StatsAPICommand{Command: "SeekReplay", Data: seekReplayData{TimeSeconds: &timeSeconds}}, nil
}

// NewSetGameSpeedCommand changes the replay speed (1.0 is normal speed).
func NewSetGameSpeedCommand(speed float64) (StatsAPICommand, error) {
	if speed < 0 || math.IsNaN(speed) || math.IsInf(speed, 0) {
		return StatsAPICommand{}, errors.New("SetGameSpeed speed must be a positive number")
	}

	return StatsAPICommand{Command: "SetGameSpeed", Data: setGameSpeedData{Speed: speed}}, nil
}

func NewSetHUDVisibilityCommand(visible bool) StatsAPICommand {
	return StatsAPICommand{Command: "SetHUDVisibility", Data: setHUDVisibilityData{BVisible: visible}}
}

func NewSetMatchPausedCommand(paused bool) StatsAPICommand {
	return StatsAPICommand{Command: "SetMatchPaused", Data: setMatchPausedData{BPaused: paused}}
}
