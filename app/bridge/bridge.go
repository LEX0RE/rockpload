package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/rlgame"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	AppName = "rockpload"

	FallbackPorts = 2

	maxRequestBody    = 1 << 10
	maxReplaySize     = 50 << 20
	downloadTimeout   = 60 * time.Second
	shutdownTimeout   = 2 * time.Second
	readHeaderTimeout = 5 * time.Second
	preflightMaxAge   = "600"
)

type ErrorCode string

const (
	ErrorDisabled             ErrorCode = "disabled"
	ErrorInvalidRequest       ErrorCode = "invalid_request"
	ErrorStatsAPIDisconnect   ErrorCode = "statsapi_disconnected"
	ErrorStatsAPINeedsWebPort ErrorCode = "statsapi_needs_websocket"
	ErrorBusy                 ErrorCode = "busy"
	ErrorReplayNotFound       ErrorCode = "replay_not_found"
	ErrorRateLimited          ErrorCode = "rate_limited"
	ErrorDownloadFailed       ErrorCode = "download_failed"
	ErrorSaveFailed           ErrorCode = "save_failed"
	ErrorCommandFailed        ErrorCode = "command_failed"
)

var replayFileIDRegexp = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type Status struct {
	App               string `json:"app"`
	Version           string `json:"version"`
	Enabled           bool   `json:"enabled"`
	StatsAPIConnected bool   `json:"statsApiConnected"`
	Transport         string `json:"transport"`
	Ready             bool   `json:"ready"`
}

type playRequest struct {
	ReplayFileID string `json:"replayFileId"`
}

type errorResponse struct {
	Error   ErrorCode `json:"error"`
	Message string    `json:"message"`
}

type Bridge struct {
	version   string
	statsAPI  *rocket_network.StatsAPI
	isEnabled func() bool
	targets   func() map[string]config.OpenReplayTarget

	httpClient *http.Client
	busy       atomic.Bool

	serverMu      sync.Mutex
	server        *http.Server
	port          int
	requestedPort int
	listenErr     error
}

func New(version string, statsAPI *rocket_network.StatsAPI, isEnabled func() bool, targets func() map[string]config.OpenReplayTarget) *Bridge {
	logger.FuncDebug()

	return &Bridge{
		version:    version,
		statsAPI:   statsAPI,
		isEnabled:  isEnabled,
		targets:    targets,
		httpClient: &http.Client{Timeout: downloadTimeout},
	}
}

func (b *Bridge) Start(port int) error {
	logger.FuncDebug()

	b.serverMu.Lock()
	running := b.server != nil && b.requestedPort == port
	b.serverMu.Unlock()

	if running {
		return nil
	}

	b.Stop()

	listener, err := listen(port)
	if err != nil {
		logger.Rlogger.Warn("All website bridge ports are in use, opening replays from websites is disabled", slog.Int("Port", port), slog.Int("LastPort", port+FallbackPorts), slog.Any("err", err))

		b.serverMu.Lock()
		b.requestedPort = port
		b.listenErr = err
		b.serverMu.Unlock()

		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", b.handleStatus)
	mux.HandleFunc("POST /replays/play", b.handlePlay)

	actualPort := listener.Addr().(*net.TCPAddr).Port
	server := &http.Server{Handler: b.guard(actualPort, mux), ReadHeaderTimeout: readHeaderTimeout}

	b.serverMu.Lock()
	b.server = server
	b.port = actualPort
	b.requestedPort = port
	b.listenErr = nil
	b.serverMu.Unlock()

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Rlogger.Error("Website bridge stopped", slog.Any("err", err))
		}
	}()

	logger.Rlogger.Info("Website bridge listening", slog.Int("Port", actualPort))

	return nil
}

func listen(port int) (net.Listener, error) {
	var err error

	for offset := 0; offset <= FallbackPorts; offset++ {
		var listener net.Listener
		listener, err = net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port+offset)))
		if err == nil {
			return listener, nil
		}
	}

	return nil, err
}

func (b *Bridge) Stop() {
	logger.FuncDebug()

	b.serverMu.Lock()
	server := b.server
	b.server = nil
	b.port = 0
	b.serverMu.Unlock()

	if server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	server.Shutdown(ctx)
}

func (b *Bridge) StatusText() (string, bool) {
	b.serverMu.Lock()
	defer b.serverMu.Unlock()

	switch {
	case b.server != nil:
		return fmt.Sprintf("Websites can reach Rockpload on port %d.", b.port), false
	case b.listenErr != nil:
		return fmt.Sprintf("Ports %d to %d are all in use, websites cannot open replays. Choose another port.", b.requestedPort, b.requestedPort+FallbackPorts), true
	}

	return "", false
}

