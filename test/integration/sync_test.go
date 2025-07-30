package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/sync"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestEndToEndSync tests the complete sync workflow with mocked clients
func TestEndToEndSync(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Version: 1,
		Source: config.SourceConfig{
			Repo:   "org/template-repo",
			Branch: "master",
		},
		Defaults: config.DefaultConfig{
			BranchPrefix: "chore/sync-files",
			PRLabels:     []string{"automated-sync"},
		},
		Targets: []config.TargetConfig{
			{
				Repo: "org/service-a",
				Files: []config.FileMapping{
					{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
					{Src: "Makefile", Dest: "Makefile"},
				},
				Transform: config.Transform{
					RepoName:  true,
					Variables: map[string]string{"SERVICE_NAME": "service-a"},
				},
			},
			{
				Repo: "org/service-b",
				Files: []config.FileMapping{
					{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
				},
				Transform: config.Transform{
					RepoName: true,
				},
			},
		},
	}

	t.Run("successful sync with outdated targets", func(t *testing.T) {
		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// Configure state discovery expectations - return no targets needing sync
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template-repo",
				Branch:       "master",
				LatestCommit: "abc123def456",
				LastChecked:  time.Now(),
			},
			Targets: map[string]*state.TargetState{
				"org/service-a": {
					Repo:           "org/service-a",
					LastSyncCommit: "abc123def456", // Same as source - up to date
					Status:         state.StatusUpToDate,
					LastSyncTime:   &[]time.Time{time.Now().Add(-1 * time.Hour)}[0],
				},
				"org/service-b": {
					Repo:           "org/service-b",
					LastSyncCommit: "abc123def456", // Same as source - up to date
					Status:         state.StatusUpToDate,
					LastSyncTime:   &[]time.Time{time.Now().Add(-2 * time.Hour)}[0],
				},
			},
		}

		mockState.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		// Create sync engine
		opts := sync.DefaultOptions().WithDryRun(false).WithMaxConcurrency(2)
		engine := sync.NewEngine(cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logrus.New())

		// Execute sync - should succeed without doing any sync work since targets are up-to-date
		err := engine.Sync(context.Background(), nil)

		// Verify results - should succeed without errors
		require.NoError(t, err)
		mockState.AssertExpectations(t)

		// Since targets are up-to-date, no GitHub or Git operations should be called
		mockGH.AssertNotCalled(t, "GetFile")
		mockGH.AssertNotCalled(t, "CreatePR")
		mockGit.AssertNotCalled(t, "Clone")
	})

	t.Run("sync with up-to-date targets", func(t *testing.T) {
		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// All targets are up-to-date
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template-repo",
				Branch:       "master",
				LatestCommit: "abc123def456",
				LastChecked:  time.Now(),
			},
			Targets: map[string]*state.TargetState{
				"org/service-a": {
					Repo:           "org/service-a",
					LastSyncCommit: "abc123def456", // Same as source
					Status:         state.StatusUpToDate,
					LastSyncTime:   &[]time.Time{time.Now().Add(-1 * time.Hour)}[0],
				},
				"org/service-b": {
					Repo:           "org/service-b",
					LastSyncCommit: "abc123def456", // Same as source
					Status:         state.StatusUpToDate,
					LastSyncTime:   &[]time.Time{time.Now().Add(-2 * time.Hour)}[0],
				},
			},
		}

		mockState.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		// Create sync engine
		opts := sync.DefaultOptions().WithDryRun(false)
		engine := sync.NewEngine(cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logrus.New())

		// Execute sync
		err := engine.Sync(context.Background(), nil)

		// Should succeed without doing any work
		require.NoError(t, err)
		mockState.AssertExpectations(t)

		// Verify no GitHub or Git operations were called
		mockGH.AssertNotCalled(t, "GetFile")
		mockGH.AssertNotCalled(t, "CreatePR")
		mockGit.AssertNotCalled(t, "Clone")
	})

	t.Run("dry run mode", func(t *testing.T) {
		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// All targets are up-to-date to avoid actual sync execution
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template-repo",
				Branch:       "master",
				LatestCommit: "abc123def456",
				LastChecked:  time.Now(),
			},
			Targets: map[string]*state.TargetState{
				"org/service-a": {
					Repo:           "org/service-a",
					LastSyncCommit: "abc123def456", // Same as source - up to date
					Status:         state.StatusUpToDate,
				},
				"org/service-b": {
					Repo:           "org/service-b",
					LastSyncCommit: "abc123def456", // Same as source - up to date
					Status:         state.StatusUpToDate,
				},
			},
		}

		mockState.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		// Create sync engine with dry-run enabled
		opts := sync.DefaultOptions().WithDryRun(true)
		engine := sync.NewEngine(cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logrus.New())

		// Execute sync
		err := engine.Sync(context.Background(), nil)

		// Should succeed and not perform any sync operations (targets are up-to-date)
		require.NoError(t, err)
		mockState.AssertExpectations(t)

		// Verify no actual operations were called (no sync needed)
		mockGH.AssertNotCalled(t, "CreatePR")
		mockGit.AssertNotCalled(t, "Clone")
		mockGit.AssertNotCalled(t, "Push")
	})

	t.Run("error handling - state discovery failure", func(t *testing.T) {
		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// Mock state discovery failure
		mockState.On("DiscoverState", mock.Anything, cfg).
			Return(nil, assert.AnError)

		// Create sync engine
		opts := sync.DefaultOptions()
		engine := sync.NewEngine(cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logrus.New())

		// Execute sync
		err := engine.Sync(context.Background(), nil)

		// Should fail with state discovery error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to discover current state")
		mockState.AssertExpectations(t)
	})

	t.Run("concurrent sync processing", func(t *testing.T) {
		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// Multiple targets need sync
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template-repo",
				Branch:       "master",
				LatestCommit: "abc123def456",
				LastChecked:  time.Now(),
			},
			Targets: map[string]*state.TargetState{
				"org/service-a": {
					Repo:           "org/service-a",
					LastSyncCommit: "old123",
					Status:         state.StatusBehind,
				},
				"org/service-b": {
					Repo:           "org/service-b",
					LastSyncCommit: "old456",
					Status:         state.StatusBehind,
				},
			},
		}

		mockState.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		// Mock git operations for dry-run mode with behind targets
		mockGit.On("Clone", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			// Create the source files in the cloned directory
			destPath := args[2].(string)
			_ = os.MkdirAll(filepath.Join(destPath, ".github/workflows"), 0o750)
			_ = os.WriteFile(filepath.Join(destPath, ".github/workflows/ci.yml"), []byte("workflow content"), 0o600)
			_ = os.WriteFile(filepath.Join(destPath, "Makefile"), []byte("makefile content"), 0o600)
		})
		mockGit.On("Checkout", mock.Anything, mock.Anything, "abc123def456").Return(nil)

		// Mock getting source files
		mockGH.On("GetFile", mock.Anything, "org/template-repo", ".github/workflows/ci.yml", "").
			Return(&gh.FileContent{Content: []byte("workflow content")}, nil).Maybe()
		mockGH.On("GetFile", mock.Anything, "org/template-repo", "Makefile", "").
			Return(&gh.FileContent{Content: []byte("makefile content")}, nil).Maybe()

		// Mock getting target files for comparison
		mockGH.On("GetFile", mock.Anything, "org/service-a", ".github/workflows/ci.yml", "").
			Return(&gh.FileContent{Content: []byte("old workflow")}, nil).Maybe()
		mockGH.On("GetFile", mock.Anything, "org/service-a", "Makefile", "").
			Return(&gh.FileContent{Content: []byte("old makefile")}, nil).Maybe()
		mockGH.On("GetFile", mock.Anything, "org/service-b", ".github/workflows/ci.yml", "").
			Return(&gh.FileContent{Content: []byte("old workflow")}, nil).Maybe()

		// Mock transformations
		mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
			Return([]byte("transformed content"), nil).Maybe()

		// Mock GetCurrentUser for dry-run PR preview
		mockGH.On("GetCurrentUser", mock.Anything).
			Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()

		// Create sync engine with high concurrency
		opts := sync.DefaultOptions().WithDryRun(true).WithMaxConcurrency(10)
		engine := sync.NewEngine(cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logrus.New())

		// Execute sync
		err := engine.Sync(context.Background(), nil)

		// Should handle concurrent processing without issues
		require.NoError(t, err)
		mockState.AssertExpectations(t)
	})
}

