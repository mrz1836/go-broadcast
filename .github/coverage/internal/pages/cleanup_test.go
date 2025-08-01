package pages

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	errTest1 = errors.New("cleanup failed: test error 1")
	errTest2 = errors.New("cleanup failed: test error 2")
)

func TestNewCleanupManager(t *testing.T) {
	basePath := "/tmp/test-cleanup"
	cm := NewCleanupManager(basePath)

	require.NotNil(t, cm)
	require.Equal(t, basePath, cm.BasePath)
	require.NotNil(t, cm.Config)

	// Check default configuration
	require.Equal(t, 30, cm.Config.PRDataRetentionDays)
	require.Equal(t, 90, cm.Config.BranchDataRetentionDays)
	require.Equal(t, 365, cm.Config.ReportRetentionDays)
	require.Equal(t, int64(500), cm.Config.MaxTotalSizeMB)
	require.Equal(t, 1000, cm.Config.MaxFilesPerType)
	require.True(t, cm.Config.PreserveMainBranch)
	require.Contains(t, cm.Config.PreserveProtectedBranches, "main")
	require.Contains(t, cm.Config.PreserveProtectedBranches, "master")
	require.Contains(t, cm.Config.PreserveProtectedBranches, "develop")
	require.False(t, cm.Config.DryRunMode)
	require.NotNil(t, cm.Config.PRPattern)
	require.NotNil(t, cm.Config.BranchPattern)
}

func TestCleanupManager_PerformCleanup(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	// Test dry run mode
	cm.Config.DryRunMode = true
	cm.Config.PRDataRetentionDays = 1
	cm.Config.BranchDataRetentionDays = 1

	// Create some test directory structure
	badgesDir := filepath.Join(tempDir, "badges")
	require.NoError(t, os.MkdirAll(badgesDir, 0o750))

	// Create an old file
	oldFile := filepath.Join(badgesDir, "old-badge.svg")
	require.NoError(t, os.WriteFile(oldFile, []byte("old badge"), 0o600))

	// Set file modification time to past
	pastTime := time.Now().Add(-48 * time.Hour) // 2 days ago
	require.NoError(t, os.Chtimes(oldFile, pastTime, pastTime))

	stats, err := cm.PerformCleanup(context.Background(), 0)
	require.NoError(t, err)
	require.NotNil(t, stats)

	// In dry run mode, files should not actually be deleted
	_, err = os.Stat(oldFile)
	require.NoError(t, err, "File should still exist in dry run mode")

	// Check stats
	require.Greater(t, stats.Duration, time.Duration(0))
	require.Empty(t, stats.Errors)
}

func TestCleanupManager_PerformCleanupWithMaxAge(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	maxAgeDays := 7
	stats, err := cm.PerformCleanup(context.Background(), maxAgeDays)
	require.NoError(t, err)
	require.NotNil(t, stats)

	// Verify that maxAgeDays override was applied
	require.Equal(t, maxAgeDays, cm.Config.PRDataRetentionDays)
	require.Equal(t, maxAgeDays, cm.Config.BranchDataRetentionDays)
}

func TestCleanupManager_ScanPRDirectories(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	// Create PR badges directory structure
	prBadgesDir := filepath.Join(tempDir, "badges", "pr")
	require.NoError(t, os.MkdirAll(prBadgesDir, 0o750))

	prReportsDir := filepath.Join(tempDir, "reports", "pr")
	require.NoError(t, os.MkdirAll(prReportsDir, 0o750))

	// Create test files
	oldBadge := filepath.Join(prBadgesDir, "123.svg")
	require.NoError(t, os.WriteFile(oldBadge, []byte("badge"), 0o600))

	oldReport := filepath.Join(prReportsDir, "456")
	require.NoError(t, os.MkdirAll(oldReport, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(oldReport, "index.html"), []byte("report"), 0o600))

	// Set files to be old
	pastTime := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(oldBadge, pastTime, pastTime))
	require.NoError(t, os.Chtimes(oldReport, pastTime, pastTime))

	cutoffTime := time.Now().Add(-24 * time.Hour)
	expired, err := cm.scanPRDirectories(context.Background(), cutoffTime)
	require.NoError(t, err)
	require.NotEmpty(t, expired)

	// Find the badge file in expired content
	var badgeFound bool
	for _, content := range expired {
		if content.Path == oldBadge && content.Type == ContentTypePR {
			badgeFound = true
			require.Equal(t, int64(5), content.Size) // "badge" is 5 bytes
			break
		}
	}
	require.True(t, badgeFound, "Old badge should be in expired content")
}

