package analysis

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewComparisonEngine(t *testing.T) {
	tests := []struct {
		name           string
		config         *ComparisonConfig
		expectDefaults bool
	}{
		{
			name:           "nil config uses defaults",
			config:         nil,
			expectDefaults: true,
		},
		{
			name: "custom config",
			config: &ComparisonConfig{
				SignificantPercentageChange: 2.0,
				SignificantLineChange:       20,
				AnalyzeFileChanges:          false,
				MaxFilesToAnalyze:           25,
				IgnoreTestFiles:             true,
				EnableTrendAnalysis:         false,
				TrendHistoryDays:            14,
				ExcellentCoverageThreshold:  95.0,
				GoodCoverageThreshold:       85.0,
				AcceptableCoverageThreshold: 75.0,
			},
			expectDefaults: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewComparisonEngine(tt.config)
			require.NotNil(t, engine)
			require.NotNil(t, engine.config)

			if tt.expectDefaults {
				require.InDelta(t, 1.0, engine.config.SignificantPercentageChange, 0.001)
				require.Equal(t, 10, engine.config.SignificantLineChange)
				require.True(t, engine.config.AnalyzeFileChanges)
				require.Equal(t, 50, engine.config.MaxFilesToAnalyze)
				require.False(t, engine.config.IgnoreTestFiles)
				require.True(t, engine.config.EnableTrendAnalysis)
				require.Equal(t, 30, engine.config.TrendHistoryDays)
				require.InDelta(t, 90.0, engine.config.ExcellentCoverageThreshold, 0.001)
				require.InDelta(t, 80.0, engine.config.GoodCoverageThreshold, 0.001)
				require.InDelta(t, 70.0, engine.config.AcceptableCoverageThreshold, 0.001)
			} else {
				require.InDelta(t, tt.config.SignificantPercentageChange, engine.config.SignificantPercentageChange, 0.001)
				require.Equal(t, tt.config.SignificantLineChange, engine.config.SignificantLineChange)
				require.Equal(t, tt.config.AnalyzeFileChanges, engine.config.AnalyzeFileChanges)
				require.Equal(t, tt.config.MaxFilesToAnalyze, engine.config.MaxFilesToAnalyze)
				require.Equal(t, tt.config.IgnoreTestFiles, engine.config.IgnoreTestFiles)
				require.Equal(t, tt.config.EnableTrendAnalysis, engine.config.EnableTrendAnalysis)
				require.Equal(t, tt.config.TrendHistoryDays, engine.config.TrendHistoryDays)
				require.InDelta(t, tt.config.ExcellentCoverageThreshold, engine.config.ExcellentCoverageThreshold, 0.001)
				require.InDelta(t, tt.config.GoodCoverageThreshold, engine.config.GoodCoverageThreshold, 0.001)
				require.InDelta(t, tt.config.AcceptableCoverageThreshold, engine.config.AcceptableCoverageThreshold, 0.001)
			}
		})
	}
}

