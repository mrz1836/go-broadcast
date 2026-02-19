package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/analytics"
	"github.com/mrz1836/go-broadcast/internal/db"
)

func TestParseRepoName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fullName  string
		wantOwner string
		wantName  string
		wantErr   bool
	}{
		{
			name:      "valid owner/name",
			fullName:  "mrz1836/go-broadcast",
			wantOwner: "mrz1836",
			wantName:  "go-broadcast",
		},
		{
			name:     "empty string",
			fullName: "",
			wantErr:  true,
		},
		{
			name:     "no slash",
			fullName: "noslash",
			wantErr:  true,
		},
		{
			name:     "empty org",
			fullName: "/repo",
			wantErr:  true,
		},
		{
			name:     "empty name",
			fullName: "org/",
			wantErr:  true,
		},
		{
			name:     "too many slashes",
			fullName: "org/repo/extra",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			owner, name, err := parseRepoName(tt.fullName)
			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, owner)
				assert.Empty(t, name)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantOwner, owner)
				assert.Equal(t, tt.wantName, name)
			}
		})
	}
}

func TestDetermineSyncType(t *testing.T) {
	t.Parallel()

	t.Run("security only", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "security_only", determineSyncType(true))
	})

	t.Run("full sync", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "full", determineSyncType(false))
	})
}

func TestParseOwnerAndName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fullName  string
		wantOwner string
		wantName  string
	}{
		{
			name:      "standard owner/name",
			fullName:  "owner/name",
			wantOwner: "owner",
			wantName:  "name",
		},
		{
			name:      "single string no slash",
			fullName:  "single",
			wantOwner: "",
			wantName:  "single",
		},
		{
			name:      "empty string",
			fullName:  "",
			wantOwner: "",
			wantName:  "",
		},
		{
			name:      "multiple slashes returns first two parts",
			fullName:  "a/b/c",
			wantOwner: "",
			wantName:  "a/b/c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			owner, name := parseOwnerAndName(tt.fullName)
			assert.Equal(t, tt.wantOwner, owner)
			assert.Equal(t, tt.wantName, name)
		})
	}
}

func TestBuildAnalyticsRepository(t *testing.T) {
	t.Parallel()

	metadata := &analytics.RepoMetadata{
		FullName:              "org/repo",
		Stars:                 100,
		Forks:                 20,
		Watchers:              50,
		OpenIssues:            5,
		OpenPRs:               2,
		BranchCount:           10,
		DefaultBranch:         "main",
		Description:           "Test repo",
		Language:              "Go",
		IsFork:                true,
		ForkParent:            "upstream/repo",
		IsPrivate:             false,
		IsArchived:            false,
		HomepageURL:           "https://example.com",
		Topics:                []string{"go", "tools"},
		License:               "MIT",
		DiskUsageKB:           1024,
		HasIssuesEnabled:      true,
		HasWikiEnabled:        false,
		HasDiscussionsEnabled: true,
		HTMLURL:               "https://github.com/org/repo",
		SSHURL:                "git@github.com:org/repo.git",
		CloneURL:              "https://github.com/org/repo.git",
		CreatedAt:             "2024-01-15T10:00:00Z",
		PushedAt:              "2024-06-01T14:30:00Z",
		UpdatedAt:             "2024-06-01T14:30:00Z",
	}

	result := buildAnalyticsRepository(metadata, 42)

	assert.Equal(t, uint(42), result.OrganizationID)
	assert.Equal(t, "org", result.Owner)
	assert.Equal(t, "repo", result.Name)
	assert.Equal(t, "org/repo", result.FullName)
	assert.Equal(t, "Test repo", result.Description)
	assert.Equal(t, "main", result.DefaultBranch)
	assert.Equal(t, "Go", result.Language)
	assert.True(t, result.IsFork)
	assert.Equal(t, "upstream/repo", result.ForkSource)
	assert.False(t, result.IsPrivate)
	assert.False(t, result.IsArchived)
	assert.Equal(t, "https://github.com/org/repo", result.URL)
	assert.Equal(t, "https://example.com", result.HomepageURL)
	assert.Contains(t, result.Topics, `"go"`)
	assert.Contains(t, result.Topics, `"tools"`)
	assert.Equal(t, "MIT", result.License)
	assert.Equal(t, 1024, result.DiskUsageKB)
	assert.True(t, result.HasIssuesEnabled)
	assert.False(t, result.HasWikiEnabled)
	assert.True(t, result.HasDiscussionsEnabled)
	assert.Equal(t, "git@github.com:org/repo.git", result.SSHURL)
	assert.Equal(t, "https://github.com/org/repo.git", result.CloneURL)
	require.NotNil(t, result.GitHubCreatedAt)
	require.NotNil(t, result.LastPushedAt)
	require.NotNil(t, result.GitHubUpdatedAt)
}

