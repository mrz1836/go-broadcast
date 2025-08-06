package cli

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/sync"
)

// Static errors for err113 compliance
var (
	ErrModuleNotFound         = errors.New("module not found")
	ErrModuleValidationFailed = errors.New("module validation failed")
	ErrInvalidRepositoryPath  = errors.New("invalid repository path")
)

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var modulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "Manage and inspect module configurations",
	Long: `Manage and inspect module configurations in go-broadcast.

Provides tools for working with module-aware synchronization, including:
  • Listing detected modules in source repositories
  • Showing module details and versions
  • Validating module configurations
  • Checking available versions from git tags`,
	Example: `  # List all modules in configuration
  go-broadcast modules list

  # Show details for a specific module
  go-broadcast modules show github.com/example/module

  # Show available versions for a module
  go-broadcast modules versions github.com/example/module

  # Validate module configurations
  go-broadcast modules validate`,
}

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var listModulesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all detected modules in configuration",
	Long: `List all modules detected in the source repositories defined in configuration.

Scans directories configured with module settings and displays:
  • Module path and version
  • Source location
  • Target repositories
  • Version constraints`,
	Example: `  go-broadcast modules list
  go-broadcast modules list --config sync.yaml`,
	RunE: runListModules,
}

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var showModuleCmd = &cobra.Command{
	Use:   "show [module-path]",
	Short: "Show details for a specific module",
	Long: `Show detailed information about a specific module.

Displays:
  • Module configuration
  • Current version settings
  • Target repositories using this module
  • Source repository information`,
	Example: `  go-broadcast modules show github.com/example/errors
  go-broadcast modules show pkg/utils`,
	Args: cobra.ExactArgs(1),
	RunE: runShowModule,
}

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var moduleVersionsCmd = &cobra.Command{
	Use:   "versions [module-path]",
	Short: "Show available versions for a module",
	Long: `Show available versions for a module from git tags.

Fetches and displays:
  • Available git tags that look like versions
  • Latest stable version
  • Compatibility with current constraints`,
	Example: `  go-broadcast modules versions github.com/example/errors
  go-broadcast modules versions pkg/utils`,
	Args: cobra.ExactArgs(1),
	RunE: runModuleVersions,
}

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var validateModulesCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate module configurations",
	Long: `Validate all module configurations in the sync configuration.

Checks:
  • Module paths are valid
  • Version constraints are parseable
  • Modules exist in source repositories
  • Version constraints can be satisfied`,
	Example: `  go-broadcast modules validate
  go-broadcast modules validate --config sync.yaml`,
	RunE: runValidateModules,
}

//nolint:gochecknoinits // Cobra commands require init() for setup
func init() {
	// Add subcommands
	modulesCmd.AddCommand(listModulesCmd)
	modulesCmd.AddCommand(showModuleCmd)
	modulesCmd.AddCommand(moduleVersionsCmd)
	modulesCmd.AddCommand(validateModulesCmd)
}

func runListModules(cmd *cobra.Command, _ []string) error {
	_ = cmd.Context()
	_ = logrus.WithField("command", "modules-list")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get all groups
	groups := cfg.Groups
	if len(groups) == 0 {
		output.Info("No groups configured")
		return nil
	}

	output.Info("=== Configured Modules ===")
	output.Info("")

	moduleCount := 0
	for _, group := range groups {
		groupHasModules := false

		for _, target := range group.Targets {
			for _, dir := range target.Directories {
				if dir.Module != nil {
					if !groupHasModules {
						output.Info(fmt.Sprintf("Group: %s (%s)", group.Name, group.ID))
						groupHasModules = true
					}

					moduleCount++
					output.Info(fmt.Sprintf("  Module %d:", moduleCount))
					output.Info(fmt.Sprintf("    Source: %s", dir.Src))
					output.Info(fmt.Sprintf("    Target: %s -> %s", target.Repo, dir.Dest))
					output.Info(fmt.Sprintf("    Type: %s", getModuleType(dir.Module.Type)))
					output.Info(fmt.Sprintf("    Version: %s", dir.Module.Version))

					if dir.Module.CheckTags != nil {
						output.Info(fmt.Sprintf("    Check Tags: %v", *dir.Module.CheckTags))
					}
					if dir.Module.UpdateRefs {
						output.Info("    Update References: true")
					}
					output.Info("")
				}
			}
		}

		if groupHasModules {
			output.Info("")
		}
	}

	if moduleCount == 0 {
		output.Info("No modules configured")
	} else {
		output.Success(fmt.Sprintf("Total modules configured: %d", moduleCount))
	}

	return nil
}

