package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTargetTestData creates the complete hierarchy: Client -> Org -> Repo -> Config -> Group -> Target
func setupTargetTestData(t *testing.T, db *gorm.DB) (*Target, *Group, *Config) {
	t.Helper()

	// Create Client (root of hierarchy)
	client := &Client{Name: "testclient"}
	require.NoError(t, db.Create(client).Error)

	// Create Organization
	org := &Organization{ClientID: client.ID, Name: "testorg"}
	require.NoError(t, db.Create(org).Error)

	// Create Repository
	repository := &Repo{
		OrganizationID: org.ID,
		Name:           "testrepo",
	}
	require.NoError(t, db.Create(repository).Error)

	// Create Config
	config := &Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	require.NoError(t, db.Create(config).Error)

	// Create Group
	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	require.NoError(t, db.Create(group).Error)

	// Create Target
	target := &Target{
		GroupID:  group.ID,
		RepoID:   repository.ID,
		Branch:   "main",
		Position: 0,
	}
	targetRepo := NewTargetRepository(db)
	require.NoError(t, targetRepo.Create(context.Background(), target))

	return target, group, config
}

// TestTargetRepository_GetByRepoName_Success tests successful retrieval by repo name
func TestTargetRepository_GetByRepoName_Success(t *testing.T) {
	t.Parallel()

	db := TestDB(t)
	ctx := context.Background()
	repo := NewTargetRepository(db)

	// Setup test data - create Client first (required by Organization)
	client := &Client{Name: "testclient"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "testorg"}
	require.NoError(t, db.Create(org).Error)

	repository := &Repo{
		OrganizationID: org.ID,
		Name:           "testrepo",
	}
	require.NoError(t, db.Create(repository).Error)

	config := &Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	require.NoError(t, db.Create(config).Error)

	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	require.NoError(t, db.Create(group).Error)

	target := &Target{
		GroupID:  group.ID,
		RepoID:   repository.ID,
		Branch:   "main",
		Position: 0,
	}
	require.NoError(t, repo.Create(ctx, target))

	// Test GetByRepoName
	result, err := repo.GetByRepoName(ctx, group.ID, "testorg/testrepo")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, target.ID, result.ID)
	assert.Equal(t, "main", result.Branch)

	// Verify preloads worked
	require.NotNil(t, result.RepoRef, "RepoRef should be preloaded")
	assert.Equal(t, "testrepo", result.RepoRef.Name)
	require.NotNil(t, result.RepoRef.Organization, "Organization should be preloaded")
	assert.Equal(t, "testorg", result.RepoRef.Organization.Name)
}

