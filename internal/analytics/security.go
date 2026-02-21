package analytics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

const (
	// SecurityWorkerLimit is the max number of concurrent security API calls.
	// Kept low (3) so the shared rate limiter can govern throughput effectively.
	SecurityWorkerLimit = 3

	// RateLimitThreshold is the minimum remaining rate limit before pausing
	RateLimitThreshold = 100
)

// SecurityAlertType represents the type of security alert
type SecurityAlertType string

const (
	AlertTypeDependabot     SecurityAlertType = "dependabot"
	AlertTypeCodeScanning   SecurityAlertType = "code_scanning"
	AlertTypeSecretScanning SecurityAlertType = "secret_scanning"
)

// SecurityAlert represents a unified security alert for database storage
type SecurityAlert struct {
	RepositoryID int64             // Foreign key to repositories table
	AlertType    SecurityAlertType // dependabot, code_scanning, secret_scanning
	AlertNumber  int               // Alert number from GitHub
	State        string            // open, dismissed, fixed, resolved
	Severity     string            // Severity level (varies by alert type)
	Title        string            // Human-readable title/description
	HTMLURL      string            // Link to the alert on GitHub
	CreatedAt    string            // ISO 8601 timestamp
	UpdatedAt    string            // ISO 8601 timestamp
	DismissedAt  *string           // ISO 8601 timestamp (nullable)
	FixedAt      *string           // ISO 8601 timestamp (nullable)
	ResolvedAt   *string           // ISO 8601 timestamp (nullable)
}

// SecurityCollectionResult holds alerts and any warnings from collection
type SecurityCollectionResult struct {
	Alerts   []SecurityAlert
	Warnings []string // User-visible warnings (e.g., "REST 404, used GraphQL fallback")
}

// SecurityCollector handles concurrent security alert collection
type SecurityCollector struct {
	ghClient gh.Client
	logger   *logrus.Logger
	throttle *Throttle
}

// NewSecurityCollector creates a new security alert collector.
// throttle may be nil for unthrottled operation.
func NewSecurityCollector(ghClient gh.Client, logger *logrus.Logger, throttle *Throttle) *SecurityCollector {
	return &SecurityCollector{
		ghClient: ghClient,
		logger:   logger,
		throttle: throttle,
	}
}

// CollectAlerts fetches all security alerts for multiple repositories concurrently.
// Returns a map of repo full name to collection results (alerts + warnings).
func (s *SecurityCollector) CollectAlerts(ctx context.Context, repos []gh.RepoInfo) (map[string]*SecurityCollectionResult, error) {
	if len(repos) == 0 {
		return make(map[string]*SecurityCollectionResult), nil
	}

	if s.logger != nil {
		s.logger.WithField("repo_count", len(repos)).Info("Starting concurrent security alert collection")
	}

	// Result map with mutex protection
	results := make(map[string]*SecurityCollectionResult)
	var resultMu sync.Mutex

	// Create errgroup with bounded concurrency
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(SecurityWorkerLimit)

	// Spawn workers for each repository
	for _, repo := range repos {
		g.Go(func() error {
			result := s.collectRepoAlerts(ctx, repo.FullName)

			resultMu.Lock()
			results[repo.FullName] = result
			resultMu.Unlock()

			if s.logger != nil {
				fields := logrus.Fields{
					"repo":        repo.FullName,
					"alert_count": len(result.Alerts),
				}
				if len(result.Warnings) > 0 {
					fields["warnings"] = len(result.Warnings)
				}
				s.logger.WithFields(fields).Debug("Collected security alerts")
			}

			return nil
		})
	}

	// Wait for all workers to complete
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("security alert collection failed: %w", err)
	}

	if s.logger != nil {
		totalAlerts := 0
		totalWarnings := 0
		for _, result := range results {
			totalAlerts += len(result.Alerts)
			totalWarnings += len(result.Warnings)
		}
		fields := logrus.Fields{
			"repos_processed": len(results),
			"total_alerts":    totalAlerts,
		}
		if totalWarnings > 0 {
			fields["total_warnings"] = totalWarnings
		}
		s.logger.WithFields(fields).Info("Security alert collection complete")
	}

	return results, nil
}

