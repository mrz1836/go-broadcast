// Package cli implements the command-line interface for go-broadcast.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// loggerContextKey is a type for context keys to avoid collisions
type loggerContextKey struct{}

// Static errors for command implementations
var (
	ErrStatusNotImplemented = fmt.Errorf("status command not yet implemented for isolated flags")
)

//nolint:gochecknoglobals // Cobra flags are designed to be global variables
var (
	showVersionMu sync.RWMutex
	showVersion   bool
)

// getShowVersion returns the showVersion flag (thread-safe)
func getShowVersion() bool {
	showVersionMu.RLock()
	defer showVersionMu.RUnlock()
	return showVersion
}

// setShowVersion sets the showVersion flag (thread-safe, for testing)
//
//nolint:unused // Available for test cleanup and reset
func setShowVersion(v bool) {
	showVersionMu.Lock()
	defer showVersionMu.Unlock()
	showVersion = v
}

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var rootCmd = &cobra.Command{
	Use:   "go-broadcast",
	Short: "Synchronize files from source repos to multiple targets",
	Long: `go-broadcast is a stateless File Sync Orchestrator that synchronizes
files from a template repository to multiple target repositories.

Key Features:
• Stateless architecture - derives all state from GitHub (branches, PRs, commits)
• File transformations - variable substitution and Go module path updates
• Comprehensive logging - verbose flags and component-specific debugging
• Dry-run support - preview changes before applying them
• Multi-target sync - synchronize to multiple repositories simultaneously
• Automatic PR creation - creates pull requests for review and merging

Common Use Cases:
• Sync CI/CD workflows across microservices
• Maintain consistent documentation standards
• Update configuration files across multiple repositories
• Distribute security policies and compliance files`,
	PersistentPreRunE: setupLogging,
	RunE:              rootRunE,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&globalFlags.ConfigFile, "config", "c", "sync.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVar(&globalFlags.DryRun, "dry-run", false, "Preview changes without making them")
	rootCmd.PersistentFlags().StringVar(&globalFlags.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&showVersion, "version", false, "Show version information")

	// New verbose flags are not added to global command to avoid conflicts
	// They will be added to individual commands that use LogConfig

	// Initialize command flags
	initStatus()
	initCancel()

	// Add commands
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(diagnoseCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(reviewPRCmd)
	rootCmd.AddCommand(modulesCmd)
	rootCmd.AddCommand(newUpgradeCmd())
}

// NewRootCmd creates a new isolated root command instance for testing
// This prevents race conditions by avoiding shared global state
func NewRootCmd() *cobra.Command {
	// Create isolated flags for this command instance
	flags := &Flags{
		ConfigFile: "sync.yaml",
		LogLevel:   "info",
	}

	// Create new command instance with isolated setup function
	cmd := &cobra.Command{
		Use:   "go-broadcast",
		Short: "Synchronize files from source repos to multiple targets",
		Long: `go-broadcast is a stateless File Sync Orchestrator that synchronizes
files from a template repository to multiple target repositories.

It derives all state from GitHub (branches, PRs, commits) and never stores
state locally. It supports file transformations and provides progress tracking.`,
		PersistentPreRunE: createSetupLogging(flags),
		RunE:              createRootRunE(),
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	// Add isolated flags
	cmd.PersistentFlags().StringVarP(&flags.ConfigFile, "config", "c", "sync.yaml", "Path to configuration file")
	cmd.PersistentFlags().BoolVar(&flags.DryRun, "dry-run", false, "Preview changes without making them")
	cmd.PersistentFlags().StringVar(&flags.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	cmd.PersistentFlags().BoolVar(&showVersion, "version", false, "Show version information")

	// Add commands with isolated flags
	cmd.AddCommand(createSyncCmd(flags))
	cmd.AddCommand(createStatusCmd(flags))
	cmd.AddCommand(createValidateCmd(flags))
	cmd.AddCommand(createDiagnoseCmd(flags))
	cmd.AddCommand(createCancelCmd(flags))
	cmd.AddCommand(createReviewPRCmd(flags))
	cmd.AddCommand(newUpgradeCmd())

	return cmd
}

// NewRootCmdWithVerbose creates a new root command with verbose flag support
// This provides verbose logging capabilities from Phase 1
func NewRootCmdWithVerbose() *cobra.Command {
	// Create LogConfig for verbose logging support
	logConfig := &LogConfig{
		ConfigFile: "sync.yaml",
		LogLevel:   "info",
		LogFormat:  "text",
	}

	// Create new command instance with logging setup function
	cmd := &cobra.Command{
		Use:   "go-broadcast",
		Short: "Synchronize files from source repos to multiple targets",
		Long: `go-broadcast is a stateless File Sync Orchestrator that synchronizes
files from a template repository to multiple target repositories.

It derives all state from GitHub (branches, PRs, commits) and never stores
state locally. It supports file transformations and provides progress tracking.`,
		PersistentPreRunE: createSetupLoggingWithVerbose(logConfig),
		RunE:              createRootRunEWithVerbose(logConfig),
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	// Add verbose flags with debug support
	addVerboseFlags(cmd, logConfig)

	// Add commands with LogConfig
	cmd.AddCommand(createSyncCmdWithVerbose(logConfig))
	cmd.AddCommand(createStatusCmdWithVerbose(logConfig))
	cmd.AddCommand(createValidateCmdWithVerbose(logConfig))
	cmd.AddCommand(createDiagnoseCmdWithVerbose(logConfig))
	cmd.AddCommand(createCancelCmdWithVerbose(logConfig))
	// review-pr uses the regular command (no verbose-specific features yet)
	cmd.AddCommand(createReviewPRCmd(&Flags{
		ConfigFile: logConfig.ConfigFile,
		DryRun:     logConfig.DryRun,
		LogLevel:   logConfig.LogLevel,
	}))
	cmd.AddCommand(newUpgradeCmd())

	return cmd
}

// GetRootCmd returns the root command for testing purposes
func GetRootCmd() *cobra.Command {
	// For test isolation, return a new isolated instance
	return NewRootCmd()
}

// Execute runs the CLI
func Execute() {
	if err := ExecuteWithContext(context.Background()); err != nil {
		output.Error(err.Error())
		os.Exit(1)
	}
}

// ExecuteWithContext runs the CLI with the provided context
// This function is more testable as it returns errors instead of calling os.Exit
func ExecuteWithContext(ctx context.Context) error {
	// Set up context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan) // Clean up signal handler when done

	// Start goroutine that exits cleanly when context is canceled or signal received
	go func() {
		select {
		case <-sigChan:
			output.Warn("Interrupt received, canceling...")
			cancel()
		case <-ctx.Done():
			// Context was canceled, exit cleanly without leaking goroutine
			return
		}
	}()

	// Execute command with context
	return rootCmd.ExecuteContext(ctx)
}

// ErrNilFlags is returned when nil flags are provided to logging setup
var ErrNilFlags = errors.New("nil flags provided")

// createSetupLogging creates an isolated logging setup function for the given flags
// It returns a configured logger instance that can be used instead of the global logger
func createSetupLogging(flags *Flags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		// Version flag is handled in RunE functions, not here

		// Guard against nil flags
		if flags == nil {
			return ErrNilFlags
		}

		// Parse log level
		level, err := logrus.ParseLevel(strings.ToLower(flags.LogLevel))
		if err != nil {
			return fmt.Errorf("invalid log level %q: %w", flags.LogLevel, err)
		}

		// Create isolated logger instance
		logger := logrus.New()
		logger.SetLevel(level)
		logger.SetFormatter(&logrus.TextFormatter{
			DisableColors:    false,
			FullTimestamp:    true,
			TimestampFormat:  "15:04:05",
			PadLevelText:     true,
			QuoteEmptyFields: true,
		})

		// Log to stderr to keep stdout clean for output
		logger.SetOutput(os.Stderr)

		// Store logger in command context for isolated access
		cmd.SetContext(context.WithValue(cmd.Context(), loggerContextKey{}, logger))

		return nil
	}
}

// rootRunE handles the root command execution when no subcommand is provided
func rootRunE(cmd *cobra.Command, _ []string) error {
	// If version flag is set, print version (thread-safe read)
	if getShowVersion() {
		return printVersion(false)
	}

	// Otherwise show help
	return cmd.Help()
}

// createRootRunE creates an isolated root run function for testing
func createRootRunE() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		// If version flag is set, print version (thread-safe read)
		if getShowVersion() {
			return printVersion(false)
		}

		// Otherwise show help
		return cmd.Help()
	}
}

// createRootRunEWithVerbose creates a root run function with verbose logging support
func createRootRunEWithVerbose(config *LogConfig) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		// If version flag is set, print version (with JSON support, thread-safe read)
		if getShowVersion() {
			return printVersion(config.JSONOutput)
		}

		// Otherwise show help
		return cmd.Help()
	}
}

