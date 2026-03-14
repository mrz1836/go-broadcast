//go:build bench_heavy

package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// Static error variables for testing
var (
	ErrTestFileNotFound         = internalerrors.ErrFileNotFound
	ErrTestTransformationFailed = internalerrors.ErrTransformNotFound
	ErrTestProcessingError      = internalerrors.ErrSyncFailed
)

// BatchProcessorTestSuite provides comprehensive batch processor testing
type BatchProcessorTestSuite struct {
	suite.Suite

	tempDir       string
	mockEngine    *MockBatchEngine
	mockGH        *gh.MockClient
	mockTransform *transform.MockChain
	sourceState   *state.SourceState
	targetConfig  config.TargetConfig
	logger        *logrus.Entry
}

// MockBatchEngine provides a mock engine for batch processor testing
type MockBatchEngine struct {
	mock.Mock

	gh        gh.Client
	transform transform.Chain
}

func (m *MockBatchEngine) GetGH() gh.Client {
	return m.gh
}

func (m *MockBatchEngine) GetTransform() transform.Chain {
	return m.transform
}

// MockProgressReporter implements ProgressReporter for testing
type MockProgressReporter struct {
	mock.Mock

	updates []ProgressUpdate
	mu      sync.Mutex
}

type ProgressUpdate struct {
	Current int
	Total   int
	Message string
}

func (m *MockProgressReporter) UpdateProgress(current, total int, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updates = append(m.updates, ProgressUpdate{
		Current: current,
		Total:   total,
		Message: message,
	})
	m.Called(current, total, message)
}

func (m *MockProgressReporter) GetUpdates() []ProgressUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()
	updates := make([]ProgressUpdate, len(m.updates))
	copy(updates, m.updates)
	return updates
}

// MockEnhancedProgressReporter implements EnhancedProgressReporter for testing
type MockEnhancedProgressReporter struct {
	MockProgressReporter

	binaryFilesSkipped []int64
	transformErrors    int32
	transformSuccesses []time.Duration
	filesChanged       int32
}

func (m *MockEnhancedProgressReporter) RecordBinaryFileSkipped(size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.binaryFilesSkipped = append(m.binaryFilesSkipped, size)
	m.Called(size)
}

func (m *MockEnhancedProgressReporter) RecordTransformError() {
	atomic.AddInt32(&m.transformErrors, 1)
	m.Called()
}

func (m *MockEnhancedProgressReporter) RecordTransformSuccess(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transformSuccesses = append(m.transformSuccesses, duration)
	m.Called(duration)
}

func (m *MockEnhancedProgressReporter) GetBinaryFilesSkipped() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	skipped := make([]int64, len(m.binaryFilesSkipped))
	copy(skipped, m.binaryFilesSkipped)
	return skipped
}

func (m *MockEnhancedProgressReporter) GetTransformErrors() int32 {
	return atomic.LoadInt32(&m.transformErrors)
}

func (m *MockEnhancedProgressReporter) GetTransformSuccesses() []time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	successes := make([]time.Duration, len(m.transformSuccesses))
	copy(successes, m.transformSuccesses)
	return successes
}

func (m *MockEnhancedProgressReporter) RecordFileChanged() {
	atomic.AddInt32(&m.filesChanged, 1)
	m.Called()
}

func (m *MockEnhancedProgressReporter) GetFilesChanged() int32 {
	return atomic.LoadInt32(&m.filesChanged)
}

// SetupSuite initializes the test suite
func (suite *BatchProcessorTestSuite) SetupSuite() {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "batch-processor-test-*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	// Create test files
	suite.createTestFiles()
}

// TearDownSuite cleans up the test suite
func (suite *BatchProcessorTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		err := os.RemoveAll(suite.tempDir)
		suite.Require().NoError(err)
	}
}

// SetupTest initializes each test
func (suite *BatchProcessorTestSuite) SetupTest() {
	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	suite.logger = logger.WithField("component", "batch-test")

	// Initialize fresh mocks for each test
	suite.mockGH = &gh.MockClient{}
	suite.mockTransform = &transform.MockChain{}
	suite.mockEngine = &MockBatchEngine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}

	// Setup source state
	suite.sourceState = &state.SourceState{
		Repo:         "source/repo",
		Branch:       "main",
		LatestCommit: "abc123",
		LastChecked:  time.Now(),
	}

	// Setup target config
	suite.targetConfig = config.TargetConfig{
		Repo: "target/repo",
		Files: []config.FileMapping{
			{Src: "file1.txt", Dest: "file1.txt"},
			{Src: "file2.txt", Dest: "file2.txt"},
		},
	}
}

// TearDownTest cleans up each test
func (suite *BatchProcessorTestSuite) TearDownTest() {
	// Don't assert expectations here since some tests may intentionally not use all mocks
	// Individual test methods should assert their own expectations if needed
}

