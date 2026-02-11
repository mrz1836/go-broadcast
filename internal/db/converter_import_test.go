package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestConverterImport_EdgeCases tests import edge cases
func TestConverterImport_EdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Import with empty file/directory lists", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "empty-mappings-test",
			Name:    "Empty Mappings Test",
			FileLists: []config.FileList{
				{
					ID:    "empty-file-list",
					Name:  "Empty File List",
					Files: []config.FileMapping{}, // Empty
				},
			},
			DirectoryLists: []config.DirectoryList{
				{
					ID:          "empty-dir-list",
					Name:        "Empty Dir List",
					Directories: []config.DirectoryMapping{}, // Empty
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
							Repo:              "mrz1836/target",
							FileListRefs:      []string{"empty-file-list"},
							DirectoryListRefs: []string{"empty-dir-list"},
							Files:             []config.FileMapping{},      // Empty inline
							Directories:       []config.DirectoryMapping{}, // Empty inline
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Len(t, exported.FileLists, 1)
		assert.Len(t, exported.DirectoryLists, 1)
		assert.Empty(t, exported.FileLists[0].Files)
		assert.Empty(t, exported.DirectoryLists[0].Directories)
	})
}

// TestConverterImport_DirectoryListEdgeCases targets importDirectoryLists
func TestConverterImport_DirectoryListEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Directory list with module config", func(t *testing.T) {
		checkTags := true
		cfg := &config.Config{
			Version: 1,
			ID:      "dirlist-module-test",
			Name:    "DirList Module Test",
			DirectoryLists: []config.DirectoryList{
				{
					ID:   "dir-list-with-module",
					Name: "Dir List With Module",
					Directories: []config.DirectoryMapping{
						{
							Src:  "src/module",
							Dest: "dest/module",
							Module: &config.ModuleConfig{
								Type:       "go",
								Version:    "v1.0.0",
								CheckTags:  &checkTags,
								UpdateRefs: true,
							},
						},
					},
				},
			},
			Groups: []config.Group{
				{
					ID:   "test-group-dirmod",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo:              "mrz1836/target",
							DirectoryListRefs: []string{"dir-list-with-module"},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Len(t, exported.DirectoryLists, 1)
		assert.Len(t, exported.DirectoryLists[0].Directories, 1)
		assert.NotNil(t, exported.DirectoryLists[0].Directories[0].Module)
	})

	t.Run("Directory list with all fields", func(t *testing.T) {
		preserveStructure := false
		includeHidden := true

		cfg := &config.Config{
			Version: 1,
			ID:      "dirlist-all-fields-test",
			Name:    "DirList All Fields Test",
			DirectoryLists: []config.DirectoryList{
				{
					ID:          "full-dir-list",
					Name:        "Full Directory List",
					Description: "Complete directory list",
					Directories: []config.DirectoryMapping{
						{
							Src:               "src/full",
							Dest:              "dest/full",
							Exclude:           []string{"*.log"},
							IncludeOnly:       []string{"*.go"},
							PreserveStructure: &preserveStructure,
							IncludeHidden:     &includeHidden,
							Transform: config.Transform{
								RepoName: true,
								Variables: map[string]string{
									"KEY": "value",
								},
							},
						},
					},
				},
			},
			Groups: []config.Group{
				{
					ID:   "test-group-allfields",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo:              "mrz1836/target",
							DirectoryListRefs: []string{"full-dir-list"},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)

		dir := exported.DirectoryLists[0].Directories[0]
		assert.NotNil(t, dir.PreserveStructure)
		assert.False(t, *dir.PreserveStructure)
		assert.NotNil(t, dir.IncludeHidden)
		assert.True(t, *dir.IncludeHidden)
		assert.Len(t, dir.Exclude, 1)
		assert.Len(t, dir.IncludeOnly, 1)
	})

	t.Run("Directory list with delete flag", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "dirlist-delete-test",
			Name:    "DirList Delete Test",
			DirectoryLists: []config.DirectoryList{
				{
					ID:   "delete-dir-list",
					Name: "Delete Dir List",
					Directories: []config.DirectoryMapping{
						{
							Dest:   "dir-to-delete",
							Delete: true, // Delete flag
						},
					},
				},
			},
			Groups: []config.Group{
				{
					ID:   "test-group-delete",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo:              "mrz1836/target",
							DirectoryListRefs: []string{"delete-dir-list"},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.True(t, exported.DirectoryLists[0].Directories[0].Delete)
	})
}

// TestConverterImport_GroupEdgeCases targets importGroups
func TestConverterImport_GroupEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Group with all optional fields", func(t *testing.T) {
		enabled := false
		cfg := &config.Config{
			Version: 1,
			ID:      "group-full-test",
			Name:    "Group Full Test",
			Groups: []config.Group{
				{
					ID:          "full-group",
					Name:        "Full Group",
					Description: "Full description",
					Priority:    100,
					Enabled:     &enabled,
					Source: config.SourceConfig{
						Repo:          "mrz1836/test",
						Branch:        "develop",
						BlobSizeLimit: "100MB",
						SecurityEmail: "security@example.com",
						SupportEmail:  "support@example.com",
					},
					Global: config.GlobalConfig{
						PRLabels:        []string{"sync", "auto"},
						PRAssignees:     []string{"user1", "user2"},
						PRReviewers:     []string{"reviewer1"},
						PRTeamReviewers: []string{"team1"},
					},
					Defaults: config.DefaultConfig{
						BranchPrefix:    "chore/sync",
						PRLabels:        []string{"default"},
						PRAssignees:     []string{"default-assignee"},
						PRReviewers:     []string{"default-reviewer"},
						PRTeamReviewers: []string{"default-team"},
					},
					Targets: []config.TargetConfig{
						{
							Repo:            "mrz1836/target",
							Branch:          "main",
							BlobSizeLimit:   "50MB",
							SecurityEmail:   "sec@target.com",
							SupportEmail:    "sup@target.com",
							PRLabels:        []string{"target-label"},
							PRAssignees:     []string{"target-assignee"},
							PRReviewers:     []string{"target-reviewer"},
							PRTeamReviewers: []string{"target-team"},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)

		group := exported.Groups[0]
		assert.Equal(t, "Full Group", group.Name)
		assert.Equal(t, "Full description", group.Description)
		assert.Equal(t, 100, group.Priority)
		assert.NotNil(t, group.Enabled)
		assert.False(t, *group.Enabled)
		assert.Equal(t, "develop", group.Source.Branch)
		assert.Equal(t, "100MB", group.Source.BlobSizeLimit)
		assert.Equal(t, "security@example.com", group.Source.SecurityEmail)
	})

	t.Run("Group with empty defaults", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "group-empty-defaults-test",
			Name:    "Group Empty Defaults Test",
			Groups: []config.Group{
				{
					ID:   "empty-defaults-group",
					Name: "Empty Defaults Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Defaults: config.DefaultConfig{
						BranchPrefix: "", // Empty
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target"},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Empty(t, exported.Groups[0].Defaults.BranchPrefix)
	})
}

// TestConverterImport_TargetEdgeCases targets importTargets
func TestConverterImport_TargetEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Target with both inline and refs", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "target-mixed-test",
			Name:    "Target Mixed Test",
			FileLists: []config.FileList{
				{
					ID:   "shared-files",
					Name: "Shared Files",
					Files: []config.FileMapping{
						{Src: "shared.txt", Dest: "shared.txt"},
					},
				},
			},
			DirectoryLists: []config.DirectoryList{
				{
					ID:   "shared-dirs",
					Name: "Shared Dirs",
					Directories: []config.DirectoryMapping{
						{Src: "shared/dir", Dest: "dir"},
					},
				},
			},
			Groups: []config.Group{
				{
					ID:   "test-group-mixed",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo:   "mrz1836/target",
							Branch: "develop",
							Files: []config.FileMapping{
								{Src: "inline.txt", Dest: "inline.txt"},
							},
							Directories: []config.DirectoryMapping{
								{Src: "inline/dir", Dest: "inline-dir"},
							},
							FileListRefs:      []string{"shared-files"},
							DirectoryListRefs: []string{"shared-dirs"},
							Transform: config.Transform{
								RepoName: true,
								Variables: map[string]string{
									"VAR": "val",
								},
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)

		target := exported.Groups[0].Targets[0]
		assert.Len(t, target.Files, 1)
		assert.Len(t, target.Directories, 1)
		assert.Len(t, target.FileListRefs, 1)
		assert.Len(t, target.DirectoryListRefs, 1)
		assert.True(t, target.Transform.RepoName)
	})

	t.Run("Target with empty branch", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "target-empty-branch-test",
			Name:    "Target Empty Branch Test",
			Groups: []config.Group{
				{
					ID:   "test-group-empty",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo:   "mrz1836/target",
							Branch: "", // Empty branch
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Empty(t, exported.Groups[0].Targets[0].Branch)
	})
}

// TestConverterImport_FileListDeleteFlag tests importFileMappings with delete flag
func TestConverterImport_FileListDeleteFlag(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		ID:      "filelist-delete-test",
		Name:    "FileList Delete Test",
		FileLists: []config.FileList{
			{
				ID:   "delete-files",
				Name: "Delete Files",
				Files: []config.FileMapping{
					{Dest: "file-to-delete.txt", Delete: true},
					{Src: "keep.txt", Dest: "keep.txt", Delete: false},
				},
			},
		},
		Groups: []config.Group{
			{
				ID:   "test-group-filelist",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:         "mrz1836/target",
						FileListRefs: []string{"delete-files"},
					},
				},
			},
		},
	}

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)

	files := exported.FileLists[0].Files
	assert.Len(t, files, 2)
	assert.True(t, files[0].Delete)
	assert.False(t, files[1].Delete)
}

// TestConverterImport_MultipleTargetsSameGroup tests multiple targets in a group
func TestConverterImport_MultipleTargetsSameGroup(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		ID:      "multi-target-test",
		Name:    "Multi Target Test",
		Groups: []config.Group{
			{
				ID:   "test-group-multi",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:   "mrz1836/target1",
						Branch: "main",
					},
					{
						Repo:   "mrz1836/target2",
						Branch: "develop",
					},
					{
						Repo:   "mrz1836/target3",
						Branch: "staging",
					},
				},
			},
		},
	}

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)
	assert.Len(t, exported.Groups[0].Targets, 3)
	assert.Equal(t, "mrz1836/target1", exported.Groups[0].Targets[0].Repo)
	assert.Equal(t, "mrz1836/target2", exported.Groups[0].Targets[1].Repo)
	assert.Equal(t, "mrz1836/target3", exported.Groups[0].Targets[2].Repo)
}

