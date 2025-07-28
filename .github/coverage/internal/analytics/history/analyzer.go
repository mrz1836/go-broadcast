// Package history provides enhanced time-series analysis for coverage data
package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/analytics/charts"
	"github.com/mrz1836/go-broadcast/coverage/internal/history"
)

// Static error definitions
var (
	ErrInsufficientDataPoints         = errors.New("insufficient data points for analysis")
	ErrInsufficientDataForPredictions = errors.New("insufficient data for predictions")
)

// TrendAnalyzer provides sophisticated time-series analysis for coverage trends
type TrendAnalyzer struct {
	config *AnalyzerConfig
	data   []AnalysisDataPoint
}

// AnalyzerConfig holds configuration for trend analysis
type AnalyzerConfig struct {
	// Analysis windows
	ShortTermDays  int // Short-term trend analysis (default: 7)
	MediumTermDays int // Medium-term trend analysis (default: 30)
	LongTermDays   int // Long-term trend analysis (default: 90)

	// Smoothing parameters
	MovingAvgWindow  int     // Moving average window size
	ExponentialAlpha float64 // Exponential smoothing alpha (0-1)

	// Significance thresholds
	SignificantChange   float64 // Minimum change to be considered significant
	VolatilityThreshold float64 // Threshold for high volatility
	TrendConfidence     float64 // Minimum confidence for trend detection

	// Prediction settings
	PredictionDays     int  // Number of days to predict ahead
	SeasonalAdjustment bool // Enable seasonal adjustment
	OutlierDetection   bool // Enable outlier detection and filtering

	// Quality thresholds
	MinDataPoints int // Minimum data points for analysis
	MaxGapDays    int // Maximum gap between data points
}

// AnalysisDataPoint represents an enhanced data point for analysis
type AnalysisDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	Coverage     float64   `json:"coverage"`
	Branch       string    `json:"branch"`
	CommitSHA    string    `json:"commit_sha"`
	PRNumber     int       `json:"pr_number,omitempty"`
	Author       string    `json:"author,omitempty"`
	FilesChanged int       `json:"files_changed,omitempty"`
	LinesAdded   int       `json:"lines_added,omitempty"`
	LinesRemoved int       `json:"lines_removed,omitempty"`
	TestsAdded   int       `json:"tests_added,omitempty"`
	IsOutlier    bool      `json:"is_outlier"`
	Smoothed     float64   `json:"smoothed_value"`
	Prediction   float64   `json:"prediction,omitempty"`
	Confidence   float64   `json:"confidence,omitempty"`
}

// TrendReport contains comprehensive trend analysis results
type TrendReport struct {
	// Summary
	Summary TrendSummary `json:"summary"`

	// Trend analysis
	ShortTermTrend  TrendAnalysis `json:"short_term_trend"`
	MediumTermTrend TrendAnalysis `json:"medium_term_trend"`
	LongTermTrend   TrendAnalysis `json:"long_term_trend"`

	// Volatility analysis
	Volatility VolatilityAnalysis `json:"volatility"`

	// Predictions
	Predictions []PredictionPoint `json:"predictions"`

	// Quality metrics
	QualityMetrics QualityMetrics `json:"quality_metrics"`

	// Chart data
	ChartData *charts.ChartData `json:"chart_data"`

	// Insights and recommendations
	Insights        []Insight        `json:"insights"`
	Recommendations []Recommendation `json:"recommendations"`

	// Metadata
	GeneratedAt    time.Time     `json:"generated_at"`
	AnalysisWindow time.Duration `json:"analysis_window"`
	DataPointCount int           `json:"data_point_count"`
}

// TrendSummary provides high-level trend information
type TrendSummary struct {
	CurrentCoverage  float64        `json:"current_coverage"`
	PreviousCoverage float64        `json:"previous_coverage"`
	Change           float64        `json:"change"`
	ChangePercent    float64        `json:"change_percent"`
	Direction        TrendDirection `json:"direction"`
	Magnitude        TrendMagnitude `json:"magnitude"`
	Confidence       float64        `json:"confidence"`
	QualityGrade     string         `json:"quality_grade"`
}

