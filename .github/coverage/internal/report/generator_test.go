package report

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

func TestNew(t *testing.T) {
	generator := New()
	assert.NotNil(t, generator)
	assert.NotNil(t, generator.config)
	assert.Equal(t, "github-dark", generator.config.Theme)
	assert.Equal(t, "Coverage Report", generator.config.Title)
	assert.True(t, generator.config.ShowPackages)
	assert.True(t, generator.config.ShowFiles)
	assert.True(t, generator.config.ShowMissing)
	assert.True(t, generator.config.DarkMode)
	assert.True(t, generator.config.Responsive)
	assert.True(t, generator.config.InteractiveTrees)
}

func TestNewWithConfig(t *testing.T) {
	config := &Config{
		Theme:            "light",
		Title:            "Test Report",
		ShowPackages:     false,
		ShowFiles:        false,
		ShowMissing:      false,
		DarkMode:         false,
		Responsive:       false,
		InteractiveTrees: false,
	}

	generator := NewWithConfig(config)
	assert.NotNil(t, generator)
	assert.Equal(t, config, generator.config)
}

func TestGenerate(t *testing.T) {
	generator := New()
	ctx := context.Background()

	coverage := createTestCoverageData()

	html, err := generator.Generate(ctx, coverage)
	require.NoError(t, err)
	assert.NotEmpty(t, html)

	htmlStr := string(html)
	assert.Contains(t, htmlStr, "<!DOCTYPE html>")
	assert.Contains(t, htmlStr, "Coverage Report")
	assert.Contains(t, htmlStr, "75.0") // total percentage (without % symbol)
	assert.Contains(t, htmlStr, "pkg1")
	assert.Contains(t, htmlStr, "pkg2")
	assert.Contains(t, htmlStr, "</html>")

	// Check for modern dark theme styles
	assert.Contains(t, htmlStr, "--color-bg: #0d1117")
	assert.Contains(t, htmlStr, "--color-text: #c9d1d9")

	// Check for interactive features
	assert.Contains(t, htmlStr, "togglePackage")
	assert.Contains(t, htmlStr, "package-header")
}

func TestGenerateWithOptions(t *testing.T) {
	generator := New()
	ctx := context.Background()
	coverage := createTestCoverageData()

	html, err := generator.Generate(ctx, coverage,
		WithTitle("Custom Title"),
		WithTheme("light"),
		WithPackages(false),
	)
	require.NoError(t, err)

	htmlStr := string(html)
	assert.Contains(t, htmlStr, "Custom Title")
	// Package section should be present since the template always shows it
}