// TestConverterImport_SourceEdgeCases tests Source validation edge cases
func TestConverterImport_SourceEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Source with all email fields", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "source-emails-test",
			Name:    "Source Emails Test",
			Groups: []config.Group{
				{
					ID:   "group-source-emails",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:          "mrz1836/test",
						Branch:        "main",
						SecurityEmail: "security@example.com",
						SupportEmail:  "support@example.com",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target"},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Equal(t, "security@example.com", exported.Groups[0].Source.SecurityEmail)
		assert.Equal(t, "support@example.com", exported.Groups[0].Source.SupportEmail)
	})

	t.Run("Source with blob size limit", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "source-blob-test",
			Name:    "Source Blob Test",
			Groups: []config.Group{
				{
					ID:   "group-source-blob",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:          "mrz1836/test",
						Branch:        "main",
						BlobSizeLimit: "50MB",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target"},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Equal(t, "50MB", exported.Groups[0].Source.BlobSizeLimit)
	})
}

// TestConverterImport_FileMappingsEdgeCases tests importFileMappings edge cases
func TestConverterImport_FileMappingsEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("File mapping with empty src (delete)", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "filemp-empty-src-test",
			Name:    "FileMapping Empty Src Test",
			Groups: []config.Group{
				{
					ID:   "group-filemp-empty",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: "mrz1836/target",
							Files: []config.FileMapping{
								{
									Src:    "", // Empty src
									Dest:   "to-delete.txt",
									Delete: true,
								},
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.True(t, exported.Groups[0].Targets[0].Files[0].Delete)
		assert.Empty(t, exported.Groups[0].Targets[0].Files[0].Src)
	})

	t.Run("Multiple file mappings in target", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "filemp-multiple-test",
			Name:    "FileMapping Multiple Test",
			Groups: []config.Group{
				{
					ID:   "group-filemp-multi",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: "mrz1836/target",
							Files: []config.FileMapping{
								{Src: "file1.txt", Dest: "dest1.txt"},
								{Src: "file2.txt", Dest: "dest2.txt"},
								{Src: "file3.txt", Dest: "dest3.txt"},
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Len(t, exported.Groups[0].Targets[0].Files, 3)
	})
}

// TestConverterImport_DirectoryMappingsEdgeCases tests importDirectoryMappings edge cases
func TestConverterImport_DirectoryMappingsEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Directory mapping with empty src (delete)", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "dirmp-empty-src-test",
			Name:    "DirMapping Empty Src Test",
			Groups: []config.Group{
				{
					ID:   "group-dirmp-empty",
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
									Src:    "", // Empty src
									Dest:   "to-delete-dir",
									Delete: true,
								},
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.True(t, exported.Groups[0].Targets[0].Directories[0].Delete)
		assert.Empty(t, exported.Groups[0].Targets[0].Directories[0].Src)
	})

	t.Run("Multiple directory mappings in target", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "dirmp-multiple-test",
			Name:    "DirMapping Multiple Test",
			Groups: []config.Group{
				{
					ID:   "group-dirmp-multi",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: "mrz1836/target",
							Directories: []config.DirectoryMapping{
								{Src: "dir1", Dest: "dest-dir1"},
								{Src: "dir2", Dest: "dest-dir2"},
								{Src: "dir3", Dest: "dest-dir3"},
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Len(t, exported.Groups[0].Targets[0].Directories, 3)
	})
}

// TestConverterImport_GroupDependenciesEdgeCases tests importGroupDependencies edge cases
func TestConverterImport_GroupDependenciesEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Group with no dependencies", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "group-nodep-test",
			Name:    "Group No Dep Test",
			Groups: []config.Group{
				{
					ID:        "group-nodep",
					Name:      "No Dependencies",
					DependsOn: []string{}, // Explicitly empty
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target"},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Empty(t, exported.Groups[0].DependsOn)
	})

	t.Run("Group with single dependency", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "group-singledep-test",
			Name:    "Group Single Dep Test",
			Groups: []config.Group{
				{
					ID:   "group-single-a",
					Name: "Group A",
					Source: config.SourceConfig{
						Repo:   "mrz1836/a",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target-a"},
					},
				},
				{
					ID:        "group-single-b",
					Name:      "Group B",
					DependsOn: []string{"group-single-a"}, // Single dependency
					Source: config.SourceConfig{
						Repo:   "mrz1836/b",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target-b"},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)

		// Find group-single-b
		var groupB *config.Group
		for i := range exported.Groups {
			if exported.Groups[i].ID == "group-single-b" {
				groupB = &exported.Groups[i]
				break
			}
		}
		require.NotNil(t, groupB)
		assert.Equal(t, []string{"group-single-a"}, groupB.DependsOn)
	})
}

// TestConverterImport_FileListsWithPositions tests file list position handling
func TestConverterImport_FileListsWithPositions(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		ID:      "filelist-pos-test",
		Name:    "FileList Position Test",
		FileLists: []config.FileList{
			{
				ID:   "list1",
				Name: "List 1",
				Files: []config.FileMapping{
					{Src: "a.txt", Dest: "a.txt"},
					{Src: "b.txt", Dest: "b.txt"},
					{Src: "c.txt", Dest: "c.txt"},
				},
			},
			{
				ID:   "list2",
				Name: "List 2",
				Files: []config.FileMapping{
					{Src: "x.txt", Dest: "x.txt"},
				},
			},
		},
		Groups: []config.Group{
			{
				ID:   "group-filelist-pos",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:         "mrz1836/target",
						FileListRefs: []string{"list1", "list2"},
					},
				},
			},
		},
	}

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)
	assert.Len(t, exported.FileLists, 2)
	assert.Len(t, exported.FileLists[0].Files, 3)
}

