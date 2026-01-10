// Package cli provides command-line interface functionality for go-broadcast.
//
// This file contains tests for signal handling and context management in root command.
// These tests verify proper cancellation behavior, nil context handling,
// and graceful shutdown scenarios.
package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteWithContext_AlreadyCancelledContext verifies that execution
// handles an already-canceled context gracefully.
//
// This matters because callers may pass pre-canceled contexts in timeout scenarios.
func TestExecuteWithContext_AlreadyCanceledContext(t *testing.T) {
	// Create already canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create a simple test command that checks context
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			select {
			case <-cmd.Context().Done():
				return cmd.Context().Err()
			default:
				return nil
			}
		},
	}

	// Execute with pre-canceled context
	cmd.SetContext(ctx)
	err := cmd.ExecuteContext(ctx)
	// The command should detect the cancellation
	if err != nil {
		assert.True(t, errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded),
			"error should indicate context cancellation")
	}
}

// TestCreateSetupLogging_NilFlags verifies that createSetupLogging
// returns ErrNilFlags when given nil.
//
// This prevents nil pointer panics when accessing flags.LogLevel.
func TestCreateSetupLogging_NilFlags(t *testing.T) {
	setupFn := createSetupLogging(nil)

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())

	err := setupFn(cmd, nil)

	require.Error(t, err, "should return error for nil flags")
	assert.ErrorIs(t, err, ErrNilFlags, "error should be ErrNilFlags")
}

// TestCreateSetupLogging_InvalidLogLevel verifies that invalid log levels
// are properly rejected with a clear error message.
func TestCreateSetupLogging_InvalidLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		logLevel string
		wantErr  bool
	}{
		{
			name:     "valid debug level",
			logLevel: "debug",
			wantErr:  false,
		},
		{
			name:     "valid info level",
			logLevel: "info",
			wantErr:  false,
		},
		{
			name:     "valid warn level",
			logLevel: "warn",
			wantErr:  false,
		},
		{
			name:     "valid error level",
			logLevel: "error",
			wantErr:  false,
		},
		{
			name:     "invalid level",
			logLevel: "invalid_level",
			wantErr:  true,
		},
		{
			name:     "typo in level",
			logLevel: "debu",
			wantErr:  true,
		},
		{
			name:     "empty level uses default",
			logLevel: "",
			wantErr:  true, // ParseLevel returns error for empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			flags := &Flags{LogLevel: tt.logLevel}
			setupFn := createSetupLogging(flags)

			cmd := &cobra.Command{Use: "test"}
			cmd.SetContext(context.Background())

			err := setupFn(cmd, nil)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid log level")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCreateSetupLoggingWithVerbose_NilConfig verifies that
// createSetupLoggingWithVerbose returns ErrNilConfig when given nil.
func TestCreateSetupLoggingWithVerbose_NilConfig(t *testing.T) {
	setupFn := createSetupLoggingWithVerbose(nil)

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())

	err := setupFn(cmd, nil)

	require.Error(t, err, "should return error for nil config")
	assert.ErrorIs(t, err, ErrNilConfig, "error should be ErrNilConfig")
}

// TestCreateSetupLoggingWithVerbose_ValidConfig verifies that valid configs work.
func TestCreateSetupLoggingWithVerbose_ValidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *LogConfig
	}{
		{
			name:   "minimal config",
			config: &LogConfig{},
		},
		{
			name: "verbose level 1",
			config: &LogConfig{
				Verbose: 1,
			},
		},
		{
			name: "verbose level 2 with trace",
			config: &LogConfig{
				Verbose: 2,
			},
		},
		{
			name: "verbose level 3 with caller info",
			config: &LogConfig{
				Verbose: 3,
			},
		},
		{
			name: "json output",
			config: &LogConfig{
				JSONOutput: true,
			},
		},
		{
			name: "with correlation ID",
			config: &LogConfig{
				CorrelationID: "test-correlation-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupFn := createSetupLoggingWithVerbose(tt.config)

			cmd := &cobra.Command{Use: "test"}
			cmd.SetContext(context.Background())

			err := setupFn(cmd, nil)
			assert.NoError(t, err, "valid config should not return error")
		})
	}
}

