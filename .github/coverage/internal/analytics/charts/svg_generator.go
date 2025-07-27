// Package charts provides server-side SVG chart generation for coverage analytics
package charts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Static error definitions
var (
	ErrNoDataPoints = errors.New("no data points provided")
	ErrNoDataSeries = errors.New("no data series provided")
)

// SVGChartGenerator handles server-side SVG chart generation without JavaScript dependencies
type SVGChartGenerator struct {
	config *ChartConfig
}

// ChartConfig holds configuration for SVG chart generation
type ChartConfig struct {
	// Dimensions
	Width   int // Chart width in pixels
	Height  int // Chart height in pixels
	Padding int // Padding around chart area

	// Visual styling
	BackgroundColor string // Background color
	GridColor       string // Grid line color
	LineColor       string // Primary line color
	FillColor       string // Area fill color
	TextColor       string // Text color
	FontFamily      string // Font family for text
	FontSize        int    // Font size in pixels

	// Grid and axes
	ShowGrid      bool    // Show grid lines
	ShowXAxis     bool    // Show X axis
	ShowYAxis     bool    // Show Y axis
	ShowLegend    bool    // Show legend
	GridLineWidth float64 // Grid line width
	LineWidth     float64 // Data line width

	// Data formatting
	TimeFormat    string // Time format for X axis labels
	PercentFormat string // Format for percentage values
	DecimalPlaces int    // Decimal places for values

	// Interactive features
	ShowTooltips   bool // Enable hover tooltips
	ShowDataPoints bool // Show individual data points
	Responsive     bool // Make chart responsive
}

// ChartData represents time-series data for chart generation
type ChartData struct {
	Points     []DataPoint `json:"points"`
	Title      string      `json:"title"`
	XAxisLabel string      `json:"x_axis_label"`
	YAxisLabel string      `json:"y_axis_label"`
	Series     []Series    `json:"series"`
	TimeRange  TimeRange   `json:"time_range"`
}

// DataPoint represents a single data point in time series
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
	Metadata  Metadata  `json:"metadata,omitempty"`
}

// Series represents a data series for multi-line charts
type Series struct {
	Name   string      `json:"name"`
	Color  string      `json:"color"`
	Points []DataPoint `json:"points"`
	Type   SeriesType  `json:"type"`
}

// SeriesType defines the type of chart series
type SeriesType string

const (
	SeriesLine    SeriesType = "line"
	SeriesArea    SeriesType = "area"
	SeriesBar     SeriesType = "bar"
	SeriesScatter SeriesType = "scatter"
)

// TimeRange represents a time range for the chart
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Metadata contains additional information about data points
type Metadata struct {
	Branch    string  `json:"branch,omitempty"`
	CommitSHA string  `json:"commit_sha,omitempty"`
	PRNumber  int     `json:"pr_number,omitempty"`
	Coverage  float64 `json:"coverage,omitempty"`
	Author    string  `json:"author,omitempty"`
}

// ChartType represents different types of charts
type ChartType string

const (
	ChartTrendLine   ChartType = "trend_line"
	ChartAreaChart   ChartType = "area_chart"
	ChartBarChart    ChartType = "bar_chart"
	ChartMultiSeries ChartType = "multi_series"
	ChartHeatmap     ChartType = "heatmap"
)

// NewSVGChartGenerator creates a new SVG chart generator with default configuration
func NewSVGChartGenerator(config *ChartConfig) *SVGChartGenerator {
	if config == nil {
		config = &ChartConfig{
			Width:           800,
			Height:          400,
			Padding:         40,
			BackgroundColor: "#ffffff",
			GridColor:       "#e1e4e8",
			LineColor:       "#0366d6",
			FillColor:       "rgba(3, 102, 214, 0.1)",
			TextColor:       "#24292e",
			FontFamily:      "system-ui, -apple-system, sans-serif",
			FontSize:        12,
			ShowGrid:        true,
			ShowXAxis:       true,
			ShowYAxis:       true,
			ShowLegend:      true,
			GridLineWidth:   0.5,
			LineWidth:       2.0,
			TimeFormat:      "Jan 02",
			PercentFormat:   "%.1f%%",
			DecimalPlaces:   1,
			ShowTooltips:    true,
			ShowDataPoints:  false,
			Responsive:      true,
		}
	}

	return &SVGChartGenerator{
		config: config,
	}
}

