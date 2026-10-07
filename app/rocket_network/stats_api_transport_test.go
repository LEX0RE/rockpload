package rocket_network

import (
	"encoding/json"
	"net"
	"net/http"
	"testing"

	"github.com/LEX0RE/rockpload/app/rlgame"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// Same envelope as the game, Data being a JSON encoded string
var statsAPIMessages = []string{
	`{"Event":"MatchCreated","Data":"{\"MatchGuid\":\"ABC\"}"}`,
	`{"Event":"CountdownBegin","Data":{"MatchGuid":"ABC"}}`,
}

func listenPort(t *testing.T, listener net.Listener) int {
	t.Helper()
	return listener.Addr().(*net.TCPAddr).Port
}

func startTCPServer(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// The TCP socket sends concatenated JSON without separator
		for _, message := range statsAPIMessages {
			conn.Write([]byte(message))
		}
	}()

	return listenPort(t, listener)
}

func startWebSocketServer(t *testing.T, messagesPerFrame int) int {
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

		for i := 0; i < len(statsAPIMessages); i += messagesPerFrame {
			frame := ""
			for j := i; j < i+messagesPerFrame && j < len(statsAPIMessages); j++ {
				frame += statsAPIMessages[j]
			}
			wsutil.WriteServerText(conn, []byte(frame))
		}

		ws.WriteFrame(conn, ws.NewCloseFrame(ws.NewCloseFrameBody(ws.StatusNormalClosure, "")))
	})}

	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })

	return listenPort(t, listener)
}

func closedPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	port := listenPort(t, listener)
	listener.Close()

	return port
}

func readAllEvents(t *testing.T, ports rlgame.StatsAPIPorts, expected StatsAPITransport) {
	t.Helper()

	stream, transport, err := connectStatsAPI(ports)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if transport != expected {
		t.Errorf("expected transport %s, got %s", expected, transport)
	}

	decoder := json.NewDecoder(stream)
	for _, message := range statsAPIMessages {
		var expectedEvent, event RLEvent
		json.Unmarshal([]byte(message), &expectedEvent)

		if err := decoder.Decode(&event); err != nil {
			t.Fatalf("decoding %s: %v", expectedEvent.Event, err)
		}

		if event.Event != expectedEvent.Event {
			t.Errorf("expected event %s, got %s", expectedEvent.Event, event.Event)
		}
	}

	var extra RLEvent
	if err := decoder.Decode(&extra); err == nil {
		t.Errorf("expected the stream to end, got %s", extra.Event)
	}
}

func TestConnectWebSocketFirst(t *testing.T) {
	ports := rlgame.StatsAPIPorts{TCP: startTCPServer(t), Web: startWebSocketServer(t, 1)}

	readAllEvents(t, ports, TransportWebSocket)
}

func TestConnectTCPWhenWebSocketDisabled(t *testing.T) {
	readAllEvents(t, rlgame.StatsAPIPorts{TCP: startTCPServer(t), Web: 0}, TransportTCP)
}

func TestConnectTCPWhenWebSocketUnreachable(t *testing.T) {
	readAllEvents(t, rlgame.StatsAPIPorts{TCP: startTCPServer(t), Web: closedPort(t)}, TransportTCP)
}

func TestConnectWebSocketSeveralMessagesPerFrame(t *testing.T) {
	readAllEvents(t, rlgame.StatsAPIPorts{Web: startWebSocketServer(t, 2)}, TransportWebSocket)
}

func TestConnectAllDisabled(t *testing.T) {
	if _, _, err := connectStatsAPI(rlgame.StatsAPIPorts{}); err == nil {
		t.Error("expected an error when both sockets are disabled")
	}
}
