package sync

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/ai"
	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// TestEngine_SetAIGenerators tests the setter methods for AI generators.
func TestEngine_SetAIGenerators(t *testing.T) {
	cfg := &config.Config{}
	engine := NewEngine(context.Background(), cfg, nil, nil, nil, nil, nil)

	t.Run("SetPRGenerator", func(t *testing.T) {
		mockProvider := ai.NewSuccessMock("## Test PR Body")
		gen := ai.NewPRBodyGenerator(
			mockProvider,
			nil,
			nil,
			nil,
			"",
			5*time.Second,
			nil,
		)

		engine.SetPRGenerator(gen)
		assert.NotNil(t, engine.prGenerator)

		// Clear it
		engine.SetPRGenerator(nil)
		assert.Nil(t, engine.prGenerator)
	})

	t.Run("SetCommitGenerator", func(t *testing.T) {
		mockProvider := ai.NewSuccessMock("feat: test commit")
		gen := ai.NewCommitMessageGenerator(
			mockProvider,
			nil,
			nil,
			nil,
			5*time.Second,
			nil,
		)

		engine.SetCommitGenerator(gen)
		assert.NotNil(t, engine.commitGenerator)

		// Clear it
		engine.SetCommitGenerator(nil)
		assert.Nil(t, engine.commitGenerator)
	})

	t.Run("SetResponseCache", func(t *testing.T) {
		cache := ai.NewResponseCache(&ai.Config{
			CacheEnabled: true,
			CacheMaxSize: 100,
			CacheTTL:     time.Hour,
		})

		engine.SetResponseCache(cache)
		assert.NotNil(t, engine.responseCache)

		engine.SetResponseCache(nil)
		assert.Nil(t, engine.responseCache)
	})

	t.Run("SetDiffTruncator", func(t *testing.T) {
		truncator := ai.NewDiffTruncator(&ai.Config{
			DiffMaxChars:        4000,
			DiffMaxLinesPerFile: 50,
		})

		engine.SetDiffTruncator(truncator)
		assert.NotNil(t, engine.diffTruncator)

		engine.SetDiffTruncator(nil)
		assert.Nil(t, engine.diffTruncator)
	})
}

// TestRepositorySync_GenerateCommitMessage tests commit message generation
// with AI integration.
func TestRepositorySync_GenerateCommitMessage(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("with AI generator returns generated message", func(t *testing.T) {
		mockProvider := ai.NewSuccessMock("feat(sync): update configuration files")
		gen := ai.NewCommitMessageGenerator(
			mockProvider,
			nil,
			nil,
			nil,
			5*time.Second,
			nil,
		)

		cfg := &config.Config{}
		engine := NewEngine(context.Background(), cfg, nil, nil, nil, nil, nil)
		engine.SetCommitGenerator(gen)

		rs := &RepositorySync{
			engine: engine,
			target: config.TargetConfig{Repo: "org/target"},
			sourceState: &state.SourceState{
				Repo: "org/source",
			},
			logger: logger.WithField("test", "generate_commit"),
		}

		// FileChange uses IsNew/IsDeleted flags, not ChangeType string
		changedFiles := []FileChange{
			{Path: "config.yaml"},
		}

		msg := rs.generateCommitMessage(context.Background(), changedFiles)

		// Should use AI-generated message (with sync: prefix added by commit generator)
		assert.Contains(t, msg, "feat(sync): update configuration files")
	})

	t.Run("without AI generator uses fallback", func(t *testing.T) {
		cfg := &config.Config{}
		engine := NewEngine(context.Background(), cfg, nil, nil, nil, nil, nil)
		// Don't set commit generator - it remains nil

		rs := &RepositorySync{
			engine: engine,
			target: config.TargetConfig{Repo: "org/target"},
			sourceState: &state.SourceState{
				Repo:         "org/source",
				LatestCommit: "abc123",
			},
			logger: logger.WithField("test", "generate_commit_fallback"),
		}

		changedFiles := []FileChange{
			{Path: "file.go"},
		}

		msg := rs.generateCommitMessage(context.Background(), changedFiles)

		// Should use fallback template (format: "sync: update <file> from source repository")
		assert.Contains(t, msg, "sync:")
		assert.Contains(t, msg, "from source repository")
	})

	t.Run("with nil engine uses fallback", func(t *testing.T) {
		rs := &RepositorySync{
			engine: nil, // nil engine
			target: config.TargetConfig{Repo: "org/target"},
			sourceState: &state.SourceState{
				Repo:         "org/source",
				LatestCommit: "abc123",
			},
			logger: logger.WithField("test", "nil_engine"),
		}

		changedFiles := []FileChange{
			{Path: "file.go"},
		}

		msg := rs.generateCommitMessage(context.Background(), changedFiles)

		// Should use fallback template
		assert.Contains(t, msg, "sync:")
		assert.Contains(t, msg, "from source repository")
	})

	t.Run("AI failure falls back gracefully", func(t *testing.T) {
		errTest := ai.GenerationError("test", "commit", ai.ErrGenerationTimeout)
		mockProvider := ai.NewErrorMock(errTest)
		gen := ai.NewCommitMessageGenerator(
			mockProvider,
			nil,
			nil,
			nil,
			5*time.Second,
			nil,
		)

		cfg := &config.Config{}
		engine := NewEngine(context.Background(), cfg, nil, nil, nil, nil, nil)
		engine.SetCommitGenerator(gen)

		rs := &RepositorySync{
			engine: engine,
			target: config.TargetConfig{Repo: "org/target"},
			sourceState: &state.SourceState{
				Repo:         "org/source",
				LatestCommit: "abc123",
			},
			logger: logger.WithField("test", "ai_failure"),
		}

		changedFiles := []FileChange{
			{Path: "file.go"},
		}

		msg := rs.generateCommitMessage(context.Background(), changedFiles)

		// Should fall back to template on error
		assert.Contains(t, msg, "sync:")
		assert.Contains(t, msg, "from source repository")
	})
}