// GenerateTrendChart creates a line chart showing coverage trends over time
func (g *SVGChartGenerator) GenerateTrendChart(ctx context.Context, data *ChartData) (string, error) {
	if len(data.Points) == 0 {
		return "", ErrNoDataPoints
	}

	// Calculate chart dimensions
	chartArea := g.calculateChartArea()

	// Calculate scales
	xScale, yScale := g.calculateScales(data, chartArea)

	// Generate SVG
	var svg strings.Builder

	// SVG header
	g.writeSVGHeader(&svg)

	// Background
	g.writeBackground(&svg)

	// Grid
	if g.config.ShowGrid {
		g.writeGrid(&svg, chartArea, xScale, yScale, data)
	}

	// Axes
	if g.config.ShowXAxis || g.config.ShowYAxis {
		g.writeAxes(&svg, chartArea, xScale, yScale, data)
	}

	// Data line
	g.writeTrendLine(&svg, data.Points, chartArea, xScale, yScale)

	// Data points
	if g.config.ShowDataPoints {
		g.writeDataPoints(&svg, data.Points, chartArea, xScale, yScale)
	}

	// Legend
	if g.config.ShowLegend {
		g.writeLegend(&svg, data)
	}

	// Title
	if data.Title != "" {
		g.writeTitle(&svg, data.Title)
	}

	// Interactive features
	if g.config.ShowTooltips {
		g.writeTooltipScript(&svg)
	}

	// SVG footer
	g.writeSVGFooter(&svg)

	return svg.String(), nil
}

// GenerateAreaChart creates an area chart with fill
func (g *SVGChartGenerator) GenerateAreaChart(ctx context.Context, data *ChartData) (string, error) {
	if len(data.Points) == 0 {
		return "", ErrNoDataPoints
	}

	chartArea := g.calculateChartArea()
	xScale, yScale := g.calculateScales(data, chartArea)

	var svg strings.Builder

	g.writeSVGHeader(&svg)
	g.writeBackground(&svg)

	if g.config.ShowGrid {
		g.writeGrid(&svg, chartArea, xScale, yScale, data)
	}

	if g.config.ShowXAxis || g.config.ShowYAxis {
		g.writeAxes(&svg, chartArea, xScale, yScale, data)
	}

	// Area fill
	g.writeAreaFill(&svg, data.Points, chartArea, xScale, yScale)

	// Line on top of area
	g.writeTrendLine(&svg, data.Points, chartArea, xScale, yScale)

	if g.config.ShowLegend {
		g.writeLegend(&svg, data)
	}

	if data.Title != "" {
		g.writeTitle(&svg, data.Title)
	}

	if g.config.ShowTooltips {
		g.writeTooltipScript(&svg)
	}

	g.writeSVGFooter(&svg)

	return svg.String(), nil
}

// GenerateMultiSeriesChart creates a chart with multiple data series
func (g *SVGChartGenerator) GenerateMultiSeriesChart(ctx context.Context, data *ChartData) (string, error) {
	if len(data.Series) == 0 {
		return "", ErrNoDataSeries
	}

	chartArea := g.calculateChartArea()
	xScale, yScale := g.calculateScalesForSeries(data, chartArea)

	var svg strings.Builder

	g.writeSVGHeader(&svg)
	g.writeBackground(&svg)

	if g.config.ShowGrid {
		g.writeGrid(&svg, chartArea, xScale, yScale, data)
	}

	if g.config.ShowXAxis || g.config.ShowYAxis {
		g.writeAxes(&svg, chartArea, xScale, yScale, data)
	}

	// Draw each series
	for _, series := range data.Series {
		g.writeSeriesLine(&svg, series, chartArea, xScale, yScale)
	}

	if g.config.ShowLegend {
		g.writeMultiSeriesLegend(&svg, data)
	}

	if data.Title != "" {
		g.writeTitle(&svg, data.Title)
	}

	if g.config.ShowTooltips {
		g.writeTooltipScript(&svg)
	}

	g.writeSVGFooter(&svg)

	return svg.String(), nil
}

