// Package github provides GitHub status check integration for PR merge blocking
package github

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// StatusCheckManager handles GitHub status check creation and management for PR merge blocking
type StatusCheckManager struct {
	client *Client
	config *StatusCheckConfig
}

// StatusStateSuccess represents a successful status state
const StatusStateSuccess = "success"

// StatusStateFailure represents a failed status state
const StatusStateFailure = "failure"

// StatusStatePending represents a pending status state
const StatusStatePending = "pending"

// StatusStateError represents an error status state
const StatusStateError = "error"

// StatusCheckConfig holds configuration for status check management
type StatusCheckConfig struct {
	// Context settings
	ContextPrefix      string   // Prefix for all status contexts
	MainContext        string   // Main coverage context
	AdditionalContexts []string // Additional contexts to create

	// Blocking settings
	EnableBlocking    bool // Enable PR merge blocking
	BlockOnFailure    bool // Block PR merge on status failure
	BlockOnError      bool // Block PR merge on status error
	RequireAllPassing bool // Require all contexts to pass

	// Threshold settings
	CoverageThreshold      float64 // Minimum coverage threshold
	QualityThreshold       string  // Minimum quality grade threshold
	AllowThresholdOverride bool    // Allow threshold override via commit message
	AllowLabelOverride     bool    // Allow threshold override via PR labels
	MinOverrideThreshold   float64 // Minimum allowed override threshold
	MaxOverrideThreshold   float64 // Maximum allowed override threshold

	// Quality gates
	EnableQualityGates bool          // Enable quality gate checks
	QualityGates       []QualityGate // List of quality gates

	// Status descriptions
	CustomDescriptions map[string]string // Custom status descriptions
	IncludeTargetURLs  bool              // Include target URLs in statuses

	// Advanced settings
	UpdateStrategy UpdateStrategy // How to update existing statuses
	RetrySettings  RetrySettings  // Retry settings for failed requests
	StatusTimeout  time.Duration  // Timeout for status checks
}

// QualityGate represents a quality gate that must pass
type QualityGate struct {
	Name        string      // Quality gate name
	Type        GateType    // Type of quality gate
	Threshold   interface{} // Threshold value (type depends on gate type)
	Required    bool        // Whether this gate is required
	Context     string      // Status context for this gate
	Description string      // Description when gate fails
}

// GateType represents different types of quality gates
type GateType string

const (
	// GateCoveragePercentage checks minimum coverage percentage
	GateCoveragePercentage GateType = "coverage_percentage"
	// GateCoverageChange checks coverage change from base
	GateCoverageChange GateType = "coverage_change"
	// GateQualityGrade checks overall quality grade
	GateQualityGrade GateType = "quality_grade"
	// GateRiskLevel checks risk level assessment
	GateRiskLevel GateType = "risk_level"
	// GateFileCount checks number of files changed
	GateFileCount GateType = "file_count"
	// GateTrendDirection checks trend direction
	GateTrendDirection GateType = "trend_direction"
)

// UpdateStrategy defines how to update existing status checks
type UpdateStrategy string

const (
	// UpdateAlways always updates the status check
	UpdateAlways UpdateStrategy = "always" // Always update status
	// UpdateOnChange only updates if the value changed
	UpdateOnChange UpdateStrategy = "on_change" // Only update if value changed
	// UpdateOnFailure only updates on failure
	UpdateOnFailure UpdateStrategy = "on_failure" // Only update on failure
)

// RetrySettings defines retry behavior for status check requests
type RetrySettings struct {
	MaxRetries    int           // Maximum number of retries
	RetryDelay    time.Duration // Delay between retries
	BackoffFactor float64       // Exponential backoff factor
}

// StatusCheckRequest represents a request to create/update status checks
type StatusCheckRequest struct {
	// Repository information
	Owner      string
	Repository string
	CommitSHA  string

	// Coverage data
	Coverage   CoverageStatusData
	Comparison ComparisonStatusData
	Quality    QualityStatusData

	// PR information (optional)
	PRNumber   int
	Branch     string
	BaseBranch string

	// Override settings
	ForceUpdate    bool
	SkipBlocking   bool
	CustomContexts map[string]StatusInfo
}

