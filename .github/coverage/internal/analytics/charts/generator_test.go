package charts

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/config"
)

func TestNewChartGenerator(t *testing.T) {
	generator := NewChartGenerator()
	if generator == nil {
		t.Fatal("NewChartGenerator returned nil")
	}
	if generator.config == nil {
		t.Error("Chart generator config should not be nil")
	}
}

func TestGenerateCoverageChart(t *testing.T) {
	generator := NewChartGenerator()
	
	tests := []struct {
		name   string
		data   []CoveragePoint
		config ChartConfig
		expect bool
	}{
		{
			name: "empty data",
			data: []CoveragePoint{},
			config: ChartConfig{
				Width:  800,
				Height: 400,
				Title:  "Empty Chart",
			},
			expect: true,
		},
		{
			name: "single point",
			data: []CoveragePoint{
				{Timestamp: time.Now(), Coverage: 85.5},
			},
			config: ChartConfig{
				Width:  800,
				Height: 400,
				Title:  "Single Point",
			},
			expect: true,
		},
		{
			name: "multiple points",
			data: []CoveragePoint{
				{Timestamp: time.Now().Add(-24 * time.Hour), Coverage: 80.0},
				{Timestamp: time.Now().Add(-12 * time.Hour), Coverage: 82.5},
				{Timestamp: time.Now(), Coverage: 85.0},
			},
			config: ChartConfig{
				Width:  800,
				Height: 400,
				Title:  "Trending Up",
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg, err := generator.GenerateCoverageChart(tt.data, tt.config)
			if (err == nil) != tt.expect {
				t.Errorf("GenerateCoverageChart() error = %v, expected success = %v", err, tt.expect)
				return
			}
			if tt.expect && !strings.Contains(svg, "<svg") {
				t.Error("Generated SVG should contain <svg tag")
			}
			if tt.expect && !strings.Contains(svg, tt.config.Title) {
				t.Error("Generated SVG should contain chart title")
			}
		})
	}
}

func TestGenerateTrendChart(t *testing.T) {
	generator := NewChartGenerator()
	
	data := []TrendPoint{
		{
			Period:    "2024-01",
			Trend:     "upward",
			Magnitude: 2.5,
			Points: []CoveragePoint{
				{Timestamp: time.Now().Add(-30 * 24 * time.Hour), Coverage: 75.0},
				{Timestamp: time.Now().Add(-15 * 24 * time.Hour), Coverage: 77.5},
				{Timestamp: time.Now(), Coverage: 80.0},
			},
		},
	}
	
	config := ChartConfig{
		Width:        1000,
		Height:       500,
		Title:        "Coverage Trends",
		ShowGrid:     true,
		ShowLegend:   true,
		ColorScheme:  "default",
	}
	
	svg, err := generator.GenerateTrendChart(data, config)
	if err != nil {
		t.Fatalf("GenerateTrendChart() error = %v", err)
	}
	
	if !strings.Contains(svg, "<svg") {
		t.Error("Generated SVG should contain <svg tag")
	}
	if !strings.Contains(svg, config.Title) {
		t.Error("Generated SVG should contain chart title")
	}
	if !strings.Contains(svg, "upward") {
		t.Error("Generated SVG should contain trend information")
	}
}

func TestGenerateComparisonChart(t *testing.T) {
	generator := NewChartGenerator()
	
	data := ComparisonData{
		Labels: []string{"Feature A", "Feature B", "Feature C"},
		Series: []ComparisonSeries{
			{
				Name:   "Current",
				Values: []float64{85.0, 90.0, 78.0},
				Color:  "#4CAF50",
			},
			{
				Name:   "Previous",
				Values: []float64{80.0, 88.0, 75.0},
				Color:  "#FF9800",
			},
		},
	}
	
	config := ChartConfig{
		Width:       800,
		Height:      400,
		Title:       "Coverage Comparison",
		ChartType:   "bar",
		ShowGrid:    true,
		ShowLegend:  true,
	}
	
	svg, err := generator.GenerateComparisonChart(data, config)
	if err != nil {
		t.Fatalf("GenerateComparisonChart() error = %v", err)
	}
	
	// Validate SVG structure
	expectedElements := []string{
		"<svg",
		config.Title,
		"Feature A",
		"Feature B", 
		"Feature C",
		"Current",
		"Previous",
	}
	
	for _, element := range expectedElements {
		if !strings.Contains(svg, element) {
			t.Errorf("Generated SVG should contain %s", element)
		}
	}
}

