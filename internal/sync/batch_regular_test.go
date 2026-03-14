package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// TestNewBatchProcessor_Regular tests the batch processor constructor
func TestNewBatchProcessor_Regular(t *testing.T) {
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

	t.Run("negative worker count uses default", func(t *testing.T) {
		processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, -5)
		assert.Equal(t, 10, processor.ConfiguredWorkerCount())
	})
}

// TestNewFileJob_Regular tests the file job constructor
func TestNewFileJob_Regular(t *testing.T) {
	trans := config.Transform{RepoName: true, Variables: map[string]string{"key": "value"}}
	job := NewFileJob("src.txt", "dest.txt", trans)

	assert.Equal(t, "src.txt", job.SourcePath)
	assert.Equal(t, "dest.txt", job.DestPath)
	assert.Equal(t, trans, job.Transform)
	assert.False(t, job.IsFromDirectory)
	assert.Nil(t, job.DirectoryMapping)
	assert.Empty(t, job.RelativePath)
	assert.Equal(t, 0, job.FileIndex)
	assert.Equal(t, 1, job.TotalFiles)
}

// TestNewDirectoryFileJob_Regular tests the directory file job constructor
func TestNewDirectoryFileJob_Regular(t *testing.T) {
	trans := config.Transform{RepoName: true}
	directoryMapping := &config.DirectoryMapping{
		Src:  "src_dir",
		Dest: "dest_dir",
	}

	job := NewDirectoryFileJob(
		"src_dir/file.txt",
		"dest_dir/file.txt",
		trans,
		directoryMapping,
		"file.txt",
		3,
		10,
	)

	assert.Equal(t, "src_dir/file.txt", job.SourcePath)
	assert.Equal(t, "dest_dir/file.txt", job.DestPath)
	assert.Equal(t, trans, job.Transform)
	assert.True(t, job.IsFromDirectory)
	assert.Equal(t, directoryMapping, job.DirectoryMapping)
	assert.Equal(t, "file.txt", job.RelativePath)
	assert.Equal(t, 3, job.FileIndex)
	assert.Equal(t, 10, job.TotalFiles)
}

// TestBatchProcessor_GetStats tests statistics retrieval
func TestBatchProcessor_GetStats(t *testing.T) {
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

	processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 7)

	stats := processor.GetStats()
	assert.Equal(t, 7, stats.WorkerCount)
	assert.Equal(t, 0, stats.TotalJobs)
	assert.Equal(t, 0, stats.ProcessedJobs)
	assert.Equal(t, 0, stats.SkippedJobs)
	assert.Equal(t, 0, stats.FailedJobs)
}

// TestBatchProcessor_SetWorkerCount tests worker count modification
func TestBatchProcessor_SetWorkerCount(t *testing.T) {
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

	processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 5)
	assert.Equal(t, 5, processor.ConfiguredWorkerCount())

	processor.SetWorkerCount(10)
	assert.Equal(t, 10, processor.ConfiguredWorkerCount())

	// Zero and negative values should be ignored
	processor.SetWorkerCount(0)
	assert.Equal(t, 10, processor.ConfiguredWorkerCount())

	processor.SetWorkerCount(-5)
	assert.Equal(t, 10, processor.ConfiguredWorkerCount())
}

// TestBatchProcessor_ProcessFiles_Empty tests processing empty job list
func TestBatchProcessor_ProcessFiles_Empty(t *testing.T) {
	ctx := context.Background()
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

	processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 2)

	changes, err := processor.ProcessFiles(ctx, "/tmp", []FileJob{})
	require.NoError(t, err)
	assert.Nil(t, changes)
}

// TestBatchProcessor_ProcessFiles_SingleFile tests single file processing
func TestBatchProcessor_ProcessFiles_SingleFile(t *testing.T) {
	ctx := context.Background()

	// Create temp directory and test file
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("Hello World"), 0o600)
	require.NoError(t, err)

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

	// Setup mocks
	mockGH.On("GetFile", mock.Anything, "target/repo", "test.txt", "").
		Return(nil, internalerrors.ErrFileNotFound).Once()
	mockTransform.On("Transform", mock.Anything, []byte("Hello World"), mock.AnythingOfType("transform.Context")).
		Return([]byte("Transformed Hello World"), nil).Once()

	processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 1)

	jobs := []FileJob{
		NewFileJob("test.txt", "test.txt", config.Transform{RepoName: true}),
	}

	changes, err := processor.ProcessFiles(ctx, tempDir, jobs)

	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "test.txt", changes[0].Path)
	assert.Equal(t, []byte("Transformed Hello World"), changes[0].Content)
	assert.Nil(t, changes[0].OriginalContent) // New files have no original content in target
	assert.True(t, changes[0].IsNew)

	mockGH.AssertExpectations(t)
	mockTransform.AssertExpectations(t)
}