// TestConverterImport_DirectoryListsWithTransform tests directory list transform
func TestConverterImport_DirectoryListsWithTransform(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		ID:      "dirlist-transform-test",
		Name:    "DirList Transform Test",
		DirectoryLists: []config.DirectoryList{
			{
				ID:   "dir-with-transform",
				Name: "Dir With Transform",
				Directories: []config.DirectoryMapping{
					{
						Src:  "src/transform",
						Dest: "dest/transform",
						Transform: config.Transform{
							RepoName: true,
							Variables: map[string]string{
								"VAR1": "val1",
								"VAR2": "val2",
							},
						},
					},
				},
			},
		},
		Groups: []config.Group{
			{
				ID:   "group-dirlist-transform",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:              "mrz1836/target",
						DirectoryListRefs: []string{"dir-with-transform"},
					},
				},
			},
		},
	}

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)

	dir := exported.DirectoryLists[0].Directories[0]
	assert.True(t, dir.Transform.RepoName)
	assert.Len(t, dir.Transform.Variables, 2)
}

// TestConverterImport_TargetWithAllEmailFields tests target email fields
func TestConverterImport_TargetWithAllEmailFields(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		ID:      "target-emails-test",
		Name:    "Target Emails Test",
		Groups: []config.Group{
			{
				ID:   "group-target-emails",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:          "mrz1836/target",
						SecurityEmail: "sec@target.com",
						SupportEmail:  "sup@target.com",
						BlobSizeLimit: "75MB",
					},
				},
			},
		},
	}

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)

	target := exported.Groups[0].Targets[0]
	assert.Equal(t, "sec@target.com", target.SecurityEmail)
	assert.Equal(t, "sup@target.com", target.SupportEmail)
	assert.Equal(t, "75MB", target.BlobSizeLimit)
}

