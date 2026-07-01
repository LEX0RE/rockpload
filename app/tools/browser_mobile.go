//go:build android || ios

package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/LEX0RE/rockpload/app/tools/logger"

	"fyne.io/fyne/v2"
)

const (
	BrowserProfilePrefix = "rockpload_profile_"
)

type EpicAuthResponse struct {
	AuthorizationCode string `json:"authorizationCode"`
}

func OpenBrowser(rawURL string) {
	logger.FuncDebug()

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		logger.Rlogger.Error("Failed to parse URL for mobile browser", "err", err)
		return
	}

	if err := fyne.CurrentApp().OpenURL(parsedURL); err != nil {
		logger.Rlogger.Error("Failed to open mobile browser", "err", err)
	}
}

func OpenAutoChromiumBrowser(rawURL string, profileId int) (authCode string, err error) {
	logger.FuncDebug()

	return "", fmt.Errorf("automated Chromium authentication is not supported on mobile: %w", context.Canceled)
}

func ClearBrowserSession() {
	logger.FuncDebug()
}

func ClearBrowserProfile(profileId int) {
	logger.FuncDebug()
}
