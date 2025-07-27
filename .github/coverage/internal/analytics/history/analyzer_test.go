package history

import (
	"context"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/config"
)

func TestNewHistoryAnalyzer(t *testing.T) {
	cfg := &config.Config{
		History: config.HistoryConfig{
			MaxEntries: 1000,
			RetentionDays: 30,
		},
	}
	
	analyzer := NewHistoryAnalyzer(cfg)
	if analyzer == nil {
		t.Fatal("NewHistoryAnalyzer returned nil")
	}
	if analyzer.config != cfg {
		t.Error("History analyzer config not set correctly")
	}
}

func TestAnalyzeTrends(t *testing.T) {
	cfg := &config.Config{
		History: config.HistoryConfig{
			MaxEntries: 1000,
			RetentionDays: 30,
		},
	}
	analyzer := NewHistoryAnalyzer(cfg)
	
	tests := []struct {
		name     string
		points   []CoverageHistoryPoint
		options  TrendAnalysisOptions
		expected string
	}{
		{
			name: "upward trend",
			points: []CoverageHistoryPoint{
				{Timestamp: time.Now().Add(-72 * time.Hour), Coverage: 75.0},
				{Timestamp: time.Now().Add(-48 * time.Hour), Coverage: 78.0},
				{Timestamp: time.Now().Add(-24 * time.Hour), Coverage: 82.0},
				{Timestamp: time.Now(), Coverage: 85.0},
			},
			options: TrendAnalysisOptions{
				WindowDays:      7,
				MinDataPoints:   3,
				ConfidenceLevel: 0.95,
			},
			expected: "upward",
		},
		{
			name: "downward trend",
			points: []CoverageHistoryPoint{
				{Timestamp: time.Now().Add(-72 * time.Hour), Coverage: 90.0},
				{Timestamp: time.Now().Add(-48 * time.Hour), Coverage: 87.0},
				{Timestamp: time.Now().Add(-24 * time.Hour), Coverage: 83.0},
				{Timestamp: time.Now(), Coverage: 80.0},
			},
			options: TrendAnalysisOptions{
				WindowDays:      7,
				MinDataPoints:   3,
				ConfidenceLevel: 0.95,
			},
			expected: "downward",
		},
		{
			name: "stable trend",
			points: []CoverageHistoryPoint{
				{Timestamp: time.Now().Add(-72 * time.Hour), Coverage: 85.0},
				{Timestamp: time.Now().Add(-48 * time.Hour), Coverage: 85.5},
				{Timestamp: time.Now().Add(-24 * time.Hour), Coverage: 84.5},
				{Timestamp: time.Now(), Coverage: 85.0},
			},
			options: TrendAnalysisOptions{
				WindowDays:      7,
				MinDataPoints:   3,
				ConfidenceLevel: 0.95,
			},
			expected: "stable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.AnalyzeTrends(context.Background(), tt.points, tt.options)
			if err != nil {
				t.Fatalf("AnalyzeTrends() error = %v", err)
			}
			
			if result.Direction != tt.expected {
				t.Errorf("AnalyzeTrends() direction = %v, expected %v", result.Direction, tt.expected)
			}
			
			if result.Confidence < 0 || result.Confidence > 1 {
				t.Errorf("AnalyzeTrends() confidence = %v, should be between 0 and 1", result.Confidence)
			}
		})
	}
}

func TestCalculateMovingAverages(t *testing.T) {
	analyzer := NewHistoryAnalyzer(&config.Config{})
	
	points := []CoverageHistoryPoint{
		{Coverage: 80.0}, {Coverage: 82.0}, {Coverage: 85.0},
		{Coverage: 83.0}, {Coverage: 87.0}, {Coverage: 90.0},
		{Coverage: 88.0}, {Coverage: 92.0}, {Coverage: 89.0},
	}
	
	tests := []struct {
		name       string
		windowSize int
		expectLen  int
	}{
		{"3-period moving average", 3, 7},
		{"5-period moving average", 5, 5},
		{"7-period moving average", 7, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.CalculateMovingAverages(points, tt.windowSize)
			if err != nil {
				t.Fatalf("CalculateMovingAverages() error = %v", err)
			}
			
			if len(result) != tt.expectLen {
				t.Errorf("CalculateMovingAverages() length = %v, expected %v", len(result), tt.expectLen)
			}
			
			// Verify moving averages are calculated correctly
			for i, avg := range result {
				if avg.Coverage < 0 || avg.Coverage > 100 {
					t.Errorf("Invalid moving average at index %d: %v", i, avg.Coverage)
				}
			}
		})
	}
}

