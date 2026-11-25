package config

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

const (
	OpenReplayIDPlaceholder      = "{REPLAY_ID}"
	WebsiteBridgePortPlaceholder = "{PORT}"
)

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
		return "Bearer " + sc.bearerToken()
	}

	return sc.Token
}

func (sc *StorageConfig) bearerToken() string {
	token := strings.TrimSpace(sc.Token)

	return strings.TrimSpace(strings.TrimPrefix(token, "Bearer"))
}

func (sc *StorageConfig) HasBlankToken() bool {
	switch sc.TokenStyle {
	case NoToken:
		return false
	case BearerToken:
		return sc.bearerToken() == ""
	default:
		return strings.TrimSpace(sc.Token) == ""
	}
}

func (sc *StorageConfig) OpenReplayTarget() (string, OpenReplayTarget, bool) {
	if sc.OpenReplayStyle != OpenReplayEnabled || sc.UploadStyle == LocalFileCopy || sc.HasBlankToken() {
		return "", OpenReplayTarget{}, false
	}

	origin, err := ParseWebsiteOrigin(sc.OpenReplayOrigin)
	if err != nil || ValidateOpenReplayPath(sc.OpenReplayPath) != nil {
		return "", OpenReplayTarget{}, false
	}

	if _, err := parseSecureURL(sc.URL); err != nil {
		return "", OpenReplayTarget{}, false
	}

	target := OpenReplayTarget{DownloadURL: strings.TrimSuffix(strings.TrimSpace(sc.URL), "/") + sc.OpenReplayPath}
	if sc.TokenStyle != NoToken {
		target.Authorization = sc.AuthorizationHeader()
	}

	return origin, target, true
}

func (a *AppConfig) openReplayStorages() map[string]*StorageConfig {
	storages := map[string]*StorageConfig{}

	for _, storage := range a.StorageSettings.Get() {
		origin, _, ok := storage.OpenReplayTarget()
		if !ok {
			continue
		}

		if _, exists := storages[origin]; !exists {
			storages[origin] = storage
		}
	}

	return storages
}

func (a *AppConfig) OpenReplayTargets() map[string]OpenReplayTarget {
	targets := map[string]OpenReplayTarget{}

	for origin, storage := range a.openReplayStorages() {
		_, target, _ := storage.OpenReplayTarget()
		targets[origin] = target
	}

	return targets
}

func (a *AppConfig) IsOpenReplayStorage(storage *StorageConfig) bool {
	if !a.BehaviorConfig.AllowWebsiteReplay.Get() {
		return false
	}

	origin, _, ok := storage.OpenReplayTarget()

	return ok && a.openReplayStorages()[origin] == storage
}
