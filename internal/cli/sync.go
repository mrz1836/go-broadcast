package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/sync"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
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

	// Initialize sync engine with real implementations
	engine, err := createSyncEngine(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize sync engine: %w", err)
	}

	// Execute sync
	if err := engine.Sync(ctx, targets); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	output.Success("Sync completed successfully")
	return nil
}

// createRunSync creates an isolated sync run function with the given flags
func createRunSync(flags *Flags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Get isolated logger from context, fallback to global if not available
		logger, ok := ctx.Value(loggerContextKey{}).(*logrus.Logger)
		if !ok {
			logger = logrus.StandardLogger()
		}
		log := logger.WithField("command", "sync")

		// Load configuration
		cfg, err := loadConfigWithFlags(flags, logger)
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
		if flags.DryRun {
			output.Warn("DRY-RUN MODE: No changes will be made to repositories")
		}

		// Initialize sync engine with real implementations
		engine, err := createSyncEngineWithFlags(ctx, cfg, flags, logger)
		if err != nil {
			return fmt.Errorf("failed to initialize sync engine: %w", err)
		}

		// Execute sync
		if err := engine.Sync(ctx, targets); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		output.Success("Sync completed successfully")
		return nil
	}
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

	return cfg, nil
}

// loadConfigWithFlags loads configuration using the given flags instead of global state
func loadConfigWithFlags(flags *Flags, logger *logrus.Logger) (*config.Config, error) {
	configPath := flags.ConfigFile

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

	logger.WithFields(logrus.Fields{
		"source":      cfg.Source.Repo,
		"targets":     len(cfg.Targets),
		"config_file": configPath,
	}).Debug("Configuration loaded")

	return cfg, nil
}

// createSyncEngine initializes the sync engine with all required dependencies
func createSyncEngine(ctx context.Context, cfg *config.Config) (*sync.Engine, error) {
	logger := logrus.StandardLogger()

	// Initialize GitHub client
	ghClient, err := gh.NewClient(ctx, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Initialize Git client
	gitClient, err := git.NewClient(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git client: %w", err)
	}

	// Initialize state discoverer
	stateDiscoverer := state.NewDiscoverer(ghClient, logger)

	// Initialize transform chain
	transformChain := transform.NewChain(logger)

	// Add repository name transformer if any target uses it
	for _, target := range cfg.Targets {
		if target.Transform.RepoName {
			transformChain.Add(transform.NewRepoTransformer())
			break
		}
	}

	// Add template variable transformer if any target uses it
	for _, target := range cfg.Targets {
		if len(target.Transform.Variables) > 0 {
			transformChain.Add(transform.NewTemplateTransformer(logrus.StandardLogger()))
			break
		}
	}

	// Create sync options
	opts := sync.DefaultOptions().
		WithDryRun(IsDryRun()).
		WithMaxConcurrency(5)

	// Create and return engine
	engine := sync.NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
	engine.SetLogger(logrus.StandardLogger())

	return engine, nil
}

// createSyncEngineWithFlags initializes the sync engine with flags instead of global state
func createSyncEngineWithFlags(ctx context.Context, cfg *config.Config, flags *Flags, logger *logrus.Logger) (*sync.Engine, error) {
	// Initialize GitHub client
	ghClient, err := gh.NewClient(ctx, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Initialize Git client
	gitClient, err := git.NewClient(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git client: %w", err)
	}

	// Initialize state discoverer
	stateDiscoverer := state.NewDiscoverer(ghClient, logger)

	// Initialize transform chain
	transformChain := transform.NewChain(logger)

	// Add repository name transformer if any target uses it
	for _, target := range cfg.Targets {
		if target.Transform.RepoName {
			transformChain.Add(transform.NewRepoTransformer())
			break
		}
	}

	// Add template variable transformer if any target uses it
	for _, target := range cfg.Targets {
		if len(target.Transform.Variables) > 0 {
			transformChain.Add(transform.NewTemplateTransformer(logger))
			break
		}
	}

	// Create sync options using flags instead of global state
	opts := sync.DefaultOptions().
		WithDryRun(flags.DryRun).
		WithMaxConcurrency(5)

	// Create and return engine
	engine := sync.NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
	engine.SetLogger(logger)

	return engine, nil
}