// TrendAnalysis contains detailed trend analysis for a specific time period
type TrendAnalysis struct {
	Period         string         `json:"period"`
	StartDate      time.Time      `json:"start_date"`
	EndDate        time.Time      `json:"end_date"`
	Direction      TrendDirection `json:"direction"`
	Slope          float64        `json:"slope"`
	RSquared       float64        `json:"r_squared"`
	Confidence     float64        `json:"confidence"`
	AverageChange  float64        `json:"average_change"`
	MaxIncrease    float64        `json:"max_increase"`
	MaxDecrease    float64        `json:"max_decrease"`
	ChangeVelocity float64        `json:"change_velocity"`
	Momentum       TrendMomentum  `json:"momentum"`
}

// VolatilityAnalysis contains volatility metrics
type VolatilityAnalysis struct {
	StandardDeviation    float64         `json:"standard_deviation"`
	Variance             float64         `json:"variance"`
	CoefficientVariation float64         `json:"coefficient_variation"`
	VolatilityLevel      VolatilityLevel `json:"volatility_level"`
	LargestFluctuation   float64         `json:"largest_fluctuation"`
	AverageFluctuation   float64         `json:"average_fluctuation"`
	StabilityScore       float64         `json:"stability_score"`
}

// PredictionPoint represents a predicted future data point
type PredictionPoint struct {
	Date               time.Time          `json:"date"`
	PredictedCoverage  float64            `json:"predicted_coverage"`
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
	Methodology        string             `json:"methodology"`
	Reliability        float64            `json:"reliability"`
}

// ConfidenceInterval represents prediction confidence bounds
type ConfidenceInterval struct {
	Lower      float64 `json:"lower"`
	Upper      float64 `json:"upper"`
	Confidence float64 `json:"confidence"`
}

// QualityMetrics contains data quality assessment
type QualityMetrics struct {
	DataCompleteness  float64       `json:"data_completeness"`
	DataConsistency   float64       `json:"data_consistency"`
	OutlierCount      int           `json:"outlier_count"`
	MissingDataPoints int           `json:"missing_data_points"`
	LargestGap        time.Duration `json:"largest_gap"`
	QualityScore      float64       `json:"quality_score"`
	ReliabilityGrade  string        `json:"reliability_grade"`
}

// Insight represents an analytical insight
type Insight struct {
	Type           InsightType            `json:"type"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Severity       InsightSeverity        `json:"severity"`
	Confidence     float64                `json:"confidence"`
	SupportingData map[string]interface{} `json:"supporting_data"`
}

// Recommendation represents an actionable recommendation
type Recommendation struct {
	Type           RecommendationType     `json:"type"`
	Priority       RecommendationPriority `json:"priority"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Actions        []string               `json:"actions"`
	ExpectedImpact float64                `json:"expected_impact"`
	Timeline       string                 `json:"timeline"`
}

// Enums for categorization

// TrendDirection represents the direction of a trend
type TrendDirection string

const (
	// TrendUp indicates an upward trend
	TrendUp TrendDirection = "up"
	// TrendDown indicates a downward trend
	TrendDown TrendDirection = "down"
	// TrendStable indicates a stable trend
	TrendStable TrendDirection = "stable"
	// TrendVolatile indicates a volatile trend
	TrendVolatile TrendDirection = "volatile"
)

// TrendMagnitude represents the strength of a trend
type TrendMagnitude string

const (
	// MagnitudeSignificant indicates a significant magnitude
	MagnitudeSignificant TrendMagnitude = "significant"
	// MagnitudeModerate indicates a moderate magnitude
	MagnitudeModerate TrendMagnitude = "moderate"
	// MagnitudeMinor indicates a minor magnitude
	MagnitudeMinor TrendMagnitude = "minor"
	// MagnitudeNegligible indicates a negligible magnitude
	MagnitudeNegligible TrendMagnitude = "negligible"
)

// TrendMomentum represents the rate of change in a trend
type TrendMomentum string

const (
	// MomentumAccelerating indicates an accelerating momentum
	MomentumAccelerating TrendMomentum = "accelerating"
	// MomentumSteady indicates a steady momentum
	MomentumSteady TrendMomentum = "steady"
	// MomentumDecelerating indicates a decelerating momentum
	MomentumDecelerating TrendMomentum = "decelerating"
)

// VolatilityLevel represents the level of variation in data
type VolatilityLevel string

