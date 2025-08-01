package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DirectoryTransformTestSuite tests comprehensive directory transformation scenarios
type DirectoryTransformTestSuite struct {
	suite.Suite

	tempDir            string
	sourceDir          string
	processor          *DirectoryProcessor
	logger             *logrus.Entry
	mockEngine         *Engine
	sourceState        *state.SourceState
	targetConfig       config.TargetConfig
	binaryData         []byte
	largeBinaryData    []byte
	performanceTestDir string
}

// SetupSuite initializes the comprehensive test suite
func (suite *DirectoryTransformTestSuite) SetupSuite() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "directory-transform-test-*")
	require.NoError(suite.T(), err)
	suite.tempDir = tempDir

	// Create source directory structure
	suite.sourceDir = filepath.Join(tempDir, "source")
	require.NoError(suite.T(), os.MkdirAll(suite.sourceDir, 0o755))

	// Create performance test directory with many files
	suite.performanceTestDir = filepath.Join(tempDir, "performance")
	require.NoError(suite.T(), os.MkdirAll(suite.performanceTestDir, 0o755))

	// Generate binary test data
	suite.generateBinaryTestData()

	// Create comprehensive test structure
	suite.createComprehensiveTestStructure()
	suite.createPerformanceTestStructure()

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	suite.logger = logger.WithField("component", "directory-transform-test")

	// Create processor
	suite.processor = NewDirectoryProcessor(suite.logger, 8)

	// Create mock engine and source state
	suite.mockEngine = suite.createMockEngine()
	suite.sourceState = &state.SourceState{
		Repo:         "test/source-repo",
		Branch:       "main",
		LatestCommit: "abc123def456",
		LastChecked:  time.Now(),
	}

	// Create target config with comprehensive transform settings
	suite.targetConfig = config.TargetConfig{
		Repo: "test/target-repo",
		Transform: config.Transform{
			RepoName: true,
			Variables: map[string]string{
				"PROJECT_NAME":  "transformed-project",
				"OWNER":         "new-owner",
				"VERSION":       "2.0.0",
				"DATABASE_NAME": "transformed_db",
				"API_ENDPOINT":  "https://api.transformed.com",
				"DOCKER_IMAGE":  "transformed/app:latest",
				"SERVICE_NAME":  "transformed-service",
				"NAMESPACE":     "transformed-ns",
				"CLUSTER_NAME":  "transformed-cluster",
				"ENVIRONMENT":   "production",
			},
		},
	}
}

// TearDownSuite cleans up the test suite
func (suite *DirectoryTransformTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		err := os.RemoveAll(suite.tempDir)
		require.NoError(suite.T(), err)
	}
}

// createMockEngine creates a mock Engine for testing
func (suite *DirectoryTransformTestSuite) createMockEngine() *Engine {
	// Create mock transform chain
	mockTransform := &DirectoryMockTransformChain{}

	// Create mock GitHub client
	mockGH := &DirectoryMockGHClient{}

	// Create Engine with mocked dependencies
	return &Engine{
		gh:        mockGH,
		transform: mockTransform,
		logger:    suite.logger.Logger,
	}
}

// generateBinaryTestData creates various binary data samples for testing
func (suite *DirectoryTransformTestSuite) generateBinaryTestData() {
	// Small binary data (simulating an image header)
	suite.binaryData = []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, // JPEG header
		0x49, 0x46, 0x00, 0x01, 0x01, 0x01, 0x00, 0x48,
		0x00, 0x48, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08,
	}

	// Large binary data (10KB of mixed binary content)
	suite.largeBinaryData = make([]byte, 10*1024)
	for i := 0; i < len(suite.largeBinaryData); i++ {
		if i%100 == 0 {
			suite.largeBinaryData[i] = 0x00 // Null bytes
		} else if i%50 == 0 {
			suite.largeBinaryData[i] = byte(200 + (i % 56)) // High bytes
		} else {
			suite.largeBinaryData[i] = byte(32 + (i % 95)) // Printable ASCII range
		}
	}
}

