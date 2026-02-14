package gh

import (
	"context"
	"fmt"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/jsonutil"
)

// DiscoverOrgRepos returns all repositories for an owner (organization or user account)
// Automatically detects whether owner is an org or user by trying org endpoint first,
// then falling back to user endpoint if org is not found.
// Uses REST API with pagination to fetch all repos (public and private)
func (g *githubClient) DiscoverOrgRepos(ctx context.Context, org string) ([]RepoInfo, error) {
	// Try organization endpoint first
	repos, err := g.tryDiscoverRepos(ctx, fmt.Sprintf("orgs/%s/repos?per_page=100&type=all", org))
	if err == nil {
		return repos, nil
	}

	// If org not found (404), try user endpoint
	if isNotFoundError(err) {
		repos, userErr := g.tryDiscoverRepos(ctx, fmt.Sprintf("users/%s/repos?per_page=100&type=all", org))
		if userErr == nil {
			return repos, nil
		}

		// Both failed - owner doesn't exist as org or user
		if isNotFoundError(userErr) {
			return nil, fmt.Errorf("%w: %s", ErrOwnerNotFound, org)
		}

		// User endpoint returned a different error (rate limit, permission, etc.)
		return nil, appErrors.WrapWithContext(userErr, fmt.Sprintf("discover repos for user %s", org))
	}

	// Org endpoint returned a different error (rate limit, permission, etc.)
	return nil, appErrors.WrapWithContext(err, fmt.Sprintf("discover repos for owner %s", org))
}

// tryDiscoverRepos attempts to fetch repos from a given API endpoint
func (g *githubClient) tryDiscoverRepos(ctx context.Context, endpoint string) ([]RepoInfo, error) {
	output, err := g.runner.Run(ctx, "gh", "api", endpoint, "--paginate")
	if err != nil {
		return nil, err
	}

	repos, err := jsonutil.UnmarshalJSON[[]RepoInfo](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse repository list")
	}

	return repos, nil
}
