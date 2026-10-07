package rlgame

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Content of the file before the WebPort key was added (Heroic install, June 2026)
const statsAPIWithoutWebPort = "[TAGame.MatchStatsExporter_TA]\r\n" +
	"\r\n" +
	"; Port the client will listen for connections on\r\n" +
	"Port=49123\r\n" +
	"\r\n" +
	"; How many times per second the game sends the update state (capped at 120, 0 disables this feature)\r\n" +
	"PacketSendRate=0"

// Content of the file after the WebPort key was added (Steam install, August 2026)
const statsAPIWithWebPort = "[TAGame.MatchStatsExporter_TA]\r\n" +
	"\r\n" +
	"; Port the client will listen for tcp connections on (must be different than WebPort, set to 0 to disable)\r\n" +
	"Port=49123\r\n" +
	"\r\n" +
	"; Port the client will listen for web connections on (must be different than Port, set to 0 to disable)\r\n" +
	"WebPort=49124\r\n" +
	"\r\n" +
	"; How many times per second the game sends the update state (capped at 120, 0 disables this feature)\r\n" +
	"PacketSendRate=1"

func TestParseRealFiles(t *testing.T) {
	tests := []struct {
		name    string
		content string
		keys    []string
	}{
		{"without WebPort", statsAPIWithoutWebPort, []string{KeyPort, KeyPacketSendRate}},
		{"with WebPort", statsAPIWithWebPort, []string{KeyPort, KeyWebPort, KeyPacketSendRate}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := ParseStatsAPI([]byte(test.content))

			if !file.CanUseForm() {
				t.Fatalf("expected form to be usable, got problem %q", file.Problem)
			}

			if len(file.Entries) != len(test.keys) {
				t.Fatalf("expected %d entries, got %d", len(test.keys), len(file.Entries))
			}

			for i, key := range test.keys {
				if file.Entries[i].Key != key {
					t.Errorf("entry %d: expected key %q, got %q", i, key, file.Entries[i].Key)
				}

				if file.Entries[i].Comment == "" {
					t.Errorf("entry %q: expected the comment above it", key)
				}
			}

			if ports, err := file.Ports(); err != nil || ports != DefaultStatsAPIPorts() {
				t.Errorf("expected ports %+v, got %+v (%v)", DefaultStatsAPIPorts(), ports, err)
			}

			if string(file.Bytes()) != test.content {
				t.Errorf("unchanged file must be written back byte for byte")
			}
		})
	}
}

func TestSetOnlyChangesValue(t *testing.T) {
	file := ParseStatsAPI([]byte(statsAPIWithWebPort))

	if err := file.Set(KeyPacketSendRate, "30"); err != nil {
		t.Fatal(err)
	}

	if err := file.Set(KeyPort, " 50000 "); err != nil {
		t.Fatal(err)
	}

	expected := strings.Replace(statsAPIWithWebPort, "PacketSendRate=1", "PacketSendRate=30", 1)
	expected = strings.Replace(expected, "Port=49123", "Port=50000", 1)

	if string(file.Bytes()) != expected {
		t.Errorf("unexpected content:\n%q\nexpected:\n%q", file.Bytes(), expected)
	}

	if err := file.Set("NewKey", "1"); !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("expected unknown keys to be refused, got %v", err)
	}
}

func TestSetKeepsSpacingAndBOM(t *testing.T) {
	content := "\xEF\xBB\xBF[tagame.matchstatsexporter_ta]\nPort = 49123\n"
	file := ParseStatsAPI([]byte(content))

	if err := file.Set(KeyPort, "1234"); err != nil {
		t.Fatal(err)
	}

	expected := "\xEF\xBB\xBF[tagame.matchstatsexporter_ta]\nPort = 1234\n"
	if string(file.Bytes()) != expected {
		t.Errorf("got %q, expected %q", file.Bytes(), expected)
	}
}

func TestUnsupportedFilesFallBack(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		rawEditable bool
	}{
		{"missing section", "[TAGame.SomethingElse]\r\nPort=1\r\n", true},
		{"empty section", "[TAGame.MatchStatsExporter_TA]\r\n; nothing\r\n", true},
		{"duplicate key", "[TAGame.MatchStatsExporter_TA]\r\nPort=1\r\nPort=2\r\n", true},
		{"unreal array", "[TAGame.MatchStatsExporter_TA]\r\n+Port=1\r\n", true},
		{"not key value", "[TAGame.MatchStatsExporter_TA]\r\nPort\r\n", true},
		{"utf16", "\xFF\xFE[\x00T\x00", false},
		{"invalid utf8", "[TAGame.MatchStatsExporter_TA]\r\nPort=\xFF\r\n", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := ParseStatsAPI([]byte(test.content))

			if file.CanUseForm() {
				t.Fatal("expected the form to be refused")
			}

			if file.RawEditable != test.rawEditable {
				t.Errorf("expected RawEditable=%v", test.rawEditable)
			}

			if err := file.Set(KeyPort, "1"); err == nil {
				t.Error("expected Set to be refused")
			}

			if string(file.Bytes()) != test.content {
				t.Error("refused file must be written back byte for byte")
			}
		})
	}
}

