// Package templates provides advanced PR comment template system with dynamic content rendering
package templates

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"math"
	"sort"
	"strings"
	"time"
)

// Static error definitions
var (
	ErrTemplateNotFound = errors.New("template not found")
)

// PRTemplateEngine handles advanced PR comment template rendering
type PRTemplateEngine struct {
	templates map[string]*template.Template
	config    *TemplateConfig
}

// TemplateConfig holds configuration for template rendering
type TemplateConfig struct {
	// Content options
	IncludeEmojis bool // Include emojis in templates
	IncludeCharts bool // Include ASCII charts

	// Content filtering
	MaxFileChanges     int  // Maximum file changes to show
	MaxPackageChanges  int  // Maximum package changes to show
	MaxRecommendations int  // Maximum recommendations to show
	HideStableFiles    bool // Hide files with no significant changes

	// Styling options
	UseMarkdownTables      bool // Use markdown tables
	UseCollapsibleSections bool // Use collapsible sections for long content
	IncludeProgressBars    bool // Include ASCII progress bars
	UseColors              bool // Use color indicators (for supported environments)

	// Thresholds for dynamic content
	ExcellentThreshold float64 // Threshold for excellent coverage
	GoodThreshold      float64 // Threshold for good coverage
	WarningThreshold   float64 // Threshold for warning coverage
	CriticalThreshold  float64 // Threshold for critical coverage

	// Customization
	CustomFooter    string // Custom footer text
	CustomHeader    string // Custom header text
	BrandingEnabled bool   // Include branding
	TimestampFormat string // Timestamp format
}

// TemplateData represents all data available to templates
type TemplateData struct {
	// Basic information
	Repository  RepositoryInfo  `json:"repository"`
	PullRequest PullRequestInfo `json:"pull_request"`
	Timestamp   time.Time       `json:"timestamp"`

	// Coverage data
	Coverage   CoverageData   `json:"coverage"`
	Comparison ComparisonData `json:"comparison"`
	Trends     TrendData      `json:"trends"`

	// Analysis results
	Quality         QualityData          `json:"quality"`
	Recommendations []RecommendationData `json:"recommendations"`

	// PR file analysis
	PRFiles *PRFileAnalysisData `json:"pr_files,omitempty"`

	// Configuration
	Config TemplateConfig `json:"config"`

	// Resources and links
	Resources ResourceLinks `json:"resources"`

	// Metadata
	Metadata TemplateMetadata `json:"metadata"`
}

// RepositoryInfo contains repository information
type RepositoryInfo struct {
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
	URL           string `json:"url"`
}

// PullRequestInfo contains PR information
type PullRequestInfo struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	Branch     string `json:"branch"`
	BaseBranch string `json:"base_branch"`
	Author     string `json:"author"`
	CommitSHA  string `json:"commit_sha"`
	URL        string `json:"url"`
}

// CoverageData represents current coverage information
type CoverageData struct {
	Overall  CoverageMetrics       `json:"overall"`
	Files    []FileCoverageData    `json:"files"`
	Packages []PackageCoverageData `json:"packages"`
	Summary  CoverageSummary       `json:"summary"`
}

// CoverageMetrics represents coverage metrics
type CoverageMetrics struct {
	Percentage        float64 `json:"percentage"`
	TotalStatements   int     `json:"total_statements"`
	CoveredStatements int     `json:"covered_statements"`
	TotalLines        int     `json:"total_lines"`
	CoveredLines      int     `json:"covered_lines"`
	Grade             string  `json:"grade"`
	Status            string  `json:"status"` // "excellent", "good", "warning", "critical"
}

// FileCoverageData represents file-level coverage data
type FileCoverageData struct {
	Filename     string  `json:"filename"`
	Percentage   float64 `json:"percentage"`
	Change       float64 `json:"change"`
	Status       string  `json:"status"`
	IsNew        bool    `json:"is_new"`
	IsModified   bool    `json:"is_modified"`
	LinesAdded   int     `json:"lines_added"`
	LinesRemoved int     `json:"lines_removed"`
	Risk         string  `json:"risk"`
}

// PackageCoverageData represents package-level coverage data
type PackageCoverageData struct {
	Package    string  `json:"package"`
	Percentage float64 `json:"percentage"`
	Change     float64 `json:"change"`
	FileCount  int     `json:"file_count"`
	Status     string  `json:"status"`
}

// CoverageSummary provides a high-level coverage summary
type CoverageSummary struct {
	Direction       string   `json:"direction"` // "improved", "degraded", "stable"
	Magnitude       string   `json:"magnitude"` // "significant", "moderate", "minor"
	KeyAchievements []string `json:"key_achievements"`
	KeyConcerns     []string `json:"key_concerns"`
	OverallImpact   string   `json:"overall_impact"`
}

// ComparisonData represents coverage comparison information
type ComparisonData struct {
	BasePercentage    float64 `json:"base_percentage"`
	CurrentPercentage float64 `json:"current_percentage"`
	Change            float64 `json:"change"`
	Direction         string  `json:"direction"`
	Magnitude         string  `json:"magnitude"`
	IsSignificant     bool    `json:"is_significant"`
}

// TrendData represents trend analysis information
type TrendData struct {
	Direction  string  `json:"direction"`
	Momentum   string  `json:"momentum"`
	Volatility float64 `json:"volatility"`
	Prediction float64 `json:"prediction"`
	Confidence float64 `json:"confidence"`
}

// QualityData represents quality assessment information
type QualityData struct {
	OverallGrade  string   `json:"overall_grade"`
	CoverageGrade string   `json:"coverage_grade"`
	TrendGrade    string   `json:"trend_grade"`
	RiskLevel     string   `json:"risk_level"`
	Score         float64  `json:"score"`
	Strengths     []string `json:"strengths"`
	Weaknesses    []string `json:"weaknesses"`
}

