package config

import (
	"encoding/json"
	"testing"
)

func openReplayStorage(name string, tokenStyle TokenStyleType, token string) *StorageConfig {
	return &StorageConfig{
		Name:             name,
		UploadStyle:      MultipartUpload,
		URL:              "https://example.com/api",
		TokenStyle:       tokenStyle,
		Token:            token,
		OpenReplayStyle:  OpenReplayEnabled,
		OpenReplayOrigin: "https://example.com",
		OpenReplayPath:   "/replays/" + OpenReplayIDPlaceholder + "/file",
	}
}

func testAppConfig(allowWebsiteReplay bool, storages ...*StorageConfig) *AppConfig {
	cfg := &AppConfig{StorageSettings: NewSetting(storageListConfig(storages), nil)}
	cfg.BehaviorConfig.AllowWebsiteReplay = NewSetting(allowWebsiteReplay, nil)

	return cfg
}

func TestOpenReplayTargetSkipsBlankToken(t *testing.T) {
	tests := []struct {
		name       string
		tokenStyle TokenStyleType
		token      string
		want       bool
		wantAuth   string
	}{
		{"no token style", NoToken, "", true, ""},
		{"raw token", RawToken, "key", true, "key"},
		{"blank raw token", RawToken, "  ", false, ""},
		{"bearer token", BearerToken, "key", true, "Bearer key"},
		{"blank bearer token", BearerToken, "", false, ""},
		{"bearer prefix only", BearerToken, "Bearer ", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, target, ok := openReplayStorage("site", tt.tokenStyle, tt.token).OpenReplayTarget()
			if ok != tt.want {
				t.Fatalf("OpenReplayTarget() ok = %v, want %v", ok, tt.want)
			}

			if target.Authorization != tt.wantAuth {
				t.Errorf("Authorization = %q, want %q", target.Authorization, tt.wantAuth)
			}
		})
	}
}

func TestOpenReplayTargetsFirstStoragePerOriginWins(t *testing.T) {
	blank := openReplayStorage("blank", RawToken, "")
	first := openReplayStorage("first", RawToken, "first-key")
	second := openReplayStorage("second", RawToken, "second-key")
	cfg := testAppConfig(true, blank, first, second)

	targets := cfg.OpenReplayTargets()
	if got := targets["https://example.com"].Authorization; got != "first-key" {
		t.Errorf("Authorization = %q, want first-key", got)
	}

	for storage, want := range map[*StorageConfig]bool{blank: false, first: true, second: false} {
		if got := cfg.IsOpenReplayStorage(storage); got != want {
			t.Errorf("IsOpenReplayStorage(%s) = %v, want %v", storage.Name, got, want)
		}
	}
}

func TestIsOpenReplayStorageFollowsGlobalSetting(t *testing.T) {
	storage := openReplayStorage("site", NoToken, "")

	if testAppConfig(false, storage).IsOpenReplayStorage(storage) {
		t.Error("IsOpenReplayStorage() = true while website replays are turned off")
	}
}

func TestFieldLinkResolve(t *testing.T) {
	values := map[string]string{WebsiteBridgePortPlaceholder: "31321"}

	fieldLink, ok := BALLCHASING_STORAGE.FieldLink("open_replay_style")
	if !ok {
		t.Fatal("FieldLink() ok = false for the ballchasing Open Replay setting")
	}

	link, ok := fieldLink.Resolve(values)
	if !ok {
		t.Fatal("Resolve() ok = false")
	}

	if want := "https://ballchasing.com/play-in-rocket-league?port=31321"; link.String() != want {
		t.Errorf("Resolve() = %q, want %q", link.String(), want)
	}

	if _, ok := ROCKY_STORAGE.FieldLink("open_replay_style"); ok {
		t.Error("FieldLink() ok = true for a storage without a link")
	}

	if _, ok := (FieldLink{Text: "Open", URL: "http://example.com"}).Resolve(values); ok {
		t.Error("Resolve() accepted a link without https")
	}
}

func TestFieldLinkKeysMatchStorageFields(t *testing.T) {
	if len(FIELD_LINK_KEYS) == 0 {
		t.Fatal("FIELD_LINK_KEYS is empty")
	}

	for _, key := range FIELD_LINK_KEYS {
		if _, ok := storageFieldByJSON[key]; !ok {
			t.Errorf("field link %q is not a storage field", key)
		}
	}
}

func TestBallchasingDownloadTarget(t *testing.T) {
	storage := *BALLCHASING_STORAGE
	storage.Token = "api-key"

	origin, target, ok := storage.OpenReplayTarget()
	if !ok {
		t.Fatal("OpenReplayTarget() ok = false")
	}

	if origin != "https://ballchasing.com" {
		t.Errorf("origin = %q", origin)
	}

	if want := "https://ballchasing.com/api/replays/" + OpenReplayIDPlaceholder + "/file"; target.DownloadURL != want {
		t.Errorf("DownloadURL = %q, want %q", target.DownloadURL, want)
	}

	if target.Authorization != "api-key" {
		t.Errorf("Authorization = %q, want api-key", target.Authorization)
	}
}

func TestBallchasingPatchTitleMigratesToTitleField(t *testing.T) {
	var storage StorageConfig
	saved := `{"name":"` + BALLCHASING_NAME + `","rename_style":1,"open_replay_style":0,"rename_path":"/replays/"}`

	if err := json.Unmarshal([]byte(saved), &storage); err != nil {
		t.Fatal(err)
	}

	if storage.RenameStyle != RenameTitleField {
		t.Errorf("RenameStyle = %v, want %v", storage.RenameStyle, RenameTitleField)
	}

	if storage.RenamePath != "" {
		t.Errorf("RenamePath = %q, want empty", storage.RenamePath)
	}

	if _, ok := storage.FieldLink("open_replay_style"); !ok {
		t.Error("the field links were not restored from the preset")
	}

	if storage.OpenReplayStyle != OpenReplayDisabled {
		t.Error("a saved Open Replay choice should be kept")
	}
}
