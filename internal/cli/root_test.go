package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRootCmd tests creation of isolated root command
func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()
	assert.Equal(t, "go-broadcast", cmd.Use)
	assert.Equal(t, "Synchronize files from source repos to multiple targets", cmd.Short)
	assert.NotNil(t, cmd.PersistentPreRunE)
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)

	// Check subcommands
	subcommands := []string{"sync", "status", "validate", "version", "diagnose"}
	for _, name := range subcommands {
		t.Run(fmt.Sprintf("HasCommand%s", name), func(t *testing.T) {
			found := false
			for _, subcmd := range cmd.Commands() {
				if subcmd.Name() == name {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected to find command: %s", name)
		})
	}
}

// TestNewRootCmdWithVerbose tests creation of root command with verbose support
func TestNewRootCmdWithVerbose(t *testing.T) {
	cmd := NewRootCmdWithVerbose()

	assert.Equal(t, "go-broadcast", cmd.Use)
	assert.NotNil(t, cmd.PersistentPreRunE)

	// Check that verbose flags are added
	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	require.NotNil(t, verboseFlag)
	assert.Equal(t, "v", verboseFlag.Shorthand)

	// Check debug flags
	debugFlags := []string{"debug-git", "debug-api", "debug-transform", "debug-config", "debug-state"}
	for _, flagName := range debugFlags {
		t.Run(fmt.Sprintf("HasFlag%s", flagName), func(t *testing.T) {
			flag := cmd.PersistentFlags().Lookup(flagName)
			assert.NotNil(t, flag, "Expected to find flag: %s", flagName)
		})
	}

	// Check log format flags
	logFormatFlag := cmd.PersistentFlags().Lookup("log-format")
	require.NotNil(t, logFormatFlag)
	assert.Equal(t, "text", logFormatFlag.DefValue)

	jsonFlag := cmd.PersistentFlags().Lookup("json")
	require.NotNil(t, jsonFlag)
}

// TestGetRootCmd tests GetRootCmd returns isolated instance
func TestGetRootCmd(t *testing.T) {
	cmd1 := GetRootCmd()
	cmd2 := GetRootCmd()
	// Should return new instances for isolation
	assert.NotSame(t, cmd1, cmd2)

	// Both should be properly configured
	assert.Equal(t, "go-broadcast", cmd1.Use)
	assert.Equal(t, "go-broadcast", cmd2.Use)
}

// TestCreateSetupLogging tests isolated logging setup
func TestCreateSetupLogging(t *testing.T) {
	testCases := []struct {
		name      string
		logLevel  string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "ValidDebugLevel",
			logLevel:  "debug",
			expectErr: false,
		},
		{
			name:      "ValidInfoLevel",
			logLevel:  "info",
			expectErr: false,
		},
		{
			name:      "ValidWarnLevel",
			logLevel:  "warn",
			expectErr: false,
		},
		{
			name:      "ValidErrorLevel",
			logLevel:  "error",
			expectErr: false,
		},
		{
			name:      "InvalidLogLevel",
			logLevel:  "invalid",
			expectErr: true,
			errMsg:    "invalid log level",
		},
		{
			name:      "CaseInsensitive",
			logLevel:  "DEBUG",
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flags := &Flags{
				LogLevel: tc.logLevel,
			}

			setupFunc := createSetupLogging(flags)
			require.NotNil(t, setupFunc)

			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			err := setupFunc(cmd, []string{})

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)

				// Check that logger was stored in context
				logger, ok := cmd.Context().Value(loggerContextKey{}).(*logrus.Logger)
				require.True(t, ok)
				require.NotNil(t, logger)

				// Verify log level was set correctly
				expectedLevel, _ := logrus.ParseLevel(strings.ToLower(tc.logLevel))
				assert.Equal(t, expectedLevel, logger.GetLevel())
			}
		})
	}
}

