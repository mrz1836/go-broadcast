// Package ai provides AI-powered text generation capabilities for go-broadcast.
// It supports multiple providers (Anthropic, OpenAI, Google) via the Genkit SDK.
package ai

import (
	"context"
	"time"
)

// Provider defines the interface for AI text generation services.
// Implementations must be safe for concurrent use.
type Provider interface {
	// Name returns the provider identifier (e.g., "anthropic", "openai", "google").
	Name() string

	// GenerateText generates text based on the given prompt.
	// Returns an error if generation fails or times out.
	GenerateText(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// IsAvailable checks if the provider is properly configured and ready.
	IsAvailable() bool
}

// TemperatureNotSet is a sentinel value indicating temperature should use provider default.
// Use this value in GenerateRequest.Temperature to defer to Config.Temperature.
const TemperatureNotSet float64 = -1.0

// GenerateRequest contains the input for text generation.
type GenerateRequest struct {
	// Prompt is the full text prompt to send to the AI.
	Prompt string

	// MaxTokens limits the response length.
	MaxTokens int

	// Temperature controls randomness (0.0-2.0).
	// Use TemperatureNotSet (-1.0) to use the provider's default.
	// Zero (0.0) is a valid temperature value (most deterministic).
	Temperature float64
}

// GenerateResponse contains the AI-generated output.
type GenerateResponse struct {
	// Content is the generated text.
	Content string

	// TokensUsed is the number of tokens consumed.
	TokensUsed int

	// FinishReason indicates why generation stopped.
	FinishReason string

	// Duration is the time taken for generation.
	Duration time.Duration
}

// FileChange represents a single file change for AI context.
// Used by both PR body and commit message generators.
type FileChange struct {
	// Path is the file path relative to repository root.
	Path string

	// ChangeType indicates the type of change: "added", "modified", or "deleted".
	ChangeType string

	// LinesAdded is the number of lines added.
	LinesAdded int

	// LinesRemoved is the number of lines removed.
	LinesRemoved int
}

// ProviderName constants for supported AI providers.
const (
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
	ProviderGoogle    = "google"
)

// GetDefaultModel returns the default model for the given provider.
func GetDefaultModel(provider string) string {
	switch provider {
	case ProviderAnthropic:
		return "claude-sonnet-4-20250514"
	case ProviderOpenAI:
		return "gpt-4o"
	case ProviderGoogle:
		return "gemini-2.5-flash"
	default:
		return ""
	}
}
