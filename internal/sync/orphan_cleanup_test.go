package sync

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// cleanupTargetRepo is the target every cleanup test syncs against
const cleanupTargetRepo = "org/repo"

// syncBranchAged builds a sync branch name for the target repo with a timestamp the
// given duration in the past.
func syncBranchAged(age time.Duration) string {
	return state.FormatSyncBranchName(
		"chore/sync-files",
		state.FormatBranchScope(cleanupTargetRepo),
		time.Now().Add(-age),
		"abc1234",
	)
}

// newCleanupSync wires a RepositorySync with the given branches and open PRs
func newCleanupSync(ghClient gh.Client, openPRs []gh.PR) *RepositorySync {
	return &RepositorySync{
		engine: &Engine{
			gh: ghClient,
			config: &config.Config{
				Groups: []config.Group{
					{
						Defaults: config.DefaultConfig{BranchPrefix: "chore/sync-files"},
					},
				},
			},
		},
		target:      config.TargetConfig{Repo: cleanupTargetRepo},
		targetState: &state.TargetState{OpenPRs: openPRs},
		logger:      logrus.NewEntry(logrus.New()),
	}
}

func prForBranch(branch string) gh.PR {
	pr := gh.PR{Number: 1, State: "open"}
	pr.Head.Ref = branch

	return pr
}

// TestOrphanCleanup_SkipsBranchesThatOnlyShareThePrefix is the important guard: a
// branch a human created under the sync prefix must never be deleted, because the
// cleanup runs unattended before every sync.
func TestOrphanCleanup_SkipsBranchesThatOnlyShareThePrefix(t *testing.T) {
	humanBranches := []string{
		"chore/sync-files-manual-fix",
		"chore/sync-files-wip",
		"chore/sync-files",
		"chore/sync-files-experiment-2024",
	}

	for _, branch := range humanBranches {
		t.Run(branch, func(t *testing.T) {
			ghClient := &TestValidationMockGHClient{
				branches: []gh.Branch{{Name: branch}},
			}

			rs := newCleanupSync(ghClient, nil)

			require.NoError(t, rs.validateAndCleanupOrphanedBranches(context.Background()))
			assert.Empty(t, ghClient.deletedBranch, "branch %q is not a generated sync branch and must be left alone", branch)
		})
	}
}

// TestOrphanCleanup_SkipsRecentBranches protects a concurrent sync run that has
// pushed its branch but has not opened its PR yet - exactly the window in which the
// 502 failure leaves a branch behind.
func TestOrphanCleanup_SkipsRecentBranches(t *testing.T) {
	recent := syncBranchAged(time.Minute)

	ghClient := &TestValidationMockGHClient{
		branches: []gh.Branch{{Name: recent}},
	}

	rs := newCleanupSync(ghClient, nil)

	require.NoError(t, rs.validateAndCleanupOrphanedBranches(context.Background()))
	assert.Empty(t, ghClient.deletedBranch, "a branch younger than the grace period must not be deleted")
}

// TestOrphanCleanup_DeletesAgedOrphans confirms the guards did not disable cleanup
func TestOrphanCleanup_DeletesAgedOrphans(t *testing.T) {
	aged := syncBranchAged(orphanedBranchMinAge + time.Hour)

	ghClient := &TestValidationMockGHClient{
		branches: []gh.Branch{{Name: aged}},
	}

	rs := newCleanupSync(ghClient, nil)

	require.NoError(t, rs.validateAndCleanupOrphanedBranches(context.Background()))
	assert.Equal(t, aged, ghClient.deletedBranch)
}

// TestOrphanCleanup_KeepsBranchesWithOpenPRs verifies an aged branch backing a live
// PR is preserved
func TestOrphanCleanup_KeepsBranchesWithOpenPRs(t *testing.T) {
	aged := syncBranchAged(orphanedBranchMinAge + time.Hour)

	ghClient := &TestValidationMockGHClient{
		branches: []gh.Branch{{Name: aged}},
	}

	rs := newCleanupSync(ghClient, []gh.PR{prForBranch(aged)})

	require.NoError(t, rs.validateAndCleanupOrphanedBranches(context.Background()))
	assert.Empty(t, ghClient.deletedBranch)
}

