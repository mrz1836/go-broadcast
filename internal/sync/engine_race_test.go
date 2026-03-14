package sync

import (
	"sync"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestEngineCurrentGroupRace tests concurrent access to currentGroup field
// Run with: go test -race -run TestEngineCurrentGroupRace
func TestEngineCurrentGroupRace(t *testing.T) {
	// Create a minimal engine for testing
	engine := &Engine{
		config: &config.Config{},
	}

	// Create test groups
	groups := []config.Group{
		{ID: "group1", Name: "Group 1"},
		{ID: "group2", Name: "Group 2"},
		{ID: "group3", Name: "Group 3"},
	}

	// Use WaitGroup to ensure all goroutines complete
	var wg sync.WaitGroup
	iterations := 100

	// Start multiple goroutines that write to currentGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(groupIdx int) {
			defer wg.Done()
			group := groups[groupIdx%len(groups)]
			for j := 0; j < iterations; j++ {
				engine.SetCurrentGroup(&group)
			}
		}(i)
	}

	// Start multiple goroutines that read from currentGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				g := engine.GetCurrentGroup()
				if g != nil {
					_ = g.ID // Access field to ensure read happens
				}
			}
		}()
	}

	// Start goroutines that both read and write
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(groupIdx int) {
			defer wg.Done()
			group := groups[groupIdx%len(groups)]
			for j := 0; j < iterations; j++ {
				if j%2 == 0 {
					engine.SetCurrentGroup(&group)
				} else {
					g := engine.GetCurrentGroup()
					if g != nil {
						_ = g.Name
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is valid (not nil or one of our test groups)
	final := engine.GetCurrentGroup()
	if final != nil {
		found := false
		for _, g := range groups {
			if final.ID == g.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Final currentGroup has unexpected ID: %s", final.ID)
		}
	}
}

// TestEngineCurrentGroupNilSafety tests that setting and getting nil is safe
func TestEngineCurrentGroupNilSafety(_ *testing.T) {
	engine := &Engine{
		config: &config.Config{},
	}

	group := &config.Group{ID: "test", Name: "Test"}

	var wg sync.WaitGroup
	iterations := 100

	// Goroutines that alternate between nil and non-nil
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if j%2 == 0 {
					engine.SetCurrentGroup(nil)
				} else {
					engine.SetCurrentGroup(group)
				}
			}
		}()
	}

	// Goroutines that read
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				g := engine.GetCurrentGroup()
				// Should never panic, even if nil
				if g != nil {
					_ = g.ID
				}
			}
		}()
	}

	wg.Wait()
}
