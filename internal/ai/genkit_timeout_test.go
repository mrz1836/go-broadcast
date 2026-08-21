package ai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPerRequestTimeout(t *testing.T) {
	assert.Equal(t, 20*time.Second, perRequestTimeout(&Config{Timeout: 30 * time.Second}))
	assert.Equal(t, 30*time.Second, perRequestTimeout(&Config{Timeout: 45 * time.Second}))

	// Unset/invalid budget falls back to the default.
	assert.Equal(t, DefaultRequestTimeout*2/3, perRequestTimeout(&Config{Timeout: 0}))
	assert.Equal(t, DefaultRequestTimeout*2/3, perRequestTimeout(&Config{Timeout: -1}))

	// Must always be strictly below the overall budget so the per-attempt
	// deadline engages and a retry can fit within the same generation ctx.
	for _, d := range []time.Duration{5 * time.Second, 30 * time.Second, 90 * time.Second} {
		assert.Lessf(t, perRequestTimeout(&Config{Timeout: d}), d,
			"per-request timeout must be < overall budget %s", d)
	}
}

func TestCompatRequestOpts(t *testing.T) {
	// Exactly one option (the request timeout); SDK default retries are kept.
	opts := compatRequestOpts(&Config{Timeout: 30 * time.Second})
	assert.Len(t, opts, 1)
}

func TestPerRequestHTTPClient(t *testing.T) {
	// Google's transport-level bound must match the same per-request timeout the
	// OpenAI-compatible providers use, so all providers behave identically.
	c := perRequestHTTPClient(&Config{Timeout: 30 * time.Second})
	assert.Equal(t, 20*time.Second, c.Timeout)
	assert.Equal(t, perRequestTimeout(&Config{Timeout: 30 * time.Second}), c.Timeout)

	// Fallback budget still yields a bounded (non-zero) client.
	c0 := perRequestHTTPClient(&Config{Timeout: 0})
	assert.Positive(t, c0.Timeout)
}