// CoverageStatusData represents coverage data for status checks
type CoverageStatusData struct {
	Percentage        float64
	TotalStatements   int
	CoveredStatements int
	Change            float64
	Trend             string
}

// ComparisonStatusData represents comparison data for status checks
type ComparisonStatusData struct {
	BasePercentage    float64
	CurrentPercentage float64
	Difference        float64
	IsSignificant     bool
	Direction         string
}

// QualityStatusData represents quality data for status checks
type QualityStatusData struct {
	Grade      string
	Score      float64
	RiskLevel  string
	Strengths  []string
	Weaknesses []string
}

// StatusInfo represents information for a single status check
type StatusInfo struct {
	Context     string
	State       string
	Description string
	TargetURL   string
	Required    bool
}

// StatusCheckResponse represents the response from status check creation
type StatusCheckResponse struct {
	// Status results
	Statuses map[string]StatusResult

	// Overall results
	AllPassing     bool
	BlockingPR     bool
	RequiredFailed []string

	// URLs and references
	StatusURL string
	ChecksURL string

	// Metadata
	CreatedAt    time.Time
	UpdatedAt    time.Time
	TotalChecks  int
	PassedChecks int
	FailedChecks int
	ErrorChecks  int
}

// StatusResult represents the result of creating a single status
type StatusResult struct {
	Context     string
	State       string
	Description string
	TargetURL   string
	Success     bool
	Error       error
	Required    bool
	Blocking    bool
}

// NewStatusCheckManager creates a new status check manager
func NewStatusCheckManager(client *Client, config *StatusCheckConfig) *StatusCheckManager {
	if config == nil {
		config = &StatusCheckConfig{
			ContextPrefix:          "gofortress",
			MainContext:            "coverage/total",
			AdditionalContexts:     []string{"coverage/trend", "coverage/quality"},
			EnableBlocking:         true,
			BlockOnFailure:         true,
			BlockOnError:           false,
			RequireAllPassing:      false,
			CoverageThreshold:      80.0,
			QualityThreshold:       "C",
			AllowThresholdOverride: true,
			AllowLabelOverride:     false,
			MinOverrideThreshold:   50.0,
			MaxOverrideThreshold:   95.0,
			EnableQualityGates:     true,
			IncludeTargetURLs:      true,
			UpdateStrategy:         UpdateAlways,
			StatusTimeout:          30 * time.Second,
			RetrySettings: RetrySettings{
				MaxRetries:    3,
				RetryDelay:    1 * time.Second,
				BackoffFactor: 2.0,
			},
		}

		// Set up default quality gates
		config.QualityGates = []QualityGate{
			{
				Name:        "Coverage Threshold",
				Type:        GateCoveragePercentage,
				Threshold:   config.CoverageThreshold,
				Required:    true,
				Context:     "coverage/threshold",
				Description: "Coverage must meet minimum threshold",
			},
			{
				Name:        "Quality Grade",
				Type:        GateQualityGrade,
				Threshold:   config.QualityThreshold,
				Required:    false,
				Context:     "coverage/quality",
				Description: "Code quality must meet minimum grade",
			},
		}
	}

	return &StatusCheckManager{
		client: client,
		config: config,
	}
}