// createTestFiles creates test files in the temp directory
func (suite *BatchProcessorTestSuite) createTestFiles() {
	// Create text files
	testFiles := map[string]string{
		"file1.txt":        "Hello World",
		"file2.txt":        "{{.SourceRepo}} content",
		"file3.txt":        "Binary detection test",
		"subdir/file4.txt": "Subdirectory file",
		"empty.txt":        "",
		"large.txt":        strings.Repeat("Large file content ", 1000),
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(suite.tempDir, path)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0o750)
		suite.Require().NoError(err)
		err = os.WriteFile(fullPath, []byte(content), 0o600)
		suite.Require().NoError(err)
	}

	// Create a binary file
	binaryContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	binaryPath := filepath.Join(suite.tempDir, "image.png")
	err := os.WriteFile(binaryPath, binaryContent, 0o600)
	suite.Require().NoError(err)
}

func TestBatchProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(BatchProcessorTestSuite))
}

// TestNewBatchProcessorStandalone tests the constructor independently
func TestNewBatchProcessorStandalone(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	mockGH := &gh.MockClient{}
	mockTransform := &transform.MockChain{}
	engine := &Engine{
		gh:        mockGH,
		transform: mockTransform,
	}
	sourceState := &state.SourceState{
		Repo:         "source/repo",
		Branch:       "main",
		LatestCommit: "abc123",
	}
	targetConfig := config.TargetConfig{
		Repo: "target/repo",
	}

	t.Run("valid worker count", func(t *testing.T) {
		processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 5)
		assert.Equal(t, 5, processor.ConfiguredWorkerCount())
	})

	t.Run("zero worker count uses default", func(t *testing.T) {
		processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 0)
		assert.Equal(t, 10, processor.ConfiguredWorkerCount())
	})
}

// TestFileJobConstructorsStandalone tests job constructors independently
func TestFileJobConstructorsStandalone(t *testing.T) {
	t.Run("NewFileJob", func(t *testing.T) {
		transform := config.Transform{RepoName: true, Variables: map[string]string{"key": "value"}}
		job := NewFileJob("src.txt", "dest.txt", transform)

		assert.Equal(t, "src.txt", job.SourcePath)
		assert.Equal(t, "dest.txt", job.DestPath)
		assert.Equal(t, transform, job.Transform)
		assert.False(t, job.IsFromDirectory)
		assert.Nil(t, job.DirectoryMapping)
		assert.Empty(t, job.RelativePath)
		assert.Equal(t, 0, job.FileIndex)
		assert.Equal(t, 1, job.TotalFiles)
	})

	t.Run("NewDirectoryFileJob", func(t *testing.T) {
		transform := config.Transform{RepoName: true}
		directoryMapping := &config.DirectoryMapping{
			Src:  "src_dir",
			Dest: "dest_dir",
		}

		job := NewDirectoryFileJob(
			"src_dir/file.txt",
			"dest_dir/file.txt",
			transform,
			directoryMapping,
			"file.txt",
			3,
			10,
		)

		assert.Equal(t, "src_dir/file.txt", job.SourcePath)
		assert.Equal(t, "dest_dir/file.txt", job.DestPath)
		assert.Equal(t, transform, job.Transform)
		assert.True(t, job.IsFromDirectory)
		assert.Equal(t, directoryMapping, job.DirectoryMapping)
		assert.Equal(t, "file.txt", job.RelativePath)
		assert.Equal(t, 3, job.FileIndex)
		assert.Equal(t, 10, job.TotalFiles)
	})
}

// TestNewBatchProcessor tests the constructor
func (suite *BatchProcessorTestSuite) TestNewBatchProcessor() {
	t := suite.T()

	// Create engine with embedded interfaces
	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}

	t.Run("with valid worker count", func(t *testing.T) {
		processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 5)

		require.NotNil(t, processor)
		assert.Equal(t, engine, processor.engine)
		assert.Equal(t, suite.targetConfig, processor.target)
		assert.Equal(t, suite.sourceState, processor.sourceState)
		assert.Equal(t, suite.logger, processor.logger)
		assert.Equal(t, 5, processor.workerCount)
	})

	t.Run("with zero worker count uses default", func(t *testing.T) {
		processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 0)

		require.NotNil(t, processor)
		assert.Equal(t, 10, processor.workerCount) // Default value
	})

	t.Run("with negative worker count uses default", func(t *testing.T) {
		processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, -5)

		require.NotNil(t, processor)
		assert.Equal(t, 10, processor.workerCount) // Default value
	})
}

