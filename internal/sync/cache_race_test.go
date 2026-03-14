package sync

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestContentCacheGetRace tests concurrent Get operations
// This specifically tests the lock upgrade race condition that was fixed
// Run with: go test -race -run TestContentCacheGetRace
func TestContentCacheGetRace(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(logrus.StandardLogger().Out)
	logger.SetLevel(logrus.ErrorLevel)

	cache := NewContentCache(1*time.Hour, 100*1024*1024, logger.WithField("test", "cache_race"))
	ctx := context.Background()

	// Pre-populate cache with some content
	testData := make(map[string]string)
	for i := 0; i < 10; i++ {
		repo := fmt.Sprintf("owner/repo%d", i)
		branch := "main"
		path := fmt.Sprintf("file%d.txt", i)
		content := fmt.Sprintf("Content for file %d", i)
		testData[fmt.Sprintf("%s:%s:%s", repo, branch, path)] = content

		err := cache.Put(ctx, repo, branch, path, content)
		if err != nil {
			t.Fatalf("Failed to populate cache: %v", err)
		}
	}

	var wg sync.WaitGroup
	iterations := 500

	// Multiple goroutines doing concurrent Gets
	// This exercises the lock upgrade path
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				repo := fmt.Sprintf("owner/repo%d", j%10)
				branch := "main"
				path := fmt.Sprintf("file%d.txt", j%10)

				content, hit, err := cache.Get(ctx, repo, branch, path)
				if err != nil {
					t.Errorf("Get failed: %v", err)
					return
				}

				if hit {
					expectedKey := fmt.Sprintf("%s:%s:%s", repo, branch, path)
					expectedContent := testData[expectedKey]
					if content != expectedContent {
						t.Errorf("Got wrong content: expected %q, got %q", expectedContent, content)
					}
				}
			}
		}(i)
	}

	// Goroutines doing concurrent Puts while Gets are happening
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterations/2; j++ {
				repo := fmt.Sprintf("owner/newrepo%d", goroutineID)
				branch := "main"
				path := fmt.Sprintf("newfile%d.txt", j)
				content := fmt.Sprintf("New content %d-%d", goroutineID, j)

				err := cache.Put(ctx, repo, branch, path, content)
				if err != nil {
					t.Errorf("Put failed: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify cache still works correctly after concurrent access
	stats := cache.GetStats()
	if stats.Hits == 0 {
		t.Error("Expected some cache hits, got 0")
	}
}

// TestContentCacheGetPutRace tests concurrent Get and Put operations
func TestContentCacheGetPutRace(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cache := NewContentCache(1*time.Hour, 100*1024*1024, logger.WithField("test", "cache_race"))
	ctx := context.Background()

	var wg sync.WaitGroup
	iterations := 200

	// Goroutines doing Gets
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("key%d", j%10)
				_, _, err := cache.Get(ctx, "owner/repo", "main", key)
				if err != nil {
					t.Errorf("Get failed: %v", err)
					return
				}
			}
		}(i)
	}

	// Goroutines doing Puts
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("key%d", j%10)
				content := fmt.Sprintf("content-%d-%d", goroutineID, j)
				err := cache.Put(ctx, "owner/repo", "main", key, content)
				if err != nil {
					t.Errorf("Put failed: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestContentCacheAccessTimeUpdate tests that access time updates don't race
// This specifically tests the lock upgrade pattern that could cause issues
func TestContentCacheAccessTimeUpdate(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cache := NewContentCache(1*time.Hour, 100*1024*1024, logger.WithField("test", "cache_race"))
	ctx := context.Background()

	// Put a single item
	err := cache.Put(ctx, "owner/repo", "main", "test.txt", "test content")
	if err != nil {
		t.Fatalf("Failed to put item: %v", err)
	}

	var wg sync.WaitGroup
	iterations := 1000

	// Many goroutines all accessing the same cached item
	// This forces concurrent access time updates
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				content, hit, err := cache.Get(ctx, "owner/repo", "main", "test.txt")
				if err != nil {
					t.Errorf("Get failed: %v", err)
					return
				}
				if !hit {
					t.Error("Expected cache hit, got miss")
					return
				}
				if content != "test content" {
					t.Errorf("Wrong content: %q", content)
					return
				}
			}
		}()
	}

	wg.Wait()

	stats := cache.GetStats()
	expectedHits := int64(50 * iterations)
	if stats.Hits != expectedHits {
		t.Errorf("Expected %d hits, got %d", expectedHits, stats.Hits)
	}
}

// TestContentCacheEvictionDuringGet tests eviction happening during Get operations
func TestContentCacheEvictionDuringGet(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Small cache that will trigger evictions
	cache := NewContentCache(100*time.Millisecond, 1024, logger.WithField("test", "cache_race"))
	ctx := context.Background()

	var wg sync.WaitGroup
	iterations := 100

	// Goroutines putting large items to trigger evictions
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("large%d-%d", goroutineID, j)
				content := fmt.Sprintf("%0100s", "x") // 100 bytes
				_ = cache.Put(ctx, "owner/repo", "main", key, content)
			}
		}(i)
	}

	// Goroutines getting items (some may be evicted)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("large%d-%d", goroutineID%5, j%iterations)
				_, _, err := cache.Get(ctx, "owner/repo", "main", key)
				if err != nil {
					t.Errorf("Get failed: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
}
