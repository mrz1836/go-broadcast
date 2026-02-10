package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestConverterImport_FileListsWithPositions tests file list position handling (58.8% -> higher)
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

// TestConverterImport_DirectoryListsWithTransform tests directory list transform (58.8% -> higher)
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

// TestConverterImport_TargetWithAllEmailFields tests target email fields (69.2% -> higher)
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

// TestConverterImport_GroupWithAllPRSettings tests all PR settings combinations (69% -> higher)
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

// TestConverterImport_UpdateExisting tests updating existing records (60-82% -> higher)
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

// TestConverterImport_EmptyConfig tests minimal config (82.1% -> higher)
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

// TestRepositoryCreate_DuplicateExternalID tests unique constraint handling (66.7% -> higher)
func TestRepositoryCreate_DuplicateExternalID(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create config
	config := &Config{
		ExternalID: "dup-test-config",
		Name:       "Test",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	t.Run("Duplicate FileList external ID", func(t *testing.T) {
		repo := NewFileListRepository(db)

		fileList1 := &FileList{
			ConfigID:   config.ID,
			ExternalID: "dup-filelist",
			Name:       "First",
		}
		err := repo.Create(ctx, fileList1)
		require.NoError(t, err)

		fileList2 := &FileList{
			ConfigID:   config.ID,
			ExternalID: "dup-filelist", // Duplicate
			Name:       "Second",
		}
		err = repo.Create(ctx, fileList2)
		assert.Error(t, err) // Should fail with constraint error
	})

	t.Run("Duplicate DirectoryList external ID", func(t *testing.T) {
		repo := NewDirectoryListRepository(db)

		dirList1 := &DirectoryList{
			ConfigID:   config.ID,
			ExternalID: "dup-dirlist",
			Name:       "First",
		}
		err := repo.Create(ctx, dirList1)
		require.NoError(t, err)

		dirList2 := &DirectoryList{
			ConfigID:   config.ID,
			ExternalID: "dup-dirlist", // Duplicate
			Name:       "Second",
		}
		err = repo.Create(ctx, dirList2)
		assert.Error(t, err) // Should fail
	})

	t.Run("Duplicate Group external ID", func(t *testing.T) {
		repo := NewGroupRepository(db)

		group1 := &Group{
			ConfigID:   config.ID,
			ExternalID: "dup-group",
			Name:       "First",
		}
		err := repo.Create(ctx, group1)
		require.NoError(t, err)

		group2 := &Group{
			ConfigID:   config.ID,
			ExternalID: "dup-group", // Duplicate
			Name:       "Second",
		}
		err = repo.Create(ctx, group2)
		assert.Error(t, err) // Should fail
	})
}

// TestQueryRepository_EdgeCases tests query repository edge cases
func TestQueryRepository_EdgeCases(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create config and group
	config := &Config{
		ExternalID: "query-test-config",
		Name:       "Test",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "query-test-group",
		Name:       "Query Test Group",
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	target := &Target{
		GroupID: group.ID,
		Repo:    "mrz1836/query-test",
	}
	err = db.Create(target).Error
	require.NoError(t, err)

	t.Run("FindByRepo with existing repo", func(t *testing.T) {
		repo := NewQueryRepository(db)
		results, err := repo.FindByRepo(ctx, "mrz1836/query-test")
		require.NoError(t, err)
		assert.NotEmpty(t, results)
	})

	t.Run("FindByPattern with no matches", func(t *testing.T) {
		repo := NewQueryRepository(db)
		results, err := repo.FindByPattern(ctx, "no-match-pattern")
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}
