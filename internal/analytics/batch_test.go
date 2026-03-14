package analytics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

func TestBuildBatchQuery(t *testing.T) {
	t.Run("builds query for single repo", func(t *testing.T) {
		repos := []gh.RepoInfo{
			{
				Name: "test-repo",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "test-owner"},
				FullName: "test-owner/test-repo",
			},
		}

		query := BuildBatchQuery(repos)

		assert.Contains(t, query, "query {")
		assert.Contains(t, query, `repo0: repository(owner: "test-owner", name: "test-repo")`)
		assert.Contains(t, query, "...RepoFields")
		assert.Contains(t, query, "fragment RepoFields on Repository")
		assert.Contains(t, query, "stargazerCount")
		assert.Contains(t, query, "forkCount")
		assert.Contains(t, query, "issues(states: [OPEN])")
		assert.Contains(t, query, "pullRequests(states: [OPEN])")
		assert.Contains(t, query, `tags: refs(refPrefix: "refs/tags/"`)
		assert.Contains(t, query, "pushedAt")
		assert.Contains(t, query, "isFork")
		assert.Contains(t, query, "parent { nameWithOwner }")
	})

	t.Run("builds query for multiple repos", func(t *testing.T) {
		repos := []gh.RepoInfo{
			{
				Name: "repo1",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "owner1"},
			},
			{
				Name: "repo2",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "owner2"},
			},
			{
				Name: "repo3",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "owner3"},
			},
		}

		query := BuildBatchQuery(repos)

		assert.Contains(t, query, `repo0: repository(owner: "owner1", name: "repo1")`)
		assert.Contains(t, query, `repo1: repository(owner: "owner2", name: "repo2")`)
		assert.Contains(t, query, `repo2: repository(owner: "owner3", name: "repo3")`)
	})

	t.Run("returns empty for no repos", func(t *testing.T) {
		repos := []gh.RepoInfo{}
		query := BuildBatchQuery(repos)
		assert.Empty(t, query)
	})
}

func TestChunkRepos(t *testing.T) {
	t.Run("chunks repos into batches", func(t *testing.T) {
		repos := make([]gh.RepoInfo, 75)
		for i := range repos {
			repos[i] = gh.RepoInfo{
				Name: "repo",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "owner"},
			}
		}

		chunks := ChunkRepos(repos, 25)

		assert.Len(t, chunks, 3)
		assert.Len(t, chunks[0], 25)
		assert.Len(t, chunks[1], 25)
		assert.Len(t, chunks[2], 25)
	})

	t.Run("handles uneven chunks", func(t *testing.T) {
		repos := make([]gh.RepoInfo, 52)
		for i := range repos {
			repos[i] = gh.RepoInfo{
				Name: "repo",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "owner"},
			}
		}

		chunks := ChunkRepos(repos, 25)

		assert.Len(t, chunks, 3)
		assert.Len(t, chunks[0], 25)
		assert.Len(t, chunks[1], 25)
		assert.Len(t, chunks[2], 2)
	})

	t.Run("uses default batch size for invalid size", func(t *testing.T) {
		repos := make([]gh.RepoInfo, 50)
		chunks := ChunkRepos(repos, 0)

		assert.Len(t, chunks, 2)
		assert.Len(t, chunks[0], DefaultBatchSize)
	})

	t.Run("handles empty repo list", func(t *testing.T) {
		repos := []gh.RepoInfo{}
		chunks := ChunkRepos(repos, 25)

		assert.Empty(t, chunks)
	})
}

