package report

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

// GeneratorTestSuite provides test suite for report generator
type GeneratorTestSuite struct {
	suite.Suite

	tempDir string
	config  *Config
}

// SetupTest creates temporary directory and config for each test
func (suite *GeneratorTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "generator_test_*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	suite.config = &Config{
		OutputDir:         suite.tempDir,
		RepositoryOwner:   "test-owner",
		RepositoryName:    "test-repo",
		BranchName:        "master",
		CommitSHA:         "abc123def456",
		GoogleAnalyticsID: "GA-123456789",
	}
}

// TearDownTest cleans up temporary directory after each test
func (suite *GeneratorTestSuite) TearDownTest() {
	if suite.tempDir != "" {
		err := os.RemoveAll(suite.tempDir)
		suite.Require().NoError(err)
	}
}

// TestNewGeneratorSuccess tests successful generator creation
func (suite *GeneratorTestSuite) TestNewGeneratorSuccess() {
	generator := NewGenerator(suite.config)

	suite.Require().NotNil(generator)
	suite.Equal(suite.config, generator.config)
	suite.NotNil(generator.renderer)
}

// TestNewGeneratorNilConfig tests generator creation with nil config
func (suite *GeneratorTestSuite) TestNewGeneratorNilConfig() {
	// Should not panic with nil config
	generator := NewGenerator(nil)
	suite.Require().NotNil(generator)
	suite.Nil(generator.config)
	suite.NotNil(generator.renderer)
}

// TestGenerateSuccess tests successful report generation
func (suite *GeneratorTestSuite) TestGenerateSuccess() {
	ctx := context.Background()
	generator := NewGenerator(suite.config)

	// Create sample coverage data
	coverageData := suite.createSampleCoverageData()

	err := generator.Generate(ctx, coverageData)
	suite.Require().NoError(err)

	// Verify output file was created
	reportPath := filepath.Join(suite.tempDir, "coverage.html")
	stat, err := os.Stat(reportPath)
	suite.Require().NoError(err)
	suite.Positive(stat.Size())

	// Verify assets were copied
	assetsDir := filepath.Join(suite.tempDir, "assets")
	stat, err = os.Stat(assetsDir)
	suite.Require().NoError(err)
	suite.True(stat.IsDir())

	// Verify some expected asset files exist
	expectedAssetFiles := []string{
		"css/coverage.css",
		"images/favicon.ico",
	}

	for _, assetFile := range expectedAssetFiles {
		assetPath := filepath.Join(assetsDir, assetFile)
		_, err := os.Stat(assetPath)
		suite.NoError(err, "Expected asset file %s should exist", assetFile)
	}
}

// TestGenerateWithNilCoverage tests generation with nil coverage data
func (suite *GeneratorTestSuite) TestGenerateWithNilCoverage() {
	ctx := context.Background()
	generator := NewGenerator(suite.config)

	err := generator.Generate(ctx, nil)
	suite.Require().NoError(err)

	// Should still create report file
	reportPath := filepath.Join(suite.tempDir, "coverage.html")
	_, err = os.Stat(reportPath)
	suite.NoError(err)
}

// TestGenerateWithEmptyConfig tests generation with empty config
func (suite *GeneratorTestSuite) TestGenerateWithEmptyConfig() {
	ctx := context.Background()
	emptyConfig := &Config{
		OutputDir: suite.tempDir,
		// All other fields empty
	}
	generator := NewGenerator(emptyConfig)
	coverageData := suite.createSampleCoverageData()

	err := generator.Generate(ctx, coverageData)
	suite.Require().NoError(err)

	// Should still create report file
	reportPath := filepath.Join(emptyConfig.OutputDir, "coverage.html")
	_, err = os.Stat(reportPath)
	suite.NoError(err)
}

// TestGenerateInvalidOutputDir tests generation with invalid output directory
func (suite *GeneratorTestSuite) TestGenerateInvalidOutputDir() {
	ctx := context.Background()

	// Use a file as output directory (should fail)
	invalidFile := filepath.Join(suite.tempDir, "not_a_directory")
	err := os.WriteFile(invalidFile, []byte("test"), 0o600)
	suite.Require().NoError(err)

	invalidConfig := &Config{
		OutputDir:       invalidFile,
		RepositoryOwner: "test-owner",
		RepositoryName:  "test-repo",
	}

	generator := NewGenerator(invalidConfig)
	coverageData := suite.createSampleCoverageData()

	err = generator.Generate(ctx, coverageData)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "creating output directory")
}

