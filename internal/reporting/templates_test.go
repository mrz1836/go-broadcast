package reporting

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMarkdownTemplateBasic tests basic markdown template rendering
func TestMarkdownTemplateBasic(t *testing.T) {
	// Create a basic PerformanceReport
	report := &PerformanceReport{
		ReportID:    "test-123",
		Version:     "1.0.0",
		Timestamp:   time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC),
		TotalTests:  10,
		PassedTests: 8,
		FailedTests: 2,
		SystemInfo: SystemInfo{
			GoVersion:  "go1.21.0",
			GOOS:       "linux",
			GOARCH:     "amd64",
			NumCPU:     8,
			GOMAXPROCS: 8,
		},
		CurrentMetrics: map[string]float64{
			"throughput": 1234.56,
			"latency":    45.67,
		},
		TestResults: []TestResult{
			{
				Name:     "TestBenchmarkSync",
				Duration: 5 * time.Second,
				Success:  true,
			},
		},
		ProfileSummary: ProfileSummary{
			CPUProfile: ProfileInfo{
				Available: true,
				Size:      1024 * 1024,
				Path:      "/tmp/cpu.prof",
			},
			TotalProfileSize: 1024 * 1024,
		},
	}

	// Create template functions for testing
	funcMap := template.FuncMap{
		"formatFloat": func(_ float64) string {
			return "formatted_float"
		},
		"formatPercent": func(_ float64) string {
			return "formatted_percent"
		},
		"formatBytes": func(_ int64) string {
			return "formatted_bytes"
		},
		"title": func(_ interface{}) string {
			return "titled"
		},
		"priorityClass": func(_ RecommendationPriority) string {
			return "priority-class"
		},
	}

	t.Run("RenderBasicMarkdownTemplate", func(t *testing.T) {
		tmpl, err := template.New("markdown").Funcs(funcMap).Parse(markdownTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, report)
		require.NoError(t, err)

		output := buf.String()

		// Verify key sections are present
		assert.Contains(t, output, "# Performance Analysis Report")
		assert.Contains(t, output, "**Report ID:** test-123")
		assert.Contains(t, output, "**Generated:** 2024-01-15 12:30:45")
		assert.Contains(t, output, "**Version:** 1.0.0")
		assert.Contains(t, output, "## Executive Summary")
		assert.Contains(t, output, "- **Total Tests:** 10")
		assert.Contains(t, output, "- **Passed Tests:** 8")
		assert.Contains(t, output, "- **Failed Tests:** 2")
		assert.Contains(t, output, "## System Information")
		assert.Contains(t, output, "- **Go Version:** go1.21.0")
		assert.Contains(t, output, "- **Operating System:** linux")
		assert.Contains(t, output, "- **Architecture:** amd64")
		assert.Contains(t, output, "- **CPU Cores:** 8")
		assert.Contains(t, output, "- **GOMAXPROCS:** 8")
	})

	t.Run("MarkdownTemplateWithBaseline", func(t *testing.T) {
		reportWithBaseline := *report
		reportWithBaseline.BaselineMetrics = map[string]float64{
			"throughput": 1000.0,
			"latency":    50.0,
		}
		reportWithBaseline.Improvements = map[string]float64{
			"throughput": 23.456,
		}
		reportWithBaseline.Regressions = map[string]float64{
			"latency": 8.66,
		}

		tmpl, err := template.New("markdown").Funcs(funcMap).Parse(markdownTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, &reportWithBaseline)
		require.NoError(t, err)

		output := buf.String()

		// Verify baseline comparison sections
		assert.Contains(t, output, "- **Compared Against:** Baseline from 2024-01-15")
		assert.Contains(t, output, "### Performance Changes")
		assert.Contains(t, output, "#### Improvements ✅")
		assert.Contains(t, output, "#### Regressions ⚠️")
	})

	t.Run("MarkdownTemplateWithRecommendations", func(t *testing.T) {
		reportWithRecs := *report
		reportWithRecs.Recommendations = []Recommendation{
			{
				Title:       "Optimize Memory Usage",
				Priority:    PriorityHigh,
				Category:    "Performance",
				Description: "Memory usage is too high",
				Action:      "Reduce allocations",
				Impact:      "25% performance improvement",
				Evidence:    []string{"High allocation rate", "GC pressure"},
				References:  []string{"Go performance guide"},
			},
		}

		tmpl, err := template.New("markdown").Funcs(funcMap).Parse(markdownTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, &reportWithRecs)
		require.NoError(t, err)

		output := buf.String()

		// Verify recommendations section
		assert.Contains(t, output, "## Recommendations")
		assert.Contains(t, output, "### Optimize Memory Usage")
		assert.Contains(t, output, "**Priority:** titled")
		assert.Contains(t, output, "**Category:** Performance")
		assert.Contains(t, output, "Memory usage is too high")
		assert.Contains(t, output, "**Recommended Action:** Reduce allocations")
		assert.Contains(t, output, "**Impact:** 25% performance improvement")
		assert.Contains(t, output, "**Evidence:**")
		assert.Contains(t, output, "- High allocation rate")
		assert.Contains(t, output, "- GC pressure")
		assert.Contains(t, output, "**References:**")
		assert.Contains(t, output, "- Go performance guide")
	})
}