// TestSetupLoggingGlobal tests global logging setup
func TestSetupLoggingGlobal(t *testing.T) {
	// Save original state
	originalLevel := logrus.GetLevel()
	originalOutput := logrus.StandardLogger().Out
	defer func() {
		logrus.SetLevel(originalLevel)
		logrus.SetOutput(originalOutput)
	}()

	testCases := []struct {
		name      string
		flags     Flags
		expectErr bool
	}{
		{
			name: "ValidConfiguration",
			flags: Flags{
				ConfigFile: "test.yaml",
				LogLevel:   "debug",
				DryRun:     true,
			},
			expectErr: false,
		},
		{
			name: "InvalidLogLevel",
			flags: Flags{
				LogLevel: "invalid",
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set global flags
			globalFlags = &tc.flags

			cmd := &cobra.Command{}
			err := setupLogging(cmd, []string{})

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify global logger was configured
				expectedLevel, _ := logrus.ParseLevel(tc.flags.LogLevel)
				assert.Equal(t, expectedLevel, logrus.GetLevel())
				assert.Equal(t, os.Stderr, logrus.StandardLogger().Out)
			}
		})
	}
}

// TestAddVerboseFlags tests verbose flag addition
func TestAddVerboseFlags(t *testing.T) {
	cmd := &cobra.Command{}
	config := &LogConfig{}

	addVerboseFlags(cmd, config)

	// Check verbose counter flag
	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	require.NotNil(t, verboseFlag)
	assert.Equal(t, "v", verboseFlag.Shorthand)
	assert.Contains(t, verboseFlag.Usage, "-vvv")

	// Check all debug flags
	debugFlags := map[string]string{
		"debug-git":       "Debug git operations",
		"debug-api":       "Debug GitHub API",
		"debug-transform": "Debug file transformations",
		"debug-config":    "Debug configuration",
		"debug-state":     "Debug state discovery",
	}

	for flag, expectedUsage := range debugFlags {
		t.Run(flag, func(t *testing.T) {
			f := cmd.PersistentFlags().Lookup(flag)
			require.NotNil(t, f)
			assert.Contains(t, f.Usage, expectedUsage)
		})
	}

	// Check format flags
	logFormatFlag := cmd.PersistentFlags().Lookup("log-format")
	require.NotNil(t, logFormatFlag)
	assert.Equal(t, "text", logFormatFlag.DefValue)

	jsonFlag := cmd.PersistentFlags().Lookup("json")
	require.NotNil(t, jsonFlag)
	assert.Equal(t, "false", jsonFlag.DefValue)

	// Check standard flags
	configFlag := cmd.PersistentFlags().Lookup("config")
	require.NotNil(t, configFlag)
	assert.Equal(t, "c", configFlag.Shorthand)
	assert.Equal(t, "sync.yaml", configFlag.DefValue)

	dryRunFlag := cmd.PersistentFlags().Lookup("dry-run")
	require.NotNil(t, dryRunFlag)
	assert.Equal(t, "false", dryRunFlag.DefValue)
}

// TestCreateSetupLoggingWithVerbose tests verbose logging setup
func TestCreateSetupLoggingWithVerbose(t *testing.T) {
	// Save original logger state
	originalLevel := logrus.GetLevel()
	originalOutput := logrus.StandardLogger().Out
	defer func() {
		logrus.SetLevel(originalLevel)
		logrus.SetOutput(originalOutput)
	}()

	testCases := []struct {
		name      string
		config    LogConfig
		expectErr bool
	}{
		{
			name: "BasicConfiguration",
			config: LogConfig{
				ConfigFile: "test.yaml",
				LogLevel:   "info",
				LogFormat:  "text",
			},
			expectErr: false,
		},
		{
			name: "JSONOutputFlag",
			config: LogConfig{
				JSONOutput: true,
				LogFormat:  "text",
			},
			expectErr: false,
		},
		{
			name: "VerboseWithDebugFlags",
			config: LogConfig{
				Verbose:   2,
				LogFormat: "json",
				Debug: DebugFlags{
					Git: true,
					API: true,
				},
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.config
			setupFunc := createSetupLoggingWithVerbose(&config)
			require.NotNil(t, setupFunc)

			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			err := setupFunc(cmd, []string{})

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify correlation ID was generated
				if config.CorrelationID == "" {
					assert.NotEmpty(t, config.CorrelationID)
				}

				// Verify JSON output flag handling
				if tc.config.JSONOutput {
					assert.Equal(t, "json", config.LogFormat)
				}
			}
		})
	}
}