// TestGeneratePermissionDenied tests generation with permission issues
func (suite *GeneratorTestSuite) TestGeneratePermissionDenied() {
	if os.Getuid() == 0 {
		suite.T().Skip("Skipping permission test when running as root")
	}

	ctx := context.Background()

	// Create read-only directory
	restrictedDir := filepath.Join(suite.tempDir, "restricted")
	err := os.MkdirAll(restrictedDir, 0o400)
	suite.Require().NoError(err)

	restrictedConfig := &Config{
		OutputDir:       restrictedDir,
		RepositoryOwner: "test-owner",
		RepositoryName:  "test-repo",
	}

	generator := NewGenerator(restrictedConfig)
	coverageData := suite.createSampleCoverageData()

	err = generator.Generate(ctx, coverageData)
	suite.Error(err)
}

// TestBuildReportDataSuccess tests successful report data building
func (suite *GeneratorTestSuite) TestBuildReportDataSuccess() {
	ctx := context.Background()
	generator := NewGenerator(suite.config)
	coverageData := suite.createSampleCoverageData()

	data := generator.buildReportData(ctx, coverageData)

	suite.Require().NotNil(data)
	suite.Equal(coverageData, data.Coverage)
	suite.Equal(suite.config.RepositoryOwner, data.RepositoryOwner)
	suite.Equal(suite.config.RepositoryName, data.RepositoryName)
	suite.Equal(suite.config.BranchName, data.BranchName)
	suite.Equal(suite.config.CommitSHA, data.CommitSHA)
	suite.Equal(suite.config.GoogleAnalyticsID, data.GoogleAnalyticsID)

	// Verify commit URL is built correctly
	expectedCommitURL := "https://github.com/test-owner/test-repo/commit/abc123def456"
	suite.Equal(expectedCommitURL, data.CommitURL)

	// Verify timestamp is recent
	suite.Less(time.Since(data.GeneratedAt), time.Minute)

	// Verify summary data
	suite.InEpsilon(coverageData.Percentage, data.Summary.TotalPercentage, 0.001)
	suite.Equal(coverageData.TotalLines, data.Summary.TotalLines)
	suite.Equal(coverageData.CoveredLines, data.Summary.CoveredLines)
	suite.Equal(coverageData.TotalLines-coverageData.CoveredLines, data.Summary.UncoveredLines)
	suite.Equal(len(coverageData.Packages), data.Summary.PackageCount)

	// Verify packages are included
	suite.Len(data.Packages, len(coverageData.Packages))

	// Verify packages are sorted
	if len(data.Packages) > 1 {
		for i := 1; i < len(data.Packages); i++ {
			suite.LessOrEqual(data.Packages[i-1].Name, data.Packages[i].Name,
				"Packages should be sorted alphabetically")
		}
	}
}

// TestBuildReportDataWithNilCoverage tests building report data with nil coverage
func (suite *GeneratorTestSuite) TestBuildReportDataWithNilCoverage() {
	ctx := context.Background()
	generator := NewGenerator(suite.config)

	// This should not panic
	data := generator.buildReportData(ctx, nil)
	suite.Require().NotNil(data)
	suite.Nil(data.Coverage)
}

// TestBuildReportDataEmptyCoverage tests building report data with empty coverage
func (suite *GeneratorTestSuite) TestBuildReportDataEmptyCoverage() {
	ctx := context.Background()
	generator := NewGenerator(suite.config)

	emptyCoverage := &parser.CoverageData{
		Packages:     make(map[string]*parser.PackageCoverage),
		Percentage:   0.0,
		TotalLines:   0,
		CoveredLines: 0,
	}

	data := generator.buildReportData(ctx, emptyCoverage)
	suite.Require().NotNil(data)
	suite.Empty(data.Packages)
	suite.Equal(0, data.Summary.PackageCount)
	suite.Equal(0, data.Summary.FileCount)
}

// TestBuildReportDataNoGitHubInfo tests building report data without GitHub info
func (suite *GeneratorTestSuite) TestBuildReportDataNoGitHubInfo() {
	ctx := context.Background()

	configNoGitHub := &Config{
		OutputDir: suite.tempDir,
		// No GitHub info
	}

	generator := NewGenerator(configNoGitHub)
	coverageData := suite.createSampleCoverageData()

	data := generator.buildReportData(ctx, coverageData)
	suite.Require().NotNil(data)

	// Commit URL should be empty
	suite.Empty(data.CommitURL)
	suite.Empty(data.RepositoryOwner)
	suite.Empty(data.RepositoryName)
}

