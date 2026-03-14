package gh

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTransient = errors.New("transient error")

func TestRateLimitedDo_SuccessfulFunction(t *testing.T) {
	t.Parallel()

	called := false
	err := rateLimitedDo(context.Background(), 1*time.Millisecond, func() error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called, "function should have been called")
}

func TestRateLimitedDo_SuccessfulWithZeroDelay(t *testing.T) {
	t.Parallel()

	called := false
	err := rateLimitedDo(context.Background(), 0, func() error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called, "function should have been called with zero delay")
}

func TestRateLimitedDo_FailThenSucceed(t *testing.T) {
	t.Parallel()

	var attempts int32
	err := rateLimitedDo(context.Background(), 1*time.Millisecond, func() error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			return errTransient
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts), "should have retried once before succeeding")
}

func TestRateLimitedDo_FailTwiceThenSucceed(t *testing.T) {
	t.Parallel()

	var attempts int32
	err := rateLimitedDo(context.Background(), 1*time.Millisecond, func() error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return errTransient
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts), "should have retried twice before succeeding on attempt 3")
}

func TestRateLimitedDo_ContextCancelledDuringPreDelay(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	called := false
	err := rateLimitedDo(ctx, 1*time.Hour, func() error {
		called = true
		return nil
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, called, "function should not have been called when context is canceled before delay")
}

func TestRateLimitedDo_ContextCancelledDuringRetryBackoff(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	var attempts int32
	err := rateLimitedDo(ctx, 1*time.Millisecond, func() error {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			// After first failure, cancel context so retry backoff sees cancellation
			cancel()
			return errTransient
		}
		return nil
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts), "should have attempted once before context was canceled")
}

func TestRateLimitedDo_ContextCancelledDuringFnExecution(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	var attempts int32
	err := rateLimitedDo(ctx, 1*time.Millisecond, func() error {
		atomic.AddInt32(&attempts, 1)
		cancel()
		return errTransient
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

func TestRateLimitedDo_AllRetriesExhausted(t *testing.T) {
	t.Parallel()

	var attempts int32

	err := rateLimitedDo(context.Background(), 1*time.Millisecond, func() error {
		atomic.AddInt32(&attempts, 1)
		return errTransient
	})

	require.Error(t, err)
	require.ErrorIs(t, err, errTransient)
	assert.Contains(t, err.Error(), "after 3 attempts")
	assert.Equal(t, int32(maxRetries), atomic.LoadInt32(&attempts), "should have exhausted all retry attempts")
}

func TestRateLimitedDo_AllRetriesExhaustedWrapsLastError(t *testing.T) {
	t.Parallel()

	var attempts int32
	err := rateLimitedDo(context.Background(), 1*time.Millisecond, func() error {
		atomic.AddInt32(&attempts, 1)
		return errTransient
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "after 3 attempts")
	require.ErrorIs(t, err, errTransient)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestRateLimitedDo_RespectsPreCallDelay(t *testing.T) {
	t.Parallel()

	delay := 50 * time.Millisecond
	start := time.Now()

	err := rateLimitedDo(context.Background(), delay, func() error {
		return nil
	})

	elapsed := time.Since(start)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed, delay, "should wait at least the specified delay before calling fn")
}
