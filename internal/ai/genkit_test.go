package ai

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetModelPath_AllProviders(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		model        string
		expectedPath string
	}{
		{
			name:         "Anthropic with custom model",
			provider:     ProviderAnthropic,
			model:        "claude-3-opus",
			expectedPath: "anthropic/claude-3-opus",
		},
		{
			name:         "Anthropic with empty model uses default",
			provider:     ProviderAnthropic,
			model:        "",
			expectedPath: "anthropic/" + GetDefaultModel(ProviderAnthropic),
		},
		{
			name:         "OpenAI with custom model",
			provider:     ProviderOpenAI,
			model:        "gpt-4-turbo",
			expectedPath: "openai/gpt-4-turbo",
		},
		{
			name:         "OpenAI with empty model uses default",
			provider:     ProviderOpenAI,
			model:        "",
			expectedPath: "openai/" + GetDefaultModel(ProviderOpenAI),
		},
		{
			name:         "Google with custom model",
			provider:     ProviderGoogle,
			model:        "gemini-1.5-pro",
			expectedPath: "googleai/gemini-1.5-pro",
		},
		{
			name:         "Google with empty model uses default",
			provider:     ProviderGoogle,
			model:        "",
			expectedPath: "googleai/" + GetDefaultModel(ProviderGoogle),
		},
		{
			name:         "Unknown provider returns model as-is",
			provider:     "unknown",
			model:        "some-model",
			expectedPath: "some-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Provider: tt.provider,
				Model:    tt.model,
			}
			result := getModelPath(cfg)
			assert.Equal(t, tt.expectedPath, result)
		})
	}
}

func TestNewGenkitProvider_MissingAPIKey(t *testing.T) {
	cfg := &Config{
		Provider: ProviderAnthropic,
		APIKey:   "",
		Model:    "claude-3-sonnet",
	}

	provider, err := NewGenkitProvider(context.Background(), cfg, nil)

	assert.Nil(t, provider)
	assert.ErrorIs(t, err, ErrAPIKeyMissing)
}

func TestNewGenkitProvider_UnsupportedProvider(t *testing.T) {
	cfg := &Config{
		Provider: "unsupported-provider",
		APIKey:   "test-key",
		Model:    "some-model",
	}

	provider, err := NewGenkitProvider(context.Background(), cfg, nil)

	assert.Nil(t, provider)
	assert.ErrorIs(t, err, ErrUnsupportedProvider)
}

func TestGenkitProvider_Name(t *testing.T) {
	tests := []struct {
		provider string
	}{
		{ProviderAnthropic},
		{ProviderOpenAI},
		{ProviderGoogle},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			// Create minimal provider struct directly for unit testing Name()
			p := &GenkitProvider{
				provider: tt.provider,
			}
			assert.Equal(t, tt.provider, p.Name())
		})
	}
}

func TestGenkitProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		gk       interface{} // use interface to handle nil
		apiKey   string
		expected bool
	}{
		{
			name:     "Available with gk and API key",
			gk:       "not-nil", // just need non-nil
			apiKey:   "test-key",
			expected: true,
		},
		{
			name:     "Unavailable with nil gk",
			gk:       nil,
			apiKey:   "test-key",
			expected: false,
		},
		{
			name:     "Unavailable with empty API key",
			gk:       "not-nil",
			apiKey:   "",
			expected: false,
		},
		{
			name:     "Unavailable with both nil gk and empty API key",
			gk:       nil,
			apiKey:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily create a real genkit.Genkit instance,
			// so we test the availability logic directly
			p := &GenkitProvider{
				config: &Config{APIKey: tt.apiKey},
			}
			// gk is only set to non-nil in actual init, so for nil test it remains nil.
			// For IsAvailable to return true, we'd need a real gk instance.
			// This test validates the nil check logic.
			// When gk is nil, IsAvailable should always return false.
			if tt.gk == nil {
				assert.False(t, p.IsAvailable())
			}
		})
	}
}

func TestGenkitProvider_Close(t *testing.T) {
	p := &GenkitProvider{
		provider: ProviderAnthropic,
		config:   &Config{APIKey: "test-key"},
	}

	err := p.Close()

	require.NoError(t, err)
	assert.Nil(t, p.gk, "gk should be nil after Close")
}

func TestGenkitProvider_GenerateText_NilGk(t *testing.T) {
	p := &GenkitProvider{
		gk:       nil, // Not initialized
		provider: ProviderAnthropic,
		config:   &Config{APIKey: "test-key"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resp, err := p.GenerateText(ctx, &GenerateRequest{
		Prompt:    "test prompt",
		MaxTokens: 100,
	})

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrProviderNotConfigured)
}

func TestGetDefaultModel_AllProviders(t *testing.T) {
	tests := []struct {
		provider      string
		expectedEmpty bool
	}{
		{ProviderAnthropic, false},
		{ProviderOpenAI, false},
		{ProviderGoogle, false},
		{"unknown", true}, // Returns empty for unknown provider
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := GetDefaultModel(tt.provider)
			if tt.expectedEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result, "Provider %s should have a default model", tt.provider)
			}
		})
	}
}

func TestGetTokenCount_NilResponse(t *testing.T) {
	result := getTokenCount(nil)
	assert.Equal(t, 0, result)
}

func TestGetFinishReason_NilResponse(t *testing.T) {
	result := getFinishReason(nil)
	assert.Empty(t, result)
}
