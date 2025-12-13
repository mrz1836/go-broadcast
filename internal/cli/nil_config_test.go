package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
)

// TestNilConfigErrorHandling verifies that all functions that previously panicked
// on nil config now return ErrNilConfig instead.
func TestNilConfigErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("performCancel returns ErrNilConfig", func(t *testing.T) {
		summary, err := performCancel(ctx, nil, []string{})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNilConfig)
		assert.Nil(t, summary)
	})

	t.Run("performCancelWithClient returns ErrNilConfig", func(t *testing.T) {
		mockClient := new(gh.MockClient)
		logConfig := &logging.LogConfig{LogLevel: "error"}

		summary, err := performCancelWithClient(ctx, nil, []string{}, mockClient, nil, logConfig)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNilConfig)
		assert.Nil(t, summary)
	})

	t.Run("performCancelWithDiscoverer returns ErrNilConfig", func(t *testing.T) {
		mockClient := new(gh.MockClient)

		summary, err := performCancelWithDiscoverer(ctx, nil, []string{}, mockClient, nil)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNilConfig)
		assert.Nil(t, summary)
	})

	t.Run("validateRepositoryAccessibility returns ErrNilConfig", func(t *testing.T) {
		logConfig := &logging.LogConfig{LogLevel: "error"}

		err := validateRepositoryAccessibility(ctx, nil, logConfig, false)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNilConfig)
	})

	t.Run("validateRepositoryAccessibilityWithClient returns ErrNilConfig", func(t *testing.T) {
		mockClient := new(gh.MockClient)

		err := validateRepositoryAccessibilityWithClient(ctx, nil, mockClient, false)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNilConfig)
	})
}

// TestErrNilConfigMessage verifies the error message is clear
func TestErrNilConfigMessage(t *testing.T) {
	assert.Equal(t, "config cannot be nil", ErrNilConfig.Error())
}
