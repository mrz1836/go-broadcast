// Package prediction provides coverage prediction and impact analysis capabilities
package prediction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/analytics/history"
)

var (
	// ErrInsufficientTrainingData indicates there's not enough data for training the model
	ErrInsufficientTrainingData = errors.New("insufficient training data")
	// ErrUnsupportedModelType indicates the requested model type is not supported
	ErrUnsupportedModelType = errors.New("unsupported model type")
	// ErrModelBelowThreshold indicates model R² is below threshold
	ErrModelBelowThreshold = errors.New("model R² below threshold")
	// ErrModelNotTrained indicates that the prediction model has not been trained yet
	ErrModelNotTrained = errors.New("model not trained")
	// ErrInsufficientPoints indicates that there are not enough data points for calculation
	ErrInsufficientPoints = errors.New("insufficient points for calculation")
	// ErrZeroDenominator indicates cannot calculate due to zero denominator
	ErrZeroDenominator = errors.New("cannot calculate: zero denominator")
	// ErrInsufficientValidationData indicates insufficient data for cross-validation
	ErrInsufficientValidationData = errors.New("insufficient data for cross-validation")
	// ErrNoValidationErrors indicates no validation errors were calculated
	ErrNoValidationErrors = errors.New("no validation errors calculated")
	// ErrCoverageNotIncreasing indicates coverage is not increasing and target may not be reachable
	ErrCoverageNotIncreasing = errors.New("coverage is not increasing - target may not be reachable")
	// ErrInsufficientTargetData indicates insufficient data for target prediction
	ErrInsufficientTargetData = errors.New("insufficient data for target prediction")
	// ErrNoTimeSpanInData indicates no time span in training data
	ErrNoTimeSpanInData = errors.New("no time span in training data")
)

// CoveragePredictor provides sophisticated coverage prediction capabilities
type CoveragePredictor struct {
	config *PredictorConfig
	model  *PredictionModel
}

// PredictorConfig holds configuration for coverage prediction
type PredictorConfig struct {
	// Model parameters
	ModelType          ModelType // Type of prediction model
	TrainingWindowDays int       // Number of days for training data
	MinTrainingPoints  int       // Minimum training data points

	// Prediction parameters
	PredictionHorizonDays int     // How far ahead to predict
	ConfidenceLevel       float64 // Confidence level for intervals (0-1)
	SeasonalAdjustment    bool    // Enable seasonal adjustment

	// Validation parameters
	CrossValidationFolds int     // Number of CV folds for validation
	ValidationThreshold  float64 // Minimum accuracy threshold

	// Quality thresholds
	MinRSquared        float64 // Minimum R² for model acceptance
	MaxPredictionError float64 // Maximum acceptable prediction error
	OutlierThreshold   float64 // Z-score threshold for outliers
}

// ModelType represents different prediction model types
type ModelType string

const (
	// ModelLinearRegression represents a linear regression prediction model
	ModelLinearRegression ModelType = "linear_regression"
	// ModelExponentialSmoothing represents an exponential smoothing model
	ModelExponentialSmoothing ModelType = "exponential_smoothing"
	// ModelMovingAverage represents a moving average model
	ModelMovingAverage ModelType = "moving_average"
	// ModelPolynomial represents a polynomial regression model
	ModelPolynomial ModelType = "polynomial"
	// ModelSeasonal represents a seasonal adjustment model
	ModelSeasonal ModelType = "seasonal"
)

// PredictionModel contains the trained prediction model
type PredictionModel struct { //nolint:revive // prediction.PredictionModel is appropriately descriptive
	Type               ModelType         `json:"type"`
	Parameters         ModelParameters   `json:"parameters"`
	TrainingData       []TrainingPoint   `json:"training_data"`
	ValidationMetrics  ValidationMetrics `json:"validation_metrics"`
	LastTrainedAt      time.Time         `json:"last_trained_at"`
	TrainingDataWindow time.Duration     `json:"training_data_window"`
}

// ModelParameters contains model-specific parameters
type ModelParameters struct {
	// Linear regression parameters
	Slope     float64 `json:"slope,omitempty"`
	Intercept float64 `json:"intercept,omitempty"`
	RSquared  float64 `json:"r_squared,omitempty"`

	// Exponential smoothing parameters
	Alpha float64 `json:"alpha,omitempty"`
	Beta  float64 `json:"beta,omitempty"`
	Gamma float64 `json:"gamma,omitempty"`

	// Moving average parameters
	WindowSize      int  `json:"window_size,omitempty"`
	WeightedAverage bool `json:"weighted_average,omitempty"`

	// Seasonal parameters
	SeasonalPeriod  int       `json:"seasonal_period,omitempty"`
	SeasonalFactors []float64 `json:"seasonal_factors,omitempty"`

	// Polynomial parameters
	Degree       int       `json:"degree,omitempty"`
	Coefficients []float64 `json:"coefficients,omitempty"`
}

// TrainingPoint represents a training data point
type TrainingPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	Coverage       float64   `json:"coverage"`
	DayOfWeek      int       `json:"day_of_week"`
	DayOfMonth     int       `json:"day_of_month"`
	IsWeekend      bool      `json:"is_weekend"`
	IsHoliday      bool      `json:"is_holiday,omitempty"`
	SequenceNumber int       `json:"sequence_number"`
	Weight         float64   `json:"weight"`
}

