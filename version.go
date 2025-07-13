// Package email version information
package email

// Version information
const (
	// Version is the current version of the go-email package
	Version = "v1.0.2"

	// VersionMajor is the major version number
	VersionMajor = 1

	// VersionMinor is the minor version number
	VersionMinor = 0

	// VersionPatch is the patch version number
	VersionPatch = 1

	// VersionPreRelease is the pre-release version identifier
	VersionPreRelease = ""

	// BuildDate is the date the binary was built (set during build)
	BuildDate = ""

	// GitCommit is the git commit hash (set during build)
	GitCommit = ""
)

// GetVersion returns the full version string
func GetVersion() string {
	v := Version
	if VersionPreRelease != "" {
		v += "-" + VersionPreRelease
	}
	return v
}

// VersionInfo contains detailed version information
type VersionInfo struct {
	Version    string `json:"version"`
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
	Patch      int    `json:"patch"`
	PreRelease string `json:"preRelease,omitempty"`
	BuildDate  string `json:"buildDate,omitempty"`
	GitCommit  string `json:"gitCommit,omitempty"`
}

// GetVersionInfo returns detailed version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:    GetVersion(),
		Major:      VersionMajor,
		Minor:      VersionMinor,
		Patch:      VersionPatch,
		PreRelease: VersionPreRelease,
		BuildDate:  BuildDate,
		GitCommit:  GitCommit,
	}
}
