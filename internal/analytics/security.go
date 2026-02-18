package analytics

import (
	"context"
	"fmt"
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

// CollectAlerts fetches all security alerts for multiple repositories concurrently
// Returns a map of repo full name to list of alerts
func (s *SecurityCollector) CollectAlerts(ctx context.Context, repos []gh.RepoInfo) (map[string][]SecurityAlert, error) {
	if len(repos) == 0 {
		return make(map[string][]SecurityAlert), nil
	}

	if s.logger != nil {
		s.logger.WithField("repo_count", len(repos)).Info("Starting concurrent security alert collection")
	}

	// Result map with mutex protection
	results := make(map[string][]SecurityAlert)
	var resultMu sync.Mutex

	// Create errgroup with bounded concurrency
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(SecurityWorkerLimit)

	// Spawn workers for each repository
	for _, repo := range repos {
		g.Go(func() error {
			alerts, err := s.collectRepoAlerts(ctx, repo.FullName)
			if err != nil {
				if s.logger != nil {
					s.logger.WithError(err).WithField("repo", repo.FullName).Warn("Failed to collect security alerts")
				}
				// Don't fail the entire operation for a single repo error
				return nil
			}

			if len(alerts) > 0 {
				resultMu.Lock()
				results[repo.FullName] = alerts
				resultMu.Unlock()

				if s.logger != nil {
					s.logger.WithFields(logrus.Fields{
						"repo":        repo.FullName,
						"alert_count": len(alerts),
					}).Debug("Collected security alerts")
				}
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
		for _, alerts := range results {
			totalAlerts += len(alerts)
		}
		s.logger.WithFields(logrus.Fields{
			"repos_with_alerts": len(results),
			"total_alerts":      totalAlerts,
		}).Info("Security alert collection complete")
	}

	return results, nil
}

// collectRepoAlerts fetches all security alert types for a single repository
func (s *SecurityCollector) collectRepoAlerts(ctx context.Context, repo string) ([]SecurityAlert, error) {
	var allAlerts []SecurityAlert

	// Collect Dependabot alerts
	var dependabotAlerts []gh.DependabotAlert
	err := s.doAPI(ctx, "dependabot-alerts:"+repo, func() error {
		var apiErr error
		dependabotAlerts, apiErr = s.ghClient.GetDependabotAlerts(ctx, repo)
		return apiErr
	})
	if err != nil {
		return nil, fmt.Errorf("dependabot alerts: %w", err)
	}
	for _, alert := range dependabotAlerts {
		allAlerts = append(allAlerts, SecurityAlert{
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

	// Collect Code Scanning alerts
	var codeScanningAlerts []gh.CodeScanningAlert
	err = s.doAPI(ctx, "code-scanning-alerts:"+repo, func() error {
		var apiErr error
		codeScanningAlerts, apiErr = s.ghClient.GetCodeScanningAlerts(ctx, repo)
		return apiErr
	})
	if err != nil {
		return nil, fmt.Errorf("code scanning alerts: %w", err)
	}
	for _, alert := range codeScanningAlerts {
		allAlerts = append(allAlerts, SecurityAlert{
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

	// Collect Secret Scanning alerts
	var secretScanningAlerts []gh.SecretScanningAlert
	err = s.doAPI(ctx, "secret-scanning-alerts:"+repo, func() error {
		var apiErr error
		secretScanningAlerts, apiErr = s.ghClient.GetSecretScanningAlerts(ctx, repo)
		return apiErr
	})
	if err != nil {
		return nil, fmt.Errorf("secret scanning alerts: %w", err)
	}
	for _, alert := range secretScanningAlerts {
		var updatedAt string
		if alert.UpdatedAt != nil {
			updatedAt = alert.UpdatedAt.Format("2006-01-02T15:04:05Z")
		}
		allAlerts = append(allAlerts, SecurityAlert{
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

	return allAlerts, nil
}

// doAPI executes fn through the throttle if available, or directly if not
func (s *SecurityCollector) doAPI(ctx context.Context, operation string, fn func() error) error {
	if s.throttle != nil {
		return s.throttle.DoWithRetry(ctx, operation, fn)
	}
	return fn()
}

// formatTimePtr formats a time pointer to ISO 8601 string, returns nil if input is nil
func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	str := t.Format("2006-01-02T15:04:05Z")
	return &str
}
