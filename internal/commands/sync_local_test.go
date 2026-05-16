package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pitchstack-gg/pitchstack-cli/internal/paths"
	"github.com/pitchstack-gg/pitchstack-cli/internal/powersync"
	"github.com/pitchstack-gg/pitchstack-cli/internal/session"
)

func TestCollectionsCountsLocalCommand(t *testing.T) {
	cfgPath := setupCommandTestProfile(t, "")
	store, err := powersync.OpenStore(paths.SyncDBPath("test"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.ApplyOperations(context.Background(), "cp-1", []powersync.Operation{
		{Bucket: "b1", OpID: "1", Op: "PUT", Table: "collections", ID: "col-1", Data: map[string]any{"id": "col-1", "name": "Binder"}},
		{Bucket: "b1", OpID: "2", Op: "PUT", Table: "collection_items", ID: "item-1", Data: map[string]any{"id": "item-1", "collectionId": "col-1", "cardId": "card-1", "quantity": 2}},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_ = store.Close()

	var stdout bytes.Buffer
	root := NewRootCommand(strings.NewReader(""), &stdout, io.Discard)
	if err := root.Run(context.Background(), []string{"pitchstack", "--config", cfgPath, "--profile", "test", "collections", "counts", "--local"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	var out struct {
		Collections []powersync.CollectionCount `json:"collections"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if len(out.Collections) != 1 || out.Collections[0].QuantityCount != 2 {
		t.Fatalf("output = %#v", out)
	}
}

func TestSyncLocalInitCommand(t *testing.T) {
	var gotAuth string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sync/powersync/client-config" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"powerSyncUrl":"https://powersync.example.test/sync","syncEpoch":"epoch-1"}`))
	}))
	defer api.Close()
	cfgPath := setupCommandTestProfile(t, api.URL)

	var stdout bytes.Buffer
	root := NewRootCommand(strings.NewReader(""), &stdout, io.Discard)
	if err := root.Run(context.Background(), []string{"pitchstack", "--config", cfgPath, "--profile", "test", "sync", "local", "init"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("auth = %q, want Bearer tok", gotAuth)
	}
	var out powersync.InitResult
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if out.DeviceID == "" || out.SyncEpoch != "epoch-1" {
		t.Fatalf("output = %#v", out)
	}
}

func TestTUICommandHelpIncludesTabFlag(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	root := NewRootCommand(strings.NewReader(""), &stdout, io.Discard)
	if err := root.Run(context.Background(), []string{"pitchstack", "tui", "--help"}); err != nil {
		t.Fatalf("help failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "--tab") {
		t.Fatalf("help output missing --tab:\n%s", stdout.String())
	}
}

func TestHasUsableLocalCredentials(t *testing.T) {
	t.Parallel()
	if !hasUsableLocalCredentials("", "refresh", time.Time{}) {
		t.Fatalf("refresh token should be usable")
	}
	if !hasUsableLocalCredentials("access", "", time.Now().Add(5*time.Minute)) {
		t.Fatalf("fresh access token should be usable")
	}
	if hasUsableLocalCredentials("access", "", time.Now().Add(30*time.Second)) {
		t.Fatalf("expiring access token without refresh token should not be usable")
	}
	if hasUsableLocalCredentials("", "", time.Time{}) {
		t.Fatalf("missing tokens should not be usable")
	}
}

func setupCommandTestProfile(t *testing.T, baseURL string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	if baseURL == "" {
		baseURL = "https://api.example.test"
	}
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	cfg := `{"currentProfile":"test","profiles":{"test":{"baseUrl":` + quoteJSON(baseURL) + `,"timeoutSeconds":5}}}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	store := session.NewStore(paths.SessionPath("test"))
	if err := store.Save(&session.Session{
		BaseURL:              baseURL,
		AccessToken:          "tok",
		RefreshToken:         "ref",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("save session: %v", err)
	}
	return cfgPath
}
