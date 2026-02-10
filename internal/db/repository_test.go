package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// RepositoryTestSuite provides a test suite for repository operations
type RepositoryTestSuite struct {
	suite.Suite

	db     *gorm.DB
	ctx    context.Context
	config *Config
}

// SetupTest runs before each test
func (s *RepositoryTestSuite) SetupTest() {
	s.db = TestDB(s.T())
	s.ctx = context.Background()

	// Create a test config for all tests
	s.config = &Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	err := s.db.Create(s.config).Error
	require.NoError(s.T(), err)
}

// TestConfigRepository tests ConfigRepository operations
func (s *RepositoryTestSuite) TestConfigRepository() {
	repo := NewConfigRepository(s.db)

	// Test Create
	cfg := &Config{
		BaseModel: BaseModel{
			Metadata: Metadata{"key": "value"},
		},
		ExternalID: "test-cfg-1",
		Name:       "Test Config 1",
		Version:    1,
	}
	err := repo.Create(s.ctx, cfg)
	require.NoError(s.T(), err)
	assert.NotZero(s.T(), cfg.ID)

	// Test GetByID
	fetched, err := repo.GetByID(s.ctx, cfg.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), cfg.ExternalID, fetched.ExternalID)
	assert.Equal(s.T(), cfg.Name, fetched.Name)
	assert.Equal(s.T(), "value", fetched.Metadata["key"])

	// Test GetByExternalID
	fetched2, err := repo.GetByExternalID(s.ctx, "test-cfg-1")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), cfg.ID, fetched2.ID)

	// Test Update
	cfg.Name = "Updated Config"
	err = repo.Update(s.ctx, cfg)
	require.NoError(s.T(), err)

	fetched3, err := repo.GetByID(s.ctx, cfg.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "Updated Config", fetched3.Name)

	// Test List
	configs, err := repo.List(s.ctx)
	require.NoError(s.T(), err)
	assert.GreaterOrEqual(s.T(), len(configs), 2) // At least our 2 test configs

	// Test Delete
	err = repo.Delete(s.ctx, cfg.ID)
	require.NoError(s.T(), err)

	_, err = repo.GetByID(s.ctx, cfg.ID)
	assert.ErrorIs(s.T(), err, ErrRecordNotFound)
}

// TestGroupRepository tests GroupRepository operations
func (s *RepositoryTestSuite) TestGroupRepository() {
	repo := NewGroupRepository(s.db)

	// Test Create
	group := &Group{
		BaseModel: BaseModel{
			Metadata: Metadata{"env": "test"},
		},
		ConfigID:   s.config.ID,
		ExternalID: "test-group-1",
		Name:       "Test Group 1",
		Priority:   10,
		Position:   0,
	}
	err := repo.Create(s.ctx, group)
	require.NoError(s.T(), err)
	assert.NotZero(s.T(), group.ID)

	// Test GetByID
	fetched, err := repo.GetByID(s.ctx, group.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), group.ExternalID, fetched.ExternalID)
	assert.Equal(s.T(), group.Name, fetched.Name)
	assert.Equal(s.T(), group.Priority, fetched.Priority)

	// Test GetByExternalID
	fetched2, err := repo.GetByExternalID(s.ctx, "test-group-1")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), group.ID, fetched2.ID)

	// Test Update
	group.Description = "Updated description"
	err = repo.Update(s.ctx, group)
	require.NoError(s.T(), err)

	fetched3, err := repo.GetByID(s.ctx, group.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "Updated description", fetched3.Description)

	// Test List
	groups, err := repo.List(s.ctx, s.config.ID)
	require.NoError(s.T(), err)
	assert.Len(s.T(), groups, 1)

	// Test soft Delete
	err = repo.Delete(s.ctx, group.ID, false)
	require.NoError(s.T(), err)

	groups2, err := repo.List(s.ctx, s.config.ID)
	require.NoError(s.T(), err)
	assert.Len(s.T(), groups2, 0) // Should be soft-deleted

	// Test hard Delete
	group2 := &Group{
		ConfigID:   s.config.ID,
		ExternalID: "test-group-2",
		Name:       "Test Group 2",
	}
	err = repo.Create(s.ctx, group2)
	require.NoError(s.T(), err)

	err = repo.Delete(s.ctx, group2.ID, true)
	require.NoError(s.T(), err)

	_, err = repo.GetByID(s.ctx, group2.ID)
	assert.ErrorIs(s.T(), err, ErrRecordNotFound)
}

