package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestConverterRoundTrip_RealSyncYAML tests import -> export with actual sync.yaml
func TestConverterRoundTrip_RealSyncYAML(t *testing.T) {
	// Load real sync.yaml - skip if not available (CI environment)
	syncPath := filepath.Join("..", "..", "sync.yaml")
	if _, err := os.Stat(syncPath); os.IsNotExist(err) {
		t.Skip("sync.yaml not available in this environment")
	}
	data, err := os.ReadFile(syncPath) //nolint:gosec // reading test fixture from known path
	require.NoError(t, err, "failed to read sync.yaml")

	var cfg config.Config
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err, "failed to unmarshal sync.yaml")

	// Set external ID if not present
	if cfg.ID == "" {
		cfg.ID = "sync-config"
	}

	// Create test database
	db := TestDB(t)
	converter := NewConverter(db)

	// Import config
	importedConfig, err := converter.ImportConfig(context.Background(), &cfg)
	require.NoError(t, err, "failed to import config")
	require.NotNil(t, importedConfig)

	// Export config
	exportedCfg, err := converter.ExportConfig(context.Background(), cfg.ID)
	require.NoError(t, err, "failed to export config")
	require.NotNil(t, exportedCfg)

	// Verify structural equality
	assert.Equal(t, cfg.Version, exportedCfg.Version, "version mismatch")
	assert.Equal(t, cfg.Name, exportedCfg.Name, "name mismatch")
	assert.Equal(t, cfg.ID, exportedCfg.ID, "ID mismatch")

	// Verify file lists
	assert.Len(t, exportedCfg.FileLists, len(cfg.FileLists), "file lists count mismatch")
	for i, original := range cfg.FileLists {
		exported := exportedCfg.FileLists[i]
		assert.Equal(t, original.ID, exported.ID, "file list %d ID mismatch", i)
		assert.Equal(t, original.Name, exported.Name, "file list %d name mismatch", i)
		assert.Equal(t, original.Description, exported.Description, "file list %d description mismatch", i)
		assert.Len(t, exported.Files, len(original.Files), "file list %d files count mismatch", i)

		for j, origFile := range original.Files {
			expFile := exported.Files[j]
			assert.Equal(t, expFile.Src, origFile.Src, "file list %d file %d src mismatch", i, j)
			assert.Equal(t, expFile.Dest, origFile.Dest, "file list %d file %d dest mismatch", i, j)
			assert.Equal(t, expFile.Delete, origFile.Delete, "file list %d file %d delete mismatch", i, j)
		}
	}

	// Verify directory lists
	assert.Len(t, exportedCfg.DirectoryLists, len(cfg.DirectoryLists), "directory lists count mismatch")
	for i, original := range cfg.DirectoryLists {
		exported := exportedCfg.DirectoryLists[i]
		assert.Equal(t, original.ID, exported.ID, "directory list %d ID mismatch", i)
		assert.Equal(t, original.Name, exported.Name, "directory list %d name mismatch", i)
		assert.Len(t, exported.Directories, len(original.Directories), "directory list %d directories count mismatch", i)
	}

	// Verify groups
	assert.Len(t, exportedCfg.Groups, len(cfg.Groups), "groups count mismatch")
	for i, original := range cfg.Groups {
		exported := exportedCfg.Groups[i]
		assert.Equal(t, original.ID, exported.ID, "group %d ID mismatch", i)
		assert.Equal(t, original.Name, exported.Name, "group %d name mismatch", i)
		assert.Equal(t, original.Description, exported.Description, "group %d description mismatch", i)
		assert.Equal(t, original.Priority, exported.Priority, "group %d priority mismatch", i)

		// Verify source
		assert.Equal(t, original.Source.Repo, exported.Source.Repo, "group %d source repo mismatch", i)
		assert.Equal(t, original.Source.Branch, exported.Source.Branch, "group %d source branch mismatch", i)

		// Verify targets
		assert.Len(t, exported.Targets, len(original.Targets), "group %d targets count mismatch", i)
		for j, origTarget := range original.Targets {
			expTarget := exported.Targets[j]
			assert.Equal(t, expTarget.Repo, origTarget.Repo, "group %d target %d repo mismatch", i, j)
			assert.Equal(t, expTarget.Branch, origTarget.Branch, "group %d target %d branch mismatch", i, j)

			// Verify file list refs
			assert.ElementsMatch(t, origTarget.FileListRefs, expTarget.FileListRefs,
				"group %d target %d file list refs mismatch", i, j)

			// Verify directory list refs
			assert.ElementsMatch(t, origTarget.DirectoryListRefs, expTarget.DirectoryListRefs,
				"group %d target %d directory list refs mismatch", i, j)

			// Verify inline files
			assert.Len(t, expTarget.Files, len(origTarget.Files),
				"group %d target %d files count mismatch", i, j)

			// Verify inline directories
			assert.Len(t, expTarget.Directories, len(origTarget.Directories),
				"group %d target %d directories count mismatch", i, j)
		}
	}
}

