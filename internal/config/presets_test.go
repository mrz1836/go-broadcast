package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_GetPreset(t *testing.T) {
	t.Run("found by ID", func(t *testing.T) {
		cfg := &Config{
			SettingsPresets: []SettingsPreset{
				{ID: "alpha", Name: "Alpha Preset"},
				{ID: "beta", Name: "Beta Preset"},
				{ID: "gamma", Name: "Gamma Preset"},
			},
		}

		preset := cfg.GetPreset("beta")
		require.NotNil(t, preset)
		assert.Equal(t, "beta", preset.ID)
		assert.Equal(t, "Beta Preset", preset.Name)
	})

	t.Run("returns pointer into slice", func(t *testing.T) {
		cfg := &Config{
			SettingsPresets: []SettingsPreset{
				{ID: "alpha", Name: "Alpha Preset"},
			},
		}

		preset := cfg.GetPreset("alpha")
		require.NotNil(t, preset)

		// Mutating through the pointer should modify the original slice element
		preset.Name = "Modified"
		assert.Equal(t, "Modified", cfg.SettingsPresets[0].Name)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		cfg := &Config{
			SettingsPresets: []SettingsPreset{
				{ID: "alpha", Name: "Alpha Preset"},
			},
		}

		preset := cfg.GetPreset("nonexistent")
		assert.Nil(t, preset)
	})

	t.Run("empty presets list returns nil", func(t *testing.T) {
		cfg := &Config{}

		preset := cfg.GetPreset("anything")
		assert.Nil(t, preset)
	})

	t.Run("returns first match when duplicates exist", func(t *testing.T) {
		cfg := &Config{
			SettingsPresets: []SettingsPreset{
				{ID: "dup", Name: "First"},
				{ID: "dup", Name: "Second"},
			},
		}

		preset := cfg.GetPreset("dup")
		require.NotNil(t, preset)
		assert.Equal(t, "First", preset.Name)
	})
}

func TestDefaultPreset(t *testing.T) {
	preset := DefaultPreset()

	t.Run("ID and metadata", func(t *testing.T) {
		assert.Equal(t, "mvp", preset.ID)
		assert.Equal(t, "MVP", preset.Name)
		assert.Equal(t, "Default preset for new repositories", preset.Description)
	})

	t.Run("feature flags", func(t *testing.T) {
		assert.True(t, preset.HasIssues)
		assert.False(t, preset.HasWiki)
		assert.False(t, preset.HasProjects)
		assert.False(t, preset.HasDiscussions)
	})

	t.Run("merge settings", func(t *testing.T) {
		assert.True(t, preset.AllowSquashMerge)
		assert.False(t, preset.AllowMergeCommit)
		assert.False(t, preset.AllowRebaseMerge)
		assert.True(t, preset.DeleteBranchOnMerge)
		assert.True(t, preset.AllowAutoMerge)
		assert.True(t, preset.AllowUpdateBranch)
	})

	t.Run("squash merge commit format", func(t *testing.T) {
		assert.Equal(t, "PR_TITLE", preset.SquashMergeCommitTitle)
		assert.Equal(t, "COMMIT_MESSAGES", preset.SquashMergeCommitMessage)
	})

	t.Run("twelve boolean and string settings total", func(t *testing.T) {
		// Count boolean settings: HasIssues, HasWiki, HasProjects, HasDiscussions,
		// AllowSquashMerge, AllowMergeCommit, AllowRebaseMerge, DeleteBranchOnMerge,
		// AllowAutoMerge, AllowUpdateBranch = 10 booleans
		// Plus SquashMergeCommitTitle, SquashMergeCommitMessage = 2 strings
		// Total: 12 settings fields (not counting ID/Name/Description/Rulesets/Labels)

		trueCount := 0
		falseCount := 0
		for _, v := range []bool{
			preset.HasIssues, preset.HasWiki, preset.HasProjects, preset.HasDiscussions,
			preset.AllowSquashMerge, preset.AllowMergeCommit, preset.AllowRebaseMerge,
			preset.DeleteBranchOnMerge, preset.AllowAutoMerge, preset.AllowUpdateBranch,
		} {
			if v {
				trueCount++
			} else {
				falseCount++
			}
		}
		assert.Equal(t, 5, trueCount, "expected 5 true boolean settings")
		assert.Equal(t, 5, falseCount, "expected 5 false boolean settings")
		assert.NotEmpty(t, preset.SquashMergeCommitTitle)
		assert.NotEmpty(t, preset.SquashMergeCommitMessage)
	})

	t.Run("rulesets", func(t *testing.T) {
		require.Len(t, preset.Rulesets, 2)

		branch := preset.Rulesets[0]
		assert.Equal(t, "branch-protection", branch.Name)
		assert.Equal(t, "branch", branch.Target)
		assert.Equal(t, "active", branch.Enforcement)
		assert.Equal(t, []string{"~DEFAULT_BRANCH"}, branch.Include)
		assert.Equal(t, []string{"deletion", "pull_request"}, branch.Rules)

		tag := preset.Rulesets[1]
		assert.Equal(t, "tag-protection", tag.Name)
		assert.Equal(t, "tag", tag.Target)
		assert.Equal(t, "active", tag.Enforcement)
		assert.Equal(t, []string{"~ALL"}, tag.Include)
		assert.Equal(t, []string{"deletion", "update"}, tag.Rules)
	})

	t.Run("labels", func(t *testing.T) {
		require.Len(t, preset.Labels, 8)

		expectedLabels := []struct {
			name        string
			color       string
			description string
		}{
			{"bug", "d73a4a", "Something isn't working"},
			{"enhancement", "a2eeef", "New feature or request"},
			{"documentation", "0075ca", "Documentation improvements"},
			{"good first issue", "7057ff", "Good for newcomers"},
			{"help wanted", "008672", "Extra attention needed"},
			{"priority: high", "b60205", "High priority"},
			{"priority: low", "c5def5", "Low priority"},
			{"wontfix", "ffffff", "Won't be addressed"},
		}

		for i, exp := range expectedLabels {
			assert.Equal(t, exp.name, preset.Labels[i].Name, "label %d name", i)
			assert.Equal(t, exp.color, preset.Labels[i].Color, "label %d color", i)
			assert.Equal(t, exp.description, preset.Labels[i].Description, "label %d description", i)
		}
	})

	t.Run("returns independent copies", func(t *testing.T) {
		p1 := DefaultPreset()
		p2 := DefaultPreset()

		p1.ID = "modified"
		p1.Labels[0].Name = "changed"
		p1.Rulesets[0].Name = "changed"

		assert.Equal(t, "mvp", p2.ID)
		assert.Equal(t, "bug", p2.Labels[0].Name)
		assert.Equal(t, "branch-protection", p2.Rulesets[0].Name)
	})
}

