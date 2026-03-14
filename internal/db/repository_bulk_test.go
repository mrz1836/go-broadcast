package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkRepository_AddFileListToAllTargets(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewBulkRepository(gormDB)
	ctx := context.Background()

	// Create a new file list to add
	newFL := &FileList{
		ConfigID:   seed.Config.ID,
		ExternalID: "bulk-test-fl",
		Name:       "Bulk Test",
		Position:   10,
	}
	require.NoError(t, gormDB.Create(newFL).Error)

	affected, err := repo.AddFileListToAllTargets(ctx, seed.Groups[0].ID, newFL.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, affected) // Both targets in group

	// Verify refs created
	var refs []TargetFileListRef
	require.NoError(t, gormDB.Where("file_list_id = ?", newFL.ID).Find(&refs).Error)
	assert.Len(t, refs, 2)
}

func TestBulkRepository_AddFileListToAllTargets_Idempotent(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewBulkRepository(gormDB)
	ctx := context.Background()

	// ai-files is already attached to target[0] via seed data
	// Adding it to all should skip target[0] and only add to target[1]
	affected, err := repo.AddFileListToAllTargets(ctx, seed.Groups[0].ID, seed.FileLists[0].ID)
	require.NoError(t, err)
	assert.Equal(t, 1, affected) // Only target[1] got the ref

	// Running again should be fully idempotent
	affected2, err := repo.AddFileListToAllTargets(ctx, seed.Groups[0].ID, seed.FileLists[0].ID)
	require.NoError(t, err)
	assert.Equal(t, 0, affected2)
}

func TestBulkRepository_RemoveFileListFromAllTargets(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewBulkRepository(gormDB)
	ctx := context.Background()

	// First add to all, then remove from all
	newFL := &FileList{
		ConfigID:   seed.Config.ID,
		ExternalID: "bulk-remove-fl",
		Name:       "Bulk Remove",
		Position:   11,
	}
	require.NoError(t, gormDB.Create(newFL).Error)

	_, err := repo.AddFileListToAllTargets(ctx, seed.Groups[0].ID, newFL.ID)
	require.NoError(t, err)

	affected, err := repo.RemoveFileListFromAllTargets(ctx, seed.Groups[0].ID, newFL.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, affected)

	// Verify refs removed
	var refs []TargetFileListRef
	require.NoError(t, gormDB.Where("file_list_id = ?", newFL.ID).Find(&refs).Error)
	assert.Empty(t, refs)
}

func TestBulkRepository_RemoveFileListFromAllTargets_NoRefs(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewBulkRepository(gormDB)
	ctx := context.Background()

	newFL := &FileList{
		ConfigID:   seed.Config.ID,
		ExternalID: "bulk-no-refs",
		Name:       "No Refs",
		Position:   12,
	}
	require.NoError(t, gormDB.Create(newFL).Error)

	affected, err := repo.RemoveFileListFromAllTargets(ctx, seed.Groups[0].ID, newFL.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, affected)
}

func TestBulkRepository_AddDirectoryListToAllTargets(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewBulkRepository(gormDB)
	ctx := context.Background()

	newDL := &DirectoryList{
		ConfigID:   seed.Config.ID,
		ExternalID: "bulk-test-dl",
		Name:       "Bulk Dir Test",
		Position:   10,
	}
	require.NoError(t, gormDB.Create(newDL).Error)

	affected, err := repo.AddDirectoryListToAllTargets(ctx, seed.Groups[0].ID, newDL.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, affected)

	var refs []TargetDirectoryListRef
	require.NoError(t, gormDB.Where("directory_list_id = ?", newDL.ID).Find(&refs).Error)
	assert.Len(t, refs, 2)
}

func TestBulkRepository_AddDirectoryListToAllTargets_Idempotent(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewBulkRepository(gormDB)
	ctx := context.Background()

	// github-workflows is already attached to target[1]
	affected, err := repo.AddDirectoryListToAllTargets(ctx, seed.Groups[0].ID, seed.DirectoryLists[0].ID)
	require.NoError(t, err)
	assert.Equal(t, 1, affected) // Only target[0] got it

	affected2, err := repo.AddDirectoryListToAllTargets(ctx, seed.Groups[0].ID, seed.DirectoryLists[0].ID)
	require.NoError(t, err)
	assert.Equal(t, 0, affected2)
}

func TestBulkRepository_RemoveDirectoryListFromAllTargets(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewBulkRepository(gormDB)
	ctx := context.Background()

	newDL := &DirectoryList{
		ConfigID:   seed.Config.ID,
		ExternalID: "bulk-remove-dl",
		Name:       "Bulk Remove Dir",
		Position:   11,
	}
	require.NoError(t, gormDB.Create(newDL).Error)

	_, err := repo.AddDirectoryListToAllTargets(ctx, seed.Groups[0].ID, newDL.ID)
	require.NoError(t, err)

	affected, err := repo.RemoveDirectoryListFromAllTargets(ctx, seed.Groups[0].ID, newDL.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, affected)

	var refs []TargetDirectoryListRef
	require.NoError(t, gormDB.Where("directory_list_id = ?", newDL.ID).Find(&refs).Error)
	assert.Empty(t, refs)
}

func TestBulkRepository_EmptyGroup(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewBulkRepository(gormDB)
	ctx := context.Background()

	// Create an empty group
	enabled := true
	emptyGroup := &Group{
		ConfigID:   seed.Config.ID,
		ExternalID: "empty-group",
		Name:       "Empty",
		Enabled:    &enabled,
		Position:   10,
	}
	require.NoError(t, gormDB.Create(emptyGroup).Error)

	affected, err := repo.AddFileListToAllTargets(ctx, emptyGroup.ID, seed.FileLists[0].ID)
	require.NoError(t, err)
	assert.Equal(t, 0, affected)

	affected, err = repo.RemoveFileListFromAllTargets(ctx, emptyGroup.ID, seed.FileLists[0].ID)
	require.NoError(t, err)
	assert.Equal(t, 0, affected)
}