func TestCompareCoverage(t *testing.T) {
	engine := NewComparisonEngine(nil)

	baseSnapshot := &CoverageSnapshot{
		Branch:    "master",
		CommitSHA: "abc123",
		Timestamp: time.Now().Add(-time.Hour),
		OverallCoverage: CoverageMetrics{
			Percentage:        80.0,
			TotalStatements:   1000,
			CoveredStatements: 800,
			TotalLines:        1200,
			CoveredLines:      960,
			TotalFunctions:    100,
			CoveredFunctions:  80,
		},
		FileCoverage: map[string]FileMetrics{
			"main.go": {
				Filename:          "main.go",
				Package:           "master",
				Percentage:        75.0,
				TotalStatements:   100,
				CoveredStatements: 75,
				UncoveredLines:    []int{10, 15, 20},
				Functions:         []string{"master", "init"},
				IsTestFile:        false,
			},
			"helper.go": {
				Filename:          "helper.go",
				Package:           "master",
				Percentage:        90.0,
				TotalStatements:   50,
				CoveredStatements: 45,
				UncoveredLines:    []int{25},
				Functions:         []string{"helper1", "helper2"},
				IsTestFile:        false,
			},
		},
		PackageCoverage: map[string]PackageMetrics{
			"master": {
				Package:           "master",
				Percentage:        80.0,
				TotalStatements:   150,
				CoveredStatements: 120,
				FileCount:         2,
			},
		},
		TestMetadata: TestMetadata{
			TestDuration:   30 * time.Second,
			TestCount:      50,
			FailedTests:    0,
			SkippedTests:   2,
			BenchmarkCount: 5,
		},
	}

	prSnapshot := &CoverageSnapshot{
		Branch:    "feature-branch",
		CommitSHA: "def456",
		Timestamp: time.Now(),
		OverallCoverage: CoverageMetrics{
			Percentage:        85.0,
			TotalStatements:   1100,
			CoveredStatements: 935,
			TotalLines:        1320,
			CoveredLines:      1122,
			TotalFunctions:    110,
			CoveredFunctions:  94,
		},
		FileCoverage: map[string]FileMetrics{
			"main.go": {
				Filename:          "main.go",
				Package:           "master",
				Percentage:        80.0,
				TotalStatements:   110,
				CoveredStatements: 88,
				UncoveredLines:    []int{10, 15},
				Functions:         []string{"master", "init", "newFunc"},
				IsTestFile:        false,
				LinesAdded:        10,
				LinesRemoved:      0,
				IsModified:        true,
			},
			"helper.go": {
				Filename:          "helper.go",
				Package:           "master",
				Percentage:        90.0,
				TotalStatements:   50,
				CoveredStatements: 45,
				UncoveredLines:    []int{25},
				Functions:         []string{"helper1", "helper2"},
				IsTestFile:        false,
			},
			"new_file.go": {
				Filename:          "new_file.go",
				Package:           "master",
				Percentage:        70.0,
				TotalStatements:   30,
				CoveredStatements: 21,
				UncoveredLines:    []int{5, 10, 15, 20, 25, 30, 35, 40, 45},
				Functions:         []string{"newFunction"},
				IsTestFile:        false,
				LinesAdded:        30,
				IsNewFile:         true,
			},
		},
		PackageCoverage: map[string]PackageMetrics{
			"master": {
				Package:           "master",
				Percentage:        82.0,
				TotalStatements:   190,
				CoveredStatements: 156,
				FileCount:         3,
			},
		},
		TestMetadata: TestMetadata{
			TestDuration:   35 * time.Second,
			TestCount:      55,
			FailedTests:    0,
			SkippedTests:   1,
			BenchmarkCount: 6,
		},
	}

	result, err := engine.CompareCoverage(context.Background(), baseSnapshot, prSnapshot)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check basic structure
	require.Equal(t, *baseSnapshot, result.BaseSnapshot)
	require.Equal(t, *prSnapshot, result.PRSnapshot)

	// Check overall change analysis
	require.InDelta(t, 5.0, result.OverallChange.PercentageChange, 0.001)
	require.Equal(t, DirectionImproved, result.OverallChange.Direction)
	require.True(t, result.OverallChange.IsSignificant)

	// Check file changes are analyzed
	require.NotEmpty(t, result.FileChanges)

	// Check package changes are analyzed
	require.NotEmpty(t, result.PackageChanges)

	// Check trend analysis is generated (since EnableTrendAnalysis is true by default)
	require.NotEmpty(t, result.TrendAnalysis.Direction)

	// Check quality assessment is generated
	require.NotEmpty(t, result.QualityAssessment.OverallGrade)

	// Check that result has recommendations field (can be empty or nil)

	// Check summary is generated
	require.NotEmpty(t, result.Summary.OverallImpact)
}

func TestAnalyzeOverallChange(t *testing.T) {
	engine := NewComparisonEngine(nil)

	tests := []struct {
		name                string
		basePercentage      float64
		prPercentage        float64
		expectedDirection   string
		expectedSignificant bool
	}{
		{
			name:                "significant improvement",
			basePercentage:      70.0,
			prPercentage:        75.0,
			expectedDirection:   DirectionImproved,
			expectedSignificant: true,
		},
		{
			name:                "significant degradation",
			basePercentage:      80.0,
			prPercentage:        75.0,
			expectedDirection:   DirectionDegraded,
			expectedSignificant: true,
		},
		{
			name:                "stable coverage",
			basePercentage:      80.0,
			prPercentage:        80.05,
			expectedDirection:   DirectionStable,
			expectedSignificant: false,
		},
		{
			name:                "minor improvement",
			basePercentage:      80.0,
			prPercentage:        80.5,
			expectedDirection:   DirectionImproved,
			expectedSignificant: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseSnapshot := &CoverageSnapshot{
				OverallCoverage: CoverageMetrics{
					Percentage:        tt.basePercentage,
					TotalStatements:   1000,
					CoveredStatements: int(tt.basePercentage * 10),
				},
			}

			prSnapshot := &CoverageSnapshot{
				OverallCoverage: CoverageMetrics{
					Percentage:        tt.prPercentage,
					TotalStatements:   1000,
					CoveredStatements: int(tt.prPercentage * 10),
				},
			}

			result := engine.analyzeOverallChange(baseSnapshot, prSnapshot)

			require.Equal(t, tt.expectedDirection, result.Direction)
			require.Equal(t, tt.expectedSignificant, result.IsSignificant)
			require.InDelta(t, tt.prPercentage-tt.basePercentage, result.PercentageChange, 0.001)
		})
	}
}