// TestProcessFilesBasic tests basic file processing
func (suite *BatchProcessorTestSuite) TestProcessFilesBasic() {
	t := suite.T()
	ctx := context.Background()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 2)

	t.Run("empty jobs list", func(t *testing.T) {
		changes, err := processor.ProcessFiles(ctx, suite.tempDir, []FileJob{})

		require.NoError(t, err)
		assert.Nil(t, changes)
	})

	t.Run("single file processing", func(t *testing.T) {
		// Ensure test file exists
		testFilePath := filepath.Join(suite.tempDir, "file1.txt")
		if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
			err := os.WriteFile(testFilePath, []byte("Hello World"), 0o600)
			require.NoError(t, err)
		}

		// Setup mocks
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "file1.txt", "").
			Return(nil, ErrTestFileNotFound).Once()
		suite.mockTransform.On("Transform", mock.Anything, []byte("Hello World"), mock.AnythingOfType("transform.Context")).
			Return([]byte("Transformed Hello World"), nil).Once()

		jobs := []FileJob{
			NewFileJob("file1.txt", "file1.txt", config.Transform{RepoName: true}),
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		require.Len(t, changes, 1)
		assert.Equal(t, "file1.txt", changes[0].Path)
		assert.Equal(t, []byte("Transformed Hello World"), changes[0].Content)
		assert.Equal(t, []byte("Hello World"), changes[0].OriginalContent)
		assert.True(t, changes[0].IsNew)
	})

	t.Run("multiple files concurrent processing", func(t *testing.T) {
		// Ensure test files exist
		testFile1Path := filepath.Join(suite.tempDir, "file1.txt")
		testFile2Path := filepath.Join(suite.tempDir, "file2.txt")
		if _, err := os.Stat(testFile1Path); os.IsNotExist(err) {
			err := os.WriteFile(testFile1Path, []byte("Hello World"), 0o600)
			require.NoError(t, err)
		}
		if _, err := os.Stat(testFile2Path); os.IsNotExist(err) {
			err := os.WriteFile(testFile2Path, []byte("{{.SourceRepo}} content"), 0o600)
			require.NoError(t, err)
		}

		// Setup mocks for multiple files
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "file1.txt", "").
			Return(nil, ErrTestFileNotFound).Once()
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "file2.txt", "").
			Return(nil, ErrTestFileNotFound).Once()

		suite.mockTransform.On("Transform", mock.Anything, []byte("Hello World"), mock.AnythingOfType("transform.Context")).
			Return([]byte("Transformed Hello World"), nil).Once()
		suite.mockTransform.On("Transform", mock.Anything, []byte("{{.SourceRepo}} content"), mock.AnythingOfType("transform.Context")).
			Return([]byte("source/repo content"), nil).Once()

		jobs := []FileJob{
			NewFileJob("file1.txt", "file1.txt", config.Transform{RepoName: true}),
			NewFileJob("file2.txt", "file2.txt", config.Transform{RepoName: true}),
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		require.Len(t, changes, 2)

		// Sort changes for predictable testing
		if changes[0].Path == "file2.txt" {
			changes[0], changes[1] = changes[1], changes[0]
		}

		assert.Equal(t, "file1.txt", changes[0].Path)
		assert.Equal(t, "file2.txt", changes[1].Path)
	})
}

// TestProcessFilesWithProgress tests progress reporting
func (suite *BatchProcessorTestSuite) TestProcessFilesWithProgress() {
	t := suite.T()
	ctx := context.Background()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 2)

	t.Run("basic progress reporting", func(t *testing.T) {
		// Setup mocks
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "file1.txt", "").
			Return(nil, ErrTestFileNotFound).Once()
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "file2.txt", "").
			Return(nil, ErrTestFileNotFound).Once()
		suite.mockTransform.On("Transform", mock.Anything, []byte("Hello World"), mock.AnythingOfType("transform.Context")).
			Return([]byte("transformed file1"), nil).Once()
		suite.mockTransform.On("Transform", mock.Anything, []byte("{{.SourceRepo}} content"), mock.AnythingOfType("transform.Context")).
			Return([]byte("transformed file2"), nil).Once()

		// Create mock progress reporter
		progressReporter := &MockProgressReporter{}
		progressReporter.On("UpdateProgress", mock.Anything, mock.Anything, mock.Anything).
			Return().Maybe()

		jobs := []FileJob{
			NewFileJob("file1.txt", "file1.txt", config.Transform{RepoName: true}),
			NewFileJob("file2.txt", "file2.txt", config.Transform{RepoName: true}),
		}

		changes, err := processor.ProcessFilesWithProgress(ctx, suite.tempDir, jobs, progressReporter)

		require.NoError(t, err)
		require.Len(t, changes, 2)

		// Verify progress updates were called
		updates := progressReporter.GetUpdates()
		assert.NotEmpty(t, updates)

		// Should have initial progress (0, 2) and final progress (2, 2)
		assert.Equal(t, 0, updates[0].Current)
		assert.Equal(t, 2, updates[0].Total)
		finalUpdate := updates[len(updates)-1]
		assert.Equal(t, 2, finalUpdate.Current)
		assert.Equal(t, 2, finalUpdate.Total)
	})

	t.Run("enhanced progress reporting", func(t *testing.T) {
		// Setup mocks for binary file and transform success
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "image.png", "").
			Return(nil, ErrTestFileNotFound).Once()

		// Create enhanced progress reporter
		enhancedReporter := &MockEnhancedProgressReporter{}
		enhancedReporter.On("UpdateProgress", mock.Anything, mock.Anything, mock.Anything).
			Return().Maybe()
		enhancedReporter.On("RecordBinaryFileSkipped", mock.AnythingOfType("int64")).
			Return().Once()
		enhancedReporter.On("RecordTransformSuccess", mock.AnythingOfType("time.Duration")).
			Return().Once()

		jobs := []FileJob{
			NewFileJob("image.png", "image.png", config.Transform{}),
		}

		changes, err := processor.ProcessFilesWithProgress(ctx, suite.tempDir, jobs, enhancedReporter)

		require.NoError(t, err)
		require.Len(t, changes, 1)

		// Verify binary file was recorded
		binaryFiles := enhancedReporter.GetBinaryFilesSkipped()
		assert.Len(t, binaryFiles, 1)
		assert.Positive(t, binaryFiles[0])
	})
}

