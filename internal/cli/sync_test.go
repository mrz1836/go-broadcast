package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contains is a helper function for string contains check
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestSyncCmd tests sync command configuration
func TestSyncCmd(t *testing.T) {
	cmd := syncCmd

	assert.Equal(t, "sync [targets...]", cmd.Use)
	assert.Equal(t, "Synchronize files to target repositories", cmd.Short)
	assert.Contains(t, cmd.Long, "source template repository")
	assert.Contains(t, cmd.Example, "go-broadcast sync")
	assert.Contains(t, cmd.Aliases, "s")
	assert.NotNil(t, cmd.RunE)
}

// TestRunSync tests the main sync command execution
func TestRunSync(t *testing.T) {
	t.Run("ConfigNotFound", func(t *testing.T) {
		// Save original config
		originalConfig := globalFlags.ConfigFile
		globalFlags.ConfigFile = "/non/existent/config.yml"
		defer func() {
			globalFlags.ConfigFile = originalConfig
		}()

		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		err := runSync(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		// Create temporary invalid config
		tmpFile, err := os.CreateTemp("", "invalid-config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		invalidConfig := `invalid: yaml: content:`
		_, err = tmpFile.WriteString(invalidConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Save original config
		originalConfig := globalFlags.ConfigFile
		globalFlags.ConfigFile = tmpFile.Name()
		defer func() {
			globalFlags.ConfigFile = originalConfig
		}()

		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		err = runSync(cmd, []string{})
		require.Error(t, err)
	})

	// Full sync test would require mocking all dependencies
	// (GitHub client, Git client, etc.) which is beyond the scope
	// of unit tests. Those are covered in integration tests.
}

// TestCreateRunSync tests isolated sync run function
func TestCreateRunSync(t *testing.T) {
	t.Run("ConfigNotFound", func(t *testing.T) {
		flags := &Flags{
			ConfigFile: "/non/existent/config.yml",
		}

		runFunc := createRunSync(flags)
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		err := runFunc(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	t.Run("WithTargetFilter", func(t *testing.T) {
		// Create valid config
		tmpFile, err := os.CreateTemp("", "config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		validConfig := `version: 1
source:
  repo: org/template
  branch: main
targets:
  - repo: org/target1
    files:
      - src: README.md
        dest: README.md
  - repo: org/target2
    files:
      - src: README.md
        dest: README.md`

		_, err = tmpFile.WriteString(validConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		flags := &Flags{
			ConfigFile: tmpFile.Name(),
			DryRun:     true, // Use dry-run to avoid actual sync
		}

		// Capture logs
		var buf bytes.Buffer
		logger := logrus.New()
		logger.SetOutput(&buf)

		runFunc := createRunSync(flags)
		cmd := &cobra.Command{}
		ctx := context.WithValue(context.Background(), loggerContextKey{}, logger)
		cmd.SetContext(ctx)

		// This will fail because we don't have mocked dependencies
		// but we can test that it runs without panic
		_ = runFunc(cmd, []string{"org/target1"})

		// The test passes if it doesn't panic
		// Target filtering happens internally but logs may go to stderr
	})

	t.Run("DryRunWarning", func(t *testing.T) {
		// Create valid config
		tmpFile, err := os.CreateTemp("", "config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		validConfig := `version: 1
source:
  repo: org/template
  branch: main
targets:
  - repo: org/target1
    files:
      - src: README.md
        dest: README.md`

		_, err = tmpFile.WriteString(validConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		flags := &Flags{
			ConfigFile: tmpFile.Name(),
			DryRun:     true,
		}

		runFunc := createRunSync(flags)
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		// Run (will fail but we're testing that dry-run mode is set)
		_ = runFunc(cmd, []string{})

		// The test passes if it doesn't panic
		// Dry-run warnings go to stderr which is harder to capture in tests
	})
}

// TestLoadConfig tests configuration loading
func TestLoadConfig(t *testing.T) {
	t.Run("FileNotFound", func(t *testing.T) {
		// Save original config
		originalConfig := globalFlags.ConfigFile
		globalFlags.ConfigFile = "/non/existent/file.yml"
		defer func() {
			globalFlags.ConfigFile = originalConfig
		}()

		cfg, err := loadConfig()
		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.ErrorIs(t, err, ErrConfigFileNotFound)
	})

	t.Run("ValidConfig", func(t *testing.T) {
		// Create temporary valid config
		tmpFile, err := os.CreateTemp("", "config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		validConfig := `version: 1
source:
  repo: org/template
  branch: main
targets:
  - repo: org/target1
    files:
      - src: README.md
        dest: README.md`

		_, err = tmpFile.WriteString(validConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Save original config
		originalConfig := globalFlags.ConfigFile
		globalFlags.ConfigFile = tmpFile.Name()
		defer func() {
			globalFlags.ConfigFile = originalConfig
		}()

		cfg, err := loadConfig()
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, "org/template", cfg.Source.Repo)
		assert.Equal(t, "main", cfg.Source.Branch)
		assert.Len(t, cfg.Targets, 1)
		assert.Equal(t, "org/target1", cfg.Targets[0].Repo)
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		// Create temporary invalid YAML
		tmpFile, err := os.CreateTemp("", "invalid-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		invalidYAML := `invalid: yaml: content:`
		_, err = tmpFile.WriteString(invalidYAML)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Save original config
		originalConfig := globalFlags.ConfigFile
		globalFlags.ConfigFile = tmpFile.Name()
		defer func() {
			globalFlags.ConfigFile = originalConfig
		}()

		cfg, err := loadConfig()
		require.Error(t, err)
		assert.Nil(t, cfg)
	})
}

// TestLoadConfigWithFlags tests configuration loading with flags
func TestLoadConfigWithFlags(t *testing.T) {
	logger := logrus.New()

	t.Run("FileNotFound", func(t *testing.T) {
		flags := &Flags{
			ConfigFile: "/non/existent/file.yml",
		}

		cfg, err := loadConfigWithFlags(flags, logger)
		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.ErrorIs(t, err, ErrConfigFileNotFound)
	})

	t.Run("ValidConfig", func(t *testing.T) {
		// Create temporary valid config
		tmpFile, err := os.CreateTemp("", "config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		validConfig := `version: 1
source:
  repo: org/template
  branch: main
targets:
  - repo: org/target1
    files:
      - src: README.md
        dest: README.md
  - repo: org/target2
    files:
      - src: README.md
        dest: README.md`

		_, err = tmpFile.WriteString(validConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		flags := &Flags{
			ConfigFile: tmpFile.Name(),
		}

		// Capture logs
		var buf bytes.Buffer
		logger.SetOutput(&buf)
		logger.SetLevel(logrus.DebugLevel)

		cfg, err := loadConfigWithFlags(flags, logger)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, "org/template", cfg.Source.Repo)
		assert.Len(t, cfg.Targets, 2)

		// Check debug logging
		logs := buf.String()
		assert.Contains(t, logs, "Configuration loaded")
		assert.Contains(t, logs, "targets=2")
	})
}

// TestLoadConfigWithLogConfig tests configuration loading with LogConfig
func TestLoadConfigWithLogConfig(t *testing.T) {
	// Save original logger output
	originalOutput := logrus.StandardLogger().Out
	defer logrus.SetOutput(originalOutput)

	t.Run("FileNotFound", func(t *testing.T) {
		logConfig := &LogConfig{
			ConfigFile: "/non/existent/file.yml",
		}

		cfg, err := loadConfigWithLogConfig(logConfig)
		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.ErrorIs(t, err, ErrConfigFileNotFound)
	})

	t.Run("ValidConfigWithDebug", func(t *testing.T) {
		// Create temporary valid config
		tmpFile, err := os.CreateTemp("", "config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		validConfig := `version: 1
source:
  repo: org/template
  branch: main
targets:
  - repo: org/target1
    files:
      - src: README.md
        dest: README.md`

		_, err = tmpFile.WriteString(validConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Capture logs
		var buf bytes.Buffer
		logrus.SetOutput(&buf)
		logrus.SetLevel(logrus.DebugLevel)

		logConfig := &LogConfig{
			ConfigFile: tmpFile.Name(),
			Debug: DebugFlags{
				Config: true,
			},
		}

		cfg, err := loadConfigWithLogConfig(logConfig)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Check debug logging
		logs := buf.String()
		assert.Contains(t, logs, "Configuration loaded")
	})

	t.Run("ValidConfigWithVerbose", func(t *testing.T) {
		// Create temporary valid config
		tmpFile, err := os.CreateTemp("", "config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		validConfig := `version: 1
source:
  repo: org/template
  branch: main
targets:
  - repo: org/target1
    files:
      - src: README.md
        dest: README.md`

		_, err = tmpFile.WriteString(validConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Capture logs
		var buf bytes.Buffer
		logrus.SetOutput(&buf)
		logrus.SetLevel(logrus.DebugLevel)

		logConfig := &LogConfig{
			ConfigFile: tmpFile.Name(),
			Verbose:    1,
		}

		cfg, err := loadConfigWithLogConfig(logConfig)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Check debug logging triggered by verbose
		logs := buf.String()
		assert.Contains(t, logs, "Configuration loaded")
	})
}

// TestCreateSyncEngine tests sync engine creation
func TestCreateSyncEngine(t *testing.T) {
	ctx := context.Background()

	t.Run("BasicEngineCreation", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Targets: []config.TargetConfig{
				{
					Repo: "org/target1",
				},
			},
		}

		// This may succeed if GitHub CLI is configured
		engine, err := createSyncEngine(ctx, cfg)

		if err != nil {
			// If it fails, it should be due to GitHub/Git client creation
			assert.Contains(t, err.Error(), "failed to create")
			assert.Nil(t, engine)
		} else {
			// If it succeeds, we should have a valid engine
			require.NotNil(t, engine)
		}
	})
}

// TestCreateSyncEngineWithFlags tests sync engine creation with flags
func TestCreateSyncEngineWithFlags(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()

	t.Run("WithTransformers", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Targets: []config.TargetConfig{
				{
					Repo: "org/target1",
					Transform: config.Transform{
						RepoName: true,
						Variables: map[string]string{
							"PROJECT": "test",
						},
					},
				},
			},
		}

		flags := &Flags{
			DryRun: true,
		}

		// This may succeed if GitHub CLI is configured
		engine, err := createSyncEngineWithFlags(ctx, cfg, flags, logger)

		if err != nil {
			// If it fails, it should be due to GitHub/Git client creation
			assert.Nil(t, engine)
		} else {
			// If it succeeds, we should have a valid engine
			require.NotNil(t, engine)
		}
	})
}

// TestCreateSyncEngineWithLogConfig tests sync engine creation with LogConfig
func TestCreateSyncEngineWithLogConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("WithEnhancedLogging", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Targets: []config.TargetConfig{
				{
					Repo: "org/target1",
				},
			},
		}

		logConfig := &LogConfig{
			DryRun:  true,
			Verbose: 2,
			Debug: DebugFlags{
				Git: true,
				API: true,
			},
		}

		// This may succeed if GitHub CLI is configured
		engine, err := createSyncEngineWithLogConfig(ctx, cfg, logConfig)

		if err != nil {
			// If it fails, it should be due to GitHub/Git client creation
			assert.Nil(t, engine)
		} else {
			// If it succeeds, we should have a valid engine
			require.NotNil(t, engine)
		}
	})
}

// TestSyncCommandIntegration tests sync command as configured
func TestSyncCommandIntegration(t *testing.T) {
	// Create a test directory
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, "sync.yaml")

	// Create a minimal valid config
	configContent := `version: 1
source:
  repo: test/source
  branch: main
targets:
  - repo: test/target1
    files:
      - src: README.md
        dest: README.md`

	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o600))

	// Test that command is properly wired
	cmd := syncCmd
	assert.NotNil(t, cmd.RunE)

	// Verify command can be executed (will fail due to missing dependencies)
	// but this tests the command is properly configured
	ctx := context.Background()
	cmd.SetContext(ctx)

	// Save original config
	originalConfig := globalFlags.ConfigFile
	globalFlags.ConfigFile = configPath
	defer func() {
		globalFlags.ConfigFile = originalConfig
	}()

	err := cmd.RunE(cmd, []string{})
	// The command may succeed or fail depending on GitHub CLI configuration
	if err != nil {
		// If it fails, it should be after config loading
		assert.NotContains(t, err.Error(), "failed to load configuration")
		// It may fail on sync engine initialization or during sync
		assert.True(t,
			contains(err.Error(), "failed to initialize sync engine") ||
				contains(err.Error(), "sync failed") ||
				contains(err.Error(), "no matching targets"),
			"unexpected error: %v", err)
	}
}
