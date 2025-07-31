// Package github provides enhanced PR comment management for coverage reporting
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/logger"
)

// PRCommentManager handles intelligent PR comment management with anti-spam and lifecycle features
type PRCommentManager struct {
	client *Client
	config *PRCommentConfig
	logger logger.Logger
}

// PRCommentConfig holds configuration for PR comment management
type PRCommentConfig struct {
	// Anti-spam settings
	MinUpdateIntervalMinutes int    // Minimum time between comment updates
	MaxCommentsPerPR         int    // Maximum comments allowed per PR
	CommentSignature         string // Unique signature to identify our comments

	// Template settings
	IncludeTrend           bool // Include trend analysis in comments
	IncludeCoverageDetails bool // Include detailed coverage breakdown
	IncludeFileAnalysis    bool // Include file-level coverage analysis
	ShowCoverageHistory    bool // Show historical coverage data

	// Badge settings
	GeneratePRBadges bool   // Generate PR-specific badges
	BadgeStyle       string // Badge style (flat, flat-square, for-the-badge)

	// Status check settings
	EnableStatusChecks  bool // Enable GitHub status checks
	FailBelowThreshold  bool // Fail status if below threshold
	BlockMergeOnFailure bool // Block PR merge on coverage failure
}

// CoverageComparison represents coverage comparison between base and PR branches
type CoverageComparison struct {
	BaseCoverage     CoverageData `json:"base_coverage"`
	PRCoverage       CoverageData `json:"pr_coverage"`
	Difference       float64      `json:"difference"`
	TrendAnalysis    TrendData    `json:"trend_analysis"`
	FileChanges      []FileChange `json:"file_changes"`
	SignificantFiles []string     `json:"significant_files"`
}

// CoverageData represents coverage information for a specific commit
type CoverageData struct {
	Percentage        float64   `json:"percentage"`
	TotalStatements   int       `json:"total_statements"`
	CoveredStatements int       `json:"covered_statements"`
	CommitSHA         string    `json:"commit_sha"`
	Branch            string    `json:"branch"`
	Timestamp         time.Time `json:"timestamp"`
}

// TrendData represents trend analysis information
type TrendData struct {
	Direction        string  `json:"direction"` // "up", "down", "stable"
	Magnitude        string  `json:"magnitude"` // "significant", "moderate", "minor"
	PercentageChange float64 `json:"percentage_change"`
	Momentum         string  `json:"momentum"` // "accelerating", "steady", "decelerating"
}

// FileChange represents coverage change for a specific file
type FileChange struct {
	Filename      string  `json:"filename"`
	BaseCoverage  float64 `json:"base_coverage"`
	PRCoverage    float64 `json:"pr_coverage"`
	Difference    float64 `json:"difference"`
	LinesAdded    int     `json:"lines_added"`
	LinesRemoved  int     `json:"lines_removed"`
	IsSignificant bool    `json:"is_significant"`
}

