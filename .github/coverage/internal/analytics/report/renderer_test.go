package report

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

// RendererTestSuite provides test suite for report renderer
type RendererTestSuite struct {
	suite.Suite

	renderer *Renderer
}

// SetupTest creates renderer for each test
func (suite *RendererTestSuite) SetupTest() {
	suite.renderer = NewRenderer()
}

// TestNewRendererSuccess tests successful renderer creation
func (suite *RendererTestSuite) TestNewRendererSuccess() {
	renderer := NewRenderer()

	suite.Require().NotNil(renderer)
	suite.NotNil(renderer.templates)
	suite.Empty(renderer.templates)
}

// TestRenderReportSuccess tests successful report rendering
func (suite *RendererTestSuite) TestRenderReportSuccess() {
	ctx := context.Background()
	data := suite.createSampleReportData()

	html, err := suite.renderer.RenderReport(ctx, data)
	suite.Require().NoError(err)
	suite.NotEmpty(html)

	htmlStr := string(html)

	// Verify basic HTML structure
	suite.Contains(htmlStr, "<!DOCTYPE html>")
	suite.Contains(htmlStr, "<html")
	suite.Contains(htmlStr, "</html>")
	suite.Contains(htmlStr, "<head>")
	suite.Contains(htmlStr, "<body>")

	// Verify coverage data is rendered
	suite.Contains(htmlStr, "85.5%") // Coverage percentage
	suite.Contains(htmlStr, "test-owner/test-repo")
	suite.Contains(htmlStr, "master") // Branch name

	// Verify packages are rendered
	suite.Contains(htmlStr, "test/package1")
	suite.Contains(htmlStr, "test/package2")
}

// TestRenderReportWithNilData tests rendering with nil data
func (suite *RendererTestSuite) TestRenderReportWithNilData() {
	ctx := context.Background()

	html, err := suite.renderer.RenderReport(ctx, nil)

	// Should return an error with nil data since template requires specific fields
	suite.Require().Error(err)
	suite.Empty(html)
}

// TestRenderReportWithEmptyData tests rendering with empty data
func (suite *RendererTestSuite) TestRenderReportWithEmptyData() {
	ctx := context.Background()
	emptyData := &Data{}

	html, err := suite.renderer.RenderReport(ctx, emptyData)
	suite.Require().NoError(err)
	suite.NotEmpty(html)

	htmlStr := string(html)
	suite.Contains(htmlStr, "<!DOCTYPE html>")
}

// TestRenderReportTemplateFunctions tests template functions
func (suite *RendererTestSuite) TestRenderReportTemplateFunctions() {
	ctx := context.Background()

	// Create data to test various template functions
	data := &Data{
		RepositoryOwner: "test-owner",
		RepositoryName:  "test-repo",
		BranchName:      "feature/long-branch-name",
		CommitSHA:       "1234567890abcdef",
		GeneratedAt:     time.Now(),
		Summary: Summary{
			TotalPercentage: 87.56789,
			TotalLines:      12345,
			CoveredLines:    10821,
		},
		Packages: []PackageReport{
			{
				Name:         "test/package",
				Percentage:   92.5,
				TotalLines:   100,
				CoveredLines: 92,
			},
		},
	}

	html, err := suite.renderer.RenderReport(ctx, data)
	suite.Require().NoError(err)

	htmlStr := string(html)

	// Test printf function (percentage formatting)
	suite.Contains(htmlStr, "87.6%") // Should be rounded to 1 decimal

	// Test commas function (number formatting)
	suite.Contains(htmlStr, "12,345") // Total lines with commas
	suite.Contains(htmlStr, "10,821") // Covered lines with commas

	// Test truncate function (commit SHA truncation)
	suite.Contains(htmlStr, "1234567") // Should be truncated to 7 chars

	// Test ge function (coverage thresholds)
	// Should have appropriate CSS classes for high coverage
	suite.Contains(htmlStr, "success") // High coverage class
}

