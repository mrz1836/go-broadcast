package sync

import (
	"context"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/sirupsen/logrus"
)

func BenchmarkFilterTargets(b *testing.B) {
	// Create test configuration with many targets
	cfg := &config.Config{
		Version: 1,
		Source: config.SourceConfig{
			Repo:   "org/template-repo",
			Branch: "main",
		},
		Targets: make([]config.TargetConfig, 100),
	}

	for i := 0; i < 100; i++ {
		cfg.Targets[i] = config.TargetConfig{
			Repo: "org/target-" + string(rune(48+i%10)),
			Files: []config.FileMapping{
				{Src: "file.txt", Dest: "file.txt"},
			},
		}
	}

	// Create current state
	currentState := &state.State{
		Source: state.SourceState{
			Repo:         "org/template-repo",
			Branch:       "main",
			LatestCommit: "abc123",
		},
		Targets: make(map[string]*state.TargetState),
	}

	for i := 0; i < 50; i++ {
		currentState.Targets["org/target-"+string(rune(48+i%10))] = &state.TargetState{
			Repo:           "org/target-" + string(rune(48+i%10)),
			LastSyncCommit: "xyz789",
			Status:         state.StatusBehind,
		}
	}

	engine := &Engine{
		config: cfg,
		options: &Options{
			Force:             false,
			MaxConcurrency:    5,
			UpdateExistingPRs: false,
		},
		logger: logrus.New(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		targets, err := engine.filterTargets(nil, currentState)
		if err != nil {
			b.Fatal(err)
		}
		_ = targets
	}
}

func BenchmarkFilterTargets_WithFilter(b *testing.B) {
	// Create test configuration
	cfg := &config.Config{
		Version: 1,
		Source: config.SourceConfig{
			Repo:   "org/template-repo",
			Branch: "main",
		},
		Targets: []config.TargetConfig{
			{Repo: "org/target-1", Files: []config.FileMapping{{Src: "file.txt", Dest: "file.txt"}}},
			{Repo: "org/target-2", Files: []config.FileMapping{{Src: "file.txt", Dest: "file.txt"}}},
			{Repo: "org/target-3", Files: []config.FileMapping{{Src: "file.txt", Dest: "file.txt"}}},
		},
	}

	currentState := &state.State{
		Source: state.SourceState{
			Repo:         "org/template-repo",
			Branch:       "main",
			LatestCommit: "abc123",
		},
		Targets: map[string]*state.TargetState{
			"org/target-1": {
				Repo:           "org/target-1",
				LastSyncCommit: "xyz789",
				Status:         state.StatusBehind,
			},
		},
	}

	engine := &Engine{
		config: cfg,
		options: &Options{
			Force:             false,
			MaxConcurrency:    5,
			UpdateExistingPRs: false,
		},
		logger: logrus.New(),
	}

	targetFilter := []string{"org/target-1", "org/target-2"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		targets, err := engine.filterTargets(targetFilter, currentState)
		if err != nil {
			b.Fatal(err)
		}
		_ = targets
	}
}

func BenchmarkNeedsSync(b *testing.B) {
	engine := &Engine{
		options: &Options{
			UpdateExistingPRs: false,
		},
		logger: logrus.New(),
	}

	target := config.TargetConfig{
		Repo: "org/target-repo",
		Files: []config.FileMapping{
			{Src: "file.txt", Dest: "file.txt"},
		},
	}

	testCases := []struct {
		name  string
		state *state.State
	}{
		{
			name: "UpToDate",
			state: &state.State{
				Source: state.SourceState{LatestCommit: "abc123"},
				Targets: map[string]*state.TargetState{
					"org/target-repo": {
						Status:         state.StatusUpToDate,
						LastSyncCommit: "abc123",
					},
				},
			},
		},
		{
			name: "Behind",
			state: &state.State{
				Source: state.SourceState{LatestCommit: "abc123"},
				Targets: map[string]*state.TargetState{
					"org/target-repo": {
						Status:         state.StatusBehind,
						LastSyncCommit: "xyz789",
					},
				},
			},
		},
		{
			name: "Pending",
			state: &state.State{
				Source: state.SourceState{LatestCommit: "abc123"},
				Targets: map[string]*state.TargetState{
					"org/target-repo": {
						Status:         state.StatusPending,
						LastSyncCommit: "xyz789",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			var result bool
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result = engine.needsSync(target, tc.state)
			}
			_ = result
		})
	}
}

func BenchmarkProgressTracking(b *testing.B) {
	testCases := []struct {
		name        string
		targetCount int
	}{
		{"Small", 5},
		{"Medium", 50},
		{"Large", 500},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				progress := NewProgressTracker(tc.targetCount, false)

				// Simulate progress updates
				for j := 0; j < tc.targetCount; j++ {
					repoName := "org/repo-" + string(rune(48+j%10))
					progress.StartRepository(repoName)

					if j%10 == 0 {
						progress.RecordError(repoName, context.DeadlineExceeded)
					} else {
						progress.RecordSuccess(repoName)
					}

					progress.FinishRepository(repoName)
				}

				results := progress.GetResults()
				_ = results
			}
		})
	}
}

func BenchmarkProgressConcurrent(b *testing.B) {
	const targetCount = 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		progress := NewProgressTracker(targetCount, false)

		// Simulate concurrent updates
		done := make(chan struct{})

		// Start multiple goroutines
		for j := 0; j < 10; j++ {
			go func(workerID int) {
				for k := 0; k < 10; k++ {
					repoName := "org/repo-" + string(rune(48+workerID)) + "-" + string(rune(48+k))
					progress.StartRepository(repoName)

					// Simulate work
					time.Sleep(time.Microsecond)

					if k%3 == 0 {
						progress.RecordError(repoName, context.DeadlineExceeded)
					} else {
						progress.RecordSuccess(repoName)
					}

					progress.FinishRepository(repoName)
				}
				done <- struct{}{}
			}(j)
		}

		// Wait for all workers
		for j := 0; j < 10; j++ {
			<-done
		}

		results := progress.GetResults()
		_ = results
	}
}

func BenchmarkOptionsValidation(b *testing.B) {
	testOptions := []*Options{
		DefaultOptions(),
		{
			DryRun:            true,
			Force:             false,
			MaxConcurrency:    1,
			UpdateExistingPRs: false,
		},
		{
			DryRun:            false,
			Force:             true,
			MaxConcurrency:    10,
			UpdateExistingPRs: true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, opts := range testOptions {
			// Validate options
			_ = opts.MaxConcurrency > 0
			_ = opts.MaxConcurrency <= 100

			// Apply defaults if needed
			if opts.MaxConcurrency <= 0 {
				opts.MaxConcurrency = 5
			}
		}
	}
}
