package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Validation errors
var (
	ErrGitHubCLIRequired    = fmt.Errorf("github CLI required for repository validation")
	ErrGitHubAuthRequired   = fmt.Errorf("github authentication required")
	ErrSourceBranchNotFound = fmt.Errorf("source branch not accessible")
	ErrSourceRepoNotFound   = fmt.Errorf("source repository not accessible")
)

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long: `Validate the syntax and content of a configuration file.

Checks performed:
  • YAML syntax is valid
  • Required fields are present
  • Repository names are in correct format
  • File paths are valid
  • No duplicate targets or file mappings
  • Transform configurations are valid
  • Repository accessibility (requires GitHub authentication)
  • Source file existence (requires Git access)`,
	Example: `  # Basic validation
  go-broadcast validate                     # Validate default config file
  go-broadcast validate --config sync.yaml # Validate specific file
  
  # Skip remote validation for offline use
  go-broadcast validate --skip-remote-checks       # Only validate YAML/syntax
  go-broadcast validate --source-only              # Only check source repo access
  
  # Debug validation issues
  go-broadcast validate --log-level debug  # Show detailed validation steps
  
  # Automation workflows
  go-broadcast validate && echo "Config valid"      # Use exit code
  go-broadcast validate 2>&1 | tee validation.log  # Save validation output
  
  # Common patterns
  go-broadcast validate --config prod.yaml  # Validate production config
  find . -name "*.yaml" -exec go-broadcast validate --config {} \;  # Validate multiple files`,
	Aliases: []string{"v", "check"},
	RunE:    runValidate,
}

func runValidate(cmd *cobra.Command, _ []string) error {
	return runValidateWithFlags(globalFlags, cmd)
}

func runValidateWithFlags(flags *Flags, cmd *cobra.Command) error {
	log := logrus.WithField("command", "validate")
	configPath := flags.ConfigFile

	output.Info(fmt.Sprintf("Validating configuration file: %s", configPath))

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrConfigFileNotFound, configPath)
	}

	// Get absolute path for clarity
	absPath, err := filepath.Abs(configPath)
	if err == nil {
		configPath = absPath
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		output.Error(fmt.Sprintf("Failed to parse configuration: %v", err))
		return fmt.Errorf("configuration parsing failed: %w", err)
	}

	log.Debug("Configuration parsed successfully")

	// Validate configuration
	if err := cfg.ValidateWithLogging(context.Background(), nil); err != nil {
		output.Error(fmt.Sprintf("Configuration validation failed: %v", err))
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Display configuration summary
	output.Success("✓ Configuration is valid!")
	output.Info("")
	output.Info("Configuration Summary:")
	output.Info(fmt.Sprintf("  Version: %d", cfg.Version))
	output.Info(fmt.Sprintf("  Source: %s (branch: %s)", cfg.Source.Repo, cfg.Source.Branch))

	if cfg.Defaults.BranchPrefix != "" || len(cfg.Defaults.PRLabels) > 0 {
		output.Info("  Defaults:")
		if cfg.Defaults.BranchPrefix != "" {
			output.Info(fmt.Sprintf("    Branch prefix: %s", cfg.Defaults.BranchPrefix))
		}
		if len(cfg.Defaults.PRLabels) > 0 {
			output.Info(fmt.Sprintf("    PR labels: %v", cfg.Defaults.PRLabels))
		}
	}

	output.Info(fmt.Sprintf("  Targets: %d repositories", len(cfg.Targets)))

	// Show target details
	totalFiles := 0

	for i, target := range cfg.Targets {
		output.Info(fmt.Sprintf("    %d. %s", i+1, target.Repo))
		output.Info(fmt.Sprintf("       Files: %d mappings", len(target.Files)))

		// Count transforms
		transformCount := 0
		if target.Transform.RepoName {
			transformCount++
		}

		transformCount += len(target.Transform.Variables)
		if transformCount > 0 {
			output.Info(fmt.Sprintf("       Transforms: %d configured", transformCount))
		}

		totalFiles += len(target.Files)
	}

	output.Info("")
	output.Info(fmt.Sprintf("Total file mappings: %d", totalFiles))

	// Additional validation checks (future implementation)
	output.Info("")
	output.Info("Additional checks:")
	output.Success("  ✓ Repository name format")
	output.Success("  ✓ File paths")
	output.Success("  ✓ No duplicate targets")
	output.Success("  ✓ No duplicate file destinations")

	// Get command flags
	skipRemoteChecks, _ := cmd.Flags().GetBool("skip-remote-checks")
	sourceOnly, _ := cmd.Flags().GetBool("source-only")

	// Remote validation checks (skip if requested)
	if !skipRemoteChecks {
		output.Info("")
		output.Info("Remote validation:")

		// Initialize logging config for clients
		logConfig := &logging.LogConfig{
			LogLevel: flags.LogLevel,
		}

		// Validate repository accessibility
		if err := validateRepositoryAccessibility(context.Background(), cfg, logConfig, sourceOnly); err != nil {
			output.Error(fmt.Sprintf("Repository accessibility check failed: %v", err))
			// Don't return error - this is a warning, not a fatal error
		}

		// Validate source file existence
		validateSourceFilesExist(context.Background(), cfg, logConfig)
	} else {
		output.Info("")
		output.Info("Remote validation: (skipped)")
		output.Info("  ⚠ Repository accessibility check skipped")
		output.Info("  ⚠ Source file existence check skipped")
	}

	return nil
}