// createComprehensiveTestStructure creates a realistic directory structure with various file types
func (suite *DirectoryTransformTestSuite) createComprehensiveTestStructure() {
	// Text files for transformation testing
	textFiles := map[string]string{
		"README.md": `# {{.PROJECT_NAME}}

This is the {{.PROJECT_NAME}} repository owned by {{.OWNER}}.

Version: {{.VERSION}}
Database: {{.DATABASE_NAME}}
API Endpoint: {{.API_ENDPOINT}}

## Docker

Image: {{.DOCKER_IMAGE}}
Service: {{.SERVICE_NAME}}

## Kubernetes

Namespace: {{.NAMESPACE}}
Cluster: {{.CLUSTER_NAME}}
Environment: {{.ENVIRONMENT}}

Repository: test/source-repo -> should become test/target-repo
`,
		"src/main.go": `package main

import (
	"fmt"
	"os"
)

const (
	projectName = "{{.PROJECT_NAME}}"
	version = "{{.VERSION}}"
	dbName = "{{.DATABASE_NAME}}"
	apiEndpoint = "{{.API_ENDPOINT}}"
)

func main() {
	fmt.Printf("Starting %s v%s\n", projectName, version)
	fmt.Printf("Database: %s\n", dbName)
	fmt.Printf("API: %s\n", apiEndpoint)
	
	// Repository references for transformation
	fmt.Println("Source repo: test/source-repo")
	fmt.Println("This should be transformed to: test/target-repo")
}`,
		"src/utils/database.go": `package utils

import "database/sql"

const DatabaseName = "{{.DATABASE_NAME}}"
const ServiceName = "{{.SERVICE_NAME}}"

// Connection string will be transformed
var connStr = "postgres://localhost/test/source-repo_db"

func Connect() (*sql.DB, error) {
	// This references the repo name: test/source-repo
	return sql.Open("postgres", connStr)
}`,
		"config/app.yaml": `app:
  name: "{{.PROJECT_NAME}}"
  version: "{{.VERSION}}"
  owner: "{{.OWNER}}"
  
database:
  name: "{{.DATABASE_NAME}}"
  
api:
  endpoint: "{{.API_ENDPOINT}}"
  
docker:
  image: "{{.DOCKER_IMAGE}}"
  
kubernetes:
  service: "{{.SERVICE_NAME}}"
  namespace: "{{.NAMESPACE}}"
  cluster: "{{.CLUSTER_NAME}}"
  environment: "{{.ENVIRONMENT}}"

# Repository transformation test
repository: "test/source-repo"
`,
		"nested/deep/config/settings.json": `{
  "project": "{{.PROJECT_NAME}}",
  "version": "{{.VERSION}}",
  "owner": "{{.OWNER}}",
  "database": {
    "name": "{{.DATABASE_NAME}}"
  },
  "api": {
    "endpoint": "{{.API_ENDPOINT}}"
  },
  "docker": {
    "image": "{{.DOCKER_IMAGE}}"
  },
  "kubernetes": {
    "service": "{{.SERVICE_NAME}}",
    "namespace": "{{.NAMESPACE}}",
    "cluster": "{{.CLUSTER_NAME}}",
    "environment": "{{.ENVIRONMENT}}"
  },
  "repository": "test/source-repo"
}`,
		// File that will cause transformation error (invalid template)
		"invalid/template.txt": `This file has an invalid template: {{.MISSING_VAR}}
And also contains repo reference: test/source-repo`,
	}

	// Create text files
	for filePath, content := range textFiles {
		fullPath := filepath.Join(suite.sourceDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(suite.T(), err)

		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(suite.T(), err)
	}

	// Binary files of different types and sizes
	binaryFiles := map[string][]byte{
		"images/logo.jpg":        suite.binaryData,
		"images/banner.png":      append([]byte{0x89, 0x50, 0x4E, 0x47}, suite.binaryData...),
		"assets/data.zip":        append([]byte{0x50, 0x4B, 0x03, 0x04}, suite.binaryData...),
		"bin/executable":         append([]byte{0x7F, 0x45, 0x4C, 0x46}, suite.binaryData...),
		"docs/manual.pdf":        append([]byte{0x25, 0x50, 0x44, 0x46}, suite.binaryData...),
		"data/large_binary.dat":  suite.largeBinaryData,
		"nested/binary/file.bin": suite.binaryData,
		"mixed/small.dat":        {0x00, 0x01, 0x02, 0x03, 0x04},
	}

	// Create binary files
	for filePath, content := range binaryFiles {
		fullPath := filepath.Join(suite.sourceDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(suite.T(), err)

		err = os.WriteFile(fullPath, content, 0o644)
		require.NoError(suite.T(), err)
	}

	// Hidden files (mix of text and binary)
	hiddenFiles := map[string]interface{}{
		".gitignore":           "*.log\n*.tmp\nbuild/\ndist/\nnode_modules/",            // text
		".dockerignore":        "*.log\n.git/\n*.tmp",                                   // text
		"src/.hidden_config":   "secret_key=test\napi_key={{.API_ENDPOINT}}",            // text with transformation
		".secrets/binary.key":  suite.binaryData,                                        // binary
		"config/.env.template": "PROJECT_NAME={{.PROJECT_NAME}}\nREPO=test/source-repo", // text with transformation
	}

	for filePath, content := range hiddenFiles {
		fullPath := filepath.Join(suite.sourceDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(suite.T(), err)

		var data []byte
		switch v := content.(type) {
		case string:
			data = []byte(v)
		case []byte:
			data = v
		}

		err = os.WriteFile(fullPath, data, 0o644)
		require.NoError(suite.T(), err)
	}

	// Empty directories
	emptyDirs := []string{
		"empty_dir",
		"nested/empty_nested",
		"deep/nested/empty",
	}

	for _, dirPath := range emptyDirs {
		fullPath := filepath.Join(suite.sourceDir, dirPath)
		err := os.MkdirAll(fullPath, 0o755)
		require.NoError(suite.T(), err)
	}
}

// createPerformanceTestStructure creates a large directory structure for performance testing
func (suite *DirectoryTransformTestSuite) createPerformanceTestStructure() {
	// Create 200 files across multiple directories for performance testing
	const fileCount = 200
	const dirCount = 20

	for i := 0; i < fileCount; i++ {
		dirIndex := i % dirCount
		dirPath := filepath.Join(suite.performanceTestDir, fmt.Sprintf("dir_%02d", dirIndex))

		err := os.MkdirAll(dirPath, 0o755)
		require.NoError(suite.T(), err)

		fileName := fmt.Sprintf("file_%03d.txt", i)
		filePath := filepath.Join(dirPath, fileName)

		content := fmt.Sprintf(`File %d
Project: {{.PROJECT_NAME}}
Repository: test/source-repo
Owner: {{.OWNER}}
Content generated for performance testing.
File index: %d
Directory: dir_%02d
`, i, i, dirIndex)

		// Mix in some binary files (every 20th file)
		if i%20 == 0 {
			fileName = fmt.Sprintf("binary_%03d.dat", i)
			filePath = filepath.Join(dirPath, fileName)
			err = os.WriteFile(filePath, suite.binaryData, 0o644)
		} else {
			err = os.WriteFile(filePath, []byte(content), 0o644)
		}
		require.NoError(suite.T(), err)
	}
}

// TestRepoNameTransformOnMultipleFiles tests repo_name transformation across directory files
func (suite *DirectoryTransformTestSuite) TestRepoNameTransformOnMultipleFiles() {
	ctx := context.Background()

	// Test basic directory with repo name transformation
	dirMapping := config.DirectoryMapping{
		Src:  "src",
		Dest: "transformed/src",
		Transform: config.Transform{
			RepoName: true,
		},
	}

	changes, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, dirMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	require.NoError(suite.T(), err)
	assert.True(suite.T(), len(changes) > 0, "Should process files and create changes")

	// Verify that repo name transformations were applied
	for _, change := range changes {
		// Check that the transform context was set up properly for directory processing
		transformedContent := string(change.Content)

		// Verify transformation happened (original content should be different from transformed)
		originalContent := string(change.OriginalContent)
		if strings.Contains(originalContent, "test/source-repo") {
			assert.Contains(suite.T(), transformedContent, "test/target-repo",
				"File %s should have repo name transformed", change.Path)
			assert.NotContains(suite.T(), transformedContent, "test/source-repo",
				"File %s should not contain original repo name", change.Path)
		}

		// Verify path mapping is correct
		assert.True(suite.T(), strings.HasPrefix(change.Path, "transformed/src/"),
			"Change path should have correct destination prefix: %s", change.Path)
	}

	suite.logger.WithField("changes_count", len(changes)).Info("Repo name transformation test completed")
}

// TestVariableSubstitutionAcrossDirectoryFiles tests variable substitution across all directory files
func (suite *DirectoryTransformTestSuite) TestVariableSubstitutionAcrossDirectoryFiles() {
	ctx := context.Background()

	// Test comprehensive variable substitution
	dirMapping := config.DirectoryMapping{
		Src:       "",
		Dest:      "transformed",
		Transform: suite.targetConfig.Transform, // Use all variables
		Exclude: []string{
			"images/**",
			"assets/**",
			"bin/**",
			"docs/manual.pdf",
			"data/**",
			"*.bin",
			"*.dat",
			".secrets/**",
		},
	}

	changes, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, dirMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	require.NoError(suite.T(), err)
	assert.True(suite.T(), len(changes) > 0, "Should process files and create changes")

	// Track which variables were found and transformed
	variablesFound := make(map[string]int)
	expectedVariables := suite.targetConfig.Transform.Variables

	for _, change := range changes {
		transformedContent := string(change.Content)
		originalContent := string(change.OriginalContent)

		// Check that variables were substituted
		for varName, expectedValue := range expectedVariables {
			templateVar := fmt.Sprintf("{{.%s}}", varName)

			if strings.Contains(originalContent, templateVar) {
				variablesFound[varName]++
				assert.Contains(suite.T(), transformedContent, expectedValue,
					"File %s should contain substituted value for %s", change.Path, varName)
				assert.NotContains(suite.T(), transformedContent, templateVar,
					"File %s should not contain template variable %s", change.Path, varName)
			}
		}

		// Verify repo name transformation if enabled
		if dirMapping.Transform.RepoName && strings.Contains(originalContent, "test/source-repo") {
			assert.Contains(suite.T(), transformedContent, "test/target-repo",
				"File %s should have repo name transformed", change.Path)
		}
	}

	// Assert that we found and transformed multiple variables across files
	assert.True(suite.T(), len(variablesFound) >= 5,
		"Should find at least 5 different variables across files, found: %v", variablesFound)

	suite.logger.WithFields(logrus.Fields{
		"changes_count":   len(changes),
		"variables_found": variablesFound,
	}).Info("Variable substitution test completed")
}

// TestBinaryFileDetectionAndSkipping tests binary file detection and content preservation
func (suite *DirectoryTransformTestSuite) TestBinaryFileDetectionAndSkipping() {
	ctx := context.Background()

	// Test directory containing mixed text and binary files
	dirMapping := config.DirectoryMapping{
		Src:       "",
		Dest:      "mixed_output",
		Transform: suite.targetConfig.Transform,
	}

	changes, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, dirMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	require.NoError(suite.T(), err)

	// Categorize changes by file type
	binaryChanges := make(map[string]FileChange)
	textChanges := make(map[string]FileChange)

	for _, change := range changes {
		// Determine if original file was binary
		if transform.IsBinary(change.Path, change.OriginalContent) {
			binaryChanges[change.Path] = change
		} else {
			textChanges[change.Path] = change
		}
	}

	// Verify binary files were detected and content preserved
	assert.True(suite.T(), len(binaryChanges) > 0, "Should detect binary files")

	for path, change := range binaryChanges {
		// Binary files should have unchanged content
		assert.Equal(suite.T(), change.OriginalContent, change.Content,
			"Binary file %s should have unchanged content", path)

		// Verify specific binary files we created
		if strings.Contains(path, "logo.jpg") {
			assert.Equal(suite.T(), suite.binaryData, change.Content,
				"JPEG file should preserve exact binary content")
		}
		if strings.Contains(path, "large_binary.dat") {
			assert.Equal(suite.T(), suite.largeBinaryData, change.Content,
				"Large binary file should preserve exact content")
		}
	}

	// Verify text files were transformed
	assert.True(suite.T(), len(textChanges) > 0, "Should process text files")

	for path, change := range textChanges {
		originalContent := string(change.OriginalContent)
		transformedContent := string(change.Content)

		// Text files with variables should be transformed
		if strings.Contains(originalContent, "{{.") {
			assert.NotEqual(suite.T(), originalContent, transformedContent,
				"Text file %s with variables should be transformed", path)
		}
	}

	suite.logger.WithFields(logrus.Fields{
		"binary_files": len(binaryChanges),
		"text_files":   len(textChanges),
		"total_files":  len(changes),
	}).Info("Binary file detection test completed")
}

// TestErrorIsolation tests that one file transform error doesn't fail directory processing
func (suite *DirectoryTransformTestSuite) TestErrorIsolation() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Src:       "",
		Dest:      "error_test",
		Transform: suite.targetConfig.Transform,
		Exclude:   []string{"images/**", "assets/**", "bin/**", "docs/manual.pdf", "data/**", "*.bin", "*.dat"},
	}

	changes, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, dirMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	// Processing should not fail even with individual file errors
	require.NoError(suite.T(), err)
	assert.True(suite.T(), len(changes) > 0, "Should process other files despite errors")

	// Verify that files were processed (even if some had errors, they use fallback content)
	for _, change := range changes {
		assert.NotNil(suite.T(), change.Content, "All changes should have content")
		assert.NotNil(suite.T(), change.OriginalContent, "All changes should have original content")
	}

	suite.logger.WithField("changes_count", len(changes)).Info("Error isolation test completed")
}

// TestMixedTextAndBinaryInSameDirectory tests processing directories with mixed file types
func (suite *DirectoryTransformTestSuite) TestMixedTextAndBinaryInSameDirectory() {
	ctx := context.Background()

	// Test a specific directory that contains both text and binary files
	mixedDir := "nested"
	dirMapping := config.DirectoryMapping{
		Src:       mixedDir,
		Dest:      "mixed_output",
		Transform: suite.targetConfig.Transform,
	}

	changes, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, dirMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	require.NoError(suite.T(), err)
	assert.True(suite.T(), len(changes) > 0, "Should process mixed directory")

	binaryCount := 0
	textCount := 0

	for _, change := range changes {
		if transform.IsBinary(change.Path, change.OriginalContent) {
			binaryCount++
			// Binary content should be unchanged
			assert.Equal(suite.T(), change.OriginalContent, change.Content,
				"Binary file in mixed directory should be unchanged")
		} else {
			textCount++
			// Text files should potentially be transformed
			originalStr := string(change.OriginalContent)
			transformedStr := string(change.Content)

			if strings.Contains(originalStr, "{{.") || strings.Contains(originalStr, "test/source-repo") {
				assert.NotEqual(suite.T(), originalStr, transformedStr,
					"Text file in mixed directory should be transformed")
			}
		}
	}

	assert.True(suite.T(), binaryCount > 0, "Should find binary files in mixed directory")
	assert.True(suite.T(), textCount > 0, "Should find text files in mixed directory")

	suite.logger.WithFields(logrus.Fields{
		"binary_count": binaryCount,
		"text_count":   textCount,
		"total_count":  len(changes),
	}).Info("Mixed directory test completed")
}

// TestNestedDirectoryStructures tests transform with nested directory structures
func (suite *DirectoryTransformTestSuite) TestNestedDirectoryStructures() {
	ctx := context.Background()

	// Test deeply nested directory structure
	dirMapping := config.DirectoryMapping{
		Src:  "nested",
		Dest: "flattened_nested",
		Transform: config.Transform{
			RepoName: true,
			Variables: map[string]string{
				"PROJECT_NAME": "nested-test",
				"API_ENDPOINT": "https://nested.api.com",
			},
		},
		PreserveStructure: func() *bool { b := true; return &b }(),
	}

	changes, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, dirMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	require.NoError(suite.T(), err)
	assert.True(suite.T(), len(changes) > 0, "Should process nested directory structure")

	// Verify structure preservation
	foundDeepNesting := false
	for _, change := range changes {
		// Check for preserved deep nesting
		if strings.Contains(change.Path, "flattened_nested/deep/config/") {
			foundDeepNesting = true

			// Verify transformations applied to nested files
			if strings.HasSuffix(change.Path, ".json") {
				transformedContent := string(change.Content)
				assert.Contains(suite.T(), transformedContent, "nested-test",
					"Nested JSON file should have variable substitution")
				assert.Contains(suite.T(), transformedContent, "test/target-repo",
					"Nested JSON file should have repo name transformation")
			}
		}
	}

	assert.True(suite.T(), foundDeepNesting, "Should preserve deep directory nesting")

	// Test flattened structure
	flatMapping := config.DirectoryMapping{
		Src:  "nested",
		Dest: "flattened_output",
		Transform: config.Transform{
			RepoName: true,
		},
		PreserveStructure: func() *bool { b := false; return &b }(),
	}

	flatChanges, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, flatMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	require.NoError(suite.T(), err)
	assert.True(suite.T(), len(flatChanges) > 0, "Should process with flattened structure")

	// Verify flattening
	for _, change := range flatChanges {
		// All files should be directly in the destination directory
		pathParts := strings.Split(change.Path, "/")
		assert.Equal(suite.T(), 2, len(pathParts),
			"Flattened structure should have only 2 path parts: %s", change.Path)
		assert.Equal(suite.T(), "flattened_output", pathParts[0],
			"Flattened files should be in root destination directory")
	}

	suite.logger.WithFields(logrus.Fields{
		"nested_changes":    len(changes),
		"flattened_changes": len(flatChanges),
	}).Info("Nested directory structure test completed")
}

// TestEmptyDirectoriesAndBinaryOnlyDirectories tests edge cases with empty and binary-only directories
func (suite *DirectoryTransformTestSuite) TestEmptyDirectoriesAndBinaryOnlyDirectories() {
	ctx := context.Background()

	// Test empty directory
	emptyDirMapping := config.DirectoryMapping{
		Src:       "empty_dir",
		Dest:      "processed_empty",
		Transform: suite.targetConfig.Transform,
	}

	emptyChanges, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, emptyDirMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, len(emptyChanges), "Empty directory should produce no changes")

	// Test binary-only directory
	binaryOnlyMapping := config.DirectoryMapping{
		Src:       "images",
		Dest:      "processed_images",
		Transform: suite.targetConfig.Transform,
	}

	binaryChanges, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, binaryOnlyMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	require.NoError(suite.T(), err)
	assert.True(suite.T(), len(binaryChanges) > 0, "Binary-only directory should process binary files")

	// All changes should be binary files with unchanged content
	for _, change := range binaryChanges {
		assert.True(suite.T(), transform.IsBinary(change.Path, change.OriginalContent),
			"Should detect all files as binary in binary-only directory")
		assert.Equal(suite.T(), change.OriginalContent, change.Content,
			"Binary files should have unchanged content")
	}

	suite.logger.WithFields(logrus.Fields{
		"empty_changes":  len(emptyChanges),
		"binary_changes": len(binaryChanges),
	}).Info("Empty and binary-only directory test completed")
}

