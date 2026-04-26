package rocket_network

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"lexore/rockpload/app/config"
	"lexore/rockpload/app/tools"
	"lexore/rockpload/app/tools/logger"

	"github.com/dank/rlapi"
)

func (ra *Auth) AuthenticateWithConfiguredOAuth(timeout time.Duration) error {
	oauth, ok := config.EpicOAuth()
	if !ok {
		return fmt.Errorf("EPIC_CLIENT_ID and EPIC_CLIENT_SECRET are required in .env")
	}

	codeCh, stopServer, err := startConfiguredOAuthCallbackServer(oauth.RedirectURI)
	if err != nil {
		return err
	}
	defer stopServer()

	tools.OpenBrowser(buildConfiguredOAuthURL(oauth))

	select {
	case code := <-codeCh:
		return ra.AuthenticateWithConfiguredCode(code)
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for Epic OAuth callback")
	}
}

func (ra *Auth) AuthenticateWithConfiguredCode(authCode string) error {
	oauth, ok := config.EpicOAuth()
	if !ok {
		return fmt.Errorf("EPIC_CLIENT_ID and EPIC_CLIENT_SECRET are required in .env")
	}

	auth, err := requestConfiguredToken(oauth, map[string]string{
		"grant_type":   "authorization_code",
		"code":         strings.TrimSpace(strings.ReplaceAll(authCode, "\"", "")),
		"redirect_uri": oauth.RedirectURI,
		"token_type":   "eg1",
	})
	if err != nil {
		return err
	}

	ra.Auth = auth
	ra.writeTokenToFile()
	ra.Sub.Notify(EventUserAuthenticated)
	return nil
}

func (ra *Auth) AuthenticateWithConfiguredRefreshToken(refreshToken string) error {
	oauth, ok := config.EpicOAuth()
	if !ok {
		return fmt.Errorf("EPIC_CLIENT_ID and EPIC_CLIENT_SECRET are required in .env")
	}

	auth, err := requestConfiguredToken(oauth, map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": strings.TrimSpace(refreshToken),
		"token_type":    "eg1",
	})
	if err != nil {
		return err
	}

	ra.Auth = auth
	ra.writeTokenToFile()
	ra.Sub.Notify(EventUserAuthenticated)
	return nil
}

func buildConfiguredOAuthURL(oauth config.EpicOAuthConfig) string {
	authURL, err := url.Parse(oauth.AuthURL)
	if err != nil {
		return oauth.AuthURL
	}

	query := authURL.Query()
	query.Set("client_id", oauth.ClientID)
	query.Set("response_type", "code")
	query.Set("redirect_uri", oauth.RedirectURI)
	if oauth.Scope != "" {
		query.Set("scope", oauth.Scope)
	}
	authURL.RawQuery = query.Encode()

	return authURL.String()
}

func startConfiguredOAuthCallbackServer(redirectURI string) (<-chan string, func(), error) {
	parsedURL, err := url.Parse(redirectURI)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid EPIC_REDIRECT_URI: %w", err)
	}
	if parsedURL.Scheme != "http" {
		return nil, nil, fmt.Errorf("EPIC_REDIRECT_URI must use http for the local callback")
	}
	if parsedURL.Host == "" || parsedURL.Path == "" {
		return nil, nil, fmt.Errorf("EPIC_REDIRECT_URI must include host, port, and path")
	}

	host := parsedURL.Hostname()
	if host == "" {
		host = "127.0.0.1"
	}
	port := parsedURL.Port()
	if port == "" {
		return nil, nil, fmt.Errorf("EPIC_REDIRECT_URI must include a fixed port")
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen on %s: %w", parsedURL.Host, err)
	}

	codeCh := make(chan string, 1)
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc(parsedURL.Path, func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Missing Epic authorization code."))
			return
		}

		select {
		case codeCh <- code:
		default:
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body><h3>Rockpload authentication complete.</h3><p>You can close this tab.</p></body></html>"))
	})

	go func() {
		err := server.Serve(listener)
		if err != nil && err != http.ErrServerClosed && logger.Rlogger != nil {
			logger.Rlogger.Error("Epic OAuth callback server failed", "err", err)
		}
	}()

	stopServer := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		_ = listener.Close()
	}

	return codeCh, stopServer, nil
}

func requestConfiguredToken(oauth config.EpicOAuthConfig, params map[string]string) (*rlapi.TokenResponse, error) {
	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}

	req, err := http.NewRequest(http.MethodPost, oauth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "UELauncher/11.0.1-14907503+++Portal+Release-Live Windows/10.0.19041.1.256.64bit")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(oauth.ClientID+":"+oauth.ClientSecret)))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			ErrorCode    string `json:"errorCode"`
			ErrorMessage string `json:"errorMessage"`
			Error        string `json:"error"`
			Description  string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &errorResp)
		return nil, fmt.Errorf("Epic token exchange failed: %s %s %s %s", resp.Status, errorResp.ErrorCode, errorResp.Error, firstNonEmpty(errorResp.ErrorMessage, errorResp.Description))
	}

	var tokenResp rlapi.TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}
	return &tokenResp, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