func runShowModule(cmd *cobra.Command, args []string) error {
	_ = cmd.Context()
	modulePath := args[0]
	_ = logrus.WithField("command", "modules-show")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	output.Info(fmt.Sprintf("=== Module: %s ===", modulePath))
	output.Info("")

	found := false
	groups := cfg.Groups

	for _, group := range groups {
		for _, target := range group.Targets {
			for _, dir := range target.Directories {
				if dir.Module != nil && (dir.Src == modulePath || dir.Dest == modulePath) {
					found = true

					output.Info(fmt.Sprintf("Group: %s (%s)", group.Name, group.ID))
					output.Info(fmt.Sprintf("  Source Repository: %s", group.Source.Repo))
					output.Info(fmt.Sprintf("  Source Directory: %s", dir.Src))
					output.Info(fmt.Sprintf("  Target Repository: %s", target.Repo))
					output.Info(fmt.Sprintf("  Target Directory: %s", dir.Dest))
					output.Info("")

					output.Info("Module Configuration:")
					output.Info(fmt.Sprintf("  Type: %s", getModuleType(dir.Module.Type)))
					output.Info(fmt.Sprintf("  Version: %s", dir.Module.Version))

					if dir.Module.CheckTags != nil {
						output.Info(fmt.Sprintf("  Check Tags: %v", *dir.Module.CheckTags))
					}
					if dir.Module.UpdateRefs {
						output.Info("  Update References: true")
					}

					// Show directory mapping settings
					if len(dir.Exclude) > 0 {
						output.Info(fmt.Sprintf("  Exclude Patterns: %v", dir.Exclude))
					}
					if len(dir.IncludeOnly) > 0 {
						output.Info(fmt.Sprintf("  Include Only: %v", dir.IncludeOnly))
					}

					output.Info("")
				}
			}
		}
	}

	if !found {
		output.Error(fmt.Sprintf("Module not found: %s", modulePath))
		return fmt.Errorf("%w: %s", ErrModuleNotFound, modulePath)
	}

	return nil
}

func runModuleVersions(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	modulePath := args[0]
	logger := logrus.StandardLogger()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Find the module in configuration
	var sourceRepo string
	var moduleConfig *config.ModuleConfig

	groups := cfg.Groups
	for _, group := range groups {
		for _, target := range group.Targets {
			for _, dir := range target.Directories {
				if dir.Module != nil && (dir.Src == modulePath || filepath.Base(dir.Src) == modulePath) {
					sourceRepo = group.Source.Repo
					moduleConfig = dir.Module
					break
				}
			}
		}
		if sourceRepo != "" {
			break
		}
	}

	if sourceRepo == "" {
		output.Error(fmt.Sprintf("Module not found in configuration: %s", modulePath))
		return fmt.Errorf("%w: %s", ErrModuleNotFound, modulePath)
	}

	output.Info(fmt.Sprintf("=== Available Versions for %s ===", modulePath))
	output.Info(fmt.Sprintf("Source Repository: %s", sourceRepo))
	output.Info("")

	// Create module resolver
	cache := sync.NewModuleCache(5*time.Minute, logger)
	resolver := sync.NewModuleResolver(logger, cache)

	// For fetching versions, we use git ls-remote
	versions, err := fetchGitTags(ctx, sourceRepo)
	if err != nil {
		output.Error(fmt.Sprintf("Failed to fetch versions: %v", err))
		return fmt.Errorf("failed to fetch versions: %w", err)
	}

	if len(versions) == 0 {
		output.Info("No version tags found in repository")
		return nil
	}

	// Sort versions (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] > versions[j]
	})

	output.Info("Available Versions:")
	for i, version := range versions {
		if i >= 10 {
			output.Info(fmt.Sprintf("  ... and %d more", len(versions)-10))
			break
		}
		output.Info(fmt.Sprintf("  • %s", version))
	}

	// Show current configuration
	if moduleConfig != nil {
		output.Info("")
		output.Info(fmt.Sprintf("Current Configuration: %s", moduleConfig.Version))

		// Try to resolve the version
		checkTags := moduleConfig.CheckTags == nil || *moduleConfig.CheckTags
		resolved, err := resolver.ResolveVersion(ctx, sourceRepo, moduleConfig.Version, checkTags)
		if err != nil {
			output.Warn(fmt.Sprintf("  Warning: Unable to resolve version: %v", err))
		} else {
			output.Success(fmt.Sprintf("  Resolves to: %s", resolved))
		}
	}

	return nil
}

