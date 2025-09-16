package sync

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// TestDirectoryStatsRegressionInPRMetadata tests the complete flow from directory processing
// to PR metadata generation to ensure directory stats are no longer zero.
// This is a regression test for the issue where all directory stats showed "files_synced: 0"
func TestDirectoryStatsRegressionInPRMetadata(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.WarnLevel) // Reduce log noise

	t.Run("PR metadata includes correct directory stats after processing", func(t *testing.T) {
		// Create a RepositorySync with directory configuration
		repoSync := &RepositorySync{
			sourceState: &state.SourceState{
				Repo:         "mrz1836/go-broadcast",
				Branch:       "master",
				LatestCommit: "abc123",
			},
			target: config.TargetConfig{
				Repo: "skyetel/reach",
				Directories: []config.DirectoryMapping{
					{
						Src:     ".github/workflows",
						Dest:    ".github/workflows",
						Exclude: []string{"scorecard.yml", "codeql-analysis.yml"},
					},
					{
						Src:  ".github/actions",
						Dest: ".github/actions",
					},
					{
						Src:  ".vscode",
						Dest: ".vscode",
					},
				},
			},
			logger: logger,
		}

		// Simulate directory processing by creating and using the processDirectoriesWithMetrics flow
		// This tests the actual path that was broken in the issue

		// Initialize syncMetrics like the real Execute method does
		repoSync.syncMetrics = &PerformanceMetrics{
			StartTime:        time.Now(),
			DirectoryMetrics: make(map[string]DirectoryMetrics),
		}

		// Simulate directory processing with realistic file counts
		// This recreates the scenario where directory stats were zero
		manager := NewDirectoryProgressManager(logger)

		directoriesProcessed := map[string]int{
			".github/workflows": 8, // These would have been zero before the fix
			".github/actions":   3, // These would have been zero before the fix
			".vscode":           2, // These would have been zero before the fix
		}

		// Simulate the directory processing flow that calls UpdateProgress
		for dirPath, fileCount := range directoriesProcessed {
			reporter := manager.GetReporter(dirPath, 5) // Low threshold to ensure enabled
			reporter.Start(fileCount)                   // Start with discovered files

			// Simulate batch processing calling UpdateProgress for each file
			// This is the key part that was broken - UpdateProgress wasn't setting FilesProcessed
			for i := 1; i <= fileCount; i++ {
				reporter.UpdateProgress(i, fileCount, "Processing file")
			}

			// Complete the directory processing
			reporter.Complete()
		}

		// Get the metrics (simulating processDirectoriesWithMetrics)
		directoryMetrics := manager.GetAllMetrics()

		// Verify that our fix works - FilesProcessed should not be zero
		for dirPath, expectedCount := range directoriesProcessed {
			assert.Contains(t, directoryMetrics, dirPath, "Should have metrics for directory %s", dirPath)
			actualMetrics := directoryMetrics[dirPath]

			// This is the critical assertion - before the fix, FilesProcessed would be 0
			assert.Equal(t, expectedCount, actualMetrics.FilesProcessed,
				"Directory %s should have correct FilesProcessed count (this was zero before the fix)", dirPath)
			assert.Equal(t, expectedCount, actualMetrics.FilesDiscovered,
				"Directory %s should have correct FilesDiscovered count", dirPath)
		}

		// Store the directory metrics in the repository sync (like the real flow)
		repoSync.syncMetrics.DirectoryMetrics = directoryMetrics

		// Set file metrics to simulate the complete picture
		repoSync.syncMetrics.FileMetrics = FileProcessingMetrics{
			FilesProcessed: 2, // Individual files processed
			FilesChanged:   8, // Actual git changes
			FilesSkipped:   5,
		}

		// Now generate PR body to ensure the stats appear correctly in metadata
		files := []FileChange{
			{Path: ".github/workflows/ci.yml", IsNew: false},
			{Path: ".github/actions/setup/action.yml", IsNew: true},
		}

		// Pass realistic actual changed files that map to our directories
		actualChangedFiles := []string{
			".github/workflows/ci.yml",
			".github/workflows/test.yml",
			".github/actions/setup/action.yml",
			".vscode/settings.json",
		}

		// Simulate the updateDirectoryMetricsWithActualChanges logic
		for dirPath := range directoryMetrics {
			metrics := directoryMetrics[dirPath]
			metrics.FilesChanged = 0 // Reset
			directoryMetrics[dirPath] = metrics
		}

		// Count actual changes per directory
		for _, filePath := range actualChangedFiles {
			for _, dirMapping := range repoSync.target.Directories {
				if isFileInDirectory(filePath, dirMapping.Dest) {
					if metrics, exists := directoryMetrics[dirMapping.Src]; exists {
						metrics.FilesChanged++
						directoryMetrics[dirMapping.Src] = metrics
						break
					}
				}
			}
		}

		// Update the repoSync with the new metrics
		repoSync.syncMetrics.DirectoryMetrics = directoryMetrics

		prBody := repoSync.generatePRBody("testcommit", files, actualChangedFiles)

		// Verify that the PR body contains correct directory stats in the YAML metadata
		// files_synced now shows actual changes, not processed files
		assert.Contains(t, prBody, "directories:", "PR body should contain directories section")
		assert.Contains(t, prBody, "files_synced: 2", "Should show correct files_synced for .github/workflows")
		assert.Contains(t, prBody, "files_synced: 1", "Should show correct files_synced for .github/actions")
		assert.Contains(t, prBody, "files_synced: 1", "Should show correct files_synced for .vscode")

		// Verify the human-readable section also shows correct stats
		assert.Contains(t, prBody, "**Files synced**: 2", "Should show correct files synced in human readable format")
		assert.Contains(t, prBody, "**Files synced**: 1", "Should show correct files synced in human readable format")

		// Verify files examined are also shown
		assert.Contains(t, prBody, "**Files examined**: 8", "Should show files examined for .github/workflows")
		assert.Contains(t, prBody, "**Files examined**: 3", "Should show files examined for .github/actions")
		assert.Contains(t, prBody, "**Files examined**: 2", "Should show files examined for .vscode")

		// Verify that no directories show zero files synced (the bug we fixed)
		assert.NotContains(t, prBody, "files_synced: 0", "No directory should show zero files synced")
		assert.NotContains(t, prBody, "**Files synced**: 0", "No directory should show zero files synced in human readable format")
	})

	t.Run("disabled reporter still contributes correct stats to PR metadata", func(t *testing.T) {
		// Test the specific edge case where progress reporting is disabled (below threshold)
		// but we still want accurate stats in PR metadata
		repoSync := &RepositorySync{
			sourceState: &state.SourceState{
				Repo:         "mrz1836/go-broadcast",
				Branch:       "master",
				LatestCommit: "def456",
			},
			target: config.TargetConfig{
				Repo: "test/repo",
				Directories: []config.DirectoryMapping{
					{
						Src:  ".vscode",
						Dest: ".vscode",
					},
				},
			},
			logger: logger,
		}

		repoSync.syncMetrics = &PerformanceMetrics{
			StartTime:        time.Now(),
			DirectoryMetrics: make(map[string]DirectoryMetrics),
		}

		manager := NewDirectoryProgressManager(logger)

		// Use high threshold to disable progress reporting, but still track metrics
		reporter := manager.GetReporter(".vscode", 50) // High threshold
		reporter.Start(3)                              // Below threshold - reporting disabled

		// Process files - this should still track FilesProcessed even when reporting is disabled
		reporter.UpdateProgress(1, 3, "File 1")
		reporter.UpdateProgress(2, 3, "File 2")
		reporter.UpdateProgress(3, 3, "File 3")
		reporter.Complete()

		directoryMetrics := manager.GetAllMetrics()
		repoSync.syncMetrics.DirectoryMetrics = directoryMetrics

		repoSync.syncMetrics.FileMetrics = FileProcessingMetrics{
			FilesProcessed: 0,
			FilesChanged:   1,
			FilesSkipped:   2,
		}

		// Set up realistic actual changed files
		actualChangedFiles := []string{".vscode/settings.json"}

		// Update FilesChanged based on actual changes (simulate updateDirectoryMetricsWithActualChanges)
		if metrics, exists := directoryMetrics[".vscode"]; exists {
			metrics.FilesChanged = 1 // One file actually changed
			directoryMetrics[".vscode"] = metrics
			repoSync.syncMetrics.DirectoryMetrics = directoryMetrics
		}

		prBody := repoSync.generatePRBody("testcommit", []FileChange{{Path: ".vscode/settings.json"}}, actualChangedFiles)

		// Even with disabled reporting, the stats should be correct in PR metadata
		assert.Contains(t, prBody, "files_synced: 1", "Should track correct stats even when progress reporting is disabled")
		assert.Contains(t, prBody, "**Files synced**: 1", "Should show correct stats in human readable format")
		assert.Contains(t, prBody, "**Files examined**: 3", "Should show files examined count")
		assert.NotContains(t, prBody, "files_synced: 0", "Should not show zero stats")
	})
}

