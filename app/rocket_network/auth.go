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
	EventUserAuthenticated tools.EventType = "user_authenticated"

	TokenFilePrefix = "token_profile_"

	REFRESH_MARGIN = 15 * time.Minute
)

type Auth struct {
	ProfileId    int                       `json:"profile_id"`
	EventManager *tools.EventManager       `json:"-"`
	egsToken     *rlapi.TokenResponse      `json:"-"`
	eosToken     *rlapi.EOSTokenResponse   `json:"-"`
	deviceAuth   *rlapi.DeviceAuthResponse `json:"-"`
	egs          *rlapi.EGS                `json:"-"`
	logger       *slog.Logger              `json:"-"`

	fileEOSMu sync.Mutex `json:"-"`
	fileEGSMu sync.Mutex `json:"-"`
	eosMu     sync.Mutex `json:"-"`
	egsMu     sync.Mutex `json:"-"`
}

type AuthTokenPath struct {
	EGS string
	EOS string
}

func NewAuth(profileId int) (a *Auth) {
	logger.FuncDebug()

	a = &Auth{EventManager: tools.NewEventManager(), ProfileId: profileId, egs: rlapi.NewEGS(), logger: logger.Rlogger.With("profileId", profileId)}
	_ = a.isAuthenticated()

	return a
}

func (a *Auth) UnmarshalJSON(data []byte) error {
	logger.FuncDebug()

	type Alias Auth

	aux := (*Alias)(a)

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	a.logger = logger.Rlogger.With("profileId", a.ProfileId)
	a.EventManager = tools.NewEventManager()
	a.egs = rlapi.NewEGS()

	return nil
}

func (a *Auth) OpenDeviceAuth() {
	logger.FuncDebug()

	tools.OpenBrowser(a.deviceAuth.VerificationURI)
}

func (a *Auth) OpenAuth() {
	logger.FuncDebug()

	defer func() {
		if r := recover(); r != nil {
			a.logger.Error("Recovered from panic:", r)
		}
	}()

	err := a.openAutoAuth()
	if err != nil {
		a.logger.Error("Failed to retrieve token from auto browser (will try manually):", slog.Any("err", err))
		a.openAuthURL()
	}
}

func (a *Auth) AuthenticateWithAuthCode(authCode string) (err error) {
	logger.FuncDebug()

	auth, err := a.egs.AuthenticateWithCode(strings.TrimSpace(strings.ReplaceAll(authCode, "\"", "")))
	if err != nil {
		a.logger.Error("Failed to authenticate with code:", slog.Any("err", err))
		return err
	}

	a.setEGSToken(auth)

	if err = a.refreshEGSAccess(); err != nil {
		a.logger.Error("Failed to exchange to EOS Token:", slog.Any("err", err))
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
		a.logger.Error("Failed to authenticate with refresh token", slog.Any("err", err))
		return err
	} else {
		a.setEOSToken(eosToken)
		a.EventManager.Notify(EventUserAuthenticated, nil)

		return nil
	}
}

func (a *Auth) GetDeviceCode() (deviceAuth *rlapi.DeviceAuthResponse, err error) {
	logger.FuncDebug()

	a.deviceAuth, err = a.egs.AuthenticateWithDevice()
	if err != nil {
		a.logger.Error("Failed to get device code", slog.Any("err", err))
		return nil, err
	}

	return a.deviceAuth, nil
}

func (a *Auth) isAuthenticated() bool {
	logger.FuncDebug()

	if a.getValidToken() != nil {
		return true
	}

	a.clearToken()
	return false
}

func (a *Auth) clearToken() {
	logger.FuncDebug()

	a.egsToken = nil
	a.eosToken = nil

	tokenPath := a.tokenPath()
	if err := os.Remove(tokenPath.EGS); err != nil && !os.IsNotExist(err) {
		a.logger.Error("Failed to clear EGS token file", slog.Any("err", err))
	}

	if err := os.Remove(tokenPath.EOS); err != nil && !os.IsNotExist(err) {
		a.logger.Error("Failed to clear EOS token file", slog.Any("err", err))
	}

	tools.ClearBrowserProfile(a.ProfileId)
}

