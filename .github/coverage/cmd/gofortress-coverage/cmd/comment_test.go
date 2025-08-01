package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/coverage/internal/analysis"
	"github.com/mrz1836/go-broadcast/coverage/internal/badge"
	"github.com/mrz1836/go-broadcast/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/coverage/internal/github"
	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
	"github.com/mrz1836/go-broadcast/coverage/internal/templates"
)

func TestConvertToSnapshot(t *testing.T) {
	coverage := &parser.CoverageData{
		Percentage:   85.5,
		TotalLines:   1000,
		CoveredLines: 855,
		Mode:         "atomic",
		Packages:     make(map[string]*parser.PackageCoverage),
	}

	branch := "feature/test"
	commitSHA := "abc123def456"

	snapshot := convertToSnapshot(coverage, branch, commitSHA)

	require.NotNil(t, snapshot)
	require.Equal(t, branch, snapshot.Branch)
	require.Equal(t, commitSHA, snapshot.CommitSHA)
	require.InDelta(t, coverage.Percentage, snapshot.OverallCoverage.Percentage, 0.001)
	require.Equal(t, coverage.TotalLines, snapshot.OverallCoverage.TotalStatements)
	require.Equal(t, coverage.CoveredLines, snapshot.OverallCoverage.CoveredStatements)
	require.Equal(t, coverage.TotalLines, snapshot.OverallCoverage.TotalLines)
	require.Equal(t, coverage.CoveredLines, snapshot.OverallCoverage.CoveredLines)
	require.NotNil(t, snapshot.FileCoverage)
	require.NotNil(t, snapshot.PackageCoverage)
	require.WithinDuration(t, time.Now(), snapshot.Timestamp, time.Second)
}

