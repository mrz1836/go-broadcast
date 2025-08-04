// Package cli provides command-line interface functionality for go-broadcast.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/spf13/cobra"
)

// DiagnosticInfo contains comprehensive system diagnostic information.
//
// This structure is designed to provide all the information needed for
// troubleshooting issues with go-broadcast, including system details,
// version information, and configuration status.
type DiagnosticInfo struct {
	Timestamp   time.Time             `json:"timestamp"`
	Version     DiagnosticVersionInfo `json:"version"`
	System      DiagnosticSystemInfo  `json:"system"`
	Environment map[string]string     `json:"environment"`
	GitVersion  string                `json:"git_version"`
	GHVersion   string                `json:"gh_cli_version"`
	Config      DiagnosticConfigInfo  `json:"config"`
}

// DiagnosticVersionInfo contains version details for go-broadcast diagnostics.
type DiagnosticVersionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
	Date    string `json:"date,omitempty"`
	GoVer   string `json:"go_version"`
	Built   string `json:"built_by,omitempty"`
}

// DiagnosticSystemInfo contains system and runtime information.
type DiagnosticSystemInfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	NumCPU   int    `json:"num_cpu"`
	Hostname string `json:"hostname"`
	UserHome string `json:"user_home"`
}

// DiagnosticConfigInfo contains configuration file status information.
type DiagnosticConfigInfo struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Valid  bool   `json:"valid"`
	Error  string `json:"error,omitempty"`
}

// diagnoseCmd is the global diagnose command instance
//
//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Collect diagnostic information",
	Long: `Collects comprehensive system information for troubleshooting.

This command gathers:
- System information (OS, architecture, CPU count)
- go-broadcast version and build details
- Git and GitHub CLI versions
- Environment variables (with sensitive data redacted)
- Configuration file status and validation

All output is in JSON format for easy analysis and sharing with support.`,
	Example: `  # Collect diagnostic information
  go-broadcast diagnose

  # Save diagnostics to file
  go-broadcast diagnose > diagnostics.json`,
	RunE: runDiagnose,
}

// createDiagnoseCmdWithVerbose creates a diagnose command with verbose logging support.
//
// Parameters:
// - config: LogConfig containing logging and debug configuration
//
// Returns:
// - Cobra command configured for diagnostic operations with verbose support
func createDiagnoseCmdWithVerbose(config *LogConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "diagnose",
		Short: "Collect diagnostic information",
		Long: `Collects comprehensive system information for troubleshooting.

This command gathers:
- System information (OS, architecture, CPU count)
- go-broadcast version and build details
- Git and GitHub CLI versions
- Environment variables (with sensitive data redacted)
- Configuration file status and validation

All output is in JSON format for easy analysis and sharing with support.`,
		Example: `  # Collect diagnostic information
  go-broadcast diagnose

  # Save diagnostics to file
  go-broadcast diagnose > diagnostics.json

  # Include diagnostics in verbose logging session
  go-broadcast diagnose && go-broadcast sync -vvv`,
		RunE: createRunDiagnoseWithVerbose(config),
	}
}

// createRunDiagnoseWithVerbose creates a diagnose run function with verbose logging support.
//
// This function creates a run function that collects comprehensive diagnostic
// information and outputs it in JSON format for troubleshooting purposes.
//
// Parameters:
// - config: LogConfig containing logging and debug configuration
//
// Returns:
// - Function that can be used as RunE for Cobra diagnose commands
func createRunDiagnoseWithVerbose(config *LogConfig) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		// Collect all diagnostic information
		info := &DiagnosticInfo{
			Timestamp:   time.Now(),
			Version:     getVersionInfo(),
			System:      getSystemInfo(),
			Environment: collectEnvironment(ctx),
			GitVersion:  getGitVersion(ctx),
			GHVersion:   getGHCLIVersion(ctx),
			Config:      getConfigInfo(ctx, config),
		}

		// Output as formatted JSON to stdout
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(info); err != nil {
			return fmt.Errorf("failed to encode diagnostic information: %w", err)
		}

		return nil
	}
}

