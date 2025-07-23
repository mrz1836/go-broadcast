package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync [targets...]",
	Short: "Synchronize files to target repositories",
	Long: `Synchronize files from the source template repository to one or more target repositories.

If no targets are specified, all targets in the configuration file will be synchronized.
Target repositories can be specified as arguments to sync only specific repos.`,
	Example: `  # Sync all targets from config file
  go-broadcast sync --config sync.yaml

  # Sync specific targets only
  go-broadcast sync org/repo1 org/repo2

  # Preview changes without making them
  go-broadcast sync --dry-run

  # Sync with debug logging
  go-broadcast sync --log-level debug`,
	Aliases: []string{"s"},
	RunE:    runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	log := logrus.WithField("command", "sync")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Filter targets if specified
	targets := args
	if len(targets) > 0 {
		log.WithField("targets", targets).Info("Syncing specific targets")
	} else {
		log.Info("Syncing all configured targets")
	}

	// Show dry-run warning
	if IsDryRun() {
		output.Warn("DRY-RUN MODE: No changes will be made to repositories")
	}

	// TODO: Initialize sync engine with real implementations
	// For now, we'll just show what would be done
	output.Info("Sync operation would execute here...")

	// Simulate sync process
	if err := simulateSync(ctx, cfg, targets); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	output.Success("Sync completed successfully")
	return nil
}

func loadConfig() (*config.Config, error) {
	configPath := GetConfigFile()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrConfigFileNotFound, configPath)
	}

	// Load and parse configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"source":      cfg.Source.Repo,
		"targets":     len(cfg.Targets),
		"config_file": configPath,
	}).Debug("Configuration loaded")

	return cfg, nil
}

func simulateSync(ctx context.Context, cfg *config.Config, targetFilter []string) error {
	// Filter targets based on command line arguments
	targets := cfg.Targets

	if len(targetFilter) > 0 {
		filtered := []config.TargetConfig{}

		for _, target := range cfg.Targets {
			for _, filter := range targetFilter {
				if target.Repo == filter {
					filtered = append(filtered, target)
					break
				}
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("%w: %v", ErrNoMatchingTargets, targetFilter)
		}

		targets = filtered
	}

	// Show what would be synced
	output.Info(fmt.Sprintf("Source repository: %s (branch: %s)", cfg.Source.Repo, cfg.Source.Branch))
	output.Info(fmt.Sprintf("Syncing to %d target(s):", len(targets)))

	for _, target := range targets {
		output.Info(fmt.Sprintf("  • %s (%d file mappings)", target.Repo, len(target.Files)))

		for _, file := range target.Files {
			output.Info(fmt.Sprintf("    - %s → %s", file.Src, file.Dest))
		}
		if target.Transform.RepoName || len(target.Transform.Variables) > 0 {
			output.Info("    Transforms:")

			if target.Transform.RepoName {
				output.Info("      - Repository name replacement")
			}
			if len(target.Transform.Variables) > 0 {
				output.Info(fmt.Sprintf("      - Variables: %v", target.Transform.Variables))
			}
		}
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return nil
}
