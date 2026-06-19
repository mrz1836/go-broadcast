package sync

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// multiGroupConfig builds a config with three groups, each owning a single
// target repository, used to prove scope resolution narrows correctly.
func multiGroupConfig() *config.Config {
	return &config.Config{
		Version: 1,
		Groups: []config.Group{
			{
				ID:      "core",
				Name:    "Core",
				Source:  config.SourceConfig{Repo: "org/template", Branch: "master"},
				Targets: []config.TargetConfig{{Repo: "org/core-a"}},
			},
			{
				ID:      "libs",
				Name:    "Libs",
				Source:  config.SourceConfig{Repo: "org/template", Branch: "master"},
				Targets: []config.TargetConfig{{Repo: "org/lib-a"}},
			},
			{
				ID:      "apps",
				Name:    "Apps",
				Source:  config.SourceConfig{Repo: "org/template", Branch: "master"},
				Targets: []config.TargetConfig{{Repo: "org/app-a"}},
			},
		},
	}
}

func TestResolveScope_TargetFilterNarrowsMultiGroup(t *testing.T) {
	// SC-1: a >=2-group config plus a single target arg resolves to exactly one
	// group / one repo.
	cfg := multiGroupConfig()

	scope, err := ResolveScope(cfg, DefaultOptions(), []string{"org/lib-a"})
	require.NoError(t, err)

	assert.Equal(t, 1, scope.GroupCount)
	assert.Equal(t, 1, scope.RepoCount)
	assert.Equal(t, []string{"org/lib-a"}, scope.Repos)
	assert.Equal(t, []string{"Libs"}, scope.Groups)
	require.Len(t, scope.Config.Groups, 1)
	assert.Equal(t, "libs", scope.Config.Groups[0].ID)
}

func TestResolveScope_GroupFilterNarrowsMultiGroup(t *testing.T) {
	// SC-2: --groups core against a multi-group config keeps only core in scope.
	cfg := multiGroupConfig()
	opts := DefaultOptions().WithGroupFilter([]string{"core"})

	scope, err := ResolveScope(cfg, opts, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, scope.GroupCount)
	assert.Equal(t, []string{"Core"}, scope.Groups)
	assert.Equal(t, []string{"org/core-a"}, scope.Repos)
}

func TestResolveScope_SkipGroupsPrecedence(t *testing.T) {
	// SC-2: --skip-groups removes a group even when also named in --groups.
	cfg := multiGroupConfig()
	opts := DefaultOptions().
		WithGroupFilter([]string{"core", "libs"}).
		WithSkipGroups([]string{"libs"})

	scope, err := ResolveScope(cfg, opts, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, scope.GroupCount)
	assert.Equal(t, []string{"Core"}, scope.Groups)
	assert.Equal(t, []string{"org/core-a"}, scope.Repos)
}

func TestResolveScope_NonMatchingTargetReturnsError(t *testing.T) {
	// A target filter matching nothing fails closed with ErrNoMatchingTargets.
	cfg := multiGroupConfig()

	_, err := ResolveScope(cfg, DefaultOptions(), []string{"org/does-not-exist"})
	require.Error(t, err)
	assert.ErrorIs(t, err, appErrors.ErrNoMatchingTargets)
}

func TestResolveScope_DisabledGroupsExcluded(t *testing.T) {
	cfg := multiGroupConfig()
	disabled := false
	cfg.Groups[1].Enabled = &disabled // disable "libs"

	scope, err := ResolveScope(cfg, DefaultOptions(), nil)
	require.NoError(t, err)

	assert.Equal(t, 2, scope.GroupCount)
	assert.Equal(t, []string{"Core", "Apps"}, scope.Groups)
}

func TestResolveScope_NoFiltersPreservesConfigPointer(t *testing.T) {
	// With no filters and all groups enabled, the original config pointer is
	// preserved so downstream consumers keying off it are unaffected.
	cfg := multiGroupConfig()

	scope, err := ResolveScope(cfg, DefaultOptions(), nil)
	require.NoError(t, err)

	assert.Same(t, cfg, scope.Config)
	assert.Equal(t, 3, scope.GroupCount)
	assert.Equal(t, 3, scope.RepoCount)
}

func TestResolveScope_NilConfig(t *testing.T) {
	scope, err := ResolveScope(nil, DefaultOptions(), nil)
	require.NoError(t, err)
	assert.Equal(t, 0, scope.RepoCount)
	assert.Nil(t, scope.Config)
}

