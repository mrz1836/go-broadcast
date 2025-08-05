package sync

import (
	"testing"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConflictResolver(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("last-wins strategy", func(t *testing.T) {
		resolver := NewConflictResolver(config.ConflictResolution{
			Strategy: "last-wins",
		}, logger)

		conflict := FileConflict{
			TargetFile: ".github/workflows/ci.yml",
			Sources: []SourceFileInfo{
				{
					SourceRepo:   "org/source1",
					SourceFile:   "ci1.yml",
					SourceID:     "src1",
					MappingIndex: 0,
					Content:      []byte("content from source1"),
				},
				{
					SourceRepo:   "org/source2",
					SourceFile:   "ci2.yml",
					SourceID:     "src2",
					MappingIndex: 1,
					Content:      []byte("content from source2"),
				},
				{
					SourceRepo:   "org/source3",
					SourceFile:   "ci3.yml",
					SourceID:     "src3",
					MappingIndex: 2,
					Content:      []byte("content from source3"),
				},
			},
		}

		resolved, err := resolver.ResolveConflicts([]FileConflict{conflict})
		require.NoError(t, err)

		// Should pick the last mapping (highest index)
		winner := resolved[".github/workflows/ci.yml"]
		assert.Equal(t, "org/source3", winner.SourceRepo)
		assert.Equal(t, "src3", winner.SourceID)
		assert.Equal(t, 2, winner.MappingIndex)
	})

	t.Run("priority strategy", func(t *testing.T) {
		resolver := NewConflictResolver(config.ConflictResolution{
			Strategy: "priority",
			Priority: []string{"security", "ci", "docs"},
		}, logger)

		conflict := FileConflict{
			TargetFile: "policy.yml",
			Sources: []SourceFileInfo{
				{
					SourceRepo:   "org/docs-templates",
					SourceFile:   "policy-docs.yml",
					SourceID:     "docs",
					MappingIndex: 0,
				},
				{
					SourceRepo:   "org/security-templates",
					SourceFile:   "policy-security.yml",
					SourceID:     "security",
					MappingIndex: 1,
				},
				{
					SourceRepo:   "org/ci-templates",
					SourceFile:   "policy-ci.yml",
					SourceID:     "ci",
					MappingIndex: 2,
				},
			},
		}

		resolved, err := resolver.ResolveConflicts([]FileConflict{conflict})
		require.NoError(t, err)

		// Should pick "security" as it has highest priority (lowest index)
		winner := resolved["policy.yml"]
		assert.Equal(t, "org/security-templates", winner.SourceRepo)
		assert.Equal(t, "security", winner.SourceID)
	})

	t.Run("priority strategy with missing source IDs", func(t *testing.T) {
		resolver := NewConflictResolver(config.ConflictResolution{
			Strategy: "priority",
			Priority: []string{"high", "medium", "low"},
		}, logger)

		conflict := FileConflict{
			TargetFile: "config.yml",
			Sources: []SourceFileInfo{
				{
					SourceRepo:   "org/source1",
					SourceFile:   "config1.yml",
					SourceID:     "not-in-priority",
					MappingIndex: 0,
				},
				{
					SourceRepo:   "org/source2",
					SourceFile:   "config2.yml",
					SourceID:     "medium",
					MappingIndex: 1,
				},
			},
		}

		resolved, err := resolver.ResolveConflicts([]FileConflict{conflict})
		require.NoError(t, err)

		// Should pick "medium" as it's the only one in priority list
		winner := resolved["config.yml"]
		assert.Equal(t, "org/source2", winner.SourceRepo)
		assert.Equal(t, "medium", winner.SourceID)
	})

	t.Run("error strategy", func(t *testing.T) {
		resolver := NewConflictResolver(config.ConflictResolution{
			Strategy: "error",
		}, logger)

		conflict := FileConflict{
			TargetFile: "conflicted.yml",
			Sources: []SourceFileInfo{
				{SourceRepo: "org/source1", SourceID: "src1"},
				{SourceRepo: "org/source2", SourceID: "src2"},
			},
		}

		_, err := resolver.ResolveConflicts([]FileConflict{conflict})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "conflict detected")
		assert.Contains(t, err.Error(), "2 sources want to sync this file")
	})

	t.Run("no conflict with single source", func(t *testing.T) {
		resolver := NewConflictResolver(config.ConflictResolution{
			Strategy: "error", // Even with error strategy
		}, logger)

		conflict := FileConflict{
			TargetFile: "single.yml",
			Sources: []SourceFileInfo{
				{
					SourceRepo: "org/source1",
					SourceFile: "single.yml",
					SourceID:   "src1",
				},
			},
		}

		resolved, err := resolver.ResolveConflicts([]FileConflict{conflict})
		require.NoError(t, err)

		// Should return the single source
		winner := resolved["single.yml"]
		assert.Equal(t, "org/source1", winner.SourceRepo)
	})

	t.Run("unknown strategy defaults to last-wins", func(t *testing.T) {
		resolver := NewConflictResolver(config.ConflictResolution{
			Strategy: "unknown-strategy",
		}, logger)

		conflict := FileConflict{
			TargetFile: "file.yml",
			Sources: []SourceFileInfo{
				{SourceRepo: "org/source1", MappingIndex: 0},
				{SourceRepo: "org/source2", MappingIndex: 1},
			},
		}

		resolved, err := resolver.ResolveConflicts([]FileConflict{conflict})
		require.NoError(t, err)

		// Should default to last-wins
		winner := resolved["file.yml"]
		assert.Equal(t, "org/source2", winner.SourceRepo)
	})
}