func TestAnalyzeFileChanges(t *testing.T) {
	config := &ComparisonConfig{
		SignificantPercentageChange: 1.0,
		SignificantLineChange:       10,
		AnalyzeFileChanges:          true,
		MaxFilesToAnalyze:           50,
		IgnoreTestFiles:             false,
	}
	engine := NewComparisonEngine(config)

	baseSnapshot := &CoverageSnapshot{
		FileCoverage: map[string]FileMetrics{
			"existing.go": {
				Filename:          "existing.go",
				Percentage:        80.0,
				TotalStatements:   100,
				CoveredStatements: 80,
			},
			"deleted.go": {
				Filename:          "deleted.go",
				Percentage:        70.0,
				TotalStatements:   50,
				CoveredStatements: 35,
			},
			"test_file_test.go": {
				Filename:          "test_file_test.go",
				Percentage:        90.0,
				TotalStatements:   20,
				CoveredStatements: 18,
				IsTestFile:        true,
			},
		},
	}

	prSnapshot := &CoverageSnapshot{
		FileCoverage: map[string]FileMetrics{
			"existing.go": {
				Filename:          "existing.go",
				Percentage:        85.0,
				TotalStatements:   110,
				CoveredStatements: 94,
				LinesAdded:        10,
				LinesRemoved:      0,
				IsModified:        true,
			},
			"new_file.go": {
				Filename:          "new_file.go",
				Percentage:        75.0,
				TotalStatements:   40,
				CoveredStatements: 30,
				LinesAdded:        40,
				IsNewFile:         true,
			},
			"test_file_test.go": {
				Filename:          "test_file_test.go",
				Percentage:        90.0,
				TotalStatements:   20,
				CoveredStatements: 18,
				IsTestFile:        true,
			},
		},
	}

	changes := engine.analyzeFileChanges(baseSnapshot, prSnapshot)

	// Should analyze all files including test files
	require.Len(t, changes, 4)

	// Find specific changes
	var existingChange, newFileChange, deletedChange *FileChangeAnalysis
	for i := range changes {
		switch changes[i].Filename {
		case "existing.go":
			existingChange = &changes[i]
		case "new_file.go":
			newFileChange = &changes[i]
		case "deleted.go":
			deletedChange = &changes[i]
		}
	}

	// Test existing file change
	require.NotNil(t, existingChange)
	require.InDelta(t, 5.0, existingChange.PercentageChange, 0.001)
	require.Equal(t, DirectionImproved, existingChange.Direction)
	require.True(t, existingChange.IsSignificant)
	require.False(t, existingChange.IsNewFile)
	require.False(t, existingChange.IsDeleted)

	// Test new file
	require.NotNil(t, newFileChange)
	require.True(t, newFileChange.IsNewFile)
	require.InDelta(t, 75.0, newFileChange.PercentageChange, 0.001)
	require.Equal(t, "new", newFileChange.Direction)

	// Test deleted file
	require.NotNil(t, deletedChange)
	require.True(t, deletedChange.IsDeleted)
	require.InDelta(t, -70.0, deletedChange.PercentageChange, 0.001)
	require.Equal(t, "deleted", deletedChange.Direction)
}

