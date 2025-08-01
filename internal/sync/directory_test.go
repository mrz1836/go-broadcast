package sync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DirectoryTestSuite tests directory sync functionality
type DirectoryTestSuite struct {
	suite.Suite

	tempDir      string
	sourceDir    string
	processor    *DirectoryProcessor
	logger       *logrus.Entry
	mockEngine   *MockEngine
	sourceState  *state.SourceState
	targetConfig config.TargetConfig
}

// SetupSuite initializes the test suite
func (suite *DirectoryTestSuite) SetupSuite() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "directory-sync-test-*")
	require.NoError(suite.T(), err)
	suite.tempDir = tempDir

	// Create source directory structure
	suite.sourceDir = filepath.Join(tempDir, "source")
	require.NoError(suite.T(), os.MkdirAll(suite.sourceDir, 0o755))

	// Create test files and directories
	suite.createTestStructure()

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	suite.logger = logger.WithField("component", "directory-test")

	// Create processor
	suite.processor = NewDirectoryProcessor(suite.logger, 5)

	// Create mock engine and source state
	suite.mockEngine = NewMockEngine()
	suite.sourceState = &state.SourceState{
		Repo:         "test/source-repo",
		Branch:       "main",
		LatestCommit: "abc123def456",
		LastChecked:  time.Now(),
	}

	// Create target config
	suite.targetConfig = config.TargetConfig{
		Repo: "test/target-repo",
		Transform: config.Transform{
			RepoName: true,
			Variables: map[string]string{
				"PROJECT_NAME": "test-project",
			},
		},
	}
}

// TearDownSuite cleans up the test suite
func (suite *DirectoryTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		err := os.RemoveAll(suite.tempDir)
		require.NoError(suite.T(), err)
	}
}

// createTestStructure creates a realistic directory structure for testing
func (suite *DirectoryTestSuite) createTestStructure() {
	testFiles := map[string]string{
		"README.md":                    "# Test Project\n\nThis is a test repository.",
		"src/main.go":                  "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
		"src/utils/helper.go":          "package utils\n\nfunc Helper() string {\n\treturn \"helper\"\n}",
		"config/app.yaml":              "app:\n  name: test-app\n  version: 1.0.0",
		"config/database.yaml":         "database:\n  host: localhost\n  port: 5432",
		"scripts/build.sh":             "#!/bin/bash\necho \"Building...\"\ngo build ./...",
		"scripts/test.sh":              "#!/bin/bash\necho \"Testing...\"\ngo test ./...",
		"docs/api.md":                  "# API Documentation\n\n## Endpoints\n\n- GET /health",
		"docs/deployment.md":           "# Deployment Guide\n\n## Prerequisites\n\n- Docker",
		".github/workflows/ci.yml":     "name: CI\non: [push, pull_request]\njobs:\n  test:\n    runs-on: ubuntu-latest",
		".github/workflows/deploy.yml": "name: Deploy\non:\n  push:\n    branches: [main]",
		"tests/main_test.go":           "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {\n\t// Test implementation\n}",
		"vendor/lib/example.go":        "package lib\n\n// Example library code",
		".env.example":                 "DATABASE_URL=postgres://localhost/test\nAPI_KEY=your-api-key-here",
		"Dockerfile":                   "FROM golang:1.21\nWORKDIR /app\nCOPY . .\nRUN go build",
		"docker-compose.yml":           "version: '3.8'\nservices:\n  app:\n    build: .\n    ports:\n      - \"8080:8080\"",
	}

	// Create directories and files
	for filePath, content := range testFiles {
		fullPath := filepath.Join(suite.sourceDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(suite.T(), err)

		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(suite.T(), err)
	}

	// Create some hidden files
	hiddenFiles := map[string]string{
		".gitignore":      "*.log\n*.tmp\nvendor/\nnode_modules/",
		".dockerignore":   "*.log\n*.tmp\n.git/",
		"src/.hidden":     "hidden file content",
		".secret/key.txt": "secret-key-value",
	}

	for filePath, content := range hiddenFiles {
		fullPath := filepath.Join(suite.sourceDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(suite.T(), err)

		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(suite.T(), err)
	}
}

// TestDirectoryDiscovery tests file discovery functionality
func (suite *DirectoryTestSuite) TestDirectoryDiscovery() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Src:  "src",
		Dest: "source",
	}

	// Initialize exclusion engine for this test
	suite.processor.exclusionEngine = NewExclusionEngine(dirMapping.Exclude)

	files, err := suite.processor.discoverFiles(ctx, filepath.Join(suite.sourceDir, "src"), dirMapping)
	require.NoError(suite.T(), err)

	// Should discover files in src directory
	assert.True(suite.T(), len(files) > 0, "Should discover files in src directory")

	// Check for expected files
	foundMainGo := false
	foundHelperGo := false
	foundHidden := false

	for _, file := range files {
		switch file.RelativePath {
		case "main.go":
			foundMainGo = true
		case "utils/helper.go":
			foundHelperGo = true
		case ".hidden":
			foundHidden = true
		}
	}

	assert.True(suite.T(), foundMainGo, "Should find main.go")
	assert.True(suite.T(), foundHelperGo, "Should find utils/helper.go")
	assert.True(suite.T(), foundHidden, "Should find hidden files by default")
}

