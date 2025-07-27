package dashboard

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/config"
)

func TestNewDashboard(t *testing.T) {
	cfg := &config.Config{
		Analytics: config.AnalyticsConfig{
			Dashboard: config.DashboardConfig{
				Port:         8080,
				RefreshRate:  30,
				MaxDataPoints: 1000,
				Theme:        "light",
			},
		},
	}
	
	dashboard := NewDashboard(cfg)
	if dashboard == nil {
		t.Fatal("NewDashboard returned nil")
	}
	if dashboard.config != cfg {
		t.Error("Dashboard config not set correctly")
	}
}

func TestGenerateDashboardHTML(t *testing.T) {
	dashboard := NewDashboard(&config.Config{})
	
	data := DashboardData{
		Title:       "Coverage Dashboard",
		RefreshRate: 30,
		Repository:  "test/repo",
		Branch:      "main",
		LastUpdate:  time.Now(),
		OverallCoverage: CoverageOverview{
			Current:    85.5,
			Previous:   83.0,
			Target:     90.0,
			Threshold:  80.0,
			Change:     2.5,
			Status:     "improving",
		},
		TrendData: []TrendDataPoint{
			{
				Date:     time.Now().Add(-24 * time.Hour),
				Coverage: 83.0,
				Files:    45,
				Lines:    2500,
			},
			{
				Date:     time.Now(),
				Coverage: 85.5,
				Files:    47,
				Lines:    2650,
			},
		},
		FileMetrics: []FileMetric{
			{
				Path:     "src/main.go",
				Coverage: 90.0,
				Lines:    150,
				Status:   "good",
			},
			{
				Path:     "src/utils.go",
				Coverage: 75.0,
				Lines:    80,
				Status:   "needs_improvement",
			},
		},
		TestSuites: []TestSuiteInfo{
			{
				Name:     "Unit Tests",
				Coverage: 88.0,
				Tests:    125,
				Status:   "passing",
			},
			{
				Name:     "Integration Tests",
				Coverage: 82.0,
				Tests:    45,
				Status:   "passing",
			},
		},
	}
	
	html, err := dashboard.GenerateDashboardHTML(data)
	if err != nil {
		t.Fatalf("GenerateDashboardHTML() error = %v", err)
	}
	
	// Verify HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("Generated HTML should contain DOCTYPE declaration")
	}
	
	if !strings.Contains(html, data.Title) {
		t.Error("Generated HTML should contain dashboard title")
	}
	
	if !strings.Contains(html, "85.5%") {
		t.Error("Generated HTML should contain current coverage")
	}
	
	if !strings.Contains(html, data.Repository) {
		t.Error("Generated HTML should contain repository name")
	}
	
	// Verify chart containers
	expectedCharts := []string{
		"coverage-trend-chart",
		"file-coverage-chart",
		"test-suite-chart",
	}
	
	for _, chartId := range expectedCharts {
		if !strings.Contains(html, chartId) {
			t.Errorf("Generated HTML should contain chart container: %s", chartId)
		}
	}
}

func TestGenerateRealTimeDashboard(t *testing.T) {
	dashboard := NewDashboard(&config.Config{})
	
	metrics := RealTimeMetrics{
		CurrentCoverage: 87.2,
		TrendDirection:  "upward",
		ActiveTests:     234,
		PassingTests:    228,
		FailingTests:    6,
		TestVelocity:    15.5,
		LastCommit: CommitInfo{
			SHA:       "abc123def456",
			Author:    "developer",
			Message:   "Add new feature tests",
			Timestamp: time.Now().Add(-2 * time.Hour),
		},
		QualityGates: []QualityGateStatus{
			{
				Name:   "Minimum Coverage",
				Status: "passing",
				Value:  "87.2%",
				Target: "80%",
			},
			{
				Name:   "Test Success Rate",
				Status: "warning",
				Value:  "97.4%",
				Target: "98%",
			},
		},
	}
	
	html, err := dashboard.GenerateRealTimeDashboard(metrics)
	if err != nil {
		t.Fatalf("GenerateRealTimeDashboard() error = %v", err)
	}
	
	// Verify real-time elements
	expectedElements := []string{
		"87.2%",                    // Current coverage
		"upward",                   // Trend direction
		"234",                      // Active tests
		"developer",                // Commit author
		"Minimum Coverage",         // Quality gate name
		"real-time-metrics",        // Real-time container
		"auto-refresh",            // Auto-refresh functionality
	}
	
	for _, element := range expectedElements {
		if !strings.Contains(html, element) {
			t.Errorf("Real-time dashboard should contain: %s", element)
		}
	}
}

