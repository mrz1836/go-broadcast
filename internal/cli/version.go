package cli

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/spf13/cobra"
)

// Build information set via ldflags
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

// initVersion initializes version command flags
func initVersion() {
	versionCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output version in JSON format")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version information including commit hash, build date, and platform details.`,
	Example: `  # Show version information
  go-broadcast version

  # Output in JSON format
  go-broadcast version --json`,
	RunE: runVersion,
}

func runVersion(_ *cobra.Command, _ []string) error {
	info := VersionInfo{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	if jsonOutput {
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
