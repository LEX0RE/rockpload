//go:build windows

package rocket_network

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unsafe"

	"lexore/rockpload/app/tools/logger"

	"golang.org/x/sys/windows"
	_ "modernc.org/sqlite"
)

const epicRedirectURL = "https://www.epicgames.com/id/api/redirect?clientId=" + launcherClientID + "&responseType=code"

type browserCookie struct {
	Name  string
	Value string
}

type chromiumInstall struct {
	Name        string
	UserDataDir string
}

type firefoxInstall struct {
	Name        string
	ProfilesDir string
}

type browserCookieJar struct {
	Browser string
	Cookies []browserCookie
}

func (ra *Auth) AuthenticateWithBrowserCookies() error {
	jars, err := loadEpicBrowserCookieJars()
	if err != nil {
		return err
	}

	var errs []string
	for _, jar := range jars {
		authCode, err := requestEpicAuthCode(jar.Cookies)
		if err == nil {
			if logger.Rlogger != nil {
				logger.Rlogger.Debug("Loaded Epic launcher auth code from browser", "browser", jar.Browser)
			}
			return ra.AuthenticateWithCode(authCode)
		}
		errs = append(errs, jar.Browser+": "+err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("no signed-in Epic browser session found (%s)", strings.Join(errs, "; "))
	}
	return fmt.Errorf("no Epic browser cookies found")
}

func loadEpicBrowserCookieJars() ([]browserCookieJar, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, fmt.Errorf("LOCALAPPDATA is not set")
	}
	appData := os.Getenv("APPDATA")

	installs := []chromiumInstall{
		// Chromium-family browsers store cookies in SQLite and encrypt values
		// with a per-browser DPAPI master key from "Local State".
		{Name: "Brave", UserDataDir: filepath.Join(localAppData, "BraveSoftware", "Brave-Browser", "User Data")},
		{Name: "Brave Beta", UserDataDir: filepath.Join(localAppData, "BraveSoftware", "Brave-Browser-Beta", "User Data")},
		{Name: "Brave Nightly", UserDataDir: filepath.Join(localAppData, "BraveSoftware", "Brave-Browser-Nightly", "User Data")},
		{Name: "Chrome", UserDataDir: filepath.Join(localAppData, "Google", "Chrome", "User Data")},
		{Name: "Chrome Beta", UserDataDir: filepath.Join(localAppData, "Google", "Chrome Beta", "User Data")},
		{Name: "Chrome Canary", UserDataDir: filepath.Join(localAppData, "Google", "Chrome SxS", "User Data")},
		{Name: "Chromium", UserDataDir: filepath.Join(localAppData, "Chromium", "User Data")},
		{Name: "Edge", UserDataDir: filepath.Join(localAppData, "Microsoft", "Edge", "User Data")},
		{Name: "Edge Beta", UserDataDir: filepath.Join(localAppData, "Microsoft", "Edge Beta", "User Data")},
		{Name: "Edge Dev", UserDataDir: filepath.Join(localAppData, "Microsoft", "Edge Dev", "User Data")},
		{Name: "Edge Canary", UserDataDir: filepath.Join(localAppData, "Microsoft", "Edge SxS", "User Data")},
		{Name: "Opera", UserDataDir: filepath.Join(appData, "Opera Software", "Opera Stable")},
		{Name: "Opera GX", UserDataDir: filepath.Join(appData, "Opera Software", "Opera GX Stable")},
		{Name: "Vivaldi", UserDataDir: filepath.Join(localAppData, "Vivaldi", "User Data")},
		{Name: "Arc", UserDataDir: filepath.Join(localAppData, "Packages", "TheBrowserCompany.Arc_ttt1ap7aakyb4", "LocalCache", "Local", "Arc", "User Data")},
		{Name: "Yandex", UserDataDir: filepath.Join(localAppData, "Yandex", "YandexBrowser", "User Data")},
	}

	var errs []string
	var jars []browserCookieJar
	for _, install := range installs {
		cookies, err := loadEpicChromiumCookies(install)
		if err == nil && len(cookies) > 0 {
			if logger.Rlogger != nil {
				logger.Rlogger.Debug("Loaded Epic cookies from browser", "browser", install.Name)
			}
			jars = append(jars, browserCookieJar{Browser: install.Name, Cookies: cookies})
			continue
		}
		if err != nil {
			errs = append(errs, install.Name+": "+err.Error())
		}
	}

	if appData != "" {
		firefoxInstalls := []firefoxInstall{
			// Firefox-family browsers store cookie values unencrypted in
			// cookies.sqlite; Windows account protection still applies to files.
			{Name: "Firefox", ProfilesDir: filepath.Join(appData, "Mozilla", "Firefox", "Profiles")},
			{Name: "Firefox Developer Edition", ProfilesDir: filepath.Join(appData, "Mozilla", "Firefox Developer Edition", "Profiles")},
			{Name: "Firefox Nightly", ProfilesDir: filepath.Join(appData, "Mozilla", "Firefox Nightly", "Profiles")},
			{Name: "LibreWolf", ProfilesDir: filepath.Join(appData, "LibreWolf", "Profiles")},
			{Name: "Waterfox", ProfilesDir: filepath.Join(appData, "Waterfox", "Profiles")},
		}
		for _, install := range firefoxInstalls {
			cookies, err := loadEpicFirefoxCookies(install.ProfilesDir)
			if err == nil && len(cookies) > 0 {
				if logger.Rlogger != nil {
					logger.Rlogger.Debug("Loaded Epic cookies from browser", "browser", install.Name)
				}
				jars = append(jars, browserCookieJar{Browser: install.Name, Cookies: cookies})
			} else if err != nil {
				errs = append(errs, install.Name+": "+err.Error())
			}
		}
	}

	if len(jars) > 0 {
		return jars, nil
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("no readable Epic browser cookies found (%s)", strings.Join(errs, "; "))
	}
	return nil, fmt.Errorf("no Epic browser cookies found")
}

