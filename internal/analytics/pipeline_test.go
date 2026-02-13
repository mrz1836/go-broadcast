package analytics

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

var (
	errAPIError               = errors.New("API error")
	errGraphQLError           = errors.New("GraphQL error")
	errQueryTooComplex        = errors.New("query is too complex")
	errQueryCostExceeded      = errors.New("query cost exceeded")
	errComplexityLimitReached = errors.New("complexity limit reached")
	errNetworkTimeout         = errors.New("network timeout")
)

func TestNewPipeline(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()

	pipeline := NewPipeline(mockClient, nil, logger)
	require.NotNil(t, pipeline)
	assert.NotNil(t, pipeline.ghClient)
	assert.NotNil(t, pipeline.logger)
}

func TestSyncOrganization(t *testing.T) {
	ctx := context.Background()

	t.Run("successful sync with multiple repos", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		logger := logrus.New()
		logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests

		pipeline := NewPipeline(mockClient, nil, logger)

		// Mock discovery response
		desc1 := "Repo 1"
		lang1 := "Go"
		repos := []gh.RepoInfo{
			{
				Name:     "repo1",
				FullName: "test-org/repo1",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "test-org"},
				Description: &desc1,
				Language:    &lang1,
			},
		}

		mockClient.On("DiscoverOrgRepos", ctx, "test-org").Return(repos, nil)

		// Mock GraphQL response
		graphQLData := map[string]interface{}{
			"repo0": map[string]interface{}{
				"nameWithOwner":  "test-org/repo1",
				"stargazerCount": float64(10),
				"forkCount":      float64(2),
			},
		}

		mockClient.On("ExecuteGraphQL", ctx, mock.Anything).Return(graphQLData, nil)

		result, err := pipeline.SyncOrganization(ctx, "test-org")
		require.NoError(t, err)
		require.Len(t, result, 1)

		metadata := result["test-org/repo1"]
		require.NotNil(t, metadata)
		assert.Equal(t, 10, metadata.Stars)
		assert.Equal(t, 2, metadata.Forks)

		mockClient.AssertExpectations(t)
	})

	t.Run("empty organization", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		pipeline := NewPipeline(mockClient, nil, nil)

		mockClient.On("DiscoverOrgRepos", ctx, "empty-org").Return([]gh.RepoInfo{}, nil)

		result, err := pipeline.SyncOrganization(ctx, "empty-org")
		require.NoError(t, err)
		assert.Empty(t, result)

		mockClient.AssertExpectations(t)
	})

	t.Run("discovery error", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		pipeline := NewPipeline(mockClient, nil, nil)

		mockClient.On("DiscoverOrgRepos", ctx, "error-org").Return(nil, errAPIError)

		result, err := pipeline.SyncOrganization(ctx, "error-org")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to discover repos")

		mockClient.AssertExpectations(t)
	})
}

func TestSyncRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("successful single repo sync", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		pipeline := NewPipeline(mockClient, nil, nil)

		// Mock GraphQL response for single repo
		graphQLData := map[string]interface{}{
			"repo0": map[string]interface{}{
				"nameWithOwner":  "owner/repo",
				"stargazerCount": float64(42),
				"forkCount":      float64(5),
				"description":    "Test repository",
			},
		}

		mockClient.On("ExecuteGraphQL", ctx, mock.Anything).Return(graphQLData, nil)

		result, err := pipeline.SyncRepository(ctx, "owner", "repo")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "owner/repo", result.FullName)
		assert.Equal(t, 42, result.Stars)
		assert.Equal(t, 5, result.Forks)
		assert.Equal(t, "Test repository", result.Description)

		mockClient.AssertExpectations(t)
	})

	t.Run("GraphQL error", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		pipeline := NewPipeline(mockClient, nil, nil)

		mockClient.On("ExecuteGraphQL", ctx, mock.Anything).
			Return(nil, errGraphQLError)

		result, err := pipeline.SyncRepository(ctx, "owner", "error-repo")
		require.Error(t, err)
		assert.Nil(t, result)

		mockClient.AssertExpectations(t)
	})
}

func TestCollectMetadata_Batching(t *testing.T) {
	ctx := context.Background()

	t.Run("splits large repo list into batches", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		logger := logrus.New()
		logger.SetLevel(logrus.WarnLevel)
		pipeline := NewPipeline(mockClient, nil, logger)

		// Create 52 repos (should result in 3 batches: 25, 25, 2)
		repos := make([]gh.RepoInfo, 52)
		for i := range repos {
			repos[i] = gh.RepoInfo{
				Name:     "repo",
				FullName: "org/repo",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "org"},
			}
		}

		mockClient.On("DiscoverOrgRepos", ctx, "big-org").Return(repos, nil)

		// Mock should receive 3 GraphQL calls
		graphQLData := map[string]interface{}{}
		mockClient.On("ExecuteGraphQL", ctx, mock.Anything).
			Return(graphQLData, nil).Times(3)

		_, err := pipeline.SyncOrganization(ctx, "big-org")
		require.NoError(t, err)

		mockClient.AssertExpectations(t)
	})
}

func TestIsComplexityError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "complexity error",
			err:      errQueryTooComplex,
			expected: true,
		},
		{
			name:     "query cost error",
			err:      errQueryCostExceeded,
			expected: true,
		},
		{
			name:     "complexity limit",
			err:      errComplexityLimitReached,
			expected: true,
		},
		{
			name:     "other error",
			err:      errNetworkTimeout,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isComplexityError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
