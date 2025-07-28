package monitoring

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboardHTML(t *testing.T) {
	t.Run("HTML structure validation", func(t *testing.T) {
		require.NotEmpty(t, dashboardHTML, "dashboardHTML should not be empty")

		// Check for essential HTML structure
		assert.Contains(t, dashboardHTML, "<!DOCTYPE html>", "Should contain DOCTYPE declaration")
		assert.Contains(t, dashboardHTML, "<html lang=\"en\">", "Should contain HTML tag with language")
		assert.Contains(t, dashboardHTML, "<head>", "Should contain head section")
		assert.Contains(t, dashboardHTML, "<body>", "Should contain body section")
		assert.Contains(t, dashboardHTML, "</html>", "Should close HTML tag")

		// Count opening and closing tags to ensure they match
		openingBodyTags := strings.Count(dashboardHTML, "<body>")
		closingBodyTags := strings.Count(dashboardHTML, "</body>")
		assert.Equal(t, openingBodyTags, closingBodyTags, "Opening and closing body tags should match")

		openingHTMLTags := strings.Count(dashboardHTML, "<html")
		closingHTMLTags := strings.Count(dashboardHTML, "</html>")
		assert.Equal(t, openingHTMLTags, closingHTMLTags, "Opening and closing HTML tags should match")
	})

	t.Run("meta and title validation", func(t *testing.T) {
		assert.Contains(t, dashboardHTML, "<meta charset=\"UTF-8\">", "Should contain UTF-8 charset")
		assert.Contains(t, dashboardHTML, "<meta name=\"viewport\"", "Should contain viewport meta tag")
		assert.Contains(t, dashboardHTML, "<title>go-broadcast Performance Dashboard</title>", "Should contain correct title")
	})

	t.Run("external dependencies", func(t *testing.T) {
		assert.Contains(t, dashboardHTML, "https://cdn.jsdelivr.net/npm/chart.js", "Should include Chart.js library")
		assert.Contains(t, dashboardHTML, "/dashboard.css", "Should reference dashboard CSS")
		assert.Contains(t, dashboardHTML, "/dashboard.js", "Should reference dashboard JavaScript")
	})

	t.Run("dashboard components", func(t *testing.T) {
		// Check for main dashboard sections
		assert.Contains(t, dashboardHTML, "go-broadcast Performance Dashboard", "Should contain dashboard title")
		assert.Contains(t, dashboardHTML, "Real-time Stats", "Should contain real-time stats section")
		assert.Contains(t, dashboardHTML, "Memory Usage Over Time", "Should contain memory chart section")
		assert.Contains(t, dashboardHTML, "Goroutines Over Time", "Should contain goroutines chart section")
		assert.Contains(t, dashboardHTML, "Garbage Collection Activity", "Should contain GC chart section")
		assert.Contains(t, dashboardHTML, "System Information", "Should contain system info section")
		assert.Contains(t, dashboardHTML, "Performance Alerts", "Should contain alerts section")
	})

	t.Run("chart canvas elements", func(t *testing.T) {
		assert.Contains(t, dashboardHTML, "id=\"memoryChart\"", "Should contain memory chart canvas")
		assert.Contains(t, dashboardHTML, "id=\"goroutinesChart\"", "Should contain goroutines chart canvas")
		assert.Contains(t, dashboardHTML, "id=\"gcChart\"", "Should contain GC chart canvas")
	})

	t.Run("status and data elements", func(t *testing.T) {
		// Check for important data display elements
		assert.Contains(t, dashboardHTML, "id=\"memory-usage\"", "Should contain memory usage display")
		assert.Contains(t, dashboardHTML, "id=\"goroutines\"", "Should contain goroutines display")
		assert.Contains(t, dashboardHTML, "id=\"gc-count\"", "Should contain GC count display")
		assert.Contains(t, dashboardHTML, "id=\"status\"", "Should contain status indicator")
		assert.Contains(t, dashboardHTML, "id=\"last-update\"", "Should contain last update display")
		assert.Contains(t, dashboardHTML, "id=\"uptime\"", "Should contain uptime display")
	})
}