// TestGroupRepositoryWithAssociations tests group preloading
func (s *RepositoryTestSuite) TestGroupRepositoryWithAssociations() {
	repo := NewGroupRepository(s.db)

	// Create a group with all associations
	group := &Group{
		ConfigID:   s.config.ID,
		ExternalID: "test-group-assoc",
		Name:       "Test Group With Associations",
		Source: Source{
			Repo:   "mrz1836/test",
			Branch: "main",
		},
		GroupGlobal: GroupGlobal{
			PRLabels: JSONStringSlice{"sync"},
		},
		GroupDefault: GroupDefault{
			BranchPrefix: "broadcast",
		},
		Dependencies: []GroupDependency{
			{DependsOnID: "other-group", Position: 0},
		},
	}
	err := s.db.Create(group).Error
	require.NoError(s.T(), err)

	// Create a target for the group
	target := &Target{
		GroupID: group.ID,
		Repo:    "mrz1836/target",
		Branch:  "main",
	}
	err = s.db.Create(target).Error
	require.NoError(s.T(), err)

	// Test ListWithAssociations
	groups, err := repo.ListWithAssociations(s.ctx, s.config.ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), groups, 1)

	g := groups[0]
	assert.Equal(s.T(), "mrz1836/test", g.Source.Repo)
	assert.Equal(s.T(), "main", g.Source.Branch)
	assert.Len(s.T(), g.GroupGlobal.PRLabels, 1)
	assert.Equal(s.T(), "sync", g.GroupGlobal.PRLabels[0])
	assert.Equal(s.T(), "broadcast", g.GroupDefault.BranchPrefix)
	assert.Len(s.T(), g.Dependencies, 1)
	assert.Equal(s.T(), "other-group", g.Dependencies[0].DependsOnID)
	assert.Len(s.T(), g.Targets, 1)
	assert.Equal(s.T(), "mrz1836/target", g.Targets[0].Repo)
}

// TestTargetRepository tests TargetRepository operations
func (s *RepositoryTestSuite) TestTargetRepository() {
	repo := NewTargetRepository(s.db)

	// Create a group first
	group := &Group{
		ConfigID:   s.config.ID,
		ExternalID: "test-group-targets",
		Name:       "Test Group for Targets",
	}
	err := s.db.Create(group).Error
	require.NoError(s.T(), err)

	// Test Create
	target := &Target{
		BaseModel: BaseModel{
			Metadata: Metadata{"type": "library"},
		},
		GroupID:  group.ID,
		Repo:     "mrz1836/test-target",
		Branch:   "main",
		Position: 0,
	}
	err = repo.Create(s.ctx, target)
	require.NoError(s.T(), err)
	assert.NotZero(s.T(), target.ID)

	// Test GetByID
	fetched, err := repo.GetByID(s.ctx, target.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), target.Repo, fetched.Repo)
	assert.Equal(s.T(), "library", fetched.Metadata["type"])

	// Test GetByRepo
	fetched2, err := repo.GetByRepo(s.ctx, group.ID, "mrz1836/test-target")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), target.ID, fetched2.ID)

	// Test Update
	target.Branch = "develop"
	err = repo.Update(s.ctx, target)
	require.NoError(s.T(), err)

	fetched3, err := repo.GetByID(s.ctx, target.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "develop", fetched3.Branch)

	// Test List
	targets, err := repo.List(s.ctx, group.ID)
	require.NoError(s.T(), err)
	assert.Len(s.T(), targets, 1)

	// Test soft Delete
	err = repo.Delete(s.ctx, target.ID, false)
	require.NoError(s.T(), err)

	targets2, err := repo.List(s.ctx, group.ID)
	require.NoError(s.T(), err)
	assert.Len(s.T(), targets2, 0)
}

// TestTargetRefManagement tests file list and directory list ref management
func (s *RepositoryTestSuite) TestTargetRefManagement() {
	repo := NewTargetRepository(s.db)

	// Create test data
	group := &Group{
		ConfigID:   s.config.ID,
		ExternalID: "test-group-refs",
		Name:       "Test Group for Refs",
	}
	err := s.db.Create(group).Error
	require.NoError(s.T(), err)

	target := &Target{
		GroupID: group.ID,
		Repo:    "mrz1836/test-refs",
	}
	err = repo.Create(s.ctx, target)
	require.NoError(s.T(), err)

	fileList := &FileList{
		ConfigID:   s.config.ID,
		ExternalID: "test-file-list",
		Name:       "Test File List",
	}
	err = s.db.Create(fileList).Error
	require.NoError(s.T(), err)

	dirList := &DirectoryList{
		ConfigID:   s.config.ID,
		ExternalID: "test-dir-list",
		Name:       "Test Directory List",
	}
	err = s.db.Create(dirList).Error
	require.NoError(s.T(), err)

	// Test AddFileListRef
	err = repo.AddFileListRef(s.ctx, target.ID, fileList.ID, 0)
	require.NoError(s.T(), err)

	// Verify ref was created
	var refCount int64
	err = s.db.Model(&TargetFileListRef{}).
		Where("target_id = ? AND file_list_id = ?", target.ID, fileList.ID).
		Count(&refCount).Error
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), refCount)

	// Test AddDirectoryListRef
	err = repo.AddDirectoryListRef(s.ctx, target.ID, dirList.ID, 0)
	require.NoError(s.T(), err)

	// Verify ref was created
	err = s.db.Model(&TargetDirectoryListRef{}).
		Where("target_id = ? AND directory_list_id = ?", target.ID, dirList.ID).
		Count(&refCount).Error
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), refCount)

	// Test RemoveFileListRef
	err = repo.RemoveFileListRef(s.ctx, target.ID, fileList.ID)
	require.NoError(s.T(), err)

	err = s.db.Model(&TargetFileListRef{}).
		Where("target_id = ? AND file_list_id = ?", target.ID, fileList.ID).
		Count(&refCount).Error
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), refCount)

	// Test RemoveDirectoryListRef
	err = repo.RemoveDirectoryListRef(s.ctx, target.ID, dirList.ID)
	require.NoError(s.T(), err)

	err = s.db.Model(&TargetDirectoryListRef{}).
		Where("target_id = ? AND directory_list_id = ?", target.ID, dirList.ID).
		Count(&refCount).Error
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), refCount)
}

