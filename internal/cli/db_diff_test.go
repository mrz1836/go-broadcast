package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/config"
)

func TestStringSlicesEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "both empty",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "equal slices",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "different lengths",
			a:        []string{"a", "b"},
			b:        []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "same length different values",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "d"},
			expected: false,
		},
		{
			name:     "same values different order",
			a:        []string{"a", "b", "c"},
			b:        []string{"c", "b", "a"},
			expected: false,
		},
		{
			name:     "one nil one empty",
			a:        nil,
			b:        []string{},
			expected: true,
		},
		{
			name:     "single element equal",
			a:        []string{"hello"},
			b:        []string{"hello"},
			expected: true,
		},
		{
			name:     "single element different",
			a:        []string{"hello"},
			b:        []string{"world"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := stringSlicesEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareConfigs(t *testing.T) {
	t.Parallel()

	t.Run("identical configs no diffs", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			Name:    "test",
			Version: 1,
			Groups: []config.Group{
				{ID: "group-1", Name: "Group 1"},
			},
		}

		diffs := compareConfigs(cfg, cfg, false)
		assert.Empty(t, diffs)
	})

	t.Run("different names", func(t *testing.T) {
		t.Parallel()

		yaml := &config.Config{Name: "yaml-config", Version: 1}
		dbCfg := &config.Config{Name: "db-config", Version: 1}

		diffs := compareConfigs(yaml, dbCfg, false)
		assert.NotEmpty(t, diffs)
		assert.Contains(t, diffs[0], "Config name differs")
	})

	t.Run("different versions", func(t *testing.T) {
		t.Parallel()

		yaml := &config.Config{Name: "cfg", Version: 1}
		dbCfg := &config.Config{Name: "cfg", Version: 2}

		diffs := compareConfigs(yaml, dbCfg, false)
		assert.NotEmpty(t, diffs)
		assert.Contains(t, diffs[0], "Config version differs")
	})

	t.Run("different group counts", func(t *testing.T) {
		t.Parallel()

		yaml := &config.Config{
			Groups: []config.Group{{ID: "g1"}, {ID: "g2"}},
		}
		dbCfg := &config.Config{
			Groups: []config.Group{{ID: "g1"}},
		}

		diffs := compareConfigs(yaml, dbCfg, false)
		hasDiff := false
		for _, d := range diffs {
			if strings.Contains(d, "Group count") {
				hasDiff = true
			}
		}
		assert.True(t, hasDiff)
	})

	t.Run("missing group in DB", func(t *testing.T) {
		t.Parallel()

		yaml := &config.Config{
			Groups: []config.Group{{ID: "g1"}, {ID: "g2"}},
		}
		dbCfg := &config.Config{
			Groups: []config.Group{{ID: "g1"}},
		}

		diffs := compareConfigs(yaml, dbCfg, false)
		foundMissing := false
		for _, d := range diffs {
			if strings.Contains(d, "Group missing in DB") {
				foundMissing = true
			}
		}
		assert.True(t, foundMissing)
	})

	t.Run("extra group in DB", func(t *testing.T) {
		t.Parallel()

		yaml := &config.Config{
			Groups: []config.Group{{ID: "g1"}},
		}
		dbCfg := &config.Config{
			Groups: []config.Group{{ID: "g1"}, {ID: "g3"}},
		}

		diffs := compareConfigs(yaml, dbCfg, false)
		foundExtra := false
		for _, d := range diffs {
			if strings.Contains(d, "Extra group in DB") {
				foundExtra = true
			}
		}
		assert.True(t, foundExtra)
	})

	t.Run("detail mode compares groups", func(t *testing.T) {
		t.Parallel()

		yaml := &config.Config{
			Groups: []config.Group{{ID: "g1", Name: "YAML Name", Source: config.SourceConfig{Repo: "org/repo"}}},
		}
		dbCfg := &config.Config{
			Groups: []config.Group{{ID: "g1", Name: "DB Name", Source: config.SourceConfig{Repo: "org/repo"}}},
		}

		diffs := compareConfigs(yaml, dbCfg, true)
		foundNameDiff := false
		for _, d := range diffs {
			if strings.Contains(d, ".name:") {
				foundNameDiff = true
			}
		}
		assert.True(t, foundNameDiff)
	})

	t.Run("file list differences", func(t *testing.T) {
		t.Parallel()

		yaml := &config.Config{
			FileLists: []config.FileList{{ID: "fl1"}, {ID: "fl2"}},
		}
		dbCfg := &config.Config{
			FileLists: []config.FileList{{ID: "fl1"}},
		}

		diffs := compareConfigs(yaml, dbCfg, false)
		foundCount := false
		foundMissing := false
		for _, d := range diffs {
			if strings.Contains(d, "FileList count") {
				foundCount = true
			}
			if strings.Contains(d, "FileList missing in DB") {
				foundMissing = true
			}
		}
		assert.True(t, foundCount)
		assert.True(t, foundMissing)
	})

	t.Run("directory list count differs", func(t *testing.T) {
		t.Parallel()

		yaml := &config.Config{}
		dbCfg := &config.Config{
			DirectoryLists: []config.DirectoryList{{ID: "dl1"}},
		}

		diffs := compareConfigs(yaml, dbCfg, false)
		foundDiff := false
		for _, d := range diffs {
			if strings.Contains(d, "DirectoryList count") {
				foundDiff = true
			}
		}
		assert.True(t, foundDiff)
	})

	t.Run("extra file list in DB", func(t *testing.T) {
		t.Parallel()

		yaml := &config.Config{
			FileLists: []config.FileList{{ID: "fl1"}},
		}
		dbCfg := &config.Config{
			FileLists: []config.FileList{{ID: "fl1"}, {ID: "fl-extra"}},
		}

		diffs := compareConfigs(yaml, dbCfg, false)
		found := false
		for _, d := range diffs {
			if strings.Contains(d, "Extra FileList in DB") {
				found = true
			}
		}
		assert.True(t, found)
	})
}

