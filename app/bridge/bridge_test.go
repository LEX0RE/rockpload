package bridge

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

func TestMain(m *testing.M) {
	logger.Rlogger = slog.New(slog.DiscardHandler)
	os.Exit(m.Run())
}

func TestDownloadReplayErrorCodes(t *testing.T) {
	tests := []struct {
		upstream   int
		wantCode   ErrorCode
		wantStatus int
	}{
		{http.StatusUnauthorized, ErrorUnauthorized, http.StatusUnauthorized},
		{http.StatusForbidden, ErrorUnauthorized, http.StatusUnauthorized},
		{http.StatusNotFound, ErrorReplayNotFound, http.StatusNotFound},
		{http.StatusTooManyRequests, ErrorRateLimited, http.StatusTooManyRequests},
		{http.StatusInternalServerError, ErrorDownloadFailed, http.StatusBadGateway},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.upstream), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.upstream)
			}))
			defer server.Close()

			request, err := http.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}

			b := &Bridge{httpClient: server.Client()}
			_, playErr := b.downloadReplay(request, "abc")
			if playErr == nil {
				t.Fatal("downloadReplay() error = nil")
			}

			if playErr.code != tt.wantCode || playErr.status != tt.wantStatus {
				t.Errorf("downloadReplay() = %s/%d, want %s/%d", playErr.code, playErr.status, tt.wantCode, tt.wantStatus)
			}
		})
	}
}