// TestTargetRepository_GetByRepoName_InvalidFormat tests error handling for invalid repo names
func TestTargetRepository_GetByRepoName_InvalidFormat(t *testing.T) {
	t.Parallel()

	db := TestDB(t)
	ctx := context.Background()
	repo := NewTargetRepository(db)

	tests := []struct {
		name     string
		repoName string
		wantErr  error
	}{
		{
			name:     "missing slash",
			repoName: "invalid-format",
			wantErr:  ErrInvalidRepoFormat,
		},
		{
			name:     "empty org - returns not found",
			repoName: "/repo",
			wantErr:  ErrRecordNotFound, // SplitN still returns 2 parts, query returns not found
		},
		{
			name:     "empty repo - returns not found",
			repoName: "org/",
			wantErr:  ErrRecordNotFound, // SplitN still returns 2 parts, query returns not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := repo.GetByRepoName(ctx, 1, tt.repoName)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestTargetRepository_GetByRepoName_NotFound tests not found error
func TestTargetRepository_GetByRepoName_NotFound(t *testing.T) {
	t.Parallel()

	db := TestDB(t)
	ctx := context.Background()
	repo := NewTargetRepository(db)

	_, err := repo.GetByRepoName(ctx, 999, "nonexistent/repo")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRecordNotFound)
}

// TestTargetRepository_ListWithAssociations tests preloading all relationships
func TestTargetRepository_ListWithAssociations(t *testing.T) {
	t.Parallel()

	db := TestDB(t)
	ctx := context.Background()
	targetRepo := NewTargetRepository(db)

	// Setup complex test data with all associations using helper
	target, group, config := setupTargetTestData(t, db)

	// Create file list
	fileList := &FileList{
		ConfigID:   config.ID,
		ExternalID: "file-list-1",
		Name:       "Test Files",
	}
	require.NoError(t, db.Create(fileList).Error)

	// Create directory list
	dirList := &DirectoryList{
		ConfigID:   config.ID,
		ExternalID: "dir-list-1",
		Name:       "Test Dirs",
	}
	require.NoError(t, db.Create(dirList).Error)

	// Create transform with polymorphic relationship
	transform := &Transform{
		OwnerType: "target",
		OwnerID:   target.ID,
		RepoName:  true,
	}
	require.NoError(t, db.Create(transform).Error)

	// Add file mapping (polymorphic relationship)
	fileMapping := &FileMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       "src/file.txt",
		Dest:      "dst/file.txt",
		Position:  0,
	}
	require.NoError(t, db.Create(fileMapping).Error)

	// Add directory mapping with transform (polymorphic relationship)
	dirMapping := &DirectoryMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       "src/dir",
		Dest:      "dst/dir",
		Position:  0,
	}
	require.NoError(t, db.Create(dirMapping).Error)

	// Create transform for directory mapping
	dirTransform := &Transform{
		OwnerType: "directory_mapping",
		OwnerID:   dirMapping.ID,
		RepoName:  true,
	}
	require.NoError(t, db.Create(dirTransform).Error)

	// Add file list reference
	require.NoError(t, targetRepo.AddFileListRef(ctx, target.ID, fileList.ID, 0))

	// Add directory list reference
	require.NoError(t, targetRepo.AddDirectoryListRef(ctx, target.ID, dirList.ID, 0))

	// Test ListWithAssociations
	targets, err := targetRepo.ListWithAssociations(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)

	result := targets[0]

	// Verify RepoRef.Organization preload
	require.NotNil(t, result.RepoRef, "RepoRef should be preloaded")
	assert.Equal(t, "testrepo", result.RepoRef.Name)
	require.NotNil(t, result.RepoRef.Organization, "Organization should be preloaded")
	assert.Equal(t, "testorg", result.RepoRef.Organization.Name)

	// Verify FileMappings preload
	require.NotEmpty(t, result.FileMappings, "FileMappings should be preloaded")
	assert.Equal(t, "src/file.txt", result.FileMappings[0].Src)
	assert.Equal(t, "dst/file.txt", result.FileMappings[0].Dest)

	// Verify DirectoryMappings preload
	require.NotEmpty(t, result.DirectoryMappings, "DirectoryMappings should be preloaded")
	assert.Equal(t, "src/dir", result.DirectoryMappings[0].Src)
	assert.Equal(t, "dst/dir", result.DirectoryMappings[0].Dest)

	// Verify DirectoryMappings.Transform preload
	require.NotEmpty(t, result.DirectoryMappings[0].Transform.OwnerType, "DirectoryMapping Transform should be preloaded")
	assert.Equal(t, "directory_mapping", result.DirectoryMappings[0].Transform.OwnerType)

	// Verify Transform preload
	require.NotEmpty(t, result.Transform.OwnerType, "Transform should be preloaded")
	assert.Equal(t, "target", result.Transform.OwnerType)

	// Verify FileListRefs preload
	require.NotEmpty(t, result.FileListRefs, "FileListRefs should be preloaded")
	require.NotNil(t, result.FileListRefs[0].FileList, "FileList should be preloaded")
	assert.Equal(t, "Test Files", result.FileListRefs[0].FileList.Name)

	// Verify DirectoryListRefs preload
	require.NotEmpty(t, result.DirectoryListRefs, "DirectoryListRefs should be preloaded")
	require.NotNil(t, result.DirectoryListRefs[0].DirectoryList, "DirectoryList should be preloaded")
	assert.Equal(t, "Test Dirs", result.DirectoryListRefs[0].DirectoryList.Name)
}

// TestTargetRepository_ListWithAssociations_Ordering tests that mappings are ordered by position
func TestTargetRepository_ListWithAssociations_Ordering(t *testing.T) {
	t.Parallel()

	db := TestDB(t)
	ctx := context.Background()
	targetRepo := NewTargetRepository(db)

	// Setup test data using helper
	target, group, _ := setupTargetTestData(t, db)

	// Add multiple file mappings with different positions (out of order)
	fileMappings := []FileMapping{
		{OwnerType: "target", OwnerID: target.ID, Src: "third.txt", Dest: "third.txt", Position: 2},
		{OwnerType: "target", OwnerID: target.ID, Src: "first.txt", Dest: "first.txt", Position: 0},
		{OwnerType: "target", OwnerID: target.ID, Src: "second.txt", Dest: "second.txt", Position: 1},
	}
	for _, fm := range fileMappings {
		require.NoError(t, db.Create(&fm).Error)
	}

	// Add multiple directory mappings with different positions (out of order)
	dirMappings := []DirectoryMapping{
		{OwnerType: "target", OwnerID: target.ID, Src: "dir3", Dest: "dir3", Position: 2},
		{OwnerType: "target", OwnerID: target.ID, Src: "dir1", Dest: "dir1", Position: 0},
		{OwnerType: "target", OwnerID: target.ID, Src: "dir2", Dest: "dir2", Position: 1},
	}
	for _, dm := range dirMappings {
		require.NoError(t, db.Create(&dm).Error)
	}

	// Test ListWithAssociations
	targets, err := targetRepo.ListWithAssociations(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)

	result := targets[0]

	// Verify FileMappings are ordered by position
	require.Len(t, result.FileMappings, 3)
	assert.Equal(t, "first.txt", result.FileMappings[0].Src)
	assert.Equal(t, "second.txt", result.FileMappings[1].Src)
	assert.Equal(t, "third.txt", result.FileMappings[2].Src)

	// Verify DirectoryMappings are ordered by position
	require.Len(t, result.DirectoryMappings, 3)
	assert.Equal(t, "dir1", result.DirectoryMappings[0].Src)
	assert.Equal(t, "dir2", result.DirectoryMappings[1].Src)
	assert.Equal(t, "dir3", result.DirectoryMappings[2].Src)
}