// TestHTMLTemplateBasic tests basic HTML template rendering
func TestHTMLTemplateBasic(t *testing.T) {
	report := &PerformanceReport{
		ReportID:    "html-test-456",
		Version:     "2.0.0",
		Timestamp:   time.Date(2024, 2, 20, 14, 15, 30, 0, time.UTC),
		TotalTests:  5,
		PassedTests: 5,
		FailedTests: 0,
		SystemInfo: SystemInfo{
			GoVersion:  "go1.21.5",
			GOOS:       "darwin",
			GOARCH:     "arm64",
			NumCPU:     12,
			GOMAXPROCS: 12,
		},
		CurrentMetrics: map[string]float64{
			"performance_score": 95.5,
			"memory_efficiency": 87.2,
		},
		TestResults: []TestResult{
			{
				Name:       "TestPerformance",
				Duration:   3 * time.Second,
				Success:    true,
				Throughput: 2000.0,
				MemoryUsed: 128,
			},
			{
				Name:    "TestFailure",
				Success: false,
				Error:   "timeout exceeded",
			},
		},
		ProfileSummary: ProfileSummary{
			CPUProfile: ProfileInfo{
				Available: true,
				Size:      2048 * 1024,
				Path:      "/tmp/cpu.prof",
			},
			MemoryProfile: ProfileInfo{
				Available: true,
				Size:      1536 * 1024,
				Path:      "/tmp/mem.prof",
			},
			GoroutineProfile: ProfileInfo{
				Available: false,
			},
			TotalProfileSize: 3584 * 1024,
		},
	}

	// Template functions for testing
	funcMap := template.FuncMap{
		"formatFloat": func(_ float64) string {
			return "float_formatted"
		},
		"formatPercent": func(_ float64) string {
			return "percent_formatted"
		},
		"formatBytes": func(_ int64) string {
			return "bytes_formatted"
		},
		"title": func(_ interface{}) string {
			return "Title_Case"
		},
		"priorityClass": func(priority RecommendationPriority) string {
			return "priority-" + string(priority)
		},
	}

	t.Run("RenderBasicHTMLTemplate", func(t *testing.T) {
		tmpl, err := template.New("html").Funcs(funcMap).Parse(htmlTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, report)
		require.NoError(t, err)

		output := buf.String()

		// Verify HTML structure
		assert.Contains(t, output, "<!DOCTYPE html>")
		assert.Contains(t, output, "<html lang=\"en\">")
		assert.Contains(t, output, "<head>")
		assert.Contains(t, output, "<body>")
		assert.Contains(t, output, "</html>")

		// Verify title and meta information
		assert.Contains(t, output, "<title>Performance Analysis Report - html-test-456</title>")
		assert.Contains(t, output, "<h1>Performance Analysis Report</h1>")
		assert.Contains(t, output, "<strong>Report ID:</strong> html-test-456")
		assert.Contains(t, output, "<strong>Generated:</strong> 2024-02-20 14:15:30")
		assert.Contains(t, output, "<strong>Version:</strong> 2.0.0")

		// Verify sections
		assert.Contains(t, output, "<h2>Executive Summary</h2>")
		assert.Contains(t, output, "<h2>System Information</h2>")
		assert.Contains(t, output, "<h2>Performance Metrics</h2>")
		assert.Contains(t, output, "<h2>Test Results</h2>")
		assert.Contains(t, output, "<h2>Profiling Summary</h2>")
		assert.Contains(t, output, "<h2>Recommendations</h2>")
		assert.Contains(t, output, "<h2>Conclusion</h2>")

		// Verify CSS is included
		assert.Contains(t, output, "<style>")
		assert.Contains(t, output, "font-family:")
		assert.Contains(t, output, ".metric-card")
		assert.Contains(t, output, ".test-result")
	})

	t.Run("HTMLTemplateWithTestResults", func(t *testing.T) {
		tmpl, err := template.New("html").Funcs(funcMap).Parse(htmlTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, report)
		require.NoError(t, err)

		output := buf.String()

		// Verify test results rendering
		assert.Contains(t, output, "TestPerformance")
		assert.Contains(t, output, "test-pass")
		assert.Contains(t, output, "status-pass")
		assert.Contains(t, output, "TestFailure")
		assert.Contains(t, output, "test-fail")
		assert.Contains(t, output, "status-fail")
		assert.Contains(t, output, "timeout exceeded")
	})

	t.Run("HTMLTemplateWithProfileSummary", func(t *testing.T) {
		tmpl, err := template.New("html").Funcs(funcMap).Parse(htmlTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, report)
		require.NoError(t, err)

		output := buf.String()

		// Verify profile sections are rendered correctly
		assert.Contains(t, output, "CPU Profile")
		assert.Contains(t, output, "Memory Profile")
		assert.Contains(t, output, "Total Profile Size")
		// Goroutine profile should not appear since Available is false
		profileCount := strings.Count(output, "Goroutine Profile")
		assert.Equal(t, 0, profileCount, "Goroutine profile should not appear when not available")
	})
}

