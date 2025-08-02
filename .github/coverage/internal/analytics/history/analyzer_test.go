package history

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// AnalyzerTestSuite provides test suite for trend analyzer
type AnalyzerTestSuite struct {
	suite.Suite

	analyzer *TrendAnalyzer
	config   *AnalyzerConfig
}

// SetupTest creates analyzer with default config for each test
func (suite *AnalyzerTestSuite) SetupTest() {
	suite.config = &AnalyzerConfig{
		ShortTermDays:       7,
		MediumTermDays:      30,
		LongTermDays:        90,
		MovingAvgWindow:     7,
		ExponentialAlpha:    0.3,
		SignificantChange:   1.0,
		VolatilityThreshold: 5.0,
		TrendConfidence:     0.7,
		PredictionDays:      14,
		SeasonalAdjustment:  true,
		OutlierDetection:    true,
		MinDataPoints:       5,
		MaxGapDays:          7,
	}
	suite.analyzer = NewTrendAnalyzer(suite.config)
}

// TestNewTrendAnalyzerSuccess tests successful analyzer creation
func (suite *AnalyzerTestSuite) TestNewTrendAnalyzerSuccess() {
	analyzer := NewTrendAnalyzer(suite.config)

	suite.Require().NotNil(analyzer)
	suite.Equal(suite.config, analyzer.config)
	suite.NotNil(analyzer.data)
	suite.Empty(analyzer.data)
}

// TestNewTrendAnalyzerNilConfig tests analyzer creation with nil config
func (suite *AnalyzerTestSuite) TestNewTrendAnalyzerNilConfig() {
	analyzer := NewTrendAnalyzer(nil)

	suite.Require().NotNil(analyzer)
	suite.Require().NotNil(analyzer.config)

	// Should have default values
	suite.Equal(7, analyzer.config.ShortTermDays)
	suite.Equal(30, analyzer.config.MediumTermDays)
	suite.Equal(90, analyzer.config.LongTermDays)
	suite.InEpsilon(0.3, analyzer.config.ExponentialAlpha, 0.001)
	suite.Equal(5, analyzer.config.MinDataPoints)
}

// TestLoadCustomDataSuccess tests loading custom data points
func (suite *AnalyzerTestSuite) TestLoadCustomDataSuccess() {
	dataPoints := suite.createSampleDataPoints()

	suite.analyzer.LoadCustomData(dataPoints)

	suite.Len(suite.analyzer.data, len(dataPoints))

	// Verify data is sorted by timestamp
	for i := 1; i < len(suite.analyzer.data); i++ {
		suite.True(suite.analyzer.data[i-1].Timestamp.Before(suite.analyzer.data[i].Timestamp) ||
			suite.analyzer.data[i-1].Timestamp.Equal(suite.analyzer.data[i].Timestamp),
			"Data should be sorted by timestamp")
	}
}

// TestLoadCustomDataEmpty tests loading empty data
func (suite *AnalyzerTestSuite) TestLoadCustomDataEmpty() {
	suite.analyzer.LoadCustomData([]AnalysisDataPoint{})

	suite.Empty(suite.analyzer.data)
}

// TestLoadCustomDataUnsorted tests loading unsorted data
func (suite *AnalyzerTestSuite) TestLoadCustomDataUnsorted() {
	dataPoints := []AnalysisDataPoint{
		{
			Timestamp: time.Now().Add(-2 * time.Hour),
			Coverage:  75.0,
			Branch:    "master",
		},
		{
			Timestamp: time.Now().Add(-4 * time.Hour),
			Coverage:  70.0,
			Branch:    "master",
		},
		{
			Timestamp: time.Now().Add(-1 * time.Hour),
			Coverage:  80.0,
			Branch:    "master",
		},
	}

	suite.analyzer.LoadCustomData(dataPoints)

	// Should be sorted after loading
	for i := 1; i < len(suite.analyzer.data); i++ {
		suite.True(suite.analyzer.data[i-1].Timestamp.Before(suite.analyzer.data[i].Timestamp),
			"Data should be sorted by timestamp after loading")
	}
}