// CommentMetadata represents metadata stored in comment for tracking
type CommentMetadata struct {
	Signature      string    `json:"signature"`
	CommentVersion string    `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	LastUpdatedAt  time.Time `json:"last_updated_at"`
	UpdateCount    int       `json:"update_count"`
	PRNumber       int       `json:"pr_number"`
	BaseSHA        string    `json:"base_sha"`
	HeadSHA        string    `json:"head_sha"`
}

// PRCommentResponse represents the response from creating/updating a PR comment
type PRCommentResponse struct {
	CommentID      int                `json:"comment_id"`
	Action         string             `json:"action"` // "created", "updated", "skipped"
	Reason         string             `json:"reason"` // Reason for action taken
	Metadata       CommentMetadata    `json:"metadata"`
	CoverageData   CoverageComparison `json:"coverage_data"`
	BadgeURLs      map[string]string  `json:"badge_urls"` // PR-specific badge URLs
	StatusCheckURL string             `json:"status_check_url"`
}

// NewPRCommentManager creates a new PR comment manager with configuration
func NewPRCommentManager(client *Client, config *PRCommentConfig) *PRCommentManager {
	if config == nil {
		config = &PRCommentConfig{
			MinUpdateIntervalMinutes: 5,
			MaxCommentsPerPR:         1,
			CommentSignature:         "gofortress-coverage-v2",
			IncludeTrend:             true,
			IncludeCoverageDetails:   true,
			IncludeFileAnalysis:      false,
			ShowCoverageHistory:      true,
			GeneratePRBadges:         true,
			BadgeStyle:               "flat",
			EnableStatusChecks:       true,
			FailBelowThreshold:       true,
			BlockMergeOnFailure:      false,
		}
	}

	return &PRCommentManager{
		client: client,
		config: config,
		logger: logger.NewFromEnv(),
	}
}

// CreateOrUpdatePRComment creates or intelligently updates a PR comment with coverage information
func (m *PRCommentManager) CreateOrUpdatePRComment(ctx context.Context, owner, repo string, prNumber int, comparison *CoverageComparison) (*PRCommentResponse, error) {
	// Get PR information first
	pr, err := m.client.GetPullRequest(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR information: %w", err)
	}

	// Find existing coverage comments
	existingComments, err := m.findExistingCoverageComments(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing comments: %w", err)
	}

	// Determine action based on anti-spam rules
	action, shouldUpdate, reason := m.determineCommentAction(existingComments, comparison)

	if !shouldUpdate {
		return &PRCommentResponse{
			Action:       action,
			Reason:       reason,
			CoverageData: *comparison,
		}, nil
	}

	// Generate comment content
	commentBody := m.generateEnhancedComment(comparison, owner, repo, prNumber, pr.Head.SHA)

	var comment *Comment
	var commentID int

	if len(existingComments) > 0 {
		// Update existing comment
		comment, err = m.client.updateComment(ctx, owner, repo, existingComments[0].ID, commentBody)
		if err != nil {
			return nil, fmt.Errorf("failed to update comment: %w", err)
		}
		commentID = comment.ID
		action = "updated"
	} else {
		// Create new comment
		comment, err = m.client.createComment(ctx, owner, repo, prNumber, commentBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create comment: %w", err)
		}
		commentID = comment.ID
		action = "created"
	}

	// Generate PR-specific badges if enabled
	badgeURLs := make(map[string]string)
	if m.config.GeneratePRBadges {
		badgeURLs = m.generatePRBadgeURLs(owner, repo, prNumber, comparison.PRCoverage.Percentage)
	}

	// Create status check if enabled
	statusCheckURL := ""
	if m.config.EnableStatusChecks {
		err = m.createCoverageStatusCheck(ctx, owner, repo, pr.Head.SHA, comparison)
		if err != nil {
			// Don't fail the entire operation if status check fails
			m.logger.WithError(err).WithFields(map[string]interface{}{
				"owner":     owner,
				"repo":      repo,
				"sha":       pr.Head.SHA,
				"operation": "status_check_creation",
			}).Warn("Failed to create GitHub status check")
		} else {
			statusCheckURL = fmt.Sprintf("https://github.com/%s/%s/commit/%s/checks", owner, repo, pr.Head.SHA)
		}
	}

	// Prepare metadata
	metadata := CommentMetadata{
		Signature:      m.config.CommentSignature,
		CommentVersion: "2.0",
		CreatedAt:      time.Now(),
		LastUpdatedAt:  time.Now(),
		UpdateCount:    1,
		PRNumber:       prNumber,
		BaseSHA:        "", // Would need to get base SHA from PR
		HeadSHA:        pr.Head.SHA,
	}

	if len(existingComments) > 0 {
		// Extract existing metadata if available
		if existingMeta := m.extractCommentMetadata(existingComments[0].Body); existingMeta != nil {
			metadata.CreatedAt = existingMeta.CreatedAt
			metadata.UpdateCount = existingMeta.UpdateCount + 1
		}
	}

	return &PRCommentResponse{
		CommentID:      commentID,
		Action:         action,
		Reason:         reason,
		Metadata:       metadata,
		CoverageData:   *comparison,
		BadgeURLs:      badgeURLs,
		StatusCheckURL: statusCheckURL,
	}, nil
}

// findExistingCoverageComments finds existing coverage comments by signature
func (m *PRCommentManager) findExistingCoverageComments(ctx context.Context, owner, repo string, prNumber int) ([]Comment, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", m.client.baseURL, owner, repo, prNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+m.client.token)
	req.Header.Set("User-Agent", m.client.config.UserAgent)

	resp, err := m.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: %d", ErrGitHubAPIError, resp.StatusCode)
	}

	var allComments []Comment
	if err := json.NewDecoder(resp.Body).Decode(&allComments); err != nil {
		return nil, fmt.Errorf("failed to decode comments: %w", err)
	}

	// Filter for our coverage comments
	var coverageComments []Comment
	for _, comment := range allComments {
		if m.isCoverageComment(comment.Body) {
			coverageComments = append(coverageComments, comment)
		}
	}

	return coverageComments, nil
}

// isCoverageComment checks if a comment is our coverage comment by signature
func (m *PRCommentManager) isCoverageComment(body string) bool {
	signatures := []string{
		m.config.CommentSignature,
		"<!-- gofortress-coverage -->",
		"<!-- coverage-comment -->",
		"## 📊 Coverage Report",
		"Generated by GoFortress Coverage",
	}

	for _, signature := range signatures {
		if strings.Contains(body, signature) {
			return true
		}
	}

	return false
}

// determineCommentAction determines what action to take based on anti-spam rules
func (m *PRCommentManager) determineCommentAction(existingComments []Comment, comparison *CoverageComparison) (string, bool, string) {
	if len(existingComments) == 0 {
		return "create", true, "No existing coverage comment found"
	}

	if len(existingComments) > m.config.MaxCommentsPerPR {
		return "skipped", false, fmt.Sprintf("Maximum comments per PR (%d) exceeded", m.config.MaxCommentsPerPR)
	}

	// Check time-based anti-spam
	lastComment := existingComments[len(existingComments)-1]
	lastUpdateTime, err := time.Parse(time.RFC3339, lastComment.UpdatedAt)
	if err == nil {
		timeSinceUpdate := time.Since(lastUpdateTime)
		minInterval := time.Duration(m.config.MinUpdateIntervalMinutes) * time.Minute

		if timeSinceUpdate < minInterval {
			return "skipped", false, fmt.Sprintf("Minimum update interval (%v) not reached", minInterval)
		}
	}

	// Check for significant changes
	if m.hasSignificantCoverageChange(comparison) {
		return "update", true, "Significant coverage change detected"
	}

	return "update", true, "Coverage data updated"
}

// hasSignificantCoverageChange determines if the coverage change is significant enough to warrant an update
func (m *PRCommentManager) hasSignificantCoverageChange(comparison *CoverageComparison) bool {
	// Consider changes significant if:
	// 1. Coverage difference > 1%
	// 2. Trend direction changed
	// 3. New files with low coverage

	if comparison.Difference > 1.0 || comparison.Difference < -1.0 {
		return true
	}

	if comparison.TrendAnalysis.Magnitude == "significant" {
		return true
	}

	for _, fileChange := range comparison.FileChanges {
		if fileChange.IsSignificant {
			return true
		}
	}

	return false
}

// generateEnhancedComment generates the enhanced comment body with all features
func (m *PRCommentManager) generateEnhancedComment(comparison *CoverageComparison, owner, repo string, prNumber int, headSHA string) string {
	var comment strings.Builder

	// Add signature and metadata
	comment.WriteString(fmt.Sprintf("<!-- %s -->\n", m.config.CommentSignature))
	comment.WriteString("<!-- metadata: ")
	metadata := CommentMetadata{
		Signature:      m.config.CommentSignature,
		CommentVersion: "2.0",
		CreatedAt:      time.Now(),
		LastUpdatedAt:  time.Now(),
		PRNumber:       prNumber,
		HeadSHA:        headSHA,
	}
	metadataJSON, _ := json.Marshal(metadata) //nolint:errchkjson // Metadata marshaling is optional
	comment.Write(metadataJSON)
	comment.WriteString(" -->\n\n")

	// Header with trend emoji
	trendEmoji := m.getTrendEmoji(comparison.TrendAnalysis.Direction)
	comment.WriteString(fmt.Sprintf("## %s Coverage Report\n\n", trendEmoji))

	// Main coverage summary
	comment.WriteString(fmt.Sprintf("**Overall Coverage: %.1f%%** %s\n\n",
		comparison.PRCoverage.Percentage,
		m.getPercentageEmoji(comparison.PRCoverage.Percentage)))

	// Coverage change analysis
	if comparison.Difference != 0 {
		changeDirection := "increased"
		changeEmoji := "📈"
		if comparison.Difference < 0 {
			changeDirection = "decreased"
			changeEmoji = "📉"
		}

		comment.WriteString(fmt.Sprintf("%s Coverage %s by **%.2f%%** (%.1f%% → %.1f%%)\n\n",
			changeEmoji, changeDirection, comparison.Difference,
			comparison.BaseCoverage.Percentage, comparison.PRCoverage.Percentage))
	} else {
		comment.WriteString("📊 Coverage remained stable\n\n")
	}

	// Coverage details
	if m.config.IncludeCoverageDetails {
		comment.WriteString("### 📋 Coverage Details\n\n")
		comment.WriteString("| Metric | Base | Current | Change |\n")
		comment.WriteString("|--------|------|---------|--------|\n")
		comment.WriteString(fmt.Sprintf("| **Percentage** | %.1f%% | %.1f%% | %+.1f%% |\n",
			comparison.BaseCoverage.Percentage, comparison.PRCoverage.Percentage, comparison.Difference))
		comment.WriteString(fmt.Sprintf("| **Covered** | %d | %d | %+d |\n",
			comparison.BaseCoverage.CoveredStatements, comparison.PRCoverage.CoveredStatements,
			comparison.PRCoverage.CoveredStatements-comparison.BaseCoverage.CoveredStatements))
		comment.WriteString(fmt.Sprintf("| **Total** | %d | %d | %+d |\n\n",
			comparison.BaseCoverage.TotalStatements, comparison.PRCoverage.TotalStatements,
			comparison.PRCoverage.TotalStatements-comparison.BaseCoverage.TotalStatements))
	}

	// Trend analysis
	if m.config.IncludeTrend && comparison.TrendAnalysis.Direction != "" {
		comment.WriteString("### 📈 Trend Analysis\n\n")
		comment.WriteString(fmt.Sprintf("- **Direction**: %s %s\n",
			m.getTrendEmoji(comparison.TrendAnalysis.Direction),
			comparison.TrendAnalysis.Direction))
		comment.WriteString(fmt.Sprintf("- **Magnitude**: %s\n", comparison.TrendAnalysis.Magnitude))
		comment.WriteString(fmt.Sprintf("- **Momentum**: %s\n\n", comparison.TrendAnalysis.Momentum))
	}

	// File-level analysis
	if m.config.IncludeFileAnalysis && len(comparison.FileChanges) > 0 {
		comment.WriteString("### 📁 File Changes\n\n")

		significantChanges := make([]FileChange, 0)
		for _, change := range comparison.FileChanges {
			if change.IsSignificant {
				significantChanges = append(significantChanges, change)
			}
		}

		if len(significantChanges) > 0 {
			comment.WriteString("| File | Base | Current | Change |\n")
			comment.WriteString("|------|------|---------|--------|\n")

			for _, change := range significantChanges {
				if len(significantChanges) > 10 {
					break // Limit to prevent overly long comments
				}
				comment.WriteString(fmt.Sprintf("| `%s` | %.1f%% | %.1f%% | %+.1f%% |\n",
					change.Filename, change.BaseCoverage, change.PRCoverage, change.Difference))
			}
			comment.WriteString("\n")
		}
	}

	// PR-specific badges
	if m.config.GeneratePRBadges {
		comment.WriteString("### 🏷️ PR Coverage Badge\n\n")
		badgeURL := m.generatePRBadgeURL(owner, repo, prNumber, comparison.PRCoverage.Percentage)
		comment.WriteString(fmt.Sprintf("![PR Coverage Badge](%s)\n\n", badgeURL))
	}

	// Footer
	comment.WriteString("---\n")
	comment.WriteString(fmt.Sprintf("*Generated by [GoFortress Coverage](https://github.com/%s/%s) 🤖*\n", owner, repo))
	comment.WriteString(fmt.Sprintf("*Updated: %s*", time.Now().Format("2006-01-02 15:04:05 UTC")))

	return comment.String()
}

// getTrendEmoji returns emoji for trend direction
func (m *PRCommentManager) getTrendEmoji(direction string) string {
	switch direction {
	case "up":
		return "📈"
	case "down":
		return "📉"
	default:
		return "📊"
	}
}

// getPercentageEmoji returns emoji based on coverage percentage
func (m *PRCommentManager) getPercentageEmoji(percentage float64) string {
	switch {
	case percentage >= 90:
		return "🟢"
	case percentage >= 80:
		return "🟡"
	case percentage >= 70:
		return "🟠"
	default:
		return "🔴"
	}
}

// generatePRBadgeURLs generates PR-specific badge URLs
func (m *PRCommentManager) generatePRBadgeURLs(owner, repo string, prNumber int, _ float64) map[string]string {
	baseURL := fmt.Sprintf("https://%s.github.io/%s/coverage/pr/%d", owner, repo, prNumber)

	return map[string]string{
		"coverage": fmt.Sprintf("%s/badge-coverage.svg", baseURL),
		"trend":    fmt.Sprintf("%s/badge-trend.svg", baseURL),
		"status":   fmt.Sprintf("%s/badge-status.svg", baseURL),
	}
}

// generatePRBadgeURL generates a single PR badge URL
func (m *PRCommentManager) generatePRBadgeURL(owner, repo string, prNumber int, _ float64) string {
	return fmt.Sprintf("https://%s.github.io/%s/coverage/pr/%d/badge-coverage-flat.svg", owner, repo, prNumber)
}

// createCoverageStatusCheck creates GitHub status check for coverage
func (m *PRCommentManager) createCoverageStatusCheck(ctx context.Context, owner, repo, sha string, comparison *CoverageComparison) error {
	var state string
	var description string

	threshold := 80.0 // Default threshold, should come from config

	if comparison.PRCoverage.Percentage >= threshold {
		state = StatusSuccess
		description = fmt.Sprintf("Coverage: %.1f%% ✅", comparison.PRCoverage.Percentage)
	} else if m.config.FailBelowThreshold {
		state = StatusFailure
		description = fmt.Sprintf("Coverage: %.1f%% (below %.1f%% threshold)",
			comparison.PRCoverage.Percentage, threshold)
	} else {
		state = StatusSuccess
		description = fmt.Sprintf("Coverage: %.1f%% (below threshold but not blocking)",
			comparison.PRCoverage.Percentage)
	}

	statusReq := &StatusRequest{
		State:       state,
		TargetURL:   fmt.Sprintf("https://%s.github.io/%s/coverage/", owner, repo),
		Description: description,
		Context:     "GoFortress/Coverage-PR",
	}

	return m.client.CreateStatus(ctx, owner, repo, sha, statusReq)
}

// extractCommentMetadata extracts metadata from comment body
func (m *PRCommentManager) extractCommentMetadata(body string) *CommentMetadata {
	re := regexp.MustCompile(`<!-- metadata: (.*?) -->`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		return nil
	}

	var metadata CommentMetadata
	if err := json.Unmarshal([]byte(matches[1]), &metadata); err != nil {
		return nil
	}

	return &metadata
}

// DeletePRComments deletes all coverage comments for a PR (cleanup utility)
func (m *PRCommentManager) DeletePRComments(ctx context.Context, owner, repo string, prNumber int) error {
	existingComments, err := m.findExistingCoverageComments(ctx, owner, repo, prNumber)
	if err != nil {
		return fmt.Errorf("failed to find existing comments: %w", err)
	}

	for _, comment := range existingComments {
		url := fmt.Sprintf("%s/repos/%s/%s/issues/comments/%d", m.client.baseURL, owner, repo, comment.ID)

		req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
		if err != nil {
			continue // Skip this comment if request creation fails
		}

		req.Header.Set("Authorization", "token "+m.client.token)
		req.Header.Set("User-Agent", m.client.config.UserAgent)

		resp, err := m.client.httpClient.Do(req)
		if err != nil {
			continue // Skip this comment if deletion fails
		}
		_ = resp.Body.Close()
	}

	return nil
}

// GetPRCommentStats returns statistics about PR comments
func (m *PRCommentManager) GetPRCommentStats(ctx context.Context, owner, repo string, prNumber int) (map[string]interface{}, error) {
	existingComments, err := m.findExistingCoverageComments(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing comments: %w", err)
	}

	stats := map[string]interface{}{
		"total_comments":    len(existingComments),
		"has_comments":      len(existingComments) > 0,
		"last_update_time":  "",
		"comment_signature": m.config.CommentSignature,
	}

	if len(existingComments) > 0 {
		lastComment := existingComments[len(existingComments)-1]
		stats["last_update_time"] = lastComment.UpdatedAt
		stats["last_comment_id"] = lastComment.ID

		if metadata := m.extractCommentMetadata(lastComment.Body); metadata != nil {
			stats["update_count"] = metadata.UpdateCount
			stats["created_at"] = metadata.CreatedAt
		}
	}

	return stats, nil
}
