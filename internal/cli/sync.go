package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/sync"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// SyncService defines the interface for sync operations
type SyncService interface {
	Sync(ctx context.Context, targets []string) error
}

// ConfigLoader defines the interface for configuration loading
type ConfigLoader interface {
	LoadConfig(configPath string) (*config.Config, error)
	ValidateConfig(cfg *config.Config) error
}

// SyncEngineFactory defines the interface for creating sync engines
type SyncEngineFactory interface {
	CreateSyncEngine(ctx context.Context, cfg *config.Config, flags *Flags, logger *logrus.Logger) (SyncService, error)
}

// DefaultConfigLoader implements ConfigLoader
type DefaultConfigLoader struct{}

// LoadConfig loads and parses configuration from file
func (d *DefaultConfigLoader) LoadConfig(configPath string) (*config.Config, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrConfigFileNotFound, configPath)
	}

	// Load and parse configuration
	return config.Load(configPath)
}

// ValidateConfig validates the configuration
func (d *DefaultConfigLoader) ValidateConfig(cfg *config.Config) error {
	return cfg.ValidateWithLogging(context.Background(), nil)
}

// DefaultSyncEngineFactory implements SyncEngineFactory
type DefaultSyncEngineFactory struct{}

// CreateSyncEngine creates a new sync engine with all dependencies
func (d *DefaultSyncEngineFactory) CreateSyncEngine(ctx context.Context, cfg *config.Config, flags *Flags, logger *logrus.Logger) (SyncService, error) {
	return createSyncEngineWithFlags(ctx, cfg, flags, logger)
}

// SyncCommand represents a testable sync command
type SyncCommand struct {
	configLoader      ConfigLoader
	syncEngineFactory SyncEngineFactory
	outputWriter      output.Writer
}

// NewSyncCommand creates a new SyncCommand with default dependencies
func NewSyncCommand() *SyncCommand {
	return &SyncCommand{
		configLoader:      &DefaultConfigLoader{},
		syncEngineFactory: &DefaultSyncEngineFactory{},
		outputWriter:      output.NewColoredWriter(os.Stdout, os.Stderr),
	}
}

// NewSyncCommandWithDependencies creates a new SyncCommand with injectable dependencies
func NewSyncCommandWithDependencies(configLoader ConfigLoader, syncEngineFactory SyncEngineFactory, outputWriter output.Writer) *SyncCommand {
	return &SyncCommand{
		configLoader:      configLoader,
		syncEngineFactory: syncEngineFactory,
		outputWriter:      outputWriter,
	}
}

// ExecuteSync runs the sync operation with the given flags and arguments
func (s *SyncCommand) ExecuteSync(ctx context.Context, flags *Flags, args []string) error {
	logger := logrus.StandardLogger()
	log := logger.WithField("command", "sync")

	// Load configuration
	cfg, err := s.configLoader.LoadConfig(flags.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if validateErr := s.configLoader.ValidateConfig(cfg); validateErr != nil {
		return fmt.Errorf("invalid configuration: %w", validateErr)
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
		s.outputWriter.Warn("DRY-RUN MODE: No changes will be made to repositories")
	}

	// Initialize sync engine
	syncEngine, err := s.syncEngineFactory.CreateSyncEngine(ctx, cfg, flags, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize sync engine: %w", err)
	}

	// Execute sync
	if err := syncEngine.Sync(ctx, targets); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	s.outputWriter.Success("Sync completed successfully")
	return nil
}

//nolint:gochecknoglobals // Package-level variables for CLI flags
var (
	groupFilter []string
	skipGroups  []string
	automerge   bool
)

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var syncCmd = &cobra.Command{
	Use:   "sync [targets...]",
	Short: "Synchronize files to target repositories",
	Long: `Synchronize files from the source template repository to one or more target repositories.

If no targets are specified, all targets in the configuration file will be synchronized.
Target repositories can be specified as arguments to sync only specific repos.

Group Filtering:
  Use --groups to sync only specific groups (by name or ID).
  Use --skip-groups to exclude specific groups from sync.
  When both are specified, skip-groups takes precedence.`,
	Example: `  # Basic operations
  go-broadcast sync                        # Sync all targets from config
  go-broadcast sync --config sync.yaml     # Use specific config file
  go-broadcast sync org/repo1 org/repo2    # Sync only specified repositories
  go-broadcast sync --dry-run              # Preview changes without making them

  # Group-based sync
  go-broadcast sync --groups "core,security"       # Sync only core and security groups
  go-broadcast sync --skip-groups "experimental"   # Sync all except experimental group
  go-broadcast sync --groups core org/repo1        # Sync specific target in core group

  # Debugging and troubleshooting
  go-broadcast sync --log-level debug      # Enable debug logging
  go-broadcast sync --log-level trace      # Maximum verbosity

  # Automerge configuration
  go-broadcast sync --automerge                         # Add automerge labels to PRs
  go-broadcast sync --automerge --groups "core"        # Automerge with group filtering

  # Common workflows
  go-broadcast validate && go-broadcast sync --dry-run  # Validate then preview
  go-broadcast sync --dry-run | tee preview.log        # Save preview output

  # For debugging capabilities, use:
  # go-broadcast sync -v --debug-git --debug-api (requires verbose support)`,
	Aliases: []string{"s"},
	RunE:    runSync,
}

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	syncCmd.Flags().StringSliceVar(&groupFilter, "groups", nil, "Sync only specified groups (by name or ID)")
	syncCmd.Flags().StringSliceVar(&skipGroups, "skip-groups", nil, "Skip specified groups during sync")
	syncCmd.Flags().BoolVar(&automerge, "automerge", false, "Add automerge labels from GO_BROADCAST_AUTOMERGE_LABELS to created PRs")
}

