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

	suite.processor = NewDirectoryProcessor(suite.logger, 5)
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

	// Create a mock tree with files in the directory
	mockTreeMap := &TreeMap{
		files: map[string]*GitTreeNode{
			"docs/README.md": {
				Path: "docs/README.md",
				Type: "blob",
				SHA:  "file1sha",
			},
			"docs/guide.md": {
				Path: "docs/guide.md",
				Type: "blob",
				SHA:  "file2sha",
			},
			"docs/api/endpoints.md": {
				Path: "docs/api/endpoints.md",
				Type: "blob",
				SHA:  "file3sha",
			},
		},
		directories: map[string]bool{
			"docs":     true,
			"docs/api": true,
		},
	}

	// Mock GitHub API calls
	suite.setupTreeAPIMock(mockTreeMap)

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

	// Create empty tree map (no files in directory)
	mockTreeMap := &TreeMap{
		files:       map[string]*GitTreeNode{},
		directories: map[string]bool{},
	}

	suite.setupTreeAPIMock(mockTreeMap)

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

	// Create a mock tree with deeply nested structure
	mockTreeMap := &TreeMap{
		files: map[string]*GitTreeNode{
			"src/main.go": {
				Path: "src/main.go",
				Type: "blob",
				SHA:  "main_sha",
			},
			"src/utils/helper.go": {
				Path: "src/utils/helper.go",
				Type: "blob",
				SHA:  "helper_sha",
			},
			"src/api/handlers/user.go": {
				Path: "src/api/handlers/user.go",
				Type: "blob",
				SHA:  "user_sha",
			},
			"src/api/handlers/auth.go": {
				Path: "src/api/handlers/auth.go",
				Type: "blob",
				SHA:  "auth_sha",
			},
			"src/config/app.yaml": {
				Path: "src/config/app.yaml",
				Type: "blob",
				SHA:  "config_sha",
			},
		},
		directories: map[string]bool{
			"src":              true,
			"src/utils":        true,
			"src/api":          true,
			"src/api/handlers": true,
			"src/config":       true,
		},
	}

	suite.setupTreeAPIMock(mockTreeMap)

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

	mockTreeMap := &TreeMap{
		files: map[string]*GitTreeNode{
			"temp/file1.txt": {
				Path: "temp/file1.txt",
				Type: "blob",
				SHA:  "file1sha",
			},
			"temp/file2.txt": {
				Path: "temp/file2.txt",
				Type: "blob",
				SHA:  "file2sha",
			},
		},
		directories: map[string]bool{
			"temp": true,
		},
	}

	suite.setupTreeAPIMock(mockTreeMap)

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

// setupTreeAPIMock sets up mock calls for GitHub Tree API
func (suite *DirectoryDeleteTestSuite) setupTreeAPIMock(treeMap *TreeMap) {
	// Mock commit lookup
	commit := &gh.Commit{
		SHA: "commit123",
	}

	suite.mockGH.On("GetCommit", mock.Anything, "org/target-repo", "").
		Return(commit, nil)

	// Convert TreeMap to GitHub Tree structure for mock
	treeNodes := make([]gh.GitTreeNode, 0, len(treeMap.files))
	for path, node := range treeMap.files {
		treeNodes = append(treeNodes, gh.GitTreeNode{
			Path: path,
			Type: node.Type,
			SHA:  node.SHA,
		})
	}

	gitTree := &gh.GitTree{
		SHA:  "tree123",
		Tree: treeNodes,
	}

	// GetGitTree is called with the commit SHA, not the tree SHA
	suite.mockGH.On("GetGitTree", mock.Anything, "org/target-repo", "commit123", true).
		Return(gitTree, nil)
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

	processor := NewDirectoryProcessor(loggerEntry, 5)
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

	// We don't need the TreeMap here since we're mocking the GitHub API calls directly

	// Setup mocks
	commit := &gh.Commit{
		SHA: "commit456",
	}

	mockGH.On("GetCommit", mock.Anything, "org/test-repo", "").
		Return(commit, nil)

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

	mockGH.On("GetGitTree", mock.Anything, "org/test-repo", "commit456", true).
		Return(gitTree, nil)

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
