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

func TestCardsTrendingCommandFormatsEnrichedItems(t *testing.T) {
	viewedAt := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/engagement/trending:list" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"resources": [{
				"resource": {"resourceType": "TRACKABLE_RESOURCE_TYPE_CARD", "resourceId": "card-alpha"},
				"viewCount": "19",
				"score": 17.40220361395269,
				"lastViewedAt": ` + quoteJSON(viewedAt) + `
			}],
			"nextPageToken": "next"
		}`))
	}))
	t.Cleanup(server.Close)

	cfgPath := setupCommandTestProfile(t, server.URL)
	installSimpleCommandCardsDB(t, paths.CardsDBPath("test"))

	var stdout, stderr bytes.Buffer
	root := NewRootCommand(strings.NewReader(""), &stdout, &stderr)
	err := root.Run(context.Background(), []string{
		"pitchstack", "--config", cfgPath, "--profile", "test",
		"cards", "trending", "--offline", "--window", "24h",
	})
	if err != nil {
		t.Fatalf("run command: %v; stderr=%s", err, stderr.String())
	}

	if gotBody["resourceType"] != "TRACKABLE_RESOURCE_TYPE_CARD" {
		t.Fatalf("resourceType = %#v, want card request; body=%#v", gotBody["resourceType"], gotBody)
	}

	var got struct {
		ResourceType  string `json:"resourceType"`
		Window        string `json:"window"`
		NextPageToken string `json:"nextPageToken"`
		Items         []struct {
			Rank          int     `json:"rank"`
			ResourceType  string  `json:"resourceType"`
			ResourceID    string  `json:"resourceId"`
			ViewCount     int64   `json:"viewCount"`
			Score         float64 `json:"score"`
			LastViewedAt  string  `json:"lastViewedAt"`
			LastViewedAgo string  `json:"lastViewedAgo"`
			Card          *struct {
				ID              string   `json:"id"`
				Name            string   `json:"name"`
				Types           []string `json:"types"`
				Cost            string   `json:"cost"`
				Pitch           string   `json:"pitch"`
				Power           string   `json:"power"`
				Defense         string   `json:"defense"`
				DefaultImageURL string   `json:"defaultImageUrl"`
			} `json:"card"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if got.ResourceType != "card" || got.Window != "24h" || got.NextPageToken != "next" {
		t.Fatalf("output metadata = %#v; stdout=%s", got, stdout.String())
	}
	if len(got.Items) != 1 {
		t.Fatalf("items len = %d, want 1; stdout=%s", len(got.Items), stdout.String())
	}
	item := got.Items[0]
	if item.Rank != 1 || item.ResourceType != "card" || item.ResourceID != "card-alpha" {
		t.Fatalf("item identity = %#v; stdout=%s", item, stdout.String())
	}
	if item.ViewCount != 19 || item.Score != 17.4022 || item.LastViewedAt != viewedAt || item.LastViewedAgo == "" {
		t.Fatalf("item metrics = %#v; stdout=%s", item, stdout.String())
	}
	if item.Card == nil || item.Card.ID != "card-alpha" || item.Card.Name != "Alpha Strike" {
		t.Fatalf("card summary = %#v; stdout=%s", item.Card, stdout.String())
	}
	if item.Card.Cost != "1" || item.Card.Pitch != "2" || item.Card.Power != "3" || item.Card.Defense != "2" {
		t.Fatalf("card stats = %#v; stdout=%s", item.Card, stdout.String())
	}
}

func TestCardsTrendingCommandRawPrintsAPIShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/engagement/trending:list" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"resources": [{
				"resource": {"resourceType": "TRACKABLE_RESOURCE_TYPE_CARD", "resourceId": "card-alpha"},
				"viewCount": 2,
				"score": 1.5
			}]
		}`))
	}))
	t.Cleanup(server.Close)

	cfgPath := setupCommandTestProfile(t, server.URL)
	var stdout, stderr bytes.Buffer
	root := NewRootCommand(strings.NewReader(""), &stdout, &stderr)
	err := root.Run(context.Background(), []string{
		"pitchstack", "--config", cfgPath, "--profile", "test",
		"cards", "trending", "--raw",
	})
	if err != nil {
		t.Fatalf("run command: %v; stderr=%s", err, stderr.String())
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if _, ok := got["items"]; ok {
		t.Fatalf("raw output should not include formatted items: %s", stdout.String())
	}
	resources, _ := got["resources"].([]any)
	if len(resources) != 1 {
		t.Fatalf("resources = %#v, want raw API resources; stdout=%s", resources, stdout.String())
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
