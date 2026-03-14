package state

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBranchPatternCacheRace tests concurrent access to the regex pattern cache.
// Run with: go test -race ./internal/state/...
func TestBranchPatternCacheRace(t *testing.T) {
	const (
		numGoroutines = 50
		numIterations = 100
	)

	prefixes := []string{
		"chore/sync-files",
		"sync/deploy",
		"feature/sync",
		"test/prefix",
		"custom/branch",
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				prefix := prefixes[(id+j)%len(prefixes)]
				pattern := getBranchPattern(prefix)

				// Verify pattern is valid and works
				branchName := prefix + "-group-20240115-120530-abc123"
				matches := pattern.FindStringSubmatch(branchName)
				assert.NotNil(t, matches, "Pattern should match branch name")
			}
		}(i)
	}

	wg.Wait()
}

// TestParseSyncBranchNameWithPrefixRace tests concurrent branch parsing
func TestParseSyncBranchNameWithPrefixRace(t *testing.T) {
	const (
		numGoroutines = 50
		numIterations = 50
	)

	branches := []struct {
		name   string
		prefix string
	}{
		{"chore/sync-files-default-20240115-120530-abc123def", "chore/sync-files"},
		{"sync/deploy-prod-20240620-093000-fedcba987", "sync/deploy"},
		{"feature/sync-config-staging-20240720-140000-xyz789", "feature/sync-config"},
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				branch := branches[(id+j)%len(branches)]
				metadata, err := parseSyncBranchNameWithPrefix(branch.name, branch.prefix)

				if err == nil {
					assert.NotNil(t, metadata)
					assert.Equal(t, branch.prefix, metadata.Prefix)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestValidateBranchPrefixRace tests concurrent prefix validation
func TestValidateBranchPrefixRace(t *testing.T) {
	const (
		numGoroutines = 50
		numIterations = 100
	)

	prefixes := []struct {
		prefix string
		valid  bool
	}{
		{"chore/sync-files", true},
		{"sync/deploy", true},
		{"invalid prefix", false},
		{"test@invalid", false},
		{"valid_prefix", true},
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				testCase := prefixes[(id+j)%len(prefixes)]
				err := ValidateBranchPrefix(testCase.prefix)

				if testCase.valid {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestStateTargetsMapConcurrentRead tests concurrent read access to State.Targets
func TestStateTargetsMapConcurrentRead(_ *testing.T) {
	const numGoroutines = 50

	// Create a pre-populated state
	state := &State{
		Source: SourceState{
			Repo:         "org/template",
			Branch:       "master",
			LatestCommit: "abc123",
			LastChecked:  time.Now(),
		},
		Targets: make(map[string]*TargetState),
	}

	// Pre-populate targets
	for i := 0; i < 100; i++ {
		repoName := string(rune('A' + i%26))
		state.Targets[repoName] = &TargetState{
			Repo:           repoName,
			LastSyncCommit: "xyz789",
			Status:         StatusBehind,
		}
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent reads - this should NOT cause a race
	for i := 0; i < numGoroutines; i++ {
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				// Read operations
				_ = state.Source.LatestCommit
				_ = len(state.Targets)

				for _, target := range state.Targets {
					_ = target.Repo
					_ = target.Status
					_ = target.LastSyncCommit
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestBranchMetadataConcurrentAccess tests concurrent access to BranchMetadata parsing
func TestBranchMetadataConcurrentAccess(t *testing.T) {
	const (
		numGoroutines = 30
		numIterations = 50
	)

	// Use only valid branch names that will parse successfully
	// Commit SHA must be hex characters only [a-fA-F0-9]+
	testBranches := []string{
		"chore/sync-files-default-20240115-120530-abc123def",
		"chore/sync-files-ci-cd-20240220-093045-fedcba987",
		"chore/sync-files-platform-20240315-160000-aabbccdd99",
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				branchName := testBranches[(id+j)%len(testBranches)]
				metadata, err := parseSyncBranchName(branchName)
				// All test branches should parse successfully
				if err != nil {
					t.Errorf("goroutine %d: unexpected error parsing %s: %v", id, branchName, err)
					return
				}
				if metadata == nil {
					t.Errorf("goroutine %d: metadata is nil for %s", id, branchName)
					return
				}
				if metadata.Prefix != "chore/sync-files" {
					t.Errorf("goroutine %d: unexpected prefix %s for %s", id, metadata.Prefix, branchName)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestFormatSyncBranchNameConcurrent tests concurrent branch name formatting
func TestFormatSyncBranchNameConcurrent(t *testing.T) {
	const (
		numGoroutines = 30
		numIterations = 50
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				timestamp := time.Date(2024, time.Month(j%12+1), id%28+1, j%24, j%60, 0, 0, time.UTC)
				result := FormatSyncBranchName("chore/sync-files", "default", timestamp, "abc123")

				assert.Contains(t, result, "chore/sync-files-default-")
				assert.Contains(t, result, "-abc123")
			}
		}(i)
	}

	wg.Wait()
}
