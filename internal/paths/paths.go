package paths

import (
	"os"
	"path/filepath"
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