// TestOrphanCleanup_BoundaryAtMinAge pins behavior either side of the grace period
func TestOrphanCleanup_BoundaryAtMinAge(t *testing.T) {
	tests := []struct {
		name         string
		age          time.Duration
		expectDelete bool
	}{
		{name: "just inside grace period", age: orphanedBranchMinAge - time.Minute, expectDelete: false},
		{name: "just outside grace period", age: orphanedBranchMinAge + time.Minute, expectDelete: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch := syncBranchAged(tt.age)
			ghClient := &TestValidationMockGHClient{
				branches: []gh.Branch{{Name: branch}},
			}

			rs := newCleanupSync(ghClient, nil)

			require.NoError(t, rs.validateAndCleanupOrphanedBranches(context.Background()))

			if tt.expectDelete {
				assert.Equal(t, branch, ghClient.deletedBranch)
			} else {
				assert.Empty(t, ghClient.deletedBranch)
			}
		})
	}
}

// TestOrphanCleanup_HandlesLegacyGroupScopedBranches ensures branches created under
// the previous group-scoped naming are still recognized and cleaned up.
func TestOrphanCleanup_HandlesLegacyGroupScopedBranches(t *testing.T) {
	legacy := "chore/sync-files-owner-forks-20240115-120530-ee542e5"

	ghClient := &TestValidationMockGHClient{
		branches: []gh.Branch{{Name: legacy}},
	}

	rs := newCleanupSync(ghClient, nil)

	require.NoError(t, rs.validateAndCleanupOrphanedBranches(context.Background()))
	assert.Equal(t, legacy, ghClient.deletedBranch)
}

// TestCreateSyncBranch_ScopedToTargetRepo verifies the generated branch identifies
// the target repository and does not carry the group name.
func TestCreateSyncBranch_ScopedToTargetRepo(t *testing.T) {
	rs := &RepositorySync{
		engine: &Engine{
			config: &config.Config{
				Groups: []config.Group{
					{
						ID: "owner-forks",
						Defaults: config.DefaultConfig{
							BranchPrefix: "chore/sync-files",
						},
					},
				},
			},
			options: &Options{DryRun: true},
		},
		target:      config.TargetConfig{Repo: "owner/repo-one"},
		sourceState: &state.SourceState{LatestCommit: "ee542e508233d6a91ca83c444cdefb557e6cc0c4"},
		logger:      logrus.NewEntry(logrus.New()),
	}

	branch := rs.createSyncBranch(context.Background())

	assert.Contains(t, branch, "chore/sync-files-repo-one-")
	assert.NotContains(t, branch, "owner-forks", "group ID must not appear in the branch name")
	assert.Contains(t, branch, "ee542e5", "short source SHA should be present")

	metadata, err := state.ParseSyncBranchName(branch, "chore/sync-files")
	require.NoError(t, err, "generated branch must be parseable")
	assert.Equal(t, "repo-one", metadata.Scope)
	assert.Equal(t, "ee542e5", metadata.CommitSHA)
}

// TestCreateSyncBranch_DistinctPerTargetInSameSecond reproduces the condition behind
// the identical names observed across targets: same group, same source commit, same
// second. Scoping by target repo is what keeps them distinct.
func TestCreateSyncBranch_DistinctPerTargetInSameSecond(t *testing.T) {
	targets := []string{
		"owner/repo-two",
		"owner/repo-one",
		"owner/repo-three",
		"owner/repo-four",
	}

	engine := &Engine{
		config: &config.Config{
			Groups: []config.Group{
				{
					ID:       "owner-forks",
					Defaults: config.DefaultConfig{BranchPrefix: "chore/sync-files"},
				},
			},
		},
		options: &Options{DryRun: true},
	}

	seen := make(map[string]string, len(targets))
	for _, repo := range targets {
		rs := &RepositorySync{
			engine:      engine,
			target:      config.TargetConfig{Repo: repo},
			sourceState: &state.SourceState{LatestCommit: "ee542e508233d6a91ca83c444cdefb557e6cc0c4"},
			logger:      logrus.NewEntry(logrus.New()),
		}

		branch := rs.createSyncBranch(context.Background())
		if previous, collided := seen[branch]; collided {
			t.Fatalf("branch collision between %s and %s: %s", previous, repo, branch)
		}
		seen[branch] = repo
	}

	assert.Len(t, seen, len(targets), "expected one unique branch per target, got %v", seen)
}