// setupLogging configures the logger based on the log level flag (global version)
func setupLogging(_ *cobra.Command, _ []string) error {
	// Version flag is handled in rootRunE, so we don't check it here

	// Parse log level
	level, err := logrus.ParseLevel(strings.ToLower(globalFlags.LogLevel))
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", globalFlags.LogLevel, err)
	}

	// Configure logrus
	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors:    false,
		FullTimestamp:    true,
		TimestampFormat:  "15:04:05",
		PadLevelText:     true,
		QuoteEmptyFields: true,
	})

	// Log to stderr to keep stdout clean for output
	logrus.SetOutput(os.Stderr)

	logrus.WithFields(logrus.Fields{
		"config":    globalFlags.ConfigFile,
		"dry_run":   globalFlags.DryRun,
		"log_level": globalFlags.LogLevel,
	}).Debug("CLI initialized")

	return nil
}

// addVerboseFlags adds verbose and debug flags to the given command.
//
// This function adds the following flags:
// - Verbose flag with counter support (-v, -vv, -vvv)
// - Component-specific debug flags (--debug-git, --debug-api, etc.)
// - Log format flag for output format selection
//
// Parameters:
// - cmd: Cobra command to add flags to
// - config: LogConfig to bind flag values to
func addVerboseFlags(cmd *cobra.Command, config *LogConfig) {
	// Add verbose flag with counter support
	cmd.PersistentFlags().CountVarP(&config.Verbose, "verbose", "v",
		`Increase output verbosity for debugging and monitoring:
  -v    Debug level (detailed operations, timing, status)
  -vv   Trace level (internal operations, API calls)
  -vvv  Trace with caller info (file:line for deep debugging)`)

	// Add component-specific debug flags
	cmd.PersistentFlags().BoolVar(&config.Debug.Git, "debug-git", false,
		"Debug git operations: commands, output, timing, authentication issues")
	cmd.PersistentFlags().BoolVar(&config.Debug.API, "debug-api", false,
		"Debug GitHub API: requests, responses, rate limits, timing")
	cmd.PersistentFlags().BoolVar(&config.Debug.Transform, "debug-transform", false,
		"Debug file transformations: variable substitutions, size changes, content")
	cmd.PersistentFlags().BoolVar(&config.Debug.Config, "debug-config", false,
		"Debug configuration: validation steps, field checks, error details")
	cmd.PersistentFlags().BoolVar(&config.Debug.State, "debug-state", false,
		"Debug state discovery: repository analysis, branch detection, PR status")

	// Add log format flag
	cmd.PersistentFlags().StringVar(&config.LogFormat, "log-format", "text",
		`Output format: "text" (human-readable, colored) or "json" (structured, machine-readable)`)

	// Add JSON output flag (legacy alias for --log-format=json)
	cmd.PersistentFlags().BoolVar(&config.JSONOutput, "json", false,
		"Enable structured JSON output for log aggregation and automation")

	// Add standard flags
	cmd.PersistentFlags().StringVarP(&config.ConfigFile, "config", "c", "sync.yaml",
		"Path to configuration file")
	cmd.PersistentFlags().BoolVar(&config.DryRun, "dry-run", false,
		"Preview changes without making them")
	cmd.PersistentFlags().StringVar(&config.LogLevel, "log-level", "info",
		"Log level (debug, info, warn, error) - overridden by verbose flags")
	cmd.PersistentFlags().BoolVar(&showVersion, "version", false, "Show version information")
}