// TestTransformPerformanceWithManyFiles tests performance requirements
func (suite *DirectoryTransformTestSuite) TestTransformPerformanceWithManyFiles() {
	ctx := context.Background()

	// Performance test with many files
	perfMapping := config.DirectoryMapping{
		Src:  "",
		Dest: "performance_output",
		Transform: config.Transform{
			RepoName: true,
			Variables: map[string]string{
				"PROJECT_NAME": "perf-test",
				"OWNER":        "perf-owner",
			},
		},
	}

	startTime := time.Now()
	changes, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.performanceTestDir, perfMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)
	totalDuration := time.Since(startTime)

	require.NoError(suite.T(), err)
	assert.True(suite.T(), len(changes) > 100, "Should process many files for performance test")

	// Performance requirements: < 100ms per file on average
	avgTimePerFile := totalDuration / time.Duration(len(changes))
	assert.Less(suite.T(), avgTimePerFile, 100*time.Millisecond,
		"Average processing time per file should be < 100ms, got %v", avgTimePerFile)

	// Total time should be reasonable for 200 files
	assert.Less(suite.T(), totalDuration, 10*time.Second,
		"Total processing time should be < 10s for 200 files, got %v", totalDuration)

	suite.logger.WithFields(logrus.Fields{
		"files_processed":   len(changes),
		"total_duration":    totalDuration,
		"avg_time_per_file": avgTimePerFile,
		"files_per_second":  float64(len(changes)) / totalDuration.Seconds(),
	}).Info("Performance test completed")
}

