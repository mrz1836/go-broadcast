package history

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

func TestNew(t *testing.T) {
	tracker := New()
	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.config)
	assert.Equal(t, ".github/coverage/history", tracker.config.StoragePath)
	assert.Equal(t, 90, tracker.config.RetentionDays)
	assert.Equal(t, 1000, tracker.config.MaxEntries)
	assert.Equal(t, 6, tracker.config.CompressionLevel)
	assert.True(t, tracker.config.AutoCleanup)
	assert.True(t, tracker.config.MetricsEnabled)
}

func TestNewWithConfig(t *testing.T) {
	config := &Config{
		StoragePath:      "/tmp/custom",
		RetentionDays:    30,
		MaxEntries:       500,
		CompressionLevel: 9,
		AutoCleanup:      false,
		MetricsEnabled:   false,
	}

	tracker := NewWithConfig(config)
	assert.NotNil(t, tracker)
	assert.Equal(t, config, tracker.config)
}

func TestRecord(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{
		StoragePath:    tempDir,
		RetentionDays:  30,
		MaxEntries:     100,
		AutoCleanup:    true,
		MetricsEnabled: true,
	}

	tracker := NewWithConfig(config)
	ctx := context.Background()
	coverage := createTestCoverage()

	err = tracker.Record(ctx, coverage,
		WithBranch("main"),
		WithCommit("abc123", "https://github.com/test/repo/commit/abc123"),
		WithMetadata("project", "test-project"),
	)

	require.NoError(t, err)

	// Verify file was created
	files, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestRecordContextCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	coverage := createTestCoverage()
	err = tracker.Record(ctx, coverage)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestGetTrend(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Record multiple entries
	for i := 0; i < 5; i++ {
		coverage := createTestCoverage()
		coverage.Percentage = float64(70 + i*5) // 70%, 75%, 80%, 85%, 90%

		recordErr := tracker.Record(ctx, coverage,
			WithBranch("main"),
			WithCommit("commit"+string(rune('1'+i)), ""),
		)
		require.NoError(t, recordErr)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get trend data
	trendData, err := tracker.GetTrend(ctx,
		WithTrendBranch("main"),
		WithTrendDays(7),
		WithMaxDataPoints(10),
	)

	require.NoError(t, err)
	assert.NotNil(t, trendData)
	assert.Len(t, trendData.Entries, 5)
	assert.NotNil(t, trendData.Summary)
	assert.NotNil(t, trendData.Analysis)
	assert.Equal(t, "up", trendData.Summary.CurrentTrend)
}

func TestGetTrendEmpty(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	trendData, err := tracker.GetTrend(ctx)

	require.NoError(t, err)
	assert.NotNil(t, trendData)
	assert.Empty(t, trendData.Entries)
	assert.NotNil(t, trendData.Summary)
	assert.NotNil(t, trendData.Analysis)
}

func TestGetLatestEntry(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Record entries
	coverage1 := createTestCoverage()
	coverage1.Percentage = 75.0
	err = tracker.Record(ctx, coverage1, WithBranch("main"), WithCommit("commit1", ""))
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	coverage2 := createTestCoverage()
	coverage2.Percentage = 85.0
	err = tracker.Record(ctx, coverage2, WithBranch("main"), WithCommit("commit2", ""))
	require.NoError(t, err)

	// Get latest entry
	latest, err := tracker.GetLatestEntry(ctx, "main")
	require.NoError(t, err)
	assert.NotNil(t, latest)
	assert.InDelta(t, 85.0, latest.Coverage.Percentage, 0.001)
	assert.Equal(t, "commit2", latest.CommitSHA)
}

func TestGetLatestEntryNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	_, err = tracker.GetLatestEntry(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no entries found for branch: nonexistent")
}

func TestCleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{
		StoragePath:   tempDir,
		RetentionDays: 1, // Very short retention for testing
		MaxEntries:    2, // Very low max entries
		AutoCleanup:   true,
	}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Record 3 entries (more than MaxEntries)
	for i := 0; i < 3; i++ {
		coverage := createTestCoverage()
		recordErr := tracker.Record(ctx, coverage,
			WithBranch("main"),
			WithCommit("commit"+string(rune('1'+i)), ""),
		)
		require.NoError(t, recordErr)
		time.Sleep(10 * time.Millisecond)
	}

	// Verify all entries exist
	files, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	require.NoError(t, err)
	assert.Len(t, files, 3)

	// Run cleanup
	err = tracker.Cleanup(ctx)
	require.NoError(t, err)

	// Verify cleanup kept only MaxEntries
	files, err = filepath.Glob(filepath.Join(tempDir, "*.json"))
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestCleanupDisabled(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{
		StoragePath:   tempDir,
		RetentionDays: 1,
		MaxEntries:    1,
		AutoCleanup:   false, // Disabled
	}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Record entries
	coverage := createTestCoverage()
	err = tracker.Record(ctx, coverage)
	require.NoError(t, err)

	err = tracker.Record(ctx, coverage)
	require.NoError(t, err)

	// Run cleanup (should do nothing)
	err = tracker.Cleanup(ctx)
	require.NoError(t, err)

	// Verify both entries still exist
	files, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestGetStatistics(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Record entries with different branches and projects
	coverage := createTestCoverage()

	err = tracker.Record(ctx, coverage,
		WithBranch("main"),
		WithMetadata("project", "project1"),
	)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	err = tracker.Record(ctx, coverage,
		WithBranch("feature"),
		WithMetadata("project", "project2"),
	)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	err = tracker.Record(ctx, coverage,
		WithBranch("main"),
		WithMetadata("project", "project1"),
	)
	require.NoError(t, err)

	// Get statistics
	stats, err := tracker.GetStatistics(ctx)
	require.NoError(t, err)

	assert.Equal(t, 3, stats.TotalEntries)
	assert.Len(t, stats.UniqueProjects, 2)
	assert.Len(t, stats.UniqueBranches, 2)
	assert.Equal(t, 2, stats.UniqueProjects["project1"])
	assert.Equal(t, 1, stats.UniqueProjects["project2"])
	assert.Equal(t, 2, stats.UniqueBranches["main"])
	assert.Equal(t, 1, stats.UniqueBranches["feature"])
	assert.Positive(t, stats.StorageSize)
}

func TestGetStatisticsEmpty(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	stats, err := tracker.GetStatistics(ctx)
	require.NoError(t, err)

	assert.Equal(t, 0, stats.TotalEntries)
	assert.Empty(t, stats.UniqueProjects)
	assert.Empty(t, stats.UniqueBranches)
	assert.Equal(t, int64(0), stats.StorageSize)
}

func TestLegacyAdd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)

	coverage := createTestCoverage()
	err = tracker.Add("main", "commit123", coverage)
	require.NoError(t, err)

	// Verify entry was created
	files, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestLegacyAddInvalidType(t *testing.T) {
	tracker := New()

	err := tracker.Add("main", "commit123", "invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported data type")
}

func TestTrendAnalysis(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Record entries with clear upward trend
	coverages := []float64{60.0, 65.0, 70.0, 75.0, 80.0}
	for i, percentage := range coverages {
		coverage := createTestCoverage()
		coverage.Percentage = percentage

		recordErr := tracker.Record(ctx, coverage,
			WithBranch("main"),
			WithCommit("commit"+string(rune('1'+i)), ""),
		)
		require.NoError(t, recordErr)
		time.Sleep(10 * time.Millisecond)
	}

	trendData, err := tracker.GetTrend(ctx, WithTrendBranch("main"))
	require.NoError(t, err)

	// Verify summary
	assert.Equal(t, 5, trendData.Summary.TotalEntries)
	assert.InDelta(t, 70.0, trendData.Summary.AveragePercentage, 0.001)
	assert.InDelta(t, 60.0, trendData.Summary.MinPercentage, 0.001)
	assert.InDelta(t, 80.0, trendData.Summary.MaxPercentage, 0.001)
	assert.Equal(t, "up", trendData.Summary.CurrentTrend)

	// Verify analysis
	assert.NotNil(t, trendData.Analysis.ShortTermTrend)
	assert.NotNil(t, trendData.Analysis.MediumTermTrend)
	assert.NotNil(t, trendData.Analysis.LongTermTrend)
	assert.GreaterOrEqual(t, trendData.Analysis.Volatility, 0.0)
	assert.NotNil(t, trendData.Analysis.Prediction)

	// Verify basic prediction structure exists
	prediction := trendData.Analysis.Prediction
	assert.NotNil(t, prediction)
}

func TestBuildInfo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	buildInfo := &BuildInfo{
		GoVersion:    "1.21.0",
		Platform:     "linux",
		Architecture: "amd64",
		BuildTime:    "2024-01-01T12:00:00Z",
		BuildNumber:  "123",
		PullRequest:  "456",
		WorkflowID:   "789",
	}

	coverage := createTestCoverage()
	err = tracker.Record(ctx, coverage,
		WithBranch("main"),
		WithCommit("abc123", "https://github.com/test/repo/commit/abc123"),
		WithBuildInfo(buildInfo),
	)
	require.NoError(t, err)

	// Retrieve and verify
	latest, err := tracker.GetLatestEntry(ctx, "main")
	require.NoError(t, err)

	assert.Equal(t, buildInfo.GoVersion, latest.BuildInfo.GoVersion)
	assert.Equal(t, buildInfo.Platform, latest.BuildInfo.Platform)
	assert.Equal(t, buildInfo.BuildNumber, latest.BuildInfo.BuildNumber)
}

func TestPackageStats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	coverage := createTestCoverage()
	err = tracker.Record(ctx, coverage, WithBranch("main"))
	require.NoError(t, err)

	latest, err := tracker.GetLatestEntry(ctx, "main")
	require.NoError(t, err)

	assert.NotNil(t, latest.PackageStats)
	assert.NotEmpty(t, latest.PackageStats)

	for pkgName, stats := range latest.PackageStats {
		assert.NotEmpty(t, pkgName)
		assert.Equal(t, "stable", stats.Trend)
		assert.GreaterOrEqual(t, stats.FileCount, 0)
		assert.False(t, stats.FirstSeen.IsZero())
		assert.False(t, stats.LastModified.IsZero())
	}
}

