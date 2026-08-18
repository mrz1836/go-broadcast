package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatBranchScope(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		expected string
	}{
		{
			name:     "owner/repo keeps only the repo name",
			repo:     "owner/repo-one",
			expected: "repo-one",
		},
		{
			name:     "bare repo name passes through",
			repo:     "go-broadcast",
			expected: "go-broadcast",
		},
		{
			name:     "dots are replaced so the name stays parseable",
			repo:     "owner/go-broadcast.v2",
			expected: "go-broadcast-v2",
		},
		{
			name:     "underscores are preserved",
			repo:     "owner/my_repo",
			expected: "my_repo",
		},
		{
			name:     "runs of invalid characters collapse to a single dash",
			repo:     "owner/weird!!!name",
			expected: "weird-name",
		},
		{
			name:     "leading and trailing dashes are trimmed",
			repo:     "owner/.hidden.",
			expected: "hidden",
		},
		{
			name:     "empty repo falls back to a placeholder",
			repo:     "",
			expected: "unknown",
		},
		{
			name:     "trailing slash falls back to a placeholder",
			repo:     "owner/",
			expected: "unknown",
		},
		{
			name:     "name of only invalid characters falls back",
			repo:     "owner/...",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatBranchScope(tt.repo))
		})
	}
}

// TestFormatBranchScope_AlwaysProducesParseableBranch is the contract that matters:
// whatever a repo is named, the generated branch must survive a round trip.
func TestFormatBranchScope_AlwaysProducesParseableBranch(t *testing.T) {
	repos := []string{
		"owner/repo-one",
		"owner/go-broadcast.v2",
		"owner/my_repo",
		"owner/weird!!!name",
		"owner/UPPER-Case",
		"owner/123numeric",
		"",
	}

	timestamp := time.Date(2026, 7, 29, 10, 53, 18, 0, time.UTC)

	for _, repo := range repos {
		t.Run(repo, func(t *testing.T) {
			branch := FormatSyncBranchName("chore/sync-files", FormatBranchScope(repo), timestamp, "ee542e5")

			metadata, err := ParseSyncBranchName(branch, "chore/sync-files")
			require.NoError(t, err, "branch %q must parse", branch)
			assert.Equal(t, timestamp, metadata.Timestamp)
			assert.Equal(t, "ee542e5", metadata.CommitSHA)
			assert.Equal(t, FormatBranchScope(repo), metadata.Scope)
		})
	}
}

// TestBranchNamesUniquePerTargetWithinSameSecond pins the property that was missing
// before: two targets synced in the same second must not collide. Previously both
// carried the group ID and produced byte-identical names.
func TestBranchNamesUniquePerTargetWithinSameSecond(t *testing.T) {
	timestamp := time.Date(2026, 7, 29, 10, 53, 18, 0, time.UTC)
	targets := []string{
		"owner/repo-two",
		"owner/repo-one",
		"owner/repo-three",
		"owner/repo-four",
	}

	seen := make(map[string]string, len(targets))
	for _, repo := range targets {
		branch := FormatSyncBranchName("chore/sync-files", FormatBranchScope(repo), timestamp, "ee542e5")

		if previous, collided := seen[branch]; collided {
			t.Fatalf("branch name collision between %q and %q: %q", previous, repo, branch)
		}
		seen[branch] = repo
	}

	assert.Len(t, seen, len(targets))
	assert.Contains(t, seen, "chore/sync-files-repo-one-20260729-105318-ee542e5")
}

// TestBranchNamesOmitGroupID verifies group identifiers stay internal to the config
// and never reach a branch published on a target repository.
func TestBranchNamesOmitGroupID(t *testing.T) {
	timestamp := time.Date(2026, 7, 29, 10, 53, 18, 0, time.UTC)

	branch := FormatSyncBranchName("chore/sync-files", FormatBranchScope("owner/repo-one"), timestamp, "ee542e5")

	assert.NotContains(t, branch, "owner-forks", "group name must not leak into the branch")
	assert.Equal(t, "chore/sync-files-repo-one-20260729-105318-ee542e5", branch)
}

