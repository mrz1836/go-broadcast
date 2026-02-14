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
	errGraphQLError           = errors.New("GraphQL error")
	errQueryTooComplex        = errors.New("query is too complex")
	errQueryCostExceeded      = errors.New("query cost exceeded")
	errComplexityLimitReached = errors.New("complexity limit reached")
	errNetworkTimeout         = errors.New("network timeout")
)

func TestNewPipeline(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()

	pipeline := NewPipeline(mockClient, nil, nil, nil, logger)
	require.NotNil(t, pipeline)
	assert.NotNil(t, pipeline.ghClient)
	assert.NotNil(t, pipeline.logger)
}

// TestSyncOrganization tests have been removed as they need to be rewritten
// to test the new database-backed logic instead of GitHub API discovery.
// The new SyncOrganization method queries configured repos from the database,
// not from GitHub directly.

func TestSyncRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("successful single repo sync", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		pipeline := NewPipeline(mockClient, nil, nil, nil, nil)

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
		pipeline := NewPipeline(mockClient, nil, nil, nil, nil)

		mockClient.On("ExecuteGraphQL", ctx, mock.Anything).
			Return(nil, errGraphQLError)

		result, err := pipeline.SyncRepository(ctx, "owner", "error-repo")
		require.Error(t, err)
		assert.Nil(t, result)

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