func TestGetEntryFilename(t *testing.T) {
	tracker := New()

	entry := &Entry{
		Timestamp: time.Date(2024, 1, 15, 14, 30, 45, 123456000, time.UTC),
		Branch:    "feature-branch",
		CommitSHA: "abc123def456",
	}

	filename := tracker.getEntryFilename(entry)
	assert.Equal(t, "20240115-143045.123456-feature-branch-abc123de.json", filename)

	// Test with empty branch (should default to master)
	entry.Branch = ""
	filename = tracker.getEntryFilename(entry)
	assert.Equal(t, "20240115-143045.123456-master-abc123de.json", filename)

	// Test with empty commit SHA (should default to nocommit)
	entry.CommitSHA = ""
	filename = tracker.getEntryFilename(entry)
	assert.Equal(t, "20240115-143045.123456-master-nocommit.json", filename)
}

func TestBranchNameSanitization(t *testing.T) {
	tracker := New()

	testCases := []struct {
		name           string
		branchName     string
		expectedSuffix string // Expected sanitized branch name in filename
	}{
		{
			name:           "branch with forward slash",
			branchName:     "6/merge",
			expectedSuffix: "6-merge",
		},
		{
			name:           "branch with backslash",
			branchName:     "feature\\branch",
			expectedSuffix: "feature-branch",
		},
		{
			name:           "branch with colon",
			branchName:     "feature:branch",
			expectedSuffix: "feature-branch",
		},
		{
			name:           "branch with multiple special characters",
			branchName:     "feature/test*branch?name",
			expectedSuffix: "feature-test-branch-name",
		},
		{
			name:           "branch with Windows forbidden characters",
			branchName:     "feature<branch>name|test",
			expectedSuffix: "feature-branch-name-test",
		},
		{
			name:           "normal branch name unchanged",
			branchName:     "feature-branch",
			expectedSuffix: "feature-branch",
		},
		{
			name:           "master branch unchanged",
			branchName:     "master",
			expectedSuffix: "master",
		},
		{
			name:           "empty branch name defaults to master",
			branchName:     "",
			expectedSuffix: "master",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := &Entry{
				Timestamp: time.Date(2024, 1, 15, 14, 30, 45, 123456000, time.UTC),
				Branch:    tc.branchName,
				CommitSHA: "abc123def456",
			}

			filename := tracker.getEntryFilename(entry)
			expectedFilename := fmt.Sprintf("20240115-143045.123456-%s-abc123de.json", tc.expectedSuffix)
			assert.Equal(t, expectedFilename, filename, "Branch '%s' should be sanitized to '%s'", tc.branchName, tc.expectedSuffix)

			// Verify the filename doesn't contain filesystem-unsafe characters
			assert.NotContains(t, filename, "/", "Filename should not contain forward slash")
			assert.NotContains(t, filename, "\\", "Filename should not contain backslash")
			assert.NotContains(t, filename, ":", "Filename should not contain colon")
			assert.NotContains(t, filename, "*", "Filename should not contain asterisk")
			assert.NotContains(t, filename, "?", "Filename should not contain question mark")
			assert.NotContains(t, filename, "\"", "Filename should not contain quote")
			assert.NotContains(t, filename, "<", "Filename should not contain less than")
			assert.NotContains(t, filename, ">", "Filename should not contain greater than")
			assert.NotContains(t, filename, "|", "Filename should not contain pipe")
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tracker := New()

	testCases := []struct {
		input    string
		expected string
	}{
		{"6/merge", "6-merge"},
		{"feature/branch-name", "feature-branch-name"},
		{"feature\\branch", "feature-branch"},
		{"feature:branch", "feature-branch"},
		{"feature*branch", "feature-branch"},
		{"feature?branch", "feature-branch"},
		{"feature\"branch", "feature-branch"},
		{"feature<branch>", "feature-branch-"},
		{"feature|branch", "feature-branch"},
		{"normal-branch", "normal-branch"},
		{"master", "master"},
		{"", "master"}, // empty should default to master
		{"///", "---"}, // multiple slashes become multiple dashes
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("input_%s", tc.input), func(t *testing.T) {
			result := tracker.sanitizeBranchName(tc.input)
			assert.Equal(t, tc.expected, result, "Input '%s' should be sanitized to '%s'", tc.input, tc.expected)
		})
	}
}

func TestFileHashes(t *testing.T) {
	tracker := New()
	coverage := createTestCoverage()

	hashes := tracker.calculateFileHashes(coverage)
	assert.NotEmpty(t, hashes)

	for fp, hash := range hashes {
		assert.NotEmpty(t, fp)
		assert.NotEmpty(t, hash)
		assert.Contains(t, hash, "hash_")
	}
}

func TestVolatilityCalculation(t *testing.T) {
	tracker := New()

	// Test with empty entries
	entries := []Entry{}
	volatility := tracker.calculateVolatility(entries)
	assert.InDelta(t, 0.0, volatility, 0.001)

	// Test with single entry
	entries = []Entry{
		{Coverage: &parser.CoverageData{Percentage: 75.0}},
	}
	volatility = tracker.calculateVolatility(entries)
	assert.InDelta(t, 0.0, volatility, 0.001)

	// Test with multiple entries
	entries = []Entry{
		{Coverage: &parser.CoverageData{Percentage: 70.0}},
		{Coverage: &parser.CoverageData{Percentage: 80.0}},
		{Coverage: &parser.CoverageData{Percentage: 75.0}},
	}
	volatility = tracker.calculateVolatility(entries)
	assert.Positive(t, volatility)
}

func TestMomentumCalculation(t *testing.T) {
	tracker := New()

	// Test with insufficient entries
	entries := []Entry{
		{Coverage: &parser.CoverageData{Percentage: 75.0}},
		{Coverage: &parser.CoverageData{Percentage: 80.0}},
	}
	momentum := tracker.calculateMomentum(entries)
	assert.InDelta(t, 0.0, momentum, 0.001)

	// Test with sufficient entries showing acceleration
	entries = []Entry{
		{Coverage: &parser.CoverageData{Percentage: 90.0}}, // newest - accelerating upward
		{Coverage: &parser.CoverageData{Percentage: 80.0}}, // middle
		{Coverage: &parser.CoverageData{Percentage: 75.0}}, // oldest
	}
	momentum = tracker.calculateMomentum(entries)
	// Recent change: 90-80 = 10, Historical change: 80-75 = 5, Momentum: 10-5 = 5
	assert.InDelta(t, 5.0, momentum, 0.001)

	// Test with constant change (no acceleration)
	entries = []Entry{
		{Coverage: &parser.CoverageData{Percentage: 85.0}}, // newest
		{Coverage: &parser.CoverageData{Percentage: 80.0}}, // middle
		{Coverage: &parser.CoverageData{Percentage: 75.0}}, // oldest
	}
	momentum = tracker.calculateMomentum(entries)
	// Recent change: 85-80 = 5, Historical change: 80-75 = 5, Momentum: 5-5 = 0
	assert.InDelta(t, 0.0, momentum, 0.001)
}

func TestConfigurationOptions(t *testing.T) {
	// Test record options
	opts := &RecordOptions{}

	WithBranch("test-branch")(opts)
	assert.Equal(t, "test-branch", opts.Branch)

	WithCommit("sha123", "https://example.com")(opts)
	assert.Equal(t, "sha123", opts.CommitSHA)
	assert.Equal(t, "https://example.com", opts.CommitURL)

	WithMetadata("key1", "value1")(opts)
	WithMetadata("key2", "value2")(opts)
	assert.Equal(t, "value1", opts.Metadata["key1"])
	assert.Equal(t, "value2", opts.Metadata["key2"])

	buildInfo := &BuildInfo{GoVersion: "1.21.0"}
	WithBuildInfo(buildInfo)(opts)
	assert.Equal(t, buildInfo, opts.BuildInfo)
}

func TestTrendOptions(t *testing.T) {
	opts := &TrendOptions{}

	WithTrendBranch("feature")(opts)
	assert.Equal(t, "feature", opts.Branch)

	WithTrendDays(7)(opts)
	assert.Equal(t, 7, opts.Days)

	WithMaxDataPoints(50)(opts)
	assert.Equal(t, 50, opts.MaxPoints)
}

// Helper function to create test coverage data
func createTestCoverage() *parser.CoverageData {
	return &parser.CoverageData{
		Mode:         "atomic",
		Percentage:   75.0,
		TotalLines:   100,
		CoveredLines: 75,
		Timestamp:    time.Now(),
		Packages: map[string]*parser.PackageCoverage{
			"main": {
				Name:         "main",
				Percentage:   75.0,
				TotalLines:   100,
				CoveredLines: 75,
				Files: map[string]*parser.FileCoverage{
					"main.go": {
						Path:         "main.go",
						Percentage:   75.0,
						TotalLines:   100,
						CoveredLines: 75,
						Statements: []parser.Statement{
							{StartLine: 1, EndLine: 10, Count: 1},
							{StartLine: 11, EndLine: 20, Count: 0},
						},
					},
				},
			},
		},
	}
}