// TestBuildReportDataFileReports tests file report generation
func (suite *GeneratorTestSuite) TestBuildReportDataFileReports() {
	ctx := context.Background()
	generator := NewGenerator(suite.config)
	coverageData := suite.createDetailedCoverageData()

	data := generator.buildReportData(ctx, coverageData)
	suite.Require().NotNil(data)
	suite.Require().NotEmpty(data.Packages)

	// Find the package with files
	var packageWithFiles *PackageReport
	for _, pkg := range data.Packages {
		if len(pkg.Files) > 0 {
			packageWithFiles = &pkg
			break
		}
	}

	suite.Require().NotNil(packageWithFiles, "Should have at least one package with files")
	suite.Require().NotEmpty(packageWithFiles.Files)

	// Verify file reports are properly generated
	for _, fileReport := range packageWithFiles.Files {
		suite.NotEmpty(fileReport.Name)
		suite.NotEmpty(fileReport.Path)
		suite.GreaterOrEqual(fileReport.Percentage, 0.0)
		suite.LessOrEqual(fileReport.Percentage, 100.0)
		suite.GreaterOrEqual(fileReport.TotalLines, 0)
		suite.GreaterOrEqual(fileReport.CoveredLines, 0)
		suite.LessOrEqual(fileReport.CoveredLines, fileReport.TotalLines)
	}

	// Verify files are sorted within packages
	if len(packageWithFiles.Files) > 1 {
		for i := 1; i < len(packageWithFiles.Files); i++ {
			suite.LessOrEqual(packageWithFiles.Files[i-1].Name, packageWithFiles.Files[i].Name,
				"Files should be sorted alphabetically within package")
		}
	}
}

// TestGenerateReportContent tests that the generated report contains expected content
func (suite *GeneratorTestSuite) TestGenerateReportContent() {
	ctx := context.Background()
	generator := NewGenerator(suite.config)
	coverageData := suite.createSampleCoverageData()

	err := generator.Generate(ctx, coverageData)
	suite.Require().NoError(err)

	// Read the generated report
	reportPath := filepath.Join(suite.tempDir, "coverage.html")
	content, err := os.ReadFile(reportPath) // #nosec G304 - test reads from known temp directory
	suite.Require().NoError(err)

	contentStr := string(content)

	// Verify essential HTML structure
	suite.Contains(contentStr, "<!DOCTYPE html>")
	suite.Contains(contentStr, "<html")
	suite.Contains(contentStr, "</html>")
	suite.Contains(contentStr, "<head>")
	suite.Contains(contentStr, "<body>")

	// Verify title contains repository info
	suite.Contains(contentStr, "test-owner/test-repo Coverage Report")

	// Verify coverage percentage is displayed
	suite.Contains(contentStr, "75.0%") // Based on sample data

	// Verify repository information
	suite.Contains(contentStr, "test-owner/test-repo")
	suite.Contains(contentStr, "master")  // branch name
	suite.Contains(contentStr, "abc123d") // truncated commit SHA

	// Verify Google Analytics ID is included
	suite.Contains(contentStr, suite.config.GoogleAnalyticsID)

	// Verify package information
	suite.Contains(contentStr, "test/package1")
	suite.Contains(contentStr, "test/package2")
}

// TestConcurrentGeneration tests concurrent report generation
func (suite *GeneratorTestSuite) TestConcurrentGeneration() {
	const numGoroutines = 5

	errChan := make(chan error, numGoroutines)
	doneChan := make(chan struct{}, numGoroutines)

	coverageData := suite.createSampleCoverageData()

	for i := 0; i < numGoroutines; i++ {
		go func(_ int) {
			defer func() { doneChan <- struct{}{} }()

			// Create separate temp directory for each goroutine
			tempDir, err := os.MkdirTemp("", "concurrent_test_*")
			if err != nil {
				errChan <- err
				return
			}
			defer func() {
				if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
					// Log error but don't fail test in cleanup
					suite.T().Logf("Failed to cleanup temp directory: %v", cleanupErr)
				}
			}()

			config := &Config{
				OutputDir:       tempDir,
				RepositoryOwner: "test-owner",
				RepositoryName:  "test-repo",
				BranchName:      "master",
				CommitSHA:       "abc123def456",
			}

			generator := NewGenerator(config)
			ctx := context.Background()

			err = generator.Generate(ctx, coverageData)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-doneChan
	}

	// Check for errors
	close(errChan)
	for err := range errChan {
		suite.T().Errorf("Concurrent generation error: %v", err)
	}
}

// Helper methods for creating test data

