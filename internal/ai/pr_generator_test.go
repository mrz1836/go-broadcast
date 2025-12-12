package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test errors - defined at package level per linting rules.
var (
	errAPIError         = errors.New("API error")
	errAPIFailed        = errors.New("API failed")
	errAIGenerationFail = errors.New("AI generation failed")
	errConnectionFailed = errors.New("connection failed")
	errTestError        = errors.New("test error")
)

func TestNewPRBodyGenerator(t *testing.T) {
	t.Run("with all parameters", func(t *testing.T) {
		mockProvider := NewMockProvider()
		cfg := &Config{
			CacheEnabled: true,
			CacheTTL:     time.Hour,
			CacheMaxSize: 100,
		}
		cache := NewResponseCache(cfg)
		truncator := NewDiffTruncator(cfg)
		retryConfig := DefaultRetryConfig()
		logger := logrus.NewEntry(logrus.New())

		gen := NewPRBodyGenerator(mockProvider, cache, truncator, retryConfig, "guidelines", 30*time.Second, logger)

		require.NotNil(t, gen)
		assert.Equal(t, mockProvider, gen.provider)
		assert.Equal(t, cache, gen.cache)
		assert.Equal(t, truncator, gen.truncator)
		assert.Equal(t, "guidelines", gen.guidelines)
		assert.Equal(t, 30*time.Second, gen.timeout)
	})

	t.Run("with nil parameters uses defaults", func(t *testing.T) {
		gen := NewPRBodyGenerator(nil, nil, nil, nil, "", 0, nil)

		require.NotNil(t, gen)
		assert.Nil(t, gen.provider)
		assert.Nil(t, gen.cache)
		assert.NotNil(t, gen.retryConfig) // DefaultRetryConfig applied
		assert.Equal(t, 30*time.Second, gen.timeout)
		assert.NotNil(t, gen.logger)
	})
}

func TestPRBodyGenerator_GenerateBody_AISuccess(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Return(
		&GenerateResponse{Content: "## What Changed\n\nAI generated content"},
		nil,
	)

	gen := NewPRBodyGenerator(mockProvider, nil, nil, nil, "", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		CommitSHA:    "abc123",
		ChangedFiles: []FileChange{{Path: "README.md", ChangeType: "modified"}},
		DiffSummary:  "diff content",
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)

	require.NoError(t, err)
	assert.Contains(t, result, "AI generated content")
	mockProvider.AssertExpectations(t)
}

func TestPRBodyGenerator_GenerateBody_CacheHit(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)

	// Pre-populate cache with prefixed key
	cache.Set("pr:diff content", "cached PR body")

	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	// GenerateText should NOT be called due to cache hit

	gen := NewPRBodyGenerator(mockProvider, cache, nil, nil, "", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo:  "owner/source",
		TargetRepo:  "owner/target",
		DiffSummary: "diff content", // Same as cached
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)

	require.NoError(t, err)
	assert.Equal(t, "cached PR body", result)
	mockProvider.AssertNotCalled(t, "GenerateText", mock.Anything, mock.Anything)
}

func TestPRBodyGenerator_GenerateBody_ProviderNil(t *testing.T) {
	gen := NewPRBodyGenerator(nil, nil, nil, nil, "", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "file.go", ChangeType: "added"}},
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)

	require.ErrorIs(t, err, ErrFallbackUsed, "should return ErrFallbackUsed when provider is nil")
	assert.Contains(t, result, "What Changed")
	assert.Contains(t, result, "owner/source")
	assert.Contains(t, result, "owner/target")
	assert.Contains(t, result, "1 file(s)")
}

func TestPRBodyGenerator_GenerateBody_ProviderUnavailable(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(false)

	gen := NewPRBodyGenerator(mockProvider, nil, nil, nil, "", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "file.go", ChangeType: "added"}},
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)

	require.ErrorIs(t, err, ErrFallbackUsed, "should return ErrFallbackUsed when provider unavailable")
	assert.Contains(t, result, "What Changed")
	mockProvider.AssertNotCalled(t, "GenerateText", mock.Anything, mock.Anything)
}

func TestPRBodyGenerator_GenerateBody_AIError(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Return(
		nil,
		errAPIError,
	)

	gen := NewPRBodyGenerator(mockProvider, nil, nil, nil, "", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "file.go", ChangeType: "modified"}},
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)

	require.ErrorIs(t, err, ErrFallbackUsed, "should return ErrFallbackUsed on AI error")
	assert.Contains(t, result, "What Changed", "should contain fallback template")
	mockProvider.AssertExpectations(t)
}