func TestDetectAnomalies(t *testing.T) {
	analyzer := NewHistoryAnalyzer(&config.Config{})
	
	// Create data with clear anomalies
	points := []CoverageHistoryPoint{
		{Timestamp: time.Now().Add(-9 * time.Hour), Coverage: 85.0},
		{Timestamp: time.Now().Add(-8 * time.Hour), Coverage: 86.0},
		{Timestamp: time.Now().Add(-7 * time.Hour), Coverage: 84.0},
		{Timestamp: time.Now().Add(-6 * time.Hour), Coverage: 85.5},
		{Timestamp: time.Now().Add(-5 * time.Hour), Coverage: 50.0}, // Anomaly
		{Timestamp: time.Now().Add(-4 * time.Hour), Coverage: 86.0},
		{Timestamp: time.Now().Add(-3 * time.Hour), Coverage: 85.0},
		{Timestamp: time.Now().Add(-2 * time.Hour), Coverage: 87.0},
		{Timestamp: time.Now().Add(-1 * time.Hour), Coverage: 98.0}, // Anomaly
		{Timestamp: time.Now(), Coverage: 85.5},
	}
	
	options := AnomalyDetectionOptions{
		Method:               "statistical",
		SensitivityLevel:     2.0,
		MinHistorySize:       5,
		IncludePositiveSpikes: true,
		IncludeNegativeSpikes: true,
	}
	
	anomalies, err := analyzer.DetectAnomalies(context.Background(), points, options)
	if err != nil {
		t.Fatalf("DetectAnomalies() error = %v", err)
	}
	
	if len(anomalies) < 1 {
		t.Error("Expected to detect at least one anomaly")
	}
	
	// Verify anomaly structure
	for _, anomaly := range anomalies {
		if anomaly.Severity == "" {
			t.Error("Anomaly severity should not be empty")
		}
		if anomaly.Type == "" {
			t.Error("Anomaly type should not be empty")
		}
		if anomaly.Score < 0 {
			t.Error("Anomaly score should not be negative")
		}
	}
}

func TestCalculateSeasonalDecomposition(t *testing.T) {
	analyzer := NewHistoryAnalyzer(&config.Config{})
	
	// Generate seasonal data
	points := make([]CoverageHistoryPoint, 168) // One week of hourly data
	baseTime := time.Now().Add(-168 * time.Hour)
	for i := 0; i < 168; i++ {
		// Simulate daily pattern with noise
		hour := i % 24
		dailyPattern := 80.0 + 10.0 * float64(hour) / 24.0
		weeklyPattern := 5.0 * float64(i%7) / 7.0
		noise := float64(i%3 - 1) // Simple noise pattern
		
		points[i] = CoverageHistoryPoint{
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
			Coverage:  dailyPattern + weeklyPattern + noise,
		}
	}
	
	options := SeasonalDecompositionOptions{
		SeasonLength: 24, // Daily seasonality
		Method:       "additive",
	}
	
	decomposition, err := analyzer.CalculateSeasonalDecomposition(context.Background(), points, options)
	if err != nil {
		t.Fatalf("CalculateSeasonalDecomposition() error = %v", err)
	}
	
	if len(decomposition.Trend) == 0 {
		t.Error("Trend component should not be empty")
	}
	if len(decomposition.Seasonal) == 0 {
		t.Error("Seasonal component should not be empty")
	}
	if len(decomposition.Residual) == 0 {
		t.Error("Residual component should not be empty")
	}
}

func TestGenerateInsights(t *testing.T) {
	analyzer := NewHistoryAnalyzer(&config.Config{})
	
	points := []CoverageHistoryPoint{
		{Timestamp: time.Now().Add(-72 * time.Hour), Coverage: 75.0, CommitSHA: "abc123"},
		{Timestamp: time.Now().Add(-48 * time.Hour), Coverage: 78.0, CommitSHA: "def456"},
		{Timestamp: time.Now().Add(-24 * time.Hour), Coverage: 82.0, CommitSHA: "ghi789"},
		{Timestamp: time.Now(), Coverage: 85.0, CommitSHA: "jkl012"},
	}
	
	insights, err := analyzer.GenerateInsights(context.Background(), points)
	if err != nil {
		t.Fatalf("GenerateInsights() error = %v", err)
	}
	
	// Verify insight structure
	if insights.TrendAnalysis == nil {
		t.Error("TrendAnalysis should not be nil")
	}
	if insights.QualityMetrics == nil {
		t.Error("QualityMetrics should not be nil")
	}
	if len(insights.Recommendations) == 0 {
		t.Error("Should provide at least one recommendation")
	}
	
	// Verify specific insights
	if insights.Summary == "" {
		t.Error("Summary should not be empty")
	}
	if insights.TrendAnalysis.Direction == "" {
		t.Error("Trend direction should not be empty")
	}
}

