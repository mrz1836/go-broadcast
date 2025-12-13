package sync

import (
	"sync"
	"testing"
	"time"
)

// TestPerformanceMetricsDirectoryMetricsRace tests concurrent access to DirectoryMetrics map
// Run with: go test -race -run TestPerformanceMetricsDirectoryMetricsRace
func TestPerformanceMetricsDirectoryMetricsRace(t *testing.T) {
	pm := &PerformanceMetrics{
		StartTime:        time.Now(),
		DirectoryMetrics: make(map[string]DirectoryMetrics),
	}

	dirPaths := []string{
		"src/dir1",
		"src/dir2",
		"src/dir3",
		"src/dir4",
		"src/dir5",
	}

	var wg sync.WaitGroup
	iterations := 100

	// Goroutines that write to DirectoryMetrics
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				dirPath := dirPaths[goroutineID%len(dirPaths)]
				metrics := DirectoryMetrics{
					FilesProcessed: j,
					FilesChanged:   j / 2,
					FilesExcluded:  j / 3,
					StartTime:      time.Now(),
					EndTime:        time.Now().Add(time.Millisecond),
				}
				pm.SetDirectoryMetric(dirPath, metrics)
			}
		}(i)
	}

	// Goroutines that read from DirectoryMetrics
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				dirPath := dirPaths[goroutineID%len(dirPaths)]
				if metrics, exists := pm.GetDirectoryMetric(dirPath); exists {
					_ = metrics.FilesProcessed
					_ = metrics.FilesChanged
				}
			}
		}(i)
	}

	// Goroutines that iterate over DirectoryMetrics
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations/2; j++ {
				count := 0
				pm.IterateDirectoryMetrics(func(_ string, metrics DirectoryMetrics) {
					count++
					_ = metrics.FilesProcessed
				})
			}
		}()
	}

	// Goroutines that both read and write
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				dirPath := dirPaths[goroutineID%len(dirPaths)]
				if j%2 == 0 {
					// Read-modify-write pattern
					if metrics, exists := pm.GetDirectoryMetric(dirPath); exists {
						metrics.FilesChanged++
						pm.SetDirectoryMetric(dirPath, metrics)
					}
				} else {
					// Just read
					_, _ = pm.GetDirectoryMetric(dirPath)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all directories have some data
	pm.IterateDirectoryMetrics(func(dirPath string, metrics DirectoryMetrics) {
		if metrics.FilesProcessed < 0 {
			t.Errorf("DirectoryMetrics for %s has negative FilesProcessed: %d", dirPath, metrics.FilesProcessed)
		}
	})
}

// TestPerformanceMetricsReadModifyWrite simulates the actual usage pattern
// from updateDirectoryMetricsWithActualChanges
func TestPerformanceMetricsReadModifyWrite(t *testing.T) {
	pm := &PerformanceMetrics{
		StartTime:        time.Now(),
		DirectoryMetrics: make(map[string]DirectoryMetrics),
	}

	// Initialize some directories
	dirPaths := []string{"dir1", "dir2", "dir3"}
	for _, dir := range dirPaths {
		pm.SetDirectoryMetric(dir, DirectoryMetrics{
			FilesProcessed: 100,
			FilesChanged:   0,
		})
	}

	var wg sync.WaitGroup
	iterations := 1000

	// Simulate multiple goroutines doing read-modify-write operations
	// This mimics the pattern in updateDirectoryMetricsWithActualChanges
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				dirPath := dirPaths[goroutineID%len(dirPaths)]

				// Read
				if metrics, exists := pm.GetDirectoryMetric(dirPath); exists {
					// Modify
					metrics.FilesChanged++
					// Write
					pm.SetDirectoryMetric(dirPath, metrics)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify counts - each directory should have been updated by multiple goroutines
	totalChanges := 0
	pm.IterateDirectoryMetrics(func(dirPath string, metrics DirectoryMetrics) {
		totalChanges += metrics.FilesChanged
		if metrics.FilesChanged == 0 {
			t.Errorf("DirectoryMetrics for %s has zero FilesChanged, expected some updates", dirPath)
		}
	})

	// Total should be less than or equal to goroutines * iterations
	// (due to potential race conditions in read-modify-write, but our mutex should prevent lost updates)
	maxExpected := 10 * iterations
	if totalChanges > maxExpected {
		t.Errorf("Total changes %d exceeds maximum expected %d", totalChanges, maxExpected)
	}
}

// TestPerformanceMetricsIterateWhileModifying tests iteration while modifications happen
func TestPerformanceMetricsIterateWhileModifying(_ *testing.T) {
	pm := &PerformanceMetrics{
		StartTime:        time.Now(),
		DirectoryMetrics: make(map[string]DirectoryMetrics),
	}

	// Pre-populate with some directories
	for i := 0; i < 5; i++ {
		pm.SetDirectoryMetric(string(rune('A'+i)), DirectoryMetrics{
			FilesProcessed: i * 10,
		})
	}

	var readerWg sync.WaitGroup
	var writerWg sync.WaitGroup
	done := make(chan bool)

	// Writer goroutine
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		for {
			select {
			case <-done:
				return
			default:
				for i := 0; i < 10; i++ {
					pm.SetDirectoryMetric(string(rune('A'+i)), DirectoryMetrics{
						FilesProcessed: i * 10,
						FilesChanged:   i * 5,
					})
				}
			}
		}
	}()

	// Reader/iterator goroutines
	for i := 0; i < 5; i++ {
		readerWg.Add(1)
		go func() {
			defer readerWg.Done()
			for j := 0; j < 100; j++ {
				count := 0
				pm.IterateDirectoryMetrics(func(_ string, metrics DirectoryMetrics) {
					count++
					// Access fields to ensure they're read
					_ = metrics.FilesProcessed
					_ = metrics.FilesChanged
				})
			}
		}()
	}

	// Wait for readers to finish first, then signal writer to stop
	readerWg.Wait()
	close(done)

	// Wait for writer to finish
	writerWg.Wait()
}