func (b *Bridge) guard(port int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLocalHostHeader(r.Host, port) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		origin := r.Header.Get("Origin")
		if _, ok := b.targets()[origin]; !ok {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
			w.Header().Set("Access-Control-Max-Age", preflightMaxAge)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (b *Bridge) status() Status {
	transport := b.statsAPI.Transport()
	enabled := b.isEnabled()

	return Status{
		App:               AppName,
		Version:           b.version,
		Enabled:           enabled,
		StatsAPIConnected: transport != "",
		Transport:         string(transport),
		Ready:             enabled && transport == rocket_network.TransportWebSocket,
	}
}

func (b *Bridge) handleStatus(w http.ResponseWriter, r *http.Request) {
	logger.FuncDebug()

	writeJSON(w, http.StatusOK, b.status())
}

func (b *Bridge) handlePlay(w http.ResponseWriter, r *http.Request) {
	logger.FuncDebug()

	if !b.isEnabled() {
		writeError(w, http.StatusForbidden, ErrorDisabled, "Opening replays from websites is disabled in Rockpload")
		return
	}

	var request playRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRequestBody)).Decode(&request); err != nil || !replayFileIDRegexp.MatchString(request.ReplayFileID) {
		writeError(w, http.StatusBadRequest, ErrorInvalidRequest, "Invalid replay id")
		return
	}

	if playErr := b.readyError(); playErr != nil {
		writeError(w, playErr.status, playErr.code, playErr.err.Error())
		return
	}

	target, ok := b.targets()[r.Header.Get("Origin")]
	if !ok {
		writeError(w, http.StatusForbidden, ErrorInvalidRequest, "This website is not allowed to open replays")
		return
	}

	downloadURL := strings.ReplaceAll(target.DownloadURL, config.OpenReplayIDPlaceholder, url.PathEscape(request.ReplayFileID))

	downloadRequest, err := http.NewRequestWithContext(r.Context(), http.MethodGet, downloadURL, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrorDownloadFailed, err.Error())
		return
	}

	if target.Authorization != "" {
		downloadRequest.Header.Set("Authorization", target.Authorization)
	}

	if playErr := b.openReplay(downloadRequest, request.ReplayFileID); playErr != nil {
		if playErr.retryAfter != "" {
			w.Header().Set("Retry-After", playErr.retryAfter)
			w.Header().Set("Access-Control-Expose-Headers", "Retry-After")
		}

		writeError(w, playErr.status, playErr.code, playErr.err.Error())
		return
	}

	writeJSON(w, http.StatusOK, b.status())
}

type playError struct {
	status     int
	code       ErrorCode
	err        error
	retryAfter string
}

func (b *Bridge) readyError() *playError {
	switch b.statsAPI.Transport() {
	case "":
		return &playError{status: http.StatusConflict, code: ErrorStatsAPIDisconnect, err: errors.New("rocket league is not running or its StatsAPI is disabled")}
	case rocket_network.TransportTCP:
		return &playError{status: http.StatusConflict, code: ErrorStatsAPINeedsWebPort, err: errors.New("the StatsAPI WebPort is disabled in Rocket League")}
	}

	return nil
}

func (b *Bridge) openReplay(downloadRequest *http.Request, replayID string) *playError {
	logger.FuncDebug()

	if !b.busy.CompareAndSwap(false, true) {
		return &playError{status: http.StatusTooManyRequests, code: ErrorBusy, err: errors.New("a replay is already being opened")}
	}
	defer b.busy.Store(false)

	replayPath, playErr := b.downloadReplay(downloadRequest, replayID)
	if playErr != nil {
		logger.Rlogger.Error("Unable to download replay for the website", slog.String("ReplayID", replayID), slog.Any("err", playErr.err))
		return playErr
	}

	command, err := rocket_network.NewLoadReplayCommand("", rlgame.GamePath(replayPath))
	if err == nil {
		err = b.statsAPI.SendCommand(command)
	}

	if err != nil {
		return &playError{status: http.StatusBadGateway, code: ErrorCommandFailed, err: err}
	}

	logger.Rlogger.Info("Replay sent to Rocket League", slog.String("ReplayID", replayID))

	return nil
}

func (b *Bridge) downloadReplay(request *http.Request, replayID string) (string, *playError) {
	logger.FuncDebug()

	request.Header.Set("User-Agent", AppName+"/"+b.version)

	response, err := b.httpClient.Do(request)
	if err != nil {
		return "", &playError{status: http.StatusBadGateway, code: ErrorDownloadFailed, err: err}
	}
	defer response.Body.Close()

	switch {
	case response.StatusCode == http.StatusNotFound:
		return "", &playError{status: http.StatusNotFound, code: ErrorReplayNotFound, err: errors.New("replay file not found")}
	case response.StatusCode == http.StatusTooManyRequests:
		return "", &playError{status: http.StatusTooManyRequests, code: ErrorRateLimited, err: errors.New("too many downloads"), retryAfter: response.Header.Get("Retry-After")}
	case response.StatusCode != http.StatusOK:
		return "", &playError{status: http.StatusBadGateway, code: ErrorDownloadFailed, err: fmt.Errorf("download failed: %s", response.Status)}
	case response.ContentLength > maxReplaySize:
		return "", &playError{status: http.StatusBadGateway, code: ErrorDownloadFailed, err: errReplayTooLarge}
	}

	replayPath, err := rlgame.SaveTempReplay(replayID+".replay", http.MaxBytesReader(nil, response.Body, maxReplaySize))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return "", &playError{status: http.StatusBadGateway, code: ErrorDownloadFailed, err: errReplayTooLarge}
		}
		return "", &playError{status: http.StatusInternalServerError, code: ErrorSaveFailed, err: err}
	}

	return replayPath, nil
}

var errReplayTooLarge = errors.New("replay file is too large")

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code ErrorCode, message string) {
	writeJSON(w, status, errorResponse{Error: code, Message: message})
}

func isLocalHostHeader(host string, port int) bool {
	return host == net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) || host == net.JoinHostPort("localhost", strconv.Itoa(port))
}