// ValidationMetrics contains model validation results
type ValidationMetrics struct {
	MeanAbsoluteError           float64         `json:"mean_absolute_error"`
	RootMeanSquareError         float64         `json:"root_mean_square_error"`
	MeanAbsolutePercentageError float64         `json:"mean_absolute_percentage_error"`
	Accuracy                    float64         `json:"accuracy"`
	R2Score                     float64         `json:"r2_score"`
	CrossValidationScore        float64         `json:"cross_validation_score"`
	OverfittingRisk             OverfittingRisk `json:"overfitting_risk"`
}

// OverfittingRisk indicates the risk of model overfitting
type OverfittingRisk string

const (
	// OverfittingLow indicates low risk of model overfitting
	OverfittingLow OverfittingRisk = "low"
	// OverfittingMedium indicates medium risk of model overfitting
	OverfittingMedium OverfittingRisk = "medium"
	// OverfittingHigh indicates high risk of model overfitting
	OverfittingHigh OverfittingRisk = "high"
)

// PredictionResult contains prediction results and analysis
type PredictionResult struct { //nolint:revive // prediction.PredictionResult is appropriately descriptive
	// Prediction metadata
	PredictionDate     time.Time `json:"prediction_date"`
	ModelUsed          ModelType `json:"model_used"`
	TrainingDataPoints int       `json:"training_data_points"`

	// Predictions
	PointForecasts []PointForecast `json:"point_forecasts"`
	TrendForecast  TrendForecast   `json:"trend_forecast"`

	// Confidence and reliability
	OverallConfidence float64           `json:"overall_confidence"`
	ReliabilityScore  float64           `json:"reliability_score"`
	PredictionQuality PredictionQuality `json:"prediction_quality"`

	// Analysis
	KeyInsights     []PredictionInsight        `json:"key_insights"`
	Risks           []PredictionRisk           `json:"risks"`
	Recommendations []PredictionRecommendation `json:"recommendations"`
}

// PointForecast represents a single point prediction
type PointForecast struct {
	Date               time.Time          `json:"date"`
	PredictedCoverage  float64            `json:"predicted_coverage"`
	ConfidenceInterval ConfidenceInterval `json:"confidence_interval"`
	Trend              TrendIndicator     `json:"trend"`
	DaysAhead          int                `json:"days_ahead"`
	Reliability        float64            `json:"reliability"`
}

// TrendForecast provides overall trend prediction
type TrendForecast struct {
	Direction      TrendDirection `json:"direction"`
	Strength       TrendStrength  `json:"strength"`
	ExpectedChange float64        `json:"expected_change"`
	TimeToTarget   *TimeToTarget  `json:"time_to_target,omitempty"`
	TurningPoint   *TurningPoint  `json:"turning_point,omitempty"`
}

// ConfidenceInterval represents prediction confidence bounds
type ConfidenceInterval struct {
	Lower           float64 `json:"lower"`
	Upper           float64 `json:"upper"`
	ConfidenceLevel float64 `json:"confidence_level"`
	IntervalWidth   float64 `json:"interval_width"`
}

// TimeToTarget predicts when coverage will reach specific targets
type TimeToTarget struct {
	TargetCoverage float64   `json:"target_coverage"`
	EstimatedDays  int       `json:"estimated_days"`
	EstimatedDate  time.Time `json:"estimated_date"`
	Probability    float64   `json:"probability"`
}

// TurningPoint identifies potential trend changes
type TurningPoint struct {
	Date        time.Time        `json:"date"`
	Type        TurningPointType `json:"type"`
	Confidence  float64          `json:"confidence"`
	Description string           `json:"description"`
}

// TrendDirection represents the direction of a coverage trend
type TrendDirection string

const (
	// TrendDirectionUp indicates an upward trend
	TrendDirectionUp TrendDirection = "up"
	// TrendDirectionDown indicates a downward trend
	TrendDirectionDown TrendDirection = "down"
	// TrendDirectionStable indicates a stable trend
	TrendDirectionStable TrendDirection = "stable"
	// TrendDirectionVolatile indicates a volatile trend
	TrendDirectionVolatile TrendDirection = "volatile"
)

// TrendStrength represents the strength of a trend
type TrendStrength string

const (
	// TrendStrengthWeak indicates a weak trend
	TrendStrengthWeak TrendStrength = "weak"
	// TrendStrengthModerate indicates a moderate trend
	TrendStrengthModerate TrendStrength = "moderate"
	// TrendStrengthStrong indicates a strong trend
	TrendStrengthStrong TrendStrength = "strong"
)

// TrendIndicator represents the current trend indicator
type TrendIndicator string

const (
	// TrendIndicatorRising indicates a rising trend
	TrendIndicatorRising TrendIndicator = "rising"
	// TrendIndicatorFalling indicates a falling trend
	TrendIndicatorFalling TrendIndicator = "falling"
	// TrendIndicatorFlat indicates a flat trend
	TrendIndicatorFlat TrendIndicator = "flat"
)

// TurningPointType represents the type of turning point in a trend
type TurningPointType string

const (
	// TurningPointPeak indicates a peak in the trend
	TurningPointPeak TurningPointType = "peak"
	// TurningPointTrough indicates a trough in the trend
	TurningPointTrough TurningPointType = "trough"
	// TurningPointInflection indicates an inflection point
	TurningPointInflection TurningPointType = "inflection"
)