// CreateStatusChecks creates comprehensive status checks for a commit
func (m *StatusCheckManager) CreateStatusChecks(ctx context.Context, request *StatusCheckRequest) (*StatusCheckResponse, error) {
	response := &StatusCheckResponse{
		Statuses:  make(map[string]StatusResult),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Collect all status checks to create
	statusChecks := m.buildStatusChecks(ctx, request)

	// Create each status check
	for context, statusInfo := range statusChecks {
		result := m.createSingleStatus(ctx, request, context, statusInfo)
		response.Statuses[context] = result
		response.TotalChecks++

		if result.Success {
			switch result.State {
			case StatusStateSuccess:
				response.PassedChecks++
			case StatusStateFailure:
				response.FailedChecks++
				if result.Required {
					response.RequiredFailed = append(response.RequiredFailed, context)
				}
			case StatusStateError:
				response.ErrorChecks++
				if result.Required {
					response.RequiredFailed = append(response.RequiredFailed, context)
				}
			}
		} else {
			response.ErrorChecks++
			if result.Required {
				response.RequiredFailed = append(response.RequiredFailed, context)
			}
		}
	}

	// Determine overall results
	response.AllPassing = response.FailedChecks == 0 && response.ErrorChecks == 0
	response.BlockingPR = m.shouldBlockPR(response, request)

	// Build status URLs
	response.StatusURL = fmt.Sprintf("https://github.com/%s/%s/commit/%s",
		request.Owner, request.Repository, request.CommitSHA)
	response.ChecksURL = fmt.Sprintf("%s/checks", response.StatusURL)

	return response, nil
}

// buildStatusChecks builds all status checks that should be created
func (m *StatusCheckManager) buildStatusChecks(ctx context.Context, request *StatusCheckRequest) map[string]StatusInfo {
	statuses := make(map[string]StatusInfo)

	// Main coverage status
	mainContext := m.buildContext(m.config.MainContext)
	mainStatus := m.buildMainCoverageStatus(ctx, request)
	statuses[mainContext] = mainStatus

	// Additional contexts
	for _, additionalContext := range m.config.AdditionalContexts {
		context := m.buildContext(additionalContext)
		status := m.buildAdditionalStatus(request, additionalContext)
		statuses[context] = status
	}

	// Quality gate statuses
	if m.config.EnableQualityGates {
		for _, gate := range m.config.QualityGates {
			context := m.buildContext(gate.Context)
			status := m.buildQualityGateStatus(request, gate)
			statuses[context] = status
		}
	}

	// Custom contexts from request
	for context, statusInfo := range request.CustomContexts {
		fullContext := m.buildContext(context)
		statuses[fullContext] = statusInfo
	}

	return statuses
}

// parseCoverageOverrideFromLabels extracts coverage threshold override from PR labels
func (m *StatusCheckManager) parseCoverageOverrideFromLabels(labels []Label) (float64, bool) {
	if !m.config.AllowLabelOverride {
		return 0, false
	}

	// Define patterns for coverage override labels
	specificThresholdPattern := regexp.MustCompile(`^coverage-override-(\d+)$`)
	genericOverridePattern := regexp.MustCompile(`^coverage-override$`)

	var foundOverride bool
	var overrideThreshold float64

	for _, label := range labels {
		// Check for specific threshold labels (e.g., "coverage-override-70")
		if matches := specificThresholdPattern.FindStringSubmatch(label.Name); len(matches) > 1 {
			if threshold, err := strconv.ParseFloat(matches[1], 64); err == nil {
				// Validate threshold is within allowed bounds
				if m.config.MinOverrideThreshold > 0 && threshold < m.config.MinOverrideThreshold {
					continue // Skip thresholds below minimum
				}
				if m.config.MaxOverrideThreshold > 0 && threshold > m.config.MaxOverrideThreshold {
					continue // Skip thresholds above maximum
				}
				overrideThreshold = threshold
				foundOverride = true
				break // Use the first valid specific threshold found
			}
		}
		// Check for generic override label (uses a default lower threshold)
		if genericOverridePattern.MatchString(label.Name) {
			// Set a reasonable default override threshold (e.g., 60%)
			defaultOverride := 60.0
			if m.config.MinOverrideThreshold > 0 && defaultOverride < m.config.MinOverrideThreshold {
				defaultOverride = m.config.MinOverrideThreshold
			}
			overrideThreshold = defaultOverride
			foundOverride = true
			// Don't break here - specific thresholds take precedence
		}
	}

	return overrideThreshold, foundOverride
}

// buildMainCoverageStatus builds the main coverage status
func (m *StatusCheckManager) buildMainCoverageStatus(ctx context.Context, request *StatusCheckRequest) StatusInfo {
	coverage := request.Coverage.Percentage
	threshold := m.config.CoverageThreshold

	// Check for threshold override via PR labels
	if m.config.AllowLabelOverride && request.PRNumber > 0 {
		// Fetch PR information to get labels
		if pr, err := m.client.GetPullRequest(ctx, request.Owner, request.Repository, request.PRNumber); err == nil {
			if overrideThreshold, hasOverride := m.parseCoverageOverrideFromLabels(pr.Labels); hasOverride {
				threshold = overrideThreshold
			}
		}
		// Silently continue if PR fetch fails - use default threshold
	}

	// Legacy support: Check for threshold override in commit message or request
	if m.config.AllowThresholdOverride {
		// Implementation would check commit message for override patterns
		// For now, using the configured threshold (or label override if applied above)
		_ = threshold // Explicitly acknowledge we're not overriding for now
	}

	var state string
	var description string

	// Determine if threshold was overridden
	isOverridden := threshold != m.config.CoverageThreshold
	overrideIndicator := ""
	if isOverridden {
		overrideIndicator = " [override]"
	}

	if coverage >= threshold {
		state = StatusStateSuccess
		description = fmt.Sprintf("Coverage: %.1f%% ✅ (≥ %.1f%%%s)", coverage, threshold, overrideIndicator)
	} else {
		if m.config.BlockOnFailure {
			state = StatusStateFailure
		} else {
			state = StatusStateSuccess
		}
		description = fmt.Sprintf("Coverage: %.1f%% ⚠️ (< %.1f%% threshold%s)", coverage, threshold, overrideIndicator)
	}

	// Add trend information if available
	if request.Coverage.Change != 0 {
		changeStr := fmt.Sprintf("%+.1f%%", request.Coverage.Change)
		description = fmt.Sprintf("%s, %s", description, changeStr)
	}

	targetURL := ""
	if m.config.IncludeTargetURLs {
		targetURL = fmt.Sprintf("https://%s.github.io/%s/coverage/", request.Owner, request.Repository)
		if request.PRNumber > 0 {
			targetURL = fmt.Sprintf("%spr/%d/", targetURL, request.PRNumber)
		}
	}

	return StatusInfo{
		Context:     m.config.MainContext,
		State:       state,
		Description: description,
		TargetURL:   targetURL,
		Required:    true,
	}
}

// buildAdditionalStatus builds additional status checks
func (m *StatusCheckManager) buildAdditionalStatus(request *StatusCheckRequest, contextType string) StatusInfo {
	switch {
	case strings.Contains(contextType, "trend"):
		return m.buildTrendStatus(request)
	case strings.Contains(contextType, "quality"):
		return m.buildQualityStatus(request)
	case strings.Contains(contextType, "comparison"):
		return m.buildComparisonStatus(request)
	default:
		return m.buildGenericStatus(request, contextType)
	}
}

// buildTrendStatus builds trend-based status check
func (m *StatusCheckManager) buildTrendStatus(request *StatusCheckRequest) StatusInfo {
	var state string
	var description string

	change := request.Coverage.Change
	trend := request.Coverage.Trend

	switch {
	case change > 1.0:
		state = StatusStateSuccess
		description = fmt.Sprintf("📈 Coverage improved by %.1f%%", change)
	case change < -1.0:
		state = StatusFailure
		description = fmt.Sprintf("📉 Coverage decreased by %.1f%%", math.Abs(change))
	default:
		state = StatusStateSuccess
		description = fmt.Sprintf("📊 Coverage stable (%+.1f%%)", change)
	}

	if trend != "" {
		description = fmt.Sprintf("%s (%s trend)", description, trend)
	}

	return StatusInfo{
		Context:     "coverage/trend",
		State:       state,
		Description: description,
		TargetURL:   "",
		Required:    false,
	}
}

// buildQualityStatus builds quality-based status check
func (m *StatusCheckManager) buildQualityStatus(request *StatusCheckRequest) StatusInfo {
	grade := request.Quality.Grade
	score := request.Quality.Score
	riskLevel := request.Quality.RiskLevel

	var state string
	var description string

	// Determine state based on grade and risk
	switch grade {
	case "A+", "A", "B+", "B":
		state = StatusStateSuccess
		description = fmt.Sprintf("🏆 Quality Grade: %s (%.0f/100)", grade, score)
	case "C":
		state = StatusStateSuccess
		description = fmt.Sprintf("⚠️ Quality Grade: %s (%.0f/100)", grade, score)
	case "D", "F":
		state = StatusFailure
		description = fmt.Sprintf("🚨 Quality Grade: %s (%.0f/100)", grade, score)
	default:
		state = StatusStatePending
		description = fmt.Sprintf("📊 Quality Score: %.0f/100", score)
	}

	if riskLevel != "" && riskLevel != "low" {
		description = fmt.Sprintf("%s, %s risk", description, riskLevel)
	}

	return StatusInfo{
		Context:     "coverage/quality",
		State:       state,
		Description: description,
		TargetURL:   "",
		Required:    false,
	}
}

// buildComparisonStatus builds comparison-based status check
func (m *StatusCheckManager) buildComparisonStatus(request *StatusCheckRequest) StatusInfo {
	base := request.Comparison.BasePercentage
	current := request.Comparison.CurrentPercentage
	diff := request.Comparison.Difference

	var state string
	var description string

	if diff > 0.1 {
		state = StatusStateSuccess
		description = fmt.Sprintf("📈 +%.1f%% vs base (%.1f%% → %.1f%%)", diff, base, current)
	} else if diff < -0.1 {
		state = StatusFailure
		description = fmt.Sprintf("📉 %.1f%% vs base (%.1f%% → %.1f%%)", diff, base, current)
	} else {
		state = StatusStateSuccess
		description = fmt.Sprintf("📊 ±0.0%% vs base (%.1f%%)", current)
	}

	return StatusInfo{
		Context:     "coverage/comparison",
		State:       state,
		Description: description,
		TargetURL:   "",
		Required:    false,
	}
}

// buildQualityGateStatus builds status for a quality gate
func (m *StatusCheckManager) buildQualityGateStatus(request *StatusCheckRequest, gate QualityGate) StatusInfo {
	var state string
	var description string

	passed := m.evaluateQualityGate(request, gate)

	if passed {
		state = StatusStateSuccess
		description = fmt.Sprintf("✅ %s: Passed", gate.Name)
	} else {
		if gate.Required {
			state = StatusStateFailure
		} else {
			state = StatusStateSuccess
		}
		description = fmt.Sprintf("❌ %s: %s", gate.Name, gate.Description)
	}

	return StatusInfo{
		Context:     gate.Context,
		State:       state,
		Description: description,
		TargetURL:   "",
		Required:    gate.Required,
	}
}

// buildGenericStatus builds a generic status check
func (m *StatusCheckManager) buildGenericStatus(request *StatusCheckRequest, contextType string) StatusInfo {
	return StatusInfo{
		Context:     contextType,
		State:       StatusSuccess,
		Description: fmt.Sprintf("Coverage: %.1f%%", request.Coverage.Percentage),
		TargetURL:   "",
		Required:    false,
	}
}

// evaluateQualityGate evaluates whether a quality gate passes
func (m *StatusCheckManager) evaluateQualityGate(request *StatusCheckRequest, gate QualityGate) bool {
	switch gate.Type {
	case GateCoveragePercentage:
		if threshold, ok := gate.Threshold.(float64); ok {
			return request.Coverage.Percentage >= threshold
		}

	case GateCoverageChange:
		if threshold, ok := gate.Threshold.(float64); ok {
			return request.Coverage.Change >= threshold
		}

	case GateQualityGrade:
		if minGrade, ok := gate.Threshold.(string); ok {
			return m.compareGrades(request.Quality.Grade, minGrade) >= 0
		}

	case GateRiskLevel:
		if maxRisk, ok := gate.Threshold.(string); ok {
			return m.compareRiskLevels(request.Quality.RiskLevel, maxRisk) <= 0
		}

	case GateTrendDirection:
		if expectedDirection, ok := gate.Threshold.(string); ok {
			return request.Comparison.Direction == expectedDirection
		}
	}

	return true // Default to passing if evaluation fails
}

// createSingleStatus creates a single status check
func (m *StatusCheckManager) createSingleStatus(ctx context.Context, request *StatusCheckRequest, context string, statusInfo StatusInfo) StatusResult {
	statusReq := &StatusRequest{
		State:       statusInfo.State,
		TargetURL:   statusInfo.TargetURL,
		Description: statusInfo.Description,
		Context:     context,
	}

	// Apply retry logic
	var err error
	for attempt := 0; attempt <= m.config.RetrySettings.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			delay := time.Duration(float64(m.config.RetrySettings.RetryDelay) *
				math.Pow(m.config.RetrySettings.BackoffFactor, float64(attempt-1)))
			time.Sleep(delay)
		}

		err = m.client.CreateStatus(ctx, request.Owner, request.Repository, request.CommitSHA, statusReq)
		if err == nil {
			break
		}
	}

	return StatusResult{
		Context:     context,
		State:       statusInfo.State,
		Description: statusInfo.Description,
		TargetURL:   statusInfo.TargetURL,
		Success:     err == nil,
		Error:       err,
		Required:    statusInfo.Required,
		Blocking:    statusInfo.Required && m.config.EnableBlocking,
	}
}

