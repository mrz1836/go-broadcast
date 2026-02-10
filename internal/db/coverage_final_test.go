package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestConverterImport_DirectoryListEdgeCases targets importDirectoryLists (58.8% -> higher)
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

// TestConverterImport_GroupEdgeCases targets importGroups (69.0% -> higher)
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

// TestConverterImport_TargetEdgeCases targets importTargets (69.2% -> higher)
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

// TestConverterImport_FileListDeleteFlag tests importFileMappings with delete flag (80% -> higher)
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

// TestConverterImport_MultipleTargetsSameGroup tests multiple targets in a group (69.2% -> higher)
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

// TestConverterExport_FileListRefs tests exportFileListRefs edge cases (75% -> higher)
func TestConverterExport_FileListRefs(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		ID:      "filelist-refs-test",
		Name:    "FileList Refs Test",
		FileLists: []config.FileList{
			{
				ID:   "list1",
				Name: "List 1",
				Files: []config.FileMapping{
					{Src: "file1.txt", Dest: "file1.txt"},
				},
			},
			{
				ID:   "list2",
				Name: "List 2",
				Files: []config.FileMapping{
					{Src: "file2.txt", Dest: "file2.txt"},
				},
			},
		},
		Groups: []config.Group{
			{
				ID:   "test-group-refs",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:         "mrz1836/target",
						FileListRefs: []string{"list1", "list2"}, // Multiple refs
					},
				},
			},
		},
	}

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)

	refs := exported.Groups[0].Targets[0].FileListRefs
	assert.Len(t, refs, 2)
	assert.ElementsMatch(t, []string{"list1", "list2"}, refs)
}