func loadEpicChromiumCookies(install chromiumInstall) ([]browserCookie, error) {
	if _, err := os.Stat(install.UserDataDir); err != nil {
		return nil, err
	}

	masterKey, err := chromiumMasterKey(filepath.Join(install.UserDataDir, "Local State"))
	if err != nil {
		return nil, err
	}

	profiles, err := chromiumProfiles(install.UserDataDir)
	if err != nil {
		return nil, err
	}

	cookieByName := map[string]browserCookie{}
	var errs []string
	for _, profile := range profiles {
		for _, dbPath := range chromiumCookieDBPaths(profile) {
			cookies, err := readChromiumCookieDB(dbPath, masterKey)
			if err != nil {
				errs = append(errs, filepath.Base(profile)+": "+err.Error())
				continue
			}
			for _, cookie := range cookies {
				cookieByName[cookie.Name] = cookie
			}
		}
	}

	cookies := make([]browserCookie, 0, len(cookieByName))
	for _, cookie := range cookieByName {
		cookies = append(cookies, cookie)
	}
	sort.Slice(cookies, func(i, j int) bool {
		return cookies[i].Name < cookies[j].Name
	})
	if len(cookies) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return cookies, nil
}

func loadEpicFirefoxCookies(profilesDir string) ([]browserCookie, error) {
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, err
	}

	cookieByName := map[string]browserCookie{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dbPath := filepath.Join(profilesDir, entry.Name(), "cookies.sqlite")
		cookies, err := readFirefoxCookieDB(dbPath)
		if err != nil {
			continue
		}
		for _, cookie := range cookies {
			cookieByName[cookie.Name] = cookie
		}
	}

	cookies := make([]browserCookie, 0, len(cookieByName))
	for _, cookie := range cookieByName {
		cookies = append(cookies, cookie)
	}
	sort.Slice(cookies, func(i, j int) bool {
		return cookies[i].Name < cookies[j].Name
	})
	return cookies, nil
}

func chromiumProfiles(userDataDir string) ([]string, error) {
	entries, err := os.ReadDir(userDataDir)
	if err != nil {
		return nil, err
	}

	profileByPath := map[string]struct{}{}
	addProfile := func(path string) {
		profileByPath[path] = struct{}{}
	}

	// Opera-style user data dirs can keep Network/Cookies at the root.
	addProfile(userDataDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "Default" || strings.HasPrefix(name, "Profile ") || strings.HasPrefix(name, "Guest Profile") {
			addProfile(filepath.Join(userDataDir, name))
		}
	}

	profiles := make([]string, 0, len(profileByPath))
	for profile := range profileByPath {
		profiles = append(profiles, profile)
	}
	sort.Strings(profiles)
	return profiles, nil
}

func chromiumCookieDBPaths(profile string) []string {
	return []string{
		filepath.Join(profile, "Network", "Cookies"),
		filepath.Join(profile, "Cookies"),
	}
}

