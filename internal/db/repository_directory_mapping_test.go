package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoryMappingRepository_Create(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewDirectoryMappingRepository(gormDB)
	ctx := context.Background()

	preserveStructure := true
	mapping := &DirectoryMapping{
		OwnerType:         "directory_list",
		OwnerID:           seed.DirectoryLists[0].ID,
		Src:               "src/dir",
		Dest:              "dest/dir",
		Exclude:           JSONStringSlice{"*.tmp"},
		PreserveStructure: &preserveStructure,
		Position:          0,
	}

	err := repo.Create(ctx, mapping)
	require.NoError(t, err)
	assert.NotZero(t, mapping.ID)
}

func TestDirectoryMappingRepository_Delete(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewDirectoryMappingRepository(gormDB)
	ctx := context.Background()

	t.Run("soft delete", func(t *testing.T) {
		mapping := &DirectoryMapping{
			OwnerType: "directory_list",
			OwnerID:   seed.DirectoryLists[0].ID,
			Src:       "soft-del",
			Dest:      "soft-del",
			Position:  10,
		}
		require.NoError(t, repo.Create(ctx, mapping))

		err := repo.Delete(ctx, mapping.ID, false)
		require.NoError(t, err)

		_, err = repo.FindByDest(ctx, "directory_list", seed.DirectoryLists[0].ID, "soft-del")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("hard delete", func(t *testing.T) {
		mapping := &DirectoryMapping{
			OwnerType: "directory_list",
			OwnerID:   seed.DirectoryLists[0].ID,
			Src:       "hard-del",
			Dest:      "hard-del",
			Position:  11,
		}
		require.NoError(t, repo.Create(ctx, mapping))

		err := repo.Delete(ctx, mapping.ID, true)
		require.NoError(t, err)

		_, err = repo.FindByDest(ctx, "directory_list", seed.DirectoryLists[0].ID, "hard-del")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}

func TestDirectoryMappingRepository_ListByOwner(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewDirectoryMappingRepository(gormDB)
	ctx := context.Background()

	// Seed data has a directory mapping on target[1]
	mappings, err := repo.ListByOwner(ctx, "target", seed.Targets[1].ID)
	require.NoError(t, err)
	assert.Len(t, mappings, 1)
	assert.Equal(t, ".github/workflows", mappings[0].Dest)
}

func TestDirectoryMappingRepository_ListByOwner_Empty(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewDirectoryMappingRepository(gormDB)
	ctx := context.Background()

	mappings, err := repo.ListByOwner(ctx, "target", 9999)
	require.NoError(t, err)
	assert.Empty(t, mappings)
}

func TestDirectoryMappingRepository_FindByDest(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewDirectoryMappingRepository(gormDB)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		mapping, err := repo.FindByDest(ctx, "target", seed.Targets[1].ID, ".github/workflows")
		require.NoError(t, err)
		assert.Equal(t, ".github/workflows", mapping.Src)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.FindByDest(ctx, "target", seed.Targets[1].ID, "nonexistent")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}
