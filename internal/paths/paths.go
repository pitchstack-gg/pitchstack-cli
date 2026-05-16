package paths

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func appConfigDir() string {
	base, err := os.UserConfigDir()
	if err == nil && base != "" {
		return filepath.Join(base, "pitchstack")
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".config", "pitchstack")
	}
	return "pitchstack"
}

func appCacheDir() string {
	base, err := os.UserCacheDir()
	if err == nil && base != "" {
		return filepath.Join(base, "pitchstack")
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".cache", "pitchstack")
	}
	return filepath.Join("pitchstack", "cache")
}

func SafeProfileName(profileName string) string {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		profileName = "default"
	}
	return regexp.MustCompile(`[^a-zA-Z0-9._-]+`).ReplaceAllString(profileName, "_")
}

func DefaultConfigPath() string {
	return filepath.Join(appConfigDir(), "config.json")
}

func DefaultSessionPath() string {
	return filepath.Join(appConfigDir(), "session.json")
}

func SessionPath(profileName string) string {
	return filepath.Join(appConfigDir(), "sessions", SafeProfileName(profileName)+".json")
}

func CardsCacheDir(profileName string) string {
	return filepath.Join(appCacheDir(), SafeProfileName(profileName), "cards")
}

func CardsDBPath(profileName string) string {
	return filepath.Join(CardsCacheDir(profileName), "pitchstack.sqlite")
}

func CardsDBMetaPath(profileName string) string {
	return filepath.Join(CardsCacheDir(profileName), "meta.json")
}

func CardsImageCacheDir(profileName string) string {
	return filepath.Join(CardsCacheDir(profileName), "images")
}

func SyncCacheDir(profileName string) string {
	return filepath.Join(appCacheDir(), SafeProfileName(profileName), "sync")
}

func SyncDBPath(profileName string) string {
	return filepath.Join(SyncCacheDir(profileName), "powersync.sqlite")
}

func SyncMetaPath(profileName string) string {
	return filepath.Join(SyncCacheDir(profileName), "meta.json")
}
