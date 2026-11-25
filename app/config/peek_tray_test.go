package config

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

func withBehaviorFile(t *testing.T, content string) {
	t.Helper()

	dir := t.TempDir()
	behaviorFile := filepath.Join(dir, "behavior.json")

	if content != "" {
		if err := os.WriteFile(behaviorFile, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	previousPaths := constant.Paths
	previousLogger := logger.Rlogger
	constant.Paths = &constant.AppPaths{BehaviorSettingsFile: behaviorFile, AppLog: filepath.Join(dir, "app.log")}
	logger.Rlogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Cleanup(func() {
		constant.Paths = previousPaths
		logger.Rlogger = previousLogger
	})
}

func TestPeekStartInTray(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{name: "start in tray", content: `{"start_in_tray": true}`, want: true},
		{name: "start in tray without exit in tray", content: `{"start_in_tray": true, "exit_in_tray": false}`, want: false},
		{name: "default", content: `{}`, want: false},
		{name: "invalid file", content: `{"start_in_tray": `, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withBehaviorFile(t, tt.content)

			if got := PeekStartInTray(); got != tt.want {
				t.Fatalf("PeekStartInTray() = %v, want %v", got, tt.want)
			}
		})
	}
}