// getVersionInfo collects version information for go-broadcast.
//
// This function retrieves build-time version information including
// version number, commit hash, build date, and Go version.
//
// Returns:
// - DiagnosticVersionInfo structure with version details
func getVersionInfo() DiagnosticVersionInfo {
	// These variables are set at build time via ldflags
	// For now, we'll use runtime information
	return DiagnosticVersionInfo{
		Version: "dev", // Will be set via ldflags in production builds
		Commit:  "",    // Will be set via ldflags in production builds
		Date:    "",    // Will be set via ldflags in production builds
		GoVer:   runtime.Version(),
		Built:   "source", // Will be set via ldflags in production builds
	}
}

// getSystemInfo collects system and runtime information.
//
// This function gathers basic system information including operating system,
// architecture, CPU count, hostname, and user home directory.
//
// Returns:
// - DiagnosticSystemInfo structure with system details
func getSystemInfo() DiagnosticSystemInfo {
	hostname, _ := os.Hostname()
	homeDir, _ := os.UserHomeDir()

	return DiagnosticSystemInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		NumCPU:   runtime.NumCPU(),
		Hostname: hostname,
		UserHome: homeDir,
	}
}

// collectEnvironment gathers relevant environment variables for diagnostics.
//
// This function collects specific environment variables that are relevant
// to go-broadcast operation while automatically redacting sensitive values
// like tokens and secrets.
//
// Parameters:
// - ctx: Context for cancellation control (currently unused but kept for future extension)
//
// Returns:
// - Map of environment variable names to their values (sensitive data redacted)
func collectEnvironment(_ context.Context) map[string]string {
	env := make(map[string]string)

	// List of environment variables relevant to go-broadcast
	relevantVars := []string{
		"PATH",
		"HOME",
		"USER",
		"SHELL",
		"GH_TOKEN",
		"GITHUB_TOKEN",
		"GO_BROADCAST_CONFIG",
		"NO_COLOR",
		"TERM",
		"CI",
		"GITHUB_ACTIONS",
		"GITHUB_WORKSPACE",
		"GITHUB_REPOSITORY",
		"RUNNER_OS",
	}

	for _, key := range relevantVars {
		value := os.Getenv(key)
		if value != "" {
			// Redact sensitive values automatically
			if isSensitiveEnvVar(key) {
				if len(value) > 8 {
					value = value[:4] + "***REDACTED***"
				} else {
					value = "***REDACTED***"
				}
			}
			env[key] = value
		}
	}

	return env
}

// isSensitiveEnvVar checks if an environment variable contains sensitive data.
//
// Parameters:
// - key: Environment variable name to check
//
// Returns:
// - true if the variable likely contains sensitive data
func isSensitiveEnvVar(key string) bool {
	sensitivePatterns := []string{
		"TOKEN",
		"SECRET",
		"KEY",
		"PASSWORD",
		"PASS",
	}

	keyUpper := strings.ToUpper(key)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(keyUpper, pattern) {
			return true
		}
	}
	return false
}

// getGitVersion retrieves the Git version information.
//
// This function executes 'git --version' to determine the installed
// Git version, which is important for troubleshooting git-related issues.
//
// Parameters:
// - ctx: Context for cancellation and timeout control
//
// Returns:
// - Git version string or error description
func getGitVersion(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return strings.TrimSpace(string(output))
}

// getGHCLIVersion retrieves the GitHub CLI version information.
//
// This function executes 'gh --version' to determine the installed
// GitHub CLI version, which is essential for API operations.
//
// Parameters:
// - ctx: Context for cancellation and timeout control
//
// Returns:
// - GitHub CLI version string or error description
func getGHCLIVersion(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "gh", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	// Extract first line which contains the main version info
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return "unknown"
}