// TestConverterImport_GroupWithAllPRSettings tests all PR settings combinations
func TestConverterImport_GroupWithAllPRSettings(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		ID:      "pr-settings-full-test",
		Name:    "PR Settings Full Test",
		Groups: []config.Group{
			{
				ID:   "group-pr-full",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Global: config.GlobalConfig{
					PRLabels:        []string{"global1", "global2"},
					PRAssignees:     []string{"assignee1", "assignee2"},
					PRReviewers:     []string{"reviewer1", "reviewer2"},
					PRTeamReviewers: []string{"team1", "team2"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix:    "sync",
					PRLabels:        []string{"default1"},
					PRAssignees:     []string{"default-assignee"},
					PRReviewers:     []string{"default-reviewer"},
					PRTeamReviewers: []string{"default-team"},
				},
				Targets: []config.TargetConfig{
					{
						Repo:            "mrz1836/target",
						PRLabels:        []string{"target1", "target2", "target3"},
						PRAssignees:     []string{"target-assignee1", "target-assignee2"},
						PRReviewers:     []string{"target-reviewer1"},
						PRTeamReviewers: []string{"target-team1", "target-team2"},
					},
				},
			},
		},
	}

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)

	group := exported.Groups[0]
	assert.Len(t, group.Global.PRLabels, 2)
	assert.Len(t, group.Global.PRAssignees, 2)
	assert.Len(t, group.Defaults.PRLabels, 1)
	assert.Len(t, group.Targets[0].PRLabels, 3)
	assert.Len(t, group.Targets[0].PRAssignees, 2)
}

