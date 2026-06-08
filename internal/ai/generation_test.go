package ai

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise the full GenkitProvider generation path end-to-end
// against a local mock AI backend (see mockserver_test.go). They make NO live
// HTTP requests and require NO API credentials, so they run as part of the
// normal test suite. The mock backend is wired in via the *_BASE_URL
// environment variables that each provider SDK already honors.

// TestGenkitProvider_Anthropic_Generate verifies the Anthropic backend through
// the full provider stack using a mock server (no live api.anthropic.com call).
func TestGenkitProvider_Anthropic_Generate(t *testing.T) {
	srv := newMockAIServer(t, "Hello, World!")

	cfg := &Config{
		Enabled:  true,
		Provider: ProviderAnthropic,
		APIKey:   "test-key",
		Model:    GetDefaultModel(ProviderAnthropic),
		Timeout:  30 * time.Second,
	}

	provider := newMockProvider(t, srv, cfg)

	assert.True(t, provider.IsAvailable(), "provider should be available")
	assert.Equal(t, ProviderAnthropic, provider.Name())

	resp, err := provider.GenerateText(context.Background(), &GenerateRequest{
		Prompt:      "Say 'Hello, World!' in exactly those words.",
		MaxTokens:   50,
		Temperature: 0,
	})

	require.NoError(t, err, "should generate text successfully")
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content)
	assert.Contains(t, resp.Content, "Hello")
	assert.Equal(t, 1, srv.callCount(), "should make exactly one backend request")
}

// TestGenkitProvider_OpenAI_Generate verifies the OpenAI backend through the
// full provider stack using a mock server (no live api.openai.com call).
func TestGenkitProvider_OpenAI_Generate(t *testing.T) {
	srv := newMockAIServer(t, "Hello, World!")

	cfg := &Config{
		Enabled:  true,
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		Model:    GetDefaultModel(ProviderOpenAI),
		Timeout:  30 * time.Second,
	}

	provider := newMockProvider(t, srv, cfg)

	assert.True(t, provider.IsAvailable(), "provider should be available")
	assert.Equal(t, ProviderOpenAI, provider.Name())

	resp, err := provider.GenerateText(context.Background(), &GenerateRequest{
		Prompt:      "Say 'Hello, World!' in exactly those words.",
		MaxTokens:   50,
		Temperature: 0,
	})

	require.NoError(t, err, "should generate text successfully")
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content)
	assert.Contains(t, resp.Content, "Hello")
	assert.Equal(t, 1, srv.callCount(), "should make exactly one backend request")
}

// TestGenkitProvider_Google_Generate verifies the Google Gemini backend through
// the full provider stack using a mock server (no live generativelanguage call).
func TestGenkitProvider_Google_Generate(t *testing.T) {
	srv := newMockAIServer(t, "Hello, World!")

	cfg := &Config{
		Enabled:  true,
		Provider: ProviderGoogle,
		APIKey:   "test-key",
		Model:    GetDefaultModel(ProviderGoogle),
		Timeout:  30 * time.Second,
	}

	provider := newMockProvider(t, srv, cfg)

	assert.True(t, provider.IsAvailable(), "provider should be available")
	assert.Equal(t, ProviderGoogle, provider.Name())

	resp, err := provider.GenerateText(context.Background(), &GenerateRequest{
		Prompt:      "Say 'Hello, World!' in exactly those words.",
		MaxTokens:   50,
		Temperature: 0,
	})

	require.NoError(t, err, "should generate text successfully")
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content)
	assert.Contains(t, resp.Content, "Hello")
	assert.Equal(t, 1, srv.callCount(), "should make exactly one backend request")
}