// TestAnalyzeTrendsSuccess tests successful trend analysis
func (suite *AnalyzerTestSuite) TestAnalyzeTrendsSuccess() {
	ctx := context.Background()
	dataPoints := suite.createSampleDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)

	report, err := suite.analyzer.AnalyzeTrends(ctx)
	suite.Require().NoError(err)
	suite.Require().NotNil(report)

	// Verify report structure
	suite.NotZero(report.GeneratedAt)
	suite.Equal(len(dataPoints), report.DataPointCount)
	suite.NotZero(report.AnalysisWindow)

	// Verify summary
	suite.NotZero(report.Summary.CurrentCoverage)
	suite.NotEmpty(report.Summary.Direction)
	suite.NotEmpty(report.Summary.Magnitude)

	// Verify trend analyses
	suite.NotEmpty(report.ShortTermTrend.Period)
	suite.NotEmpty(report.MediumTermTrend.Period)
	suite.NotEmpty(report.LongTermTrend.Period)

	// Verify volatility analysis
	suite.GreaterOrEqual(report.Volatility.StandardDeviation, 0.0)
	suite.NotEmpty(report.Volatility.VolatilityLevel)

	// Verify predictions
	suite.NotEmpty(report.Predictions)
	suite.Len(report.Predictions, suite.config.PredictionDays)

	// Verify quality metrics
	suite.GreaterOrEqual(report.QualityMetrics.QualityScore, 0.0)
	suite.LessOrEqual(report.QualityMetrics.QualityScore, 100.0)

	// Verify insights and recommendations are properly initialized
	suite.Require().NotNil(report.Insights)
	suite.Require().NotNil(report.Recommendations)
}

// TestAnalyzeTrendsInsufficientData tests analysis with insufficient data
func (suite *AnalyzerTestSuite) TestAnalyzeTrendsInsufficientData() {
	ctx := context.Background()

	// Load insufficient data points
	dataPoints := []AnalysisDataPoint{
		{
			Timestamp: time.Now(),
			Coverage:  75.0,
			Branch:    "master",
		},
	}
	suite.analyzer.LoadCustomData(dataPoints)

	_, err := suite.analyzer.AnalyzeTrends(ctx)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "insufficient data points")
}

// TestAnalyzeTrendsNoData tests analysis with no data
func (suite *AnalyzerTestSuite) TestAnalyzeTrendsNoData() {
	ctx := context.Background()

	_, err := suite.analyzer.AnalyzeTrends(ctx)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "insufficient data points")
}

// TestDetectOutliersSuccess tests outlier detection
func (suite *AnalyzerTestSuite) TestDetectOutliersSuccess() {
	// Create data with obvious outliers
	dataPoints := []AnalysisDataPoint{
		{Timestamp: time.Now().Add(-10 * time.Hour), Coverage: 75.0},
		{Timestamp: time.Now().Add(-9 * time.Hour), Coverage: 76.0},
		{Timestamp: time.Now().Add(-8 * time.Hour), Coverage: 74.0},
		{Timestamp: time.Now().Add(-7 * time.Hour), Coverage: 75.5},
		{Timestamp: time.Now().Add(-6 * time.Hour), Coverage: 95.0}, // Outlier
		{Timestamp: time.Now().Add(-5 * time.Hour), Coverage: 74.5},
		{Timestamp: time.Now().Add(-4 * time.Hour), Coverage: 76.5},
		{Timestamp: time.Now().Add(-3 * time.Hour), Coverage: 30.0}, // Outlier
		{Timestamp: time.Now().Add(-2 * time.Hour), Coverage: 75.8},
		{Timestamp: time.Now().Add(-1 * time.Hour), Coverage: 74.2},
	}

	suite.analyzer.LoadCustomData(dataPoints)
	suite.analyzer.detectOutliers()

	// Count outliers
	outlierCount := 0
	for _, point := range suite.analyzer.data {
		if point.IsOutlier {
			outlierCount++
		}
	}

	suite.Positive(outlierCount, "Should detect some outliers")
	suite.Less(outlierCount, len(dataPoints), "Should not mark all points as outliers")
}