const (
	// VolatilityLow indicates low volatility
	VolatilityLow VolatilityLevel = "low"
	// VolatilityMedium indicates medium volatility
	VolatilityMedium VolatilityLevel = "medium"
	// VolatilityHigh indicates high volatility
	VolatilityHigh VolatilityLevel = "high"
)

// InsightType represents the type of analytical insight
type InsightType string

const (
	// InsightTrend indicates a trend insight
	InsightTrend InsightType = "trend"
	// InsightAnomaly indicates an anomaly insight
	InsightAnomaly InsightType = "anomaly"
	// InsightMilestone indicates a milestone insight
	InsightMilestone InsightType = "milestone"
	// InsightRegression indicates a regression insight
	InsightRegression InsightType = "regression"
	// InsightOpportunity indicates an opportunity insight
	InsightOpportunity InsightType = "opportunity"
)

// InsightSeverity represents the severity level of an insight
type InsightSeverity string

const (
	// SeverityInfo indicates informational severity
	SeverityInfo InsightSeverity = "info"
	// SeverityWarning indicates warning severity
	SeverityWarning InsightSeverity = "warning"
	// SeverityCritical indicates critical severity
	SeverityCritical InsightSeverity = "critical"
)

// RecommendationType represents the type of recommendation
type RecommendationType string

const (
	// RecommendationProcess indicates a process recommendation
	RecommendationProcess RecommendationType = "process"
	// RecommendationTesting indicates a testing recommendation
	RecommendationTesting RecommendationType = "testing"
	// RecommendationGoals indicates a goals recommendation
	RecommendationGoals RecommendationType = "goals"
	// RecommendationMonitoring indicates a monitoring recommendation
	RecommendationMonitoring RecommendationType = "monitoring"
)

// RecommendationPriority represents the priority level of a recommendation
type RecommendationPriority string

const (
	// PriorityHigh indicates high priority
	PriorityHigh RecommendationPriority = "high"
	// PriorityMedium indicates medium priority
	PriorityMedium RecommendationPriority = "medium"
	// PriorityLow indicates low priority
	PriorityLow RecommendationPriority = "low"
)