// TestCacheIntegration_IdenticalDiffs verifies that the response cache
// deduplicates identical diffs across multiple repository syncs, reducing
// backend calls. The mock server's call count proves only one request is made.
func TestCacheIntegration_IdenticalDiffs(t *testing.T) {
	srv := newMockAIServer(t, "update README with new section")

	cfg := &Config{
		Enabled:      true,
		Provider:     ProviderAnthropic,
		APIKey:       "test-key",
		Model:        GetDefaultModel(ProviderAnthropic),
		Timeout:      30 * time.Second,
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}

	logger := logrus.NewEntry(logrus.New())
	provider := newMockProvider(t, srv, cfg)

	cache := NewResponseCache(cfg)
	retryConfig := DefaultRetryConfig()

	gen := NewCommitMessageGenerator(provider, cache, nil, retryConfig, cfg, cfg.Timeout, logger)

	// Same diff for "multiple repos".
	diff := "diff --git a/README.md b/README.md\n+## New Section\n+Some content"

	commitCtx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target1",
		ChangedFiles: []FileChange{
			{Path: "README.md", ChangeType: "modified", LinesAdded: 2, LinesRemoved: 0},
		},
		DiffSummary: diff,
	}

	ctx := context.Background()

	// First call - cache miss, should call the backend.
	result1, err := gen.GenerateMessage(ctx, commitCtx)
	require.NoError(t, err)
	assert.NotEmpty(t, result1)
	assert.LessOrEqual(t, len(result1), 72, "commit message should be within limit")

	hits1, misses1, _ := cache.Stats()
	assert.Equal(t, int64(0), hits1)
	assert.Equal(t, int64(1), misses1)
	assert.Equal(t, 1, srv.callCount(), "first call should hit the backend once")

	// Second call with same diff - should hit cache, not the backend.
	commitCtx.TargetRepo = "owner/target2"
	result2, err := gen.GenerateMessage(ctx, commitCtx)
	require.NoError(t, err)
	assert.Equal(t, result1, result2, "cached response should match")

	hits2, misses2, _ := cache.Stats()
	assert.Equal(t, int64(1), hits2)
	assert.Equal(t, int64(1), misses2)
	assert.Equal(t, 1, srv.callCount(), "second call should be served from cache")
}

// TestEndToEnd_PRGeneration tests the full PR body generation workflow against a
// mock backend. It verifies that generated PR bodies contain the required
// sections (What Changed, Why, Testing, Impact) and are properly formatted.
func TestEndToEnd_PRGeneration(t *testing.T) {
	mockBody := `## What Changed

* Updated documentation and CI workflow

## Why It Was Necessary

To keep the target repository in sync with upstream.

## Testing Performed

* Verified the sync configuration

## Impact / Risk

* Low risk documentation update`

	srv := newMockAIServer(t, mockBody)

	cfg := &Config{
		Enabled:  true,
		Provider: ProviderAnthropic,
		APIKey:   "test-key",
		Model:    GetDefaultModel(ProviderAnthropic),
		Timeout:  30 * time.Second,
	}

	logger := logrus.NewEntry(logrus.New())
	provider := newMockProvider(t, srv, cfg)

	retryConfig := DefaultRetryConfig()
	gen := NewPRBodyGenerator(provider, nil, nil, retryConfig, cfg, "", cfg.Timeout, logger)

	prCtx := &PRContext{
		SourceRepo: "mrz1836/go-broadcast",
		TargetRepo: "example/target",
		CommitSHA:  "abc123def456",
		ChangedFiles: []FileChange{
			{Path: "README.md", ChangeType: "modified", LinesAdded: 10, LinesRemoved: 5},
			{Path: ".github/workflows/ci.yml", ChangeType: "added", LinesAdded: 50, LinesRemoved: 0},
		},
		DiffSummary: `diff --git a/README.md b/README.md
+## New Feature
+Added documentation for new sync feature.
diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
+name: CI
+on: push`,
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should contain the required sections.
	assert.Contains(t, result, "What Changed")
	assert.Contains(t, result, "Why")
	assert.Contains(t, result, "Testing")
	assert.Contains(t, result, "Impact")
}

// TestEndToEnd_CommitGeneration tests the full commit message generation
// workflow against a mock backend. It verifies that generated commit messages
// follow conventional commit format with the "sync:" prefix and stay within the
// 72-character limit.
func TestEndToEnd_CommitGeneration(t *testing.T) {
	srv := newMockAIServer(t, "add installation instructions to README")

	cfg := &Config{
		Enabled:  true,
		Provider: ProviderAnthropic,
		APIKey:   "test-key",
		Model:    GetDefaultModel(ProviderAnthropic),
		Timeout:  30 * time.Second,
	}

	logger := logrus.NewEntry(logrus.New())
	provider := newMockProvider(t, srv, cfg)

	retryConfig := DefaultRetryConfig()
	gen := NewCommitMessageGenerator(provider, nil, nil, retryConfig, cfg, cfg.Timeout, logger)

	commitCtx := &CommitContext{
		SourceRepo: "mrz1836/go-broadcast",
		TargetRepo: "example/target",
		ChangedFiles: []FileChange{
			{Path: "README.md", ChangeType: "modified", LinesAdded: 5, LinesRemoved: 2},
		},
		DiffSummary: `diff --git a/README.md b/README.md
+## Installation
+Run: go install github.com/mrz1836/go-broadcast@latest`,
	}

	result, err := gen.GenerateMessage(context.Background(), commitCtx)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should be valid conventional commit format.
	assert.LessOrEqual(t, len(result), 72, "commit message should be within 72 chars")
	assert.Contains(t, result, "sync:", "should have sync: prefix")
}
