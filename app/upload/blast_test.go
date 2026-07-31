package upload

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type capturedPut struct {
	contentLength    int64
	transferEncoding []string
	authorization    string
	rawQuery         string
	body             []byte
}

type capturedSession struct {
	authorization string
	body          []byte
	rawQuery      string
}

type blastFake struct {
	server  *httptest.Server
	put     capturedPut
	session capturedSession

	sessionStatus int
	putStatus     int
	statusStatus  int
	uploadStatus  string
	rejection     *string
	statusCalls   int
}

func newBlastFake() *blastFake {
	f := &blastFake{sessionStatus: 200, putStatus: 200, statusStatus: 200, uploadStatus: "processed"}

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/community-stats/rl/replays/upload-session", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.session = capturedSession{authorization: r.Header.Get("Authorization"), body: body, rawQuery: r.URL.RawQuery}

		if f.sessionStatus != 200 {
			w.WriteHeader(f.sessionStatus)
			w.Write([]byte(`{"code":"unauthorized","message":"Unauthorized."}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"fileId":    "11111111-1111-4111-8111-111111111111",
			"fileKey":   "replays/11111111.replay",
			"uploadUrl": f.server.URL + "/s3/object?X-Amz-Signature=abc&X-Amz-Expires=900",
		})
	})

	mux.HandleFunc("/s3/object", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.put = capturedPut{
			contentLength:    r.ContentLength,
			transferEncoding: r.TransferEncoding,
			authorization:    r.Header.Get("Authorization"),
			rawQuery:         r.URL.RawQuery,
			body:             body,
		}

		w.WriteHeader(f.putStatus)
	})

	mux.HandleFunc("/v1/community-stats/rl/replays/", func(w http.ResponseWriter, r *http.Request) {
		f.statusCalls++

		if f.statusStatus != 200 {
			w.WriteHeader(f.statusStatus)
			w.Write([]byte(`{"code":"replay-upload-not-found","message":"Not found."}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":              strings.TrimPrefix(r.URL.Path, "/v1/community-stats/rl/replays/"),
			"status":          f.uploadStatus,
			"rejectionReason": f.rejection,
			"createdAt":       "2026-07-31T10:00:00Z",
			"updatedAt":       nil,
		})
	})

	f.server = httptest.NewServer(mux)

	return f
}

func (f *blastFake) blast(token string) *Blast {
	storage := *config.BLAST_STORAGE
	storage.URL = f.server.URL
	storage.SendReplay = true
	storage.Token = token

	return NewBlast(&storage)
}

func writeReplay(t *testing.T, payload string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "match.replay")
	if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestBlastHappyPath(t *testing.T) {
	logger.Rlogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	f := newBlastFake()
	defer f.server.Close()

	payload := "fake replay payload"
	path := writeReplay(t, payload)

	if err := f.blast("my-token").UploadReplay(path, ReplayUpload{}); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if f.session.authorization != "Bearer my-token" {
		t.Errorf("session auth = %q", f.session.authorization)
	}
	if len(f.session.body) != 0 {
		t.Errorf("session body should be empty, got %q", f.session.body)
	}
	if f.session.rawQuery != "" {
		t.Errorf("session query should be empty, got %q", f.session.rawQuery)
	}

	if f.put.authorization != "" {
		t.Errorf("presigned PUT must not carry an Authorization header, got %q", f.put.authorization)
	}
	if f.put.rawQuery != "X-Amz-Signature=abc&X-Amz-Expires=900" {
		t.Errorf("presigned query was altered: %q", f.put.rawQuery)
	}
	if len(f.put.transferEncoding) != 0 {
		t.Errorf("PUT must not be chunked, got %v", f.put.transferEncoding)
	}
	if f.put.contentLength != int64(len(payload)) {
		t.Errorf("PUT ContentLength = %d, want %d", f.put.contentLength, len(payload))
	}
	if string(f.put.body) != payload {
		t.Errorf("PUT body = %q", f.put.body)
	}
	if f.statusCalls != 1 {
		t.Errorf("expected 1 status call for a processed replay, got %d", f.statusCalls)
	}
}

func TestBlastRejectedIsNotRetried(t *testing.T) {
	logger.Rlogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	f := newBlastFake()
	defer f.server.Close()

	reason := "corrupt replay"
	f.uploadStatus = "rejected"
	f.rejection = &reason

	// A rejection is deterministic: returning nil lets the uploader cache the match so
	// it is never downloaded and uploaded again.
	if err := f.blast("my-token").UploadReplay(writeReplay(t, "x"), ReplayUpload{}); err != nil {
		t.Fatalf("rejected replay must not report an error, got %v", err)
	}
	if f.statusCalls != 1 {
		t.Errorf("expected polling to stop on a terminal status, got %d calls", f.statusCalls)
	}
}