func TestCleanupManager_ScanBranchDirectories(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	// Create badges directory
	badgesDir := filepath.Join(tempDir, "badges")
	require.NoError(t, os.MkdirAll(badgesDir, 0o750))

	// Create test branch badges (non-protected)
	oldBranch := filepath.Join(badgesDir, "feature-branch.svg")
	require.NoError(t, os.WriteFile(oldBranch, []byte("badge"), 0o600))

	// Create protected branch badge (should not be cleaned)
	mainBranch := filepath.Join(badgesDir, "main.svg")
	require.NoError(t, os.WriteFile(mainBranch, []byte("badge"), 0o600))

	// Set files to be old
	pastTime := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(oldBranch, pastTime, pastTime))
	require.NoError(t, os.Chtimes(mainBranch, pastTime, pastTime))

	// Create reports directory
	reportsDir := filepath.Join(tempDir, "reports")
	require.NoError(t, os.MkdirAll(reportsDir, 0o750))

	// Create test branch report
	featureReportDir := filepath.Join(reportsDir, "feature-branch")
	require.NoError(t, os.MkdirAll(featureReportDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(featureReportDir, "report.html"), []byte("report"), 0o600))
	require.NoError(t, os.Chtimes(featureReportDir, pastTime, pastTime))

	cutoffTime := time.Now().Add(-24 * time.Hour)
	expired, err := cm.scanBranchDirectories(context.Background(), cutoffTime)
	require.NoError(t, err)

	// Should find the feature branch badge but not the main branch badge
	var featureBadgeFound, mainBadgeFound bool
	for _, content := range expired {
		if content.Path == oldBranch {
			featureBadgeFound = true
		}
		if content.Path == mainBranch {
			mainBadgeFound = true
		}
	}
	require.True(t, featureBadgeFound, "Feature branch badge should be in expired content")
	require.False(t, mainBadgeFound, "Main branch badge should be preserved")
}

func TestCleanupManager_ScanDirectory(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	testDir := filepath.Join(tempDir, "test")
	require.NoError(t, os.MkdirAll(testDir, 0o750))

	// Create test files and directories
	oldFile := filepath.Join(testDir, "old.txt")
	require.NoError(t, os.WriteFile(oldFile, []byte("content"), 0o600))

	oldDir := filepath.Join(testDir, "olddir")
	require.NoError(t, os.MkdirAll(oldDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "file.txt"), []byte("content"), 0o600))

	newFile := filepath.Join(testDir, "new.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("content"), 0o600))

	// Set old files to past time
	pastTime := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(oldFile, pastTime, pastTime))
	require.NoError(t, os.Chtimes(oldDir, pastTime, pastTime))

	var expired []ExpiredContent
	cutoffTime := time.Now().Add(-24 * time.Hour)
	err := cm.scanDirectory(testDir, ContentTypeBranch, cutoffTime, &expired)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(expired), 1) // At least oldFile should be expired

	// Check that we found the old file
	var fileFound bool
	for _, content := range expired {
		if content.Path == oldFile {
			fileFound = true
			require.Equal(t, ContentTypeBranch, content.Type)
			require.Equal(t, int64(7), content.Size) // "content" is 7 bytes
		}
	}
	require.True(t, fileFound, "Old file should be found")
}