// TestCommandCreationFunctions tests individual command creation
func TestCommandCreationFunctions(t *testing.T) {
	flags := &Flags{
		ConfigFile: "test.yaml",
		LogLevel:   "info",
		DryRun:     false,
	}

	t.Run("CreateSyncCmd", func(t *testing.T) {
		cmd := createSyncCmd(flags)
		assert.Equal(t, "sync [targets...]", cmd.Use)
		assert.Contains(t, cmd.Short, "Synchronize files")
		assert.NotNil(t, cmd.RunE)
		assert.Contains(t, cmd.Aliases, "s")
	})

	t.Run("CreateStatusCmd", func(t *testing.T) {
		cmd := createStatusCmd(flags)
		assert.Equal(t, "status", cmd.Use)
		assert.Contains(t, cmd.Short, "Show status")
		assert.NotNil(t, cmd.RunE)
		assert.Contains(t, cmd.Aliases, "st")
	})

	t.Run("CreateValidateCmd", func(t *testing.T) {
		cmd := createValidateCmd(flags)
		assert.Equal(t, "validate", cmd.Use)
		assert.Contains(t, cmd.Short, "Validate configuration")
		assert.NotNil(t, cmd.RunE)
		assert.Contains(t, cmd.Aliases, "v")
		assert.Contains(t, cmd.Aliases, "check")
	})

	t.Run("CreateVersionCmd", func(t *testing.T) {
		cmd := createVersionCmd(flags)
		assert.Equal(t, "version", cmd.Use)
		assert.Contains(t, cmd.Short, "Print version")
		assert.NotNil(t, cmd.RunE)
	})
}

// TestVerboseCommandCreationFunctions tests verbose command creation
func TestVerboseCommandCreationFunctions(t *testing.T) {
	config := &LogConfig{
		ConfigFile: "test.yaml",
		LogLevel:   "info",
		LogFormat:  "text",
	}

	t.Run("CreateSyncCmdWithVerbose", func(t *testing.T) {
		cmd := createSyncCmdWithVerbose(config)
		assert.Equal(t, "sync [targets...]", cmd.Use)
		assert.NotNil(t, cmd.RunE)
		assert.Contains(t, cmd.Example, "-vvv")
		assert.Contains(t, cmd.Example, "--debug-git")
	})

	t.Run("CreateStatusCmdWithVerbose", func(t *testing.T) {
		cmd := createStatusCmdWithVerbose(config)
		assert.Equal(t, "status", cmd.Use)
		assert.NotNil(t, cmd.RunE)
		assert.Contains(t, cmd.Example, "--debug-state")
	})

	t.Run("CreateValidateCmdWithVerbose", func(t *testing.T) {
		cmd := createValidateCmdWithVerbose(config)
		assert.Equal(t, "validate", cmd.Use)
		assert.NotNil(t, cmd.RunE)
		assert.Contains(t, cmd.Example, "--debug-config")
	})

	t.Run("CreateVersionCmdWithVerbose", func(t *testing.T) {
		cmd := createVersionCmdWithVerbose(config)
		assert.Equal(t, "version", cmd.Use)
		assert.NotNil(t, cmd.RunE)
		assert.Contains(t, cmd.Example, "--json")
	})
}

// TestCreateRunStatus tests status run function creation
func TestCreateRunStatus(t *testing.T) {
	flags := &Flags{}
	runFunc := createRunStatus(flags)
	require.NotNil(t, runFunc)

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runFunc(cmd, []string{})
	require.Error(t, err)
	assert.Equal(t, ErrStatusNotImplemented, err)
}

