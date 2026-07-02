//go:build !android && !ios

package tools

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	BrowserProfilePrefix = "rockpload_profile_"
)

type EpicAuthResponse struct {
	AuthorizationCode string `json:"authorizationCode"`
}

func OpenBrowser(url string) {
	logger.FuncDebug()

	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "rundll32"
		args = append(args, "url.dll,FileProtocolHandler")
	case "darwin":
		cmd = "open"
	}
	args = append(args, url)
	exec.Command(cmd, args...).Start()
}

func OpenAutoChromiumBrowser(url string, profileId int) (authCode string, err error) {
	logger.FuncDebug()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // TODO Check when profile folder exist and make it headless so user dont see navigator
		chromedp.Flag("user-data-dir", constant.Paths.BrowserSession),
		chromedp.Flag("profile-directory", getBrowserProfilePath(profileId)),
		chromedp.Flag("disable-restore-session-state", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 5*time.Minute)
	defer cancelTimeout()

	var pageContent string

	logger.Rlogger.Debug("Opening browser for Epic Games authentication...")

	err = chromedp.Run(ctx,
		chromedp.Navigate(url),

		chromedp.ActionFunc(func(ctx context.Context) error {
			for {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				var text string
				evaluateErr := chromedp.Evaluate(`document.body.innerText`, &text).Do(ctx)

				if evaluateErr != nil {
					if strings.Contains(evaluateErr.Error(), "target closed") || strings.Contains(evaluateErr.Error(), "session deleted") {
						logger.Rlogger.Warn("User hsas closed the browser.")
						return evaluateErr
					}

				} else if strings.Contains(text, `"authorizationCode"`) {
					pageContent = text
					logger.Rlogger.Debug("Auth retrieved, waiting for browser to save profile.")

					return nil
				}

				select {
				case <-time.After(500 * time.Millisecond):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}),
	)

	if err != nil {
		logger.Rlogger.Error("Error or expiration of the delay for authentification", slog.Any("err", err))
		return "", err
	}

	if pageContent == "" {
		logger.Rlogger.Error("Page is empty, unable to extract the code")
		return "", context.Canceled
	}

	var authResp EpicAuthResponse
	if err := json.Unmarshal([]byte(pageContent), &authResp); err != nil {
		logger.Rlogger.Error("Unable to parse the JSON response", slog.Any("pageContent", pageContent), slog.Any("err", err))
		return "", err
	}

	return authResp.AuthorizationCode, nil
}

func ClearBrowserSession() {
	logger.FuncDebug()

	if err := os.RemoveAll(constant.Paths.BrowserSession); err != nil {
		logger.Rlogger.Error("Failed to clear browser session", slog.Any("err", err))
	}
}

func ClearBrowserProfile(profileId int) {
	logger.FuncDebug()

	profileFolderName := getBrowserProfilePath(profileId)
	fullProfilePath := filepath.Join(constant.Paths.BrowserSession, profileFolderName)

	if err := os.RemoveAll(fullProfilePath); err != nil {
		logger.Rlogger.Error("Failed to clear browser profile", slog.Any("err", err))
	}
}

func getBrowserProfilePath(profileId int) string {
	logger.FuncDebug()

	return BrowserProfilePrefix + strconv.Itoa(profileId)
}
