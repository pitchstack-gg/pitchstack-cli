package updater

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pitchstack-gg/pitchstack-cli/internal/buildinfo"
	"github.com/pitchstack-gg/pitchstack-cli/internal/paths"
)

const (
	DefaultRepo       = "pitchstack-gg/pitchstack-cli"
	defaultAPIBaseURL = "https://api.github.com"
	binaryName        = "pitchstack"
)

type Release struct {
	Version      string
	TagName      string
	HTMLURL      string
	AssetURL     string
	AssetName    string
	ChecksumsURL string
}

type CheckOptions struct {
	Repo        string
	APIBaseURL  string
	CachePath   string
	HTTPClient  *http.Client
	Now         func() time.Time
	MaxCacheAge time.Duration
}

type InstallOptions struct {
	Repo       string
	APIBaseURL string
	HTTPClient *http.Client
	Current    string
	InstallDir string
	CachePath  string
	Force      bool
	Stdin      *os.File
	Stdout     *os.File
	Stderr     *os.File
}

type CheckResult struct {
	Release         Release
	UpdateAvailable bool
	FromCache       bool
}

type InstallResult struct {
	Release   Release
	Target    string
	Installed bool
}

type releaseResponse struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type cacheFile struct {
	CheckedAt string  `json:"checkedAt"`
	Release   Release `json:"release"`
}

func Check(ctx context.Context, current string, opts CheckOptions) (CheckResult, error) {
	if buildinfo.IsDevelopmentVersion(current) {
		return CheckResult{}, nil
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	maxAge := opts.MaxCacheAge
	if maxAge <= 0 {
		maxAge = 24 * time.Hour
	}
	cachePath := strings.TrimSpace(opts.CachePath)
	if cachePath == "" {
		cachePath = paths.UpdateCheckCachePath()
	}
	if cached, ok := readFreshCache(cachePath, now(), maxAge); ok {
		return CheckResult{
			Release:         cached.Release,
			UpdateAvailable: CompareVersions(cached.Release.Version, current) > 0,
			FromCache:       true,
		}, nil
	}

	release, err := Latest(ctx, opts.Repo, opts.APIBaseURL, opts.HTTPClient)
	if err != nil {
		return CheckResult{}, err
	}
	_ = writeCache(cachePath, cacheFile{
		CheckedAt: now().UTC().Format(time.RFC3339),
		Release:   release,
	})
	return CheckResult{
		Release:         release,
		UpdateAvailable: CompareVersions(release.Version, current) > 0,
	}, nil
}

func Latest(ctx context.Context, repo string, apiBaseURL string, client *http.Client) (Release, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		repo = DefaultRepo
	}
	apiBaseURL = strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = defaultAPIBaseURL
	}
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL+"/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pitchstack-cli")

	res, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Release{}, fmt.Errorf("latest release request failed: %s", res.Status)
	}

	var payload releaseResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return Release{}, fmt.Errorf("decode latest release: %w", err)
	}
	release := Release{
		Version: strings.TrimPrefix(strings.TrimSpace(payload.TagName), "v"),
		TagName: payload.TagName,
		HTMLURL: payload.HTMLURL,
	}
	suffix, err := releaseAssetSuffix()
	if err != nil {
		return Release{}, err
	}
	for _, asset := range payload.Assets {
		switch {
		case strings.HasSuffix(asset.Name, suffix):
			release.AssetName = asset.Name
			release.AssetURL = asset.BrowserDownloadURL
		case asset.Name == "checksums.txt" || strings.HasSuffix(asset.Name, "checksums.txt"):
			release.ChecksumsURL = asset.BrowserDownloadURL
		}
	}
	if release.Version == "" {
		return Release{}, errors.New("latest release did not include a tag name")
	}
	return release, nil
}