// TestRenderReportCoverageThresholds tests coverage threshold styling
func (suite *RendererTestSuite) TestRenderReportCoverageThresholds() {
	testCases := []struct {
		name               string
		coverage           float64
		expectedClass      string
		shouldContainClass bool
	}{
		{
			name:               "Excellent coverage (>=95%)",
			coverage:           96.0,
			expectedClass:      "excellent",
			shouldContainClass: true,
		},
		{
			name:               "Good coverage (>=85%)",
			coverage:           87.0,
			expectedClass:      "success",
			shouldContainClass: true,
		},
		{
			name:               "Acceptable coverage (>=75%)",
			coverage:           78.0,
			expectedClass:      "warning",
			shouldContainClass: true,
		},
		{
			name:               "Low coverage (>=65%)",
			coverage:           68.0,
			expectedClass:      "low",
			shouldContainClass: true,
		},
		{
			name:               "Poor coverage (<65%)",
			coverage:           45.0,
			expectedClass:      "danger",
			shouldContainClass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx := context.Background()
			data := &Data{
				RepositoryOwner: "test-owner",
				RepositoryName:  "test-repo",
				Summary: Summary{
					TotalPercentage: tc.coverage,
					TotalLines:      100,
					CoveredLines:    int(tc.coverage),
				},
				Packages: []PackageReport{
					{
						Name:         "test/package",
						Percentage:   tc.coverage,
						TotalLines:   100,
						CoveredLines: int(tc.coverage),
					},
				},
			}

			html, err := suite.renderer.RenderReport(ctx, data)
			suite.Require().NoError(err)

			htmlStr := string(html)
			if tc.shouldContainClass {
				suite.Contains(htmlStr, tc.expectedClass,
					"Coverage %.1f%% should have class %s", tc.coverage, tc.expectedClass)
			}
		})
	}
}

// TestRenderReportGoogleAnalytics tests Google Analytics inclusion
func (suite *RendererTestSuite) TestRenderReportGoogleAnalytics() {
	ctx := context.Background()

	testCases := []struct {
		name                   string
		googleAnalyticsID      string
		shouldContainAnalytics bool
	}{
		{
			name:                   "With Google Analytics ID",
			googleAnalyticsID:      "GA-123456789",
			shouldContainAnalytics: true,
		},
		{
			name:                   "Without Google Analytics ID",
			googleAnalyticsID:      "",
			shouldContainAnalytics: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			data := &Data{
				RepositoryOwner:   "test-owner",
				RepositoryName:    "test-repo",
				GoogleAnalyticsID: tc.googleAnalyticsID,
			}

			html, err := suite.renderer.RenderReport(ctx, data)
			suite.Require().NoError(err)

			htmlStr := string(html)
			if tc.shouldContainAnalytics {
				suite.Contains(htmlStr, "gtag")
				suite.Contains(htmlStr, tc.googleAnalyticsID)
				suite.Contains(htmlStr, "googletagmanager.com")
			} else {
				suite.NotContains(htmlStr, "gtag")
				suite.NotContains(htmlStr, "googletagmanager.com")
			}
		})
	}
}

// TestRenderReportCommitURL tests commit URL rendering
func (suite *RendererTestSuite) TestRenderReportCommitURL() {
	ctx := context.Background()

	testCases := []struct {
		name           string
		commitURL      string
		shouldHaveLink bool
	}{
		{
			name:           "With commit URL",
			commitURL:      "https://github.com/owner/repo/commit/abc123",
			shouldHaveLink: true,
		},
		{
			name:           "Without commit URL",
			commitURL:      "",
			shouldHaveLink: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			data := &Data{
				RepositoryOwner: "test-owner",
				RepositoryName:  "test-repo",
				CommitSHA:       "abc123def456",
				CommitURL:       tc.commitURL,
			}

			html, err := suite.renderer.RenderReport(ctx, data)
			suite.Require().NoError(err)

			htmlStr := string(html)
			if tc.shouldHaveLink {
				suite.Contains(htmlStr, `href="`+tc.commitURL+`"`)
			}
		})
	}
}

// TestRenderReportTrendStatus tests coverage trend rendering
func (suite *RendererTestSuite) TestRenderReportTrendStatus() {
	ctx := context.Background()

	testCases := []struct {
		name         string
		changeStatus string
		expectedIcon string
		expectedText string
	}{
		{
			name:         "Improved coverage",
			changeStatus: "improved",
			expectedIcon: "📈",
			expectedText: "Improved",
		},
		{
			name:         "Declined coverage",
			changeStatus: "declined",
			expectedIcon: "📉",
			expectedText: "Declined",
		},
		{
			name:         "Stable coverage",
			changeStatus: "stable",
			expectedIcon: "➡️",
			expectedText: "Stable",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			data := &Data{
				RepositoryOwner: "test-owner",
				RepositoryName:  "test-repo",
				Summary: Summary{
					ChangeStatus:     tc.changeStatus,
					PreviousCoverage: 80.0,
				},
			}

			html, err := suite.renderer.RenderReport(ctx, data)
			suite.Require().NoError(err)

			htmlStr := string(html)
			if tc.changeStatus != "" {
				suite.Contains(htmlStr, tc.expectedIcon)
				suite.Contains(htmlStr, tc.expectedText)
			}
		})
	}
}