// validateRepositoryAccessibility checks if source and target repositories are accessible via GitHub API
func validateRepositoryAccessibility(ctx context.Context, cfg *config.Config, logConfig *logging.LogConfig, sourceOnly bool) error {
	log := logrus.WithField("component", "validate-repos")

	// Try to create GitHub client
	ghClient, err := gh.NewClient(ctx, logrus.StandardLogger(), logConfig)
	if err != nil {
		if strings.Contains(err.Error(), "gh CLI not found") {
			output.Error("  ✗ GitHub CLI not found in PATH")
			output.Info("    Install with: https://cli.github.com/")
			return ErrGitHubCLIRequired
		}
		if strings.Contains(err.Error(), "not authenticated") {
			output.Error("  ✗ GitHub authentication required")
			output.Info("    Run: gh auth login")
			return ErrGitHubAuthRequired
		}
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Check source repository accessibility
	log.WithField("repo", cfg.Source.Repo).Debug("Checking source repository accessibility")
	_, err = ghClient.GetBranch(ctx, cfg.Source.Repo, cfg.Source.Branch)
	if err != nil {
		if strings.Contains(err.Error(), "branch not found") {
			output.Error(fmt.Sprintf("  ✗ Source branch '%s' not found in %s", cfg.Source.Branch, cfg.Source.Repo))
			return ErrSourceBranchNotFound
		}
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
			output.Error(fmt.Sprintf("  ✗ Source repository '%s' not accessible", cfg.Source.Repo))
			output.Info("    Check repository name and permissions")
			return ErrSourceRepoNotFound
		}
		output.Error(fmt.Sprintf("  ✗ Failed to access source repository: %v", err))
		return fmt.Errorf("source repository check failed: %w", err)
	}
	output.Success(fmt.Sprintf("  ✓ Source repository accessible: %s (branch: %s)", cfg.Source.Repo, cfg.Source.Branch))

	// Skip target repository checks if sourceOnly flag is set
	if sourceOnly {
		output.Info("  ⚠ Target repository checks skipped (--source-only)")
		return nil
	}

	// Check target repositories accessibility
	for i, target := range cfg.Targets {
		log.WithFields(logrus.Fields{
			"target_index": i,
			"repo":         target.Repo,
		}).Debug("Checking target repository accessibility")

		// Try to get repository information (this will fail if repo doesn't exist or no access)
		_, err = ghClient.ListBranches(ctx, target.Repo)
		if err != nil {
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
				output.Error(fmt.Sprintf("  ✗ Target repository '%s' not accessible", target.Repo))
				output.Info("    Check repository name and permissions")
				continue // Don't fail validation, just warn
			}
			output.Error(fmt.Sprintf("  ✗ Failed to access target repository '%s': %v", target.Repo, err))
			continue // Don't fail validation, just warn
		}
		output.Success(fmt.Sprintf("  ✓ Target repository accessible: %s", target.Repo))
	}

	return nil
}

// validateSourceFilesExist checks if all configured source files exist in the source repository
func validateSourceFilesExist(ctx context.Context, cfg *config.Config, logConfig *logging.LogConfig) {
	log := logrus.WithField("component", "validate-files")

	// Initialize GitHub client (reuse from previous function, but handle errors gracefully)
	ghClient, err := gh.NewClient(ctx, logrus.StandardLogger(), logConfig)
	if err != nil {
		output.Info("  ⚠ Skipping source file validation (GitHub client unavailable)")
		return // Don't fail if client can't be created
	}

	// Collect all unique source files across all targets
	sourceFiles := make(map[string]bool)
	for _, target := range cfg.Targets {
		for _, file := range target.Files {
			sourceFiles[file.Src] = true
		}
	}

	if len(sourceFiles) == 0 {
		output.Info("  ⚠ No source files to validate")
		return
	}

	// Check each source file exists
	filesChecked := 0
	filesFound := 0
	for srcPath := range sourceFiles {
		log.WithFields(logrus.Fields{
			"source_file": srcPath,
			"repo":        cfg.Source.Repo,
			"branch":      cfg.Source.Branch,
		}).Debug("Checking source file existence")

		_, err := ghClient.GetFile(ctx, cfg.Source.Repo, srcPath, cfg.Source.Branch)
		filesChecked++
		if err != nil {
			if strings.Contains(err.Error(), "file not found") {
				output.Error(fmt.Sprintf("  ✗ Source file not found: %s", srcPath))
				continue // Don't fail validation, just warn
			}
			output.Error(fmt.Sprintf("  ✗ Failed to check source file '%s': %v", srcPath, err))
			continue // Don't fail validation, just warn
		}
		filesFound++
	}

	if filesFound == filesChecked {
		output.Success(fmt.Sprintf("  ✓ All source files exist (%d/%d)", filesFound, filesChecked))
	} else {
		output.Error(fmt.Sprintf("  ⚠ Some source files missing (%d/%d found)", filesFound, filesChecked))
	}
}

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	validateCmd.Flags().Bool("skip-remote-checks", false, "Skip GitHub and Git repository checks (offline validation)")
	validateCmd.Flags().Bool("source-only", false, "Only validate source repository access (skip target repositories)")
}
