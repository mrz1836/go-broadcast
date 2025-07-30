package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigStructCreation tests basic struct creation and field access
func TestConfigStructCreation(t *testing.T) {
	t.Run("CreateConfig", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Source: SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Defaults: DefaultConfig{
				BranchPrefix: "sync/",
				PRLabels:     []string{"automated"},
			},
			Targets: []TargetConfig{
				{
					Repo: "org/service",
					Files: []FileMapping{
						{
							Src:  "file.txt",
							Dest: "dest.txt",
						},
					},
					Transform: Transform{
						RepoName: true,
						Variables: map[string]string{
							"KEY": "value",
						},
					},
				},
			},
		}

		require.NotNil(t, config)
		assert.Equal(t, 1, config.Version)
		assert.Equal(t, "org/template", config.Source.Repo)
		assert.Equal(t, "main", config.Source.Branch)
		assert.Equal(t, "sync/", config.Defaults.BranchPrefix)
		assert.Len(t, config.Defaults.PRLabels, 1)
		assert.Len(t, config.Targets, 1)
		assert.True(t, config.Targets[0].Transform.RepoName)
		assert.Equal(t, "value", config.Targets[0].Transform.Variables["KEY"])
	})
}

// TestSourceConfigDefaults tests SourceConfig zero values
func TestSourceConfigDefaults(t *testing.T) {
	source := SourceConfig{}

	assert.Empty(t, source.Repo)
	assert.Empty(t, source.Branch)
}

// TestDefaultConfigDefaults tests DefaultConfig zero values
func TestDefaultConfigDefaults(t *testing.T) {
	defaults := DefaultConfig{}

	assert.Empty(t, defaults.BranchPrefix)
	assert.Nil(t, defaults.PRLabels)
}

// TestTargetConfigDefaults tests TargetConfig zero values
func TestTargetConfigDefaults(t *testing.T) {
	target := TargetConfig{}

	assert.Empty(t, target.Repo)
	assert.Nil(t, target.Files)
	assert.False(t, target.Transform.RepoName)
	assert.Nil(t, target.Transform.Variables)
	assert.Nil(t, target.PRLabels)
	assert.Nil(t, target.PRAssignees)
	assert.Nil(t, target.PRReviewers)
	assert.Nil(t, target.PRTeamReviewers)
}

// TestFileMappingDefaults tests FileMapping zero values
func TestFileMappingDefaults(t *testing.T) {
	file := FileMapping{}

	assert.Empty(t, file.Src)
	assert.Empty(t, file.Dest)
}

// TestTransformDefaults tests Transform zero values
func TestTransformDefaults(t *testing.T) {
	transform := Transform{}

	assert.False(t, transform.RepoName)
	assert.Nil(t, transform.Variables)
}

// TestConfigWithEmptySlices tests behavior with empty slices
func TestConfigWithEmptySlices(t *testing.T) {
	config := &Config{
		Version: 1,
		Source: SourceConfig{
			Repo:   "org/repo",
			Branch: "main",
		},
		Defaults: DefaultConfig{
			BranchPrefix: "prefix",
			PRLabels:     []string{}, // Empty slice
		},
		Targets: []TargetConfig{}, // Empty targets
	}

	require.NotNil(t, config)
	assert.Empty(t, config.Defaults.PRLabels)
	assert.Empty(t, config.Targets)
}

// TestTargetWithEmptyFileList tests target with empty file list
func TestTargetWithEmptyFileList(t *testing.T) {
	target := TargetConfig{
		Repo:  "org/repo",
		Files: []FileMapping{}, // Empty file list
	}

	require.NotNil(t, target)
	assert.Empty(t, target.Files)
}

// TestTransformWithEmptyVariables tests transform with empty variables
func TestTransformWithEmptyVariables(t *testing.T) {
	transform := Transform{
		RepoName:  true,
		Variables: map[string]string{}, // Empty map
	}

	require.NotNil(t, transform)
	assert.True(t, transform.RepoName)
	assert.Empty(t, transform.Variables)
}

// TestConfigFieldModification tests that struct fields can be modified
func TestConfigFieldModification(t *testing.T) {
	config := &Config{
		Version: 1,
	}

	// Modify fields
	config.Source.Repo = "new/repo"
	config.Source.Branch = "develop"
	config.Defaults.BranchPrefix = "feature/"
	config.Defaults.PRLabels = append(config.Defaults.PRLabels, "label1", "label2")

	assert.Equal(t, "new/repo", config.Source.Repo)
	assert.Equal(t, "develop", config.Source.Branch)
	assert.Equal(t, "feature/", config.Defaults.BranchPrefix)
	assert.Len(t, config.Defaults.PRLabels, 2)
}

