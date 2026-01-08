package pitchstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pitchstack-gg/pitchstack-cli/internal/session"
)

func TestCreatePollSaveCLILoginSession(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/auth/cli/sessions":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if strings.TrimSpace(asString(body["baseUrl"])) == "" {
				http.Error(w, "missing baseUrl", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId":           "sess-1",
				"sessionSecret":       "secret-1",
				"verificationPath":    "/v1/auth/cli/sessions/sess-1/login",
				"expiresAt":           time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339),
				"pollIntervalSeconds": 1,
			})
			return
		case r.Method == http.MethodPost && r.URL.Path == "/v1/auth/cli/sessions/sess-1:poll":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if strings.TrimSpace(asString(body["sessionSecret"])) != "secret-1" {
				http.Error(w, "bad secret", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "CLI_LOGIN_SESSION_STATUS_COMPLETE",
				"login": map[string]any{
					"userId":               "user-1",
					"roles":                []string{"member"},
					"accessToken":          "at-1",
					"refreshToken":         "rt-1",
					"accessTokenExpiresAt": time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
				},
			})
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	store := session.NewStore(filepath.Join(t.TempDir(), "session.json"))
	svc := NewService(ServiceDeps{BaseURL: server.URL, TimeoutSeconds: 2, Sessions: store})

	ctx := context.Background()
	created, err := svc.CreateCLILoginSession(ctx, "https://auth.example")
	if err != nil {
		t.Fatalf("CreateCLILoginSession error: %v", err)
	}
	if created.VerificationURL == "" {
		t.Fatalf("expected VerificationURL to be populated")
	}

	poll, err := svc.PollCLILoginSession(ctx, created.SessionID, created.SessionSecret)
	if err != nil {
		t.Fatalf("PollCLILoginSession error: %v", err)
	}
	if poll.Login == nil || poll.Login.RefreshToken != "rt-1" {
		t.Fatalf("expected login tokens")
	}

	sess, err := svc.SaveLoginResult(ctx, poll.Login)
	if err != nil {
		t.Fatalf("SaveLoginResult error: %v", err)
	}
	if strings.TrimSpace(sess.RefreshToken) != "rt-1" {
		t.Fatalf("expected stored refresh token")
	}
}

func TestAuthRetryOn401RefreshesAndRetries(t *testing.T) {
	t.Parallel()

	var meCalls atomic.Int32
	var refreshCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/auth/token/refresh":
			refreshCalls.Add(1)
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if strings.TrimSpace(asString(body["refreshToken"])) != "rt-1" {
				http.Error(w, "bad refresh token", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"accessToken":          "at-2",
				"refreshToken":         "rt-2",
				"accessTokenExpiresAt": time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
			})
			return
		case r.Method == http.MethodGet && r.URL.Path == "/v1/me":
			call := meCalls.Add(1)
			auth := r.Header.Get("Authorization")
			if call == 1 {
				if auth != "Bearer at-1" {
					http.Error(w, "expected first call with at-1", http.StatusUnauthorized)
					return
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if auth != "Bearer at-2" {
				http.Error(w, "expected retried call with at-2", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"user": map[string]any{
					"userId": "user-1",
					"email":  "u@example.com",
				},
			})
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	storePath := filepath.Join(t.TempDir(), "session.json")
	store := session.NewStore(storePath)
	if err := store.Save(&session.Session{
		BaseURL:              server.URL,
		UserID:               "user-1",
		AccessToken:          "at-1",
		RefreshToken:         "rt-1",
		AccessTokenExpiresAt: time.Now().Add(10 * time.Minute).UTC(),
	}); err != nil {
		t.Fatalf("save session: %v", err)
	}

	svc := NewService(ServiceDeps{BaseURL: server.URL, TimeoutSeconds: 2, Sessions: store})

	if _, err := svc.Me(context.Background()); err != nil {
		t.Fatalf("Me error: %v", err)
	}
	if meCalls.Load() != 2 {
		t.Fatalf("expected 2 me calls, got %d", meCalls.Load())
	}
	if refreshCalls.Load() != 1 {
		t.Fatalf("expected 1 refresh call, got %d", refreshCalls.Load())
	}

	// Ensure token rotation was persisted.
	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	if !strings.Contains(string(data), `"refreshToken": "rt-2"`) {
		t.Fatalf("expected rotated refresh token to be saved")
	}
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
