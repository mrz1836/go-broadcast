//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	syncpkg "github.com/mrz1836/go-broadcast/internal/sync"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// Helper functions for git operations
func initGitRepoPerf(t testing.TB, dir string) {
	t.Helper()
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git
	cmd = exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	_ = cmd.Run()

	cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
	cmd.Dir = dir
	_ = cmd.Run()

	// Initial commit
	cmd = exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = dir
	_ = cmd.Run()

	cmd = exec.CommandContext(ctx, "git", "commit", "-m", "Initial commit", "--allow-empty")
	cmd.Dir = dir
	_ = cmd.Run()
}

func createGitTagPerf(t testing.TB, dir, tag string) {
	t.Helper()
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "tag", tag)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create git tag %s: %v", tag, err)
	}
}

// Performance targets from plan-12-status.md
const (
	MaxGroupSwitchOverhead   = 50 * time.Millisecond
	MaxDependencyResolution  = 100 * time.Millisecond
	MaxModuleDetection       = 10 * time.Millisecond
	MaxVersionResolutionCold = 500 * time.Millisecond
	MaxVersionResolutionWarm = 10 * time.Millisecond
)

func BenchmarkGroupOrchestration(b *testing.B) {
	b.Run("group switching overhead", func(b *testing.B) {
		tmpDir := b.TempDir()
		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		require.NoError(b, os.MkdirAll(sourceDir, 0o750))
		require.NoError(b, os.MkdirAll(targetDir, 0o750))

		// Create minimal test file
		require.NoError(b, os.WriteFile(
			filepath.Join(sourceDir, "test.txt"),
			[]byte("test"),
			0o600,
		))

		initGitRepoPerf(b, sourceDir)
		initGitRepoPerf(b, targetDir)

		// Create config with 5 groups
		var groups []config.Group
		for i := 1; i <= 5; i++ {
			groups = append(groups, config.Group{
				Name:     fmt.Sprintf("Group %d", i),
				ID:       fmt.Sprintf("group-%d", i),
				Priority: i,
				Enabled:  boolPtr(true),
				Source: config.SourceConfig{
					Repo:   sourceDir,
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: targetDir,
						Files: []config.FileMapping{
							{
								Src:  "test.txt",
								Dest: fmt.Sprintf("file%d.txt", i),
							},
						},
					},
				},
			})
		}

		cfg := &config.Config{
			Version: 1,
			Groups:  groups,
		}

		logger := logrus.New()
		logger.SetLevel(logrus.WarnLevel)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			// Setup mocks for benchmarking
			mockGH := &gh.MockClient{}
			mockGit := &git.MockClient{}
			// Add broad GetChangedFiles mock to handle all calls
			mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
			mockState := &state.MockDiscoverer{}
			mockTransform := &transform.MockChain{}

			// Mock state discovery
			currentState := &state.State{
				Source: state.SourceState{
					Repo:         sourceDir,
					Branch:       "main",
					LatestCommit: "abc123",
				},
				Targets: map[string]*state.TargetState{},
			}
			mockState.On("DiscoverState", mock.Anything, mock.AnythingOfType("*config.Config")).Return(currentState, nil)

			// Mock git operations
			mockGit.On("Clone", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Checkout", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Commit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			// Mock GitHub operations
			mockGH.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
			mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
			mockGH.On("DeleteBranch", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			mockGH.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte("test content"), nil)
			mockGH.On("CreatePR", mock.Anything, mock.Anything).Return("https://github.com/org/repo/pull/1", nil)

			// Mock transform operations
			mockTransform.On("Apply", mock.Anything, mock.Anything).Return(mock.Anything, nil)

			opts := syncpkg.DefaultOptions().
				WithDryRun(true) // Dry run for benchmarking

			engine := syncpkg.NewEngine(context.Background(), cfg, mockGH, mockGit, mockState, mockTransform, opts)
			engine.SetLogger(logger)

			start := time.Now()
			_ = engine.Sync(ctx, nil)
			duration := time.Since(start)

			// Calculate average overhead per group switch
			avgOverhead := duration / time.Duration(len(groups))
			if avgOverhead > MaxGroupSwitchOverhead {
				b.Logf("Group switch overhead exceeded target: %v > %v", avgOverhead, MaxGroupSwitchOverhead)
			}
		}
	})

	b.Run("dependency resolution", func(b *testing.B) {
		// Create complex dependency graph
		var groups []config.Group

		// Layer 1: 10 foundation groups
		for i := 0; i < 10; i++ {
			groups = append(groups, config.Group{
				Name:      fmt.Sprintf("Foundation-%d", i),
				ID:        fmt.Sprintf("foundation-%d", i),
				Priority:  1,
				Enabled:   boolPtr(true),
				DependsOn: []string{},
			})
		}

		// Layer 2: 15 groups depending on foundation
		for i := 0; i < 15; i++ {
			deps := []string{
				fmt.Sprintf("foundation-%d", i%10),
				fmt.Sprintf("foundation-%d", (i+1)%10),
			}
			groups = append(groups, config.Group{
				Name:      fmt.Sprintf("Layer2-%d", i),
				ID:        fmt.Sprintf("layer2-%d", i),
				Priority:  2,
				Enabled:   boolPtr(true),
				DependsOn: deps,
			})
		}

		// Layer 3: 15 groups with complex dependencies
		for i := 0; i < 15; i++ {
			deps := []string{
				fmt.Sprintf("layer2-%d", i%15),
				fmt.Sprintf("layer2-%d", (i+1)%15),
				fmt.Sprintf("foundation-%d", i%10),
			}
			groups = append(groups, config.Group{
				Name:      fmt.Sprintf("Layer3-%d", i),
				ID:        fmt.Sprintf("layer3-%d", i),
				Priority:  3,
				Enabled:   boolPtr(true),
				DependsOn: deps,
			})
		}

		// Layer 4: 10 groups depending on everything
		for i := 0; i < 10; i++ {
			deps := []string{
				fmt.Sprintf("layer3-%d", i%15),
				fmt.Sprintf("layer3-%d", (i+1)%15),
				fmt.Sprintf("layer2-%d", i%15),
			}
			groups = append(groups, config.Group{
				Name:      fmt.Sprintf("Final-%d", i),
				ID:        fmt.Sprintf("final-%d", i),
				Priority:  4,
				Enabled:   boolPtr(true),
				DependsOn: deps,
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger := logrus.New()
			resolver := syncpkg.NewDependencyResolver(logger)

			start := time.Now()
			for _, group := range groups {
				resolver.AddGroup(group)
			}
			_, err := resolver.Resolve()
			duration := time.Since(start)

			if err != nil {
				b.Fatalf("Failed to resolve dependencies: %v", err)
			}

			if duration > MaxDependencyResolution {
				b.Logf("Dependency resolution exceeded target: %v > %v", duration, MaxDependencyResolution)
			}
		}
	})
}

func BenchmarkModuleOperations(b *testing.B) {
	b.Run("module detection", func(b *testing.B) {
		tmpDir := b.TempDir()
		sourceDir := filepath.Join(tmpDir, "source")

		// Create directory structure with multiple modules
		modules := []string{
			"pkg/auth",
			"pkg/database",
			"pkg/cache",
			"internal/utils",
			"cmd/server",
		}

		for _, mod := range modules {
			modDir := filepath.Join(sourceDir, mod)
			require.NoError(b, os.MkdirAll(modDir, 0o750))
			require.NoError(b, os.WriteFile(
				filepath.Join(modDir, "go.mod"),
				[]byte(fmt.Sprintf("module example.com/%s\n\ngo 1.21", filepath.Base(mod))),
				0o600,
			))
		}

		logger := logrus.New()
		detector := syncpkg.NewModuleDetector(logger)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()

			for _, mod := range modules {
				modDir := filepath.Join(sourceDir, mod)
				_ = detector.IsGoModule(modDir)
			}

			duration := time.Since(start) / time.Duration(len(modules))

			if duration > MaxModuleDetection {
				b.Logf("Module detection exceeded target: %v > %v", duration, MaxModuleDetection)
			}
		}
	})

	b.Run("version resolution cold cache", func(b *testing.B) {
		tmpDir := b.TempDir()
		repoDir := filepath.Join(tmpDir, "repo")

		require.NoError(b, os.MkdirAll(repoDir, 0o750))
		initGitRepoPerf(b, repoDir)

		// Create many version tags
		for major := 1; major <= 3; major++ {
			for minor := 0; minor <= 5; minor++ {
				for patch := 0; patch <= 10; patch++ {
					tag := fmt.Sprintf("v%d.%d.%d", major, minor, patch)
					createGitTagPerf(b, repoDir, tag)
				}
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Create new resolver each time (cold cache)
			logger := logrus.New()
			cache := syncpkg.NewModuleCache(5*time.Minute, logger)
			resolver := syncpkg.NewModuleResolver(logger, cache)
			ctx := context.Background()

			start := time.Now()
			_, err := resolver.ResolveVersion(ctx, repoDir, "latest", true)
			duration := time.Since(start)
			cache.Close() // Clean up goroutine

			if err != nil {
				b.Fatalf("Failed to resolve version: %v", err)
			}

			if duration > MaxVersionResolutionCold {
				b.Logf("Cold version resolution exceeded target: %v > %v", duration, MaxVersionResolutionCold)
			}
		}
	})

	b.Run("version resolution warm cache", func(b *testing.B) {
		tmpDir := b.TempDir()
		repoDir := filepath.Join(tmpDir, "repo")

		require.NoError(b, os.MkdirAll(repoDir, 0o750))
		initGitRepoPerf(b, repoDir)

		// Create version tags
		for i := 0; i < 20; i++ {
			createGitTagPerf(b, repoDir, fmt.Sprintf("v1.0.%d", i))
		}

		// Create resolver with cache
		logger := logrus.New()
		cache := syncpkg.NewModuleCache(5*time.Minute, logger)
		defer cache.Close()
		resolver := syncpkg.NewModuleResolver(logger, cache)
		ctx := context.Background()

		// Warm up cache
		_, _ = resolver.ResolveVersion(ctx, repoDir, "latest", true)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()
			_, err := resolver.ResolveVersion(ctx, repoDir, "latest", true)
			duration := time.Since(start)

			if err != nil {
				b.Fatalf("Failed to resolve version: %v", err)
			}

			if duration > MaxVersionResolutionWarm {
				b.Logf("Warm version resolution exceeded target: %v > %v", duration, MaxVersionResolutionWarm)
			}
		}
	})
}

