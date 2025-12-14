package sync

import (
	"sync"
	"testing"
)

// TestGitHubAPIUpdateAverageTreeSizeRace tests concurrent updates to average tree size
// This specifically tests the spin lock that was replaced with a mutex
// Run with: go test -race -run TestGitHubAPIUpdateAverageTreeSizeRace
func TestGitHubAPIUpdateAverageTreeSizeRace(t *testing.T) {
	api := &GitHubAPI{
		stats: &APIStats{},
	}

	var wg sync.WaitGroup
	iterations := 1000

	// Multiple goroutines updating the average concurrently
	// This would cause contention in the old CAS loop implementation
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Vary the size to create realistic updates
				size := (goroutineID * 100) + j
				api.updateAverageTreeSize(size)
			}
		}(i)
	}

	wg.Wait()

	// Verify that we got some average
	avg := api.stats.AverageTreeSize.Load()
	if avg <= 0 {
		t.Errorf("Expected positive average, got %d", avg)
	}
}

// TestGitHubAPIUpdateAverageTreeSizeSequential verifies correctness
func TestGitHubAPIUpdateAverageTreeSizeSequential(t *testing.T) {
	api := &GitHubAPI{
		stats: &APIStats{},
	}

	// First update should set the average to the value
	api.updateAverageTreeSize(1000)
	if avg := api.stats.AverageTreeSize.Load(); avg != 1000 {
		t.Errorf("First update: expected 1000, got %d", avg)
	}

	// Second update should use weighted average: (1000*9 + 2000) / 10 = 1100
	api.updateAverageTreeSize(2000)
	expected := int64(1100)
	if avg := api.stats.AverageTreeSize.Load(); avg != expected {
		t.Errorf("Second update: expected %d, got %d", expected, avg)
	}

	// Third update: (1100*9 + 500) / 10 = 1040
	api.updateAverageTreeSize(500)
	expected = int64(1040)
	if avg := api.stats.AverageTreeSize.Load(); avg != expected {
		t.Errorf("Third update: expected %d, got %d", expected, avg)
	}
}

// TestGitHubAPIUpdateAverageTreeSizeHeavyContention tests under extreme contention
func TestGitHubAPIUpdateAverageTreeSizeHeavyContention(t *testing.T) {
	api := &GitHubAPI{
		stats: &APIStats{},
	}

	// Initialize with a base value
	api.stats.AverageTreeSize.Store(5000)

	var wg sync.WaitGroup
	goroutines := 100
	iterations := 500

	// Many goroutines all updating at once
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Use a consistent size per goroutine to make it easier to reason about
				size := 1000 + goroutineID
				api.updateAverageTreeSize(size)
			}
		}(i)
	}

	wg.Wait()

	// The final average should be within a reasonable range
	avg := api.stats.AverageTreeSize.Load()
	if avg < 1000 || avg > 10000 {
		t.Errorf("Average %d is outside expected range [1000, 10000]", avg)
	}
}

// TestGitHubAPIUpdateAverageTreeSizeStressTest is a stress test with varied operations
func TestGitHubAPIUpdateAverageTreeSizeStressTest(t *testing.T) {
	api := &GitHubAPI{
		stats: &APIStats{},
	}

	var wg sync.WaitGroup
	done := make(chan bool)

	// Writer goroutines
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			count := 0
			for {
				select {
				case <-done:
					return
				default:
					size := 1000 + (count % 5000)
					api.updateAverageTreeSize(size)
					count++
				}
			}
		}(i)
	}

	// Reader goroutines (reading the average)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					avg := api.stats.AverageTreeSize.Load()
					if avg < 0 {
						t.Errorf("Got negative average: %d", avg)
					}
				}
			}
		}()
	}

	// Let it run for a bit
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			api.updateAverageTreeSize(i * 10)
		}
		close(done)
	}()

	wg.Wait()
}

// TestGitHubAPIStatsGetStatsRace tests concurrent access to GetStats
func TestGitHubAPIStatsGetStatsRace(t *testing.T) {
	api := &GitHubAPI{
		stats: &APIStats{},
	}

	var wg sync.WaitGroup
	iterations := 500

	// Goroutines updating various stats
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				api.stats.TreeFetches.Add(1)
				api.stats.CacheHits.Add(1)
				api.stats.CacheMisses.Add(1)
				api.updateAverageTreeSize(1000 + j)
			}
		}(i)
	}

	// Goroutines reading stats
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				treeFetches, cacheHits, cacheMisses, retries, rateLimits, avgTreeSize := api.stats.GetStats()
				// Just access the values to ensure they're read
				_ = treeFetches
				_ = cacheHits
				_ = cacheMisses
				_ = retries
				_ = rateLimits
				if avgTreeSize < 0 {
					t.Errorf("Got negative avgTreeSize: %d", avgTreeSize)
				}
			}
		}()
	}

	wg.Wait()

	// Verify final stats are reasonable
	treeFetches, cacheHits, cacheMisses, _, _, avgTreeSize := api.stats.GetStats()
	expectedUpdates := int64(10 * iterations)
	if treeFetches != expectedUpdates {
		t.Errorf("Expected %d tree fetches, got %d", expectedUpdates, treeFetches)
	}
	if cacheHits != expectedUpdates {
		t.Errorf("Expected %d cache hits, got %d", expectedUpdates, cacheHits)
	}
	if cacheMisses != expectedUpdates {
		t.Errorf("Expected %d cache misses, got %d", expectedUpdates, cacheMisses)
	}
	if avgTreeSize <= 0 {
		t.Error("Expected positive avgTreeSize")
	}
}
