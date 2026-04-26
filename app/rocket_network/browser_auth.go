package rocket_network

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"lexore/rockpload/app/tools"
	"lexore/rockpload/app/tools/logger"
)

const launcherClientID = "34a02cf8f4414e29b15921876da36f9a"
const epicLoginURL = "https://www.epicgames.com/id/login"
const epicLogoutURL = "https://www.epicgames.com/id/logout"

// AuthenticateWithBrowser opens the user's default browser and captures
// Epic's localhost callback automatically without requiring copy/paste.
func (ra *Auth) AuthenticateWithBrowser(timeout time.Duration) error {
	return ra.AuthenticateWithBrowserUntil(timeout, nil)
}

func (ra *Auth) AuthenticateWithInstalledBrowser(timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastErr error

	for {
		if err := ra.AuthenticateWithBrowserCookies(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ticker.C:
		case <-deadline:
			if lastErr != nil {
				return fmt.Errorf("timed out waiting for Epic browser session: %w", lastErr)
			}
			return fmt.Errorf("timed out waiting for Epic browser session")
		}
	}
}

func (ra *Auth) AuthenticateWithLauncherClient(timeout time.Duration, forceAccountPicker bool) error {
	if forceAccountPicker {
		ra.ClearToken()
		tools.OpenBrowser(epicLogoutURL)
		time.Sleep(2 * time.Second)
	}

	if err := ra.AuthenticateWithBrowserCookies(); err == nil {
		return nil
	}

	// The launcher client returns its usable code through Epic's JSON
	// endpoint, not through a normal browser redirect. Opening login here is
	// only to establish/refresh the user's Epic browser session; the app polls
	// installed browser cookies and requests the launcher code itself.
	tools.OpenBrowser(epicLoginURL)

	deadline := time.After(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastErr error

	for {
		if err := ra.AuthenticateWithBrowserCookies(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ticker.C:
		case <-deadline:
			if lastErr != nil {
				return fmt.Errorf("timed out waiting for Epic launcher session: %w", lastErr)
			}
			return fmt.Errorf("timed out waiting for Epic launcher session")
		}
	}
}

func (ra *Auth) AuthenticateWithBrowserUntil(timeout time.Duration, stop <-chan struct{}) error {
	codeCh, stopServer, err := startLocalAuthCallbackServer()
	if err != nil {
		return err
	}
	defer stopServer()

	ra.OpenAuthURL()

	select {
	case code := <-codeCh:
		return ra.AuthenticateWithCode(code)
	case <-stop:
		return fmt.Errorf("browser auth stopped")
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for Epic auth callback")
	}
}

func startLocalAuthCallbackServer() (<-chan string, func(), error) {
	cert, err := generateLocalhostCertificate()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate localhost certificate: %w", err)
	}

	listener, err := tls.Listen("tcp", ":443", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen on localhost:443: %w", err)
	}

	codeCh := make(chan string, 1)
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc("/launcher/authorized", func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Missing auth code."))
			return
		}

		select {
		case codeCh <- code:
		default:
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body><h3>Rockpload authentication complete.</h3><p>You can close this tab.</p></body></html>"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rockpload local auth endpoint"))
	})

	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Rlogger.Error("Local auth callback server failed", "err", err)
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

func startLocalBrowserLoginServer() (<-chan struct{}, func(), string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to listen for browser login completion: %w", err)
	}

	doneCh := make(chan struct{}, 1)
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc("/epic-login-complete", func(w http.ResponseWriter, r *http.Request) {
		select {
		case doneCh <- struct{}{}:
		default:
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body><h3>Rockpload is finishing Epic sign-in.</h3><p>You can close this tab once Rockpload shows your account.</p></body></html>"))
	})

	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Rlogger.Error("Local browser login server failed", "err", err)
		}
	}()

	stopServer := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		_ = listener.Close()
	}

	return doneCh, stopServer, "http://" + listener.Addr().String() + "/epic-login-complete", nil
}

func generateLocalhostCertificate() (tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	notBefore := time.Now().Add(-10 * time.Minute)
	notAfter := time.Now().Add(24 * time.Hour)
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	certificateDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	cert, err := tls.X509KeyPair(certificatePEM, privateKeyPEM)
	if err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}
