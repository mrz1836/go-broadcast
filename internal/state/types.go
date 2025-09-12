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

	// Branch is the target branch for PRs (defaults to repo's default branch)
	Branch string

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

	// DirectorySync contains directory sync information
	DirectorySync *DirectorySyncInfo `json:"directory_sync,omitempty"`
}

// SyncBranch represents a sync branch with parsed metadata
type SyncBranch struct {
	// Name is the full branch name
	Name string

	// Metadata contains parsed information from the branch name
	Metadata *BranchMetadata
}

// BranchMetadata contains information parsed from sync branch names
// Format: chore/sync-files-{groupID}-YYYYMMDD-HHMMSS-{commit}
type BranchMetadata struct {
	// Timestamp is when this sync branch was created
	Timestamp time.Time

	// CommitSHA is the source commit this branch was created from
	CommitSHA string

	// Prefix is the branch prefix (e.g., "chore/sync-files")
	Prefix string

	// GroupID is the group identifier that created this sync
	GroupID string
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

// DirectorySyncInfo holds directory-specific sync metadata and performance metrics
type DirectorySyncInfo struct {
	// DirectoryMappings contains the directory mapping configurations used in syncs
	DirectoryMappings []DirectoryMappingInfo `json:"directory_mappings,omitempty"`

	// SyncedFiles tracks which files came from directory sync vs individual file sync
	SyncedFiles *SyncedFilesInfo `json:"synced_files,omitempty"`

	// PerformanceMetrics contains directory sync performance metrics extracted from PR metadata
	PerformanceMetrics *DirectoryPerformanceMetrics `json:"performance_metrics,omitempty"`

	// LastDirectorySync is when the last directory sync occurred
	LastDirectorySync *time.Time `json:"last_directory_sync,omitempty"`
}

// DirectoryMappingInfo contains information about directory mappings used in sync
type DirectoryMappingInfo struct {
	// Source is the source directory path
	Source string `json:"source"`

	// Destination is the destination directory path
	Destination string `json:"destination"`

	// ExcludePatterns are the glob patterns that were excluded
	ExcludePatterns []string `json:"exclude_patterns,omitempty"`

	// PreserveStructure indicates if nested structure was preserved
	PreserveStructure bool `json:"preserve_structure"`

	// IncludeHidden indicates if hidden files were included
	IncludeHidden bool `json:"include_hidden"`

	// TransformApplied indicates if transformations were applied
	TransformApplied bool `json:"transform_applied"`
}

// SyncedFilesInfo tracks which files came from directory sync vs individual file sync
type SyncedFilesInfo struct {
	// DirectorySyncedFiles contains files that were synced via directory mappings
	DirectorySyncedFiles []string `json:"directory_synced_files,omitempty"`

	// IndividualSyncedFiles contains files that were synced via individual file mappings
	IndividualSyncedFiles []string `json:"individual_synced_files,omitempty"`

	// TotalFiles is the total number of files synced
	TotalFiles int `json:"total_files"`

	// DirectoryFileCount is the number of files synced via directory mappings
	DirectoryFileCount int `json:"directory_file_count"`

	// IndividualFileCount is the number of files synced via individual mappings
	IndividualFileCount int `json:"individual_file_count"`
}

// DirectoryPerformanceMetrics contains performance metrics for directory sync operations
type DirectoryPerformanceMetrics struct {
	// DirectoryMetrics maps source directory paths to their processing metrics
	DirectoryMetrics map[string]*DirectoryProcessingMetrics `json:"directory_metrics,omitempty"`

	// OverallMetrics contains aggregate metrics for the entire directory sync
	OverallMetrics *OverallSyncMetrics `json:"overall_metrics,omitempty"`

	// APIMetrics contains API usage metrics
	APIMetrics *APISyncMetrics `json:"api_metrics,omitempty"`

	// ExtractedFromPR indicates if these metrics were extracted from PR metadata
	ExtractedFromPR bool `json:"extracted_from_pr"`

	// PRNumber is the PR number these metrics were extracted from (if applicable)
	PRNumber *int `json:"pr_number,omitempty"`
}

// DirectoryProcessingMetrics tracks metrics for individual directory processing
type DirectoryProcessingMetrics struct {
	// FilesDiscovered is the total number of files discovered in the directory
	FilesDiscovered int `json:"files_discovered"`

	// FilesProcessed is the number of files actually processed
	FilesProcessed int `json:"files_processed"`

	// FilesExcluded is the number of files excluded by patterns
	FilesExcluded int `json:"files_excluded"`

	// FilesSkipped is the number of files skipped (e.g., binary files)
	FilesSkipped int `json:"files_skipped"`

	// FilesErrored is the number of files that had processing errors
	FilesErrored int `json:"files_errored"`

	// DirectoriesWalked is the number of directories traversed
	DirectoriesWalked int `json:"directories_walked"`

	// TotalSize is the total size of all discovered files in bytes
	TotalSize int64 `json:"total_size"`

	// ProcessedSize is the size of files actually processed in bytes
	ProcessedSize int64 `json:"processed_size"`

	// ProcessingDuration is the time taken to process this directory
	ProcessingDuration time.Duration `json:"processing_duration"`

	// BinaryFilesSkipped is the number of binary files that were skipped
	BinaryFilesSkipped int `json:"binary_files_skipped"`

	// BinaryFilesSize is the total size of binary files that were skipped
	BinaryFilesSize int64 `json:"binary_files_size"`

	// TransformMetrics contains transformation-related metrics
	TransformMetrics *TransformationMetrics `json:"transform_metrics,omitempty"`
}

// TransformationMetrics tracks transformation performance
type TransformationMetrics struct {
	// TransformSuccesses is the number of successful transformations
	TransformSuccesses int `json:"transform_successes"`

	// TransformErrors is the number of transformation errors
	TransformErrors int `json:"transform_errors"`

	// TotalTransformDuration is the total time spent on transformations
	TotalTransformDuration time.Duration `json:"total_transform_duration"`

	// TransformCount is the total number of transformations attempted
	TransformCount int `json:"transform_count"`

	// AverageTransformDuration is the average time per transformation
	AverageTransformDuration time.Duration `json:"average_transform_duration"`
}

// OverallSyncMetrics contains aggregate metrics for the entire sync operation
type OverallSyncMetrics struct {
	// StartTime is when the sync operation started
	StartTime time.Time `json:"start_time"`

	// EndTime is when the sync operation completed
	EndTime time.Time `json:"end_time"`

	// Duration is the total sync duration
	Duration time.Duration `json:"duration"`

	// TotalFilesProcessed is the total number of files processed across all directories
	TotalFilesProcessed int `json:"total_files_processed"`

	// TotalFilesChanged is the number of files that were actually changed
	TotalFilesChanged int `json:"total_files_changed"`

	// TotalFilesSkipped is the total number of files skipped
	TotalFilesSkipped int `json:"total_files_skipped"`

	// ProcessingTimeMs is the total processing time in milliseconds
	ProcessingTimeMs int64 `json:"processing_time_ms"`
}

// APISyncMetrics contains API usage metrics for sync operations
type APISyncMetrics struct {
	// TotalAPIRequests is the total number of API requests made
	TotalAPIRequests int `json:"total_api_requests"`

	// APICallsSaved is the number of API calls saved through optimizations
	APICallsSaved int `json:"api_calls_saved"`

	// CacheHits is the number of cache hits
	CacheHits int `json:"cache_hits"`

	// CacheMisses is the number of cache misses
	CacheMisses int `json:"cache_misses"`

	// CacheHitRatio is the cache hit ratio (0.0 to 1.0)
	CacheHitRatio float64 `json:"cache_hit_ratio"`
}