func TestResolvedScope_Summary(t *testing.T) {
	// SC-5: the summary lists the in-scope groups and the repo count.
	cfg := multiGroupConfig()
	scope, err := ResolveScope(cfg, DefaultOptions(), nil)
	require.NoError(t, err)

	summary := scope.Summary()
	assert.Contains(t, summary, "3 group(s), 3 repo(s)")
	assert.Contains(t, summary, "Core")
	assert.Contains(t, summary, "org/core-a")
	assert.Contains(t, summary, "Estimated PRs/writes: 3")
}

func TestResolveEstimateGroups_MatchesResolveScope(t *testing.T) {
	// The rate-limit estimate derives its groups from ResolveScope, so a
	// group filter narrows the estimate too (no drift).
	cfg := multiGroupConfig()
	opts := DefaultOptions().WithGroupFilter([]string{"core"})

	groups := resolveEstimateGroups(cfg, opts)
	require.Len(t, groups, 1)
	assert.Equal(t, "core", groups[0].ID)
}

// TestEngineSync_MultiGroupTargetFilterTouchesOneRepo proves SC-1 end-to-end:
// a multi-group config plus a single target arg runs the single-group path and
// touches exactly one repository.
func TestEngineSync_MultiGroupTargetFilterTouchesOneRepo(t *testing.T) {
	scope := output.CaptureOutput()
	defer scope.Restore()

	cfg := &config.Config{
		Version: 1,
		Groups: []config.Group{
			{
				ID:     "core",
				Name:   "Core",
				Source: config.SourceConfig{Repo: "org/template", Branch: "master"},
				Targets: []config.TargetConfig{
					{Repo: "org/core-a", Files: []config.FileMapping{{Src: "f.txt", Dest: "f.txt"}}},
				},
			},
			{
				ID:     "libs",
				Name:   "Libs",
				Source: config.SourceConfig{Repo: "org/template", Branch: "master"},
				Targets: []config.TargetConfig{
					{Repo: "org/lib-a", Files: []config.FileMapping{{Src: "f.txt", Dest: "f.txt"}}},
				},
			},
		},
	}

	ghClient := &gh.MockClient{}
	gitClient := &git.MockClient{}
	gitClient.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"f.txt"}, nil).Maybe()
	gitClient.On("Diff", mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()
	gitClient.On("DiffIgnoreWhitespace", mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()
	stateDiscoverer := &state.MockDiscoverer{}
	transformChain := &transform.MockChain{}

	ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
	ghClient.On("GetRateLimit", mock.Anything).Return(healthyRateLimit(), nil).Maybe()
	ghClient.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, gh.ErrFileNotFound).Maybe()
	ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "test-user"}, nil).Maybe()

	// State reports the targeted repo as up-to-date so the sync resolves to a
	// single repo and performs no writes; capturing the discovered config lets
	// us assert only the narrowed group reached state discovery.
	var discoveredRepos []string
	currentState := &state.State{
		Source: state.SourceState{Repo: "org/template", Branch: "master", LatestCommit: "abc123"},
		Targets: map[string]*state.TargetState{
			"org/lib-a": {Repo: "org/lib-a", LastSyncCommit: "abc123", Status: state.StatusUpToDate},
		},
	}
	stateDiscoverer.On("DiscoverState", mock.Anything, mock.Anything).Return(currentState, nil).Run(func(args mock.Arguments) {
		cfgArg, ok := args.Get(1).(*config.Config)
		require.True(t, ok)
		for _, g := range cfgArg.Groups {
			for _, target := range g.Targets {
				discoveredRepos = append(discoveredRepos, target.Repo)
			}
		}
	})

	engine := NewEngine(context.Background(), cfg, ghClient, gitClient, stateDiscoverer, transformChain, &Options{MaxConcurrency: 2, RateLimitPreflightEnabled: true, RateLimitPrimaryMarginPercent: 20, RateLimitSecondaryReserve: 10})
	engine.SetLogger(logrus.New())

	// Target only the repo living in the second group.
	err := engine.Sync(context.Background(), []string{"org/lib-a"})
	require.NoError(t, err)

	// Exactly one repo (the targeted one) reached state discovery — the
	// multi-group blast is gone.
	assert.Equal(t, []string{"org/lib-a"}, discoveredRepos)

	// The resolved-scope summary is printed before any write (SC-5).
	assert.Contains(t, scope.Stdout.String(), "1 group(s), 1 repo(s)")
	assert.Contains(t, scope.Stdout.String(), "org/lib-a")
}