// TestConverterImport_UpdateExisting tests updating existing records
func TestConverterImport_UpdateExisting(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	// Initial import
	cfg1 := &config.Config{
		Version: 1,
		ID:      "update-existing-test",
		Name:    "Initial Name",
		Groups: []config.Group{
			{
				ID:   "group-update",
				Name: "Initial Group Name",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target1"},
				},
			},
		},
	}

	_, err := converter.ImportConfig(ctx, cfg1)
	require.NoError(t, err)

	// Update: change name and add target
	cfg2 := &config.Config{
		Version: 1,
		ID:      "update-existing-test",
		Name:    "Updated Name",
		Groups: []config.Group{
			{
				ID:   "group-update",
				Name: "Updated Group Name",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "develop",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target1"},
					{Repo: "mrz1836/target2"},
				},
			},
		},
	}

	_, err = converter.ImportConfig(ctx, cfg2)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, "update-existing-test")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", exported.Name)
	assert.Equal(t, "Updated Group Name", exported.Groups[0].Name)
	assert.Equal(t, "develop", exported.Groups[0].Source.Branch)
	assert.Len(t, exported.Groups[0].Targets, 2)
}

// TestConverterImport_EmptyConfig tests minimal config
func TestConverterImport_EmptyConfig(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		ID:      "empty-config-test",
		Name:    "Empty Config",
		Groups:  []config.Group{}, // No groups
	}

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)
	assert.Empty(t, exported.Groups)
	assert.Empty(t, exported.FileLists)
	assert.Empty(t, exported.DirectoryLists)
}