// collectRepoAlerts fetches all security alert types for a single repository.
// Never returns an error — all issues are captured as warnings in the result.
func (s *SecurityCollector) collectRepoAlerts(ctx context.Context, repo string) *SecurityCollectionResult {
	result := &SecurityCollectionResult{}

	// === Dependabot / Vulnerability Alerts ===
	s.collectDependabotAlerts(ctx, repo, result)

	// === Code Scanning Alerts ===
	s.collectCodeScanningAlerts(ctx, repo, result)

	// === Secret Scanning Alerts ===
	s.collectSecretScanningAlerts(ctx, repo, result)

	return result
}

// collectDependabotAlerts tries REST first, falls back to GraphQL on 404
func (s *SecurityCollector) collectDependabotAlerts(ctx context.Context, repo string, result *SecurityCollectionResult) {
	var dependabotAlerts []gh.DependabotAlert
	err := s.doAPI(ctx, "dependabot-alerts:"+repo, func() error {
		var apiErr error
		dependabotAlerts, apiErr = s.ghClient.GetDependabotAlerts(ctx, repo)
		return apiErr
	})

	if err == nil {
		// REST worked — convert alerts
		for _, alert := range dependabotAlerts {
			result.Alerts = append(result.Alerts, SecurityAlert{
				AlertType:   AlertTypeDependabot,
				AlertNumber: alert.Number,
				State:       alert.State,
				Severity:    alert.SecurityVulnerability.Severity,
				Title: fmt.Sprintf("%s vulnerability in %s",
					alert.SecurityVulnerability.Severity,
					alert.DependencyPackage),
				HTMLURL:     alert.HTMLURL,
				CreatedAt:   alert.CreatedAt.Format("2006-01-02T15:04:05Z"),
				UpdatedAt:   alert.UpdatedAt.Format("2006-01-02T15:04:05Z"),
				DismissedAt: formatTimePtr(alert.DismissedAt),
				FixedAt:     formatTimePtr(alert.FixedAt),
			})
		}
		return
	}

	// Check if it's a 404 — try GraphQL fallback
	if errors.Is(err, gh.ErrSecurityNotAvailable) {
		s.collectVulnerabilityAlertsGraphQL(ctx, repo, result)
		return
	}

	// Actual API error
	result.Warnings = append(result.Warnings,
		fmt.Sprintf("dependabot: failed to fetch alerts: %v", err))
}

// collectVulnerabilityAlertsGraphQL fetches vulnerability alerts via GraphQL as fallback
func (s *SecurityCollector) collectVulnerabilityAlertsGraphQL(ctx context.Context, repo string, result *SecurityCollectionResult) {
	var graphqlAlerts []gh.VulnerabilityAlert
	err := s.doAPI(ctx, "graphql-vuln-alerts:"+repo, func() error {
		var apiErr error
		graphqlAlerts, apiErr = s.ghClient.GetVulnerabilityAlertsGraphQL(ctx, repo)
		return apiErr
	})
	if err != nil {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("dependabot: REST API returned 404, GraphQL fallback also failed: %v", err))
		return
	}

	// Convert GraphQL alerts to unified SecurityAlert
	for _, alert := range graphqlAlerts {
		severity := normalizeSeverity(alert.Severity)
		result.Alerts = append(result.Alerts, SecurityAlert{
			AlertType:   AlertTypeDependabot,
			AlertNumber: alert.Number,
			State:       strings.ToLower(alert.State), // GraphQL returns OPEN/DISMISSED/FIXED
			Severity:    severity,
			Title: fmt.Sprintf("%s vulnerability in %s: %s",
				severity, alert.PackageName, alert.AdvisorySummary),
			HTMLURL:     alert.AdvisoryPermalink,
			CreatedAt:   alert.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   alert.CreatedAt.Format("2006-01-02T15:04:05Z"), // GraphQL doesn't have updatedAt
			DismissedAt: formatTimePtr(alert.DismissedAt),
			FixedAt:     formatTimePtr(alert.FixedAt),
		})
	}

	if len(graphqlAlerts) > 0 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("dependabot: REST API unavailable (404), used GraphQL fallback (%d alerts found)", len(graphqlAlerts)))
	} else {
		result.Warnings = append(result.Warnings,
			"dependabot: REST API unavailable (404), GraphQL fallback found no open alerts")
	}
}

