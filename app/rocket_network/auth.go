package rocket_network

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools"
	"github.com/LEX0RE/rockpload/app/tools/logger"

	"github.com/dank/rlapi"
)

const (
	EventUserAuthenticated = "user_authenticated"

	TokenFilePrefix = "token_profile_"

	REFRESH_MARGIN = 15 * time.Minute
)

type Auth struct {
	ProfileId    int                 `json:"profile_id"`
	EventManager *tools.EventManager `json:"-"`

	mu         sync.Mutex                `json:"-"`
	egsToken   *rlapi.TokenResponse      `json:"-"`
	eosToken   *rlapi.EOSTokenResponse   `json:"-"`
	deviceAuth *rlapi.DeviceAuthResponse `json:"-"`
	egs        *rlapi.EGS                `json:"-"`
}

type AuthTokenPath struct {
	EGS string
	EOS string
}

func NewAuth(profileId int) (a *Auth) {
	logger.FuncDebug()

	a = &Auth{EventManager: tools.NewEventManager(), ProfileId: profileId, mu: sync.Mutex{}, egs: rlapi.NewEGS()}
	a.Authenticate()

	return a
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

func (a *Auth) Authenticate() {
	logger.FuncDebug()

	if a.IsAuthenticated() {
		return
	}

	a.readTokens()
}

func (a *Auth) IsAuthenticated() bool {
	return a.GetValidToken() != nil
}

func (a *Auth) GetValidToken() *rlapi.EOSTokenResponse {
	logger.FuncDebug()

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.eosToken == nil && a.egsToken == nil {
		return nil
	}

	if a.eosToken != nil {
		tokenExpireAt, err := time.Parse(time.RFC3339, a.eosToken.ExpiresAt)
		if err != nil {
			logger.Rlogger.Debug("Failed to parse expire date:", slog.Any("err", err))
			tokenExpireAt = time.Now()
		}

		if time.Now().Before(tokenExpireAt.Add(-REFRESH_MARGIN)) {
			return a.eosToken
		}

		refreshTokenExpireAt, err := time.Parse(time.RFC3339, a.eosToken.RefreshExpiresAt)
		if err != nil {
			logger.Rlogger.Debug("Failed to parse expire date:", slog.Any("err", err))
			refreshTokenExpireAt = time.Now()
		}

		if time.Now().Before(refreshTokenExpireAt.Add(-REFRESH_MARGIN)) {
			newRefreshToken, err := a.egs.RefreshEOSToken(a.eosToken.RefreshToken)

			if err == nil {
				a.setEOSToken(newRefreshToken)
				return a.eosToken
			} else {
				logger.Rlogger.Error("Failed to refresh EOS token", slog.Any("err", err))
			}
		}
	}

	if err := a.refreshEGSAccess(); err != nil {
		logger.Rlogger.Error("Failed to refresh EGS token", slog.Any("err", err))
		return nil
	}

	return a.eosToken
}

func (a *Auth) OpenDeviceAuth() {
	logger.FuncDebug()

	tools.OpenBrowser(a.deviceAuth.VerificationURI)
}

func (a *Auth) OpenAuth() {
	logger.FuncDebug()

	defer func() {
		if r := recover(); r != nil {
			logger.Rlogger.Error("Recovered from panic:", r)
		}
	}()

	err := a.openAutoAuth()
	if err != nil {
		logger.Rlogger.Error("Failed to retrieve token from auto browser (will try manually):", slog.Any("err", err))
		a.openAuthURL()
	}
}

func (a *Auth) AuthenticateWithAuthCode(authCode string) (err error) {
	logger.FuncDebug()

	auth, err := a.egs.AuthenticateWithCode(strings.TrimSpace(strings.ReplaceAll(authCode, "\"", "")))
	if err != nil {
		logger.Rlogger.Error("Failed to authenticate with code:", slog.Any("err", err))
		return err
	}

	a.setEGSToken(auth)

	if err = a.refreshEGSAccess(); err != nil {
		logger.Rlogger.Error("Failed to exchange to EOS Token:", slog.Any("err", err))
		return err
	}

	a.EventManager.Notify(EventUserAuthenticated, nil)

	return nil
}

func (a *Auth) AuthenticateWithDeviceCode() (err error) {
	logger.FuncDebug()

	var eosToken *rlapi.EOSTokenResponse

	for range a.deviceAuth.ExpiresIn / a.deviceAuth.Interval {
		eosToken, err = a.egs.WaitForDeviceAuthorization(a.deviceAuth)
		if err == nil {
			break
		}

		time.Sleep(time.Duration(a.deviceAuth.Interval) * time.Second)
	}

	if err != nil {
		logger.Rlogger.Error("Failed to authenticate with refresh token", slog.Any("err", err))
		return err
	} else {
		a.setEOSToken(eosToken)
		a.EventManager.Notify(EventUserAuthenticated, nil)

		return nil
	}
}

func (a *Auth) ClearToken() {
	logger.FuncDebug()
	a.egsToken = nil
	a.eosToken = nil

	tokenPath := a.tokenPath()
	if err := os.Remove(tokenPath.EGS); err != nil && !os.IsNotExist(err) {
		logger.Rlogger.Error("Failed to clear EGS token file", slog.Any("err", err))
	}

	if err := os.Remove(tokenPath.EOS); err != nil && !os.IsNotExist(err) {
		logger.Rlogger.Error("Failed to clear EOS token file", slog.Any("err", err))
	}
}

func (a *Auth) setEGSToken(token *rlapi.TokenResponse) {
	logger.FuncDebug()

	a.egsToken = token
	a.saveEGSToken()
}

func (a *Auth) setEOSToken(token *rlapi.EOSTokenResponse) {
	logger.FuncDebug()

	a.eosToken = token
	a.saveEOSToken()
}

func (a *Auth) tokenPath() AuthTokenPath {
	logger.FuncDebug()

	basePath := filepath.Join(constant.TokensPath, TokenFilePrefix+fmt.Sprint(a.ProfileId))
	return AuthTokenPath{
		EGS: basePath + "_egs",
		EOS: basePath + "_eos",
	}
}

func (a *Auth) GetDeviceCode() (deviceAuth *rlapi.DeviceAuthResponse, err error) {
	logger.FuncDebug()
	a.deviceAuth, err = a.egs.AuthenticateWithDevice()
	if err != nil {
		logger.Rlogger.Error("Failed to get device code", slog.Any("err", err))
		return nil, err
	}

	return a.deviceAuth, nil
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

	err = a.AuthenticateWithAuthCode(tokenEntry)
	if err != nil {
		logger.Rlogger.Error("Authentication failed:", slog.Any("err", err))
		return err
	}

	return nil
}

func (a *Auth) readTokens() {
	logger.FuncDebug()

	a.mu.Lock()
	defer a.mu.Unlock()

	tokenPath := a.tokenPath()

	if refreshTokenData, err := os.ReadFile(tokenPath.EGS); err == nil && len(strings.TrimSpace(string(refreshTokenData))) > 0 {
		refreshToken := strings.TrimSpace(string(refreshTokenData))
		token, err := a.egs.AuthenticateWithRefreshToken(refreshToken)

		if err != nil {
			logger.Rlogger.Error("Failed to authenticate with EGS refresh token", slog.Any("err", err))
			return
		}

		a.setEGSToken(token)
		a.refreshEGSAccess()
		a.EventManager.Notify(EventUserAuthenticated, nil)

		return
	}

	if refreshTokenData, err := os.ReadFile(tokenPath.EOS); err == nil && len(strings.TrimSpace(string(refreshTokenData))) > 0 {
		refreshToken := strings.TrimSpace(string(refreshTokenData))
		token, err := a.egs.RefreshEOSToken(refreshToken)

		if err != nil {
			logger.Rlogger.Error("Failed to authenticate with EOS refresh token", slog.Any("err", err))
			return
		}

		a.setEOSToken(token)
		a.EventManager.Notify(EventUserAuthenticated, nil)

		return
	}

	logger.Rlogger.Error("no valid authentication token")
}

func (a *Auth) refreshEGSAccess() (err error) {
	logger.FuncDebug()

	if a.egsToken == nil {
		return fmt.Errorf("no valid authentication token")
	}

	tokenExpireAt, err := time.Parse(time.RFC3339, a.egsToken.ExpiresAt)
	if err != nil {
		logger.Rlogger.Debug("Failed to parse expire date:", slog.Any("err", err))
		tokenExpireAt = time.Now()
	}

	if !time.Now().Before(tokenExpireAt.Add(-REFRESH_MARGIN)) {
		newEGSToken, err := a.egs.AuthenticateWithRefreshToken(a.egsToken.RefreshToken)
		if err != nil {
			logger.Rlogger.Error("Failed to authenticate with EGS refresh token", slog.Any("err", err))
			return err
		}

		a.setEGSToken(newEGSToken)
	}

	code, err := a.egs.GetExchangeCode(a.egsToken.AccessToken)
	if err != nil {
		logger.Rlogger.Error("Failed to get exchange code from EGS token:", slog.Any("err", err))
		return err
	}

	eosToken, err := a.egs.ExchangeEOSToken(code)
	if err != nil {
		logger.Rlogger.Error("Failed to exchange to EOS token:", slog.Any("err", err))
		return err
	}

	a.setEOSToken(eosToken)

	return nil
}

func (a *Auth) saveEGSToken() {
	logger.FuncDebug()

	if a.egsToken != nil {
		if err := tools.SaveFilePath(a.tokenPath().EGS, []byte(a.egsToken.RefreshToken)); err != nil {
			logger.Rlogger.Error("Failed to save EGS refresh token", slog.Any("err", err))
		}
	} else {
		logger.Rlogger.Error("Failed to save nil EGS refresh token")
	}

	logger.Rlogger.Info("EGS token saved successfully")
}

func (a *Auth) saveEOSToken() {
	logger.FuncDebug()

	if a.eosToken != nil {
		if err := tools.SaveFilePath(a.tokenPath().EOS, []byte(a.eosToken.RefreshToken)); err != nil {
			logger.Rlogger.Error("Failed to save EOS refresh token", slog.Any("err", err))
		}
	} else {
		logger.Rlogger.Error("Failed to save nil EOS refresh token")
	}

	logger.Rlogger.Info("EOS token saved successfully")
}
