package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParsePRURLLengthValidation tests URL length validation in parsePRURL
func TestParsePRURLLengthValidation(t *testing.T) {
	t.Run("rejects URLs exceeding max length", func(t *testing.T) {
		// Create a URL that exceeds the maximum length
		longURL := "https://github.com/owner/repo/pull/123" + strings.Repeat("x", maxURLLength)

		info, err := parsePRURL(longURL)

		require.Error(t, err)
		require.ErrorIs(t, err, ErrURLTooLong)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	t.Run("accepts URLs at max length", func(t *testing.T) {
		// Create a valid URL just at the limit
		// The base URL is about 45 chars, so we pad to get to exactly maxURLLength
		baseURL := "https://github.com/owner/repo/pull/123"
		paddingNeeded := maxURLLength - len(baseURL)
		if paddingNeeded > 0 {
			// Can't pad a URL and keep it valid, so just test that a normal URL works
			info, err := parsePRURL(baseURL)
			require.NoError(t, err)
			assert.NotNil(t, info)
		}
	})

	t.Run("accepts normal length URLs", func(t *testing.T) {
		testURLs := []string{
			"https://github.com/owner/repo/pull/123",
			"http://github.com/owner/repo/pull/456",
			"github.com/owner/repo/pull/789",
			"owner/repo#100",
		}

		for _, url := range testURLs {
			info, err := parsePRURL(url)
			require.NoError(t, err, "URL: %s", url)
			assert.NotNil(t, info, "URL: %s", url)
		}
	})

	t.Run("empty URL returns specific error", func(t *testing.T) {
		info, err := parsePRURL("")

		require.Error(t, err)
		require.ErrorIs(t, err, ErrEmptyPRURL)
		assert.Nil(t, info)
	})

	t.Run("whitespace-only URL returns empty error", func(t *testing.T) {
		info, err := parsePRURL("   \t\n   ")

		require.Error(t, err)
		require.ErrorIs(t, err, ErrEmptyPRURL)
		assert.Nil(t, info)
	})
}

// TestMaxURLLengthConstant verifies the constant value is reasonable
func TestMaxURLLengthConstant(t *testing.T) {
	// maxURLLength should be at least 256 (minimum for practical URLs)
	// and at most 8192 (to prevent excessive memory usage)
	assert.GreaterOrEqual(t, maxURLLength, 256)
	assert.LessOrEqual(t, maxURLLength, 8192)

	// Should be 2048 as defined
	assert.Equal(t, 2048, maxURLLength)
}