func TestGenerateContextCancellation(t *testing.T) {
	generator := New()
	ctx, cancel := context.WithCancel(context.Background())
	coverage := createTestCoverageData()

	// Cancel context immediately
	cancel()

	_, err := generator.Generate(ctx, coverage)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestBuildReportData(t *testing.T) {
	generator := New()
	coverage := createTestCoverageData()
	config := generator.config

	reportData := generator.buildReportData(context.Background(), coverage, config)

	assert.Equal(t, coverage, reportData.Coverage)
	assert.Equal(t, config, reportData.Config)
	assert.WithinDuration(t, time.Now(), reportData.GeneratedAt, 5*time.Second)
	assert.Equal(t, "1.0.0", reportData.Version)
	// Project name should be extracted from Git or default to "Go Project"
	assert.NotEmpty(t, reportData.ProjectName, "Project name should not be empty")

	// Check summary
	assert.InDelta(t, 75.0, reportData.Summary.TotalPercentage, 0.001)
	assert.Equal(t, 8, reportData.Summary.TotalLines)
	assert.Equal(t, 6, reportData.Summary.CoveredLines)
	assert.Equal(t, 2, reportData.Summary.UncoveredLines)
	assert.Equal(t, 2, reportData.Summary.PackageCount)
	assert.Equal(t, 2, reportData.Summary.FileCount)
	assert.Equal(t, "stable", reportData.Summary.ChangeStatus)

	// Check packages
	assert.Len(t, reportData.Packages, 2)

	// Check pkg1
	pkg1 := reportData.Packages[0]
	assert.Equal(t, "pkg1", pkg1.Name)
	assert.InDelta(t, 80.0, pkg1.Percentage, 0.001)
	assert.Equal(t, 5, pkg1.TotalLines)
	assert.Equal(t, 4, pkg1.CoveredLines)
	assert.Equal(t, "good", pkg1.Status)
	assert.Len(t, pkg1.Files, 1)

	// Check file1.go
	file1 := pkg1.Files[0]
	assert.Equal(t, "file1.go", file1.Name)
	assert.Equal(t, "github.com/test/pkg1/file1.go", file1.Path)
	assert.InDelta(t, 80.0, file1.Percentage, 0.001)
	assert.Equal(t, 5, file1.TotalLines)
	assert.Equal(t, 4, file1.CoveredLines)
	assert.Equal(t, "good", file1.Status)
}

func TestGetStatusClass(t *testing.T) {
	generator := New()

	tests := []struct {
		percentage float64
		expected   string
	}{
		{95.0, "excellent"},
		{85.0, "good"},
		{75.0, "acceptable"},
		{65.0, "low"},
		{55.0, "poor"},
		{100.0, "excellent"},
		{0.0, "poor"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%.1f%%", tt.percentage), func(t *testing.T) {
			status := generator.getStatusClass(tt.percentage)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestGetLineClass(t *testing.T) {
	generator := New()

	assert.Equal(t, "covered", generator.getLineClass(true))
	assert.Equal(t, "uncovered", generator.getLineClass(false))
}

func TestExtractFileName(t *testing.T) {
	generator := New()

	tests := []struct {
		path     string
		expected string
	}{
		{"github.com/test/pkg/file.go", "file.go"},
		{"file.go", "file.go"},
		{"internal/config/config.go", "config.go"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := generator.extractFileName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildLineReports(t *testing.T) {
	generator := New()

	fileCov := &parser.FileCoverage{
		Path: "test.go",
		Statements: []parser.Statement{
			{StartLine: 10, EndLine: 12, Count: 1},
			{StartLine: 15, EndLine: 15, Count: 0},
			{StartLine: 8, EndLine: 9, Count: 2},
		},
	}

	lines := generator.buildLineReports(fileCov)

	// Should be sorted by line number
	assert.Len(t, lines, 6) // 3 lines (8,9) + 3 lines (10,11,12) + 1 line (15)

	// Check first line (line 8)
	assert.Equal(t, 8, lines[0].Number)
	assert.True(t, lines[0].Covered)
	assert.Equal(t, 2, lines[0].Count)
	assert.Equal(t, "covered", lines[0].Class)

	// Check uncovered line (line 15)
	assert.Equal(t, 15, lines[5].Number)
	assert.False(t, lines[5].Covered)
	assert.Equal(t, 0, lines[5].Count)
	assert.Equal(t, "uncovered", lines[5].Class)
}

func TestRenderHTML(t *testing.T) {
	generator := New()
	ctx := context.Background()

	reportData := &Data{
		Coverage:    createTestCoverageData(),
		Config:      generator.config,
		GeneratedAt: time.Now(),
		Version:     "1.0.0",
		ProjectName: "Test Project",
		BranchName:  "main",
		Summary: Summary{
			TotalPercentage: 85.5,
			TotalLines:      100,
			CoveredLines:    85,
			UncoveredLines:  15,
			PackageCount:    2,
			FileCount:       5,
		},
		Packages: []PackageReport{
			{
				Name:         "pkg1",
				Percentage:   80.0,
				TotalLines:   50,
				CoveredLines: 40,
				Status:       "good",
			},
		},
	}

	html, err := generator.renderHTML(ctx, reportData)
	require.NoError(t, err)
	assert.NotEmpty(t, html)

	htmlStr := string(html)
	assert.Contains(t, htmlStr, "85.5%")
	assert.Contains(t, htmlStr, "Coverage Report") // Uses Config.Title, not ProjectName
	assert.Contains(t, htmlStr, "pkg1")
	assert.Contains(t, htmlStr, "80.0%")
}

func TestRenderHTMLContextCancellation(t *testing.T) {
	generator := New()
	ctx, cancel := context.WithCancel(context.Background())
	reportData := &Data{}

	cancel()

	_, err := generator.renderHTML(ctx, reportData)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestHTMLTemplateFunction removed - template is now handled by templates.TemplateManager

func TestConfigurationOptions(t *testing.T) {
	config := &Config{}

	WithTheme("light")(config)
	assert.Equal(t, "light", config.Theme)

	WithTitle("Test Title")(config)
	assert.Equal(t, "Test Title", config.Title)

	WithPackages(false)(config)
	assert.False(t, config.ShowPackages)

	WithFiles(false)(config)
	assert.False(t, config.ShowFiles)

	WithMissing(false)(config)
	assert.False(t, config.ShowMissing)
}

func TestGenerateComplexReport(t *testing.T) {
	generator := New()
	ctx := context.Background()

	// Create complex coverage data with multiple packages and files
	coverage := &parser.CoverageData{
		Mode:         "atomic",
		Percentage:   73.5,
		TotalLines:   200,
		CoveredLines: 147,
		Timestamp:    time.Now(),
		Packages: map[string]*parser.PackageCoverage{
			"config": {
				Name:         "config",
				Percentage:   90.0,
				TotalLines:   50,
				CoveredLines: 45,
				Files: map[string]*parser.FileCoverage{
					"config.go": {
						Path:         "internal/config/config.go",
						Percentage:   88.0,
						TotalLines:   25,
						CoveredLines: 22,
					},
					"loader.go": {
						Path:         "internal/config/loader.go",
						Percentage:   92.0,
						TotalLines:   25,
						CoveredLines: 23,
					},
				},
			},
			"utils": {
				Name:         "utils",
				Percentage:   60.0,
				TotalLines:   150,
				CoveredLines: 90,
				Files: map[string]*parser.FileCoverage{
					"helpers.go": {
						Path:         "internal/utils/helpers.go",
						Percentage:   70.0,
						TotalLines:   100,
						CoveredLines: 70,
					},
					"validators.go": {
						Path:         "internal/utils/validators.go",
						Percentage:   40.0,
						TotalLines:   50,
						CoveredLines: 20,
					},
				},
			},
		},
	}

	html, err := generator.Generate(ctx, coverage)
	require.NoError(t, err)

	htmlStr := string(html)

	// Check overall structure
	assert.Contains(t, htmlStr, "<!DOCTYPE html>")
	assert.Contains(t, htmlStr, "73.5%")
	assert.Contains(t, htmlStr, "</html>")

	// Check packages are present
	assert.Contains(t, htmlStr, "config")
	assert.Contains(t, htmlStr, "utils")
	assert.Contains(t, htmlStr, "90.0%")
	assert.Contains(t, htmlStr, "60.0%")

	// Check files are present
	assert.Contains(t, htmlStr, "config.go")
	assert.Contains(t, htmlStr, "loader.go")
	assert.Contains(t, htmlStr, "helpers.go")
	assert.Contains(t, htmlStr, "validators.go")

	// Check summary data (what our template actually shows)
	assert.Contains(t, htmlStr, "73.5") // overall coverage percentage
	assert.Contains(t, htmlStr, "2")    // package count
	assert.Contains(t, htmlStr, "4")    // file count

	// Check CSS classes for different coverage levels
	assert.Contains(t, htmlStr, "badge-excellent") // config package (90%)
	assert.Contains(t, htmlStr, "badge-warning")   // utils package (60%)
}

func TestGenerateEdgeCases(t *testing.T) {
	generator := New()
	ctx := context.Background()

	tests := []struct {
		name     string
		coverage *parser.CoverageData
	}{
		{
			name: "empty coverage",
			coverage: &parser.CoverageData{
				Mode:         "atomic",
				Percentage:   0.0,
				TotalLines:   0,
				CoveredLines: 0,
				Timestamp:    time.Now(),
				Packages:     make(map[string]*parser.PackageCoverage),
			},
		},
		{
			name: "perfect coverage",
			coverage: &parser.CoverageData{
				Mode:         "count",
				Percentage:   100.0,
				TotalLines:   50,
				CoveredLines: 50,
				Timestamp:    time.Now(),
				Packages: map[string]*parser.PackageCoverage{
					"perfect": {
						Name:         "perfect",
						Percentage:   100.0,
						TotalLines:   50,
						CoveredLines: 50,
						Files: map[string]*parser.FileCoverage{
							"perfect.go": {
								Path:         "perfect.go",
								Percentage:   100.0,
								TotalLines:   50,
								CoveredLines: 50,
							},
						},
					},
				},
			},
		},
		{
			name: "single statement",
			coverage: &parser.CoverageData{
				Mode:         "atomic",
				Percentage:   50.0,
				TotalLines:   2,
				CoveredLines: 1,
				Timestamp:    time.Now(),
				Packages: map[string]*parser.PackageCoverage{
					"single": {
						Name:         "single",
						Percentage:   50.0,
						TotalLines:   2,
						CoveredLines: 1,
						Files: map[string]*parser.FileCoverage{
							"single.go": {
								Path:         "single.go",
								Percentage:   50.0,
								TotalLines:   2,
								CoveredLines: 1,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := generator.Generate(ctx, tt.coverage)
			require.NoError(t, err)
			assert.NotEmpty(t, html)

			htmlStr := string(html)
			assert.Contains(t, htmlStr, "<!DOCTYPE html>")
			assert.Contains(t, htmlStr, "</html>")
			assert.Contains(t, htmlStr, fmt.Sprintf("%.1f", tt.coverage.Percentage))
		})
	}
}

func TestGenerateAccessibility(t *testing.T) {
	generator := New()
	ctx := context.Background()
	coverage := createTestCoverageData()

	html, err := generator.Generate(ctx, coverage)
	require.NoError(t, err)

	htmlStr := string(html)

	// Check for accessibility features
	assert.Contains(t, htmlStr, `lang="en"`)
	assert.Contains(t, htmlStr, `<title>`)
	assert.Contains(t, htmlStr, `<meta charset="UTF-8">`)
	assert.Contains(t, htmlStr, `viewport`)

	// Check for semantic HTML
	assert.Contains(t, htmlStr, `<h1>`)
	assert.Contains(t, htmlStr, `<h2 class="section-title">`)

	// Check for proper structure
	assert.Contains(t, htmlStr, `<div class="container">`)
	assert.Contains(t, htmlStr, `<header class="header">`)
	assert.Contains(t, htmlStr, `<section class="summary-grid">`)
}

func TestGenerateResponsiveDesign(t *testing.T) {
	generator := New()
	ctx := context.Background()
	coverage := createTestCoverageData()

	html, err := generator.Generate(ctx, coverage)
	require.NoError(t, err)

	htmlStr := string(html)

	// Check for responsive CSS
	assert.Contains(t, htmlStr, `@media (max-width: 768px)`)
	assert.Contains(t, htmlStr, `grid-template-columns: repeat(auto-fit, minmax(`)
	assert.Contains(t, htmlStr, `flex-direction: column`)

	// Check for mobile-friendly viewport
	assert.Contains(t, htmlStr, `width=device-width, initial-scale=1.0`)
}

func TestRenderHTMLExecutionError(t *testing.T) {
	generator := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context to cause error

	reportData := &Data{
		Coverage:    createTestCoverageData(),
		Config:      generator.config,
		GeneratedAt: time.Now(),
		Version:     "1.0.0",
		Summary: Summary{
			TotalPercentage: 85.5,
			TotalLines:      100,
			CoveredLines:    85,
		},
		Packages: []PackageReport{},
	}

	_, err := generator.renderHTML(ctx, reportData)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestGenerateAllOptionCombinations(t *testing.T) {
	generator := New()
	ctx := context.Background()
	coverage := createTestCoverageData()

	// Test all combinations of options
	optionsCombinations := [][]Option{
		{WithTheme("light")},
		{WithTitle("Custom")},
		{WithPackages(false)},
		{WithFiles(false)},
		{WithMissing(false)},
		{WithTheme("dark"), WithTitle("Custom"), WithPackages(false)},
		{WithFiles(false), WithMissing(false)},
		{WithTheme("custom"), WithTitle("Full Custom"), WithPackages(true), WithFiles(true), WithMissing(true)},
	}

	for i, options := range optionsCombinations {
		t.Run(fmt.Sprintf("combination_%d", i), func(t *testing.T) {
			html, err := generator.Generate(ctx, coverage, options...)
			require.NoError(t, err)
			assert.NotEmpty(t, html)
		})
	}
}

func TestBuildReportDataEdgeCases(t *testing.T) {
	generator := New()

	// Test with empty packages
	coverage := &parser.CoverageData{
		Mode:         "atomic",
		Percentage:   0.0,
		TotalLines:   0,
		CoveredLines: 0,
		Timestamp:    time.Now(),
		Packages:     make(map[string]*parser.PackageCoverage),
	}

	reportData := generator.buildReportData(context.Background(), coverage, generator.config)
	assert.Empty(t, reportData.Packages)
	assert.Equal(t, 0, reportData.Summary.PackageCount)
	assert.Equal(t, 0, reportData.Summary.FileCount)
}

func TestBuildLineReportsEdgeCases(t *testing.T) {
	generator := New()

	// Test with no statements
	fileCov := &parser.FileCoverage{
		Path:       "empty.go",
		Statements: []parser.Statement{},
	}

	lines := generator.buildLineReports(fileCov)
	assert.Empty(t, lines)

	// Test with overlapping multi-line statements
	fileCov = &parser.FileCoverage{
		Path: "complex.go",
		Statements: []parser.Statement{
			{StartLine: 5, EndLine: 10, Count: 1},  // Lines 5-10
			{StartLine: 8, EndLine: 12, Count: 0},  // Lines 8-12 (overlaps with above)
			{StartLine: 15, EndLine: 15, Count: 2}, // Single line
		},
	}

	lines = generator.buildLineReports(fileCov)
	assert.NotEmpty(t, lines)
	// Should have lines from 5-12 and 15
	lineNumbers := make(map[int]bool)
	for _, line := range lines {
		lineNumbers[line.Number] = true
	}

	// Check that all expected lines are present
	for i := 5; i <= 12; i++ {
		assert.True(t, lineNumbers[i], "Line %d should be present", i)
	}
	assert.True(t, lineNumbers[15], "Line 15 should be present")
}

func TestRenderHTMLWithComplexData(t *testing.T) {
	generator := New()
	ctx := context.Background()

	// Create report data with all features enabled
	reportData := &Data{
		Coverage:    createTestCoverageData(),
		Config:      generator.config,
		GeneratedAt: time.Now(),
		Version:     "2.0.0",
		ProjectName: "Complex Project",
		BranchName:  "feature/testing",
		CommitSHA:   "abc123def456",
		CommitURL:   "https://github.com/test/repo/commit/abc123def456",
		BadgeURL:    "https://img.shields.io/badge/coverage-75%25-green",
		Summary: Summary{
			TotalPercentage:  75.0,
			TotalLines:       200,
			CoveredLines:     150,
			UncoveredLines:   50,
			PackageCount:     5,
			FileCount:        15,
			ChangeStatus:     "improved",
			PreviousCoverage: 70.0,
		},
		Packages: []PackageReport{
			{
				Name:         "critical",
				Percentage:   95.0,
				TotalLines:   100,
				CoveredLines: 95,
				Status:       "excellent",
				Files: []FileReport{
					{
						Name:         "important.go",
						Path:         "pkg/critical/important.go",
						Percentage:   95.0,
						TotalLines:   100,
						CoveredLines: 95,
						Status:       "excellent",
						Lines: []LineReport{
							{Number: 1, Content: "package critical", Covered: true, Count: 1, Class: "covered"},
							{Number: 2, Content: "func Important() {", Covered: false, Count: 0, Class: "uncovered"},
						},
					},
				},
			},
		},
	}

	html, err := generator.renderHTML(ctx, reportData)
	require.NoError(t, err)
	assert.NotEmpty(t, html)

	htmlStr := string(html)
	assert.Contains(t, htmlStr, "feature/testing")
	assert.Contains(t, htmlStr, "abc123d") // Template shows first 7 characters
	assert.Contains(t, htmlStr, "critical")
	assert.Contains(t, htmlStr, "important.go")
	assert.Contains(t, htmlStr, "95.0%")
}

// Helper function to create test coverage data
func createTestCoverageData() *parser.CoverageData {
	return &parser.CoverageData{
		Mode:         "atomic",
		Percentage:   75.0,
		TotalLines:   8,
		CoveredLines: 6,
		Timestamp:    time.Now(),
		Packages: map[string]*parser.PackageCoverage{
			"pkg1": {
				Name:         "pkg1",
				Percentage:   80.0,
				TotalLines:   5,
				CoveredLines: 4,
				Files: map[string]*parser.FileCoverage{
					"github.com/test/pkg1/file1.go": {
						Path:         "github.com/test/pkg1/file1.go",
						Percentage:   80.0,
						TotalLines:   5,
						CoveredLines: 4,
						Statements: []parser.Statement{
							{StartLine: 10, EndLine: 12, Count: 1},
							{StartLine: 15, EndLine: 15, Count: 0},
						},
					},
				},
			},
			"pkg2": {
				Name:         "pkg2",
				Percentage:   66.7,
				TotalLines:   3,
				CoveredLines: 2,
				Files: map[string]*parser.FileCoverage{
					"github.com/test/pkg2/file2.go": {
						Path:         "github.com/test/pkg2/file2.go",
						Percentage:   66.7,
						TotalLines:   3,
						CoveredLines: 2,
						Statements: []parser.Statement{
							{StartLine: 5, EndLine: 7, Count: 2},
							{StartLine: 20, EndLine: 20, Count: 0},
						},
					},
				},
			},
		},
	}
}