// TestDirectoryStatsRegressionScenarios tests specific scenarios that would have failed before the fix
func TestDirectoryStatsRegressionScenarios(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.WarnLevel)

	t.Run("rate limited progress updates still track file counts", func(t *testing.T) {
		// Test that rate limiting of log messages doesn't affect metrics tracking
		manager := NewDirectoryProgressManager(logger)

		reporter := manager.GetReporter(".github/workflows", 1)
		reporter.SetUpdateInterval(1000 * time.Millisecond) // Long interval for rate limiting
		reporter.Start(5)

		// Make rapid progress updates that would be rate limited for logging
		reporter.UpdateProgress(1, 5, "File 1")
		reporter.UpdateProgress(2, 5, "File 2") // This would be rate limited for logging
		reporter.UpdateProgress(3, 5, "File 3") // This would be rate limited for logging
		reporter.UpdateProgress(4, 5, "File 4") // This would be rate limited for logging
		reporter.UpdateProgress(5, 5, "File 5") // This would be rate limited for logging

		metrics := reporter.GetMetrics()

		// Despite rate limiting, metrics should track all progress
		assert.Equal(t, 5, metrics.FilesProcessed, "Rate limiting should not affect metrics tracking")
	})

	t.Run("multiple directories with different file counts", func(t *testing.T) {
		// Simulate the exact scenario from the original PR that had zero stats
		manager := NewDirectoryProgressManager(logger)

		testDirs := []struct {
			path  string
			files int
		}{
			{".github/ISSUE_TEMPLATE", 0},   // Empty directory
			{".github/workflows", 2},        // Files with exclusions
			{".github/actions", 0},          // No files processed
			{".vscode", 1},                  // Single file
			{".github/tech-conventions", 0}, // Another empty directory
		}

		for _, dir := range testDirs {
			reporter := manager.GetReporter(dir.path, 50) // High threshold
			reporter.Start(dir.files)

			// Only call UpdateProgress if there are files to process
			if dir.files > 0 {
				for i := 1; i <= dir.files; i++ {
					reporter.UpdateProgress(i, dir.files, "Processing file")
				}
			}
			reporter.Complete()
		}

		allMetrics := manager.GetAllMetrics()

		// Verify each directory has correct stats (not zero)
		for _, dir := range testDirs {
			require.Contains(t, allMetrics, dir.path, "Should have metrics for directory %s", dir.path)
			assert.Equal(t, dir.files, allMetrics[dir.path].FilesProcessed,
				"Directory %s should have correct FilesProcessed (%d, not 0)", dir.path, dir.files)
		}
	})
}

// isFileInDirectory checks if a file path belongs to a specific directory (helper for tests)
func isFileInDirectory(filePath, directoryPath string) bool {
	// Normalize paths to use forward slashes
	filePath = filepath.ToSlash(filePath)
	directoryPath = filepath.ToSlash(directoryPath)

	// Check if file is directly in the directory or a subdirectory
	return strings.HasPrefix(filePath, directoryPath+"/") || filePath == directoryPath
}