func TestPRBodyGenerator_GenerateBody_Timeout(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		// Wait for context to be canceled
		<-ctx.Done()
	}).Return(nil, context.DeadlineExceeded)

	gen := NewPRBodyGenerator(mockProvider, nil, nil, nil, "", 50*time.Millisecond, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "file.go", ChangeType: "modified"}},
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)

	require.ErrorIs(t, err, ErrFallbackUsed, "should return ErrFallbackUsed on timeout")
	assert.Contains(t, result, "What Changed", "should contain fallback template")
}

func TestPRBodyGenerator_GenerateBody_GuidelinesInjection(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)

	// Valid PR body format (multiline with ## headers)
	validPRBody := "## What Changed\n* Updated files\n\n## Why It Was Necessary\n* Keep sync"

	var capturedRequest *GenerateRequest
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		capturedRequest = args.Get(1).(*GenerateRequest)
	}).Return(&GenerateResponse{Content: validPRBody}, nil)

	gen := NewPRBodyGenerator(mockProvider, nil, nil, nil, "Custom Guidelines Here", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		// PRGuidelines is empty, should use generator's guidelines
	}

	_, err := gen.GenerateBody(context.Background(), prCtx)
	require.NoError(t, err)

	// Verify guidelines were used in prompt WITHOUT modifying the original context
	assert.Empty(t, prCtx.PRGuidelines, "original context should not be modified")
	assert.Contains(t, capturedRequest.Prompt, "Custom Guidelines Here", "generator's guidelines should be used in prompt")
}

func TestPRBodyGenerator_GenerateBody_ExistingGuidelines(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)

	// Valid PR body format (multiline with ## headers)
	validPRBody := "## What Changed\n* Updated files\n\n## Why It Was Necessary\n* Keep sync"

	var capturedRequest *GenerateRequest
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		capturedRequest = args.Get(1).(*GenerateRequest)
	}).Return(&GenerateResponse{Content: validPRBody}, nil)

	gen := NewPRBodyGenerator(mockProvider, nil, nil, nil, "Generator Guidelines", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		PRGuidelines: "Existing Context Guidelines", // Already has guidelines
	}

	_, err := gen.GenerateBody(context.Background(), prCtx)
	require.NoError(t, err)

	// Existing guidelines should be preserved
	assert.Equal(t, "Existing Context Guidelines", prCtx.PRGuidelines)
	assert.Contains(t, capturedRequest.Prompt, "Existing Context Guidelines")
	assert.NotContains(t, capturedRequest.Prompt, "Generator Guidelines")
}

func TestPRBodyGenerator_StaticFallback(t *testing.T) {
	gen := NewPRBodyGenerator(nil, nil, nil, nil, "", 5*time.Second, nil)

	tests := []struct {
		name       string
		sourceRepo string
		targetRepo string
		fileCount  int
	}{
		{"standard sync", "owner/source", "owner/target", 5},
		{"single file", "org/repo", "org/other", 1},
		{"no files", "a/b", "c/d", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.staticFallback(tt.sourceRepo, tt.targetRepo, tt.fileCount)

			assert.Contains(t, result, "## What Changed")
			assert.Contains(t, result, "## Why It Was Necessary")
			assert.Contains(t, result, "## Testing Performed")
			assert.Contains(t, result, "## Impact / Risk")
			assert.Contains(t, result, tt.sourceRepo)
			assert.Contains(t, result, tt.targetRepo)
			assert.Contains(t, result, "file(s)")
		})
	}
}

func TestPRBodyGenerator_GenerateFallback_NilContext(t *testing.T) {
	gen := NewPRBodyGenerator(nil, nil, nil, nil, "", 5*time.Second, nil)

	result := gen.generateFallback(nil)

	assert.Contains(t, result, "unknown")
	assert.Contains(t, result, "0 file(s)")
}

func TestPRBodyGenerator_WithCacheError(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)

	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Return(
		nil,
		errAIGenerationFail,
	)

	gen := NewPRBodyGenerator(mockProvider, cache, nil, nil, "", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "file.go", ChangeType: "modified"}},
		DiffSummary:  "some diff",
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)

	require.ErrorIs(t, err, ErrFallbackUsed, "cache path should return ErrFallbackUsed on error")
	assert.Contains(t, result, "What Changed")
}

// Tests for BuildPRPrompt

