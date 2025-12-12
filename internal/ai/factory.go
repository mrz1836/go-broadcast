package ai

import (
	"context"

	"github.com/sirupsen/logrus"
)

// NewProvider creates an AI provider based on the configuration.
// It validates the configuration and returns an appropriate provider implementation.
// Currently supports Genkit-based providers for Anthropic, OpenAI, and Google.
func NewProvider(ctx context.Context, cfg *Config, logger *logrus.Entry) (Provider, error) {
	// Validate configuration
	if !cfg.Enabled {
		return nil, ErrProviderNotConfigured
	}

	if cfg.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}

	// Validate provider type
	switch cfg.Provider {
	case ProviderAnthropic, ProviderOpenAI, ProviderGoogle:
		// Valid providers
	default:
		return nil, ErrUnsupportedProvider
	}

	// Create Genkit-based provider
	return NewGenkitProvider(ctx, cfg, logger)
}

// NewProviderFromEnv creates an AI provider by loading configuration from environment variables.
// This is a convenience function for typical usage.
func NewProviderFromEnv(ctx context.Context, logger *logrus.Entry) (Provider, error) {
	cfg := LoadConfig()
	return NewProvider(ctx, cfg, logger)
}

// MustNewProvider creates a provider or panics if it fails.
// This is useful for initialization where failure should be fatal.
func MustNewProvider(ctx context.Context, cfg *Config, logger *logrus.Entry) Provider {
	provider, err := NewProvider(ctx, cfg, logger)
	if err != nil {
		panic(err)
	}
	return provider
}