func chromiumMasterKey(localStatePath string) ([]byte, error) {
	data, err := os.ReadFile(localStatePath)
	if err != nil {
		return nil, err
	}

	var localState struct {
		OSCrypt struct {
			EncryptedKey string `json:"encrypted_key"`
		} `json:"os_crypt"`
	}
	if err := json.Unmarshal(data, &localState); err != nil {
		return nil, err
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(localState.OSCrypt.EncryptedKey)
	if err != nil {
		return nil, err
	}
	encryptedKey = bytes.TrimPrefix(encryptedKey, []byte("DPAPI"))
	return dpapiDecrypt(encryptedKey)
}

func readChromiumCookieDB(dbPath string, masterKey []byte) ([]browserCookie, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "rockpload-cookies-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	tempDB := filepath.Join(tempDir, "Cookies")
	if err := copyFile(dbPath, tempDB); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", tempDB+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT name, value, encrypted_value
		FROM cookies
		WHERE host_key LIKE '%epicgames.com%'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cookies []browserCookie
	decryptFailures := 0
	for rows.Next() {
		var name string
		var value string
		var encryptedValue []byte
		if err := rows.Scan(&name, &value, &encryptedValue); err != nil {
			continue
		}

		if value == "" && len(encryptedValue) > 0 {
			decrypted, err := decryptChromiumCookie(encryptedValue, masterKey)
			if err != nil {
				decryptFailures++
				continue
			}
			value = decrypted
		}
		if name != "" && value != "" {
			cookies = append(cookies, browserCookie{Name: name, Value: value})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cookies) == 0 && decryptFailures > 0 {
		return nil, fmt.Errorf("found Epic cookies but could not decrypt them")
	}
	return cookies, nil
}

func readFirefoxCookieDB(dbPath string) ([]browserCookie, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "rockpload-firefox-cookies-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	tempDB := filepath.Join(tempDir, "cookies.sqlite")
	if err := copyFile(dbPath, tempDB); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", tempDB+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT name, value
		FROM moz_cookies
		WHERE host LIKE '%epicgames.com%'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cookies []browserCookie
	for rows.Next() {
		var cookie browserCookie
		if err := rows.Scan(&cookie.Name, &cookie.Value); err != nil {
			continue
		}
		if cookie.Name != "" && cookie.Value != "" {
			cookies = append(cookies, cookie)
		}
	}
	return cookies, rows.Err()
}

func decryptChromiumCookie(encryptedValue []byte, masterKey []byte) (string, error) {
	if bytes.HasPrefix(encryptedValue, []byte("v10")) || bytes.HasPrefix(encryptedValue, []byte("v11")) || bytes.HasPrefix(encryptedValue, []byte("v20")) {
		if len(encryptedValue) < 3+12+16 {
			return "", fmt.Errorf("encrypted cookie is too short")
		}

		block, err := aes.NewCipher(masterKey)
		if err != nil {
			return "", err
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return "", err
		}

		nonce := encryptedValue[3:15]
		ciphertext := encryptedValue[15:]
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return "", err
		}
		return normalizeChromiumCookieValue(plaintext), nil
	}

	plaintext, err := dpapiDecrypt(encryptedValue)
	if err != nil {
		return "", err
	}
	return normalizeChromiumCookieValue(plaintext), nil
}

func normalizeChromiumCookieValue(value []byte) string {
	// Chrome 130+ prefixes decrypted cookie values with a 32-byte host hash.
	// The HTTP Cookie header only wants the original value, so strip it when
	// the prefix looks binary and the remainder looks header-safe.
	if len(value) > 32 && hasControlBytes(value[:32]) && !hasControlBytes(value[32:]) {
		return string(value[32:])
	}
	return string(value)
}

func hasControlBytes(value []byte) bool {
	for _, b := range value {
		if b < 0x20 || b == 0x7f {
			return true
		}
	}
	return false
}

func requestEpicAuthCode(cookies []browserCookie) (string, error) {
	req, err := http.NewRequest(http.MethodGet, epicRedirectURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	req.Header.Set("Referer", "https://www.epicgames.com/id/login")
	req.Header.Set("Cookie", cookieHeader(cookies))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Epic auth code request failed: %s", resp.Status)
	}

	var payload struct {
		AuthorizationCode *string `json:"authorizationCode"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if payload.AuthorizationCode == nil || strings.TrimSpace(*payload.AuthorizationCode) == "" {
		return "", fmt.Errorf("Epic browser session is not signed in")
	}

	return strings.TrimSpace(*payload.AuthorizationCode), nil
}

func cookieHeader(cookies []browserCookie) string {
	parts := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}
	return strings.Join(parts, "; ")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func dpapiDecrypt(data []byte) ([]byte, error) {
	var in windows.DataBlob
	var out windows.DataBlob

	in.Size = uint32(len(data))
	if len(data) > 0 {
		in.Data = &data[0]
	}

	err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, 0, &out)
	if err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))

	decrypted := unsafe.Slice(out.Data, out.Size)
	result := make([]byte, len(decrypted))
	copy(result, decrypted)
	return result, nil
}