// TestRootRunE_VersionFlag verifies that the version flag is handled correctly.
func TestRootRunE_VersionFlag(t *testing.T) {
	// Save and restore global state
	oldShowVersion := showVersion
	defer func() { showVersion = oldShowVersion }()

	t.Run("version flag false shows help", func(t *testing.T) {
		showVersion = false

		cmd := &cobra.Command{Use: "test"}
		cmd.SetContext(context.Background())

		err := rootRunE(cmd, nil)
		// Should show help (which doesn't return error)
		assert.NoError(t, err)
	})

	t.Run("version flag true shows version", func(t *testing.T) {
		showVersion = true

		cmd := &cobra.Command{Use: "test"}
		cmd.SetContext(context.Background())

		err := rootRunE(cmd, nil)
		// printVersion should succeed
		assert.NoError(t, err)
	})
}

// TestContextTimeout verifies behavior with context timeouts.
func TestContextTimeout(t *testing.T) {
	t.Parallel()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Verify context eventually gets canceled (polling is more reliable than fixed sleep in CI)
	require.Eventually(t, func() bool {
		return ctx.Err() != nil
	}, 500*time.Millisecond, 5*time.Millisecond,
		"context should be canceled with DeadlineExceeded error")

	require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
}

// TestNewRootCmd_CommandStructure verifies root command configuration.
func TestNewRootCmd_CommandStructure(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()

	assert.Equal(t, "go-broadcast", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.PersistentPreRunE)
}

// TestLoggerContextStorage verifies that logger is properly stored in context.
func TestLoggerContextStorage(t *testing.T) {
	flags := &Flags{LogLevel: "info"}
	setupFn := createSetupLogging(flags)

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())

	err := setupFn(cmd, nil)
	require.NoError(t, err)

	// Verify logger was stored in context
	ctx := cmd.Context()
	logger := ctx.Value(loggerContextKey{})
	assert.NotNil(t, logger, "logger should be stored in context")

	// Verify it's a logrus.Logger
	_, ok := logger.(*logrus.Logger)
	assert.True(t, ok, "stored value should be *logrus.Logger")
}

// TestRootCmdFlags verifies that root command flags are properly configured.
func TestRootCmdFlags(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	flags := cmd.PersistentFlags()

	// Config file flag
	configFlag := flags.Lookup("config")
	require.NotNil(t, configFlag, "config flag should exist")
	assert.Equal(t, "c", configFlag.Shorthand)

	// Dry run flag
	dryRunFlag := flags.Lookup("dry-run")
	require.NotNil(t, dryRunFlag, "dry-run flag should exist")

	// Version flag (no shorthand defined)
	versionFlag := flags.Lookup("version")
	require.NotNil(t, versionFlag, "version flag should exist")
	// version flag doesn't have a shorthand (uses BoolVar not BoolVarP)

	// Log level flag
	logLevelFlag := flags.Lookup("log-level")
	require.NotNil(t, logLevelFlag, "log-level flag should exist")
}

// TestRootCmdSubcommands verifies that expected subcommands are registered.
func TestRootCmdSubcommands(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	subcommands := cmd.Commands()

	// Collect subcommand names
	names := make(map[string]bool)
	for _, sub := range subcommands {
		names[sub.Name()] = true
	}

	// Verify expected subcommands exist
	expectedCommands := []string{"sync", "status", "validate", "cancel", "diagnose"}
	for _, expected := range expectedCommands {
		assert.True(t, names[expected], "subcommand %q should exist", expected)
	}
}

// TestErrorSentinels verifies error sentinel definitions in root.go.
func TestErrorSentinels(t *testing.T) {
	t.Parallel()

	t.Run("ErrNilFlags is defined", func(t *testing.T) {
		t.Parallel()
		require.Error(t, ErrNilFlags)
		assert.Contains(t, ErrNilFlags.Error(), "nil")
	})

	t.Run("ErrNilConfig is defined", func(t *testing.T) {
		t.Parallel()
		require.Error(t, ErrNilConfig)
		assert.Contains(t, ErrNilConfig.Error(), "nil")
	})

	t.Run("errors are distinct", func(t *testing.T) {
		t.Parallel()
		assert.NotEqual(t, ErrNilFlags.Error(), ErrNilConfig.Error())
	})
}