// createSetupLoggingWithVerbose creates a verbose logging setup function.
//
// This function creates a setup function that uses the LoggerService to configure
// logging with support for verbose flags, trace level, and component-specific debugging.
//
// Parameters:
// - config: LogConfig containing logging and debug configuration
//
// Returns:
// - Function that can be used as PersistentPreRunE for Cobra commands
func createSetupLoggingWithVerbose(config *LogConfig) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		// Version flag is handled in RunE functions, not here

		// Guard against nil config
		if config == nil {
			return ErrNilConfig
		}

		ctx := cmd.Context()

		// Generate correlation ID for this execution
		if config.CorrelationID == "" {
			config.CorrelationID = logging.GenerateCorrelationID()
		}

		// Handle JSON output flag
		if config.JSONOutput {
			config.LogFormat = "json"
		}

		// Create logger service with the configuration
		loggerService := NewLoggerService(config)

		// Configure logrus with verbose logging capabilities
		if err := loggerService.ConfigureLogger(ctx); err != nil {
			return fmt.Errorf("failed to configure logger: %w", err)
		}

		// Configure structured output if JSON format is requested
		if err := logging.ConfigureLogger(logrus.StandardLogger(), config); err != nil {
			return fmt.Errorf("failed to configure structured logging: %w", err)
		}

		// Log initialization details at debug level with correlation ID
		logrus.WithFields(logrus.Fields{
			logging.StandardFields.CorrelationID: config.CorrelationID,
			logging.StandardFields.Component:     logging.ComponentNames.CLI,
			"config":                             config.ConfigFile,
			"dry_run":                            config.DryRun,
			"log_level":                          config.LogLevel,
			"verbose":                            config.Verbose,
			"log_format":                         config.LogFormat,
			"json_output":                        config.JSONOutput,
			"debug_git":                          config.Debug.Git,
			"debug_api":                          config.Debug.API,
		}).Debug("CLI initialized with verbose logging")

		return nil
	}
}

