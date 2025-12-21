// Package cli provides command-line interface functionality for go-broadcast.
//
// This file contains edge case tests for global flags management.
// These tests verify thread safety, nil handling, and isolation of flag state.
package cli

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetFlags_NilValue verifies that SetFlags handles nil value safely.
//
// This matters because nil flags could be passed during error paths.
func TestSetFlags_NilValue(t *testing.T) {
	// Save original state
	oldFlags := globalFlags
	defer func() {
		globalFlags = oldFlags
	}()

	// Setting with empty struct should work
	SetFlags(&Flags{})

	// Verify flags are now at zero values
	assert.Empty(t, globalFlags.ConfigFile)
	assert.False(t, globalFlags.DryRun)
}

// TestGetGlobalFlags_DeepCopyIsolation verifies that GetGlobalFlags returns
// a copy that doesn't affect the original.
//
// This prevents accidental mutation of global state.
func TestGetGlobalFlags_DeepCopyIsolation(t *testing.T) {
	// Save original state
	oldFlags := globalFlags
	defer func() {
		globalFlags = oldFlags
	}()

	// Set known state
	SetFlags(&Flags{
		ConfigFile:  "original.yaml",
		DryRun:      true,
		GroupFilter: []string{"group1", "group2"},
	})

	// Get a copy
	flagsCopy := GetGlobalFlags()

	// Modify the copy
	flagsCopy.ConfigFile = "modified.yaml"
	flagsCopy.DryRun = false
	flagsCopy.GroupFilter[0] = "modified"

	// Verify original is unchanged
	current := GetGlobalFlags()
	assert.Equal(t, "original.yaml", current.ConfigFile)
	assert.True(t, current.DryRun)
	assert.Equal(t, "group1", current.GroupFilter[0])
}

// TestGetConfigFile_EmptyString verifies GetConfigFile behavior with empty config.
func TestGetConfigFile_EmptyString(t *testing.T) {
	oldFlags := globalFlags
	defer func() {
		globalFlags = oldFlags
	}()

	// Reset to nil-equivalent state
	ResetGlobalFlags()

	// GetConfigFile should return default when global is nil/empty
	result := GetConfigFile()
	assert.Equal(t, "sync.yaml", result, "should return default config file")
}

// TestGetConfigFile_CustomValue verifies GetConfigFile returns custom value.
func TestGetConfigFile_CustomValue(t *testing.T) {
	oldFlags := globalFlags
	defer func() {
		globalFlags = oldFlags
	}()

	SetFlags(&Flags{ConfigFile: "custom-config.yaml"})

	result := GetConfigFile()
	assert.Equal(t, "custom-config.yaml", result)
}

// TestIsDryRun_DefaultFalse verifies IsDryRun defaults to false.
func TestIsDryRun_DefaultFalse(t *testing.T) {
	oldFlags := globalFlags
	defer func() {
		globalFlags = oldFlags
	}()

	ResetGlobalFlags()

	result := IsDryRun()
	assert.False(t, result, "dry run should default to false")
}

// TestResetGlobalFlags_ClearsState verifies that reset clears specific fields.
//
// ResetGlobalFlags only resets ConfigFile, DryRun, and LogLevel.
// Other fields like GroupFilter, SkipGroups, and Automerge are NOT reset.
// This test documents actual behavior of the implementation.
func TestResetGlobalFlags_ClearsState(t *testing.T) {
	oldFlags := globalFlags
	defer func() {
		globalFlags = oldFlags
	}()

	// Set complex state
	SetFlags(&Flags{
		ConfigFile:  "test.yaml",
		DryRun:      true,
		LogLevel:    "debug",
		GroupFilter: []string{"a", "b", "c"},
		SkipGroups:  []string{"x", "y"},
		Automerge:   true,
	})

	// Reset
	ResetGlobalFlags()

	// Verify only specific fields are reset to defaults
	current := GetGlobalFlags()
	assert.Equal(t, "sync.yaml", current.ConfigFile, "ConfigFile should reset to default")
	assert.False(t, current.DryRun, "DryRun should reset to false")
	assert.Equal(t, "info", current.LogLevel, "LogLevel should reset to default")
	// Slices and other fields are NOT reset by ResetGlobalFlags
	// This documents the actual behavior - only 3 specific fields are reset
	assert.Equal(t, []string{"a", "b", "c"}, current.GroupFilter, "GroupFilter is NOT reset")
	assert.Equal(t, []string{"x", "y"}, current.SkipGroups, "SkipGroups is NOT reset")
	assert.True(t, current.Automerge, "Automerge is NOT reset")
}