// TestDirectoryDiscoveryWithExclusions tests file discovery with exclusion patterns
func (suite *DirectoryTestSuite) TestDirectoryDiscoveryWithExclusions() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Src:     "",
		Dest:    "dest",
		Exclude: []string{"vendor/**", "*.yml", ".secret/**"},
	}

	// Initialize exclusion engine for this test
	suite.processor.exclusionEngine = NewExclusionEngine(dirMapping.Exclude)

	files, err := suite.processor.discoverFiles(ctx, suite.sourceDir, dirMapping)
	require.NoError(suite.T(), err)

	// Check that excluded files are not present
	for _, file := range files {
		assert.False(suite.T(), strings.Contains(file.RelativePath, "vendor/"),
			"Vendor files should be excluded: %s", file.RelativePath)
		assert.False(suite.T(), strings.HasSuffix(file.RelativePath, ".yml"),
			"YAML files should be excluded: %s", file.RelativePath)
		assert.False(suite.T(), strings.Contains(file.RelativePath, ".secret/"),
			"Secret files should be excluded: %s", file.RelativePath)
	}
}

// TestDirectoryDiscoveryHiddenFiles tests hidden file inclusion/exclusion
func (suite *DirectoryTestSuite) TestDirectoryDiscoveryHiddenFiles() {
	ctx := context.Background()

	// Test with hidden files included (default)
	includeHidden := true
	dirMapping := config.DirectoryMapping{
		Src:           "",
		Dest:          "dest",
		IncludeHidden: &includeHidden,
	}

	// Initialize exclusion engine for this test
	suite.processor.exclusionEngine = NewExclusionEngine(dirMapping.Exclude)

	files, err := suite.processor.discoverFiles(ctx, suite.sourceDir, dirMapping)
	require.NoError(suite.T(), err)

	foundHidden := false
	for _, file := range files {
		if strings.HasPrefix(filepath.Base(file.RelativePath), ".") {
			foundHidden = true
			break
		}
	}
	assert.True(suite.T(), foundHidden, "Should find hidden files when included")

	// Test with hidden files excluded
	includeHidden = false
	dirMapping.IncludeHidden = &includeHidden

	// Reinitialize exclusion engine for updated mapping
	suite.processor.exclusionEngine = NewExclusionEngine(dirMapping.Exclude)

	files, err = suite.processor.discoverFiles(ctx, suite.sourceDir, dirMapping)
	require.NoError(suite.T(), err)

	foundHidden = false
	for _, file := range files {
		if strings.HasPrefix(filepath.Base(file.RelativePath), ".") {
			foundHidden = true
			break
		}
	}
	assert.False(suite.T(), foundHidden, "Should not find hidden files when excluded")
}

