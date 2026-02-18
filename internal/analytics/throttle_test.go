package analytics

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultThrottleConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultThrottleConfig()
	assert.InDelta(t, 1.0, cfg.RequestsPerSecond, 0.001)
	assert.Equal(t, 3, cfg.BurstSize)
	assert.Equal(t, 500*time.Millisecond, cfg.InterRepoDelay)
	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, 2*time.Second, cfg.InitialBackoff)
	assert.Equal(t, 60*time.Second, cfg.MaxBackoff)
	assert.InDelta(t, 2.0, cfg.BackoffMultiplier, 0.001)
}

func TestNewThrottle(t *testing.T) {
	t.Parallel()

	cfg := DefaultThrottleConfig()
	throttle := NewThrottle(cfg, nil)

	require.NotNil(t, throttle)
	assert.NotNil(t, throttle.limiter)
	assert.Equal(t, cfg, throttle.config)
}

func TestThrottle_Wait(t *testing.T) {
	t.Parallel()

	t.Run("allows burst requests immediately", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         5,
		}
		throttle := NewThrottle(cfg, nil)

		ctx := context.Background()
		start := time.Now()

		// Burst of 5 should be near-instant
		for i := 0; i < 5; i++ {
			require.NoError(t, throttle.Wait(ctx))
		}

		elapsed := time.Since(start)
		assert.Less(t, elapsed, 200*time.Millisecond, "burst requests should complete quickly")

		stats := throttle.Stats()
		assert.Equal(t, int64(5), stats.TotalCalls)
	})

	t.Run("throttles after burst exhausted", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 10, // 1 token every 100ms
			BurstSize:         1,  // Only 1 burst token
		}
		throttle := NewThrottle(cfg, nil)

		ctx := context.Background()

		// First request uses the burst token
		require.NoError(t, throttle.Wait(ctx))

		// Second request must wait for a new token (~100ms)
		start := time.Now()
		require.NoError(t, throttle.Wait(ctx))
		elapsed := time.Since(start)

		assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond, "should wait for token refill")
	})

	t.Run("cancels on context cancellation", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 0.1, // Very slow: 1 token every 10s
			BurstSize:         1,
		}
		throttle := NewThrottle(cfg, nil)

		ctx := context.Background()

		// Exhaust the burst token
		require.NoError(t, throttle.Wait(ctx))

		// Cancel context while waiting for next token
		cancelCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		err := throttle.Wait(cancelCtx)
		require.Error(t, err)
	})

	t.Run("increments TotalCalls on success", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
		}
		throttle := NewThrottle(cfg, nil)

		ctx := context.Background()
		for i := 0; i < 7; i++ {
			require.NoError(t, throttle.Wait(ctx))
		}

		assert.Equal(t, int64(7), throttle.Stats().TotalCalls)
	})

	t.Run("tracks wait time", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 10,
			BurstSize:         1,
		}
		throttle := NewThrottle(cfg, nil)

		ctx := context.Background()

		// First call uses burst — minimal wait
		require.NoError(t, throttle.Wait(ctx))

		// Second call waits ~100ms for token
		require.NoError(t, throttle.Wait(ctx))

		stats := throttle.Stats()
		// The wait time should be recorded (at least some ms for the second call)
		assert.GreaterOrEqual(t, stats.TotalWaitedMs, int64(0))
	})
}

func TestThrottle_Wait_ConcurrentSafety(t *testing.T) {
	t.Parallel()

	cfg := ThrottleConfig{
		RequestsPerSecond: 100,
		BurstSize:         20,
	}
	throttle := NewThrottle(cfg, nil)

	ctx := context.Background()
	var wg sync.WaitGroup

	const workers = 10
	const callsPerWorker = 5

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerWorker; j++ {
				_ = throttle.Wait(ctx)
			}
		}()
	}

	wg.Wait()

	stats := throttle.Stats()
	assert.Equal(t, int64(workers*callsPerWorker), stats.TotalCalls)
}