// TestTargetConfigAppend tests appending to targets slice
func TestTargetConfigAppend(t *testing.T) {
	config := &Config{
		Version: 1,
		Targets: []TargetConfig{},
	}

	// Append targets
	config.Targets = append(config.Targets, TargetConfig{
		Repo: "org/service1",
		Files: []FileMapping{
			{Src: "a.txt", Dest: "b.txt"},
		},
	})

	config.Targets = append(config.Targets, TargetConfig{
		Repo: "org/service2",
		Files: []FileMapping{
			{Src: "c.txt", Dest: "d.txt"},
		},
	})

	assert.Len(t, config.Targets, 2)
	assert.Equal(t, "org/service1", config.Targets[0].Repo)
	assert.Equal(t, "org/service2", config.Targets[1].Repo)
}

// TestTransformVariablesModification tests modifying transform variables
func TestTransformVariablesModification(t *testing.T) {
	transform := &Transform{
		Variables: make(map[string]string),
	}

	// Add variables
	transform.Variables["KEY1"] = "value1"
	transform.Variables["KEY2"] = "value2"

	assert.Len(t, transform.Variables, 2)
	assert.Equal(t, "value1", transform.Variables["KEY1"])
	assert.Equal(t, "value2", transform.Variables["KEY2"])

	// Modify variable
	transform.Variables["KEY1"] = "modified"
	assert.Equal(t, "modified", transform.Variables["KEY1"])

	// Delete variable
	delete(transform.Variables, "KEY2")
	assert.Len(t, transform.Variables, 1)
	_, exists := transform.Variables["KEY2"]
	assert.False(t, exists)
}

// TestTargetConfigPRFields tests TargetConfig PR-related fields
func TestTargetConfigPRFields(t *testing.T) {
	t.Run("all PR fields can be set and accessed", func(t *testing.T) {
		target := TargetConfig{
			Repo:            "org/service",
			PRLabels:        []string{"label1", "label2"},
			PRAssignees:     []string{"user1", "user2"},
			PRReviewers:     []string{"reviewer1"},
			PRTeamReviewers: []string{"team1", "team2"},
		}

		assert.Equal(t, "org/service", target.Repo)
		assert.Equal(t, []string{"label1", "label2"}, target.PRLabels)
		assert.Equal(t, []string{"user1", "user2"}, target.PRAssignees)
		assert.Equal(t, []string{"reviewer1"}, target.PRReviewers)
		assert.Equal(t, []string{"team1", "team2"}, target.PRTeamReviewers)
	})

	t.Run("PR fields can be empty slices", func(t *testing.T) {
		target := TargetConfig{
			Repo:            "org/service",
			PRLabels:        []string{},
			PRAssignees:     []string{},
			PRReviewers:     []string{},
			PRTeamReviewers: []string{},
		}

		assert.Empty(t, target.PRLabels)
		assert.Empty(t, target.PRAssignees)
		assert.Empty(t, target.PRReviewers)
		assert.Empty(t, target.PRTeamReviewers)
	})

	t.Run("single element PR fields", func(t *testing.T) {
		target := TargetConfig{
			Repo:            "org/service",
			PRLabels:        []string{"single-label"},
			PRAssignees:     []string{"single-user"},
			PRReviewers:     []string{"single-reviewer"},
			PRTeamReviewers: []string{"single-team"},
		}

		assert.Len(t, target.PRLabels, 1)
		assert.Equal(t, "single-label", target.PRLabels[0])
		assert.Len(t, target.PRAssignees, 1)
		assert.Equal(t, "single-user", target.PRAssignees[0])
		assert.Len(t, target.PRReviewers, 1)
		assert.Equal(t, "single-reviewer", target.PRReviewers[0])
		assert.Len(t, target.PRTeamReviewers, 1)
		assert.Equal(t, "single-team", target.PRTeamReviewers[0])
	})

	t.Run("PR fields can be modified after creation", func(t *testing.T) {
		target := TargetConfig{
			Repo: "org/service",
		}

		// Initially nil/empty
		assert.Nil(t, target.PRLabels)
		assert.Nil(t, target.PRAssignees)

		// Add labels
		target.PRLabels = append(target.PRLabels, "new-label")
		target.PRAssignees = append(target.PRAssignees, "new-assignee")

		assert.Len(t, target.PRLabels, 1)
		assert.Equal(t, "new-label", target.PRLabels[0])
		assert.Len(t, target.PRAssignees, 1)
		assert.Equal(t, "new-assignee", target.PRAssignees[0])

		// Add more
		target.PRLabels = append(target.PRLabels, "another-label")
		target.PRAssignees = append(target.PRAssignees, "another-assignee")

		assert.Len(t, target.PRLabels, 2)
		assert.Len(t, target.PRAssignees, 2)
	})
}