// TestBranchTimestampRoundTripsAcrossTimezones guards a bug that silently breaks any
// age comparison: names were formatted in local time but parsed back as UTC, so a
// branch created seconds ago could parse to an instant hours in the past.
func TestBranchTimestampRoundTripsAcrossTimezones(t *testing.T) {
	zones := []string{"UTC", "America/New_York", "Asia/Tokyo", "Australia/Sydney", "America/Los_Angeles"}
	instant := time.Date(2026, 7, 29, 14, 53, 18, 0, time.UTC)

	for _, zoneName := range zones {
		t.Run(zoneName, func(t *testing.T) {
			location, err := time.LoadLocation(zoneName)
			require.NoError(t, err)

			// Same instant, expressed in a different zone
			branch := FormatSyncBranchName("chore/sync-files", "repo-one", instant.In(location), "ee542e5")

			metadata, err := ParseSyncBranchName(branch, "chore/sync-files")
			require.NoError(t, err)
			assert.True(t, instant.Equal(metadata.Timestamp),
				"expected %s to round trip, got %s (branch %q)", instant, metadata.Timestamp, branch)
		})
	}
}

// TestBranchAgeIsAccurateWhenGeneratedLocally is the practical consequence of the
// round trip above: a freshly created branch must not look old.
func TestBranchAgeIsAccurateWhenGeneratedLocally(t *testing.T) {
	branch := FormatSyncBranchName("chore/sync-files", "repo-one", time.Now(), "ee542e5")

	metadata, err := ParseSyncBranchName(branch, "chore/sync-files")
	require.NoError(t, err)

	age := time.Since(metadata.Timestamp)
	assert.Less(t, age.Abs(), time.Minute, "a branch just created should read as seconds old, not hours")
}

// TestParseSyncBranchName_LegacyGroupScopedBranches ensures branches created before
// the scope segment changed meaning still parse. Live repos hold these today, and
// failing to parse them would make cleanup and state discovery blind to them.
func TestParseSyncBranchName_LegacyGroupScopedBranches(t *testing.T) {
	legacy := "chore/sync-files-owner-forks-20260729-105318-ee542e5"

	metadata, err := ParseSyncBranchName(legacy, "chore/sync-files")
	require.NoError(t, err)
	assert.Equal(t, "owner-forks", metadata.Scope)
	assert.Equal(t, "ee542e5", metadata.CommitSHA)
	assert.Equal(t, time.Date(2026, 7, 29, 10, 53, 18, 0, time.UTC), metadata.Timestamp)
}

// TestParseSyncBranchName_RejectsPrefixOnlyMatches is the guard that keeps orphan
// cleanup from deleting hand-made branches that merely start with the sync prefix.
func TestParseSyncBranchName_RejectsPrefixOnlyMatches(t *testing.T) {
	notSyncBranches := []string{
		"chore/sync-files",
		"chore/sync-files-manual-fix",
		"chore/sync-files-wip",
		"chore/sync-files-20260729-105318-ee542e5",          // missing scope segment
		"chore/sync-files-repo-two-2026-07-29-105318-abc12", // wrong date shape
		"chore/sync-files-repo-two-20260729-105318-nothex",  // commit is not hex
		"chore/sync-files-repo-two-20260729-105318-abc-x",   // trailing segment
	}

	for _, name := range notSyncBranches {
		t.Run(name, func(t *testing.T) {
			_, err := ParseSyncBranchName(name, "chore/sync-files")
			require.ErrorIs(t, err, ErrNotSyncBranch)
		})
	}
}

// TestParseSyncBranchName_RejectsInvalidTimestamp covers a name that matches the
// shape but encodes an impossible date.
func TestParseSyncBranchName_RejectsInvalidTimestamp(t *testing.T) {
	_, err := ParseSyncBranchName("chore/sync-files-repo-two-20261345-995918-abc123", "chore/sync-files")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotSyncBranch, "shape matched, so the failure should be the timestamp")
}
