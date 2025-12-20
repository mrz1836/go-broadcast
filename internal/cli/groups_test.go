package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestFilterConfigByGroups_Comprehensive tests the group filtering functionality with additional edge cases
func TestFilterConfigByGroups_Comprehensive(t *testing.T) {
	t.Parallel()

	// Helper to create a test config with multiple groups
	createTestConfig := func() *config.Config {
		return &config.Config{
			Version: 1,
			Name:    "test-config",
			ID:      "test-config-id",
			Groups: []config.Group{
				{
					Name: "Core Group",
					ID:   "core",
					Source: config.SourceConfig{
						Repo:   "org/template",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "org/target1"},
					},
				},
				{
					Name: "Security Group",
					ID:   "security",
					Source: config.SourceConfig{
						Repo:   "org/template",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "org/target2"},
					},
				},
				{
					Name: "Experimental",
					ID:   "experimental",
					Source: config.SourceConfig{
						Repo:   "org/template",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "org/target3"},
					},
				},
			},
			FileLists: []config.FileList{
				{
					ID:    "common",
					Name:  "Common Files",
					Files: []config.FileMapping{{Src: "README.md", Dest: "README.md"}},
				},
			},
			DirectoryLists: []config.DirectoryList{
				{
					ID:          "scripts",
					Name:        "Script Directories",
					Directories: []config.DirectoryMapping{{Src: "scripts/", Dest: "scripts/"}},
				},
			},
		}
	}

	t.Run("NoFilters_ReturnsOriginalConfig", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		result := FilterConfigByGroups(cfg, nil, nil)

		// Should return exact same config pointer when no filters
		assert.Same(t, cfg, result)
		assert.Len(t, result.Groups, 3)
	})

	t.Run("EmptySliceFilters_ReturnsOriginalConfig", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		result := FilterConfigByGroups(cfg, []string{}, []string{})

		// Should return exact same config pointer when empty filters
		assert.Same(t, cfg, result)
		assert.Len(t, result.Groups, 3)
	})

	t.Run("FilterByName_SingleGroup", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		result := FilterConfigByGroups(cfg, []string{"Core Group"}, nil)

		require.Len(t, result.Groups, 1)
		assert.Equal(t, "Core Group", result.Groups[0].Name)
		assert.Equal(t, "core", result.Groups[0].ID)
	})

	t.Run("FilterByID_SingleGroup", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		result := FilterConfigByGroups(cfg, []string{"security"}, nil)

		require.Len(t, result.Groups, 1)
		assert.Equal(t, "Security Group", result.Groups[0].Name)
		assert.Equal(t, "security", result.Groups[0].ID)
	})

	t.Run("FilterByMultipleGroups_MixedNameAndID", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		// Filter by name "Core Group" and ID "security"
		result := FilterConfigByGroups(cfg, []string{"Core Group", "security"}, nil)

		require.Len(t, result.Groups, 2)
		names := []string{result.Groups[0].Name, result.Groups[1].Name}
		assert.Contains(t, names, "Core Group")
		assert.Contains(t, names, "Security Group")
	})

	t.Run("SkipByName_ExcludesGroup", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		result := FilterConfigByGroups(cfg, nil, []string{"Experimental"})

		require.Len(t, result.Groups, 2)
		for _, group := range result.Groups {
			assert.NotEqual(t, "Experimental", group.Name)
			assert.NotEqual(t, "experimental", group.ID)
		}
	})

	t.Run("SkipByID_ExcludesGroup", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		result := FilterConfigByGroups(cfg, nil, []string{"core"})

		require.Len(t, result.Groups, 2)
		for _, group := range result.Groups {
			assert.NotEqual(t, "core", group.ID)
		}
	})

	t.Run("SkipTakesPrecedence_OverInclude", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		// Include all three groups, but skip experimental
		result := FilterConfigByGroups(cfg, []string{"core", "security", "experimental"}, []string{"experimental"})

		require.Len(t, result.Groups, 2)
		ids := []string{result.Groups[0].ID, result.Groups[1].ID}
		assert.Contains(t, ids, "core")
		assert.Contains(t, ids, "security")
		assert.NotContains(t, ids, "experimental")
	})

	t.Run("CombinedFilters_IncludeAndSkip", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		// Include core and security, skip security
		result := FilterConfigByGroups(cfg, []string{"core", "security"}, []string{"security"})

		require.Len(t, result.Groups, 1)
		assert.Equal(t, "core", result.Groups[0].ID)
	})

	t.Run("NoMatches_ReturnsEmptyGroups", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		result := FilterConfigByGroups(cfg, []string{"nonexistent-group"}, nil)

		assert.Empty(t, result.Groups)
		// Config metadata should still be preserved
		assert.Equal(t, cfg.Version, result.Version)
		assert.Equal(t, cfg.Name, result.Name)
		assert.Equal(t, cfg.ID, result.ID)
	})

	t.Run("SkipAllGroups_ReturnsEmptyGroups", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		result := FilterConfigByGroups(cfg, nil, []string{"core", "security", "experimental"})

		assert.Empty(t, result.Groups)
	})

	t.Run("PreservesConfigMetadata", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		result := FilterConfigByGroups(cfg, []string{"core"}, nil)

		// Verify all config fields are preserved
		assert.Equal(t, cfg.Version, result.Version)
		assert.Equal(t, cfg.Name, result.Name)
		assert.Equal(t, cfg.ID, result.ID)
		assert.Equal(t, cfg.FileLists, result.FileLists)
		assert.Equal(t, cfg.DirectoryLists, result.DirectoryLists)
	})

	t.Run("FilterCreatesNewConfig_DoesNotMutateOriginal", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()
		originalGroupCount := len(cfg.Groups)

		result := FilterConfigByGroups(cfg, []string{"core"}, nil)

		// Result should be a new config, not the original
		assert.NotSame(t, cfg, result)
		// Original config should be unchanged
		assert.Len(t, cfg.Groups, originalGroupCount)
		// Result should have filtered groups
		assert.Len(t, result.Groups, 1)
	})

	t.Run("EmptyConfig_NoGroups", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{
			Version: 1,
			Groups:  []config.Group{},
		}

		result := FilterConfigByGroups(cfg, []string{"core"}, nil)

		assert.Empty(t, result.Groups)
	})

	t.Run("CaseSensitiveMatching", func(t *testing.T) {
		t.Parallel()
		cfg := createTestConfig()

		// "CORE" should not match "core" (case-sensitive)
		result := FilterConfigByGroups(cfg, []string{"CORE"}, nil)

		assert.Empty(t, result.Groups)
	})
}