// TestTemplateEdgeCases tests edge cases and error conditions
func TestTemplateEdgeCases(t *testing.T) {
	t.Run("EmptyReport", func(t *testing.T) {
		report := &PerformanceReport{}
		funcMap := template.FuncMap{
			"formatFloat":   func(_ float64) string { return "0.00" },
			"formatPercent": func(_ float64) string { return "0.0%" },
			"formatBytes":   func(_ int64) string { return "0 B" },
			"title":         func(_ interface{}) string { return "" },
			"priorityClass": func(_ RecommendationPriority) string { return "" },
		}

		// Test markdown template with empty data
		tmpl, err := template.New("markdown").Funcs(funcMap).Parse(markdownTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, report)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "# Performance Analysis Report")
		assert.Contains(t, output, "No specific recommendations at this time")
	})

	t.Run("MissingTemplateFunctions", func(t *testing.T) {
		// Test parsing template with undefined functions - should fail
		_, err := template.New("markdown").Parse(markdownTemplate)
		require.Error(t, err, "Template parsing should fail with undefined functions")
		assert.Contains(t, err.Error(), "function")
		assert.Contains(t, err.Error(), "not defined")
	})

	t.Run("ZeroValues", func(t *testing.T) {
		report := &PerformanceReport{
			TotalTests:  0,
			PassedTests: 0,
			FailedTests: 0,
			CurrentMetrics: map[string]float64{
				"zero_metric": 0.0,
			},
			TestResults: []TestResult{},
		}

		funcMap := template.FuncMap{
			"formatFloat":   func(_ float64) string { return "0.00" },
			"formatPercent": func(_ float64) string { return "0.0%" },
			"formatBytes":   func(_ int64) string { return "0 B" },
			"title":         func(_ interface{}) string { return "Zero Metric" },
			"priorityClass": func(_ RecommendationPriority) string { return "" },
		}

		tmpl, err := template.New("markdown").Funcs(funcMap).Parse(markdownTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, report)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "- **Total Tests:** 0")
		assert.Contains(t, output, "- **Passed Tests:** 0")
		assert.Contains(t, output, "- **Failed Tests:** 0")
	})
}