func TestAnalyzeFileChangesIgnoreTestFiles(t *testing.T) {
	config := &ComparisonConfig{
		IgnoreTestFiles:             true,
		SignificantPercentageChange: 1.0,
		SignificantLineChange:       10,
		MaxFilesToAnalyze:           50,
	}
	engine := NewComparisonEngine(config)

	baseSnapshot := &CoverageSnapshot{
		FileCoverage: map[string]FileMetrics{
			"main.go": {
				Filename:          "main.go",
				Percentage:        80.0,
				TotalStatements:   100,
				CoveredStatements: 80,
				IsTestFile:        false,
			},
			"main_test.go": {
				Filename:          "main_test.go",
				Percentage:        90.0,
				TotalStatements:   50,
				CoveredStatements: 45,
				IsTestFile:        true,
			},
		},
	}

	prSnapshot := &CoverageSnapshot{
		FileCoverage: map[string]FileMetrics{
			"main.go": {
				Filename:          "main.go",
				Percentage:        85.0, // 5% increase = significant change
				TotalStatements:   100,
				CoveredStatements: 85,
				IsTestFile:        false,
			},
			"main_test.go": {
				Filename:          "main_test.go",
				Percentage:        95.0, // 5% increase = significant change
				TotalStatements:   50,
				CoveredStatements: 48,
				IsTestFile:        true,
			},
		},
	}

	changes := engine.analyzeFileChanges(baseSnapshot, prSnapshot)

	// Should only analyze non-test files (test files should be ignored)
	require.Len(t, changes, 1)
	require.Equal(t, "main.go", changes[0].Filename)
}

func TestAnalyzePackageChanges(t *testing.T) {
	engine := NewComparisonEngine(nil)

	baseSnapshot := &CoverageSnapshot{
		PackageCoverage: map[string]PackageMetrics{
			"master": {
				Package:    "master",
				Percentage: 80.0,
				FileCount:  2,
			},
			"utils": {
				Package:    "utils",
				Percentage: 75.0,
				FileCount:  3,
			},
			"removed": {
				Package:    "removed",
				Percentage: 60.0,
				FileCount:  1,
			},
		},
	}

	prSnapshot := &CoverageSnapshot{
		PackageCoverage: map[string]PackageMetrics{
			"master": {
				Package:    "master",
				Percentage: 85.0,
				FileCount:  2,
			},
			"utils": {
				Package:    "utils",
				Percentage: 70.0,
				FileCount:  3,
			},
			"newpkg": {
				Package:    "newpkg",
				Percentage: 90.0,
				FileCount:  1,
			},
		},
	}

	changes := engine.analyzePackageChanges(baseSnapshot, prSnapshot)

	// Should only analyze packages that exist in both snapshots
	require.Len(t, changes, 2)

	// Find specific changes
	var mainChange, utilsChange *PackageChangeAnalysis
	for i := range changes {
		switch changes[i].Package {
		case "master":
			mainChange = &changes[i]
		case "utils":
			utilsChange = &changes[i]
		}
	}

	// Test main package improvement
	require.NotNil(t, mainChange)
	require.InDelta(t, 5.0, mainChange.PercentageChange, 0.001)
	require.Equal(t, DirectionImproved, mainChange.Direction)
	require.True(t, mainChange.IsSignificant)

	// Test utils package degradation
	require.NotNil(t, utilsChange)
	require.InDelta(t, -5.0, utilsChange.PercentageChange, 0.001)
	require.Equal(t, DirectionDegraded, utilsChange.Direction)
	require.True(t, utilsChange.IsSignificant)
}

func TestAnalyzeTrends(t *testing.T) {
	engine := NewComparisonEngine(nil)

	baseSnapshot := &CoverageSnapshot{
		Timestamp: time.Now().Add(-time.Hour),
		OverallCoverage: CoverageMetrics{
			Percentage: 75.0,
		},
	}

	prSnapshot := &CoverageSnapshot{
		Timestamp: time.Now(),
		OverallCoverage: CoverageMetrics{
			Percentage: 80.0,
		},
	}

	trends := engine.analyzeTrends(baseSnapshot, prSnapshot)

	require.Equal(t, "upward", trends.Direction)
	require.Equal(t, "accelerating", trends.Momentum)
	require.InDelta(t, 5.0, trends.Volatility, 0.001)
	require.Equal(t, 2, trends.HistoricalContext.DataPoints)
	require.InDelta(t, 82.5, trends.Prediction.NextCoverage, 0.001)
}