// Helper methods for SVG generation

// ChartArea represents the drawable area within the SVG
type ChartArea struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Scale represents a scale for mapping data to coordinates
type Scale struct {
	Min    float64
	Max    float64
	Range  float64
	Factor float64
}

func (g *SVGChartGenerator) calculateChartArea() ChartArea {
	return ChartArea{
		X:      g.config.Padding,
		Y:      g.config.Padding,
		Width:  g.config.Width - (2 * g.config.Padding),
		Height: g.config.Height - (2 * g.config.Padding),
	}
}

func (g *SVGChartGenerator) calculateScales(data *ChartData, chartArea ChartArea) (*Scale, *Scale) {
	// X scale (time)
	var minTime, maxTime time.Time
	if len(data.Points) > 0 {
		minTime = data.Points[0].Timestamp
		maxTime = data.Points[0].Timestamp

		for _, point := range data.Points {
			if point.Timestamp.Before(minTime) {
				minTime = point.Timestamp
			}
			if point.Timestamp.After(maxTime) {
				maxTime = point.Timestamp
			}
		}
	}

	timeDiff := maxTime.Sub(minTime).Seconds()
	xScale := &Scale{
		Min:    0,
		Max:    timeDiff,
		Range:  timeDiff,
		Factor: float64(chartArea.Width) / timeDiff,
	}

	// Y scale (values)
	var minVal, maxVal float64
	if len(data.Points) > 0 {
		minVal = data.Points[0].Value
		maxVal = data.Points[0].Value

		for _, point := range data.Points {
			if point.Value < minVal {
				minVal = point.Value
			}
			if point.Value > maxVal {
				maxVal = point.Value
			}
		}
	}

	// Add some padding to Y scale
	valueRange := maxVal - minVal
	padding := valueRange * 0.1
	minVal -= padding
	maxVal += padding

	// Ensure we don't go below 0 for percentages
	if minVal < 0 && maxVal <= 100 {
		minVal = 0
	}

	yScale := &Scale{
		Min:    minVal,
		Max:    maxVal,
		Range:  maxVal - minVal,
		Factor: float64(chartArea.Height) / (maxVal - minVal),
	}

	return xScale, yScale
}

func (g *SVGChartGenerator) calculateScalesForSeries(data *ChartData, chartArea ChartArea) (*Scale, *Scale) {
	// Find global min/max across all series
	var minTime, maxTime time.Time
	var minVal, maxVal float64
	hasData := false

	for _, series := range data.Series {
		for _, point := range series.Points {
			if !hasData {
				minTime = point.Timestamp
				maxTime = point.Timestamp
				minVal = point.Value
				maxVal = point.Value
				hasData = true
				continue
			}

			if point.Timestamp.Before(minTime) {
				minTime = point.Timestamp
			}
			if point.Timestamp.After(maxTime) {
				maxTime = point.Timestamp
			}
			if point.Value < minVal {
				minVal = point.Value
			}
			if point.Value > maxVal {
				maxVal = point.Value
			}
		}
	}

	if !hasData {
		return &Scale{}, &Scale{}
	}

	timeDiff := maxTime.Sub(minTime).Seconds()
	xScale := &Scale{
		Min:    0,
		Max:    timeDiff,
		Range:  timeDiff,
		Factor: float64(chartArea.Width) / timeDiff,
	}

	valueRange := maxVal - minVal
	padding := valueRange * 0.1
	minVal -= padding
	maxVal += padding

	if minVal < 0 && maxVal <= 100 {
		minVal = 0
	}

	yScale := &Scale{
		Min:    minVal,
		Max:    maxVal,
		Range:  maxVal - minVal,
		Factor: float64(chartArea.Height) / (maxVal - minVal),
	}

	return xScale, yScale
}

