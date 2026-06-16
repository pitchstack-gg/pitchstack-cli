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