// TestBatchProcessor_ProcessFiles_FileNotFound tests missing file handling
func TestBatchProcessor_ProcessFiles_FileNotFound(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

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

	processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 1)

	jobs := []FileJob{
		NewFileJob("nonexistent.txt", "nonexistent.txt", config.Transform{}),
	}

	changes, err := processor.ProcessFiles(ctx, tempDir, jobs)

	require.NoError(t, err)
	assert.Empty(t, changes)
}

// TestBatchProcessor_ProcessFiles_ContentUnchanged tests unchanged content handling
func TestBatchProcessor_ProcessFiles_ContentUnchanged(t *testing.T) {
	ctx := context.Background()

	// Create temp directory and test file
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "unchanged.txt")
	err := os.WriteFile(testFilePath, []byte("Same Content"), 0o600)
	require.NoError(t, err)

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

	// Mock returns same content as source
	mockGH.On("GetFile", mock.Anything, "target/repo", "unchanged.txt", "").
		Return(&gh.FileContent{Content: []byte("Same Content")}, nil).Once()

	processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 1)

	jobs := []FileJob{
		NewFileJob("unchanged.txt", "unchanged.txt", config.Transform{}), // No transformation
	}

	changes, err := processor.ProcessFiles(ctx, tempDir, jobs)

	require.NoError(t, err)
	assert.Empty(t, changes) // Should be empty since content unchanged

	mockGH.AssertExpectations(t)
}

// TestBatchProcessor_ProcessFiles_ExistingFileUpdated tests that OriginalContent
// is set to the TARGET repo's existing content (not source content).
// REGRESSION TEST: Prevents bug where OriginalContent was incorrectly set to srcContent,
// causing synthetic diffs to be empty when source and transformed content matched.
func TestBatchProcessor_ProcessFiles_ExistingFileUpdated(t *testing.T) {
	ctx := context.Background()

	// Create temp directory with source file content
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "workflow.yml")
	// Source content (what we're syncing from) - has new permission line
	srcContent := []byte("name: CI\npermissions: {}\n")
	err := os.WriteFile(testFilePath, srcContent, 0o600)
	require.NoError(t, err)

	logger := logrus.NewEntry(logrus.New())
	mockGH := &gh.MockClient{}
	engine := &Engine{
		gh:        mockGH,
		transform: nil, // No transformer needed - files sync without transformation
	}
	sourceState := &state.SourceState{
		Repo:         "source/repo",
		Branch:       "main",
		LatestCommit: "abc123",
	}
	targetConfig := config.TargetConfig{
		Repo:   "target/repo",
		Branch: "development",
	}

	// Target repo has DIFFERENT content (older version without permissions)
	existingTargetContent := []byte("name: CI\n")

	// Mock GetFile returns existing TARGET content
	mockGH.On("GetFile", mock.Anything, "target/repo", "workflow.yml", "development").
		Return(&gh.FileContent{Content: existingTargetContent}, nil).Once()

	processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 1)

	// No transform config - file syncs as-is (common for workflow files)
	jobs := []FileJob{
		NewFileJob("workflow.yml", "workflow.yml", config.Transform{}),
	}

	changes, err := processor.ProcessFiles(ctx, tempDir, jobs)

	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "workflow.yml", changes[0].Path)
	assert.Equal(t, srcContent, changes[0].Content) // New content is source content

	// CRITICAL: OriginalContent must be TARGET repo's existing content (not source)
	// This enables correct diff generation: old=existing target, new=transformed source
	assert.Equal(t, existingTargetContent, changes[0].OriginalContent,
		"OriginalContent should be target repo's existing content for accurate diffs")

	assert.False(t, changes[0].IsNew)

	mockGH.AssertExpectations(t)
}

// TestBatchProcessor_CollectResults tests result collection and filtering
func TestBatchProcessor_CollectResults(t *testing.T) {
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

	processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 1)

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
		Error:  internalerrors.ErrSyncFailed,
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
}

// TestBatchProcessor_ProcessFilesWithProgress_Empty tests progress reporting with empty jobs
func TestBatchProcessor_ProcessFilesWithProgress_Empty(t *testing.T) {
	ctx := context.Background()
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

	processor := NewBatchProcessor(engine, targetConfig, sourceState, logger, 2)

	changes, err := processor.ProcessFilesWithProgress(ctx, "/tmp", []FileJob{}, nil)
	require.NoError(t, err)
	assert.Nil(t, changes)
}