// createSyncCmdWithVerbose creates a sync command with verbose logging support.
//
// This function creates a sync command that uses LogConfig for verbose logging
// capabilities including verbose flags and component-specific debug settings.
//
// Parameters:
// - config: LogConfig containing logging and debug configuration
//
// Returns:
// - Cobra command configured for sync operations with verbose support
func createSyncCmdWithVerbose(config *LogConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "sync [targets...]",
		Short: "Synchronize files to target repositories",
		Long: `Synchronize files from the source template repository to one or more target repositories.

If no targets are specified, all targets in the configuration file will be synchronized.
Target repositories can be specified as arguments to sync only specific repos.`,
		Example: `  # Basic operations
  go-broadcast sync                        # Sync all targets
  go-broadcast sync org/repo1 org/repo2    # Sync specific repositories
  go-broadcast sync --dry-run              # Preview changes only

  # Debugging and monitoring
  go-broadcast sync -v                     # Debug level output
  go-broadcast sync -vv                    # Trace level with details
  go-broadcast sync -vvv                   # Trace with caller info

  # Component-specific debugging
  go-broadcast sync --debug-git            # Show git commands and output
  go-broadcast sync --debug-api            # Show GitHub API requests
  go-broadcast sync --debug-transform      # Show file transformations
  go-broadcast sync --debug-config         # Show configuration validation
  go-broadcast sync --debug-state          # Show state discovery process

  # Structured logging for automation
  go-broadcast sync --json -v              # JSON output for log analysis
  go-broadcast sync --log-format json      # Alternative JSON syntax

  # Comprehensive debugging sessions
  go-broadcast sync -vv --debug-git --debug-api    # Multiple components
  go-broadcast sync -vvv --debug-git 2> debug.log # Save to file

  # Performance monitoring
  go-broadcast sync --json 2>&1 | jq 'select(.duration_ms > 1000)'

  # Troubleshooting authentication
  go-broadcast sync --debug-git -v         # Debug git auth issues

  # Production monitoring with structured logs
  go-broadcast sync --json 2>&1 | fluentd`,
		Aliases: []string{"s"},
		RunE:    createRunSyncWithVerbose(config),
	}
}