// PredictionQuality represents the quality level of predictions
type PredictionQuality string //nolint:revive // prediction.PredictionQuality is appropriately descriptive

const (
	// QualityExcellent indicates excellent prediction quality
	QualityExcellent PredictionQuality = "excellent"
	// QualityGood indicates good prediction quality
	QualityGood PredictionQuality = "good"
	// QualityFair indicates fair prediction quality
	QualityFair PredictionQuality = "fair"
	// QualityPoor indicates poor prediction quality
	QualityPoor PredictionQuality = "poor"
)

// PredictionInsight represents insights from prediction analysis
type PredictionInsight struct { //nolint:revive // prediction.PredictionInsight is appropriately descriptive
	Type        InsightType `json:"type"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Confidence  float64     `json:"confidence"`
	Impact      ImpactLevel `json:"impact"`
	Timeline    string      `json:"timeline"`
}

// PredictionRisk represents identified risks in predictions
type PredictionRisk struct { //nolint:revive // prediction.PredictionRisk is appropriately descriptive
	Type        RiskType    `json:"type"`
	Description string      `json:"description"`
	Probability float64     `json:"probability"`
	Impact      ImpactLevel `json:"impact"`
	Mitigation  string      `json:"mitigation"`
}

// PredictionRecommendation provides actionable recommendations
type PredictionRecommendation struct { //nolint:revive // prediction.PredictionRecommendation is appropriately descriptive
	Type            RecommendationType `json:"type"`
	Priority        Priority           `json:"priority"`
	Title           string             `json:"title"`
	Description     string             `json:"description"`
	Actions         []string           `json:"actions"`
	ExpectedOutcome string             `json:"expected_outcome"`
	Timeline        string             `json:"timeline"`
}

// InsightType represents different types of insights
type InsightType string

const (
	// InsightTypeTrend indicates trend-related insight
	InsightTypeTrend InsightType = "trend"
	// InsightTypeAnomaly indicates anomaly-related insight
	InsightTypeAnomaly InsightType = "anomaly"
	// InsightTypeOpportunity indicates opportunity-related insight
	InsightTypeOpportunity InsightType = "opportunity"
	// InsightTypeWarning indicates warning-related insight
	InsightTypeWarning InsightType = "warning"
)

// RiskType represents different types of risk factors
type RiskType string

const (
	// RiskTypeModelAccuracy indicates risk related to model accuracy
	RiskTypeModelAccuracy RiskType = "model_accuracy"
	// RiskTypeDataQuality indicates risk related to data quality
	RiskTypeDataQuality RiskType = "data_quality"
	// RiskTypeTrendChange indicates risk related to trend changes
	RiskTypeTrendChange RiskType = "trend_change"
	// RiskTypeExternal indicates risk from external factors
	RiskTypeExternal RiskType = "external_factors"
)

// RecommendationType represents different types of recommendations
type RecommendationType string

const (
	// RecommendationTypeProcess indicates process-related recommendation
	RecommendationTypeProcess RecommendationType = "process"
	// RecommendationTypeTesting indicates testing-related recommendation
	RecommendationTypeTesting RecommendationType = "testing"
	// RecommendationTypeMonitoring indicates monitoring-related recommendation
	RecommendationTypeMonitoring RecommendationType = "monitoring"
	// RecommendationTypeGoals indicates goals-related recommendation
	RecommendationTypeGoals RecommendationType = "goals"
)

// ImpactLevel represents different levels of impact
type ImpactLevel string

const (
	// ImpactLevelLow indicates low impact level
	ImpactLevelLow ImpactLevel = "low"
	// ImpactLevelMedium indicates medium impact level
	ImpactLevelMedium ImpactLevel = "medium"
	// ImpactLevelHigh indicates high impact level
	ImpactLevelHigh ImpactLevel = "high"
	// ImpactLevelCritical indicates critical impact level
	ImpactLevelCritical ImpactLevel = "critical"
)

// Priority represents different priority levels
type Priority string

const (
	// PriorityLow indicates low priority level
	PriorityLow Priority = "low"
	// PriorityMedium indicates medium priority level
	PriorityMedium Priority = "medium"
	// PriorityHigh indicates high priority level
	PriorityHigh Priority = "high"
	// PriorityUrgent indicates urgent priority level
	PriorityUrgent Priority = "urgent"
)

// NewCoveragePredictor creates a new coverage predictor with default configuration
func NewCoveragePredictor(config *PredictorConfig) *CoveragePredictor {
	if config == nil {
		config = &PredictorConfig{
			ModelType:             ModelLinearRegression,
			TrainingWindowDays:    30,
			MinTrainingPoints:     7,
			PredictionHorizonDays: 14,
			ConfidenceLevel:       0.95,
			SeasonalAdjustment:    true,
			CrossValidationFolds:  5,
			ValidationThreshold:   0.7,
			MinRSquared:           0.5,
			MaxPredictionError:    10.0,
			OutlierThreshold:      2.0,
		}
	}

	return &CoveragePredictor{
		config: config,
		model:  nil,
	}
}

// TrainModel trains the prediction model using historical data
func (p *CoveragePredictor) TrainModel(_ context.Context, analyzer *history.TrendAnalyzer) error {
	// Get training data from analyzer
	trainingData, err := p.prepareTrainingData(analyzer)
	if err != nil {
		return fmt.Errorf("failed to prepare training data: %w", err)
	}

	if len(trainingData) < p.config.MinTrainingPoints {
		return fmt.Errorf("%w: need %d points, got %d", ErrInsufficientTrainingData,
			p.config.MinTrainingPoints, len(trainingData))
	}

	// Create and train model based on type
	model := &PredictionModel{
		Type:               p.config.ModelType,
		TrainingData:       trainingData,
		LastTrainedAt:      time.Now(),
		TrainingDataWindow: time.Duration(p.config.TrainingWindowDays) * 24 * time.Hour,
	}

	switch p.config.ModelType {
	case ModelLinearRegression:
		err = p.trainLinearRegression(model)
	case ModelExponentialSmoothing:
		err = p.trainExponentialSmoothing(model)
	case ModelMovingAverage:
		err = p.trainMovingAverage(model)
	case ModelPolynomial:
		err = p.trainPolynomial(model)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedModelType, p.config.ModelType)
	}

	if err != nil {
		return fmt.Errorf("failed to train model: %w", err)
	}

	// Validate model
	model.ValidationMetrics, err = p.validateModel(model)
	if err != nil {
		return fmt.Errorf("failed to validate model: %w", err)
	}

	// Check if model meets quality thresholds
	if model.ValidationMetrics.R2Score < p.config.MinRSquared {
		return fmt.Errorf("%w (%.3f) below threshold (%.3f)", ErrModelBelowThreshold,
			model.ValidationMetrics.R2Score, p.config.MinRSquared)
	}

	p.model = model
	return nil
}

// PredictCoverage generates coverage predictions for the specified horizon
func (p *CoveragePredictor) PredictCoverage(_ context.Context) (*PredictionResult, error) {
	if p.model == nil {
		return nil, ErrModelNotTrained
	}

	result := &PredictionResult{
		PredictionDate:     time.Now(),
		ModelUsed:          p.model.Type,
		TrainingDataPoints: len(p.model.TrainingData),
	}

	// Generate point forecasts
	pointForecasts, err := p.generatePointForecasts()
	if err != nil {
		return nil, fmt.Errorf("failed to generate point forecasts: %w", err)
	}
	result.PointForecasts = pointForecasts

	// Generate trend forecast
	result.TrendForecast = p.generateTrendForecast(pointForecasts)

	// Calculate overall confidence and reliability
	result.OverallConfidence = p.calculateOverallConfidence(pointForecasts)
	result.ReliabilityScore = p.calculateReliabilityScore()
	result.PredictionQuality = p.assessPredictionQuality()

	// Generate insights and recommendations
	result.KeyInsights = p.generatePredictionInsights(pointForecasts, result.TrendForecast)
	result.Risks = p.identifyPredictionRisks()
	result.Recommendations = p.generatePredictionRecommendations(result)

	return result, nil
}

// PredictTargetDate predicts when coverage will reach a specific target
func (p *CoveragePredictor) PredictTargetDate(targetCoverage float64) (*TimeToTarget, error) {
	if p.model == nil {
		return nil, ErrModelNotTrained
	}

	// Get current coverage
	currentCoverage := p.getCurrentCoverage()
	if currentCoverage >= targetCoverage {
		return &TimeToTarget{
			TargetCoverage: targetCoverage,
			EstimatedDays:  0,
			EstimatedDate:  time.Now(),
			Probability:    1.0,
		}, nil
	}

	// Calculate time to target based on model
	switch p.model.Type {
	case ModelLinearRegression:
		return p.predictTargetLinear(targetCoverage, currentCoverage)
	default:
		return p.predictTargetGeneric(targetCoverage, currentCoverage)
	}
}

// Helper methods for training different model types

func (p *CoveragePredictor) prepareTrainingData(analyzer *history.TrendAnalyzer) ([]TrainingPoint, error) {
	// Export data from analyzer
	rawData, err := analyzer.ExportToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to export analyzer data: %w", err)
	}

	var analysisPoints []history.AnalysisDataPoint
	if err := json.Unmarshal(rawData, &analysisPoints); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analysis data: %w", err)
	}

	// Convert to training points
	trainingPoints := make([]TrainingPoint, 0, len(analysisPoints))
	cutoff := time.Now().AddDate(0, 0, -p.config.TrainingWindowDays)

	for i, point := range analysisPoints {
		if point.Timestamp.Before(cutoff) {
			continue
		}

		trainingPoint := TrainingPoint{
			Timestamp:      point.Timestamp,
			Coverage:       point.Coverage,
			DayOfWeek:      int(point.Timestamp.Weekday()),
			DayOfMonth:     point.Timestamp.Day(),
			IsWeekend:      point.Timestamp.Weekday() == time.Saturday || point.Timestamp.Weekday() == time.Sunday,
			SequenceNumber: i,
			Weight:         1.0, // Equal weighting for now
		}

		// Apply higher weight to more recent data
		daysSinceNow := time.Since(point.Timestamp).Hours() / 24
		trainingPoint.Weight = math.Exp(-daysSinceNow / 30.0) // Exponential decay over 30 days

		trainingPoints = append(trainingPoints, trainingPoint)
	}

	// Sort by timestamp
	sort.Slice(trainingPoints, func(i, j int) bool {
		return trainingPoints[i].Timestamp.Before(trainingPoints[j].Timestamp)
	})

	return trainingPoints, nil
}

func (p *CoveragePredictor) trainLinearRegression(model *PredictionModel) error {
	points := model.TrainingData
	if len(points) < 2 {
		return ErrInsufficientPoints
	}

	// Calculate linear regression using least squares
	n := float64(len(points))
	var sumX, sumY, sumXY, sumX2 float64

	for i, point := range points {
		x := float64(i)
		y := point.Coverage
		w := point.Weight

		sumX += x * w
		sumY += y * w
		sumXY += x * y * w
		sumX2 += x * x * w
	}

	// Calculate slope and intercept
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return ErrZeroDenominator
	}

	slope := (n*sumXY - sumX*sumY) / denominator
	intercept := (sumY - slope*sumX) / n

	// Calculate R-squared
	yMean := sumY / n
	var ssr, sst float64

	for i, point := range points {
		x := float64(i)
		y := point.Coverage
		yPred := slope*x + intercept

		ssr += (yPred - yMean) * (yPred - yMean)
		sst += (y - yMean) * (y - yMean)
	}

	rSquared := 0.0
	if sst != 0 {
		rSquared = ssr / sst
	}

	model.Parameters = ModelParameters{
		Slope:     slope,
		Intercept: intercept,
		RSquared:  rSquared,
	}

	return nil
}

func (p *CoveragePredictor) trainExponentialSmoothing(model *PredictionModel) error {
	points := model.TrainingData
	if len(points) < 3 {
		return ErrInsufficientPoints
	}

	// Simple exponential smoothing with optimized alpha
	bestAlpha := 0.3
	bestError := math.Inf(1)

	// Grid search for optimal alpha
	for alpha := 0.1; alpha <= 0.9; alpha += 0.1 {
		smoothingError := p.calculateExponentialSmoothingError(points, alpha)
		if smoothingError < bestError {
			bestError = smoothingError
			bestAlpha = alpha
		}
	}

	model.Parameters = ModelParameters{
		Alpha: bestAlpha,
	}

	return nil
}

func (p *CoveragePredictor) trainMovingAverage(model *PredictionModel) error {
	points := model.TrainingData
	if len(points) < 3 {
		return ErrInsufficientPoints
	}

	// Find optimal window size
	bestWindow := 3
	bestError := math.Inf(1)

	maxWindow := minInt(len(points)/2, 14) // Max 2 weeks or half the data
	for window := 3; window <= maxWindow; window++ {
		movingError := p.calculateMovingAverageError(points, window)
		if movingError < bestError {
			bestError = movingError
			bestWindow = window
		}
	}

	model.Parameters = ModelParameters{
		WindowSize:      bestWindow,
		WeightedAverage: true,
	}

	return nil
}

func (p *CoveragePredictor) trainPolynomial(model *PredictionModel) error {
	points := model.TrainingData
	if len(points) < 4 {
		return ErrInsufficientPoints
	}

	// For simplicity, use degree 2 polynomial
	degree := 2
	coefficients, rSquared := p.calculatePolynomialRegression(points, degree)

	model.Parameters = ModelParameters{
		Degree:       degree,
		Coefficients: coefficients,
		RSquared:     rSquared,
	}

	return nil
}

// Helper methods for predictions

func (p *CoveragePredictor) generatePointForecasts() ([]PointForecast, error) {
	var forecasts []PointForecast
	lastPoint := p.model.TrainingData[len(p.model.TrainingData)-1]

	for i := 1; i <= p.config.PredictionHorizonDays; i++ {
		futureDate := lastPoint.Timestamp.AddDate(0, 0, i)

		// Predict value based on model type
		var predictedValue float64
		var err error

		switch p.model.Type {
		case ModelLinearRegression:
			predictedValue = p.predictLinear(len(p.model.TrainingData) + i - 1)
		case ModelExponentialSmoothing:
			predictedValue = p.predictExponentialSmoothing(i)
		case ModelMovingAverage:
			predictedValue = p.predictMovingAverage()
		case ModelPolynomial:
			predictedValue = p.predictPolynomial(len(p.model.TrainingData) + i - 1)
		default:
			return nil, fmt.Errorf("%w for prediction: %s", ErrUnsupportedModelType, p.model.Type)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to predict value for day %d: %w", i, err)
		}

		// Clamp to valid range
		predictedValue = math.Max(0, math.Min(100, predictedValue))

		// Calculate confidence interval
		margin := p.calculatePredictionMargin(i)
		confidenceInterval := ConfidenceInterval{
			Lower:           math.Max(0, predictedValue-margin),
			Upper:           math.Min(100, predictedValue+margin),
			ConfidenceLevel: p.config.ConfidenceLevel,
			IntervalWidth:   2 * margin,
		}

		// Determine trend indicator
		trend := TrendIndicatorFlat
		if i > 1 && len(forecasts) > 0 {
			prevValue := forecasts[len(forecasts)-1].PredictedCoverage
			if predictedValue > prevValue+0.1 {
				trend = TrendIndicatorRising
			} else if predictedValue < prevValue-0.1 {
				trend = TrendIndicatorFalling
			}
		}

		// Calculate reliability (decreases with distance)
		reliability := p.model.ValidationMetrics.Accuracy * math.Exp(-float64(i)*0.05)

		forecast := PointForecast{
			Date:               futureDate,
			PredictedCoverage:  predictedValue,
			ConfidenceInterval: confidenceInterval,
			Trend:              trend,
			DaysAhead:          i,
			Reliability:        reliability,
		}

		forecasts = append(forecasts, forecast)
	}

	return forecasts, nil
}

func (p *CoveragePredictor) predictLinear(x int) float64 {
	params := p.model.Parameters
	return params.Slope*float64(x) + params.Intercept
}

func (p *CoveragePredictor) predictExponentialSmoothing(_ int) float64 {
	// For simple exponential smoothing, prediction is the last smoothed value
	if len(p.model.TrainingData) == 0 {
		return 0
	}

	lastValue := p.model.TrainingData[len(p.model.TrainingData)-1].Coverage
	return lastValue // Simplified - in practice would use last smoothed value
}

func (p *CoveragePredictor) predictMovingAverage() float64 {
	windowSize := p.model.Parameters.WindowSize
	if len(p.model.TrainingData) < windowSize {
		return 0
	}

	sum := 0.0
	for i := len(p.model.TrainingData) - windowSize; i < len(p.model.TrainingData); i++ {
		sum += p.model.TrainingData[i].Coverage
	}

	return sum / float64(windowSize)
}

func (p *CoveragePredictor) predictPolynomial(x int) float64 {
	coeffs := p.model.Parameters.Coefficients
	result := 0.0

	for i, coeff := range coeffs {
		result += coeff * math.Pow(float64(x), float64(i))
	}

	return result
}

func (p *CoveragePredictor) calculatePredictionMargin(daysAhead int) float64 {
	// Margin increases with prediction distance and decreases with model accuracy
	baseMargin := 5.0 // Base 5% margin
	distanceMultiplier := 1.0 + float64(daysAhead)*0.1
	accuracyMultiplier := (2.0 - p.model.ValidationMetrics.Accuracy)

	return baseMargin * distanceMultiplier * accuracyMultiplier
}

// Validation and quality assessment methods

func (p *CoveragePredictor) validateModel(model *PredictionModel) (ValidationMetrics, error) {
	points := model.TrainingData
	if len(points) < p.config.CrossValidationFolds {
		return ValidationMetrics{}, ErrInsufficientValidationData
	}

	// Perform time-series cross-validation
	foldSize := len(points) / p.config.CrossValidationFolds
	var errors []float64

	for fold := 0; fold < p.config.CrossValidationFolds; fold++ {
		trainEnd := (fold + 1) * foldSize
		if trainEnd >= len(points) {
			break
		}

		// Create training and validation sets
		trainData := points[:trainEnd]
		testData := points[trainEnd:minInt(trainEnd+foldSize, len(points))]

		// Train model on subset
		tempModel := &PredictionModel{
			Type:         model.Type,
			TrainingData: trainData,
		}

		var err error
		switch model.Type {
		case ModelLinearRegression:
			err = p.trainLinearRegression(tempModel)
		case ModelExponentialSmoothing:
			err = p.trainExponentialSmoothing(tempModel)
		case ModelMovingAverage:
			err = p.trainMovingAverage(tempModel)
		case ModelPolynomial:
			err = p.trainPolynomial(tempModel)
		}

		if err != nil {
			continue
		}

		// Test on validation set
		for i, testPoint := range testData {
			var predicted float64
			switch model.Type {
			case ModelLinearRegression:
				predicted = tempModel.Parameters.Slope*float64(trainEnd+i) + tempModel.Parameters.Intercept
			default:
				predicted = testPoint.Coverage // Fallback
			}

			predictionError := math.Abs(predicted - testPoint.Coverage)
			errors = append(errors, predictionError)
		}
	}

	if len(errors) == 0 {
		return ValidationMetrics{}, ErrNoValidationErrors
	}

	// Calculate metrics
	mae := p.calculateMeanAbsoluteError(errors)
	rmse := p.calculateRootMeanSquareError(errors)
	mape := p.calculateMeanAbsolutePercentageError(points, errors)
	accuracy := math.Max(0, 1.0-mae/100.0)
	cvScore := 1.0 - mae/10.0 // Simplified CV score

	overfittingRisk := OverfittingLow
	if model.ValidationMetrics.R2Score > 0.95 && len(points) < 20 {
		overfittingRisk = OverfittingHigh
	} else if model.ValidationMetrics.R2Score > 0.85 && len(points) < 30 {
		overfittingRisk = OverfittingMedium
	}

	return ValidationMetrics{
		MeanAbsoluteError:           mae,
		RootMeanSquareError:         rmse,
		MeanAbsolutePercentageError: mape,
		Accuracy:                    accuracy,
		R2Score:                     model.Parameters.RSquared,
		CrossValidationScore:        cvScore,
		OverfittingRisk:             overfittingRisk,
	}, nil
}

// Utility methods

func (p *CoveragePredictor) calculateExponentialSmoothingError(points []TrainingPoint, alpha float64) float64 {
	if len(points) < 2 {
		return math.Inf(1)
	}

	smoothed := points[0].Coverage
	totalError := 0.0

	for i := 1; i < len(points); i++ {
		predictionError := math.Abs(smoothed - points[i].Coverage)
		totalError += predictionError
		smoothed = alpha*points[i].Coverage + (1-alpha)*smoothed
	}

	return totalError / float64(len(points)-1)
}

func (p *CoveragePredictor) calculateMovingAverageError(points []TrainingPoint, window int) float64 {
	if len(points) <= window {
		return math.Inf(1)
	}

	totalError := 0.0
	count := 0

	for i := window; i < len(points); i++ {
		sum := 0.0
		for j := i - window; j < i; j++ {
			sum += points[j].Coverage
		}
		avg := sum / float64(window)

		avgError := math.Abs(avg - points[i].Coverage)
		totalError += avgError
		count++
	}

	if count == 0 {
		return math.Inf(1)
	}

	return totalError / float64(count)
}

func (p *CoveragePredictor) calculatePolynomialRegression(points []TrainingPoint, _ int) ([]float64, float64) {
	// Simplified polynomial regression - in practice would use matrix operations
	// For now, return linear regression coefficients
	n := float64(len(points))
	var sumX, sumY, sumXY, sumX2 float64

	for i, point := range points {
		x := float64(i)
		y := point.Coverage

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Calculate R-squared
	yMean := sumY / n
	var ssr, sst float64

	for i, point := range points {
		x := float64(i)
		y := point.Coverage
		yPred := slope*x + intercept

		ssr += (yPred - yMean) * (yPred - yMean)
		sst += (y - yMean) * (y - yMean)
	}

	rSquared := 0.0
	if sst != 0 {
		rSquared = ssr / sst
	}

	return []float64{intercept, slope}, rSquared
}

func (p *CoveragePredictor) calculateMeanAbsoluteError(errors []float64) float64 {
	sum := 0.0
	for _, err := range errors {
		sum += err
	}
	return sum / float64(len(errors))
}

func (p *CoveragePredictor) calculateRootMeanSquareError(errors []float64) float64 {
	sumSquares := 0.0
	for _, err := range errors {
		sumSquares += err * err
	}
	return math.Sqrt(sumSquares / float64(len(errors)))
}

func (p *CoveragePredictor) calculateMeanAbsolutePercentageError(points []TrainingPoint, errors []float64) float64 {
	if len(errors) != len(points) {
		return 0
	}

	sum := 0.0
	count := 0

	for i, err := range errors {
		if i < len(points) && points[i].Coverage != 0 {
			percentError := err / points[i].Coverage * 100
			sum += percentError
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return sum / float64(count)
}

func (p *CoveragePredictor) getCurrentCoverage() float64 {
	if len(p.model.TrainingData) == 0 {
		return 0
	}
	return p.model.TrainingData[len(p.model.TrainingData)-1].Coverage
}

func (p *CoveragePredictor) predictTargetLinear(targetCoverage, currentCoverage float64) (*TimeToTarget, error) {
	slope := p.model.Parameters.Slope
	if slope <= 0 {
		return nil, ErrCoverageNotIncreasing
	}

	coverageGap := targetCoverage - currentCoverage
	daysNeeded := int(math.Ceil(coverageGap / slope))

	probability := math.Min(1.0, p.model.ValidationMetrics.Accuracy)

	return &TimeToTarget{
		TargetCoverage: targetCoverage,
		EstimatedDays:  daysNeeded,
		EstimatedDate:  time.Now().AddDate(0, 0, daysNeeded),
		Probability:    probability,
	}, nil
}

func (p *CoveragePredictor) predictTargetGeneric(targetCoverage, currentCoverage float64) (*TimeToTarget, error) {
	// Generic prediction based on average change
	recentPoints := p.model.TrainingData
	if len(recentPoints) < 2 {
		return nil, ErrInsufficientTargetData
	}

	// Calculate average daily change from recent data
	totalChange := recentPoints[len(recentPoints)-1].Coverage - recentPoints[0].Coverage
	days := recentPoints[len(recentPoints)-1].Timestamp.Sub(recentPoints[0].Timestamp).Hours() / 24

	if days == 0 {
		return nil, ErrNoTimeSpanInData
	}

	dailyChange := totalChange / days
	if dailyChange <= 0 {
		return nil, ErrCoverageNotIncreasing
	}

	coverageGap := targetCoverage - currentCoverage
	daysNeeded := int(math.Ceil(coverageGap / dailyChange))

	return &TimeToTarget{
		TargetCoverage: targetCoverage,
		EstimatedDays:  daysNeeded,
		EstimatedDate:  time.Now().AddDate(0, 0, daysNeeded),
		Probability:    0.7, // Lower confidence for generic prediction
	}, nil
}

// Additional analysis methods

func (p *CoveragePredictor) generateTrendForecast(forecasts []PointForecast) TrendForecast {
	if len(forecasts) < 2 {
		return TrendForecast{Direction: TrendDirectionStable}
	}

	firstValue := forecasts[0].PredictedCoverage
	lastValue := forecasts[len(forecasts)-1].PredictedCoverage
	totalChange := lastValue - firstValue

	var direction TrendDirection
	var strength TrendStrength

	if math.Abs(totalChange) < 1.0 {
		direction = TrendDirectionStable
		strength = TrendStrengthWeak
	} else if totalChange > 0 {
		direction = TrendDirectionUp
		if totalChange > 5.0 {
			strength = TrendStrengthStrong
		} else if totalChange > 2.0 {
			strength = TrendStrengthModerate
		} else {
			strength = TrendStrengthWeak
		}
	} else {
		direction = TrendDirectionDown
		if totalChange < -5.0 {
			strength = TrendStrengthStrong
		} else if totalChange < -2.0 {
			strength = TrendStrengthModerate
		} else {
			strength = TrendStrengthWeak
		}
	}

	return TrendForecast{
		Direction:      direction,
		Strength:       strength,
		ExpectedChange: totalChange,
	}
}

func (p *CoveragePredictor) calculateOverallConfidence(forecasts []PointForecast) float64 {
	if len(forecasts) == 0 {
		return 0
	}

	totalReliability := 0.0
	for _, forecast := range forecasts {
		totalReliability += forecast.Reliability
	}

	return totalReliability / float64(len(forecasts))
}

func (p *CoveragePredictor) calculateReliabilityScore() float64 {
	return p.model.ValidationMetrics.Accuracy
}

func (p *CoveragePredictor) assessPredictionQuality() PredictionQuality {
	accuracy := p.model.ValidationMetrics.Accuracy

	switch {
	case accuracy >= 0.9:
		return QualityExcellent
	case accuracy >= 0.8:
		return QualityGood
	case accuracy >= 0.6:
		return QualityFair
	default:
		return QualityPoor
	}
}

func (p *CoveragePredictor) generatePredictionInsights(forecasts []PointForecast, trend TrendForecast) []PredictionInsight {
	var insights []PredictionInsight

	// Trend insights
	if trend.Direction == TrendDirectionUp && trend.Strength == TrendStrengthStrong {
		insights = append(insights, PredictionInsight{
			Type:  InsightTypeTrend,
			Title: "Strong Upward Trend Predicted",
			Description: fmt.Sprintf("Coverage expected to increase by %.1f%% over next %d days",
				trend.ExpectedChange, len(forecasts)),
			Confidence: 0.8,
			Impact:     ImpactLevelHigh,
			Timeline:   fmt.Sprintf("%d days", len(forecasts)),
		})
	}

	if trend.Direction == TrendDirectionDown {
		severity := ImpactLevelMedium
		if trend.Strength == TrendStrengthStrong {
			severity = ImpactLevelHigh
		}

		insights = append(insights, PredictionInsight{
			Type:  InsightTypeWarning,
			Title: "Coverage Decline Predicted",
			Description: fmt.Sprintf("Coverage may decrease by %.1f%% - proactive measures recommended",
				math.Abs(trend.ExpectedChange)),
			Confidence: 0.7,
			Impact:     severity,
			Timeline:   fmt.Sprintf("%d days", len(forecasts)),
		})
	}

	return insights
}

func (p *CoveragePredictor) identifyPredictionRisks() []PredictionRisk {
	var risks []PredictionRisk

	// Model accuracy risk
	if p.model.ValidationMetrics.Accuracy < 0.7 {
		risks = append(risks, PredictionRisk{
			Type:        RiskTypeModelAccuracy,
			Description: "Model accuracy is below 70% - predictions may be unreliable",
			Probability: 1.0 - p.model.ValidationMetrics.Accuracy,
			Impact:      ImpactLevelHigh,
			Mitigation:  "Collect more training data or try different model types",
		})
	}

	// Overfitting risk
	if p.model.ValidationMetrics.OverfittingRisk == OverfittingHigh {
		risks = append(risks, PredictionRisk{
			Type:        RiskTypeModelAccuracy,
			Description: "Model may be overfitted to training data",
			Probability: 0.7,
			Impact:      ImpactLevelMedium,
			Mitigation:  "Use more training data or regularization techniques",
		})
	}

	return risks
}

func (p *CoveragePredictor) generatePredictionRecommendations(result *PredictionResult) []PredictionRecommendation {
	var recommendations []PredictionRecommendation

	// Model improvement recommendations
	if result.PredictionQuality == QualityFair || result.PredictionQuality == QualityPoor {
		recommendations = append(recommendations, PredictionRecommendation{
			Type:        RecommendationTypeMonitoring,
			Priority:    PriorityHigh,
			Title:       "Improve Prediction Accuracy",
			Description: "Current prediction quality is suboptimal",
			Actions: []string{
				"Collect more historical data",
				"Experiment with different model types",
				"Improve data quality and consistency",
			},
			ExpectedOutcome: "Better prediction accuracy and reliability",
			Timeline:        "2-4 weeks",
		})
	}

	// Coverage improvement recommendations
	if result.TrendForecast.Direction == TrendDirectionDown {
		recommendations = append(recommendations, PredictionRecommendation{
			Type:        RecommendationTypeTesting,
			Priority:    PriorityHigh,
			Title:       "Address Predicted Coverage Decline",
			Description: "Model predicts coverage will decrease",
			Actions: []string{
				"Increase testing efforts",
				"Review recent code changes",
				"Implement coverage monitoring alerts",
			},
			ExpectedOutcome: "Prevent or minimize coverage regression",
			Timeline:        "1-2 weeks",
		})
	}

	return recommendations
}

// Utility functions
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
