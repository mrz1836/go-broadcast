package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
)

func uintPtr(v uint) *uint { return &v }
func intPtr(v int) *int    { return &v }

func TestConvertSyncRunToDB(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ended := now.Add(5 * time.Minute)

	run := &BroadcastSyncRun{
		ID:                1,
		ExternalID:        "SR-20260219-abc123",
		GroupID:           uintPtr(10),
		SourceRepoID:      uintPtr(20),
		SourceBranch:      "main",
		SourceCommit:      "abc123def456",
		StartedAt:         now,
		EndedAt:           &ended,
		DurationMs:        300000,
		Status:            "completed",
		Trigger:           "manual",
		TotalTargets:      5,
		SuccessfulTargets: 3,
		FailedTargets:     1,
		SkippedTargets:    1,
		TotalFilesChanged: 10,
		TotalLinesAdded:   100,
		TotalLinesRemoved: 50,
		ErrorSummary:      "1 target failed",
	}

	dbRun := convertSyncRunToDB(run)
	require.NotNil(t, dbRun)

	assert.Equal(t, uint(1), dbRun.ID)
	assert.Equal(t, "SR-20260219-abc123", dbRun.ExternalID)
	assert.Equal(t, uintPtr(10), dbRun.GroupID)
	assert.Equal(t, uintPtr(20), dbRun.SourceRepoID)
	assert.Equal(t, "main", dbRun.SourceBranch)
	assert.Equal(t, "abc123def456", dbRun.SourceCommit)
	assert.Equal(t, now, dbRun.StartedAt)
	assert.Equal(t, &ended, dbRun.EndedAt)
	assert.Equal(t, int64(300000), dbRun.DurationMs)
	assert.Equal(t, "completed", dbRun.Status)
	assert.Equal(t, "manual", dbRun.Trigger)
	assert.Equal(t, 5, dbRun.TotalTargets)
	assert.Equal(t, 3, dbRun.SuccessfulTargets)
	assert.Equal(t, 1, dbRun.FailedTargets)
	assert.Equal(t, 1, dbRun.SkippedTargets)
	assert.Equal(t, 10, dbRun.TotalFilesChanged)
	assert.Equal(t, 100, dbRun.TotalLinesAdded)
	assert.Equal(t, 50, dbRun.TotalLinesRemoved)
	assert.Equal(t, "1 target failed", dbRun.ErrorSummary)
}

func TestConvertTargetResultToDB(t *testing.T) {
	t.Parallel()

	t.Run("with error message", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		ended := now.Add(time.Minute)

		result := &BroadcastSyncTargetResult{
			ID:                 1,
			BroadcastSyncRunID: 10,
			TargetID:           20,
			RepoID:             30,
			StartedAt:          now,
			EndedAt:            &ended,
			DurationMs:         60000,
			Status:             "failed",
			BranchName:         "sync/main",
			SourceCommitSHA:    "abc123",
			FilesProcessed:     5,
			FilesChanged:       3,
			FilesDeleted:       1,
			LinesAdded:         50,
			LinesRemoved:       20,
			BytesChanged:       1024,
			PRNumber:           intPtr(42),
			PRURL:              "https://github.com/org/repo/pull/42",
			PRState:            "open",
			ErrorMessage:       "sync failed: permission denied",
		}

		dbResult := convertTargetResultToDB(result)
		require.NotNil(t, dbResult)

		assert.Equal(t, uint(1), dbResult.ID)
		assert.Equal(t, uint(10), dbResult.BroadcastSyncRunID)
		assert.Equal(t, uint(20), dbResult.TargetID)
		assert.Equal(t, uint(30), dbResult.RepoID)
		assert.Equal(t, "failed", dbResult.Status)
		assert.Equal(t, "sync/main", dbResult.BranchName)
		assert.Equal(t, intPtr(42), dbResult.PRNumber)
		assert.Equal(t, "sync failed: permission denied", dbResult.ErrorMessage)
		assert.NotNil(t, dbResult.ErrorDetails)
		assert.Equal(t, "sync failed: permission denied", dbResult.ErrorDetails["message"])
	})

	t.Run("without error message", func(t *testing.T) {
		t.Parallel()

		result := &BroadcastSyncTargetResult{
			ID:     2,
			Status: "success",
		}

		dbResult := convertTargetResultToDB(result)
		require.NotNil(t, dbResult)
		assert.Equal(t, "success", dbResult.Status)
		assert.Nil(t, dbResult.ErrorDetails)
	})
}

