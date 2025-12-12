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
//
// IMPORTANT: These tests make REAL API calls and cost money.
// They are skipped by default. To run them:
// 1. Comment out the t.Skip() line in the test you want to run
// 2. Ensure the appropriate API key environment variable is set
// 3. Run with: go test -tags=integration ./internal/ai/... -run TestName

// TestGenkitProvider_Anthropic_Integration tests real API calls to Anthropic Claude.
//
// SKIPPED BY DEFAULT: This test makes real API calls that cost money and require
// valid API credentials. It is skipped to prevent accidental charges when running
// the test suite locally with API keys in environment variables.
//
// To run this test:
//  1. Comment out the t.Skip() line below
//  2. Set ANTHROPIC_API_KEY environment variable
//  3. Run: go test -tags=integration ./internal/ai/... -run TestGenkitProvider_Anthropic_Integration -v
func TestGenkitProvider_Anthropic_Integration(t *testing.T) {
	// SKIP: Remove this line to enable real API testing (will incur costs)
	t.Skip("Real AI API test - comment out this line to run (requires ANTHROPIC_API_KEY)")

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
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

// TestGenkitProvider_OpenAI_Integration tests real API calls to OpenAI GPT models.
//
// SKIPPED BY DEFAULT: This test makes real API calls that cost money and require
// valid API credentials. It is skipped to prevent accidental charges when running
// the test suite locally with API keys in environment variables.
//
// To run this test:
//  1. Comment out the t.Skip() line below
//  2. Set OPENAI_API_KEY environment variable
//  3. Run: go test -tags=integration ./internal/ai/... -run TestGenkitProvider_OpenAI_Integration -v
func TestGenkitProvider_OpenAI_Integration(t *testing.T) {
	// SKIP: Remove this line to enable real API testing (will incur costs)
	t.Skip("Real AI API test - comment out this line to run (requires OPENAI_API_KEY)")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
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

// TestGenkitProvider_Google_Integration tests real API calls to Google Gemini models.
//
// SKIPPED BY DEFAULT: This test makes real API calls that cost money and require
// valid API credentials. It is skipped to prevent accidental charges when running
// the test suite locally with API keys in environment variables.
//
// To run this test:
//  1. Comment out the t.Skip() line below
//  2. Set GEMINI_API_KEY environment variable
//  3. Run: go test -tags=integration ./internal/ai/... -run TestGenkitProvider_Google_Integration -v
func TestGenkitProvider_Google_Integration(t *testing.T) {
	// SKIP: Remove this line to enable real API testing (will incur costs)
	t.Skip("Real AI API test - comment out this line to run (requires GEMINI_API_KEY)")

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set")
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

// TestCacheIntegration_IdenticalDiffs tests that the response cache correctly deduplicates
// identical diffs across multiple repository syncs, reducing API calls and costs.
//
// SKIPPED BY DEFAULT: This test makes real API calls that cost money and require
// valid API credentials. It is skipped to prevent accidental charges when running
// the test suite locally with API keys in environment variables.
//
// To run this test:
//  1. Comment out the t.Skip() line below
//  2. Set ANTHROPIC_API_KEY environment variable
//  3. Run: go test -tags=integration ./internal/ai/... -run TestCacheIntegration_IdenticalDiffs -v
func TestCacheIntegration_IdenticalDiffs(t *testing.T) {
	// SKIP: Remove this line to enable real API testing (will incur costs)
	t.Skip("Real AI API test - comment out this line to run (requires ANTHROPIC_API_KEY)")

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
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

// TestEndToEnd_PRGeneration tests the full PR body generation workflow with real AI.
// It verifies that generated PR bodies contain the required sections (What Changed,
// Why, Testing, Impact) and are properly formatted.
//
// SKIPPED BY DEFAULT: This test makes real API calls that cost money and require
// valid API credentials. It is skipped to prevent accidental charges when running
// the test suite locally with API keys in environment variables.
//
// To run this test:
//  1. Comment out the t.Skip() line below
//  2. Set ANTHROPIC_API_KEY environment variable
//  3. Run: go test -tags=integration ./internal/ai/... -run TestEndToEnd_PRGeneration -v
func TestEndToEnd_PRGeneration(t *testing.T) {
	// SKIP: Remove this line to enable real API testing (will incur costs)
	t.Skip("Real AI API test - comment out this line to run (requires ANTHROPIC_API_KEY)")

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
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

// TestEndToEnd_CommitGeneration tests the full commit message generation workflow with real AI.
// It verifies that generated commit messages follow conventional commit format with the
// "sync:" prefix and are within the 72-character limit.
//
// SKIPPED BY DEFAULT: This test makes real API calls that cost money and require
// valid API credentials. It is skipped to prevent accidental charges when running
// the test suite locally with API keys in environment variables.
//
// To run this test:
//  1. Comment out the t.Skip() line below
//  2. Set ANTHROPIC_API_KEY environment variable
//  3. Run: go test -tags=integration ./internal/ai/... -run TestEndToEnd_CommitGeneration -v
func TestEndToEnd_CommitGeneration(t *testing.T) {
	// SKIP: Remove this line to enable real API testing (will incur costs)
	t.Skip("Real AI API test - comment out this line to run (requires ANTHROPIC_API_KEY)")

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
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
