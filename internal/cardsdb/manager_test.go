package cardsdb

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerEnsure_FirstDownloadStoresDecompressedDatabase(t *testing.T) {
	t.Parallel()
	want := []byte("sqlite bytes")
	published := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/LAST_PUBLISHED":
			_, _ = w.Write([]byte(published.Format(time.RFC3339Nano)))
		case "/pitchstack.sqlite.gz":
			w.Header().Set("ETag", `"abc"`)
			_, _ = w.Write(gzipBytes(t, want))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	manager := &Manager{
		DBPath:         filepath.Join(dir, "pitchstack.sqlite"),
		MetaPath:       filepath.Join(dir, "meta.json"),
		DBURL:          server.URL + "/pitchstack.sqlite.gz",
		LastUpdatedURL: server.URL + "/LAST_PUBLISHED",
		HTTPClient:     server.Client(),
	}

	result, err := manager.Ensure(context.Background(), EnsureOptions{})
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if !result.Updated {
		t.Fatalf("Ensure() Updated = false, want true")
	}
	got, err := os.ReadFile(manager.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("db bytes = %q, want %q", got, want)
	}
	if result.Meta == nil || result.Meta.ETag != `"abc"` || !result.Meta.LastPublishedAt.Equal(published) {
		t.Fatalf("metadata = %#v", result.Meta)
	}
}

func TestManagerEnsure_NotModifiedKeepsCurrentDatabase(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-None-Match"); got != `"old"` {
			t.Fatalf("If-None-Match = %q, want old etag", got)
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "pitchstack.sqlite")
	metaPath := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(dbPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	manager := &Manager{
		DBPath:     dbPath,
		MetaPath:   metaPath,
		DBURL:      server.URL,
		HTTPClient: server.Client(),
	}
	if err := manager.saveMeta(&Metadata{ETag: `"old"`}); err != nil {
		t.Fatal(err)
	}

	result, err := manager.Ensure(context.Background(), EnsureOptions{})
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if result.Updated {
		t.Fatalf("Ensure() Updated = true, want false")
	}
	got, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "existing" {
		t.Fatalf("db bytes = %q, want existing", got)
	}
}

func TestManagerEnsure_FailedRefreshPreservesExistingDatabase(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "pitchstack.sqlite")
	if err := os.WriteFile(dbPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	manager := &Manager{
		DBPath:     dbPath,
		MetaPath:   filepath.Join(dir, "meta.json"),
		DBURL:      server.URL,
		HTTPClient: server.Client(),
	}

	result, err := manager.Ensure(context.Background(), EnsureOptions{Force: true})
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if result.Updated {
		t.Fatalf("Ensure() Updated = true, want false")
	}
}

func TestManagerEnsure_OfflineRequiresExistingDatabase(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	manager := &Manager{
		DBPath:   filepath.Join(dir, "missing.sqlite"),
		MetaPath: filepath.Join(dir, "meta.json"),
	}

	if _, err := manager.Ensure(context.Background(), EnsureOptions{Offline: true}); err == nil {
		t.Fatalf("Ensure() error = nil, want error")
	}
}

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