func TestConvertTrendData(t *testing.T) {
	tests := []struct {
		name     string
		trend    analysis.TrendAnalysis
		expected github.TrendData
	}{
		{
			name: "positive trend with momentum",
			trend: analysis.TrendAnalysis{
				Direction: "up",
				Momentum:  "increasing",
			},
			expected: github.TrendData{
				Direction:        "up",
				Magnitude:        "minor",
				PercentageChange: 0,
				Momentum:         "increasing",
			},
		},
		{
			name: "negative trend",
			trend: analysis.TrendAnalysis{
				Direction: "down",
				Momentum:  "decreasing",
			},
			expected: github.TrendData{
				Direction:        "down",
				Magnitude:        "minor",
				PercentageChange: 0,
				Momentum:         "decreasing",
			},
		},
		{
			name: "stable trend",
			trend: analysis.TrendAnalysis{
				Direction: "stable",
				Momentum:  "stable",
			},
			expected: github.TrendData{
				Direction:        "stable",
				Magnitude:        "minor",
				PercentageChange: 0,
				Momentum:         "stable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTrendData(tt.trend)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertFileChanges(t *testing.T) {
	changes := []analysis.FileChangeAnalysis{
		{
			Filename:         "main.go",
			BasePercentage:   80.0,
			PRPercentage:     85.0,
			PercentageChange: 5.0,
			LinesAdded:       10,
			LinesRemoved:     2,
			IsSignificant:    true,
		},
		{
			Filename:         "helper.go",
			BasePercentage:   75.0,
			PRPercentage:     74.0,
			PercentageChange: -1.0,
			LinesAdded:       3,
			LinesRemoved:     5,
			IsSignificant:    false,
		},
	}

	result := convertFileChanges(changes)

	require.Len(t, result, 2)
	require.Equal(t, "main.go", result[0].Filename)
	require.InDelta(t, 80.0, result[0].BaseCoverage, 0.001)
	require.InDelta(t, 85.0, result[0].PRCoverage, 0.001)
	require.InDelta(t, 5.0, result[0].Difference, 0.001)
	require.Equal(t, 10, result[0].LinesAdded)
	require.Equal(t, 2, result[0].LinesRemoved)
	require.True(t, result[0].IsSignificant)

	require.Equal(t, "helper.go", result[1].Filename)
	require.InDelta(t, 75.0, result[1].BaseCoverage, 0.001)
	require.InDelta(t, 74.0, result[1].PRCoverage, 0.001)
	require.InDelta(t, -1.0, result[1].Difference, 0.001)
	require.Equal(t, 3, result[1].LinesAdded)
	require.Equal(t, 5, result[1].LinesRemoved)
	require.False(t, result[1].IsSignificant)
}

func TestConvertFileChangesEmpty(t *testing.T) {
	changes := []analysis.FileChangeAnalysis{}
	result := convertFileChanges(changes)
	require.Empty(t, result)
}

func TestExtractSignificantFiles(t *testing.T) {
	changes := []analysis.FileChangeAnalysis{
		{
			Filename:      "main.go",
			IsSignificant: true,
		},
		{
			Filename:      "helper.go",
			IsSignificant: false,
		},
		{
			Filename:      "important.go",
			IsSignificant: true,
		},
	}

	result := extractSignificantFiles(changes)

	require.Len(t, result, 2)
	require.Contains(t, result, "main.go")
	require.Contains(t, result, "important.go")
	require.NotContains(t, result, "helper.go")
}

func TestExtractSignificantFilesEmpty(t *testing.T) {
	changes := []analysis.FileChangeAnalysis{
		{
			Filename:      "helper.go",
			IsSignificant: false,
		},
	}

	result := extractSignificantFiles(changes)
	require.Empty(t, result)
}

func TestBuildTemplateData(t *testing.T) {
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Owner:      "testowner",
			Repository: "testrepo",
			CommitSHA:  "abc123def456",
		},
	}

	prNumber := 123
	badgeURL := "https://example.com/badge.svg"
	reportURL := "https://example.com/report"

	comparison := &github.CoverageComparison{
		BaseCoverage: github.CoverageData{
			Percentage:        80.0,
			TotalStatements:   1000,
			CoveredStatements: 800,
		},
		PRCoverage: github.CoverageData{
			Percentage:        85.0,
			TotalStatements:   1000,
			CoveredStatements: 850,
		},
		Difference: 5.0,
		TrendAnalysis: github.TrendData{
			Direction: "up",
			Magnitude: "minor",
			Momentum:  "increasing",
		},
		PRFileAnalysis: &github.PRFileAnalysis{
			Summary: github.PRFileSummary{
				TotalFiles:     5,
				GoFilesCount:   3,
				TestFilesCount: 1,
				HasGoChanges:   true,
			},
		},
	}

	coverageData := &parser.CoverageData{
		Percentage: 85.0,
	}

	result := buildTemplateData(cfg, prNumber, comparison, coverageData, badgeURL, reportURL)

	require.NotNil(t, result)
	require.Equal(t, "testowner", result.Repository.Owner)
	require.Equal(t, "testrepo", result.Repository.Name)
	require.Equal(t, "main", result.Repository.DefaultBranch)
	require.Equal(t, "https://github.com/testowner/testrepo", result.Repository.URL)

	require.Equal(t, prNumber, result.PullRequest.Number)
	require.Equal(t, "current", result.PullRequest.Branch)
	require.Equal(t, "main", result.PullRequest.BaseBranch)
	require.Equal(t, cfg.GitHub.CommitSHA, result.PullRequest.CommitSHA)
	require.Equal(t, "https://github.com/testowner/testrepo/pull/123", result.PullRequest.URL)

	require.InDelta(t, 85.0, result.Coverage.Overall.Percentage, 0.001)
	require.Equal(t, 1000, result.Coverage.Overall.TotalStatements)
	require.Equal(t, 850, result.Coverage.Overall.CoveredStatements)
	require.Equal(t, "B+", result.Coverage.Overall.Grade)
	require.Equal(t, "good", result.Coverage.Overall.Status)

	require.InDelta(t, 80.0, result.Comparison.BasePercentage, 0.001)
	require.InDelta(t, 85.0, result.Comparison.CurrentPercentage, 0.001)
	require.InDelta(t, 5.0, result.Comparison.Change, 0.001)
	require.Equal(t, "up", result.Comparison.Direction)
	require.True(t, result.Comparison.IsSignificant)

	require.Equal(t, "B+", result.Quality.OverallGrade)
	require.Equal(t, "B+", result.Quality.CoverageGrade)
	require.Equal(t, "A", result.Quality.TrendGrade)
	require.Equal(t, "low", result.Quality.RiskLevel)
	require.InDelta(t, 85.0, result.Quality.Score, 0.001)

	require.Equal(t, badgeURL, result.Resources.BadgeURL)
	require.Equal(t, reportURL, result.Resources.ReportURL)
	require.Equal(t, "https://testowner.github.io/testrepo/coverage/", result.Resources.DashboardURL)
	require.Equal(t, "https://testowner.github.io/testrepo/coverage/pr/123/badge.svg", result.Resources.PRBadgeURL)
	require.Equal(t, "https://testowner.github.io/testrepo/coverage/pr/123/", result.Resources.PRReportURL)
	require.Equal(t, "https://testowner.github.io/testrepo/coverage/trends/", result.Resources.HistoricalURL)

	require.WithinDuration(t, time.Now(), result.Timestamp, time.Second)
}

func TestCalculateQualityGrade(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		expected   string
	}{
		{"A+ grade", 96.0, "A+"},
		{"A+ grade boundary", 95.0, "A+"},
		{"A grade", 92.0, "A"},
		{"A grade boundary", 90.0, "A"},
		{"B+ grade", 87.0, "B+"},
		{"B+ grade boundary", 85.0, "B+"},
		{"B grade", 82.0, "B"},
		{"B grade boundary", 80.0, "B"},
		{"C grade", 75.0, "C"},
		{"C grade boundary", 70.0, "C"},
		{"D grade", 65.0, "D"},
		{"D grade boundary", 60.0, "D"},
		{"F grade", 55.0, "F"},
		{"F grade low", 10.0, "F"},
		{"F grade zero", 0.0, "F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateQualityGrade(tt.percentage)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateCoverageStatus(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		expected   string
	}{
		{"excellent status", 95.0, "excellent"},
		{"excellent boundary", 90.0, "excellent"},
		{"good status", 85.0, "good"},
		{"good boundary", 80.0, "good"},
		{"warning status", 75.0, "warning"},
		{"warning boundary", 70.0, "warning"},
		{"critical status", 65.0, "critical"},
		{"critical low", 30.0, "critical"},
		{"critical zero", 0.0, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateCoverageStatus(tt.percentage)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateRiskLevel(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		expected   string
	}{
		{"low risk", 85.0, "low"},
		{"low risk boundary", 80.0, "low"},
		{"medium risk", 70.0, "medium"},
		{"medium risk boundary", 60.0, "medium"},
		{"high risk", 50.0, "high"},
		{"high risk boundary", 40.0, "high"},
		{"critical risk", 30.0, "critical"},
		{"critical risk low", 10.0, "critical"},
		{"critical risk zero", 0.0, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateRiskLevel(tt.percentage)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateTrendGrade(t *testing.T) {
	tests := []struct {
		name      string
		direction string
		expected  string
	}{
		{"up direction", "up", "A"},
		{"improved direction", "improved", "A"},
		{"stable direction", "stable", "B"},
		{"down direction", "down", "D"},
		{"degraded direction", "degraded", "D"},
		{"unknown direction", "unknown", "C"},
		{"empty direction", "", "C"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTrendGrade(tt.direction)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineOverallImpact(t *testing.T) {
	tests := []struct {
		name       string
		difference float64
		expected   string
	}{
		{"positive impact", 2.5, "positive"},
		{"positive boundary", 1.1, "positive"},
		{"negative impact", -2.5, "negative"},
		{"negative boundary", -1.1, "negative"},
		{"neutral positive", 0.5, "neutral"},
		{"neutral negative", -0.5, "neutral"},
		{"neutral zero", 0.0, "neutral"},
		{"neutral boundary positive", 1.0, "neutral"},
		{"neutral boundary negative", -1.0, "neutral"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineOverallImpact(tt.difference)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineBadgeTrend(t *testing.T) {
	tests := []struct {
		name      string
		direction string
		expected  badge.TrendDirection
	}{
		{"up direction", "up", badge.TrendUp},
		{"UP direction", "UP", badge.TrendUp},
		{"improved direction", "improved", badge.TrendUp},
		{"IMPROVED direction", "IMPROVED", badge.TrendUp},
		{"down direction", "down", badge.TrendDown},
		{"DOWN direction", "DOWN", badge.TrendDown},
		{"degraded direction", "degraded", badge.TrendDown},
		{"DEGRADED direction", "DEGRADED", badge.TrendDown},
		{"stable direction", "stable", badge.TrendStable},
		{"unknown direction", "unknown", badge.TrendStable},
		{"empty direction", "", badge.TrendStable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineBadgeTrend(tt.direction)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertPRFileAnalysis(t *testing.T) {
	tests := []struct {
		name     string
		analysis *github.PRFileAnalysis
		expected *templates.PRFileAnalysisData
	}{
		{
			name:     "nil analysis",
			analysis: nil,
			expected: nil,
		},
		{
			name: "complete analysis",
			analysis: &github.PRFileAnalysis{
				Summary: github.PRFileSummary{
					TotalFiles:          10,
					GoFilesCount:        5,
					TestFilesCount:      2,
					ConfigFilesCount:    1,
					DocumentationCount:  1,
					GeneratedFilesCount: 1,
					OtherFilesCount:     0,
					HasGoChanges:        true,
					HasTestChanges:      true,
					HasConfigChanges:    false,
					TotalAdditions:      100,
					TotalDeletions:      50,
					GoAdditions:         80,
					GoDeletions:         30,
				},
				GoFiles: []github.PRFile{
					{
						Filename:  "main.go",
						Status:    "modified",
						Additions: 10,
						Deletions: 5,
						Changes:   15,
					},
				},
				TestFiles: []github.PRFile{
					{
						Filename:  "main_test.go",
						Status:    "added",
						Additions: 20,
						Deletions: 0,
						Changes:   20,
					},
				},
			},
			expected: &templates.PRFileAnalysisData{
				Summary: templates.PRFileSummaryData{
					TotalFiles:          10,
					GoFilesCount:        5,
					TestFilesCount:      2,
					ConfigFilesCount:    1,
					DocumentationCount:  1,
					GeneratedFilesCount: 1,
					OtherFilesCount:     0,
					HasGoChanges:        true,
					HasTestChanges:      true,
					HasConfigChanges:    false,
					TotalAdditions:      100,
					TotalDeletions:      50,
					GoAdditions:         80,
					GoDeletions:         30,
					SummaryText:         "", // This would be set by GetSummaryText() method
				},
				GoFiles: []templates.PRFileData{
					{
						Filename:  "main.go",
						Status:    "modified",
						Additions: 10,
						Deletions: 5,
						Changes:   15,
					},
				},
				TestFiles: []templates.PRFileData{
					{
						Filename:  "main_test.go",
						Status:    "added",
						Additions: 20,
						Deletions: 0,
						Changes:   20,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertPRFileAnalysis(tt.analysis)
			if tt.expected == nil {
				require.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			require.Equal(t, tt.expected.Summary.TotalFiles, result.Summary.TotalFiles)
			require.Equal(t, tt.expected.Summary.GoFilesCount, result.Summary.GoFilesCount)
			require.Equal(t, tt.expected.Summary.TestFilesCount, result.Summary.TestFilesCount)
			require.Equal(t, tt.expected.Summary.HasGoChanges, result.Summary.HasGoChanges)
			require.Equal(t, tt.expected.Summary.HasTestChanges, result.Summary.HasTestChanges)
			require.Equal(t, tt.expected.Summary.TotalAdditions, result.Summary.TotalAdditions)
			require.Equal(t, tt.expected.Summary.TotalDeletions, result.Summary.TotalDeletions)

			if len(tt.expected.GoFiles) > 0 {
				require.Len(t, result.GoFiles, len(tt.expected.GoFiles))
				require.Equal(t, tt.expected.GoFiles[0].Filename, result.GoFiles[0].Filename)
				require.Equal(t, tt.expected.GoFiles[0].Status, result.GoFiles[0].Status)
				require.Equal(t, tt.expected.GoFiles[0].Additions, result.GoFiles[0].Additions)
			}

			if len(tt.expected.TestFiles) > 0 {
				require.Len(t, result.TestFiles, len(tt.expected.TestFiles))
				require.Equal(t, tt.expected.TestFiles[0].Filename, result.TestFiles[0].Filename)
				require.Equal(t, tt.expected.TestFiles[0].Status, result.TestFiles[0].Status)
				require.Equal(t, tt.expected.TestFiles[0].Additions, result.TestFiles[0].Additions)
			}
		})
	}
}

func TestConvertPRFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []github.PRFile
		expected []templates.PRFileData
	}{
		{
			name:     "empty files",
			files:    []github.PRFile{},
			expected: []templates.PRFileData{},
		},
		{
			name: "single file",
			files: []github.PRFile{
				{
					Filename:  "main.go",
					Status:    "modified",
					Additions: 10,
					Deletions: 5,
					Changes:   15,
				},
			},
			expected: []templates.PRFileData{
				{
					Filename:  "main.go",
					Status:    "modified",
					Additions: 10,
					Deletions: 5,
					Changes:   15,
				},
			},
		},
		{
			name: "multiple files",
			files: []github.PRFile{
				{
					Filename:  "main.go",
					Status:    "modified",
					Additions: 10,
					Deletions: 5,
					Changes:   15,
				},
				{
					Filename:  "helper.go",
					Status:    "added",
					Additions: 20,
					Deletions: 0,
					Changes:   20,
				},
			},
			expected: []templates.PRFileData{
				{
					Filename:  "main.go",
					Status:    "modified",
					Additions: 10,
					Deletions: 5,
					Changes:   15,
				},
				{
					Filename:  "helper.go",
					Status:    "added",
					Additions: 20,
					Deletions: 0,
					Changes:   20,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertPRFiles(tt.files)
			require.Equal(t, tt.expected, result)
		})
	}
}