// getConfigInfo analyzes the configuration file status.
//
// This function checks if the configuration file exists and attempts
// to validate it, providing diagnostic information about any issues.
//
// Parameters:
// - ctx: Context for cancellation control
// - logConfig: LogConfig containing the config file path
//
// Returns:
// - DiagnosticConfigInfo structure with configuration file status
func getConfigInfo(ctx context.Context, logConfig *LogConfig) DiagnosticConfigInfo {
	info := DiagnosticConfigInfo{
		Path:   logConfig.ConfigFile,
		Exists: false,
		Valid:  false,
	}

	// Check if file exists
	if _, err := os.Stat(info.Path); err == nil {
		info.Exists = true

		// Try to load and validate configuration
		cfg, loadErr := config.Load(info.Path)
		if loadErr != nil {
			info.Error = fmt.Sprintf("load error: %v", loadErr)
			return info
		}

		// Attempt validation
		if validationErr := cfg.ValidateWithLogging(ctx, logConfig); validationErr != nil {
			info.Error = fmt.Sprintf("validation error: %v", validationErr)
			return info
		}

		info.Valid = true
	} else {
		info.Error = fmt.Sprintf("file not found: %v", err)
	}

	return info
}

// runDiagnose is the global diagnose command run function.
//
// This function collects diagnostic information using the global flags
// and outputs it in JSON format for troubleshooting purposes.
//
// Parameters:
// - cmd: Cobra command being executed
// - args: Command line arguments (unused)
//
// Returns:
// - Error if diagnosis fails
func runDiagnose(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Create a LogConfig from global flags
	logConfig := &LogConfig{
		ConfigFile: globalFlags.ConfigFile,
		LogLevel:   globalFlags.LogLevel,
		DryRun:     globalFlags.DryRun,
		LogFormat:  "text", // Default format for global version
	}

	// Collect all diagnostic information
	info := &DiagnosticInfo{
		Timestamp:   time.Now(),
		Version:     getVersionInfo(),
		System:      getSystemInfo(),
		Environment: collectEnvironment(ctx),
		GitVersion:  getGitVersion(ctx),
		GHVersion:   getGHCLIVersion(ctx),
		Config:      getConfigInfo(ctx, logConfig),
	}

	// Output as formatted JSON to stdout
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(info); err != nil {
		return fmt.Errorf("failed to encode diagnostic information: %w", err)
	}

	return nil
}

// createDiagnoseCmd creates an isolated diagnose command with the given flags.
//
// This function creates a diagnose command that uses the legacy Flags structure
// for compatibility with the isolated command pattern.
//
// Parameters:
// - flags: Flags containing basic CLI configuration
//
// Returns:
// - Cobra command configured for diagnostic operations
func createDiagnoseCmd(flags *Flags) *cobra.Command {
	return &cobra.Command{
		Use:   "diagnose",
		Short: "Collect diagnostic information",
		Long: `Collects comprehensive system information for troubleshooting.

This command gathers:
- System information (OS, architecture, CPU count)
- go-broadcast version and build details
- Git and GitHub CLI versions
- Environment variables (with sensitive data redacted)
- Configuration file status and validation

All output is in JSON format for easy analysis and sharing with support.`,
		Example: `  # Collect diagnostic information
  go-broadcast diagnose

  # Save diagnostics to file
  go-broadcast diagnose > diagnostics.json`,
		RunE: createRunDiagnose(flags),
	}
}

// createRunDiagnose creates a diagnose run function with the given flags.
//
// This function creates a run function that collects diagnostic information
// using the legacy Flags structure for isolated command execution.
//
// Parameters:
// - flags: Flags containing basic CLI configuration
//
// Returns:
// - Function that can be used as RunE for Cobra diagnose commands
func createRunDiagnose(flags *Flags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		// Create a LogConfig from the provided flags
		logConfig := &LogConfig{
			ConfigFile: flags.ConfigFile,
			LogLevel:   flags.LogLevel,
			DryRun:     flags.DryRun,
			LogFormat:  "text", // Default format for isolated version
		}

		// Collect all diagnostic information
		info := &DiagnosticInfo{
			Timestamp:   time.Now(),
			Version:     getVersionInfo(),
			System:      getSystemInfo(),
			Environment: collectEnvironment(ctx),
			GitVersion:  getGitVersion(ctx),
			GHVersion:   getGHCLIVersion(ctx),
			Config:      getConfigInfo(ctx, logConfig),
		}

		// Output as formatted JSON to stdout
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(info); err != nil {
			return fmt.Errorf("failed to encode diagnostic information: %w", err)
		}

		return nil
	}
}
