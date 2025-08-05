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
			Defaults: DefaultConfig{
				BranchPrefix: "sync/",
				PRLabels:     []string{"automated"},
			},
			Mappings: []SourceMapping{
				{
					Source: SourceConfig{
						Repo:   "org/template",
						Branch: "main",
						ID:     "primary",
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
				},
			},
		}

		require.NotNil(t, config)
		assert.Equal(t, 1, config.Version)
		assert.Equal(t, "sync/", config.Defaults.BranchPrefix)
		assert.Len(t, config.Defaults.PRLabels, 1)
		assert.Len(t, config.Mappings, 1)
		assert.Equal(t, "org/template", config.Mappings[0].Source.Repo)
		assert.Equal(t, "primary", config.Mappings[0].Source.ID)
		assert.Len(t, config.Mappings[0].Targets, 1)
		assert.True(t, config.Mappings[0].Targets[0].Transform.RepoName)
		assert.Equal(t, "value", config.Mappings[0].Targets[0].Transform.Variables["KEY"])
	})
}

// TestSourceConfigDefaults tests SourceConfig zero values
func TestSourceConfigDefaults(t *testing.T) {
	source := SourceConfig{}

	assert.Empty(t, source.Repo)
	assert.Empty(t, source.Branch)
	assert.Empty(t, source.ID)
}

// TestDefaultConfigDefaults tests DefaultConfig zero values
func TestDefaultConfigDefaults(t *testing.T) {
	defaults := DefaultConfig{}

	assert.Empty(t, defaults.BranchPrefix)
	assert.Nil(t, defaults.PRLabels)
	assert.Nil(t, defaults.PRAssignees)
	assert.Nil(t, defaults.PRReviewers)
	assert.Nil(t, defaults.PRTeamReviewers)
}

// TestTargetConfigDefaults tests TargetConfig zero values
func TestTargetConfigDefaults(t *testing.T) {
	target := TargetConfig{}

	assert.Empty(t, target.Repo)
	assert.Nil(t, target.Files)
	assert.Nil(t, target.Directories)
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

// TestDirectoryMappingDefaults tests DirectoryMapping zero values
func TestDirectoryMappingDefaults(t *testing.T) {
	dir := DirectoryMapping{}

	assert.Empty(t, dir.Src)
	assert.Empty(t, dir.Dest)
	assert.Nil(t, dir.Exclude)
	assert.Nil(t, dir.IncludeOnly)
	assert.Nil(t, dir.PreserveStructure)
	assert.Nil(t, dir.IncludeHidden)
	assert.False(t, dir.Transform.RepoName)
	assert.Nil(t, dir.Transform.Variables)
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
		Defaults: DefaultConfig{
			BranchPrefix: "prefix",
			PRLabels:     []string{}, // Empty slice
		},
		Mappings: []SourceMapping{}, // Empty mappings
	}

	require.NotNil(t, config)
	assert.Empty(t, config.Defaults.PRLabels)
	assert.Empty(t, config.Mappings)
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
		Mappings: []SourceMapping{
			{
				Source: SourceConfig{},
				Targets: []TargetConfig{
					{Repo: "org/target"},
				},
			},
		},
	}

	// Modify fields
	config.Mappings[0].Source.Repo = "new/repo"
	config.Mappings[0].Source.Branch = "develop"
	config.Mappings[0].Source.ID = "modified"
	config.Defaults.BranchPrefix = "feature/"
	config.Defaults.PRLabels = append(config.Defaults.PRLabels, "label1", "label2")

	assert.Equal(t, "new/repo", config.Mappings[0].Source.Repo)
	assert.Equal(t, "develop", config.Mappings[0].Source.Branch)
	assert.Equal(t, "modified", config.Mappings[0].Source.ID)
	assert.Equal(t, "feature/", config.Defaults.BranchPrefix)
	assert.Len(t, config.Defaults.PRLabels, 2)
}