func TestBuildPRPrompt(t *testing.T) {
	t.Run("with full context", func(t *testing.T) {
		ctx := &PRContext{
			SourceRepo: "owner/source",
			TargetRepo: "owner/target",
			CommitSHA:  "abc123def456",
			ChangedFiles: []FileChange{
				{Path: "README.md", ChangeType: "modified", LinesAdded: 10, LinesRemoved: 5},
				{Path: "config.yaml", ChangeType: "added", LinesAdded: 20, LinesRemoved: 0},
			},
			DiffSummary:  "diff --git a/README.md\n+new line",
			PRGuidelines: "Custom guidelines here",
		}

		prompt := BuildPRPrompt(ctx)

		assert.Contains(t, prompt, "owner/source")
		assert.Contains(t, prompt, "owner/target")
		assert.Contains(t, prompt, "abc123def456")
		assert.Contains(t, prompt, "README.md")
		assert.Contains(t, prompt, "config.yaml")
		assert.Contains(t, prompt, "Custom guidelines here")
		assert.Contains(t, prompt, "2 files")
	})

	t.Run("nil context returns empty", func(t *testing.T) {
		prompt := BuildPRPrompt(nil)
		assert.Empty(t, prompt)
	})

	t.Run("empty context uses defaults", func(t *testing.T) {
		ctx := &PRContext{}
		prompt := BuildPRPrompt(ctx)

		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "0 files")
	})
}

// Tests for error helper functions

func TestGenerationError(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		err := GenerationError("anthropic", "PR body", errAPIFailed)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AI generation failed")
		assert.Contains(t, err.Error(), "anthropic")
		assert.Contains(t, err.Error(), "PR body")
		assert.Contains(t, err.Error(), "API failed")
	})

	t.Run("with nil error returns nil", func(t *testing.T) {
		err := GenerationError("anthropic", "PR body", nil)
		assert.NoError(t, err)
	})
}

func TestProviderError(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		err := ProviderError("openai", "initialize", errConnectionFailed)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AI provider error")
		assert.Contains(t, err.Error(), "openai")
		assert.Contains(t, err.Error(), "initialize")
		assert.Contains(t, err.Error(), "connection failed")
	})

	t.Run("with nil error returns nil", func(t *testing.T) {
		err := ProviderError("openai", "initialize", nil)
		assert.NoError(t, err)
	})
}

func TestRateLimitError(t *testing.T) {
	err := RateLimitError("anthropic", "60 seconds")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AI provider error")
	assert.Contains(t, err.Error(), "anthropic")
	assert.Contains(t, err.Error(), "rate limit")
	assert.Contains(t, err.Error(), "60 seconds")
}

func TestConfigError(t *testing.T) {
	err := ConfigError("temperature", "must be between 0.0 and 2.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AI configuration error")
	assert.Contains(t, err.Error(), "temperature")
	assert.Contains(t, err.Error(), "must be between 0.0 and 2.0")
}

// Tests for mock helper functions

func TestNewSuccessMock(t *testing.T) {
	mockProvider := NewSuccessMock("Generated content here")

	assert.True(t, mockProvider.IsAvailable())
	assert.Equal(t, "mock", mockProvider.Name())

	resp, err := mockProvider.GenerateText(context.Background(), &GenerateRequest{
		Prompt: "test prompt",
	})
	require.NoError(t, err)
	assert.Equal(t, "Generated content here", resp.Content)
}

func TestNewUnavailableMock(t *testing.T) {
	mockProvider := NewUnavailableMock()

	assert.False(t, mockProvider.IsAvailable())
	assert.Equal(t, "mock", mockProvider.Name())
}

func TestNewErrorMock(t *testing.T) {
	mockProvider := NewErrorMock(errTestError)

	assert.True(t, mockProvider.IsAvailable())

	resp, err := mockProvider.GenerateText(context.Background(), &GenerateRequest{
		Prompt: "test prompt",
	})
	assert.Nil(t, resp)
	assert.Equal(t, errTestError, err)
}

func TestNewRateLimitMock(t *testing.T) {
	mockProvider := NewRateLimitMock("success after retry")

	assert.True(t, mockProvider.IsAvailable())

	// First call should fail with rate limit
	resp1, err1 := mockProvider.GenerateText(context.Background(), &GenerateRequest{Prompt: "test"})
	require.Error(t, err1)
	assert.Nil(t, resp1)
	assert.Contains(t, err1.Error(), "rate limit")

	// Second call should succeed
	resp2, err2 := mockProvider.GenerateText(context.Background(), &GenerateRequest{Prompt: "test"})
	require.NoError(t, err2)
	assert.Equal(t, "success after retry", resp2.Content)
}

func TestNewTimeoutMock(t *testing.T) {
	mockProvider := NewTimeoutMock()

	assert.True(t, mockProvider.IsAvailable())

	resp, err := mockProvider.GenerateText(context.Background(), &GenerateRequest{Prompt: "test"})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrGenerationTimeout)
}