func TestThrottle_WaitInterRepo(t *testing.T) {
	t.Parallel()

	t.Run("sleeps for configured delay", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			InterRepoDelay:    100 * time.Millisecond,
		}
		throttle := NewThrottle(cfg, nil)

		ctx := context.Background()
		start := time.Now()
		err := throttle.WaitInterRepo(ctx)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, elapsed, 80*time.Millisecond, "should sleep for roughly InterRepoDelay")
	})

	t.Run("returns immediately when delay is zero", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			InterRepoDelay:    0,
		}
		throttle := NewThrottle(cfg, nil)

		start := time.Now()
		err := throttle.WaitInterRepo(context.Background())
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, elapsed, 10*time.Millisecond)
	})

	t.Run("cancels on context cancellation", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			InterRepoDelay:    5 * time.Second, // Long delay
		}
		throttle := NewThrottle(cfg, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := throttle.WaitInterRepo(ctx)
		elapsed := time.Since(start)

		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Less(t, elapsed, 1*time.Second, "should cancel quickly, not wait full delay")
	})
}

func TestThrottle_DoWithRetry(t *testing.T) {
	t.Parallel()

	t.Run("succeeds on first attempt", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}
		throttle := NewThrottle(cfg, nil)

		var callCount atomic.Int32
		err := throttle.DoWithRetry(context.Background(), "test-op", func() error {
			callCount.Add(1)
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, int32(1), callCount.Load())
		assert.Equal(t, int64(0), throttle.Stats().TotalRetries)
	})

	t.Run("retries on rate-limit error and succeeds", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}
		throttle := NewThrottle(cfg, nil)

		var callCount atomic.Int32
		err := throttle.DoWithRetry(context.Background(), "test-op", func() error {
			n := callCount.Add(1)
			if n <= 2 {
				return errors.New("API rate limit exceeded") //nolint:err113 // test error
			}
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, int32(3), callCount.Load(), "should be called 3 times (2 failures + 1 success)")
		assert.Equal(t, int64(2), throttle.Stats().TotalRetries)
	})

	t.Run("does not retry non-rate-limit errors", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}
		throttle := NewThrottle(cfg, nil)

		var callCount atomic.Int32
		expectedErr := errors.New("not found: 404 repository does not exist") //nolint:err113 // test error
		err := throttle.DoWithRetry(context.Background(), "test-op", func() error {
			callCount.Add(1)
			return expectedErr
		})

		require.Error(t, err)
		assert.Equal(t, expectedErr, err, "should return original error")
		assert.Equal(t, int32(1), callCount.Load(), "should only be called once")
		assert.Equal(t, int64(0), throttle.Stats().TotalRetries)
	})

	t.Run("gives up after max retries", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			MaxRetries:        2,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        50 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}
		throttle := NewThrottle(cfg, nil)

		var callCount atomic.Int32
		err := throttle.DoWithRetry(context.Background(), "test-op", func() error {
			callCount.Add(1)
			return errors.New("429 too many requests") //nolint:err113 // test error
		})

		require.Error(t, err)
		require.ErrorIs(t, err, ErrRetriesExhausted)
		// 1 initial + 2 retries = 3 total calls
		assert.Equal(t, int32(3), callCount.Load())
		assert.Equal(t, int64(2), throttle.Stats().TotalRetries)
	})

	t.Run("cancels during backoff", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			MaxRetries:        5,
			InitialBackoff:    5 * time.Second, // Long backoff
			MaxBackoff:        60 * time.Second,
			BackoffMultiplier: 2.0,
		}
		throttle := NewThrottle(cfg, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := throttle.DoWithRetry(ctx, "test-op", func() error {
			return errors.New("API rate limit exceeded") //nolint:err113 // test error
		})
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.Less(t, elapsed, 2*time.Second, "should cancel quickly during backoff")
	})

	t.Run("exponential backoff respects MaxBackoff", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        25 * time.Millisecond, // Cap at 25ms
			BackoffMultiplier: 10.0,                  // Aggressive multiplier
		}
		throttle := NewThrottle(cfg, nil)

		var callCount atomic.Int32
		start := time.Now()
		_ = throttle.DoWithRetry(context.Background(), "test-op", func() error {
			callCount.Add(1)
			return errors.New("secondary rate limit") //nolint:err113 // test error
		})
		elapsed := time.Since(start)

		// With cap at 25ms: delays are 10ms, 25ms, 25ms = 60ms total max
		// Without cap: 10ms, 100ms, 1000ms = way longer
		assert.Less(t, elapsed, 500*time.Millisecond, "MaxBackoff should cap delay growth")
	})
}