func TestGenerateComparisonDashboard(t *testing.T) {
	dashboard := NewDashboard(&config.Config{})
	
	comparison := ComparisonDashboardData{
		Title:       "Branch Comparison",
		Branches: []BranchComparison{
			{
				Name:     "main",
				Coverage: 85.0,
				Files:    50,
				Lines:    3000,
				Status:   "stable",
			},
			{
				Name:     "feature/new-api",
				Coverage: 87.5,
				Files:    52,
				Lines:    3150,
				Status:   "improving",
			},
			{
				Name:     "develop",
				Coverage: 83.5,
				Files:    51,
				Lines:    3080,
				Status:   "declining",
			},
		},
		PRComparisons: []PRComparison{
			{
				Number:        123,
				Title:         "Add API endpoints",
				Author:        "dev1",
				CoverageImpact: 2.5,
				Status:        "positive",
			},
			{
				Number:        124,
				Title:         "Refactor core module",
				Author:        "dev2",
				CoverageImpact: -1.0,
				Status:        "negative",
			},
		},
	}
	
	html, err := dashboard.GenerateComparisonDashboard(comparison)
	if err != nil {
		t.Fatalf("GenerateComparisonDashboard() error = %v", err)
	}
	
	// Verify comparison elements
	if !strings.Contains(html, "main") {
		t.Error("Should contain main branch")
	}
	
	if !strings.Contains(html, "feature/new-api") {
		t.Error("Should contain feature branch")
	}
	
	if !strings.Contains(html, "Add API endpoints") {
		t.Error("Should contain PR title")
	}
	
	if !strings.Contains(html, "comparison-chart") {
		t.Error("Should contain comparison chart container")
	}
}

func TestGenerateTeamDashboard(t *testing.T) {
	dashboard := NewDashboard(&config.Config{})
	
	teamData := TeamDashboardData{
		Title: "Team Coverage Dashboard",
		Team:  "Backend Team",
		Members: []TeamMemberMetrics{
			{
				Name:            "Alice Developer",
				Username:        "alice",
				Commits:         25,
				CoverageImpact:  3.5,
				TestsAdded:      15,
				AverageCoverage: 88.0,
				Ranking:         1,
			},
			{
				Name:            "Bob Engineer",
				Username:        "bob",
				Commits:         18,
				CoverageImpact:  1.2,
				TestsAdded:      8,
				AverageCoverage: 85.5,
				Ranking:         2,
			},
		},
		TeamMetrics: TeamOverallMetrics{
			TotalCommits:     43,
			AverageCoverage:  86.8,
			TotalTests:       23,
			CoverageVelocity: 2.4,
			QualityScore:     92.0,
		},
		Repositories: []RepositoryMetrics{
			{
				Name:     "backend-api",
				Coverage: 88.0,
				Files:    45,
				Contributors: 2,
				Status:   "healthy",
			},
		},
	}
	
	html, err := dashboard.GenerateTeamDashboard(teamData)
	if err != nil {
		t.Fatalf("GenerateTeamDashboard() error = %v", err)
	}
	
	// Verify team dashboard elements
	expectedElements := []string{
		"Backend Team",
		"Alice Developer",
		"Bob Engineer",
		"86.8%",         // Average coverage
		"backend-api",   // Repository name
		"team-metrics",  // Team metrics container
	}
	
	for _, element := range expectedElements {
		if !strings.Contains(html, element) {
			t.Errorf("Team dashboard should contain: %s", element)
		}
	}
}

func TestDashboardTemplateRendering(t *testing.T) {
	dashboard := NewDashboard(&config.Config{})
	
	templateData := map[string]interface{}{
		"Title":      "Test Dashboard",
		"Coverage":   85.5,
		"Repository": "test/repo",
		"Items": []map[string]interface{}{
			{"Name": "Item 1", "Value": 100},
			{"Name": "Item 2", "Value": 200},
		},
	}
	
	template := `
		<h1>{{.Title}}</h1>
		<p>Coverage: {{.Coverage}}%</p>
		<p>Repository: {{.Repository}}</p>
		<ul>
		{{range .Items}}
			<li>{{.Name}}: {{.Value}}</li>
		{{end}}
		</ul>
	`
	
	rendered, err := dashboard.renderTemplate(template, templateData)
	if err != nil {
		t.Fatalf("renderTemplate() error = %v", err)
	}
	
	expectedElements := []string{
		"Test Dashboard",
		"85.5%",
		"test/repo",
		"Item 1: 100",
		"Item 2: 200",
	}
	
	for _, element := range expectedElements {
		if !strings.Contains(rendered, element) {
			t.Errorf("Rendered template should contain: %s", element)
		}
	}
}

