package cli

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/output"
)

// Build information set via ldflags
//
//nolint:gochecknoglobals // Build variables are set via ldflags during compilation
var (
	version   = devVersionString
	commit    = unknownString
	buildDate = unknownString
)

// VersionInfo contains version information
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// printVersion prints version information based on the format
func printVersion(jsonFormat bool) error {
	info := VersionInfo{
		Version:   getVersionWithFallback(),
		Commit:    getCommitWithFallback(),
		BuildDate: getBuildDateWithFallback(),
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	if jsonFormat {
		encoder := json.NewEncoder(output.Stdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(info)
	}

	// Text output
	output.Info(fmt.Sprintf("go-broadcast %s", info.Version))
	output.Info(fmt.Sprintf("Commit:     %s", info.Commit))
	output.Info(fmt.Sprintf("Build Date: %s", info.BuildDate))
	output.Info(fmt.Sprintf("Go Version: %s", info.GoVersion))
	output.Info(fmt.Sprintf("Platform:   %s/%s", info.OS, info.Arch))

	return nil
}

// SetVersionInfo allows setting version information programmatically
// This is useful for testing or when not using ldflags
func SetVersionInfo(v, c, d string) {
	if v != "" {
		version = v
	}
	if c != "" {
		commit = c
	}
	if d != "" {
		buildDate = d
	}
}

// GetVersion returns the current version string with fallback to build info
func GetVersion() string {
	return getVersionWithFallback()
}

// GetCommit returns the current commit hash with fallback to build info
func GetCommit() string {
	return getCommitWithFallback()
}

// GetBuildDate returns the build date with fallback to build info
func GetBuildDate() string {
	return getBuildDateWithFallback()
}

// GetVersionInfo returns complete version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   getVersionWithFallback(),
		Commit:    getCommitWithFallback(),
		BuildDate: getBuildDateWithFallback(),
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// getVersionWithFallback returns the version information with fallback to BuildInfo
func getVersionWithFallback() string {
	// If version was set via ldflags, use it
	if version != devVersionString && version != "" {
		return version
	}

	// Try to get version from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		// Check if there's a module version (from go install @version)
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			// For go install @version, use the version as-is (already includes 'v' prefix)
			return info.Main.Version
		}

		// Try to get VCS revision as fallback for development builds
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				// Use short commit hash for readability
				if len(setting.Value) > 7 {
					return setting.Value[:7]
				}
				return setting.Value
			}
		}
	}

	// Default to dev version string if nothing else is available
	return devVersionString
}

// getCommitWithFallback returns the commit hash with fallback to BuildInfo
func getCommitWithFallback() string {
	// If commit was set via ldflags, use it
	if commit != unknownString && commit != "" {
		return commit
	}

	// Try to get from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				// For commit display, use short hash for readability
				if len(setting.Value) > 7 {
					return setting.Value[:7]
				}
				return setting.Value
			}
		}

		// For go install builds, try to extract commit from module sum if available
		if info.Main.Sum != "" {
			// Module sum format: h1:base64hash - extract first 7 chars of hash
			if parts := strings.Split(info.Main.Sum, ":"); len(parts) == 2 && len(parts[1]) >= 7 {
				return parts[1][:7]
			}
		}
	}

	return unknownString
}

// getBuildDateWithFallback returns the build date with fallback to BuildInfo
func getBuildDateWithFallback() string {
	// If build date was set via ldflags, use it
	if buildDate != unknownString && buildDate != "" {
		return buildDate
	}

	// Try to get from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.time" && setting.Value != "" {
				// VCS time is in RFC3339 format, convert to a more readable format
				if t, err := time.Parse(time.RFC3339, setting.Value); err == nil {
					return t.Format("2006-01-02_15:04:05_UTC")
				}
				return setting.Value
			}
		}

		// For go install builds without VCS info, use a generic marker
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return "go-install"
		}
	}

	return unknownString
}
