package gh

import (
	"context"
	"fmt"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/jsonutil"
)

// GetDependabotAlerts retrieves Dependabot security alerts for a repository
// Returns empty slice if Dependabot is not enabled (404 response)
func (g *githubClient) GetDependabotAlerts(ctx context.Context, repo string) ([]DependabotAlert, error) {
	output, err := g.runner.Run(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/dependabot/alerts", repo),
		"-F", "state=open",
		"-F", "per_page=100",
		"--paginate")
	if err != nil {
		if isNotFoundError(err) {
			// Dependabot not enabled for this repository
			g.logger.Debugf("Dependabot not enabled for %s (404)", repo)
			return []DependabotAlert{}, nil
		}
		return nil, appErrors.WrapWithContext(err, "get dependabot alerts")
	}

	alerts, err := jsonutil.UnmarshalJSON[[]DependabotAlert](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse dependabot alerts")
	}

	// Populate extracted fields for easier database storage
	for i := range alerts {
		alerts[i].DependencyPackage = alerts[i].Dependency.Package.Name
		alerts[i].DependencyManifest = alerts[i].Dependency.ManifestPath
	}

	return alerts, nil
}

// GetCodeScanningAlerts retrieves code scanning alerts for a repository
// Returns empty slice if code scanning is not enabled (404 response)
func (g *githubClient) GetCodeScanningAlerts(ctx context.Context, repo string) ([]CodeScanningAlert, error) {
	output, err := g.runner.Run(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/code-scanning/alerts", repo),
		"-F", "state=open",
		"-F", "per_page=100",
		"--paginate")
	if err != nil {
		if isNotFoundError(err) {
			// Code scanning not enabled for this repository
			g.logger.Debugf("Code scanning not enabled for %s (404)", repo)
			return []CodeScanningAlert{}, nil
		}
		return nil, appErrors.WrapWithContext(err, "get code scanning alerts")
	}

	alerts, err := jsonutil.UnmarshalJSON[[]CodeScanningAlert](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse code scanning alerts")
	}

	return alerts, nil
}

// GetSecretScanningAlerts retrieves secret scanning alerts for a repository
// Returns empty slice if secret scanning is not enabled (404 response)
func (g *githubClient) GetSecretScanningAlerts(ctx context.Context, repo string) ([]SecretScanningAlert, error) {
	output, err := g.runner.Run(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/secret-scanning/alerts", repo),
		"-F", "state=open",
		"-F", "per_page=100",
		"--paginate")
	if err != nil {
		if isNotFoundError(err) {
			// Secret scanning not enabled for this repository
			g.logger.Debugf("Secret scanning not enabled for %s (404)", repo)
			return []SecretScanningAlert{}, nil
		}
		return nil, appErrors.WrapWithContext(err, "get secret scanning alerts")
	}

	alerts, err := jsonutil.UnmarshalJSON[[]SecretScanningAlert](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse secret scanning alerts")
	}

	return alerts, nil
}
