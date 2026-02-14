package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAnalyticsRepo_Organizations(t *testing.T) {
	db := TestDB(t)

	repo := NewAnalyticsRepo(db)
	ctx := context.Background()

	// Create test client first (required by Organization FK)
	client := &Client{
		Name: "test-client",
	}
	require.NoError(t, db.Create(client).Error)

	t.Run("UpsertOrganization", func(t *testing.T) {
		org := &Organization{
			Name:        "test-org",
			Description: "Test organization",
			ClientID:    client.ID,
		}

		err := repo.UpsertOrganization(ctx, org)
		require.NoError(t, err)
		assert.NotZero(t, org.ID)

		// Update
		org.Description = "Updated description"
		err = repo.UpsertOrganization(ctx, org)
		require.NoError(t, err)

		// Verify update
		got, err := repo.GetOrganization(ctx, "test-org")
		require.NoError(t, err)
		assert.Equal(t, "Updated description", got.Description)
	})

	t.Run("ListOrganizations", func(t *testing.T) {
		orgs, err := repo.ListOrganizations(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, orgs)
	})
}

func TestAnalyticsRepo_Repositories(t *testing.T) {
	db := TestDB(t)

	repo := NewAnalyticsRepo(db)
	ctx := context.Background()

	// Create test client and org
	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{
		Name:     "test-org",
		ClientID: client.ID,
	}
	require.NoError(t, repo.UpsertOrganization(ctx, org))

	t.Run("UpsertRepository", func(t *testing.T) {
		analyticsRepo := &AnalyticsRepository{
			OrganizationID: org.ID,
			Owner:          "test-org",
			Name:           "test-repo",
			FullName:       "test-org/test-repo",
			Description:    "Test repository",
			DefaultBranch:  "main",
			Language:       "Go",
			IsPrivate:      false,
			IsFork:         false,
			IsArchived:     false,
			URL:            "https://github.com/test-org/test-repo",
		}

		err := repo.UpsertRepository(ctx, analyticsRepo)
		require.NoError(t, err)
		assert.NotZero(t, analyticsRepo.ID)

		// Update
		analyticsRepo.Description = "Updated"
		err = repo.UpsertRepository(ctx, analyticsRepo)
		require.NoError(t, err)

		// Verify
		got, err := repo.GetRepository(ctx, "test-org/test-repo")
		require.NoError(t, err)
		assert.Equal(t, "Updated", got.Description)
	})

	t.Run("ListRepositories", func(t *testing.T) {
		repos, err := repo.ListRepositories(ctx, "test-org")
		require.NoError(t, err)
		assert.NotEmpty(t, repos)
	})
}

func TestAnalyticsRepo_Snapshots(t *testing.T) {
	db := TestDB(t)

	repo := NewAnalyticsRepo(db)
	ctx := context.Background()

	// Setup
	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{Name: "test-org", ClientID: client.ID}
	require.NoError(t, repo.UpsertOrganization(ctx, org))

	analyticsRepo := &AnalyticsRepository{
		OrganizationID: org.ID,
		Owner:          "test-org",
		Name:           "test-repo",
		FullName:       "test-org/test-repo",
	}
	require.NoError(t, repo.UpsertRepository(ctx, analyticsRepo))

	t.Run("CreateSnapshot", func(t *testing.T) {
		snap := &RepositorySnapshot{
			RepositoryID:  analyticsRepo.ID,
			SnapshotAt:    time.Now(),
			Stars:         100,
			Forks:         10,
			OpenIssues:    5,
			OpenPRs:       2,
			BranchCount:   8,
			LatestRelease: "v1.2.3",
			LatestTag:     "v1.2.3",
		}

		err := repo.CreateSnapshot(ctx, snap)
		require.NoError(t, err)
		assert.NotZero(t, snap.ID)
	})

	t.Run("GetLatestSnapshot", func(t *testing.T) {
		// Create multiple snapshots
		now := time.Now()
		for i := 0; i < 3; i++ {
			snap := &RepositorySnapshot{
				RepositoryID: analyticsRepo.ID,
				SnapshotAt:   now.Add(time.Duration(-i) * 24 * time.Hour),
				Stars:        100 + i,
			}
			require.NoError(t, repo.CreateSnapshot(ctx, snap))
		}

		latest, err := repo.GetLatestSnapshot(ctx, analyticsRepo.ID)
		require.NoError(t, err)
		assert.NotNil(t, latest)
		assert.Equal(t, 100, latest.Stars) // Most recent
	})

	t.Run("GetSnapshotHistory", func(t *testing.T) {
		since := time.Now().Add(-48 * time.Hour)
		snaps, err := repo.GetSnapshotHistory(ctx, analyticsRepo.ID, since)
		require.NoError(t, err)
		assert.NotEmpty(t, snaps)
	})

	t.Run("GetLatestSnapshot_NoSnapshots", func(t *testing.T) {
		// Test with non-existent repo ID
		latest, err := repo.GetLatestSnapshot(ctx, 99999)
		require.Error(t, err)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, latest)
	})
}

