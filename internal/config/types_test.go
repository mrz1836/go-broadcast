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
			Groups: []Group{
				{
					Name: "test-group",
					ID:   "test",
					Source: SourceConfig{
						Repo: "org/template",
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
				},
			},
		}

		require.NotNil(t, config)
		assert.Equal(t, 1, config.Version)
		require.Len(t, config.Groups, 1)
		group := config.Groups[0]
		assert.Equal(t, "org/template", group.Source.Repo)
		assert.Equal(t, "sync/", group.Defaults.BranchPrefix)
		assert.Len(t, group.Defaults.PRLabels, 1)
		assert.Len(t, group.Targets, 1)
		assert.True(t, group.Targets[0].Transform.RepoName)
		assert.Equal(t, "value", group.Targets[0].Transform.Variables["KEY"])
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
		Groups: []Group{
			{
				Name: "test-group",
				ID:   "test",
				Source: SourceConfig{
					Repo: "org/repo",
				},
				Defaults: DefaultConfig{
					BranchPrefix: "prefix",
					PRLabels:     []string{}, // Empty slice
				},
				Targets: []TargetConfig{}, // Empty targets
			},
		},
	}

	require.NotNil(t, config)
	require.Len(t, config.Groups, 1)
	group := config.Groups[0]
	assert.Empty(t, group.Defaults.PRLabels)
	assert.Empty(t, group.Targets)
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

// TestConfigFieldModification tests that group fields can be modified
func TestConfigFieldModification(t *testing.T) {
	config := &Config{
		Version: 1,
		Groups: []Group{
			{
				Name: "test-group",
				ID:   "test",
			},
		},
	}

	// Modify fields
	group := &config.Groups[0]
	group.Source.Repo = "new/repo"
	group.Source.Branch = "develop"
	group.Defaults.BranchPrefix = "feature/"
	group.Defaults.PRLabels = append(group.Defaults.PRLabels, "label1", "label2")

	assert.Equal(t, "new/repo", group.Source.Repo)
	assert.Equal(t, "develop", group.Source.Branch)
	assert.Equal(t, "feature/", group.Defaults.BranchPrefix)
	assert.Len(t, group.Defaults.PRLabels, 2)
}

