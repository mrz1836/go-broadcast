package sync

import (
	"context"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// dbMetricsAdapter adapts the db.BroadcastSyncRepo to the SyncMetricsRecorder interface
type dbMetricsAdapter struct {
	repo db.BroadcastSyncRepo
}

// NewDBMetricsAdapter creates a new database metrics adapter
func NewDBMetricsAdapter(repo db.BroadcastSyncRepo) SyncMetricsRecorder {
	return &dbMetricsAdapter{repo: repo}
}

// CreateSyncRun creates a new sync run record
func (a *dbMetricsAdapter) CreateSyncRun(ctx context.Context, run *BroadcastSyncRun) error {
	dbRun := convertSyncRunToDB(run)
	if err := a.repo.CreateSyncRun(ctx, dbRun); err != nil {
		return err
	}
	// Copy back the generated ID
	run.ID = dbRun.ID
	return nil
}

// UpdateSyncRun updates an existing sync run record
func (a *dbMetricsAdapter) UpdateSyncRun(ctx context.Context, run *BroadcastSyncRun) error {
	dbRun := convertSyncRunToDB(run)
	return a.repo.UpdateSyncRun(ctx, dbRun)
}

// CreateTargetResult creates a new target result record
func (a *dbMetricsAdapter) CreateTargetResult(ctx context.Context, result *BroadcastSyncTargetResult) error {
	dbResult := convertTargetResultToDB(result)
	if err := a.repo.CreateTargetResult(ctx, dbResult); err != nil {
		return err
	}
	// Copy back the generated ID
	result.ID = dbResult.ID
	return nil
}

// CreateFileChanges batch creates file change records
func (a *dbMetricsAdapter) CreateFileChanges(ctx context.Context, changes []BroadcastSyncFileChange) error {
	if len(changes) == 0 {
		return nil
	}

	dbChanges := make([]db.BroadcastSyncFileChange, len(changes))
	for i, change := range changes {
		dbChanges[i] = convertFileChangeToDB(&change)
	}

	return a.repo.CreateFileChanges(ctx, dbChanges)
}

// convertSyncRunToDB converts sync engine's BroadcastSyncRun to db model
func convertSyncRunToDB(run *BroadcastSyncRun) *db.BroadcastSyncRun {
	return &db.BroadcastSyncRun{
		BaseModel: db.BaseModel{
			ID: run.ID,
		},
		ExternalID:        run.ExternalID,
		GroupID:           run.GroupID,
		SourceRepoID:      run.SourceRepoID,
		SourceBranch:      run.SourceBranch,
		SourceCommit:      run.SourceCommit,
		StartedAt:         run.StartedAt,
		EndedAt:           run.EndedAt,
		DurationMs:        run.DurationMs,
		Status:            run.Status,
		Trigger:           run.Trigger,
		TotalTargets:      run.TotalTargets,
		SuccessfulTargets: run.SuccessfulTargets,
		FailedTargets:     run.FailedTargets,
		SkippedTargets:    run.SkippedTargets,
		TotalFilesChanged: run.TotalFilesChanged,
		TotalLinesAdded:   run.TotalLinesAdded,
		TotalLinesRemoved: run.TotalLinesRemoved,
		ErrorSummary:      run.ErrorSummary,
	}
}

// convertTargetResultToDB converts sync engine's BroadcastSyncTargetResult to db model
func convertTargetResultToDB(result *BroadcastSyncTargetResult) *db.BroadcastSyncTargetResult {
	dbResult := &db.BroadcastSyncTargetResult{
		BaseModel: db.BaseModel{
			ID: result.ID,
		},
		BroadcastSyncRunID: result.BroadcastSyncRunID,
		TargetID:           result.TargetID,
		RepoID:             result.RepoID,
		StartedAt:          result.StartedAt,
		EndedAt:            result.EndedAt,
		DurationMs:         result.DurationMs,
		Status:             result.Status,
		BranchName:         result.BranchName,
		SourceCommitSHA:    result.SourceCommitSHA,
		FilesProcessed:     result.FilesProcessed,
		FilesChanged:       result.FilesChanged,
		FilesDeleted:       result.FilesDeleted,
		LinesAdded:         result.LinesAdded,
		LinesRemoved:       result.LinesRemoved,
		BytesChanged:       result.BytesChanged,
		PRNumber:           result.PRNumber,
		PRURL:              result.PRURL,
		PRState:            result.PRState,
		ErrorMessage:       result.ErrorMessage,
	}

	// Convert error details if present
	if result.ErrorMessage != "" {
		dbResult.ErrorDetails = db.Metadata{
			"message": result.ErrorMessage,
		}
	}

	return dbResult
}

// convertFileChangeToDB converts sync engine's BroadcastSyncFileChange to db model
func convertFileChangeToDB(change *BroadcastSyncFileChange) db.BroadcastSyncFileChange {
	return db.BroadcastSyncFileChange{
		BaseModel: db.BaseModel{
			ID: change.ID,
		},
		BroadcastSyncTargetResultID: change.BroadcastSyncTargetResultID,
		FilePath:                    change.FilePath,
		SourcePath:                  change.SourcePath,
		ChangeType:                  change.ChangeType,
		LinesAdded:                  change.LinesAdded,
		LinesRemoved:                change.LinesRemoved,
		SizeBytes:                   change.SizeBytes,
		Position:                    change.Position,
	}
}
