package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileMappingRepository_Create(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewFileMappingRepository(gormDB)
	ctx := context.Background()

	mapping := &FileMapping{
		OwnerType:  "file_list",
		OwnerID:    seed.FileLists[0].ID,
		Src:        "new-file.txt",
		Dest:       "new-file.txt",
		DeleteFlag: false,
		Position:   0,
	}

	err := repo.Create(ctx, mapping)
	require.NoError(t, err)
	assert.NotZero(t, mapping.ID)
}

func TestFileMappingRepository_Delete(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewFileMappingRepository(gormDB)
	ctx := context.Background()

	t.Run("soft delete", func(t *testing.T) {
		mapping := &FileMapping{
			OwnerType: "file_list",
			OwnerID:   seed.FileLists[0].ID,
			Src:       "delete-soft.txt",
			Dest:      "delete-soft.txt",
			Position:  10,
		}
		require.NoError(t, repo.Create(ctx, mapping))

		err := repo.Delete(ctx, mapping.ID, false)
		require.NoError(t, err)

		// Soft-deleted should not appear in queries
		_, err = repo.FindByDest(ctx, "file_list", seed.FileLists[0].ID, "delete-soft.txt")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("hard delete", func(t *testing.T) {
		mapping := &FileMapping{
			OwnerType: "file_list",
			OwnerID:   seed.FileLists[0].ID,
			Src:       "delete-hard.txt",
			Dest:      "delete-hard.txt",
			Position:  11,
		}
		require.NoError(t, repo.Create(ctx, mapping))

		err := repo.Delete(ctx, mapping.ID, true)
		require.NoError(t, err)

		_, err = repo.FindByDest(ctx, "file_list", seed.FileLists[0].ID, "delete-hard.txt")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}

func TestFileMappingRepository_ListByOwner(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewFileMappingRepository(gormDB)
	ctx := context.Background()

	// Seed data already has file mappings on target[0]
	mappings, err := repo.ListByOwner(ctx, "target", seed.Targets[0].ID)
	require.NoError(t, err)
	assert.Len(t, mappings, 2) // .cursorrules and codecov.yml

	// Verify ordering by position
	assert.Equal(t, ".cursorrules", mappings[0].Dest)
	assert.Equal(t, "codecov.yml", mappings[1].Dest)
}

func TestFileMappingRepository_ListByOwner_Empty(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewFileMappingRepository(gormDB)
	ctx := context.Background()

	mappings, err := repo.ListByOwner(ctx, "target", 9999)
	require.NoError(t, err)
	assert.Empty(t, mappings)
}

func TestFileMappingRepository_FindByDest(t *testing.T) {
	gormDB, seed := TestDBWithSeed(t)
	repo := NewFileMappingRepository(gormDB)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		mapping, err := repo.FindByDest(ctx, "target", seed.Targets[0].ID, ".cursorrules")
		require.NoError(t, err)
		assert.Equal(t, ".cursorrules", mapping.Src)
		assert.Equal(t, ".cursorrules", mapping.Dest)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.FindByDest(ctx, "target", seed.Targets[0].ID, "nonexistent.txt")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("wrong owner type", func(t *testing.T) {
		_, err := repo.FindByDest(ctx, "file_list", seed.Targets[0].ID, ".cursorrules")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}