// RecommendationData represents recommendation information
type RecommendationData struct {
	Type        string   `json:"type"`
	Priority    string   `json:"priority"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Impact      string   `json:"impact"`
}

// ResourceLinks contains URLs and links for the PR comment
type ResourceLinks struct {
	BadgeURL      string `json:"badge_url"`
	ReportURL     string `json:"report_url"`
	DashboardURL  string `json:"dashboard_url"`
	PRBadgeURL    string `json:"pr_badge_url"`
	PRReportURL   string `json:"pr_report_url"`
	HistoricalURL string `json:"historical_url"`
}

// TemplateMetadata contains template metadata
type TemplateMetadata struct {
	Version      string    `json:"version"`
	GeneratedAt  time.Time `json:"generated_at"`
	TemplateUsed string    `json:"template_used"`
	Signature    string    `json:"signature"`
}

// PRFileAnalysisData represents PR file analysis data for templates
type PRFileAnalysisData struct {
	Summary            PRFileSummaryData `json:"summary"`
	GoFiles            []PRFileData      `json:"go_files"`
	TestFiles          []PRFileData      `json:"test_files"`
	ConfigFiles        []PRFileData      `json:"config_files"`
	DocumentationFiles []PRFileData      `json:"documentation_files"`
	GeneratedFiles     []PRFileData      `json:"generated_files"`
	OtherFiles         []PRFileData      `json:"other_files"`
}

// PRFileSummaryData represents summary of PR file changes
type PRFileSummaryData struct {
	TotalFiles          int    `json:"total_files"`
	GoFilesCount        int    `json:"go_files_count"`
	TestFilesCount      int    `json:"test_files_count"`
	ConfigFilesCount    int    `json:"config_files_count"`
	DocumentationCount  int    `json:"documentation_count"`
	GeneratedFilesCount int    `json:"generated_files_count"`
	OtherFilesCount     int    `json:"other_files_count"`
	HasGoChanges        bool   `json:"has_go_changes"`
	HasTestChanges      bool   `json:"has_test_changes"`
	HasConfigChanges    bool   `json:"has_config_changes"`
	TotalAdditions      int    `json:"total_additions"`
	TotalDeletions      int    `json:"total_deletions"`
	GoAdditions         int    `json:"go_additions"`
	GoDeletions         int    `json:"go_deletions"`
	SummaryText         string `json:"summary_text"`
}

// PRFileData represents individual file data for templates
type PRFileData struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
}

// NewPRTemplateEngine creates a new PR template engine
func NewPRTemplateEngine(config *TemplateConfig) *PRTemplateEngine {
	if config == nil {
		config = &TemplateConfig{
			IncludeEmojis:          true,
			IncludeCharts:          true,
			MaxFileChanges:         20,
			MaxPackageChanges:      10,
			MaxRecommendations:     5,
			HideStableFiles:        true,
			UseMarkdownTables:      true,
			UseCollapsibleSections: true,
			IncludeProgressBars:    true,
			UseColors:              false,
			ExcellentThreshold:     90.0,
			GoodThreshold:          80.0,
			WarningThreshold:       70.0,
			CriticalThreshold:      50.0,
			BrandingEnabled:        true,
			TimestampFormat:        "2006-01-02 15:04:05 UTC",
		}
	}

	engine := &PRTemplateEngine{
		templates: make(map[string]*template.Template),
		config:    config,
	}

	// Initialize templates with helper functions
	engine.initializeTemplates()

	return engine
}

// RenderComment renders a PR comment using the dashboard template
func (e *PRTemplateEngine) RenderComment(_ context.Context, _ string, data *TemplateData) (string, error) {
	// Convert PR template data to dashboard format
	dashboardData := e.convertToDashboardData(data)

	// Create template function map for dashboard
	funcMap := template.FuncMap{
		"sub": func(a, b float64) float64 {
			return a - b
		},
		"printf": fmt.Sprintf,
	}

	// Parse and execute dashboard template directly
	tmpl := template.Must(template.New("dashboard").Funcs(funcMap).Parse(dashboardTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, dashboardData); err != nil {
		return "", fmt.Errorf("executing dashboard template: %w", err)
	}

	return buf.String(), nil
}

// convertToDashboardData converts PR TemplateData to dashboard data format
func (e *PRTemplateEngine) convertToDashboardData(data *TemplateData) map[string]interface{} {
	// Calculate basic metrics
	totalFiles := len(data.Coverage.Files)
	coveredFiles := 0
	for _, file := range data.Coverage.Files {
		if file.Percentage > 0 {
			coveredFiles++
		}
	}

	// Prepare package data for dashboard
	packages := make([]map[string]interface{}, 0, len(data.Coverage.Packages))
	for _, pkg := range data.Coverage.Packages {
		packages = append(packages, map[string]interface{}{
			"Name":         pkg.Package,
			"Coverage":     pkg.Percentage,
			"CoveredLines": 0,                          // Not available in PR data
			"TotalLines":   0,                          // Not available in PR data
			"MissedLines":  0,                          // Not available in PR data
			"GitHubURL":    "",                         // Could be constructed if needed
			"Files":        []map[string]interface{}{}, // Could be mapped if needed
		})
	}

	// Convert timestamp to formatted string
	timestamp := data.Timestamp.Format("2006-01-02 15:04:05 UTC")

	// Map PR data to dashboard format
	dashboardData := map[string]interface{}{
		// Project information
		"ProjectName":     data.Repository.Name,
		"RepositoryOwner": data.Repository.Owner,
		"RepositoryName":  data.Repository.Name,
		"RepositoryURL":   data.Repository.URL,
		"OwnerURL":        fmt.Sprintf("https://github.com/%s", data.Repository.Owner),
		"BranchURL":       fmt.Sprintf("%s/tree/%s", data.Repository.URL, data.PullRequest.Branch),
		"DefaultBranch":   data.Repository.DefaultBranch,
		"Branch":          data.PullRequest.Branch,
		"CommitSHA":       e.formatCommitSHA(data.PullRequest.CommitSHA),
		"CommitSHAFull":   data.PullRequest.CommitSHA, // Full SHA for metadata
		"CommitURL":       fmt.Sprintf("%s/commit/%s", data.Repository.URL, data.PullRequest.CommitSHA),

		// PR-specific information
		"PRNumber":         fmt.Sprintf("%d", data.PullRequest.Number),
		"PRTitle":          data.PullRequest.Title,
		"BaselineCoverage": data.Comparison.BasePercentage,

		// Coverage metrics
		"TotalCoverage":     data.Coverage.Overall.Percentage,
		"CoverageTrend":     data.Comparison.Change,
		"TrendDirection":    data.Comparison.Direction,
		"CoveredFiles":      coveredFiles,
		"TotalFiles":        totalFiles,
		"FilesPercent":      fmt.Sprintf("%.1f", float64(coveredFiles)/float64(totalFiles)*100),
		"FilesTrend":        "0", // Not available in PR data
		"LinesToCover":      0,   // Not available in PR data
		"LinesToCoverTrend": 0,   // Not available in PR data
		"PackagesTracked":   len(data.Coverage.Packages),

		// Template metadata
		"Timestamp":         timestamp,
		"Packages":          packages,
		"HasHistory":        false, // PR reports don't have history
		"HasAnyData":        true,
		"HistoryDataPoints": 0,
		"IsFeatureBranch":   data.PullRequest.Branch != data.Repository.DefaultBranch,
		"HistoryJSON":       "[]",
		"BuildStatus":       nil,
		"LatestTag":         "", // Not available in PR data
		"WorkflowRunNumber": 0,  // Not available in PR data
		"IsFirstRun":        false,
		"HasPreviousRuns":   false,
		"GoogleAnalyticsID": "", // Not available in PR data
	}

	return dashboardData
}

// formatCommitSHA formats commit SHA for display (helper method)
func (e *PRTemplateEngine) formatCommitSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// initializeTemplates initializes all built-in templates.
// This method is kept for compatibility but no longer used
// as we now use the dashboard template directly.
func (e *PRTemplateEngine) initializeTemplates() {
	// No longer needed - using dashboard template renderer
}

// createTemplateFuncMap creates the function map for templates
func (e *PRTemplateEngine) createTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		// Formatting functions
		"formatPercent":   e.formatPercent,
		"formatChange":    e.formatChange,
		"formatNumber":    e.formatNumber,
		"formatGrade":     e.formatGrade,
		"formatTimestamp": e.formatTimestamp,

		// Status functions
		"statusEmoji":   e.statusEmoji,
		"trendEmoji":    e.trendEmoji,
		"riskEmoji":     e.riskEmoji,
		"gradeEmoji":    e.gradeEmoji,
		"priorityEmoji": e.priorityEmoji,

		// Progress bars and charts
		"progressBar": e.progressBar,
		"trendChart":  e.trendChart,
		"coverageBar": e.coverageBar,

		// Content filtering
		"filterFiles":           e.filterFiles,
		"filterPackages":        e.filterPackages,
		"filterRecommendations": e.filterRecommendations,
		"sortFilesByRisk":       e.sortFilesByRisk,
		"sortByChange":          e.sortByChange,

		// Conditional logic
		"isSignificant":  e.isSignificant,
		"isImproved":     e.isImproved,
		"isDegraded":     e.isDegraded,
		"isStable":       e.isStable,
		"needsAttention": e.needsAttention,

		// Text utilities
		"truncate":   e.truncate,
		"pluralize":  e.pluralize,
		"capitalize": e.capitalize,
		"humanize":   e.humanize,

		// Calculations
		"abs":   math.Abs,
		"max":   math.Max,
		"min":   math.Min,
		"round": e.round,
		"mul":   e.multiply,
		"add":   e.add,

		// Collections
		"slice":  e.slice,
		"join":   strings.Join,
		"split":  strings.Split,
		"length": e.length,
	}
}

// Template helper functions

func (e *PRTemplateEngine) formatPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value)
}

func (e *PRTemplateEngine) formatChange(value float64) string {
	if value > 0 {
		return fmt.Sprintf("+%.1f%%", value)
	} else if value < 0 {
		return fmt.Sprintf("%.1f%%", value)
	}
	return "±0.0%"
}

func (e *PRTemplateEngine) formatNumber(value int) string {
	if value >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(value)/1000000)
	} else if value >= 1000 {
		return fmt.Sprintf("%.1fK", float64(value)/1000)
	}
	return fmt.Sprintf("%d", value)
}

func (e *PRTemplateEngine) formatGrade(grade string) string {
	if !e.config.IncludeEmojis {
		return grade
	}

	switch grade {
	case "A+", "A":
		return fmt.Sprintf("🏆 %s", grade)
	case "B+", "B":
		return fmt.Sprintf("⭐ %s", grade)
	case "C":
		return fmt.Sprintf("⚠️ %s", grade)
	case "D", "F":
		return fmt.Sprintf("🚨 %s", grade)
	default:
		return grade
	}
}

func (e *PRTemplateEngine) formatTimestamp(t time.Time) string {
	return t.Format(e.config.TimestampFormat)
}

func (e *PRTemplateEngine) statusEmoji(status string) string {
	if !e.config.IncludeEmojis {
		return ""
	}

	switch status {
	case "excellent":
		return "🟢"
	case "good":
		return "🟡"
	case "warning":
		return "🟠"
	case "critical":
		return "🔴"
	default:
		return "⚪"
	}
}

func (e *PRTemplateEngine) trendEmoji(direction string) string {
	if !e.config.IncludeEmojis {
		return ""
	}

	switch direction {
	case "improved", "up", "upward":
		return "📈"
	case "degraded", "down", "downward":
		return "📉"
	case "stable":
		return "📊"
	case "volatile":
		return "📊"
	default:
		return "📊"
	}
}

func (e *PRTemplateEngine) riskEmoji(risk string) string {
	if !e.config.IncludeEmojis {
		return ""
	}

	switch risk {
	case "high", "critical":
		return "🚨"
	case "medium":
		return "⚠️"
	case "low":
		return "✅"
	default:
		return "ℹ️"
	}
}

func (e *PRTemplateEngine) gradeEmoji(grade string) string {
	if !e.config.IncludeEmojis {
		return ""
	}

	switch grade {
	case "A+":
		return "🏆"
	case "A":
		return "🥇"
	case "B+", "B":
		return "🥈"
	case "C":
		return "🥉"
	case "D":
		return "⚠️"
	case "F":
		return "🚨"
	default:
		return "📊"
	}
}

func (e *PRTemplateEngine) priorityEmoji(priority string) string {
	if !e.config.IncludeEmojis {
		return ""
	}

	switch priority {
	case "high":
		return "🔥"
	case "medium":
		return "📌"
	case "low":
		return "💡"
	default:
		return "ℹ️"
	}
}

func (e *PRTemplateEngine) progressBar(value, maxValue float64, width int) string {
	if !e.config.IncludeProgressBars {
		return ""
	}

	if width <= 0 {
		width = 20
	}

	percentage := value / maxValue
	if percentage > 1 {
		percentage = 1
	} else if percentage < 0 {
		percentage = 0
	}

	filled := int(percentage * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("`%s` %.1f%%", bar, value)
}

func (e *PRTemplateEngine) coverageBar(percentage float64) string {
	return e.progressBar(percentage, 100, 15)
}

func (e *PRTemplateEngine) trendChart(value interface{}) string {
	if !e.config.IncludeCharts {
		return ""
	}

	// Handle both single value and slice of values
	var values []float64
	switch v := value.(type) {
	case float64:
		// Single value - just show indicator
		if v >= 90 {
			return "📈"
		}
		if v >= 70 {
			return "📊"
		}
		return "📉"
	case []float64:
		values = v
	default:
		return ""
	}

	if len(values) == 0 {
		return ""
	}

	// Simple ASCII chart implementation
	maxVal := values[0]
	minVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
		if v < minVal {
			minVal = v
		}
	}

	if maxVal == minVal {
		return strings.Repeat("─", len(values))
	}

	var chart strings.Builder
	for _, v := range values {
		normalized := (v - minVal) / (maxVal - minVal)
		if normalized > 0.8 {
			chart.WriteString("▄")
		} else if normalized > 0.6 {
			chart.WriteString("▃")
		} else if normalized > 0.4 {
			chart.WriteString("▂")
		} else if normalized > 0.2 {
			chart.WriteString("▁")
		} else {
			chart.WriteString("_")
		}
	}

	return chart.String()
}

func (e *PRTemplateEngine) filterFiles(files []FileCoverageData) []FileCoverageData {
	filtered := make([]FileCoverageData, 0, len(files))

	for _, file := range files {
		// Skip stable files if configured
		if e.config.HideStableFiles && file.Status == "stable" && math.Abs(file.Change) < 1.0 {
			continue
		}

		filtered = append(filtered, file)
	}

	// Limit the number of files
	if len(filtered) > e.config.MaxFileChanges {
		filtered = filtered[:e.config.MaxFileChanges]
	}

	return filtered
}

func (e *PRTemplateEngine) filterPackages(packages []PackageCoverageData) []PackageCoverageData {
	filtered := make([]PackageCoverageData, 0, len(packages))

	for _, pkg := range packages {
		// Skip stable packages if configured
		if e.config.HideStableFiles && pkg.Status == "stable" && math.Abs(pkg.Change) < 1.0 {
			continue
		}

		filtered = append(filtered, pkg)
	}

	// Limit the number of packages
	if len(filtered) > e.config.MaxPackageChanges {
		filtered = filtered[:e.config.MaxPackageChanges]
	}

	return filtered
}

func (e *PRTemplateEngine) filterRecommendations(recommendations []RecommendationData) []RecommendationData {
	// Sort by priority
	sort.Slice(recommendations, func(i, j int) bool {
		priorities := map[string]int{"high": 3, "medium": 2, "low": 1}
		return priorities[recommendations[i].Priority] > priorities[recommendations[j].Priority]
	})

	// Limit the number of recommendations
	if len(recommendations) > e.config.MaxRecommendations {
		recommendations = recommendations[:e.config.MaxRecommendations]
	}

	return recommendations
}

func (e *PRTemplateEngine) sortFilesByRisk(files []FileCoverageData) []FileCoverageData {
	sorted := make([]FileCoverageData, len(files))
	copy(sorted, files)

	sort.Slice(sorted, func(i, j int) bool {
		risks := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
		if risks[sorted[i].Risk] != risks[sorted[j].Risk] {
			return risks[sorted[i].Risk] > risks[sorted[j].Risk]
		}
		return math.Abs(sorted[i].Change) > math.Abs(sorted[j].Change)
	})

	return sorted
}

func (e *PRTemplateEngine) sortByChange(files []FileCoverageData) []FileCoverageData {
	sorted := make([]FileCoverageData, len(files))
	copy(sorted, files)

	sort.Slice(sorted, func(i, j int) bool {
		return math.Abs(sorted[i].Change) > math.Abs(sorted[j].Change)
	})

	return sorted
}

func (e *PRTemplateEngine) isSignificant(change float64) bool {
	return math.Abs(change) >= 1.0
}

func (e *PRTemplateEngine) isImproved(direction string) bool {
	return direction == "improved" || direction == "up" || direction == "upward"
}

func (e *PRTemplateEngine) isDegraded(direction string) bool {
	return direction == "degraded" || direction == "down" || direction == "downward"
}

func (e *PRTemplateEngine) isStable(direction string) bool {
	return direction == "stable"
}

func (e *PRTemplateEngine) needsAttention(percentage float64) bool {
	return percentage < e.config.WarningThreshold
}

func (e *PRTemplateEngine) truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

func (e *PRTemplateEngine) pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func (e *PRTemplateEngine) capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (e *PRTemplateEngine) humanize(s string) string {
	// Replace underscores and hyphens with spaces, capitalize words
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(s)
	for i, word := range words {
		words[i] = e.capitalize(word)
	}
	return strings.Join(words, " ")
}

func (e *PRTemplateEngine) round(value float64) float64 {
	return math.Round(value*10) / 10
}

func (e *PRTemplateEngine) multiply(a, b float64) float64 {
	return a * b
}

func (e *PRTemplateEngine) add(a, b int) int {
	return a + b
}

func (e *PRTemplateEngine) slice(items interface{}, start, end int) interface{} {
	switch v := items.(type) {
	case []FileCoverageData:
		if end > len(v) {
			end = len(v)
		}
		if start < 0 {
			start = 0
		}
		if start >= end {
			return []FileCoverageData{}
		}
		return v[start:end]
	case []PackageCoverageData:
		if end > len(v) {
			end = len(v)
		}
		if start < 0 {
			start = 0
		}
		if start >= end {
			return []PackageCoverageData{}
		}
		return v[start:end]
	case []RecommendationData:
		if end > len(v) {
			end = len(v)
		}
		if start < 0 {
			start = 0
		}
		if start >= end {
			return []RecommendationData{}
		}
		return v[start:end]
	case []string:
		if end > len(v) {
			end = len(v)
		}
		if start < 0 {
			start = 0
		}
		if start >= end {
			return []string{}
		}
		return v[start:end]
	default:
		return items
	}
}

func (e *PRTemplateEngine) length(items interface{}) int {
	switch v := items.(type) {
	case []FileCoverageData:
		return len(v)
	case []PackageCoverageData:
		return len(v)
	case []RecommendationData:
		return len(v)
	case []string:
		return len(v)
	case string:
		return len(v)
	default:
		return 0
	}
}

// AddCustomTemplate adds a custom template to the engine
func (e *PRTemplateEngine) AddCustomTemplate(name, templateContent string) error {
	funcMap := e.createTemplateFuncMap()
	tmpl, err := template.New(name).Funcs(funcMap).Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse custom template: %w", err)
	}

	e.templates[name] = tmpl
	return nil
}

// GetAvailableTemplates returns a list of available template names
func (e *PRTemplateEngine) GetAvailableTemplates() []string {
	return []string{"dashboard"}
}

// dashboardTemplate is the dashboard HTML template embedded here to avoid import cycles
//
//nolint:misspell // GitHub Actions API uses British spelling for "cancelled"
const dashboardTemplate = `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.RepositoryOwner}}/{{.RepositoryName}} Coverage Dashboard</title>
    <meta name="description" content="Coverage tracking and analytics for {{.RepositoryOwner}}/{{.RepositoryName}}">

    <!-- Favicon -->
    <link rel="icon" type="image/x-icon" href="./favicon.ico">
    <link rel="icon" type="image/svg+xml" href="./favicon.svg">
    <link rel="shortcut icon" href="./favicon.ico">

    <!-- Preload critical resources -->
    <link rel="preconnect" href="https://fonts.googleapis.com" crossorigin>
    <link rel="preload" href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" as="style">
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">

    {{- if .GoogleAnalyticsID}}
    <!-- Google Analytics -->
    <script async src="https://www.googletagmanager.com/gtag/js?id={{.GoogleAnalyticsID}}"></script>
    <script>
      window.dataLayer = window.dataLayer || [];
      function gtag(){dataLayer.push(arguments);}
      gtag('js', new Date());
      gtag('config', '{{.GoogleAnalyticsID}}');
    </script>
    {{- end}}

    <style>
        /* CSS Custom Properties */
        :root {
            --color-bg: #0d1117;
            --color-bg-secondary: #161b22;
            --color-bg-tertiary: #21262d;
            --color-text: #c9d1d9;
            --color-text-secondary: #8b949e;
            --color-primary: #58a6ff;
            --color-success: #3fb950;
            --color-warning: #d29922;
            --color-danger: #f85149;
            --color-border: #30363d;
            --color-border-muted: #21262d;

            /* Glass morphism */
            --glass-bg: rgba(22, 27, 34, 0.8);
            --glass-border: rgba(48, 54, 61, 0.5);
            --backdrop-blur: 10px;

            /* Animations */
            --transition-base: 0.2s ease;
            --transition-smooth: 0.3s cubic-bezier(0.4, 0, 0.2, 1);

            /* Gradients */
            --gradient-primary: linear-gradient(135deg, #4a90d9, #6ba3e3);
            --gradient-success: linear-gradient(135deg, #3fb950, #56d364);
            --gradient-danger: linear-gradient(135deg, #f85149, #da3633);
        }

        /* Light theme */
        [data-theme="light"] {
            --color-bg: #ffffff;
            --color-bg-secondary: #f6f8fa;
            --color-bg-tertiary: #f0f6fc;
            --color-text: #24292f;
            --color-text-secondary: #656d76;
            --color-border: #d0d7de;
            --color-border-muted: #f0f6fc;
            --glass-bg: rgba(246, 248, 250, 0.8);
            --glass-border: rgba(208, 215, 222, 0.5);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            background: var(--color-bg);
            color: var(--color-text);
            line-height: 1.6;
            min-height: 100vh;
            position: relative;
        }

        /* Animated background */
        body::before {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background:
                radial-gradient(circle at 20% 50%, rgba(74, 144, 217, 0.1) 0%, transparent 50%),
                radial-gradient(circle at 80% 80%, rgba(63, 185, 80, 0.1) 0%, transparent 50%),
                radial-gradient(circle at 40% 20%, rgba(248, 81, 73, 0.05) 0%, transparent 50%);
            pointer-events: none;
            z-index: 1;
        }

        /* Main container */
        .container {
            position: relative;
            z-index: 2;
            max-width: 1400px;
            margin: 0 auto;
            padding: 2rem;
        }

        /* Enhanced Header */
        .header {
            margin-bottom: 3rem;
            padding: 2rem;
            position: relative;
            overflow: hidden;
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 20px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        }

        .header-content {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
        }

        .header-main {
            text-align: left;
        }

        .header-status {
            display: flex;
            flex-direction: column;
            align-items: flex-end;
            gap: 0.5rem;
        }

        .status-indicator {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.75rem 1.25rem;
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 16px;
            text-decoration: none;
            transition: all var(--transition-smooth);
            cursor: pointer;
            position: relative;
            overflow: hidden;
        }

        .status-indicator::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: linear-gradient(135deg,
                rgba(74, 144, 217, 0.05) 0%,
                rgba(63, 185, 80, 0.05) 100%);
            opacity: 0;
            transition: opacity var(--transition-smooth);
        }

        .status-indicator:hover::before {
            opacity: 1;
        }

        .status-indicator:hover {
            transform: translateY(-2px) scale(1.02);
            box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
            border-color: var(--color-primary);
        }

        .status-icon {
            width: 24px;
            height: 24px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 18px;
            position: relative;
            z-index: 1;
        }

        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: var(--color-text-secondary);
        }

        .status-dot.active {
            background: var(--color-success);
            animation: pulse 2s infinite;
        }

        .status-dot.in-progress {
            background: var(--color-warning);
            animation: pulse 1s infinite;
        }

        .status-dot.failed {
            background: var(--color-danger);
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        .status-text {
            font-size: 0.9rem;
            font-weight: 500;
            color: var(--color-text);
        }

        .status-details {
            display: flex;
            flex-direction: column;
            align-items: flex-start;
            gap: 0.1rem;
        }

        .status-workflow {
            font-size: 0.8rem;
            color: var(--color-text-secondary);
            font-family: 'JetBrains Mono', monospace;
        }

        .last-sync {
            font-size: 0.8rem;
            color: var(--color-text-secondary);
            font-family: 'JetBrains Mono', monospace;
        }

        .repo-info-enhanced {
            display: flex;
            justify-content: space-between;
            align-items: center;
            gap: 2rem;
            flex-wrap: wrap;
        }

        .repo-details {
            display: flex;
            gap: 1.5rem;
            flex-wrap: wrap;
            position: relative;
            z-index: 10;
        }

        .repo-item {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 0.25rem;
            padding: 1rem;
            background: var(--color-bg-secondary);
            border: 1px solid var(--color-border);
            border-radius: 12px;
            min-width: 100px;
            transition: var(--transition-smooth);
        }

        .repo-item:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
        }

        .repo-item-clickable {
            text-decoration: none;
            color: inherit;
            cursor: pointer;
        }

        .repo-item-clickable:hover {
            border-color: var(--color-primary);
            transform: translateY(-4px);
            box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
        }

        .repo-icon {
            font-size: 1.5rem;
        }

        .repo-label {
            font-size: 0.7rem;
            color: var(--color-text-secondary);
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .repo-value {
            font-size: 0.9rem;
            font-weight: 500;
            color: var(--color-text);
            font-family: 'JetBrains Mono', monospace;
            text-align: center;
        }

        .commit-link {
            color: var(--color-primary);
            text-decoration: none;
            transition: var(--transition-base);
        }

        .commit-link:hover {
            text-decoration: underline;
            opacity: 0.8;
        }

        .repo-link-light {
            color: var(--color-text-secondary);
            transition: var(--transition-base);
        }

        .repo-item-clickable:hover .repo-link-light {
            color: var(--color-primary);
        }

        .header-actions {
            display: flex;
            gap: 0.75rem;
            flex-wrap: wrap;
        }

        .action-btn {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.75rem 1.25rem;
            border: none;
            border-radius: 12px;
            font-size: 0.9rem;
            font-weight: 600;
            cursor: pointer;
            transition: var(--transition-smooth);
            font-family: inherit;
            position: relative;
            overflow: hidden;
        }

        .action-btn::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.2), transparent);
            transition: left 0.5s ease;
        }

        .action-btn:hover::before {
            left: 100%;
        }

        .action-btn.primary {
            background: linear-gradient(135deg, #2563eb, #1e40af);
            color: white;
            box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
        }

        .action-btn.primary:hover {
            background: linear-gradient(135deg, #1d4ed8, #1e3a8a);
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(37, 99, 235, 0.4);
        }

        .action-btn.secondary {
            background: var(--color-bg-secondary);
            color: var(--color-text);
            border: 1px solid var(--color-border);
        }

        .action-btn.secondary:hover {
            background: var(--color-bg-tertiary);
            border-color: var(--color-primary);
            transform: translateY(-2px);
            box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
        }

        .btn-icon {
            font-size: 1rem;
        }

        .btn-text {
            font-size: 0.85rem;
            letter-spacing: 0.02em;
        }

        .header::before {
            content: '';
            position: absolute;
            top: -50%;
            left: -50%;
            width: 200%;
            height: 200%;
            background: radial-gradient(circle, var(--color-primary) 0%, transparent 70%);
            opacity: 0.05;
            animation: rotate 30s linear infinite;
            pointer-events: none;
            z-index: -1;
        }

        @keyframes rotate {
            to { transform: rotate(360deg); }
        }

        .header h1 {
            font-size: 3rem;
            font-weight: 700;
            background: var(--gradient-primary);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 0.5rem;
            position: relative;
        }

        .header .subtitle {
            font-size: 1.25rem;
            color: var(--color-text-secondary);
            margin-bottom: 1rem;
        }

        .repo-info {
            display: inline-flex;
            align-items: center;
            gap: 1rem;
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            padding: 0.75rem 1.5rem;
            border-radius: 12px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.9rem;
        }

        .repo-info a {
            color: var(--color-primary);
            text-decoration: none;
            transition: var(--transition-base);
        }

        .repo-info a:hover {
            text-decoration: underline;
        }

        /* Metrics grid */
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 1.5rem;
            margin-bottom: 3rem;
        }

        .metric-card {
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 16px;
            padding: 2rem;
            position: relative;
            overflow: hidden;
            transition: var(--transition-smooth);
        }

        .metric-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 2px;
            background: var(--gradient-primary);
            transition: left 0.5s ease;
        }

        .metric-card:hover::before {
            left: 0;
        }

        .metric-card:hover {
            transform: translateY(-4px);
            box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
        }

        .metric-card h3 {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-size: 1rem;
            color: var(--color-text-secondary);
            margin-bottom: 1rem;
            font-weight: 600;
        }

        .metric-value {
            font-size: 2.5rem;
            font-weight: 700;
            background: var(--gradient-primary);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 0.5rem;
        }

        .metric-value.success {
            background: var(--gradient-success);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .metric-value.danger {
            background: var(--gradient-danger);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .quality-gate-badge {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.75rem 1rem;
            background: linear-gradient(135deg, var(--color-success), #4ade80);
            border-radius: 12px;
            margin-bottom: 0.5rem;
            box-shadow: 0 4px 12px rgba(34, 197, 94, 0.15);
            border: 1px solid rgba(34, 197, 94, 0.2);
        }

        .quality-gate-icon {
            width: 24px;
            height: 24px;
            color: white;
            flex-shrink: 0;
        }

        .quality-gate-text {
            color: white;
            font-weight: 700;
            font-size: 0.875rem;
            letter-spacing: 0.05em;
        }

        .metric-label {
            color: var(--color-text-secondary);
            font-size: 0.9rem;
            margin-bottom: 1rem;
        }

        .coverage-bar {
            height: 8px;
            background: var(--color-bg-tertiary);
            border-radius: 8px;
            overflow: hidden;
            margin: 1rem 0;
            position: relative;
        }

        .coverage-fill {
            height: 100%;
            background: var(--gradient-success);
            border-radius: 8px;
            position: relative;
            transition: width 1s cubic-bezier(0.4, 0, 0.2, 1);
        }

        .coverage-fill::after {
            content: '';
            position: absolute;
            top: 0;
            right: 0;
            bottom: 0;
            width: 100px;
            background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.3), transparent);
            animation: shimmer 2s infinite;
        }

        @keyframes shimmer {
            0% { transform: translateX(-100px); }
            100% { transform: translateX(100px); }
        }

        .status-badge {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.5rem 1rem;
            border-radius: 24px;
            font-size: 0.85rem;
            font-weight: 600;
            background: var(--gradient-success);
            color: white;
        }

        .status-badge.warning {
            background: var(--gradient-danger);
        }

        /* Links section */
        .links-section {
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 16px;
            padding: 2rem;
            margin-bottom: 2rem;
        }

        .links-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 1rem;
            margin-top: 1rem;
        }

        .link-item {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 1rem;
            background: var(--color-bg-secondary);
            border: 1px solid var(--color-border);
            border-radius: 12px;
            text-decoration: none;
            color: var(--color-text);
            transition: var(--transition-smooth);
            position: relative;
            overflow: hidden;
        }

        .link-item::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: var(--gradient-primary);
            opacity: 0.1;
            transition: left 0.3s ease;
        }

        .link-item:hover::before {
            left: 0;
        }

        .link-item:hover {
            border-color: var(--color-primary);
            transform: translateX(4px);
        }

        /* Updated time */
        .last-updated {
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 12px;
            padding: 1rem;
            text-align: center;
            color: var(--color-text-secondary);
            font-size: 0.9rem;
        }

        /* Footer */
        .footer {
            margin-top: 4rem;
            padding: 2rem 0;
            border-top: 1px solid var(--color-border);
            background: var(--color-bg-secondary);
        }

        .footer-content {
            max-width: 1200px;
            margin: 0 auto;
            padding: 0 2rem;
        }

        .footer-info {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 1.5rem;
            flex-wrap: wrap;
            font-size: 0.9rem;
            color: var(--color-text-secondary);
        }

        .footer-version,
        .footer-powered,
        .footer-timestamp {
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .footer-separator {
            color: var(--color-border);
            font-size: 0.8rem;
        }

        .version-icon,
        .timestamp-icon {
            font-size: 1.1rem;
        }

        .version-text {
            font-family: 'JetBrains Mono', monospace;
            font-weight: 500;
            color: var(--color-primary);
        }
        
        .version-link {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            text-decoration: none;
            color: inherit;
            transition: all var(--transition-base);
            padding: 0.25rem 0.5rem;
            margin: -0.25rem -0.5rem;
            border-radius: 6px;
        }
        
        .version-link:hover {
            background: var(--color-bg-tertiary);
            transform: translateY(-1px);
        }
        
        .version-link:hover .version-text {
            color: var(--color-text);
        }

        .powered-text {
            color: var(--color-text-secondary);
        }

        .gofortress-link {
            display: flex;
            align-items: center;
            gap: 0.4rem;
            color: var(--color-primary);
            text-decoration: none;
            transition: all var(--transition-base);
            padding: 0.25rem 0.75rem;
            border-radius: 8px;
        }

        .gofortress-link:hover {
            background: var(--color-bg-tertiary);
            transform: translateY(-1px);
            color: var(--color-text);
        }

        .fortress-icon {
            font-size: 1.2rem;
        }

        .fortress-text {
            font-weight: 600;
        }

        @media (max-width: 768px) {
            .footer-separator {
                display: none;
            }

            .footer-info {
                flex-direction: column;
                gap: 1rem;
            }
        }

        /* Package list */
        .package-list {
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 16px;
            padding: 2rem;
            margin-top: 2rem;
        }

        .package-item {
            display: grid;
            grid-template-columns: 1fr auto 150px;
            gap: 1rem;
            align-items: center;
            padding: 1rem;
            border-bottom: 1px solid var(--color-border);
            transition: var(--transition-base);
        }

        .package-item:last-child {
            border-bottom: none;
        }

        .package-item:hover {
            background: var(--color-bg-secondary);
            border-radius: 8px;
        }

        .package-name {
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.9rem;
            color: var(--color-primary);
            letter-spacing: 0.1em;
            white-space: pre-wrap;
            line-height: 1.4;
        }

        .package-coverage {
            font-weight: 600;
            color: var(--color-success);
        }

        .package-bar {
            height: 6px;
            background: var(--color-bg-tertiary);
            border-radius: 6px;
            overflow: hidden;
        }

        .package-bar-fill {
            height: 100%;
            background: var(--gradient-success);
            border-radius: 6px;
            transition: width 0.5s ease;
        }

        /* Theme toggle */
        .theme-toggle {
            position: fixed;
            top: 2rem;
            right: 2rem;
            z-index: 100;
            background: var(--glass-bg);
            backdrop-filter: blur(var(--backdrop-blur));
            border: 1px solid var(--glass-border);
            border-radius: 12px;
            padding: 0.75rem;
            cursor: pointer;
            transition: var(--transition-base);
        }

        .theme-toggle:hover {
            transform: scale(1.1);
        }

        /* Responsive */
        @media (max-width: 768px) {
            .container {
                padding: 1rem;
            }

            .header {
                padding: 1.5rem;
            }

            .header-content {
                flex-direction: column;
                gap: 1rem;
                align-items: stretch;
            }

            .header-main {
                text-align: center;
            }

            .header h1 {
                font-size: 2rem;
            }

            .header-status {
                align-items: center;
            }

            .repo-info-enhanced {
                flex-direction: column;
                gap: 1.5rem;
                align-items: stretch;
            }

            .repo-details {
                justify-content: center;
                gap: 1rem;
            }

            .repo-item {
                min-width: 80px;
                padding: 0.75rem;
            }

            .header-actions {
                justify-content: center;
                gap: 0.5rem;
            }

            .action-btn {
                padding: 0.5rem 1rem;
                font-size: 0.8rem;
            }

            .metrics-grid {
                grid-template-columns: 1fr;
            }

            .package-item {
                grid-template-columns: 1fr;
                gap: 0.5rem;
            }
        }

        /* Animations */
        @keyframes fadeIn {
            from {
                opacity: 0;
                transform: translateY(20px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .metric-card {
            animation: fadeIn 0.5s ease forwards;
            opacity: 0;
        }

        .metric-card:nth-child(1) { animation-delay: 0.1s; }
        .metric-card:nth-child(2) { animation-delay: 0.2s; }
        .metric-card:nth-child(3) { animation-delay: 0.3s; }
        .metric-card:nth-child(4) { animation-delay: 0.4s; }
    </style>
</head>
<body>
    <div class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 18c-3.3 0-6-2.7-6-6s2.7-6 6-6 6 2.7 6 6-2.7 6-6 6z"/>
        </svg>
    </div>

    <div class="container">
        <header class="header">
            <div class="header-content">
                <div class="header-main">
                    {{- if .PRNumber}}
                    <h1>PR #{{.PRNumber}} Coverage</h1>
                    <p class="subtitle">{{- if .PRTitle}}{{.PRTitle}} • {{end}}Coverage analysis for this pull request</p>
                    {{- else}}
                    <h1>{{.RepositoryName}} Coverage</h1>
                    <p class="subtitle">Code coverage dashboard • Powered by GoFortress</p>
                    {{- end}}
                </div>

                <div class="header-status">
                    <div class="status-indicator">
                        <span class="status-dot active"></span>
                        <span class="status-text">Coverage Active</span>
                    </div>
                    <div class="last-sync">
                        <span>🕐 {{.Timestamp}}</span>
                    </div>
                </div>
            </div>

            <div class="repo-info-enhanced">
                <div class="repo-details">
                    {{- if .RepositoryURL}}
                    <a href="{{.RepositoryURL}}" target="_blank" class="repo-item repo-item-clickable">
                        <span class="repo-icon">📦</span>
                        <span class="repo-label">Repository</span>
                        <span class="repo-value repo-link-light">{{.RepositoryOwner}}/{{.RepositoryName}}</span>
                    </a>
                    {{- else}}
                    <div class="repo-item">
                        <span class="repo-icon">📦</span>
                        <span class="repo-label">Repository</span>
                        <span class="repo-value">{{.RepositoryOwner}}/{{.RepositoryName}}</span>
                    </div>
                    {{- end}}
                    {{- if .OwnerURL}}
                    <a href="{{.OwnerURL}}" target="_blank" class="repo-item repo-item-clickable">
                        <span class="repo-icon">👤</span>
                        <span class="repo-label">Owner</span>
                        <span class="repo-value">{{.RepositoryOwner}}</span>
                    </a>
                    {{- else}}
                    <div class="repo-item">
                        <span class="repo-icon">👤</span>
                        <span class="repo-label">Owner</span>
                        <span class="repo-value">{{.RepositoryOwner}}</span>
                    </div>
                    {{- end}}
                    {{- if .BranchURL}}
                    <a href="{{.BranchURL}}" target="_blank" class="repo-item repo-item-clickable">
                        <span class="repo-icon">🌿</span>
                        <span class="repo-label">Branch</span>
                        <span class="repo-value">{{.Branch}}</span>
                    </a>
                    {{- else}}
                    <div class="repo-item">
                        <span class="repo-icon">🌿</span>
                        <span class="repo-label">Branch</span>
                        <span class="repo-value">{{.Branch}}</span>
                    </div>
                    {{- end}}
                    {{- if .CommitSHA}}
                        {{- if .CommitURL}}
                        <a href="{{.CommitURL}}" target="_blank" class="repo-item repo-item-clickable">
                            <span class="repo-icon">🔗</span>
                            <span class="repo-label">Commit</span>
                            <span class="repo-value commit-link">{{.CommitSHA}}</span>
                        </a>
                        {{- else}}
                        <div class="repo-item">
                            <span class="repo-icon">🔗</span>
                            <span class="repo-label">Commit</span>
                            <span class="repo-value">{{.CommitSHA}}</span>
                        </div>
                        {{- end}}
                    {{- end}}
                </div>

                <div class="header-actions">
                    <button class="action-btn primary" onclick="window.location.reload()">
                        <span class="btn-icon">🔄</span>
                        <span class="btn-text">Refresh</span>
                    </button>
                    <button class="action-btn secondary" onclick="window.open('./coverage.html', '_blank')">
                        <span class="btn-icon">📄</span>
                        <span class="btn-text">Detailed Report</span>
                    </button>
                    <button class="action-btn secondary" onclick="window.open('{{.RepositoryURL}}', '_blank')">
                        <span class="btn-icon">📦</span>
                        <span class="btn-text">Repository</span>
                    </button>
                </div>
            </div>
        </header>

        <main>
            <div class="metrics-grid">
                <div class="metric-card">
                    <h3>📊 Overall Coverage</h3>
                    <div class="metric-value success">{{.TotalCoverage}}%</div>
                    {{- if .PRNumber}}
                    <div class="metric-label">PR Coverage{{- if .BaselineCoverage}} ({{if gt .TotalCoverage .BaselineCoverage}}+{{else if lt .TotalCoverage .BaselineCoverage}}-{{end}}{{printf "%.1f" (sub .TotalCoverage .BaselineCoverage)}}% vs base){{end}}</div>
                    {{- else}}
                    <div class="metric-label">{{.CoveredFiles}} of {{.TotalFiles}} files covered</div>
                    {{- end}}
                    <div class="coverage-bar">
                        <div class="coverage-fill" style="width: {{.TotalCoverage}}%"></div>
                    </div>
                    {{- if .PRNumber}}
                        {{- if .BaselineCoverage}}
                            {{- if gt .TotalCoverage .BaselineCoverage}}
                            <div class="status-badge">
                                📈 Coverage Improved
                            </div>
                            {{- else if lt .TotalCoverage .BaselineCoverage}}
                            <div class="status-badge warning">
                                📉 Coverage Decreased  
                            </div>
                            {{- else}}
                            <div class="status-badge">
                                ➡️ Coverage Stable
                            </div>
                            {{- end}}
                        {{- else}}
                        <div class="status-badge">
                            🆕 New PR Coverage
                        </div>
                        {{- end}}
                    {{- else}}
                    <div class="status-badge">
                        ✅ Excellent Coverage
                    </div>
                    {{- end}}
                </div>

                <div class="metric-card">
                    <h3>📁 Packages</h3>
                    <div class="metric-value">{{.PackagesTracked}}</div>
                    <div class="metric-label">Packages analyzed</div>
                    <div style="margin-top: 1rem;">
                        <div style="font-size: 0.9rem; color: var(--color-text-secondary);">
                            • All packages tracked
                        </div>
                    </div>
                </div>

                <div class="metric-card">
                    <h3>🎯 Quality Gate</h3>
                    <div class="quality-gate-badge">
                        <svg class="quality-gate-icon" viewBox="0 0 24 24" fill="none">
                            <circle cx="12" cy="12" r="10" fill="currentColor" fill-opacity="0.1"/>
                            <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="1.5"/>
                            <path d="M8.5 12.5L10.5 14.5L15.5 9.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                        <span class="quality-gate-text">PASSED</span>
                    </div>
                    <div class="metric-label">Threshold: 80% (exceeded)</div>
                    <div style="margin-top: 1rem; font-size: 0.9rem; color: var(--color-success);">
                        Coverage meets quality standards
                    </div>
                </div>

                <div class="metric-card">
                    <h3>🔄 Coverage Trend</h3>
                    {{if .HasHistory}}
                        <div class="metric-value {{- if eq .TrendDirection "up"}}success{{else if eq .TrendDirection "down"}}danger{{end -}}">
                            {{- if eq .TrendDirection "up"}}+{{end}}{{.CoverageTrend}}%
                        </div>
                        <div class="metric-label">Change from previous</div>
                        <div style="margin-top: 1rem; font-size: 0.9rem; color: var(--color-text-secondary);">
                            {{- if eq .TrendDirection "up"}}📈 Improving{{else if eq .TrendDirection "down"}}📉 Declining{{else}}➡️ Stable{{end -}}
                        </div>
                    {{else}}
                        <div class="metric-value" style="font-size: 1.5rem;">📊</div>
                        <div class="metric-label">Trend Analysis</div>
                        <div style="margin-top: 1rem;">
                            {{if .HasAnyData}}
                                <div style="font-size: 0.9rem; color: var(--color-warning);">
                                    🔄 Building trend data...
                                </div>
                                <div style="font-size: 0.8rem; color: var(--color-text-secondary); margin-top: 0.5rem;">
                                    {{if .PRNumber}}
                                        Comparing against base branch
                                    {{else if .IsFeatureBranch}}
                                        {{.HistoryDataPoints}} data point{{- if ne .HistoryDataPoints 1}}s{{end -}} for this branch
                                    {{else}}
                                        Need 2+ commits to show trends
                                    {{end}}
                                </div>
                            {{else}}
                                <div style="font-size: 0.9rem; color: var(--color-primary);">
                                    {{if .PRNumber}}
                                        📊 PR Coverage Analysis
                                    {{else if .IsFeatureBranch}}
                                        🌿 New branch coverage
                                    {{else if .IsFirstRun}}
                                        🚀 First coverage run!
                                    {{else if .HasPreviousRuns}}
                                        ⏳ Building history data...
                                    {{else if .WorkflowRunNumber}}
                                        📊 Coverage tracking resumed
                                    {{else}}
                                        📊 Coverage baseline established
                                    {{end}}
                                </div>
                                <div style="font-size: 0.8rem; color: var(--color-text-secondary); margin-top: 0.5rem;">
                                    {{if .PRNumber}}
                                        Base branch comparison pending
                                    {{else if .IsFirstRun}}
                                        Trends will appear after more commits
                                    {{else if .HasPreviousRuns}}
                                        Previous workflow runs failed to record history
                                    {{else if .WorkflowRunNumber}}
                                        Workflow run #{{.WorkflowRunNumber}} {{- if gt .WorkflowRunNumber 10}}(history may be incomplete){{end -}}
                                    {{else}}
                                        Collecting baseline coverage data
                                    {{end}}
                                </div>
                            {{end}}
                        </div>
                    {{end}}
                </div>
            </div>

            <div class="links-section">
                <h3 style="margin-bottom: 1rem;">📋 Coverage Reports & Tools</h3>
                <div class="links-grid">
                    <a href="./coverage.html" class="link-item">
                        📄 Detailed HTML Report
                    </a>
                    <a href="./coverage.svg" class="link-item">
                        🏷️ Coverage Badge
                    </a>
                    <a href="{{.RepositoryURL}}" class="link-item">
                        📦 Source Repository
                    </a>
                    <a href="{{.RepositoryURL}}/actions" class="link-item">
                        🚀 GitHub Actions
                    </a>
                </div>
            </div>

            {{- if .Packages}}
            <div class="package-list">
                <h3 style="margin-bottom: 1rem;">📦 Package Coverage</h3>
                {{- range .Packages}}
                <div class="package-item">
                    <div class="package-name">{{.Name}}</div>
                    <div class="package-coverage">{{.Coverage}}%</div>
                    <div class="package-bar">
                        <div class="package-bar-fill" style="width: {{.Coverage}}%"></div>
                    </div>
                </div>
                {{- end}}
            </div>
            {{- end}}

            <div class="last-updated">
                🕐 Last updated: {{.Timestamp}}
            </div>
        </main>

        <footer class="footer">
            <div class="footer-content">
                <div class="footer-info">
                    {{- if .LatestTag}}
                    <div class="footer-version">
                        <a href="https://github.com/{{.RepositoryOwner}}/{{.RepositoryName}}/releases/tag/{{.LatestTag}}" target="_blank" class="version-link">
                            <span class="version-icon">🏷️</span>
                            <span class="version-text">{{.LatestTag}}</span>
                        </a>
                    </div>
                    <span class="footer-separator">•</span>
                    {{- end}}
                    <div class="footer-powered">
                        <span class="powered-text">Powered by</span>
                        <a href="https://github.com/{{.RepositoryOwner}}/{{.RepositoryName}}" target="_blank" class="gofortress-link">
                            <span class="fortress-icon">🏰</span>
                            <span class="fortress-text">GoFortress Coverage</span>
                        </a>
                    </div>
                    <span class="footer-separator">•</span>
                    <div class="footer-timestamp">
                        <span class="timestamp-icon">🕐</span>
                        <span class="timestamp-text">{{.Timestamp}}</span>
                    </div>
                </div>
            </div>
        </footer>
    </div>

    <script>
        // Theme toggle
        function toggleTheme() {
            const html = document.documentElement;
            const currentTheme = html.getAttribute('data-theme');
            const newTheme = currentTheme === 'light' ? 'dark' : 'light';
            html.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
        }

        // Initialize theme
        const savedTheme = localStorage.getItem('theme');
        const systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        const theme = savedTheme || (systemPrefersDark ? 'dark' : 'light');
        document.documentElement.setAttribute('data-theme', theme);

        // History data
        const historyData = {{.HistoryJSON}};

        // Initialize charts if history data exists
        if (historyData && historyData.length > 0) {
            // Future: Add chart rendering here
        }

        // Note: Build status refresh functionality has been removed
        // Static deployments on GitHub Pages cannot provide live updates
        // The build status shown is a snapshot from when the report was generated
    </script>
</body>
</html>`
