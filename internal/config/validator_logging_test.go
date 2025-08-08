package config

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

// TestValidateGroupSourceWithLogging tests the validateGroupSourceWithLogging function
func TestValidateGroupSourceWithLogging(t *testing.T) {
	t.Run("successful validation with debug logging", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Source: SourceConfig{
				Repo:   "org/repo",
				Branch: "main",
			},
		}

		ctx := context.Background()
		err := config.validateGroupSourceWithLogging(ctx, logConfig, group)
		require.NoError(t, err)
	})

	t.Run("successful validation without debug logging", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: false,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Source: SourceConfig{
				Repo:   "org/repo",
				Branch: "main",
			},
		}

		ctx := context.Background()
		err := config.validateGroupSourceWithLogging(ctx, logConfig, group)
		require.NoError(t, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Source: SourceConfig{
				Repo:   "org/repo",
				Branch: "main",
			},
		}

		// Create a canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := config.validateGroupSourceWithLogging(ctx, logConfig, group)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "group source validation canceled")
	})

	t.Run("invalid source configuration with debug logging", func(t *testing.T) {
		// Create a test hook and add it to the standard logger
		hook := test.NewGlobal()
		defer hook.Reset()

		// Set log level to capture all messages
		originalLevel := logrus.GetLevel()
		logrus.SetLevel(logrus.TraceLevel)
		defer logrus.SetLevel(originalLevel)

		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Source: SourceConfig{
				Repo:   "", // Invalid empty repo
				Branch: "main",
			},
		}

		ctx := context.Background()
		err := config.validateGroupSourceWithLogging(ctx, logConfig, group)
		require.Error(t, err)

		// Verify error logging occurred
		assert.NotEmpty(t, hook.Entries, "Expected log entries but found none")
	})

	t.Run("nil log config", func(t *testing.T) {
		config := &Config{}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Source: SourceConfig{
				Repo:   "org/repo",
				Branch: "main",
			},
		}

		ctx := context.Background()
		err := config.validateGroupSourceWithLogging(ctx, nil, group)
		require.NoError(t, err)
	})

	t.Run("context timeout during validation", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Source: SourceConfig{
				Repo:   "org/repo",
				Branch: "main",
			},
		}

		// Create a context with immediate timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Give context time to expire
		time.Sleep(10 * time.Millisecond)

		err := config.validateGroupSourceWithLogging(ctx, logConfig, group)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "group source validation canceled")
	})
}

// TestValidateGroupGlobalWithLogging tests the validateGroupGlobalWithLogging function
func TestValidateGroupGlobalWithLogging(t *testing.T) {
	t.Run("successful validation with debug logging", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Global: GlobalConfig{
				PRLabels:    []string{"label1", "label2"},
				PRAssignees: []string{"user1"},
				PRReviewers: []string{"reviewer1"},
			},
		}

		ctx := context.Background()
		err := config.validateGroupGlobalWithLogging(ctx, logConfig, group)
		require.NoError(t, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Global: GlobalConfig{
				PRLabels: []string{"label1"},
			},
		}

		// Create a canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := config.validateGroupGlobalWithLogging(ctx, logConfig, group)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "group global validation canceled")
	})

	t.Run("invalid label with debug logging", func(t *testing.T) {
		// Create a test hook and add it to the standard logger
		hook := test.NewGlobal()
		defer hook.Reset()

		// Set log level to capture all messages
		originalLevel := logrus.GetLevel()
		logrus.SetLevel(logrus.TraceLevel)
		defer logrus.SetLevel(originalLevel)

		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Global: GlobalConfig{
				PRLabels: []string{""}, // Invalid empty label
			},
		}

		ctx := context.Background()
		err := config.validateGroupGlobalWithLogging(ctx, logConfig, group)
		require.Error(t, err)

		// Verify error logging occurred
		assert.NotEmpty(t, hook.Entries, "Expected log entries but found none")
	})

	t.Run("nil log config", func(t *testing.T) {
		config := &Config{}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Global: GlobalConfig{
				PRLabels: []string{"label1"},
			},
		}

		ctx := context.Background()
		err := config.validateGroupGlobalWithLogging(ctx, nil, group)
		require.NoError(t, err)
	})
}