func TestNewEmptyResponseMock(t *testing.T) {
	mockProvider := NewEmptyResponseMock()

	assert.True(t, mockProvider.IsAvailable())

	resp, err := mockProvider.GenerateText(context.Background(), &GenerateRequest{Prompt: "test"})
	require.NoError(t, err)
	assert.Empty(t, resp.Content)
}

// Tests for LoadPRGuidelines

func TestLoadPRGuidelines(t *testing.T) {
	t.Run("nonexistent directory returns fallback", func(t *testing.T) {
		result := LoadPRGuidelines("/nonexistent/path")
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "What Changed")
	})

	t.Run("returns fallback content", func(t *testing.T) {
		// With non-existent path, should return default template
		result := LoadPRGuidelines(".")
		// Either from file or fallback
		assert.NotEmpty(t, result)
	})
}

// Test generator with pre-built mocks

func TestPRBodyGenerator_WithSuccessMock(t *testing.T) {
	mockProvider := NewSuccessMock("## What Changed\n\nAI generated content")
	gen := NewPRBodyGenerator(mockProvider, nil, nil, nil, "", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "file.go", ChangeType: "modified"}},
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)
	require.NoError(t, err)
	assert.Contains(t, result, "AI generated content")
}

func TestPRBodyGenerator_WithUnavailableMock(t *testing.T) {
	mockProvider := NewUnavailableMock()
	gen := NewPRBodyGenerator(mockProvider, nil, nil, nil, "", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "file.go", ChangeType: "modified"}},
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)
	require.ErrorIs(t, err, ErrFallbackUsed, "should return ErrFallbackUsed when unavailable")
	assert.Contains(t, result, "What Changed") // Should use fallback
}

func TestPRBodyGenerator_WithErrorMock(t *testing.T) {
	mockProvider := NewErrorMock(errAPIError)
	gen := NewPRBodyGenerator(mockProvider, nil, nil, nil, "", 5*time.Second, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "file.go", ChangeType: "modified"}},
	}

	result, err := gen.GenerateBody(context.Background(), prCtx)
	require.ErrorIs(t, err, ErrFallbackUsed, "should return ErrFallbackUsed on error")
	assert.Contains(t, result, "What Changed")
}

// Tests for factory.go validation paths

func TestNewProvider_ValidationErrors(t *testing.T) {
	t.Run("disabled config returns error", func(t *testing.T) {
		cfg := &Config{
			Enabled: false,
			APIKey:  "test-key",
		}
		provider, err := NewProvider(context.Background(), cfg, nil)
		assert.Nil(t, provider)
		assert.ErrorIs(t, err, ErrProviderNotConfigured)
	})

	t.Run("missing API key returns error", func(t *testing.T) {
		cfg := &Config{
			Enabled:  true,
			APIKey:   "",
			Provider: ProviderAnthropic,
		}
		provider, err := NewProvider(context.Background(), cfg, nil)
		assert.Nil(t, provider)
		assert.ErrorIs(t, err, ErrAPIKeyMissing)
	})

	t.Run("unsupported provider returns error", func(t *testing.T) {
		cfg := &Config{
			Enabled:  true,
			APIKey:   "test-key",
			Provider: "unsupported-provider",
		}
		provider, err := NewProvider(context.Background(), cfg, nil)
		assert.Nil(t, provider)
		assert.ErrorIs(t, err, ErrUnsupportedProvider)
	})
}

func TestMustNewProvider_Panics(t *testing.T) {
	t.Run("panics on disabled config", func(t *testing.T) {
		cfg := &Config{
			Enabled: false,
		}
		assert.Panics(t, func() {
			MustNewProvider(context.Background(), cfg, nil)
		})
	})
}

// Test SetupGenerateTextOnce

func TestSetupGenerateTextOnce(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.SetupAvailable(true)
	mockProvider.SetupName("mock")
	mockProvider.SetupGenerateTextOnce(&GenerateResponse{Content: "first response"}, nil)
	mockProvider.SetupGenerateTextOnce(&GenerateResponse{Content: "second response"}, nil)

	// First call
	resp1, err1 := mockProvider.GenerateText(context.Background(), &GenerateRequest{Prompt: "test"})
	require.NoError(t, err1)
	assert.Equal(t, "first response", resp1.Content)

	// Second call
	resp2, err2 := mockProvider.GenerateText(context.Background(), &GenerateRequest{Prompt: "test"})
	require.NoError(t, err2)
	assert.Equal(t, "second response", resp2.Content)
}
