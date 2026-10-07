package rlgame

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func createInstall(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(dir, "TAGame", "Config"), 0755); err != nil {
		t.Fatal(err)
	}
}

func createWineLog(t *testing.T, prefix string, drives map[string]string, baseDirectory string) string {
	t.Helper()

	logFolder := filepath.Join(prefix, "drive_c", "users", "steamuser", "Documents", "My Games", "Rocket League", "TAGame", "Logs")
	if err := os.MkdirAll(logFolder, 0755); err != nil {
		t.Fatal(err)
	}

	dosdevices := filepath.Join(prefix, "dosdevices")
	if err := os.MkdirAll(dosdevices, 0755); err != nil {
		t.Fatal(err)
	}

	for drive, target := range drives {
		if err := os.Symlink(target, filepath.Join(dosdevices, drive)); err != nil {
			t.Fatal(err)
		}
	}

	content := "Log: Log file open, 10/05/26 12:00:00\r\n" +
		"Init: Command line: -AUTH_PASSWORD=secret\r\n" +
		"Init: Base directory: " + baseDirectory + "\r\n" +
		"[0000.49] Log: Purging 3-day cache\r\n"

	logPath := filepath.Join(logFolder, LaunchLogName)
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	return logPath
}

func TestInstallDirFromWineLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Wine paths are only resolved outside Windows")
	}

	root := t.TempDir()

	tests := []struct {
		name          string
		prefix        string
		drives        map[string]string
		install       string
		baseDirectory string
	}{
		{
			name:          "heroic",
			prefix:        filepath.Join(root, "Heroic", "Prefixes", "default", "Rocket League"),
			drives:        map[string]string{"c:": "../drive_c", "x:": root},
			install:       filepath.Join(root, "Games", "Heroic", "rocketleague"),
			baseDirectory: `X:\Games\Heroic\rocketleague\Binaries\Win64\`,
		},
		{
			name:          "steam proton",
			prefix:        filepath.Join(root, "Steam", "steamapps", "compatdata", "252950", "pfx"),
			drives:        map[string]string{"c:": "../drive_c", "s:": filepath.Join(root, "Steam")},
			install:       filepath.Join(root, "Steam", "steamapps", "common", "rocketleague"),
			baseDirectory: `S:\steamapps\common\rocketleague\Binaries\Win64\`,
		},
		{
			name:          "drive c without dosdevices link",
			prefix:        filepath.Join(root, "NoLinks"),
			drives:        map[string]string{},
			install:       filepath.Join(root, "NoLinks", "drive_c", "Program Files", "Epic Games", "rocketleague"),
			baseDirectory: `C:\Program Files\Epic Games\rocketleague\Binaries\Win64\`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			createInstall(t, test.install)
			logPath := createWineLog(t, test.prefix, test.drives, test.baseDirectory)

			dir, err := InstallDirFromLog(logPath)
			if err != nil {
				t.Fatal(err)
			}

			expected, _ := filepath.EvalSymlinks(test.install)
			actual, _ := filepath.EvalSymlinks(dir)
			if actual != expected {
				t.Errorf("expected %q, got %q", expected, dir)
			}
		})
	}
}

func TestInstallDirFromLogMissingInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Wine paths are only resolved outside Windows")
	}

	root := t.TempDir()
	logPath := createWineLog(t, filepath.Join(root, "prefix"), map[string]string{"x:": root}, `X:\missing\Binaries\Win64\`)

	if _, err := InstallDirFromLog(logPath); err == nil {
		t.Error("expected an error when the install folder does not exist")
	}
}

func TestLocatorManualFirst(t *testing.T) {
	manual := t.TempDir()
	createInstall(t, manual)

	locator := NewLocator()
	locator.SetManualDir(manual)
	locator.Refresh()

	active, ok := locator.Active()
	if !ok || active.Dir != filepath.Clean(manual) || active.Source != "Manual" {
		t.Errorf("expected the manual install first, got %+v", active)
	}

	invalid := NewLocator()
	invalid.SetManualDir(filepath.Join(manual, "missing"))
	for _, install := range invalid.Refresh() {
		if install.Source == "Manual" {
			t.Error("an invalid manual folder must be ignored")
		}
	}
}

func TestIsWindowsDrivePath(t *testing.T) {
	tests := map[string]bool{
		`C:\Program Files`: true,
		`s:/steamapps`:     true,
		`/home/user`:       false,
		`C:`:               false,
		`1:\folder`:        false,
	}

	for path, expected := range tests {
		if isWindowsDrivePath(path) != expected {
			t.Errorf("%q: expected %v", path, expected)
		}
	}
}

func TestStatsAPIConfigPathPrefersUserFile(t *testing.T) {
	dir := t.TempDir()
	createInstall(t, dir)
	install := Install{Dir: dir}

	defaultPath := filepath.Join(dir, "TAGame", "Config", StatsAPIFileName)
	if install.StatsAPIConfigPath() != defaultPath {
		t.Errorf("expected %q when %s is missing", defaultPath, StatsAPIUserFileName)
	}

	userPath := filepath.Join(dir, "TAGame", "Config", StatsAPIUserFileName)
	if err := os.WriteFile(userPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	if install.StatsAPIConfigPath() != userPath {
		t.Errorf("expected %q to take priority", userPath)
	}
}
