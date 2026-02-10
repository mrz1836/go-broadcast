// +build bench_heavy

package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// BenchmarkDBQueryByFile benchmarks query by file path
func BenchmarkDBQueryByFile(b *testing.B) {
	// Create test database with multiple targets
	tmpDir := b.TempDir()
	tmpPath := filepath.Join(tmpDir, "bench-query-file.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(b, err)

	gormDB := database.DB()

	// Create config
	cfg := &db.Config{
		ExternalID: "bench-config",
		Name:       "Bench Config",
		Version:    1,
	}
	require.NoError(b, gormDB.Create(cfg).Error)

	// Create group
	group := &db.Group{
		ConfigID:   cfg.ID,
		ExternalID: "bench-group",
		Name:       "Bench Group",
	}
	require.NoError(b, gormDB.Create(group).Error)

	// Create 100 targets, each with the same file
	testFile := ".github/workflows/ci.yml"
	for i := 0; i < 100; i++ {
		target := &db.Target{
			GroupID: group.ID,
			Repo:    "mrz1836/bench-repo-" + string(rune(i)),
			Branch:  "main",
		}
		require.NoError(b, gormDB.Create(target).Error)

		// Add file mapping
		mapping := &db.FileMapping{
			OwnerType: "target",
			OwnerID:   target.ID,
			Src:       testFile,
			Dest:      testFile,
		}
		require.NoError(b, gormDB.Create(mapping).Error)
	}

	require.NoError(b, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldFile := dbQueryFile
	defer func() {
		dbPath = oldDBPath
		dbQueryFile = oldFile
	}()

	dbPath = tmpPath
	dbQueryFile = testFile
	dbQueryRepo = ""
	dbQueryFileList = ""
	dbQueryContains = ""
	dbQueryJSON = false

	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = runDBQuery(nil, nil)
	}
}

// BenchmarkDBQueryByRepo benchmarks query by repo
func BenchmarkDBQueryByRepo(b *testing.B) {
	// Create test database
	tmpDir := b.TempDir()
	tmpPath := filepath.Join(tmpDir, "bench-query-repo.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(b, err)

	gormDB := database.DB()

	// Create config
	cfg := &db.Config{
		ExternalID: "bench-config",
		Name:       "Bench Config",
		Version:    1,
	}
	require.NoError(b, gormDB.Create(cfg).Error)

	// Create group
	group := &db.Group{
		ConfigID:   cfg.ID,
		ExternalID: "bench-group",
		Name:       "Bench Group",
	}
	require.NoError(b, gormDB.Create(group).Error)

	// Create target with many file mappings
	target := &db.Target{
		GroupID: group.ID,
		Repo:    "mrz1836/target-repo",
		Branch:  "main",
	}
	require.NoError(b, gormDB.Create(target).Error)

	// Add 100 file mappings
	for i := 0; i < 100; i++ {
		mapping := &db.FileMapping{
			OwnerType: "target",
			OwnerID:   target.ID,
			Src:       "file-" + string(rune(i)) + ".txt",
			Dest:      "file-" + string(rune(i)) + ".txt",
		}
		require.NoError(b, gormDB.Create(mapping).Error)
	}

	require.NoError(b, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldRepo := dbQueryRepo
	defer func() {
		dbPath = oldDBPath
		dbQueryRepo = oldRepo
	}()

	dbPath = tmpPath
	dbQueryFile = ""
	dbQueryRepo = "mrz1836/target-repo"
	dbQueryFileList = ""
	dbQueryContains = ""
	dbQueryJSON = false

	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = runDBQuery(nil, nil)
	}
}

// BenchmarkDBQueryByPattern benchmarks pattern search
func BenchmarkDBQueryByPattern(b *testing.B) {
	// Create test database with various files
	tmpDir := b.TempDir()
	tmpPath := filepath.Join(tmpDir, "bench-query-pattern.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(b, err)

	gormDB := database.DB()

	// Create config
	cfg := &db.Config{
		ExternalID: "bench-config",
		Name:       "Bench Config",
		Version:    1,
	}
	require.NoError(b, gormDB.Create(cfg).Error)

	// Create group
	group := &db.Group{
		ConfigID:   cfg.ID,
		ExternalID: "bench-group",
		Name:       "Bench Group",
	}
	require.NoError(b, gormDB.Create(group).Error)

	// Create multiple targets with workflow files
	for i := 0; i < 50; i++ {
		target := &db.Target{
			GroupID: group.ID,
			Repo:    "mrz1836/bench-repo-" + string(rune(i)),
			Branch:  "main",
		}
		require.NoError(b, gormDB.Create(target).Error)

		// Add workflow file mappings
		for j := 0; j < 5; j++ {
			mapping := &db.FileMapping{
				OwnerType: "target",
				OwnerID:   target.ID,
				Src:       ".github/workflows/workflow-" + string(rune(j)) + ".yml",
				Dest:      ".github/workflows/workflow-" + string(rune(j)) + ".yml",
			}
			require.NoError(b, gormDB.Create(mapping).Error)
		}
	}

	require.NoError(b, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldContains := dbQueryContains
	defer func() {
		dbPath = oldDBPath
		dbQueryContains = oldContains
	}()

	dbPath = tmpPath
	dbQueryFile = ""
	dbQueryRepo = ""
	dbQueryFileList = ""
	dbQueryContains = "workflows"
	dbQueryJSON = false

	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = runDBQuery(nil, nil)
	}
}

// BenchmarkDBQueryByFileList benchmarks file list reference queries
func BenchmarkDBQueryByFileList(b *testing.B) {
	// Create test database
	tmpDir := b.TempDir()
	tmpPath := filepath.Join(tmpDir, "bench-query-list.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(b, err)

	gormDB := database.DB()

	// Create config
	cfg := &db.Config{
		ExternalID: "bench-config",
		Name:       "Bench Config",
		Version:    1,
	}
	require.NoError(b, gormDB.Create(cfg).Error)

	// Create file list
	fileList := &db.FileList{
		ConfigID:   cfg.ID,
		ExternalID: "test-files",
		Name:       "Test Files",
	}
	require.NoError(b, gormDB.Create(fileList).Error)

	// Create group
	group := &db.Group{
		ConfigID:   cfg.ID,
		ExternalID: "bench-group",
		Name:       "Bench Group",
	}
	require.NoError(b, gormDB.Create(group).Error)

	// Create 50 targets, all referencing the same file list
	for i := 0; i < 50; i++ {
		target := &db.Target{
			GroupID: group.ID,
			Repo:    "mrz1836/bench-repo-" + string(rune(i)),
			Branch:  "main",
		}
		require.NoError(b, gormDB.Create(target).Error)

		// Add file list ref
		ref := &db.TargetFileListRef{
			TargetID:   target.ID,
			FileListID: fileList.ID,
		}
		require.NoError(b, gormDB.Create(ref).Error)
	}

	require.NoError(b, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldFileList := dbQueryFileList
	defer func() {
		dbPath = oldDBPath
		dbQueryFileList = oldFileList
	}()

	dbPath = tmpPath
	dbQueryFile = ""
	dbQueryRepo = ""
	dbQueryFileList = "test-files"
	dbQueryContains = ""
	dbQueryJSON = false

	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = runDBQuery(nil, nil)
	}
}

// BenchmarkDBQueryJSON benchmarks JSON output serialization
func BenchmarkDBQueryJSON(b *testing.B) {
	// Create test database
	tmpDir := b.TempDir()
	tmpPath := filepath.Join(tmpDir, "bench-query-json.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(b, err)

	gormDB := database.DB()

	// Create config
	cfg := &db.Config{
		ExternalID: "bench-config",
		Name:       "Bench Config",
		Version:    1,
	}
	require.NoError(b, gormDB.Create(cfg).Error)

	// Create group
	group := &db.Group{
		ConfigID:   cfg.ID,
		ExternalID: "bench-group",
		Name:       "Bench Group",
	}
	require.NoError(b, gormDB.Create(group).Error)

	// Create 50 targets with the same file
	testFile := ".github/workflows/ci.yml"
	for i := 0; i < 50; i++ {
		target := &db.Target{
			GroupID: group.ID,
			Repo:    "mrz1836/bench-repo-" + string(rune(i)),
			Branch:  "main",
		}
		require.NoError(b, gormDB.Create(target).Error)

		// Add file mapping
		mapping := &db.FileMapping{
			OwnerType: "target",
			OwnerID:   target.ID,
			Src:       testFile,
			Dest:      testFile,
		}
		require.NoError(b, gormDB.Create(mapping).Error)
	}

	require.NoError(b, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldFile := dbQueryFile
	oldJSON := dbQueryJSON
	defer func() {
		dbPath = oldDBPath
		dbQueryFile = oldFile
		dbQueryJSON = oldJSON
	}()

	dbPath = tmpPath
	dbQueryFile = testFile
	dbQueryRepo = ""
	dbQueryFileList = ""
	dbQueryContains = ""
	dbQueryJSON = true

	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = runDBQuery(nil, nil)
	}
}