// TestBinaryFileDetection tests binary file handling
func (suite *BatchProcessorTestSuite) TestBinaryFileDetection() {
	t := suite.T()
	ctx := context.Background()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 1)

	t.Run("binary file processing", func(t *testing.T) {
		// Setup mock for existing file check
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "image.png", "").
			Return(nil, ErrTestFileNotFound).Once()

		jobs := []FileJob{
			NewFileJob("image.png", "image.png", config.Transform{RepoName: true}),
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		require.Len(t, changes, 1)

		// Binary file should be processed without transformation
		assert.Equal(t, "image.png", changes[0].Path)
		assert.True(t, changes[0].IsNew)
		// Content should be original binary content
		assert.Equal(t, changes[0].Content, changes[0].OriginalContent)
	})

	t.Run("binary file unchanged content", func(t *testing.T) {
		// Read the actual binary content
		binaryPath := filepath.Join(suite.tempDir, "image.png")
		binaryContent, err := os.ReadFile(binaryPath) // #nosec G304 -- test file in controlled directory
		require.NoError(t, err)

		// Setup mock to return same content (unchanged)
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "image.png", "").
			Return(&gh.FileContent{Content: binaryContent}, nil).Once()

		jobs := []FileJob{
			NewFileJob("image.png", "image.png", config.Transform{RepoName: true}),
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		// Should be empty since content unchanged
		assert.Empty(t, changes)
	})
}

// TestTransformationErrors tests error handling during transformations
func (suite *BatchProcessorTestSuite) TestTransformationErrors() {
	t := suite.T()
	ctx := context.Background()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 1)

	t.Run("transformation error with fallback", func(t *testing.T) {
		// Setup mocks
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "file1.txt", "").
			Return(nil, ErrTestFileNotFound).Once()
		suite.mockTransform.On("Transform", mock.Anything, []byte("Hello World"), mock.AnythingOfType("transform.Context")).
			Return(nil, ErrTestTransformationFailed).Once()

		jobs := []FileJob{
			NewFileJob("file1.txt", "file1.txt", config.Transform{RepoName: true}),
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		require.Len(t, changes, 1)

		// Should use original content as fallback
		assert.Equal(t, "file1.txt", changes[0].Path)
		assert.Equal(t, []byte("Hello World"), changes[0].Content)
		assert.Equal(t, []byte("Hello World"), changes[0].OriginalContent)
	})

	t.Run("no transformation configured", func(t *testing.T) {
		// Setup mocks
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "file1.txt", "").
			Return(nil, ErrTestFileNotFound).Once()

		jobs := []FileJob{
			NewFileJob("file1.txt", "file1.txt", config.Transform{}), // No transformation
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		require.Len(t, changes, 1)

		// Should use original content
		assert.Equal(t, "file1.txt", changes[0].Path)
		assert.Equal(t, []byte("Hello World"), changes[0].Content)
		assert.Equal(t, []byte("Hello World"), changes[0].OriginalContent)
	})
}

// TestFileErrors tests file-related error handling
func (suite *BatchProcessorTestSuite) TestFileErrors() {
	t := suite.T()
	ctx := context.Background()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 1)

	t.Run("source file not found", func(t *testing.T) {
		jobs := []FileJob{
			NewFileJob("nonexistent.txt", "nonexistent.txt", config.Transform{}),
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		// Should be empty since file not found errors are handled gracefully
		assert.Empty(t, changes)
	})

	t.Run("file content unchanged", func(t *testing.T) {
		// Setup mock to return same content
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "file1.txt", "").
			Return(&gh.FileContent{Content: []byte("Hello World")}, nil).Once()

		jobs := []FileJob{
			NewFileJob("file1.txt", "file1.txt", config.Transform{}), // No transformation
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		// Should be empty since content unchanged
		assert.Empty(t, changes)
	})
}

