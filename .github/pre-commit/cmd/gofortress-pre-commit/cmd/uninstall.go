package cmd

import (
	"fmt"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/git"
	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
//
//nolint:gochecknoglobals // Required by cobra
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall git pre-commit hooks",
	Long: `Uninstall the GoFortress pre-commit system hooks from your git repository.

This command will:
  - Remove .git/hooks/pre-commit (or other specified hook types)
  - Only remove hooks that were installed by GoFortress pre-commit system
  - Preserve any hooks not created by this tool`,
	Example: `  # Uninstall pre-commit hook
  gofortress-pre-commit uninstall

  # Uninstall multiple hook types
  gofortress-pre-commit uninstall --hook-type pre-commit --hook-type pre-push`,
	RunE: runUninstall,
}

//nolint:gochecknoinits // Required by cobra
func init() {
	uninstallCmd.Flags().StringSliceVar(&hookTypes, "hook-type", []string{"pre-commit"}, "Hook types to uninstall")
}

func runUninstall(_ *cobra.Command, _ []string) error {
	// Get the repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	if verbose {
		printInfo("Repository root: %s", repoRoot)
	}

	// Create installer (also handles uninstallation)
	installer := git.NewInstaller(repoRoot, ".github/pre-commit")

	// Uninstall each hook type
	var uninstalled []string
	var notFound []string

	for _, hookType := range hookTypes {
		if verbose {
			printInfo("Uninstalling %s hook...", hookType)
		}

		removed, err := installer.UninstallHook(hookType)
		if err != nil {
			return fmt.Errorf("failed to uninstall %s hook: %w", hookType, err)
		}

		if removed {
			uninstalled = append(uninstalled, hookType)
		} else {
			notFound = append(notFound, hookType)
		}
	}

	// Summary
	if len(uninstalled) > 0 {
		printSuccess("Successfully uninstalled hooks: %v", uninstalled)
	}
	if len(notFound) > 0 {
		printInfo("Hooks not found or not managed by GoFortress: %v", notFound)
	}
	if len(uninstalled) == 0 && len(notFound) == 0 {
		printWarning("No hooks were uninstalled")
	}

	return nil
}