func TestCleanupManager_RemoveExpiredContent(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	// Create test files
	testFile1 := filepath.Join(tempDir, "file1.txt")
	testFile2 := filepath.Join(tempDir, "dir1")
	require.NoError(t, os.WriteFile(testFile1, []byte("content"), 0o600))
	require.NoError(t, os.MkdirAll(testFile2, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testFile2, "inner.txt"), []byte("inner"), 0o600))

	expired := []ExpiredContent{
		{
			Path: testFile1,
			Type: ContentTypePR,
			Size: 7,
		},
		{
			Path: testFile2,
			Type: ContentTypeReport,
			Size: 5,
		},
	}

	stats, err := cm.removeExpiredContent(context.Background(), expired)
	require.NoError(t, err)
	require.NotNil(t, stats)

	require.Equal(t, 2, stats.FilesDeleted)
	require.Equal(t, 1, stats.PRFilesDeleted)
	require.Equal(t, 1, stats.ReportFilesDeleted)
	require.Equal(t, int64(12), stats.SizeFreed)

	// Verify files are actually deleted
	_, err = os.Stat(testFile1)
	require.True(t, os.IsNotExist(err), "File should be deleted")

	_, err = os.Stat(testFile2)
	require.True(t, os.IsNotExist(err), "Directory should be deleted")
}

func TestCleanupManager_ShouldPreserveBranch(t *testing.T) {
	cm := NewCleanupManager("/tmp")

	tests := []struct {
		branchName string
		expected   bool
	}{
		{"main", true},
		{"master", true},
		{"develop", true},
		{"feature-branch", false},
		{"bugfix/123", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.branchName, func(t *testing.T) {
			result := cm.shouldPreserveBranch(tt.branchName)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanupManager_IsFileExpired(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0o600))

	// Set file time to past
	pastTime := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(testFile, pastTime, pastTime))

	cutoffTime := time.Now().Add(-24 * time.Hour)

	// File should be expired
	require.True(t, cm.isFileExpired(testFile, cutoffTime))

	// File should not be expired with earlier cutoff
	earlierCutoff := time.Now().Add(-72 * time.Hour)
	require.False(t, cm.isFileExpired(testFile, earlierCutoff))

	// Non-existent file should not be expired
	require.False(t, cm.isFileExpired("/nonexistent", cutoffTime))
}

func TestCleanupManager_IsDirExpired(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	testDir := filepath.Join(tempDir, "testdir")
	require.NoError(t, os.MkdirAll(testDir, 0o750))

	// Create files with different timestamps
	oldFile := filepath.Join(testDir, "old.txt")
	newFile := filepath.Join(testDir, "new.txt")

	require.NoError(t, os.WriteFile(oldFile, []byte("old"), 0o600))
	require.NoError(t, os.WriteFile(newFile, []byte("new"), 0o600))

	// Set old file to past, new file to recent
	pastTime := time.Now().Add(-48 * time.Hour)
	recentTime := time.Now().Add(-1 * time.Hour)

	require.NoError(t, os.Chtimes(oldFile, pastTime, pastTime))
	require.NoError(t, os.Chtimes(newFile, recentTime, recentTime))

	cutoffTime := time.Now().Add(-24 * time.Hour)

	// Directory should not be expired because it has a recent file
	require.False(t, cm.isDirExpired(testDir, cutoffTime))

	// Directory should be expired with earlier cutoff
	veryRecentCutoff := time.Now()
	require.True(t, cm.isDirExpired(testDir, veryRecentCutoff))
}

func TestCleanupManager_GetFileSize(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("hello world")
	require.NoError(t, os.WriteFile(testFile, content, 0o600))

	size := cm.getFileSize(testFile)
	require.Equal(t, int64(len(content)), size)

	// Non-existent file should return 0
	size = cm.getFileSize("/nonexistent")
	require.Equal(t, int64(0), size)
}

func TestCleanupManager_GetDirSize(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	testDir := filepath.Join(tempDir, "testdir")
	require.NoError(t, os.MkdirAll(testDir, 0o750))

	// Create files
	file1 := filepath.Join(testDir, "file1.txt")
	file2 := filepath.Join(testDir, "subdir", "file2.txt")
	content1 := []byte("hello")
	content2 := []byte("world!")

	require.NoError(t, os.WriteFile(file1, content1, 0o600))
	require.NoError(t, os.MkdirAll(filepath.Dir(file2), 0o750))
	require.NoError(t, os.WriteFile(file2, content2, 0o600))

	size := cm.getDirSize(testDir)
	expectedSize := int64(len(content1) + len(content2))
	require.Equal(t, expectedSize, size)
}

func TestCleanupManager_GetFileModTime(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0o600))

	modTime := cm.getFileModTime(testFile)
	require.False(t, modTime.IsZero())

	// Non-existent file should return zero time
	zeroTime := cm.getFileModTime("/nonexistent")
	require.True(t, zeroTime.IsZero())
}