// TestDetectOutliersNoOutliers tests outlier detection with no outliers
func (suite *AnalyzerTestSuite) TestDetectOutliersNoOutliers() {
	// Create data with no outliers (all similar values)
	dataPoints := []AnalysisDataPoint{
		{Timestamp: time.Now().Add(-5 * time.Hour), Coverage: 75.0},
		{Timestamp: time.Now().Add(-4 * time.Hour), Coverage: 75.1},
		{Timestamp: time.Now().Add(-3 * time.Hour), Coverage: 74.9},
		{Timestamp: time.Now().Add(-2 * time.Hour), Coverage: 75.2},
		{Timestamp: time.Now().Add(-1 * time.Hour), Coverage: 74.8},
	}

	suite.analyzer.LoadCustomData(dataPoints)
	suite.analyzer.detectOutliers()

	// Should detect no outliers
	outlierCount := 0
	for _, point := range suite.analyzer.data {
		if point.IsOutlier {
			outlierCount++
		}
	}

	suite.Equal(0, outlierCount, "Should detect no outliers in similar data")
}

// TestApplySmoothing tests exponential smoothing
func (suite *AnalyzerTestSuite) TestApplySmoothing() {
	dataPoints := suite.createSampleDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)
	suite.analyzer.applySmoothing()

	// Verify smoothed values are set
	for i, point := range suite.analyzer.data {
		suite.NotZero(point.Smoothed, "Smoothed value should be set for point %d", i)

		if i == 0 {
			// First point should equal original coverage
			suite.InEpsilon(point.Coverage, point.Smoothed, 0.001, "First smoothed value should equal coverage")
		} else {
			// Smoothed values should be within reasonable range
			suite.GreaterOrEqual(point.Smoothed, 0.0)
			suite.LessOrEqual(point.Smoothed, 100.0)
		}
	}
}

// TestGenerateSummarySuccess tests summary generation
func (suite *AnalyzerTestSuite) TestGenerateSummarySuccess() {
	dataPoints := suite.createSampleDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)

	summary := suite.analyzer.generateSummary()

	suite.Greater(summary.CurrentCoverage, 0.0)
	suite.Greater(summary.PreviousCoverage, 0.0)
	suite.NotEmpty(summary.Direction)
	suite.NotEmpty(summary.Magnitude)
	suite.GreaterOrEqual(summary.Confidence, 0.0)
	suite.LessOrEqual(summary.Confidence, 1.0)
	suite.NotEmpty(summary.QualityGrade)
}

// TestGenerateSummaryEmptyData tests summary generation with empty data
func (suite *AnalyzerTestSuite) TestGenerateSummaryEmptyData() {
	summary := suite.analyzer.generateSummary()

	suite.InDelta(0.0, summary.CurrentCoverage, 0.001)
	suite.InDelta(0.0, summary.PreviousCoverage, 0.001)
	suite.InDelta(0.0, summary.Change, 0.001)
	suite.InDelta(0.0, summary.ChangePercent, 0.001)
}

// TestAnalyzePeriodTrendSuccess tests period trend analysis
func (suite *AnalyzerTestSuite) TestAnalyzePeriodTrendSuccess() {
	dataPoints := suite.createSampleDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)
	suite.analyzer.applySmoothing()

	trendAnalysis := suite.analyzer.analyzePeriodTrend(7, "test-period")

	suite.Equal("test-period", trendAnalysis.Period)
	suite.NotEmpty(trendAnalysis.Direction)
	suite.NotZero(trendAnalysis.StartDate)
	suite.NotZero(trendAnalysis.EndDate)
	suite.GreaterOrEqual(trendAnalysis.RSquared, 0.0)
	suite.LessOrEqual(trendAnalysis.RSquared, 1.0)
	suite.GreaterOrEqual(trendAnalysis.Confidence, 0.0)
	suite.LessOrEqual(trendAnalysis.Confidence, 1.0)
}

// TestAnalyzePeriodTrendInsufficientData tests period trend with insufficient data
func (suite *AnalyzerTestSuite) TestAnalyzePeriodTrendInsufficientData() {
	// Load only one data point
	dataPoints := []AnalysisDataPoint{
		{Timestamp: time.Now(), Coverage: 75.0},
	}
	suite.analyzer.LoadCustomData(dataPoints)

	trendAnalysis := suite.analyzer.analyzePeriodTrend(7, "test-period")

	suite.Equal("test-period", trendAnalysis.Period)
	// Other fields should be zero/empty for insufficient data
}

