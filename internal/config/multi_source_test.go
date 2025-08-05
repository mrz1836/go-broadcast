package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiSourceConfigStructure tests multi-source configuration structure
func TestMultiSourceConfigStructure(t *testing.T) {
	t.Run("multi-source config with multiple mappings", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Mappings: []SourceMapping{
				{
					Source: SourceConfig{
						Repo:   "org/template-repo",
						Branch: "main",
						ID:     "template",
					},
					Targets: []TargetConfig{
						{
							Repo: "org/target1",
							Files: []FileMapping{
								{Src: "file1.txt", Dest: "file1.txt"},
							},
						},
						{
							Repo: "org/target2",
							Files: []FileMapping{
								{Src: "file2.txt", Dest: "file2.txt"},
							},
						},
					},
				},
			},
		}

		// Should have one mapping
		assert.Len(t, config.Mappings, 1)

		// Source should be properly structured
		assert.Equal(t, "org/template-repo", config.Mappings[0].Source.Repo)
		assert.Equal(t, "main", config.Mappings[0].Source.Branch)
		assert.Equal(t, "template", config.Mappings[0].Source.ID)

		// Targets should be properly structured
		assert.Len(t, config.Mappings[0].Targets, 2)
		assert.Equal(t, "org/target1", config.Mappings[0].Targets[0].Repo)
		assert.Equal(t, "org/target2", config.Mappings[0].Targets[1].Repo)
	})

	t.Run("multi-source config with empty mappings", func(t *testing.T) {
		config := &Config{
			Version:  1,
			Mappings: []SourceMapping{},
		}

		assert.Empty(t, config.Mappings)
	})

	t.Run("multi-source config with multiple sources", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Mappings: []SourceMapping{
				{
					Source: SourceConfig{
						Repo:   "org/source1",
						Branch: "main",
						ID:     "source1",
					},
					Targets: []TargetConfig{
						{Repo: "org/target1"},
					},
				},
				{
					Source: SourceConfig{
						Repo:   "org/source2",
						Branch: "develop",
						ID:     "source2",
					},
					Targets: []TargetConfig{
						{Repo: "org/target2"},
					},
				},
			},
		}

		// Should have two mappings
		assert.Len(t, config.Mappings, 2)
		assert.Equal(t, "org/source1", config.Mappings[0].Source.Repo)
		assert.Equal(t, "org/source2", config.Mappings[1].Source.Repo)
	})
}

func TestGetAllTargets(t *testing.T) {
	config := &Config{
		Mappings: []SourceMapping{
			{
				Source: SourceConfig{Repo: "org/source1"},
				Targets: []TargetConfig{
					{Repo: "org/target1"},
					{Repo: "org/target2"},
				},
			},
			{
				Source: SourceConfig{Repo: "org/source2"},
				Targets: []TargetConfig{
					{Repo: "org/target2"}, // Duplicate
					{Repo: "org/target3"},
				},
			},
		},
	}

	targets := config.GetAllTargets()

	// Should have 3 unique targets
	assert.Len(t, targets, 3)
	assert.True(t, targets["org/target1"])
	assert.True(t, targets["org/target2"])
	assert.True(t, targets["org/target3"])
}

func TestGetTargetMappings(t *testing.T) {
	config := &Config{
		Mappings: []SourceMapping{
			{
				Source: SourceConfig{
					Repo:   "org/source1",
					Branch: "main",
					ID:     "src1",
				},
				Targets: []TargetConfig{
					{
						Repo: "org/target1",
						Files: []FileMapping{
							{Src: "file1.txt", Dest: "file1.txt"},
						},
					},
					{
						Repo: "org/target2",
						Files: []FileMapping{
							{Src: "file2.txt", Dest: "file2.txt"},
						},
					},
				},
				Defaults: &DefaultConfig{
					BranchPrefix: "sync/source1",
				},
			},
			{
				Source: SourceConfig{
					Repo:   "org/source2",
					Branch: "main",
					ID:     "src2",
				},
				Targets: []TargetConfig{
					{
						Repo: "org/target1", // Same target, different source
						Files: []FileMapping{
							{Src: "other.txt", Dest: "other.txt"},
						},
					},
				},
			},
		},
	}

	t.Run("target with multiple sources", func(t *testing.T) {
		pairs := config.GetTargetMappings("org/target1")

		assert.Len(t, pairs, 2)

		// Should have both sources
		sources := make(map[string]bool)
		for _, pair := range pairs {
			sources[pair.Source.Repo] = true
		}
		assert.True(t, sources["org/source1"])
		assert.True(t, sources["org/source2"])

		// Check defaults are preserved
		for _, pair := range pairs {
			if pair.Source.Repo == "org/source1" {
				assert.NotNil(t, pair.Defaults)
				assert.Equal(t, "sync/source1", pair.Defaults.BranchPrefix)
			}
		}
	})

	t.Run("target with single source", func(t *testing.T) {
		pairs := config.GetTargetMappings("org/target2")

		assert.Len(t, pairs, 1)
		assert.Equal(t, "org/source1", pairs[0].Source.Repo)
		assert.Equal(t, "org/target2", pairs[0].Target.Repo)
	})

	t.Run("non-existent target", func(t *testing.T) {
		pairs := config.GetTargetMappings("org/nonexistent")
		assert.Empty(t, pairs)
	})
}