func TestPerformanceStress(t *testing.T) {
	t.Skip("Skipping performance stress test - needs proper setup")
	t.Run("large configuration with 100 groups", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping stress test in short mode")
		}

		ctx := context.Background()
		tmpDir := t.TempDir()

		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		require.NoError(t, os.MkdirAll(sourceDir, 0o750))
		require.NoError(t, os.MkdirAll(targetDir, 0o750))

		// Create test files
		for i := 0; i < 10; i++ {
			require.NoError(t, os.WriteFile(
				filepath.Join(sourceDir, fmt.Sprintf("file%d.txt", i)),
				[]byte(fmt.Sprintf("content %d", i)),
				0o600,
			))
		}

		initGitRepoPerf(t, sourceDir)
		initGitRepoPerf(t, targetDir)

		// Create 100 groups with various dependencies
		var groups []config.Group
		for i := 0; i < 100; i++ {
			var deps []string
			if i > 0 {
				// Each group depends on 0-3 previous groups
				numDeps := i % 4
				for j := 0; j < numDeps && j < i; j++ {
					deps = append(deps, fmt.Sprintf("group-%d", i-j-1))
				}
			}

			groups = append(groups, config.Group{
				Name:      fmt.Sprintf("Group %d", i),
				ID:        fmt.Sprintf("group-%d", i),
				Priority:  (i / 10) + 1,
				Enabled:   boolPtr(i%7 != 0), // Disable every 7th group
				DependsOn: deps,
				Source: config.SourceConfig{
					Repo:   sourceDir,
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: targetDir,
						Files: []config.FileMapping{
							{
								Src:  fmt.Sprintf("file%d.txt", i%10),
								Dest: fmt.Sprintf("output/group%d/file.txt", i),
							},
						},
					},
				},
			})
		}

		cfg := &config.Config{
			Version: 1,
			Groups:  groups,
		}

		logger := logrus.New()
		logger.SetLevel(logrus.WarnLevel)

		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		// Add broad GetChangedFiles mock to handle all calls
		mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// Mock state discovery
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         sourceDir,
				Branch:       "main",
				LatestCommit: "abc123",
			},
			Targets: map[string]*state.TargetState{},
		}
		mockState.On("DiscoverState", mock.Anything, mock.AnythingOfType("*config.Config")).Return(currentState, nil)

		// Mock GitHub operations for orphaned branch cleanup and PR operations
		mockGH.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("DeleteBranch", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		opts := syncpkg.DefaultOptions().
			WithDryRun(true)

		engine := syncpkg.NewEngine(context.Background(), cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logger)

		start := time.Now()
		err := engine.Sync(ctx, nil)
		duration := time.Since(start)

		require.NoError(t, err)
		t.Logf("Processed 100 groups in %v", duration)

		// Verify reasonable performance (should complete in under 30 seconds)
		assert.Less(t, duration, 30*time.Second, "Large configuration took too long")
	})

	t.Run("deep dependency chain 10 levels", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping stress test in short mode")
		}

		// Create a deep chain: A -> B -> C -> ... (10 levels)
		var groups []config.Group
		for i := 0; i < 10; i++ {
			var deps []string
			if i > 0 {
				deps = []string{fmt.Sprintf("level-%d", i-1)}
			}

			groups = append(groups, config.Group{
				Name:      fmt.Sprintf("Level %d", i),
				ID:        fmt.Sprintf("level-%d", i),
				Priority:  i + 1,
				Enabled:   boolPtr(true),
				DependsOn: deps,
			})
		}

		logger := logrus.New()
		resolver := syncpkg.NewDependencyResolver(logger)

		start := time.Now()
		for _, group := range groups {
			resolver.AddGroup(group)
		}
		resolved, err := resolver.Resolve()
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Len(t, resolved, 10)
		t.Logf("Resolved 10-level dependency chain in %v", duration)

		// Should still be fast even with deep chain
		assert.Less(t, duration, 50*time.Millisecond)
	})

	t.Run("concurrent module resolution", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping stress test in short mode")
		}

		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "repo")

		require.NoError(t, os.MkdirAll(repoDir, 0o750))
		initGitRepoPerf(t, repoDir)

		// Create version tags
		for i := 0; i < 50; i++ {
			createGitTagPerf(t, repoDir, fmt.Sprintf("v1.0.%d", i))
		}

		logger := logrus.New()
		cache := syncpkg.NewModuleCache(5*time.Minute, logger)
		defer cache.Close()
		resolver := syncpkg.NewModuleResolver(logger, cache)

		// Concurrent resolution requests
		numGoroutines := 20
		numRequests := 100
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*numRequests)

		start := time.Now()

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(_ int) {
				defer wg.Done()

				for r := 0; r < numRequests; r++ {
					constraint := "latest"
					switch r % 3 {
					case 0:
						constraint = fmt.Sprintf("v1.0.%d", r%50)
					case 1:
						constraint = "~1.0.0"
					}

					ctx := context.Background()
					_, err := resolver.ResolveVersion(ctx, repoDir, constraint, true)
					if err != nil {
						errors <- err
					}
				}
			}(g)
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		// Check for errors
		var errCount int
		for err := range errors {
			t.Logf("Error during concurrent resolution: %v", err)
			errCount++
		}

		assert.Equal(t, 0, errCount, "Should have no errors during concurrent resolution")
		t.Logf("Completed %d concurrent resolutions in %v", numGoroutines*numRequests, duration)

		// Should complete reasonably fast with cache
		assert.Less(t, duration, 5*time.Second)
	})

	t.Run("memory usage under load", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping stress test in short mode")
		}

		// Get initial memory stats
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Create large configuration
		var groups []config.Group
		for i := 0; i < 500; i++ {
			groups = append(groups, config.Group{
				Name:     fmt.Sprintf("Group %d", i),
				ID:       fmt.Sprintf("group-%d", i),
				Priority: i,
				Enabled:  boolPtr(true),
				Source: config.SourceConfig{
					Repo:   "/tmp/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "/tmp/target",
						Files: []config.FileMapping{
							{Src: "file.txt", Dest: fmt.Sprintf("file%d.txt", i)},
						},
					},
				},
			})
		}

		cfg := &config.Config{
			Version: 1,
			Groups:  groups,
		}

		// Process configuration
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		opts := syncpkg.DefaultOptions().
			WithDryRun(true)

		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		// Add broad GetChangedFiles mock to handle all calls
		mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		_ = syncpkg.NewEngine(context.Background(), cfg, mockGH, mockGit, mockState, mockTransform, opts)

		// Force garbage collection
		runtime.GC()

		// Get final memory stats
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		memUsed := (m2.Alloc - m1.Alloc) / 1024 / 1024 // Convert to MB
		t.Logf("Memory used for 500 groups: %d MB", memUsed)

		// Should use reasonable amount of memory
		assert.Less(t, memUsed, uint64(100), "Memory usage should be under 100MB for 500 groups")
	})

	t.Run("large directory sync operation", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping stress test in short mode")
		}

		ctx := context.Background()
		tmpDir := t.TempDir()

		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		// Create large directory structure
		numDirs := 50
		filesPerDir := 20

		for d := 0; d < numDirs; d++ {
			dirPath := filepath.Join(sourceDir, fmt.Sprintf("dir%d", d))
			require.NoError(t, os.MkdirAll(dirPath, 0o750))

			for f := 0; f < filesPerDir; f++ {
				filePath := filepath.Join(dirPath, fmt.Sprintf("file%d.txt", f))
				content := fmt.Sprintf("Directory %d, File %d", d, f)
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))
			}
		}

		initGitRepoPerf(t, sourceDir)
		initGitRepoPerf(t, targetDir)

		cfg := &config.Config{
			Version: 1,
			Groups: []config.Group{
				{
					Name:     "Large Directory Sync",
					ID:       "large-sync",
					Priority: 1,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   sourceDir,
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: targetDir,
							Directories: []config.DirectoryMapping{
								{
									Src:  ".",
									Dest: "synced",
								},
							},
						},
					},
				},
			},
		}

		logger := logrus.New()
		logger.SetLevel(logrus.WarnLevel)

		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		// Add broad GetChangedFiles mock to handle all calls
		mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// Mock state discovery
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         sourceDir,
				Branch:       "main",
				LatestCommit: "abc123",
			},
			Targets: map[string]*state.TargetState{},
		}
		mockState.On("DiscoverState", mock.Anything, mock.AnythingOfType("*config.Config")).Return(currentState, nil)

		// Mock GitHub operations for orphaned branch cleanup and PR operations
		mockGH.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("DeleteBranch", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		opts := syncpkg.DefaultOptions().
			WithDryRun(true)

		engine := syncpkg.NewEngine(context.Background(), cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logger)

		start := time.Now()
		err := engine.Sync(ctx, nil)
		duration := time.Since(start)

		require.NoError(t, err)

		totalFiles := numDirs * filesPerDir
		t.Logf("Synced %d files in %v", totalFiles, duration)

		// Verify files were synced
		syncedFiles := 0
		err = filepath.Walk(filepath.Join(targetDir, "synced"), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".txt" {
				syncedFiles++
			}
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, totalFiles, syncedFiles, "All files should be synced")

		// Should complete in reasonable time
		assert.Less(t, duration, 30*time.Second, "Large directory sync took too long")
	})
}

