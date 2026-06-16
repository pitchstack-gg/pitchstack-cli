package buildinfo

import "strings"

var (
	Version = "dev"
	Commit  = "none"
)

func IsDevelopment() bool {
	version := strings.TrimSpace(Version)
	return version == "" || strings.EqualFold(version, "dev")
}