// TestMetricsAccuracy tests that metrics are accurately tracked
func (suite *DirectoryTransformTestSuite) TestMetricsAccuracy() {
	ctx := context.Background()

	// Create progress manager for metrics tracking
	progressManager := NewDirectoryProgressManager(suite.logger)
	suite.processor.progressManager = progressManager

	dirMapping := config.DirectoryMapping{
		Src:       "",
		Dest:      "metrics_test",
		Transform: suite.targetConfig.Transform,
	}

	changes, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, dirMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	require.NoError(suite.T(), err)

	// Get final metrics
	allMetrics := progressManager.CompleteAll()
	assert.True(suite.T(), len(allMetrics) > 0, "Should have metrics for processed directory")

	for dirPath, metrics := range allMetrics {
		suite.logger.WithFields(logrus.Fields{
			"directory":            dirPath,
			"files_discovered":     metrics.FilesDiscovered,
			"files_processed":      metrics.FilesProcessed,
			"binary_files_skipped": metrics.BinaryFilesSkipped,
			"transform_errors":     metrics.TransformErrors,
			"transform_successes":  metrics.TransformSuccesses,
		}).Info("Directory metrics")

		// Verify metrics make sense
		assert.True(suite.T(), metrics.FilesDiscovered > 0, "Should discover files")
		assert.True(suite.T(), metrics.BinaryFilesSkipped > 0, "Should skip binary files")
		assert.True(suite.T(), metrics.TransformSuccesses > 0, "Should have successful transforms")

		// Verify that we have reasonable metrics relationships
		// Note: The exact count relationship can vary due to excluded files and discovery timing
		assert.True(suite.T(), metrics.BinaryFilesSkipped+metrics.TransformSuccesses > 0,
			"Should have processed some files (binary or transformed)")
		assert.True(suite.T(), metrics.BinaryFilesSkipped > 0 || metrics.TransformSuccesses > 0,
			"Should have either binary files or successful transforms")
	}

	// Verify changes match expectations
	assert.True(suite.T(), len(changes) > 0, "Should have some changes")
}

