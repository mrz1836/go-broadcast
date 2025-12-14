package cli

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGlobalFlagsConcurrentAccess tests thread-safety of global flags access.
// This test verifies that the mutex protection prevents race conditions
// when multiple goroutines read and write global flags concurrently.
func TestGlobalFlagsConcurrentAccess(t *testing.T) {
	// Save original flags
	originalFlags := GetGlobalFlags()
	defer SetFlags(originalFlags)

	t.Run("concurrent reads are safe", func(_ *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		// Set initial flags
		SetFlags(&Flags{
			ConfigFile: "test.yaml",
			DryRun:     true,
			LogLevel:   "debug",
		})

		// Spawn multiple goroutines that read flags concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = GetConfigFile()
				_ = IsDryRun()
				_ = GetGlobalFlags()
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent reads and writes are safe", func(_ *testing.T) {
		var wg sync.WaitGroup
		numReaders := 50
		numWriters := 10

		// Start readers
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_ = GetConfigFile()
					_ = IsDryRun()
				}
			}()
		}

		// Start writers
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					SetFlags(&Flags{
						ConfigFile: "config" + string(rune('0'+id)) + ".yaml",
						DryRun:     id%2 == 0,
						LogLevel:   "info",
					})
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("GetGlobalFlags returns a copy", func(t *testing.T) {
		SetFlags(&Flags{
			ConfigFile:  "original.yaml",
			DryRun:      true,
			LogLevel:    "info",
			GroupFilter: []string{"group1", "group2"},
		})

		// Get a copy
		flagsCopy := GetGlobalFlags()

		// Modify the copy
		flagsCopy.ConfigFile = "modified.yaml"
		flagsCopy.DryRun = false
		flagsCopy.GroupFilter[0] = "modified"

		// Original should be unchanged
		assert.Equal(t, "original.yaml", GetConfigFile())
		assert.True(t, IsDryRun())

		// Get another copy and verify slices are independent
		anotherCopy := GetGlobalFlags()
		assert.Equal(t, "group1", anotherCopy.GroupFilter[0])
	})

	t.Run("ResetGlobalFlags is thread-safe", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 50

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ResetGlobalFlags()
				_ = GetConfigFile()
				_ = IsDryRun()
			}()
		}

		wg.Wait()

		// After reset, should have default values
		assert.Equal(t, "sync.yaml", GetConfigFile())
		assert.False(t, IsDryRun())
	})
}
