package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProviderFromEnv_DisabledByDefault(t *testing.T) {
	// Clear AI env vars to test default disabled state
	envVars := []string{
		"GO_BROADCAST_AI_ENABLED",
		"GO_BROADCAST_AI_API_KEY",
		"GO_BROADCAST_AI_PROVIDER",
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GEMINI_API_KEY",
	}
	for _, v := range envVars {
		t.Setenv(v, "")
	}

	provider, err := NewProviderFromEnv(context.Background(), nil)

	assert.Nil(t, provider)
	assert.ErrorIs(t, err, ErrProviderNotConfigured)
}

func TestNewProviderFromEnv_EnabledWithoutAPIKey(t *testing.T) {
	t.Setenv("GO_BROADCAST_AI_ENABLED", "true")
	t.Setenv("GO_BROADCAST_AI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")

	provider, err := NewProviderFromEnv(context.Background(), nil)

	assert.Nil(t, provider)
	assert.ErrorIs(t, err, ErrAPIKeyMissing)
}

func TestNewProviderFromEnv_UsesProviderSpecificAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		envVar     string
		apiKey     string
		expectName string
	}{
		{
			name:       "Anthropic provider key",
			provider:   ProviderAnthropic,
			envVar:     "ANTHROPIC_API_KEY",
			apiKey:     "anthropic-test-key",
			expectName: ProviderAnthropic,
		},
		{
			name:       "OpenAI provider key",
			provider:   ProviderOpenAI,
			envVar:     "OPENAI_API_KEY",
			apiKey:     "openai-test-key",
			expectName: ProviderOpenAI,
		},
		{
			name:       "Google provider key",
			provider:   ProviderGoogle,
			envVar:     "GEMINI_API_KEY",
			apiKey:     "google-test-key",
			expectName: ProviderGoogle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all API keys first
			t.Setenv("GO_BROADCAST_AI_API_KEY", "")
			t.Setenv("ANTHROPIC_API_KEY", "")
			t.Setenv("OPENAI_API_KEY", "")
			t.Setenv("GEMINI_API_KEY", "")

			// Set test config
			t.Setenv("GO_BROADCAST_AI_ENABLED", "true")
			t.Setenv("GO_BROADCAST_AI_PROVIDER", tt.provider)
			t.Setenv(tt.envVar, tt.apiKey)

			provider, err := NewProviderFromEnv(context.Background(), nil)

			require.NoError(t, err)
			require.NotNil(t, provider)
			assert.Equal(t, tt.expectName, provider.Name())
		})
	}
}

func TestNewProviderFromEnv_PrefersMainAPIKey(t *testing.T) {
	// Set both main and provider-specific keys
	t.Setenv("GO_BROADCAST_AI_ENABLED", "true")
	t.Setenv("GO_BROADCAST_AI_PROVIDER", ProviderAnthropic)
	t.Setenv("GO_BROADCAST_AI_API_KEY", "main-api-key")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-specific-key")

	provider, err := NewProviderFromEnv(context.Background(), nil)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, ProviderAnthropic, provider.Name())
}

func TestNewProviderFromEnv_UnsupportedProvider(t *testing.T) {
	t.Setenv("GO_BROADCAST_AI_ENABLED", "true")
	t.Setenv("GO_BROADCAST_AI_API_KEY", "test-key")
	t.Setenv("GO_BROADCAST_AI_PROVIDER", "unsupported-provider")

	provider, err := NewProviderFromEnv(context.Background(), nil)

	assert.Nil(t, provider)
	assert.ErrorIs(t, err, ErrUnsupportedProvider)
}

// TestNewProvider_AllValidProviders and TestMustNewProvider_Success
// are not included because they would require valid API keys to actually
// initialize the Genkit plugins. The OpenAI plugin validates API keys
// during initialization and panics on invalid keys.
// Provider initialization with valid keys is tested via integration tests.

func TestMustNewProvider_PanicsOnMissingAPIKey(t *testing.T) {
	cfg := &Config{
		Enabled:  true,
		APIKey:   "",
		Provider: ProviderAnthropic,
	}

	assert.Panics(t, func() {
		MustNewProvider(context.Background(), cfg, nil)
	})
}

func TestMustNewProvider_PanicsOnUnsupportedProvider(t *testing.T) {
	cfg := &Config{
		Enabled:  true,
		APIKey:   "test-key",
		Provider: "unsupported",
	}

	assert.Panics(t, func() {
		MustNewProvider(context.Background(), cfg, nil)
	})
}