// TestCreateRunValidate tests validate run function
func TestCreateRunValidate(t *testing.T) {
	t.Run("ConfigNotFound", func(t *testing.T) {
		flags := &Flags{
			ConfigFile: "/non/existent/config.yml",
		}

		runFunc := createRunValidate(flags)
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		err := runFunc(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	t.Run("ValidConfig", func(t *testing.T) {
		// Create temporary valid config
		tmpFile, err := os.CreateTemp("", "config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		validConfig := TestValidConfig

		_, err = tmpFile.WriteString(validConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		flags := &Flags{
			ConfigFile: tmpFile.Name(),
		}

		runFunc := createRunValidate(flags)
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		err = runFunc(cmd, []string{})
		require.NoError(t, err)
	})
}

// TestCreateRunVersion tests version run function
func TestCreateRunVersion(t *testing.T) {
	flags := &Flags{}
	runFunc := createRunVersion(flags)
	require.NotNil(t, runFunc)

	// No need to capture output - we just verify it runs without error
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runFunc(cmd, []string{})
	require.NoError(t, err)
}

// TestCreateRunValidateWithVerbose tests verbose validate run function
func TestCreateRunValidateWithVerbose(t *testing.T) {
	// Create temporary valid config
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	validConfig := TestValidConfig

	_, err = tmpFile.WriteString(validConfig)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	config := &LogConfig{
		ConfigFile: tmpFile.Name(),
		LogLevel:   "info",
	}

	runFunc := createRunValidateWithVerbose(config)
	cmd := &cobra.Command{}

	err = runFunc(cmd, []string{})
	require.NoError(t, err)
}

// TestCreateRunStatusWithVerbose tests verbose status run function
func TestCreateRunStatusWithVerbose(t *testing.T) {
	config := &LogConfig{}
	runFunc := createRunStatusWithVerbose(config)
	require.NotNil(t, runFunc)

	cmd := &cobra.Command{}
	err := runFunc(cmd, []string{})
	require.Error(t, err)
	assert.Equal(t, ErrStatusNotImplemented, err)
}

// TestCreateRunVersionWithVerbose tests verbose version run function
func TestCreateRunVersionWithVerbose(t *testing.T) {
	config := &LogConfig{}
	runFunc := createRunVersionWithVerbose(config)
	require.NotNil(t, runFunc)

	// No need to capture output - we just verify it runs without error
	cmd := &cobra.Command{}
	err := runFunc(cmd, []string{})
	require.NoError(t, err)
}

// TestLoggerContextKey tests logger context storage
func TestLoggerContextKey(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()

	// Store logger in context
	ctx = context.WithValue(ctx, loggerContextKey{}, logger)

	// Retrieve logger
	retrieved, ok := ctx.Value(loggerContextKey{}).(*logrus.Logger)
	require.True(t, ok)
	assert.Same(t, logger, retrieved)
}

// TestRootCommandIntegration tests the root command integration
func TestRootCommandIntegration(t *testing.T) {
	// This test verifies the global root command is properly configured
	cmd := rootCmd

	assert.Equal(t, "go-broadcast", cmd.Use)
	assert.NotNil(t, cmd.PersistentPreRunE)

	// Check that all expected subcommands exist
	expectedCommands := []string{"sync", "status", "validate", "version", "diagnose"}
	for _, expected := range expectedCommands {
		found := false
		for _, sub := range cmd.Commands() {
			if sub.Name() == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find command: %s", expected)
	}

	// Check global flags
	configFlag := cmd.PersistentFlags().Lookup("config")
	require.NotNil(t, configFlag)
	assert.Equal(t, "c", configFlag.Shorthand)

	dryRunFlag := cmd.PersistentFlags().Lookup("dry-run")
	require.NotNil(t, dryRunFlag)

	logLevelFlag := cmd.PersistentFlags().Lookup("log-level")
	require.NotNil(t, logLevelFlag)
	assert.Equal(t, "info", logLevelFlag.DefValue)
}