func TestCleanupManager_GetDirModTime(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCleanupManager(tempDir)

	testDir := filepath.Join(tempDir, "testdir")
	require.NoError(t, os.MkdirAll(testDir, 0o750))

	// Create files
	file1 := filepath.Join(testDir, "file1.txt")
	file2 := filepath.Join(testDir, "file2.txt")

	require.NoError(t, os.WriteFile(file1, []byte("content1"), 0o600))
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	require.NoError(t, os.WriteFile(file2, []byte("content2"), 0o600))

	modTime := cm.getDirModTime(testDir)
	require.False(t, modTime.IsZero())

	// Get modification time of file2 (should be more recent)
	file2Info, err := os.Stat(file2)
	require.NoError(t, err)

	// Directory mod time should be the most recent file's mod time
	require.True(t, modTime.Equal(file2Info.ModTime()) || modTime.After(file2Info.ModTime().Add(-time.Second)))
}

func TestFormatCleanupStats(t *testing.T) {
	stats := &CleanupStats{
		FilesDeleted:       10,
		PRFilesDeleted:     3,
		BranchFilesDeleted: 5,
		ReportFilesDeleted: 2,
		DirectoriesDeleted: 2,
		SizeFreed:          1024 * 1024, // 1MB
		Duration:           30 * time.Second,
		Errors:             []error{},
	}

	// Test normal mode
	result := FormatCleanupStats(stats, false)
	require.Contains(t, result, "Removed 10 files")
	require.Contains(t, result, "1.0 MB")
	require.Contains(t, result, "3 PR-related files")
	require.Contains(t, result, "5 branch-related files")
	require.Contains(t, result, "2 report files")
	require.Contains(t, result, "2 directories")
	require.Contains(t, result, "30s")

	// Test dry run mode
	result = FormatCleanupStats(stats, true)
	require.Contains(t, result, "Would remove 10 files")

	// Test empty stats
	emptyStats := &CleanupStats{
		FilesDeleted: 0,
		Duration:     5 * time.Second,
	}
	result = FormatCleanupStats(emptyStats, false)
	require.Contains(t, result, "No expired content found")

	// Test with errors - use static errors
	statsWithErrors := &CleanupStats{
		FilesDeleted: 5,
		SizeFreed:    512,
		Duration:     10 * time.Second,
		Errors:       []error{errTest1, errTest2},
	}
	result = FormatCleanupStats(statsWithErrors, false)
	require.Contains(t, result, "Errors encountered: 2")
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536 * 1024 * 1024, "1.5 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestContentType_String(t *testing.T) {
	tests := []struct {
		contentType ContentType
		expected    string
	}{
		{ContentTypePR, "pr"},
		{ContentTypeBranch, "branch"},
		{ContentTypeReport, "report"},
		{ContentTypeBadge, "badge"},
	}

	for _, tt := range tests {
		t.Run(string(tt.contentType), func(t *testing.T) {
			require.Equal(t, tt.expected, string(tt.contentType))
		})
	}
}

func TestExpiredContent_Struct(t *testing.T) {
	now := time.Now()
	content := ExpiredContent{
		Path:         "/test/path",
		Type:         ContentTypePR,
		Age:          24 * time.Hour,
		Size:         1024,
		PRNumber:     "42",
		BranchName:   "feature",
		LastModified: now,
	}

	require.Equal(t, "/test/path", content.Path)
	require.Equal(t, ContentTypePR, content.Type)
	require.Equal(t, 24*time.Hour, content.Age)
	require.Equal(t, int64(1024), content.Size)
	require.Equal(t, "42", content.PRNumber)
	require.Equal(t, "feature", content.BranchName)
	require.Equal(t, now, content.LastModified)
}
