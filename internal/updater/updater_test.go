package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a    string
		b    string
		want int
	}{
		{a: "v0.2.0", b: "0.1.9", want: 1},
		{a: "0.2.0", b: "0.2.0", want: 0},
		{a: "0.2.0", b: "0.2", want: 0},
		{a: "0.2.0", b: "0.3.0", want: -1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			if got := CompareVersions(tt.a, tt.b); got != tt.want {
				t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestInstallLatestDownloadsVerifiesAndInstallsReleaseAsset(t *testing.T) {
	t.Parallel()

	suffix, err := releaseAssetSuffix()
	if err != nil {
		t.Skip(err)
	}
	assetName := "pitchstack_0.2.0" + suffix
	archive := releaseArchive(t, "updated binary")
	sum := sha256.Sum256(archive)
	checksums := hex.EncodeToString(sum[:]) + "  " + assetName + "\n"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/pitchstack-gg/pitchstack-cli/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{
				"tag_name":"v0.2.0",
				"html_url":"%s/releases/tag/v0.2.0",
				"assets":[
					{"name":%q,"browser_download_url":"%s/archive.tgz"},
					{"name":"checksums.txt","browser_download_url":"%s/checksums.txt"}
				]
			}`, server.URL, assetName, server.URL, server.URL)
		case "/archive.tgz":
			_, _ = w.Write(archive)
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	installDir := t.TempDir()
	result, err := InstallLatest(context.Background(), InstallOptions{
		APIBaseURL: server.URL,
		Current:    "0.1.0",
		InstallDir: installDir,
		CachePath:  filepath.Join(t.TempDir(), "update-check.json"),
	})
	if err != nil {
		t.Fatalf("InstallLatest: %v", err)
	}
	if !result.Installed {
		t.Fatalf("Installed = false, want true")
	}
	target := filepath.Join(installDir, "pitchstack")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read installed binary: %v", err)
	}
	if strings.TrimSpace(string(data)) != "updated binary" {
		t.Fatalf("installed binary = %q", data)
	}
}

func releaseArchive(t *testing.T, content string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	data := []byte(content)
	if err := tw.WriteHeader(&tar.Header{
		Name: "pitchstack",
		Mode: 0o755,
		Size: int64(len(data)),
	}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("write tar content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}