func TestCalculateMagnitude(t *testing.T) {
	engine := NewComparisonEngine(nil)

	tests := []struct {
		name     string
		change   float64
		expected string
	}{
		{"significant change", 10.0, "significant"},
		{"moderate change", 3.0, "moderate"},
		{"minor change", 1.0, "minor"},
		{"negligible change", 0.1, "negligible"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.calculateMagnitude(tt.change)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateRisk(t *testing.T) {
	engine := NewComparisonEngine(nil)

	tests := []struct {
		name     string
		change   FileChangeAnalysis
		expected string
	}{
		{
			name: "high risk - low coverage and degraded",
			change: FileChangeAnalysis{
				PRPercentage: 40.0,
				Direction:    DirectionDegraded,
			},
			expected: "high",
		},
		{
			name: "medium risk - low coverage",
			change: FileChangeAnalysis{
				PRPercentage: 60.0,
				Direction:    DirectionStable,
			},
			expected: "medium",
		},
		{
			name: "medium risk - large change",
			change: FileChangeAnalysis{
				PRPercentage:     85.0,
				PercentageChange: -15.0,
			},
			expected: "medium",
		},
		{
			name: "low risk - good coverage and stable",
			change: FileChangeAnalysis{
				PRPercentage:     85.0,
				PercentageChange: 2.0,
				Direction:        DirectionImproved,
			},
			expected: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.calculateRisk(tt.change)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateCoverageGrade(t *testing.T) {
	engine := NewComparisonEngine(nil)

	tests := []struct {
		coverage float64
		expected string
	}{
		{96.0, "A+"},
		{92.0, "A"},
		{87.0, "B+"},
		{82.0, "B"},
		{75.0, "C"},
		{65.0, "D"},
		{45.0, "F"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := engine.calculateCoverageGrade(tt.coverage)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateTrendGrade(t *testing.T) {
	engine := NewComparisonEngine(nil)

	tests := []struct {
		name     string
		change   OverallChangeAnalysis
		expected string
	}{
		{
			name: "significant improvement",
			change: OverallChangeAnalysis{
				Direction:     DirectionImproved,
				IsSignificant: true,
			},
			expected: "A",
		},
		{
			name: "minor improvement",
			change: OverallChangeAnalysis{
				Direction:     DirectionImproved,
				IsSignificant: false,
			},
			expected: "B",
		},
		{
			name: "stable",
			change: OverallChangeAnalysis{
				Direction: DirectionStable,
			},
			expected: "B",
		},
		{
			name: "minor degradation",
			change: OverallChangeAnalysis{
				Direction:     DirectionDegraded,
				IsSignificant: false,
			},
			expected: "C",
		},
		{
			name: "significant degradation",
			change: OverallChangeAnalysis{
				Direction:     DirectionDegraded,
				IsSignificant: true,
			},
			expected: "D",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.calculateTrendGrade(&tt.change)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateRiskLevel(t *testing.T) {
	engine := NewComparisonEngine(nil)

	tests := []struct {
		name        string
		coverage    float64
		change      OverallChangeAnalysis
		fileChanges []FileChangeAnalysis
		expected    string
	}{
		{
			name:     "critical risk - very low coverage",
			coverage: 50.0,
			change: OverallChangeAnalysis{
				Direction:     DirectionDegraded,
				IsSignificant: true,
			},
			fileChanges: []FileChangeAnalysis{
				{Risk: "high"},
				{Risk: "high"},
				{Risk: "high"},
				{Risk: "high"},
				{Risk: "high"},
				{Risk: "high"},
			},
			expected: "critical",
		},
		{
			name:     "high risk - low coverage and degradation",
			coverage: 65.0,
			change: OverallChangeAnalysis{
				Direction:     DirectionDegraded,
				IsSignificant: true,
			},
			fileChanges: []FileChangeAnalysis{
				{Risk: "high"},
				{Risk: "high"},
			},
			expected: "high",
		},
		{
			name:     "medium risk - acceptable coverage",
			coverage: 75.0,
			change: OverallChangeAnalysis{
				Direction: DirectionStable,
			},
			fileChanges: []FileChangeAnalysis{
				{Risk: "medium"},
			},
			expected: "medium",
		},
		{
			name:     "low risk - good coverage and stable",
			coverage: 85.0,
			change: OverallChangeAnalysis{
				Direction: DirectionImproved,
			},
			fileChanges: []FileChangeAnalysis{
				{Risk: "low"},
			},
			expected: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.calculateRiskLevel(tt.coverage, &tt.change, tt.fileChanges)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateQualityScore(t *testing.T) {
	engine := NewComparisonEngine(nil)

	tests := []struct {
		name     string
		coverage float64
		change   OverallChangeAnalysis
		expected float64
	}{
		{
			name:     "high coverage with improvement",
			coverage: 90.0,
			change: OverallChangeAnalysis{
				Direction:     DirectionImproved,
				IsSignificant: true,
			},
			expected: 95.0,
		},
		{
			name:     "high coverage with degradation",
			coverage: 90.0,
			change: OverallChangeAnalysis{
				Direction:     DirectionDegraded,
				IsSignificant: true,
			},
			expected: 80.0,
		},
		{
			name:     "coverage at 100 should cap at 100",
			coverage: 100.0,
			change: OverallChangeAnalysis{
				Direction:     DirectionImproved,
				IsSignificant: true,
			},
			expected: 100.0,
		},
		{
			name:     "negative score should floor at 0",
			coverage: 5.0,
			change: OverallChangeAnalysis{
				Direction:     DirectionDegraded,
				IsSignificant: true,
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.calculateQualityScore(tt.coverage, &tt.change)
			require.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestIsTestFile(t *testing.T) {
	engine := NewComparisonEngine(nil)

	tests := []struct {
		filename string
		expected bool
	}{
		{"main.go", false},
		{"main_test.go", true},
		{"pkg/test/helper.go", true},
		{"pkg/tests/integration.go", true},
		{"normal_file.go", false},
		{"test_helper.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := engine.isTestFile(tt.filename)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadCoverageSnapshot(t *testing.T) {
	engine := NewComparisonEngine(nil)

	// Create test snapshot
	snapshot := &CoverageSnapshot{
		Branch:    "test-branch",
		CommitSHA: "abc123",
		Timestamp: time.Now(),
		OverallCoverage: CoverageMetrics{
			Percentage:        85.0,
			TotalStatements:   1000,
			CoveredStatements: 850,
		},
	}

	// Create temp file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "coverage.json")

	data, err := json.MarshalIndent(snapshot, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(filePath, data, 0o600)
	require.NoError(t, err)

	// Test loading
	loaded, err := engine.LoadCoverageSnapshot(context.Background(), filePath)
	require.NoError(t, err)
	require.Equal(t, snapshot.Branch, loaded.Branch)
	require.Equal(t, snapshot.CommitSHA, loaded.CommitSHA)
	require.InDelta(t, snapshot.OverallCoverage.Percentage, loaded.OverallCoverage.Percentage, 0.001)

	// Test loading non-existent file
	_, err = engine.LoadCoverageSnapshot(context.Background(), "/nonexistent/file.json")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to open coverage file")

	// Test loading invalid JSON
	invalidPath := filepath.Join(tempDir, "invalid.json")
	err = os.WriteFile(invalidPath, []byte("invalid json"), 0o600)
	require.NoError(t, err)

	_, err = engine.LoadCoverageSnapshot(context.Background(), invalidPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse coverage snapshot")
}

func TestSaveComparisonResult(t *testing.T) {
	engine := NewComparisonEngine(nil)

	result := &ComparisonResult{
		BaseSnapshot: CoverageSnapshot{
			Branch: "master",
			OverallCoverage: CoverageMetrics{
				Percentage: 80.0,
			},
		},
		PRSnapshot: CoverageSnapshot{
			Branch: "feature",
			OverallCoverage: CoverageMetrics{
				Percentage: 85.0,
			},
		},
		OverallChange: OverallChangeAnalysis{
			PercentageChange: 5.0,
			Direction:        DirectionImproved,
		},
	}

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "subdir", "result.json")

	// Test saving (should create directory)
	err := engine.SaveComparisonResult(context.Background(), result, filePath)
	require.NoError(t, err)

	// Verify file exists and contains correct data
	data, err := os.ReadFile(filePath) //nolint:gosec // Test file path is safe
	require.NoError(t, err)

	var loaded ComparisonResult
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)
	require.Equal(t, result.BaseSnapshot.Branch, loaded.BaseSnapshot.Branch)
	require.Equal(t, result.PRSnapshot.Branch, loaded.PRSnapshot.Branch)
	require.InDelta(t, result.OverallChange.PercentageChange, loaded.OverallChange.PercentageChange, 0.001)
}

func TestGenerateQualityAssessment(t *testing.T) {
	engine := NewComparisonEngine(nil)

	prSnapshot := &CoverageSnapshot{
		OverallCoverage: CoverageMetrics{
			Percentage: 85.0,
		},
	}

	overallChange := &OverallChangeAnalysis{
		Direction:     DirectionImproved,
		IsSignificant: true,
	}

	fileChanges := []FileChangeAnalysis{
		{PRPercentage: 90.0, Risk: "low"},
		{PRPercentage: 80.0, Risk: "medium"},
		{PRPercentage: 40.0, Risk: "high"},
	}

	assessment := engine.generateQualityAssessment(prSnapshot, overallChange, fileChanges)

	require.Equal(t, "B+", assessment.CoverageGrade)
	require.Equal(t, "A", assessment.TrendGrade)
	require.Equal(t, "B+", assessment.OverallGrade)
	require.Equal(t, "low", assessment.RiskLevel)
	require.InDelta(t, 90.0, assessment.QualityScore, 0.001)
	require.NotEmpty(t, assessment.Strengths)
	require.NotEmpty(t, assessment.Weaknesses)
}

func TestGenerateRecommendations(t *testing.T) {
	config := &ComparisonConfig{
		AcceptableCoverageThreshold: 70.0,
	}
	engine := NewComparisonEngine(config)

	result := &ComparisonResult{
		PRSnapshot: CoverageSnapshot{
			OverallCoverage: CoverageMetrics{
				Percentage: 60.0, // Below threshold
			},
		},
		OverallChange: OverallChangeAnalysis{
			Direction:     DirectionDegraded,
			IsSignificant: true,
		},
		FileChanges: []FileChangeAnalysis{
			{Risk: "high"},
			{Risk: "high"},
			{Risk: "medium"},
		},
	}

	recommendations := engine.generateRecommendations(result)

	require.NotEmpty(t, recommendations)

	// Should have coverage recommendation due to low coverage
	var coverageRec *Recommendation
	for i := range recommendations {
		if recommendations[i].Type == "coverage" {
			coverageRec = &recommendations[i]
			break
		}
	}
	require.NotNil(t, coverageRec)
	require.Equal(t, "high", coverageRec.Priority)

	// Should have testing recommendation due to high-risk files
	var testingRec *Recommendation
	for i := range recommendations {
		if recommendations[i].Type == "testing" {
			testingRec = &recommendations[i]
			break
		}
	}
	require.NotNil(t, testingRec)

	// Should have process recommendation due to significant degradation
	var processRec *Recommendation
	for i := range recommendations {
		if recommendations[i].Type == "process" {
			processRec = &recommendations[i]
			break
		}
	}
	require.NotNil(t, processRec)
	require.Equal(t, "high", processRec.Priority)
}

func TestGenerateSummary(t *testing.T) {
	config := &ComparisonConfig{
		AcceptableCoverageThreshold: 70.0,
		ExcellentCoverageThreshold:  90.0,
	}
	engine := NewComparisonEngine(config)

	result := &ComparisonResult{
		PRSnapshot: CoverageSnapshot{
			OverallCoverage: CoverageMetrics{
				Percentage: 85.0,
			},
		},
		OverallChange: OverallChangeAnalysis{
			PercentageChange: 5.0,
			Direction:        DirectionImproved,
			IsSignificant:    true,
		},
		FileChanges: []FileChangeAnalysis{
			{IsSignificant: true, PercentageChange: 10.0},
			{IsSignificant: true, PercentageChange: -5.0},
			{IsSignificant: false, PercentageChange: 1.0},
		},
		Recommendations: []Recommendation{
			{Priority: "high", Title: "Fix Critical Issue"},
			{Priority: "medium", Title: "Minor Improvement"},
		},
	}

	summary := engine.generateSummary(result)

	require.Equal(t, "positive", summary.OverallImpact)
	require.Contains(t, summary.KeyChanges[0], "improved by 5.0%")
	require.Contains(t, summary.KeyChanges[1], "2 files with significant coverage changes")
	require.Contains(t, summary.Highlights[0], "Significant coverage improvement achieved")
	require.Contains(t, summary.NextSteps[0], "Fix Critical Issue")
}
