package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepositoryGetByExternalID_NotFound tests error paths
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

// TestRepositoryGetByID_NotFound tests GetByID error paths
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

// TestRepositoryDelete_HardDelete tests hard delete paths
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

// TestRepositoryListWithDirectories tests DirectoryList preloading
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

// TestRepositoryListWithAssociations tests Target preloading
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

// TestTargetRepositoryReferences tests file/directory list ref operations
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

// TestRepositoryList_Empty tests list operations on empty tables
func TestRepositoryList_Empty(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create config
	config := &Config{
		ExternalID: "empty-list-test",
		Name:       "Empty List Test",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	t.Run("List empty file lists", func(t *testing.T) {
		repo := NewFileListRepository(db)
		lists, err := repo.List(ctx, config.ID)
		require.NoError(t, err)
		assert.Empty(t, lists)
	})

	t.Run("List empty directory lists", func(t *testing.T) {
		repo := NewDirectoryListRepository(db)
		lists, err := repo.List(ctx, config.ID)
		require.NoError(t, err)
		assert.Empty(t, lists)
	})

	t.Run("List empty configs", func(t *testing.T) {
		repo := NewConfigRepository(db)
		configs, err := repo.List(ctx)
		require.NoError(t, err)
		// At least one config exists (the one we just created)
		assert.NotEmpty(t, configs)
	})
}

// TestRepositorySoftDelete tests soft delete functionality
func TestRepositorySoftDelete(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create config
	config := &Config{
		ExternalID: "soft-delete-test",
		Name:       "Soft Delete Test",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	t.Run("Soft delete group", func(t *testing.T) {
		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "group-soft-delete",
			Name:       "Group To Soft Delete",
		}
		err := db.Create(group).Error
		require.NoError(t, err)

		repo := NewGroupRepository(db)

		// Soft delete (hard=false)
		err = repo.Delete(ctx, group.ID, false)
		require.NoError(t, err)

		// Should not appear in normal queries
		_, err = repo.GetByID(ctx, group.ID)
		require.ErrorIs(t, err, ErrRecordNotFound)

		// But should exist with Unscoped
		var count int64
		db.Unscoped().Model(&Group{}).Where("id = ?", group.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Soft delete file list", func(t *testing.T) {
		fileList := &FileList{
			ConfigID:   config.ID,
			ExternalID: "filelist-soft-delete",
			Name:       "FileList To Soft Delete",
		}
		err := db.Create(fileList).Error
		require.NoError(t, err)

		repo := NewFileListRepository(db)
		err = repo.Delete(ctx, fileList.ID, false)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, fileList.ID)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Soft delete directory list", func(t *testing.T) {
		dirList := &DirectoryList{
			ConfigID:   config.ID,
			ExternalID: "dirlist-soft-delete",
			Name:       "DirList To Soft Delete",
		}
		err := db.Create(dirList).Error
		require.NoError(t, err)

		repo := NewDirectoryListRepository(db)
		err = repo.Delete(ctx, dirList.ID, false)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, dirList.ID)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}

// TestRepositoryCreate_DuplicateExternalID tests unique constraint handling
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

// TestRepositoryUpdate_EdgeCases tests update edge cases
func TestRepositoryUpdate_EdgeCases(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create config
	config := &Config{
		ExternalID: "update-edge-test",
		Name:       "Test",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	t.Run("Update FileList name", func(t *testing.T) {
		fileList := &FileList{
			ConfigID:   config.ID,
			ExternalID: "file-list-update",
			Name:       "Original Name",
		}
		err := db.Create(fileList).Error
		require.NoError(t, err)

		repo := NewFileListRepository(db)

		fileList.Name = "Updated Name"
		fileList.Description = "New description"
		err = repo.Update(ctx, fileList)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, fileList.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", fetched.Name)
		assert.Equal(t, "New description", fetched.Description)
	})

	t.Run("Update DirectoryList position", func(t *testing.T) {
		dirList := &DirectoryList{
			ConfigID:   config.ID,
			ExternalID: "dir-list-update",
			Name:       "Test Dir List",
			Position:   0,
		}
		err := db.Create(dirList).Error
		require.NoError(t, err)

		repo := NewDirectoryListRepository(db)

		dirList.Position = 5
		err = repo.Update(ctx, dirList)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, dirList.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, fetched.Position)
	})

	t.Run("Update Target metadata", func(t *testing.T) {
		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "update-group",
			Name:       "Update Group",
		}
		err := db.Create(group).Error
		require.NoError(t, err)

		target := &Target{
			BaseModel: BaseModel{
				Metadata: Metadata{"key": "value"},
			},
			GroupID: group.ID,
			Repo:    "mrz1836/test",
		}
		err = db.Create(target).Error
		require.NoError(t, err)

		repo := NewTargetRepository(db)

		target.Metadata = Metadata{"key": "updated", "new_key": "new_value"}
		err = repo.Update(ctx, target)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, target.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated", fetched.Metadata["key"])
		assert.Equal(t, "new_value", fetched.Metadata["new_key"])
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
