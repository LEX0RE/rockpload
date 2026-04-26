package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type EpicOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scope        string
	AuthURL      string
	TokenURL     string
}

func EpicAuthMode() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("EPIC_AUTH_MODE")))
	if mode == "" {
		// Launcher mode uses Epic's launcher client, which has the Rocket
		// League exchange-code permission that regular EOS clients lack.
		return "launcher"
	}
	return mode
}

func LoadDotEnv() {
	paths := []string{".env"}

	if executable, err := os.Executable(); err == nil {
		// Packaged/dev-installed binaries are often launched outside the repo,
		// so also look next to the executable for local configuration.
		paths = append(paths, filepath.Join(filepath.Dir(executable), ".env"))
	}

	for _, path := range paths {
		loadDotEnvFile(path)
	}
}

func EpicOAuth() (EpicOAuthConfig, bool) {
	cfg := EpicOAuthConfig{
		ClientID:     strings.TrimSpace(os.Getenv("EPIC_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("EPIC_CLIENT_SECRET")),
		RedirectURI:  strings.TrimSpace(os.Getenv("EPIC_REDIRECT_URI")),
		Scope:        strings.TrimSpace(os.Getenv("EPIC_SCOPE")),
		AuthURL:      strings.TrimSpace(os.Getenv("EPIC_AUTH_URL")),
		TokenURL:     strings.TrimSpace(os.Getenv("EPIC_TOKEN_URL")),
	}

	if cfg.RedirectURI == "" {
		cfg.RedirectURI = "http://127.0.0.1:8765/epic/callback"
	}
	if cfg.Scope == "" {
		cfg.Scope = "basic_profile"
	}
	if cfg.AuthURL == "" {
		cfg.AuthURL = "https://www.epicgames.com/id/authorize"
	}
	if cfg.TokenURL == "" {
		cfg.TokenURL = "https://account-public-service-prod03.ol.epicgames.com/account/api/oauth/token"
	}

	return cfg, cfg.ClientID != "" && cfg.ClientSecret != ""
}

func loadDotEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}