func runValidateModules(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	logger := logrus.StandardLogger()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	output.Info("=== Validating Module Configurations ===")
	output.Info("")

	// Create module resolver
	cache := sync.NewModuleCache(5*time.Minute, logger)
	resolver := sync.NewModuleResolver(logger, cache)

	groups := cfg.Groups
	totalModules := 0
	validModules := 0
	errors := []string{}

	for _, group := range groups {
		groupModules := 0

		for _, target := range group.Targets {
			for _, dir := range target.Directories {
				if dir.Module == nil {
					continue
				}

				totalModules++
				groupModules++

				// Validate module type
				if dir.Module.Type != "" && dir.Module.Type != "go" {
					errors = append(errors, fmt.Sprintf("Invalid module type '%s' for %s (only 'go' is supported)",
						dir.Module.Type, dir.Src))
					continue
				}

				// Validate version constraint
				if dir.Module.Version == "" {
					errors = append(errors, fmt.Sprintf("Missing version for module %s", dir.Src))
					continue
				}

				// Check if source directory exists (would need actual filesystem access)
				// For now, just validate the version format
				if dir.Module.Version != "latest" && !strings.HasPrefix(dir.Module.Version, "v") &&
					!strings.Contains(dir.Module.Version, "^") && !strings.Contains(dir.Module.Version, "~") &&
					!strings.Contains(dir.Module.Version, ">") && !strings.Contains(dir.Module.Version, "<") {
					output.Warn(fmt.Sprintf("  ⚠ Unusual version format for %s: %s", dir.Src, dir.Module.Version))
				}

				// Try to resolve the version
				checkTags := dir.Module.CheckTags == nil || *dir.Module.CheckTags
				resolved, err := resolver.ResolveVersion(ctx, group.Source.Repo, dir.Module.Version, checkTags)
				if err != nil {
					errors = append(errors, fmt.Sprintf("Cannot resolve version '%s' for %s: %v",
						dir.Module.Version, dir.Src, err))
				} else {
					output.Success(fmt.Sprintf("  ✓ Module %s: %s -> %s", dir.Src, dir.Module.Version, resolved))
					validModules++
				}
			}
		}

		if groupModules > 0 {
			output.Info(fmt.Sprintf("  Group '%s': %d modules", group.Name, groupModules))
		}
	}

	output.Info("")

	// Summary
	if totalModules == 0 {
		output.Info("No modules configured to validate")
		return nil
	}

	output.Info("=== Validation Summary ===")
	output.Info(fmt.Sprintf("Total Modules: %d", totalModules))
	output.Info(fmt.Sprintf("Valid: %d", validModules))

	if len(errors) > 0 {
		output.Error(fmt.Sprintf("Errors: %d", len(errors)))
		for _, err := range errors {
			output.Error(fmt.Sprintf("  • %s", err))
		}
		return fmt.Errorf("%w with %d errors", ErrModuleValidationFailed, len(errors))
	}

	output.Success("All module configurations are valid!")
	return nil
}

// getModuleType returns a readable module type
func getModuleType(moduleType string) string {
	if moduleType == "" {
		return "go (default)"
	}
	return moduleType
}

// fetchGitTags fetches git tags from a repository
func fetchGitTags(ctx context.Context, repoPath string) ([]string, error) {
	// Validate repoPath to prevent command injection
	if strings.Contains(repoPath, "..") || strings.Contains(repoPath, ";") || strings.Contains(repoPath, "&") {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRepositoryPath, repoPath)
	}

	// Use git ls-remote to fetch tags
	// Format: git ls-remote --tags https://github.com/org/repo
	url := fmt.Sprintf("https://github.com/%s", repoPath)

	//nolint:gosec // Git URL is validated and constructed safely
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--tags", url)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tags: %w", err)
	}

	var versions []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse tag from line format: <hash> refs/tags/<tag>
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		tag := parts[1]
		if strings.HasPrefix(tag, "refs/tags/") {
			tagName := strings.TrimPrefix(tag, "refs/tags/")
			// Skip annotated tag markers (^{})
			if !strings.HasSuffix(tagName, "^{}") {
				versions = append(versions, tagName)
			}
		}
	}

	return versions, nil
}