func TestLoadFromReader_SettingsPresets(t *testing.T) {
	t.Run("parse single preset from YAML", func(t *testing.T) {
		input := `
version: 1
settings_presets:
  - id: "go-lib"
    name: "Go Library"
    description: "Preset for Go libraries"
    has_issues: true
    has_wiki: false
    has_projects: false
    has_discussions: true
    allow_squash_merge: true
    allow_merge_commit: false
    allow_rebase_merge: true
    delete_branch_on_merge: true
    allow_auto_merge: true
    allow_update_branch: true
    squash_merge_commit_title: "PR_TITLE"
    squash_merge_commit_message: "PR_BODY"
    rulesets:
      - name: "main-protection"
        target: "branch"
        enforcement: "active"
        include: ["~DEFAULT_BRANCH"]
        rules: ["deletion", "pull_request"]
    labels:
      - name: "bug"
        color: "d73a4a"
        description: "Something isn't working"
      - name: "feature"
        color: "a2eeef"
        description: "New feature"
groups:
  - name: "test"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Len(t, cfg.SettingsPresets, 1)

		preset := cfg.SettingsPresets[0]
		assert.Equal(t, "go-lib", preset.ID)
		assert.Equal(t, "Go Library", preset.Name)
		assert.Equal(t, "Preset for Go libraries", preset.Description)

		assert.True(t, preset.HasIssues)
		assert.False(t, preset.HasWiki)
		assert.False(t, preset.HasProjects)
		assert.True(t, preset.HasDiscussions)

		assert.True(t, preset.AllowSquashMerge)
		assert.False(t, preset.AllowMergeCommit)
		assert.True(t, preset.AllowRebaseMerge)
		assert.True(t, preset.DeleteBranchOnMerge)
		assert.True(t, preset.AllowAutoMerge)
		assert.True(t, preset.AllowUpdateBranch)

		assert.Equal(t, "PR_TITLE", preset.SquashMergeCommitTitle)
		assert.Equal(t, "PR_BODY", preset.SquashMergeCommitMessage)

		require.Len(t, preset.Rulesets, 1)
		assert.Equal(t, "main-protection", preset.Rulesets[0].Name)
		assert.Equal(t, "branch", preset.Rulesets[0].Target)
		assert.Equal(t, "active", preset.Rulesets[0].Enforcement)
		assert.Equal(t, []string{"~DEFAULT_BRANCH"}, preset.Rulesets[0].Include)
		assert.Equal(t, []string{"deletion", "pull_request"}, preset.Rulesets[0].Rules)

		require.Len(t, preset.Labels, 2)
		assert.Equal(t, "bug", preset.Labels[0].Name)
		assert.Equal(t, "d73a4a", preset.Labels[0].Color)
		assert.Equal(t, "feature", preset.Labels[1].Name)
		assert.Equal(t, "a2eeef", preset.Labels[1].Color)
	})

	t.Run("parse multiple presets from YAML", func(t *testing.T) {
		input := `
version: 1
settings_presets:
  - id: "mvp"
    name: "MVP"
    has_issues: true
    has_wiki: false
    has_projects: false
    has_discussions: false
    allow_squash_merge: true
    allow_merge_commit: false
    allow_rebase_merge: false
    delete_branch_on_merge: true
    allow_auto_merge: false
    allow_update_branch: true
  - id: "full"
    name: "Full Featured"
    has_issues: true
    has_wiki: true
    has_projects: true
    has_discussions: true
    allow_squash_merge: true
    allow_merge_commit: true
    allow_rebase_merge: true
    delete_branch_on_merge: false
    allow_auto_merge: true
    allow_update_branch: true
groups:
  - name: "test"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Len(t, cfg.SettingsPresets, 2)

		assert.Equal(t, "mvp", cfg.SettingsPresets[0].ID)
		assert.Equal(t, "MVP", cfg.SettingsPresets[0].Name)
		assert.False(t, cfg.SettingsPresets[0].AllowAutoMerge)

		assert.Equal(t, "full", cfg.SettingsPresets[1].ID)
		assert.Equal(t, "Full Featured", cfg.SettingsPresets[1].Name)
		assert.True(t, cfg.SettingsPresets[1].AllowAutoMerge)
	})

	t.Run("GetPreset works on parsed config", func(t *testing.T) {
		input := `
version: 1
settings_presets:
  - id: "preset-a"
    name: "Preset A"
    has_issues: true
    has_wiki: false
    has_projects: false
    has_discussions: false
    allow_squash_merge: true
    allow_merge_commit: false
    allow_rebase_merge: false
    delete_branch_on_merge: true
    allow_auto_merge: false
    allow_update_branch: false
  - id: "preset-b"
    name: "Preset B"
    has_issues: false
    has_wiki: true
    has_projects: true
    has_discussions: true
    allow_squash_merge: false
    allow_merge_commit: true
    allow_rebase_merge: true
    delete_branch_on_merge: false
    allow_auto_merge: true
    allow_update_branch: true
groups:
  - name: "test"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.NoError(t, err)

		presetA := cfg.GetPreset("preset-a")
		require.NotNil(t, presetA)
		assert.Equal(t, "Preset A", presetA.Name)
		assert.True(t, presetA.HasIssues)
		assert.False(t, presetA.HasWiki)

		presetB := cfg.GetPreset("preset-b")
		require.NotNil(t, presetB)
		assert.Equal(t, "Preset B", presetB.Name)
		assert.False(t, presetB.HasIssues)
		assert.True(t, presetB.HasWiki)

		assert.Nil(t, cfg.GetPreset("nonexistent"))
	})

	t.Run("preset with empty rulesets and labels", func(t *testing.T) {
		input := `
version: 1
settings_presets:
  - id: "minimal"
    name: "Minimal"
    has_issues: false
    has_wiki: false
    has_projects: false
    has_discussions: false
    allow_squash_merge: false
    allow_merge_commit: false
    allow_rebase_merge: false
    delete_branch_on_merge: false
    allow_auto_merge: false
    allow_update_branch: false
groups:
  - name: "test"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.NoError(t, err)
		require.Len(t, cfg.SettingsPresets, 1)

		preset := cfg.SettingsPresets[0]
		assert.Equal(t, "minimal", preset.ID)
		assert.Empty(t, preset.Rulesets)
		assert.Empty(t, preset.Labels)
		assert.Empty(t, preset.Description)
		assert.Empty(t, preset.SquashMergeCommitTitle)
		assert.Empty(t, preset.SquashMergeCommitMessage)
	})

	t.Run("no settings_presets is valid", func(t *testing.T) {
		input := `
version: 1
groups:
  - name: "test"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.NoError(t, err)
		assert.Empty(t, cfg.SettingsPresets)
	})

	t.Run("preset with multiple rulesets and labels", func(t *testing.T) {
		input := `
version: 1
settings_presets:
  - id: "complex"
    name: "Complex"
    has_issues: true
    has_wiki: true
    has_projects: true
    has_discussions: true
    allow_squash_merge: true
    allow_merge_commit: true
    allow_rebase_merge: true
    delete_branch_on_merge: true
    allow_auto_merge: true
    allow_update_branch: true
    rulesets:
      - name: "branch-protection"
        target: "branch"
        enforcement: "active"
        include: ["~DEFAULT_BRANCH"]
        exclude: ["refs/heads/release/*"]
        rules: ["deletion", "pull_request"]
      - name: "tag-protection"
        target: "tag"
        enforcement: "evaluate"
        include: ["~ALL"]
        rules: ["deletion", "update"]
      - name: "release-branch"
        target: "branch"
        enforcement: "disabled"
        include: ["refs/heads/release/*"]
        rules: ["pull_request"]
    labels:
      - name: "bug"
        color: "d73a4a"
        description: "Bug report"
      - name: "feature"
        color: "a2eeef"
      - name: "urgent"
        color: "b60205"
        description: "Needs immediate attention"
groups:
  - name: "test"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.NoError(t, err)
		require.Len(t, cfg.SettingsPresets, 1)

		preset := cfg.SettingsPresets[0]
		require.Len(t, preset.Rulesets, 3)

		assert.Equal(t, "branch-protection", preset.Rulesets[0].Name)
		assert.Equal(t, []string{"refs/heads/release/*"}, preset.Rulesets[0].Exclude)

		assert.Equal(t, "tag-protection", preset.Rulesets[1].Name)
		assert.Equal(t, "evaluate", preset.Rulesets[1].Enforcement)

		assert.Equal(t, "release-branch", preset.Rulesets[2].Name)
		assert.Equal(t, "disabled", preset.Rulesets[2].Enforcement)

		require.Len(t, preset.Labels, 3)
		assert.Equal(t, "bug", preset.Labels[0].Name)
		assert.Equal(t, "feature", preset.Labels[1].Name)
		assert.Empty(t, preset.Labels[1].Description, "description is optional")
		assert.Equal(t, "urgent", preset.Labels[2].Name)
	})
}

