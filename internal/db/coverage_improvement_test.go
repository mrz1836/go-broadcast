package db

import (
	"context"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBoolPtr tests the boolPtr helper function (was 0% coverage)
func TestBoolPtr(t *testing.T) {
	// Test true
	truePtr := boolPtr(true)
	require.NotNil(t, truePtr)
	assert.True(t, *truePtr)

	// Test false
	falsePtr := boolPtr(false)
	require.NotNil(t, falsePtr)
	assert.False(t, *falsePtr)
}

// TestBoolVal_NilPointer tests the nil case of boolVal (was 66.7% coverage)
func TestBoolVal_NilPointer(t *testing.T) {
	// Test nil with true default
	result := boolVal(nil, true)
	assert.True(t, result)

	// Test nil with false default
	result = boolVal(nil, false)
	assert.False(t, result)
}

// TestRepositoryGetByExternalID_NotFound tests error paths (many at 50% coverage)
func TestRepositoryGetByExternalID_NotFound(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	t.Run("Config not found", func(t *testing.T) {
		repo := NewConfigRepository(db)
		_, err := repo.GetByExternalID(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Group not found", func(t *testing.T) {
		repo := NewGroupRepository(db)
		_, err := repo.GetByExternalID(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("FileList not found", func(t *testing.T) {
		repo := NewFileListRepository(db)
		_, err := repo.GetByExternalID(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("DirectoryList not found", func(t *testing.T) {
		repo := NewDirectoryListRepository(db)
		_, err := repo.GetByExternalID(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Target not found by repo", func(t *testing.T) {
		repo := NewTargetRepository(db)
		_, err := repo.GetByRepo(ctx, 1, "mrz1836/non-existent")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}

// TestRepositoryGetByID_NotFound tests GetByID error paths (50-83% coverage)
func TestRepositoryGetByID_NotFound(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	t.Run("FileList not found", func(t *testing.T) {
		repo := NewFileListRepository(db)
		_, err := repo.GetByID(ctx, 99999)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("DirectoryList not found", func(t *testing.T) {
		repo := NewDirectoryListRepository(db)
		_, err := repo.GetByID(ctx, 99999)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Target not found", func(t *testing.T) {
		repo := NewTargetRepository(db)
		_, err := repo.GetByID(ctx, 99999)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}

// TestRepositoryDelete_HardDelete tests hard delete paths (50-66% coverage)
func TestRepositoryDelete_HardDelete(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create config first
	config := &Config{
		ExternalID: "test-config-delete",
		Name:       "Test",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	t.Run("FileList hard delete", func(t *testing.T) {
		fileList := &FileList{
			ConfigID:   config.ID,
			ExternalID: "file-list-hard",
			Name:       "Test File List",
		}
		err := db.Create(fileList).Error
		require.NoError(t, err)

		repo := NewFileListRepository(db)
		err = repo.Delete(ctx, fileList.ID, true) // Hard delete
		require.NoError(t, err)

		// Should not be found even with Unscoped
		var count int64
		db.Unscoped().Model(&FileList{}).Where("id = ?", fileList.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("DirectoryList hard delete", func(t *testing.T) {
		dirList := &DirectoryList{
			ConfigID:   config.ID,
			ExternalID: "dir-list-hard",
			Name:       "Test Dir List",
		}
		err := db.Create(dirList).Error
		require.NoError(t, err)

		repo := NewDirectoryListRepository(db)
		err = repo.Delete(ctx, dirList.ID, true) // Hard delete
		require.NoError(t, err)

		var count int64
		db.Unscoped().Model(&DirectoryList{}).Where("id = ?", dirList.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Target hard delete", func(t *testing.T) {
		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "test-group-delete",
			Name:       "Test Group",
		}
		err := db.Create(group).Error
		require.NoError(t, err)

		target := &Target{
			GroupID: group.ID,
			Repo:    "mrz1836/test-delete",
		}
		err = db.Create(target).Error
		require.NoError(t, err)

		repo := NewTargetRepository(db)
		err = repo.Delete(ctx, target.ID, true) // Hard delete
		require.NoError(t, err)

		var count int64
		db.Unscoped().Model(&Target{}).Where("id = ?", target.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

// TestRepositoryListWithDirectories tests DirectoryList preloading (0% coverage)
func TestRepositoryListWithDirectories(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create config
	config := &Config{
		ExternalID: "test-config",
		Name:       "Test",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	// Create directory list with directories
	dirList := &DirectoryList{
		ConfigID:    config.ID,
		ExternalID:  "dir-list-1",
		Name:        "Test Dir List",
		Description: "Test",
	}
	err = db.Create(dirList).Error
	require.NoError(t, err)

	// Create directory mappings
	mapping := &DirectoryMapping{
		OwnerType: "directory_list",
		OwnerID:   dirList.ID,
		Src:       "src/dir",
		Dest:      "dest/dir",
		Position:  0,
	}
	err = db.Create(mapping).Error
	require.NoError(t, err)

	// Test ListWithDirectories
	repo := NewDirectoryListRepository(db)
	lists, err := repo.ListWithDirectories(ctx, config.ID)
	require.NoError(t, err)
	require.Len(t, lists, 1)
	assert.Len(t, lists[0].Directories, 1)
	assert.Equal(t, "src/dir", lists[0].Directories[0].Src)
}

// TestRepositoryListWithAssociations tests Target preloading (0% coverage)
func TestRepositoryListWithAssociations(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create config
	config := &Config{
		ExternalID: "test-config",
		Name:       "Test",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	// Create group
	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	// Create target with associations
	target := &Target{
		GroupID: group.ID,
		Repo:    "mrz1836/test",
		Branch:  "main",
		Transform: Transform{
			RepoName: true,
		},
	}
	err = db.Create(target).Error
	require.NoError(t, err)

	// Create file mapping
	fileMapping := &FileMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       "src/file.txt",
		Dest:      "dest/file.txt",
		Position:  0,
	}
	err = db.Create(fileMapping).Error
	require.NoError(t, err)

	// Create directory mapping
	dirMapping := &DirectoryMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       "src/dir",
		Dest:      "dest/dir",
		Position:  0,
	}
	err = db.Create(dirMapping).Error
	require.NoError(t, err)

	// Test ListWithAssociations
	repo := NewTargetRepository(db)
	targets, err := repo.ListWithAssociations(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Len(t, targets[0].FileMappings, 1)
	assert.Len(t, targets[0].DirectoryMappings, 1)
	assert.Equal(t, "src/file.txt", targets[0].FileMappings[0].Src)
}

// TestModelsHooks_ValidationErrors tests validation failures in hooks (0-66% coverage)
func TestModelsHooks_ValidationErrors(t *testing.T) {
	db := TestDB(t)

	t.Run("Group BeforeCreate missing external_id", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		group := &Group{
			ConfigID: config.ID,
			// Missing ExternalID
			Name: "Test Group",
		}
		err = db.Create(group).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "external_id")
	})

	t.Run("FileList BeforeCreate missing external_id", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config-2",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		fileList := &FileList{
			ConfigID: config.ID,
			// Missing ExternalID
			Name: "Test File List",
		}
		err = db.Create(fileList).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "external_id")
	})

	t.Run("DirectoryList BeforeCreate missing external_id", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config-3",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		dirList := &DirectoryList{
			ConfigID: config.ID,
			// Missing ExternalID
			Name: "Test Dir List",
		}
		err = db.Create(dirList).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "external_id")
	})

	t.Run("Target BeforeCreate invalid repo", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config-4",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "test-group",
			Name:       "Test Group",
		}
		err = db.Create(group).Error
		require.NoError(t, err)

		target := &Target{
			GroupID: group.ID,
			Repo:    "invalid repo format", // Invalid repo name
		}
		err = db.Create(target).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo")
	})
}

// TestModelsHooks_BeforeUpdate tests update validation (0% coverage)
func TestModelsHooks_BeforeUpdate(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	t.Run("Target BeforeUpdate validation", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "test-group",
			Name:       "Test Group",
		}
		err = db.Create(group).Error
		require.NoError(t, err)

		target := &Target{
			GroupID: group.ID,
			Repo:    "mrz1836/test",
		}
		err = db.Create(target).Error
		require.NoError(t, err)

		// Try to update to invalid state
		target.Repo = "" // Invalid: empty repo
		err = db.WithContext(ctx).Save(target).Error
		assert.Error(t, err)
	})

	t.Run("FileMapping BeforeUpdate validation", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config-fm-update",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "test-group-fm-update",
			Name:       "Test Group",
		}
		err = db.Create(group).Error
		require.NoError(t, err)

		target := &Target{
			GroupID: group.ID,
			Repo:    "mrz1836/test",
		}
		err = db.Create(target).Error
		require.NoError(t, err)

		mapping := &FileMapping{
			OwnerType: "target",
			OwnerID:   target.ID,
			Src:       "src/file.txt",
			Dest:      "dest/file.txt",
		}
		err = db.Create(mapping).Error
		require.NoError(t, err)

		// Update should still validate
		mapping.DeleteFlag = true // Setting delete flag with src is valid
		err = db.WithContext(ctx).Save(mapping).Error
		require.NoError(t, err)
	})
}

// TestJSONModuleConfigScan_Error tests Scan error paths (66.7% coverage)
func TestJSONModuleConfigScan_Error(t *testing.T) {
	var jmc JSONModuleConfig

	// Test nil value
	err := jmc.Scan(nil)
	require.NoError(t, err)

	// Test invalid JSON
	err = jmc.Scan([]byte(`{invalid json`))
	assert.Error(t, err)
}

// TestJSONStringSliceScan_Error tests Scan error paths (66.7% coverage)
func TestJSONStringSliceScan_Error(t *testing.T) {
	var jss JSONStringSlice

	// Test nil value
	err := jss.Scan(nil)
	require.NoError(t, err)

	// Test invalid JSON
	err = jss.Scan([]byte(`[invalid json`))
	assert.Error(t, err)
}

// TestJSONStringMapScan_Error tests Scan error paths (66.7% coverage)
func TestJSONStringMapScan_Error(t *testing.T) {
	var jsm JSONStringMap

	// Test nil value
	err := jsm.Scan(nil)
	require.NoError(t, err)

	// Test invalid JSON
	err = jsm.Scan([]byte(`{invalid: json}`))
	assert.Error(t, err)
}

// TestMetadataScan_Error tests Scan error paths (66.7% coverage)
func TestMetadataScan_Error(t *testing.T) {
	var meta Metadata

	// Test nil value
	err := meta.Scan(nil)
	require.NoError(t, err)

	// Test invalid JSON
	err = meta.Scan([]byte(`{bad json`))
	assert.Error(t, err)
}

// TestTargetRepositoryReferences tests file/directory list ref operations (66-75% coverage)
func TestTargetRepositoryReferences(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Setup
	config := &Config{
		ExternalID: "test-config",
		Name:       "Test",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	fileList := &FileList{
		ConfigID:   config.ID,
		ExternalID: "file-list-1",
		Name:       "File List",
	}
	err = db.Create(fileList).Error
	require.NoError(t, err)

	dirList := &DirectoryList{
		ConfigID:   config.ID,
		ExternalID: "dir-list-1",
		Name:       "Dir List",
	}
	err = db.Create(dirList).Error
	require.NoError(t, err)

	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	target := &Target{
		GroupID: group.ID,
		Repo:    "mrz1836/test",
	}
	err = db.Create(target).Error
	require.NoError(t, err)

	repo := NewTargetRepository(db)

	// Test AddFileListRef
	err = repo.AddFileListRef(ctx, target.ID, fileList.ID, 0)
	require.NoError(t, err)

	// Test RemoveFileListRef
	err = repo.RemoveFileListRef(ctx, target.ID, fileList.ID)
	require.NoError(t, err)

	// Test AddDirectoryListRef
	err = repo.AddDirectoryListRef(ctx, target.ID, dirList.ID, 0)
	require.NoError(t, err)

	// Test RemoveDirectoryListRef
	err = repo.RemoveDirectoryListRef(ctx, target.ID, dirList.ID)
	require.NoError(t, err)
}

// TestConverterImport_EdgeCases tests import edge cases (58-82% coverage improvement)
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
