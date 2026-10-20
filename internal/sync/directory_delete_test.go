package sync

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// DirectoryDeleteTestSuite tests directory deletion functionality
type DirectoryDeleteTestSuite struct {
	suite.Suite

	processor   *DirectoryProcessor
	mockEngine  *Engine
	mockGH      *gh.MockClient
	target      config.TargetConfig
	sourceState *state.SourceState
	logger      *logrus.Entry
}

// SetupSuite initializes the test suite
func (suite *DirectoryDeleteTestSuite) SetupSuite() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	suite.logger = logrus.NewEntry(logger)

	suite.processor = NewDirectoryProcessor(suite.logger, 5, nil)
	suite.mockGH = &gh.MockClient{}

	suite.mockEngine = &Engine{
		gh: suite.mockGH,
	}

	suite.target = config.TargetConfig{
		Repo: "org/target-repo",
	}

	suite.sourceState = &state.SourceState{
		Repo:         "org/source-repo",
		Branch:       "main",
		LatestCommit: "abc123",
	}
}

// SetupTest creates fresh mocks for each test
func (suite *DirectoryDeleteTestSuite) SetupTest() {
	suite.mockGH = &gh.MockClient{}
	suite.mockEngine = &Engine{
		gh: suite.mockGH,
	}
}

// TearDownTest cleans up after each test
func (suite *DirectoryDeleteTestSuite) TearDownTest() {
	suite.mockGH.AssertExpectations(suite.T())
}

// TestProcessDirectoryDeletion_Success tests successful directory deletion
func (suite *DirectoryDeleteTestSuite) TestProcessDirectoryDeletion_Success() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Src:    "",
		Dest:   "docs",
		Delete: true,
	}

	// Mock a target tree with files in the directory
	suite.setupTreeAPIMock(
		gh.GitTreeNode{Path: "docs", Type: "tree", SHA: "docssha"},
		gh.GitTreeNode{Path: "docs/README.md", Type: "blob", SHA: "file1sha"},
		gh.GitTreeNode{Path: "docs/guide.md", Type: "blob", SHA: "file2sha"},
		gh.GitTreeNode{Path: "docs/api", Type: "tree", SHA: "apisha"},
		gh.GitTreeNode{Path: "docs/api/endpoints.md", Type: "blob", SHA: "file3sha"},
	)

	// Mock GetFile calls for existing content
	suite.mockGH.On("GetFile", ctx, "org/target-repo", "docs/README.md", "").
		Return(&gh.FileContent{Content: []byte("# README")}, nil)
	suite.mockGH.On("GetFile", ctx, "org/target-repo", "docs/guide.md", "").
		Return(&gh.FileContent{Content: []byte("# Guide")}, nil)
	suite.mockGH.On("GetFile", ctx, "org/target-repo", "docs/api/endpoints.md", "").
		Return(&gh.FileContent{Content: []byte("# API Endpoints")}, nil)

	changes, err := suite.processor.processDirectoryDeletion(ctx, dirMapping, suite.target, suite.mockEngine, suite.logger)

	suite.Require().NoError(err)
	suite.Require().Len(changes, 3)

	// Verify all files are marked for deletion
	expectedFiles := []string{"docs/README.md", "docs/guide.md", "docs/api/endpoints.md"}
	actualFiles := make([]string, len(changes))
	for i, change := range changes {
		actualFiles[i] = change.Path
		suite.True(change.IsDeleted, "File %s should be marked for deletion", change.Path)
		suite.Nil(change.Content, "Deleted file should have nil content")
		suite.NotNil(change.OriginalContent, "Should have original content for tracking")
	}

	suite.ElementsMatch(expectedFiles, actualFiles)
}

// TestProcessDirectoryDeletion_NonExistentDirectory tests deletion of non-existent directory
func (suite *DirectoryDeleteTestSuite) TestProcessDirectoryDeletion_NonExistentDirectory() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Src:    "",
		Dest:   "nonexistent",
		Delete: true,
	}

	// Mock an empty target tree (no files in directory)
	suite.setupTreeAPIMock()

	changes, err := suite.processor.processDirectoryDeletion(ctx, dirMapping, suite.target, suite.mockEngine, suite.logger)

	suite.Require().Error(err)
	suite.Require().ErrorIs(err, internalerrors.ErrFileNotFound)
	suite.Nil(changes)
}