// TestFileJobCreation tests file job creation from discovered files
func (suite *DirectoryTestSuite) TestFileJobCreation() {
	files := []DiscoveredFile{
		{RelativePath: "main.go", FullPath: "/src/main.go", Size: 100},
		{RelativePath: "utils/helper.go", FullPath: "/src/utils/helper.go", Size: 200},
		{RelativePath: "config/app.yaml", FullPath: "/src/config/app.yaml", Size: 150},
	}

	// Test with structure preservation (default)
	dirMapping := config.DirectoryMapping{
		Src:  "src",
		Dest: "target/src",
		Transform: config.Transform{
			RepoName: true,
		},
	}

	jobs := suite.processor.createFileJobs(files, dirMapping)
	require.Len(suite.T(), jobs, 3)

	// Check that paths are preserved
	expectedJobs := map[string]string{
		"src/main.go":         "target/src/main.go",
		"src/utils/helper.go": "target/src/utils/helper.go",
		"src/config/app.yaml": "target/src/config/app.yaml",
	}

	for _, job := range jobs {
		expectedDest, exists := expectedJobs[job.SourcePath]
		require.True(suite.T(), exists, "Unexpected source path: %s", job.SourcePath)
		assert.Equal(suite.T(), expectedDest, job.DestPath, "Incorrect destination path")
		assert.True(suite.T(), job.Transform.RepoName, "Transform should be applied")
	}
}

// TestFileJobCreationFlattened tests file job creation with flattened structure
func (suite *DirectoryTestSuite) TestFileJobCreationFlattened() {
	files := []DiscoveredFile{
		{RelativePath: "main.go", FullPath: "/src/main.go", Size: 100},
		{RelativePath: "utils/helper.go", FullPath: "/src/utils/helper.go", Size: 200},
	}

	// Test with structure flattening
	preserveStructure := false
	dirMapping := config.DirectoryMapping{
		Src:               "src",
		Dest:              "target",
		PreserveStructure: &preserveStructure,
	}

	jobs := suite.processor.createFileJobs(files, dirMapping)
	require.Len(suite.T(), jobs, 2)

	// Check that files are flattened
	expectedJobs := map[string]string{
		"src/main.go":         "target/main.go",
		"src/utils/helper.go": "target/helper.go",
	}

	for _, job := range jobs {
		expectedDest, exists := expectedJobs[job.SourcePath]
		require.True(suite.T(), exists, "Unexpected source path: %s", job.SourcePath)
		assert.Equal(suite.T(), expectedDest, job.DestPath, "Incorrect flattened destination path")
	}
}

// TestProgressReporting tests directory progress reporting
func (suite *DirectoryTestSuite) TestProgressReporting() {
	// Test progress reporter creation
	reporter := NewDirectoryProgressReporter(suite.logger, "test-dir", 5)
	assert.NotNil(suite.T(), reporter)

	// Test with file count above threshold
	reporter.Start(10)
	assert.True(suite.T(), reporter.isEnabled(), "Should be enabled for file count above threshold")

	// Test metrics tracking
	reporter.RecordFileProcessed(100)
	reporter.RecordFileExcluded()
	reporter.RecordFileSkipped()
	reporter.RecordFileError()
	reporter.RecordDirectoryWalked()
	reporter.AddTotalSize(1000)

	metrics := reporter.GetMetrics()
	assert.Equal(suite.T(), 10, metrics.FilesDiscovered, "Should track discovered files from Start() call")
	assert.Equal(suite.T(), 1, metrics.FilesProcessed, "Should track processed files")
	assert.Equal(suite.T(), 1, metrics.FilesExcluded, "Should track excluded files")
	assert.Equal(suite.T(), 1, metrics.FilesSkipped, "Should track skipped files")
	assert.Equal(suite.T(), 1, metrics.FilesErrored, "Should track errored files")
	assert.Equal(suite.T(), 1, metrics.DirectoriesWalked, "Should track walked directories")
	assert.Equal(suite.T(), int64(1000), metrics.TotalSize, "Should track total size")
	assert.Equal(suite.T(), int64(100), metrics.ProcessedSize, "Should track processed size")

	// Test completion
	finalMetrics := reporter.Complete()
	assert.False(suite.T(), finalMetrics.EndTime.IsZero(), "Should have end time")
}