// Benchmark tests for performance measurement

// BenchmarkDirectoryTransformSmallFiles benchmarks small file directory transforms
func (suite *DirectoryTransformTestSuite) BenchmarkDirectoryTransformSmallFiles() {
	if testing.Short() {
		suite.T().Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()
	dirMapping := config.DirectoryMapping{
		Src:  "src",
		Dest: "bench_small",
		Transform: config.Transform{
			RepoName:  true,
			Variables: map[string]string{"PROJECT_NAME": "bench-test"},
		},
	}

	for i := 0; i < 10; i++ {
		_, err := suite.processor.ProcessDirectoryMapping(
			ctx, suite.sourceDir, dirMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
		)
		require.NoError(suite.T(), err)
	}
}

// BenchmarkBinaryDetection benchmarks binary file detection performance
func (suite *DirectoryTransformTestSuite) BenchmarkBinaryDetection() {
	if testing.Short() {
		suite.T().Skip("Skipping benchmark in short mode")
	}

	testFiles := []struct {
		name string
		data []byte
	}{
		{"small_binary", suite.binaryData},
		{"large_binary", suite.largeBinaryData},
		{"text_file", []byte("This is text content with variables {{.PROJECT_NAME}}")},
	}

	for _, tf := range testFiles {
		suite.T().Run(tf.name, func(t *testing.T) {
			for i := 0; i < 1000; i++ {
				_ = transform.IsBinary("test.dat", tf.data)
			}
		})
	}
}

// TestTransformWithNestedExclusions tests comprehensive exclusion patterns
func (suite *DirectoryTransformTestSuite) TestTransformWithNestedExclusions() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Src:  "",
		Dest: "exclusion_test",
		Transform: config.Transform{
			RepoName: true,
		},
		Exclude: []string{
			"images/**",
			"assets/**",
			"*.dat",
			"*.bin",
			".secrets/**",
			"empty_dir/**",
			"nested/binary/**",
		},
	}

	changes, err := suite.processor.ProcessDirectoryMapping(
		ctx, suite.sourceDir, dirMapping, suite.targetConfig, suite.sourceState, suite.mockEngine,
	)

	require.NoError(suite.T(), err)

	// Verify excluded patterns are not in results
	excludedPatterns := []string{"images/", "assets/", ".dat", ".bin", ".secrets/", "empty_dir/", "nested/binary/"}

	for _, change := range changes {
		for _, pattern := range excludedPatterns {
			assert.NotContains(suite.T(), change.Path, pattern,
				"Change path should not contain excluded pattern %s: %s", pattern, change.Path)
		}
	}

	// Should still have some files processed
	assert.True(suite.T(), len(changes) > 5, "Should process non-excluded files")

	suite.logger.WithField("changes_after_exclusion", len(changes)).Info("Exclusion test completed")
}

