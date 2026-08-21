package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// Commit generation constants
const (
	// DefaultCommitMaxTokens is the default maximum tokens for commit message generation.
	// Commit messages should be short (~50 chars for subject), so 100 tokens is plenty.
	DefaultCommitMaxTokens = 100

	// DefaultCommitTimeout is the default timeout for commit message generation.
	DefaultCommitTimeout = 30 * time.Second
)

// CommitMessageGenerator orchestrates AI-powered commit message generation.
// Thread-safe for concurrent use.
type CommitMessageGenerator struct {
	provider    Provider
	cache       *ResponseCache
	truncator   *DiffTruncator
	retryConfig *RetryConfig
	config      *Config
	timeout     time.Duration
	logger      *logrus.Entry
}

// NewCommitMessageGenerator creates a new commit message generator.
// All parameters except logger are optional - if provider is nil, generation will
// fall back to static templates.
func NewCommitMessageGenerator(
	provider Provider,
	cache *ResponseCache,
	truncator *DiffTruncator,
	retryConfig *RetryConfig,
	config *Config,
	timeout time.Duration,
	logger *logrus.Entry,
) *CommitMessageGenerator {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}
	if timeout == 0 {
		timeout = DefaultCommitTimeout
	}

	return &CommitMessageGenerator{
		provider:    provider,
		cache:       cache,
		truncator:   truncator,
		retryConfig: retryConfig,
		config:      config,
		timeout:     timeout,
		logger:      logger,
	}
}

// GenerateMessage generates commit message, falls back to static on failure.
// ALWAYS validates AI response through ValidateCommitMessage before returning.
// NEVER returns error that would block sync - always returns usable message.
// Returns ErrFallbackUsed when AI generation failed and fallback was used.
// Callers can check errors.Is(err, ErrFallbackUsed) to know if AI generated the message.
func (g *CommitMessageGenerator) GenerateMessage(ctx context.Context, commitCtx *CommitContext) (string, error) {
	// Guard against nil context - use fallback
	if commitCtx == nil {
		g.logger.Debug("Commit context is nil, using fallback commit message")
		return g.generateFallback(nil), ErrFallbackUsed
	}

	// Check if provider is available
	if g.provider == nil || !g.provider.IsAvailable() {
		g.logger.Debug("AI provider not available, using fallback commit message")
		return g.generateFallback(commitCtx), ErrFallbackUsed
	}

	// Apply timeout only if parent context doesn't have a shorter deadline
	if dl, ok := ctx.Deadline(); !ok || time.Until(dl) > g.timeout {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, g.timeout)
		defer cancel()
	}

	// Extract authoritative facts from the FULL (untruncated) diff. These drive
	// both the prompt injection and the post-generation subject guard, so the
	// commit subject can never state a version number that is not in the diff.
	diffForFacts := commitCtx.FullDiff
	if diffForFacts == "" {
		diffForFacts = commitCtx.DiffSummary
	}
	changeset := ExtractChangeset(diffForFacts)

	// Local copy with verified changes for the prompt (avoid mutating input).
	localCtx := *commitCtx
	localCtx.VerifiedChanges = RenderVerifiedChanges(changeset)

	var response string
	var err error

	// Use cache if available
	cacheKey := commitCtx.FullDiff
	if cacheKey == "" {
		cacheKey = commitCtx.DiffSummary
	}
	if g.cache != nil {
		g.logger.Info("Generating commit message with AI...")
		var cacheHit bool
		response, cacheHit, err = g.cache.GetOrGenerate(ctx, "commit:", cacheKey, func(ctx context.Context) (string, error) {
			return g.generateFromAI(ctx, &localCtx)
		})

		if cacheHit {
			g.logger.Debug("Using cached AI response for commit message")
		}
	} else {
		// No cache, generate directly
		g.logger.Info("Generating commit message with AI...")
		response, err = g.generateFromAI(ctx, &localCtx)
	}

	if err != nil {
		// Check if FailOnError is enabled - return blocking error
		if g.config != nil && g.config.FailOnError {
			g.logger.WithError(err).Error("AI generation failed and fail_on_error is enabled")
			return "", FailedError(g.provider.Name(), "commit message", g.config.APIKeySource, err)
		}
		g.logger.WithError(err).Warn("AI generation failed, using fallback commit message")
		return g.generateFallback(commitCtx), ErrFallbackUsed
	}

	// CRITICAL: Always validate AI response
	validated := ValidateCommitMessage(response)
	if validated == "" {
		g.logger.Warn("AI generated empty commit message, using fallback")
		return g.generateFallback(commitCtx), ErrFallbackUsed
	}

	// Deterministic guard: if the subject cites a version not in the diff, replace
	// it with a verified subject built from the changeset (or fall back).
	guarded, hallucinated := GuardCommitSubject(validated, changeset)
	if hallucinated {
		g.logger.WithField("original", validated).Warn("Commit subject cited a version not in the diff; using verified subject")
		if guarded == "" {
			return g.generateFallback(commitCtx), ErrFallbackUsed
		}
		if validated = ValidateCommitMessage(guarded); validated == "" {
			return g.generateFallback(commitCtx), ErrFallbackUsed
		}
	}

	return validated, nil
}

// generateFromAI calls provider with retry logic.
func (g *CommitMessageGenerator) generateFromAI(ctx context.Context, commitCtx *CommitContext) (string, error) {
	prompt := BuildCommitPrompt(commitCtx)

	resp, err := GenerateWithRetry(ctx, g.retryConfig, g.logger, func(ctx context.Context) (*GenerateResponse, error) {
		return g.provider.GenerateText(ctx, &GenerateRequest{
			Prompt:      prompt,
			MaxTokens:   DefaultCommitMaxTokens,
			Temperature: TemperatureNotSet, // Use provider's configured temperature
		})
	})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// generateFallback returns a static message. It prefers a specific, verified
// subject derived deterministically from the diff (e.g. "sync: bump MAGE_X_VERSION
// to v1.26.3") and only falls back to a generic file-count message when no
// significant key changes are available.
func (g *CommitMessageGenerator) generateFallback(commitCtx *CommitContext) string {
	if commitCtx == nil || len(commitCtx.ChangedFiles) == 0 {
		return "sync: update files from source repository"
	}

	// Prefer a precise, machine-verified subject when the diff reveals one.
	diff := commitCtx.FullDiff
	if diff == "" {
		diff = commitCtx.DiffSummary
	}
	if diff != "" {
		if subject := DeterministicCommitSubject(ExtractChangeset(diff)); subject != "" {
			return subject
		}
	}

	if len(commitCtx.ChangedFiles) == 1 {
		return fmt.Sprintf("sync: update %s from source repository", commitCtx.ChangedFiles[0].Path)
	}

	return fmt.Sprintf("sync: update %d files from source repository", len(commitCtx.ChangedFiles))
}
