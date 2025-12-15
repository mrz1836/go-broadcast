package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RetryConfig configures retry behavior for AI API calls.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (default: 3).
	MaxAttempts int

	// InitialDelay is the initial delay between retries (default: 1s).
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries (default: 10s).
	MaxDelay time.Duration

	// Multiplier is the delay multiplier for exponential backoff (default: 2.0).
	Multiplier float64
}

// DefaultRetryConfig returns sensible defaults for retry behavior.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}
}

// RetryConfigFromConfig creates retry config from AI config.
func RetryConfigFromConfig(cfg *Config) *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  cfg.RetryMaxAttempts,
		InitialDelay: cfg.RetryInitialDelay,
		MaxDelay:     cfg.RetryMaxDelay,
		Multiplier:   2.0,
	}
}

// GenerateWithRetry wraps an AI generation call with retry logic.
// It uses exponential backoff for transient failures.
func GenerateWithRetry(
	ctx context.Context,
	cfg *RetryConfig,
	logger *logrus.Entry,
	generate func(context.Context) (*GenerateResponse, error),
) (*GenerateResponse, error) {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if attempt == 1 && logger != nil {
			logger.Info("Calling AI provider...")
		}
		resp, err := generate(ctx)
		if err == nil {
			if attempt > 1 && logger != nil {
				logger.WithField("attempt", attempt).Info("AI generation succeeded after retry")
			}
			return resp, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			if logger != nil {
				logger.WithError(err).Debug("Non-retryable AI error, failing immediately")
			}
			return nil, err
		}

		// Don't retry if context is done
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Don't wait after last attempt
		if attempt == cfg.MaxAttempts {
			break
		}

		if logger != nil {
			logger.WithFields(logrus.Fields{
				"attempt":     attempt,
				"maxAttempts": cfg.MaxAttempts,
				"delay":       delay.String(),
				"error":       err.Error(),
			}).Warn("AI generation failed, retrying")
		}

		// Wait with exponential backoff
		// Use NewTimer instead of time.After to avoid goroutine leak on context cancellation
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}

		// Increase delay for next attempt
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return nil, fmt.Errorf("AI generation failed after %d attempts: %w",
		cfg.MaxAttempts, lastErr)
}

// isRetryableError determines if an error warrants a retry.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Rate limits
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") {
		return true
	}

	// Server errors
	if strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "service unavailable") ||
		strings.Contains(errStr, "internal server error") {
		return true
	}

	// Timeouts
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") {
		return true
	}

	// Network errors
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "temporary") ||
		strings.Contains(errStr, "eof") {
		return true
	}

	// Overloaded
	if strings.Contains(errStr, "overloaded") ||
		strings.Contains(errStr, "capacity") {
		return true
	}

	return false
}

// IsRetryableError is exported for testing.
func IsRetryableError(err error) bool {
	return isRetryableError(err)
}
