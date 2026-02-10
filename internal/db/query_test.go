package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryRepository_FindByFile_InlineMapping(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	repo := NewQueryRepository(db)

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	target := &Target{GroupID: group.ID, Repo: "mrz1836/test-repo"}
	require.NoError(t, db.Create(target).Error)

	// Create file mapping
	fileMapping := &FileMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       ".github/workflows/ci.yml",
		Dest:      ".github/workflows/ci.yml",
	}
	require.NoError(t, db.Create(fileMapping).Error)

	// Query by file
	targets, err := repo.FindByFile(ctx, ".github/workflows/ci.yml")
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "mrz1836/test-repo", targets[0].Repo)
}

func TestQueryRepository_FindByFile_ViaFileList(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	repo := NewQueryRepository(db)

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	target := &Target{GroupID: group.ID, Repo: "mrz1836/test-repo"}
	require.NoError(t, db.Create(target).Error)

	// Create file list with mapping
	fileList := &FileList{
		ConfigID:   config.ID,
		ExternalID: "ai-files",
		Name:       "AI Files",
	}
	require.NoError(t, db.Create(fileList).Error)

	fileMapping := &FileMapping{
		OwnerType: "file_list",
		OwnerID:   fileList.ID,
		Src:       "README.md",
		Dest:      "README.md",
	}
	require.NoError(t, db.Create(fileMapping).Error)

	// Create reference from target to file list
	ref := &TargetFileListRef{
		TargetID:   target.ID,
		FileListID: fileList.ID,
		Position:   0,
	}
	require.NoError(t, db.Create(ref).Error)

	// Query by file
	targets, err := repo.FindByFile(ctx, "README.md")
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "mrz1836/test-repo", targets[0].Repo)
}

func TestQueryRepository_FindByFile_MultipleTargets(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	repo := NewQueryRepository(db)

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	// Create two targets with the same file
	target1 := &Target{GroupID: group.ID, Repo: "mrz1836/repo1"}
	require.NoError(t, db.Create(target1).Error)

	target2 := &Target{GroupID: group.ID, Repo: "mrz1836/repo2"}
	require.NoError(t, db.Create(target2).Error)

	// Create file mappings for both
	fileMapping1 := &FileMapping{
		OwnerType: "target",
		OwnerID:   target1.ID,
		Src:       "LICENSE",
		Dest:      "LICENSE",
	}
	require.NoError(t, db.Create(fileMapping1).Error)

	fileMapping2 := &FileMapping{
		OwnerType: "target",
		OwnerID:   target2.ID,
		Src:       "LICENSE",
		Dest:      "LICENSE",
	}
	require.NoError(t, db.Create(fileMapping2).Error)

	// Query by file
	targets, err := repo.FindByFile(ctx, "LICENSE")
	require.NoError(t, err)
	require.Len(t, targets, 2)

	repos := []string{targets[0].Repo, targets[1].Repo}
	assert.Contains(t, repos, "mrz1836/repo1")
	assert.Contains(t, repos, "mrz1836/repo2")
}

func TestQueryRepository_FindByFile_BySrc(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	repo := NewQueryRepository(db)

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	target := &Target{GroupID: group.ID, Repo: "mrz1836/test-repo"}
	require.NoError(t, db.Create(target).Error)

	// Create file mapping with different src and dest
	fileMapping := &FileMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       "source/config.yml",
		Dest:      "config/app.yml",
	}
	require.NoError(t, db.Create(fileMapping).Error)

	// Query by src
	targets, err := repo.FindByFile(ctx, "source/config.yml")
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "mrz1836/test-repo", targets[0].Repo)

	// Query by dest
	targets2, err := repo.FindByFile(ctx, "config/app.yml")
	require.NoError(t, err)
	require.Len(t, targets2, 1)
	assert.Equal(t, "mrz1836/test-repo", targets2[0].Repo)
}

func TestQueryRepository_FindByRepo(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	repo := NewQueryRepository(db)

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	target := &Target{GroupID: group.ID, Repo: "mrz1836/test-repo"}
	require.NoError(t, db.Create(target).Error)

	// Create file mapping
	fileMapping := &FileMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       "README.md",
		Dest:      "README.md",
	}
	require.NoError(t, db.Create(fileMapping).Error)

	// Create directory mapping
	dirMapping := &DirectoryMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       ".github/workflows",
		Dest:      ".github/workflows",
	}
	require.NoError(t, db.Create(dirMapping).Error)

	// Query by repo
	result, err := repo.FindByRepo(ctx, "mrz1836/test-repo")
	require.NoError(t, err)
	assert.Equal(t, target.ID, result.ID)
	assert.Len(t, result.FileMappings, 1)
	assert.Equal(t, "README.md", result.FileMappings[0].Dest)
	assert.Len(t, result.DirectoryMappings, 1)
	assert.Equal(t, ".github/workflows", result.DirectoryMappings[0].Dest)
}

