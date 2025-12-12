package ai

import (
	"errors"
	"fmt"
)

// Error templates for AI operations following project pattern from internal/errors/api_errors.go.
var (
	errAIGenerationTemplate = errors.New("AI generation failed")
	errAIProviderTemplate   = errors.New("AI provider error")
	errAIConfigTemplate     = errors.New("AI configuration error")
)

// Sentinel errors for AI operations.
var (
	ErrProviderNotConfigured = errors.New("AI provider not configured")
	ErrAPIKeyMissing         = errors.New("AI API key not provided")
	ErrUnsupportedProvider   = errors.New("unsupported AI provider")
	ErrGenerationTimeout     = errors.New("AI generation timed out")
	ErrCacheFull             = errors.New("AI response cache full")
	ErrEmptyResponse         = errors.New("AI returned empty response")
	// ErrFallbackUsed indicates AI generation failed and fallback was used.
	// This is NOT a fatal error - the returned message is valid, but callers
	// can check for this error to know if AI actually generated the content.
	ErrFallbackUsed = errors.New("AI generation failed, fallback used")
	// ErrInvalidFormat indicates AI response was in wrong format (e.g., commit message instead of PR body).
	ErrInvalidFormat = errors.New("AI response was in invalid format")
)

// GenerationError creates a standardized AI generation error.
//
// Example usage:
//
//	return GenerationError("anthropic", "PR body", err)
//	// Returns: "AI generation failed: anthropic 'PR body': <original error>"
func GenerationError(provider, context string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s '%s': %w", errAIGenerationTemplate, provider, context, err)
}

// ProviderError creates a standardized AI provider error.
//
// Example usage:
//
//	return ProviderError("anthropic", "initialize", err)
//	// Returns: "AI provider error: anthropic 'initialize': <original error>"
func ProviderError(provider, operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s '%s': %w", errAIProviderTemplate, provider, operation, err)
}

// ConfigError creates a standardized AI configuration error.
//
// Example usage:
//
//	return ConfigError("invalid temperature", "must be between 0.0 and 1.0")
//	// Returns: "AI configuration error: invalid temperature: must be between 0.0 and 1.0"
func ConfigError(field, reason string) error {
	return fmt.Errorf("%w: %s: %s", errAIConfigTemplate, field, reason)
}

// RateLimitError creates a standardized rate limit error for AI providers.
//
// Example usage:
//
//	return RateLimitError("anthropic", "60 seconds")
//	// Returns: "AI provider error: anthropic 'rate limit': retry after 60 seconds"
func RateLimitError(provider, retryAfter string) error {
	return fmt.Errorf("%w: %s 'rate limit': retry after %s", errAIProviderTemplate, provider, retryAfter)
}