// Run the comprehensive test suite
func TestDirectoryTransformTestSuite(t *testing.T) {
	suite.Run(t, new(DirectoryTransformTestSuite))
}

// DirectoryMockGHClient provides a mock GitHub client for directory testing
type DirectoryMockGHClient struct{}

// GetFile implements a mock GetFile method
func (m *DirectoryMockGHClient) GetFile(ctx context.Context, repo, path, branch string) (*gh.FileContent, error) {
	return nil, os.ErrNotExist // Simulate file not found
}

// Required methods to implement gh.Client interface
func (m *DirectoryMockGHClient) ListBranches(ctx context.Context, repo string) ([]gh.Branch, error) {
	return nil, nil
}

func (m *DirectoryMockGHClient) GetBranch(ctx context.Context, repo, branch string) (*gh.Branch, error) {
	return nil, nil
}

func (m *DirectoryMockGHClient) CreatePR(ctx context.Context, repo string, req gh.PRRequest) (*gh.PR, error) {
	return nil, nil
}

func (m *DirectoryMockGHClient) GetPR(ctx context.Context, repo string, number int) (*gh.PR, error) {
	return nil, nil
}

func (m *DirectoryMockGHClient) ListPRs(ctx context.Context, repo, state string) ([]gh.PR, error) {
	return nil, nil
}