func TestDashboardChartGeneration(t *testing.T) {
	dashboard := NewDashboard(&config.Config{})
	
	tests := []struct {
		name      string
		chartType string
		data      interface{}
	}{
		{
			name:      "trend chart",
			chartType: "trend",
			data: []TrendDataPoint{
				{Date: time.Now(), Coverage: 85.0},
				{Date: time.Now().Add(time.Hour), Coverage: 87.0},
			},
		},
		{
			name:      "file coverage chart",
			chartType: "file_coverage",
			data: []FileMetric{
				{Path: "file1.go", Coverage: 90.0},
				{Path: "file2.go", Coverage: 75.0},
			},
		},
		{
			name:      "test suite chart",
			chartType: "test_suite",
			data: []TestSuiteInfo{
				{Name: "Unit", Coverage: 88.0, Tests: 100},
				{Name: "Integration", Coverage: 82.0, Tests: 50},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chartHTML, err := dashboard.generateChart(tt.chartType, tt.data)
			if err != nil {
				t.Errorf("generateChart(%s) error = %v", tt.chartType, err)
			}
			
			if chartHTML == "" {
				t.Errorf("generateChart(%s) returned empty HTML", tt.chartType)
			}
			
			// Charts should contain SVG or canvas elements
			if !strings.Contains(chartHTML, "<svg") && !strings.Contains(chartHTML, "<canvas") {
				t.Errorf("Chart HTML should contain SVG or canvas element")
			}
		})
	}
}

func TestDashboardFiltering(t *testing.T) {
	dashboard := NewDashboard(&config.Config{})
	
	data := DashboardData{
		FileMetrics: []FileMetric{
			{Path: "src/good.go", Coverage: 90.0, Status: "good"},
			{Path: "src/ok.go", Coverage: 80.0, Status: "ok"},
			{Path: "src/poor.go", Coverage: 60.0, Status: "needs_improvement"},
		},
	}
	
	tests := []struct {
		name           string
		filter         DashboardFilter
		expectedCount  int
	}{
		{
			name: "no filter",
			filter: DashboardFilter{},
			expectedCount: 3,
		},
		{
			name: "minimum coverage filter",
			filter: DashboardFilter{
				MinCoverage: 75.0,
			},
			expectedCount: 2,
		},
		{
			name: "status filter",
			filter: DashboardFilter{
				Status: []string{"good"},
			},
			expectedCount: 1,
		},
		{
			name: "path pattern filter",
			filter: DashboardFilter{
				PathPattern: "src/g*",
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := dashboard.applyFilter(data, tt.filter)
			if len(filtered.FileMetrics) != tt.expectedCount {
				t.Errorf("Expected %d files after filtering, got %d", 
					tt.expectedCount, len(filtered.FileMetrics))
			}
		})
	}
}

func TestDashboardExport(t *testing.T) {
	dashboard := NewDashboard(&config.Config{})
	
	data := DashboardData{
		Title:           "Export Test",
		OverallCoverage: CoverageOverview{Current: 85.5},
		FileMetrics: []FileMetric{
			{Path: "test.go", Coverage: 90.0},
		},
	}
	
	tests := []struct {
		name   string
		format string
	}{
		{"HTML export", "html"},
		{"PDF export", "pdf"},
		{"JSON export", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exported, err := dashboard.ExportDashboard(data, tt.format)
			if err != nil {
				t.Errorf("ExportDashboard(%s) error = %v", tt.format, err)
			}
			
			if len(exported) == 0 {
				t.Errorf("ExportDashboard(%s) returned empty data", tt.format)
			}
		})
	}
}

func TestDashboardWebSocketUpdates(t *testing.T) {
	dashboard := NewDashboard(&config.Config{})
	
	// Simulate WebSocket update
	update := DashboardUpdate{
		Type:      "coverage_change",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"coverage": 86.0,
			"change":   0.5,
		},
	}
	
	updateJSON, err := dashboard.GenerateWebSocketUpdate(update)
	if err != nil {
		t.Fatalf("GenerateWebSocketUpdate() error = %v", err)
	}
	
	if !strings.Contains(updateJSON, "coverage_change") {
		t.Error("WebSocket update should contain update type")
	}
	
	if !strings.Contains(updateJSON, "86.0") {
		t.Error("WebSocket update should contain coverage data")
	}
}

func BenchmarkGenerateDashboardHTML(b *testing.B) {
	dashboard := NewDashboard(&config.Config{})
	
	// Generate large dataset
	fileMetrics := make([]FileMetric, 100)
	for i := 0; i < 100; i++ {
		fileMetrics[i] = FileMetric{
			Path:     fmt.Sprintf("src/file%d.go", i),
			Coverage: 80.0 + float64(i%20),
			Lines:    100 + i*10,
			Status:   "good",
		}
	}
	
	data := DashboardData{
		Title:           "Benchmark Dashboard",
		Repository:      "test/repo",
		OverallCoverage: CoverageOverview{Current: 85.0},
		FileMetrics:     fileMetrics,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dashboard.GenerateDashboardHTML(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateRealTimeDashboard(b *testing.B) {
	dashboard := NewDashboard(&config.Config{})
	
	metrics := RealTimeMetrics{
		CurrentCoverage: 87.2,
		TrendDirection:  "upward",
		ActiveTests:     234,
		PassingTests:    228,
		FailingTests:    6,
		QualityGates: []QualityGateStatus{
			{Name: "Coverage", Status: "passing", Value: "87.2%"},
			{Name: "Tests", Status: "passing", Value: "97.4%"},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dashboard.GenerateRealTimeDashboard(metrics)
		if err != nil {
			b.Fatal(err)
		}
	}
}