func TestQueryRepository_FindByRepo_NotFound(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	repo := NewQueryRepository(db)

	// Query for non-existent repo
	_, err := repo.FindByRepo(ctx, "mrz1836/non-existent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRecordNotFound)
}

func TestQueryRepository_FindByFileList(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	repo := NewQueryRepository(db)

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	fileList := &FileList{
		ConfigID:   config.ID,
		ExternalID: "ai-files",
		Name:       "AI Files",
	}
	require.NoError(t, db.Create(fileList).Error)

	target1 := &Target{GroupID: group.ID, Repo: "mrz1836/repo1"}
	require.NoError(t, db.Create(target1).Error)

	target2 := &Target{GroupID: group.ID, Repo: "mrz1836/repo2"}
	require.NoError(t, db.Create(target2).Error)

	// Create references
	ref1 := &TargetFileListRef{TargetID: target1.ID, FileListID: fileList.ID, Position: 0}
	require.NoError(t, db.Create(ref1).Error)

	ref2 := &TargetFileListRef{TargetID: target2.ID, FileListID: fileList.ID, Position: 0}
	require.NoError(t, db.Create(ref2).Error)

	// Query by file list
	targets, err := repo.FindByFileList(ctx, fileList.ID)
	require.NoError(t, err)
	require.Len(t, targets, 2)

	repos := []string{targets[0].Repo, targets[1].Repo}
	assert.Contains(t, repos, "mrz1836/repo1")
	assert.Contains(t, repos, "mrz1836/repo2")
}

func TestQueryRepository_FindByDirectoryList(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	repo := NewQueryRepository(db)

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	dirList := &DirectoryList{
		ConfigID:   config.ID,
		ExternalID: "workflows",
		Name:       "Workflow Directories",
	}
	require.NoError(t, db.Create(dirList).Error)

	target := &Target{GroupID: group.ID, Repo: "mrz1836/test-repo"}
	require.NoError(t, db.Create(target).Error)

	// Create reference
	ref := &TargetDirectoryListRef{TargetID: target.ID, DirectoryListID: dirList.ID, Position: 0}
	require.NoError(t, db.Create(ref).Error)

	// Query by directory list
	targets, err := repo.FindByDirectoryList(ctx, dirList.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "mrz1836/test-repo", targets[0].Repo)
}

func TestQueryRepository_FindByPattern(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	repo := NewQueryRepository(db)

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	target := &Target{GroupID: group.ID, Repo: "mrz1836/test-repo"}
	require.NoError(t, db.Create(target).Error)

	// Create multiple file mappings
	mappings := []string{
		".github/workflows/ci.yml",
		".github/workflows/release.yml",
		".github/CODEOWNERS",
		"README.md",
	}
	for i, dest := range mappings {
		fm := &FileMapping{
			OwnerType: "target",
			OwnerID:   target.ID,
			Src:       dest,
			Dest:      dest,
			Position:  i,
		}
		require.NoError(t, db.Create(fm).Error)
	}

	// Query by pattern
	results, err := repo.FindByPattern(ctx, "workflows")
	require.NoError(t, err)
	require.Len(t, results, 2)

	dests := []string{results[0].Dest, results[1].Dest}
	assert.Contains(t, dests, ".github/workflows/ci.yml")
	assert.Contains(t, dests, ".github/workflows/release.yml")

	// Query by another pattern
	results2, err := repo.FindByPattern(ctx, ".github")
	require.NoError(t, err)
	assert.Len(t, results2, 3) // All .github files

	// Query with no matches
	results3, err := repo.FindByPattern(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, results3)
}

func TestQueryRepository_FindByPattern_MatchesSrc(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	repo := NewQueryRepository(db)

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	target := &Target{GroupID: group.ID, Repo: "mrz1836/test-repo"}
	require.NoError(t, db.Create(target).Error)

	// Create file mapping with src
	fm := &FileMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       "templates/config.yml",
		Dest:      "config/app.yml",
	}
	require.NoError(t, db.Create(fm).Error)

	// Query by src pattern
	results, err := repo.FindByPattern(ctx, "templates")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "templates/config.yml", results[0].Src)

	// Query by dest pattern
	results2, err := repo.FindByPattern(ctx, "config")
	require.NoError(t, err)
	require.Len(t, results2, 1)
	assert.Equal(t, "config/app.yml", results2[0].Dest)
}