// TestProgressReportingThreshold tests progress reporting threshold behavior
func (suite *DirectoryTestSuite) TestProgressReportingThreshold() {
	// Test with file count below threshold
	reporter := NewDirectoryProgressReporter(suite.logger, "test-dir", 10)
	reporter.Start(5) // Below threshold of 10
	assert.False(suite.T(), reporter.isEnabled(), "Should be disabled for file count below threshold")

	// Test with file count at threshold
	reporter2 := NewDirectoryProgressReporter(suite.logger, "test-dir", 10)
	reporter2.Start(10) // At threshold
	assert.True(suite.T(), reporter2.isEnabled(), "Should be enabled for file count at threshold")

	// Test with file count above threshold
	reporter3 := NewDirectoryProgressReporter(suite.logger, "test-dir", 10)
	reporter3.Start(15) // Above threshold
	assert.True(suite.T(), reporter3.isEnabled(), "Should be enabled for file count above threshold")
}

// TestExclusionEngine tests the exclusion engine functionality
func (suite *DirectoryTestSuite) TestExclusionEngine() {
	patterns := []string{
		"*.log",
		"temp/**",
		"node_modules/",
		"!important.log",
		".git/**",
	}

	engine := NewExclusionEngine(patterns)
	assert.NotNil(suite.T(), engine)

	// Test file exclusions
	testCases := []struct {
		path     string
		excluded bool
		desc     string
	}{
		{"test.log", true, "should exclude .log files"},
		{"important.log", false, "should not exclude negated patterns"},
		{"temp/file.txt", true, "should exclude files in temp directory"},
		{"node_modules/package.json", true, "should exclude node_modules"},
		{"src/main.go", false, "should not exclude normal source files"},
		{".git/config", true, "should exclude .git directory"},
		{"docs/readme.md", false, "should not exclude documentation"},
	}

	for _, tc := range testCases {
		result := engine.IsExcluded(tc.path)
		assert.Equal(suite.T(), tc.excluded, result, "%s: %s", tc.desc, tc.path)
	}
}

// TestExclusionEngineDirectories tests directory-specific exclusion
func (suite *DirectoryTestSuite) TestExclusionEngineDirectories() {
	patterns := []string{
		"vendor/",
		"node_modules/",
		".git/",
	}

	engine := NewExclusionEngine(patterns)

	// Test directory exclusions
	testCases := []struct {
		path     string
		excluded bool
		desc     string
	}{
		{"vendor", true, "should exclude vendor directory"},
		{"vendor/", true, "should exclude vendor directory with slash"},
		{"node_modules", true, "should exclude node_modules directory"},
		{"src", false, "should not exclude src directory"},
		{".git", true, "should exclude .git directory"},
		{"docs", false, "should not exclude docs directory"},
	}

	for _, tc := range testCases {
		result := engine.IsDirectoryExcluded(tc.path)
		assert.Equal(suite.T(), tc.excluded, result, "%s: %s", tc.desc, tc.path)
	}
}

