package pitchstack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/pitchstack-gg/pitchstack-cli/internal/session"
)

func TestListActivityFeed_BuildsQueryAndAuth(t *testing.T) {
	t.Parallel()

	var gotAuth string
	var gotPath string
	var gotQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"activityId":"a1"}],"nextToken":"t2"}`))
	}))
	t.Cleanup(srv.Close)

	store := session.NewStore(filepath.Join(t.TempDir(), "session.json"))
	if err := store.Save(&session.Session{
		BaseURL:              srv.URL,
		AccessToken:          "tok",
		AccessTokenExpiresAt: time.Now().Add(10 * time.Minute),
		RefreshToken:         "ref",
	}); err != nil {
		t.Fatalf("save session: %v", err)
	}

	svc := NewService(ServiceDeps{
		BaseURL:        srv.URL,
		TimeoutSeconds: 5,
		Sessions:       store,
	})

	ps := int32(10)
	resp, err := svc.ListActivityFeed(context.Background(), &ListActivityFeedRequest{
		PageSize:  &ps,
		NextToken: "nxt",
		Scopes: []ActivityScope{
			ActivityScopeFollowing,
			ActivityScopeShared,
		},
	})
	if err != nil {
		t.Fatalf("ListActivityFeed: %v", err)
	}

	if gotAuth != "Bearer tok" {
		t.Fatalf("Authorization header = %q, want %q", gotAuth, "Bearer tok")
	}
	if gotPath != "/v1/activity" {
		t.Fatalf("path = %q, want %q", gotPath, "/v1/activity")
	}
	if gotQuery.Get("pageSize") != "10" {
		t.Fatalf("pageSize = %q, want %q", gotQuery.Get("pageSize"), "10")
	}
	if gotQuery.Get("nextToken") != "nxt" {
		t.Fatalf("nextToken = %q, want %q", gotQuery.Get("nextToken"), "nxt")
	}
	if len(gotQuery["scopes"]) != 2 || gotQuery["scopes"][0] != string(ActivityScopeFollowing) || gotQuery["scopes"][1] != string(ActivityScopeShared) {
		t.Fatalf("scopes = %#v, want [%q %q]", gotQuery["scopes"], ActivityScopeFollowing, ActivityScopeShared)
	}

	if resp == nil || resp.NextToken != "t2" || len(resp.Items) != 1 || resp.Items[0].ActivityID != "a1" {
		t.Fatalf("response = %#v", resp)
	}
}

func TestListActivityFeed_RequiresLogin(t *testing.T) {
	t.Parallel()

	svc := NewService(ServiceDeps{
		BaseURL:        "https://example.com",
		TimeoutSeconds: 5,
		Sessions:       session.NewStore(filepath.Join(t.TempDir(), "session.json")),
	})
	_, err := svc.ListActivityFeed(context.Background(), &ListActivityFeedRequest{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