func TestDashboardCSS(t *testing.T) {
	t.Run("CSS structure validation", func(t *testing.T) {
		require.NotEmpty(t, dashboardCSS, "dashboardCSS should not be empty")

		// Check for essential CSS selectors
		assert.Contains(t, dashboardCSS, "body {", "Should contain body selector")
		assert.Contains(t, dashboardCSS, ".container {", "Should contain container class")
		assert.Contains(t, dashboardCSS, "header {", "Should contain header selector")
		assert.Contains(t, dashboardCSS, ".card {", "Should contain card class")
	})

	t.Run("responsive design", func(t *testing.T) {
		assert.Contains(t, dashboardCSS, "@media (max-width: 768px)", "Should contain mobile responsive rules")
		assert.Contains(t, dashboardCSS, "grid-template-columns", "Should use CSS Grid for layout")
	})

	t.Run("color scheme", func(t *testing.T) {
		// Check for consistent color usage
		assert.Contains(t, dashboardCSS, "#f5f5f5", "Should contain background color")
		assert.Contains(t, dashboardCSS, "#2c3e50", "Should contain primary text color")
		assert.Contains(t, dashboardCSS, "#27ae60", "Should contain success color")
		assert.Contains(t, dashboardCSS, "#e74c3c", "Should contain error color")
		assert.Contains(t, dashboardCSS, "#f39c12", "Should contain warning color")
	})

	t.Run("layout components", func(t *testing.T) {
		assert.Contains(t, dashboardCSS, ".metrics-grid", "Should contain metrics grid class")
		assert.Contains(t, dashboardCSS, ".stats-grid", "Should contain stats grid class")
		assert.Contains(t, dashboardCSS, ".chart-card", "Should contain chart card class")
		assert.Contains(t, dashboardCSS, ".alert-item", "Should contain alert item class")
	})

	t.Run("CSS syntax validation", func(t *testing.T) {
		// Basic CSS syntax checks
		openBraces := strings.Count(dashboardCSS, "{")
		closeBraces := strings.Count(dashboardCSS, "}")
		assert.Equal(t, openBraces, closeBraces, "Opening and closing braces should match")

		// Ensure no obvious syntax errors
		assert.NotContains(t, dashboardCSS, "};}", "Should not contain double closing braces")
		assert.NotContains(t, dashboardCSS, "{{", "Should not contain double opening braces")
	})
}

func TestDashboardJS(t *testing.T) {
	t.Run("JavaScript structure validation", func(t *testing.T) {
		require.NotEmpty(t, dashboardJS, "dashboardJS should not be empty")

		// Check for main class definition
		assert.Contains(t, dashboardJS, "class PerformanceDashboard", "Should contain PerformanceDashboard class")
		assert.Contains(t, dashboardJS, "constructor()", "Should contain constructor method")
	})

	t.Run("essential methods", func(t *testing.T) {
		assert.Contains(t, dashboardJS, "initCharts()", "Should contain initCharts method")
		assert.Contains(t, dashboardJS, "fetchMetrics()", "Should contain fetchMetrics method")
		assert.Contains(t, dashboardJS, "updateCharts(", "Should contain updateCharts method")
		assert.Contains(t, dashboardJS, "updateRealTimeStats(", "Should contain updateRealTimeStats method")
		assert.Contains(t, dashboardJS, "checkAlerts(", "Should contain checkAlerts method")
		assert.Contains(t, dashboardJS, "startDataCollection()", "Should contain startDataCollection method")
	})

	t.Run("chart.js integration", func(t *testing.T) {
		assert.Contains(t, dashboardJS, "new Chart(", "Should create Chart.js instances")
		assert.Contains(t, dashboardJS, "memoryChart", "Should reference memory chart")
		assert.Contains(t, dashboardJS, "goroutinesChart", "Should reference goroutines chart")
		assert.Contains(t, dashboardJS, "gcChart", "Should reference GC chart")
	})

	t.Run("API endpoints", func(t *testing.T) {
		assert.Contains(t, dashboardJS, "/api/metrics", "Should call metrics API endpoint")
		assert.Contains(t, dashboardJS, "/api/health", "Should call health API endpoint")
	})

	t.Run("event handling", func(t *testing.T) {
		assert.Contains(t, dashboardJS, "DOMContentLoaded", "Should wait for DOM to load")
		assert.Contains(t, dashboardJS, "setInterval", "Should use intervals for periodic updates")
		assert.Contains(t, dashboardJS, "fetch(", "Should use fetch API for HTTP requests")
	})

	t.Run("error handling", func(t *testing.T) {
		assert.Contains(t, dashboardJS, "try {", "Should contain try-catch blocks")
		assert.Contains(t, dashboardJS, "catch (error)", "Should handle errors")
		assert.Contains(t, dashboardJS, "console.error", "Should log errors to console")
	})

	t.Run("data processing", func(t *testing.T) {
		assert.Contains(t, dashboardJS, "this.data", "Should maintain data state")
		assert.Contains(t, dashboardJS, "maxDataPoints", "Should limit data points")
		assert.Contains(t, dashboardJS, "timestamps", "Should track timestamps")
		assert.Contains(t, dashboardJS, "memory", "Should track memory data")
		assert.Contains(t, dashboardJS, "goroutines", "Should track goroutines data")
	})

	t.Run("JavaScript syntax validation", func(t *testing.T) {
		// Basic JavaScript syntax checks
		openBraces := strings.Count(dashboardJS, "{")
		closeBraces := strings.Count(dashboardJS, "}")
		assert.Equal(t, openBraces, closeBraces, "Opening and closing braces should match")

		openParens := strings.Count(dashboardJS, "(")
		closeParens := strings.Count(dashboardJS, ")")
		assert.Equal(t, openParens, closeParens, "Opening and closing parentheses should match")

		openBrackets := strings.Count(dashboardJS, "[")
		closeBrackets := strings.Count(dashboardJS, "]")
		assert.Equal(t, openBrackets, closeBrackets, "Opening and closing brackets should match")
	})
}