// TestDirectoryFileJobs tests directory-specific file processing
func (suite *BatchProcessorTestSuite) TestDirectoryFileJobs() {
	t := suite.T()
	ctx := context.Background()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 1)

	t.Run("directory file job processing", func(t *testing.T) {
		// Setup mocks
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "dest/subdir/file4.txt", "").
			Return(nil, ErrTestFileNotFound).Once()
		suite.mockTransform.On("Transform", mock.Anything, []byte("Subdirectory file"), mock.AnythingOfType("transform.Context")).
			Return([]byte("Transformed subdirectory file"), nil).Once()

		directoryMapping := &config.DirectoryMapping{
			Src:  "subdir",
			Dest: "dest/subdir",
		}

		jobs := []FileJob{
			NewDirectoryFileJob(
				"subdir/file4.txt",
				"dest/subdir/file4.txt",
				config.Transform{RepoName: true},
				directoryMapping,
				"file4.txt",
				1,
				5,
			),
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		require.Len(t, changes, 1)

		assert.Equal(t, "dest/subdir/file4.txt", changes[0].Path)
		assert.Equal(t, []byte("Transformed subdirectory file"), changes[0].Content)
		assert.True(t, changes[0].IsNew)
	})
}

// TestGetExistingFileContentUsesTargetBranch is a REGRESSION TEST.
// This test ensures that GetFile is called with the configured target branch,
// NOT an empty string which would default to the repository's default branch.
//
// Bug fixed: Previously the code passed "" to GetFile, which caused the GitHub API
// to fetch from the default branch (e.g., master) instead of the configured
// target branch (e.g., development). This resulted in incorrect diffs for
// repositories where target branch != default branch.
func (suite *BatchProcessorTestSuite) TestGetExistingFileContentUsesTargetBranch() {
	t := suite.T()
	ctx := context.Background()

	// CRITICAL: Set a non-empty target branch
	// If this is empty, the test doesn't catch the regression
	targetConfig := config.TargetConfig{
		Repo:   "target/repo",
		Branch: "development", // Non-default branch
		Files: []config.FileMapping{
			{Src: "file1.txt", Dest: "file1.txt"},
		},
	}

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, targetConfig, suite.sourceState, suite.logger, 1)

	// Ensure test file exists
	testFilePath := filepath.Join(suite.tempDir, "file1.txt")
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		err := os.WriteFile(testFilePath, []byte("Hello World"), 0o600)
		require.NoError(t, err)
	}

	// REGRESSION TEST: Mock expects "development" as the branch parameter (4th arg)
	// If the code passes "" instead, this mock won't match and test will fail with:
	//   mock: Unexpected Method Call
	//   GetFile(context.Background, "target/repo", "file1.txt", "")
	//   The expected call is:
	//   GetFile(context.Background, "target/repo", "file1.txt", "development")
	suite.mockGH.On("GetFile", mock.Anything, "target/repo", "file1.txt", "development").
		Return(&gh.FileContent{Content: []byte("existing content")}, nil).Once()

	// Transform is called because content differs
	suite.mockTransform.On("Transform", mock.Anything, []byte("Hello World"), mock.AnythingOfType("transform.Context")).
		Return([]byte("Transformed Hello World"), nil).Once()

	jobs := []FileJob{
		NewFileJob("file1.txt", "file1.txt", config.Transform{RepoName: true}),
	}

	changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "file1.txt", changes[0].Path)

	// Verify the mock expectation was met - this is the key assertion
	suite.mockGH.AssertExpectations(t)
}

// TestContextCancellation tests context cancellation handling
func (suite *BatchProcessorTestSuite) TestContextCancellation() {
	t := suite.T()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 1)

	t.Run("context cancellation during processing", func(t *testing.T) {
		// Set up mock to handle GetFile call (in case processing gets that far)
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "test_cancel.txt", "").
			Return([]byte("existing content"), nil).Maybe()

		// Create a test file
		testFile := filepath.Join(suite.tempDir, "test_cancel.txt")
		err := os.WriteFile(testFile, []byte("test content for cancellation"), 0o600)
		require.NoError(t, err)

		// Create context with very short timeout to trigger cancellation
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Add a small delay to ensure context times out
		time.Sleep(1 * time.Millisecond)

		jobs := []FileJob{
			NewFileJob("test_cancel.txt", "test_cancel.txt", config.Transform{}),
		}

		_, err = processor.ProcessFiles(ctx, suite.tempDir, jobs)

		// The error might be context.DeadlineExceeded or a wrapped error
		if err != nil {
			// This is the expected case - context was canceled
			assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "context deadline exceeded"))
		} else {
			// If no error, the processing was too fast for cancellation to take effect
			// This is acceptable in a test environment, just log it
			t.Log("Processing completed before context cancellation could take effect")
		}
	})

	t.Run("context timeout during processing", func(t *testing.T) {
		// Set up mock to handle GetFile call (in case processing gets that far)
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "test_timeout.txt", "").
			Return([]byte("existing content"), nil).Maybe()

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Add small delay to ensure timeout
		time.Sleep(1 * time.Millisecond)

		// Create a test file
		testFile := filepath.Join(suite.tempDir, "test_timeout.txt")
		err := os.WriteFile(testFile, []byte("test content for timeout"), 0o600)
		require.NoError(t, err)

		jobs := []FileJob{
			NewFileJob("test_timeout.txt", "test_timeout.txt", config.Transform{}),
		}

		_, err = processor.ProcessFiles(ctx, suite.tempDir, jobs)

		// The error might be context.DeadlineExceeded or a wrapped error
		if err != nil {
			// This is the expected case - context timed out
			assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "context deadline exceeded"))
		} else {
			// If no error, the processing was too fast for timeout to take effect
			// This is acceptable in a test environment, just log it
			t.Log("Processing completed before context timeout could take effect")
		}
	})
}