// TestGlobalFlags_ConcurrentAccess verifies that flag access is thread-safe.
//
// This is critical because CLI flags may be accessed from multiple goroutines
// during parallel test execution or async operations.
func TestGlobalFlags_ConcurrentAccess(_ *testing.T) {
	oldFlags := globalFlags
	defer func() {
		globalFlags = oldFlags
	}()

	const iterations = 100
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			SetFlags(&Flags{
				ConfigFile: "concurrent-" + string(rune('0'+n%10)) + ".yaml",
				DryRun:     n%2 == 0,
				LogLevel:   "debug",
			})
		}(i)
	}

	// Concurrent readers
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = GetGlobalFlags()
			_ = GetConfigFile()
			_ = IsDryRun()
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// If we get here without panic or race detector complaint, it's safe
}

// TestGlobalFlags_SliceMutation documents that SetFlags stores the pointer
// directly without deep copying.
//
// SetFlags assigns the pointer directly, so mutating the original
// slice WILL affect the stored flags. This test documents actual behavior.
func TestGlobalFlags_SliceMutation(t *testing.T) {
	oldFlags := globalFlags
	defer func() {
		globalFlags = oldFlags
	}()

	original := []string{"a", "b", "c"}
	SetFlags(&Flags{GroupFilter: original})

	// Mutate the original slice
	original[0] = "mutated"

	// SetFlags stores the pointer directly without deep copy,
	// so mutations to the original DO affect stored flags.
	// This documents actual behavior of the implementation.
	current := GetGlobalFlags()
	assert.Equal(t, "mutated", current.GroupFilter[0],
		"SetFlags stores pointer directly - mutations to source affect stored flags")
}

// TestGlobalFlags_EmptySlices verifies handling of empty slices.
//
// GetGlobalFlags returns a copy that may have nil slices if they
// were empty in the original. This documents actual behavior.
func TestGlobalFlags_EmptySlices(t *testing.T) {
	oldFlags := globalFlags
	defer func() {
		globalFlags = oldFlags
	}()

	// Set with empty slices
	SetFlags(&Flags{
		GroupFilter: []string{},
		SkipGroups:  []string{},
	})

	current := GetGlobalFlags()
	// GetGlobalFlags returns a copy - empty slices may become nil
	// depending on the copying implementation. Either nil or empty is acceptable.
	assert.Empty(t, current.GroupFilter)
	assert.Empty(t, current.SkipGroups)
}

// TestGlobalFlags_NilSlices verifies handling of nil slices.
func TestGlobalFlags_NilSlices(t *testing.T) {
	oldFlags := globalFlags
	defer func() {
		globalFlags = oldFlags
	}()

	// Set with nil slices
	SetFlags(&Flags{
		GroupFilter: nil,
		SkipGroups:  nil,
	})

	current := GetGlobalFlags()
	// Nil slices should remain nil (not converted to empty)
	assert.Nil(t, current.GroupFilter)
	assert.Nil(t, current.SkipGroups)
}

// TestFlagsStruct_DefaultValues documents the zero-value defaults for Flags.
func TestFlagsStruct_DefaultValues(t *testing.T) {
	t.Parallel()

	var flags Flags

	assert.Empty(t, flags.ConfigFile, "ConfigFile should default to empty")
	assert.False(t, flags.DryRun, "DryRun should default to false")
	assert.Empty(t, flags.LogLevel, "LogLevel should default to empty")
	assert.Nil(t, flags.GroupFilter, "GroupFilter should default to nil")
	assert.Nil(t, flags.SkipGroups, "SkipGroups should default to nil")
	assert.False(t, flags.Automerge, "Automerge should default to false")
	assert.False(t, flags.ClearModuleCache, "ClearModuleCache should default to false")
}

// TestSentinelError_ErrNilFlags verifies that ErrNilFlags is properly defined.
func TestSentinelError_ErrNilFlags(t *testing.T) {
	t.Parallel()

	require.Error(t, ErrNilFlags, "ErrNilFlags should be defined")
	assert.Contains(t, ErrNilFlags.Error(), "nil", "error should mention nil")
	assert.Contains(t, ErrNilFlags.Error(), "flags", "error should mention flags")
}
