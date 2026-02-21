package gh

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/jsonutil"
)

// ErrSecurityNotAvailable indicates a security endpoint returned 404,
// meaning the feature is not enabled or the token lacks required scopes.
// Callers should check for this with errors.Is and consider fallback strategies.
var ErrSecurityNotAvailable = errors.New("security feature not available via REST API")

// GetDependabotAlerts retrieves Dependabot security alerts for a repository via REST API.
//
// Returns ErrSecurityNotAvailable if:
//   - The REST endpoint returns HTTP 404 (token lacks security_events scope or Dependabot not enabled)
//
// Returns empty slice with nil error if:
//   - The endpoint is accessible but the repository has no open Dependabot alerts
//
// Returns other errors for actual API failures (auth, network, rate limits, etc.)
func (g *githubClient) GetDependabotAlerts(ctx context.Context, repo string) ([]DependabotAlert, error) {
	output, err := g.runner.Run(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/dependabot/alerts", repo),
		"-F", "state=open",
		"-F", "per_page=100",
		"--paginate")
	if err != nil {
		if isNotFoundError(err) {
			g.logger.Debugf("Dependabot REST API returned 404 for %s (token may lack security_events scope)", repo)
			return nil, fmt.Errorf("%w: dependabot alerts for %s", ErrSecurityNotAvailable, repo)
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

// GetCodeScanningAlerts retrieves code scanning alerts for a repository via REST API.
//
// Returns ErrSecurityNotAvailable if:
//   - The REST endpoint returns HTTP 404 (code scanning not configured or token lacks scope)
//
// Returns empty slice with nil error if:
//   - The endpoint is accessible but the repository has no open code scanning alerts
//
// Returns other errors for actual API failures (auth, network, rate limits, etc.)
func (g *githubClient) GetCodeScanningAlerts(ctx context.Context, repo string) ([]CodeScanningAlert, error) {
	output, err := g.runner.Run(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/code-scanning/alerts", repo),
		"-F", "state=open",
		"-F", "per_page=100",
		"--paginate")
	if err != nil {
		if isNotFoundError(err) {
			g.logger.Debugf("Code scanning REST API returned 404 for %s", repo)
			return nil, fmt.Errorf("%w: code scanning alerts for %s", ErrSecurityNotAvailable, repo)
		}
		return nil, appErrors.WrapWithContext(err, "get code scanning alerts")
	}

	alerts, err := jsonutil.UnmarshalJSON[[]CodeScanningAlert](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse code scanning alerts")
	}

	return alerts, nil
}

// GetSecretScanningAlerts retrieves secret scanning alerts for a repository via REST API.
//
// Returns ErrSecurityNotAvailable if:
//   - The REST endpoint returns HTTP 404 (secret scanning not configured or token lacks scope)
//
// Returns empty slice with nil error if:
//   - The endpoint is accessible but the repository has no open secret scanning alerts
//
// Returns other errors for actual API failures (auth, network, rate limits, etc.)
func (g *githubClient) GetSecretScanningAlerts(ctx context.Context, repo string) ([]SecretScanningAlert, error) {
	output, err := g.runner.Run(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/secret-scanning/alerts", repo),
		"-F", "state=open",
		"-F", "per_page=100",
		"--paginate")
	if err != nil {
		if isNotFoundError(err) {
			g.logger.Debugf("Secret scanning REST API returned 404 for %s", repo)
			return nil, fmt.Errorf("%w: secret scanning alerts for %s", ErrSecurityNotAvailable, repo)
		}
		return nil, appErrors.WrapWithContext(err, "get secret scanning alerts")
	}

	alerts, err := jsonutil.UnmarshalJSON[[]SecretScanningAlert](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse secret scanning alerts")
	}

	return alerts, nil
}

// graphqlVulnerabilityResponse models the GraphQL response for vulnerability alerts
type graphqlVulnerabilityResponse struct {
	Repository struct {
		VulnerabilityAlerts struct {
			TotalCount int `json:"totalCount"`
			Nodes      []struct {
				Number                int       `json:"number"`
				CreatedAt             time.Time `json:"createdAt"`
				State                 string    `json:"state"`
				SecurityVulnerability struct {
					Severity string `json:"severity"`
					Package  struct {
						Name      string `json:"name"`
						Ecosystem string `json:"ecosystem"`
					} `json:"package"`
					Advisory struct {
						Summary   string `json:"summary"`
						GhsaID    string `json:"ghsaId"`
						Permalink string `json:"permalink"`
					} `json:"advisory"`
					VulnerableVersionRange string `json:"vulnerableVersionRange"`
					FirstPatchedVersion    *struct {
						Identifier string `json:"identifier"`
					} `json:"firstPatchedVersion"`
				} `json:"securityVulnerability"`
				VulnerableManifestPath string     `json:"vulnerableManifestPath"`
				DismissedAt            *time.Time `json:"dismissedAt"`
				FixedAt                *time.Time `json:"fixedAt"`
			} `json:"nodes"`
		} `json:"vulnerabilityAlerts"`
	} `json:"repository"`
}

// GetVulnerabilityAlertsGraphQL retrieves vulnerability alerts via the GraphQL API.
// This is the fallback when REST Dependabot endpoints return 404 due to token scope limitations.
// The GraphQL vulnerabilityAlerts query works with standard repository read access.
func (g *githubClient) GetVulnerabilityAlertsGraphQL(ctx context.Context, repo string) ([]VulnerabilityAlert, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format %q, expected owner/name", repo) //nolint:err113 // user-facing
	}
	owner, name := parts[0], parts[1]

	query := fmt.Sprintf(`{
  repository(owner: %q, name: %q) {
    vulnerabilityAlerts(first: 100, states: OPEN) {
      totalCount
      nodes {
        number
        createdAt
        state
        securityVulnerability {
          severity
          package { name ecosystem }
          advisory { summary ghsaId permalink }
          vulnerableVersionRange
          firstPatchedVersion { identifier }
        }
        vulnerableManifestPath
        dismissedAt
        fixedAt
      }
    }
  }
}`, owner, name)

	data, err := g.ExecuteGraphQL(ctx, query)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "graphql vulnerability alerts")
	}

	// Re-marshal and unmarshal through our typed struct for safe extraction
	jsonBytes, err := jsonutil.MarshalJSON(data)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "marshal graphql response")
	}

	response, err := jsonutil.UnmarshalJSON[graphqlVulnerabilityResponse](jsonBytes)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse graphql vulnerability response")
	}

	// Convert to VulnerabilityAlert slice
	nodes := response.Repository.VulnerabilityAlerts.Nodes
	alerts := make([]VulnerabilityAlert, 0, len(nodes))
	for _, node := range nodes {
		alert := VulnerabilityAlert{
			Number:                 node.Number,
			State:                  node.State,
			CreatedAt:              node.CreatedAt,
			DismissedAt:            node.DismissedAt,
			FixedAt:                node.FixedAt,
			Severity:               node.SecurityVulnerability.Severity,
			PackageName:            node.SecurityVulnerability.Package.Name,
			PackageEcosystem:       node.SecurityVulnerability.Package.Ecosystem,
			AdvisorySummary:        node.SecurityVulnerability.Advisory.Summary,
			AdvisoryGHSAID:         node.SecurityVulnerability.Advisory.GhsaID,
			AdvisoryPermalink:      node.SecurityVulnerability.Advisory.Permalink,
			VulnerableVersionRange: node.SecurityVulnerability.VulnerableVersionRange,
			ManifestPath:           node.VulnerableManifestPath,
		}
		if node.SecurityVulnerability.FirstPatchedVersion != nil {
			alert.FirstPatchedVersion = node.SecurityVulnerability.FirstPatchedVersion.Identifier
		}
		alerts = append(alerts, alert)
	}

	if g.logger != nil {
		g.logger.Debugf("GraphQL: found %d vulnerability alerts for %s (totalCount=%d)",
			len(alerts), repo, response.Repository.VulnerabilityAlerts.TotalCount)
	}

	return alerts, nil
}