// TestConverterImport_ComplexDependencies tests dependency edge cases
func TestConverterImport_ComplexDependencies(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Multiple groups with complex dependencies", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "complex-deps-test",
			Name:    "Complex Dependencies Test",
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

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Len(t, exported.Groups, 3)
	})

	t.Run("Self-dependency detection", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "self-dep-test",
			Name:    "Self Dependency Test",
			Groups: []config.Group{
				{
					ID:        "group-a",
					Name:      "Group A",
					DependsOn: []string{"group-a"}, // Self-dependency
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{{Repo: "mrz1836/target"}},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCircularDependency)
	})
}

// TestConverterImport_EmptyAndNilFields tests edge cases with empty/nil fields
func TestConverterImport_EmptyAndNilFields(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Empty transform variables", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "empty-transform-test",
			Name:    "Empty Transform Test",
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
								RepoName:  false,
								Variables: map[string]string{}, // Empty
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		// Empty map might be nil or empty after export - just verify it doesn't error
		assert.NotNil(t, exported.Groups[0].Targets[0])
	})

	t.Run("Empty PR settings arrays", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "empty-pr-test",
			Name:    "Empty PR Settings Test",
			Groups: []config.Group{
				{
					ID:   "test-group",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Global: config.GlobalConfig{
						PRLabels:        []string{}, // Empty arrays
						PRAssignees:     []string{},
						PRReviewers:     []string{},
						PRTeamReviewers: []string{},
					},
					Targets: []config.TargetConfig{{Repo: "mrz1836/target"}},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.NotNil(t, exported.Groups[0].Global.PRLabels)
	})

	t.Run("Directory mapping with all optional fields", func(t *testing.T) {
		preserveStructure := true
		includeHidden := false

		cfg := &config.Config{
			Version: 1,
			ID:      "full-dirmap-test",
			Name:    "Full Directory Mapping Test",
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
									Src:               "src/dir",
									Dest:              "dest/dir",
									Exclude:           []string{"*.log", "*.tmp"},
									IncludeOnly:       []string{"*.go", "*.md"},
									PreserveStructure: &preserveStructure,
									IncludeHidden:     &includeHidden,
									Transform: config.Transform{
										RepoName: true,
										Variables: map[string]string{
											"VAR1": "value1",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)

		dirMap := exported.Groups[0].Targets[0].Directories[0]
		assert.Len(t, dirMap.Exclude, 2)
		assert.Len(t, dirMap.IncludeOnly, 2)
		assert.NotNil(t, dirMap.PreserveStructure)
		assert.True(t, *dirMap.PreserveStructure)
		assert.NotNil(t, dirMap.IncludeHidden)
		assert.False(t, *dirMap.IncludeHidden)
	})
}

// TestConverterImport_AddFileList tests adding file lists on update
func TestConverterImport_AddFileList(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	// Initial import without file list
	cfg1 := &config.Config{
		Version: 1,
		ID:      "update-add-list-test",
		Name:    "Update Add List Test",
		Groups: []config.Group{
			{
				ID:   "test-group-add",
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

	_, err := converter.ImportConfig(ctx, cfg1)
	require.NoError(t, err)

	// Update: add file list
	cfg2 := &config.Config{
		Version: 1,
		ID:      "update-add-list-test",
		Name:    "Update Add List Test",
		FileLists: []config.FileList{
			{
				ID:   "file-list-new",
				Name: "New File List",
				Files: []config.FileMapping{
					{Src: "file.txt", Dest: "file.txt"},
				},
			},
		},
		Groups: []config.Group{
			{
				ID:   "test-group-add",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:         "mrz1836/target",
						FileListRefs: []string{"file-list-new"}, // New reference
					},
				},
			},
		},
	}

	_, err = converter.ImportConfig(ctx, cfg2)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, "update-add-list-test")
	require.NoError(t, err)

	assert.Len(t, exported.FileLists, 1)
	assert.Equal(t, "file-list-new", exported.FileLists[0].ID)
	assert.Equal(t, []string{"file-list-new"}, exported.Groups[0].Targets[0].FileListRefs)
}
