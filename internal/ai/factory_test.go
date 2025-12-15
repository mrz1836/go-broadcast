package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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

// TestNewProviderFromEnv_UsesProviderSpecificAPIKey and TestNewProviderFromEnv_PrefersMainAPIKey
// are located in integration_test.go because they initialize real Genkit providers
// which may make network requests during plugin initialization.

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