func (a *Auth) getValidToken() *rlapi.EOSTokenResponse {
	logger.FuncDebug()

	a.eosMu.Lock()
	defer a.eosMu.Unlock()

	if a.eosToken == nil && a.egsToken == nil {
		a.readTokens()

		if a.eosToken == nil && a.egsToken == nil {
			a.logger.Debug("No Token available to authenticate")
			return nil
		}
	}

	if a.eosToken != nil {
		if a.isAccessTokenValid() {
			return a.eosToken
		}

		refreshTokenExpireAt, err := time.Parse(time.RFC3339, a.eosToken.RefreshExpiresAt)
		if err != nil {
			a.logger.Debug("Failed to parse expire date:", slog.Any("err", err))
			refreshTokenExpireAt = time.Now()
		}

		if time.Now().Before(refreshTokenExpireAt.Add(-REFRESH_MARGIN)) {
			newRefreshToken, err := a.egs.RefreshEOSToken(a.eosToken.RefreshToken)

			if err == nil {
				a.setEOSToken(newRefreshToken)
				return a.eosToken
			} else {
				a.logger.Error("Failed to refresh EOS token", slog.Any("err", err))
			}
		}
	}

	if err := a.refreshEGSAccess(); err != nil {
		a.logger.Error("Failed to refresh EGS token", slog.Any("err", err))
		return nil
	}

	return a.eosToken
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
		a.logger.Error("Failed to retrieve token:", slog.Any("err", err))
		return err
	}

	err = a.AuthenticateWithAuthCode(tokenEntry)
	if err != nil {
		a.logger.Error("Authentication failed:", slog.Any("err", err))
		return err
	}

	return nil
}

func (a *Auth) readTokens() {
	logger.FuncDebug()

	tokenPath := a.tokenPath()

	if refreshTokenData, err := os.ReadFile(tokenPath.EGS); err == nil && len(strings.TrimSpace(string(refreshTokenData))) > 0 {
		refreshToken := strings.TrimSpace(string(refreshTokenData))
		token, err := a.egs.AuthenticateWithRefreshToken(refreshToken)

		if err != nil {
			a.logger.Error("Failed to authenticate with EGS refresh token", slog.Any("err", err))
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
			a.logger.Error("Failed to authenticate with EOS refresh token", slog.Any("err", err))
			return
		}

		a.setEOSToken(token)
		a.EventManager.Notify(EventUserAuthenticated, nil)

		return
	}

	a.logger.Error("no valid authentication token")
}

func (a *Auth) refreshEGSAccess() (err error) {
	logger.FuncDebug()

	a.egsMu.Lock()
	defer a.egsMu.Unlock()

	if a.egsToken == nil {
		a.readTokens()

		if a.egsToken == nil {
			return fmt.Errorf("no valid authentication token")
		}
	}

	tokenExpireAt, err := time.Parse(time.RFC3339, a.egsToken.ExpiresAt)
	if err != nil {
		a.logger.Debug("Failed to parse expire date:", slog.Any("err", err))
		tokenExpireAt = time.Now()
	}

	if !time.Now().Before(tokenExpireAt.Add(-REFRESH_MARGIN)) {
		newEGSToken, err := a.egs.AuthenticateWithRefreshToken(a.egsToken.RefreshToken)
		if err != nil {
			a.logger.Error("Failed to authenticate with EGS refresh token", slog.Any("err", err))
			return err
		}

		a.setEGSToken(newEGSToken)
	}

	code, err := a.egs.GetExchangeCode(a.egsToken.AccessToken)
	if err != nil {
		a.logger.Error("Failed to get exchange code from EGS token:", slog.Any("err", err))
		return err
	}

	eosToken, err := a.egs.ExchangeEOSToken(code)
	if err != nil {
		a.logger.Error("Failed to exchange to EOS token:", slog.Any("err", err))
		return err
	}

	a.setEOSToken(eosToken)

	return nil
}

func (a *Auth) saveEGSToken() {
	logger.FuncDebug()

	a.fileEGSMu.Lock()
	defer a.fileEGSMu.Unlock()

	if a.egsToken != nil {
		if err := tools.SaveFilePath(a.tokenPath().EGS, []byte(a.egsToken.RefreshToken)); err != nil {
			a.logger.Error("Failed to save EGS refresh token", slog.Any("err", err))
		}
	} else {
		a.logger.Error("Failed to save nil EGS refresh token")
	}

	a.logger.Info("EGS token saved successfully")
}

func (a *Auth) saveEOSToken() {
	logger.FuncDebug()

	a.fileEOSMu.Lock()
	defer a.fileEOSMu.Unlock()

	if a.eosToken != nil {
		if err := tools.SaveFilePath(a.tokenPath().EOS, []byte(a.eosToken.RefreshToken)); err != nil {
			a.logger.Error("Failed to save EOS refresh token", slog.Any("err", err))
		}
	} else {
		a.logger.Error("Failed to save nil EOS refresh token")
	}

	a.logger.Info("EOS token saved successfully")
}

func (a *Auth) isAccessTokenValid() bool {
	logger.FuncDebug()

	tokenExpireAt, err := time.Parse(time.RFC3339, a.eosToken.ExpiresAt)
	if err != nil {
		a.logger.Debug("Failed to parse expire date:", slog.Any("err", err))
		tokenExpireAt = time.Now()
	}

	return time.Now().Before(tokenExpireAt.Add(-REFRESH_MARGIN))
}