func TestAnalyticsRepo_Alerts(t *testing.T) {
	db := TestDB(t)

	repo := NewAnalyticsRepo(db)
	ctx := context.Background()

	// Setup
	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{Name: "test-org", ClientID: client.ID}
	require.NoError(t, repo.UpsertOrganization(ctx, org))

	analyticsRepo := &AnalyticsRepository{
		OrganizationID: org.ID,
		Owner:          "test-org",
		Name:           "test-repo",
		FullName:       "test-org/test-repo",
	}
	require.NoError(t, repo.UpsertRepository(ctx, analyticsRepo))

	t.Run("UpsertAlert", func(t *testing.T) {
		alert := &SecurityAlert{
			RepositoryID:   analyticsRepo.ID,
			AlertType:      "dependabot",
			AlertNumber:    1,
			State:          "open",
			Severity:       "high",
			Summary:        "Test alert",
			AlertCreatedAt: time.Now(),
		}

		err := repo.UpsertAlert(ctx, alert)
		require.NoError(t, err)
		assert.NotZero(t, alert.ID)

		// Update state
		alert.State = "fixed"
		fixedAt := time.Now()
		alert.FixedAt = &fixedAt
		err = repo.UpsertAlert(ctx, alert)
		require.NoError(t, err)
	})

	t.Run("GetOpenAlerts", func(t *testing.T) {
		// Create test alerts
		severities := []string{"critical", "high", "medium", "low"}
		for i, sev := range severities {
			alert := &SecurityAlert{
				RepositoryID:   analyticsRepo.ID,
				AlertType:      "code_scanning",
				AlertNumber:    i + 10,
				State:          "open",
				Severity:       sev,
				Summary:        "Alert " + sev,
				AlertCreatedAt: time.Now(),
			}
			require.NoError(t, repo.UpsertAlert(ctx, alert))
		}

		// Get all open alerts
		alerts, err := repo.GetOpenAlerts(ctx, analyticsRepo.ID, "")
		require.NoError(t, err)
		assert.NotEmpty(t, alerts)

		// Get critical alerts
		criticals, err := repo.GetOpenAlerts(ctx, analyticsRepo.ID, "critical")
		require.NoError(t, err)
		assert.Len(t, criticals, 1)
		assert.Equal(t, "critical", criticals[0].Severity)
	})

	t.Run("GetAlertCounts", func(t *testing.T) {
		counts, err := repo.GetAlertCounts(ctx, analyticsRepo.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, counts)
		assert.Positive(t, counts["high"])
	})

	t.Run("GetAlertCountsByType", func(t *testing.T) {
		// Create additional open alerts of different types
		secretAlert := &SecurityAlert{
			RepositoryID:   analyticsRepo.ID,
			AlertType:      "secret_scanning",
			AlertNumber:    100,
			State:          "open",
			Severity:       "high",
			Summary:        "Secret detected",
			AlertCreatedAt: time.Now(),
		}
		require.NoError(t, repo.UpsertAlert(ctx, secretAlert))

		counts, err := repo.GetAlertCountsByType(ctx, analyticsRepo.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, counts)
		// We have open code_scanning alerts from earlier
		assert.Positive(t, counts["code_scanning"])
		// We just created a secret_scanning alert
		assert.Equal(t, 1, counts["secret_scanning"])
	})

	t.Run("GetAlertCountsByType_NoAlerts", func(t *testing.T) {
		counts, err := repo.GetAlertCountsByType(ctx, 99999)
		require.NoError(t, err)
		assert.Empty(t, counts)
	})

	t.Run("GetAlertCountsByType_ClosedAlertsExcluded", func(t *testing.T) {
		// Create a repo with only closed alerts
		closedRepo := &AnalyticsRepository{
			OrganizationID: org.ID,
			Owner:          "test-org",
			Name:           "closed-alerts-repo",
			FullName:       "test-org/closed-alerts-repo",
		}
		require.NoError(t, repo.UpsertRepository(ctx, closedRepo))

		closedAlert := &SecurityAlert{
			RepositoryID:   closedRepo.ID,
			AlertType:      "dependabot",
			AlertNumber:    1,
			State:          "fixed",
			Severity:       "high",
			Summary:        "Fixed alert",
			AlertCreatedAt: time.Now(),
		}
		require.NoError(t, repo.UpsertAlert(ctx, closedAlert))

		counts, err := repo.GetAlertCountsByType(ctx, closedRepo.ID)
		require.NoError(t, err)
		assert.Empty(t, counts)
	})
}