func (g *SVGChartGenerator) writeSVGHeader(svg *strings.Builder) {
	if g.config.Responsive {
		fmt.Fprintf(svg, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" style="max-width: 100%%; height: auto;">`,
			g.config.Width, g.config.Height)
	} else {
		fmt.Fprintf(svg, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`,
			g.config.Width, g.config.Height)
	}
	svg.WriteString("\n")
}

func (g *SVGChartGenerator) writeSVGFooter(svg *strings.Builder) {
	svg.WriteString("</svg>\n")
}

func (g *SVGChartGenerator) writeBackground(svg *strings.Builder) {
	fmt.Fprintf(svg, `  <rect width="%d" height="%d" fill="%s"/>`,
		g.config.Width, g.config.Height, g.config.BackgroundColor)
	svg.WriteString("\n")
}

func (g *SVGChartGenerator) writeGrid(svg *strings.Builder, chartArea ChartArea, xScale, yScale *Scale, data *ChartData) {
	// Vertical grid lines (time)
	gridLines := 5
	for i := 0; i <= gridLines; i++ {
		x := chartArea.X + int(float64(i)*float64(chartArea.Width)/float64(gridLines))
		fmt.Fprintf(svg, `  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%.1f"/>`,
			x, chartArea.Y, x, chartArea.Y+chartArea.Height, g.config.GridColor, g.config.GridLineWidth)
		svg.WriteString("\n")
	}

	// Horizontal grid lines (values)
	for i := 0; i <= gridLines; i++ {
		y := chartArea.Y + int(float64(i)*float64(chartArea.Height)/float64(gridLines))
		fmt.Fprintf(svg, `  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%.1f"/>`,
			chartArea.X, y, chartArea.X+chartArea.Width, y, g.config.GridColor, g.config.GridLineWidth)
		svg.WriteString("\n")
	}
}

func (g *SVGChartGenerator) writeAxes(svg *strings.Builder, chartArea ChartArea, xScale, yScale *Scale, data *ChartData) {
	// X axis
	if g.config.ShowXAxis {
		fmt.Fprintf(svg, `  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1"/>`,
			chartArea.X, chartArea.Y+chartArea.Height, chartArea.X+chartArea.Width, chartArea.Y+chartArea.Height, g.config.TextColor)
		svg.WriteString("\n")

		// X axis labels
		g.writeXAxisLabels(svg, chartArea, xScale, data)
	}

	// Y axis
	if g.config.ShowYAxis {
		svg.WriteString(fmt.Sprintf(`  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1"/>`,
			chartArea.X, chartArea.Y, chartArea.X, chartArea.Y+chartArea.Height, g.config.TextColor))
		svg.WriteString("\n")

		// Y axis labels
		g.writeYAxisLabels(svg, chartArea, yScale)
	}
}

func (g *SVGChartGenerator) writeXAxisLabels(svg *strings.Builder, chartArea ChartArea, xScale *Scale, data *ChartData) {
	if len(data.Points) == 0 {
		return
	}

	labelCount := 5
	for i := 0; i <= labelCount; i++ {
		ratio := float64(i) / float64(labelCount)
		x := chartArea.X + int(ratio*float64(chartArea.Width))

		// Find corresponding time
		timeIndex := int(ratio * float64(len(data.Points)-1))
		if timeIndex >= len(data.Points) {
			timeIndex = len(data.Points) - 1
		}

		timeStr := data.Points[timeIndex].Timestamp.Format(g.config.TimeFormat)

		svg.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" text-anchor="middle" font-family="%s" font-size="%d" fill="%s">%s</text>`,
			x, chartArea.Y+chartArea.Height+15, g.config.FontFamily, g.config.FontSize, g.config.TextColor, timeStr))
		svg.WriteString("\n")
	}
}

