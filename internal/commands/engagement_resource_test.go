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

func TestResourceLikeCommandsSendScopedRefs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		path         string
		resourceType string
		resourceID   string
		liked        bool
	}{
		{
			name:         "decks like",
			args:         []string{"decks", "like", "--id", "deck-1"},
			path:         "/v1/engagement/likes:like",
			resourceType: "LIKEABLE_RESOURCE_TYPE_DECK",
			resourceID:   "deck-1",
			liked:        true,
		},
		{
			name:         "decks unlike",
			args:         []string{"decks", "unlike", "--id", "deck-1"},
			path:         "/v1/engagement/likes:unlike",
			resourceType: "LIKEABLE_RESOURCE_TYPE_DECK",
			resourceID:   "deck-1",
			liked:        false,
		},
		{
			name:         "collections like",
			args:         []string{"collections", "like", "--id", "collection-1"},
			path:         "/v1/engagement/likes:like",
			resourceType: "LIKEABLE_RESOURCE_TYPE_COLLECTION",
			resourceID:   "collection-1",
			liked:        true,
		},
		{
			name:         "collections unlike",
			args:         []string{"collections", "unlike", "--id", "collection-1"},
			path:         "/v1/engagement/likes:unlike",
			resourceType: "LIKEABLE_RESOURCE_TYPE_COLLECTION",
			resourceID:   "collection-1",
			liked:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != tt.path {
					http.NotFound(w, r)
					return
				}
				if gotAuth := r.Header.Get("Authorization"); gotAuth != "Bearer tok" {
					t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
				}
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Errorf("decode body: %v", err)
					http.Error(w, "bad body", http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"resource": map[string]any{
						"resourceType": tt.resourceType,
						"resourceId":   tt.resourceID,
					},
					"liked":      tt.liked,
					"totalLikes": "3",
					"changed":    true,
				})
			}))
			t.Cleanup(server.Close)

			cfgPath := writeTestConfigAndSession(t, server.URL)
			var stdout bytes.Buffer
			root := NewRootCommand(strings.NewReader(""), &stdout, io.Discard)
			args := append([]string{"pitchstack", "--config", cfgPath, "--profile", "test"}, tt.args...)
			if err := root.Run(context.Background(), args); err != nil {
				t.Fatalf("run command: %v", err)
			}

			resource, _ := gotBody["resource"].(map[string]any)
			if resource["resourceType"] != tt.resourceType {
				t.Fatalf("resourceType = %#v, want %q; body=%#v", resource["resourceType"], tt.resourceType, gotBody)
			}
			if resource["resourceId"] != tt.resourceID {
				t.Fatalf("resourceId = %#v, want %q; body=%#v", resource["resourceId"], tt.resourceID, gotBody)
			}
		})
	}
}

func TestResourceTrendingCommandsSendScopedTypes(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		resourceType string
	}{
		{
			name:         "cards trending",
			args:         []string{"cards", "trending", "--window", "24h", "--page-size", "5", "--next-token", "next"},
			resourceType: "TRACKABLE_RESOURCE_TYPE_CARD",
		},
		{
			name:         "decks trending",
			args:         []string{"decks", "trending", "--window", "7d", "--page-size", "5", "--next-token", "next"},
			resourceType: "TRACKABLE_RESOURCE_TYPE_DECK",
		},
		{
			name:         "collections trending",
			args:         []string{"collections", "trending", "--window", "30d", "--page-size", "5", "--next-token", "next"},
			resourceType: "TRACKABLE_RESOURCE_TYPE_COLLECTION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/v1/engagement/trending:list" {
					http.NotFound(w, r)
					return
				}
				if gotAuth := r.Header.Get("Authorization"); gotAuth != "Bearer tok" {
					t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
				}
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Errorf("decode body: %v", err)
					http.Error(w, "bad body", http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"resources":[]}`))
			}))
			t.Cleanup(server.Close)

			cfgPath := writeTestConfigAndSession(t, server.URL)
			var stdout bytes.Buffer
			root := NewRootCommand(strings.NewReader(""), &stdout, io.Discard)
			args := append([]string{"pitchstack", "--config", cfgPath, "--profile", "test"}, tt.args...)
			if err := root.Run(context.Background(), args); err != nil {
				t.Fatalf("run command: %v", err)
			}

			if gotBody["resourceType"] != tt.resourceType {
				t.Fatalf("resourceType = %#v, want %q; body=%#v", gotBody["resourceType"], tt.resourceType, gotBody)
			}
			if gotBody["pageSize"] != float64(5) {
				t.Fatalf("pageSize = %#v, want 5; body=%#v", gotBody["pageSize"], gotBody)
			}
			if gotBody["nextPageToken"] != "next" {
				t.Fatalf("nextPageToken = %#v, want next; body=%#v", gotBody["nextPageToken"], gotBody)
			}
		})
	}
}

func writeTestConfigAndSession(t *testing.T, baseURL string) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

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
