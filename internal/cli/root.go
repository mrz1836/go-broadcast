// Package cli implements the command-line interface for go-broadcast.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// loggerContextKey is a type for context keys to avoid collisions
type loggerContextKey struct{}

// Static errors for command implementations
var (
	ErrStatusNotImplemented = fmt.Errorf("status command not yet implemented for isolated flags")
)

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var rootCmd = &cobra.Command{
	Use:   "go-broadcast",
	Short: "Synchronize files from template repos to multiple targets",
	Long: `go-broadcast is a stateless File Sync Orchestrator that synchronizes
files from a template repository to multiple target repositories.

It derives all state from GitHub (branches, PRs, commits) and never stores
state locally. It supports file transformations and provides progress tracking.`,
	PersistentPreRunE: setupLogging,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&globalFlags.ConfigFile, "config", "c", "sync.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVar(&globalFlags.DryRun, "dry-run", false, "Preview changes without making them")
	rootCmd.PersistentFlags().StringVar(&globalFlags.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	// Initialize command flags
	initStatus()
	initVersion()

	// Add commands
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)
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
		Short: "Synchronize files from template repos to multiple targets",
		Long: `go-broadcast is a stateless File Sync Orchestrator that synchronizes
files from a template repository to multiple target repositories.

It derives all state from GitHub (branches, PRs, commits) and never stores
state locally. It supports file transformations and provides progress tracking.`,
		PersistentPreRunE: createSetupLogging(flags),
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	// Add isolated flags
	cmd.PersistentFlags().StringVarP(&flags.ConfigFile, "config", "c", "sync.yaml", "Path to configuration file")
	cmd.PersistentFlags().BoolVar(&flags.DryRun, "dry-run", false, "Preview changes without making them")
	cmd.PersistentFlags().StringVar(&flags.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	// Add commands with isolated flags
	cmd.AddCommand(createSyncCmd(flags))
	cmd.AddCommand(createStatusCmd(flags))
	cmd.AddCommand(createValidateCmd(flags))
	cmd.AddCommand(createVersionCmd(flags))

	return cmd
}

// GetRootCmd returns the root command for testing purposes
func GetRootCmd() *cobra.Command {
	// For backward compatibility and test isolation, return a new isolated instance
	return NewRootCmd()
}

// Execute runs the CLI
func Execute() {
	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		output.Warn("Interrupt received, canceling...")
		cancel()
	}()

	// Execute command with context
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		output.Error(err.Error())
		os.Exit(1)
	}
}

// createSetupLogging creates an isolated logging setup function for the given flags
// It returns a configured logger instance that can be used instead of the global logger
func createSetupLogging(flags *Flags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
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

// setupLogging configures the logger based on the log level flag (global version)
func setupLogging(_ *cobra.Command, _ []string) error {
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

// createSyncCmd creates an isolated sync command with the given flags
func createSyncCmd(flags *Flags) *cobra.Command {
	return &cobra.Command{
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
		Use:     "validate",
		Short:   "Validate configuration file",
		Long:    `Validate the configuration file for syntax and semantic errors.`,
		Aliases: []string{"v", "check"},
		RunE:    createRunValidate(flags),
	}
}

// createVersionCmd creates an isolated version command with the given flags
func createVersionCmd(flags *Flags) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print detailed version information including build details.`,
		RunE:  createRunVersion(flags),
	}
}

// createRunStatus creates an isolated status run function with the given flags
func createRunStatus(_ *Flags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		// Get isolated logger from context, fallback to global if not available
		logger, ok := ctx.Value(loggerContextKey{}).(*logrus.Logger)
		if !ok {
			logger = logrus.StandardLogger()
		}
		_ = logger.WithField("command", "status")

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
		_ = logger.WithField("command", "validate")

		cfg, err := loadConfigWithFlags(flags, logger)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		output.Success("Configuration is valid")
		return nil
	}
}

// createRunVersion creates an isolated version run function with the given flags
func createRunVersion(_ *Flags) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		// Get isolated logger from context, fallback to global if not available
		logger, ok := ctx.Value(loggerContextKey{}).(*logrus.Logger)
		if !ok {
			logger = logrus.StandardLogger()
		}
		_ = logger.WithField("command", "version")

		// Use the same implementation as the global version command
		return runVersion(cmd, []string{})
	}
}