func TestLoadFromReader_SettingsPresetsUnknownFields(t *testing.T) {
	t.Run("unknown field in settings_preset is rejected", func(t *testing.T) {
		input := `
version: 1
settings_presets:
  - id: "test"
    name: "Test"
    has_issues: true
    has_wiki: false
    has_projects: false
    has_discussions: false
    allow_squash_merge: true
    allow_merge_commit: false
    allow_rebase_merge: false
    delete_branch_on_merge: true
    allow_auto_merge: false
    allow_update_branch: false
    unknown_setting: true
groups:
  - name: "test"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse YAML")
		assert.Nil(t, cfg)
	})

	t.Run("unknown field in ruleset within preset is rejected", func(t *testing.T) {
		input := `
version: 1
settings_presets:
  - id: "test"
    name: "Test"
    has_issues: false
    has_wiki: false
    has_projects: false
    has_discussions: false
    allow_squash_merge: false
    allow_merge_commit: false
    allow_rebase_merge: false
    delete_branch_on_merge: false
    allow_auto_merge: false
    allow_update_branch: false
    rulesets:
      - name: "branch-protection"
        target: "branch"
        enforcement: "active"
        include: ["~DEFAULT_BRANCH"]
        rules: ["deletion"]
        bogus_field: "invalid"
groups:
  - name: "test"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse YAML")
		assert.Nil(t, cfg)
	})

	t.Run("unknown field in label within preset is rejected", func(t *testing.T) {
		input := `
version: 1
settings_presets:
  - id: "test"
    name: "Test"
    has_issues: false
    has_wiki: false
    has_projects: false
    has_discussions: false
    allow_squash_merge: false
    allow_merge_commit: false
    allow_rebase_merge: false
    delete_branch_on_merge: false
    allow_auto_merge: false
    allow_update_branch: false
    labels:
      - name: "bug"
        color: "d73a4a"
        description: "Bug"
        priority: 1
groups:
  - name: "test"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse YAML")
		assert.Nil(t, cfg)
	})

	t.Run("misspelled preset field is rejected", func(t *testing.T) {
		input := `
version: 1
settings_presets:
  - id: "test"
    name: "Test"
    has_issues: true
    has_wiki: false
    has_projects: false
    has_discussions: false
    allow_squash_merge: true
    allow_merge_commit: false
    allow_rebase_merge: false
    delete_branch_on_merge: true
    allow_auto_merge: false
    allow_update_branch: false
    squash_merge_commit_titl: "PR_TITLE"
groups:
  - name: "test"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse YAML")
		assert.Nil(t, cfg)
	})
}