// TestCalculateLinearRegression tests linear regression calculation
func (suite *AnalyzerTestSuite) TestCalculateLinearRegression() {
	// Create data with known linear trend
	dataPoints := []AnalysisDataPoint{
		{Timestamp: time.Now().Add(-5 * time.Hour), Coverage: 70.0, Smoothed: 70.0},
		{Timestamp: time.Now().Add(-4 * time.Hour), Coverage: 72.0, Smoothed: 72.0},
		{Timestamp: time.Now().Add(-3 * time.Hour), Coverage: 74.0, Smoothed: 74.0},
		{Timestamp: time.Now().Add(-2 * time.Hour), Coverage: 76.0, Smoothed: 76.0},
		{Timestamp: time.Now().Add(-1 * time.Hour), Coverage: 78.0, Smoothed: 78.0},
	}

	slope, rSquared := suite.analyzer.calculateLinearRegression(dataPoints)

	suite.Greater(slope, 0.0, "Should detect positive slope")
	suite.Greater(rSquared, 0.8, "Should have high R-squared for linear data")
	suite.LessOrEqual(rSquared, 1.0, "R-squared should not exceed 1.0")
}

// TestCalculateLinearRegressionInsufficientData tests regression with insufficient data
func (suite *AnalyzerTestSuite) TestCalculateLinearRegressionInsufficientData() {
	dataPoints := []AnalysisDataPoint{
		{Coverage: 75.0, Smoothed: 75.0},
	}

	slope, rSquared := suite.analyzer.calculateLinearRegression(dataPoints)

	suite.InDelta(0.0, slope, 0.001)
	suite.InDelta(0.0, rSquared, 0.001)
}

// TestAnalyzeVolatilitySuccess tests volatility analysis
func (suite *AnalyzerTestSuite) TestAnalyzeVolatilitySuccess() {
	dataPoints := suite.createVolatileDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)

	volatility := suite.analyzer.analyzeVolatility()

	suite.GreaterOrEqual(volatility.StandardDeviation, 0.0)
	suite.GreaterOrEqual(volatility.Variance, 0.0)
	suite.GreaterOrEqual(volatility.CoefficientVariation, 0.0)
	suite.NotEmpty(volatility.VolatilityLevel)
	suite.GreaterOrEqual(volatility.LargestFluctuation, 0.0)
	suite.GreaterOrEqual(volatility.AverageFluctuation, 0.0)
	suite.GreaterOrEqual(volatility.StabilityScore, 0.0)
	suite.LessOrEqual(volatility.StabilityScore, 100.0)
}

// TestAnalyzeVolatilityStableData tests volatility analysis with stable data
func (suite *AnalyzerTestSuite) TestAnalyzeVolatilityStableData() {
	// Create very stable data
	dataPoints := []AnalysisDataPoint{
		{Timestamp: time.Now().Add(-5 * time.Hour), Coverage: 75.0},
		{Timestamp: time.Now().Add(-4 * time.Hour), Coverage: 75.0},
		{Timestamp: time.Now().Add(-3 * time.Hour), Coverage: 75.0},
		{Timestamp: time.Now().Add(-2 * time.Hour), Coverage: 75.0},
		{Timestamp: time.Now().Add(-1 * time.Hour), Coverage: 75.0},
	}
	suite.analyzer.LoadCustomData(dataPoints)

	volatility := suite.analyzer.analyzeVolatility()

	suite.InDelta(0.0, volatility.StandardDeviation, 0.001)
	suite.InDelta(0.0, volatility.Variance, 0.001)
	suite.Equal(VolatilityLow, volatility.VolatilityLevel)
	suite.InDelta(100.0, volatility.StabilityScore, 0.001)
}

// TestGeneratePredictionsSuccess tests prediction generation
func (suite *AnalyzerTestSuite) TestGeneratePredictionsSuccess() {
	dataPoints := suite.createSampleDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)
	suite.analyzer.applySmoothing()

	predictions, err := suite.analyzer.generatePredictions()
	suite.Require().NoError(err)
	suite.Len(predictions, suite.config.PredictionDays)

	for i, prediction := range predictions {
		suite.NotZero(prediction.Date, "Prediction %d should have date", i)
		suite.GreaterOrEqual(prediction.PredictedCoverage, 0.0)
		suite.LessOrEqual(prediction.PredictedCoverage, 100.0)
		suite.GreaterOrEqual(prediction.ConfidenceInterval.Lower, 0.0)
		suite.LessOrEqual(prediction.ConfidenceInterval.Upper, 100.0)
		suite.GreaterOrEqual(prediction.Reliability, 0.0)
		suite.LessOrEqual(prediction.Reliability, 1.0)
		suite.NotEmpty(prediction.Methodology)

		// Confidence interval should be valid
		suite.LessOrEqual(prediction.ConfidenceInterval.Lower, prediction.PredictedCoverage)
		suite.GreaterOrEqual(prediction.ConfidenceInterval.Upper, prediction.PredictedCoverage)
	}
}

