//go:build integration
// +build integration

package gh

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubClient_Integration(t *testing.T) {
	// Skip if no GitHub token is available
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN not set, skipping integration test")
	}

	client, err := NewClient(logrus.New())
	require.NoError(t, err)

	ctx := context.Background()

	// Test with a well-known public repository
	repo := "github/gitignore"

	t.Run("ListBranches", func(t *testing.T) {
		branches, err := client.ListBranches(ctx, repo)
		require.NoError(t, err)
		assert.NotEmpty(t, branches)

		// Should have at least main/master branch
		var hasMainBranch bool
		for _, branch := range branches {
			if branch.Name == "main" || branch.Name == "master" {
				hasMainBranch = true
				break
			}
		}
		assert.True(t, hasMainBranch, "Repository should have main or master branch")
	})

	t.Run("GetBranch", func(t *testing.T) {
		branch, err := client.GetBranch(ctx, repo, "main")
		require.NoError(t, err)
		require.NotNil(t, branch)
		assert.Equal(t, "main", branch.Name)
	})

	t.Run("GetFile", func(t *testing.T) {
		file, err := client.GetFile(ctx, repo, "README.md", "main")
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