// TestConverterRoundTrip_EmptyLists tests import/export with empty file/directory lists
func TestConverterRoundTrip_EmptyLists(t *testing.T) {
	cfg := &config.Config{
		Version:        1,
		ID:             "empty-lists-test",
		Name:           "Empty Lists Test",
		FileLists:      []config.FileList{},
		DirectoryLists: []config.DirectoryList{},
		Groups: []config.Group{
			{
				ID:   "test-group",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "mrz1836/target",
					},
				},
			},
		},
	}

	db := TestDB(t)
	converter := NewConverter(db)

	// Import
	_, err := converter.ImportConfig(context.Background(), cfg)
	require.NoError(t, err)

	// Export
	exported, err := converter.ExportConfig(context.Background(), cfg.ID)
	require.NoError(t, err)

	assert.Equal(t, cfg.ID, exported.ID)
	assert.Equal(t, cfg.Version, exported.Version)
	assert.Len(t, exported.Groups, 1)
	assert.Empty(t, exported.FileLists)
	assert.Empty(t, exported.DirectoryLists)
}

// TestConverterRoundTrip_InlineAndRefs tests targets with both inline files and refs
func TestConverterRoundTrip_InlineAndRefs(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		ID:      "inline-refs-test",
		Name:    "Inline and Refs Test",
		FileLists: []config.FileList{
			{
				ID:   "shared-files",
				Name: "Shared Files",
				Files: []config.FileMapping{
					{Src: "shared/file1.txt", Dest: "file1.txt"},
					{Src: "shared/file2.txt", Dest: "file2.txt"},
				},
			},
		},
		DirectoryLists: []config.DirectoryList{
			{
				ID:   "shared-dirs",
				Name: "Shared Directories",
				Directories: []config.DirectoryMapping{
					{Src: "shared/dir1", Dest: "dir1"},
				},
			},
		},
		Groups: []config.Group{
			{
				ID:   "test-group",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "mrz1836/target",
						Files: []config.FileMapping{
							{Src: "inline/file.txt", Dest: "inline.txt"},
						},
						Directories: []config.DirectoryMapping{
							{Src: "inline/dir", Dest: "inline-dir"},
						},
						FileListRefs:      []string{"shared-files"},
						DirectoryListRefs: []string{"shared-dirs"},
					},
				},
			},
		},
	}

	db := TestDB(t)
	converter := NewConverter(db)

	// Import
	_, err := converter.ImportConfig(context.Background(), cfg)
	require.NoError(t, err)

	// Export
	exported, err := converter.ExportConfig(context.Background(), cfg.ID)
	require.NoError(t, err)

	// Verify target has both inline and refs
	target := exported.Groups[0].Targets[0]
	assert.Len(t, target.Files, 1, "inline files should be preserved")
	assert.Len(t, target.Directories, 1, "inline directories should be preserved")
	assert.Equal(t, []string{"shared-files"}, target.FileListRefs, "file list refs should be preserved")
	assert.Equal(t, []string{"shared-dirs"}, target.DirectoryListRefs, "directory list refs should be preserved")
}

