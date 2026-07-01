package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/constant"
)

type AppFilterHandler struct {
	slog.Handler
}

type safeMultiWriter struct {
	mu      sync.Mutex
	writers []io.Writer
}

func (s *safeMultiWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.writers {
		w.Write(p)
	}
	return len(p), nil
}

var Rlogger *slog.Logger
var lCounter = 0

func SetLogger() {
	level := os.Getenv("SLOG_LEVEL")

	slogOpts := &slog.HandlerOptions{}
	switch level {
	case "debug":
		slogOpts = &slog.HandlerOptions{Level: slog.LevelDebug}
	case "info":
		slogOpts = &slog.HandlerOptions{Level: slog.LevelInfo}
	case "warn":
		slogOpts = &slog.HandlerOptions{Level: slog.LevelWarn}
	case "error":
		slogOpts = &slog.HandlerOptions{Level: slog.LevelError}
	default:
		slogOpts = &slog.HandlerOptions{Level: slog.LevelInfo}
	}

	var writer io.Writer = os.Stdout

	logFile, err := os.OpenFile(constant.Paths.AppLog, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err == nil {
		logFile.WriteString(fmt.Sprintf("\n\n--- App Started at %s ---\n", time.Now().Format(time.RFC3339)))
		writer = &safeMultiWriter{writers: []io.Writer{os.Stdout, logFile}}

		err = redirectStderr(logFile)
		if err != nil {
			fmt.Println("Unable to redirect stderr:", err)
		}
	}

	baseHandler := slog.NewTextHandler(writer, slogOpts)

	filteredHandler := &AppFilterHandler{baseHandler}

	logger := slog.New(filteredHandler)

	slog.SetDefault(logger)

	Rlogger = logger.With("component", "rockpload")
}

func (h *AppFilterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, level)
}

func (h *AppFilterHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level == slog.LevelDebug {
		var component string

		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "component" {
				component = a.Value.String()
			}
			return true
		})

		if component != "rockpload" {
			return nil
		}
	}

	return h.Handler.Handle(ctx, r)
}

func FuncDebug() {
	if os.Getenv("FUNC_DEBUG") != "true" {
		return
	}

	pc, _, line, ok := runtime.Caller(1)
	if !ok {
		return
	}

	fn := runtime.FuncForPC(pc)
	Rlogger.Debug("[" + fn.Name() + ":" + strconv.Itoa(line) + "]")
}

func Step() {
	pc, _, line, ok := runtime.Caller(1)
	if !ok {
		return
	}

	fn := runtime.FuncForPC(pc)
	Rlogger.Debug("[" + fn.Name() + ":" + strconv.Itoa(line) + "] - Step " + strconv.Itoa(lCounter))

	lCounter++
}