// createStatusCmdWithVerbose creates a status command with verbose logging support.
//
// Parameters:
// - config: LogConfig containing logging and debug configuration
//
// Returns:
// - Cobra command configured for status operations with verbose support
func createStatusCmdWithVerbose(config *LogConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show synchronization status of target repositories",
		Long: `Display the current synchronization status of all target repositories.

Shows comprehensive status information including:
- Last successful sync commit SHA and timestamp
- Available updates from source repository
- Active pull request status and details
- Repository sync health and any issues
- Time since last sync and sync frequency`,
		Example: `  # Basic status check
  go-broadcast status                    # Show status for all targets
  go-broadcast status --config sync.yaml # Use specific config file

  # Detailed status with debugging
  go-broadcast status -v                 # Debug level status details
  go-broadcast status --debug-state -v   # Debug state discovery process

  # Automation and monitoring
  go-broadcast status --json             # Machine-readable status output
  go-broadcast status --json | jq .      # Pretty-print JSON status

  # Common status workflows
  go-broadcast status 2>&1 | tee status.log     # Save status to file
  go-broadcast status && go-broadcast sync       # Check status then sync`,
		Aliases: []string{"st"},
		RunE:    createRunStatusWithVerbose(config),
	}
}

// createValidateCmdWithVerbose creates a validate command with verbose logging support.
//
// Parameters:
// - config: LogConfig containing logging and debug configuration
//
// Returns:
// - Cobra command configured for validation operations with verbose support
func createValidateCmdWithVerbose(config *LogConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Validate the configuration file for syntax and semantic errors.

Performs comprehensive validation including:
- YAML syntax verification
- Required field presence
- Repository name format validation
- File path safety checks
- Transform configuration validation
- Duplicate detection across targets and file mappings`,
		Example: `  # Basic validation
  go-broadcast validate                    # Validate default config file
  go-broadcast validate --config sync.yaml # Validate specific file

  # Detailed validation debugging
  go-broadcast validate --debug-config -v  # Show validation steps
  go-broadcast validate -vv               # Trace level validation details

  # Automation and CI/CD
  go-broadcast validate && echo "Config OK" # Exit code validation
  go-broadcast validate --json            # Machine-readable output

  # Common validation workflows
  go-broadcast validate --config prod.yaml --debug-config  # Debug production config
  go-broadcast validate 2>&1 | tee validation.log         # Save validation output`,
		Aliases: []string{"v", "check"},
		RunE:    createRunValidateWithVerbose(config),
	}
}

// createSyncCmd creates an isolated sync command with the given flags
func createSyncCmd(flags *Flags) *cobra.Command {
	return &cobra.Command{
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

  # For troubleshooting, use verbose commands:
  # go-broadcast sync -v --debug-git (requires go-broadcast with verbose support)`,
		Aliases: []string{"s"},
		RunE:    createRunSync(flags),
	}
}

// createStatusCmd creates an isolated status command with the given flags
func createStatusCmd(flags *Flags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of target repositories",
		Long: `Display the current status of all target repositories, including:
- Last sync commit
- Available updates
- Pull request status`,
		Aliases: []string{"st"},
		RunE:    createRunStatus(flags),
	}
}

// createValidateCmd creates an isolated validate command with the given flags
func createValidateCmd(flags *Flags) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Validate the configuration file for syntax and semantic errors.

Performs comprehensive validation including:
- YAML syntax verification
- Required field presence
- Repository name format validation
- File path safety checks
- Transform configuration validation
- Duplicate detection across targets and file mappings`,
		Example: `  # Basic validation
  go-broadcast validate                    # Validate default config file
  go-broadcast validate --config sync.yaml # Validate specific file

  # Debug validation issues
  go-broadcast validate --log-level debug  # Show detailed validation steps

  # Automation workflows
  go-broadcast validate && echo "Config valid"      # Use exit code
  go-broadcast validate 2>&1 | tee validation.log  # Save output

  # For detailed debugging, use verbose validate command:
  # go-broadcast validate --debug-config -v (requires verbose support)`,
		Aliases: []string{"v", "check"},
		RunE:    createRunValidate(flags),
	}
}

// createRunStatus creates an isolated status run function with the given flags
func createRunStatus(_ *Flags) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		// For now, use a placeholder implementation - this will be implemented properly
		// when we address the status command's specific needs
		return ErrStatusNotImplemented
	}
}

// createRunValidate creates an isolated validate run function with the given flags
func createRunValidate(flags *Flags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		// Get isolated logger from context, fallback to global if not available
		logger, ok := ctx.Value(loggerContextKey{}).(*logrus.Logger)
		if !ok {
			logger = logrus.StandardLogger()
		}

		cfg, err := loadConfigWithFlags(flags, logger)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if err := cfg.ValidateWithLogging(context.Background(), nil); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		output.Success("Configuration is valid")
		return nil
	}
}