func TestPerformanceMetrics(t *testing.T) {
	analyzer := NewHistoryAnalyzer(&config.Config{})
	
	points := []CoverageHistoryPoint{
		{Coverage: 85.0, Files: []FileInfo{{Name: "file1.go", Coverage: 90.0}}},
		{Coverage: 87.0, Files: []FileInfo{{Name: "file1.go", Coverage: 88.0}}},
		{Coverage: 89.0, Files: []FileInfo{{Name: "file1.go", Coverage: 92.0}}},
	}
	
	metrics, err := analyzer.CalculatePerformanceMetrics(context.Background(), points)
	if err != nil {
		t.Fatalf("CalculatePerformanceMetrics() error = %v", err)
	}
	
	if metrics.AverageCoverage <= 0 {
		t.Error("Average coverage should be positive")
	}
	if metrics.StandardDeviation < 0 {
		t.Error("Standard deviation should not be negative")
	}
	if metrics.CoverageVelocity == 0 {
		t.Error("Coverage velocity should be calculated")
	}
}

func TestTimeSeriesForecasting(t *testing.T) {
	analyzer := NewHistoryAnalyzer(&config.Config{})
	
	// Generate trending data
	points := make([]CoverageHistoryPoint, 30)
	baseTime := time.Now().Add(-30 * 24 * time.Hour)
	for i := 0; i < 30; i++ {
		points[i] = CoverageHistoryPoint{
			Timestamp: baseTime.Add(time.Duration(i) * 24 * time.Hour),
			Coverage:  75.0 + float64(i)*0.5, // Linear growth
		}
	}
	
	options := ForecastOptions{
		HorizonDays:     7,
		ConfidenceLevel: 0.95,
		Method:          "linear",
	}
	
	forecast, err := analyzer.GenerateTimeSeriesForecast(context.Background(), points, options)
	if err != nil {
		t.Fatalf("GenerateTimeSeriesForecast() error = %v", err)
	}
	
	if len(forecast.Predictions) != options.HorizonDays {
		t.Errorf("Expected %d predictions, got %d", options.HorizonDays, len(forecast.Predictions))
	}
	
	if forecast.Confidence < 0 || forecast.Confidence > 1 {
		t.Errorf("Forecast confidence should be between 0 and 1, got %v", forecast.Confidence)
	}
	
	// Verify predictions are reasonable
	for i, pred := range forecast.Predictions {
		if pred.Coverage < 0 || pred.Coverage > 100 {
			t.Errorf("Prediction %d has invalid coverage: %v", i, pred.Coverage)
		}
	}
}

func BenchmarkAnalyzeTrends(b *testing.B) {
	cfg := &config.Config{
		History: config.HistoryConfig{
			MaxEntries: 1000,
			RetentionDays: 30,
		},
	}
	analyzer := NewHistoryAnalyzer(cfg)
	
	// Generate large dataset
	points := make([]CoverageHistoryPoint, 1000)
	baseTime := time.Now().Add(-1000 * time.Hour)
	for i := 0; i < 1000; i++ {
		points[i] = CoverageHistoryPoint{
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
			Coverage:  75.0 + float64(i%100)*0.2,
		}
	}
	
	options := TrendAnalysisOptions{
		WindowDays:      7,
		MinDataPoints:   10,
		ConfidenceLevel: 0.95,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeTrends(context.Background(), points, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDetectAnomalies(b *testing.B) {
	analyzer := NewHistoryAnalyzer(&config.Config{})
	
	// Generate dataset with some anomalies
	points := make([]CoverageHistoryPoint, 500)
	baseTime := time.Now().Add(-500 * time.Hour)
	for i := 0; i < 500; i++ {
		coverage := 85.0
		if i%50 == 0 { // Add anomalies every 50 points
			coverage = 50.0
		}
		points[i] = CoverageHistoryPoint{
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
			Coverage:  coverage,
		}
	}
	
	options := AnomalyDetectionOptions{
		Method:                "statistical",
		SensitivityLevel:      2.0,
		MinHistorySize:        10,
		IncludePositiveSpikes: true,
		IncludeNegativeSpikes: true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.DetectAnomalies(context.Background(), points, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}