func TestOtherSectionsAreIgnored(t *testing.T) {
	content := "[Other]\r\n+Array=1\r\nPort=1\r\n\r\n" + statsAPIWithoutWebPort
	file := ParseStatsAPI([]byte(content))

	if !file.CanUseForm() {
		t.Fatalf("unexpected problem %q", file.Problem)
	}

	if value, _ := file.Get(KeyPort); value != "49123" {
		t.Errorf("expected the StatsAPI port, got %q", value)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		key   string
		value string
		ok    bool
	}{
		{KeyPacketSendRate, "0", true},
		{KeyPacketSendRate, "120", true},
		{KeyPacketSendRate, "121", false},
		{KeyPacketSendRate, "-1", false},
		{KeyPacketSendRate, "fast", false},
		{KeyPacketSendRate, "29.97", true},
		{KeyPacketSendRate, "NaN", false},
		{KeyPort, "65535", true},
		{KeyPort, "65536", false},
		{KeyWebPort, "0", true},
		{"UnknownKey", "anything", true},
	}

	for _, test := range tests {
		err := ValidateStatsAPIValue(test.key, test.value)
		if (err == nil) != test.ok {
			t.Errorf("%s=%s: expected ok=%v, got %v", test.key, test.value, test.ok, err)
		}
	}

	file := ParseStatsAPI([]byte(statsAPIWithWebPort))
	file.Set(KeyWebPort, "49123")

	if err := file.Validate(); err == nil {
		t.Error("expected Port and WebPort to be required different")
	}

	file.Set(KeyPort, "0")
	file.Set(KeyWebPort, "0")

	if err := file.Validate(); err != nil {
		t.Errorf("both ports disabled should be valid, got %v", err)
	}
}

func TestRawTextKeepsLineEndings(t *testing.T) {
	file := ParseStatsAPI([]byte(statsAPIWithoutWebPort))

	raw := file.RawText()
	if strings.Contains(raw, "\r") {
		t.Fatal("raw text must use \\n for the editor")
	}

	if err := file.SetRawText(strings.Replace(raw, "PacketSendRate=0", "PacketSendRate=60", 1)); err != nil {
		t.Fatal(err)
	}

	expected := strings.Replace(statsAPIWithoutWebPort, "PacketSendRate=0", "PacketSendRate=60", 1)
	if string(file.Bytes()) != expected {
		t.Errorf("got %q, expected %q", file.Bytes(), expected)
	}
}

func TestSaveRefusesExternalChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), StatsAPIFileName)
	if err := os.WriteFile(path, []byte(statsAPIWithoutWebPort), 0644); err != nil {
		t.Fatal(err)
	}

	file, err := LoadStatsAPIFile(path)
	if err != nil {
		t.Fatal(err)
	}

	file.Set(KeyPacketSendRate, "30")

	if err := os.WriteFile(path, []byte(statsAPIWithWebPort), 0644); err != nil {
		t.Fatal(err)
	}

	if err := file.Save(); !errors.Is(err, ErrFileChanged) {
		t.Fatalf("expected ErrFileChanged, got %v", err)
	}

	file, _ = LoadStatsAPIFile(path)
	file.Set(KeyPacketSendRate, "30")

	if err := file.Save(); err != nil {
		t.Fatal(err)
	}

	saved, _ := os.ReadFile(path)
	if string(saved) != strings.Replace(statsAPIWithWebPort, "PacketSendRate=1", "PacketSendRate=30", 1) {
		t.Errorf("unexpected saved content %q", saved)
	}
}

func TestPorts(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected StatsAPIPorts
		ok       bool
	}{
		{"missing keys use game defaults", "[TAGame.MatchStatsExporter_TA]\r\nPacketSendRate=30\r\n", DefaultStatsAPIPorts(), true},
		{"tcp disabled", "[TAGame.MatchStatsExporter_TA]\r\nPort=0\r\nWebPort=50000\r\n", StatsAPIPorts{TCP: 0, Web: 50000}, true},
		{"invalid port", "[TAGame.MatchStatsExporter_TA]\r\nPort=abc\r\n", StatsAPIPorts{}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ports, err := ParseStatsAPI([]byte(test.content)).Ports()

			if (err == nil) != test.ok {
				t.Fatalf("expected ok=%v, got %v", test.ok, err)
			}

			if ports != test.expected {
				t.Errorf("expected %+v, got %+v", test.expected, ports)
			}
		})
	}
}

func TestIsSendRateDisabled(t *testing.T) {
	for value, expected := range map[string]bool{"0": true, "0.0": true, " 0 ": true, "30": false, "0.5": false, "": false} {
		if IsSendRateDisabled(value) != expected {
			t.Errorf("%q: expected %v", value, expected)
		}
	}
}
