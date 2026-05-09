package tools

import (
	"log/slog"
	"os"
	"os/exec"
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
		chromedp.Flag("user-data-dir", constant.BrowserSession),
		chromedp.Flag("profile-directory", BrowserProfilePrefix+strconv.Itoa(profileId)),
		chromedp.Flag("disable-restore-session-state", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("excludeSwitches", "enable-automation"),
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
				var text string
				err := chromedp.Evaluate(`document.body.innerText`, &text).Do(ctx)

				if err == nil && strings.Contains(text, `"authorizationCode"`) {
					pageContent = text

					logger.Rlogger.Debug("Auth retrieved, waiting for browser to save profile.")

					return chromedp.Cancel(ctx)
				}
				time.Sleep(500 * time.Millisecond)
			}
		}),
	)

	if err != nil {
		logger.Rlogger.Error("Error or expiration of the delay for authentification", slog.Any("err", err))
		return "", err
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
	err := os.RemoveAll(constant.BrowserSession)
	if err != nil {
		logger.Rlogger.Error("Failed to clear browser profile", slog.Any("err", err))
	}
}

func ClearTokenFile(tokenPath string) {
	logger.FuncDebug()
	err := os.Remove(tokenPath)
	if err != nil {
		logger.Rlogger.Error("Failed to clear token file", slog.Any("err", err))
	}
}
