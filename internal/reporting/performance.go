// Package reporting provides performance analysis and reporting functionality for the broadcast system.
package reporting

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"time"
)

// Reporting errors
var (
	ErrBaselineNotFound = errors.New("baseline file not found")
)

// PerformanceReport represents a comprehensive performance analysis report
type PerformanceReport struct {
	Timestamp       time.Time          `json:"timestamp"`
	ReportID        string             `json:"report_id"`
	Version         string             `json:"version"`
	BaselineMetrics map[string]float64 `json:"baseline_metrics"`
	CurrentMetrics  map[string]float64 `json:"current_metrics"`
	Improvements    map[string]float64 `json:"improvements"`
	Regressions     map[string]float64 `json:"regressions"`
	Recommendations []Recommendation   `json:"recommendations"`
	SystemInfo      SystemInfo         `json:"system_info"`
	TestResults     []TestResult       `json:"test_results"`
	ProfileSummary  ProfileSummary     `json:"profile_summary"`

	// Metadata
	Duration    time.Duration `json:"duration"`
	TotalTests  int           `json:"total_tests"`
	PassedTests int           `json:"passed_tests"`
	FailedTests int           `json:"failed_tests"`
}

// Recommendation represents a performance optimization recommendation
type Recommendation struct {
	ID          string                 `json:"id"`
	Priority    RecommendationPriority `json:"priority"`
	Category    string                 `json:"category"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Action      string                 `json:"action"`
	Impact      string                 `json:"impact"`
	Evidence    []string               `json:"evidence"`
	References  []string               `json:"references,omitempty"`
}

// RecommendationPriority defines recommendation priority levels
type RecommendationPriority string

const (
	// PriorityHigh indicates high-priority recommendations that should be addressed immediately
	PriorityHigh RecommendationPriority = "high"
	// PriorityMedium indicates medium-priority recommendations that should be addressed soon
	PriorityMedium RecommendationPriority = "medium"
	// PriorityLow indicates low-priority recommendations that can be addressed when convenient
	PriorityLow RecommendationPriority = "low"
)

// SystemInfo contains system information for the report
type SystemInfo struct {
	GoVersion  string    `json:"go_version"`
	GOOS       string    `json:"goos"`
	GOARCH     string    `json:"goarch"`
	NumCPU     int       `json:"num_cpu"`
	GOMAXPROCS int       `json:"gomaxprocs"`
	Timestamp  time.Time `json:"timestamp"`
}

// TestResult represents the result of a performance test
type TestResult struct {
	Name       string        `json:"name"`
	Duration   time.Duration `json:"duration"`
	Success    bool          `json:"success"`
	Throughput float64       `json:"throughput"`
	MemoryUsed int64         `json:"memory_used_mb"`
	Error      string        `json:"error,omitempty"`
}

// ProfileSummary contains summary information from profiling
type ProfileSummary struct {
	CPUProfile       ProfileInfo `json:"cpu_profile"`
	MemoryProfile    ProfileInfo `json:"memory_profile"`
	GoroutineProfile ProfileInfo `json:"goroutine_profile"`
	TotalProfileSize int64       `json:"total_profile_size_bytes"`
}

// ProfileInfo contains information about a specific profile
type ProfileInfo struct {
	Available bool   `json:"available"`
	Size      int64  `json:"size_bytes"`
	Path      string `json:"path,omitempty"`
}

// ReportConfig configures report generation
type ReportConfig struct {
	OutputDirectory     string                 `json:"output_directory"`
	BaselineFile        string                 `json:"baseline_file"`
	IncludeProfiles     bool                   `json:"include_profiles"`
	GenerateHTML        bool                   `json:"generate_html"`
	GenerateJSON        bool                   `json:"generate_json"`
	GenerateMarkdown    bool                   `json:"generate_markdown"`
	ComparisonThreshold float64                `json:"comparison_threshold"`
	CustomMetrics       map[string]interface{} `json:"custom_metrics,omitempty"`
}

// DefaultReportConfig returns default report configuration
func DefaultReportConfig() ReportConfig {
	return ReportConfig{
		OutputDirectory:     "./reports",
		BaselineFile:        "baseline.json",
		IncludeProfiles:     true,
		GenerateHTML:        true,
		GenerateJSON:        true,
		GenerateMarkdown:    true,
		ComparisonThreshold: 5.0, // 5% threshold for significance
	}
}

// PerformanceReporter manages performance report generation
type PerformanceReporter struct {
	config   ReportConfig
	baseline *PerformanceReport
}

// NewPerformanceReporter creates a new performance reporter
func NewPerformanceReporter(config ReportConfig) *PerformanceReporter {
	return &PerformanceReporter{
		config: config,
	}
}

// LoadBaseline loads baseline metrics from file
func (pr *PerformanceReporter) LoadBaseline() error {
	baselinePath := filepath.Join(pr.config.OutputDirectory, pr.config.BaselineFile)

	data, err := os.ReadFile(baselinePath) //nolint:gosec // Reading from configured baseline file path
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrBaselineNotFound, baselinePath)
		}
		return fmt.Errorf("failed to read baseline file: %w", err)
	}

	var baseline PerformanceReport
	if err := json.Unmarshal(data, &baseline); err != nil {
		return fmt.Errorf("failed to parse baseline file: %w", err)
	}

	pr.baseline = &baseline
	return nil
}

// SaveBaseline saves current metrics as baseline
func (pr *PerformanceReporter) SaveBaseline(report *PerformanceReport) error {
	if err := os.MkdirAll(pr.config.OutputDirectory, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	baselinePath := filepath.Join(pr.config.OutputDirectory, pr.config.BaselineFile)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}

	if err := os.WriteFile(baselinePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write baseline file: %w", err)
	}

	pr.baseline = report
	return nil
}

// GenerateReport creates a comprehensive performance report
func (pr *PerformanceReporter) GenerateReport(currentMetrics map[string]float64, testResults []TestResult, profileSummary ProfileSummary) (*PerformanceReport, error) {
	report := &PerformanceReport{
		Timestamp:      time.Now(),
		ReportID:       generateReportID(),
		Version:        "1.0",
		CurrentMetrics: currentMetrics,
		Improvements:   make(map[string]float64),
		Regressions:    make(map[string]float64),
		TestResults:    testResults,
		ProfileSummary: profileSummary,
		SystemInfo:     getSystemInfo(),
	}

	// Add custom metrics if configured
	if pr.config.CustomMetrics != nil {
		for key, value := range pr.config.CustomMetrics {
			if floatVal, ok := value.(float64); ok {
				report.CurrentMetrics[key] = floatVal
			}
		}
	}

	// Calculate test statistics
	report.TotalTests = len(testResults)
	for _, result := range testResults {
		if result.Success {
			report.PassedTests++
		} else {
			report.FailedTests++
		}
	}

	// Compare with baseline if available
	if pr.baseline != nil {
		report.BaselineMetrics = pr.baseline.CurrentMetrics
		pr.calculatePerformanceChanges(report)
	}

	// Generate recommendations
	report.Recommendations = pr.generateRecommendations(report)

	return report, nil
}

// calculatePerformanceChanges compares current metrics with baseline
func (pr *PerformanceReporter) calculatePerformanceChanges(report *PerformanceReport) {
	for metric, currentValue := range report.CurrentMetrics {
		if baselineValue, exists := report.BaselineMetrics[metric]; exists {
			change := ((currentValue - baselineValue) / baselineValue) * 100

			if abs(change) >= pr.config.ComparisonThreshold {
				if change < 0 {
					// Negative change is improvement for metrics like duration, memory usage
					if isLowerBetterMetric(metric) {
						report.Improvements[metric] = abs(change)
					} else {
						report.Regressions[metric] = abs(change)
					}
				} else {
					// Positive change is improvement for metrics like throughput
					if isHigherBetterMetric(metric) {
						report.Improvements[metric] = change
					} else {
						report.Regressions[metric] = change
					}
				}
			}
		}
	}
}

// generateRecommendations creates performance optimization recommendations
func (pr *PerformanceReporter) generateRecommendations(report *PerformanceReport) []Recommendation {
	var recommendations []Recommendation

	// Memory-related recommendations
	recommendations = append(recommendations, pr.analyzeMemoryMetrics(report)...)

	// Performance regression recommendations
	recommendations = append(recommendations, pr.analyzeRegressions(report)...)

	// Test failure recommendations
	recommendations = append(recommendations, pr.analyzeTestFailures(report)...)

	// Profile-based recommendations
	recommendations = append(recommendations, pr.analyzeProfiles(report)...)

	// Sort by priority
	sort.Slice(recommendations, func(i, j int) bool {
		priorityOrder := map[RecommendationPriority]int{
			PriorityHigh:   3,
			PriorityMedium: 2,
			PriorityLow:    1,
		}
		return priorityOrder[recommendations[i].Priority] > priorityOrder[recommendations[j].Priority]
	})

	return recommendations
}

// analyzeMemoryMetrics generates memory-related recommendations
func (pr *PerformanceReporter) analyzeMemoryMetrics(report *PerformanceReport) []Recommendation {
	var recommendations []Recommendation

	// Check memory usage
	if memUsage, exists := report.CurrentMetrics["memory_usage_mb"]; exists && memUsage > 500 {
		recommendations = append(recommendations, Recommendation{
			ID:          "mem-high-usage",
			Priority:    PriorityHigh,
			Category:    "Memory",
			Title:       "High Memory Usage Detected",
			Description: fmt.Sprintf("Current memory usage is %.1f MB, which is above the recommended threshold", memUsage),
			Action:      "Consider implementing more aggressive memory pooling or review memory allocation patterns",
			Impact:      "High memory usage can lead to GC pressure and reduced performance",
			Evidence:    []string{fmt.Sprintf("Memory usage: %.1f MB", memUsage)},
		})
	}

	// Check memory growth
	if pr.baseline != nil {
		if currentMem, exists := report.CurrentMetrics["memory_usage_mb"]; exists {
			if baselineMem, exists := report.BaselineMetrics["memory_usage_mb"]; exists {
				growth := ((currentMem - baselineMem) / baselineMem) * 100
				if growth > 20 {
					recommendations = append(recommendations, Recommendation{
						ID:          "mem-growth",
						Priority:    PriorityMedium,
						Category:    "Memory",
						Title:       "Memory Usage Growth",
						Description: fmt.Sprintf("Memory usage has increased by %.1f%% since baseline", growth),
						Action:      "Investigate potential memory leaks or inefficient allocation patterns",
						Impact:      "Memory growth may indicate memory leaks or inefficient resource usage",
						Evidence: []string{
							fmt.Sprintf("Current: %.1f MB", currentMem),
							fmt.Sprintf("Baseline: %.1f MB", baselineMem),
							fmt.Sprintf("Growth: %.1f%%", growth),
						},
					})
				}
			}
		}
	}

	return recommendations
}

// analyzeRegressions generates recommendations for performance regressions
func (pr *PerformanceReporter) analyzeRegressions(report *PerformanceReport) []Recommendation {
	recommendations := make([]Recommendation, 0, len(report.Regressions))

	for metric, regression := range report.Regressions {
		priority := PriorityMedium
		if regression > 25 {
			priority = PriorityHigh
		} else if regression < 10 {
			priority = PriorityLow
		}

		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("regression-%s", strings.ReplaceAll(metric, "_", "-")),
			Priority:    priority,
			Category:    "Performance Regression",
			Title:       fmt.Sprintf("Performance Regression in %s", formatMetricName(metric)),
			Description: fmt.Sprintf("Performance has regressed by %.1f%% for %s", regression, metric),
			Action:      "Review recent changes and profile the affected code path",
			Impact:      "Performance regression may affect user experience and system throughput",
			Evidence:    []string{fmt.Sprintf("Regression: %.1f%%", regression)},
		})
	}

	return recommendations
}

// analyzeTestFailures generates recommendations for test failures
func (pr *PerformanceReporter) analyzeTestFailures(report *PerformanceReport) []Recommendation {
	var recommendations []Recommendation

	if report.FailedTests > 0 {
		failureRate := float64(report.FailedTests) / float64(report.TotalTests) * 100

		priority := PriorityMedium
		if failureRate > 20 {
			priority = PriorityHigh
		}

		var failedTestNames []string
		for _, test := range report.TestResults {
			if !test.Success {
				failedTestNames = append(failedTestNames, test.Name)
			}
		}

		recommendations = append(recommendations, Recommendation{
			ID:          "test-failures",
			Priority:    priority,
			Category:    "Test Failures",
			Title:       "Performance Test Failures",
			Description: fmt.Sprintf("%d out of %d tests failed (%.1f%% failure rate)", report.FailedTests, report.TotalTests, failureRate),
			Action:      "Investigate and fix failing performance tests",
			Impact:      "Test failures may indicate performance degradation or infrastructure issues",
			Evidence:    append([]string{fmt.Sprintf("Failed tests: %d/%d", report.FailedTests, report.TotalTests)}, failedTestNames...),
		})
	}

	return recommendations
}

// analyzeProfiles generates recommendations based on profiling data
func (pr *PerformanceReporter) analyzeProfiles(report *PerformanceReport) []Recommendation {
	var recommendations []Recommendation

	// Check if profiles are unusually large
	if report.ProfileSummary.TotalProfileSize > 100*1024*1024 { // 100MB
		recommendations = append(recommendations, Recommendation{
			ID:          "large-profiles",
			Priority:    PriorityLow,
			Category:    "Profiling",
			Title:       "Large Profile Files",
			Description: fmt.Sprintf("Profile files are large (%.1f MB total)", float64(report.ProfileSummary.TotalProfileSize)/(1024*1024)),
			Action:      "Consider reducing profiling duration or sample rate",
			Impact:      "Large profile files may consume significant disk space",
			Evidence:    []string{fmt.Sprintf("Total profile size: %.1f MB", float64(report.ProfileSummary.TotalProfileSize)/(1024*1024))},
		})
	}

	return recommendations
}

// SaveReport saves the report in the configured formats
func (pr *PerformanceReporter) SaveReport(report *PerformanceReport) error {
	if err := os.MkdirAll(pr.config.OutputDirectory, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	baseFileName := fmt.Sprintf("performance_report_%s", report.ReportID)

	// Save JSON report
	if pr.config.GenerateJSON {
		if err := pr.saveJSONReport(report, filepath.Join(pr.config.OutputDirectory, baseFileName+".json")); err != nil {
			return fmt.Errorf("failed to save JSON report: %w", err)
		}
	}

	// Save Markdown report
	if pr.config.GenerateMarkdown {
		if err := pr.saveMarkdownReport(report, filepath.Join(pr.config.OutputDirectory, baseFileName+".md")); err != nil {
			return fmt.Errorf("failed to save Markdown report: %w", err)
		}
	}

	// Save HTML report
	if pr.config.GenerateHTML {
		if err := pr.saveHTMLReport(report, filepath.Join(pr.config.OutputDirectory, baseFileName+".html")); err != nil {
			return fmt.Errorf("failed to save HTML report: %w", err)
		}
	}

	return nil
}

// saveJSONReport saves the report as JSON
func (pr *PerformanceReporter) saveJSONReport(report *PerformanceReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0o600)
}

// saveMarkdownReport saves the report as Markdown
func (pr *PerformanceReporter) saveMarkdownReport(report *PerformanceReport, filename string) error {
	tmpl := template.Must(template.New("markdown").Funcs(pr.getTemplateFuncs()).Parse(markdownTemplate))

	file, err := os.Create(filename) //nolint:gosec // Creating output report file
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }() // Ignore error in defer

	return tmpl.Execute(file, report)
}

// saveHTMLReport saves the report as HTML
func (pr *PerformanceReporter) saveHTMLReport(report *PerformanceReport, filename string) error {
	tmpl := template.Must(template.New("html").Funcs(pr.getTemplateFuncs()).Parse(htmlTemplate))

	file, err := os.Create(filename) //nolint:gosec // Creating output report file
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }() // Ignore error in defer

	return tmpl.Execute(file, report)
}

// Helper functions

func generateReportID() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}

func getSystemInfo() SystemInfo {
	return SystemInfo{
		GoVersion:  runtime.Version(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		NumCPU:     runtime.NumCPU(),
		GOMAXPROCS: runtime.GOMAXPROCS(-1),
		Timestamp:  time.Now(),
	}
}

func isLowerBetterMetric(metric string) bool {
	lowerBetterMetrics := []string{
		"duration", "latency", "memory", "allocation", "gc_pause", "response_time",
	}

	for _, lbm := range lowerBetterMetrics {
		if strings.Contains(strings.ToLower(metric), lbm) {
			return true
		}
	}
	return false
}

func isHigherBetterMetric(metric string) bool {
	higherBetterMetrics := []string{
		"throughput", "ops_per_sec", "requests_per_sec", "performance", "speed",
	}

	for _, hbm := range higherBetterMetrics {
		if strings.Contains(strings.ToLower(metric), hbm) {
			return true
		}
	}
	return false
}

func formatMetricName(metric string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(metric, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// getTemplateFuncs returns template functions for report generation
func (pr *PerformanceReporter) getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatFloat": func(f float64) string {
			return fmt.Sprintf("%.2f", f)
		},
		"formatPercent": func(f float64) string {
			return fmt.Sprintf("%.1f%%", f)
		},
		"formatBytes": func(bytes int64) string {
			const unit = 1024
			if bytes < unit {
				return fmt.Sprintf("%d B", bytes)
			}
			div, exp := int64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
		},
		"title": func(v interface{}) string {
			s := fmt.Sprintf("%v", v)
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
		},
		"priorityClass": func(priority RecommendationPriority) string {
			switch priority {
			case PriorityHigh:
				return "priority-high"
			case PriorityMedium:
				return "priority-medium"
			case PriorityLow:
				return "priority-low"
			default:
				return ""
			}
		},
	}
}