func (g *SVGChartGenerator) writeYAxisLabels(svg *strings.Builder, chartArea ChartArea, yScale *Scale) {
	labelCount := 5
	for i := 0; i <= labelCount; i++ {
		ratio := float64(i) / float64(labelCount)
		y := chartArea.Y + chartArea.Height - int(ratio*float64(chartArea.Height))

		value := yScale.Min + (ratio * yScale.Range)
		valueStr := fmt.Sprintf(g.config.PercentFormat, value)

		svg.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" text-anchor="end" font-family="%s" font-size="%d" fill="%s">%s</text>`,
			chartArea.X-5, y+4, g.config.FontFamily, g.config.FontSize, g.config.TextColor, valueStr))
		svg.WriteString("\n")
	}
}

func (g *SVGChartGenerator) writeTrendLine(svg *strings.Builder, points []DataPoint, chartArea ChartArea, xScale, yScale *Scale) {
	if len(points) < 2 {
		return
	}

	var pathData strings.Builder

	for i, point := range points {
		x := g.mapTimeToX(point.Timestamp, points[0].Timestamp, xScale, chartArea)
		y := g.mapValueToY(point.Value, yScale, chartArea)

		if i == 0 {
			pathData.WriteString(fmt.Sprintf("M %d %d", x, y))
		} else {
			pathData.WriteString(fmt.Sprintf(" L %d %d", x, y))
		}
	}

	svg.WriteString(fmt.Sprintf(`  <path d="%s" stroke="%s" stroke-width="%.1f" fill="none"/>`,
		pathData.String(), g.config.LineColor, g.config.LineWidth))
	svg.WriteString("\n")
}

func (g *SVGChartGenerator) writeAreaFill(svg *strings.Builder, points []DataPoint, chartArea ChartArea, xScale, yScale *Scale) {
	if len(points) < 2 {
		return
	}

	var pathData strings.Builder

	// Start from bottom left
	firstX := g.mapTimeToX(points[0].Timestamp, points[0].Timestamp, xScale, chartArea)
	bottomY := chartArea.Y + chartArea.Height
	pathData.WriteString(fmt.Sprintf("M %d %d", firstX, bottomY))

	// Draw to first point
	firstY := g.mapValueToY(points[0].Value, yScale, chartArea)
	pathData.WriteString(fmt.Sprintf(" L %d %d", firstX, firstY))

	// Draw line through all points
	for _, point := range points[1:] {
		x := g.mapTimeToX(point.Timestamp, points[0].Timestamp, xScale, chartArea)
		y := g.mapValueToY(point.Value, yScale, chartArea)
		pathData.WriteString(fmt.Sprintf(" L %d %d", x, y))
	}

	// Close to bottom right
	lastX := g.mapTimeToX(points[len(points)-1].Timestamp, points[0].Timestamp, xScale, chartArea)
	pathData.WriteString(fmt.Sprintf(" L %d %d", lastX, bottomY))
	pathData.WriteString(" Z")

	svg.WriteString(fmt.Sprintf(`  <path d="%s" fill="%s"/>`,
		pathData.String(), g.config.FillColor))
	svg.WriteString("\n")
}

func (g *SVGChartGenerator) writeDataPoints(svg *strings.Builder, points []DataPoint, chartArea ChartArea, xScale, yScale *Scale) {
	for _, point := range points {
		x := g.mapTimeToX(point.Timestamp, points[0].Timestamp, xScale, chartArea)
		y := g.mapValueToY(point.Value, yScale, chartArea)

		svg.WriteString(fmt.Sprintf(`  <circle cx="%d" cy="%d" r="3" fill="%s"/>`,
			x, y, g.config.LineColor))
		svg.WriteString("\n")
	}
}

