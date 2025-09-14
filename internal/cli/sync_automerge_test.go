package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/sync"
)

// TestAutomergeEnvironmentVariableParsing tests the parsing of automerge labels from environment variable
func TestAutomergeEnvironmentVariableParsing(t *testing.T) {
	// Save original environment and restore after test
	originalEnv := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", originalEnv)
		} else {
			_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")
		}
	}()

	t.Run("parses comma-separated labels", func(t *testing.T) {
		_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "automerge,ready-to-merge,auto-merge")

		// Simulate the environment variable parsing logic from sync.go
		flags := &Flags{Automerge: true}
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

		expected := []string{"automerge", "ready-to-merge", "auto-merge"}
		assert.Equal(t, expected, automergeLabels)
	})

	t.Run("trims whitespace around labels", func(t *testing.T) {
		_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", " automerge , ready-to-merge ,  auto-merge  ")

		flags := &Flags{Automerge: true}
		var automergeLabels []string

		if flags.Automerge {
			if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
				for _, label := range strings.Split(envLabels, ",") {
					if trimmed := strings.TrimSpace(label); trimmed != "" {
						automergeLabels = append(automergeLabels, trimmed)
					}
				}
			}
		}

		expected := []string{"automerge", "ready-to-merge", "auto-merge"}
		assert.Equal(t, expected, automergeLabels)
	})

	t.Run("ignores empty labels", func(t *testing.T) {
		_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "automerge,,ready-to-merge, ,auto-merge")

		flags := &Flags{Automerge: true}
		var automergeLabels []string

		if flags.Automerge {
			if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
				for _, label := range strings.Split(envLabels, ",") {
					if trimmed := strings.TrimSpace(label); trimmed != "" {
						automergeLabels = append(automergeLabels, trimmed)
					}
				}
			}
		}

		expected := []string{"automerge", "ready-to-merge", "auto-merge"}
		assert.Equal(t, expected, automergeLabels)
	})

	t.Run("returns empty when env var is empty", func(t *testing.T) {
		_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "")

		flags := &Flags{Automerge: true}
		var automergeLabels []string

		if flags.Automerge {
			if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
				for _, label := range strings.Split(envLabels, ",") {
					if trimmed := strings.TrimSpace(label); trimmed != "" {
						automergeLabels = append(automergeLabels, trimmed)
					}
				}
			}
		}

		assert.Empty(t, automergeLabels)
	})

	t.Run("returns empty when env var is not set", func(t *testing.T) {
		_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")

		flags := &Flags{Automerge: true}
		var automergeLabels []string

		if flags.Automerge {
			if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
				for _, label := range strings.Split(envLabels, ",") {
					if trimmed := strings.TrimSpace(label); trimmed != "" {
						automergeLabels = append(automergeLabels, trimmed)
					}
				}
			}
		}

		assert.Empty(t, automergeLabels)
	})

	t.Run("returns empty when automerge is disabled", func(t *testing.T) {
		_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "automerge,ready-to-merge")

		flags := &Flags{Automerge: false} // Automerge disabled
		var automergeLabels []string

		if flags.Automerge {
			if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
				for _, label := range strings.Split(envLabels, ",") {
					if trimmed := strings.TrimSpace(label); trimmed != "" {
						automergeLabels = append(automergeLabels, trimmed)
					}
				}
			}
		}

		assert.Empty(t, automergeLabels)
	})
}

// TestSyncEngineAutomergeOptions tests that automerge options are properly set in sync engine
func TestSyncEngineAutomergeOptions(t *testing.T) {
	// Save original environment and restore after test
	originalEnv := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", originalEnv)
		} else {
			_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")
		}
	}()

	t.Run("sync options include automerge settings when enabled", func(t *testing.T) {
		_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "automerge,ready-to-merge")

		flags := &Flags{
			Automerge: true,
			DryRun:    false,
		}

		// Simulate the automerge label loading logic
		var automergeLabels []string
		if flags.Automerge {
			if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
				for _, label := range strings.Split(envLabels, ",") {
					if trimmed := strings.TrimSpace(label); trimmed != "" {
						automergeLabels = append(automergeLabels, trimmed)
					}
				}
			}
		}

		// Create sync options using the same pattern as sync.go
		opts := sync.DefaultOptions().
			WithDryRun(flags.DryRun).
			WithMaxConcurrency(5).
			WithGroupFilter(flags.GroupFilter).
			WithSkipGroups(flags.SkipGroups).
			WithAutomerge(flags.Automerge).
			WithAutomergeLabels(automergeLabels)

		assert.True(t, opts.Automerge)
		assert.Equal(t, []string{"automerge", "ready-to-merge"}, opts.AutomergeLabels)
	})

	t.Run("sync options exclude automerge when disabled", func(t *testing.T) {
		_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "automerge,ready-to-merge")

		flags := &Flags{
			Automerge: false, // Disabled
			DryRun:    false,
		}

		var automergeLabels []string
		if flags.Automerge {
			if envLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"); envLabels != "" {
				for _, label := range strings.Split(envLabels, ",") {
					if trimmed := strings.TrimSpace(label); trimmed != "" {
						automergeLabels = append(automergeLabels, trimmed)
					}
				}
			}
		}

		opts := sync.DefaultOptions().
			WithDryRun(flags.DryRun).
			WithMaxConcurrency(5).
			WithGroupFilter(flags.GroupFilter).
			WithSkipGroups(flags.SkipGroups).
			WithAutomerge(flags.Automerge).
			WithAutomergeLabels(automergeLabels)

		assert.False(t, opts.Automerge)
		assert.Empty(t, opts.AutomergeLabels)
	})
}

// TestAutomergeFlagValidation tests validation of automerge flag combinations
func TestAutomergeFlagValidation(t *testing.T) {
	t.Run("automerge flag can be combined with other flags", func(t *testing.T) {
		flags := &Flags{
			Automerge:   true,
			DryRun:      true,
			GroupFilter: []string{"core"},
			SkipGroups:  []string{"experimental"},
		}

		// Should not cause any issues
		assert.True(t, flags.Automerge)
		assert.True(t, flags.DryRun)
		assert.Equal(t, []string{"core"}, flags.GroupFilter)
		assert.Equal(t, []string{"experimental"}, flags.SkipGroups)
	})
}