func TestGenerateHeatmapChart(t *testing.T) {
	generator := NewChartGenerator()
	
	data := HeatmapData{
		XLabels: []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
		YLabels: []string{"Week 1", "Week 2", "Week 3"},
		Values: [][]float64{
			{85.0, 87.5, 90.0, 88.0, 92.0},
			{82.0, 85.0, 88.5, 90.5, 89.0},
			{78.0, 80.5, 85.0, 87.0, 91.0},
		},
	}
	
	config := ChartConfig{
		Width:      1000,
		Height:     300,
		Title:      "Coverage Heatmap",
		ChartType:  "heatmap",
		ShowGrid:   false,
		ColorScheme: "viridis",
	}
	
	svg, err := generator.GenerateHeatmapChart(data, config)
	if err != nil {
		t.Fatalf("GenerateHeatmapChart() error = %v", err)
	}
	
	if !strings.Contains(svg, "<svg") {
		t.Error("Generated SVG should contain <svg tag")
	}
	if !strings.Contains(svg, config.Title) {
		t.Error("Generated SVG should contain chart title")
	}
}

func TestValidateChartConfig(t *testing.T) {
	tests := []struct {
		name   string
		config ChartConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: ChartConfig{
				Width:  800,
				Height: 400,
				Title:  "Test Chart",
			},
			valid: true,
		},
		{
			name: "zero width",
			config: ChartConfig{
				Width:  0,
				Height: 400,
				Title:  "Test Chart",
			},
			valid: false,
		},
		{
			name: "zero height",
			config: ChartConfig{
				Width:  800,
				Height: 0,
				Title:  "Test Chart",
			},
			valid: false,
		},
		{
			name: "empty title",
			config: ChartConfig{
				Width:  800,
				Height: 400,
				Title:  "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChartConfig(tt.config)
			if (err == nil) != tt.valid {
				t.Errorf("validateChartConfig() error = %v, expected valid = %v", err, tt.valid)
			}
		})
	}
}

func TestChartStyles(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		expect bool
	}{
		{"default scheme", "default", true},
		{"viridis scheme", "viridis", true},
		{"plasma scheme", "plasma", true},
		{"invalid scheme", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			styles := getColorScheme(tt.scheme)
			if (len(styles) > 0) != tt.expect {
				t.Errorf("getColorScheme(%s) returned %d colors, expected valid = %v", 
					tt.scheme, len(styles), tt.expect)
			}
		})
	}
}

func TestGenerateProgressChart(t *testing.T) {
	generator := NewChartGenerator()
	
	progress := ProgressData{
		Current:    85.5,
		Target:     90.0,
		Threshold:  80.0,
		Segments: []ProgressSegment{
			{Label: "Unit Tests", Value: 90.0, Color: "#4CAF50"},
			{Label: "Integration Tests", Value: 85.0, Color: "#FF9800"},
			{Label: "E2E Tests", Value: 75.0, Color: "#F44336"},
		},
	}
	
	config := ChartConfig{
		Width:     400,
		Height:    400,
		Title:     "Coverage Progress",
		ChartType: "donut",
	}
	
	svg, err := generator.GenerateProgressChart(progress, config)
	if err != nil {
		t.Fatalf("GenerateProgressChart() error = %v", err)
	}
	
	expectedElements := []string{
		"<svg",
		"Coverage Progress",
		"Unit Tests",
		"Integration Tests",
		"E2E Tests",
		"85.5", // Current value
		"90.0", // Target value
	}
	
	for _, element := range expectedElements {
		if !strings.Contains(svg, element) {
			t.Errorf("Generated SVG should contain %s", element)
		}
	}
}

func BenchmarkGenerateCoverageChart(b *testing.B) {
	generator := NewChartGenerator()
	
	// Generate test data
	data := make([]CoveragePoint, 100)
	baseTime := time.Now().Add(-100 * 24 * time.Hour)
	for i := 0; i < 100; i++ {
		data[i] = CoveragePoint{
			Timestamp: baseTime.Add(time.Duration(i) * 24 * time.Hour),
			Coverage:  75.0 + float64(i)*0.1,
		}
	}
	
	config := ChartConfig{
		Width:  800,
		Height: 400,
		Title:  "Benchmark Chart",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.GenerateCoverageChart(data, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateComparisonChart(b *testing.B) {
	generator := NewChartGenerator()
	
	data := ComparisonData{
		Labels: []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"},
		Series: []ComparisonSeries{
			{
				Name:   "Series 1",
				Values: []float64{85, 90, 78, 82, 88, 92, 79, 86, 91, 87},
				Color:  "#4CAF50",
			},
			{
				Name:   "Series 2",
				Values: []float64{80, 88, 75, 85, 90, 89, 83, 81, 87, 84},
				Color:  "#FF9800",
			},
		},
	}
	
	config := ChartConfig{
		Width:     800,
		Height:    400,
		Title:     "Benchmark Comparison",
		ChartType: "bar",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.GenerateComparisonChart(data, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}