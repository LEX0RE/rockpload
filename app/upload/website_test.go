package upload

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

func TestMain(m *testing.M) {
	logger.Rlogger = slog.New(slog.DiscardHandler)
	os.Exit(m.Run())
}

func TestUploadTitleCutsToMaxLength(t *testing.T) {
	if got := uploadTitle("2026-11-24 Player Win.replay"); got != "2026-11-24 Player Win" {
		t.Errorf("uploadTitle() = %q", got)
	}

	long := strings.Repeat("é", maxTitleLength+10) + ".replay"
	if got := uploadTitle(long); utf8.RuneCountInString(got) != maxTitleLength || !utf8.ValidString(got) {
		t.Errorf("uploadTitle() has %d characters, want %d", utf8.RuneCountInString(got), maxTitleLength)
	}
}

func TestUploadMultipartTitleField(t *testing.T) {
	tests := []struct {
		name        string
		renameStyle config.RenameStyleType
		wantTitle   string
	}{
		{"title field", config.RenameTitleField, "My Replay"},
		{"rename disabled", config.RenameDisabled, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotTitle, gotQueryTitle string
			var gotPatch bool

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPatch {
					gotPatch = true
					return
				}

				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Errorf("ParseMultipartForm() error = %v", err)
				}

				gotTitle = r.FormValue("title")
				gotQueryTitle = r.URL.Query().Get("title")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id":"abc"}`))
			}))
			defer server.Close()

			replayPath := filepath.Join(t.TempDir(), "My Replay.replay")
			if err := os.WriteFile(replayPath, []byte("replay"), 0o600); err != nil {
				t.Fatal(err)
			}

			storage := &config.StorageConfig{
				Name:        "test",
				Enabled:     true,
				UploadStyle: config.MultipartUpload,
				URL:         server.URL,
				ReplayPath:  "/v2/upload",
				TokenStyle:  config.NoToken,
				RenameStyle: tt.renameStyle,
			}

			if err := NewWebsite(storage, "", nil).UploadReplay(replayPath, ReplayUpload{}); err != nil {
				t.Fatalf("UploadReplay() error = %v", err)
			}

			if gotTitle != tt.wantTitle {
				t.Errorf("title field = %q, want %q", gotTitle, tt.wantTitle)
			}

			if gotQueryTitle != "" {
				t.Errorf("title must not be sent as a query param, got %q", gotQueryTitle)
			}

			if gotPatch {
				t.Error("title field style must not patch the title after upload")
			}
		})
	}
}