func (g *SVGChartGenerator) writeSeriesLine(svg *strings.Builder, series Series, chartArea ChartArea, xScale, yScale *Scale) {
	if len(series.Points) < 2 {
		return
	}

	var pathData strings.Builder

	for i, point := range series.Points {
		x := g.mapTimeToX(point.Timestamp, series.Points[0].Timestamp, xScale, chartArea)
		y := g.mapValueToY(point.Value, yScale, chartArea)

		if i == 0 {
			pathData.WriteString(fmt.Sprintf("M %d %d", x, y))
		} else {
			pathData.WriteString(fmt.Sprintf(" L %d %d", x, y))
		}
	}

	color := series.Color
	if color == "" {
		color = g.config.LineColor
	}

	svg.WriteString(fmt.Sprintf(`  <path d="%s" stroke="%s" stroke-width="%.1f" fill="none"/>`,
		pathData.String(), color, g.config.LineWidth))
	svg.WriteString("\n")
}

func (g *SVGChartGenerator) writeLegend(svg *strings.Builder, data *ChartData) {
	legendY := 20
	svg.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" font-family="%s" font-size="%d" fill="%s">Coverage Trend</text>`,
		g.config.Width-150, legendY, g.config.FontFamily, g.config.FontSize, g.config.TextColor))
	svg.WriteString("\n")
}

func (g *SVGChartGenerator) writeMultiSeriesLegend(svg *strings.Builder, data *ChartData) {
	legendX := g.config.Width - 150
	legendY := 20

	for i, series := range data.Series {
		y := legendY + (i * 20)

		// Legend color box
		svg.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="12" height="12" fill="%s"/>`,
			legendX, y-10, series.Color))
		svg.WriteString("\n")

		// Legend text
		svg.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" font-family="%s" font-size="%d" fill="%s">%s</text>`,
			legendX+18, y, g.config.FontFamily, g.config.FontSize, g.config.TextColor, series.Name))
		svg.WriteString("\n")
	}
}

func (g *SVGChartGenerator) writeTitle(svg *strings.Builder, title string) {
	titleY := 25
	svg.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" text-anchor="middle" font-family="%s" font-size="%d" font-weight="bold" fill="%s">%s</text>`,
		g.config.Width/2, titleY, g.config.FontFamily, g.config.FontSize+2, g.config.TextColor, title))
	svg.WriteString("\n")
}

func (g *SVGChartGenerator) writeTooltipScript(svg *strings.Builder) {
	// Basic tooltip functionality without external dependencies
	// This would be enhanced with actual tooltip implementation
	svg.WriteString(`  <!-- Tooltip functionality would be implemented here -->`)
	svg.WriteString("\n")
}

// Helper functions for coordinate mapping

func (g *SVGChartGenerator) mapTimeToX(timestamp, baseTime time.Time, xScale *Scale, chartArea ChartArea) int {
	seconds := timestamp.Sub(baseTime).Seconds()
	ratio := seconds / xScale.Range
	return chartArea.X + int(ratio*float64(chartArea.Width))
}

func (g *SVGChartGenerator) mapValueToY(value float64, yScale *Scale, chartArea ChartArea) int {
	ratio := (value - yScale.Min) / yScale.Range
	return chartArea.Y + chartArea.Height - int(ratio*float64(chartArea.Height))
}

// Utility functions for data processing

// FilterLastNDays filters data points to only include the last N days
func FilterLastNDays(points []DataPoint, days int) []DataPoint {
	if len(points) == 0 {
		return points
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var filtered []DataPoint

	for _, point := range points {
		if point.Timestamp.After(cutoff) {
			filtered = append(filtered, point)
		}
	}

	return filtered
}

// CalculateMovingAverage calculates moving average for the given window
func CalculateMovingAverage(points []DataPoint, windowSize int) []DataPoint {
	if len(points) < windowSize {
		return points
	}

	var averaged []DataPoint

	for i := windowSize - 1; i < len(points); i++ {
		sum := 0.0
		for j := i - windowSize + 1; j <= i; j++ {
			sum += points[j].Value
		}

		avgPoint := DataPoint{
			Timestamp: points[i].Timestamp,
			Value:     sum / float64(windowSize),
			Label:     fmt.Sprintf("%d-day avg", windowSize),
		}

		averaged = append(averaged, avgPoint)
	}

	return averaged
}