// TestGeneratePredictionsInsufficientData tests predictions with insufficient data
func (suite *AnalyzerTestSuite) TestGeneratePredictionsInsufficientData() {
	dataPoints := []AnalysisDataPoint{
		{Timestamp: time.Now(), Coverage: 75.0},
		{Timestamp: time.Now().Add(-1 * time.Hour), Coverage: 74.0},
	}
	suite.analyzer.LoadCustomData(dataPoints)

	_, err := suite.analyzer.generatePredictions()
	suite.Require().Error(err)
	suite.Contains(err.Error(), "insufficient data for predictions")
}

// TestCalculateQualityMetrics tests quality metrics calculation
func (suite *AnalyzerTestSuite) TestCalculateQualityMetrics() {
	dataPoints := suite.createSampleDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)
	suite.analyzer.detectOutliers()

	metrics := suite.analyzer.calculateQualityMetrics()

	suite.GreaterOrEqual(metrics.DataCompleteness, 0.0)
	suite.LessOrEqual(metrics.DataCompleteness, 100.0)
	suite.GreaterOrEqual(metrics.DataConsistency, 0.0)
	suite.LessOrEqual(metrics.DataConsistency, 100.0)
	suite.GreaterOrEqual(metrics.OutlierCount, 0)
	suite.GreaterOrEqual(metrics.QualityScore, 0.0)
	suite.LessOrEqual(metrics.QualityScore, 100.0)
	suite.NotEmpty(metrics.ReliabilityGrade)

	// Grade should be valid
	validGrades := []string{"A", "B", "C", "D", "F"}
	suite.Contains(validGrades, metrics.ReliabilityGrade)
}

// TestDetermineDirection tests direction determination
func (suite *AnalyzerTestSuite) TestDetermineDirection() {
	testCases := []struct {
		name        string
		change      float64
		expectedDir TrendDirection
	}{
		{
			name:        "Significant positive change",
			change:      5.0,
			expectedDir: TrendUp,
		},
		{
			name:        "Significant negative change",
			change:      -5.0,
			expectedDir: TrendDown,
		},
		{
			name:        "Small positive change",
			change:      0.5,
			expectedDir: TrendStable,
		},
		{
			name:        "Small negative change",
			change:      -0.5,
			expectedDir: TrendStable,
		},
		{
			name:        "No change",
			change:      0.0,
			expectedDir: TrendStable,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			direction := suite.analyzer.determineDirection(tc.change)
			suite.Equal(tc.expectedDir, direction)
		})
	}
}

// TestDetermineMagnitude tests magnitude determination
func (suite *AnalyzerTestSuite) TestDetermineMagnitude() {
	testCases := []struct {
		name        string
		change      float64
		expectedMag TrendMagnitude
	}{
		{
			name:        "Significant change",
			change:      7.0,
			expectedMag: MagnitudeSignificant,
		},
		{
			name:        "Moderate change",
			change:      3.0,
			expectedMag: MagnitudeModerate,
		},
		{
			name:        "Minor change",
			change:      1.0,
			expectedMag: MagnitudeMinor,
		},
		{
			name:        "Negligible change",
			change:      0.1,
			expectedMag: MagnitudeNegligible,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			magnitude := suite.analyzer.determineMagnitude(tc.change)
			suite.Equal(tc.expectedMag, magnitude)
		})
	}
}