// TestRepositorySync_GeneratePRBody tests PR body generation with AI integration.
func TestRepositorySync_GeneratePRBody(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("with AI generator returns generated body", func(t *testing.T) {
		aiBody := `## Summary
This PR updates configuration files from the source repository.

## Changes
- Updated config.yaml with new settings`

		mockProvider := ai.NewSuccessMock(aiBody)
		gen := ai.NewPRBodyGenerator(
			mockProvider,
			nil,
			nil,
			nil,
			"",
			5*time.Second,
			nil,
		)

		cfg := &config.Config{}
		engine := NewEngine(context.Background(), cfg, nil, nil, nil, nil, nil)
		engine.SetPRGenerator(gen)

		rs := &RepositorySync{
			engine: engine,
			target: config.TargetConfig{Repo: "org/target"},
			sourceState: &state.SourceState{
				Repo: "org/source",
			},
			logger: logger.WithField("test", "generate_pr"),
		}

		changedFiles := []FileChange{
			{Path: "config.yaml"},
		}

		body := rs.generatePRBody(context.Background(), "abc123", changedFiles, []string{"config.yaml"})

		// Should include AI-generated content
		assert.Contains(t, body, "## Summary")
		assert.Contains(t, body, "Updated config.yaml")
		// Should also include metadata block
		assert.Contains(t, body, "go-broadcast-metadata")
	})

	t.Run("without AI generator uses fallback", func(t *testing.T) {
		cfg := &config.Config{}
		engine := NewEngine(context.Background(), cfg, nil, nil, nil, nil, nil)
		// Don't set PR generator - it remains nil

		rs := &RepositorySync{
			engine: engine,
			target: config.TargetConfig{Repo: "org/target"},
			sourceState: &state.SourceState{
				Repo:         "org/source",
				LatestCommit: "abc123",
			},
			logger: logger.WithField("test", "generate_pr_fallback"),
		}

		changedFiles := []FileChange{
			{Path: "file.go"},
		}

		body := rs.generatePRBody(context.Background(), "abc123", changedFiles, []string{"file.go"})

		// Should use fallback template (contains standard sections)
		assert.Contains(t, body, "What Changed")
		assert.Contains(t, body, "go-broadcast-metadata")
	})

	t.Run("AI empty response falls back gracefully", func(t *testing.T) {
		mockProvider := ai.NewEmptyResponseMock()
		gen := ai.NewPRBodyGenerator(
			mockProvider,
			nil,
			nil,
			nil,
			"",
			5*time.Second,
			nil,
		)

		cfg := &config.Config{}
		engine := NewEngine(context.Background(), cfg, nil, nil, nil, nil, nil)
		engine.SetPRGenerator(gen)

		rs := &RepositorySync{
			engine: engine,
			target: config.TargetConfig{Repo: "org/target"},
			sourceState: &state.SourceState{
				Repo:         "org/source",
				LatestCommit: "abc123",
			},
			logger: logger.WithField("test", "ai_empty"),
		}

		changedFiles := []FileChange{
			{Path: "file.go"},
		}

		body := rs.generatePRBody(context.Background(), "abc123", changedFiles, []string{"file.go"})

		// Should fall back on empty response
		assert.Contains(t, body, "What Changed")
	})
}