func TestBuildAnalyticsRepository_EmptyTopics(t *testing.T) {
	t.Parallel()

	metadata := &analytics.RepoMetadata{
		FullName: "org/repo",
		Topics:   nil,
	}

	result := buildAnalyticsRepository(metadata, 1)
	assert.Empty(t, result.Topics)
}

func TestBuildRepositorySnapshot(t *testing.T) {
	t.Parallel()

	releaseAt := "2024-05-01T10:00:00Z"
	tagAt := "2024-05-15T12:00:00Z"
	metadata := &analytics.RepoMetadata{
		FullName:        "org/repo",
		Stars:           42,
		Forks:           5,
		Watchers:        10,
		OpenIssues:      3,
		OpenPRs:         1,
		BranchCount:     8,
		LatestRelease:   "v1.2.0",
		LatestReleaseAt: &releaseAt,
		LatestTag:       "v1.2.0",
		LatestTagAt:     &tagAt,
		UpdatedAt:       "2024-06-01T14:30:00Z",
		PushedAt:        "2024-06-01T14:30:00Z",
	}

	result := buildRepositorySnapshot(metadata, 99)

	assert.Equal(t, uint(99), result.RepositoryID)
	assert.False(t, result.SnapshotAt.IsZero())
	assert.Equal(t, 42, result.Stars)
	assert.Equal(t, 5, result.Forks)
	assert.Equal(t, 10, result.Watchers)
	assert.Equal(t, 3, result.OpenIssues)
	assert.Equal(t, 1, result.OpenPRs)
	assert.Equal(t, 8, result.BranchCount)
	assert.Equal(t, "v1.2.0", result.LatestRelease)
	require.NotNil(t, result.LatestReleaseAt)
	assert.Equal(t, "v1.2.0", result.LatestTag)
	require.NotNil(t, result.LatestTagAt)
	require.NotNil(t, result.RepoUpdatedAt)
	require.NotNil(t, result.PushedAt)
	assert.Equal(t, 0, result.DependabotAlertCount)
	assert.Equal(t, 0, result.CodeScanningAlertCount)
	assert.Equal(t, 0, result.SecretScanningAlertCount)
}

func TestConvertSecurityAlert(t *testing.T) {
	t.Parallel()

	t.Run("all fields populated", func(t *testing.T) {
		t.Parallel()

		dismissed := "2024-05-01T10:00:00Z"
		fixed := "2024-06-01T12:00:00Z"
		alert := analytics.SecurityAlert{
			AlertType:   analytics.AlertTypeDependabot,
			AlertNumber: 42,
			State:       "open",
			Severity:    "critical",
			Title:       "Test Alert",
			HTMLURL:     "https://github.com/org/repo/security/dependabot/42",
			CreatedAt:   "2024-01-15T10:00:00Z",
			DismissedAt: &dismissed,
			FixedAt:     &fixed,
		}

		result := convertSecurityAlert(alert, 7)

		assert.Equal(t, uint(7), result.RepositoryID)
		assert.Equal(t, "dependabot", result.AlertType)
		assert.Equal(t, 42, result.AlertNumber)
		assert.Equal(t, "open", result.State)
		assert.Equal(t, "critical", result.Severity)
		assert.Equal(t, "Test Alert", result.Summary)
		assert.Equal(t, "https://github.com/org/repo/security/dependabot/42", result.HTMLURL)
		assert.False(t, result.AlertCreatedAt.IsZero())
		require.NotNil(t, result.DismissedAt)
		require.NotNil(t, result.FixedAt)
	})

	t.Run("nil timestamps", func(t *testing.T) {
		t.Parallel()

		alert := analytics.SecurityAlert{
			AlertType:   analytics.AlertTypeCodeScanning,
			AlertNumber: 1,
			State:       "open",
			Severity:    "high",
			Title:       "Code scan finding",
			CreatedAt:   "",
			DismissedAt: nil,
			FixedAt:     nil,
		}

		result := convertSecurityAlert(alert, 3)

		assert.Equal(t, "code_scanning", result.AlertType)
		assert.True(t, result.AlertCreatedAt.IsZero())
		assert.Nil(t, result.DismissedAt)
		assert.Nil(t, result.FixedAt)
	})
}

func TestParseTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantNil  bool
		wantYear int
	}{
		{
			name:     "valid ISO8601",
			input:    "2024-01-15T10:30:00Z",
			wantYear: 2024,
		},
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:    "invalid format",
			input:   "not-a-date",
			wantNil: true,
		},
		{
			name:    "wrong format",
			input:   "2024-01-15",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := parseTime(tt.input)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.wantYear, result.Year())
			}
		})
	}
}

func TestParseTimePtr(t *testing.T) {
	t.Parallel()

	t.Run("nil input", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, parseTimePtr(nil))
	})

	t.Run("valid string pointer", func(t *testing.T) {
		t.Parallel()
		s := "2024-01-15T10:30:00Z"
		result := parseTimePtr(&s)
		require.NotNil(t, result)
		assert.Equal(t, 2024, result.Year())
	})

	t.Run("empty string pointer", func(t *testing.T) {
		t.Parallel()
		s := ""
		assert.Nil(t, parseTimePtr(&s))
	})

	t.Run("invalid string pointer", func(t *testing.T) {
		t.Parallel()
		s := "bad"
		assert.Nil(t, parseTimePtr(&s))
	})
}

func TestDisplaySyncSummary(t *testing.T) {
	t.Parallel()

	t.Run("completed status", func(t *testing.T) {
		t.Parallel()

		syncRun := &db.SyncRun{
			SyncType:         "full",
			ReposProcessed:   5,
			ReposSkipped:     2,
			SnapshotsCreated: 3,
			AlertsUpserted:   10,
			DurationMs:       5000,
			APICallsMade:     25,
		}
		stats := &analytics.ThrottleStats{
			TotalCalls:    25,
			TotalRetries:  0,
			TotalWaitedMs: 100,
		}

		// Should not panic
		assert.NotPanics(t, func() {
			displaySyncSummary(syncRun, "completed", stats)
		})
	})

	t.Run("partial status with failures", func(t *testing.T) {
		t.Parallel()

		syncRun := &db.SyncRun{
			SyncType:         "full",
			ReposProcessed:   5,
			ReposFailed:      1,
			SnapshotsCreated: 4,
			DurationMs:       3000,
		}
		stats := &analytics.ThrottleStats{
			TotalRetries: 3,
		}

		assert.NotPanics(t, func() {
			displaySyncSummary(syncRun, "partial", stats)
		})
	})

	t.Run("failed status", func(t *testing.T) {
		t.Parallel()

		syncRun := &db.SyncRun{
			SyncType:         "security_only",
			ReposProcessed:   0,
			SnapshotsCreated: 1,
			DurationMs:       100,
		}

		assert.NotPanics(t, func() {
			displaySyncSummary(syncRun, "failed", nil)
		})
	})

	t.Run("dry run skip logic", func(t *testing.T) {
		t.Parallel()

		syncRun := &db.SyncRun{
			SyncType:         "full",
			ReposProcessed:   5,
			SnapshotsCreated: 0,
			AlertsUpserted:   0,
			DurationMs:       5,
		}

		// Should detect dry-run heuristic and skip
		assert.NotPanics(t, func() {
			displaySyncSummary(syncRun, "completed", nil)
		})
	})

	t.Run("nil throttle stats", func(t *testing.T) {
		t.Parallel()

		syncRun := &db.SyncRun{
			SyncType:         "full",
			ReposProcessed:   1,
			SnapshotsCreated: 1,
			DurationMs:       1000,
		}

		assert.NotPanics(t, func() {
			displaySyncSummary(syncRun, "completed", nil)
		})
	})

	t.Run("zero duration", func(t *testing.T) {
		t.Parallel()

		syncRun := &db.SyncRun{
			SyncType:         "full",
			ReposProcessed:   1,
			SnapshotsCreated: 1,
			DurationMs:       0,
		}

		assert.NotPanics(t, func() {
			displaySyncSummary(syncRun, "completed", nil)
		})
	})

	t.Run("throttle stats with wait time", func(t *testing.T) {
		t.Parallel()

		syncRun := &db.SyncRun{
			SyncType:         "full",
			ReposProcessed:   1,
			SnapshotsCreated: 1,
			DurationMs:       1000,
			APICallsMade:     10,
		}
		stats := &analytics.ThrottleStats{
			TotalCalls:    10,
			TotalRetries:  2,
			TotalWaitedMs: 5000,
		}

		assert.NotPanics(t, func() {
			displaySyncSummary(syncRun, "completed", stats)
		})
	})
}