func TestParseBatchResponse(t *testing.T) {
	t.Run("parses single repo response", func(t *testing.T) {
		repos := []gh.RepoInfo{
			{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "owner"},
			},
		}

		publishedAt := "2024-01-15T10:00:00Z"

		data := map[string]interface{}{
			"repo0": map[string]interface{}{
				"nameWithOwner":  "owner/test-repo",
				"stargazerCount": float64(42),
				"forkCount":      float64(5),
				"description":    "A test repository",
				"updatedAt":      "2024-01-20T12:00:00Z",
				"pushedAt":       "2024-01-19T08:30:00Z",
				"isFork":         true,
				"parent": map[string]interface{}{
					"nameWithOwner": "upstream/test-repo",
				},
				"watchers": map[string]interface{}{
					"totalCount": float64(10),
				},
				"issues": map[string]interface{}{
					"totalCount": float64(3),
				},
				"pullRequests": map[string]interface{}{
					"totalCount": float64(1),
				},
				"refs": map[string]interface{}{
					"totalCount": float64(5),
				},
				"defaultBranchRef": map[string]interface{}{
					"name": "main",
				},
				"latestRelease": map[string]interface{}{
					"tagName":     "v1.2.3",
					"publishedAt": publishedAt,
				},
				"tags": map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{
							"name": "v1.2.3",
							"target": map[string]interface{}{
								"tagger": map[string]interface{}{
									"date": "2024-01-15T10:00:00Z",
								},
							},
						},
					},
				},
			},
		}

		result, err := ParseBatchResponse(data, repos)
		require.NoError(t, err)
		require.Len(t, result, 1)

		metadata := result["owner/test-repo"]
		require.NotNil(t, metadata)
		assert.Equal(t, "owner/test-repo", metadata.FullName)
		assert.Equal(t, 42, metadata.Stars)
		assert.Equal(t, 5, metadata.Forks)
		assert.Equal(t, 10, metadata.Watchers)
		assert.Equal(t, 3, metadata.OpenIssues)
		assert.Equal(t, 1, metadata.OpenPRs)
		assert.Equal(t, 5, metadata.BranchCount)
		assert.Equal(t, "main", metadata.DefaultBranch)
		assert.Equal(t, "A test repository", metadata.Description)
		assert.Equal(t, "v1.2.3", metadata.LatestRelease)
		assert.NotNil(t, metadata.LatestReleaseAt)
		assert.Equal(t, publishedAt, *metadata.LatestReleaseAt)
		assert.Equal(t, "v1.2.3", metadata.LatestTag)
		assert.NotNil(t, metadata.LatestTagAt)
		assert.Equal(t, "2024-01-15T10:00:00Z", *metadata.LatestTagAt)
		assert.Equal(t, "2024-01-20T12:00:00Z", metadata.UpdatedAt)
		assert.Equal(t, "2024-01-19T08:30:00Z", metadata.PushedAt)
		assert.True(t, metadata.IsFork)
		assert.Equal(t, "upstream/test-repo", metadata.ForkParent)
	})

	t.Run("non-fork repo has empty fork fields", func(t *testing.T) {
		repos := []gh.RepoInfo{
			{
				Name:     "my-repo",
				FullName: "owner/my-repo",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "owner"},
			},
		}

		data := map[string]interface{}{
			"repo0": map[string]interface{}{
				"nameWithOwner":  "owner/my-repo",
				"stargazerCount": float64(10),
				"isFork":         false,
			},
		}

		result, err := ParseBatchResponse(data, repos)
		require.NoError(t, err)
		require.Len(t, result, 1)

		metadata := result["owner/my-repo"]
		require.NotNil(t, metadata)
		assert.False(t, metadata.IsFork)
		assert.Empty(t, metadata.ForkParent)
		assert.Empty(t, metadata.PushedAt)
	})

	t.Run("handles missing optional fields", func(t *testing.T) {
		repos := []gh.RepoInfo{
			{
				Name:     "minimal-repo",
				FullName: "owner/minimal-repo",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "owner"},
			},
		}

		data := map[string]interface{}{
			"repo0": map[string]interface{}{
				"nameWithOwner":  "owner/minimal-repo",
				"stargazerCount": float64(0),
			},
		}

		result, err := ParseBatchResponse(data, repos)
		require.NoError(t, err)
		require.Len(t, result, 1)

		metadata := result["owner/minimal-repo"]
		require.NotNil(t, metadata)
		assert.Equal(t, 0, metadata.Stars)
		assert.Equal(t, 0, metadata.Forks)
		assert.Empty(t, metadata.Description)
	})

	t.Run("skips repos with missing data", func(t *testing.T) {
		repos := []gh.RepoInfo{
			{
				Name:     "repo1",
				FullName: "owner/repo1",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "owner"},
			},
			{
				Name:     "repo2",
				FullName: "owner/repo2",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "owner"},
			},
		}

		// Only repo0 has data, repo1 is missing (might be private/inaccessible)
		data := map[string]interface{}{
			"repo0": map[string]interface{}{
				"nameWithOwner":  "owner/repo1",
				"stargazerCount": float64(10),
			},
		}

		result, err := ParseBatchResponse(data, repos)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "owner/repo1")
		assert.NotContains(t, result, "owner/repo2")
	})
}
