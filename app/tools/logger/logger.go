package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/constant"
)

const (
	MAX_LOG_FILE_BACKUP = 20
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
var funcDebugEnabled = os.Getenv("FUNC_DEBUG") == "true"

func SetLogger() {
	if err := rotateOnStartup(); err != nil {
		fmt.Printf("ERROR: Code execution failed during log rotation: %v\n", err)
	}

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
	if err != nil {
		fmt.Printf("ERROR: Unable to open log file (%s): %v\n", constant.Paths.AppLog, err)
	} else {
		fmt.Fprintf(logFile, "\n\n--- App Started at %s ---\n", time.Now().Format(time.RFC3339))
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
	if !funcDebugEnabled || Rlogger == nil {
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

func rotateOnStartup() error {
	if _, err := os.Stat(constant.Paths.AppLog); os.IsNotExist(err) {
		return nil
	}

	basePath := strings.TrimSuffix(constant.Paths.AppLog, ".log")

	for i := MAX_LOG_FILE_BACKUP - 1; i >= 1; i-- {
		oldPath := basePath + "-" + strconv.Itoa(i) + ".log"
		newPath := basePath + "-" + strconv.Itoa(i+1) + ".log"

		if _, err := os.Stat(oldPath); err == nil {
			_ = os.Remove(newPath)
			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("failed to shift backup from %s to %s: %w", oldPath, newPath, err)
			}
		}
	}

	nextPath := basePath + "-1.log"
	_ = os.Remove(nextPath)
	if err := os.Rename(constant.Paths.AppLog, nextPath); err != nil {
		return fmt.Errorf("failed to shift primary log to %s: %w", nextPath, err)
	}

	return nil
}