// shouldBlockPR determines if the PR should be blocked based on status results
func (m *StatusCheckManager) shouldBlockPR(response *StatusCheckResponse, request *StatusCheckRequest) bool {
	if !m.config.EnableBlocking || request.SkipBlocking {
		return false
	}

	// Block if any required checks failed
	if len(response.RequiredFailed) > 0 {
		return true
	}

	// Block if require all passing is enabled and any check failed
	if m.config.RequireAllPassing && (response.FailedChecks > 0 || response.ErrorChecks > 0) {
		return true
	}

	return false
}

// Helper methods

func (m *StatusCheckManager) buildContext(context string) string {
	if m.config.ContextPrefix != "" {
		return fmt.Sprintf("%s/%s", m.config.ContextPrefix, context)
	}
	return context
}

func (m *StatusCheckManager) compareGrades(grade1, grade2 string) int {
	gradeValues := map[string]int{
		"A+": 6, "A": 5, "B+": 4, "B": 3, "C": 2, "D": 1, "F": 0,
	}

	val1, ok1 := gradeValues[grade1]
	val2, ok2 := gradeValues[grade2]

	if !ok1 || !ok2 {
		return 0
	}

	return val1 - val2
}

func (m *StatusCheckManager) compareRiskLevels(risk1, risk2 string) int {
	riskValues := map[string]int{
		"low": 1, "medium": 2, "high": 3, "critical": 4,
	}

	val1, ok1 := riskValues[risk1]
	val2, ok2 := riskValues[risk2]

	if !ok1 || !ok2 {
		return 0
	}

	return val1 - val2
}

// GetStatusCheckSummary returns a summary of current status checks
func (m *StatusCheckManager) GetStatusCheckSummary(_ context.Context, _, _, commitSHA string) (map[string]interface{}, error) {
	// This would typically query the GitHub API to get current status checks
	// For now, returning a placeholder structure

	summary := map[string]interface{}{
		"commit_sha":     commitSHA,
		"total_checks":   0,
		"passed_checks":  0,
		"failed_checks":  0,
		"pending_checks": 0,
		"blocking":       false,
		"contexts":       []string{},
		"last_updated":   time.Now(),
	}

	return summary, nil
}