// TestConverterRoundTrip_TransformInheritance tests transform at target and directory level
func TestConverterRoundTrip_TransformInheritance(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		ID:      "transform-test",
		Name:    "Transform Inheritance Test",
		Groups: []config.Group{
			{
				ID:   "test-group",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "mrz1836/target",
						Transform: config.Transform{
							RepoName: true,
							Variables: map[string]string{
								"VAR1": "value1",
								"VAR2": "value2",
							},
						},
						Directories: []config.DirectoryMapping{
							{
								Src:  "dir1",
								Dest: "dir1",
								Transform: config.Transform{
									RepoName: false,
									Variables: map[string]string{
										"DIR_VAR": "dir_value",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	db := TestDB(t)
	converter := NewConverter(db)

	// Import
	_, err := converter.ImportConfig(context.Background(), cfg)
	require.NoError(t, err)

	// Export
	exported, err := converter.ExportConfig(context.Background(), cfg.ID)
	require.NoError(t, err)

	// Verify target-level transform
	target := exported.Groups[0].Targets[0]
	assert.True(t, target.Transform.RepoName, "target transform repo_name should be preserved")
	assert.Len(t, target.Transform.Variables, 2, "target transform variables count")
	assert.Equal(t, "value1", target.Transform.Variables["VAR1"])
	assert.Equal(t, "value2", target.Transform.Variables["VAR2"])

	// Verify directory-level transform
	dirMapping := target.Directories[0]
	assert.False(t, dirMapping.Transform.RepoName, "directory transform repo_name should be preserved")
	assert.Len(t, dirMapping.Transform.Variables, 1, "directory transform variables count")
	assert.Equal(t, "dir_value", dirMapping.Transform.Variables["DIR_VAR"])
}

// TestConverterRoundTrip_ModuleConfig tests module configuration in directory mappings
func TestConverterRoundTrip_ModuleConfig(t *testing.T) {
	checkTags := true
	cfg := &config.Config{
		Version: 1,
		ID:      "module-test",
		Name:    "Module Config Test",
		Groups: []config.Group{
			{
				ID:   "test-group",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "mrz1836/target",
						Directories: []config.DirectoryMapping{
							{
								Src:  "module-dir",
								Dest: "module-dir",
								Module: &config.ModuleConfig{
									Type:       "go",
									Version:    "v1.2.3",
									CheckTags:  &checkTags,
									UpdateRefs: true,
								},
							},
						},
					},
				},
			},
		},
	}

	db := TestDB(t)
	converter := NewConverter(db)

	// Import
	_, err := converter.ImportConfig(context.Background(), cfg)
	require.NoError(t, err)

	// Export
	exported, err := converter.ExportConfig(context.Background(), cfg.ID)
	require.NoError(t, err)

	// Verify module config
	dirMapping := exported.Groups[0].Targets[0].Directories[0]
	require.NotNil(t, dirMapping.Module, "module config should be preserved")
	assert.Equal(t, "go", dirMapping.Module.Type)
	assert.Equal(t, "v1.2.3", dirMapping.Module.Version)
	assert.NotNil(t, dirMapping.Module.CheckTags)
	assert.True(t, *dirMapping.Module.CheckTags)
	assert.True(t, dirMapping.Module.UpdateRefs)
}

// TestConverterRoundTrip_BoolPointers tests proper handling of *bool fields
func TestConverterRoundTrip_BoolPointers(t *testing.T) {
	enabled := true
	preserveStructure := false
	includeHidden := true

	cfg := &config.Config{
		Version: 1,
		ID:      "bool-test",
		Name:    "Bool Pointer Test",
		Groups: []config.Group{
			{
				ID:      "test-group",
				Name:    "Test Group",
				Enabled: &enabled,
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "mrz1836/target",
						Directories: []config.DirectoryMapping{
							{
								Src:               "dir",
								Dest:              "dir",
								PreserveStructure: &preserveStructure,
								IncludeHidden:     &includeHidden,
							},
						},
					},
				},
			},
		},
	}

	db := TestDB(t)
	converter := NewConverter(db)

	// Import
	_, err := converter.ImportConfig(context.Background(), cfg)
	require.NoError(t, err)

	// Export
	exported, err := converter.ExportConfig(context.Background(), cfg.ID)
	require.NoError(t, err)

	// Verify group enabled
	assert.NotNil(t, exported.Groups[0].Enabled)
	assert.True(t, *exported.Groups[0].Enabled)

	// Verify directory mapping bools
	dirMapping := exported.Groups[0].Targets[0].Directories[0]
	assert.NotNil(t, dirMapping.PreserveStructure)
	assert.False(t, *dirMapping.PreserveStructure)
	assert.NotNil(t, dirMapping.IncludeHidden)
	assert.True(t, *dirMapping.IncludeHidden)
}

// TestConverterRoundTrip_PRSettings tests PR labels, assignees, reviewers
func TestConverterRoundTrip_PRSettings(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		ID:      "pr-settings-test",
		Name:    "PR Settings Test",
		Groups: []config.Group{
			{
				ID:   "test-group",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Global: config.GlobalConfig{
					PRLabels:        []string{"global-label"},
					PRAssignees:     []string{"global-assignee"},
					PRReviewers:     []string{"global-reviewer"},
					PRTeamReviewers: []string{"global-team"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix:    "chore/sync",
					PRLabels:        []string{"default-label"},
					PRAssignees:     []string{"default-assignee"},
					PRReviewers:     []string{"default-reviewer"},
					PRTeamReviewers: []string{"default-team"},
				},
				Targets: []config.TargetConfig{
					{
						Repo:            "mrz1836/target",
						PRLabels:        []string{"target-label"},
						PRAssignees:     []string{"target-assignee"},
						PRReviewers:     []string{"target-reviewer"},
						PRTeamReviewers: []string{"target-team"},
					},
				},
			},
		},
	}

	db := TestDB(t)
	converter := NewConverter(db)

	// Import
	_, err := converter.ImportConfig(context.Background(), cfg)
	require.NoError(t, err)

	// Export
	exported, err := converter.ExportConfig(context.Background(), cfg.ID)
	require.NoError(t, err)

	// Verify global settings
	global := exported.Groups[0].Global
	assert.Equal(t, []string{"global-label"}, global.PRLabels)
	assert.Equal(t, []string{"global-assignee"}, global.PRAssignees)
	assert.Equal(t, []string{"global-reviewer"}, global.PRReviewers)
	assert.Equal(t, []string{"global-team"}, global.PRTeamReviewers)

	// Verify defaults
	defaults := exported.Groups[0].Defaults
	assert.Equal(t, "chore/sync", defaults.BranchPrefix)
	assert.Equal(t, []string{"default-label"}, defaults.PRLabels)
	assert.Equal(t, []string{"default-assignee"}, defaults.PRAssignees)
	assert.Equal(t, []string{"default-reviewer"}, defaults.PRReviewers)
	assert.Equal(t, []string{"default-team"}, defaults.PRTeamReviewers)

	// Verify target overrides
	target := exported.Groups[0].Targets[0]
	assert.Equal(t, []string{"target-label"}, target.PRLabels)
	assert.Equal(t, []string{"target-assignee"}, target.PRAssignees)
	assert.Equal(t, []string{"target-reviewer"}, target.PRReviewers)
	assert.Equal(t, []string{"target-team"}, target.PRTeamReviewers)
}

// TestConverterRoundTrip_GroupDependencies tests group dependency ordering
func TestConverterRoundTrip_GroupDependencies(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		ID:      "deps-test",
		Name:    "Dependencies Test",
		Groups: []config.Group{
			{
				ID:   "group-a",
				Name: "Group A",
				Source: config.SourceConfig{
					Repo:   "mrz1836/a",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{Repo: "mrz1836/target-a"}},
			},
			{
				ID:        "group-b",
				Name:      "Group B",
				DependsOn: []string{"group-a"},
				Source: config.SourceConfig{
					Repo:   "mrz1836/b",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{Repo: "mrz1836/target-b"}},
			},
			{
				ID:        "group-c",
				Name:      "Group C",
				DependsOn: []string{"group-a", "group-b"},
				Source: config.SourceConfig{
					Repo:   "mrz1836/c",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{Repo: "mrz1836/target-c"}},
			},
		},
	}

	db := TestDB(t)
	converter := NewConverter(db)

	// Import
	_, err := converter.ImportConfig(context.Background(), cfg)
	require.NoError(t, err)

	// Export
	exported, err := converter.ExportConfig(context.Background(), cfg.ID)
	require.NoError(t, err)

	// Verify dependencies preserved
	var groupB, groupC *config.Group
	for i := range exported.Groups {
		if exported.Groups[i].ID == "group-b" {
			groupB = &exported.Groups[i]
		}
		if exported.Groups[i].ID == "group-c" {
			groupC = &exported.Groups[i]
		}
	}

	require.NotNil(t, groupB, "group-b should exist")
	require.NotNil(t, groupC, "group-c should exist")

	assert.Equal(t, []string{"group-a"}, groupB.DependsOn)
	assert.ElementsMatch(t, []string{"group-a", "group-b"}, groupC.DependsOn)
}

// TestConverterImport_CircularDependency tests circular dependency detection
func TestConverterImport_CircularDependency(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		ID:      "circular-test",
		Name:    "Circular Dependency Test",
		Groups: []config.Group{
			{
				ID:        "group-a",
				Name:      "Group A",
				DependsOn: []string{"group-b"}, // A depends on B
				Source: config.SourceConfig{
					Repo:   "mrz1836/a",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{Repo: "mrz1836/target-a"}},
			},
			{
				ID:        "group-b",
				Name:      "Group B",
				DependsOn: []string{"group-a"}, // B depends on A (cycle!)
				Source: config.SourceConfig{
					Repo:   "mrz1836/b",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{Repo: "mrz1836/target-b"}},
			},
		},
	}

	db := TestDB(t)
	converter := NewConverter(db)

	// Import should fail with circular dependency error
	_, err := converter.ImportConfig(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircularDependency)
}

