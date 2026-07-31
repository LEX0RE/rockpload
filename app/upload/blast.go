package upload

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/rocket_network"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	// The upload session route answers with a short lived presigned URL, so the replay
	// is pushed straight to the object storage and the ingestion result is polled back.
	blastStatusPath     = "/v1/community-stats/rl/replays/"
	blastStatusAttempts = 3
	blastStatusDelay    = 400 * time.Millisecond

	// Reading the nil upload id validates a token without creating a pending upload
	// record: a valid token answers 404 while an invalid one answers 401.
	blastPingUploadID = "00000000-0000-0000-0000-000000000000"

	// Ingestion is deterministic, so a rejected replay is never retried.
	blastStatusRejected  = "rejected"
	blastStatusProcessed = "processed"
)

type Blast struct {
	config *config.StorageConfig
}

type blastUploadSession struct {
	FileID    string `json:"fileId"`
	FileKey   string `json:"fileKey"`
	UploadURL string `json:"uploadUrl"`
}

type blastUploadStatus struct {
	ID              string  `json:"id"`
	Status          string  `json:"status"`
	RejectionReason *string `json:"rejectionReason"`
}

func NewBlast(config *config.StorageConfig) *Blast {
	return &Blast{config: config}
}

func (b *Blast) UploadReplay(filePath string, replayUpload ReplayUpload) error {
	logger.FuncDebug()

	if !b.config.SendReplay {
		return nil
	}

	if strings.TrimSpace(b.config.Token) == "" {
		return fmt.Errorf("a token is required to upload to %s", b.config.Name)
	}

	if b.config.SendPing {
		if err := b.Ping(); err != nil {
			return fmt.Errorf("%s ping failed: %w", b.config.Name, err)
		}
	}

	session, err := b.createUploadSession()
	if err != nil {
		return err
	}

	logger.Rlogger.Debug("Upload session created",
		slog.Any("fileId", session.FileID),
		slog.Any("fileKey", session.FileKey),
		slog.Any("matchGUID", replayUpload.Replay.Match.MatchGUID))

	if err := b.putReplay(session.UploadURL, filePath); err != nil {
		return err
	}

	b.logIngestion(session.FileID)

	return nil
}

func (b *Blast) UploadLive(liveStats *rocket_network.LiveStats) error {
	logger.FuncDebug()

	if !b.config.SendLive {
		return nil
	}

	return fmt.Errorf("%s can't send live data", b.config.Name)
}

func (b *Blast) Ping() error {
	logger.FuncDebug()

	if !b.config.SendPing {
		return nil
	}

	resp, err := b.do("GET", b.config.URL+blastStatusPath+blastPingUploadID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// The nil upload id never exists, so a 404 proves the token was accepted.
	if resp.StatusCode == http.StatusNotFound || (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	return fmt.Errorf("token is invalid: %s\n%s", resp.Status, string(body))
}

func (b *Blast) GetConfig() *config.StorageConfig {
	return b.config
}

func (b *Blast) createUploadSession() (*blastUploadSession, error) {
	logger.FuncDebug()

	resp, err := b.do("POST", b.config.URL+b.config.ReplayPath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload session response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to create upload session: %s\n%s", resp.Status, string(body))
	}

	var session blastUploadSession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("failed to parse upload session response: %w", err)
	}

	if session.UploadURL == "" {
		return nil, fmt.Errorf("upload session is missing an upload url")
	}

	return &session, nil
}

func (b *Blast) putReplay(uploadURL string, filePath string) error {
	logger.FuncDebug()

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to read file size: %w", err)
	}

	req, err := http.NewRequest("PUT", uploadURL, file)
	if err != nil {
		return err
	}

	// The presigned URL carries its own credentials: an Authorization header or extra
	// query parameters would break its signature. ContentLength has to be set because
	// the object storage rejects the chunked encoding used for a body of unknown size.
	req.ContentLength = info.Size()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("replay upload failed: %s\n%s", resp.Status, string(body))
	}

	logger.Rlogger.Debug("Upload successful")

	return nil
}

// logIngestion reports the ingestion result of an already uploaded replay. The replay
// is never resent: the upload itself succeeded, and a rejection is deterministic so
// retrying it would download and upload the same rejected replay on every run.
func (b *Blast) logIngestion(fileID string) {
	logger.FuncDebug()

	for attempt := range blastStatusAttempts {
		if attempt > 0 {
			time.Sleep(blastStatusDelay)
		}

		status, err := b.uploadStatus(fileID)
		if err != nil {
			logger.Rlogger.Debug("Failed to read upload status", slog.Any("fileId", fileID), slog.Any("err", err))
			continue
		}

		switch status.Status {
		case blastStatusRejected:
			reason := ""
			if status.RejectionReason != nil {
				reason = *status.RejectionReason
			}

			logger.Rlogger.Error("Replay was rejected", slog.Any("fileId", fileID), slog.Any("reason", reason))

			return
		case blastStatusProcessed:
			logger.Rlogger.Debug("Replay was processed", slog.Any("fileId", fileID))

			return
		}
	}

	logger.Rlogger.Debug("Replay is still being processed", slog.Any("fileId", fileID))
}

func (b *Blast) uploadStatus(fileID string) (*blastUploadStatus, error) {
	logger.FuncDebug()

	resp, err := b.do("GET", b.config.URL+blastStatusPath+fileID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload status response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to read upload status: %s\n%s", resp.Status, string(body))
	}

	var status blastUploadStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse upload status response: %w", err)
	}

	return &status, nil
}

func (b *Blast) do(method string, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// The token is sent as a bearer token, whether or not it was pasted with the prefix.
	token := strings.TrimSpace(b.config.Token)
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer"))
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}

	return client.Do(req)
}