// TestProcessDirectoryDeletion_NestedDirectories tests deletion of nested directories
func (suite *DirectoryDeleteTestSuite) TestProcessDirectoryDeletion_NestedDirectories() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Src:    "",
		Dest:   "src",
		Delete: true,
	}

	// Mock a target tree with a deeply nested structure
	suite.setupTreeAPIMock(
		gh.GitTreeNode{Path: "src", Type: "tree", SHA: "src_sha"},
		gh.GitTreeNode{Path: "src/main.go", Type: "blob", SHA: "main_sha"},
		gh.GitTreeNode{Path: "src/utils", Type: "tree", SHA: "utils_sha"},
		gh.GitTreeNode{Path: "src/utils/helper.go", Type: "blob", SHA: "helper_sha"},
		gh.GitTreeNode{Path: "src/api", Type: "tree", SHA: "api_sha"},
		gh.GitTreeNode{Path: "src/api/handlers", Type: "tree", SHA: "handlers_sha"},
		gh.GitTreeNode{Path: "src/api/handlers/user.go", Type: "blob", SHA: "user_sha"},
		gh.GitTreeNode{Path: "src/api/handlers/auth.go", Type: "blob", SHA: "auth_sha"},
		gh.GitTreeNode{Path: "src/config", Type: "tree", SHA: "config_dir_sha"},
		gh.GitTreeNode{Path: "src/config/app.yaml", Type: "blob", SHA: "config_sha"},
	)

	// Mock GetFile calls for all files
	suite.mockGH.On("GetFile", ctx, "org/target-repo", "src/main.go", "").
		Return(&gh.FileContent{Content: []byte("package main")}, nil)
	suite.mockGH.On("GetFile", ctx, "org/target-repo", "src/utils/helper.go", "").
		Return(&gh.FileContent{Content: []byte("package utils")}, nil)
	suite.mockGH.On("GetFile", ctx, "org/target-repo", "src/api/handlers/user.go", "").
		Return(&gh.FileContent{Content: []byte("package handlers")}, nil)
	suite.mockGH.On("GetFile", ctx, "org/target-repo", "src/api/handlers/auth.go", "").
		Return(&gh.FileContent{Content: []byte("package handlers")}, nil)
	suite.mockGH.On("GetFile", ctx, "org/target-repo", "src/config/app.yaml", "").
		Return(&gh.FileContent{Content: []byte("app: config")}, nil)

	changes, err := suite.processor.processDirectoryDeletion(ctx, dirMapping, suite.target, suite.mockEngine, suite.logger)

	suite.Require().NoError(err)
	suite.Require().Len(changes, 5)

	// Verify all files are marked for deletion
	expectedFiles := []string{
		"src/main.go",
		"src/utils/helper.go",
		"src/api/handlers/user.go",
		"src/api/handlers/auth.go",
		"src/config/app.yaml",
	}

	actualFiles := make([]string, len(changes))
	for i, change := range changes {
		actualFiles[i] = change.Path
		suite.True(change.IsDeleted, "File %s should be marked for deletion", change.Path)
		suite.False(change.IsNew, "File %s should not be marked as new", change.Path)
	}

	suite.ElementsMatch(expectedFiles, actualFiles)
}

// TestProcessDirectoryDeletion_GetFileErrors tests handling of GetFile errors
func (suite *DirectoryDeleteTestSuite) TestProcessDirectoryDeletion_GetFileErrors() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Src:    "",
		Dest:   "temp",
		Delete: true,
	}

	suite.setupTreeAPIMock(
		gh.GitTreeNode{Path: "temp", Type: "tree", SHA: "tempsha"},
		gh.GitTreeNode{Path: "temp/file1.txt", Type: "blob", SHA: "file1sha"},
		gh.GitTreeNode{Path: "temp/file2.txt", Type: "blob", SHA: "file2sha"},
	)

	// Mock GetFile - one succeeds, one fails
	suite.mockGH.On("GetFile", ctx, "org/target-repo", "temp/file1.txt", "").
		Return(&gh.FileContent{Content: []byte("content1")}, nil)
	suite.mockGH.On("GetFile", ctx, "org/target-repo", "temp/file2.txt", "").
		Return(nil, assert.AnError)

	changes, err := suite.processor.processDirectoryDeletion(ctx, dirMapping, suite.target, suite.mockEngine, suite.logger)

	suite.Require().NoError(err)
	suite.Require().Len(changes, 2)

	// Both files should still be processed for deletion
	for _, change := range changes {
		suite.True(change.IsDeleted)
	}

	// One should have content, one should have nil content due to error
	hasContent := 0
	nilContent := 0
	for _, change := range changes {
		if change.OriginalContent != nil {
			hasContent++
		} else {
			nilContent++
		}
	}

	suite.Equal(1, hasContent)
	suite.Equal(1, nilContent)
}

// setupTreeAPIMock serves the given entries as the target's recursive tree listing.
// The target has no configured branch, so its default branch is requested as HEAD
// in a single recursive tree request.
func (suite *DirectoryDeleteTestSuite) setupTreeAPIMock(nodes ...gh.GitTreeNode) {
	suite.mockGH.On("GetGitTree", mock.Anything, "org/target-repo", defaultBranchRef, true).
		Return(&gh.GitTree{SHA: "tree123", Tree: nodes}, nil).Once()
}

