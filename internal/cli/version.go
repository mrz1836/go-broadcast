package cli

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/mrz1836/go-broadcast/internal/output"
)

// Build information set via ldflags
//
//nolint:gochecknoglobals // Build variables are set via ldflags during compilation
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
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
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
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