// collectCodeScanningAlerts fetches code scanning alerts via REST
func (s *SecurityCollector) collectCodeScanningAlerts(ctx context.Context, repo string, result *SecurityCollectionResult) {
	var codeScanningAlerts []gh.CodeScanningAlert
	err := s.doAPI(ctx, "code-scanning-alerts:"+repo, func() error {
		var apiErr error
		codeScanningAlerts, apiErr = s.ghClient.GetCodeScanningAlerts(ctx, repo)
		return apiErr
	})
	if err != nil {
		if errors.Is(err, gh.ErrSecurityNotAvailable) {
			result.Warnings = append(result.Warnings,
				"code_scanning: REST API unavailable (404) — feature may not be enabled or token lacks scope")
		} else {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("code_scanning: failed to fetch alerts: %v", err))
		}
		return
	}

	for _, alert := range codeScanningAlerts {
		result.Alerts = append(result.Alerts, SecurityAlert{
			AlertType:   AlertTypeCodeScanning,
			AlertNumber: alert.Number,
			State:       alert.State,
			Severity:    alert.Rule.Severity,
			Title: fmt.Sprintf("%s: %s",
				alert.Rule.ID,
				alert.Rule.Description),
			HTMLURL:     alert.HTMLURL,
			CreatedAt:   alert.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   alert.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			DismissedAt: formatTimePtr(alert.DismissedAt),
			FixedAt:     formatTimePtr(alert.FixedAt),
		})
	}
}

// collectSecretScanningAlerts fetches secret scanning alerts via REST
func (s *SecurityCollector) collectSecretScanningAlerts(ctx context.Context, repo string, result *SecurityCollectionResult) {
	var secretScanningAlerts []gh.SecretScanningAlert
	err := s.doAPI(ctx, "secret-scanning-alerts:"+repo, func() error {
		var apiErr error
		secretScanningAlerts, apiErr = s.ghClient.GetSecretScanningAlerts(ctx, repo)
		return apiErr
	})
	if err != nil {
		if errors.Is(err, gh.ErrSecurityNotAvailable) {
			result.Warnings = append(result.Warnings,
				"secret_scanning: REST API unavailable (404) — feature may not be enabled or token lacks scope")
		} else {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("secret_scanning: failed to fetch alerts: %v", err))
		}
		return
	}

	for _, alert := range secretScanningAlerts {
		var updatedAt string
		if alert.UpdatedAt != nil {
			updatedAt = alert.UpdatedAt.Format("2006-01-02T15:04:05Z")
		}
		result.Alerts = append(result.Alerts, SecurityAlert{
			AlertType:   AlertTypeSecretScanning,
			AlertNumber: alert.Number,
			State:       alert.State,
			Severity:    "high", // Secret scanning doesn't have severity, default to high
			Title:       fmt.Sprintf("%s detected", alert.SecretTypeDisplayName),
			HTMLURL:     alert.HTMLURL,
			CreatedAt:   alert.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   updatedAt,
			ResolvedAt:  formatTimePtr(alert.ResolvedAt),
		})
	}
}

// doAPI executes fn through the throttle if available, or directly if not
func (s *SecurityCollector) doAPI(ctx context.Context, operation string, fn func() error) error {
	if s.throttle != nil {
		return s.throttle.DoWithRetry(ctx, operation, fn)
	}
	return fn()
}

// normalizeSeverity converts GraphQL severity (CRITICAL, HIGH, MODERATE, LOW)
// to the REST API format (critical, high, medium, low)
func normalizeSeverity(severity string) string {
	switch strings.ToUpper(severity) {
	case "CRITICAL":
		return "critical"
	case "HIGH":
		return "high"
	case "MODERATE":
		return "medium"
	case "LOW":
		return "low"
	default:
		return strings.ToLower(severity)
	}
}

// formatTimePtr formats a time pointer to ISO 8601 string, returns nil if input is nil
func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	str := t.Format("2006-01-02T15:04:05Z")
	return &str
}