// createSampleCoverageData creates sample coverage data for testing
func (suite *GeneratorTestSuite) createSampleCoverageData() *parser.CoverageData {
	return &parser.CoverageData{
		Packages: map[string]*parser.PackageCoverage{
			"test/package1": {
				Name:         "test/package1",
				Percentage:   80.0,
				TotalLines:   100,
				CoveredLines: 80,
				Files: map[string]*parser.FileCoverage{
					"file1.go": {
						Path: "test/package1/file1.go",
						Statements: []parser.Statement{
							{StartLine: 1, EndLine: 1, Count: 5, NumStmt: 1},
							{StartLine: 2, EndLine: 2, Count: 0, NumStmt: 1},
							{StartLine: 3, EndLine: 3, Count: 3, NumStmt: 1},
						},
					},
				},
			},
			"test/package2": {
				Name:         "test/package2",
				Percentage:   70.0,
				TotalLines:   50,
				CoveredLines: 35,
				Files: map[string]*parser.FileCoverage{
					"file2.go": {
						Path: "test/package2/file2.go",
						Statements: []parser.Statement{
							{StartLine: 1, EndLine: 2, Count: 2, NumStmt: 1},
							{StartLine: 3, EndLine: 3, Count: 0, NumStmt: 1},
						},
					},
				},
			},
		},
		Percentage:   75.0,
		TotalLines:   150,
		CoveredLines: 115,
	}
}

// createDetailedCoverageData creates more detailed coverage data for testing
func (suite *GeneratorTestSuite) createDetailedCoverageData() *parser.CoverageData {
	return &parser.CoverageData{
		Packages: map[string]*parser.PackageCoverage{
			"package/alpha": {
				Name:         "package/alpha",
				Percentage:   85.0,
				TotalLines:   200,
				CoveredLines: 170,
				Files: map[string]*parser.FileCoverage{
					"alpha.go": {
						Path: "package/alpha/alpha.go",
						Statements: []parser.Statement{
							{StartLine: 1, EndLine: 5, Count: 10, NumStmt: 5},
							{StartLine: 6, EndLine: 10, Count: 0, NumStmt: 5},
							{StartLine: 11, EndLine: 15, Count: 5, NumStmt: 5},
						},
					},
					"beta.go": {
						Path: "package/alpha/beta.go",
						Statements: []parser.Statement{
							{StartLine: 1, EndLine: 3, Count: 7, NumStmt: 3},
							{StartLine: 4, EndLine: 6, Count: 2, NumStmt: 3},
						},
					},
				},
			},
			"package/gamma": {
				Name:         "package/gamma",
				Percentage:   60.0,
				TotalLines:   100,
				CoveredLines: 60,
				Files: map[string]*parser.FileCoverage{
					"gamma.go": {
						Path: "package/gamma/gamma.go",
						Statements: []parser.Statement{
							{StartLine: 1, EndLine: 10, Count: 3, NumStmt: 10},
							{StartLine: 11, EndLine: 20, Count: 0, NumStmt: 10},
						},
					},
				},
			},
		},
		Percentage:   75.0,
		TotalLines:   300,
		CoveredLines: 230,
	}
}

// TestRun runs the test suite
func TestGeneratorTestSuite(t *testing.T) {
	suite.Run(t, new(GeneratorTestSuite))
}

// Benchmark tests
func BenchmarkNewGenerator(b *testing.B) {
	config := &Config{
		OutputDir:       "/tmp",
		RepositoryOwner: "test-owner",
		RepositoryName:  "test-repo",
	}

	for i := 0; i < b.N; i++ {
		_ = NewGenerator(config)
	}
}

func BenchmarkBuildReportData(b *testing.B) {
	config := &Config{
		OutputDir:       "/tmp",
		RepositoryOwner: "test-owner",
		RepositoryName:  "test-repo",
		BranchName:      "master",
		CommitSHA:       "abc123def456",
	}

	generator := NewGenerator(config)
	ctx := context.Background()

	// Create sample coverage data
	coverageData := &parser.CoverageData{
		Packages: map[string]*parser.PackageCoverage{
			"test/package": {
				Name:         "test/package",
				Percentage:   75.0,
				TotalLines:   100,
				CoveredLines: 75,
				Files: map[string]*parser.FileCoverage{
					"file.go": {
						Path: "test/package/file.go",
						Statements: []parser.Statement{
							{StartLine: 1, EndLine: 1, Count: 5, NumStmt: 1},
							{StartLine: 2, EndLine: 2, Count: 0, NumStmt: 1},
						},
					},
				},
			},
		},
		Percentage:   75.0,
		TotalLines:   100,
		CoveredLines: 75,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.buildReportData(ctx, coverageData)
	}
}