// TestCalculateQualityGrade tests quality grade calculation
func (suite *AnalyzerTestSuite) TestCalculateQualityGrade() {
	testCases := []struct {
		name          string
		coverage      float64
		expectedGrade string
	}{
		{
			name:          "Excellent coverage",
			coverage:      97.0,
			expectedGrade: "A+",
		},
		{
			name:          "Very good coverage",
			coverage:      92.0,
			expectedGrade: "A",
		},
		{
			name:          "Good coverage",
			coverage:      82.0,
			expectedGrade: "B",
		},
		{
			name:          "Acceptable coverage",
			coverage:      72.0,
			expectedGrade: "C",
		},
		{
			name:          "Poor coverage",
			coverage:      50.0,
			expectedGrade: "F",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			grade := suite.analyzer.calculateQualityGrade(tc.coverage)
			suite.Equal(tc.expectedGrade, grade)
		})
	}
}

// TestGenerateInsights tests insight generation
func (suite *AnalyzerTestSuite) TestGenerateInsights() {
	dataPoints := suite.createSampleDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)

	ctx := context.Background()
	report, err := suite.analyzer.AnalyzeTrends(ctx)
	suite.Require().NoError(err)

	insights := suite.analyzer.generateInsights(report)

	// Should generate some insights
	suite.NotNil(insights)

	// Validate insight structure
	for _, insight := range insights {
		suite.NotEmpty(insight.Type)
		suite.NotEmpty(insight.Title)
		suite.NotEmpty(insight.Description)
		suite.NotEmpty(insight.Severity)
		suite.GreaterOrEqual(insight.Confidence, 0.0)
		suite.LessOrEqual(insight.Confidence, 1.0)
	}
}

// TestGenerateRecommendations tests recommendation generation
func (suite *AnalyzerTestSuite) TestGenerateRecommendations() {
	dataPoints := suite.createSampleDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)

	ctx := context.Background()
	report, err := suite.analyzer.AnalyzeTrends(ctx)
	suite.Require().NoError(err)

	// Modify report to trigger specific recommendations
	report.Summary.CurrentCoverage = 75.0 // Below 80% threshold
	report.Volatility.VolatilityLevel = VolatilityHigh
	report.QualityMetrics.QualityScore = 65.0 // Below 70% threshold

	recommendations := suite.analyzer.generateRecommendations(report)

	// Should generate recommendations for low coverage, high volatility, and poor quality
	suite.NotEmpty(recommendations)

	// Validate recommendation structure
	for _, rec := range recommendations {
		suite.NotEmpty(rec.Type)
		suite.NotEmpty(rec.Priority)
		suite.NotEmpty(rec.Title)
		suite.NotEmpty(rec.Description)
		suite.NotEmpty(rec.Actions)
		suite.GreaterOrEqual(rec.ExpectedImpact, 0.0)
		suite.NotEmpty(rec.Timeline)
	}
}

// TestExportToJSON tests JSON export functionality
func (suite *AnalyzerTestSuite) TestExportToJSON() {
	dataPoints := suite.createSampleDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)

	jsonData, err := suite.analyzer.ExportToJSON()
	suite.Require().NoError(err)
	suite.NotEmpty(jsonData)

	// Should be valid JSON
	suite.Contains(string(jsonData), "{")
	suite.Contains(string(jsonData), "}")
}

// TestGetDataForPeriod tests period data retrieval
func (suite *AnalyzerTestSuite) TestGetDataForPeriod() {
	dataPoints := suite.createSampleDataPoints()
	suite.analyzer.LoadCustomData(dataPoints)

	now := time.Now()
	start := now.Add(-5 * time.Hour)
	end := now.Add(-1 * time.Hour)

	periodData := suite.analyzer.GetDataForPeriod(start, end)

	// All returned data should be within the period
	for _, point := range periodData {
		suite.True(point.Timestamp.After(start))
		suite.True(point.Timestamp.Before(end))
	}
}

// TestConcurrentAnalysis tests concurrent trend analysis
func (suite *AnalyzerTestSuite) TestConcurrentAnalysis() {
	const numGoroutines = 5

	errChan := make(chan error, numGoroutines)
	doneChan := make(chan struct{}, numGoroutines)

	dataPoints := suite.createSampleDataPoints()

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { doneChan <- struct{}{} }()

			analyzer := NewTrendAnalyzer(suite.config)
			analyzer.LoadCustomData(dataPoints)

			ctx := context.Background()
			_, err := analyzer.AnalyzeTrends(ctx)
			if err != nil {
				errChan <- err
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-doneChan
	}

	// Check for errors
	close(errChan)
	for err := range errChan {
		suite.T().Errorf("Concurrent analysis error: %v", err)
	}
}