func TestConvertFileChangeToDB(t *testing.T) {
	t.Parallel()

	change := &BroadcastSyncFileChange{
		ID:                          1,
		BroadcastSyncTargetResultID: 10,
		FilePath:                    "path/to/file.go",
		SourcePath:                  "src/file.go",
		ChangeType:                  FileChangeTypeAdded,
		LinesAdded:                  30,
		LinesRemoved:                0,
		SizeBytes:                   2048,
		Position:                    0,
	}

	dbChange := convertFileChangeToDB(change)
	assert.Equal(t, uint(1), dbChange.ID)
	assert.Equal(t, uint(10), dbChange.BroadcastSyncTargetResultID)
	assert.Equal(t, "path/to/file.go", dbChange.FilePath)
	assert.Equal(t, "src/file.go", dbChange.SourcePath)
	assert.Equal(t, "added", dbChange.ChangeType)
	assert.Equal(t, 30, dbChange.LinesAdded)
	assert.Equal(t, 0, dbChange.LinesRemoved)
	assert.Equal(t, int64(2048), dbChange.SizeBytes)
	assert.Equal(t, 0, dbChange.Position)
}

func TestNewDBMetricsAdapter(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	repoRepo := db.NewRepoRepository(testDB)
	targetRepo := db.NewTargetRepository(testDB)
	groupRepo := db.NewGroupRepository(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, repoRepo, targetRepo, groupRepo)
	require.NotNil(t, adapter)
}

func TestDBMetricsAdapter_LookupGroupID_NotConfigured(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, nil, nil, nil)

	_, err := adapter.LookupGroupID(context.Background(), "some-group")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group repository not configured")
}

func TestDBMetricsAdapter_LookupRepoID_NotConfigured(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, nil, nil, nil)

	_, err := adapter.LookupRepoID(context.Background(), "org/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo repository not configured")
}

func TestDBMetricsAdapter_LookupTargetID_NotConfigured(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, nil, nil, nil)

	_, err := adapter.LookupTargetID(context.Background(), 1, "org/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target repository not configured")
}

func TestDBMetricsAdapter_LookupRepoID_InvalidFormat(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	repoRepo := db.NewRepoRepository(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, repoRepo, nil, nil)

	_, err := adapter.LookupRepoID(context.Background(), "invalid-no-slash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repo full name")
}

func TestDBMetricsAdapter_LookupGroupID_NotFound(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	groupRepo := db.NewGroupRepository(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, nil, nil, groupRepo)

	_, err := adapter.LookupGroupID(context.Background(), "nonexistent-group")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to look up group")
}

func TestDBMetricsAdapter_LookupRepoID_NotFound(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	repoRepo := db.NewRepoRepository(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, repoRepo, nil, nil)

	_, err := adapter.LookupRepoID(context.Background(), "nonexistent/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to look up repo")
}

func TestDBMetricsAdapter_LookupTargetID_NotFound(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	targetRepo := db.NewTargetRepository(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, nil, targetRepo, nil)

	_, err := adapter.LookupTargetID(context.Background(), 999, "nonexistent/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to look up target")
}

