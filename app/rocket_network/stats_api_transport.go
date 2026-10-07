package rocket_network

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/rlgame"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type StatsAPITransport string

const (
	TransportTCP       StatsAPITransport = "TCP"
	TransportWebSocket StatsAPITransport = "WebSocket"
)

var errAllSocketsDisabled = errors.New("both StatsAPI sockets are disabled")

// statsAPIConn reads the game events as one continuous JSON stream and sends commands to the game.
type statsAPIConn interface {
	io.ReadCloser
	Send(payload []byte) error
}

// connectStatsAPI opens the WebSocket, falling back to the TCP socket when the WebSocket is disabled or unreachable
// (older game versions have no WebPort). Only one is used at a time so events are not received twice.
func connectStatsAPI(ports rlgame.StatsAPIPorts) (statsAPIConn, StatsAPITransport, error) {
	logger.FuncDebug()

	err := errAllSocketsDisabled

	if ports.Web != 0 {
		var stream statsAPIConn
		stream, err = dialWebSocket(ports.Web)
		if err == nil {
			return stream, TransportWebSocket, nil
		}
	}

	if ports.TCP != 0 {
		var conn net.Conn
		conn, err = net.DialTimeout("tcp", localAddress(ports.TCP), dialTimeout)
		if err == nil {
			return &tcpConn{Conn: conn}, TransportTCP, nil
		}
	}

	return nil, "", err
}

// tcpConn only receives events, the game reads commands on the WebSocket.
type tcpConn struct {
	net.Conn
}

func (c *tcpConn) Send(payload []byte) error {
	logger.FuncDebug()

	return ErrCommandsNeedWebSocket
}

func localAddress(port int) string {
	return net.JoinHostPort("localhost", strconv.Itoa(port))
}

// webSocketStream exposes the WebSocket messages as one continuous JSON stream, like the TCP socket.
type webSocketStream struct {
	*io.PipeReader
	conn    net.Conn
	writeMu *sync.Mutex
}

func (w *webSocketStream) Close() error {
	w.PipeReader.Close()
	return w.conn.Close()
}

func (w *webSocketStream) Send(payload []byte) error {
	logger.FuncDebug()

	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	w.conn.SetWriteDeadline(time.Now().Add(writeTimeout))

	return wsutil.WriteClientText(w.conn, payload)
}

// lockedConn shares the write lock with Send, so pong and close answers are never mixed with a command frame
type lockedConn struct {
	net.Conn
	reader  io.Reader
	writeMu *sync.Mutex
}

func (c *lockedConn) Read(payload []byte) (int, error) {
	return c.reader.Read(payload)
}

func (c *lockedConn) Write(payload []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return c.Conn.Write(payload)
}

func dialWebSocket(port int) (statsAPIConn, error) {
	logger.FuncDebug()

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	conn, bufferedReader, _, err := ws.Dial(ctx, "ws://"+localAddress(port))
	if err != nil {
		return nil, err
	}

	// Frames received with the handshake response are kept in bufferedReader
	var connReader io.Reader = conn
	if bufferedReader != nil {
		connReader = bufferedReader
	}

	reader, writer := io.Pipe()
	writeMu := &sync.Mutex{}
	controlConn := &lockedConn{Conn: conn, reader: connReader, writeMu: writeMu}

	go func() {
		for {
			// Ping and close frames are answered by ReadServerData
			data, opCode, err := wsutil.ReadServerData(controlConn)
			if err != nil {
				var closed wsutil.ClosedError
				if errors.As(err, &closed) {
					writer.Close()
				} else {
					writer.CloseWithError(err)
				}
				return
			}

			if opCode != ws.OpText && opCode != ws.OpBinary {
				continue
			}

			if _, err := writer.Write(data); err != nil {
				return
			}
		}
	}()

	return &webSocketStream{PipeReader: reader, conn: conn, writeMu: writeMu}, nil
}