// createRunSyncWithVerbose creates a sync run function with verbose logging support.
//
// This function creates a run function that uses LogConfig for verbose sync operations
// with component-specific debugging and verbose logging capabilities.
//
// Parameters:
// - config: LogConfig containing logging and debug configuration
//
// Returns:
// - Function that can be used as RunE for Cobra sync commands
func createRunSyncWithVerbose(config *LogConfig) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		log := logrus.WithField("command", "sync")

		// For Phase 1, delegate to existing sync implementation
		// This maintains functionality while adding verbose flag support
		// In later phases, this will include component-specific debugging

		// Load configuration using LogConfig
		cfg, err := loadConfigWithLogConfig(config)
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
		if config.DryRun {
			output.Warn("DRY-RUN MODE: No changes will be made to repositories")
		}

		// Initialize sync engine with LogConfig
		engine, err := createSyncEngineWithLogConfig(ctx, cfg, config)
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

// createRunStatusWithVerbose creates a status run function with verbose logging support.
//
// Parameters:
// - config: LogConfig containing logging and debug configuration
//
// Returns:
// - Function that can be used as RunE for Cobra status commands
func createRunStatusWithVerbose(_ *LogConfig) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		// For Phase 1, maintain existing behavior
		return ErrStatusNotImplemented
	}
}

// createRunValidateWithVerbose creates a validate run function with verbose logging support.
//
// Parameters:
// - config: LogConfig containing logging and debug configuration
//
// Returns:
// - Function that can be used as RunE for Cobra validate commands
func createRunValidateWithVerbose(config *LogConfig) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		cfg, err := loadConfigWithLogConfig(config)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if err := cfg.ValidateWithLogging(context.Background(), config); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		output.Success("Configuration is valid")
		return nil
	}
}

// createCancelCmd creates an isolated cancel command with the given flags
func createCancelCmd(flags *Flags) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel [targets...]",
		Short: "Cancel active sync operations",
		Long: `Cancel active sync operations by closing open pull requests and optionally deleting sync branches.

This command finds all open sync pull requests for the specified targets (or all targets if none specified)
and closes them with a descriptive comment. By default, it also deletes the associated sync branches
to clean up the repositories.`,
		Example: `  # Cancel all active syncs
  go-broadcast cancel --config sync.yaml

  # Cancel syncs for specific repositories
  go-broadcast cancel org/repo1 org/repo2

  # Preview what would be canceled (dry run)
  go-broadcast cancel --dry-run --config sync.yaml

  # Close PRs but keep sync branches
  go-broadcast cancel --keep-branches --config sync.yaml`,
		Aliases: []string{"c"},
		RunE:    createRunCancel(flags),
	}
}

// createCancelCmdWithVerbose creates a cancel command with verbose logging support
func createCancelCmdWithVerbose(config *LogConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel [targets...]",
		Short: "Cancel active sync operations",
		Long: `Cancel active sync operations by closing open pull requests and optionally deleting sync branches.

This command finds all open sync pull requests for the specified targets (or all targets if none specified)
and closes them with a descriptive comment. By default, it also deletes the associated sync branches
to clean up the repositories.`,
		Example: `  # Cancel all active syncs
  go-broadcast cancel --config sync.yaml

  # Cancel syncs for specific repositories
  go-broadcast cancel org/repo1 org/repo2

  # Preview what would be canceled (dry run)
  go-broadcast cancel --dry-run --config sync.yaml

  # Debug cancel operations
  go-broadcast cancel -v --debug-api --config sync.yaml`,
		Aliases: []string{"c"},
		RunE:    createRunCancelWithVerbose(config),
	}
}

// createRunCancel creates an isolated cancel run function with the given flags
func createRunCancel(_ *Flags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Delegate to the global cancel implementation
		// This maintains functionality while providing isolated flag support
		return runCancel(cmd, args)
	}
}

// createRunCancelWithVerbose creates a cancel run function with verbose logging support
func createRunCancelWithVerbose(_ *LogConfig) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// For Phase 1, delegate to existing cancel implementation
		return runCancel(cmd, args)
	}
}
