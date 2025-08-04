// Package cmd implements the CLI commands for gofortress-pre-commit
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/git"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Required by cobra
var (
	force     bool
	hookTypes []string
)

// installCmd represents the install command
//
//nolint:gochecknoglobals // Required by cobra
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install git pre-commit hooks",
	Long: `Install the GoFortress pre-commit system hooks into your git repository.

This command will:
  - Create .git/hooks/pre-commit (or other specified hook types)
  - Make the hook executable
  - Preserve any existing hooks (unless --force is used)
  - Configure the hook to use the GoFortress pre-commit system`,
	Example: `  # Install pre-commit hook
  gofortress-pre-commit install

  # Force install, overwriting existing hooks
  gofortress-pre-commit install --force

  # Install multiple hook types
  gofortress-pre-commit install --hook-type pre-commit --hook-type pre-push`,
	RunE: runInstall,
}

//nolint:gochecknoinits // Required by cobra
func init() {
	installCmd.Flags().BoolVarP(&force, "force", "f", false, "Force installation, overwriting existing hooks")
	installCmd.Flags().StringSliceVar(&hookTypes, "hook-type", []string{"pre-commit"}, "Hook types to install")
}

func runInstall(_ *cobra.Command, _ []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if pre-commit system is enabled
	if !cfg.Enabled {
		printWarning("Pre-commit system is disabled in configuration (ENABLE_PRE_COMMIT_SYSTEM=false)")
		printInfo("To enable, set ENABLE_PRE_COMMIT_SYSTEM=true in .github/.env.shared")
		return nil
	}

	// Get the repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	if verbose {
		printInfo("Repository root: %s", repoRoot)
	}

	// Create installer with configuration for enhanced validation
	installer := git.NewInstallerWithConfig(repoRoot, cfg.Directory, cfg)

	// Install each hook type
	installed := make([]string, 0, len(hookTypes))
	for _, hookType := range hookTypes {
		if verbose {
			printInfo("Installing %s hook...", hookType)
		}

		err := installer.InstallHook(hookType, force)
		if err != nil {
			if !force && os.IsExist(err) {
				printWarning("Hook already exists: %s (use --force to overwrite)", hookType)
				continue
			}
			return fmt.Errorf("failed to install %s hook: %w", hookType, err)
		}

		installed = append(installed, hookType)
	}

	// Summary
	if len(installed) > 0 {
		printSuccess("Successfully installed hooks: %v", installed)
		printInfo("Run 'git commit' to test the pre-commit hook")
		printInfo("Run '%s run --help' to see available checks", filepath.Base(os.Args[0]))
	} else {
		printWarning("No hooks were installed")
	}

	return nil
}
