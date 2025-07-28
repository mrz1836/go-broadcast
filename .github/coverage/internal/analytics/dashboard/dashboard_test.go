package dashboard

import (
	"context"
	"testing"
	"time"
)

func TestNewAnalyticsDashboard(t *testing.T) {
	// Test with nil config
	dashboard := NewAnalyticsDashboard(nil)
	if dashboard == nil {
		t.Fatal("NewAnalyticsDashboard returned nil")
	}
	if dashboard.config == nil {
		t.Fatal("Dashboard config should not be nil")
	}
	if dashboard.config.Title != "Coverage Dashboard" {
		t.Error("Default title not set correctly")
	}

	// Test with provided config
	config := &DashboardConfig{
		Title:                "Test Dashboard",
		Theme:                ThemeLight,
		RefreshInterval:      30 * time.Second,
		EnablePredictions:    true,
		EnableImpactAnalysis: true,
	}
	dashboard = NewAnalyticsDashboard(config)
	if dashboard.config.Title != "Test Dashboard" {
		t.Error("Dashboard config not set correctly")
	}
}

func TestGenerateDashboard(t *testing.T) {
	dashboard := NewAnalyticsDashboard(nil)

	request := &Request{
		TimeRange: TimeRange{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		},
		IncludePredictions: true,
		IncludeTeamData:    false,
		RefreshCache:       false,
	}

	ctx := context.Background()
	data, err := dashboard.GenerateDashboard(ctx, request)
	if err != nil {
		t.Fatalf("GenerateDashboard() error = %v", err)
	}

	if data == nil {
		t.Fatal("GenerateDashboard returned nil data")
	}

	// Check that current metrics are populated
	if data.CurrentMetrics.Coverage == 0 {
		t.Error("Current metrics coverage should be populated")
	}
	if data.CurrentMetrics.LastUpdated.IsZero() {
		t.Error("Current metrics timestamp should be populated")
	}
}

func TestGenerateHTML(t *testing.T) {
	dashboard := NewAnalyticsDashboard(nil)

	data := &DashboardData{
		CurrentMetrics: CurrentMetrics{
			Coverage:       78.5,
			CoverageChange: 2.3,
			TrendDirection: "Upward",
			TrendStrength:  "Strong",
			LastUpdated:    time.Now(),
		},
		GeneratedAt: time.Now(),
	}

	ctx := context.Background()
	html, err := dashboard.GenerateHTML(ctx, data)
	if err != nil {
		t.Fatalf("GenerateHTML() error = %v", err)
	}

	if html == "" {
		t.Fatal("GenerateHTML returned empty string")
	}

	// Basic HTML structure checks
	if len(html) < 50 {
		t.Error("Generated HTML seems too short")
	}

	// Should contain basic HTML elements
	expectedElements := []string{"<html>", "<body>", "</html>", "</body>"}
	for _, element := range expectedElements {
		if !contains(html, element) {
			t.Errorf("Generated HTML should contain %s", element)
		}
	}
}

func TestSetComponents(_ *testing.T) {
	dashboard := NewAnalyticsDashboard(nil)

	// Test that SetComponents doesn't panic (it's a placeholder)
	dashboard.SetComponents(nil, nil, nil, nil, nil)
	// If we get here without panic, the test passes
}

func TestTimePresets(t *testing.T) {
	presets := []TimePreset{
		PresetLast24Hours,
		PresetLast7Days,
		PresetLast30Days,
		PresetLast90Days,
		PresetLastYear,
		PresetCustom,
	}

	expectedValues := []string{"24h", "7d", "30d", "90d", "1y", "custom"}

	for i, preset := range presets {
		if string(preset) != expectedValues[i] {
			t.Errorf("TimePreset %d: expected %s, got %s", i, expectedValues[i], string(preset))
		}
	}
}

func TestDashboardThemes(t *testing.T) {
	themes := []DashboardTheme{
		ThemeAuto,
		ThemeLight,
		ThemeDark,
	}

	expectedValues := []string{"auto", "light", "dark"}

	for i, theme := range themes {
		if string(theme) != expectedValues[i] {
			t.Errorf("DashboardTheme %d: expected %s, got %s", i, expectedValues[i], string(theme))
		}
	}
}

// Helper function to check if a string contains a substring
func contains(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