func TestCompareGroups(t *testing.T) {
	t.Parallel()

	t.Run("identical groups no diffs", func(t *testing.T) {
		t.Parallel()

		enabled := true
		g := config.Group{
			Name:        "Test Group",
			Description: "Test",
			Priority:    1,
			Enabled:     &enabled,
			Source:      config.SourceConfig{Repo: "org/repo"},
			Targets:     []config.TargetConfig{{Repo: "org/target1"}},
			DependsOn:   []string{"other"},
		}

		diffs := compareGroups("test-id", g, g)
		assert.Empty(t, diffs)
	})

	t.Run("name differs", func(t *testing.T) {
		t.Parallel()

		yaml := config.Group{Name: "YAML Name"}
		dbG := config.Group{Name: "DB Name"}

		diffs := compareGroups("test-id", yaml, dbG)
		assert.NotEmpty(t, diffs)
		assert.Contains(t, diffs[0], ".name:")
	})

	t.Run("description differs", func(t *testing.T) {
		t.Parallel()

		yaml := config.Group{Description: "Desc A"}
		dbG := config.Group{Description: "Desc B"}

		diffs := compareGroups("test-id", yaml, dbG)
		assert.NotEmpty(t, diffs)
		assert.Contains(t, diffs[0], ".description:")
	})

	t.Run("priority differs", func(t *testing.T) {
		t.Parallel()

		yaml := config.Group{Priority: 1}
		dbG := config.Group{Priority: 2}

		diffs := compareGroups("test-id", yaml, dbG)
		assert.NotEmpty(t, diffs)
		assert.Contains(t, diffs[0], ".priority:")
	})

	t.Run("enabled differs", func(t *testing.T) {
		t.Parallel()

		enabledTrue := true
		enabledFalse := false
		yaml := config.Group{Enabled: &enabledTrue}
		dbG := config.Group{Enabled: &enabledFalse}

		diffs := compareGroups("test-id", yaml, dbG)
		assert.NotEmpty(t, diffs)
		assert.Contains(t, diffs[0], ".enabled:")
	})

	t.Run("enabled nil vs true defaults to no diff", func(t *testing.T) {
		t.Parallel()

		enabled := true
		yaml := config.Group{Enabled: nil}
		dbG := config.Group{Enabled: &enabled}

		diffs := compareGroups("test-id", yaml, dbG)
		enabledDiffFound := false
		for _, d := range diffs {
			if strings.Contains(d, ".enabled:") {
				enabledDiffFound = true
			}
		}
		assert.False(t, enabledDiffFound)
	})

	t.Run("source repo differs", func(t *testing.T) {
		t.Parallel()

		yaml := config.Group{Source: config.SourceConfig{Repo: "org/repoA"}}
		dbG := config.Group{Source: config.SourceConfig{Repo: "org/repoB"}}

		diffs := compareGroups("test-id", yaml, dbG)
		assert.NotEmpty(t, diffs)
		assert.Contains(t, diffs[0], ".source.repo:")
	})

	t.Run("target count differs", func(t *testing.T) {
		t.Parallel()

		yaml := config.Group{Targets: []config.TargetConfig{{Repo: "a"}, {Repo: "b"}}}
		dbG := config.Group{Targets: []config.TargetConfig{{Repo: "a"}}}

		diffs := compareGroups("test-id", yaml, dbG)
		found := false
		for _, d := range diffs {
			if strings.Contains(d, "target count") {
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("depends_on differs", func(t *testing.T) {
		t.Parallel()

		yaml := config.Group{DependsOn: []string{"dep-a"}}
		dbG := config.Group{DependsOn: []string{"dep-b"}}

		diffs := compareGroups("test-id", yaml, dbG)
		found := false
		for _, d := range diffs {
			if strings.Contains(d, ".depends_on:") {
				found = true
			}
		}
		assert.True(t, found)
	})
}