func TestDBMetricsAdapter_CreateSyncRun(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, nil, nil, nil)

	run := &BroadcastSyncRun{
		ExternalID:   "SR-20260219-test001",
		SourceBranch: "main",
		Status:       SyncRunStatusRunning,
		Trigger:      "manual",
		StartedAt:    time.Now(),
	}

	err := adapter.CreateSyncRun(context.Background(), run)
	require.NoError(t, err)
	assert.Positive(t, run.ID, "ID should be populated after create")
}

func TestDBMetricsAdapter_UpdateSyncRun(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, nil, nil, nil)

	// First create a run
	run := &BroadcastSyncRun{
		ExternalID:   "SR-20260219-test002",
		SourceBranch: "main",
		Status:       SyncRunStatusRunning,
		Trigger:      "manual",
		StartedAt:    time.Now(),
	}
	require.NoError(t, adapter.CreateSyncRun(context.Background(), run))
	require.Positive(t, run.ID)

	// Now update it
	run.Status = SyncRunStatusSuccess
	err := adapter.UpdateSyncRun(context.Background(), run)
	require.NoError(t, err)
}

func TestDBMetricsAdapter_CreateTargetResult(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, nil, nil, nil)

	// Create a parent run first (FK requirement)
	run := &BroadcastSyncRun{
		ExternalID:   "SR-20260219-test003",
		SourceBranch: "main",
		Status:       SyncRunStatusRunning,
		Trigger:      "manual",
		StartedAt:    time.Now(),
	}
	require.NoError(t, adapter.CreateSyncRun(context.Background(), run))
	require.Positive(t, run.ID)

	result := &BroadcastSyncTargetResult{
		BroadcastSyncRunID: run.ID,
		TargetID:           1,
		RepoID:             1,
		Status:             "success",
		StartedAt:          time.Now(),
	}

	err := adapter.CreateTargetResult(context.Background(), result)
	require.NoError(t, err)
	assert.Positive(t, result.ID, "ID should be populated after create")
}

func TestDBMetricsAdapter_CreateFileChanges(t *testing.T) {
	t.Parallel()

	testDB := db.TestDB(t)
	syncRepo := db.NewBroadcastSyncRepo(testDB)
	adapter := NewDBMetricsAdapter(syncRepo, nil, nil, nil)

	t.Run("empty slice returns nil", func(t *testing.T) {
		t.Parallel()

		err := adapter.CreateFileChanges(context.Background(), []BroadcastSyncFileChange{})
		require.NoError(t, err)
	})

	t.Run("non-empty slice persisted without error", func(t *testing.T) {
		t.Parallel()

		// Create a run + target result to satisfy FKs
		localDB := db.TestDB(t)
		localSyncRepo := db.NewBroadcastSyncRepo(localDB)
		localAdapter := NewDBMetricsAdapter(localSyncRepo, nil, nil, nil)

		run := &BroadcastSyncRun{
			ExternalID:   "SR-20260219-test004",
			SourceBranch: "main",
			Status:       SyncRunStatusRunning,
			Trigger:      "manual",
			StartedAt:    time.Now(),
		}
		require.NoError(t, localAdapter.CreateSyncRun(context.Background(), run))

		result := &BroadcastSyncTargetResult{
			BroadcastSyncRunID: run.ID,
			TargetID:           1,
			RepoID:             1,
			Status:             "success",
			StartedAt:          time.Now(),
		}
		require.NoError(t, localAdapter.CreateTargetResult(context.Background(), result))

		changes := []BroadcastSyncFileChange{
			{
				BroadcastSyncTargetResultID: result.ID,
				FilePath:                    "README.md",
				ChangeType:                  FileChangeTypeModified,
				LinesAdded:                  5,
				LinesRemoved:                2,
			},
			{
				BroadcastSyncTargetResultID: result.ID,
				FilePath:                    "main.go",
				ChangeType:                  FileChangeTypeAdded,
				LinesAdded:                  20,
			},
		}

		err := localAdapter.CreateFileChanges(context.Background(), changes)
		require.NoError(t, err)
	})
}
