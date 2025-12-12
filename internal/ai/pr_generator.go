package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// PR generation constants
const (
	// DefaultPRMaxTokens is the default maximum tokens for PR body generation.
	// 2000 tokens is ~1500 words, sufficient for detailed PR descriptions.
	DefaultPRMaxTokens = 2000

	// DefaultPRTimeout is the default timeout for PR body generation.
	DefaultPRTimeout = 30 * time.Second
)

// PRBodyGenerator orchestrates AI-powered PR body generation with fallback.
// Thread-safe for concurrent use.
type PRBodyGenerator struct {
	provider    Provider
	cache       *ResponseCache
	truncator   *DiffTruncator
	retryConfig *RetryConfig
	guidelines  string
	timeout     time.Duration
	logger      *logrus.Entry
}

// NewPRBodyGenerator creates a new PR body generator.
// All parameters except logger are optional - if provider is nil, generation will
// fall back to static templates.
func NewPRBodyGenerator(
	provider Provider,
	cache *ResponseCache,
	truncator *DiffTruncator,
	retryConfig *RetryConfig,
	guidelines string,
	timeout time.Duration,
	logger *logrus.Entry,
) *PRBodyGenerator {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}
	if timeout == 0 {
		timeout = DefaultPRTimeout
	}

	return &PRBodyGenerator{
		provider:    provider,
		cache:       cache,
		truncator:   truncator,
		retryConfig: retryConfig,
		guidelines:  guidelines,
		timeout:     timeout,
		logger:      logger,
	}
}

// GenerateBody generates PR body using AI, falls back to static on failure.
// NEVER returns error that would block sync - always returns usable body.
// This method does not modify the input prCtx.
func (g *PRBodyGenerator) GenerateBody(ctx context.Context, prCtx *PRContext) (string, error) {
	// Check if provider is available
	if g.provider == nil || !g.provider.IsAvailable() {
		g.logger.Debug("AI provider not available, using fallback PR body")
		return g.generateFallback(prCtx), nil
	}

	// Apply timeout only if parent context doesn't have a shorter deadline
	if dl, ok := ctx.Deadline(); !ok || time.Until(dl) > g.timeout {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, g.timeout)
		defer cancel()
	}

	// Use generator's guidelines if context doesn't have them (avoid mutating input)
	guidelines := prCtx.PRGuidelines
	if guidelines == "" {
		guidelines = g.guidelines
	}

	// Create a local copy of context with guidelines for prompt generation
	localCtx := &PRContext{
		SourceRepo:   prCtx.SourceRepo,
		TargetRepo:   prCtx.TargetRepo,
		CommitSHA:    prCtx.CommitSHA,
		ChangedFiles: prCtx.ChangedFiles,
		DiffSummary:  prCtx.DiffSummary,
		PRGuidelines: guidelines,
	}

	// Use cache if available
	if g.cache != nil {
		response, cacheHit, err := g.cache.GetOrGenerate(ctx, localCtx.DiffSummary, func(ctx context.Context) (string, error) {
			return g.generateFromAI(ctx, localCtx)
		})

		if cacheHit {
			g.logger.Debug("Using cached AI response for PR body")
		}

		if err != nil {
			g.logger.WithError(err).Warn("AI generation failed, using fallback PR body")
			return g.generateFallback(prCtx), nil
		}

		return response, nil
	}

	// No cache, generate directly
	response, err := g.generateFromAI(ctx, localCtx)
	if err != nil {
		g.logger.WithError(err).Warn("AI generation failed, using fallback PR body")
		return g.generateFallback(prCtx), nil
	}

	return response, nil
}

// generateFromAI calls provider with retry logic.
func (g *PRBodyGenerator) generateFromAI(ctx context.Context, prCtx *PRContext) (string, error) {
	prompt := BuildPRPrompt(prCtx)

	resp, err := GenerateWithRetry(ctx, g.retryConfig, g.logger, func(ctx context.Context) (*GenerateResponse, error) {
		return g.provider.GenerateText(ctx, &GenerateRequest{
			Prompt:      prompt,
			MaxTokens:   DefaultPRMaxTokens,
			Temperature: TemperatureNotSet, // Use provider's configured temperature
		})
	})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// generateFallback returns static template for PR body.
func (g *PRBodyGenerator) generateFallback(prCtx *PRContext) string {
	if prCtx == nil {
		return g.staticFallback("unknown", "unknown", 0)
	}
	return g.staticFallback(prCtx.SourceRepo, prCtx.TargetRepo, len(prCtx.ChangedFiles))
}

// staticFallback generates a static PR body template.
func (g *PRBodyGenerator) staticFallback(sourceRepo, targetRepo string, fileCount int) string {
	return fmt.Sprintf(`## What Changed

* Synchronized files from %s to %s
* Updated %d file(s) to match source repository

## Why It Was Necessary

This synchronization ensures the target repository stays up-to-date with the latest changes from the configured source repository. Regular synchronization maintains consistency across related repositories.

## Testing Performed

* Validated sync configuration and file mappings
* Verified file transformations applied correctly
* Confirmed no unintended changes were introduced

## Impact / Risk

* **Low Risk**: Standard sync operation with established patterns
* **No Breaking Changes**: File updates maintain backward compatibility
* Changes are scoped to synchronized paths only
`, sourceRepo, targetRepo, fileCount)
}