// TestTargetConfigAppend tests appending to targets slice within a group
func TestTargetConfigAppend(t *testing.T) {
	config := &Config{
		Version: 1,
		Groups: []Group{
			{
				Name:    "test-group",
				ID:      "test",
				Targets: []TargetConfig{},
			},
		},
	}

	// Append targets to group
	group := &config.Groups[0]
	group.Targets = append(group.Targets, TargetConfig{
		Repo: "org/service1",
		Files: []FileMapping{
			{Src: "a.txt", Dest: "b.txt"},
		},
	})

	group.Targets = append(group.Targets, TargetConfig{
		Repo: "org/service2",
		Files: []FileMapping{
			{Src: "c.txt", Dest: "d.txt"},
		},
	})

	assert.Len(t, group.Targets, 2)
	assert.Equal(t, "org/service1", group.Targets[0].Repo)
	assert.Equal(t, "org/service2", group.Targets[1].Repo)
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

// TestConfigWithGlobalSection tests Config with group GlobalConfig field
func TestConfigWithGlobalSection(t *testing.T) {
	config := &Config{
		Version: 1,
		Groups: []Group{
			{
				Name: "test-group",
				ID:   "test",
				Source: SourceConfig{
					Repo: "org/template",
				},
				Global: GlobalConfig{
					PRLabels:    []string{"automated-sync", "chore"},
					PRAssignees: []string{"platform-team"},
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
	require.Len(t, config.Groups, 1)
	group := config.Groups[0]
	assert.Equal(t, "org/template", group.Source.Repo)
	assert.Equal(t, []string{"automated-sync", "chore"}, group.Global.PRLabels)
	assert.Equal(t, []string{"platform-team"}, group.Global.PRAssignees)
	assert.Len(t, group.Targets, 1)
	assert.Equal(t, []string{"critical"}, group.Targets[0].PRLabels)
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

// TestNewGroupType tests the new Group type
func TestNewGroupType(t *testing.T) {
	t.Run("CreateGroup", func(t *testing.T) {
		group := Group{
			Name:        "Test Group",
			ID:          "test-group",
			Description: "A test group",
			Priority:    1,
			DependsOn:   []string{"another-group"},
			Enabled:     boolPtr(true),
			Source: SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Global: GlobalConfig{
				PRLabels: []string{"automated"},
			},
			Defaults: DefaultConfig{
				BranchPrefix: "sync/",
			},
			Targets: []TargetConfig{
				{
					Repo: "org/service",
					Files: []FileMapping{
						{Src: "file.txt", Dest: "dest.txt"},
					},
				},
			},
		}

		assert.Equal(t, "Test Group", group.Name)
		assert.Equal(t, "test-group", group.ID)
		assert.Equal(t, "A test group", group.Description)
		assert.Equal(t, 1, group.Priority)
		assert.Equal(t, []string{"another-group"}, group.DependsOn)
		assert.NotNil(t, group.Enabled)
		assert.True(t, *group.Enabled)
		assert.Equal(t, "org/template", group.Source.Repo)
		assert.Equal(t, "main", group.Source.Branch)
		assert.Len(t, group.Targets, 1)
	})

	t.Run("GroupDefaults", func(t *testing.T) {
		group := Group{}

		assert.Empty(t, group.Name)
		assert.Empty(t, group.ID)
		assert.Empty(t, group.Description)
		assert.Equal(t, 0, group.Priority) // Default priority
		assert.Nil(t, group.DependsOn)
		assert.Nil(t, group.Enabled) // nil means default (true)
		assert.Empty(t, group.Source.Repo)
		assert.Nil(t, group.Targets)
	})
}

// TestNewModuleConfig tests the new ModuleConfig type
func TestNewModuleConfig(t *testing.T) {
	t.Run("CreateModuleConfig", func(t *testing.T) {
		module := ModuleConfig{
			Type:       "go",
			Version:    "v1.2.3",
			CheckTags:  boolPtr(true),
			UpdateRefs: true,
		}

		assert.Equal(t, "go", module.Type)
		assert.Equal(t, "v1.2.3", module.Version)
		assert.NotNil(t, module.CheckTags)
		assert.True(t, *module.CheckTags)
		assert.True(t, module.UpdateRefs)
	})

	t.Run("ModuleConfigDefaults", func(t *testing.T) {
		module := ModuleConfig{}

		assert.Empty(t, module.Type)
		assert.Empty(t, module.Version)
		assert.Nil(t, module.CheckTags)    // nil means default (true)
		assert.False(t, module.UpdateRefs) // Default false
	})

	t.Run("ModuleConfigSemverVersions", func(t *testing.T) {
		testCases := []struct {
			name    string
			version string
		}{
			{"exact version", "v1.2.3"},
			{"latest", "latest"},
			{"semver constraint", "^1.0.0"},
			{"semver range", ">=1.0.0 <2.0.0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				module := ModuleConfig{
					Type:    "go",
					Version: tc.version,
				}

				assert.Equal(t, tc.version, module.Version)
			})
		}
	})
}

// TestDirectoryMappingWithModule tests DirectoryMapping with Module field
func TestDirectoryMappingWithModule(t *testing.T) {
	t.Run("DirectoryWithModule", func(t *testing.T) {
		dirMapping := DirectoryMapping{
			Src:  "pkg/shared",
			Dest: "vendor/github.com/org/shared",
			Module: &ModuleConfig{
				Type:       "go",
				Version:    "v1.0.0",
				CheckTags:  boolPtr(true),
				UpdateRefs: true,
			},
		}

		assert.Equal(t, "pkg/shared", dirMapping.Src)
		assert.Equal(t, "vendor/github.com/org/shared", dirMapping.Dest)
		assert.NotNil(t, dirMapping.Module)
		assert.Equal(t, "go", dirMapping.Module.Type)
		assert.Equal(t, "v1.0.0", dirMapping.Module.Version)
	})

	t.Run("DirectoryWithoutModule", func(t *testing.T) {
		dirMapping := DirectoryMapping{
			Src:  "scripts",
			Dest: "scripts",
		}

		assert.Equal(t, "scripts", dirMapping.Src)
		assert.Equal(t, "scripts", dirMapping.Dest)
		assert.Nil(t, dirMapping.Module) // No module config
	})
}

// TestNewConfigWithGroups tests Config with new Groups field
func TestNewConfigWithGroups(t *testing.T) {
	t.Run("ConfigWithGroups", func(t *testing.T) {
		config := Config{
			Version: 1,
			Name:    "Multi-Group Sync",
			ID:      "multi-sync-2025",
			Groups: []Group{
				{
					Name:     "Infrastructure",
					ID:       "infra",
					Priority: 1,
					Enabled:  boolPtr(true),
					Source: SourceConfig{
						Repo:   "org/infra-templates",
						Branch: "main",
					},
					Targets: []TargetConfig{
						{Repo: "org/service1"},
					},
				},
				{
					Name:      "Security",
					ID:        "security",
					Priority:  2,
					DependsOn: []string{"infra"},
					Enabled:   boolPtr(true),
					Source: SourceConfig{
						Repo:   "org/security-templates",
						Branch: "main",
					},
					Targets: []TargetConfig{
						{Repo: "org/service1"},
					},
				},
			},
		}

		assert.Equal(t, 1, config.Version)
		assert.Equal(t, "Multi-Group Sync", config.Name)
		assert.Equal(t, "multi-sync-2025", config.ID)
		assert.Len(t, config.Groups, 2)
		assert.Equal(t, "Infrastructure", config.Groups[0].Name)
		assert.Equal(t, "Security", config.Groups[1].Name)
		assert.Equal(t, []string{"infra"}, config.Groups[1].DependsOn)
	})
}

// TestBoolPtrHelper tests the boolPtr helper function
func TestBoolPtrHelper(t *testing.T) {
	t.Run("BoolPtrTrue", func(t *testing.T) {
		ptr := boolPtr(true)
		assert.NotNil(t, ptr)
		assert.True(t, *ptr)
	})

	t.Run("BoolPtrFalse", func(t *testing.T) {
		ptr := boolPtr(false)
		assert.NotNil(t, ptr)
		assert.False(t, *ptr)
	})
}
