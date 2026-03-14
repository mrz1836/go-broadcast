package gh

import (
	"context"
	"fmt"
	"strings"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/jsonutil"
)

// CreateRepository creates a new GitHub repository
func (g *githubClient) CreateRepository(ctx context.Context, opts CreateRepoOptions) (*Repository, error) {
	visibility := "--private"
	if !opts.Private {
		visibility = "--public"
	}

	args := []string{
		"repo", "create", opts.Name,
		"--description", opts.Description,
		visibility,
		"--clone=false",
	}

	var result *Repository
	err := rateLimitedDo(ctx, defaultAPIDelay, func() error {
		_, runErr := g.runner.Run(ctx, "gh", args...)
		return runErr
	})
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "create repository")
	}

	// Fetch the created repository details
	result, err = g.GetRepository(ctx, opts.Name)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "get created repository")
	}

	return result, nil
}

// UpdateRepoSettings updates repository settings via PATCH /repos/{owner}/{repo}
func (g *githubClient) UpdateRepoSettings(ctx context.Context, repo string, settings RepoSettings) error {
	jsonData, err := jsonutil.MarshalJSON(settings)
	if err != nil {
		return appErrors.WrapWithContext(err, "marshal settings")
	}

	return rateLimitedDo(ctx, defaultAPIDelay, func() error {
		_, runErr := g.runner.RunWithInput(ctx, jsonData, "gh", "api",
			fmt.Sprintf("repos/%s", repo), "--method", "PATCH", "--input", "-")
		return runErr
	})
}

// GetRepoSettings retrieves repository settings as a typed RepoSettings struct
func (g *githubClient) GetRepoSettings(ctx context.Context, repo string) (*RepoSettings, error) {
	var output []byte
	err := rateLimitedDo(ctx, 0, func() error {
		var runErr error
		output, runErr = g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s", repo))
		return runErr
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("%w: %s", ErrRepositoryNotFound, repo)
		}
		return nil, appErrors.WrapWithContext(err, "get repo settings")
	}

	settings, err := jsonutil.UnmarshalJSON[RepoSettings](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse repo settings")
	}

	return &settings, nil
}

// CreateOrUpdateRuleset creates or updates a ruleset by name
func (g *githubClient) CreateOrUpdateRuleset(ctx context.Context, repo string, ruleset Ruleset) error {
	// List existing rulesets to find by name
	existing, err := g.ListRulesets(ctx, repo)
	if err != nil {
		// If listing fails (e.g., 404 for repos without rulesets), try create directly
		if !isNotFoundError(err) {
			return appErrors.WrapWithContext(err, "list rulesets for upsert")
		}
	}

	// Check if ruleset with same name exists
	for _, r := range existing {
		if r.Name == ruleset.Name {
			// Update existing ruleset
			jsonData, marshalErr := jsonutil.MarshalJSON(ruleset)
			if marshalErr != nil {
				return appErrors.WrapWithContext(marshalErr, "marshal ruleset update")
			}
			return rateLimitedDo(ctx, defaultAPIDelay, func() error {
				_, runErr := g.runner.RunWithInput(ctx, jsonData, "gh", "api",
					fmt.Sprintf("repos/%s/rulesets/%d", repo, r.ID),
					"--method", "PUT", "--input", "-")
				return runErr
			})
		}
	}

	// Create new ruleset
	jsonData, err := jsonutil.MarshalJSON(ruleset)
	if err != nil {
		return appErrors.WrapWithContext(err, "marshal ruleset")
	}

	return rateLimitedDo(ctx, defaultAPIDelay, func() error {
		_, runErr := g.runner.RunWithInput(ctx, jsonData, "gh", "api",
			fmt.Sprintf("repos/%s/rulesets", repo),
			"--method", "POST", "--input", "-")
		return runErr
	})
}

// ListRulesets lists all rulesets for a repository
func (g *githubClient) ListRulesets(ctx context.Context, repo string) ([]Ruleset, error) {
	var output []byte
	err := rateLimitedDo(ctx, 0, func() error {
		var runErr error
		output, runErr = g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/rulesets", repo))
		return runErr
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, appErrors.WrapWithContext(err, "list rulesets")
	}

	rulesets, err := jsonutil.UnmarshalJSON[[]Ruleset](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse rulesets")
	}

	return rulesets, nil
}

// SyncLabels idempotently syncs labels (PATCH existing, POST missing)
func (g *githubClient) SyncLabels(ctx context.Context, repo string, labels []Label) error {
	// List existing labels for idempotent sync
	existing, err := g.ListLabels(ctx, repo)
	if err != nil {
		return appErrors.WrapWithContext(err, "list labels for sync")
	}

	existingMap := make(map[string]bool, len(existing))
	for _, l := range existing {
		existingMap[strings.ToLower(l.Name)] = true
	}

	for _, label := range labels {
		jsonData, marshalErr := jsonutil.MarshalJSON(label)
		if marshalErr != nil {
			return appErrors.WrapWithContext(marshalErr, "marshal label")
		}

		if existingMap[strings.ToLower(label.Name)] {
			// Update existing label
			encodedName := strings.ReplaceAll(label.Name, " ", "%20")
			syncErr := rateLimitedDo(ctx, defaultAPIDelay, func() error {
				_, runErr := g.runner.RunWithInput(ctx, jsonData, "gh", "api",
					fmt.Sprintf("repos/%s/labels/%s", repo, encodedName),
					"--method", "PATCH", "--input", "-")
				return runErr
			})
			if syncErr != nil {
				if g.logger != nil {
					g.logger.WithError(syncErr).Warnf("Failed to update label %q", label.Name)
				}
			}
		} else {
			// Create new label
			syncErr := rateLimitedDo(ctx, defaultAPIDelay, func() error {
				_, runErr := g.runner.RunWithInput(ctx, jsonData, "gh", "api",
					fmt.Sprintf("repos/%s/labels", repo),
					"--method", "POST", "--input", "-")
				return runErr
			})
			if syncErr != nil {
				if g.logger != nil {
					g.logger.WithError(syncErr).Warnf("Failed to create label %q", label.Name)
				}
			}
		}
	}

	return nil
}

// ListLabels lists all labels for a repository
func (g *githubClient) ListLabels(ctx context.Context, repo string) ([]Label, error) {
	var output []byte
	err := rateLimitedDo(ctx, 0, func() error {
		var runErr error
		output, runErr = g.runner.Run(ctx, "gh", "api",
			fmt.Sprintf("repos/%s/labels", repo), "--paginate")
		return runErr
	})
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "list labels")
	}

	labels, err := jsonutil.UnmarshalJSON[[]Label](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse labels")
	}

	return labels, nil
}

// SetTopics replaces all topics for a repository
func (g *githubClient) SetTopics(ctx context.Context, repo string, topics []string) error {
	payload := map[string][]string{"names": topics}
	jsonData, err := jsonutil.MarshalJSON(payload)
	if err != nil {
		return appErrors.WrapWithContext(err, "marshal topics")
	}

	return rateLimitedDo(ctx, defaultAPIDelay, func() error {
		_, runErr := g.runner.RunWithInput(ctx, jsonData, "gh", "api",
			fmt.Sprintf("repos/%s/topics", repo),
			"--method", "PUT", "--input", "-")
		return runErr
	})
}
