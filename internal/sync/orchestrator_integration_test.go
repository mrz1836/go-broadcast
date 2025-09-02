package sync

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/testutil"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// TestOrchestrator_MultiGroupSync tests a realistic scenario:
// - Group A has 2 targets with changes (should create 2 PRs)
// - Group B has 1 target with no changes (should skip)
func TestOrchestrator_MultiGroupSync(t *testing.T) {
	// Create temporary directory structure for testing
	tmpDir := testutil.CreateTempDir(t)
	sourceDir := tmpDir + "/source"
	testutil.CreateTestDirectory(t, sourceDir)

	// Create test files in source
	testutil.WriteTestFile(t, sourceDir+"/README.md", "# Test Project\nUpdated content")
	testutil.WriteTestFile(t, sourceDir+"/config.yml", "version: 2\ntest: true")

	// Setup mock clients
	ghClient := &gh.MockClient{}
	gitClient := &git.MockClient{}
	stateDiscoverer := &state.MockDiscoverer{}
	transformChain := &transform.MockChain{}

	// Configure test configuration with 2 groups
	cfg := &config.Config{
		Version: 1,
		Name:    "multi-group-test",
		Groups: []config.Group{
			{
				ID:       "group-a",
				Name:     "Group A - High Priority",
				Priority: 1,
				Enabled:  boolPtr(true),
				Source: config.SourceConfig{
					Repo:   "org/template",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "org/target-a1",
						Files: []config.FileMapping{
							{Src: "README.md", Dest: "README.md"},
						},
					},
					{
						Repo: "org/target-a2",
						Files: []config.FileMapping{
							{Src: "config.yml", Dest: "config.yml"},
						},
					},
				},
			},
			{
				ID:       "group-b",
				Name:     "Group B - Lower Priority",
				Priority: 2,
				Enabled:  boolPtr(true),
				Source: config.SourceConfig{
					Repo:   "org/template",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "org/target-b1",
						Files: []config.FileMapping{
							{Src: "README.md", Dest: "README.md"},
						},
					},
				},
			},
		},
	}

	// Setup state discovery mock - Group A targets need sync, Group B is up-to-date
	mockState := &state.State{
		Source: state.SourceState{
			Repo:         "org/template",
			Branch:       "main",
			LatestCommit: "source-commit-123",
		},
		Targets: map[string]*state.TargetState{
			"org/target-a1": {
				Repo:           "org/target-a1",
				LastSyncCommit: "old-commit-a1",
				Status:         state.StatusBehind,
			},
			"org/target-a2": {
				Repo:           "org/target-a2",
				LastSyncCommit: "old-commit-a2",
				Status:         state.StatusBehind,
			},
			"org/target-b1": {
				Repo:           "org/target-b1",
				LastSyncCommit: "source-commit-123", // Same as source - up to date
				Status:         state.StatusUpToDate,
			},
		},
	}

	stateDiscoverer.On("DiscoverState", mock.Anything, mock.Anything).Return(mockState, nil)

	// Mock GitHub operations for all repositories
	ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()

	// Mock GetFile for checking existing content (files don't exist initially, so they'll be synced)
	ghClient.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, gh.ErrFileNotFound)

	// Mock GetCurrentUser for PR creation
	ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser"}, nil)

	// Mock git clone operations for source (using full GitHub URL)
	gitClient.On("Clone", mock.Anything, "https://github.com/org/template.git", mock.MatchedBy(func(_ string) bool {
		return true // Accept any path for source clones
	})).Return(nil).Run(func(args mock.Arguments) {
		destPath := args[2].(string)
		testutil.CreateTestDirectory(t, destPath)
		testutil.WriteTestFile(t, destPath+"/README.md", "# Test Project\nUpdated content")
		testutil.WriteTestFile(t, destPath+"/config.yml", "version: 2\ntest: true")
	})

	// Mock git clone operations for Group A targets (they will be processed)
	gitClient.On("Clone", mock.Anything, "https://github.com/org/target-a1.git", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		testutil.CreateTestDirectory(t, args[2].(string))
	})
	gitClient.On("Clone", mock.Anything, "https://github.com/org/target-a2.git", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		testutil.CreateTestDirectory(t, args[2].(string))
	})

	// Group B target won't be cloned since it's up-to-date

	// Mock checkout operations - first for source commit, then for sync branches
	gitClient.On("Checkout", mock.Anything, mock.Anything, "source-commit-123").Return(nil)
	gitClient.On("Checkout", mock.Anything, mock.Anything, mock.MatchedBy(func(_ string) bool {
		// Accept any sync branch name
		return true
	})).Return(nil)
	gitClient.On("CreateBranch", mock.Anything, mock.Anything, mock.MatchedBy(func(_ string) bool {
		return true // Accept any branch name
	})).Return(nil)
	gitClient.On("CheckoutBranch", mock.Anything, mock.Anything, mock.MatchedBy(func(_ string) bool {
		return true
	})).Return(nil).Maybe()

	// Mock commit and push operations (only for Group A targets that have changes)
	gitClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	gitClient.On("Commit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	gitClient.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.Anything).Return("commit-sha-123", nil)

	// Mock PR creation (should be called twice for Group A targets)
	prCreateCount := 0
	ghClient.On("CreatePR", mock.Anything, mock.Anything, mock.MatchedBy(func(_ gh.PRRequest) bool {
		prCreateCount++
		return true // Accept any PR creation for Group A targets
	})).Return(&gh.PR{
		Number: prCreateCount,
	}, nil)

	// Mock transform operations (maybe - not all files may need transformation)
	transformChain.On("Transform", mock.Anything, mock.Anything, mock.Anything).Return(func(_ context.Context, content []byte, _ config.TargetConfig) []byte {
		return content // Return content as-is for simplicity
	}, nil).Maybe()

	// Create engine with mocks
	engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, DefaultOptions())
	engine.SetLogger(logrus.New())

	// Execute the sync
	err := engine.Sync(context.Background(), []string{})

	// Verify results
	require.NoError(t, err)

	// Verify that exactly 2 PRs were created (for Group A targets)
	ghClient.AssertNumberOfCalls(t, "CreatePR", 2)

	// Verify mock expectations
	stateDiscoverer.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	gitClient.AssertExpectations(t)
	transformChain.AssertExpectations(t)
}