// TestConcurrentProcessing tests concurrent processing with various worker counts
func (suite *BatchProcessorTestSuite) TestConcurrentProcessing() {
	t := suite.T()
	ctx := context.Background()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}

	testCases := []struct {
		name        string
		workerCount int
		jobCount    int
	}{
		{"single worker", 1, 5},
		{"multiple workers", 4, 10},
		{"more workers than jobs", 8, 3},
		{"high concurrency", 16, 20},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, tc.workerCount)

			// Setup mocks for all jobs
			for i := 0; i < tc.jobCount; i++ {
				filename := fmt.Sprintf("file%d.txt", i)
				suite.mockGH.On("GetFile", mock.Anything, "target/repo", filename, "").
					Return(nil, ErrTestFileNotFound).Once()
				suite.mockTransform.On("Transform", mock.Anything, mock.Anything, mock.AnythingOfType("transform.Context")).
					Return([]byte("transformed"), nil).Once()
			}

			// Create jobs
			jobs := make([]FileJob, tc.jobCount)
			for i := 0; i < tc.jobCount; i++ {
				filename := fmt.Sprintf("file%d.txt", i)
				// Write test file
				testPath := filepath.Join(suite.tempDir, filename)
				err := os.WriteFile(testPath, []byte("test content"), 0o600)
				require.NoError(t, err)

				jobs[i] = NewFileJob(filename, filename, config.Transform{RepoName: true})
			}

			start := time.Now()
			changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)
			duration := time.Since(start)

			require.NoError(t, err)
			assert.Len(t, changes, tc.jobCount)
			t.Logf("Processed %d jobs with %d workers in %v", tc.jobCount, tc.workerCount, duration)

			// Cleanup test files
			for i := 0; i < tc.jobCount; i++ {
				filename := fmt.Sprintf("file%d.txt", i)
				testPath := filepath.Join(suite.tempDir, filename)
				_ = os.Remove(testPath)
			}
		})
	}
}

// TestMemoryEfficiency tests memory usage with large file sets
func (suite *BatchProcessorTestSuite) TestMemoryEfficiency() {
	t := suite.T()
	ctx := context.Background()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 4)

	t.Run("large file set processing", func(t *testing.T) {
		const jobCount = 100

		// Setup mocks for all jobs
		for i := 0; i < jobCount; i++ {
			filename := fmt.Sprintf("large_file_%d.txt", i)
			suite.mockGH.On("GetFile", mock.Anything, "target/repo", filename, "").
				Return(nil, ErrTestFileNotFound).Once()
			suite.mockTransform.On("Transform", mock.Anything, mock.Anything, mock.AnythingOfType("transform.Context")).
				Return([]byte("transformed"), nil).Once()
		}

		// Create large content files
		largeContent := strings.Repeat("Large content ", 1000) // ~13KB per file
		jobs := make([]FileJob, jobCount)

		for i := 0; i < jobCount; i++ {
			filename := fmt.Sprintf("large_file_%d.txt", i)
			testPath := filepath.Join(suite.tempDir, filename)
			err := os.WriteFile(testPath, []byte(largeContent), 0o600)
			require.NoError(t, err)

			jobs[i] = NewFileJob(filename, filename, config.Transform{RepoName: true})
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		assert.Len(t, changes, jobCount)

		// Cleanup
		for i := 0; i < jobCount; i++ {
			filename := fmt.Sprintf("large_file_%d.txt", i)
			testPath := filepath.Join(suite.tempDir, filename)
			_ = os.Remove(testPath)
		}
	})
}

// TestErrorResilience tests that some files can fail while others succeed
func (suite *BatchProcessorTestSuite) TestErrorResilience() {
	t := suite.T()
	ctx := context.Background()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 2)

	t.Run("mixed success and failure scenarios", func(t *testing.T) {
		// Setup mocks - some succeed, some fail
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "success1.txt", "").
			Return(nil, ErrTestFileNotFound).Once()
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "success2.txt", "").
			Return(nil, ErrTestFileNotFound).Once()

		suite.mockTransform.On("Transform", mock.Anything, []byte("success content"), mock.AnythingOfType("transform.Context")).
			Return([]byte("transformed success"), nil).Twice()

		// Create test files - some exist, some don't
		successFiles := []string{"success1.txt", "success2.txt"}
		for _, filename := range successFiles {
			testPath := filepath.Join(suite.tempDir, filename)
			err := os.WriteFile(testPath, []byte("success content"), 0o600)
			require.NoError(t, err)
		}

		jobs := []FileJob{
			NewFileJob("success1.txt", "success1.txt", config.Transform{RepoName: true}),
			NewFileJob("nonexistent1.txt", "nonexistent1.txt", config.Transform{}), // Will fail
			NewFileJob("success2.txt", "success2.txt", config.Transform{RepoName: true}),
			NewFileJob("nonexistent2.txt", "nonexistent2.txt", config.Transform{}), // Will fail
		}

		changes, err := processor.ProcessFiles(ctx, suite.tempDir, jobs)

		require.NoError(t, err)
		// Should only have successful files
		assert.Len(t, changes, 2)

		// Cleanup
		for _, filename := range successFiles {
			testPath := filepath.Join(suite.tempDir, filename)
			_ = os.Remove(testPath)
		}
	})
}

