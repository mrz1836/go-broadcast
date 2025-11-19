package sync

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestOriginalPRIssueScenario simulates the original issue from PR #23:
// - PR showed "Files processed: 22 (9 changed, 13 skipped)"
// - But only 4 files actually changed in git
// This test verifies the fix shows correct statistics
func TestOriginalPRIssueScenario(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise

	// Simulate the original scenario: go-broadcast examined 22 files
	// attempted to change 9 files, but only 4 actually changed in git
	rs := &RepositorySync{
		target: config.TargetConfig{
			Repo: "mrz1836/go-lucky",
		},
		logger: logger.WithField("component", "pr-test"),
		syncMetrics: &PerformanceMetrics{
			FileMetrics: FileProcessingMetrics{
				FilesProcessed:       22, // Total files examined (matches original)
				FilesChanged:         4,  // Actual git changes (the fix!)
				FilesAttempted:       9,  // Files go-broadcast attempted to change
				FilesSkipped:         13, // Files skipped during processing (22-9=13)
				FilesActuallyChanged: 4,  // Alias for clarity
				ProcessingTimeMs:     150,
			},
		},
	}

	// Simulate the attempted changes (9 files that go-broadcast tried to change)
	attemptedFiles := []FileChange{
		{Path: "README.md", IsNew: false},
		{Path: "config/settings.yml", IsNew: false},
		{Path: "src/main.go", IsNew: false},
		{Path: "src/utils.go", IsNew: false},
		{Path: "tests/main_test.go", IsNew: false},
		{Path: "docker-compose.yml", IsNew: false},
		{Path: "scripts/deploy.sh", IsNew: false},
		{Path: "docs/api.md", IsNew: false},
		{Path: "Dockerfile", IsNew: false},
	}

	// Simulate actual git changes (only 4 files actually changed)
	actualGitChanges := []string{
		"README.md",
		"src/main.go",
		"tests/main_test.go",
		"Dockerfile",
	}

	// Generate "What Changed" section
	var summaryBuilder strings.Builder
	rs.writeChangeSummary(&summaryBuilder, attemptedFiles, actualGitChanges)
	summary := summaryBuilder.String()

	// Generate Performance Metrics section
	var metricsBuilder strings.Builder
	rs.writePerformanceMetrics(&metricsBuilder)
	metrics := metricsBuilder.String()

	t.Logf("What Changed section:\n%s", summary)
	t.Logf("\nPerformance Metrics section:\n%s", metrics)

	// VERIFY THE FIX: Summary should show actual changes (4), not attempted (9)
	assert.Contains(t, summary, "4 individual file(s)",
		"Summary should show 4 actual changed files, not 9 attempted files")

	// Should NOT show the incorrect attempted count in summary
	assert.NotContains(t, summary, "9 individual file(s)",
		"Summary should not show attempted changes count")

	// VERIFY THE FIX: Metrics should show accurate breakdown
	// Before fix: would show "Files processed: 22 (9 changed, 13 skipped)"
	// After fix: should show "Files processed: 22 (4 changed, 13 skipped)"
	assert.Contains(t, metrics, "**Files processed**: 22",
		"Should show total files processed")
	assert.Contains(t, metrics, "4 changed",
		"Should show changed files")
	assert.Contains(t, metrics, "13 skipped",
		"Should show files skipped")
	assert.Contains(t, metrics, "**Files attempted to change**: 9",
		"Should show attempted changes breakdown")

	// The key verification: ensure we're NOT showing the old incorrect format
	assert.NotContains(t, metrics, "Files processed: 22 (9 changed, 13 skipped)",
		"Should not show old incorrect format")

	// Verify we show the new correct format (with markdown formatting)
	assert.Contains(t, metrics, "**Files processed**: 22 (4 changed, 0 deleted, 13 skipped)",
		"Should show new correct format with actual git changes")

	t.Logf("✅ Fix verified! PR will now show correct statistics:")
	t.Logf("   - What Changed: 4 individual file(s) (actual git changes)")
	t.Logf("   - Metrics: Files processed: 22 (4 changed, 0 deleted, 13 skipped)")
	t.Logf("   - Breakdown: Files attempted to change: 9")
}

// TestPRBodyGenerationAccuracy verifies the complete PR body shows accurate statistics
func TestPRBodyGenerationAccuracy(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	rs := &RepositorySync{
		target: config.TargetConfig{
			Repo: "test/repo",
		},
		logger: logger.WithField("component", "pr-test"),
		syncMetrics: &PerformanceMetrics{
			FileMetrics: FileProcessingMetrics{
				FilesProcessed:       15,
				FilesChanged:         3, // Actual git changes
				FilesAttempted:       7, // Attempted changes
				FilesSkipped:         8,
				FilesActuallyChanged: 3,
				ProcessingTimeMs:     200,
			},
		},
	}

	attemptedFiles := []FileChange{
		{Path: "file1.txt", IsNew: false},
		{Path: "file2.txt", IsNew: false},
		{Path: "file3.txt", IsNew: true},
	}

	actualChangedFiles := []string{"file1.txt", "file3.txt"}

	// Test individual components
	var summaryBuilder strings.Builder
	rs.writeChangeSummary(&summaryBuilder, attemptedFiles, actualChangedFiles)
	summary := summaryBuilder.String()

	// Verify the fix
	assert.Contains(t, summary, "2 individual file(s)",
		"Should show actual changed files count")
	assert.NotContains(t, summary, "3 individual file(s)",
		"Should not show attempted files count in summary")
}
