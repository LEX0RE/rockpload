package config

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

const OpenReplayIDPlaceholder = "{REPLAY_ID}"

var (
	ErrWebsiteOriginInvalid  = errors.New("use the website address, like https://example.com")
	ErrWebsiteOriginInsecure = errors.New("only localhost can use http, use https for other websites")
	ErrOpenReplayPathInvalid = errors.New("the path must contain " + OpenReplayIDPlaceholder)
)

func ParseWebsiteOrigin(raw string) (string, error) {
	parsed, err := parseSecureURL(raw)
	if err != nil {
		return "", err
	}

	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host), nil
}

func ValidateOpenReplayPath(path string) error {
	if !strings.HasPrefix(path, "/") || !strings.Contains(path, OpenReplayIDPlaceholder) {
		return ErrOpenReplayPathInvalid
	}

	return nil
}

func parseSecureURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, ErrWebsiteOriginInvalid
	}

	switch strings.ToLower(parsed.Scheme) {
	case "https":
	case "http":
		if !isLoopbackHost(parsed.Hostname()) {
			return nil, ErrWebsiteOriginInsecure
		}
	default:
		return nil, ErrWebsiteOriginInvalid
	}

	return parsed, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type OpenReplayTarget struct {
	DownloadURL   string
	Authorization string
}

func (sc *StorageConfig) AuthorizationHeader() string {
	if sc.TokenStyle == BearerToken {
		token := strings.TrimSpace(sc.Token)
		token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer"))

		return "Bearer " + token
	}

	return sc.Token
}

func (a *AppConfig) OpenReplayTargets() map[string]OpenReplayTarget {
	targets := map[string]OpenReplayTarget{}

	for _, storage := range a.StorageSettings.Get() {
		if storage.OpenReplayStyle != OpenReplayEnabled || storage.UploadStyle == LocalFileCopy {
			continue
		}

		origin, err := ParseWebsiteOrigin(storage.OpenReplayOrigin)
		if err != nil || ValidateOpenReplayPath(storage.OpenReplayPath) != nil {
			continue
		}

		if _, err := parseSecureURL(storage.URL); err != nil {
			continue
		}

		if _, exists := targets[origin]; exists {
			continue
		}

		target := OpenReplayTarget{DownloadURL: strings.TrimSuffix(strings.TrimSpace(storage.URL), "/") + storage.OpenReplayPath}
		if storage.TokenStyle != NoToken {
			target.Authorization = storage.AuthorizationHeader()
		}

		targets[origin] = target
	}

	return targets
}
