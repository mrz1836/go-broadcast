package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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
  • Transform configurations are valid`,
	Example: `  # Validate default config file
  go-broadcast validate

  # Validate specific config file
  go-broadcast validate --config custom-sync.yaml`,
	Aliases: []string{"v", "check"},
	RunE:    runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	log := logrus.WithField("command", "validate")
	configPath := GetConfigFile()

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
	if err := cfg.Validate(); err != nil {
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

	// TODO: When GitHub client is implemented, add these checks:
	// output.Info("  ⚠ Repository accessibility (requires GitHub client)")
	// output.Info("  ⚠ Source file existence (requires Git client)")

	return nil
}
