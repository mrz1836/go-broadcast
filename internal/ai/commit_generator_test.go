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
	errCommitAPIError     = errors.New("API error")
	errCommitAIGeneration = errors.New("AI generation failed")
)

func TestNewCommitMessageGenerator(t *testing.T) {
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

		gen := NewCommitMessageGenerator(mockProvider, cache, truncator, retryConfig, 30*time.Second, logger)

		require.NotNil(t, gen)
		assert.Equal(t, mockProvider, gen.provider)
		assert.Equal(t, cache, gen.cache)
		assert.Equal(t, truncator, gen.truncator)
		assert.Equal(t, 30*time.Second, gen.timeout)
	})

	t.Run("with nil parameters uses defaults", func(t *testing.T) {
		gen := NewCommitMessageGenerator(nil, nil, nil, nil, 0, nil)

		require.NotNil(t, gen)
		assert.Nil(t, gen.provider)
		assert.Nil(t, gen.cache)
		assert.NotNil(t, gen.retryConfig)
		assert.Equal(t, 30*time.Second, gen.timeout)
		assert.NotNil(t, gen.logger)
	})
}

func TestCommitMessageGenerator_GenerateMessage_AISuccess(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Return(
		&GenerateResponse{Content: "sync: update workflow files for CI"},
		nil,
	)

	gen := NewCommitMessageGenerator(mockProvider, nil, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		ChangedFiles: []FileChange{
			{Path: ".github/workflows/ci.yml", ChangeType: "modified"},
		},
		DiffSummary: "diff content",
	}

	result, err := gen.GenerateMessage(context.Background(), commitCtx)

	require.NoError(t, err)
	assert.Equal(t, "sync: update workflow files for CI", result)
	mockProvider.AssertExpectations(t)
}

func TestCommitMessageGenerator_GenerateMessage_AIResponseValidated(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	// AI returns message that needs validation (e.g., wrong prefix)
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Return(
		&GenerateResponse{Content: "chore: update config files."},
		nil,
	)

	gen := NewCommitMessageGenerator(mockProvider, nil, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "config.yaml", ChangeType: "modified"}},
	}

	result, err := gen.GenerateMessage(context.Background(), commitCtx)

	require.NoError(t, err)
	// Validator should convert chore: to sync: and remove trailing period
	assert.Equal(t, "sync: update config files", result)
}

func TestCommitMessageGenerator_GenerateMessage_CacheHit(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)

	// Pre-populate cache with a valid commit message
	cache.Set("diff content", "sync: cached commit message")

	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	// GenerateText should NOT be called due to cache hit

	gen := NewCommitMessageGenerator(mockProvider, cache, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo:  "owner/source",
		TargetRepo:  "owner/target",
		DiffSummary: "diff content", // Same as cached
	}

	result, err := gen.GenerateMessage(context.Background(), commitCtx)

	require.NoError(t, err)
	assert.Equal(t, "sync: cached commit message", result)
	mockProvider.AssertNotCalled(t, "GenerateText", mock.Anything, mock.Anything)
}

func TestCommitMessageGenerator_GenerateMessage_ProviderNil(t *testing.T) {
	gen := NewCommitMessageGenerator(nil, nil, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "README.md", ChangeType: "modified"}},
	}

	result, err := gen.GenerateMessage(context.Background(), commitCtx)

	require.NoError(t, err)
	assert.Equal(t, "sync: update README.md from source repository", result)
}

func TestCommitMessageGenerator_GenerateMessage_ProviderUnavailable(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(false)

	gen := NewCommitMessageGenerator(mockProvider, nil, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "README.md", ChangeType: "modified"}},
	}

	result, err := gen.GenerateMessage(context.Background(), commitCtx)

	require.NoError(t, err)
	assert.Equal(t, "sync: update README.md from source repository", result)
	mockProvider.AssertNotCalled(t, "GenerateText", mock.Anything, mock.Anything)
}

func TestCommitMessageGenerator_GenerateMessage_AIError(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Return(
		nil,
		errCommitAPIError,
	)

	gen := NewCommitMessageGenerator(mockProvider, nil, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		ChangedFiles: []FileChange{
			{Path: "file1.go", ChangeType: "modified"},
			{Path: "file2.go", ChangeType: "modified"},
		},
	}

	result, err := gen.GenerateMessage(context.Background(), commitCtx)

	require.NoError(t, err, "should not return error - falls back gracefully")
	assert.Equal(t, "sync: update 2 files from source repository", result)
	mockProvider.AssertExpectations(t)
}

