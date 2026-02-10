package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
)

// TestDBImportExportRoundTrip tests importing a YAML file and re-exporting it
func TestDBImportExportRoundTrip(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	testDBPath := filepath.Join(tmpDir, "test.db")
	exportPath := filepath.Join(tmpDir, "export.yaml")

	// Create test YAML configuration
	testConfig := &config.Config{
		Version: 1,
		Name:    "test-config",
		ID:      "test-cfg",
		FileLists: []config.FileList{
			{
				ID:          "test-files",
				Name:        "Test Files",
				Description: "Test file list",
				Files: []config.FileMapping{
					{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
					{Src: "README.md", Dest: "README.md"},
				},
			},
		},
		DirectoryLists: []config.DirectoryList{
			{
				ID:          "test-dirs",
				Name:        "Test Directories",
				Description: "Test directory list",
				Directories: []config.DirectoryMapping{
					{Src: "scripts/", Dest: "scripts/"},
				},
			},
		},
		Groups: []config.Group{
			{
				Name:        "Test Group",
				ID:          "test-group",
				Description: "A test group",
				Priority:    10,
				Source: config.SourceConfig{
					Repo:   "mrz1836/template",
					Branch: "main",
				},
				Global: config.GlobalConfig{
					PRLabels: []string{"automated"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "sync",
					PRLabels:     []string{"sync"},
				},
				Targets: []config.TargetConfig{
					{
						Repo:              "mrz1836/target1",
						Branch:            "main",
						FileListRefs:      []string{"test-files"},
						DirectoryListRefs: []string{"test-dirs"},
						Files: []config.FileMapping{
							{Src: "LICENSE", Dest: "LICENSE"},
						},
					},
					{
						Repo: "mrz1836/target2",
						Directories: []config.DirectoryMapping{
							{Src: "docs/", Dest: "docs/"},
						},
					},
				},
			},
		},
	}

	// Write test YAML file
	yamlPath := filepath.Join(tmpDir, "test.yaml")
	yamlData, err := yaml.Marshal(testConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(yamlPath, yamlData, 0o600))

	// Initialize database
	database, err := db.Open(db.OpenOptions{
		Path:        testDBPath,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	converter := db.NewConverter(database.DB())

	// Import configuration
	dbConfig, err := converter.ImportConfig(ctx, testConfig)
	require.NoError(t, err)
	assert.Equal(t, "test-config", dbConfig.Name)
	assert.Equal(t, "test-cfg", dbConfig.ExternalID)

	// Verify import counts
	var groupCount, targetCount, fileListCount, dirListCount int64
	database.DB().Model(&db.Group{}).Where("config_id = ?", dbConfig.ID).Count(&groupCount)
	assert.Equal(t, int64(1), groupCount)

	database.DB().Model(&db.FileList{}).Where("config_id = ?", dbConfig.ID).Count(&fileListCount)
	assert.Equal(t, int64(1), fileListCount)

	database.DB().Model(&db.DirectoryList{}).Where("config_id = ?", dbConfig.ID).Count(&dirListCount)
	assert.Equal(t, int64(1), dirListCount)

	database.DB().Model(&db.Target{}).Joins("JOIN groups ON targets.group_id = groups.id").
		Where("groups.config_id = ?", dbConfig.ID).Count(&targetCount)
	assert.Equal(t, int64(2), targetCount)

	// Export configuration
	exportedConfig, err := converter.ExportConfig(ctx, "test-cfg")
	require.NoError(t, err)

	// Verify exported structure
	assert.Equal(t, testConfig.Name, exportedConfig.Name)
	assert.Equal(t, testConfig.ID, exportedConfig.ID)
	assert.Equal(t, testConfig.Version, exportedConfig.Version)
	assert.Len(t, exportedConfig.Groups, 1)
	assert.Len(t, exportedConfig.FileLists, 1)
	assert.Len(t, exportedConfig.DirectoryLists, 1)

	// Verify group details
	exportedGroup := exportedConfig.Groups[0]
	assert.Equal(t, "Test Group", exportedGroup.Name)
	assert.Equal(t, "test-group", exportedGroup.ID)
	assert.Equal(t, 10, exportedGroup.Priority)
	assert.Equal(t, "mrz1836/template", exportedGroup.Source.Repo)
	assert.Equal(t, "main", exportedGroup.Source.Branch)
	assert.Len(t, exportedGroup.Targets, 2)

	// Verify targets
	target1 := exportedGroup.Targets[0]
	assert.Equal(t, "mrz1836/target1", target1.Repo)
	assert.Equal(t, "main", target1.Branch)
	assert.Contains(t, target1.FileListRefs, "test-files")
	assert.Contains(t, target1.DirectoryListRefs, "test-dirs")
	assert.Len(t, target1.Files, 1)

	target2 := exportedGroup.Targets[1]
	assert.Equal(t, "mrz1836/target2", target2.Repo)
	assert.Len(t, target2.Directories, 1)

	// Verify file list
	fileList := exportedConfig.FileLists[0]
	assert.Equal(t, "test-files", fileList.ID)
	assert.Equal(t, "Test Files", fileList.Name)
	assert.Len(t, fileList.Files, 2)

	// Write exported YAML
	exportedYAML, err := yaml.Marshal(exportedConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(exportPath, exportedYAML, 0o600))

	// Load exported YAML and verify it's valid
	loadedConfig, err := config.Load(exportPath)
	require.NoError(t, err)
	assert.Equal(t, exportedConfig.Name, loadedConfig.Name)
	assert.Equal(t, exportedConfig.ID, loadedConfig.ID)
}

// TestDBImportForceReplace tests that force flag replaces existing config
func TestDBImportForceReplace(t *testing.T) {
	tmpDir := t.TempDir()
	testDBPath := filepath.Join(tmpDir, "test.db")

	// Initialize database
	database, err := db.Open(db.OpenOptions{
		Path:        testDBPath,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	converter := db.NewConverter(database.DB())

	// Create initial config
	initialConfig := &config.Config{
		Version: 1,
		Name:    "initial",
		ID:      "test-cfg",
		Groups: []config.Group{
			{
				Name: "Group1",
				ID:   "group1",
				Source: config.SourceConfig{
					Repo:   "mrz1836/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target1"},
				},
			},
		},
	}

	// Import initial config
	_, err = converter.ImportConfig(ctx, initialConfig)
	require.NoError(t, err)

	// Verify initial state
	var groupCount int64
	database.DB().Model(&db.Group{}).Count(&groupCount)
	assert.Equal(t, int64(1), groupCount)

	// Create updated config with same ID
	updatedConfig := &config.Config{
		Version: 1,
		Name:    "updated",
		ID:      "test-cfg",
		Groups: []config.Group{
			{
				Name: "Group1",
				ID:   "group1",
				Source: config.SourceConfig{
					Repo:   "mrz1836/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target1"},
					{Repo: "mrz1836/target2"},
				},
			},
			{
				Name: "Group2",
				ID:   "group2",
				Source: config.SourceConfig{
					Repo:   "mrz1836/source2",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target3"},
				},
			},
		},
	}

	// Import updated config (force replace)
	_, err = converter.ImportConfig(ctx, updatedConfig)
	require.NoError(t, err)

	// Verify updated state
	database.DB().Model(&db.Group{}).Count(&groupCount)
	assert.Equal(t, int64(2), groupCount)

	var targetCount int64
	database.DB().Model(&db.Target{}).Count(&targetCount)
	assert.Equal(t, int64(3), targetCount)

	// Export and verify name changed
	exportedConfig, err := converter.ExportConfig(ctx, "test-cfg")
	require.NoError(t, err)
	assert.Equal(t, "updated", exportedConfig.Name)
	assert.Len(t, exportedConfig.Groups, 2)
}

// TestDBDiffDetectsChanges tests that diff command detects configuration differences
func TestDBDiffDetectsChanges(t *testing.T) {
	tmpDir := t.TempDir()
	testDBPath := filepath.Join(tmpDir, "test.db")

	// Initialize database
	database, err := db.Open(db.OpenOptions{
		Path:        testDBPath,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	converter := db.NewConverter(database.DB())

	// Create config in database
	dbConfigData := &config.Config{
		Version: 1,
		Name:    "db-version",
		ID:      "test-cfg",
		Groups: []config.Group{
			{
				Name: "Group1",
				ID:   "group1",
				Source: config.SourceConfig{
					Repo:   "mrz1836/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target1"},
				},
			},
		},
	}

	_, err = converter.ImportConfig(ctx, dbConfigData)
	require.NoError(t, err)

	// Create different YAML config
	yamlConfigData := &config.Config{
		Version: 1,
		Name:    "yaml-version",
		ID:      "test-cfg",
		Groups: []config.Group{
			{
				Name: "Group1",
				ID:   "group1",
				Source: config.SourceConfig{
					Repo:   "mrz1836/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target1"},
					{Repo: "mrz1836/target2"},
				},
			},
			{
				Name: "Group2",
				ID:   "group2",
				Source: config.SourceConfig{
					Repo:   "mrz1836/source2",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target3"},
				},
			},
		},
	}

	// Export database config
	exportedDBConfig, err := converter.ExportConfig(ctx, "test-cfg")
	require.NoError(t, err)

	// Compare configs
	diffs := compareConfigs(yamlConfigData, exportedDBConfig, false)

	// Verify differences detected
	assert.NotEmpty(t, diffs, "Should detect differences between configs")

	// Check for specific differences
	diffStr := ""
	for _, d := range diffs {
		diffStr += d + "\n"
	}

	assert.Contains(t, diffStr, "name differs", "Should detect name difference")
	assert.Contains(t, diffStr, "Group count differs", "Should detect group count difference")
}

// TestDBImportValidatesConfig tests that import validates configuration before importing
func TestDBImportValidatesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	testDBPath := filepath.Join(tmpDir, "test.db")

	// Initialize database
	database, err := db.Open(db.OpenOptions{
		Path:        testDBPath,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	converter := db.NewConverter(database.DB())

	// Create invalid config (missing required repo field)
	invalidConfig := &config.Config{
		Version: 1,
		Name:    "invalid",
		ID:      "test-invalid",
		Groups: []config.Group{
			{
				Name: "InvalidGroup",
				ID:   "invalid",
				Source: config.SourceConfig{
					Repo:   "", // Invalid: empty repo
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target1"},
				},
			},
		},
	}

	// Attempt to import (should fail validation)
	_, err = converter.ImportConfig(ctx, invalidConfig)
	assert.Error(t, err, "Should reject invalid configuration")
}

// TestDBExportEmptyDatabase tests exporting from an empty database
func TestDBExportEmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	testDBPath := filepath.Join(tmpDir, "test.db")

	// Initialize empty database
	database, err := db.Open(db.OpenOptions{
		Path:        testDBPath,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	converter := db.NewConverter(database.DB())

	// Attempt to export from empty database
	_, err = converter.ExportConfig(ctx, "nonexistent")
	assert.Error(t, err, "Should fail to export from empty database")
}

// TestDBImportReferenceResolution tests that file list and directory list references are correctly resolved
func TestDBImportReferenceResolution(t *testing.T) {
	tmpDir := t.TempDir()
	testDBPath := filepath.Join(tmpDir, "test.db")

	// Initialize database
	database, err := db.Open(db.OpenOptions{
		Path:        testDBPath,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	converter := db.NewConverter(database.DB())

	// Create config with references
	testConfig := &config.Config{
		Version: 1,
		Name:    "ref-test",
		ID:      "ref-test",
		FileLists: []config.FileList{
			{
				ID:   "common-files",
				Name: "Common Files",
				Files: []config.FileMapping{
					{Src: "README.md", Dest: "README.md"},
				},
			},
		},
		DirectoryLists: []config.DirectoryList{
			{
				ID:   "common-dirs",
				Name: "Common Directories",
				Directories: []config.DirectoryMapping{
					{Src: "scripts/", Dest: "scripts/"},
				},
			},
		},
		Groups: []config.Group{
			{
				Name: "RefGroup",
				ID:   "ref-group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:              "mrz1836/target1",
						FileListRefs:      []string{"common-files"},
						DirectoryListRefs: []string{"common-dirs"},
					},
				},
			},
		},
	}

	// Import
	_, err = converter.ImportConfig(ctx, testConfig)
	require.NoError(t, err)

	// Verify references were resolved
	var refCount int64
	database.DB().Model(&db.TargetFileListRef{}).Count(&refCount)
	assert.Equal(t, int64(1), refCount, "Should have one file list reference")

	database.DB().Model(&db.TargetDirectoryListRef{}).Count(&refCount)
	assert.Equal(t, int64(1), refCount, "Should have one directory list reference")

	// Export and verify references are preserved
	exportedConfig, err := converter.ExportConfig(ctx, "ref-test")
	require.NoError(t, err)

	exportedTarget := exportedConfig.Groups[0].Targets[0]
	assert.Contains(t, exportedTarget.FileListRefs, "common-files")
	assert.Contains(t, exportedTarget.DirectoryListRefs, "common-dirs")
}
