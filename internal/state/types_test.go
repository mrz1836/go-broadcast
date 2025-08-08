package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

func TestSyncStatus_String(t *testing.T) {
	tests := []struct {
		status   SyncStatus
		expected string
	}{
		{StatusUnknown, "unknown"},
		{StatusUpToDate, "up-to-date"},
		{StatusBehind, "behind"},
		{StatusPending, "pending"},
		{StatusConflict, "conflict"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			require.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestState_Structure(t *testing.T) {
	now := time.Now()
	state := State{
		Source: SourceState{
			Repo:         "org/template-repo",
			Branch:       "master",
			LatestCommit: "abc123",
			LastChecked:  now,
		},
		Targets: map[string]*TargetState{
			"org/service-a": {
				Repo: "org/service-a",
				SyncBranches: []SyncBranch{
					{
						Name: "chore/sync-files-20240101-120000-abc123",
						Metadata: &BranchMetadata{
							Timestamp: now.Add(-24 * time.Hour),
							CommitSHA: "abc123",
							Prefix:    "chore/sync-files",
						},
					},
				},
				OpenPRs: []gh.PR{
					{
						Number: 42,
						State:  "open",
						Title:  "Sync from source repository",
					},
				},
				LastSyncCommit: "abc123",
				LastSyncTime:   &now,
				Status:         StatusUpToDate,
			},
			"org/service-b": {
				Repo:           "org/service-b",
				SyncBranches:   []SyncBranch{},
				OpenPRs:        []gh.PR{},
				LastSyncCommit: "def456",
				LastSyncTime:   nil,
				Status:         StatusBehind,
			},
		},
	}

	// Verify source state
	require.Equal(t, "org/template-repo", state.Source.Repo)
	require.Equal(t, "master", state.Source.Branch)
	require.Equal(t, "abc123", state.Source.LatestCommit)
	require.Equal(t, now, state.Source.LastChecked)

	// Verify targets
	require.Len(t, state.Targets, 2)

	// Verify service-a
	serviceA := state.Targets["org/service-a"]
	require.NotNil(t, serviceA)
	require.Equal(t, "org/service-a", serviceA.Repo)
	require.Len(t, serviceA.SyncBranches, 1)
	require.Len(t, serviceA.OpenPRs, 1)
	require.Equal(t, "abc123", serviceA.LastSyncCommit)
	require.NotNil(t, serviceA.LastSyncTime)
	require.Equal(t, StatusUpToDate, serviceA.Status)

	// Verify service-b
	serviceB := state.Targets["org/service-b"]
	require.NotNil(t, serviceB)
	require.Equal(t, "org/service-b", serviceB.Repo)
	require.Empty(t, serviceB.SyncBranches)
	require.Empty(t, serviceB.OpenPRs)
	require.Equal(t, "def456", serviceB.LastSyncCommit)
	require.Nil(t, serviceB.LastSyncTime)
	require.Equal(t, StatusBehind, serviceB.Status)
}

func TestSourceState_DefaultValues(t *testing.T) {
	var source SourceState
	require.Empty(t, source.Repo)
	require.Empty(t, source.Branch)
	require.Empty(t, source.LatestCommit)
	require.True(t, source.LastChecked.IsZero())
}

func TestTargetState_DefaultValues(t *testing.T) {
	var target TargetState
	require.Empty(t, target.Repo)
	require.Nil(t, target.SyncBranches)
	require.Nil(t, target.OpenPRs)
	require.Empty(t, target.LastSyncCommit)
	require.Nil(t, target.LastSyncTime)
	require.Equal(t, SyncStatus(""), target.Status) // Zero value
	require.Nil(t, target.DirectorySync)            // Should be nil for backward compatibility
}

func TestSyncBranch_Structure(t *testing.T) {
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	branch := SyncBranch{
		Name: "chore/sync-files-20240101-120000-abc123",
		Metadata: &BranchMetadata{
			Timestamp: timestamp,
			CommitSHA: "abc123",
			Prefix:    "chore/sync-files",
		},
	}

	require.Equal(t, "chore/sync-files-20240101-120000-abc123", branch.Name)
	require.NotNil(t, branch.Metadata)
	require.Equal(t, timestamp, branch.Metadata.Timestamp)
	require.Equal(t, "abc123", branch.Metadata.CommitSHA)
	require.Equal(t, "chore/sync-files", branch.Metadata.Prefix)
}

func TestBranchMetadata_Structure(t *testing.T) {
	metadata := &BranchMetadata{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		CommitSHA: "abc123def456",
		Prefix:    "chore/sync-files",
	}

	require.Equal(t, 2024, metadata.Timestamp.Year())
	require.Equal(t, time.January, metadata.Timestamp.Month())
	require.Equal(t, 1, metadata.Timestamp.Day())
	require.Equal(t, 12, metadata.Timestamp.Hour())
	require.Equal(t, 0, metadata.Timestamp.Minute())
	require.Equal(t, 0, metadata.Timestamp.Second())
	require.Equal(t, "abc123def456", metadata.CommitSHA)
	require.Equal(t, "chore/sync-files", metadata.Prefix)
}

func TestState_EmptyTargets(t *testing.T) {
	state := State{
		Source: SourceState{
			Repo: "org/template",
		},
		Targets: map[string]*TargetState{},
	}

	require.Empty(t, state.Targets)
	require.Equal(t, "org/template", state.Source.Repo)
}

func TestState_NilTargets(t *testing.T) {
	state := State{
		Source: SourceState{
			Repo: "org/template",
		},
		Targets: nil,
	}

	require.Nil(t, state.Targets)
}

func TestTargetState_MultipleOpenPRs(t *testing.T) {
	target := TargetState{
		Repo: "org/service",
		OpenPRs: []gh.PR{
			{Number: 1, State: "open", Title: "First PR"},
			{Number: 2, State: "open", Title: "Second PR"},
			{Number: 3, State: "open", Title: "Third PR"},
		},
		Status: StatusPending,
	}

	require.Len(t, target.OpenPRs, 3)
	require.Equal(t, 1, target.OpenPRs[0].Number)
	require.Equal(t, 2, target.OpenPRs[1].Number)
	require.Equal(t, 3, target.OpenPRs[2].Number)
	require.Equal(t, StatusPending, target.Status)
}

func TestTargetState_MultipleSyncBranches(t *testing.T) {
	now := time.Now()
	target := TargetState{
		Repo: "org/service",
		SyncBranches: []SyncBranch{
			{
				Name: "chore/sync-files-20240101-120000-abc123",
				Metadata: &BranchMetadata{
					Timestamp: now.Add(-48 * time.Hour),
					CommitSHA: "abc123",
					Prefix:    "chore/sync-files",
				},
			},
			{
				Name: "chore/sync-files-20240102-120000-def456",
				Metadata: &BranchMetadata{
					Timestamp: now.Add(-24 * time.Hour),
					CommitSHA: "def456",
					Prefix:    "chore/sync-files",
				},
			},
			{
				Name: "chore/sync-files-20240103-120000-ghi789",
				Metadata: &BranchMetadata{
					Timestamp: now,
					CommitSHA: "ghi789",
					Prefix:    "chore/sync-files",
				},
			},
		},
	}

	require.Len(t, target.SyncBranches, 3)

	// Verify branches are in expected order
	require.Equal(t, "abc123", target.SyncBranches[0].Metadata.CommitSHA)
	require.Equal(t, "def456", target.SyncBranches[1].Metadata.CommitSHA)
	require.Equal(t, "ghi789", target.SyncBranches[2].Metadata.CommitSHA)
}

func TestSyncBranch_NilMetadata(t *testing.T) {
	branch := SyncBranch{
		Name:     "chore/sync-files-invalid",
		Metadata: nil,
	}

	require.Equal(t, "chore/sync-files-invalid", branch.Name)
	require.Nil(t, branch.Metadata)
}

func TestBranchMetadata_EmptyCommitSHA(t *testing.T) {
	metadata := &BranchMetadata{
		Timestamp: time.Now(),
		CommitSHA: "",
		Prefix:    "chore/sync-files",
	}

	require.Empty(t, metadata.CommitSHA)
	require.Equal(t, "chore/sync-files", metadata.Prefix)
	require.False(t, metadata.Timestamp.IsZero())
}

func TestSyncStatus_Validation(t *testing.T) {
	// Test all valid statuses
	validStatuses := []SyncStatus{
		StatusUnknown,
		StatusUpToDate,
		StatusBehind,
		StatusPending,
		StatusConflict,
	}

	for _, status := range validStatuses {
		require.NotEmpty(t, status)
	}

	// Test that status values are distinct
	statusMap := make(map[SyncStatus]bool)
	for _, status := range validStatuses {
		require.False(t, statusMap[status], "Duplicate status value: %s", status)
		statusMap[status] = true
	}
}

func TestState_ComplexScenario(t *testing.T) {
	// Create a complex state with multiple targets in different states
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	state := State{
		Source: SourceState{
			Repo:         "org/template",
			Branch:       "develop",
			LatestCommit: "latest123",
			LastChecked:  now,
		},
		Targets: map[string]*TargetState{
			// Up-to-date target
			"org/service-current": {
				Repo:           "org/service-current",
				LastSyncCommit: "latest123",
				LastSyncTime:   &now,
				Status:         StatusUpToDate,
				SyncBranches:   []SyncBranch{},
				OpenPRs:        []gh.PR{},
			},
			// Behind target with open PR
			"org/service-behind": {
				Repo:           "org/service-behind",
				LastSyncCommit: "old456",
				LastSyncTime:   &lastWeek,
				Status:         StatusPending,
				SyncBranches: []SyncBranch{
					{
						Name: "chore/sync-files-20240101-120000-latest123",
						Metadata: &BranchMetadata{
							Timestamp: yesterday,
							CommitSHA: "latest123",
							Prefix:    "chore/sync-files",
						},
					},
				},
				OpenPRs: []gh.PR{
					{
						Number: 99,
						State:  "open",
						Title:  "Sync from source repository",
						Head: struct {
							Ref string `json:"ref"`
							SHA string `json:"sha"`
						}{
							Ref: "chore/sync-files-20240101-120000-latest123",
						},
					},
				},
			},
			// Conflicted target
			"org/service-conflict": {
				Repo:           "org/service-conflict",
				LastSyncCommit: "conflict789",
				LastSyncTime:   &yesterday,
				Status:         StatusConflict,
				SyncBranches: []SyncBranch{
					{
						Name: "chore/sync-files-20240101-120000-latest123",
						Metadata: &BranchMetadata{
							Timestamp: now,
							CommitSHA: "latest123",
							Prefix:    "chore/sync-files",
						},
					},
				},
				OpenPRs: []gh.PR{
					{
						Number: 100,
						State:  "open",
						Title:  "Sync from source repository (conflicts)",
						Labels: []struct {
							Name string `json:"name"`
						}{
							{Name: "conflicts"},
						},
					},
				},
			},
			// Unknown status target (never synced)
			"org/service-new": {
				Repo:           "org/service-new",
				LastSyncCommit: "",
				LastSyncTime:   nil,
				Status:         StatusUnknown,
				SyncBranches:   []SyncBranch{},
				OpenPRs:        []gh.PR{},
			},
		},
	}

	// Verify we have all targets
	require.Len(t, state.Targets, 4)

	// Verify each target's status
	require.Equal(t, StatusUpToDate, state.Targets["org/service-current"].Status)
	require.Equal(t, StatusPending, state.Targets["org/service-behind"].Status)
	require.Equal(t, StatusConflict, state.Targets["org/service-conflict"].Status)
	require.Equal(t, StatusUnknown, state.Targets["org/service-new"].Status)

	// Verify sync times
	require.NotNil(t, state.Targets["org/service-current"].LastSyncTime)
	require.NotNil(t, state.Targets["org/service-behind"].LastSyncTime)
	require.NotNil(t, state.Targets["org/service-conflict"].LastSyncTime)
	require.Nil(t, state.Targets["org/service-new"].LastSyncTime)
}

func TestDirectorySyncInfo_Structure(t *testing.T) {
	now := time.Now()
	syncInfo := DirectorySyncInfo{
		DirectoryMappings: []DirectoryMappingInfo{
			{
				Source:            "src/templates",
				Destination:       "internal/templates",
				ExcludePatterns:   []string{"*.tmp", "*.log"},
				PreserveStructure: true,
				IncludeHidden:     false,
				TransformApplied:  true,
			},
		},
		SyncedFiles: &SyncedFilesInfo{
			DirectorySyncedFiles:  []string{"internal/templates/config.go", "internal/templates/handler.go"},
			IndividualSyncedFiles: []string{"README.md", "Dockerfile"},
			TotalFiles:            4,
			DirectoryFileCount:    2,
			IndividualFileCount:   2,
		},
		PerformanceMetrics: &DirectoryPerformanceMetrics{
			DirectoryMetrics: map[string]*DirectoryProcessingMetrics{
				"src/templates": {
					FilesDiscovered:    10,
					FilesProcessed:     8,
					FilesExcluded:      2,
					FilesSkipped:       0,
					FilesErrored:       0,
					DirectoriesWalked:  3,
					TotalSize:          1024,
					ProcessedSize:      512,
					ProcessingDuration: 500 * time.Millisecond,
					BinaryFilesSkipped: 0,
					BinaryFilesSize:    0,
					TransformMetrics: &TransformationMetrics{
						TransformSuccesses:       8,
						TransformErrors:          0,
						TotalTransformDuration:   200 * time.Millisecond,
						TransformCount:           8,
						AverageTransformDuration: 25 * time.Millisecond,
					},
				},
			},
			OverallMetrics: &OverallSyncMetrics{
				StartTime:           now.Add(-5 * time.Minute),
				EndTime:             now,
				Duration:            5 * time.Minute,
				TotalFilesProcessed: 8,
				TotalFilesChanged:   4,
				TotalFilesSkipped:   2,
				ProcessingTimeMs:    300000,
			},
			APIMetrics: &APISyncMetrics{
				TotalAPIRequests: 20,
				APICallsSaved:    10,
				CacheHits:        15,
				CacheMisses:      5,
				CacheHitRatio:    0.75,
			},
			ExtractedFromPR: true,
			PRNumber:        &[]int{123}[0],
		},
		LastDirectorySync: &now,
	}

	// Verify directory mappings
	require.Len(t, syncInfo.DirectoryMappings, 1)
	mapping := syncInfo.DirectoryMappings[0]
	require.Equal(t, "src/templates", mapping.Source)
	require.Equal(t, "internal/templates", mapping.Destination)
	require.Equal(t, []string{"*.tmp", "*.log"}, mapping.ExcludePatterns)
	require.True(t, mapping.PreserveStructure)
	require.False(t, mapping.IncludeHidden)
	require.True(t, mapping.TransformApplied)

	// Verify synced files info
	require.NotNil(t, syncInfo.SyncedFiles)
	require.Len(t, syncInfo.SyncedFiles.DirectorySyncedFiles, 2)
	require.Len(t, syncInfo.SyncedFiles.IndividualSyncedFiles, 2)
	require.Equal(t, 4, syncInfo.SyncedFiles.TotalFiles)
	require.Equal(t, 2, syncInfo.SyncedFiles.DirectoryFileCount)
	require.Equal(t, 2, syncInfo.SyncedFiles.IndividualFileCount)

	// Verify performance metrics
	require.NotNil(t, syncInfo.PerformanceMetrics)
	require.Len(t, syncInfo.PerformanceMetrics.DirectoryMetrics, 1)
	require.True(t, syncInfo.PerformanceMetrics.ExtractedFromPR)
	require.NotNil(t, syncInfo.PerformanceMetrics.PRNumber)
	require.Equal(t, 123, *syncInfo.PerformanceMetrics.PRNumber)

	// Verify directory processing metrics
	dirMetrics := syncInfo.PerformanceMetrics.DirectoryMetrics["src/templates"]
	require.NotNil(t, dirMetrics)
	require.Equal(t, 10, dirMetrics.FilesDiscovered)
	require.Equal(t, 8, dirMetrics.FilesProcessed)
	require.Equal(t, 2, dirMetrics.FilesExcluded)
	require.Equal(t, int64(1024), dirMetrics.TotalSize)
	require.Equal(t, int64(512), dirMetrics.ProcessedSize)

	// Verify transformation metrics
	require.NotNil(t, dirMetrics.TransformMetrics)
	require.Equal(t, 8, dirMetrics.TransformMetrics.TransformSuccesses)
	require.Equal(t, 0, dirMetrics.TransformMetrics.TransformErrors)
	require.Equal(t, 25*time.Millisecond, dirMetrics.TransformMetrics.AverageTransformDuration)

	// Verify overall metrics
	require.NotNil(t, syncInfo.PerformanceMetrics.OverallMetrics)
	require.Equal(t, 8, syncInfo.PerformanceMetrics.OverallMetrics.TotalFilesProcessed)
	require.Equal(t, 4, syncInfo.PerformanceMetrics.OverallMetrics.TotalFilesChanged)
	require.Equal(t, 5*time.Minute, syncInfo.PerformanceMetrics.OverallMetrics.Duration)

	// Verify API metrics
	require.NotNil(t, syncInfo.PerformanceMetrics.APIMetrics)
	require.Equal(t, 20, syncInfo.PerformanceMetrics.APIMetrics.TotalAPIRequests)
	require.Equal(t, 10, syncInfo.PerformanceMetrics.APIMetrics.APICallsSaved)
	require.InDelta(t, 0.75, syncInfo.PerformanceMetrics.APIMetrics.CacheHitRatio, 0.001)

	// Verify last directory sync time
	require.NotNil(t, syncInfo.LastDirectorySync)
	require.Equal(t, now, *syncInfo.LastDirectorySync)
}

func TestDirectorySyncInfo_DefaultValues(t *testing.T) {
	var syncInfo DirectorySyncInfo
	require.Nil(t, syncInfo.DirectoryMappings)
	require.Nil(t, syncInfo.SyncedFiles)
	require.Nil(t, syncInfo.PerformanceMetrics)
	require.Nil(t, syncInfo.LastDirectorySync)
}

func TestTargetState_WithDirectorySync(t *testing.T) {
	now := time.Now()
	target := TargetState{
		Repo:           "org/service-with-dirs",
		LastSyncCommit: "abc123",
		LastSyncTime:   &now,
		Status:         StatusUpToDate,
		DirectorySync: &DirectorySyncInfo{
			DirectoryMappings: []DirectoryMappingInfo{
				{
					Source:            "templates",
					Destination:       "internal/templates",
					PreserveStructure: true,
					IncludeHidden:     true,
				},
			},
			SyncedFiles: &SyncedFilesInfo{
				DirectorySyncedFiles: []string{"internal/templates/main.go"},
				TotalFiles:           1,
				DirectoryFileCount:   1,
				IndividualFileCount:  0,
			},
			LastDirectorySync: &now,
		},
	}

	// Verify target state
	require.Equal(t, "org/service-with-dirs", target.Repo)
	require.Equal(t, StatusUpToDate, target.Status)
	require.NotNil(t, target.DirectorySync)

	// Verify directory sync info
	require.Len(t, target.DirectorySync.DirectoryMappings, 1)
	require.NotNil(t, target.DirectorySync.SyncedFiles)
	require.Equal(t, 1, target.DirectorySync.SyncedFiles.TotalFiles)
	require.NotNil(t, target.DirectorySync.LastDirectorySync)
}

func TestSyncedFilesInfo_Calculations(t *testing.T) {
	syncedFiles := SyncedFilesInfo{
		DirectorySyncedFiles:  []string{"dir1/file1.go", "dir1/file2.go", "dir2/file3.go"},
		IndividualSyncedFiles: []string{"README.md", "Dockerfile"},
		TotalFiles:            5,
		DirectoryFileCount:    3,
		IndividualFileCount:   2,
	}

	// Verify counts match actual file slices
	require.Equal(t, len(syncedFiles.DirectorySyncedFiles), syncedFiles.DirectoryFileCount)
	require.Equal(t, len(syncedFiles.IndividualSyncedFiles), syncedFiles.IndividualFileCount)
	require.Equal(t, syncedFiles.DirectoryFileCount+syncedFiles.IndividualFileCount, syncedFiles.TotalFiles)
}

func TestDirectoryProcessingMetrics_CompleteScenario(t *testing.T) {
	metrics := DirectoryProcessingMetrics{
		FilesDiscovered:    50,
		FilesProcessed:     45,
		FilesExcluded:      3,
		FilesSkipped:       2,
		FilesErrored:       0,
		DirectoriesWalked:  5,
		TotalSize:          1024 * 1024, // 1MB
		ProcessedSize:      900 * 1024,  // 900KB
		ProcessingDuration: 2 * time.Second,
		BinaryFilesSkipped: 2,
		BinaryFilesSize:    100 * 1024, // 100KB
		TransformMetrics: &TransformationMetrics{
			TransformSuccesses:       45,
			TransformErrors:          0,
			TotalTransformDuration:   500 * time.Millisecond,
			TransformCount:           45,
			AverageTransformDuration: 11 * time.Millisecond, // ~500ms / 45
		},
	}

	// Verify basic metrics
	require.Equal(t, 50, metrics.FilesDiscovered)
	require.Equal(t, 45, metrics.FilesProcessed)
	require.Equal(t, 3, metrics.FilesExcluded)
	require.Equal(t, 2, metrics.FilesSkipped)
	require.Equal(t, 0, metrics.FilesErrored)

	// Verify size calculations make sense
	require.Greater(t, metrics.TotalSize, metrics.ProcessedSize)
	require.Equal(t, int64(1024*1024), metrics.TotalSize)
	require.Equal(t, int64(900*1024), metrics.ProcessedSize)

	// Verify binary file handling
	require.Equal(t, 2, metrics.BinaryFilesSkipped)
	require.Equal(t, int64(100*1024), metrics.BinaryFilesSize)

	// Verify transformation metrics
	require.NotNil(t, metrics.TransformMetrics)
	require.Equal(t, 45, metrics.TransformMetrics.TransformSuccesses)
	require.Equal(t, 0, metrics.TransformMetrics.TransformErrors)
	require.Equal(t, 45, metrics.TransformMetrics.TransformCount)
}

func TestAPISyncMetrics_CacheRatioCalculation(t *testing.T) {
	// Test perfect cache hit ratio
	perfectMetrics := APISyncMetrics{
		TotalAPIRequests: 20,
		APICallsSaved:    10,
		CacheHits:        20,
		CacheMisses:      0,
		CacheHitRatio:    1.0,
	}

	require.InDelta(t, 1.0, perfectMetrics.CacheHitRatio, 0.001)
	require.Equal(t, 20, perfectMetrics.CacheHits)
	require.Equal(t, 0, perfectMetrics.CacheMisses)

	// Test no cache hits
	noCacheMetrics := APISyncMetrics{
		TotalAPIRequests: 30,
		APICallsSaved:    0,
		CacheHits:        0,
		CacheMisses:      30,
		CacheHitRatio:    0.0,
	}

	require.InDelta(t, 0.0, noCacheMetrics.CacheHitRatio, 0.001)
	require.Equal(t, 0, noCacheMetrics.CacheHits)
	require.Equal(t, 30, noCacheMetrics.CacheMisses)

	// Test partial cache hits
	partialMetrics := APISyncMetrics{
		TotalAPIRequests: 40,
		APICallsSaved:    15,
		CacheHits:        30,
		CacheMisses:      10,
		CacheHitRatio:    0.75, // 30 / (30 + 10)
	}

	require.InDelta(t, 0.75, partialMetrics.CacheHitRatio, 0.001)
	require.Equal(t, 30, partialMetrics.CacheHits)
	require.Equal(t, 10, partialMetrics.CacheMisses)
}

func TestDirectoryPerformanceMetrics_PRExtraction(t *testing.T) {
	prNumber := 456
	metrics := DirectoryPerformanceMetrics{
		ExtractedFromPR: true,
		PRNumber:        &prNumber,
		DirectoryMetrics: map[string]*DirectoryProcessingMetrics{
			"src/config": {
				FilesDiscovered: 5,
				FilesProcessed:  5,
			},
		},
	}

	require.True(t, metrics.ExtractedFromPR)
	require.NotNil(t, metrics.PRNumber)
	require.Equal(t, 456, *metrics.PRNumber)
	require.Len(t, metrics.DirectoryMetrics, 1)
	require.Contains(t, metrics.DirectoryMetrics, "src/config")
}

func TestBackwardCompatibility_ExistingStateStillWorks(t *testing.T) {
	// Create a target state without directory sync info (simulating existing state)
	now := time.Now()
	target := TargetState{
		Repo:           "org/legacy-service",
		LastSyncCommit: "old123",
		LastSyncTime:   &now,
		Status:         StatusUpToDate,
		// DirectorySync intentionally omitted to test backward compatibility
	}

	// Verify all existing fields work
	require.Equal(t, "org/legacy-service", target.Repo)
	require.Equal(t, "old123", target.LastSyncCommit)
	require.NotNil(t, target.LastSyncTime)
	require.Equal(t, StatusUpToDate, target.Status)

	// Verify new field is nil (backward compatible)
	require.Nil(t, target.DirectorySync)

	// Verify we can safely check for directory sync presence
	hasDirectorySync := target.DirectorySync != nil
	require.False(t, hasDirectorySync)

	// Verify we can safely add directory sync to existing state
	target.DirectorySync = &DirectorySyncInfo{
		DirectoryMappings: []DirectoryMappingInfo{
			{
				Source:      "new-templates",
				Destination: "templates",
			},
		},
	}

	require.NotNil(t, target.DirectorySync)
	require.Len(t, target.DirectorySync.DirectoryMappings, 1)
}
