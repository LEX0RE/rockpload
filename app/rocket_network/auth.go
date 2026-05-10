package rocket_network

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"github.com/dank/rlapi"
)

const (
	EventUserAuthenticated = "user_authenticated"

	TokenFilePrefix = "token_profile_"
)

type Auth struct {
	ProfileId    int                 `json:"profile_id"`
	EventManager *tools.EventManager `json:"-"`

	eosToken *rlapi.EOSTokenResponse `json:"-"`
	egs      *rlapi.EGS              `json:"-"`
}

func NewAuth(profileId int) (a *Auth, err error) {
	logger.FuncDebug()
	a = &Auth{EventManager: tools.NewEventManager(), ProfileId: profileId}

	a.egs = rlapi.NewEGS()

	if err := a.Authenticate(); err != nil {
		logger.Rlogger.Error("Failed to authenticate:", slog.Any("err", err))
		return a, err
	}

	return a, nil
}

func (a *Auth) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	type Alias Auth

	aux := (*Alias)(a)

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	a.EventManager = tools.NewEventManager()
	a.egs = rlapi.NewEGS()

	return nil
}

func (a *Auth) Authenticate() (err error) {
	logger.FuncDebug()

	if a.IsAuthenticated() {
		return nil
	}

	if err := a.retrieveToken(); err != nil {
		logger.Rlogger.Error("Failed to retrieve token:", slog.Any("err", err))
		return err
	}

	return nil
}

func (a *Auth) IsAuthenticated() bool {
	return a.eosToken != nil
}

func (a *Auth) OpenAuth() {
	logger.FuncDebug()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()

	err := a.openAutoAuth()
	if err != nil {
		logger.Rlogger.Error("Failed to retrieve token from auto browser (will try manually):", slog.Any("err", err))
		a.openAuthURL()
	}

}

func (a *Auth) AuthenticateWithCode(authCode string) (err error) {
	logger.FuncDebug()
	auth, err := a.egs.AuthenticateWithCode(strings.TrimSpace(strings.ReplaceAll(authCode, "\"", "")))
	if err != nil {
		logger.Rlogger.Error("Failed to authenticate with code:", slog.Any("err", err))
		return err
	}

	_, err = a.exchangeToEOSToken(auth)
	if err != nil {
		logger.Rlogger.Error("Failed to exchange to EOS Token:", slog.Any("err", err))
		return err
	}

	return nil
}

func (a *Auth) ClearToken() {
	logger.FuncDebug()
	a.eosToken = nil

	err := os.Remove(a.tokenPath())
	if err != nil && !os.IsNotExist(err) {
		logger.Rlogger.Error("Failed to clear token file", slog.Any("err", err))
	}
}

func (a *Auth) tokenPath() string {
	return filepath.Join(constant.TokensPath, TokenFilePrefix+fmt.Sprint(a.ProfileId))
}

// User interaction, but any browser
func (a *Auth) openAuthURL() {
	logger.FuncDebug()

	tools.OpenBrowser(a.egs.GetAuthURL())
}

// No user interaction, but chrome
func (a *Auth) openAutoAuth() (err error) {
	logger.FuncDebug()

	tokenEntry, err := tools.OpenAutoChromiumBrowser(a.egs.GetAuthURL(), a.ProfileId)
	if err != nil {
		logger.Rlogger.Error("Failed to retrieve token:", slog.Any("err", err))
		return err
	}

	err = a.AuthenticateWithCode(tokenEntry)
	if err != nil {
		logger.Rlogger.Error("Authentication failed:", slog.Any("err", err))
		return err
	}

	return nil
}

func (a *Auth) retrieveToken() (err error) {
	logger.FuncDebug()

	if refreshTokenData, err := os.ReadFile(a.tokenPath()); err == nil && len(strings.TrimSpace(string(refreshTokenData))) > 0 {
		refreshToken := strings.TrimSpace(string(refreshTokenData))
		eosToken, err := a.egs.RefreshEOSToken(refreshToken)

		if err != nil {
			logger.Rlogger.Error("Failed to authenticate with refresh token", slog.Any("err", err))
			return err
		}

		a.eosToken = eosToken
		a.onAuth()

		return nil
	}

	return fmt.Errorf("no valid authentication token")
}

func (a *Auth) exchangeToEOSToken(token *rlapi.TokenResponse) (eosToken *rlapi.EOSTokenResponse, err error) {
	logger.FuncDebug()

	code, err := a.egs.GetExchangeCode(token.AccessToken)
	if err != nil {
		logger.Rlogger.Error("Failed to get exchange code:", slog.Any("err", err))
		return nil, err
	}

	authToken, err := a.egs.ExchangeEOSToken(code)
	if err != nil {
		logger.Rlogger.Error("Failed to exchange to EOS token:", slog.Any("err", err))
		return nil, err
	} else {
		a.eosToken = authToken
		a.onAuth()

		return authToken, nil
	}
}

func (a *Auth) onAuth() {
	a.writeTokenToFile()
	a.EventManager.Notify(EventUserAuthenticated, nil)
}

func (a *Auth) writeTokenToFile() {
	logger.FuncDebug()
	if a.eosToken != nil {
		if err := tools.SaveFilePath(a.tokenPath(), []byte(a.eosToken.RefreshToken)); err != nil {
			logger.Rlogger.Error("Failed to save refresh token", slog.Any("err", err))
		}
	}
}