func TestBlastPendingStopsAfterBudget(t *testing.T) {
	logger.Rlogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	f := newBlastFake()
	defer f.server.Close()

	f.uploadStatus = "pending"

	if err := f.blast("my-token").UploadReplay(writeReplay(t, "x"), ReplayUpload{}); err != nil {
		t.Fatalf("a pending replay is uploaded, got %v", err)
	}
	if f.statusCalls != blastStatusAttempts {
		t.Errorf("expected %d status calls, got %d", blastStatusAttempts, f.statusCalls)
	}
}

func TestBlastNotFoundStatusIsNotFatal(t *testing.T) {
	logger.Rlogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	f := newBlastFake()
	defer f.server.Close()

	f.statusStatus = 404

	if err := f.blast("my-token").UploadReplay(writeReplay(t, "x"), ReplayUpload{}); err != nil {
		t.Fatalf("an unreadable status must not fail the upload, got %v", err)
	}
}

func TestBlastFailures(t *testing.T) {
	logger.Rlogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("missing token", func(t *testing.T) {
		f := newBlastFake()
		defer f.server.Close()

		if err := f.blast("  ").UploadReplay(writeReplay(t, "x"), ReplayUpload{}); err == nil {
			t.Fatal("expected an error without a token")
		}
	})

	t.Run("session rejected", func(t *testing.T) {
		f := newBlastFake()
		defer f.server.Close()

		f.sessionStatus = 401

		if err := f.blast("bad").UploadReplay(writeReplay(t, "x"), ReplayUpload{}); err == nil {
			t.Fatal("expected an error when the session is refused")
		}
	})

	t.Run("presigned put rejected", func(t *testing.T) {
		f := newBlastFake()
		defer f.server.Close()

		f.putStatus = 403

		if err := f.blast("my-token").UploadReplay(writeReplay(t, "x"), ReplayUpload{}); err == nil {
			t.Fatal("expected an error when the object storage refuses the upload")
		}
	})

	t.Run("send replay disabled", func(t *testing.T) {
		f := newBlastFake()
		defer f.server.Close()

		blast := f.blast("my-token")
		blast.GetConfig().SendReplay = false

		if err := blast.UploadReplay(writeReplay(t, "x"), ReplayUpload{}); err != nil {
			t.Fatalf("a disabled storage is a no-op, got %v", err)
		}
		if f.statusCalls != 0 {
			t.Error("a disabled storage must not call the API")
		}
	})
}

func TestBlastTokenPastedWithPrefix(t *testing.T) {
	logger.Rlogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	f := newBlastFake()
	defer f.server.Close()

	if err := f.blast("Bearer my-token").UploadReplay(writeReplay(t, "x"), ReplayUpload{}); err != nil {
		t.Fatal(err)
	}
	if f.session.authorization != "Bearer my-token" {
		t.Errorf("session auth = %q, want a single Bearer prefix", f.session.authorization)
	}
}

func TestBlastPing(t *testing.T) {
	logger.Rlogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("valid token answers not found", func(t *testing.T) {
		f := newBlastFake()
		defer f.server.Close()

		f.statusStatus = 404

		blast := f.blast("my-token")
		blast.GetConfig().SendPing = true

		if err := blast.Ping(); err != nil {
			t.Fatalf("404 on the nil upload id means the token is valid, got %v", err)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		f := newBlastFake()
		defer f.server.Close()

		f.statusStatus = 401

		blast := f.blast("bad")
		blast.GetConfig().SendPing = true

		if err := blast.Ping(); err == nil {
			t.Fatal("expected an error for an invalid token")
		}
	})

	t.Run("disabled", func(t *testing.T) {
		f := newBlastFake()
		defer f.server.Close()

		if err := f.blast("").Ping(); err != nil {
			t.Fatalf("ping is a no-op when disabled, got %v", err)
		}
	})
}

func TestBlastCannotSendLive(t *testing.T) {
	logger.Rlogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	f := newBlastFake()
	defer f.server.Close()

	blast := f.blast("my-token")
	if err := blast.UploadLive(nil); err != nil {
		t.Fatalf("live is a no-op when disabled, got %v", err)
	}

	blast.GetConfig().SendLive = true
	if err := blast.UploadLive(nil); err == nil {
		t.Fatal("expected an error as live data is not supported")
	}
}

// TestBlastLiveSession checks the upload session against the real API. It is skipped
// unless BLAST_TOKEN is set, as creating a session records a pending upload:
//
//	BLAST_TOKEN=<token> go test ./app/upload/ -run TestBlastLiveSession -v
func TestBlastLiveSession(t *testing.T) {
	logger.Rlogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	token := os.Getenv("BLAST_TOKEN")
	if token == "" {
		t.Skip("BLAST_TOKEN is not set")
	}

	storage := *config.BLAST_STORAGE
	storage.SendReplay = true
	storage.Token = token

	session, err := NewBlast(&storage).createUploadSession()
	if err != nil {
		t.Fatalf("failed to create an upload session: %v", err)
	}

	t.Logf("fileId=%s fileKey=%s", session.FileID, session.FileKey)

	if session.FileID == "" || session.FileKey == "" {
		t.Errorf("incomplete upload session: %+v", session)
	}
}