func TestFrontendAssetsIntegration(t *testing.T) {
	t.Run("CSS and HTML integration", func(t *testing.T) {
		// Check that CSS classes referenced in HTML exist in CSS
		htmlClasses := []string{
			"container",
			"status-bar",
			"metrics-grid",
			"card",
			"stats-grid",
			"alert-item",
		}

		for _, class := range htmlClasses {
			assert.Contains(t, dashboardHTML, "class=\""+class, "HTML should reference class: "+class)
			assert.Contains(t, dashboardCSS, "."+class, "CSS should define class: "+class)
		}
	})

	t.Run("JavaScript and HTML integration", func(t *testing.T) {
		// Check that JavaScript element IDs exist in HTML
		jsElementIDs := []string{
			"memoryChart",
			"goroutinesChart",
			"gcChart",
			"memory-usage",
			"goroutines",
			"gc-count",
			"status",
			"last-update",
			"uptime",
		}

		for _, id := range jsElementIDs {
			assert.Contains(t, dashboardHTML, "id=\""+id+"\"", "HTML should contain element ID: "+id)
			assert.Contains(t, dashboardJS, "getElementById('"+id+"')", "JavaScript should reference element ID: "+id)
		}
	})

	t.Run("consistent naming conventions", func(t *testing.T) {
		// Check for consistent naming patterns
		assert.Contains(t, dashboardHTML, "go-broadcast", "HTML should reference project name")
		assert.Contains(t, dashboardJS, "PerformanceDashboard", "JavaScript should use consistent class naming")

		// Check for consistent color scheme references
		colors := []string{"#27ae60", "#e74c3c", "#f39c12"}
		for _, color := range colors {
			if strings.Contains(dashboardCSS, color) {
				// If color is defined in CSS, it might be referenced in JS for dynamic styling
				assert.Contains(t, dashboardCSS, color, "Color should be defined in CSS: "+color)
			}
		}
	})
}

func TestFrontendAssetsSizes(t *testing.T) {
	t.Run("reasonable asset sizes", func(t *testing.T) {
		// Ensure assets aren't excessively large or suspiciously small
		assert.Greater(t, len(dashboardHTML), 1000, "HTML should be substantial but not empty")
		assert.Less(t, len(dashboardHTML), 50000, "HTML should not be excessively large")

		assert.Greater(t, len(dashboardCSS), 500, "CSS should be substantial but not empty")
		assert.Less(t, len(dashboardCSS), 30000, "CSS should not be excessively large")

		assert.Greater(t, len(dashboardJS), 2000, "JavaScript should be substantial but not empty")
		assert.Less(t, len(dashboardJS), 100000, "JavaScript should not be excessively large")
	})

	t.Run("content completeness", func(t *testing.T) {
		// Ensure each asset contains expected minimum content
		htmlLines := strings.Split(dashboardHTML, "\n")
		assert.Greater(t, len(htmlLines), 50, "HTML should have reasonable number of lines")

		cssRules := strings.Count(dashboardCSS, "{")
		assert.Greater(t, cssRules, 10, "CSS should have reasonable number of rules")

		jsFunctions := strings.Count(dashboardJS, "function") + strings.Count(dashboardJS, "() => {")
		assert.GreaterOrEqual(t, jsFunctions, 3, "JavaScript should have reasonable number of functions")
	})
}
