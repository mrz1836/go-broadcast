package gh

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverOrgRepos(t *testing.T) {
	ctx := context.Background()

	t.Run("successful discovery with multiple repos", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		desc1 := "Test repo 1"
		lang1 := "Go"
		desc2 := "Test repo 2"
		lang2 := "Python"

		repos := []RepoInfo{
			{
				Name:     "repo1",
				FullName: "test-org/repo1",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "test-org"},
				Description:   &desc1,
				Language:      &lang1,
				Private:       false,
				Fork:          false,
				Archived:      false,
				DefaultBranch: "main",
				HTMLURL:       "https://github.com/test-org/repo1",
			},
			{
				Name:     "repo2",
				FullName: "test-org/repo2",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "test-org"},
				Description:   &desc2,
				Language:      &lang2,
				Private:       true,
				Fork:          false,
				Archived:      false,
				DefaultBranch: "master",
				HTMLURL:       "https://github.com/test-org/repo2",
			},
		}

		output, err := json.Marshal(repos)
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{"api", "orgs/test-org/repos?per_page=100&type=all", "--paginate"}).
			Return(output, nil)

		result, err := client.DiscoverOrgRepos(ctx, "test-org")
		require.NoError(t, err)
		assert.Len(t, result, 2)

		// Verify first repo fields
		assert.Equal(t, "repo1", result[0].Name)
		assert.Equal(t, "test-org/repo1", result[0].FullName)
		assert.Equal(t, "test-org", result[0].Owner.Login)
		assert.NotNil(t, result[0].Description)
		assert.Equal(t, "Test repo 1", *result[0].Description)
		assert.NotNil(t, result[0].Language)
		assert.Equal(t, "Go", *result[0].Language)
		assert.False(t, result[0].Private)
		assert.False(t, result[0].Fork)
		assert.False(t, result[0].Archived)
		assert.Equal(t, "main", result[0].DefaultBranch)

		mockRunner.AssertExpectations(t)
	})

	t.Run("empty organization", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		output, err := json.Marshal([]RepoInfo{})
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{"api", "orgs/empty-org/repos?per_page=100&type=all", "--paginate"}).
			Return(output, nil)

		result, err := client.DiscoverOrgRepos(ctx, "empty-org")
		require.NoError(t, err)
		assert.Empty(t, result)

		mockRunner.AssertExpectations(t)
	})

	t.Run("organization not found", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		// First call to org endpoint returns 404
		mockRunner.On("Run", ctx, "gh", []string{"api", "orgs/nonexistent-org/repos?per_page=100&type=all", "--paginate"}).
			Return(nil, &CommandError{Stderr: "404 Not Found"})

		// Second call to user endpoint also returns 404
		mockRunner.On("Run", ctx, "gh", []string{"api", "users/nonexistent-org/repos?per_page=100&type=all", "--paginate"}).
			Return(nil, &CommandError{Stderr: "404 Not Found"})

		result, err := client.DiscoverOrgRepos(ctx, "nonexistent-org")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner not found")
		assert.Nil(t, result)

		mockRunner.AssertExpectations(t)
	})

	t.Run("user account discovery", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		desc1 := "User repo"
		lang1 := "JavaScript"

		repos := []RepoInfo{
			{
				Name:     "user-repo",
				FullName: "testuser/user-repo",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "testuser"},
				Description:   &desc1,
				Language:      &lang1,
				Private:       false,
				Fork:          false,
				Archived:      false,
				DefaultBranch: "main",
				HTMLURL:       "https://github.com/testuser/user-repo",
			},
		}

		output, err := json.Marshal(repos)
		require.NoError(t, err)

		// Org endpoint returns 404 (user account, not org)
		mockRunner.On("Run", ctx, "gh", []string{"api", "orgs/testuser/repos?per_page=100&type=all", "--paginate"}).
			Return(nil, &CommandError{Stderr: "404 Not Found"})

		// User endpoint succeeds
		mockRunner.On("Run", ctx, "gh", []string{"api", "users/testuser/repos?per_page=100&type=all", "--paginate"}).
			Return(output, nil)

		result, err := client.DiscoverOrgRepos(ctx, "testuser")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "user-repo", result[0].Name)
		assert.Equal(t, "testuser/user-repo", result[0].FullName)
		assert.Equal(t, "testuser", result[0].Owner.Login)

		mockRunner.AssertExpectations(t)
	})
}