// TestAddCommasFunction tests the addCommas helper function
func (suite *RendererTestSuite) TestAddCommasFunction() {
	testCases := []struct {
		name     string
		input    int
		expected string
	}{
		{
			name:     "Single digit",
			input:    5,
			expected: "5",
		},
		{
			name:     "Two digits",
			input:    42,
			expected: "42",
		},
		{
			name:     "Three digits",
			input:    123,
			expected: "123",
		},
		{
			name:     "Four digits",
			input:    1234,
			expected: "1,234",
		},
		{
			name:     "Five digits",
			input:    12345,
			expected: "12,345",
		},
		{
			name:     "Six digits",
			input:    123456,
			expected: "123,456",
		},
		{
			name:     "Seven digits",
			input:    1234567,
			expected: "1,234,567",
		},
		{
			name:     "Eight digits",
			input:    12345678,
			expected: "12,345,678",
		},
		{
			name:     "Zero",
			input:    0,
			expected: "0",
		},
		{
			name:     "Large number",
			input:    1000000,
			expected: "1,000,000",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := addCommas(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

// TestRenderReportPackageExpansion tests package expansion functionality
func (suite *RendererTestSuite) TestRenderReportPackageExpansion() {
	ctx := context.Background()
	data := suite.createSampleReportDataWithFiles()

	html, err := suite.renderer.RenderReport(ctx, data)
	suite.Require().NoError(err)

	htmlStr := string(html)

	// Verify JavaScript for package expansion is included
	suite.Contains(htmlStr, "togglePackage")
	suite.Contains(htmlStr, "onclick=\"togglePackage")

	// Verify package toggle elements
	suite.Contains(htmlStr, "package-toggle")
	suite.Contains(htmlStr, "▶") // Collapsed state icon

	// Verify package files are initially hidden
	suite.Contains(htmlStr, `style="display: none;"`)
}

// TestRenderReportSearchFunctionality tests search functionality
func (suite *RendererTestSuite) TestRenderReportSearchFunctionality() {
	ctx := context.Background()
	data := suite.createSampleReportData()

	html, err := suite.renderer.RenderReport(ctx, data)
	suite.Require().NoError(err)

	htmlStr := string(html)

	// Verify search input exists
	suite.Contains(htmlStr, `id="searchInput"`)
	suite.Contains(htmlStr, `placeholder="Search packages and files..."`)

	// Verify search JavaScript files are loaded
	suite.Contains(htmlStr, "./assets/js/theme.js")
}

// TestRenderReportThemeToggle tests theme toggle functionality
func (suite *RendererTestSuite) TestRenderReportThemeToggle() {
	ctx := context.Background()
	data := suite.createSampleReportData()

	html, err := suite.renderer.RenderReport(ctx, data)
	suite.Require().NoError(err)

	htmlStr := string(html)

	// Verify theme toggle elements
	suite.Contains(htmlStr, "toggleTheme")
	suite.Contains(htmlStr, "theme-toggle")
	suite.Contains(htmlStr, `data-theme="auto"`)

	// Verify theme JavaScript files are loaded
	suite.Contains(htmlStr, "./assets/js/theme.js")
	suite.Contains(htmlStr, "./assets/js/coverage-time.js")
}

// TestRenderReportErrorHandling tests error handling in template rendering
func (suite *RendererTestSuite) TestRenderReportErrorHandling() {
	ctx := context.Background()

	// Test with data that might cause template errors
	invalidData := map[string]interface{}{
		"InvalidField": func() string { return "function" }, // Functions can't be serialized
	}

	html, err := suite.renderer.RenderReport(ctx, invalidData)

	// Should return an error with invalid/incomplete data
	suite.Require().Error(err)
	suite.Empty(html)
}

// TestConcurrentRendering tests concurrent template rendering
func (suite *RendererTestSuite) TestConcurrentRendering() {
	const numGoroutines = 10

	errChan := make(chan error, numGoroutines)
	doneChan := make(chan struct{}, numGoroutines)

	data := suite.createSampleReportData()

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { doneChan <- struct{}{} }()

			ctx := context.Background()
			html, err := suite.renderer.RenderReport(ctx, data)
			if err != nil {
				errChan <- err
				return
			}

			if len(html) == 0 {
				errChan <- assert.AnError
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-doneChan
	}

	// Check for errors
	close(errChan)
	for err := range errChan {
		suite.T().Errorf("Concurrent rendering error: %v", err)
	}
}

// TestRenderReportLargeData tests rendering with large datasets
func (suite *RendererTestSuite) TestRenderReportLargeData() {
	ctx := context.Background()
	data := suite.createLargeReportData()

	html, err := suite.renderer.RenderReport(ctx, data)
	suite.Require().NoError(err)
	suite.NotEmpty(html)

	// Verify all packages are rendered
	htmlStr := string(html)
	for _, pkg := range data.Packages {
		suite.Contains(htmlStr, pkg.Name)
	}
}

// Helper methods for creating test data

// createSampleReportData creates sample report data for testing
func (suite *RendererTestSuite) createSampleReportData() *Data {
	return &Data{
		Coverage: &parser.CoverageData{
			Percentage:   85.5,
			TotalLines:   1000,
			CoveredLines: 855,
		},
		GeneratedAt:     time.Now(),
		ProjectName:     "Test Project",
		RepositoryOwner: "test-owner",
		RepositoryName:  "test-repo",
		BranchName:      "master",
		CommitSHA:       "abc123def456",
		CommitURL:       "https://github.com/test-owner/test-repo/commit/abc123def456",
		Summary: Summary{
			TotalPercentage: 85.5,
			TotalLines:      1000,
			CoveredLines:    855,
			UncoveredLines:  145,
			PackageCount:    2,
			FileCount:       5,
		},
		Packages: []PackageReport{
			{
				Name:         "test/package1",
				Percentage:   90.0,
				TotalLines:   500,
				CoveredLines: 450,
			},
			{
				Name:         "test/package2",
				Percentage:   81.0,
				TotalLines:   500,
				CoveredLines: 405,
			},
		},
		GoogleAnalyticsID: "GA-123456789",
	}
}

// createSampleReportDataWithFiles creates report data with file details
func (suite *RendererTestSuite) createSampleReportDataWithFiles() *Data {
	data := suite.createSampleReportData()

	// Add files to packages
	data.Packages[0].Files = []FileReport{
		{
			Name:         "file1.go",
			Path:         "test/package1/file1.go",
			Percentage:   95.0,
			TotalLines:   200,
			CoveredLines: 190,
		},
		{
			Name:         "file2.go",
			Path:         "test/package1/file2.go",
			Percentage:   85.0,
			TotalLines:   300,
			CoveredLines: 255,
		},
	}

	data.Packages[1].Files = []FileReport{
		{
			Name:         "handler.go",
			Path:         "test/package2/handler.go",
			Percentage:   75.0,
			TotalLines:   400,
			CoveredLines: 300,
		},
	}

	return data
}

// createLargeReportData creates a large dataset for testing
func (suite *RendererTestSuite) createLargeReportData() *Data {
	packages := make([]PackageReport, 50)
	totalLines := 0
	totalCovered := 0

	for i := 0; i < 50; i++ {
		lines := 100 + i*10
		covered := int(float64(lines) * 0.8) // 80% coverage

		packages[i] = PackageReport{
			Name:         fmt.Sprintf("package/test%02d", i),
			Percentage:   float64(covered) / float64(lines) * 100,
			TotalLines:   lines,
			CoveredLines: covered,
		}

		totalLines += lines
		totalCovered += covered
	}

	return &Data{
		GeneratedAt:     time.Now(),
		RepositoryOwner: "test-owner",
		RepositoryName:  "test-repo",
		Summary: Summary{
			TotalPercentage: float64(totalCovered) / float64(totalLines) * 100,
			TotalLines:      totalLines,
			CoveredLines:    totalCovered,
			PackageCount:    len(packages),
		},
		Packages: packages,
	}
}

// TestRun runs the test suite
func TestRendererTestSuite(t *testing.T) {
	suite.Run(t, new(RendererTestSuite))
}

// Benchmark tests
func BenchmarkNewRenderer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewRenderer()
	}
}

func BenchmarkRenderReport(b *testing.B) {
	renderer := NewRenderer()
	ctx := context.Background()

	data := &Data{
		RepositoryOwner: "test-owner",
		RepositoryName:  "test-repo",
		Summary: Summary{
			TotalPercentage: 85.0,
			TotalLines:      1000,
			CoveredLines:    850,
		},
		Packages: []PackageReport{
			{
				Name:         "test/package",
				Percentage:   85.0,
				TotalLines:   1000,
				CoveredLines: 850,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := renderer.RenderReport(ctx, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAddCommas(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = addCommas(1234567)
	}
}