func InstallLatest(ctx context.Context, opts InstallOptions) (InstallResult, error) {
	current := opts.Current
	if strings.TrimSpace(current) == "" {
		current = buildinfo.Version
	}
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	release, err := Latest(ctx, opts.Repo, opts.APIBaseURL, client)
	if err != nil {
		return InstallResult{}, err
	}
	if release.AssetURL == "" {
		return InstallResult{}, errors.New("latest release does not include an asset for this OS/architecture")
	}
	if release.ChecksumsURL == "" {
		return InstallResult{}, errors.New("latest release does not include checksums.txt")
	}
	if !opts.Force && !buildinfo.IsDevelopmentVersion(current) && CompareVersions(release.Version, current) <= 0 {
		return InstallResult{Release: release}, nil
	}

	target, err := installTarget(opts.InstallDir)
	if err != nil {
		return InstallResult{}, err
	}
	tmpDir, err := os.MkdirTemp("", "pitchstack-update-*")
	if err != nil {
		return InstallResult{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, release.AssetName)
	if err := downloadFile(ctx, client, release.AssetURL, archivePath); err != nil {
		return InstallResult{}, err
	}
	checksumsPath := filepath.Join(tmpDir, "checksums.txt")
	if err := downloadFile(ctx, client, release.ChecksumsURL, checksumsPath); err != nil {
		return InstallResult{}, err
	}
	if err := verifyChecksum(archivePath, checksumsPath, release.AssetName); err != nil {
		return InstallResult{}, err
	}
	extractedPath, err := extractBinary(archivePath, tmpDir)
	if err != nil {
		return InstallResult{}, err
	}
	if err := installBinary(extractedPath, target, opts); err != nil {
		return InstallResult{}, err
	}
	cachePath := strings.TrimSpace(opts.CachePath)
	if cachePath == "" {
		cachePath = paths.UpdateCheckCachePath()
	}
	_ = writeCache(cachePath, cacheFile{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Release:   release,
	})
	return InstallResult{Release: release, Target: target, Installed: true}, nil
}

func CompareVersions(a string, b string) int {
	aa := versionParts(a)
	bb := versionParts(b)
	for i := 0; i < len(aa) || i < len(bb); i++ {
		var av, bv int
		if i < len(aa) {
			av = aa[i]
		}
		if i < len(bb) {
			bv = bb[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

func versionParts(version string) []int {
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	if idx := strings.IndexAny(version, "-+"); idx >= 0 {
		version = version[:idx]
	}
	fields := strings.Split(version, ".")
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			parts = append(parts, 0)
			continue
		}
		n, err := strconv.Atoi(field)
		if err != nil {
			return nil
		}
		parts = append(parts, n)
	}
	return parts
}

func releaseAssetSuffix() (string, error) {
	goos := runtime.GOOS
	switch goos {
	case "darwin", "linux":
	default:
		return "", fmt.Errorf("unsupported OS: %s", goos)
	}
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}
	return "_" + goos + "_" + arch + ".tar.gz", nil
}

func readFreshCache(path string, now time.Time, maxAge time.Duration) (cacheFile, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheFile{}, false
	}
	var cached cacheFile
	if err := json.Unmarshal(data, &cached); err != nil {
		return cacheFile{}, false
	}
	checkedAt, err := time.Parse(time.RFC3339, cached.CheckedAt)
	if err != nil || cached.Release.Version == "" {
		return cacheFile{}, false
	}
	return cached, now.Sub(checkedAt) >= 0 && now.Sub(checkedAt) < maxAge
}

func writeCache(path string, cached cacheFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func installTarget(installDir string) (string, error) {
	if strings.TrimSpace(installDir) != "" {
		return filepath.Join(installDir, binaryName), nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve current executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil && resolved != "" {
		executable = resolved
	}
	return executable, nil
}

func downloadFile(ctx context.Context, client *http.Client, url string, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "pitchstack-cli")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("download %s failed: %s", url, res.Status)
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, res.Body); err != nil {
		return err
	}
	return out.Close()
}

func verifyChecksum(archivePath string, checksumsPath string, archiveName string) error {
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}
	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == archiveName || strings.TrimPrefix(fields[1], "*") == archiveName {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum not found for %s", archiveName)
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return fmt.Errorf("hash archive: %w", err)
	}
	actual := hex.EncodeToString(sum.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch for %s", archiveName)
	}
	return nil
}

func extractBinary(archivePath string, tmpDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("open gzip archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != binaryName {
			continue
		}
		outPath := filepath.Join(tmpDir, binaryName)
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", fmt.Errorf("create extracted binary: %w", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return "", fmt.Errorf("extract binary: %w", err)
		}
		if err := out.Close(); err != nil {
			return "", err
		}
		return outPath, nil
	}
	return "", fmt.Errorf("%s not found in archive", binaryName)
}

func installBinary(source string, target string, opts InstallOptions) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}
	if err := copyAndReplace(source, target); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrPermission) {
		var pathErr *os.PathError
		if !(errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrPermission)) {
			return err
		}
	}
	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("install %s: permission denied; re-run with sudo or use --install-dir", target)
	}
	cmd := exec.Command("sudo", "install", "-m", "0755", source, target)
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	return cmd.Run()
}

func copyAndReplace(source string, target string) error {
	tmp := target + ".new"
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