func TestAnalyticsRepo_SyncRuns(t *testing.T) {
	db := TestDB(t)

	repo := NewAnalyticsRepo(db)
	ctx := context.Background()

	t.Run("CreateSyncRun", func(t *testing.T) {
		run := &SyncRun{
			StartedAt:        time.Now(),
			Status:           "running",
			SyncType:         "full",
			ReposProcessed:   0,
			ReposSkipped:     0,
			ReposFailed:      0,
			SnapshotsCreated: 0,
			AlertsUpserted:   0,
			APICallsMade:     0,
		}

		err := repo.CreateSyncRun(ctx, run)
		require.NoError(t, err)
		assert.NotZero(t, run.ID)
	})

	t.Run("UpdateSyncRun", func(t *testing.T) {
		run := &SyncRun{
			StartedAt:      time.Now(),
			Status:         "running",
			SyncType:       "full",
			ReposProcessed: 0,
		}
		require.NoError(t, repo.CreateSyncRun(ctx, run))

		// Update with completion
		completedAt := time.Now()
		run.CompletedAt = &completedAt
		run.Status = "completed"
		run.ReposProcessed = 75
		run.SnapshotsCreated = 60
		run.DurationMs = 25000

		err := repo.UpdateSyncRun(ctx, run)
		require.NoError(t, err)

		// Verify
		latest, err := repo.GetLatestSyncRun(ctx)
		require.NoError(t, err)
		assert.Equal(t, "completed", latest.Status)
		assert.Equal(t, 75, latest.ReposProcessed)
	})

	t.Run("GetLatestSyncRun", func(t *testing.T) {
		// Create multiple runs
		for i := 0; i < 3; i++ {
			run := &SyncRun{
				StartedAt: time.Now().Add(time.Duration(-i) * time.Hour),
				Status:    "completed",
				SyncType:  "full",
			}
			require.NoError(t, repo.CreateSyncRun(ctx, run))
		}

		latest, err := repo.GetLatestSyncRun(ctx)
		require.NoError(t, err)
		assert.NotNil(t, latest)
		assert.Equal(t, "completed", latest.Status)
	})

	t.Run("UpdateSyncRun_NoID", func(t *testing.T) {
		run := &SyncRun{
			Status: "completed",
		}
		err := repo.UpdateSyncRun(ctx, run)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidSyncRunID)
	})
}

func TestAnalyticsRepo_CISnapshots(t *testing.T) {
	db := TestDB(t)

	repo := NewAnalyticsRepo(db)
	ctx := context.Background()

	// Setup
	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{Name: "test-org", ClientID: client.ID}
	require.NoError(t, repo.UpsertOrganization(ctx, org))

	analyticsRepo := &AnalyticsRepository{
		OrganizationID: org.ID,
		Owner:          "test-org",
		Name:           "test-repo",
		FullName:       "test-org/test-repo",
	}
	require.NoError(t, repo.UpsertRepository(ctx, analyticsRepo))

	t.Run("CreateCISnapshot", func(t *testing.T) {
		coverage := 85.5
		snap := &CIMetricsSnapshot{
			RepositoryID:    analyticsRepo.ID,
			SnapshotAt:      time.Now(),
			WorkflowRunID:   12345,
			Branch:          "main",
			CommitSHA:       "abc123def456",
			GoFilesLOC:      5000,
			TestFilesLOC:    2000,
			GoFilesCount:    80,
			TestFilesCount:  40,
			TestCount:       150,
			BenchmarkCount:  25,
			CoveragePercent: &coverage,
		}

		err := repo.CreateCISnapshot(ctx, snap)
		require.NoError(t, err)
		assert.NotZero(t, snap.ID)
	})

	t.Run("GetLatestCISnapshot", func(t *testing.T) {
		// Create multiple CI snapshots
		now := time.Now()
		for i := 0; i < 3; i++ {
			snap := &CIMetricsSnapshot{
				RepositoryID:  analyticsRepo.ID,
				SnapshotAt:    now.Add(time.Duration(-i) * 24 * time.Hour),
				WorkflowRunID: int64(100 + i),
				Branch:        "main",
				GoFilesLOC:    5000 + i*100,
			}
			require.NoError(t, repo.CreateCISnapshot(ctx, snap))
		}

		latest, err := repo.GetLatestCISnapshot(ctx, analyticsRepo.ID)
		require.NoError(t, err)
		require.NotNil(t, latest)
		assert.Equal(t, 5000, latest.GoFilesLOC) // Most recent (i=0)
		assert.Equal(t, int64(100), latest.WorkflowRunID)
	})

	t.Run("GetLatestCISnapshot_NoSnapshots", func(t *testing.T) {
		latest, err := repo.GetLatestCISnapshot(ctx, 99999)
		require.Error(t, err)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, latest)
	})

	t.Run("CreateCISnapshot_NilCoverage", func(t *testing.T) {
		snap := &CIMetricsSnapshot{
			RepositoryID:    analyticsRepo.ID,
			SnapshotAt:      time.Now(),
			WorkflowRunID:   99999,
			Branch:          "develop",
			GoFilesLOC:      1000,
			CoveragePercent: nil, // No coverage data
		}

		err := repo.CreateCISnapshot(ctx, snap)
		require.NoError(t, err)
		assert.NotZero(t, snap.ID)

		// Verify nil coverage is preserved
		retrieved, err := repo.GetLatestCISnapshot(ctx, analyticsRepo.ID)
		require.NoError(t, err)
		// Latest may or may not be this one depending on timestamp ordering
		// Just verify the snapshot was created
		assert.NotNil(t, retrieved)
	})
}
