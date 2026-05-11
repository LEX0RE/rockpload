package logger

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"strconv"
)

type AppFilterHandler struct {
	slog.Handler
}

var Rlogger *slog.Logger
var lCounter = 0

func SetLogger() {
	level := os.Getenv("SLOG_LEVEL")

	slogOpts := &slog.HandlerOptions{}
	if level == "debug" {
		slogOpts = &slog.HandlerOptions{Level: slog.LevelDebug}
	}

	baseHandler := slog.NewTextHandler(os.Stdout, slogOpts)

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
