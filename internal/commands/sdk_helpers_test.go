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
	"github.com/pitchstack-gg/pitchstack-cli/internal/session"
)

func TestCardsPricesBatchCommand_FileStdinAndFlagOverride(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/prices:batchGet" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"prices":[]}`))
	}))
	t.Cleanup(server.Close)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	cfg := `{"currentProfile":"test","profiles":{"test":{"baseUrl":` + quoteJSON(server.URL) + `,"timeoutSeconds":5}}}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	store := session.NewStore(paths.SessionPath("test"))
	if err := store.Save(&session.Session{
		BaseURL:              server.URL,
		AccessToken:          "tok",
		RefreshToken:         "ref",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("save session: %v", err)
	}

	var stdout bytes.Buffer
	root := NewRootCommand(strings.NewReader(`{"productIds":["from-file"],"source":"file-source"}`), &stdout, io.Discard)
	err := root.Run(context.Background(), []string{
		"pitchstack", "--config", cfgPath, "--profile", "test",
		"cards", "prices", "batch", "--file", "-", "--product-id", "from-flag", "--source", "flag-source",
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("Authorization = %q, want Bearer tok", gotAuth)
	}
	ids, _ := gotBody["productIds"].([]any)
	if len(ids) != 1 || ids[0] != "from-flag" {
		t.Fatalf("productIds = %#v, want [from-flag]", gotBody["productIds"])
	}
	if gotBody["source"] != "flag-source" {
		t.Fatalf("source = %#v, want flag-source", gotBody["source"])
	}
}

func TestUploadFileToSignedURL_PutsBytesAndHeaders(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotContentType string
	var gotCustom string
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotCustom = r.Header.Get("X-Test")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	path := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
		t.Fatalf("write upload file: %v", err)
	}

	err := uploadFileToSignedURL(context.Background(), server.Client(), server.URL, map[string]string{"X-Test": "ok"}, path, "text/plain")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("method = %q, want PUT", gotMethod)
	}
	if gotContentType != "text/plain" {
		t.Fatalf("content-type = %q, want text/plain", gotContentType)
	}
	if gotCustom != "ok" {
		t.Fatalf("X-Test = %q, want ok", gotCustom)
	}
	if gotBody != "payload" {
		t.Fatalf("body = %q, want payload", gotBody)
	}
}

func TestRootCommandGroupsExposeConsolidatedHelp(t *testing.T) {
	t.Parallel()

	groups := []string{"auth", "me", "social", "cards", "collections", "decks", "pulls", "config", "version"}
	for _, group := range groups {
		t.Run(group, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			root := NewRootCommand(strings.NewReader(""), &stdout, io.Discard)
			if err := root.Run(context.Background(), []string{"pitchstack", group, "--help"}); err != nil {
				t.Fatalf("help failed: %v", err)
			}
			if !strings.Contains(stdout.String(), group) {
				t.Fatalf("help output for %q did not mention command; output=%q", group, stdout.String())
			}
		})
	}
}

func TestRemovedRootCommandGroupsAreNotExposed(t *testing.T) {
	t.Parallel()

	groups := []string{"profile", "activity", "groups", "events", "pricing", "news", "notifications"}
	for _, group := range groups {
		t.Run(group, func(t *testing.T) {
			t.Parallel()
			root := NewRootCommand(strings.NewReader(""), io.Discard, io.Discard)
			if err := root.Run(context.Background(), []string{"pitchstack", group, "--help"}); err == nil {
				t.Fatalf("%q should not be available at root", group)
			}
		})
	}
}

func TestAuthCanonicalCommandsAndHiddenRootShortcuts(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"pitchstack", "auth", "login", "--help"},
		{"pitchstack", "auth", "signup", "--help"},
		{"pitchstack", "auth", "whoami", "--help"},
		{"pitchstack", "auth", "logout", "--help"},
		{"pitchstack", "login", "--help"},
		{"pitchstack", "signup", "--help"},
		{"pitchstack", "whoami", "--help"},
		{"pitchstack", "logout", "--help"},
	} {
		var stdout bytes.Buffer
		root := NewRootCommand(strings.NewReader(""), &stdout, io.Discard)
		if err := root.Run(context.Background(), args); err != nil {
			t.Fatalf("%v help failed: %v", args, err)
		}
	}

	var stdout bytes.Buffer
	root := NewRootCommand(strings.NewReader(""), &stdout, io.Discard)
	if err := root.Run(context.Background(), []string{"pitchstack", "--help"}); err != nil {
		t.Fatalf("root help failed: %v", err)
	}
	help := stdout.String()
	for _, hidden := range []string{"login", "signup", "whoami", "logout"} {
		if strings.Contains(help, "\n   "+hidden+" ") {
			t.Fatalf("%q should be hidden from root help; output=%q", hidden, help)
		}
	}
}

func quoteJSON(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}
