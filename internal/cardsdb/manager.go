package cardsdb

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultCardsDBURL            = "https://cards.pitchstack.gg/pitchstack/pitchstack.sqlite.gz"
	DefaultCardsDBLastUpdatedURL = "https://cards.pitchstack.gg/pitchstack/LAST_PUBLISHED"
	DefaultRefreshInterval       = time.Hour
)

type Status struct {
	Phase   string
	Message string
}

type Metadata struct {
	ETag            string    `json:"etag,omitempty"`
	LastModified    string    `json:"lastModified,omitempty"`
	FetchedAt       time.Time `json:"fetchedAt,omitempty"`
	LastCheckedAt   time.Time `json:"lastCheckedAt,omitempty"`
	LastCheckError  string    `json:"lastCheckError,omitempty"`
	KnownOutdatedAt time.Time `json:"knownOutdatedAt,omitempty"`
	LastPublishedAt time.Time `json:"lastPublishedAt,omitempty"`
	InstalledAt     time.Time `json:"installedAt,omitempty"`
}

type Manager struct {
	DBPath         string
	MetaPath       string
	DBURL          string
	LastUpdatedURL string
	HTTPClient     *http.Client
	OnStatus       func(Status)
}

type EnsureOptions struct {
	Force           bool
	Offline         bool
	AutoRefresh     *bool
	RefreshInterval time.Duration
}

type EnsureResult struct {
	DBPath            string
	Meta              *Metadata
	Updated           bool
	Outdated          bool
	LatestPublishedAt time.Time
}

func (m *Manager) Ensure(ctx context.Context, opts EnsureOptions) (*EnsureResult, error) {
	if strings.TrimSpace(m.DBPath) == "" {
		return nil, errors.New("cards database path is required")
	}
	if strings.TrimSpace(m.MetaPath) == "" {
		return nil, errors.New("cards database metadata path is required")
	}

	meta, _ := m.loadMeta()
	hasDB := fileExists(m.DBPath)
	now := time.Now().UTC()
	if opts.Offline {
		if !hasDB {
			return nil, fmt.Errorf("cards database not installed at %s; rerun without --offline first", m.DBPath)
		}
		m.status("ready", "Using cached card database.")
		return &EnsureResult{DBPath: m.DBPath, Meta: meta, Updated: false}, nil
	}

	dbURL := strings.TrimSpace(m.DBURL)
	if dbURL == "" {
		dbURL = DefaultCardsDBURL
	}
	refreshInterval := opts.RefreshInterval
	if refreshInterval <= 0 {
		refreshInterval = DefaultRefreshInterval
	}
	autoRefresh := true
	if opts.AutoRefresh != nil {
		autoRefresh = *opts.AutoRefresh
	}

	if hasDB && !opts.Force && refreshInterval > 0 && meta != nil && !meta.LastCheckedAt.IsZero() && now.Sub(meta.LastCheckedAt) < refreshInterval {
		m.status("ready", "Using cached card database.")
		return &EnsureResult{DBPath: m.DBPath, Meta: meta, Updated: false}, nil
	}

	if hasDB && !opts.Force && strings.TrimSpace(m.LastUpdatedURL) == "" {
		m.status("ready", "Using cached card database.")
		return &EnsureResult{DBPath: m.DBPath, Meta: meta, Updated: false}, nil
	}

	var latestPublishedAt time.Time
	if strings.TrimSpace(m.LastUpdatedURL) != "" {
		m.status("checking", "Checking card database freshness...")
		if latest, err := m.fetchLastPublished(ctx); err == nil {
			latestPublishedAt = latest
			if meta == nil {
				meta = &Metadata{}
			}
			meta.LastCheckedAt = now
			meta.LastCheckError = ""
			if hasDB && !opts.Force && !latest.IsZero() && meta != nil && !meta.LastPublishedAt.IsZero() && !latest.After(meta.LastPublishedAt) {
				meta.KnownOutdatedAt = time.Time{}
				if err := m.saveMeta(meta); err != nil {
					return nil, err
				}
				m.status("ready", "Card database is up to date.")
				return &EnsureResult{DBPath: m.DBPath, Meta: meta, Updated: false}, nil
			}
			if hasDB && !opts.Force && !autoRefresh && !latest.IsZero() && (meta.LastPublishedAt.IsZero() || latest.After(meta.LastPublishedAt)) {
				meta.KnownOutdatedAt = latest
				if err := m.saveMeta(meta); err != nil {
					return nil, err
				}
				m.status("outdated", "Card database update is available.")
				return &EnsureResult{DBPath: m.DBPath, Meta: meta, Updated: false, Outdated: true, LatestPublishedAt: latest}, nil
			}
		} else if hasDB && !opts.Force {
			if meta == nil {
				meta = &Metadata{}
			}
			meta.LastCheckedAt = now
			meta.LastCheckError = err.Error()
			if err := m.saveMeta(meta); err != nil {
				return nil, err
			}
			m.status("ready", "Using cached card database after freshness check failed.")
			return &EnsureResult{DBPath: m.DBPath, Meta: meta, Updated: false}, nil
		}
	}

	m.status("downloading", "Downloading card database...")
	downloaded, nextMeta, notModified, err := m.download(ctx, dbURL, meta, opts.Force)
	if err != nil {
		if hasDB {
			if meta != nil {
				meta.LastCheckedAt = now
				if err := m.saveMeta(meta); err != nil {
					return nil, err
				}
			}
			m.status("ready", "Using cached card database after refresh failed.")
			return &EnsureResult{DBPath: m.DBPath, Meta: meta, Updated: false}, nil
		}
		return nil, err
	}
	if notModified {
		if meta == nil {
			meta = &Metadata{}
		}
		if !latestPublishedAt.IsZero() {
			meta.LastPublishedAt = latestPublishedAt
		}
		meta.LastCheckedAt = now
		meta.LastCheckError = ""
		meta.KnownOutdatedAt = time.Time{}
		if err := m.saveMeta(meta); err != nil {
			return nil, err
		}
		m.status("ready", "Card database is up to date.")
		return &EnsureResult{DBPath: m.DBPath, Meta: meta, Updated: false}, nil
	}

	m.status("installing", "Installing card database...")
	if err := m.install(downloaded); err != nil {
		return nil, err
	}
	if !latestPublishedAt.IsZero() {
		nextMeta.LastPublishedAt = latestPublishedAt
	}
	nextMeta.InstalledAt = now
	nextMeta.LastCheckedAt = now
	nextMeta.LastCheckError = ""
	nextMeta.KnownOutdatedAt = time.Time{}
	if nextMeta.FetchedAt.IsZero() {
		nextMeta.FetchedAt = now
	}
	if err := m.saveMeta(nextMeta); err != nil {
		return nil, err
	}
	m.status("ready", "Card database ready.")
	return &EnsureResult{DBPath: m.DBPath, Meta: nextMeta, Updated: true}, nil
}

