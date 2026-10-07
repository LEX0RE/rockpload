package rocket_network

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/LEX0RE/rockpload/app/rlgame"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func TestMain(m *testing.M) {
	logger.Rlogger = slog.New(slog.DiscardHandler)
	os.Exit(m.Run())
}

func TestCommandJSON(t *testing.T) {
	mustCommand := func(command StatsAPICommand, err error) StatsAPICommand {
		t.Helper()

		if err != nil {
			t.Fatal(err)
		}

		return command
	}

	tests := []struct {
		name     string
		command  StatsAPICommand
		expected string
	}{
		{"change pov", mustCommand(NewChangePOVCommand(FocusBall, PerspectiveFly)), `{"Command":"ChangePOV","Data":{"Focus":"Ball","Perspective":"Fly"}}`},
		{"change pov focus only", mustCommand(NewChangePOVCommand("3", "")), `{"Command":"ChangePOV","Data":{"Focus":"3"}}`},
		{"load replay", mustCommand(NewLoadReplayCommand("ABC.replay", "")), `{"Command":"LoadReplay","Data":{"FileName":"ABC.replay"}}`},
		{"seek frame", mustCommand(NewSeekReplayFrameCommand(0)), `{"Command":"SeekReplay","Data":{"Frame":0}}`},
		{"seek time", mustCommand(NewSeekReplayTimeCommand(12.5)), `{"Command":"SeekReplay","Data":{"TimeSeconds":12.5}}`},
		{"game speed", mustCommand(NewSetGameSpeedCommand(0.5)), `{"Command":"SetGameSpeed","Data":{"Speed":0.5}}`},
		{"hud", NewSetHUDVisibilityCommand(false), `{"Command":"SetHUDVisibility","Data":{"bVisible":false}}`},
		{"pause", NewSetMatchPausedCommand(true), `{"Command":"SetMatchPaused","Data":{"bPaused":true}}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, err := json.Marshal(test.command)
			if err != nil {
				t.Fatal(err)
			}

			if string(payload) != test.expected {
				t.Errorf("got %s, expected %s", payload, test.expected)
			}
		})
	}
}

func TestCommandValidation(t *testing.T) {
	invalid := map[string]error{}

	_, invalid["pov empty"] = NewChangePOVCommand(" ", "")
	_, invalid["pov perspective"] = NewChangePOVCommand(FocusBall, "Drone")
	_, invalid["replay empty"] = NewLoadReplayCommand("", "")
	_, invalid["seek frame"] = NewSeekReplayFrameCommand(-1)
	_, invalid["seek time"] = NewSeekReplayTimeCommand(-1)
	_, invalid["speed"] = NewSetGameSpeedCommand(-0.5)

	for name, err := range invalid {
		if err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestSendCommandNotConnected(t *testing.T) {
	if err := NewStatsAPI().SendCommand(NewSetMatchPausedCommand(true)); !errors.Is(err, ErrStatsAPINotConnected) {
		t.Errorf("expected ErrStatsAPINotConnected, got %v", err)
	}
}

func startWebSocketCommandServer(t *testing.T, received chan<- []byte) int {
	t.Helper()

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			return
		}
		defer conn.Close()

		// A ping must be answered without being mixed with the command frame
		wsutil.WriteServerMessage(conn, ws.OpPing, nil)

		for {
			data, opCode, err := wsutil.ReadClientData(conn)
			if err != nil {
				return
			}

			if opCode == ws.OpText {
				received <- data
				return
			}
		}
	})}

	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })

	return listenPort(t, listener)
}

func sendAndCheck(t *testing.T, ports rlgame.StatsAPIPorts, received <-chan []byte, expected StatsAPITransport) {
	t.Helper()

	conn, transport, err := connectStatsAPI(ports)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Keep reading like the listener does, so control frames are handled
	go io.Copy(io.Discard, conn)

	statsAPI := NewStatsAPI()
	statsAPI.setConn(conn, transport)

	if statsAPI.Transport() != expected {
		t.Errorf("expected transport %s, got %s", expected, statsAPI.Transport())
	}

	command := NewSetMatchPausedCommand(true)
	if err := statsAPI.SendCommand(command); err != nil {
		t.Fatal(err)
	}

	select {
	case payload := <-received:
		expectedPayload, _ := json.Marshal(command)
		if string(payload) != string(expectedPayload) {
			t.Errorf("game received %s, expected %s", payload, expectedPayload)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the command was not received")
	}
}

func TestSendCommandRefusedOnTCP(t *testing.T) {
	conn, transport, err := connectStatsAPI(rlgame.StatsAPIPorts{TCP: startTCPServer(t)})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	statsAPI := NewStatsAPI()
	statsAPI.setConn(conn, transport)

	if err := statsAPI.SendCommand(NewSetMatchPausedCommand(true)); !errors.Is(err, ErrCommandsNeedWebSocket) {
		t.Errorf("expected ErrCommandsNeedWebSocket, got %v", err)
	}
}

func TestSendCommandWebSocket(t *testing.T) {
	received := make(chan []byte, 1)
	sendAndCheck(t, rlgame.StatsAPIPorts{Web: startWebSocketCommandServer(t, received)}, received, TransportWebSocket)
}