// TestConverterImport_MissingReference tests missing file_list_ref validation
func TestConverterImport_MissingReference(t *testing.T) {
	cfg := &config.Config{
		Version:   1,
		ID:        "missing-ref-test",
		Name:      "Missing Reference Test",
		FileLists: []config.FileList{}, // Empty, but target references a list
		Groups: []config.Group{
			{
				ID:   "test-group",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:         "mrz1836/target",
						FileListRefs: []string{"non-existent-list"}, // Reference to non-existent list
					},
				},
			},
		},
	}

	db := TestDB(t)
	converter := NewConverter(db)

	// Import should fail with reference not found error
	_, err := converter.ImportConfig(context.Background(), cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrReferenceNotFound)
}

// TestConverterImport_Update tests updating an existing config
func TestConverterImport_Update(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)

	// Initial import
	cfg1 := &config.Config{
		Version: 1,
		ID:      "update-test",
		Name:    "Initial Config",
		Groups: []config.Group{
			{
				ID:   "group-1",
				Name: "Group 1",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{Repo: "mrz1836/target-1"}},
			},
		},
	}

	_, err := converter.ImportConfig(context.Background(), cfg1)
	require.NoError(t, err)

	// Update with new group
	cfg2 := &config.Config{
		Version: 1,
		ID:      "update-test",
		Name:    "Updated Config",
		Groups: []config.Group{
			{
				ID:   "group-1",
				Name: "Group 1 Updated",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "develop",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target-1"},
					{Repo: "mrz1836/target-2"}, // New target
				},
			},
			{
				ID:   "group-2",
				Name: "Group 2",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test2",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{Repo: "mrz1836/target-3"}},
			},
		},
	}

	_, err = converter.ImportConfig(context.Background(), cfg2)
	require.NoError(t, err)

	// Export and verify
	exported, err := converter.ExportConfig(context.Background(), "update-test")
	require.NoError(t, err)

	assert.Equal(t, "Updated Config", exported.Name)
	assert.Len(t, exported.Groups, 2)

	// Find group-1
	var group1 *config.Group
	for i := range exported.Groups {
		if exported.Groups[i].ID == "group-1" {
			group1 = &exported.Groups[i]
			break
		}
	}

	require.NotNil(t, group1)
	assert.Equal(t, "Group 1 Updated", group1.Name)
	assert.Equal(t, "develop", group1.Source.Branch)
	assert.Len(t, group1.Targets, 2)
}

