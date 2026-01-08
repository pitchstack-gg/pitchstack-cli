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

func DefaultConfigPath() string {
	return filepath.Join(appConfigDir(), "config.json")
}

func DefaultSessionPath() string {
	return filepath.Join(appConfigDir(), "session.json")
}

func SessionPath(profileName string) string {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		profileName = "default"
	}
	safe := regexp.MustCompile(`[^a-zA-Z0-9._-]+`).ReplaceAllString(profileName, "_")
	return filepath.Join(appConfigDir(), "sessions", safe+".json")
}
