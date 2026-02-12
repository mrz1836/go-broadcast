package gh

import (
	"context"
	"fmt"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/jsonutil"
)

// DiscoverOrgRepos returns all repositories for an organization
// Uses REST API with pagination to fetch all repos (public and private)
func (g *githubClient) DiscoverOrgRepos(ctx context.Context, org string) ([]RepoInfo, error) {
	// Use gh api with --paginate to handle large org repos
	// per_page=100 is the max, type=all includes both public and private repos
	output, err := g.runner.Run(ctx, "gh", "api",
		fmt.Sprintf("orgs/%s/repos?per_page=100&type=all", org),
		"--paginate")
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("organization not found: %s", org)
		}
		return nil, appErrors.WrapWithContext(err, fmt.Sprintf("discover repos for org %s", org))
	}

	repos, err := jsonutil.UnmarshalJSON[[]RepoInfo](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse repository list")
	}

	return repos, nil
}