// TestValidateGroupDefaultsWithLogging tests the validateGroupDefaultsWithLogging function
func TestValidateGroupDefaultsWithLogging(t *testing.T) {
	t.Run("successful validation with debug logging", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Defaults: DefaultConfig{
				BranchPrefix: "chore/sync",
				PRLabels:     []string{"automated", "sync"},
			},
		}

		ctx := context.Background()
		err := config.validateGroupDefaultsWithLogging(ctx, logConfig, group)
		require.NoError(t, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Defaults: DefaultConfig{
				BranchPrefix: "feature/test",
				PRLabels:     []string{"test"},
			},
		}

		// Create a canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := config.validateGroupDefaultsWithLogging(ctx, logConfig, group)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "group defaults validation canceled")
	})

	t.Run("invalid variable substitution with debug logging", func(t *testing.T) {
		// Create a test hook and add it to the standard logger
		hook := test.NewGlobal()
		defer hook.Reset()

		// Set log level to capture all messages
		originalLevel := logrus.GetLevel()
		logrus.SetLevel(logrus.TraceLevel)
		defer logrus.SetLevel(originalLevel)

		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Defaults: DefaultConfig{
				BranchPrefix: "chore/sync",
				PRLabels:     []string{""}, // Invalid empty label
			},
		}

		ctx := context.Background()
		err := config.validateGroupDefaultsWithLogging(ctx, logConfig, group)
		require.Error(t, err)

		// Verify error logging occurred
		assert.NotEmpty(t, hook.Entries, "Expected log entries but found none")
	})

	t.Run("nil log config", func(t *testing.T) {
		config := &Config{}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Defaults: DefaultConfig{
				BranchPrefix: "feature/test",
				PRLabels:     []string{"test"},
			},
		}

		ctx := context.Background()
		err := config.validateGroupDefaultsWithLogging(ctx, nil, group)
		require.NoError(t, err)
	})

	t.Run("context timeout", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Defaults: DefaultConfig{
				BranchPrefix: "feature/test",
				PRLabels:     []string{"test"},
			},
		}

		// Create a context with immediate timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Give context time to expire
		time.Sleep(10 * time.Millisecond)

		err := config.validateGroupDefaultsWithLogging(ctx, logConfig, group)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "group defaults validation canceled")
	})
}

// TestTargetConfig_ValidateWithLogging tests the TargetConfig.validateWithLogging function
func TestTargetConfig_ValidateWithLogging(t *testing.T) {
	t.Run("successful validation with debug logging", func(t *testing.T) {
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		target := &TargetConfig{
			Repo: "org/target",
			Files: []FileMapping{
				{Src: "file.txt", Dest: "dest.txt"},
			},
		}

		logger := logrus.WithField("test", "true")
		ctx := context.Background()
		err := target.validateWithLogging(ctx, logConfig, logger)
		require.NoError(t, err)
	})

	t.Run("context cancellation early", func(t *testing.T) {
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		target := &TargetConfig{
			Repo: "org/target",
			Files: []FileMapping{
				{Src: "file.txt", Dest: "dest.txt"},
			},
		}

		logger := logrus.WithField("test", "true")

		// Create a canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := target.validateWithLogging(ctx, logConfig, logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target validation canceled")
	})

	t.Run("invalid target configuration with debug logging", func(t *testing.T) {
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		target := &TargetConfig{
			Repo: "", // Invalid empty repo
			Files: []FileMapping{
				{Src: "file.txt", Dest: "dest.txt"},
			},
		}

		logger := logrus.WithField("test", "true")
		ctx := context.Background()
		err := target.validateWithLogging(ctx, logConfig, logger)
		require.Error(t, err)
	})

	t.Run("nil log config", func(t *testing.T) {
		target := &TargetConfig{
			Repo: "org/target",
			Files: []FileMapping{
				{Src: "file.txt", Dest: "dest.txt"},
			},
		}

		logger := logrus.WithField("test", "true")
		ctx := context.Background()
		err := target.validateWithLogging(ctx, nil, logger)
		require.NoError(t, err)
	})

	t.Run("validation without debug logging", func(t *testing.T) {
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: false,
			},
		}

		target := &TargetConfig{
			Repo: "org/target",
			Files: []FileMapping{
				{Src: "file.txt", Dest: "dest.txt"},
			},
		}

		logger := logrus.WithField("test", "true")
		ctx := context.Background()
		err := target.validateWithLogging(ctx, logConfig, logger)
		require.NoError(t, err)
	})

	t.Run("context timeout during validation", func(t *testing.T) {
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		target := &TargetConfig{
			Repo: "org/target",
			Files: []FileMapping{
				{Src: "file.txt", Dest: "dest.txt"},
			},
		}

		logger := logrus.WithField("test", "true")

		// Create a context with immediate timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Give context time to expire
		time.Sleep(10 * time.Millisecond)

		err := target.validateWithLogging(ctx, logConfig, logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target validation canceled")
	})
}