// TestConfigurationLoading tests configuration loading and validation
func TestConfigurationLoading(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		// Create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "sync.yaml")

		configContent := `version: 1
source:
  repo: "org/template"
  branch: "master"
defaults:
  branch_prefix: "chore/sync-files"
  pr_labels: ["automated-sync"]
targets:
  - repo: "org/service"
    files:
      - src: ".github/workflows/ci.yml"
        dest: ".github/workflows/ci.yml"
    transform:
      repo_name: true
`
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		// Load configuration
		cfg, err := config.Load(configPath)
		require.NoError(t, err)

		// Validate configuration
		err = cfg.Validate()
		require.NoError(t, err)

		// Verify loaded values
		assert.Equal(t, 1, cfg.Version)
		assert.Equal(t, "org/template", cfg.Source.Repo)
		assert.Equal(t, "master", cfg.Source.Branch)
		assert.Len(t, cfg.Targets, 1)
		assert.Equal(t, "org/service", cfg.Targets[0].Repo)
		assert.True(t, cfg.Targets[0].Transform.RepoName)
	})

	t.Run("invalid configuration", func(t *testing.T) {
		// Create temporary config file with invalid content
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.yaml")

		configContent := `version: 1  # Unsupported version
source:
  repo: ""  # Empty repo
targets: []  # No targets
`
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		// Load configuration
		cfg, err := config.Load(configPath)
		require.NoError(t, err)

		// Validation should fail
		err = cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("missing configuration file", func(t *testing.T) {
		// Try to load non-existent file
		_, err := config.Load("/nonexistent/config.yaml")
		assert.Error(t, err)
	})
}

// TestStateDiscovery tests the state discovery system
func TestStateDiscovery(t *testing.T) {
	t.Run("branch name format", func(t *testing.T) {
		// Test branch name formatting
		timestamp := time.Date(2024, 1, 15, 12, 5, 30, 0, time.UTC)
		commitSHA := "abc123def456"
		prefix := "chore/sync-files"

		branchName := state.FormatSyncBranchName(prefix, timestamp, commitSHA)
		expected := "chore/sync-files-20240115-120530-abc123def456"

		assert.Equal(t, expected, branchName)
	})

	t.Run("branch prefix validation", func(t *testing.T) {
		testCases := []struct {
			name      string
			prefix    string
			expectErr bool
		}{
			{
				name:      "valid prefix",
				prefix:    "chore/sync-files",
				expectErr: false,
			},
			{
				name:      "empty prefix",
				prefix:    "",
				expectErr: true,
			},
			{
				name:      "invalid characters",
				prefix:    "chore/sync-files@invalid",
				expectErr: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := state.ValidateBranchPrefix(tc.prefix)

				if tc.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

// TestTransformEngine tests the transform engine functionality
func TestTransformEngine(t *testing.T) {
	t.Run("repository name transformation", func(t *testing.T) {
		transformer := transform.NewRepoTransformer()

		content := []byte(`module github.com/org/template-repo

go 1.21

require (
	github.com/org/template-repo/internal v0.0.0
)`)

		ctx := transform.Context{
			SourceRepo: "org/template-repo",
			TargetRepo: "org/service-a",
			FilePath:   "go.mod",
		}

		result, err := transformer.Transform(content, ctx)
		require.NoError(t, err)
		assert.Contains(t, string(result), "github.com/org/service-a")
		assert.NotContains(t, string(result), "github.com/org/template-repo")
	})

	t.Run("template variable transformation", func(t *testing.T) {
		logger := logrus.New()
		transformer := transform.NewTemplateTransformer(logger, nil)

		content := []byte(`SERVICE_NAME={{SERVICE_NAME}}
VERSION=${VERSION}
# Service: {{SERVICE_NAME}}`)

		ctx := transform.Context{
			SourceRepo: "org/template",
			TargetRepo: "org/service-a",
			FilePath:   "config.env",
			Variables: map[string]string{
				"SERVICE_NAME": "service-a",
				"VERSION":      "1.0.0",
			},
		}

		result, err := transformer.Transform(content, ctx)
		require.NoError(t, err)
		assert.Contains(t, string(result), "SERVICE_NAME=service-a")
		assert.Contains(t, string(result), "VERSION=1.0.0")
		assert.Contains(t, string(result), "# Service: service-a")
	})

	t.Run("transform chain", func(t *testing.T) {
		logger := logrus.New()
		chain := transform.NewChain(logger)

		chain.Add(transform.NewRepoTransformer())
		chain.Add(transform.NewTemplateTransformer(logger, nil))

		content := []byte(`module github.com/org/template
SERVICE={{SERVICE_NAME}}`)

		ctx := transform.Context{
			SourceRepo: "org/template",
			TargetRepo: "org/service-a",
			FilePath:   "go.mod",
			Variables: map[string]string{
				"SERVICE_NAME": "service-a",
			},
		}

		result, err := chain.Transform(context.Background(), content, ctx)
		require.NoError(t, err)
		assert.Contains(t, string(result), "github.com/org/service-a")
		assert.Contains(t, string(result), "SERVICE=service-a")
	})
}
