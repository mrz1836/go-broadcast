package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	gosync "sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
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

// Rate-limit preflight flag names (used for both registration and override
// detection via cobra's Changed()).
const (
	flagRateLimitPreflight        = "rate-limit-preflight"
	flagIgnoreRateLimitPreflight  = "ignore-rate-limit-preflight"
	flagRateLimitMarginPercent    = "rate-limit-margin-percent"
	flagRateLimitSecondaryReserve = "rate-limit-secondary-reserve"
	flagRateLimitFailClosed       = "rate-limit-fail-closed"
)

//nolint:gochecknoglobals // Package-level variables for CLI flags
var (
	syncFlagsMu      gosync.RWMutex // Protects sync flag variables for thread-safety
	groupFilter      []string
	skipGroups       []string
	automerge        bool
	clearModuleCache bool

	// Rate-limit preflight flags. Defaults mirror the documented config defaults
	// so that, absent any --config rate_limit_preflight block, the gate behaves
	// per AC-7. CLI values override config only when the flag is explicitly set
	// (see currentRateLimitOverrides / Changed()).
	rateLimitPreflight        = true
	ignoreRateLimitPreflight  bool
	rateLimitMarginPercent    = config.DefaultRateLimitPrimaryMarginPercent
	rateLimitSecondaryReserve = config.DefaultRateLimitSecondaryReserve
	rateLimitFailClosed       bool

	// syncFlagSet is the sync command's flag set, captured in init() so that
	// override detection (Changed()) does not statically reference the syncCmd
	// variable — that would form a package initialization cycle
	// (syncCmd -> runSync -> createSyncEngine -> override detection -> syncCmd).
	syncFlagSet *pflag.FlagSet
)

// getGroupFilter returns a copy of the group filter slice (thread-safe)
func getGroupFilter() []string {
	syncFlagsMu.RLock()
	defer syncFlagsMu.RUnlock()
	return append([]string(nil), groupFilter...)
}

// getSkipGroups returns a copy of the skip groups slice (thread-safe)
func getSkipGroups() []string {
	syncFlagsMu.RLock()
	defer syncFlagsMu.RUnlock()
	return append([]string(nil), skipGroups...)
}

// getAutomerge returns the automerge flag (thread-safe)
func getAutomerge() bool {
	syncFlagsMu.RLock()
	defer syncFlagsMu.RUnlock()
	return automerge
}

// getClearModuleCache returns the clear module cache flag (thread-safe)
func getClearModuleCache() bool {
	syncFlagsMu.RLock()
	defer syncFlagsMu.RUnlock()
	return clearModuleCache
}

// rateLimitPreflightOverrides captures the CLI override intent for the
// rate-limit preflight. A nil pointer field means "not overridden — use the
// config default"; a non-nil field overrides config. The ignore escape hatch is
// CLI-only (no config equivalent) so it is always carried.
type rateLimitPreflightOverrides struct {
	enabled    *bool
	ignore     bool
	margin     *int
	reserve    *int
	failClosed *bool
}

// mergeRateLimitPreflight applies the resolved rate-limit preflight settings to
// opts. Config (via config.ResolveRateLimitPreflight) provides the base values;
// any explicitly-set CLI override replaces the corresponding base value. This is
// the single source of truth used by every engine builder, so the
// config-feeds-Options / flags-override-config behavior is consistent.
func mergeRateLimitPreflight(opts *sync.Options, cfg *config.Config, ov rateLimitPreflightOverrides) *sync.Options {
	enabled, margin, reserve, failClosed := config.ResolveRateLimitPreflight(cfg)

	if ov.enabled != nil {
		enabled = *ov.enabled
	}
	if ov.margin != nil {
		margin = *ov.margin
	}
	if ov.reserve != nil {
		reserve = *ov.reserve
	}
	if ov.failClosed != nil {
		failClosed = *ov.failClosed
	}

	return opts.
		WithRateLimitPreflight(enabled).
		WithRateLimitMargins(margin, reserve).
		WithRateLimitFailClosed(failClosed).
		WithIgnoreRateLimitPreflight(ov.ignore)
}