func TestDetectConflicts(t *testing.T) {
	tasks := []Task{
		{
			Source: config.SourceConfig{
				Repo: "org/source1",
				ID:   "src1",
			},
			Target: config.TargetConfig{
				Repo: "org/target1",
				Files: []config.FileMapping{
					{Src: "ci.yml", Dest: ".github/workflows/ci.yml"},
					{Src: "readme.md", Dest: "README.md"},
				},
			},
			MappingIdx: 0,
		},
		{
			Source: config.SourceConfig{
				Repo: "org/source2",
				ID:   "src2",
			},
			Target: config.TargetConfig{
				Repo: "org/target1",
				Files: []config.FileMapping{
					{Src: "workflow.yml", Dest: ".github/workflows/ci.yml"}, // Conflict!
					{Src: "config.yml", Dest: "config.yml"},
				},
			},
			MappingIdx: 1,
		},
		{
			Source: config.SourceConfig{
				Repo: "org/source3",
				ID:   "src3",
			},
			Target: config.TargetConfig{
				Repo: "org/target2",
				Files: []config.FileMapping{
					{Src: "other.yml", Dest: "other.yml"}, // No conflict
				},
			},
			MappingIdx: 2,
		},
	}

	conflicts := DetectConflicts(tasks)

	// Should detect one conflict
	assert.Len(t, conflicts, 1)

	// Check the conflicting file
	sources, exists := conflicts[".github/workflows/ci.yml"]
	assert.True(t, exists)
	assert.Len(t, sources, 2)

	// Verify the sources
	sourceRepos := make(map[string]bool)
	for _, src := range sources {
		sourceRepos[src.SourceRepo] = true
	}
	assert.True(t, sourceRepos["org/source1"])
	assert.True(t, sourceRepos["org/source2"])

	// Non-conflicting files should not be in the map
	_, exists = conflicts["README.md"]
	assert.False(t, exists)
	_, exists = conflicts["config.yml"]
	assert.False(t, exists)
	_, exists = conflicts["other.yml"]
	assert.False(t, exists)
}

func TestComplexConflictScenario(t *testing.T) {
	// Test a scenario with multiple conflicts and different resolution strategies
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	conflicts := []FileConflict{
		{
			TargetFile: "Makefile",
			Sources: []SourceFileInfo{
				{SourceRepo: "org/base", SourceID: "base", MappingIndex: 0},
				{SourceRepo: "org/golang", SourceID: "golang", MappingIndex: 1},
				{SourceRepo: "org/custom", SourceID: "custom", MappingIndex: 2},
			},
		},
		{
			TargetFile: ".github/workflows/ci.yml",
			Sources: []SourceFileInfo{
				{SourceRepo: "org/ci-basic", SourceID: "ci-basic", MappingIndex: 0},
				{SourceRepo: "org/ci-advanced", SourceID: "ci-advanced", MappingIndex: 1},
			},
		},
		{
			TargetFile: "README.md",
			Sources: []SourceFileInfo{
				{SourceRepo: "org/docs", SourceID: "docs", MappingIndex: 0},
			},
		},
	}

	t.Run("resolve with priority", func(t *testing.T) {
		resolver := NewConflictResolver(config.ConflictResolution{
			Strategy: "priority",
			Priority: []string{"custom", "golang", "ci-advanced", "base", "ci-basic", "docs"},
		}, logger)

		resolved, err := resolver.ResolveConflicts(conflicts)
		require.NoError(t, err)
		assert.Len(t, resolved, 3)

		// Check resolutions
		assert.Equal(t, "org/custom", resolved["Makefile"].SourceRepo)
		assert.Equal(t, "org/ci-advanced", resolved[".github/workflows/ci.yml"].SourceRepo)
		assert.Equal(t, "org/docs", resolved["README.md"].SourceRepo)
	})

	t.Run("resolve with last-wins", func(t *testing.T) {
		resolver := NewConflictResolver(config.ConflictResolution{
			Strategy: "last-wins",
		}, logger)

		resolved, err := resolver.ResolveConflicts(conflicts)
		require.NoError(t, err)
		assert.Len(t, resolved, 3)

		// Check resolutions - should pick highest mapping index
		assert.Equal(t, "org/custom", resolved["Makefile"].SourceRepo)
		assert.Equal(t, 2, resolved["Makefile"].MappingIndex)
		assert.Equal(t, "org/ci-advanced", resolved[".github/workflows/ci.yml"].SourceRepo)
		assert.Equal(t, 1, resolved[".github/workflows/ci.yml"].MappingIndex)
	})
}
