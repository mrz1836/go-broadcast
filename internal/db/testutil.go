package db

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDB creates an in-memory SQLite database for testing
// Auto-migrates all models and registers t.Cleanup() for automatic cleanup
//
// Unlike OpenSQLite, this disables PrepareStmt to avoid livelock under the
// race detector. PrepareStmt's sync.RWMutex combined with MaxOpenConns=1
// causes concurrent read/write tests to hang when goroutine scheduling
// changes under -race.
func TestDB(t testing.TB) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: false,
	})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Apply same connection pool settings as OpenSQLite
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Apply essential pragmas (skip performance-only pragmas for in-memory test DBs)
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	}
	for _, pragma := range pragmas {
		if err := db.Exec(pragma).Error; err != nil {
			t.Fatalf("failed to set pragma %q: %v", pragma, err)
		}
	}

	// Auto-migrate all models
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("failed to auto-migrate test database: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err != nil {
			t.Logf("failed to get sql.DB for cleanup: %v", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			t.Logf("failed to close test database: %v", err)
		}
	})

	return db
}

// SeedData holds pre-created test data matching sync.yaml patterns
type SeedData struct {
	Config            *Config
	Clients           []*Client
	Organizations     []*Organization
	Repos             []*Repo
	Groups            []*Group
	Sources           []*Source
	GroupGlobals      []*GroupGlobal
	GroupDefaults     []*GroupDefault
	Targets           []*Target
	FileLists         []*FileList
	DirectoryLists    []*DirectoryList
	FileMappings      []*FileMapping
	DirectoryMappings []*DirectoryMapping
	Transforms        []*Transform
	FileListRefs      []*TargetFileListRef
	DirListRefs       []*TargetDirectoryListRef
}

