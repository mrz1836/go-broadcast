package analytics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// ErrRetriesExhausted is returned when all retry attempts fail due to rate limiting
var ErrRetriesExhausted = errors.New("rate-limit retries exhausted")

// ThrottleConfig configures the rate limiter and retry behavior
type ThrottleConfig struct {
	RequestsPerSecond float64       // Token refill rate (default: 1.0)
	BurstSize         int           // Max tokens available at once (default: 3)
	InterRepoDelay    time.Duration // Pause between repos (default: 500ms)
	MaxRetries        int           // Max retry attempts on rate-limit errors (default: 5)
	InitialBackoff    time.Duration // First retry delay (default: 2s)
	MaxBackoff        time.Duration // Ceiling for exponential backoff (default: 60s)
	BackoffMultiplier float64       // Backoff growth factor (default: 2.0)
}

// DefaultThrottleConfig returns conservative defaults for GitHub API rate limiting
func DefaultThrottleConfig() ThrottleConfig {
	return ThrottleConfig{
		RequestsPerSecond: 1.0,
		BurstSize:         3,
		InterRepoDelay:    500 * time.Millisecond,
		MaxRetries:        5,
		InitialBackoff:    2 * time.Second,
		MaxBackoff:        60 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// ThrottleStats holds counters for throttle activity
type ThrottleStats struct {
	TotalCalls    int64 // Total API calls made through the throttle
	TotalRetries  int64 // Total retry attempts due to rate-limit errors
	TotalWaitedMs int64 // Total milliseconds spent waiting for tokens
}

// Throttle governs the rate of GitHub API calls across all goroutines
type Throttle struct {
	limiter *rate.Limiter
	config  ThrottleConfig
	logger  *logrus.Logger

	totalCalls    atomic.Int64
	totalRetries  atomic.Int64
	totalWaitedMs atomic.Int64
}

// NewThrottle creates a new Throttle with the given configuration
func NewThrottle(cfg ThrottleConfig, logger *logrus.Logger) *Throttle {
	return &Throttle{
		limiter: rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.BurstSize),
		config:  cfg,
		logger:  logger,
	}
}

// Wait blocks until a rate-limit token is available or ctx is canceled
func (t *Throttle) Wait(ctx context.Context) error {
	start := time.Now()
	err := t.limiter.Wait(ctx)
	waited := time.Since(start).Milliseconds()
	if waited > 0 {
		t.totalWaitedMs.Add(waited)
	}
	if err == nil {
		t.totalCalls.Add(1)
	}
	return err
}

// WaitInterRepo sleeps for InterRepoDelay between repos, respecting context cancellation
func (t *Throttle) WaitInterRepo(ctx context.Context) error {
	if t.config.InterRepoDelay <= 0 {
		return nil
	}

	timer := time.NewTimer(t.config.InterRepoDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// DoWithRetry executes fn with rate limiting and exponential backoff on rate-limit errors.
// It calls Wait(ctx) before each attempt. Non-rate-limit errors are returned immediately.
func (t *Throttle) DoWithRetry(ctx context.Context, operation string, fn func() error) error {
	backoff := t.config.InitialBackoff

	for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
		// Wait for a rate-limit token
		if err := t.Wait(ctx); err != nil {
			return fmt.Errorf("throttle wait for %s: %w", operation, err)
		}

		// Execute the operation
		err := fn()
		if err == nil {
			return nil
		}

		// If it's not a rate-limit error, return immediately
		if !isAPIRateLimitError(err) {
			return err
		}

		// If we've exhausted retries, return the error
		if attempt == t.config.MaxRetries {
			return fmt.Errorf("%s after %d retries (%w): %w", operation, t.config.MaxRetries, ErrRetriesExhausted, err)
		}

		t.totalRetries.Add(1)

		if t.logger != nil {
			t.logger.WithFields(logrus.Fields{
				"operation": operation,
				"attempt":   attempt + 1,
				"backoff":   backoff.String(),
			}).Warn("Rate-limited, backing off")
		}

		// Exponential backoff with context cancellation
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		// Grow backoff for next attempt
		backoff = time.Duration(float64(backoff) * t.config.BackoffMultiplier)
		if backoff > t.config.MaxBackoff {
			backoff = t.config.MaxBackoff
		}
	}

	// Should not be reached, but just in case
	return fmt.Errorf("%s: %w", operation, ErrRetriesExhausted)
}

// isAPIRateLimitError checks if an error indicates a GitHub API rate limit
func isAPIRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "abuse") ||
		strings.Contains(errStr, "secondary rate") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "api rate limit exceeded") ||
		// HTTP status codes often present in error strings
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "429")
}

// Stats returns a snapshot of throttle activity counters
func (t *Throttle) Stats() ThrottleStats {
	return ThrottleStats{
		TotalCalls:    t.totalCalls.Load(),
		TotalRetries:  t.totalRetries.Load(),
		TotalWaitedMs: t.totalWaitedMs.Load(),
	}
}

// Config returns the throttle configuration
func (t *Throttle) Config() ThrottleConfig {
	return t.config
}
