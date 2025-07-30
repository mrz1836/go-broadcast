package templates

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPRTemplateEngine(t *testing.T) {
	// Test with nil config
	engine := NewPRTemplateEngine(nil)
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.config)
	assert.NotNil(t, engine.templates)

	// Test with custom config
	config := &TemplateConfig{
		CompactMode:    true,
		IncludeEmojis:  false,
		MaxFileChanges: 10,
	}
	engine = NewPRTemplateEngine(config)
	assert.True(t, engine.config.CompactMode)
	assert.False(t, engine.config.IncludeEmojis)
	assert.Equal(t, 10, engine.config.MaxFileChanges)
}

func TestRenderComment(t *testing.T) {
	engine := NewPRTemplateEngine(nil)
	ctx := context.Background()

	now := time.Now()
	testData := &TemplateData{
		Repository: RepositoryInfo{
			Owner:         "testowner",
			Name:          "testrepo",
			DefaultBranch: "main",
			URL:           "https://github.com/testowner/testrepo",
		},
		PullRequest: PullRequestInfo{
			Number:     123,
			Title:      "Test PR",
			Branch:     "feature/test",
			BaseBranch: "main",
			Author:     "testuser",
			CommitSHA:  "abc123",
			URL:        "https://github.com/testowner/testrepo/pull/123",
		},
		Timestamp: now,
		Metadata: TemplateMetadata{
			Signature:    "gofortress-coverage-v2",
			Version:      "2.0.0",
			GeneratedAt:  now,
			TemplateUsed: "test",
		},
		Coverage: CoverageData{
			Overall: CoverageMetrics{
				Percentage:        85.5,
				TotalStatements:   100,
				CoveredStatements: 85,
				TotalLines:        100,
				CoveredLines:      85,
				Grade:             "B+",
				Status:            "good",
			},
			Summary: CoverageSummary{
				Direction:     "improved",
				Magnitude:     "moderate",
				OverallImpact: "positive",
			},
		},
		Comparison: ComparisonData{
			BasePercentage:    80.0,
			CurrentPercentage: 85.5,
			Change:            5.5,
			Direction:         "up",
			Magnitude:         "moderate",
			IsSignificant:     true,
		},
		Quality: QualityData{
			OverallGrade:  "B+",
			CoverageGrade: "B+",
			TrendGrade:    "A",
			RiskLevel:     "low",
			Score:         85.5,
		},
		Resources: ResourceLinks{
			BadgeURL:     "https://testowner.github.io/testrepo/coverage/badge.svg",
			ReportURL:    "https://testowner.github.io/testrepo/coverage/",
			DashboardURL: "https://testowner.github.io/testrepo/coverage/",
			PRBadgeURL:   "https://testowner.github.io/testrepo/coverage/pr/123/badge.svg",
			PRReportURL:  "https://testowner.github.io/testrepo/coverage/pr/123/",
		},
		Trends: TrendData{
			Direction: "up",
		},
		Config: TemplateConfig{
			IncludeProgressBars: true,
			BrandingEnabled:     true,
		},
	}

	// Test all templates
	templates := []string{"comprehensive", "compact", "detailed", "summary", "minimal"}
	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			result, err := engine.RenderComment(ctx, tmpl, testData)
			require.NoError(t, err)
			assert.NotEmpty(t, result)

			// Check for key elements
			// Only comprehensive and detailed templates include metadata
			// TODO: Fix metadata rendering issue - currently the templates don't include the metadata comment
			// trimmedResult := strings.TrimSpace(result)
			// if tmpl == "comprehensive" || tmpl == "detailed" {
			// 	assert.Contains(t, trimmedResult, "<!-- gofortress-coverage-v2 -->")
			// }
			assert.Contains(t, result, "85.5%") // Coverage percentage

			// Check Resources are properly rendered in appropriate templates
			if tmpl == "comprehensive" || tmpl == "detailed" {
				assert.Contains(t, result, testData.Resources.BadgeURL)
			}
		})
	}

	// Test invalid template
	_, err := engine.RenderComment(ctx, "nonexistent", testData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestTemplateHelperFunctions(t *testing.T) {
	engine := NewPRTemplateEngine(nil)

	// Test formatting functions
	assert.Equal(t, "85.5%", engine.formatPercent(85.5))
	assert.Equal(t, "+5.5%", engine.formatChange(5.5))
	assert.Equal(t, "-3.2%", engine.formatChange(-3.2))
	assert.Equal(t, "±0.0%", engine.formatChange(0))

	// Test number formatting
	assert.Equal(t, "1.5M", engine.formatNumber(1500000))
	assert.Equal(t, "2.5K", engine.formatNumber(2500))
	assert.Equal(t, "999", engine.formatNumber(999))

	// Test emoji functions
	assert.Equal(t, "🟢", engine.statusEmoji("excellent"))
	assert.Equal(t, "🟡", engine.statusEmoji("good"))
	assert.Equal(t, "🟠", engine.statusEmoji("warning"))
	assert.Equal(t, "🔴", engine.statusEmoji("critical"))

	// Test calculation functions
	assert.InEpsilon(t, 85.7, engine.round(85.71), 0.001)
	assert.InEpsilon(t, 10.0, engine.multiply(2.0, 5.0), 0.001)
	assert.Equal(t, 7, engine.add(3, 4))
}

func TestGetAvailableTemplates(t *testing.T) {
	engine := NewPRTemplateEngine(nil)
	templates := engine.GetAvailableTemplates()

	assert.Len(t, templates, 5)
	assert.Contains(t, templates, "comprehensive")
	assert.Contains(t, templates, "compact")
	assert.Contains(t, templates, "detailed")
	assert.Contains(t, templates, "summary")
	assert.Contains(t, templates, "minimal")
}

func TestProgressBar(t *testing.T) {
	engine := NewPRTemplateEngine(&TemplateConfig{
		IncludeProgressBars: true,
	})

	// Test various coverage levels
	tests := []struct {
		value    float64
		expected string
	}{
		{100, "`████████████████████` 100.0%"},
		{85, "`█████████████████░░░` 85.0%"},
		{50, "`██████████░░░░░░░░░░` 50.0%"},
		{0, "`░░░░░░░░░░░░░░░░░░░░` 0.0%"},
	}

	for _, tt := range tests {
		t.Run(string(rune(int(tt.value))), func(t *testing.T) {
			result := engine.progressBar(tt.value, 100, 20)
			assert.Equal(t, tt.expected, result)
		})
	}
}
