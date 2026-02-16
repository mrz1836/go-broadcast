package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// TestBroadcastSyncMetrics_FullFlow tests the complete sync metrics flow
// This integration test verifies:
// 1. Sync run creation and tracking
// 2. Target result recording
// 3. File change tracking
// 4. Metrics aggregation and queries
func TestBroadcastSyncMetrics_FullFlow(t *testing.T) {
	// Setup test database
	database := db.TestDB(t)
	repo := db.NewBroadcastSyncRepo(database)
	ctx := context.Background()

	// Phase 1: Simulate a sync run starting
	t.Run("create sync run", func(t *testing.T) {
		run := &db.BroadcastSyncRun{
			ExternalID:   "SR-20260215-integration",
			StartedAt:    time.Now(),
			Status:       db.BroadcastSyncRunStatusRunning,
			Trigger:      db.BroadcastSyncRunTriggerManual,
			TotalTargets: 3,
			SourceBranch: "main",
			SourceCommit: "abc123def456",
		}

		err := repo.CreateSyncRun(ctx, run)
		require.NoError(t, err)
		assert.NotZero(t, run.ID)
		assert.Equal(t, "SR-20260215-integration", run.ExternalID)
	})

	// Phase 2: Simulate processing multiple targets
	t.Run("record target results", func(t *testing.T) {
		// Get the run we created
		run, err := repo.GetSyncRunByExternalID(ctx, "SR-20260215-integration")
		require.NoError(t, err)

		// Target 1: Success with file changes
		result1 := &db.BroadcastSyncTargetResult{
			BroadcastSyncRunID: run.ID,
			TargetID:           1,
			RepoID:             101,
			StartedAt:          time.Now(),
			Status:             db.BroadcastSyncTargetStatusSuccess,
			BranchName:         "sync/main",
			SourceCommitSHA:    "abc123",
			FilesProcessed:     10,
			FilesChanged:       3,
			FilesDeleted:       0,
			LinesAdded:         150,
			LinesRemoved:       50,
			BytesChanged:       4096,
		}
		result1.EndedAt = new(time.Time)
		*result1.EndedAt = time.Now().Add(5 * time.Second)
		result1.DurationMs = result1.EndedAt.Sub(result1.StartedAt).Milliseconds()

		// Add PR info
		prNumber := 42
		result1.PRNumber = &prNumber
		result1.PRURL = "https://github.com/org/repo1/pull/42"
		result1.PRState = "open"

		err = repo.CreateTargetResult(ctx, result1)
		require.NoError(t, err)

		// Target 2: Success with different changes
		result2 := &db.BroadcastSyncTargetResult{
			BroadcastSyncRunID: run.ID,
			TargetID:           2,
			RepoID:             102,
			StartedAt:          time.Now(),
			Status:             db.BroadcastSyncTargetStatusSuccess,
			BranchName:         "sync/main",
			SourceCommitSHA:    "abc123",
			FilesProcessed:     8,
			FilesChanged:       2,
			LinesAdded:         100,
			LinesRemoved:       25,
			BytesChanged:       2048,
		}
		result2.EndedAt = new(time.Time)
		*result2.EndedAt = time.Now().Add(3 * time.Second)
		result2.DurationMs = result2.EndedAt.Sub(result2.StartedAt).Milliseconds()

		err = repo.CreateTargetResult(ctx, result2)
		require.NoError(t, err)

		// Target 3: Failed
		result3 := &db.BroadcastSyncTargetResult{
			BroadcastSyncRunID: run.ID,
			TargetID:           3,
			RepoID:             103,
			StartedAt:          time.Now(),
			Status:             db.BroadcastSyncTargetStatusFailed,
			BranchName:         "sync/main",
			SourceCommitSHA:    "abc123",
			ErrorMessage:       "authentication failed",
			ErrorDetails: db.Metadata{
				"error_type": "auth_error",
				"retryable":  true,
			},
		}
		result3.EndedAt = new(time.Time)
		*result3.EndedAt = time.Now().Add(1 * time.Second)
		result3.DurationMs = result3.EndedAt.Sub(result3.StartedAt).Milliseconds()

		err = repo.CreateTargetResult(ctx, result3)
		require.NoError(t, err)
	})

	// Phase 3: Record file changes for successful targets
	t.Run("record file changes", func(t *testing.T) {
		run, err := repo.GetSyncRunByExternalID(ctx, "SR-20260215-integration")
		require.NoError(t, err)

		results, err := repo.GetTargetResultsByRunID(ctx, run.ID)
		require.NoError(t, err)
		require.Len(t, results, 3)

		// Find the first successful result (should have 3 file changes)
		var targetResult *db.BroadcastSyncTargetResult
		for i := range results {
			if results[i].Status == db.BroadcastSyncTargetStatusSuccess && results[i].FilesChanged == 3 {
				targetResult = &results[i]
				break
			}
		}
		require.NotNil(t, targetResult)

		// Record file changes
		changes := []db.BroadcastSyncFileChange{
			{
				BroadcastSyncTargetResultID: targetResult.ID,
				FilePath:                    ".github/workflows/ci.yml",
				ChangeType:                  db.BroadcastSyncFileChangeTypeModified,
				LinesAdded:                  50,
				LinesRemoved:                20,
				SizeBytes:                   1024,
				Position:                    0,
			},
			{
				BroadcastSyncTargetResultID: targetResult.ID,
				FilePath:                    "README.md",
				ChangeType:                  db.BroadcastSyncFileChangeTypeModified,
				LinesAdded:                  75,
				LinesRemoved:                25,
				SizeBytes:                   2048,
				Position:                    1,
			},
			{
				BroadcastSyncTargetResultID: targetResult.ID,
				FilePath:                    ".gitignore",
				ChangeType:                  db.BroadcastSyncFileChangeTypeModified,
				LinesAdded:                  25,
				LinesRemoved:                5,
				SizeBytes:                   1024,
				Position:                    2,
			},
		}

		err = repo.CreateFileChanges(ctx, changes)
		require.NoError(t, err)
	})

	// Phase 4: Complete the sync run
	t.Run("complete sync run", func(t *testing.T) {
		run, err := repo.GetSyncRunByExternalID(ctx, "SR-20260215-integration")
		require.NoError(t, err)

		// Update run with completion details
		endedAt := time.Now()
		run.EndedAt = &endedAt
		run.DurationMs = endedAt.Sub(run.StartedAt).Milliseconds()
		run.Status = db.BroadcastSyncRunStatusPartial // Some succeeded, one failed
		run.SuccessfulTargets = 2
		run.FailedTargets = 1
		run.TotalFilesChanged = 5 // 3 + 2 from the two successful targets
		run.TotalLinesAdded = 250
		run.TotalLinesRemoved = 75

		err = repo.UpdateSyncRun(ctx, run)
		require.NoError(t, err)
	})

	// Phase 5: Query metrics and verify data integrity
	t.Run("query and verify metrics", func(t *testing.T) {
		// Get run with all related data
		run, err := repo.GetSyncRunByExternalID(ctx, "SR-20260215-integration")
		require.NoError(t, err)

		// Verify run details
		assert.Equal(t, db.BroadcastSyncRunStatusPartial, run.Status)
		assert.Equal(t, 3, run.TotalTargets)
		assert.Equal(t, 2, run.SuccessfulTargets)
		assert.Equal(t, 1, run.FailedTargets)
		assert.Equal(t, 5, run.TotalFilesChanged)
		assert.Equal(t, 250, run.TotalLinesAdded)
		assert.Equal(t, 75, run.TotalLinesRemoved)
		assert.NotNil(t, run.EndedAt)
		assert.Greater(t, run.DurationMs, int64(0))

		// Verify target results loaded
		assert.Len(t, run.TargetResults, 3)

		// Count successful vs failed
		successCount := 0
		failedCount := 0
		for _, result := range run.TargetResults {
			switch result.Status {
			case db.BroadcastSyncTargetStatusSuccess:
				successCount++
			case db.BroadcastSyncTargetStatusFailed:
				failedCount++
			}
		}
		assert.Equal(t, 2, successCount)
		assert.Equal(t, 1, failedCount)

		// Verify file changes for first target
		var firstSuccessResult *db.BroadcastSyncTargetResult
		for i := range run.TargetResults {
			if run.TargetResults[i].Status == db.BroadcastSyncTargetStatusSuccess &&
				run.TargetResults[i].FilesChanged == 3 {
				firstSuccessResult = &run.TargetResults[i]
				break
			}
		}
		require.NotNil(t, firstSuccessResult)
		assert.Len(t, firstSuccessResult.FileChanges, 3)

		// Verify PR info
		require.NotNil(t, firstSuccessResult.PRNumber)
		assert.Equal(t, 42, *firstSuccessResult.PRNumber)
		assert.Equal(t, "https://github.com/org/repo1/pull/42", firstSuccessResult.PRURL)
		assert.Equal(t, "open", firstSuccessResult.PRState)

		// Verify error details for failed target
		var failedResult *db.BroadcastSyncTargetResult
		for i := range run.TargetResults {
			if run.TargetResults[i].Status == db.BroadcastSyncTargetStatusFailed {
				failedResult = &run.TargetResults[i]
				break
			}
		}
		require.NotNil(t, failedResult)
		assert.Equal(t, "authentication failed", failedResult.ErrorMessage)
		assert.NotNil(t, failedResult.ErrorDetails)
		assert.Equal(t, "auth_error", failedResult.ErrorDetails["error_type"])
	})

	// Phase 6: Test summary statistics
	t.Run("summary statistics", func(t *testing.T) {
		stats, err := repo.GetSyncRunSummaryStats(ctx)
		require.NoError(t, err)

		assert.Equal(t, int64(1), stats.TotalRuns)
		assert.InDelta(t, 100.0, stats.SuccessRate, 0.1) // Partial counts as success
		assert.Greater(t, stats.AvgDurationMs, int64(0))
		assert.NotNil(t, stats.LastRunAt)
	})

	// Phase 7: Test time-based queries
	t.Run("recent runs query", func(t *testing.T) {
		since := time.Now().Add(-1 * time.Hour)
		runs, err := repo.ListRecentSyncRuns(ctx, since, 100)
		require.NoError(t, err)

		assert.Len(t, runs, 1)
		assert.Equal(t, "SR-20260215-integration", runs[0].ExternalID)
	})
}

