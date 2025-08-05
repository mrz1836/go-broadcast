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
	Example: `  # Basic operations
  go-broadcast sync                        # Sync all targets from config
  go-broadcast sync --config sync.yaml     # Use specific config file
  go-broadcast sync org/repo1 org/repo2    # Sync only specified repositories
  go-broadcast sync --dry-run              # Preview changes without making them

  # Debugging and troubleshooting
  go-broadcast sync --log-level debug      # Enable debug logging
  go-broadcast sync --log-level trace      # Maximum verbosity

  # Common workflows
  go-broadcast validate && go-broadcast sync --dry-run  # Validate then preview
  go-broadcast sync --dry-run | tee preview.log        # Save preview output

  # For debugging capabilities, use:
  # go-broadcast sync -v --debug-git --debug-api (requires verbose support)`,
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
	if err := cfg.ValidateWithLogging(context.Background(), nil); err != nil {
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
	if err := cfg.ValidateWithLogging(context.Background(), nil); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Count total mappings and targets for logging
	totalTargets := 0
	for _, mapping := range cfg.Mappings {
		totalTargets += len(mapping.Targets)
	}

	logger.WithFields(logrus.Fields{
		"mappings":    len(cfg.Mappings),
		"targets":     totalTargets,
		"config_file": configPath,
	}).Debug("Configuration loaded")

	return cfg, nil
}

// createSyncEngine initializes the sync engine with all required dependencies
func createSyncEngine(ctx context.Context, cfg *config.Config) (*sync.Engine, error) {
	logger := logrus.StandardLogger()

	// Initialize GitHub client
	ghClient, err := gh.NewClient(ctx, logger, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Initialize Git client
	gitClient, err := git.NewClient(logger, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git client: %w", err)
	}

	// Initialize state discoverer
	stateDiscoverer := state.NewDiscoverer(ghClient, logger, nil)

	// Initialize transform chain
	transformChain := transform.NewChain(logger)

	// Add repository name transformer if any target uses it
	for _, mapping := range cfg.Mappings {
		for _, target := range mapping.Targets {
			if target.Transform.RepoName {
				transformChain.Add(transform.NewRepoTransformer())
				break
			}
		}
	}

	// Add template variable transformer if any target uses it
	for _, mapping := range cfg.Mappings {
		for _, target := range mapping.Targets {
			if len(target.Transform.Variables) > 0 {
				transformChain.Add(transform.NewTemplateTransformer(logrus.StandardLogger(), nil))
				break
			}
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
	ghClient, err := gh.NewClient(ctx, logger, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Initialize Git client
	gitClient, err := git.NewClient(logger, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git client: %w", err)
	}

	// Initialize state discoverer
	stateDiscoverer := state.NewDiscoverer(ghClient, logger, nil)

	// Initialize transform chain
	transformChain := transform.NewChain(logger)

	// Add repository name transformer if any target uses it
	for _, mapping := range cfg.Mappings {
		for _, target := range mapping.Targets {
			if target.Transform.RepoName {
				transformChain.Add(transform.NewRepoTransformer())
				break
			}
		}
	}

	// Add template variable transformer if any target uses it
	for _, mapping := range cfg.Mappings {
		for _, target := range mapping.Targets {
			if len(target.Transform.Variables) > 0 {
				transformChain.Add(transform.NewTemplateTransformer(logger, nil))
				break
			}
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

// loadConfigWithLogConfig loads configuration using LogConfig instead of Flags.
//
// This function provides configuration loading with verbose logging
// support and component-specific debug settings.
//
// Parameters:
// - logConfig: LogConfig containing configuration and debug settings
//
// Returns:
// - Loaded and validated configuration
// - Error if loading or validation fails
//
// Side Effects:
// - Logs configuration details when debug logging is enabled
func loadConfigWithLogConfig(logConfig *LogConfig) (*config.Config, error) {
	configPath := logConfig.ConfigFile

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrConfigFileNotFound, configPath)
	}

	// Load and parse configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	// Validate configuration with LogConfig
	if err := cfg.ValidateWithLogging(context.Background(), logConfig); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Log configuration details when debug is enabled
	if logConfig.Debug.Config || logConfig.Verbose >= 1 {
		// Count total mappings and targets for logging
		totalTargets := 0
		for _, mapping := range cfg.Mappings {
			totalTargets += len(mapping.Targets)
		}

		logrus.WithFields(logrus.Fields{
			"mappings":    len(cfg.Mappings),
			"targets":     totalTargets,
			"config_file": configPath,
		}).Debug("Configuration loaded")
	}

	return cfg, nil
}

// createSyncEngineWithLogConfig initializes the sync engine with LogConfig.
//
// This function creates a sync engine using the LogConfig for
// component-specific debugging and verbose logging capabilities.
//
// Parameters:
// - ctx: Context for cancellation control
// - cfg: Application configuration
// - logConfig: Logging configuration with debug settings
//
// Returns:
// - Configured sync engine instance
// - Error if initialization fails
//
// Side Effects:
// - Creates GitHub and Git clients with appropriate logging
// - Configures transform chain based on configuration
func createSyncEngineWithLogConfig(ctx context.Context, cfg *config.Config, logConfig *LogConfig) (*sync.Engine, error) {
	logger := logrus.StandardLogger()

	// Initialize GitHub client with verbose logging
	ghClient, err := gh.NewClient(ctx, logger, logConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Initialize Git client with verbose logging
	gitClient, err := git.NewClient(logger, logConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git client: %w", err)
	}

	// Initialize state discoverer with LogConfig
	stateDiscoverer := state.NewDiscoverer(ghClient, logger, logConfig)

	// Initialize transform chain
	transformChain := transform.NewChain(logger)

	// Add repository name transformer if any target uses it
	for _, mapping := range cfg.Mappings {
		for _, target := range mapping.Targets {
			if target.Transform.RepoName {
				transformChain.Add(transform.NewRepoTransformer())
				break
			}
		}
	}

	// Add template variable transformer if any target uses it
	for _, mapping := range cfg.Mappings {
		for _, target := range mapping.Targets {
			if len(target.Transform.Variables) > 0 {
				transformChain.Add(transform.NewTemplateTransformer(logger, logConfig))
				break
			}
		}
	}

	// Create sync options using LogConfig instead of global state
	opts := sync.DefaultOptions().
		WithDryRun(logConfig.DryRun).
		WithMaxConcurrency(5)

	// Create and return engine
	engine := sync.NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
	engine.SetLogger(logger)

	return engine, nil
}