// TestTemplateFunctionIntegration tests how template functions work with the templates
func TestTemplateFunctionIntegration(t *testing.T) {
	t.Run("FormatFloatFunction", func(t *testing.T) {
		report := &PerformanceReport{
			CurrentMetrics: map[string]float64{
				"test_metric": 123.456789,
			},
		}

		funcMap := template.FuncMap{
			"formatFloat": func(_ float64) string {
				return "123.46" // Test specific formatting
			},
			"formatPercent": func(_ float64) string { return "0.0%" },
			"formatBytes":   func(_ int64) string { return "0 B" },
			"title":         func(_ interface{}) string { return "Test Metric" },
			"priorityClass": func(_ RecommendationPriority) string { return "" },
		}

		tmpl, err := template.New("markdown").Funcs(funcMap).Parse(markdownTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, report)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "123.46")
	})

	t.Run("TitleFunction", func(t *testing.T) {
		report := &PerformanceReport{
			CurrentMetrics: map[string]float64{
				"memory_usage": 456.789,
			},
		}

		funcMap := template.FuncMap{
			"formatFloat":   func(_ float64) string { return "456.79" },
			"formatPercent": func(_ float64) string { return "0.0%" },
			"formatBytes":   func(_ int64) string { return "0 B" },
			"title":         func(_ interface{}) string { return "Memory Usage" },
			"priorityClass": func(_ RecommendationPriority) string { return "" },
		}

		tmpl, err := template.New("markdown").Funcs(funcMap).Parse(markdownTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, report)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Memory Usage")
	})

	t.Run("PriorityClassFunction", func(t *testing.T) {
		report := &PerformanceReport{
			Recommendations: []Recommendation{
				{
					Title:    "Test Recommendation",
					Priority: PriorityHigh,
				},
			},
		}

		funcMap := template.FuncMap{
			"formatFloat":   func(_ float64) string { return "0.00" },
			"formatPercent": func(_ float64) string { return "0.0%" },
			"formatBytes":   func(_ int64) string { return "0 B" },
			"title":         func(_ interface{}) string { return "Test" },
			"priorityClass": func(_ RecommendationPriority) string {
				return "priority-high"
			},
		}

		tmpl, err := template.New("html").Funcs(funcMap).Parse(htmlTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, report)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "priority-high")
	})
}

// TestTemplateConditionals tests conditional rendering in templates
func TestTemplateConditionals(t *testing.T) {
	t.Run("BaselineMetricsConditional", func(t *testing.T) {
		// Test without baseline metrics
		reportWithoutBaseline := &PerformanceReport{
			ReportID: "test",
		}

		funcMap := template.FuncMap{
			"formatFloat":   func(_ float64) string { return "0.00" },
			"formatPercent": func(_ float64) string { return "0.0%" },
			"formatBytes":   func(_ int64) string { return "0 B" },
			"title":         func(_ interface{}) string { return "" },
			"priorityClass": func(_ RecommendationPriority) string { return "" },
		}

		tmpl, err := template.New("markdown").Funcs(funcMap).Parse(markdownTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, reportWithoutBaseline)
		require.NoError(t, err)

		output := buf.String()
		// Should not contain baseline-specific text
		assert.NotContains(t, output, "Compared Against")
		assert.NotContains(t, output, "Performance Changes")

		// Test with baseline metrics
		reportWithBaseline := &PerformanceReport{
			ReportID: "test",
			BaselineMetrics: map[string]float64{
				"test": 1.0,
			},
			Improvements: map[string]float64{
				"test": 10.0,
			},
		}

		buf.Reset()
		err = tmpl.Execute(&buf, reportWithBaseline)
		require.NoError(t, err)

		output = buf.String()
		assert.Contains(t, output, "Performance Changes")
		assert.Contains(t, output, "Improvements ✅")
	})

	t.Run("RecommendationsConditional", func(t *testing.T) {
		// Test without recommendations
		reportWithoutRecs := &PerformanceReport{
			ReportID: "test",
		}

		funcMap := template.FuncMap{
			"formatFloat":   func(_ float64) string { return "0.00" },
			"formatPercent": func(_ float64) string { return "0.0%" },
			"formatBytes":   func(_ int64) string { return "0 B" },
			"title":         func(_ interface{}) string { return "" },
			"priorityClass": func(_ RecommendationPriority) string { return "" },
		}

		tmpl, err := template.New("markdown").Funcs(funcMap).Parse(markdownTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, reportWithoutRecs)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "No specific recommendations at this time")
	})

	t.Run("FailedTestsConditional", func(t *testing.T) {
		reportWithFailures := &PerformanceReport{
			ReportID:    "test",
			FailedTests: 3,
		}

		funcMap := template.FuncMap{
			"formatFloat":   func(_ float64) string { return "0.00" },
			"formatPercent": func(_ float64) string { return "0.0%" },
			"formatBytes":   func(_ int64) string { return "0 B" },
			"title":         func(_ interface{}) string { return "" },
			"priorityClass": func(_ RecommendationPriority) string { return "" },
		}

		tmpl, err := template.New("markdown").Funcs(funcMap).Parse(markdownTemplate)
		require.NoError(t, err)

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, reportWithFailures)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "3 performance test(s) failed")
	})
}
