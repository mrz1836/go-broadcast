//go:build integration

package gh

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestGitHubClient_Integration(t *testing.T) {
	// Skip if no GitHub token is available
	token := os.Getenv("GH_PAT_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		t.Skip("GH_PAT_TOKEN or GITHUB_TOKEN not set, skipping integration test")
	}

	ctx := context.Background()

	client, err := NewClient(ctx, logrus.New(), &logging.LogConfig{})
	require.NoError(t, err)

	// Test with a well-known public repository
	repo := "github/gitignore"

	t.Run("ListBranches", func(t *testing.T) {
		branches, err := client.ListBranches(ctx, repo)
		require.NoError(t, err)
		assert.NotEmpty(t, branches)

		// Should have at least one main branch
		var hasMainBranch bool
		for _, branch := range branches {
			if isMainBranch(branch.Name) {
				hasMainBranch = true
				break
			}
		}
		assert.True(t, hasMainBranch, "Repository should have a main branch")
	})

	t.Run("GetBranch", func(t *testing.T) {
		branch, err := client.GetBranch(ctx, repo, "master")
		require.NoError(t, err)
		require.NotNil(t, branch)
		assert.Equal(t, "master", branch.Name)
	})

	t.Run("GetFile", func(t *testing.T) {
		file, err := client.GetFile(ctx, repo, "README.md", "master")
		require.NoError(t, err)
		require.NotNil(t, file)
		assert.Equal(t, "README.md", file.Path)
		assert.NotEmpty(t, file.Content)
		assert.Contains(t, string(file.Content), "gitignore")
	})

	t.Run("ListPRs", func(t *testing.T) {
		// Test listing closed PRs (less likely to change)
		prs, err := client.ListPRs(ctx, repo, "closed")
		require.NoError(t, err)
		assert.NotEmpty(t, prs)

		// Verify all returned PRs are closed
		for _, pr := range prs {
			assert.Equal(t, "closed", pr.State)
		}
	})
}
