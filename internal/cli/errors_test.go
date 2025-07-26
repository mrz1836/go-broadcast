package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCliErrorsDefinition tests that CLI errors are properly defined
func TestCliErrorsDefinition(t *testing.T) {
	t.Run("ErrConfigFileNotFound", func(t *testing.T) {
		require.Error(t, ErrConfigFileNotFound)
		assert.Equal(t, "configuration file not found", ErrConfigFileNotFound.Error())
	})

	t.Run("ErrNoMatchingTargets", func(t *testing.T) {
		require.Error(t, ErrNoMatchingTargets)
		assert.Equal(t, "no matching targets found", ErrNoMatchingTargets.Error())
	})
}

// TestCliErrorsAreDistinct tests that errors are distinct
func TestCliErrorsAreDistinct(t *testing.T) {
	assert.NotEqual(t, ErrConfigFileNotFound, ErrNoMatchingTargets)

	// Ensure they are not equal to generic errors
	// Ensure they are not equal to generic errors with same message
	genericErr1 := errors.New("configuration file not found") //nolint:err113 // test error
	genericErr2 := errors.New("no matching targets found")    //nolint:err113 // test error
	require.NotErrorIs(t, ErrConfigFileNotFound, genericErr1)
	require.NotErrorIs(t, ErrNoMatchingTargets, genericErr2)
}

// TestCliErrorsUsageWithErrorsIs tests using errors.Is with CLI errors
func TestCliErrorsUsageWithErrorsIs(t *testing.T) {
	// Test wrapping and unwrapping
	additionalContext := errors.New("additional context") //nolint:err113 // test error
	wrappedErr := errors.Join(ErrConfigFileNotFound, additionalContext)

	require.ErrorIs(t, wrappedErr, ErrConfigFileNotFound)
	assert.NotErrorIs(t, wrappedErr, ErrNoMatchingTargets)
}

// TestCliErrorsInErrorChains tests CLI errors in error chains
func TestCliErrorsInErrorChains(t *testing.T) {
	// Create error chain
	err1 := ErrConfigFileNotFound
	err2 := errors.New("while processing sync") //nolint:err113 // test error
	combinedErr := errors.Join(err1, err2)

	assert.Contains(t, combinedErr.Error(), "configuration file not found")
	assert.Contains(t, combinedErr.Error(), "while processing sync")
	assert.ErrorIs(t, combinedErr, ErrConfigFileNotFound)
}

// TestCliErrorsSentinelBehavior tests that errors behave as sentinels
func TestCliErrorsSentinelBehavior(t *testing.T) {
	// Sentinel errors should be comparable
	err1 := ErrConfigFileNotFound
	err2 := ErrConfigFileNotFound

	assert.Equal(t, err1, err2)
	assert.ErrorIs(t, err1, err2)
}

// TestCliErrorsImmutability tests that error messages cannot be changed
func TestCliErrorsImmutability(t *testing.T) {
	// Get error message
	msg1 := ErrConfigFileNotFound.Error()
	msg2 := ErrNoMatchingTargets.Error()

	// Messages should remain consistent
	assert.Equal(t, "configuration file not found", msg1)
	assert.Equal(t, "no matching targets found", msg2)

	// Multiple calls should return the same message
	assert.Equal(t, msg1, ErrConfigFileNotFound.Error())
	assert.Equal(t, msg2, ErrNoMatchingTargets.Error())
}