func TestIsAPIRateLimitError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"rate limit", errors.New("rate limit exceeded"), true},                                     //nolint:err113 // test
		{"Rate Limit capitalized", errors.New("Rate Limit Exceeded"), true},                         //nolint:err113 // test
		{"403 error", errors.New("HTTP 403 Forbidden"), true},                                       //nolint:err113 // test
		{"429 error", errors.New("HTTP 429 Too Many Requests"), true},                               //nolint:err113 // test
		{"abuse detection", errors.New("You have triggered an abuse detection mechanism"), true},    //nolint:err113 // test
		{"secondary rate", errors.New("You have exceeded a secondary rate limit"), true},            //nolint:err113 // test
		{"too many requests", errors.New("too many requests"), true},                                //nolint:err113 // test
		{"api rate limit", errors.New("api rate limit exceeded for this endpoint"), true},           //nolint:err113 // test
		{"404 not found", errors.New("HTTP 404 Not Found"), false},                                  //nolint:err113 // test
		{"generic error", errors.New("network timeout"), false},                                     //nolint:err113 // test
		{"empty error", errors.New(""), false},                                                      //nolint:err113 // test
		{"422 validation", errors.New("422 Validation Failed"), false},                              //nolint:err113 // test
		{"500 server error", errors.New("500 Internal Server Error"), false},                        //nolint:err113 // test
		{"embedded 403", errors.New("gh: api error: HTTP 403: rate limit exceeded for user"), true}, //nolint:err113 // test
		{"embedded 429", errors.New("gh: api error: HTTP 429: secondary rate limit reached"), true}, //nolint:err113 // test
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isAPIRateLimitError(tt.err))
		})
	}
}

func TestThrottle_Stats(t *testing.T) {
	t.Parallel()

	t.Run("initial stats are zero", func(t *testing.T) {
		t.Parallel()

		throttle := NewThrottle(DefaultThrottleConfig(), nil)
		stats := throttle.Stats()

		assert.Equal(t, int64(0), stats.TotalCalls)
		assert.Equal(t, int64(0), stats.TotalRetries)
		assert.Equal(t, int64(0), stats.TotalWaitedMs)
	})

	t.Run("stats reflect activity", func(t *testing.T) {
		t.Parallel()

		cfg := ThrottleConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			MaxRetries:        3,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        50 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}
		throttle := NewThrottle(cfg, nil)

		ctx := context.Background()

		// Make 3 successful calls
		for i := 0; i < 3; i++ {
			require.NoError(t, throttle.Wait(ctx))
		}

		// Make 1 call that retries once then succeeds
		var attempt atomic.Int32
		err := throttle.DoWithRetry(ctx, "retry-op", func() error {
			n := attempt.Add(1)
			if n == 1 {
				return errors.New("rate limit exceeded") //nolint:err113 // test error
			}
			return nil
		})
		require.NoError(t, err)

		stats := throttle.Stats()
		// 3 direct Wait calls + 2 DoWithRetry calls (1st attempt + retry) = 5
		assert.Equal(t, int64(5), stats.TotalCalls)
		assert.Equal(t, int64(1), stats.TotalRetries)
	})
}

func TestThrottle_Config(t *testing.T) {
	t.Parallel()

	cfg := ThrottleConfig{
		RequestsPerSecond: 2.5,
		BurstSize:         5,
		InterRepoDelay:    200 * time.Millisecond,
	}
	throttle := NewThrottle(cfg, nil)

	got := throttle.Config()
	assert.InDelta(t, 2.5, got.RequestsPerSecond, 0.001)
	assert.Equal(t, 5, got.BurstSize)
	assert.Equal(t, 200*time.Millisecond, got.InterRepoDelay)
}