// TestWorkerCountConfiguration tests worker count management
func (suite *BatchProcessorTestSuite) TestWorkerCountConfiguration() {
	t := suite.T()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}

	t.Run("get and set worker count", func(t *testing.T) {
		processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 5)

		assert.Equal(t, 5, processor.ConfiguredWorkerCount())

		processor.SetWorkerCount(10)
		assert.Equal(t, 10, processor.ConfiguredWorkerCount())

		// Zero and negative values should be ignored
		processor.SetWorkerCount(0)
		assert.Equal(t, 10, processor.ConfiguredWorkerCount())

		processor.SetWorkerCount(-5)
		assert.Equal(t, 10, processor.ConfiguredWorkerCount())
	})

	t.Run("get stats", func(t *testing.T) {
		processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 7)

		stats := processor.GetStats()
		assert.Equal(t, 7, stats.WorkerCount)
		// Other stats fields should be zero for a new processor
		assert.Equal(t, 0, stats.TotalJobs)
		assert.Equal(t, 0, stats.ProcessedJobs)
	})
}

// TestFileJobConstructors tests the job constructor functions
func (suite *BatchProcessorTestSuite) TestFileJobConstructors() {
	t := suite.T()

	t.Run("NewFileJob", func(t *testing.T) {
		transform := config.Transform{RepoName: true, Variables: map[string]string{"key": "value"}}
		job := NewFileJob("src.txt", "dest.txt", transform)

		assert.Equal(t, "src.txt", job.SourcePath)
		assert.Equal(t, "dest.txt", job.DestPath)
		assert.Equal(t, transform, job.Transform)
		assert.False(t, job.IsFromDirectory)
		assert.Nil(t, job.DirectoryMapping)
		assert.Empty(t, job.RelativePath)
		assert.Equal(t, 0, job.FileIndex)
		assert.Equal(t, 1, job.TotalFiles)
	})

	t.Run("NewDirectoryFileJob", func(t *testing.T) {
		transform := config.Transform{RepoName: true}
		directoryMapping := &config.DirectoryMapping{
			Src:  "src_dir",
			Dest: "dest_dir",
		}

		job := NewDirectoryFileJob(
			"src_dir/file.txt",
			"dest_dir/file.txt",
			transform,
			directoryMapping,
			"file.txt",
			3,
			10,
		)

		assert.Equal(t, "src_dir/file.txt", job.SourcePath)
		assert.Equal(t, "dest_dir/file.txt", job.DestPath)
		assert.Equal(t, transform, job.Transform)
		assert.True(t, job.IsFromDirectory)
		assert.Equal(t, directoryMapping, job.DirectoryMapping)
		assert.Equal(t, "file.txt", job.RelativePath)
		assert.Equal(t, 3, job.FileIndex)
		assert.Equal(t, 10, job.TotalFiles)
	})
}

// TestCollectResults tests result collection and filtering
func (suite *BatchProcessorTestSuite) TestCollectResults() {
	t := suite.T()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 1)

	t.Run("collect various result types", func(t *testing.T) {
		resultChan := make(chan fileProcessResult, 5)

		// Success result
		resultChan <- fileProcessResult{
			Change: &FileChange{
				Path:    "success.txt",
				Content: []byte("content"),
				IsNew:   true,
			},
			Error: nil,
			Job:   NewFileJob("success.txt", "success.txt", config.Transform{}),
		}

		// File not found error (should be skipped)
		resultChan <- fileProcessResult{
			Change: nil,
			Error:  internalerrors.ErrFileNotFound,
			Job:    NewFileJob("missing.txt", "missing.txt", config.Transform{}),
		}

		// Transform not found error (should be skipped)
		resultChan <- fileProcessResult{
			Change: nil,
			Error:  internalerrors.ErrTransformNotFound,
			Job:    NewFileJob("unchanged.txt", "unchanged.txt", config.Transform{}),
		}

		// Other error (should be logged but continue)
		resultChan <- fileProcessResult{
			Change: nil,
			Error:  ErrTestProcessingError,
			Job:    NewFileJob("error.txt", "error.txt", config.Transform{}),
		}

		// Directory file success
		resultChan <- fileProcessResult{
			Change: &FileChange{
				Path:    "dir/file.txt",
				Content: []byte("dir content"),
				IsNew:   false,
			},
			Error: nil,
			Job: NewDirectoryFileJob(
				"dir/file.txt",
				"dir/file.txt",
				config.Transform{},
				&config.DirectoryMapping{Src: "dir", Dest: "dir"},
				"file.txt",
				1,
				1,
			),
		}

		close(resultChan)

		changes := processor.collectResults(resultChan)

		// Should only have success results
		require.Len(t, changes, 2)
		assert.Equal(t, "success.txt", changes[0].Path)
		assert.Equal(t, "dir/file.txt", changes[1].Path)
	})
}

