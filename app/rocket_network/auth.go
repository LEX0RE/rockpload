package rocket_network

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/LEX0RE/rockpload/app/config"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"github.com/dank/rlapi"
)

const (
	EventUserAuthenticated    = "user_authenticated"
)

type Auth struct {
	EGS *rlapi.EGS
	Auth *rlapi.TokenResponse
	Sub *tools.Subscription
}

func NewAuth() (ra *Auth, err error) {
	logger.FuncDebug()
	ra = &Auth{Sub: tools.NewSubscription()}

	ra.EGS = rlapi.NewEGS()
	err = ra.retrieveToken()
	if err != nil {
		logger.Rlogger.Error("Failed to retrieve token:", slog.Any("err", err))
		return ra, err
	}

	return ra, nil
}

func (ra *Auth) OpenAuth() {
	logger.FuncDebug()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()

	err := ra.openAutoAuth()
	if err != nil {
		logger.Rlogger.Error("Failed to retrieve token from auto browser (will try manually):", slog.Any("err", err))
		ra.openAuthURL()
	}

}

func (ra *Auth) AuthenticateWithCode(authCode string) (err error) {
	logger.FuncDebug()
	auth, err := ra.EGS.AuthenticateWithCode(strings.TrimSpace(strings.ReplaceAll(authCode, "\"", "")))
	if err != nil {
		logger.Rlogger.Error("Failed to authenticate with code:", slog.Any("err", err))
		return err
	}

	ra.Auth = auth
	ra.writeTokenToFile()
	ra.Sub.Notify(EventUserAuthenticated)

	return nil
}

func (ra *Auth) ClearBrowserProfile() {
	logger.FuncDebug()
	err := os.RemoveAll(config.BrowserSession)
	if err != nil {
		logger.Rlogger.Error("Failed to clear browser profile", slog.Any("err", err))
	}
}

func (ra *Auth) ClearToken() {
	logger.FuncDebug()
	ra.Auth = nil

	err := os.Remove(config.RLToken)
	if err != nil && !os.IsNotExist(err) {
		logger.Rlogger.Error("Failed to clear token file", slog.Any("err", err))
	}
}

// User interaction, but any browser
func (ra *Auth) openAuthURL() {
	logger.FuncDebug()

	tools.OpenBrowser(ra.EGS.GetAuthURL())
}

// No user interaction, but chrome
func (ra *Auth) openAutoAuth() (err error) {
	logger.FuncDebug()

	tokenEntry, err := tools.OpenAutoChromiumBrowser(ra.EGS.GetAuthURL())
	if err != nil {
		logger.Rlogger.Error("Failed to retrieve token:", slog.Any("err", err))
		return err
	}

	err = ra.AuthenticateWithCode(tokenEntry)
	if err != nil {
		logger.Rlogger.Error("Authentication failed:", slog.Any("err", err))
		return err
	}

	return nil
}

func (ra *Auth) retrieveToken() (err error) {
	logger.FuncDebug()

	// Wait a bit if file is not found
	for range 10 {
		if _, err := os.Stat(config.RLToken); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if refreshTokenData, err := os.ReadFile(config.RLToken); err == nil && len(strings.TrimSpace(string(refreshTokenData))) > 0 {
		refreshToken := strings.TrimSpace(string(refreshTokenData))
		auth, err := ra.EGS.AuthenticateWithRefreshToken(refreshToken)

		if err != nil {
			logger.Rlogger.Error("Failed to authenticate with refresh token", slog.Any("err", err))
			return err
		} else {
			ra.Auth = auth
			ra.writeTokenToFile()
			ra.Sub.Notify(EventUserAuthenticated)

			return nil
		}
	}

	return fmt.Errorf("no valid authentication token")
}

func (ra *Auth) writeTokenToFile() {
	logger.FuncDebug()
	if ra.Auth != nil {
		err := os.WriteFile(config.RLToken, []byte(ra.Auth.RefreshToken), 0644)
		if err != nil {
			logger.Rlogger.Error("Failed to save refresh token", slog.Any("err", err))
		}
	}
}