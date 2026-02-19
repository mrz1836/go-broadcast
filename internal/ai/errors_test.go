package ai

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestGeneration = errors.New("generation failed")

func TestFailedError(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns nil", func(t *testing.T) {
		t.Parallel()

		result := FailedError("anthropic", "commit message", "ANTHROPIC_API_KEY", nil)
		assert.NoError(t, result)
	})

	t.Run("wraps error with context", func(t *testing.T) {
		t.Parallel()

		result := FailedError("anthropic", "commit message", "ANTHROPIC_API_KEY", errTestGeneration)
		require.Error(t, result)
		require.ErrorIs(t, result, ErrAIGenerationFailed)
		assert.Contains(t, result.Error(), "ANTHROPIC_API_KEY")
		assert.Contains(t, result.Error(), "anthropic")
		assert.Contains(t, result.Error(), "commit message")
		assert.Contains(t, result.Error(), "generation failed")
	})

	t.Run("different provider and context", func(t *testing.T) {
		t.Parallel()

		result := FailedError("openai", "PR body", "OPENAI_API_KEY", errTestGeneration)
		require.Error(t, result)
		assert.Contains(t, result.Error(), "openai")
		assert.Contains(t, result.Error(), "PR body")
		assert.Contains(t, result.Error(), "OPENAI_API_KEY")
	})
}

func TestGenerationError_Helpers(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns nil", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, GenerationError("anthropic", "test", nil))
	})

	t.Run("wraps error", func(t *testing.T) {
		t.Parallel()
		err := GenerationError("anthropic", "PR body", errTestGeneration)
		require.Error(t, err)
		require.ErrorIs(t, err, errAIGenerationTemplate)
		assert.Contains(t, err.Error(), "anthropic")
		assert.Contains(t, err.Error(), "PR body")
	})
}

func TestProviderError_Helpers(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns nil", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, ProviderError("anthropic", "init", nil))
	})

	t.Run("wraps error", func(t *testing.T) {
		t.Parallel()
		err := ProviderError("openai", "connect", errTestGeneration)
		require.Error(t, err)
		require.ErrorIs(t, err, errAIProviderTemplate)
		assert.Contains(t, err.Error(), "openai")
		assert.Contains(t, err.Error(), "connect")
	})
}

func TestConfigError_Helpers(t *testing.T) {
	t.Parallel()

	err := ConfigError("temperature", "must be between 0.0 and 1.0")
	require.Error(t, err)
	require.ErrorIs(t, err, errAIConfigTemplate)
	assert.Contains(t, err.Error(), "temperature")
	assert.Contains(t, err.Error(), "must be between 0.0 and 1.0")
}

func TestRateLimitError_Helpers(t *testing.T) {
	t.Parallel()

	err := RateLimitError("anthropic", "60 seconds")
	require.Error(t, err)
	require.ErrorIs(t, err, errAIProviderTemplate)
	assert.Contains(t, err.Error(), "rate limit")
	assert.Contains(t, err.Error(), "60 seconds")
}
