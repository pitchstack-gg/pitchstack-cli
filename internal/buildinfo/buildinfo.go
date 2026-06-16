package buildinfo

import "strings"

var (
	Version = "dev"
	Commit  = "none"
)

func IsDevelopment() bool {
	return IsDevelopmentVersion(Version)
}

func IsDevelopmentVersion(version string) bool {
	version = strings.TrimSpace(version)
	return version == "" || strings.EqualFold(version, "dev")
}
