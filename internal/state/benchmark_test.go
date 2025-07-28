package state

import (
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/benchmark"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

func BenchmarkBranchParsing(b *testing.B) {
	// Test parsing branch names to extract metadata
	branches := []string{
		"sync/template-20240115-142530-abc123",
		"sync/template-20240115-142530-def456",
		"feature/new-feature",
		"main",
		"sync/custom-prefix-20240115-142530-ghi789",
	}

	// Create a discovery service to test parsing
	discovery := NewDiscoverer(nil, nil, nil)

	benchmark.WithMemoryTracking(b, func() {
		for _, branch := range branches {
			// Use the actual ParseBranchName method
			metadata, err := discovery.ParseBranchName(branch)
			_ = metadata
			_ = err
		}
	})
}

func BenchmarkPRParsing(b *testing.B) {
	// Simulate PR info parsing using actual PR type
	prs := []gh.PR{
		{
			Number: 123,
			State:  "open",
			Title:  "Sync template updates",
			Body:   "This PR syncs the latest template changes...",
			Head: struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			}{
				Ref: "sync/template-20240115-142530-abc123",
				SHA: "abc123",
			},
			Base: struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			}{
				Ref: "main",
				SHA: "def456",
			},
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			Number: 124,
			State:  "closed",
			Title:  "Previous sync",
			Body:   "Previous template sync...",
			Head: struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			}{
				Ref: "sync/template-20240114-100000-xyz789",
				SHA: "xyz789",
			},
			Base: struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			}{
				Ref: "main",
				SHA: "ghi123",
			},
			CreatedAt: time.Now().Add(-48 * time.Hour),
			UpdatedAt: time.Now().Add(-24 * time.Hour),
		},
	}

	benchmark.WithMemoryTracking(b, func() {
		for _, pr := range prs {
			// Simulate PR parsing/analysis
			_ = pr.State == "open"
			_ = len(pr.Head.Ref) > 5 && pr.Head.Ref[:5] == "sync/"
			_ = time.Since(pr.UpdatedAt)
		}
	})
}

func BenchmarkStateComparison(b *testing.B) {
	// Benchmark comparing source and target states
	sourceState := &SourceState{
		Repo:         "org/template-repo",
		Branch:       "main",
		LatestCommit: "abc123def456789",
	}

	targetStates := map[string]*TargetState{
		"org/target-1": {
			Repo:           "org/target-1",
			LastSyncCommit: "abc123def456789",
			Status:         StatusUpToDate,
		},
		"org/target-2": {
			Repo:           "org/target-2",
			LastSyncCommit: "xyz789abc123456",
			Status:         StatusBehind,
		},
		"org/target-3": {
			Repo:           "org/target-3",
			LastSyncCommit: "",
			Status:         StatusUnknown,
		},
	}

	benchmark.WithMemoryTracking(b, func() {
		for _, target := range targetStates {
			// Compare commits
			needsSync := target.LastSyncCommit != sourceState.LatestCommit
			_ = needsSync

			// Check status
			switch target.Status {
			case StatusUpToDate:
				_ = false
			case StatusBehind:
				_ = true
			case StatusPending:
				_ = true
			case StatusConflict:
				_ = false
			default:
				_ = true
			}
		}
	})
}

func BenchmarkSyncBranchGeneration(b *testing.B) {
	// Benchmark generating sync branch names
	timestamp := time.Now()
	commit := "abc123def456789"

	benchmark.WithMemoryTracking(b, func() {
		// Simulate branch name generation
		branchName := "sync/template-" +
			timestamp.Format("20060102-150405") + "-" +
			commit[:7]
		_ = branchName
	})
}

func BenchmarkStateAggregation(b *testing.B) {
	// Benchmark aggregating state from multiple sources
	state := &State{
		Source: SourceState{
			Repo:         "org/template-repo",
			Branch:       "main",
			LatestCommit: "abc123def456789",
		},
		Targets: make(map[string]*TargetState),
	}

	// Add many targets
	for i := 0; i < 100; i++ {
		state.Targets[string(rune(i))] = &TargetState{
			Repo:           string(rune(i)),
			LastSyncCommit: "xyz789",
			Status:         StatusBehind,
		}
	}

	benchmark.WithMemoryTracking(b, func() {
		// Count targets by status
		statusCounts := make(map[SyncStatus]int)
		for _, target := range state.Targets {
			statusCounts[target.Status]++
		}

		// Calculate sync needed
		syncNeeded := 0
		for _, target := range state.Targets {
			if target.Status == StatusBehind || target.Status == StatusUnknown {
				syncNeeded++
			}
		}
		_ = syncNeeded
	})
}
