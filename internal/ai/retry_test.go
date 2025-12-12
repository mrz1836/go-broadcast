package ai

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test errors - defined at package level per linting rules.
var (
	errRateLimitExceeded   = errors.New("rate limit exceeded")
	errRateLimit           = errors.New("rate limit")
	errTimeout             = errors.New("timeout")
	errInvalidAPIKey       = errors.New("invalid API key")
	errHTTP429             = errors.New("HTTP 429: too many requests")
	errTooManyRequests     = errors.New("too many requests")
	errHTTP500             = errors.New("HTTP 500: internal server error")
	errHTTP502             = errors.New("HTTP 502: bad gateway")
	errHTTP503             = errors.New("HTTP 503: service unavailable")
	errServiceUnavailable  = errors.New("service unavailable")
	errInternalServerError = errors.New("internal server error")
	errRequestTimeout      = errors.New("request timeout")
	errDeadlineExceeded    = errors.New("context deadline exceeded")
	errConnectionRefused   = errors.New("connection refused")
	errNetworkUnreachable  = errors.New("network is unreachable")
	errTemporaryFailure    = errors.New("temporary failure in name resolution")
	errUnexpectedEOF       = errors.New("unexpected EOF")
	errServerOverloaded    = errors.New("server overloaded")
	errAtCapacity          = errors.New("at capacity")
	errHTTP400             = errors.New("HTTP 400: bad request")
	errHTTP404             = errors.New("HTTP 404: not found")
	errInvalidPrompt       = errors.New("prompt contains invalid content")
	errUnknown             = errors.New("something went wrong")

	// Case-insensitive test errors - intentionally capitalized to test case-insensitivity.
	errRateLimitUpper          = errors.New("RATE LIMIT")
	errRateLimitMixed          = errors.New("Rate Limit") //nolint:staticcheck // testing case-insensitivity
	errTimeoutUpper            = errors.New("TIMEOUT")
	errTimeoutMixed            = errors.New("Timeout")
	errServiceUnavailableUpper = errors.New("SERVICE UNAVAILABLE")
	errServiceUnavailableMixed = errors.New("Service Unavailable") //nolint:staticcheck // testing case-insensitivity
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, 3, cfg.MaxAttempts)
	assert.Equal(t, 1*time.Second, cfg.InitialDelay)
	assert.Equal(t, 10*time.Second, cfg.MaxDelay)
	assert.InEpsilon(t, 2.0, cfg.Multiplier, 0.001)
}

func TestRetryConfigFromConfig(t *testing.T) {
	aiCfg := &Config{
		RetryMaxAttempts:  5,
		RetryInitialDelay: 500 * time.Millisecond,
		RetryMaxDelay:     30 * time.Second,
	}

	cfg := RetryConfigFromConfig(aiCfg)

	require.NotNil(t, cfg)
	assert.Equal(t, 5, cfg.MaxAttempts)
	assert.Equal(t, 500*time.Millisecond, cfg.InitialDelay)
	assert.Equal(t, 30*time.Second, cfg.MaxDelay)
	assert.InEpsilon(t, 2.0, cfg.Multiplier, 0.001)
}

func TestGenerateWithRetry_SuccessFirstAttempt(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	var callCount int32
	generator := func(_ context.Context) (*GenerateResponse, error) {
		atomic.AddInt32(&callCount, 1)
		return &GenerateResponse{Content: "success"}, nil
	}

	resp, err := GenerateWithRetry(ctx, cfg, logger, generator)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "success", resp.Content)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount), "should only call once on success")
}

func TestGenerateWithRetry_SuccessAfterRetry(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	var callCount int32
	generator := func(_ context.Context) (*GenerateResponse, error) {
		count := atomic.AddInt32(&callCount, 1)
		if count < 2 {
			return nil, errRateLimitExceeded // retryable
		}
		return &GenerateResponse{Content: "success"}, nil
	}

	resp, err := GenerateWithRetry(ctx, cfg, logger, generator)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "success", resp.Content)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount), "should retry once before success")
}

func TestGenerateWithRetry_MaxAttemptsExhausted(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 5 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
		Multiplier:   2.0,
	}
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	var callCount int32
	generator := func(_ context.Context) (*GenerateResponse, error) {
		atomic.AddInt32(&callCount, 1)
		return nil, errTimeout // retryable
	}

	resp, err := GenerateWithRetry(ctx, cfg, logger, generator)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed after 3 attempts")
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount), "should try MaxAttempts times")
}

func TestGenerateWithRetry_NonRetryableError(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	var callCount int32
	generator := func(_ context.Context) (*GenerateResponse, error) {
		atomic.AddInt32(&callCount, 1)
		return nil, errInvalidAPIKey // NOT retryable
	}

	resp, err := GenerateWithRetry(ctx, cfg, logger, generator)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid API key")
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount), "should NOT retry non-retryable errors")
}

func TestGenerateWithRetry_ContextCancellation(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	logger := logrus.NewEntry(logrus.New())

	var callCount int32
	generator := func(_ context.Context) (*GenerateResponse, error) {
		count := atomic.AddInt32(&callCount, 1)
		if count == 1 {
			// First call fails, then cancel context
			cancel()
			return nil, errRateLimit // retryable
		}
		return &GenerateResponse{Content: "success"}, nil
	}

	resp, err := GenerateWithRetry(ctx, cfg, logger, generator)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, context.Canceled, err)
}