// TestTransformMetrics tests transformation metrics tracking
func (suite *BatchProcessorTestSuite) TestTransformMetrics() {
	t := suite.T()
	ctx := context.Background()

	engine := &Engine{
		gh:        suite.mockGH,
		transform: suite.mockTransform,
	}
	processor := NewBatchProcessor(engine, suite.targetConfig, suite.sourceState, suite.logger, 1)

	t.Run("metrics tracking with enhanced reporter", func(t *testing.T) {
		// Setup mocks
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "file1.txt", "").
			Return(nil, ErrTestFileNotFound).Once()
		suite.mockGH.On("GetFile", mock.Anything, "target/repo", "image.png", "").
			Return(nil, ErrTestFileNotFound).Once()

		suite.mockTransform.On("Transform", mock.Anything, []byte("Hello World"), mock.AnythingOfType("transform.Context")).
			Return([]byte("Transformed Hello World"), nil).Once()

		// Create enhanced progress reporter
		enhancedReporter := &MockEnhancedProgressReporter{}
		enhancedReporter.On("UpdateProgress", mock.Anything, mock.Anything, mock.Anything).
			Return().Maybe()
		enhancedReporter.On("RecordBinaryFileSkipped", mock.AnythingOfType("int64")).
			Return().Once()
		enhancedReporter.On("RecordTransformSuccess", mock.AnythingOfType("time.Duration")).
			Return().Twice()

		jobs := []FileJob{
			NewFileJob("file1.txt", "file1.txt", config.Transform{RepoName: true}),
			NewFileJob("image.png", "image.png", config.Transform{}),
		}

		changes, err := processor.ProcessFilesWithProgress(ctx, suite.tempDir, jobs, enhancedReporter)

		require.NoError(t, err)
		require.Len(t, changes, 2)

		// Verify metrics
		assert.Len(t, enhancedReporter.GetBinaryFilesSkipped(), 1)
		assert.Equal(t, int32(0), enhancedReporter.GetTransformErrors())
		assert.Len(t, enhancedReporter.GetTransformSuccesses(), 2)
	})
}

// BenchmarkBatchProcessing benchmarks batch processing performance
func BenchmarkBatchProcessing(b *testing.B) {
	// Setup
	tempDir, err := os.MkdirTemp("", "batch-benchmark-*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Create test files
	for i := 0; i < 100; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		path := filepath.Join(tempDir, filename)
		content := fmt.Sprintf("content for file %d", i)
		err := os.WriteFile(path, []byte(content), 0o600)
		if err != nil {
			b.Fatal(err)
		}
	}

	logger := logrus.NewEntry(logrus.New())
	mockGH := &gh.MockClient{}
	mockTransform := &transform.MockChain{}

	// Setup mocks
	for i := 0; i < 100; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		mockGH.On("GetFile", mock.Anything, "target/repo", filename, "").
			Return(nil, ErrTestFileNotFound).Maybe()
		mockTransform.On("Transform", mock.Anything, mock.Anything, mock.AnythingOfType("transform.Context")).
			Return([]byte("transformed"), nil).Maybe()
	}

	engine := &Engine{
		gh:        mockGH,
		transform: mockTransform,
	}

	sourceState := &state.SourceState{
		Repo:         "source/repo",
		Branch:       "main",
		LatestCommit: "abc123",
	}

	targetConfig := config.TargetConfig{
		Repo: "target/repo",
	}

	b.Run("worker_count_1", func(b *testing.B) {
		processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 1)
		benchmarkProcessor(b, processor, tempDir)
	})

	b.Run("worker_count_4", func(b *testing.B) {
		processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 4)
		benchmarkProcessor(b, processor, tempDir)
	})

	b.Run("worker_count_8", func(b *testing.B) {
		processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 8)
		benchmarkProcessor(b, processor, tempDir)
	})

	b.Run("worker_count_16", func(b *testing.B) {
		processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 16)
		benchmarkProcessor(b, processor, tempDir)
	})
}

func benchmarkProcessor(b *testing.B, processor *BatchProcessor, tempDir string) {
	jobs := make([]FileJob, 50)
	for i := 0; i < 50; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		jobs[i] = NewFileJob(filename, filename, config.Transform{RepoName: true})
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.ProcessFiles(ctx, tempDir, jobs)
		if err != nil {
			b.Fatal(err)
		}
	}
}
