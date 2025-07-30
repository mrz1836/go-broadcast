// Package state provides sync state discovery and management
package state

import (
	"time"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// State represents the complete sync state across all repositories
type State struct {
	// Source contains the state of the source repository
	Source SourceState

	// Targets contains the state of each target repository
	Targets map[string]*TargetState
}

// SourceState represents the state of the source repository
type SourceState struct {
	// Repo is the repository name (e.g., "org/template-repo")
	Repo string

	// Branch is the branch being synced from
	Branch string

	// LatestCommit is the SHA of the latest commit
	LatestCommit string

	// LastChecked is when this state was last updated
	LastChecked time.Time
}

// TargetState represents the sync state of a target repository
type TargetState struct {
	// Repo is the repository name (e.g., "org/service-a")
	Repo string

	// SyncBranches contains all sync branches found
	SyncBranches []SyncBranch

	// OpenPRs contains all open sync PRs
	OpenPRs []gh.PR

	// LastSyncCommit is the SHA of the last synced commit
	LastSyncCommit string

	// LastSyncTime is when the last sync occurred
	LastSyncTime *time.Time

	// Status indicates the current sync status
	Status SyncStatus
}

// SyncBranch represents a sync branch with parsed metadata
type SyncBranch struct {
	// Name is the full branch name
	Name string

	// Metadata contains parsed information from the branch name
	Metadata *BranchMetadata
}

// BranchMetadata contains information parsed from sync branch names
// Format: chore/sync-files-YYYYMMDD-HHMMSS-{commit}
type BranchMetadata struct {
	// Timestamp is when this sync branch was created
	Timestamp time.Time

	// CommitSHA is the source commit this branch was created from
	CommitSHA string

	// Prefix is the branch prefix (e.g., "chore/sync-files")
	Prefix string
}

// SyncStatus represents the status of a sync operation
type SyncStatus string

const (
	// StatusUnknown indicates the sync status cannot be determined
	StatusUnknown SyncStatus = "unknown"

	// StatusUpToDate indicates the target is synced with source
	StatusUpToDate SyncStatus = "up-to-date"

	// StatusBehind indicates the target is behind the source
	StatusBehind SyncStatus = "behind"

	// StatusPending indicates a sync is in progress (PR open)
	StatusPending SyncStatus = "pending"

	// StatusConflict indicates there are conflicts preventing sync
	StatusConflict SyncStatus = "conflict"
)
