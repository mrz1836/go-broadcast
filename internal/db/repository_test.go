package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// RepositoryTestSuite provides a test suite for repository operations
type RepositoryTestSuite struct {
	suite.Suite

	db     *gorm.DB
	ctx    context.Context //nolint:containedctx // test suite pattern requires context in struct
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
	s.Require().NoError(err)
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
	s.Require().NoError(err)
	s.NotZero(cfg.ID)

	// Test GetByID
	fetched, err := repo.GetByID(s.ctx, cfg.ID)
	s.Require().NoError(err)
	s.Equal(cfg.ExternalID, fetched.ExternalID)
	s.Equal(cfg.Name, fetched.Name)
	s.Equal("value", fetched.Metadata["key"])

	// Test GetByExternalID
	fetched2, err := repo.GetByExternalID(s.ctx, "test-cfg-1")
	s.Require().NoError(err)
	s.Equal(cfg.ID, fetched2.ID)

	// Test Update
	cfg.Name = "Updated Config"
	err = repo.Update(s.ctx, cfg)
	s.Require().NoError(err)

	fetched3, err := repo.GetByID(s.ctx, cfg.ID)
	s.Require().NoError(err)
	s.Equal("Updated Config", fetched3.Name)

	// Test List
	configs, err := repo.List(s.ctx)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(configs), 2) // At least our 2 test configs

	// Test Delete
	err = repo.Delete(s.ctx, cfg.ID)
	s.Require().NoError(err)

	_, err = repo.GetByID(s.ctx, cfg.ID)
	s.ErrorIs(err, ErrRecordNotFound)
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
	s.Require().NoError(err)
	s.NotZero(group.ID)

	// Test GetByID
	fetched, err := repo.GetByID(s.ctx, group.ID)
	s.Require().NoError(err)
	s.Equal(group.ExternalID, fetched.ExternalID)
	s.Equal(group.Name, fetched.Name)
	s.Equal(group.Priority, fetched.Priority)

	// Test GetByExternalID
	fetched2, err := repo.GetByExternalID(s.ctx, "test-group-1")
	s.Require().NoError(err)
	s.Equal(group.ID, fetched2.ID)

	// Test Update
	group.Description = "Updated description"
	err = repo.Update(s.ctx, group)
	s.Require().NoError(err)

	fetched3, err := repo.GetByID(s.ctx, group.ID)
	s.Require().NoError(err)
	s.Equal("Updated description", fetched3.Description)

	// Test List
	groups, err := repo.List(s.ctx, s.config.ID)
	s.Require().NoError(err)
	s.Len(groups, 1)

	// Test soft Delete
	err = repo.Delete(s.ctx, group.ID, false)
	s.Require().NoError(err)

	groups2, err := repo.List(s.ctx, s.config.ID)
	s.Require().NoError(err)
	s.Empty(groups2) // Should be soft-deleted

	// Test hard Delete
	group2 := &Group{
		ConfigID:   s.config.ID,
		ExternalID: "test-group-2",
		Name:       "Test Group 2",
	}
	err = repo.Create(s.ctx, group2)
	s.Require().NoError(err)

	err = repo.Delete(s.ctx, group2.ID, true)
	s.Require().NoError(err)

	_, err = repo.GetByID(s.ctx, group2.ID)
	s.ErrorIs(err, ErrRecordNotFound)
}