func TestGenerateWithRetry_ExponentialBackoff(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	}
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	var timestamps []time.Time
	generator := func(_ context.Context) (*GenerateResponse, error) {
		timestamps = append(timestamps, time.Now())
		return nil, errTimeout // retryable
	}

	start := time.Now()
	_, _ = GenerateWithRetry(ctx, cfg, logger, generator)
	elapsed := time.Since(start)

	// Should have 4 attempts
	assert.Len(t, timestamps, 4)

	// Total delay should be approximately 10ms + 20ms + 40ms = 70ms
	// (delays between attempts 1-2, 2-3, 3-4, no delay after last attempt)
	assert.GreaterOrEqual(t, elapsed, 60*time.Millisecond, "should have waited for retries")
	assert.Less(t, elapsed, 200*time.Millisecond, "delays should be exponential, not excessive")
}

func TestGenerateWithRetry_MaxDelayClamp(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     60 * time.Millisecond, // Clamp delay early
		Multiplier:   2.0,
	}
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	var timestamps []time.Time
	generator := func(_ context.Context) (*GenerateResponse, error) {
		timestamps = append(timestamps, time.Now())
		return nil, errTimeout // retryable
	}

	start := time.Now()
	_, _ = GenerateWithRetry(ctx, cfg, logger, generator)
	elapsed := time.Since(start)

	// Delays: 50ms, 60ms (clamped from 100), 60ms (clamped from 120) = 170ms
	// Without clamping would be: 50ms + 100ms + 200ms = 350ms
	assert.Less(t, elapsed, 250*time.Millisecond, "delays should be clamped to MaxDelay")
}

func TestGenerateWithRetry_NilLogger(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:  2,
		InitialDelay: 5 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
		Multiplier:   2.0,
	}
	ctx := context.Background()

	generator := func(_ context.Context) (*GenerateResponse, error) {
		return &GenerateResponse{Content: "success"}, nil
	}

	// Should not panic with nil logger
	resp, err := GenerateWithRetry(ctx, cfg, nil, generator)

	require.NoError(t, err)
	assert.Equal(t, "success", resp.Content)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantRetry   bool
		description string
	}{
		// Nil error
		{
			name:        "nil error",
			err:         nil,
			wantRetry:   false,
			description: "nil errors should not be retried",
		},
		// Rate limits
		{
			name:        "rate limit error",
			err:         errRateLimitExceeded,
			wantRetry:   true,
			description: "rate limit should be retried",
		},
		{
			name:        "429 error",
			err:         errHTTP429,
			wantRetry:   true,
			description: "429 status should be retried",
		},
		{
			name:        "too many requests",
			err:         errTooManyRequests,
			wantRetry:   true,
			description: "too many requests should be retried",
		},
		// Server errors
		{
			name:        "500 error",
			err:         errHTTP500,
			wantRetry:   true,
			description: "500 should be retried",
		},
		{
			name:        "502 error",
			err:         errHTTP502,
			wantRetry:   true,
			description: "502 should be retried",
		},
		{
			name:        "503 error",
			err:         errHTTP503,
			wantRetry:   true,
			description: "503 should be retried",
		},
		{
			name:        "service unavailable",
			err:         errServiceUnavailable,
			wantRetry:   true,
			description: "service unavailable should be retried",
		},
		{
			name:        "internal server error",
			err:         errInternalServerError,
			wantRetry:   true,
			description: "internal server error should be retried",
		},
		// Timeouts
		{
			name:        "timeout error",
			err:         errRequestTimeout,
			wantRetry:   true,
			description: "timeout should be retried",
		},
		{
			name:        "deadline exceeded",
			err:         errDeadlineExceeded,
			wantRetry:   true,
			description: "deadline exceeded should be retried",
		},
		// Network errors
		{
			name:        "connection error",
			err:         errConnectionRefused,
			wantRetry:   true,
			description: "connection errors should be retried",
		},
		{
			name:        "network error",
			err:         errNetworkUnreachable,
			wantRetry:   true,
			description: "network errors should be retried",
		},
		{
			name:        "temporary failure",
			err:         errTemporaryFailure,
			wantRetry:   true,
			description: "temporary failures should be retried",
		},
		{
			name:        "EOF error",
			err:         errUnexpectedEOF,
			wantRetry:   true,
			description: "EOF errors should be retried",
		},
		// Overload
		{
			name:        "overloaded error",
			err:         errServerOverloaded,
			wantRetry:   true,
			description: "overloaded should be retried",
		},
		{
			name:        "capacity error",
			err:         errAtCapacity,
			wantRetry:   true,
			description: "capacity errors should be retried",
		},
		// Non-retryable errors
		{
			name:        "invalid API key",
			err:         errInvalidAPIKey,
			wantRetry:   false,
			description: "auth errors should NOT be retried",
		},
		{
			name:        "bad request",
			err:         errHTTP400,
			wantRetry:   false,
			description: "400 errors should NOT be retried",
		},
		{
			name:        "not found",
			err:         errHTTP404,
			wantRetry:   false,
			description: "404 errors should NOT be retried",
		},
		{
			name:        "invalid prompt",
			err:         errInvalidPrompt,
			wantRetry:   false,
			description: "content errors should NOT be retried",
		},
		{
			name:        "unknown error",
			err:         errUnknown,
			wantRetry:   false,
			description: "unknown errors should NOT be retried",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryableError(tt.err)
			assert.Equal(t, tt.wantRetry, got, tt.description)
		})
	}
}

func TestIsRetryableError_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantRetry bool
	}{
		{"RATE LIMIT", errRateLimitUpper, true},
		{"Rate Limit", errRateLimitMixed, true},
		{"TIMEOUT", errTimeoutUpper, true},
		{"Timeout", errTimeoutMixed, true},
		{"SERVICE UNAVAILABLE", errServiceUnavailableUpper, true},
		{"Service Unavailable", errServiceUnavailableMixed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryableError(tt.err)
			assert.Equal(t, tt.wantRetry, got, "error matching should be case insensitive")
		})
	}
}