// TestValidateWithLoggingComplexScenarios tests complex validation scenarios
func TestValidateWithLoggingComplexScenarios(t *testing.T) {
	t.Run("nested validation with multiple contexts", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		// Create a group with all configurations
		group := Group{
			Name: "complex-group",
			ID:   "complex-id",
			Source: SourceConfig{
				Repo:   "org/source",
				Branch: "main",
			},
			Global: GlobalConfig{
				PRLabels:    []string{"automated", "sync"},
				PRAssignees: []string{"bot"},
				PRReviewers: []string{"team-lead"},
			},
			Defaults: DefaultConfig{
				BranchPrefix:    "chore/sync-from-template",
				PRLabels:        []string{"automated", "template-sync"},
				PRAssignees:     []string{"bot-user"},
				PRReviewers:     []string{"lead-dev"},
				PRTeamReviewers: []string{"devops-team"},
			},
		}

		ctx := context.Background()

		// Validate each component
		err := config.validateGroupSourceWithLogging(ctx, logConfig, group)
		require.NoError(t, err)

		err = config.validateGroupGlobalWithLogging(ctx, logConfig, group)
		require.NoError(t, err)

		err = config.validateGroupDefaultsWithLogging(ctx, logConfig, group)
		require.NoError(t, err)
	})

	t.Run("validation with partial cancellation", func(t *testing.T) {
		config := &Config{}
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Source: SourceConfig{
				Repo:   "org/repo",
				Branch: "main",
			},
			Global: GlobalConfig{
				PRLabels: []string{"label"},
			},
		}

		// First validation succeeds
		ctx1 := context.Background()
		err := config.validateGroupSourceWithLogging(ctx1, logConfig, group)
		require.NoError(t, err)

		// Second validation with canceled context fails
		ctx2, cancel := context.WithCancel(context.Background())
		cancel()
		err = config.validateGroupGlobalWithLogging(ctx2, logConfig, group)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "canceled")
	})

	t.Run("validation with varying debug levels", func(t *testing.T) {
		config := &Config{}
		group := Group{
			Name: "test-group",
			ID:   "test-id",
			Source: SourceConfig{
				Repo:   "org/repo",
				Branch: "main",
			},
		}

		ctx := context.Background()

		// Test with debug enabled
		logConfigDebug := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}
		err := config.validateGroupSourceWithLogging(ctx, logConfigDebug, group)
		require.NoError(t, err)

		// Test with debug disabled
		logConfigNoDebug := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: false,
			},
		}
		err = config.validateGroupSourceWithLogging(ctx, logConfigNoDebug, group)
		require.NoError(t, err)

		// Test with nil config
		err = config.validateGroupSourceWithLogging(ctx, nil, group)
		require.NoError(t, err)
	})
}