// TestSourceMappingAppend tests appending to mappings slice
func TestSourceMappingAppend(t *testing.T) {
	config := &Config{
		Version:  1,
		Mappings: []SourceMapping{},
	}

	// Append mappings
	config.Mappings = append(config.Mappings, SourceMapping{
		Source: SourceConfig{
			Repo: "org/source1",
			ID:   "source1",
		},
		Targets: []TargetConfig{
			{
				Repo: "org/target1",
				Files: []FileMapping{
					{Src: "a.txt", Dest: "b.txt"},
				},
			},
		},
	})

	config.Mappings = append(config.Mappings, SourceMapping{
		Source: SourceConfig{
			Repo: "org/source2",
			ID:   "source2",
		},
		Targets: []TargetConfig{
			{
				Repo: "org/target2",
				Files: []FileMapping{
					{Src: "c.txt", Dest: "d.txt"},
				},
			},
		},
	})

	assert.Len(t, config.Mappings, 2)
	assert.Equal(t, "org/source1", config.Mappings[0].Source.Repo)
	assert.Equal(t, "source1", config.Mappings[0].Source.ID)
	assert.Equal(t, "org/source2", config.Mappings[1].Source.Repo)
	assert.Equal(t, "source2", config.Mappings[1].Source.ID)
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

// TestGlobalConfigDefaults tests GlobalConfig zero values
func TestGlobalConfigDefaults(t *testing.T) {
	global := GlobalConfig{}

	assert.Nil(t, global.PRLabels)
	assert.Nil(t, global.PRAssignees)
	assert.Nil(t, global.PRReviewers)
	assert.Nil(t, global.PRTeamReviewers)
}

// TestGlobalConfigFields tests GlobalConfig PR-related fields
func TestGlobalConfigFields(t *testing.T) {
	t.Run("all PR fields can be set and accessed", func(t *testing.T) {
		global := GlobalConfig{
			PRLabels:        []string{"global-label1", "global-label2"},
			PRAssignees:     []string{"global-user1", "global-user2"},
			PRReviewers:     []string{"global-reviewer1"},
			PRTeamReviewers: []string{"global-team1", "global-team2"},
		}

		assert.Equal(t, []string{"global-label1", "global-label2"}, global.PRLabels)
		assert.Equal(t, []string{"global-user1", "global-user2"}, global.PRAssignees)
		assert.Equal(t, []string{"global-reviewer1"}, global.PRReviewers)
		assert.Equal(t, []string{"global-team1", "global-team2"}, global.PRTeamReviewers)
	})

	t.Run("PR fields can be empty slices", func(t *testing.T) {
		global := GlobalConfig{
			PRLabels:        []string{},
			PRAssignees:     []string{},
			PRReviewers:     []string{},
			PRTeamReviewers: []string{},
		}

		assert.Empty(t, global.PRLabels)
		assert.Empty(t, global.PRAssignees)
		assert.Empty(t, global.PRReviewers)
		assert.Empty(t, global.PRTeamReviewers)
	})

	t.Run("single element PR fields", func(t *testing.T) {
		global := GlobalConfig{
			PRLabels:        []string{"single-global-label"},
			PRAssignees:     []string{"single-global-user"},
			PRReviewers:     []string{"single-global-reviewer"},
			PRTeamReviewers: []string{"single-global-team"},
		}

		assert.Len(t, global.PRLabels, 1)
		assert.Equal(t, "single-global-label", global.PRLabels[0])
		assert.Len(t, global.PRAssignees, 1)
		assert.Equal(t, "single-global-user", global.PRAssignees[0])
		assert.Len(t, global.PRReviewers, 1)
		assert.Equal(t, "single-global-reviewer", global.PRReviewers[0])
		assert.Len(t, global.PRTeamReviewers, 1)
		assert.Equal(t, "single-global-team", global.PRTeamReviewers[0])
	})

	t.Run("PR fields can be modified after creation", func(t *testing.T) {
		global := GlobalConfig{}

		// Initially nil/empty
		assert.Nil(t, global.PRLabels)
		assert.Nil(t, global.PRAssignees)

		// Add labels
		global.PRLabels = append(global.PRLabels, "new-global-label")
		global.PRAssignees = append(global.PRAssignees, "new-global-assignee")

		assert.Len(t, global.PRLabels, 1)
		assert.Equal(t, "new-global-label", global.PRLabels[0])
		assert.Len(t, global.PRAssignees, 1)
		assert.Equal(t, "new-global-assignee", global.PRAssignees[0])

		// Add more
		global.PRLabels = append(global.PRLabels, "another-global-label")
		global.PRAssignees = append(global.PRAssignees, "another-global-assignee")

		assert.Len(t, global.PRLabels, 2)
		assert.Len(t, global.PRAssignees, 2)
	})
}

// TestConfigWithGlobalSection tests Config with GlobalConfig field
func TestConfigWithGlobalSection(t *testing.T) {
	config := &Config{
		Version: 1,
		Global: GlobalConfig{
			PRLabels:    []string{"automated-sync", "chore"},
			PRAssignees: []string{"platform-team"},
		},
		Mappings: []SourceMapping{
			{
				Source: SourceConfig{
					Repo: "org/template",
					ID:   "main",
				},
				Targets: []TargetConfig{
					{
						Repo: "org/service",
						Files: []FileMapping{
							{Src: "file.txt", Dest: "dest.txt"},
						},
						PRLabels: []string{"critical"},
					},
				},
			},
		},
	}

	require.NotNil(t, config)
	assert.Equal(t, 1, config.Version)
	assert.Equal(t, "org/template", config.Mappings[0].Source.Repo)
	assert.Equal(t, []string{"automated-sync", "chore"}, config.Global.PRLabels)
	assert.Equal(t, []string{"platform-team"}, config.Global.PRAssignees)
	assert.Len(t, config.Mappings, 1)
	assert.Equal(t, []string{"critical"}, config.Mappings[0].Targets[0].PRLabels)
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

// TestSourceMappingDefaults tests SourceMapping zero values
func TestSourceMappingDefaults(t *testing.T) {
	mapping := SourceMapping{}

	assert.Empty(t, mapping.Source.Repo)
	assert.Empty(t, mapping.Source.Branch)
	assert.Empty(t, mapping.Source.ID)
	assert.Nil(t, mapping.Targets)
}

// TestConflictResolutionDefaults tests ConflictResolution zero values
func TestConflictResolutionDefaults(t *testing.T) {
	conflict := ConflictResolution{}

	assert.Empty(t, conflict.Strategy)
	assert.Nil(t, conflict.Priority)
	// CustomRules field was removed - no assertion needed
}

// TestConflictResolutionStrategies tests different conflict resolution strategies
func TestConflictResolutionStrategies(t *testing.T) {
	tests := []struct {
		name     string
		conflict *ConflictResolution
	}{
		{
			name: "error strategy",
			conflict: &ConflictResolution{
				Strategy: "error",
			},
		},
		{
			name: "priority strategy with order",
			conflict: &ConflictResolution{
				Strategy: "priority",
				Priority: []string{"source1", "source2", "source3"},
			},
		},
		{
			name: "merge strategy",
			conflict: &ConflictResolution{
				Strategy: "merge",
			},
		},
		{
			name: "priority strategy",
			conflict: &ConflictResolution{
				Strategy: "priority",
				Priority: []string{"core", "extensions"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.conflict)
			assert.NotEmpty(t, tt.conflict.Strategy)
		})
	}
}

// TestCustomRuleDefaults was removed as CustomRule type no longer exists

// TestConfigWithMultipleSourceMappings tests config with multiple source mappings
func TestConfigWithMultipleSourceMappings(t *testing.T) {
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
					{
						Repo: "org/target1",
						Files: []FileMapping{
							{Src: "file1.txt", Dest: "file1.txt"},
						},
					},
				},
			},
			{
				Source: SourceConfig{
					Repo:   "org/source2",
					Branch: "develop",
					ID:     "source2",
				},
				Targets: []TargetConfig{
					{
						Repo: "org/target2",
						Files: []FileMapping{
							{Src: "file2.txt", Dest: "file2.txt"},
						},
					},
					{
						Repo: "org/target3",
						Files: []FileMapping{
							{Src: "file3.txt", Dest: "file3.txt"},
						},
					},
				},
			},
		},
		ConflictResolution: &ConflictResolution{
			Strategy: "priority",
			Priority: []string{"source1", "source2"},
		},
	}

	require.NotNil(t, config)
	assert.Equal(t, 1, config.Version)
	assert.Len(t, config.Mappings, 2)

	// First mapping
	assert.Equal(t, "org/source1", config.Mappings[0].Source.Repo)
	assert.Equal(t, "source1", config.Mappings[0].Source.ID)
	assert.Len(t, config.Mappings[0].Targets, 1)

	// Second mapping
	assert.Equal(t, "org/source2", config.Mappings[1].Source.Repo)
	assert.Equal(t, "source2", config.Mappings[1].Source.ID)
	assert.Len(t, config.Mappings[1].Targets, 2)

	// Conflict resolution
	assert.NotNil(t, config.ConflictResolution)
	assert.Equal(t, "priority", config.ConflictResolution.Strategy)
	assert.Equal(t, []string{"source1", "source2"}, config.ConflictResolution.Priority)
}
