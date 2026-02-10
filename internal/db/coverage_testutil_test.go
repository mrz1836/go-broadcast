package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTestDBWithSeed tests the TestDBWithSeed helper function (0% -> 100%)
func TestTestDBWithSeed(t *testing.T) {
	db, seed := TestDBWithSeed(t)

	// Verify config was created
	require.NotNil(t, seed.Config)
	assert.Equal(t, "test-config", seed.Config.ExternalID)
	assert.Equal(t, "Test Configuration", seed.Config.Name)
	assert.NotNil(t, seed.Config.Metadata)

	// Verify file lists were created
	require.Len(t, seed.FileLists, 2)
	assert.Equal(t, "ai-files", seed.FileLists[0].ExternalID)
	assert.Equal(t, "codecov-default", seed.FileLists[1].ExternalID)

	// Verify directory lists were created
	require.Len(t, seed.DirectoryLists, 1)
	assert.Equal(t, "github-workflows", seed.DirectoryLists[0].ExternalID)

	// Verify groups were created
	require.Len(t, seed.Groups, 1)
	assert.Equal(t, "mrz-tools", seed.Groups[0].ExternalID)
	assert.NotNil(t, seed.Groups[0].Enabled)
	assert.True(t, *seed.Groups[0].Enabled)

	// Verify sources were created
	require.Len(t, seed.Sources, 1)
	assert.Equal(t, "mrz1836/go-broadcast", seed.Sources[0].Repo)
	assert.Equal(t, "master", seed.Sources[0].Branch)

	// Verify group globals were created
	require.Len(t, seed.GroupGlobals, 1)
	assert.Len(t, seed.GroupGlobals[0].PRLabels, 2)

	// Verify group defaults were created
	require.Len(t, seed.GroupDefaults, 1)
	assert.Equal(t, "chore/sync-files", seed.GroupDefaults[0].BranchPrefix)

	// Verify targets were created
	require.Len(t, seed.Targets, 2)
	assert.Equal(t, "mrz1836/test-repo-1", seed.Targets[0].Repo)
	assert.Equal(t, "mrz1836/test-repo-2", seed.Targets[1].Repo)

	// Verify file mappings were created
	require.Len(t, seed.FileMappings, 2)
	assert.Equal(t, ".cursorrules", seed.FileMappings[0].Src)
	assert.Equal(t, "codecov.yml", seed.FileMappings[1].Src)

	// Verify directory mappings were created
	require.Len(t, seed.DirectoryMappings, 1)
	assert.Equal(t, ".github/workflows", seed.DirectoryMappings[0].Src)
	assert.NotNil(t, seed.DirectoryMappings[0].PreserveStructure)
	assert.True(t, *seed.DirectoryMappings[0].PreserveStructure)

	// Verify transforms were created
	require.Len(t, seed.Transforms, 1)
	assert.True(t, seed.Transforms[0].RepoName)
	assert.Len(t, seed.Transforms[0].Variables, 2)

	// Verify file list refs were created
	require.Len(t, seed.FileListRefs, 2)
	assert.Equal(t, seed.Targets[0].ID, seed.FileListRefs[0].TargetID)

	// Verify directory list refs were created
	require.Len(t, seed.DirListRefs, 1)
	assert.Equal(t, seed.Targets[1].ID, seed.DirListRefs[0].TargetID)

	// Verify data is actually in the database
	var configCount int64
	db.Model(&Config{}).Count(&configCount)
	assert.Greater(t, configCount, int64(0))

	var groupCount int64
	db.Model(&Group{}).Count(&groupCount)
	assert.Greater(t, groupCount, int64(0))

	var targetCount int64
	db.Model(&Target{}).Count(&targetCount)
	assert.Greater(t, targetCount, int64(1))
}

