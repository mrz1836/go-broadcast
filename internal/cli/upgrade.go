package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/output"
	versionpkg "github.com/mrz1836/go-broadcast/internal/version"
)

var (
	// ErrDevVersionNoForce is returned when trying to upgrade a dev version without --force
	ErrDevVersionNoForce = errors.New("cannot upgrade development build without --force")
	// ErrVersionParseFailed is returned when version cannot be parsed from output
	ErrVersionParseFailed = errors.New("could not parse version from output")
)

// UpgradeConfig holds configuration for the upgrade command
type UpgradeConfig struct {
	Force     bool
	CheckOnly bool
}

// newUpgradeCmd creates the upgrade command
func newUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade go-broadcast to the latest version",
		Long: `Upgrade the go-broadcast system to the latest version available.

This command will:
  - Check the latest version available on GitHub
  - Compare with the currently installed version
  - Upgrade if a newer version is available`,
		Example: `  # Check for available updates
  go-broadcast upgrade --check

  # Upgrade to latest version
  go-broadcast upgrade

  # Force upgrade even if already on latest
  go-broadcast upgrade --force`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			config := UpgradeConfig{}
			var err error

			config.Force, err = cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

			config.CheckOnly, err = cmd.Flags().GetBool("check")
			if err != nil {
				return err
			}

			return runUpgradeWithConfig(cmd, config)
		},
	}

	// Add flags
	cmd.Flags().BoolP("force", "f", false, "Force upgrade even if already on latest version")
	cmd.Flags().Bool("check", false, "Check for updates without upgrading")
	cmd.Flags().BoolP("verbose", "v", false, "Show release notes after upgrade")

	return cmd
}

func runUpgradeWithConfig(cmd *cobra.Command, config UpgradeConfig) error {
	currentVersion := GetCurrentVersion()

	// Handle development version or commit hash
	if currentVersion == "dev" || currentVersion == "" || isLikelyCommitHash(currentVersion) {
		if !config.Force && !config.CheckOnly {
			output.Warn(fmt.Sprintf("Current version appears to be a development build (%s)", currentVersion))
			output.Info("Use --force to upgrade anyway")
			return ErrDevVersionNoForce
		}
	}

	output.Info(fmt.Sprintf("Current version: %s", formatVersion(currentVersion)))

	// Fetch latest release
	output.Info("Checking for updates...")
	release, err := versionpkg.GetLatestRelease("mrz1836", "go-broadcast")
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	output.Info(fmt.Sprintf("Latest version: %s", formatVersion(latestVersion)))

	// Compare versions
	isNewer := versionpkg.IsNewerVersion(currentVersion, latestVersion)

	if !isNewer && !config.Force {
		output.Success(fmt.Sprintf("You are already on the latest version (%s)", formatVersion(currentVersion)))
		return nil
	}

	if config.CheckOnly {
		if isNewer {
			output.Warn(fmt.Sprintf("A newer version is available: %s → %s", formatVersion(currentVersion), formatVersion(latestVersion)))
			output.Info("Run 'go-broadcast upgrade' to upgrade")
		} else {
			output.Success("You are on the latest version")
		}
		return nil
	}

	// Perform upgrade
	if isNewer {
		output.Info(fmt.Sprintf("Upgrading from %s to %s...", formatVersion(currentVersion), formatVersion(latestVersion)))
	} else if config.Force {
		output.Info(fmt.Sprintf("Force reinstalling version %s...", formatVersion(latestVersion)))
	}

	// Run go install command
	installCmd := fmt.Sprintf("github.com/mrz1836/go-broadcast/cmd/go-broadcast@v%s", latestVersion)

	output.Info(fmt.Sprintf("Running: go install %s", installCmd))

	execCmd := exec.CommandContext(context.Background(), "go", "install", installCmd) //nolint:gosec // Command is constructed safely
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("failed to upgrade: %w", err)
	}

	output.Success(fmt.Sprintf("Successfully upgraded to version %s", formatVersion(latestVersion)))

	// Show release notes if available and verbose
	verbose, _ := cmd.Flags().GetBool("verbose")
	if release.Body != "" && verbose {
		output.Info(fmt.Sprintf("\nRelease notes for v%s:", latestVersion))
		lines := strings.Split(release.Body, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				output.Info(fmt.Sprintf("  %s", line))
			}
		}
	}

	return nil
}

func formatVersion(v string) string {
	if v == "dev" || v == "" {
		return "dev"
	}
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

// GetInstalledVersion attempts to get the version of the installed binary
func GetInstalledVersion() (string, error) {
	// Try to run the binary with --version flag
	cmd := exec.CommandContext(context.Background(), "go-broadcast", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	// Parse the version from output
	// Expected format: "go-broadcast version X.Y.Z"
	outputStr := strings.TrimSpace(string(output))
	parts := strings.Fields(outputStr)

	for i, part := range parts {
		if part == "version" && i+1 < len(parts) {
			version := parts[i+1]
			// Clean up version string
			version = strings.TrimPrefix(version, "v")
			return version, nil
		}
	}

	return "", fmt.Errorf("%w: %s", ErrVersionParseFailed, outputStr)
}

// CheckGoInstalled verifies that Go is installed and available
func CheckGoInstalled() error {
	cmd := exec.CommandContext(context.Background(), "go", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go is not installed or not in PATH: %w", err)
	}
	return nil
}

// GetGoPath returns the GOPATH/bin directory where binaries are installed
func GetGoPath() (string, error) {
	cmd := exec.CommandContext(context.Background(), "go", "env", "GOPATH")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GOPATH: %w", err)
	}

	gopath := strings.TrimSpace(string(output))
	if gopath == "" {
		// Use default GOPATH
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		gopath = fmt.Sprintf("%s/go", home)
	}

	return fmt.Sprintf("%s/bin", gopath), nil
}

// IsInPath checks if go-broadcast binary is in PATH
func IsInPath() bool {
	_, err := exec.LookPath("go-broadcast")
	return err == nil
}

// GetBinaryLocation returns the location of the go-broadcast binary
func GetBinaryLocation() (string, error) {
	if runtime.GOOS == "windows" {
		return exec.LookPath("go-broadcast.exe")
	}
	return exec.LookPath("go-broadcast")
}

// isLikelyCommitHash checks if a version string looks like a commit hash
func isLikelyCommitHash(version string) bool {
	// Remove any -dirty suffix
	version = strings.TrimSuffix(version, "-dirty")

	// Commit hashes are typically 7-40 hex characters
	if len(version) < 7 || len(version) > 40 {
		return false
	}

	// Check if all characters are valid hex
	for _, c := range version {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}

	return true
}

// GetCurrentVersion returns the current version of go-broadcast
func GetCurrentVersion() string {
	return GetVersion()
}