func TestMultiSourceConfigParsing(t *testing.T) {
	yaml := `
version: 1
mappings:
  - source:
      repo: "org/ci-templates"
      branch: "main"
      id: "ci"
    targets:
      - repo: "org/service-a"
        files:
          - src: "workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
      - repo: "org/service-b"
        files:
          - src: "workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
  - source:
      repo: "org/security-templates"
      branch: "main"
      id: "security"
    targets:
      - repo: "org/service-a"
        files:
          - src: "policies/security.yml"
            dest: "security/policies.yml"
    defaults:
      branch_prefix: "sync/security"
      pr_labels: ["security", "automated"]
global:
  pr_labels: ["automated-sync"]
  pr_assignees: ["platform-team"]
defaults:
  branch_prefix: "chore/sync"
conflict_resolution:
  strategy: "priority"
  priority: ["security", "ci"]
`

	config, err := LoadFromReader(strings.NewReader(yaml))
	require.NoError(t, err)

	// Should be version 1
	assert.Equal(t, 1, config.Version)
	assert.Len(t, config.Mappings, 2)

	// Version should be 1
	assert.Equal(t, 1, config.Version)

	// Check first mapping
	assert.Equal(t, "org/ci-templates", config.Mappings[0].Source.Repo)
	assert.Equal(t, "main", config.Mappings[0].Source.Branch)
	assert.Equal(t, "ci", config.Mappings[0].Source.ID)
	assert.Len(t, config.Mappings[0].Targets, 2)

	// Check second mapping with defaults
	assert.Equal(t, "org/security-templates", config.Mappings[1].Source.Repo)
	assert.NotNil(t, config.Mappings[1].Defaults)
	assert.Equal(t, "sync/security", config.Mappings[1].Defaults.BranchPrefix)
	assert.Equal(t, []string{"security", "automated"}, config.Mappings[1].Defaults.PRLabels)

	// Check global config
	assert.Equal(t, []string{"automated-sync"}, config.Global.PRLabels)
	assert.Equal(t, []string{"platform-team"}, config.Global.PRAssignees)

	// Check defaults
	assert.Equal(t, "chore/sync", config.Defaults.BranchPrefix)

	// Check conflict resolution
	assert.NotNil(t, config.ConflictResolution)
	assert.Equal(t, "priority", config.ConflictResolution.Strategy)
	assert.Equal(t, []string{"security", "ci"}, config.ConflictResolution.Priority)
}

func TestMultiSourceConfigWithDefaults(t *testing.T) {
	yaml := `
version: 1
mappings:
  - source:
      repo: "org/template-repo"
      branch: "master"
      id: "template"
    targets:
      - repo: "org/target1"
        files:
          - src: "file1.txt"
            dest: "file1.txt"
      - repo: "org/target2"
        directories:
          - src: ".github"
            dest: ".github"
            exclude: ["*.md"]
global:
  pr_labels: ["automated"]
defaults:
  branch_prefix: "sync/files"
`

	config, err := LoadFromReader(strings.NewReader(yaml))
	require.NoError(t, err)

	// Should be version 1 with multi-source format
	assert.Len(t, config.Mappings, 1)
	assert.Equal(t, "org/template-repo", config.Mappings[0].Source.Repo)
	assert.Equal(t, "master", config.Mappings[0].Source.Branch)
	assert.Equal(t, "template", config.Mappings[0].Source.ID)
	assert.Len(t, config.Mappings[0].Targets, 2)

	// Global and defaults should be preserved
	assert.Equal(t, []string{"automated"}, config.Global.PRLabels)
	assert.Equal(t, "sync/files", config.Defaults.BranchPrefix)
}
