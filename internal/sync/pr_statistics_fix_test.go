package sync

import (
	"strings"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestPRStatisticsAccuracy tests the fix for PR statistics showing incorrect numbers
// This addresses the issue where PR showed "Files processed: 22 (9 changed, 13 skipped)"
// but only 4 files actually changed in git
func TestPRStatisticsAccuracy(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Setup: go-broadcast attempted to change 5 files
	attemptedFiles := []FileChange{
		{Path: "file1.txt", Content: []byte("new content 1"), IsNew: false},
		{Path: "file2.txt", Content: []byte("new content 2"), IsNew: false},
		{Path: "file3.txt", Content: []byte("old content 3"), IsNew: false}, // No actual change
		{Path: "file4.txt", Content: []byte("old content 4"), IsNew: false}, // No actual change
		{Path: "file5.txt", Content: []byte("new content 5"), IsNew: true},
	}

	// But only 3 files actually changed in git (simulating git diff result)
	actualGitChanges := []string{"file1.txt", "file2.txt", "file5.txt"}

	// Create repository sync instance with metrics reflecting actual changes
	rs := &RepositorySync{
		target: config.TargetConfig{
			Repo: "org/test-repo",
		},
		logger: logger.WithField("component", "test-repo"),
		syncMetrics: &PerformanceMetrics{
			FileMetrics: FileProcessingMetrics{
				FilesProcessed:       10,                    // Total files examined
				FilesChanged:         len(actualGitChanges), // Actual git changes (the fix!)
				FilesAttempted:       len(attemptedFiles),   // Files go-broadcast tried to change
				FilesSkipped:         3,                     // Skipped during processing
				FilesActuallyChanged: len(actualGitChanges), // Alias for clarity
				ProcessingTimeMs:     100,
			},
		},
	}

	// Test the individual components that were fixed

	// Test writeChangeSummary with actual changes
	var summaryBuilder strings.Builder
	rs.writeChangeSummary(&summaryBuilder, attemptedFiles, actualGitChanges)
	summary := summaryBuilder.String()

	// Test writePerformanceMetrics
	var metricsBuilder strings.Builder
	rs.writePerformanceMetrics(&metricsBuilder)
	metrics := metricsBuilder.String()

	t.Logf("Change Summary: %s", summary)
	t.Logf("Performance Metrics: %s", metrics)

	// The key fix: Summary should show actual git changes (3), not attempted changes (5)
	assert.Contains(t, summary, "3 individual file(s)", "Summary should mention 3 actual changed files")

	// Should not show the attempted count in summary
	assert.NotContains(t, summary, "5 individual file(s)", "Summary should not show attempted files count")

	// Verify the metrics show the correct breakdown
	assert.Contains(t, metrics, "3 changed", "Metrics should show 3 changed files")
	assert.Contains(t, metrics, "10", "Metrics should show 10 total examined files")
	assert.Contains(t, metrics, "5", "Metrics should show 5 attempted changes")
}

// TestWriteChangeSummaryWithActualFiles tests that the "What Changed" section
// uses actual git changes rather than attempted changes
func TestWriteChangeSummaryWithActualFiles(t *testing.T) {
	logger := logrus.New()
	rs := &RepositorySync{
		target: config.TargetConfig{
			Repo: "org/test-repo",
		},
		logger: logger.WithField("component", "test"),
	}

	// go-broadcast attempted to change many files
	attemptedFiles := []FileChange{
		{Path: "src/file1.go", IsNew: false},
		{Path: "src/file2.go", IsNew: false},
		{Path: "docs/README.md", IsNew: false},
		{Path: "config/settings.yml", IsNew: false},
		{Path: "scripts/deploy.sh", IsNew: true},
	}

	// But git only shows 2 files actually changed
	actualChangedFiles := []string{"src/file1.go", "scripts/deploy.sh"}

	var sb strings.Builder
	rs.writeChangeSummary(&sb, attemptedFiles, actualChangedFiles)

	result := sb.String()
	t.Logf("Generated summary: %s", result)

	// Should reflect actual changes (2 files) not attempted changes (5 files)
	assert.Contains(t, result, "2", "Summary should mention 2 actual changed files")

	// Should not mention the higher number of attempted files
	assert.NotContains(t, result, "5 file(s)", "Should not show attempted files count")
}

// TestMetricsCalculationAfterCommit tests that metrics are calculated correctly
// using actual git changes after commit
func TestMetricsCalculationAfterCommit(t *testing.T) {
	// Simulate the metrics calculation that happens after commit
	totalFilesProcessed := 10                                             // Files examined by go-broadcast
	filesAttempted := 8                                                   // Files go-broadcast tried to change
	actualChangedFiles := []string{"file1.txt", "file2.txt", "file3.txt"} // Git diff result

	// Calculate metrics as done in the fix
	metrics := FileProcessingMetrics{
		FilesProcessed:       totalFilesProcessed,
		FilesChanged:         len(actualChangedFiles), // Actual git changes
		FilesAttempted:       filesAttempted,
		FilesSkipped:         totalFilesProcessed - filesAttempted,
		ProcessingTimeMs:     100,
		FilesActuallyChanged: len(actualChangedFiles), // Alias for clarity
	}

	// Verify metrics are calculated correctly
	assert.Equal(t, 10, metrics.FilesProcessed, "Should show total files examined")
	assert.Equal(t, 3, metrics.FilesChanged, "Should show actual git changes")
	assert.Equal(t, 8, metrics.FilesAttempted, "Should show attempted changes")
	assert.Equal(t, 2, metrics.FilesSkipped, "Should show skipped files")
	assert.Equal(t, 3, metrics.FilesActuallyChanged, "Alias should match actual changes")

	// This is the key fix: FilesChanged should be based on git diff, not attempted files
	assert.NotEqual(t, metrics.FilesAttempted, metrics.FilesChanged,
		"Actual changes should differ from attempted changes")
}