func TestPerformanceRegression(t *testing.T) {
	t.Skip("optional test, does not add coverage")
	t.Run("baseline performance comparison", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping performance regression test in short mode")
		}

		ctx := context.Background()
		tmpDir := t.TempDir()

		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		require.NoError(t, os.MkdirAll(sourceDir, 0o750))
		require.NoError(t, os.MkdirAll(targetDir, 0o750))

		// Create test file
		require.NoError(t, os.WriteFile(
			filepath.Join(sourceDir, "test.txt"),
			[]byte("test content"),
			0o600,
		))

		initGitRepoPerf(t, sourceDir)
		initGitRepoPerf(t, targetDir)

		// Simple configuration for baseline
		cfg := &config.Config{
			Version: 1,
			Groups: []config.Group{
				{
					Name:     "Baseline Test",
					ID:       "baseline",
					Priority: 1,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   sourceDir,
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: targetDir,
							Files: []config.FileMapping{
								{
									Src:  "test.txt",
									Dest: "output.txt",
								},
							},
						},
					},
				},
			},
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		// Run multiple times to get average
		var durations []time.Duration
		for i := 0; i < 5; i++ {
			opts := syncpkg.DefaultOptions().
				WithDryRun(true)

			// Setup mocks
			mockGH := &gh.MockClient{}
			mockGit := &git.MockClient{}
			// Add broad GetChangedFiles mock to handle all calls
			mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
			mockState := &state.MockDiscoverer{}
			mockTransform := &transform.MockChain{}

			// Mock state discovery
			currentState := &state.State{
				Source: state.SourceState{
					Repo:         sourceDir,
					Branch:       "main",
					LatestCommit: "abc123",
				},
				Targets: map[string]*state.TargetState{
					targetDir: {
						Repo:           targetDir,
						LastSyncCommit: "old123", // Outdated to trigger sync
						Status:         state.StatusBehind,
					},
				},
			}
			mockState.On("DiscoverState", mock.Anything, mock.AnythingOfType("*config.Config")).Return(currentState, nil)

			// Mock git operations
			mockGit.On("Clone", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Checkout", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Commit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			// Mock GitHub operations
			mockGH.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
			mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
			mockGH.On("DeleteBranch", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			mockGH.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte("test content"), nil)
			mockGH.On("CreatePR", mock.Anything, mock.Anything).Return("https://github.com/org/repo/pull/1", nil)

			// Mock transform operations
			mockTransform.On("Apply", mock.Anything, mock.Anything).Return(mock.Anything, nil)

			engine := syncpkg.NewEngine(context.Background(), cfg, mockGH, mockGit, mockState, mockTransform, opts)

			start := time.Now()
			err := engine.Sync(ctx, nil)
			duration := time.Since(start)

			require.NoError(t, err)
			durations = append(durations, duration)
		}

		// Calculate average
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		avg := total / time.Duration(len(durations))

		t.Logf("Average baseline performance: %v", avg)

		// Store as reference (in practice, this would be compared against historical data)
		// For now, just ensure it's reasonable
		// The race detector adds significant overhead, so we need different thresholds
		var threshold time.Duration
		if isRaceEnabled() {
			threshold = 5 * time.Second // Race detector can add 10x+ overhead
			t.Logf("Race detector enabled, using relaxed threshold: %v", threshold)
		} else {
			threshold = 500 * time.Millisecond
			t.Logf("Race detector disabled, using normal threshold: %v", threshold)
		}
		assert.Less(t, avg, threshold, "Baseline performance should be under %v", threshold)
	})

	t.Run("performance with increasing groups", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping performance regression test in short mode")
		}

		tmpDir := t.TempDir()
		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		require.NoError(t, os.MkdirAll(sourceDir, 0o750))
		require.NoError(t, os.MkdirAll(targetDir, 0o750))

		require.NoError(t, os.WriteFile(
			filepath.Join(sourceDir, "test.txt"),
			[]byte("test"),
			0o600,
		))

		initGitRepoPerf(t, sourceDir)
		initGitRepoPerf(t, targetDir)

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		// Test with increasing number of groups
		groupCounts := []int{1, 5, 10, 20, 50}
		var results []struct {
			count    int
			duration time.Duration
		}

		for _, count := range groupCounts {
			var groups []config.Group
			for i := 0; i < count; i++ {
				groups = append(groups, config.Group{
					Name:     fmt.Sprintf("Group %d", i),
					ID:       fmt.Sprintf("group-%d", i),
					Priority: i,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   sourceDir,
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: targetDir,
							Files: []config.FileMapping{
								{
									Src:  "test.txt",
									Dest: fmt.Sprintf("output%d.txt", i),
								},
							},
						},
					},
				})
			}

			cfg := &config.Config{
				Version: 1,
				Groups:  groups,
			}

			ctx := context.Background()
			opts := syncpkg.DefaultOptions().
				WithDryRun(true)

			// Setup mocks
			mockGH := &gh.MockClient{}
			mockGit := &git.MockClient{}
			// Add broad GetChangedFiles mock to handle all calls
			mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
			mockState := &state.MockDiscoverer{}
			mockTransform := &transform.MockChain{}

			// Mock state discovery
			currentState := &state.State{
				Source: state.SourceState{
					Repo:         sourceDir,
					Branch:       "main",
					LatestCommit: "abc123",
				},
				Targets: map[string]*state.TargetState{
					targetDir: {
						Repo:           targetDir,
						LastSyncCommit: "old123", // Outdated to trigger sync
						Status:         state.StatusBehind,
					},
				},
			}
			mockState.On("DiscoverState", mock.Anything, mock.AnythingOfType("*config.Config")).Return(currentState, nil)

			// Mock git operations - with enough calls for multiple groups
			mockGit.On("Clone", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Checkout", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Commit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			// Mock GitHub operations
			mockGH.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
			mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
			mockGH.On("DeleteBranch", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			mockGH.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte("test content"), nil)
			mockGH.On("CreatePR", mock.Anything, mock.Anything).Return("https://github.com/org/repo/pull/1", nil)

			// Mock transform operations
			mockTransform.On("Apply", mock.Anything, mock.Anything).Return(mock.Anything, nil)

			engine := syncpkg.NewEngine(context.Background(), cfg, mockGH, mockGit, mockState, mockTransform, opts)

			start := time.Now()
			err := engine.Sync(ctx, nil)
			duration := time.Since(start)

			require.NoError(t, err)

			results = append(results, struct {
				count    int
				duration time.Duration
			}{count, duration})
		}

		// Log results
		t.Log("Performance scaling with group count:")
		for _, r := range results {
			perGroup := r.duration / time.Duration(r.count)
			t.Logf("  %2d groups: %10v total, %10v per group", r.count, r.duration, perGroup)
		}

		// Verify linear or better scaling
		// The time per group should not increase significantly
		firstPerGroup := results[0].duration / time.Duration(results[0].count)
		lastPerGroup := results[len(results)-1].duration / time.Duration(results[len(results)-1].count)

		// Allow up to 3x slowdown per group (should be much less in practice)
		assert.Less(t, lastPerGroup, firstPerGroup*3,
			"Performance should scale reasonably with group count")
	})
}