// TestProcessDirectoryDeletion_TreeUnavailable tests that deletion fails rather than
// deleting part of a directory when the target tree cannot be listed in full
func (suite *DirectoryDeleteTestSuite) TestProcessDirectoryDeletion_TreeUnavailable() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Dest:   "docs",
		Delete: true,
	}

	suite.mockGH.On("GetGitTree", mock.Anything, "org/target-repo", defaultBranchRef, true).
		Return(&gh.GitTree{Truncated: true, Tree: []gh.GitTreeNode{{Path: "docs/README.md", Type: "blob"}}}, nil).Once()

	changes, err := suite.processor.processDirectoryDeletion(ctx, dirMapping, suite.target, suite.mockEngine, suite.logger)

	suite.Require().Error(err)
	suite.Require().ErrorIs(err, ErrTargetTreeUnavailable)
	suite.Nil(changes)
}

// TestProcessDirectoryDeletion_RepositoryRoot tests that a root destination is never
// expanded into a deletion of every file in the repository
func (suite *DirectoryDeleteTestSuite) TestProcessDirectoryDeletion_RepositoryRoot() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Dest:   "/",
		Delete: true,
	}

	changes, err := suite.processor.processDirectoryDeletion(ctx, dirMapping, suite.target, suite.mockEngine, suite.logger)

	suite.Require().Error(err)
	suite.Require().ErrorIs(err, ErrDirectoryDeletionRoot)
	suite.Nil(changes)
}

// TestProcessDirectoryDeletion_ReusesTargetTree tests that a processor holding the
// target sync's tree lists files from it instead of requesting another tree
func (suite *DirectoryDeleteTestSuite) TestProcessDirectoryDeletion_ReusesTargetTree() {
	ctx := context.Background()

	dirMapping := config.DirectoryMapping{
		Dest:   "docs",
		Delete: true,
	}

	suite.mockGH.On("GetGitTree", mock.Anything, "org/target-repo", defaultBranchRef, true).
		Return(&gh.GitTree{Tree: []gh.GitTreeNode{{Path: "docs/README.md", Type: "blob", SHA: "readmesha"}}}, nil).Once()
	suite.mockGH.On("GetFile", mock.Anything, "org/target-repo", "docs/README.md", "").
		Return(&gh.FileContent{Content: []byte("# README")}, nil).Once()

	shared := newTargetTree(suite.mockGH, suite.target.Repo, suite.target.Branch, suite.logger)
	processor := NewDirectoryProcessor(suite.logger, 1, &DirectoryProcessorOptions{targetTree: shared})
	defer processor.Close()

	changes, err := processor.processDirectoryDeletion(ctx, dirMapping, suite.target, suite.mockEngine, suite.logger)

	suite.Require().NoError(err)
	suite.Require().Len(changes, 1)
	suite.Equal(1, shared.stats().TreeRequests, "the shared tree should serve the deletion")
}

// TestSuite runs the directory deletion test suite
func TestDirectoryDeleteSuite(t *testing.T) {
	suite.Run(t, new(DirectoryDeleteTestSuite))
}

// TestDirectoryDeletion_Integration tests directory deletion integrated with ProcessDirectoryMapping
func TestDirectoryDeletion_Integration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	loggerEntry := logrus.NewEntry(logger)

	processor := NewDirectoryProcessor(loggerEntry, 5, nil)
	mockGH := &gh.MockClient{}

	engine := &Engine{
		gh: mockGH,
	}

	target := config.TargetConfig{
		Repo: "org/test-repo",
	}

	sourceState := &state.SourceState{
		Repo:         "org/source-repo",
		Branch:       "main",
		LatestCommit: "abc123",
	}

	// Test directory mapping with delete flag
	dirMapping := config.DirectoryMapping{
		Src:    "",
		Dest:   "old-docs",
		Delete: true,
	}

	// Setup mocks
	gitTree := &gh.GitTree{
		SHA: "tree456",
		Tree: []gh.GitTreeNode{
			{
				Path: "old-docs/index.md",
				Type: "blob",
				SHA:  "indexsha",
			},
		},
	}

	// No branch is configured, so the default branch is listed as HEAD
	mockGH.On("GetGitTree", mock.Anything, "org/test-repo", defaultBranchRef, true).
		Return(gitTree, nil).Once()

	mockGH.On("GetFile", mock.Anything, "org/test-repo", "old-docs/index.md", "").
		Return(&gh.FileContent{Content: []byte("# Old Documentation")}, nil)

	ctx := context.Background()
	changes, err := processor.ProcessDirectoryMapping(ctx, "", dirMapping, target, sourceState, engine)

	require.NoError(t, err)
	assert.Len(t, changes, 1)
	assert.True(t, changes[0].IsDeleted)
	assert.Equal(t, "old-docs/index.md", changes[0].Path)

	mockGH.AssertExpectations(t)
}