func runSync(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	log := logrus.WithField("command", "sync")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		// Display configuration error to user before returning
		if strings.Contains(err.Error(), "invalid configuration") {
			output.Error(fmt.Sprintf("Configuration validation failed: %v", err))
		} else {
			output.Error(fmt.Sprintf("Failed to load configuration: %v", err))
		}
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Log group filters if specified
	if len(groupFilter) > 0 {
		log.WithField("groups", groupFilter).Info("Filtering to specific groups")
	}
	if len(skipGroups) > 0 {
		log.WithField("skip_groups", skipGroups).Info("Skipping specified groups")
	}

	// Filter targets if specified
	targets := args
	if len(targets) > 0 {
		log.WithField("targets", targets).Info("Syncing specific targets")
	} else if len(groupFilter) == 0 && len(skipGroups) == 0 {
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

	groups := cfg.Groups
	sourceRepo := ""
	targetsCount := 0
	if len(groups) > 0 {
		sourceRepo = groups[0].Source.Repo
		targetsCount = len(groups[0].Targets)
	}
	logger.WithFields(logrus.Fields{
		"source":      sourceRepo,
		"targets":     targetsCount,
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

	// Add email transformer FIRST if any source or target has email configuration
	// This must run before repo name transformer to prevent email addresses from being corrupted
	groups := cfg.Groups
	for _, group := range groups {
		// Check if source has email configuration
		if group.Source.SecurityEmail != "" || group.Source.SupportEmail != "" {
			transformChain.Add(transform.NewEmailTransformer())
			goto emailTransformerAdded
		}
		// Check if any target has email configuration
		for _, target := range group.Targets {
			if target.SecurityEmail != "" || target.SupportEmail != "" {
				transformChain.Add(transform.NewEmailTransformer())
				goto emailTransformerAdded
			}
		}
	}
emailTransformerAdded:

	// Add template variable transformer if any target uses it
	for _, group := range groups {
		for _, target := range group.Targets {
			if len(target.Transform.Variables) > 0 {
				transformChain.Add(transform.NewTemplateTransformer(logrus.StandardLogger(), nil))
				goto templateTransformerAdded
			}
		}
	}
templateTransformerAdded:

	// Add repository name transformer LAST if any target uses it
	// This runs last to avoid corrupting email addresses during transformation
	for _, group := range groups {
		for _, target := range group.Targets {
			if target.Transform.RepoName {
				transformChain.Add(transform.NewRepoTransformer())
				goto repoTransformerAdded
			}
		}
	}
repoTransformerAdded:

	// Load automerge labels from environment if automerge is enabled
	var automergeLabels []string
	if automerge {
		if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
			// Split comma-separated labels and trim whitespace
			for _, label := range strings.Split(envLabels, ",") {
				if trimmed := strings.TrimSpace(label); trimmed != "" {
					automergeLabels = append(automergeLabels, trimmed)
				}
			}
		}
	}

	// Create sync options
	opts := sync.DefaultOptions().
		WithDryRun(IsDryRun()).
		WithMaxConcurrency(5).
		WithGroupFilter(groupFilter).
		WithSkipGroups(skipGroups).
		WithAutomerge(automerge).
		WithAutomergeLabels(automergeLabels)

	// Create and return engine
	engine := sync.NewEngine(ctx, cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
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

	// Add email transformer FIRST if any source or target has email configuration
	// This must run before repo name transformer to prevent email addresses from being corrupted
	groups := cfg.Groups
	for _, group := range groups {
		// Check if source has email configuration
		if group.Source.SecurityEmail != "" || group.Source.SupportEmail != "" {
			transformChain.Add(transform.NewEmailTransformer())
			goto emailTransformerAdded2
		}
		// Check if any target has email configuration
		for _, target := range group.Targets {
			if target.SecurityEmail != "" || target.SupportEmail != "" {
				transformChain.Add(transform.NewEmailTransformer())
				goto emailTransformerAdded2
			}
		}
	}
emailTransformerAdded2:

	// Add template variable transformer if any target uses it
	for _, group := range groups {
		for _, target := range group.Targets {
			if len(target.Transform.Variables) > 0 {
				transformChain.Add(transform.NewTemplateTransformer(logger, nil))
				goto templateTransformerAdded2
			}
		}
	}
templateTransformerAdded2:

	// Add repository name transformer LAST if any target uses it
	// This runs last to avoid corrupting email addresses during transformation
	for _, group := range groups {
		for _, target := range group.Targets {
			if target.Transform.RepoName {
				transformChain.Add(transform.NewRepoTransformer())
				goto repoTransformerAdded2
			}
		}
	}
repoTransformerAdded2:

	// Load automerge labels from environment if automerge is enabled
	var automergeLabels []string
	if flags.Automerge {
		if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
			// Split comma-separated labels and trim whitespace
			for _, label := range strings.Split(envLabels, ",") {
				if trimmed := strings.TrimSpace(label); trimmed != "" {
					automergeLabels = append(automergeLabels, trimmed)
				}
			}
		}
	}

	// Create sync options using flags instead of global state
	opts := sync.DefaultOptions().
		WithDryRun(flags.DryRun).
		WithMaxConcurrency(5).
		WithGroupFilter(flags.GroupFilter).
		WithSkipGroups(flags.SkipGroups).
		WithAutomerge(flags.Automerge).
		WithAutomergeLabels(automergeLabels)

	// Create and return engine
	engine := sync.NewEngine(ctx, cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
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
		groups := cfg.Groups
		sourceRepo := ""
		targetsCount := 0
		if len(groups) > 0 {
			sourceRepo = groups[0].Source.Repo
			targetsCount = len(groups[0].Targets)
		}
		logrus.WithFields(logrus.Fields{
			"source":      sourceRepo,
			"targets":     targetsCount,
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

	// Add email transformer FIRST if any source or target has email configuration
	// This must run before repo name transformer to prevent email addresses from being corrupted
	groups := cfg.Groups
	for _, group := range groups {
		// Check if source has email configuration
		if group.Source.SecurityEmail != "" || group.Source.SupportEmail != "" {
			transformChain.Add(transform.NewEmailTransformer())
			goto emailTransformerAdded3
		}
		// Check if any target has email configuration
		for _, target := range group.Targets {
			if target.SecurityEmail != "" || target.SupportEmail != "" {
				transformChain.Add(transform.NewEmailTransformer())
				goto emailTransformerAdded3
			}
		}
	}
emailTransformerAdded3:

	// Add template variable transformer if any target uses it
	for _, group := range groups {
		for _, target := range group.Targets {
			if len(target.Transform.Variables) > 0 {
				transformChain.Add(transform.NewTemplateTransformer(logger, logConfig))
				goto templateTransformerAdded3
			}
		}
	}
templateTransformerAdded3:

	// Add repository name transformer LAST if any target uses it
	// This runs last to avoid corrupting email addresses during transformation
	for _, group := range groups {
		for _, target := range group.Targets {
			if target.Transform.RepoName {
				transformChain.Add(transform.NewRepoTransformer())
				goto repoTransformerAdded3
			}
		}
	}
repoTransformerAdded3:

	// Load automerge labels from environment if automerge is enabled
	var automergeLabels []string
	if logConfig.Automerge {
		if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
			// Split comma-separated labels and trim whitespace
			for _, label := range strings.Split(envLabels, ",") {
				if trimmed := strings.TrimSpace(label); trimmed != "" {
					automergeLabels = append(automergeLabels, trimmed)
				}
			}
		}
	}

	// Create sync options using LogConfig instead of global state
	opts := sync.DefaultOptions().
		WithDryRun(logConfig.DryRun).
		WithMaxConcurrency(5).
		WithGroupFilter(logConfig.GroupFilter).
		WithSkipGroups(logConfig.SkipGroups).
		WithAutomerge(logConfig.Automerge).
		WithAutomergeLabels(automergeLabels)

	// Create and return engine
	engine := sync.NewEngine(ctx, cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
	engine.SetLogger(logger)

	return engine, nil
}