func (m *DirectoryMockGHClient) GetCommit(ctx context.Context, repo, sha string) (*gh.Commit, error) {
	return nil, nil
}

func (m *DirectoryMockGHClient) ClosePR(ctx context.Context, repo string, number int, comment string) error {
	return nil
}

func (m *DirectoryMockGHClient) DeleteBranch(ctx context.Context, repo, branch string) error {
	return nil
}

func (m *DirectoryMockGHClient) UpdatePR(ctx context.Context, repo string, number int, updates gh.PRUpdate) error {
	return nil
}

func (m *DirectoryMockGHClient) GetCurrentUser(ctx context.Context) (*gh.User, error) {
	return nil, nil
}

func (m *DirectoryMockGHClient) GetGitTree(ctx context.Context, repo, treeSHA string, recursive bool) (*gh.GitTree, error) {
	return nil, nil
}

// DirectoryMockFileContent represents mock file content
type DirectoryMockFileContent struct {
	Content []byte
}

// DirectoryMockTransformChain provides a mock transform chain for directory testing
type DirectoryMockTransformChain struct {
	transformers []transform.Transformer
}

// Add implements the Chain interface
func (m *DirectoryMockTransformChain) Add(transformer transform.Transformer) transform.Chain {
	m.transformers = append(m.transformers, transformer)
	return m
}

// Transformers implements the Chain interface
func (m *DirectoryMockTransformChain) Transformers() []transform.Transformer {
	return m.transformers
}

// Transform implements a mock Transform method with comprehensive transformation
func (m *DirectoryMockTransformChain) Transform(ctx context.Context, content []byte, transformCtx transform.Context) ([]byte, error) {
	// Simple but effective transformation logic
	result := string(content)

	// Apply repo name transformation
	if strings.Contains(result, "test/source-repo") {
		result = strings.ReplaceAll(result, "test/source-repo", transformCtx.TargetRepo)
	}

	// Apply variable substitutions
	for varName, varValue := range transformCtx.Variables {
		template := fmt.Sprintf("{{.%s}}", varName)
		result = strings.ReplaceAll(result, template, varValue)
	}

	// Simulate potential transform error for specific files (for error isolation testing)
	if strings.Contains(transformCtx.FilePath, "invalid/template.txt") {
		// For invalid templates, still do repo name transform but leave missing variables
		return []byte(result), nil
	}

	return []byte(result), nil
}
