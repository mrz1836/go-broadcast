// Package templates provides advanced PR comment template system with dynamic content rendering
package templates

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"math"
	"sort"
	"strings"
	"time"
)

// PRTemplateEngine handles advanced PR comment template rendering
type PRTemplateEngine struct {
	templates map[string]*template.Template
	config    *TemplateConfig
}

// TemplateConfig holds configuration for template rendering
type TemplateConfig struct {
	// Template selection
	DefaultTemplate string // Default template to use
	CompactMode     bool   // Use compact template variants
	IncludeEmojis   bool   // Include emojis in templates
	IncludeCharts   bool   // Include ASCII charts

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

	// Configuration
	Config TemplateConfig `json:"config"`

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

// TemplateMetadata contains template metadata
type TemplateMetadata struct {
	Version      string    `json:"version"`
	GeneratedAt  time.Time `json:"generated_at"`
	TemplateUsed string    `json:"template_used"`
	Signature    string    `json:"signature"`
}

// NewPRTemplateEngine creates a new PR template engine
func NewPRTemplateEngine(config *TemplateConfig) *PRTemplateEngine {
	if config == nil {
		config = &TemplateConfig{
			DefaultTemplate:        "comprehensive",
			CompactMode:            false,
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

// RenderComment renders a PR comment using the specified template
func (e *PRTemplateEngine) RenderComment(ctx context.Context, templateName string, data *TemplateData) (string, error) {
	// Use default template if none specified
	if templateName == "" {
		templateName = e.config.DefaultTemplate
	}

	// Add configuration to template data
	data.Config = *e.config
	data.Metadata = TemplateMetadata{
		Version:      "2.0",
		GeneratedAt:  time.Now(),
		TemplateUsed: templateName,
		Signature:    "gofortress-coverage-v2",
	}

	// Get the template
	tmpl, exists := e.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	// Render the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// initializeTemplates initializes all built-in templates
func (e *PRTemplateEngine) initializeTemplates() {
	funcMap := e.createTemplateFuncMap()

	// Comprehensive template (default)
	e.templates["comprehensive"] = template.Must(template.New("comprehensive").Funcs(funcMap).Parse(comprehensiveTemplate))

	// Compact template
	e.templates["compact"] = template.Must(template.New("compact").Funcs(funcMap).Parse(compactTemplate))

	// Detailed template
	e.templates["detailed"] = template.Must(template.New("detailed").Funcs(funcMap).Parse(detailedTemplate))

	// Summary template
	e.templates["summary"] = template.Must(template.New("summary").Funcs(funcMap).Parse(summaryTemplate))

	// Minimal template
	e.templates["minimal"] = template.Must(template.New("minimal").Funcs(funcMap).Parse(minimalTemplate))
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

func (e *PRTemplateEngine) progressBar(value, max float64, width int) string {
	if !e.config.IncludeProgressBars {
		return ""
	}

	if width <= 0 {
		width = 20
	}

	percentage := value / max
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

func (e *PRTemplateEngine) trendChart(values []float64) string {
	if !e.config.IncludeCharts || len(values) == 0 {
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
	var filtered []FileCoverageData

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
	var filtered []PackageCoverageData

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

func (e *PRTemplateEngine) slice(items interface{}, start, end int) interface{} {
	// Implementation depends on the specific type
	// This is a placeholder for slice operations
	return items
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
	var names []string
	for name := range e.templates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