func TestCommitMessageGenerator_GenerateMessage_EmptyAfterValidation(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	// AI returns only whitespace which becomes empty after validation
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Return(
		&GenerateResponse{Content: "   \n\t  "},
		nil,
	)

	gen := NewCommitMessageGenerator(mockProvider, nil, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		ChangedFiles: []FileChange{
			{Path: "file1.go", ChangeType: "modified"},
			{Path: "file2.go", ChangeType: "modified"},
			{Path: "file3.go", ChangeType: "modified"},
		},
	}

	result, err := gen.GenerateMessage(context.Background(), commitCtx)

	require.NoError(t, err, "should fall back when validation returns empty")
	assert.Equal(t, "sync: update 3 files from source repository", result)
}

func TestCommitMessageGenerator_GenerateMessage_Timeout(t *testing.T) {
	mockProvider := NewMockProvider()
	mockProvider.On("IsAvailable").Return(true)
	mockProvider.On("GenerateText", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		<-ctx.Done() // Wait for context cancellation
	}).Return(nil, context.DeadlineExceeded)

	gen := NewCommitMessageGenerator(mockProvider, nil, nil, nil, 50*time.Millisecond, nil)

	commitCtx := &CommitContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{{Path: "file.go", ChangeType: "modified"}},
	}

	result, err := gen.GenerateMessage(context.Background(), commitCtx)

	require.NoError(t, err, "should fall back on timeout")
	assert.Contains(t, result, "sync:")
}

func TestCommitMessageGenerator_Fallback_SingleFile(t *testing.T) {
	gen := NewCommitMessageGenerator(nil, nil, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		ChangedFiles: []FileChange{
			{Path: "README.md", ChangeType: "modified"},
		},
	}

	result := gen.generateFallback(commitCtx)

	assert.Equal(t, "sync: update README.md from source repository", result)
}

func TestCommitMessageGenerator_Fallback_MultipleFiles(t *testing.T) {
	gen := NewCommitMessageGenerator(nil, nil, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		ChangedFiles: []FileChange{
			{Path: "file1.go", ChangeType: "modified"},
			{Path: "file2.go", ChangeType: "added"},
			{Path: "file3.go", ChangeType: "deleted"},
		},
	}

	result := gen.generateFallback(commitCtx)

	assert.Equal(t, "sync: update 3 files from source repository", result)
}

func TestCommitMessageGenerator_Fallback_NoFiles(t *testing.T) {
	gen := NewCommitMessageGenerator(nil, nil, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: []FileChange{},
	}

	result := gen.generateFallback(commitCtx)

	assert.Equal(t, "sync: update files from source repository", result)
}

func TestCommitMessageGenerator_Fallback_NilContext(t *testing.T) {
	gen := NewCommitMessageGenerator(nil, nil, nil, nil, 5*time.Second, nil)

	result := gen.generateFallback(nil)

	assert.Equal(t, "sync: update files from source repository", result)
}

func TestCommitMessageGenerator_WithCacheError(t *testing.T) {
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
		errCommitAIGeneration,
	)

	gen := NewCommitMessageGenerator(mockProvider, cache, nil, nil, 5*time.Second, nil)

	commitCtx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		ChangedFiles: []FileChange{
			{Path: "file.go", ChangeType: "modified"},
		},
		DiffSummary: "some diff",
	}

	result, err := gen.GenerateMessage(context.Background(), commitCtx)

	require.NoError(t, err, "cache path should also fall back gracefully")
	assert.Equal(t, "sync: update file.go from source repository", result)
}

// Tests for BuildCommitPrompt

func TestBuildCommitPrompt(t *testing.T) {
	t.Run("with full context", func(t *testing.T) {
		ctx := &CommitContext{
			SourceRepo: "owner/source",
			TargetRepo: "owner/target",
			ChangedFiles: []FileChange{
				{Path: "README.md", ChangeType: "modified"},
				{Path: "config.yaml", ChangeType: "added"},
			},
			DiffSummary: "diff --git a/README.md\n+new line",
			GroupName:   "docs",
		}

		prompt := BuildCommitPrompt(ctx)

		assert.Contains(t, prompt, "owner/source")
		assert.Contains(t, prompt, "owner/target")
		assert.Contains(t, prompt, "README.md")
		assert.Contains(t, prompt, "config.yaml")
		assert.Contains(t, prompt, "docs")
		assert.Contains(t, prompt, "sync")
	})

	t.Run("nil context returns empty", func(t *testing.T) {
		prompt := BuildCommitPrompt(nil)
		assert.Empty(t, prompt)
	})

	t.Run("empty context works", func(t *testing.T) {
		ctx := &CommitContext{}
		prompt := BuildCommitPrompt(ctx)

		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "sync")
	})
}