// TestOrchestrator_GroupSkippedDueToDependencies tests group dependency handling
func TestOrchestrator_GroupSkippedDueToDependencies(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Groups: []config.Group{
			{
				ID:       "group-base",
				Name:     "Base Group",
				Priority: 1,
				Enabled:  boolPtr(true),
				Source:   config.SourceConfig{Repo: "org/source"},
				Targets:  []config.TargetConfig{{Repo: "org/base-target"}},
			},
			{
				ID:        "group-dependent",
				Name:      "Dependent Group",
				Priority:  2,
				DependsOn: []string{"group-base"}, // Depends on group-base
				Enabled:   boolPtr(true),
				Source:    config.SourceConfig{Repo: "org/source"},
				Targets:   []config.TargetConfig{{Repo: "org/dependent-target"}},
			},
		},
	}

	engine := &Engine{config: cfg, logger: logrus.New()}
	orch := NewGroupOrchestrator(cfg, engine, logrus.New())

	// Mock executor that fails group-base
	executor := &testGroupExecutor{
		errorsToReturn: map[string]error{
			"group-base": ErrGroupFailed,
		},
	}
	orch.executeGroup = executor.executeGroup

	err := orch.ExecuteGroups(context.Background(), cfg.Groups)

	// Should return error due to failed group
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 groups failed")

	// Verify execution - base group executed and failed, dependent skipped
	assert.Contains(t, executor.executedGroups, "group-base")
	assert.NotContains(t, executor.executedGroups, "group-dependent")

	// Verify status tracking
	baseStatus, exists := orch.GetGroupStatusByID("group-base")
	assert.True(t, exists)
	assert.Equal(t, "failed", baseStatus.State)

	depStatus, exists := orch.GetGroupStatusByID("group-dependent")
	assert.True(t, exists)
	assert.Equal(t, "skipped", depStatus.State)
	assert.Equal(t, "Dependencies failed", depStatus.Message)
}
