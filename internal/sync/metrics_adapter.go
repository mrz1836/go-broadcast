package sync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// Sentinel errors for metrics adapter
var (
	ErrGroupRepoNotConfigured  = errors.New("group repository not configured")
	ErrRepoRepoNotConfigured   = errors.New("repo repository not configured")
	ErrTargetRepoNotConfigured = errors.New("target repository not configured")
	ErrInvalidRepoFullName     = errors.New("invalid repo full name: expected org/repo format")
)

// dbMetricsAdapter adapts the db.BroadcastSyncRepo to the SyncMetricsRecorder interface
type dbMetricsAdapter struct {
	repo       db.BroadcastSyncRepo
	repoRepo   db.RepoRepository
	targetRepo db.TargetRepository
	groupRepo  db.GroupRepository
}

// NewDBMetricsAdapter creates a new database metrics adapter
func NewDBMetricsAdapter(repo db.BroadcastSyncRepo, repoRepo db.RepoRepository, targetRepo db.TargetRepository, groupRepo db.GroupRepository) SyncMetricsRecorder {
	return &dbMetricsAdapter{
		repo:       repo,
		repoRepo:   repoRepo,
		targetRepo: targetRepo,
		groupRepo:  groupRepo,
	}
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

// LookupGroupID resolves a config group external ID string to a DB uint ID
func (a *dbMetricsAdapter) LookupGroupID(ctx context.Context, groupExternalID string) (uint, error) {
	if a.groupRepo == nil {
		return 0, ErrGroupRepoNotConfigured
	}
	group, err := a.groupRepo.GetByExternalID(ctx, groupExternalID)
	if err != nil {
		return 0, fmt.Errorf("failed to look up group %q: %w", groupExternalID, err)
	}
	return group.ID, nil
}

// LookupRepoID resolves an "org/repo" string to a DB uint ID
func (a *dbMetricsAdapter) LookupRepoID(ctx context.Context, repoFullName string) (uint, error) {
	if a.repoRepo == nil {
		return 0, ErrRepoRepoNotConfigured
	}
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("%w: %q", ErrInvalidRepoFullName, repoFullName)
	}
	repo, err := a.repoRepo.GetByFullName(ctx, parts[0], parts[1])
	if err != nil {
		return 0, fmt.Errorf("failed to look up repo %q: %w", repoFullName, err)
	}
	return repo.ID, nil
}

// LookupTargetID resolves a group DB ID + repo full name to a target DB uint ID
func (a *dbMetricsAdapter) LookupTargetID(ctx context.Context, groupDBID uint, repoFullName string) (uint, error) {
	if a.targetRepo == nil {
		return 0, ErrTargetRepoNotConfigured
	}
	target, err := a.targetRepo.GetByRepoName(ctx, groupDBID, repoFullName)
	if err != nil {
		return 0, fmt.Errorf("failed to look up target for group %d, repo %q: %w", groupDBID, repoFullName, err)
	}
	return target.ID, nil
}

// UpdateRepoSyncTimestamp updates the repo's last broadcast sync timestamp and run ID
func (a *dbMetricsAdapter) UpdateRepoSyncTimestamp(ctx context.Context, repoID uint, syncAt time.Time, runID uint) error {
	if a.repoRepo == nil {
		return ErrRepoRepoNotConfigured
	}
	return a.repoRepo.UpdateLastBroadcastSyncTimestamp(ctx, repoID, syncAt, runID)
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