// TestTargetRepository_FileListRef_AddRemove tests many-to-many file list operations
func TestTargetRepository_FileListRef_AddRemove(t *testing.T) {
	t.Parallel()

	db := TestDB(t)
	ctx := context.Background()
	targetRepo := NewTargetRepository(db)

	// Setup test data using helper
	target, group, config := setupTargetTestData(t, db)

	fileList := &FileList{
		ConfigID:   config.ID,
		ExternalID: "file-list-1",
		Name:       "Test Files",
	}
	require.NoError(t, db.Create(fileList).Error)

	// Test AddFileListRef
	err := targetRepo.AddFileListRef(ctx, target.ID, fileList.ID, 0)
	require.NoError(t, err)

	// Verify reference was created
	targets, err := targetRepo.ListWithAssociations(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Len(t, targets[0].FileListRefs, 1)
	assert.Equal(t, fileList.ID, targets[0].FileListRefs[0].FileListID)

	// Test RemoveFileListRef
	err = targetRepo.RemoveFileListRef(ctx, target.ID, fileList.ID)
	require.NoError(t, err)

	// Verify reference was removed
	targets, err = targetRepo.ListWithAssociations(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Empty(t, targets[0].FileListRefs, "FileListRefs should be empty after removal")
}

// TestTargetRepository_DirectoryListRef_AddRemove tests many-to-many directory list operations
func TestTargetRepository_DirectoryListRef_AddRemove(t *testing.T) {
	t.Parallel()

	db := TestDB(t)
	ctx := context.Background()
	targetRepo := NewTargetRepository(db)

	// Setup test data using helper
	target, group, config := setupTargetTestData(t, db)

	dirList := &DirectoryList{
		ConfigID:   config.ID,
		ExternalID: "dir-list-1",
		Name:       "Test Dirs",
	}
	require.NoError(t, db.Create(dirList).Error)

	// Test AddDirectoryListRef
	err := targetRepo.AddDirectoryListRef(ctx, target.ID, dirList.ID, 0)
	require.NoError(t, err)

	// Verify reference was created
	targets, err := targetRepo.ListWithAssociations(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Len(t, targets[0].DirectoryListRefs, 1)
	assert.Equal(t, dirList.ID, targets[0].DirectoryListRefs[0].DirectoryListID)

	// Test RemoveDirectoryListRef
	err = targetRepo.RemoveDirectoryListRef(ctx, target.ID, dirList.ID)
	require.NoError(t, err)

	// Verify reference was removed
	targets, err = targetRepo.ListWithAssociations(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Empty(t, targets[0].DirectoryListRefs, "DirectoryListRefs should be empty after removal")
}

// TestTargetRepository_MultipleListRefs tests adding multiple file/directory lists
func TestTargetRepository_MultipleListRefs(t *testing.T) {
	t.Parallel()

	db := TestDB(t)
	ctx := context.Background()
	targetRepo := NewTargetRepository(db)

	// Setup test data using helper
	target, group, config := setupTargetTestData(t, db)

	// Create multiple file lists
	fileLists := []*FileList{
		{ConfigID: config.ID, ExternalID: "list-1", Name: "List 1"},
		{ConfigID: config.ID, ExternalID: "list-2", Name: "List 2"},
		{ConfigID: config.ID, ExternalID: "list-3", Name: "List 3"},
	}
	for _, fl := range fileLists {
		require.NoError(t, db.Create(fl).Error)
	}

	// Add file list refs with different positions (out of order)
	require.NoError(t, targetRepo.AddFileListRef(ctx, target.ID, fileLists[2].ID, 2))
	require.NoError(t, targetRepo.AddFileListRef(ctx, target.ID, fileLists[0].ID, 0))
	require.NoError(t, targetRepo.AddFileListRef(ctx, target.ID, fileLists[1].ID, 1))

	// Verify refs are ordered by position
	targets, err := targetRepo.ListWithAssociations(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Len(t, targets[0].FileListRefs, 3)

	// Should be ordered by position: 0, 1, 2
	assert.Equal(t, "List 1", targets[0].FileListRefs[0].FileList.Name)
	assert.Equal(t, "List 2", targets[0].FileListRefs[1].FileList.Name)
	assert.Equal(t, "List 3", targets[0].FileListRefs[2].FileList.Name)
}