// Helper methods for creating test data

// createSampleDataPoints creates sample data points for testing
func (suite *AnalyzerTestSuite) createSampleDataPoints() []AnalysisDataPoint {
	now := time.Now()
	return []AnalysisDataPoint{
		{
			Timestamp: now.Add(-10 * time.Hour),
			Coverage:  70.0,
			Branch:    "master",
			CommitSHA: "abc123",
		},
		{
			Timestamp: now.Add(-8 * time.Hour),
			Coverage:  72.5,
			Branch:    "master",
			CommitSHA: "def456",
		},
		{
			Timestamp: now.Add(-6 * time.Hour),
			Coverage:  75.0,
			Branch:    "master",
			CommitSHA: "ghi789",
		},
		{
			Timestamp: now.Add(-4 * time.Hour),
			Coverage:  77.5,
			Branch:    "master",
			CommitSHA: "jkl012",
		},
		{
			Timestamp: now.Add(-2 * time.Hour),
			Coverage:  80.0,
			Branch:    "master",
			CommitSHA: "mno345",
		},
		{
			Timestamp: now.Add(-1 * time.Hour),
			Coverage:  82.5,
			Branch:    "master",
			CommitSHA: "pqr678",
		},
	}
}

// createVolatileDataPoints creates data points with high volatility
func (suite *AnalyzerTestSuite) createVolatileDataPoints() []AnalysisDataPoint {
	now := time.Now()
	return []AnalysisDataPoint{
		{Timestamp: now.Add(-10 * time.Hour), Coverage: 70.0},
		{Timestamp: now.Add(-9 * time.Hour), Coverage: 85.0},
		{Timestamp: now.Add(-8 * time.Hour), Coverage: 60.0},
		{Timestamp: now.Add(-7 * time.Hour), Coverage: 90.0},
		{Timestamp: now.Add(-6 * time.Hour), Coverage: 55.0},
		{Timestamp: now.Add(-5 * time.Hour), Coverage: 95.0},
		{Timestamp: now.Add(-4 * time.Hour), Coverage: 50.0},
		{Timestamp: now.Add(-3 * time.Hour), Coverage: 80.0},
		{Timestamp: now.Add(-2 * time.Hour), Coverage: 65.0},
		{Timestamp: now.Add(-1 * time.Hour), Coverage: 88.0},
	}
}

// TestRun runs the test suite
func TestAnalyzerTestSuite(t *testing.T) {
	suite.Run(t, new(AnalyzerTestSuite))
}

// Benchmark tests
func BenchmarkNewTrendAnalyzer(b *testing.B) {
	config := &AnalyzerConfig{
		ShortTermDays:  7,
		MediumTermDays: 30,
		LongTermDays:   90,
		MinDataPoints:  5,
	}

	for i := 0; i < b.N; i++ {
		_ = NewTrendAnalyzer(config)
	}
}

func BenchmarkAnalyzeTrends(b *testing.B) {
	config := &AnalyzerConfig{
		ShortTermDays:  7,
		MediumTermDays: 30,
		LongTermDays:   90,
		MinDataPoints:  5,
		PredictionDays: 7,
	}

	analyzer := NewTrendAnalyzer(config)

	// Create sample data
	dataPoints := make([]AnalysisDataPoint, 20)
	now := time.Now()
	for i := 0; i < 20; i++ {
		dataPoints[i] = AnalysisDataPoint{
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
			Coverage:  70.0 + float64(i)*0.5,
			Branch:    "master",
		}
	}

	analyzer.LoadCustomData(dataPoints)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := analyzer.AnalyzeTrends(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGeneratePredictions(b *testing.B) {
	config := &AnalyzerConfig{
		PredictionDays: 14,
		MinDataPoints:  5,
	}

	analyzer := NewTrendAnalyzer(config)

	// Create sample data
	dataPoints := make([]AnalysisDataPoint, 10)
	now := time.Now()
	for i := 0; i < 10; i++ {
		dataPoints[i] = AnalysisDataPoint{
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
			Coverage:  70.0 + float64(i)*0.5,
			Smoothed:  70.0 + float64(i)*0.5,
		}
	}

	analyzer.LoadCustomData(dataPoints)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.generatePredictions()
		if err != nil {
			b.Fatal(err)
		}
	}
}