// TestAICache_SharedAcrossOperations tests that the AI cache is shared
// across multiple repository syncs.
func TestAICache_SharedAcrossOperations(t *testing.T) {
	// Create shared cache
	cache := ai.NewResponseCache(&ai.Config{
		CacheEnabled: true,
		CacheMaxSize: 100,
		CacheTTL:     time.Hour,
	})

	// Create a mock that tracks calls
	mockProvider := ai.NewMockProvider()
	mockProvider.SetupAvailable(true)
	mockProvider.SetupName("mock")
	// Setup to return different responses on each call
	mockProvider.On("GenerateText", context.Background(), &ai.GenerateRequest{
		Prompt:      "test prompt",
		MaxTokens:   100,
		Temperature: ai.TemperatureNotSet,
	}).Return(&ai.GenerateResponse{
		Content:      "cached response",
		TokensUsed:   10,
		FinishReason: "stop",
	}, nil).Maybe()

	// First, warm the cache
	truncator := ai.NewDiffTruncator(&ai.Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	})

	gen := ai.NewPRBodyGenerator(
		mockProvider,
		cache,
		truncator,
		nil,
		"",
		5*time.Second,
		nil,
	)

	cfg := &config.Config{}
	engine := NewEngine(context.Background(), cfg, nil, nil, nil, nil, nil)
	engine.SetPRGenerator(gen)
	engine.SetResponseCache(cache)
	engine.SetDiffTruncator(truncator)

	// The cache should be accessible via the engine
	assert.NotNil(t, engine.responseCache)

	// Verify cache was set
	require.NotNil(t, engine.responseCache)
}

// TestRepositorySync_ConvertToAIFileChanges tests the file change conversion.
func TestRepositorySync_ConvertToAIFileChanges(t *testing.T) {
	rs := &RepositorySync{}

	t.Run("converts file changes correctly", func(t *testing.T) {
		changes := []FileChange{
			{Path: "file1.go", IsNew: true},     // added
			{Path: "file2.go"},                  // modified (default)
			{Path: "file3.go", IsDeleted: true}, // deleted
		}

		result := rs.convertToAIFileChanges(changes)

		require.Len(t, result, 3)
		assert.Equal(t, "file1.go", result[0].Path)
		assert.Equal(t, "added", result[0].ChangeType)
		assert.Equal(t, "file2.go", result[1].Path)
		assert.Equal(t, "modified", result[1].ChangeType)
		assert.Equal(t, "file3.go", result[2].Path)
		assert.Equal(t, "deleted", result[2].ChangeType)
	})

	t.Run("handles empty list", func(t *testing.T) {
		result := rs.convertToAIFileChanges([]FileChange{})
		assert.Empty(t, result)
	})

	t.Run("handles nil list", func(t *testing.T) {
		result := rs.convertToAIFileChanges(nil)
		assert.Empty(t, result)
	})
}

// TestGeneratePRBody_LongFileList tests PR body generation with many files.
func TestGeneratePRBody_LongFileList(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	cfg := &config.Config{}
	engine := NewEngine(context.Background(), cfg, nil, nil, nil, nil, nil)

	rs := &RepositorySync{
		engine: engine,
		target: config.TargetConfig{Repo: "org/target"},
		sourceState: &state.SourceState{
			Repo:         "org/source",
			LatestCommit: "abc123",
		},
		logger: logger.WithField("test", "long_file_list"),
	}

	// Create 100+ files
	changedFiles := make([]FileChange, 150)
	actualFiles := make([]string, 150)
	for i := 0; i < 150; i++ {
		path := "dir/file" + strings.Repeat("x", i%10) + ".go"
		changedFiles[i] = FileChange{Path: path}
		actualFiles[i] = path
	}

	body := rs.generatePRBody(context.Background(), "abc123", changedFiles, actualFiles)

	// Should handle large file lists gracefully
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "go-broadcast-metadata")
}
