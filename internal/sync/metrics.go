package sync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

// SyncMetricsRecorder defines the interface for recording sync metrics
// This abstraction allows the Engine to record metrics without depending on db package
type SyncMetricsRecorder interface {
	// CreateSyncRun creates a new sync run record
	CreateSyncRun(ctx context.Context, run *BroadcastSyncRun) error

	// UpdateSyncRun updates an existing sync run record
	UpdateSyncRun(ctx context.Context, run *BroadcastSyncRun) error

	// CreateTargetResult creates a new target result record
	CreateTargetResult(ctx context.Context, result *BroadcastSyncTargetResult) error

	// CreateFileChanges batch creates file change records
	CreateFileChanges(ctx context.Context, changes []BroadcastSyncFileChange) error

	// LookupGroupID resolves a config group external ID string to a DB uint ID
	LookupGroupID(ctx context.Context, groupExternalID string) (uint, error)

	// LookupRepoID resolves an "org/repo" string to a DB uint ID
	LookupRepoID(ctx context.Context, repoFullName string) (uint, error)

	// LookupTargetID resolves a group DB ID + repo full name to a target DB uint ID
	LookupTargetID(ctx context.Context, groupDBID uint, repoFullName string) (uint, error)
}

// BroadcastSyncRun is a minimal representation for the sync engine
// (Full model lives in internal/db)
type BroadcastSyncRun struct {
	ID                uint
	ExternalID        string
	GroupID           *uint
	SourceRepoID      *uint
	SourceBranch      string
	SourceCommit      string
	StartedAt         time.Time
	EndedAt           *time.Time
	DurationMs        int64
	Status            string
	Trigger           string
	TotalTargets      int
	SuccessfulTargets int
	FailedTargets     int
	SkippedTargets    int
	TotalFilesChanged int
	TotalLinesAdded   int
	TotalLinesRemoved int
	ErrorSummary      string
}

// BroadcastSyncTargetResult is a minimal representation for the sync engine
type BroadcastSyncTargetResult struct {
	ID                 uint
	BroadcastSyncRunID uint
	TargetID           uint
	RepoID             uint
	StartedAt          time.Time
	EndedAt            *time.Time
	DurationMs         int64
	Status             string
	BranchName         string
	SourceCommitSHA    string
	FilesProcessed     int
	FilesChanged       int
	FilesDeleted       int
	LinesAdded         int
	LinesRemoved       int
	BytesChanged       int64
	PRNumber           *int
	PRURL              string
	PRState            string
	ErrorMessage       string
}

// BroadcastSyncFileChange is a minimal representation for the sync engine
type BroadcastSyncFileChange struct {
	ID                          uint
	BroadcastSyncTargetResultID uint
	FilePath                    string
	SourcePath                  string
	ChangeType                  string
	LinesAdded                  int
	LinesRemoved                int
	SizeBytes                   int64
	Position                    int
}

// Status constants for sync runs
const (
	SyncRunStatusPending = "pending"
	SyncRunStatusRunning = "running"
	SyncRunStatusSuccess = "success"
	SyncRunStatusPartial = "partial"
	SyncRunStatusFailed  = "failed"
	SyncRunStatusSkipped = "skipped"
)

// Status constants for target results
const (
	TargetStatusPending   = "pending"
	TargetStatusSuccess   = "success"
	TargetStatusFailed    = "failed"
	TargetStatusSkipped   = "skipped"
	TargetStatusNoChanges = "no_changes"
)

// Change type constants for file changes
const (
	FileChangeTypeAdded    = "added"
	FileChangeTypeModified = "modified"
	FileChangeTypeDeleted  = "deleted"
)

// Trigger constants for sync runs
const (
	TriggerManual = "manual"
	TriggerCron   = "cron"
	TriggerCI     = "ci"
)

// GenerateSyncRunExternalID generates a unique external ID for a sync run
// Format: SR-{YYYYMMDD}-{random6}
func GenerateSyncRunExternalID() string {
	now := time.Now().UTC()
	dateStr := now.Format("20060102")

	// Generate 6 random hex characters
	randBytes := make([]byte, 3) // 3 bytes = 6 hex chars
	if _, err := rand.Read(randBytes); err != nil {
		// Fallback to timestamp-based randomness if crypto/rand fails
		randBytes = []byte{byte(now.UnixNano() & 0xFF), byte((now.UnixNano() >> 8) & 0xFF), byte((now.UnixNano() >> 16) & 0xFF)}
	}
	randStr := hex.EncodeToString(randBytes)

	return fmt.Sprintf("SR-%s-%s", dateStr, randStr)
}

// DetermineTrigger determines the trigger type based on environment
func DetermineTrigger(options *Options) string {
	// Check environment for CI indicators
	// GitHub Actions
	if isCI := isRunningInCI(); isCI {
		return TriggerCI
	}

	// Check if cron-specific option is set (future enhancement)
	// For now, everything else is manual
	return TriggerManual
}

// isRunningInCI checks if we're running in a CI environment
func isRunningInCI() bool {
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS", "JENKINS_URL", "BUILDKITE"}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}
