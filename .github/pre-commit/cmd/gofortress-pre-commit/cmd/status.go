package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/git"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/output"
)

// statusCmd represents the status command
//
//nolint:gochecknoglobals // Required by cobra
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installation status of git hooks",
	Long: `Show the current installation status of GoFortress pre-commit hooks.

This command displays:
  - Which hooks are installed
  - Whether they are GoFortress hooks or other hooks
  - File permissions and last modified time
  - Configuration status
  - Any conflicts or issues`,
	Example: `  # Show status of all hook types
  gofortress-pre-commit status

  # Show verbose status information
  gofortress-pre-commit status --verbose`,
	RunE: runStatus,
}

func runStatus(_ *cobra.Command, _ []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		printError("Failed to load configuration: %v", err)
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get the repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		printError("Failed to find git repository: %v", err)
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	if verbose {
		printInfo("Repository root: %s", repoRoot)
		printInfo("Pre-commit directory: %s", cfg.Directory)
		printInfo("System enabled: %t", cfg.Enabled)
	}

	// Create installer for status checking
	installer := git.NewInstallerWithConfig(repoRoot, cfg.Directory, cfg)

	// Check status of common hook types
	supportedHooks := []string{"pre-commit", "pre-push", "commit-msg", "post-commit"}

	printHeader("GoFortress Pre-commit System Status")

	// System-level status
	if cfg.Enabled {
		printSuccess("✓ Pre-commit system is enabled")
	} else {
		printWarning("⚠ Pre-commit system is disabled (ENABLE_PRE_COMMIT_SYSTEM=false)")
	}

	// Hook status
	printSubheader("Git Hook Status")

	foundHooks := false
	for _, hookType := range supportedHooks {
		status, err := installer.GetInstallationStatus(hookType)
		if err != nil {
			printError("Failed to check %s hook status: %v", hookType, err)
			continue
		}

		if !status.Installed && !status.ConflictingHook {
			if verbose {
				printDetail("  %s: Not installed", hookType)
			}
			continue
		}

		foundHooks = true

		if status.IsOurHook {
			if status.Executable {
				printSuccess("  ✓ %s: %s", hookType, status.Message)
			} else {
				printWarning("  ⚠ %s: %s", hookType, status.Message)
			}
		} else if status.ConflictingHook {
			printWarning("  ⚠ %s: %s", hookType, status.Message)
			if verbose {
				printDetail("    Use --force to overwrite existing hook")
			}
		}

		if verbose && (status.Installed || status.ConflictingHook) {
			printDetail("    Path: %s", status.HookPath)
			printDetail("    Permissions: %s", status.FileMode)
			printDetail("    Modified: %s", status.ModTime.Format("2006-01-02 15:04:05"))
		}
	}

	if !foundHooks {
		printWarning("No hooks are currently installed")
		printInfo("Run 'gofortress-pre-commit install' to install pre-commit hooks")
	}

	// Configuration status
	if verbose {
		printSubheader("Configuration Status")
		printDetail("  Checks enabled:")
		printDetail("    fumpt: %t", cfg.Checks.Fumpt)
		printDetail("    lint: %t", cfg.Checks.Lint)
		printDetail("    mod-tidy: %t", cfg.Checks.ModTidy)
		printDetail("    whitespace: %t", cfg.Checks.Whitespace)
		printDetail("    eof: %t", cfg.Checks.EOF)
		printDetail("  Timeout: %d seconds", cfg.Timeout)
		printDetail("  Parallel workers: %d", cfg.Performance.ParallelWorkers)
	}

	return nil
}

func printHeader(text string) {
	formatter := output.NewDefault()
	formatter.Info("\n=== %s ===", text)
}

func printSubheader(text string) {
	formatter := output.NewDefault()
	formatter.Info("\n%s:", text)
}

func printDetail(format string, args ...interface{}) {
	formatter := output.NewDefault()
	formatter.Info(format, args...)
}