// TestFileListRepository tests FileListRepository operations
func (s *RepositoryTestSuite) TestFileListRepository() {
	repo := NewFileListRepository(s.db)

	// Test Create
	fileList := &FileList{
		ConfigID:    s.config.ID,
		ExternalID:  "test-file-list-1",
		Name:        "Test File List 1",
		Description: "Test description",
		Position:    0,
	}
	err := repo.Create(s.ctx, fileList)
	require.NoError(s.T(), err)
	assert.NotZero(s.T(), fileList.ID)

	// Test GetByID
	fetched, err := repo.GetByID(s.ctx, fileList.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), fileList.Name, fetched.Name)

	// Test GetByExternalID
	fetched2, err := repo.GetByExternalID(s.ctx, "test-file-list-1")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), fileList.ID, fetched2.ID)

	// Test Update
	fileList.Description = "Updated description"
	err = repo.Update(s.ctx, fileList)
	require.NoError(s.T(), err)

	fetched3, err := repo.GetByID(s.ctx, fileList.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "Updated description", fetched3.Description)

	// Test List
	lists, err := repo.List(s.ctx, s.config.ID)
	require.NoError(s.T(), err)
	assert.GreaterOrEqual(s.T(), len(lists), 1)

	// Test soft Delete
	err = repo.Delete(s.ctx, fileList.ID, false)
	require.NoError(s.T(), err)

	lists2, err := repo.List(s.ctx, s.config.ID)
	require.NoError(s.T(), err)
	assert.Len(s.T(), lists2, 0)
}

// TestFileListWithFiles tests file list preloading
func (s *RepositoryTestSuite) TestFileListWithFiles() {
	repo := NewFileListRepository(s.db)

	// Create file list with file mappings
	fileList := &FileList{
		ConfigID:   s.config.ID,
		ExternalID: "test-file-list-files",
		Name:       "Test File List With Files",
	}
	err := s.db.Create(fileList).Error
	require.NoError(s.T(), err)

	// Create file mappings
	mapping1 := &FileMapping{
		OwnerType: "file_list",
		OwnerID:   fileList.ID,
		Src:       ".github/workflows/ci.yml",
		Dest:      ".github/workflows/ci.yml",
		Position:  0,
	}
	err = s.db.Create(mapping1).Error
	require.NoError(s.T(), err)

	mapping2 := &FileMapping{
		OwnerType: "file_list",
		OwnerID:   fileList.ID,
		Src:       "README.md",
		Dest:      "README.md",
		Position:  1,
	}
	err = s.db.Create(mapping2).Error
	require.NoError(s.T(), err)

	// Test ListWithFiles
	lists, err := repo.ListWithFiles(s.ctx, s.config.ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), lists, 1)

	fl := lists[0]
	assert.Len(s.T(), fl.Files, 2)
	assert.Equal(s.T(), ".github/workflows/ci.yml", fl.Files[0].Dest)
	assert.Equal(s.T(), "README.md", fl.Files[1].Dest)
}

// TestDirectoryListRepository tests DirectoryListRepository operations
func (s *RepositoryTestSuite) TestDirectoryListRepository() {
	repo := NewDirectoryListRepository(s.db)

	// Test Create
	dirList := &DirectoryList{
		ConfigID:    s.config.ID,
		ExternalID:  "test-dir-list-1",
		Name:        "Test Directory List 1",
		Description: "Test description",
		Position:    0,
	}
	err := repo.Create(s.ctx, dirList)
	require.NoError(s.T(), err)
	assert.NotZero(s.T(), dirList.ID)

	// Test GetByID
	fetched, err := repo.GetByID(s.ctx, dirList.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), dirList.Name, fetched.Name)

	// Test GetByExternalID
	fetched2, err := repo.GetByExternalID(s.ctx, "test-dir-list-1")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), dirList.ID, fetched2.ID)

	// Test Update
	dirList.Description = "Updated description"
	err = repo.Update(s.ctx, dirList)
	require.NoError(s.T(), err)

	fetched3, err := repo.GetByID(s.ctx, dirList.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "Updated description", fetched3.Description)

	// Test List
	lists, err := repo.List(s.ctx, s.config.ID)
	require.NoError(s.T(), err)
	assert.GreaterOrEqual(s.T(), len(lists), 1)

	// Test Delete
	err = repo.Delete(s.ctx, dirList.ID, false)
	require.NoError(s.T(), err)

	lists2, err := repo.List(s.ctx, s.config.ID)
	require.NoError(s.T(), err)
	assert.Len(s.T(), lists2, 0)
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