// TestDBWithSeed creates a test database with realistic seed data
// Returns both the database and references to all created records
func TestDBWithSeed(t testing.TB) (*gorm.DB, *SeedData) {
	t.Helper()

	db := TestDB(t)
	seed := &SeedData{}

	// Create Config
	seed.Config = &Config{
		ExternalID: "test-config",
		Name:       "Test Configuration",
		Version:    1,
	}
	seed.Config.Metadata = Metadata{"test": "config"}
	if err := db.Create(seed.Config).Error; err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	// Create FileLists (must be created before targets reference them)
	aiFilesList := &FileList{
		ConfigID:    seed.Config.ID,
		ExternalID:  "ai-files",
		Name:        "AI Configuration Files",
		Description: "Standard AI config files",
		Position:    0,
	}
	aiFilesList.Metadata = Metadata{"category": "ai"}

	codecovList := &FileList{
		ConfigID:    seed.Config.ID,
		ExternalID:  "codecov-default",
		Name:        "Codecov Configuration",
		Description: "Default codecov settings",
		Position:    1,
	}
	codecovList.Metadata = Metadata{"category": "testing"}

	seed.FileLists = []*FileList{aiFilesList, codecovList}
	for _, fl := range seed.FileLists {
		if err := db.Create(fl).Error; err != nil {
			t.Fatalf("failed to seed file list: %v", err)
		}
	}

	// Create DirectoryLists
	workflowsList := &DirectoryList{
		ConfigID:    seed.Config.ID,
		ExternalID:  "github-workflows",
		Name:        "GitHub Workflows",
		Description: "Standard CI/CD workflows",
		Position:    0,
	}
	workflowsList.Metadata = Metadata{"category": "ci"}
	seed.DirectoryLists = []*DirectoryList{workflowsList}
	for _, dl := range seed.DirectoryLists {
		if err := db.Create(dl).Error; err != nil {
			t.Fatalf("failed to seed directory list: %v", err)
		}
	}

	// Create Client → Organization → Repo chain
	client := &Client{
		Name:        "MrZ",
		Description: "Test client",
	}
	client.Metadata = Metadata{"client": "test"}
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("failed to seed client: %v", err)
	}
	seed.Clients = []*Client{client}

	org := &Organization{
		ClientID:    client.ID,
		Name:        "mrz1836",
		Description: "MrZ GitHub org",
	}
	org.Metadata = Metadata{"org": "test"}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("failed to seed organization: %v", err)
	}
	seed.Organizations = []*Organization{org}

	sourceRepo := &Repo{
		OrganizationID: org.ID,
		Name:           "go-broadcast",
		Description:    "Source repo",
	}
	sourceRepo.Metadata = Metadata{"repo": "source"}
	if err := db.Create(sourceRepo).Error; err != nil {
		t.Fatalf("failed to seed source repo: %v", err)
	}

	targetRepo1 := &Repo{
		OrganizationID: org.ID,
		Name:           "test-repo-1",
		Description:    "Target repo 1",
	}
	if err := db.Create(targetRepo1).Error; err != nil {
		t.Fatalf("failed to seed target repo 1: %v", err)
	}

	targetRepo2 := &Repo{
		OrganizationID: org.ID,
		Name:           "test-repo-2",
		Description:    "Target repo 2",
	}
	if err := db.Create(targetRepo2).Error; err != nil {
		t.Fatalf("failed to seed target repo 2: %v", err)
	}
	seed.Repos = []*Repo{sourceRepo, targetRepo1, targetRepo2}

	// Create Group
	enabled := true
	mrzToolsGroup := &Group{
		ConfigID:    seed.Config.ID,
		ExternalID:  "mrz-tools",
		Name:        "MrZ Tools",
		Description: "Standard tooling sync group",
		Priority:    0,
		Enabled:     &enabled,
		Position:    0,
	}
	mrzToolsGroup.Metadata = Metadata{"owner": "mrz"}
	seed.Groups = []*Group{mrzToolsGroup}
	for _, g := range seed.Groups {
		if err := db.Create(g).Error; err != nil {
			t.Fatalf("failed to seed group: %v", err)
		}
	}

	// Create Source for group
	mainSource := &Source{
		GroupID:       seed.Groups[0].ID,
		RepoID:        sourceRepo.ID,
		Branch:        "master",
		BlobSizeLimit: "10m",
		SecurityEmail: "security@example.com",
		SupportEmail:  "support@example.com",
	}
	mainSource.Metadata = Metadata{"source": "main"}
	seed.Sources = []*Source{mainSource}
	for _, s := range seed.Sources {
		if err := db.Create(s).Error; err != nil {
			t.Fatalf("failed to seed source: %v", err)
		}
	}

	// Create GroupGlobal
	groupGlobal := &GroupGlobal{
		GroupID:         seed.Groups[0].ID,
		PRLabels:        JSONStringSlice{"automated-sync", "mrz-tools"},
		PRAssignees:     JSONStringSlice{"mrz1836"},
		PRReviewers:     JSONStringSlice{},
		PRTeamReviewers: JSONStringSlice{},
	}
	groupGlobal.Metadata = Metadata{"global": "config"}
	seed.GroupGlobals = []*GroupGlobal{groupGlobal}
	for _, gg := range seed.GroupGlobals {
		if err := db.Create(gg).Error; err != nil {
			t.Fatalf("failed to seed group global: %v", err)
		}
	}

	// Create GroupDefault
	groupDefault := &GroupDefault{
		GroupID:         seed.Groups[0].ID,
		BranchPrefix:    "chore/sync-files",
		PRLabels:        JSONStringSlice{"automated-sync"},
		PRAssignees:     JSONStringSlice{},
		PRReviewers:     JSONStringSlice{},
		PRTeamReviewers: JSONStringSlice{},
	}
	groupDefault.Metadata = Metadata{"defaults": "config"}
	seed.GroupDefaults = []*GroupDefault{groupDefault}
	for _, gd := range seed.GroupDefaults {
		if err := db.Create(gd).Error; err != nil {
			t.Fatalf("failed to seed group default: %v", err)
		}
	}

	// Create Targets
	target1 := &Target{
		GroupID:         seed.Groups[0].ID,
		RepoID:          targetRepo1.ID,
		Branch:          "main",
		BlobSizeLimit:   "",
		SecurityEmail:   "",
		SupportEmail:    "",
		PRLabels:        JSONStringSlice{"sync"},
		PRAssignees:     JSONStringSlice{},
		PRReviewers:     JSONStringSlice{},
		PRTeamReviewers: JSONStringSlice{},
		Position:        0,
	}
	target1.Metadata = Metadata{"target": "one"}

	target2 := &Target{
		GroupID:         seed.Groups[0].ID,
		RepoID:          targetRepo2.ID,
		Branch:          "develop",
		BlobSizeLimit:   "",
		SecurityEmail:   "",
		SupportEmail:    "",
		PRLabels:        JSONStringSlice{},
		PRAssignees:     JSONStringSlice{},
		PRReviewers:     JSONStringSlice{},
		PRTeamReviewers: JSONStringSlice{},
		Position:        1,
	}
	target2.Metadata = Metadata{"target": "two"}

	seed.Targets = []*Target{target1, target2}
	for _, tgt := range seed.Targets {
		if err := db.Create(tgt).Error; err != nil {
			t.Fatalf("failed to seed target: %v", err)
		}
	}

	// Create inline FileMappings for first target
	fileMapping1 := &FileMapping{
		OwnerType:  "target",
		OwnerID:    seed.Targets[0].ID,
		Src:        ".cursorrules",
		Dest:       ".cursorrules",
		DeleteFlag: false,
		Position:   0,
	}
	fileMapping1.Metadata = Metadata{"type": "inline"}

	fileMapping2 := &FileMapping{
		OwnerType:  "target",
		OwnerID:    seed.Targets[0].ID,
		Src:        "codecov.yml",
		Dest:       "codecov.yml",
		DeleteFlag: false,
		Position:   1,
	}
	fileMapping2.Metadata = Metadata{"type": "inline"}

	seed.FileMappings = []*FileMapping{fileMapping1, fileMapping2}
	for _, fm := range seed.FileMappings {
		if err := db.Create(fm).Error; err != nil {
			t.Fatalf("failed to seed file mapping: %v", err)
		}
	}

	// Create DirectoryMapping for second target
	preserveStructure := true
	includeHidden := true
	dirMapping := &DirectoryMapping{
		OwnerType:         "target",
		OwnerID:           seed.Targets[1].ID,
		Src:               ".github/workflows",
		Dest:              ".github/workflows",
		Exclude:           JSONStringSlice{"*.backup"},
		IncludeOnly:       JSONStringSlice{},
		PreserveStructure: &preserveStructure,
		IncludeHidden:     &includeHidden,
		DeleteFlag:        false,
		ModuleConfig:      nil,
		Position:          0,
	}
	dirMapping.Metadata = Metadata{"type": "inline"}
	seed.DirectoryMappings = []*DirectoryMapping{dirMapping}
	for _, dm := range seed.DirectoryMappings {
		if err := db.Create(dm).Error; err != nil {
			t.Fatalf("failed to seed directory mapping: %v", err)
		}
	}

	// Create Transform for first target
	transform := &Transform{
		OwnerType: "target",
		OwnerID:   seed.Targets[0].ID,
		RepoName:  true,
		Variables: JSONStringMap{
			"PROJECT_NAME": "test-repo-1",
			"OWNER":        "mrz1836",
		},
	}
	transform.Metadata = Metadata{"transform": "target"}
	seed.Transforms = []*Transform{transform}
	for _, tf := range seed.Transforms {
		if err := db.Create(tf).Error; err != nil {
			t.Fatalf("failed to seed transform: %v", err)
		}
	}

	// Create FileListRefs (M2M)
	seed.FileListRefs = []*TargetFileListRef{
		{
			TargetID:   seed.Targets[0].ID,
			FileListID: seed.FileLists[0].ID, // "ai-files"
			Position:   0,
			Metadata:   Metadata{"ref": "first"},
		},
		{
			TargetID:   seed.Targets[0].ID,
			FileListID: seed.FileLists[1].ID, // "codecov-default"
			Position:   1,
			Metadata:   Metadata{"ref": "second"},
		},
	}
	for _, ref := range seed.FileListRefs {
		if err := db.Create(ref).Error; err != nil {
			t.Fatalf("failed to seed file list ref: %v", err)
		}
	}

	// Create DirectoryListRefs (M2M)
	seed.DirListRefs = []*TargetDirectoryListRef{
		{
			TargetID:        seed.Targets[1].ID,
			DirectoryListID: seed.DirectoryLists[0].ID, // "github-workflows"
			Position:        0,
			Metadata:        Metadata{"ref": "workflows"},
		},
	}
	for _, ref := range seed.DirListRefs {
		if err := db.Create(ref).Error; err != nil {
			t.Fatalf("failed to seed directory list ref: %v", err)
		}
	}

	return db, seed
}
