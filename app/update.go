package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/LEX0RE/rockpload/app/tools/logger"
)

type UpdateSource string

const (
	UpdateSourceGitHub  UpdateSource = "GitHub"
	UpdateSourceWebsite UpdateSource = "Rocky website"
)

type UpdateInfo struct {
	Version  string       `json:"version"`
	URL      string       `json:"url"`
	Checksum string       `json:"checksum"`
	Source   UpdateSource `json:"-"`
}

type Updater struct {
	UpdateInfo *UpdateInfo
}

const (
	githubReleasesURL = "https://api.github.com/repos/LEX0RE/rockpload/releases/latest"
	websiteUpdateURL  = "https://lexore.ca/rocky/api/rockpload"
)

func NewUpdater() *Updater {
	logger.FuncDebug()

	return &Updater{}
}

func (u *Updater) CheckForUpdate(currentVersion string) (bool, error) {
	logger.FuncDebug()

	info, needUpdate, err := checkGitHubUpdate(currentVersion)
	if err != nil {
		logger.Rlogger.Warn("GitHub update check failed, falling back to Rocky website", slog.Any("err", err))

		info, needUpdate, err = checkWebsiteUpdate(currentVersion)
		if err != nil {
			return false, err
		}
	}

	if !needUpdate {
		logger.Rlogger.Info("No update available")
		return false, nil
	}

	logger.Rlogger.Info("New version available: "+info.Version, slog.Any("current", currentVersion), slog.Any("source", info.Source))
	u.UpdateInfo = info

	return true, nil
}

func formatVersion(v string) string {
	if len(v) > 0 && v[0] != 'v' {
		return "v" + v
	}
	return v
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

func checkGitHubUpdate(currentVersion string) (*UpdateInfo, bool, error) {
	logger.FuncDebug()

	req, err := http.NewRequest(http.MethodGet, githubReleasesURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("github release check failed with status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, false, err
	}

	if release.TagName == "" {
		return nil, false, fmt.Errorf("no release found on GitHub")
	}

	if semver.Compare(formatVersion(release.TagName), formatVersion(currentVersion)) != 1 {
		return nil, false, nil
	}

	assetSuffix := "-" + runtime.GOOS
	switch runtime.GOOS {
	case "windows":
		assetSuffix += ".exe"
	case "android":
		assetSuffix += ".apk"
	}

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, assetSuffix) {
			return &UpdateInfo{
				Version: strings.TrimPrefix(release.TagName, "v"),
				URL:     asset.BrowserDownloadURL,
				Source:  UpdateSourceGitHub,
			}, true, nil
		}
	}

	return nil, false, fmt.Errorf("no matching asset found for %s in GitHub release %s", runtime.GOOS, release.TagName)
}

func checkWebsiteUpdate(currentVersion string) (*UpdateInfo, bool, error) {
	logger.FuncDebug()

	url := fmt.Sprintf(websiteUpdateURL+"?os=%s", runtime.GOOS)

	resp, err := http.Get(url)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	var info UpdateInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, false, err
	}

	if info.Version == "" {
		return nil, false, nil
	}

	if semver.Compare(formatVersion(info.Version), formatVersion(currentVersion)) != 1 {
		return nil, false, nil
	}

	info.Source = UpdateSourceWebsite

	return &info, true, nil
}

func verifyChecksum(data []byte, expected string) bool {
	logger.FuncDebug()

	if expected == "" {
		return true
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]) == expected
}