// TestGroupRepositoryWithAssociations tests group preloading
func (s *RepositoryTestSuite) TestGroupRepositoryWithAssociations() {
	repo := NewGroupRepository(s.db)

	// Create Client -> Organization -> Repo chain
	client := &Client{Name: "test"}
	s.Require().NoError(s.db.Create(client).Error)
	org := &Organization{ClientID: client.ID, Name: "mrz1836"}
	s.Require().NoError(s.db.Create(org).Error)
	sourceRepo := &Repo{OrganizationID: org.ID, Name: "test"}
	s.Require().NoError(s.db.Create(sourceRepo).Error)
	targetRepo := &Repo{OrganizationID: org.ID, Name: "target"}
	s.Require().NoError(s.db.Create(targetRepo).Error)

	// Create a group with all associations
	group := &Group{
		ConfigID:   s.config.ID,
		ExternalID: "test-group-assoc",
		Name:       "Test Group With Associations",
		Source: Source{
			RepoID: sourceRepo.ID,
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
	s.Require().NoError(err)

	// Create a target for the group
	target := &Target{
		GroupID: group.ID,
		RepoID:  targetRepo.ID,
		Branch:  "main",
	}
	err = s.db.Create(target).Error
	s.Require().NoError(err)

	// Test ListWithAssociations
	groups, err := repo.ListWithAssociations(s.ctx, s.config.ID)
	s.Require().NoError(err)
	s.Require().Len(groups, 1)

	g := groups[0]
	s.Equal(sourceRepo.ID, g.Source.RepoID)
	s.Equal("main", g.Source.Branch)
	s.Len(g.GroupGlobal.PRLabels, 1)
	s.Equal("sync", g.GroupGlobal.PRLabels[0])
	s.Equal("broadcast", g.GroupDefault.BranchPrefix)
	s.Len(g.Dependencies, 1)
	s.Equal("other-group", g.Dependencies[0].DependsOnID)
	s.Len(g.Targets, 1)
	s.Equal(targetRepo.ID, g.Targets[0].RepoID)
}

// TestTargetRepository tests TargetRepository operations
func (s *RepositoryTestSuite) TestTargetRepository() {
	repo := NewTargetRepository(s.db)

	// Create Client -> Organization -> Repo chain
	client := &Client{Name: "test-targets"}
	s.Require().NoError(s.db.Create(client).Error)
	org := &Organization{ClientID: client.ID, Name: "mrz1836-targets"}
	s.Require().NoError(s.db.Create(org).Error)
	testRepo := &Repo{OrganizationID: org.ID, Name: "test-target"}
	s.Require().NoError(s.db.Create(testRepo).Error)

	// Create a group first
	group := &Group{
		ConfigID:   s.config.ID,
		ExternalID: "test-group-targets",
		Name:       "Test Group for Targets",
	}
	err := s.db.Create(group).Error
	s.Require().NoError(err)

	// Test Create
	target := &Target{
		BaseModel: BaseModel{
			Metadata: Metadata{"type": "library"},
		},
		GroupID:  group.ID,
		RepoID:   testRepo.ID,
		Branch:   "main",
		Position: 0,
	}
	err = repo.Create(s.ctx, target)
	s.Require().NoError(err)
	s.NotZero(target.ID)

	// Test GetByID
	fetched, err := repo.GetByID(s.ctx, target.ID)
	s.Require().NoError(err)
	s.Equal(target.RepoID, fetched.RepoID)
	s.Equal("library", fetched.Metadata["type"])

	// Test GetByRepoName
	fetched2, err := repo.GetByRepoName(s.ctx, group.ID, "mrz1836-targets/test-target")
	s.Require().NoError(err)
	s.Equal(target.ID, fetched2.ID)

	// Test Update
	target.Branch = "develop"
	err = repo.Update(s.ctx, target)
	s.Require().NoError(err)

	fetched3, err := repo.GetByID(s.ctx, target.ID)
	s.Require().NoError(err)
	s.Equal("develop", fetched3.Branch)

	// Test List
	targets, err := repo.List(s.ctx, group.ID)
	s.Require().NoError(err)
	s.Len(targets, 1)

	// Test soft Delete
	err = repo.Delete(s.ctx, target.ID, false)
	s.Require().NoError(err)

	targets2, err := repo.List(s.ctx, group.ID)
	s.Require().NoError(err)
	s.Empty(targets2)
}

// TestTargetRefManagement tests file list and directory list ref management
func (s *RepositoryTestSuite) TestTargetRefManagement() {
	repo := NewTargetRepository(s.db)

	// Create Client -> Organization -> Repo chain
	client := &Client{Name: "test-refs"}
	s.Require().NoError(s.db.Create(client).Error)
	org := &Organization{ClientID: client.ID, Name: "mrz1836-refs"}
	s.Require().NoError(s.db.Create(org).Error)
	testRepo := &Repo{OrganizationID: org.ID, Name: "test-refs"}
	s.Require().NoError(s.db.Create(testRepo).Error)

	// Create test data
	group := &Group{
		ConfigID:   s.config.ID,
		ExternalID: "test-group-refs",
		Name:       "Test Group for Refs",
	}
	err := s.db.Create(group).Error
	s.Require().NoError(err)

	target := &Target{
		GroupID: group.ID,
		RepoID:  testRepo.ID,
	}
	err = repo.Create(s.ctx, target)
	s.Require().NoError(err)

	fileList := &FileList{
		ConfigID:   s.config.ID,
		ExternalID: "test-file-list",
		Name:       "Test File List",
	}
	err = s.db.Create(fileList).Error
	s.Require().NoError(err)

	dirList := &DirectoryList{
		ConfigID:   s.config.ID,
		ExternalID: "test-dir-list",
		Name:       "Test Directory List",
	}
	err = s.db.Create(dirList).Error
	s.Require().NoError(err)

	// Test AddFileListRef
	err = repo.AddFileListRef(s.ctx, target.ID, fileList.ID, 0)
	s.Require().NoError(err)

	// Verify ref was created
	var refCount int64
	err = s.db.Model(&TargetFileListRef{}).
		Where("target_id = ? AND file_list_id = ?", target.ID, fileList.ID).
		Count(&refCount).Error
	s.Require().NoError(err)
	s.Equal(int64(1), refCount)

	// Test AddDirectoryListRef
	err = repo.AddDirectoryListRef(s.ctx, target.ID, dirList.ID, 0)
	s.Require().NoError(err)

	// Verify ref was created
	err = s.db.Model(&TargetDirectoryListRef{}).
		Where("target_id = ? AND directory_list_id = ?", target.ID, dirList.ID).
		Count(&refCount).Error
	s.Require().NoError(err)
	s.Equal(int64(1), refCount)

	// Test RemoveFileListRef
	err = repo.RemoveFileListRef(s.ctx, target.ID, fileList.ID)
	s.Require().NoError(err)

	err = s.db.Model(&TargetFileListRef{}).
		Where("target_id = ? AND file_list_id = ?", target.ID, fileList.ID).
		Count(&refCount).Error
	s.Require().NoError(err)
	s.Equal(int64(0), refCount)

	// Test RemoveDirectoryListRef
	err = repo.RemoveDirectoryListRef(s.ctx, target.ID, dirList.ID)
	s.Require().NoError(err)

	err = s.db.Model(&TargetDirectoryListRef{}).
		Where("target_id = ? AND directory_list_id = ?", target.ID, dirList.ID).
		Count(&refCount).Error
	s.Require().NoError(err)
	s.Equal(int64(0), refCount)
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
	s.Require().NoError(err)
	s.NotZero(fileList.ID)

	// Test GetByID
	fetched, err := repo.GetByID(s.ctx, fileList.ID)
	s.Require().NoError(err)
	s.Equal(fileList.Name, fetched.Name)

	// Test GetByExternalID
	fetched2, err := repo.GetByExternalID(s.ctx, "test-file-list-1")
	s.Require().NoError(err)
	s.Equal(fileList.ID, fetched2.ID)

	// Test Update
	fileList.Description = "Updated description"
	err = repo.Update(s.ctx, fileList)
	s.Require().NoError(err)

	fetched3, err := repo.GetByID(s.ctx, fileList.ID)
	s.Require().NoError(err)
	s.Equal("Updated description", fetched3.Description)

	// Test List
	lists, err := repo.List(s.ctx, s.config.ID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(lists), 1)

	// Test soft Delete
	err = repo.Delete(s.ctx, fileList.ID, false)
	s.Require().NoError(err)

	lists2, err := repo.List(s.ctx, s.config.ID)
	s.Require().NoError(err)
	s.Empty(lists2)
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
	s.Require().NoError(err)

	// Create file mappings
	mapping1 := &FileMapping{
		OwnerType: "file_list",
		OwnerID:   fileList.ID,
		Src:       ".github/workflows/ci.yml",
		Dest:      ".github/workflows/ci.yml",
		Position:  0,
	}
	err = s.db.Create(mapping1).Error
	s.Require().NoError(err)

	mapping2 := &FileMapping{
		OwnerType: "file_list",
		OwnerID:   fileList.ID,
		Src:       "README.md",
		Dest:      "README.md",
		Position:  1,
	}
	err = s.db.Create(mapping2).Error
	s.Require().NoError(err)

	// Test ListWithFiles
	lists, err := repo.ListWithFiles(s.ctx, s.config.ID)
	s.Require().NoError(err)
	s.Require().Len(lists, 1)

	fl := lists[0]
	s.Len(fl.Files, 2)
	s.Equal(".github/workflows/ci.yml", fl.Files[0].Dest)
	s.Equal("README.md", fl.Files[1].Dest)
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
	s.Require().NoError(err)
	s.NotZero(dirList.ID)

	// Test GetByID
	fetched, err := repo.GetByID(s.ctx, dirList.ID)
	s.Require().NoError(err)
	s.Equal(dirList.Name, fetched.Name)

	// Test GetByExternalID
	fetched2, err := repo.GetByExternalID(s.ctx, "test-dir-list-1")
	s.Require().NoError(err)
	s.Equal(dirList.ID, fetched2.ID)

	// Test Update
	dirList.Description = "Updated description"
	err = repo.Update(s.ctx, dirList)
	s.Require().NoError(err)

	fetched3, err := repo.GetByID(s.ctx, dirList.ID)
	s.Require().NoError(err)
	s.Equal("Updated description", fetched3.Description)

	// Test List
	lists, err := repo.List(s.ctx, s.config.ID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(lists), 1)

	// Test Delete
	err = repo.Delete(s.ctx, dirList.ID, false)
	s.Require().NoError(err)

	lists2, err := repo.List(s.ctx, s.config.ID)
	s.Require().NoError(err)
	s.Empty(lists2)
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