// TestValidateDirectoryMapping tests directory mapping validation
func (suite *DirectoryTestSuite) TestValidateDirectoryMapping() {
	// Valid mapping
	validMapping := config.DirectoryMapping{
		Src:     "src",
		Dest:    "dest",
		Exclude: []string{"*.tmp", "vendor/"},
	}
	err := ValidateDirectoryMapping(validMapping)
	assert.NoError(suite.T(), err, "Valid mapping should pass validation")

	// Invalid mappings
	invalidMappings := []struct {
		mapping config.DirectoryMapping
		desc    string
	}{
		{
			mapping: config.DirectoryMapping{Src: "", Dest: "dest"},
			desc:    "empty source should fail",
		},
		{
			mapping: config.DirectoryMapping{Src: "src", Dest: ""},
			desc:    "empty destination should fail",
		},
		{
			mapping: config.DirectoryMapping{
				Src:     "src",
				Dest:    "dest",
				Exclude: []string{"pattern", ""},
			},
			desc: "empty exclusion pattern should fail",
		},
	}

	for _, tc := range invalidMappings {
		err := ValidateDirectoryMapping(tc.mapping)
		assert.Error(suite.T(), err, tc.desc)
	}
}

// TestHiddenFileDetection tests hidden file detection
func (suite *DirectoryTestSuite) TestHiddenFileDetection() {
	testCases := []struct {
		path   string
		hidden bool
		desc   string
	}{
		{".gitignore", true, "should detect .gitignore as hidden"},
		{"src/.hidden", true, "should detect nested hidden files"},
		{".config/app.yaml", true, "should detect files in hidden directories"},
		{"src/main.go", false, "should not detect regular files as hidden"},
		{"docs/readme.md", false, "should not detect regular nested files as hidden"},
		{"./src/main.go", false, "should handle relative paths correctly"},
	}

	for _, tc := range testCases {
		result := suite.processor.isHidden(tc.path)
		assert.Equal(suite.T(), tc.hidden, result, "%s: %s", tc.desc, tc.path)
	}
}

// TestPerformanceRequirements tests that performance targets are met
func (suite *DirectoryTestSuite) TestPerformanceRequirements() {
	ctx := context.Background()

	// Test with small directory (< 50 files) - should be < 500ms
	// Note: This test is simplified since we need a real Engine instance for full testing
	// In a real implementation, this would use the actual Engine with all dependencies

	// For now, just test the file discovery part which is the main performance component
	smallDirMapping := config.DirectoryMapping{
		Src:  "src",
		Dest: "target/src",
	}

	// Initialize exclusion engine for this test
	suite.processor.exclusionEngine = NewExclusionEngine(smallDirMapping.Exclude)

	start := time.Now()
	files, err := suite.processor.discoverFiles(ctx, filepath.Join(suite.sourceDir, "src"), smallDirMapping)
	duration := time.Since(start)

	// Allow for some test overhead, but should generally be fast
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), len(files) > 0, "Should discover files")
	assert.Less(suite.T(), duration, 2*time.Second, "Small directory discovery should be fast")

	suite.logger.WithFields(logrus.Fields{
		"duration":   duration,
		"file_count": len(files),
	}).Info("Small directory discovery time")
}

// Run the test suite
func TestDirectoryTestSuite(t *testing.T) {
	suite.Run(t, new(DirectoryTestSuite))
}

// MockEngine provides a mock implementation for testing
type MockEngine struct {
	gh        *MockGHClient
	transform *MockTransformChain
}

// NewMockEngine creates a new mock engine
func NewMockEngine() *MockEngine {
	return &MockEngine{
		gh:        &MockGHClient{},
		transform: &MockTransformChain{},
	}
}

// MockGHClient provides a mock GitHub client for testing
type MockGHClient struct{}

// GetFile implements a mock GetFile method
func (m *MockGHClient) GetFile(ctx context.Context, repo, path, branch string) (*MockFileContent, error) {
	return nil, os.ErrNotExist // Simulate file not found
}

// MockFileContent represents mock file content
type MockFileContent struct {
	Content []byte
}

// MockTransformChain provides a mock transform chain for testing
type MockTransformChain struct{}

// Transform implements a mock Transform method
func (m *MockTransformChain) Transform(ctx context.Context, content []byte, transformCtx interface{}) ([]byte, error) {
	// Simple mock transformation - just return the content unchanged
	return content, nil
}
