package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	CurrentProfile string             `json:"currentProfile,omitempty"`
	Profiles       map[string]Profile `json:"profiles,omitempty"`
}

type Profile struct {
	BaseURL               string `json:"baseUrl,omitempty"`
	OAuthBaseURL          string `json:"oauthBaseUrl,omitempty"`
	TimeoutSeconds        int    `json:"timeoutSeconds,omitempty"`
	CardsDBURL            string `json:"cardsDbUrl,omitempty"`
	CardsDBLastUpdatedURL string `json:"cardsDbLastUpdatedUrl,omitempty"`
	PowerSyncURL          string `json:"powerSyncUrl,omitempty"`
	SyncEnabled           *bool  `json:"syncEnabled,omitempty"`
}

type Deps struct {
	Path    string
	Config  *Config
	Profile Profile
}

func Default() *Config {
	return &Config{
		CurrentProfile: "default",
		Profiles: map[string]Profile{
			"default": {
				BaseURL:               "https://api.pitchstack.gg",
				OAuthBaseURL:          "https://auth.pitchstack.gg",
				TimeoutSeconds:        30,
				CardsDBURL:            "https://cards.pitchstack.gg/pitchstack/pitchstack.sqlite.gz",
				CardsDBLastUpdatedURL: "https://cards.pitchstack.gg/pitchstack/LAST_PUBLISHED",
			},
		},
	}
}

func (c *Config) Profile(name string) (Profile, bool) {
	if c == nil {
		return Profile{}, false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Profile{}, false
	}
	p, ok := c.Profiles[name]
	return p, ok
}

func Load(path string) (*Config, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("config path must not be empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if cfg.CurrentProfile == "" {
		cfg.CurrentProfile = "default"
	}
	if cfg.Profiles == nil || len(cfg.Profiles) == 0 {
		cfg.Profiles = Default().Profiles
	} else {
		def := Default()
		defProf, _ := def.Profile("default")
		for name, prof := range cfg.Profiles {
			if strings.TrimSpace(prof.BaseURL) == "" {
				prof.BaseURL = defProf.BaseURL
			}
			if strings.TrimSpace(prof.OAuthBaseURL) == "" {
				prof.OAuthBaseURL = defProf.OAuthBaseURL
			}
			if strings.TrimSpace(prof.CardsDBURL) == "" {
				prof.CardsDBURL = defProf.CardsDBURL
			}
			if strings.TrimSpace(prof.CardsDBLastUpdatedURL) == "" {
				prof.CardsDBLastUpdatedURL = defProf.CardsDBLastUpdatedURL
			}
			cfg.Profiles[name] = prof
		}
	}
	return &cfg, nil
}

func WriteDefault(path string) error {
	cfg := Default()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
