//go:build integration

package ai

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests require API keys to run.
// Use build tag: go test -tags=integration ./internal/ai/...

func TestGenkitProvider_Anthropic_Integration(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	cfg := &Config{
		Enabled:  true,
		Provider: ProviderAnthropic,
		APIKey:   apiKey,
		Model:    GetDefaultModel(ProviderAnthropic),
		Timeout:  30 * time.Second,
	}

	logger := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	provider, err := NewGenkitProvider(ctx, cfg, logger)
	require.NoError(t, err, "should create Anthropic provider")
	require.NotNil(t, provider)

	assert.True(t, provider.IsAvailable(), "provider should be available")
	assert.Equal(t, ProviderAnthropic, provider.Name())

	// Test actual generation
	resp, err := provider.GenerateText(ctx, &GenerateRequest{
		Prompt:      "Say 'Hello, World!' in exactly those words.",
		MaxTokens:   50,
		Temperature: 0,
	})

	require.NoError(t, err, "should generate text successfully")
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content)
	assert.Contains(t, resp.Content, "Hello")
}

func TestGenkitProvider_OpenAI_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	cfg := &Config{
		Enabled:  true,
		Provider: ProviderOpenAI,
		APIKey:   apiKey,
		Model:    GetDefaultModel(ProviderOpenAI),
		Timeout:  30 * time.Second,
	}

	logger := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	provider, err := NewGenkitProvider(ctx, cfg, logger)
	require.NoError(t, err, "should create OpenAI provider")
	require.NotNil(t, provider)

	assert.True(t, provider.IsAvailable(), "provider should be available")
	assert.Equal(t, ProviderOpenAI, provider.Name())

	// Test actual generation
	resp, err := provider.GenerateText(ctx, &GenerateRequest{
		Prompt:      "Say 'Hello, World!' in exactly those words.",
		MaxTokens:   50,
		Temperature: 0,
	})

	require.NoError(t, err, "should generate text successfully")
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content)
	assert.Contains(t, resp.Content, "Hello")
}

func TestGenkitProvider_Google_Integration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	cfg := &Config{
		Enabled:  true,
		Provider: ProviderGoogle,
		APIKey:   apiKey,
		Model:    GetDefaultModel(ProviderGoogle),
		Timeout:  30 * time.Second,
	}

	logger := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	provider, err := NewGenkitProvider(ctx, cfg, logger)
	require.NoError(t, err, "should create Google provider")
	require.NotNil(t, provider)

	assert.True(t, provider.IsAvailable(), "provider should be available")
	assert.Equal(t, ProviderGoogle, provider.Name())

	// Test actual generation
	resp, err := provider.GenerateText(ctx, &GenerateRequest{
		Prompt:      "Say 'Hello, World!' in exactly those words.",
		MaxTokens:   50,
		Temperature: 0,
	})

	require.NoError(t, err, "should generate text successfully")
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content)
	assert.Contains(t, resp.Content, "Hello")
}

func TestCacheIntegration_IdenticalDiffs(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	cfg := &Config{
		Enabled:      true,
		Provider:     ProviderAnthropic,
		APIKey:       apiKey,
		Model:        GetDefaultModel(ProviderAnthropic),
		Timeout:      30 * time.Second,
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}

	logger := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	provider, err := NewGenkitProvider(ctx, cfg, logger)
	require.NoError(t, err)

	cache := NewResponseCache(cfg)
	retryConfig := DefaultRetryConfig()

	gen := NewCommitMessageGenerator(provider, cache, nil, retryConfig, cfg.Timeout, logger)

	// Same diff for "multiple repos"
	diff := "diff --git a/README.md b/README.md\n+## New Section\n+Some content"

	commitCtx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target1",
		ChangedFiles: []FileChange{
			{Path: "README.md", ChangeType: "modified", LinesAdded: 2, LinesRemoved: 0},
		},
		DiffSummary: diff,
	}

	// First call - cache miss, should call AI
	result1, err := gen.GenerateMessage(ctx, commitCtx)
	require.NoError(t, err)
	assert.NotEmpty(t, result1)
	assert.True(t, len(result1) <= 72, "commit message should be within limit")

	hits1, misses1, _ := cache.Stats()
	assert.Equal(t, int64(0), hits1)
	assert.Equal(t, int64(1), misses1)

	// Second call with same diff - should hit cache
	commitCtx.TargetRepo = "owner/target2"
	result2, err := gen.GenerateMessage(ctx, commitCtx)
	require.NoError(t, err)
	assert.Equal(t, result1, result2, "cached response should match")

	hits2, misses2, _ := cache.Stats()
	assert.Equal(t, int64(1), hits2)
	assert.Equal(t, int64(1), misses2)
}

func TestEndToEnd_PRGeneration(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	cfg := &Config{
		Enabled:  true,
		Provider: ProviderAnthropic,
		APIKey:   apiKey,
		Model:    GetDefaultModel(ProviderAnthropic),
		Timeout:  30 * time.Second,
	}

	logger := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	provider, err := NewGenkitProvider(ctx, cfg, logger)
	require.NoError(t, err)

	retryConfig := DefaultRetryConfig()
	gen := NewPRBodyGenerator(provider, nil, nil, retryConfig, "", cfg.Timeout, logger)

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

	result, err := gen.GenerateBody(ctx, prCtx)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should contain the required sections
	assert.Contains(t, result, "What Changed")
	assert.Contains(t, result, "Why")
	assert.Contains(t, result, "Testing")
	assert.Contains(t, result, "Impact")
}

func TestEndToEnd_CommitGeneration(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	cfg := &Config{
		Enabled:  true,
		Provider: ProviderAnthropic,
		APIKey:   apiKey,
		Model:    GetDefaultModel(ProviderAnthropic),
		Timeout:  30 * time.Second,
	}

	logger := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	provider, err := NewGenkitProvider(ctx, cfg, logger)
	require.NoError(t, err)

	retryConfig := DefaultRetryConfig()
	gen := NewCommitMessageGenerator(provider, nil, nil, retryConfig, cfg.Timeout, logger)

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

	result, err := gen.GenerateMessage(ctx, commitCtx)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should be valid conventional commit format
	assert.True(t, len(result) <= 72, "commit message should be within 72 chars")
	assert.Contains(t, result, "sync:", "should have sync: prefix")
}
