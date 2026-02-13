package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/db"
)

func TestHasChanged(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		current  *db.RepositorySnapshot
		previous *db.RepositorySnapshot
		want     bool
	}{
		{
			name: "first snapshot (nil previous)",
			current: &db.RepositorySnapshot{
				Stars:      100,
				Forks:      10,
				OpenIssues: 5,
			},
			previous: nil,
			want:     true,
		},
		{
			name: "identical snapshots",
			current: &db.RepositorySnapshot{
				SnapshotAt:    now,
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			previous: &db.RepositorySnapshot{
				SnapshotAt:    now.Add(-24 * time.Hour),
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			want: false,
		},
		{
			name: "stars changed",
			current: &db.RepositorySnapshot{
				Stars:         101,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			previous: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			want: true,
		},
		{
			name: "forks changed",
			current: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         11,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			previous: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			want: true,
		},
		{
			name: "open issues changed",
			current: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    6,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			previous: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			want: true,
		},
		{
			name: "open PRs changed",
			current: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       3,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			previous: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			want: true,
		},
		{
			name: "branch count changed",
			current: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   9,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			previous: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			want: true,
		},
		{
			name: "latest release changed",
			current: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.4",
				LatestTag:     "v1.2.3",
			},
			previous: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			want: true,
		},
		{
			name: "latest tag changed",
			current: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.4",
			},
			previous: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			want: true,
		},
		{
			name: "multiple changes",
			current: &db.RepositorySnapshot{
				Stars:         105,
				Forks:         12,
				OpenIssues:    3,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.3.0",
				LatestTag:     "v1.3.0",
			},
			previous: &db.RepositorySnapshot{
				Stars:         100,
				Forks:         10,
				OpenIssues:    5,
				OpenPRs:       2,
				BranchCount:   8,
				LatestRelease: "v1.2.3",
				LatestTag:     "v1.2.3",
			},
			want: true,
		},
		{
			name: "pushed_at changed",
			current: &db.RepositorySnapshot{
				Stars:       100,
				Forks:       10,
				OpenIssues:  5,
				OpenPRs:     2,
				BranchCount: 8,
				PushedAt:    timePtr(now),
			},
			previous: &db.RepositorySnapshot{
				Stars:       100,
				Forks:       10,
				OpenIssues:  5,
				OpenPRs:     2,
				BranchCount: 8,
				PushedAt:    timePtr(now.Add(-24 * time.Hour)),
			},
			want: true,
		},
		{
			name: "pushed_at nil to set",
			current: &db.RepositorySnapshot{
				Stars:       100,
				Forks:       10,
				OpenIssues:  5,
				OpenPRs:     2,
				BranchCount: 8,
				PushedAt:    timePtr(now),
			},
			previous: &db.RepositorySnapshot{
				Stars:       100,
				Forks:       10,
				OpenIssues:  5,
				OpenPRs:     2,
				BranchCount: 8,
				PushedAt:    nil,
			},
			want: true,
		},
		{
			name: "pushed_at both nil is unchanged",
			current: &db.RepositorySnapshot{
				Stars:       100,
				Forks:       10,
				OpenIssues:  5,
				OpenPRs:     2,
				BranchCount: 8,
				PushedAt:    nil,
			},
			previous: &db.RepositorySnapshot{
				Stars:       100,
				Forks:       10,
				OpenIssues:  5,
				OpenPRs:     2,
				BranchCount: 8,
				PushedAt:    nil,
			},
			want: false,
		},
		{
			name: "dependabot alert count changed",
			current: &db.RepositorySnapshot{
				Stars:                100,
				Forks:                10,
				OpenIssues:           5,
				OpenPRs:              2,
				BranchCount:          8,
				DependabotAlertCount: 3,
			},
			previous: &db.RepositorySnapshot{
				Stars:                100,
				Forks:                10,
				OpenIssues:           5,
				OpenPRs:              2,
				BranchCount:          8,
				DependabotAlertCount: 1,
			},
			want: true,
		},
		{
			name: "code scanning alert count changed",
			current: &db.RepositorySnapshot{
				Stars:                  100,
				Forks:                  10,
				OpenIssues:             5,
				OpenPRs:                2,
				BranchCount:            8,
				CodeScanningAlertCount: 2,
			},
			previous: &db.RepositorySnapshot{
				Stars:                  100,
				Forks:                  10,
				OpenIssues:             5,
				OpenPRs:                2,
				BranchCount:            8,
				CodeScanningAlertCount: 0,
			},
			want: true,
		},
		{
			name: "secret scanning alert count changed",
			current: &db.RepositorySnapshot{
				Stars:                    100,
				Forks:                    10,
				OpenIssues:               5,
				OpenPRs:                  2,
				BranchCount:              8,
				SecretScanningAlertCount: 1,
			},
			previous: &db.RepositorySnapshot{
				Stars:                    100,
				Forks:                    10,
				OpenIssues:               5,
				OpenPRs:                  2,
				BranchCount:              8,
				SecretScanningAlertCount: 0,
			},
			want: true,
		},
		{
			name: "all alert counts identical is unchanged",
			current: &db.RepositorySnapshot{
				Stars:                    100,
				Forks:                    10,
				OpenIssues:               5,
				OpenPRs:                  2,
				BranchCount:              8,
				DependabotAlertCount:     2,
				CodeScanningAlertCount:   1,
				SecretScanningAlertCount: 0,
			},
			previous: &db.RepositorySnapshot{
				Stars:                    100,
				Forks:                    10,
				OpenIssues:               5,
				OpenPRs:                  2,
				BranchCount:              8,
				DependabotAlertCount:     2,
				CodeScanningAlertCount:   1,
				SecretScanningAlertCount: 0,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasChanged(tt.current, tt.previous)
			assert.Equal(t, tt.want, got, "HasChanged() result mismatch")
		})
	}
}

func TestTimeEqual(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	tests := []struct {
		name string
		a    *time.Time
		b    *time.Time
		want bool
	}{
		{"both nil", nil, nil, true},
		{"a nil b set", nil, &now, false},
		{"a set b nil", &now, nil, false},
		{"same time", &now, &now, true},
		{"different times", &now, &later, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeEqual(tt.a, tt.b)
			assert.Equal(t, tt.want, got, "timeEqual() result mismatch")
		})
	}
}

// timePtr returns a pointer to the given time value
func timePtr(t time.Time) *time.Time {
	return &t
}
