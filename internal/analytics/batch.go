package analytics

import (
	"fmt"
	"strings"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

const (
	// DefaultBatchSize is the default number of repos per GraphQL query
	// 25 repos per query = 3 calls for 75 repos (96% reduction in API calls)
	DefaultBatchSize = 25

	// FallbackBatchSize is used if GraphQL complexity limit is hit
	FallbackBatchSize = 10
)

// RepoMetadata represents the metadata collected from GraphQL for a single repository
type RepoMetadata struct {
	FullName        string
	Stars           int
	Forks           int
	Watchers        int
	OpenIssues      int
	OpenPRs         int
	BranchCount     int
	DefaultBranch   string
	Description     string
	LatestRelease   string
	LatestReleaseAt *string
	LatestTag       string
	LatestTagAt     *string
	UpdatedAt       string
	PushedAt        string // last code push timestamp
	IsFork          bool
	ForkParent      string // parent repo nameWithOwner
}

// BuildBatchQuery creates an aliased GraphQL query for multiple repos
// Each repo is aliased as repo0, repo1, etc. to allow batch fetching
func BuildBatchQuery(repos []gh.RepoInfo) string {
	if len(repos) == 0 {
		return ""
	}

	var sb strings.Builder

	// Start query
	sb.WriteString("query {\n")

	// Add aliased repository queries
	for i, repo := range repos {
		alias := fmt.Sprintf("repo%d", i)
		sb.WriteString(fmt.Sprintf(`  %s: repository(owner: "%s", name: "%s") {
    ...RepoFields
  }
`, alias, repo.Owner.Login, repo.Name))
	}

	// Add fragment definition
	sb.WriteString(`}

fragment RepoFields on Repository {
  nameWithOwner
  stargazerCount
  forkCount
  pushedAt
  isFork
  parent { nameWithOwner }
  watchers {
    totalCount
  }
  issues(states: [OPEN]) {
    totalCount
  }
  pullRequests(states: [OPEN]) {
    totalCount
  }
  refs(refPrefix: "refs/heads/") {
    totalCount
  }
  defaultBranchRef {
    name
  }
  description
  updatedAt
  latestRelease {
    tagName
    publishedAt
  }
  tags: refs(refPrefix: "refs/tags/", last: 1, orderBy: {field: TAG_COMMIT_DATE, direction: DESC}) {
    nodes {
      name
      target {
        ... on Tag {
          tagger {
            date
          }
        }
        ... on Commit {
          committedDate
        }
      }
    }
  }
}
`)

	return sb.String()
}

// ChunkRepos splits repos into batches of given size
func ChunkRepos(repos []gh.RepoInfo, batchSize int) [][]gh.RepoInfo {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	var chunks [][]gh.RepoInfo
	for i := 0; i < len(repos); i += batchSize {
		end := i + batchSize
		if end > len(repos) {
			end = len(repos)
		}
		chunks = append(chunks, repos[i:end])
	}

	return chunks
}

// ParseBatchResponse extracts per-repo data from aliased GraphQL response
// The response structure matches the aliased query format (repo0, repo1, etc.)
func ParseBatchResponse(data map[string]interface{}, repos []gh.RepoInfo) (map[string]*RepoMetadata, error) {
	result := make(map[string]*RepoMetadata)

	for i, repo := range repos {
		alias := fmt.Sprintf("repo%d", i)
		repoData, ok := data[alias].(map[string]interface{})
		if !ok {
			// Repo might not exist or be inaccessible, skip it
			continue
		}

		metadata := &RepoMetadata{
			FullName: repo.FullName,
		}

		// Extract scalar fields
		if nameWithOwner, ok := repoData["nameWithOwner"].(string); ok {
			metadata.FullName = nameWithOwner
		}
		if stars, ok := repoData["stargazerCount"].(float64); ok {
			metadata.Stars = int(stars)
		}
		if forks, ok := repoData["forkCount"].(float64); ok {
			metadata.Forks = int(forks)
		}
		if desc, ok := repoData["description"].(string); ok {
			metadata.Description = desc
		}
		if updatedAt, ok := repoData["updatedAt"].(string); ok {
			metadata.UpdatedAt = updatedAt
		}
		if pushedAt, ok := repoData["pushedAt"].(string); ok {
			metadata.PushedAt = pushedAt
		}
		if isFork, ok := repoData["isFork"].(bool); ok {
			metadata.IsFork = isFork
		}
		if parent, ok := repoData["parent"].(map[string]interface{}); ok {
			if nwo, ok := parent["nameWithOwner"].(string); ok {
				metadata.ForkParent = nwo
			}
		}

		// Extract watchers (nested object)
		if watchers, ok := repoData["watchers"].(map[string]interface{}); ok {
			if count, ok := watchers["totalCount"].(float64); ok {
				metadata.Watchers = int(count)
			}
		}

		// Extract open issues count
		if issues, ok := repoData["issues"].(map[string]interface{}); ok {
			if count, ok := issues["totalCount"].(float64); ok {
				metadata.OpenIssues = int(count)
			}
		}

		// Extract open PRs count
		if prs, ok := repoData["pullRequests"].(map[string]interface{}); ok {
			if count, ok := prs["totalCount"].(float64); ok {
				metadata.OpenPRs = int(count)
			}
		}

		// Extract branch count (refs with refPrefix: "refs/heads/")
		if refs, ok := repoData["refs"].(map[string]interface{}); ok {
			if count, ok := refs["totalCount"].(float64); ok {
				metadata.BranchCount = int(count)
			}
		}

		// Extract default branch
		if defaultBranch, ok := repoData["defaultBranchRef"].(map[string]interface{}); ok {
			if name, ok := defaultBranch["name"].(string); ok {
				metadata.DefaultBranch = name
			}
		}

		// Extract latest release
		if latestRelease, ok := repoData["latestRelease"].(map[string]interface{}); ok {
			if tagName, ok := latestRelease["tagName"].(string); ok {
				metadata.LatestRelease = tagName
			}
			if publishedAt, ok := latestRelease["publishedAt"].(string); ok {
				metadata.LatestReleaseAt = &publishedAt
			}
		}

		// Extract latest tag from aliased "tags" field
		if tags, ok := repoData["tags"].(map[string]interface{}); ok {
			if nodes, ok := tags["nodes"].([]interface{}); ok && len(nodes) > 0 {
				if node, ok := nodes[0].(map[string]interface{}); ok {
					if name, ok := node["name"].(string); ok {
						metadata.LatestTag = name
					}
					// Extract tag date from target (can be Tag or Commit)
					if target, ok := node["target"].(map[string]interface{}); ok {
						if tagger, ok := target["tagger"].(map[string]interface{}); ok {
							if date, ok := tagger["date"].(string); ok {
								metadata.LatestTagAt = &date
							}
						} else if committedDate, ok := target["committedDate"].(string); ok {
							metadata.LatestTagAt = &committedDate
						}
					}
				}
			}
		}

		result[repo.FullName] = metadata
	}

	return result, nil
}