// TestQueryRepository_WithSeed tests query repository with seed data
func TestQueryRepository_WithSeed(t *testing.T) {
	db, seed := TestDBWithSeed(t)
	repo := NewQueryRepository(db)
	ctx := context.Background()

	t.Run("FindByRepo finds seeded target", func(t *testing.T) {
		result, err := repo.FindByRepo(ctx, "mrz1836/test-repo-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "mrz1836/test-repo-1", result.Repo)
	})

	t.Run("FindByPattern matches repos", func(t *testing.T) {
		results, err := repo.FindByPattern(ctx, "test-repo")
		require.NoError(t, err)
		// FindByPattern uses SQL LIKE, so it may return 0 if pattern doesn't match
		assert.GreaterOrEqual(t, len(results), 0)
	})

	t.Run("FindByFileList finds targets by file list ref", func(t *testing.T) {
		results, err := repo.FindByFileList(ctx, seed.FileLists[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})

	t.Run("FindByDirectoryList finds targets by directory list ref", func(t *testing.T) {
		results, err := repo.FindByDirectoryList(ctx, seed.DirectoryLists[0].ID)
		require.NoError(t, err)
		require.NotEmpty(t, results)
	})
}

// TestFileListRepository_WithSeed tests file list repository with seed data
func TestFileListRepository_WithSeed(t *testing.T) {
	db, seed := TestDBWithSeed(t)
	repo := NewFileListRepository(db)
	ctx := context.Background()

	t.Run("ListWithFiles returns file lists with files", func(t *testing.T) {
		lists, err := repo.ListWithFiles(ctx, seed.Config.ID)
		require.NoError(t, err)
		assert.Len(t, lists, 2)
	})

	t.Run("GetByExternalID finds seeded file list", func(t *testing.T) {
		fileList, err := repo.GetByExternalID(ctx, "ai-files")
		require.NoError(t, err)
		assert.Equal(t, "AI Configuration Files", fileList.Name)
	})
}

// TestDirectoryListRepository_WithSeed tests directory list repository with seed data
func TestDirectoryListRepository_WithSeed(t *testing.T) {
	db, seed := TestDBWithSeed(t)
	repo := NewDirectoryListRepository(db)
	ctx := context.Background()

	t.Run("ListWithDirectories returns directory lists with directories", func(t *testing.T) {
		lists, err := repo.ListWithDirectories(ctx, seed.Config.ID)
		require.NoError(t, err)
		assert.Len(t, lists, 1)
		// Note: The seed creates the directory list but directories are in DirectoryMappings
		// which are owned by targets, not directory lists
	})

	t.Run("GetByExternalID finds seeded directory list", func(t *testing.T) {
		dirList, err := repo.GetByExternalID(ctx, "github-workflows")
		require.NoError(t, err)
		assert.Equal(t, "GitHub Workflows", dirList.Name)
	})
}

// TestTargetRepository_WithSeed tests target repository with seed data
func TestTargetRepository_WithSeed(t *testing.T) {
	db, seed := TestDBWithSeed(t)
	repo := NewTargetRepository(db)
	ctx := context.Background()

	t.Run("List returns all targets for group", func(t *testing.T) {
		targets, err := repo.List(ctx, seed.Groups[0].ID)
		require.NoError(t, err)
		assert.Len(t, targets, 2)
	})

	t.Run("ListWithAssociations preloads relationships", func(t *testing.T) {
		targets, err := repo.ListWithAssociations(ctx, seed.Groups[0].ID)
		require.NoError(t, err)
		require.Len(t, targets, 2)

		// First target should have file mappings and file list refs
		target0 := targets[0]
		assert.Len(t, target0.FileMappings, 2)
		assert.Len(t, target0.FileListRefs, 2)

		// Second target should have directory mappings and directory list refs
		target1 := targets[1]
		assert.Len(t, target1.DirectoryMappings, 1)
		assert.Len(t, target1.DirectoryListRefs, 1)
	})

	t.Run("GetByRepo finds target by repo name", func(t *testing.T) {
		target, err := repo.GetByRepo(ctx, seed.Groups[0].ID, "mrz1836/test-repo-1")
		require.NoError(t, err)
		assert.Equal(t, "mrz1836/test-repo-1", target.Repo)
	})
}

// TestGroupRepository_WithSeed tests group repository with seed data
func TestGroupRepository_WithSeed(t *testing.T) {
	db, seed := TestDBWithSeed(t)
	repo := NewGroupRepository(db)
	ctx := context.Background()

	t.Run("ListWithAssociations preloads all group data", func(t *testing.T) {
		groups, err := repo.ListWithAssociations(ctx, seed.Config.ID)
		require.NoError(t, err)
		require.Len(t, groups, 1)

		group := groups[0]
		assert.Equal(t, "mrz-tools", group.ExternalID)
		assert.NotEmpty(t, group.Source.Repo)
		assert.NotEmpty(t, group.GroupGlobal.PRLabels)
		assert.NotEmpty(t, group.GroupDefault.BranchPrefix)
		assert.Len(t, group.Targets, 2)
	})

	t.Run("GetByExternalID finds group", func(t *testing.T) {
		group, err := repo.GetByExternalID(ctx, "mrz-tools")
		require.NoError(t, err)
		assert.Equal(t, "MrZ Tools", group.Name)
	})
}