// currentRateLimitOverrides reads the sync command's CLI flags and returns the
// overrides the user explicitly set. Cobra's Changed() distinguishes an
// explicitly-set flag from its default, so unset flags fall through to config.
func currentRateLimitOverrides() rateLimitPreflightOverrides {
	syncFlagsMu.RLock()
	defer syncFlagsMu.RUnlock()

	ov := rateLimitPreflightOverrides{ignore: ignoreRateLimitPreflight}

	flags := syncFlagSet
	if flags == nil {
		// Flags not registered (e.g. an alternate command path); use config base.
		return ov
	}

	if flags.Changed(flagRateLimitPreflight) {
		v := rateLimitPreflight
		ov.enabled = &v
	}
	if flags.Changed(flagRateLimitMarginPercent) {
		v := rateLimitMarginPercent
		ov.margin = &v
	}
	if flags.Changed(flagRateLimitSecondaryReserve) {
		v := rateLimitSecondaryReserve
		ov.reserve = &v
	}
	if flags.Changed(flagRateLimitFailClosed) {
		v := rateLimitFailClosed
		ov.failClosed = &v
	}

	return ov
}

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var syncCmd = &cobra.Command{
	Use:   "sync [targets...]",
	Short: "Synchronize files to target repositories",
	Long: `Synchronize files from the source template repository to one or more target repositories.

If no targets are specified, all targets in the configuration file will be synchronized.
Target repositories can be specified as arguments to sync only specific repos.

Configuration Source:
  By default, configuration is loaded from the YAML file (--config).
  Use --from-db to load configuration from the database instead.

Group Filtering:
  Use --groups to sync only specific groups (by name or ID).
  Use --skip-groups to exclude specific groups from sync.
  When both are specified, skip-groups takes precedence.`,
	Example: `  # Basic operations
  go-broadcast sync                        # Sync all targets from config
  go-broadcast sync --config sync.yaml     # Use specific config file
  go-broadcast sync org/repo1 org/repo2    # Sync only specified repositories
  go-broadcast sync --dry-run              # Preview changes without making them

  # Database-backed configuration
  go-broadcast sync --from-db              # Load configuration from database
  go-broadcast sync --from-db --groups "core"  # Sync specific groups from database

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
	syncCmd.Flags().BoolVar(&clearModuleCache, "clear-cache", false, "Clear module version cache before sync")

	// Rate-limit preflight flags (override the config rate_limit_preflight block).
	syncCmd.Flags().BoolVar(&rateLimitPreflight, flagRateLimitPreflight, true, "Enable the pre-sync GitHub rate-limit preflight gate")
	syncCmd.Flags().BoolVar(&ignoreRateLimitPreflight, flagIgnoreRateLimitPreflight, false, "Force the sync through even if the rate-limit preflight would halt")
	syncCmd.Flags().IntVar(&rateLimitMarginPercent, flagRateLimitMarginPercent, config.DefaultRateLimitPrimaryMarginPercent, "Percent of the primary rate-limit budget to keep as headroom")
	syncCmd.Flags().IntVar(&rateLimitSecondaryReserve, flagRateLimitSecondaryReserve, config.DefaultRateLimitSecondaryReserve, "Number of the 80/min secondary content-write slots to keep in reserve")
	syncCmd.Flags().BoolVar(&rateLimitFailClosed, flagRateLimitFailClosed, false, "Halt the sync when the rate-limit probe is unavailable (default fails open)")

	// Capture the flag set for override detection without statically referencing
	// syncCmd from the engine-builder call graph (avoids an init cycle).
	syncFlagSet = syncCmd.Flags()
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

	// Log group filters if specified (using thread-safe getters)
	gf := getGroupFilter()
	sg := getSkipGroups()
	if len(gf) > 0 {
		log.WithField("groups", gf).Info("Filtering to specific groups")
	}
	if len(sg) > 0 {
		log.WithField("skip_groups", sg).Info("Skipping specified groups")
	}

	// Filter targets if specified
	targets := args
	if len(targets) > 0 {
		log.WithField("targets", targets).Info("Syncing specific targets")
	} else if len(gf) == 0 && len(sg) == 0 {
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

	// Attach sync metrics recorder if database is available
	closeMetrics := tryAttachMetricsRecorder(engine, logrus.StandardLogger())
	defer closeMetrics()

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

		// Attach sync metrics recorder if database is available
		closeMetrics := tryAttachMetricsRecorder(engine, logger)
		defer closeMetrics()

		// Execute sync
		if err := engine.Sync(ctx, targets); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		output.Success("Sync completed successfully")
		return nil
	}
}

func loadConfig() (*config.Config, error) {
	// Warn if both flags are specified
	if GetFromDB() && GetConfigFile() != "sync.yaml" {
		logrus.Warn("Both --from-db and --config specified; using --from-db (--config ignored)")
	}

	// Check if loading from database
	if GetFromDB() {
		return loadConfigFromDB()
	}

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

// loadConfigFromDB loads configuration from the database
func loadConfigFromDB() (*config.Config, error) {
	ctx := context.Background()

	// Open database using shared helper (handles existence check + auto-migration)
	database, err := openDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	// Create converter
	converter := db.NewConverter(database.DB())

	// Get first config (default behavior)
	var dbConfig db.Config
	if dbErr := database.DB().First(&dbConfig).Error; dbErr != nil {
		return nil, fmt.Errorf("no configuration found in database: %w", dbErr)
	}

	// Export configuration
	cfg, err := converter.ExportConfig(ctx, dbConfig.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("failed to export configuration from database: %w", err)
	}

	// DB-loaded configs must run through ApplyDefaultsAndResolve to reach parity with YAML — see internal/config/parser.go.
	if err := config.ApplyDefaultsAndResolve(cfg); err != nil {
		return nil, fmt.Errorf("failed to apply defaults and resolve list references: %w", err)
	}

	// Validate configuration
	if err := cfg.ValidateWithLogging(context.Background(), nil); err != nil {
		return nil, fmt.Errorf("invalid configuration from database: %w", err)
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

// tryAttachMetricsRecorder attempts to open the database and attach a sync metrics
// recorder to the engine. If the database does not exist or cannot be opened, it
// logs and returns a no-op closer so sync continues without metrics.
// The returned function must be called (typically via defer) when the sync completes.
func tryAttachMetricsRecorder(engine *sync.Engine, log *logrus.Logger) func() {
	path := getDBPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Debug("Database not found; sync metrics will not be recorded")
		return func() {}
	}

	database, err := db.Open(db.OpenOptions{
		Path:        path,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	if err != nil {
		log.WithError(err).Warn("Failed to open database; sync metrics will not be recorded")
		return func() {}
	}

	syncRepo := db.NewBroadcastSyncRepo(database.DB())
	repoRepo := db.NewRepoRepository(database.DB())
	targetRepo := db.NewTargetRepository(database.DB())
	groupRepo := db.NewGroupRepository(database.DB())
	adapter := sync.NewDBMetricsAdapter(syncRepo, repoRepo, targetRepo, groupRepo)
	engine.SetSyncMetricsRecorder(adapter)

	return func() { _ = database.Close() }
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

	// Load automerge labels from environment if automerge is enabled (thread-safe)
	var automergeLabels []string
	autoMergeEnabled := getAutomerge()
	if autoMergeEnabled {
		if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
			// Split comma-separated labels and trim whitespace
			for _, label := range strings.Split(envLabels, ",") {
				if trimmed := strings.TrimSpace(label); trimmed != "" {
					automergeLabels = append(automergeLabels, trimmed)
				}
			}
		}
	}

	// Create sync options (using thread-safe getters)
	opts := sync.DefaultOptions().
		WithDryRun(IsDryRun()).
		WithMaxConcurrency(5).
		WithGroupFilter(getGroupFilter()).
		WithSkipGroups(getSkipGroups()).
		WithAutomerge(autoMergeEnabled).
		WithAutomergeLabels(automergeLabels).
		WithClearModuleCache(getClearModuleCache())

	// Apply rate-limit preflight settings (config base + CLI overrides)
	opts = mergeRateLimitPreflight(opts, cfg, currentRateLimitOverrides())

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

	// Apply rate-limit preflight settings (config base + CLI overrides)
	opts = mergeRateLimitPreflight(opts, cfg, currentRateLimitOverrides())

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

	// Apply rate-limit preflight settings (config base + CLI overrides)
	opts = mergeRateLimitPreflight(opts, cfg, currentRateLimitOverrides())

	// Create and return engine
	engine := sync.NewEngine(ctx, cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
	engine.SetLogger(logger)

	return engine, nil
}