// TestConverterExport_SingleGroup tests exporting a single group
func TestConverterExport_SingleGroup(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)

	// Import config with multiple groups
	cfg := &config.Config{
		Version: 1,
		ID:      "multi-group-test",
		Name:    "Multi Group Test",
		Groups: []config.Group{
			{
				ID:   "group-1",
				Name: "Group 1",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test1",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{Repo: "mrz1836/target-1"}},
			},
			{
				ID:   "group-2",
				Name: "Group 2",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test2",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{Repo: "mrz1836/target-2"}},
			},
		},
	}

	importedConfig, err := converter.ImportConfig(context.Background(), cfg)
	require.NoError(t, err)

	// Export single group
	group, err := converter.ExportGroup(context.Background(), importedConfig.ID, "group-1")
	require.NoError(t, err)
	require.NotNil(t, group)

	assert.Equal(t, "group-1", group.ID)
	assert.Equal(t, "Group 1", group.Name)
	assert.Equal(t, "mrz1836/test1", group.Source.Repo)
	assert.Len(t, group.Targets, 1)
}

// TestConverterRoundTrip_DeleteFlags tests delete flag preservation
func TestConverterRoundTrip_DeleteFlags(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		ID:      "delete-test",
		Name:    "Delete Flags Test",
		Groups: []config.Group{
			{
				ID:   "test-group",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "mrz1836/target",
						Files: []config.FileMapping{
							{Dest: "file-to-delete.txt", Delete: true},
							{Src: "keep.txt", Dest: "keep.txt", Delete: false},
						},
						Directories: []config.DirectoryMapping{
							{Dest: "dir-to-delete", Delete: true},
							{Src: "keep-dir", Dest: "keep-dir", Delete: false},
						},
					},
				},
			},
		},
	}

	db := TestDB(t)
	converter := NewConverter(db)

	// Import
	_, err := converter.ImportConfig(context.Background(), cfg)
	require.NoError(t, err)

	// Export
	exported, err := converter.ExportConfig(context.Background(), cfg.ID)
	require.NoError(t, err)

	// Verify delete flags
	target := exported.Groups[0].Targets[0]
	assert.True(t, target.Files[0].Delete, "file delete flag should be true")
	assert.False(t, target.Files[1].Delete, "file delete flag should be false")
	assert.True(t, target.Directories[0].Delete, "directory delete flag should be true")
	assert.False(t, target.Directories[1].Delete, "directory delete flag should be false")
}
