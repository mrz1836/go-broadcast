package charts

import (
	"context"
	"testing"
	"time"
)

func TestNewSVGChartGenerator(t *testing.T) {
	// Test with nil config
	generator := NewSVGChartGenerator(nil)
	if generator == nil {
		t.Fatal("NewSVGChartGenerator returned nil")
	}
	if generator.config == nil {
		t.Fatal("Generator config should not be nil")
	}

	// Test with provided config
	config := &ChartConfig{
		Width:     800,
		Height:    400,
		ShowGrid:  true,
		ShowXAxis: true,
		ShowYAxis: true,
	}
	generator = NewSVGChartGenerator(config)
	if generator.config.Width != 800 {
		t.Error("Config width not set correctly")
	}
	if generator.config.Height != 400 {
		t.Error("Config height not set correctly")
	}
}

func TestGenerateTrendChart(t *testing.T) {
	generator := NewSVGChartGenerator(nil)

	data := &ChartData{
		Title:      "Coverage Trend",
		XAxisLabel: "Time",
		YAxisLabel: "Coverage %",
		Points: []DataPoint{
			{Timestamp: time.Now().Add(-24 * time.Hour), Value: 75.0},
			{Timestamp: time.Now().Add(-12 * time.Hour), Value: 80.0},
			{Timestamp: time.Now(), Value: 85.0},
		},
	}

	svg, err := generator.GenerateTrendChart(context.TODO(), data)
	if err != nil {
		t.Fatalf("GenerateTrendChart() error = %v", err)
	}

	if svg == "" {
		t.Fatal("GenerateTrendChart returned empty SVG")
	}

	// Basic SVG structure checks
	if !contains(svg, "<svg") {
		t.Error("Generated output should contain SVG element")
	}
	if !contains(svg, "</svg>") {
		t.Error("Generated output should be properly closed SVG")
	}
}

func TestGenerateAreaChart(t *testing.T) {
	generator := NewSVGChartGenerator(nil)

	data := &ChartData{
		Title:      "File Coverage",
		XAxisLabel: "Files",
		YAxisLabel: "Coverage %",
		Points: []DataPoint{
			{Value: 90.0, Label: "file1.go"},
			{Value: 75.0, Label: "file2.go"},
			{Value: 85.0, Label: "file3.go"},
		},
	}

	svg, err := generator.GenerateAreaChart(context.TODO(), data)
	if err != nil {
		t.Fatalf("GenerateAreaChart() error = %v", err)
	}

	if svg == "" {
		t.Fatal("GenerateAreaChart returned empty SVG")
	}

	// Basic SVG structure checks
	if !contains(svg, "<svg") {
		t.Error("Generated output should contain SVG element")
	}
	if !contains(svg, "</svg>") {
		t.Error("Generated output should be properly closed SVG")
	}
}

func TestSeriesTypes(t *testing.T) { //nolint:revive // function naming
	types := []SeriesType{
		SeriesLine,
		SeriesArea,
		SeriesBar,
		SeriesScatter,
	}

	expectedValues := []string{"line", "area", "bar", "scatter"}

	for i, seriesType := range types {
		if string(seriesType) != expectedValues[i] {
			t.Errorf("SeriesType %d: expected %s, got %s", i, expectedValues[i], string(seriesType))
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
