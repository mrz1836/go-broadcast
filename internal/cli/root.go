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

// setupLogging configures the logger based on the log level flag
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