// NewTrendAnalyzer creates a new trend analyzer with default configuration
func NewTrendAnalyzer(config *AnalyzerConfig) *TrendAnalyzer {
	if config == nil {
		config = &AnalyzerConfig{
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
	}

	return &TrendAnalyzer{
		config: config,
		data:   make([]AnalysisDataPoint, 0),
	}
}

// LoadHistoryData loads historical data from the existing history system
func (ta *TrendAnalyzer) LoadHistoryData(ctx context.Context, historyTracker *history.Tracker, branch string, days int) error {
	// Get trend data which includes historical entries
	trendData, err := historyTracker.GetTrend(ctx,
		history.WithTrendBranch(branch),
		history.WithTrendDays(days),
	)
	if err != nil {
		return fmt.Errorf("failed to load history data: %w", err)
	}

	// Convert trend entries to analysis data points
	if trendData.Entries == nil {
		ta.data = make([]AnalysisDataPoint, 0)
		return nil
	}

	ta.data = make([]AnalysisDataPoint, 0, len(trendData.Entries))
	for _, entry := range trendData.Entries {
		point := AnalysisDataPoint{
			Timestamp: entry.Timestamp,
			Coverage:  entry.Coverage.Percentage,
			Branch:    entry.Branch,
			CommitSHA: entry.CommitSHA,
		}
		ta.data = append(ta.data, point)
	}

	// Sort by timestamp
	sort.Slice(ta.data, func(i, j int) bool {
		return ta.data[i].Timestamp.Before(ta.data[j].Timestamp)
	})

	return nil
}

// LoadCustomData loads custom analysis data points
func (ta *TrendAnalyzer) LoadCustomData(dataPoints []AnalysisDataPoint) {
	ta.data = make([]AnalysisDataPoint, len(dataPoints))
	copy(ta.data, dataPoints)

	// Sort by timestamp
	sort.Slice(ta.data, func(i, j int) bool {
		return ta.data[i].Timestamp.Before(ta.data[j].Timestamp)
	})
}

// AnalyzeTrends performs comprehensive trend analysis
func (ta *TrendAnalyzer) AnalyzeTrends(ctx context.Context) (*TrendReport, error) {
	if len(ta.data) < ta.config.MinDataPoints {
		return nil, fmt.Errorf("%w: need at least %d, got %d",
			ErrInsufficientDataPoints, ta.config.MinDataPoints, len(ta.data))
	}

	// Pre-process data
	ta.preprocessData()

	report := &TrendReport{
		GeneratedAt:    time.Now(),
		DataPointCount: len(ta.data),
	}

	if len(ta.data) > 0 {
		report.AnalysisWindow = ta.data[len(ta.data)-1].Timestamp.Sub(ta.data[0].Timestamp)
	}

	// Generate summary
	report.Summary = ta.generateSummary()

	// Analyze trends for different time periods
	report.ShortTermTrend = ta.analyzePeriodTrend(ta.config.ShortTermDays, "short-term")
	report.MediumTermTrend = ta.analyzePeriodTrend(ta.config.MediumTermDays, "medium-term")
	report.LongTermTrend = ta.analyzePeriodTrend(ta.config.LongTermDays, "long-term")

	// Analyze volatility
	report.Volatility = ta.analyzeVolatility()

	// Generate predictions
	predictions, err := ta.generatePredictions()
	if err != nil {
		return nil, fmt.Errorf("failed to generate predictions: %w", err)
	}
	report.Predictions = predictions

	// Calculate quality metrics
	report.QualityMetrics = ta.calculateQualityMetrics()

	// Generate chart data
	report.ChartData = ta.generateChartData()

	// Generate insights and recommendations
	report.Insights = ta.generateInsights(report)
	report.Recommendations = ta.generateRecommendations(report)

	return report, nil
}

// preprocessData performs data preprocessing including outlier detection and smoothing
func (ta *TrendAnalyzer) preprocessData() {
	if len(ta.data) == 0 {
		return
	}

	// Detect outliers if enabled
	if ta.config.OutlierDetection {
		ta.detectOutliers()
	}

	// Apply smoothing
	ta.applySmoothing()
}

// detectOutliers identifies and marks outlier data points
func (ta *TrendAnalyzer) detectOutliers() {
	if len(ta.data) < 3 {
		return
	}

	// Calculate mean and standard deviation
	sum := 0.0
	for _, point := range ta.data {
		sum += point.Coverage
	}
	mean := sum / float64(len(ta.data))

	sumSquares := 0.0
	for _, point := range ta.data {
		diff := point.Coverage - mean
		sumSquares += diff * diff
	}
	stdDev := math.Sqrt(sumSquares / float64(len(ta.data)))

	// Mark outliers (beyond 2 standard deviations)
	threshold := 2.0
	for i := range ta.data {
		diff := math.Abs(ta.data[i].Coverage - mean)
		ta.data[i].IsOutlier = diff > threshold*stdDev
	}
}

// applySmoothing applies exponential smoothing to the data
func (ta *TrendAnalyzer) applySmoothing() {
	if len(ta.data) == 0 {
		return
	}

	// Initialize first smoothed value
	ta.data[0].Smoothed = ta.data[0].Coverage

	// Apply exponential smoothing
	alpha := ta.config.ExponentialAlpha
	for i := 1; i < len(ta.data); i++ {
		if ta.data[i].IsOutlier {
			// Use previous smoothed value for outliers
			ta.data[i].Smoothed = ta.data[i-1].Smoothed
		} else {
			ta.data[i].Smoothed = alpha*ta.data[i].Coverage + (1-alpha)*ta.data[i-1].Smoothed
		}
	}
}

// generateSummary creates a high-level trend summary
func (ta *TrendAnalyzer) generateSummary() TrendSummary {
	if len(ta.data) == 0 {
		return TrendSummary{}
	}

	current := ta.data[len(ta.data)-1].Coverage
	previous := current
	if len(ta.data) > 1 {
		previous = ta.data[len(ta.data)-2].Coverage
	}

	change := current - previous
	changePercent := 0.0
	if previous != 0 {
		changePercent = (change / previous) * 100
	}

	direction := ta.determineDirection(change)
	magnitude := ta.determineMagnitude(math.Abs(change))
	confidence := ta.calculateTrendConfidence()

	return TrendSummary{
		CurrentCoverage:  current,
		PreviousCoverage: previous,
		Change:           change,
		ChangePercent:    changePercent,
		Direction:        direction,
		Magnitude:        magnitude,
		Confidence:       confidence,
		QualityGrade:     ta.calculateQualityGrade(current),
	}
}

// analyzePeriodTrend analyzes trend for a specific time period
func (ta *TrendAnalyzer) analyzePeriodTrend(days int, period string) TrendAnalysis {
	// Filter data for the specified period
	cutoff := time.Now().AddDate(0, 0, -days)
	var periodData []AnalysisDataPoint

	for _, point := range ta.data {
		if point.Timestamp.After(cutoff) {
			periodData = append(periodData, point)
		}
	}

	if len(periodData) < 2 {
		return TrendAnalysis{Period: period}
	}

	// Calculate linear regression
	slope, rSquared := ta.calculateLinearRegression(periodData)

	// Calculate other metrics
	var changes []float64
	maxIncrease := 0.0
	maxDecrease := 0.0

	for i := 1; i < len(periodData); i++ {
		change := periodData[i].Coverage - periodData[i-1].Coverage
		changes = append(changes, change)

		if change > maxIncrease {
			maxIncrease = change
		}
		if change < maxDecrease {
			maxDecrease = change
		}
	}

	averageChange := 0.0
	if len(changes) > 0 {
		sum := 0.0
		for _, change := range changes {
			sum += change
		}
		averageChange = sum / float64(len(changes))
	}

	direction := ta.determineDirection(slope)
	momentum := ta.determineMomentum(periodData)
	confidence := ta.calculatePeriodConfidence(rSquared, len(periodData))

	return TrendAnalysis{
		Period:         period,
		StartDate:      periodData[0].Timestamp,
		EndDate:        periodData[len(periodData)-1].Timestamp,
		Direction:      direction,
		Slope:          slope,
		RSquared:       rSquared,
		Confidence:     confidence,
		AverageChange:  averageChange,
		MaxIncrease:    maxIncrease,
		MaxDecrease:    maxDecrease,
		ChangeVelocity: ta.calculateChangeVelocity(periodData),
		Momentum:       momentum,
	}
}

// calculateLinearRegression calculates slope and R-squared for linear trend
func (ta *TrendAnalyzer) calculateLinearRegression(data []AnalysisDataPoint) (float64, float64) {
	if len(data) < 2 {
		return 0, 0
	}

	n := float64(len(data))
	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for i, point := range data {
		x := float64(i)
		y := point.Smoothed

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	// Calculate slope (beta1)
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, 0
	}

	slope := (n*sumXY - sumX*sumY) / denominator

	// Calculate R-squared
	yMean := sumY / n
	var ssr, sst float64

	for i, point := range data {
		x := float64(i)
		y := point.Smoothed
		yPred := slope*x + (yMean - slope*(sumX/n))

		ssr += (yPred - yMean) * (yPred - yMean)
		sst += (y - yMean) * (y - yMean)
	}

	rSquared := 0.0
	if sst != 0 {
		rSquared = ssr / sst
	}

	return slope, rSquared
}

// analyzeVolatility calculates volatility metrics
func (ta *TrendAnalyzer) analyzeVolatility() VolatilityAnalysis {
	if len(ta.data) < 2 {
		return VolatilityAnalysis{}
	}

	// Calculate changes between consecutive points
	var changes []float64
	for i := 1; i < len(ta.data); i++ {
		change := math.Abs(ta.data[i].Coverage - ta.data[i-1].Coverage)
		changes = append(changes, change)
	}

	// Calculate mean change
	sum := 0.0
	for _, change := range changes {
		sum += change
	}
	meanChange := sum / float64(len(changes))

	// Calculate standard deviation
	sumSquares := 0.0
	for _, change := range changes {
		diff := change - meanChange
		sumSquares += diff * diff
	}
	stdDev := math.Sqrt(sumSquares / float64(len(changes)))

	// Calculate variance
	variance := sumSquares / float64(len(changes))

	// Calculate coefficient of variation
	coeffVar := 0.0
	if meanChange != 0 {
		coeffVar = stdDev / meanChange
	}

	// Find largest fluctuation
	maxChange := 0.0
	for _, change := range changes {
		if change > maxChange {
			maxChange = change
		}
	}

	// Determine volatility level
	level := VolatilityLow
	if stdDev > ta.config.VolatilityThreshold {
		level = VolatilityHigh
	} else if stdDev > ta.config.VolatilityThreshold/2 {
		level = VolatilityMedium
	}

	// Calculate stability score (inverse of volatility)
	stabilityScore := math.Max(0, 100-stdDev*10)

	return VolatilityAnalysis{
		StandardDeviation:    stdDev,
		Variance:             variance,
		CoefficientVariation: coeffVar,
		VolatilityLevel:      level,
		LargestFluctuation:   maxChange,
		AverageFluctuation:   meanChange,
		StabilityScore:       stabilityScore,
	}
}

// generatePredictions creates future coverage predictions
func (ta *TrendAnalyzer) generatePredictions() ([]PredictionPoint, error) {
	if len(ta.data) < 3 {
		return nil, ErrInsufficientDataForPredictions
	}

	var predictions []PredictionPoint

	// Use linear regression for simple prediction
	slope, rSquared := ta.calculateLinearRegression(ta.data)

	lastPoint := ta.data[len(ta.data)-1]

	for i := 1; i <= ta.config.PredictionDays; i++ {
		futureDate := lastPoint.Timestamp.AddDate(0, 0, i)

		// Simple linear prediction
		predictedValue := lastPoint.Smoothed + slope*float64(i)

		// Clamp to reasonable bounds
		predictedValue = math.Max(0, math.Min(100, predictedValue))

		// Calculate confidence based on R-squared and distance
		confidence := rSquared * math.Exp(-float64(i)*0.1)
		reliability := math.Max(0.1, confidence)

		// Calculate confidence interval
		margin := (1.0 - confidence) * 10.0 // Simple margin calculation

		prediction := PredictionPoint{
			Date:              futureDate,
			PredictedCoverage: predictedValue,
			ConfidenceInterval: ConfidenceInterval{
				Lower:      math.Max(0, predictedValue-margin),
				Upper:      math.Min(100, predictedValue+margin),
				Confidence: confidence,
			},
			Methodology: "linear_regression",
			Reliability: reliability,
		}

		predictions = append(predictions, prediction)
	}

	return predictions, nil
}

// calculateQualityMetrics assesses data quality
func (ta *TrendAnalyzer) calculateQualityMetrics() QualityMetrics {
	if len(ta.data) == 0 {
		return QualityMetrics{}
	}

	// Count outliers
	outlierCount := 0
	for _, point := range ta.data {
		if point.IsOutlier {
			outlierCount++
		}
	}

	// Calculate data completeness (simplified)
	expectedDataPoints := int(ta.data[len(ta.data)-1].Timestamp.Sub(ta.data[0].Timestamp).Hours() / 24)
	completeness := float64(len(ta.data)) / float64(expectedDataPoints) * 100
	if completeness > 100 {
		completeness = 100
	}

	// Calculate largest gap
	largestGap := time.Duration(0)
	for i := 1; i < len(ta.data); i++ {
		gap := ta.data[i].Timestamp.Sub(ta.data[i-1].Timestamp)
		if gap > largestGap {
			largestGap = gap
		}
	}

	// Calculate consistency (based on volatility)
	volatility := ta.analyzeVolatility()
	consistency := math.Max(0, 100-volatility.StandardDeviation*5)

	// Calculate overall quality score
	qualityScore := (completeness + consistency) / 2

	// Determine reliability grade
	reliabilityGrade := "F"
	switch {
	case qualityScore >= 90:
		reliabilityGrade = "A"
	case qualityScore >= 80:
		reliabilityGrade = "B"
	case qualityScore >= 70:
		reliabilityGrade = "C"
	case qualityScore >= 60:
		reliabilityGrade = "D"
	}

	return QualityMetrics{
		DataCompleteness:  completeness,
		DataConsistency:   consistency,
		OutlierCount:      outlierCount,
		MissingDataPoints: expectedDataPoints - len(ta.data),
		LargestGap:        largestGap,
		QualityScore:      qualityScore,
		ReliabilityGrade:  reliabilityGrade,
	}
}

// generateChartData creates chart data for visualization
func (ta *TrendAnalyzer) generateChartData() *charts.ChartData {
	points := make([]charts.DataPoint, 0, len(ta.data))

	for _, point := range ta.data {
		chartPoint := charts.DataPoint{
			Timestamp: point.Timestamp,
			Value:     point.Coverage,
			Label:     fmt.Sprintf("%.1f%%", point.Coverage),
			Metadata: charts.Metadata{
				Branch:    point.Branch,
				CommitSHA: point.CommitSHA,
				PRNumber:  point.PRNumber,
				Coverage:  point.Coverage,
				Author:    point.Author,
			},
		}
		points = append(points, chartPoint)
	}

	chartData := &charts.ChartData{
		Points:     points,
		Title:      "Coverage Trend Analysis",
		XAxisLabel: "Time",
		YAxisLabel: "Coverage (%)",
	}

	if len(ta.data) > 0 {
		chartData.TimeRange = charts.TimeRange{
			Start: ta.data[0].Timestamp,
			End:   ta.data[len(ta.data)-1].Timestamp,
		}
	}

	return chartData
}

// Helper methods for calculations and determinations

func (ta *TrendAnalyzer) determineDirection(change float64) TrendDirection {
	if math.Abs(change) < ta.config.SignificantChange {
		return TrendStable
	}

	if change > 0 {
		return TrendUp
	}

	return TrendDown
}

func (ta *TrendAnalyzer) determineMagnitude(change float64) TrendMagnitude {
	switch {
	case change >= 5.0:
		return MagnitudeSignificant
	case change >= 2.0:
		return MagnitudeModerate
	case change >= 0.5:
		return MagnitudeMinor
	default:
		return MagnitudeNegligible
	}
}

func (ta *TrendAnalyzer) determineMomentum(data []AnalysisDataPoint) TrendMomentum {
	if len(data) < 3 {
		return MomentumSteady
	}

	// Compare recent trend with earlier trend
	mid := len(data) / 2
	earlySlope, _ := ta.calculateLinearRegression(data[:mid])
	recentSlope, _ := ta.calculateLinearRegression(data[mid:])

	slopeDiff := recentSlope - earlySlope

	if math.Abs(slopeDiff) < 0.1 {
		return MomentumSteady
	}

	if slopeDiff > 0 {
		return MomentumAccelerating
	}

	return MomentumDecelerating
}

func (ta *TrendAnalyzer) calculateTrendConfidence() float64 {
	if len(ta.data) < 2 {
		return 0
	}

	_, rSquared := ta.calculateLinearRegression(ta.data)
	return rSquared
}

func (ta *TrendAnalyzer) calculatePeriodConfidence(rSquared float64, dataPoints int) float64 {
	// Confidence increases with R-squared and number of data points
	dataConfidence := math.Min(1.0, float64(dataPoints)/10.0)
	return rSquared * dataConfidence
}

func (ta *TrendAnalyzer) calculateChangeVelocity(data []AnalysisDataPoint) float64 {
	if len(data) < 2 {
		return 0
	}

	totalChange := data[len(data)-1].Coverage - data[0].Coverage
	timeSpan := data[len(data)-1].Timestamp.Sub(data[0].Timestamp).Hours() / 24

	if timeSpan == 0 {
		return 0
	}

	return totalChange / timeSpan // Change per day
}

func (ta *TrendAnalyzer) calculateQualityGrade(coverage float64) string {
	switch {
	case coverage >= 95:
		return "A+"
	case coverage >= 90:
		return "A"
	case coverage >= 85:
		return "B+"
	case coverage >= 80:
		return "B"
	case coverage >= 70:
		return "C"
	case coverage >= 60:
		return "D"
	default:
		return "F"
	}
}

// generateInsights creates analytical insights based on the trend report
func (ta *TrendAnalyzer) generateInsights(report *TrendReport) []Insight {
	var insights []Insight

	// Trend insights
	if report.Summary.Direction == TrendUp && report.Summary.Magnitude != MagnitudeNegligible {
		insights = append(insights, Insight{
			Type:  InsightTrend,
			Title: "Positive Coverage Trend",
			Description: fmt.Sprintf("Coverage has increased by %.1f%% with %s confidence",
				report.Summary.Change, report.Summary.Magnitude),
			Severity:   SeverityInfo,
			Confidence: report.Summary.Confidence,
		})
	}

	if report.Summary.Direction == TrendDown && report.Summary.Magnitude != MagnitudeNegligible {
		severity := SeverityWarning
		if report.Summary.Magnitude == MagnitudeSignificant {
			severity = SeverityCritical
		}

		insights = append(insights, Insight{
			Type:  InsightRegression,
			Title: "Coverage Regression Detected",
			Description: fmt.Sprintf("Coverage has decreased by %.1f%% - attention needed",
				math.Abs(report.Summary.Change)),
			Severity:   severity,
			Confidence: report.Summary.Confidence,
		})
	}

	// Volatility insights
	if report.Volatility.VolatilityLevel == VolatilityHigh {
		insights = append(insights, Insight{
			Type:  InsightAnomaly,
			Title: "High Coverage Volatility",
			Description: fmt.Sprintf("Coverage shows high volatility (σ=%.1f) - consider process improvements",
				report.Volatility.StandardDeviation),
			Severity:   SeverityWarning,
			Confidence: 0.9,
		})
	}

	// Milestone insights
	currentCoverage := report.Summary.CurrentCoverage
	milestones := []float64{50, 60, 70, 80, 90, 95}

	for _, milestone := range milestones {
		if currentCoverage >= milestone && currentCoverage < milestone+5 {
			insights = append(insights, Insight{
				Type:        InsightMilestone,
				Title:       fmt.Sprintf("%.0f%% Coverage Milestone Reached", milestone),
				Description: fmt.Sprintf("Congratulations on reaching %.0f%% coverage!", milestone),
				Severity:    SeverityInfo,
				Confidence:  1.0,
			})
			break
		}
	}

	return insights
}

// generateRecommendations creates actionable recommendations
func (ta *TrendAnalyzer) generateRecommendations(report *TrendReport) []Recommendation {
	var recommendations []Recommendation

	// Coverage improvement recommendations
	if report.Summary.CurrentCoverage < 80 {
		recommendations = append(recommendations, Recommendation{
			Type:        RecommendationTesting,
			Priority:    PriorityHigh,
			Title:       "Improve Test Coverage",
			Description: "Current coverage is below 80% threshold",
			Actions: []string{
				"Identify uncovered code paths",
				"Add unit tests for critical functions",
				"Consider integration tests for complex workflows",
			},
			ExpectedImpact: 80 - report.Summary.CurrentCoverage,
			Timeline:       "2-3 weeks",
		})
	}

	// Volatility recommendations
	if report.Volatility.VolatilityLevel == VolatilityHigh {
		recommendations = append(recommendations, Recommendation{
			Type:        RecommendationProcess,
			Priority:    PriorityMedium,
			Title:       "Stabilize Coverage Process",
			Description: "High volatility indicates inconsistent testing practices",
			Actions: []string{
				"Implement pre-commit hooks for testing",
				"Add coverage gates to CI/CD pipeline",
				"Establish team testing guidelines",
			},
			ExpectedImpact: 5.0, // Reduce volatility
			Timeline:       "1-2 weeks",
		})
	}

	// Monitoring recommendations
	if report.QualityMetrics.QualityScore < 70 {
		recommendations = append(recommendations, Recommendation{
			Type:        RecommendationMonitoring,
			Priority:    PriorityMedium,
			Title:       "Improve Data Quality",
			Description: "Coverage data quality is below acceptable threshold",
			Actions: []string{
				"Ensure consistent test execution",
				"Fix gaps in coverage reporting",
				"Implement data validation checks",
			},
			ExpectedImpact: 30.0, // Improve quality score
			Timeline:       "1 week",
		})
	}

	return recommendations
}

// ExportToJSON exports the analysis data to JSON format
func (ta *TrendAnalyzer) ExportToJSON() ([]byte, error) {
	return json.MarshalIndent(ta.data, "", "  ")
}

// GetDataForPeriod returns data points for a specific time period
func (ta *TrendAnalyzer) GetDataForPeriod(start, end time.Time) []AnalysisDataPoint {
	var periodData []AnalysisDataPoint

	for _, point := range ta.data {
		if point.Timestamp.After(start) && point.Timestamp.Before(end) {
			periodData = append(periodData, point)
		}
	}

	return periodData
}
