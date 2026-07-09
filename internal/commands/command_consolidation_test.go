package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSocialUsersCommandsRouteToProfileAPIs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		path string
	}{
		{
			name: "get",
			args: []string{"social", "users", "get", "--user-id", "user-1"},
			path: "/v1/users/user-1/profile",
		},
		{
			name: "search",
			args: []string{"social", "users", "search", "--q", "az", "--page-size", "5", "--next-token", "next"},
			path: "/v1/users/search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				if gotPath != tt.path {
					http.NotFound(w, r)
					return
				}
				if gotAuth := r.Header.Get("Authorization"); gotAuth != "Bearer tok" {
					t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
				}
				w.Header().Set("Content-Type", "application/json")
				if tt.name == "search" {
					_ = json.NewEncoder(w).Encode(map[string]any{"users": []any{}})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"profile": map[string]any{"userId": "user-1"}})
			}))
			t.Cleanup(server.Close)

			runCommandForTest(t, server.URL, "", tt.args...)
			if gotPath != tt.path {
				t.Fatalf("path = %q, want %q", gotPath, tt.path)
			}
		})
	}
}

func TestNestedSocialCommandsRouteExistingAPIs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		path string
		body string
	}{
		{
			name: "activity",
			args: []string{"social", "activity", "list", "--scope", "following", "--page-size", "5"},
			path: "/v1/activity",
			body: `{"items":[]}`,
		},
		{
			name: "groups",
			args: []string{"social", "groups", "list", "--page-size", "5"},
			path: "/v1/groups:mine",
			body: `{"groups":[]}`,
		},
		{
			name: "events",
			args: []string{"social", "events", "list", "--page-size", "5"},
			path: "/v1/events",
			body: `{"events":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				if gotPath != tt.path {
					http.NotFound(w, r)
					return
				}
				if gotAuth := r.Header.Get("Authorization"); gotAuth != "Bearer tok" {
					t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			runCommandForTest(t, server.URL, "", tt.args...)
			if gotPath != tt.path {
				t.Fatalf("path = %q, want %q", gotPath, tt.path)
			}
		})
	}
}

func TestMePriceWatchesRoutesExistingAPI(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodGet || gotPath != "/v1/price-watches" {
			http.NotFound(w, r)
			return
		}
		if gotAuth := r.Header.Get("Authorization"); gotAuth != "Bearer tok" {
			t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"watches":[]}`))
	}))
	t.Cleanup(server.Close)

	runCommandForTest(t, server.URL, "", "me", "price-watches", "list", "--active-only")
	if gotPath != "/v1/price-watches" {
		t.Fatalf("path = %q, want /v1/price-watches", gotPath)
	}
}

func TestDeckSideboardGuideUpsertUsesTargetsAndPlayOrder(t *testing.T) {
	var gotPath string
	var gotBody struct {
		ID      string `json:"id"`
		Targets []struct {
			TargetType string `json:"targetType"`
			Target     string `json:"target"`
		} `json:"targets"`
		Guide     string `json:"guide"`
		PlayOrder string `json:"playOrder"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPut || gotPath != "/v1/deck_versions/dv-1/sideboard_guides" {
			http.NotFound(w, r)
			return
		}
		if gotAuth := r.Header.Get("Authorization"); gotAuth != "Bearer tok" {
			t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sideboardGuide":{"id":"guide-1"}}`))
	}))
	t.Cleanup(server.Close)

	runCommandForTest(t, server.URL, "", "decks", "versions", "sideboard-guides", "upsert", "--deck-version-id", "dv-1", "--id", "guide-1", "--target-type", "hero", "--target", "hero-1", "--play-order", "first", "--guide", "sideboard notes")
	if gotPath != "/v1/deck_versions/dv-1/sideboard_guides" {
		t.Fatalf("path = %q, want /v1/deck_versions/dv-1/sideboard_guides", gotPath)
	}
	if gotBody.ID != "guide-1" || gotBody.Guide != "sideboard notes" || gotBody.PlayOrder != "SIDEBOARD_GUIDE_PLAY_ORDER_FIRST" {
		t.Fatalf("body = %#v", gotBody)
	}
	if len(gotBody.Targets) != 1 || gotBody.Targets[0].TargetType != "SIDEBOARD_GUIDE_TARGET_TYPE_HERO" || gotBody.Targets[0].Target != "hero-1" {
		t.Fatalf("targets = %#v", gotBody.Targets)
	}
}

func TestNewClientSurfaceCommandsRoute(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		path  string
		query string
	}{
		{
			name:  "card identifiers",
			args:  []string{"cards", "identifiers", "--page-size", "2", "--next-token", "next"},
			path:  "/v1/cards:identifiers",
			query: "nextToken=next&pageSize=2",
		},
		{
			name: "price watch digest",
			args: []string{"me", "price-watches", "digest", "--digest-date", "2026-07-08"},
			path: "/v1/price-watch-digests/by-date/2026-07-08",
		},
		{
			name: "email change status",
			args: []string{"auth", "email", "change-status"},
			path: "/v1/auth/email-change",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotQuery string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotQuery = r.URL.RawQuery
				if gotPath != tt.path {
					http.NotFound(w, r)
					return
				}
				if gotAuth := r.Header.Get("Authorization"); gotAuth != "Bearer tok" {
					t.Errorf("Authorization = %q, want Bearer tok", gotAuth)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(server.Close)

			runCommandForTest(t, server.URL, "", tt.args...)
			if gotPath != tt.path {
				t.Fatalf("path = %q, want %q", gotPath, tt.path)
			}
			if tt.query != "" && gotQuery != tt.query {
				t.Fatalf("query = %q, want %q", gotQuery, tt.query)
			}
		})
	}
}

func runCommandForTest(t *testing.T, baseURL string, stdin string, args ...string) string {
	t.Helper()

	cfgPath := writeTestConfigAndSession(t, baseURL)
	var stdout bytes.Buffer
	root := NewRootCommand(strings.NewReader(stdin), &stdout, io.Discard)
	fullArgs := append([]string{"pitchstack", "--config", cfgPath, "--profile", "test"}, args...)
	if err := root.Run(context.Background(), fullArgs); err != nil {
		t.Fatalf("run command %v: %v", args, err)
	}
	return stdout.String()
}
