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
	}), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		destPath := args[2].(string)
		testutil.CreateTestDirectory(t, destPath)
		testutil.WriteTestFile(t, destPath+"/README.md", "# Test Project\nUpdated content")
		testutil.WriteTestFile(t, destPath+"/config.yml", "version: 2\ntest: true")
	})

	// Mock git clone operations for Group A targets (they will be processed)
	gitClient.On("Clone", mock.Anything, "https://github.com/org/target-a1.git", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		testutil.CreateTestDirectory(t, args[2].(string))
	})
	gitClient.On("Clone", mock.Anything, "https://github.com/org/target-a2.git", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
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
	gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"changed-file.txt"}, nil)

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
	engine := NewEngine(context.Background(), cfg, ghClient, gitClient, stateDiscoverer, transformChain, DefaultOptions())
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

// TestOrchestratorIntegration_ExistingBranchRecovery tests end-to-end scenarios
// where sync operations encounter existing branches and recover gracefully
func TestOrchestratorIntegration_ExistingBranchRecovery(t *testing.T) {
	// Create temporary directory structure for testing
	tmpDir := testutil.CreateTempDir(t)
	sourceDir := tmpDir + "/source"
	testutil.CreateTestDirectory(t, sourceDir)

	// Create test files in source
	testutil.WriteTestFile(t, sourceDir+"/README.md", "# Updated Project\nContent from failed sync")
	testutil.WriteTestFile(t, sourceDir+"/config.yml", "version: 3\nrecovery: true")

	ctx := context.Background()

	t.Run("force push recovery after failed sync", func(t *testing.T) {
		// Setup mock clients
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		// Mock git operations with branch conflict and recovery
		gitClient.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, destPath+"/README.md", "# Updated Project\nContent from failed sync")
			testutil.WriteTestFile(t, destPath+"/config.yml", "version: 3\nrecovery: true")
		})
		gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), "commit456").Return(nil)
		gitClient.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("newcommit123", nil)
		gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"README.md", "config.yml"}, nil)

		// Simulate existing branch scenario: first push fails, force push succeeds
		gitClient.On("Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), false).Return(git.ErrBranchAlreadyExists)
		gitClient.On("Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), true).Return(nil)

		// Mock GitHub operations
		ghClient.On("ListBranches", mock.Anything, "test/target-repo").Return([]gh.Branch{{Name: "master"}}, nil)
		ghClient.On("GetFile", mock.Anything, "test/target-repo", mock.AnythingOfType("string"), "").Return(nil, gh.ErrFileNotFound)
		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser"}, nil)
		ghClient.On("CreatePR", mock.Anything, "test/target-repo", mock.AnythingOfType("gh.PRRequest")).Return(&gh.PR{Number: 42, Title: "Recovery PR"}, nil)

		// Mock state operations
		mockState := &state.State{
			Source: state.SourceState{
				Repo:         "test/source-repo",
				Branch:       "main",
				LatestCommit: "commit456",
			},
			Targets: map[string]*state.TargetState{
				"test/target-repo": {
					Repo:           "test/target-repo",
					LastSyncCommit: "old-commit",
					Status:         state.StatusBehind,
				},
			},
		}
		stateDiscoverer.On("DiscoverState", mock.Anything, mock.Anything).Return(mockState, nil)

		// Mock transform operations
		transformChain.On("Transform", mock.Anything, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("transform.Context")).Return([]byte("transformed content"), nil)

		// Configure test with single group that should encounter branch conflict
		cfg := &config.Config{
			Version: 1,
			Name:    "existing-branch-test",
			Groups: []config.Group{
				{
					ID:       "recovery-group",
					Name:     "Branch Recovery Test",
					Priority: 1,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   "test/source-repo",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: "test/target-repo",
							Files: []config.FileMapping{
								{Src: "README.md", Dest: "README.md"},
								{Src: "config.yml", Dest: "config.yml"},
							},
							Transform: config.Transform{
								RepoName: true,
							},
						},
					},
					Defaults: config.DefaultConfig{
						BranchPrefix: "chore/sync-files",
					},
				},
			},
		}

		// Create engine with test configuration
		engine := &Engine{
			config:    cfg,
			git:       gitClient,
			gh:        ghClient,
			state:     stateDiscoverer,
			transform: transformChain,
			logger:    logrus.New(),
			options:   &Options{CleanupTempFiles: false, DryRun: false, Force: false, MaxConcurrency: 5},
		}

		// Execute sync via orchestrator
		orchestrator := NewGroupOrchestrator(cfg, engine, logrus.New())
		err := orchestrator.ExecuteGroups(ctx, cfg.Groups)

		// Should complete successfully despite branch conflict
		require.NoError(t, err)

		// Verify recovery behavior
		gitClient.AssertExpectations(t)
		ghClient.AssertExpectations(t)

		// Verify both push attempts were made (normal, then force)
		gitClient.AssertCalled(t, "Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), false)
		gitClient.AssertCalled(t, "Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), true)

		// Verify PR was created after successful recovery
		ghClient.AssertCalled(t, "CreatePR", mock.Anything, "test/target-repo", mock.AnythingOfType("gh.PRRequest"))

		// Verify group status is success after recovery
		groupStatus, exists := orchestrator.GetGroupStatusByID("recovery-group")
		require.True(t, exists)
		assert.Equal(t, "success", groupStatus.State)
	})

	t.Run("branch exists locally - checkout existing and continue", func(t *testing.T) {
		// Setup mock clients
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		// Mock git operations with local branch conflict
		gitClient.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, destPath+"/README.md", "# Local Branch Test\nContent for local recovery")
		})
		gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), "commit789").Return(nil)
		// CreateBranch fails because branch exists locally
		gitClient.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(git.ErrBranchAlreadyExists)
		// Checkout existing branch succeeds
		gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("localcommit456", nil)
		gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"README.md"}, nil)
		gitClient.On("Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), false).Return(nil)

		// Mock GitHub operations
		ghClient.On("ListBranches", mock.Anything, "test/target-repo2").Return([]gh.Branch{{Name: "master"}}, nil)
		ghClient.On("GetFile", mock.Anything, "test/target-repo2", mock.AnythingOfType("string"), "").Return(nil, gh.ErrFileNotFound)
		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser"}, nil)
		ghClient.On("CreatePR", mock.Anything, "test/target-repo2", mock.AnythingOfType("gh.PRRequest")).Return(&gh.PR{Number: 43, Title: "Local Recovery PR"}, nil)

		// Mock state operations
		mockState := &state.State{
			Source: state.SourceState{
				Repo:         "test/source-repo2",
				Branch:       "main",
				LatestCommit: "commit789",
			},
			Targets: map[string]*state.TargetState{
				"test/target-repo2": {
					Repo:           "test/target-repo2",
					LastSyncCommit: "old-commit-2",
					Status:         state.StatusBehind,
				},
			},
		}
		stateDiscoverer.On("DiscoverState", mock.Anything, mock.Anything).Return(mockState, nil)

		// Mock transform operations
		transformChain.On("Transform", mock.Anything, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("transform.Context")).Return([]byte("transformed content"), nil)

		cfg := &config.Config{
			Version: 1,
			Name:    "local-branch-test",
			Groups: []config.Group{
				{
					ID:       "local-recovery-group",
					Name:     "Local Branch Recovery Test",
					Priority: 1,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   "test/source-repo2",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: "test/target-repo2",
							Files: []config.FileMapping{
								{Src: "README.md", Dest: "README.md"},
							},
						},
					},
					Defaults: config.DefaultConfig{
						BranchPrefix: "chore/sync-files",
					},
				},
			},
		}

		engine := &Engine{
			config:    cfg,
			git:       gitClient,
			gh:        ghClient,
			state:     stateDiscoverer,
			transform: transformChain,
			logger:    logrus.New(),
			options:   &Options{CleanupTempFiles: false, DryRun: false, Force: false, MaxConcurrency: 5},
		}

		orchestrator := NewGroupOrchestrator(cfg, engine, logrus.New())
		err := orchestrator.ExecuteGroups(ctx, cfg.Groups)

		require.NoError(t, err)

		// Verify local branch recovery behavior
		gitClient.AssertExpectations(t)
		ghClient.AssertExpectations(t)

		// Verify CreateBranch was called and failed
		gitClient.AssertCalled(t, "CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"))
		// Verify checkout was called for branch recovery
		gitClient.AssertCalled(t, "Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"))
		// Verify PR was still created successfully
		ghClient.AssertCalled(t, "CreatePR", mock.Anything, "test/target-repo2", mock.AnythingOfType("gh.PRRequest"))

		groupStatus, exists := orchestrator.GetGroupStatusByID("local-recovery-group")
		require.True(t, exists)
		assert.Equal(t, "success", groupStatus.State)
	})
}