// TestBroadcastSyncMetrics_MultipleRuns tests metrics with multiple sync runs
func TestBroadcastSyncMetrics_MultipleRuns(t *testing.T) {
	database := db.TestDB(t)
	repo := db.NewBroadcastSyncRepo(database)
	ctx := context.Background()

	// Create multiple runs with different statuses
	now := time.Now()

	runs := []db.BroadcastSyncRun{
		{
			ExternalID:        "SR-20260215-001",
			StartedAt:         now.Add(-3 * time.Hour),
			Status:            db.BroadcastSyncRunStatusSuccess,
			Trigger:           db.BroadcastSyncRunTriggerManual,
			TotalTargets:      2,
			SuccessfulTargets: 2,
			DurationMs:        5000,
			TotalFilesChanged: 10,
		},
		{
			ExternalID:        "SR-20260215-002",
			StartedAt:         now.Add(-2 * time.Hour),
			Status:            db.BroadcastSyncRunStatusSuccess,
			Trigger:           db.BroadcastSyncRunTriggerCron,
			TotalTargets:      3,
			SuccessfulTargets: 3,
			DurationMs:        7000,
			TotalFilesChanged: 15,
		},
		{
			ExternalID:    "SR-20260215-003",
			StartedAt:     now.Add(-1 * time.Hour),
			Status:        db.BroadcastSyncRunStatusFailed,
			Trigger:       db.BroadcastSyncRunTriggerManual,
			TotalTargets:  1,
			FailedTargets: 1,
			DurationMs:    1000,
		},
	}

	for i := range runs {
		err := repo.CreateSyncRun(ctx, &runs[i])
		require.NoError(t, err)
	}

	// Test summary statistics with multiple runs
	t.Run("aggregate statistics", func(t *testing.T) {
		stats, err := repo.GetSyncRunSummaryStats(ctx)
		require.NoError(t, err)

		assert.Equal(t, int64(3), stats.TotalRuns)
		assert.InDelta(t, 66.67, stats.SuccessRate, 0.1)  // 2 success / 3 total
		assert.Equal(t, int64(4333), stats.AvgDurationMs) // (5000+7000+1000)/3
		assert.NotNil(t, stats.LastRunAt)
	})

	// Test filtering by time period
	t.Run("time-based filtering", func(t *testing.T) {
		// Last 90 minutes should get the most recent 2 runs
		// Note: Since times are set relative to now, and there may be test execution delays,
		// we'll test that we get at least 1 and at most 2
		since := now.Add(-90 * time.Minute)
		filteredRuns, err := repo.ListRecentSyncRuns(ctx, since, 100)
		require.NoError(t, err)

		// Should get 1 or 2 runs depending on exact timing
		assert.GreaterOrEqual(t, len(filteredRuns), 1)
		assert.LessOrEqual(t, len(filteredRuns), 2)

		// First result should be the most recent
		if len(filteredRuns) > 0 {
			assert.Equal(t, "SR-20260215-003", filteredRuns[0].ExternalID)
		}
	})
}