func (m *Manager) loadMeta() (*Metadata, error) {
	data, err := os.ReadFile(m.MetaPath)
	if err != nil {
		return nil, err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (m *Manager) saveMeta(meta *Metadata) error {
	if meta == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.MetaPath), 0o755); err != nil {
		return fmt.Errorf("create cards metadata dir: %w", err)
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("encode cards metadata: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(m.MetaPath, data, 0o600); err != nil {
		return fmt.Errorf("write cards metadata: %w", err)
	}
	return nil
}

func (m *Manager) fetchLastPublished(ctx context.Context) (time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(m.LastUpdatedURL), nil)
	if err != nil {
		return time.Time{}, err
	}
	resp, err := m.httpClient().Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return time.Time{}, fmt.Errorf("fetch LAST_PUBLISHED: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return time.Time{}, err
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return time.Time{}, errors.New("LAST_PUBLISHED response is empty")
	}
	t, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func (m *Manager) download(ctx context.Context, dbURL string, meta *Metadata, force bool) ([]byte, *Metadata, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dbURL, nil)
	if err != nil {
		return nil, nil, false, err
	}
	req.Header.Set("Cache-Control", "no-cache")
	if !force && meta != nil {
		if strings.TrimSpace(meta.ETag) != "" {
			req.Header.Set("If-None-Match", meta.ETag)
		} else if strings.TrimSpace(meta.LastModified) != "" {
			req.Header.Set("If-Modified-Since", meta.LastModified)
		}
	}

	resp, err := m.httpClient().Do(req)
	if err != nil {
		return nil, nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil, meta, true, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, false, fmt.Errorf("download card database: HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, false, err
	}
	data, err := maybeGunzip(raw)
	if err != nil {
		return nil, nil, false, err
	}
	nextMeta := &Metadata{
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		FetchedAt:    time.Now().UTC(),
	}
	return data, nextMeta, false, nil
}

func (m *Manager) install(data []byte) error {
	if len(data) == 0 {
		return errors.New("downloaded card database is empty")
	}
	if err := os.MkdirAll(filepath.Dir(m.DBPath), 0o755); err != nil {
		return fmt.Errorf("create cards db dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(m.DBPath), ".pitchstack.sqlite.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp cards db: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp cards db: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp cards db: %w", err)
	}
	if err := os.Rename(tmpPath, m.DBPath); err != nil {
		return fmt.Errorf("install cards db: %w", err)
	}
	return nil
}

func (m *Manager) httpClient() *http.Client {
	if m.HTTPClient != nil {
		return m.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (m *Manager) status(phase, message string) {
	if m.OnStatus != nil {
		m.OnStatus(Status{Phase: phase, Message: message})
	}
}

func maybeGunzip(data []byte) ([]byte, error) {
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		return data, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
