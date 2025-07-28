// Package badge generates SVG coverage badges
package badge

import (
	"context"
	"fmt"
	"math"
	"strings"
)

// Generator creates professional SVG badges matching GitHub's design language
type Generator struct {
	config *Config
}

// Config holds badge generation configuration
type Config struct {
	Style           string
	Label           string
	Logo            string
	LogoColor       string
	ThresholdConfig ThresholdConfig
}

// ThresholdConfig defines coverage thresholds for color coding
type ThresholdConfig struct {
	Excellent  float64 // 90%+ - bright green
	Good       float64 // 80%+ - green
	Acceptable float64 // 70%+ - yellow
	Low        float64 // 60%+ - orange
	// Below Low = red
}

// Data represents data needed to generate a badge
type Data struct {
	Label     string
	Message   string
	Color     string
	Style     string
	Logo      string
	LogoColor string
	AriaLabel string
}

// TrendDirection represents coverage trend
type TrendDirection int

const (
	// TrendUp indicates coverage is trending upward
	TrendUp TrendDirection = iota
	// TrendDown indicates coverage is trending downward
	TrendDown
	// TrendStable indicates coverage is stable
	TrendStable
)

// New creates a new badge generator with default configuration
func New() *Generator {
	return &Generator{
		config: &Config{
			Style:     "flat",
			Label:     "coverage",
			Logo:      "",
			LogoColor: "white",
			ThresholdConfig: ThresholdConfig{
				Excellent:  90.0,
				Good:       80.0,
				Acceptable: 70.0,
				Low:        60.0,
			},
		},
	}
}

// NewWithConfig creates a new badge generator with custom configuration
func NewWithConfig(config *Config) *Generator {
	return &Generator{config: config}
}

// Generate creates an SVG badge for the given coverage percentage
func (g *Generator) Generate(ctx context.Context, percentage float64, options ...Option) ([]byte, error) {
	opts := &Options{
		Style:     g.config.Style,
		Label:     g.config.Label,
		Logo:      g.config.Logo,
		LogoColor: g.config.LogoColor,
	}

	// Apply options
	for _, opt := range options {
		opt(opts)
	}

	color := g.getColorForPercentage(percentage)
	message := fmt.Sprintf("%.1f%%", percentage)

	badgeData := Data{
		Label:     opts.Label,
		Message:   message,
		Color:     color,
		Style:     opts.Style,
		Logo:      opts.Logo,
		LogoColor: opts.LogoColor,
		AriaLabel: fmt.Sprintf("Code coverage: %.1f percent", percentage),
	}

	return g.renderSVG(ctx, badgeData)
}

// GenerateTrendBadge creates a badge showing coverage trend
func (g *Generator) GenerateTrendBadge(ctx context.Context, current, previous float64, options ...Option) ([]byte, error) {
	diff := current - previous
	var trend string
	var color string

	switch {
	case diff > 0.1:
		trend = fmt.Sprintf("↑ +%.1f%%", diff)
		color = g.getColorByName("excellent")
	case diff < -0.1:
		trend = fmt.Sprintf("↓ %.1f%%", diff)
		color = g.getColorByName("low")
	default:
		trend = "→ stable"
		color = "#8b949e" // neutral gray
	}

	opts := &Options{
		Style: g.config.Style,
		Label: "trend",
	}

	for _, opt := range options {
		opt(opts)
	}

	badgeData := Data{
		Label:     opts.Label,
		Message:   trend,
		Color:     color,
		Style:     opts.Style,
		AriaLabel: fmt.Sprintf("Coverage trend: %s", trend),
	}

	return g.renderSVG(ctx, badgeData)
}

// getColorForPercentage returns the appropriate color based on coverage percentage
func (g *Generator) getColorForPercentage(percentage float64) string {
	switch {
	case percentage >= g.config.ThresholdConfig.Excellent:
		return "#3fb950" // Bright green (GitHub green)
	case percentage >= g.config.ThresholdConfig.Good:
		return "#7c3aed" // Purple (GitHub purple)
	case percentage >= g.config.ThresholdConfig.Acceptable:
		return "#d29922" // Yellow (GitHub yellow)
	case percentage >= g.config.ThresholdConfig.Low:
		return "#fb8500" // Orange
	default:
		return "#f85149" // Red (GitHub red)
	}
}

// getColorByName returns color by threshold name
func (g *Generator) getColorByName(name string) string {
	switch name {
	case "excellent":
		return "#3fb950"
	case "good":
		return "#7c3aed"
	case "acceptable":
		return "#d29922"
	case "low":
		return "#fb8500"
	case "poor":
		return "#f85149"
	default:
		return "#8b949e" // neutral gray
	}
}

// renderSVG generates the actual SVG content
func (g *Generator) renderSVG(ctx context.Context, data Data) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Calculate dimensions
	labelWidth := g.calculateTextWidth(data.Label)
	messageWidth := g.calculateTextWidth(data.Message)
	logoWidth := 0
	if data.Logo != "" {
		logoWidth = 16 // Standard logo width
	}

	totalWidth := labelWidth + messageWidth + logoWidth + 20 // padding
	height := 20

	// Generate SVG based on style
	switch data.Style {
	case "flat-square":
		return g.renderFlatSquareBadge(data, totalWidth, height, labelWidth, messageWidth, logoWidth), nil
	case "for-the-badge":
		return g.renderForTheBadge(data, totalWidth, height+8, labelWidth, messageWidth, logoWidth), nil
	default: // flat
		return g.renderFlatBadge(data, totalWidth, labelWidth, messageWidth, logoWidth), nil
	}
}

// renderFlatBadge generates a flat-style badge
func (g *Generator) renderFlatBadge(data Data, width, labelWidth, messageWidth, logoWidth int) []byte {
	height := 20
	template := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d" role="img" aria-label="%s">
  <title>%s</title>
  <linearGradient id="s" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r">
    <rect width="%d" height="%d" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#r)">
    <rect width="%d" height="%d" fill="#555"/>
    <rect x="%d" width="%d" height="%d" fill="%s"/>
    <rect width="%d" height="%d" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="11">
    %s
    <text aria-hidden="true" x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="14">%s</text>
    <text aria-hidden="true" x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="14">%s</text>
  </g>
</svg>`

	labelX := logoWidth + labelWidth/2 + 6
	messageX := logoWidth + labelWidth + messageWidth/2 + 8
	logoSvg := ""

	if data.Logo != "" {
		logoSvg = fmt.Sprintf(`<image x="5" y="3" width="14" height="14" xlink:href="%s"/>`, data.Logo)
	}

	return []byte(fmt.Sprintf(template,
		width, height, data.AriaLabel, data.AriaLabel,
		width, height,
		logoWidth+labelWidth+8, height,
		logoWidth+labelWidth+8, messageWidth+8, height, data.Color,
		width, height,
		logoSvg,
		labelX, data.Label,
		labelX, data.Label,
		messageX, data.Message,
		messageX, data.Message,
	))
}

// renderFlatSquareBadge generates a flat-square style badge
func (g *Generator) renderFlatSquareBadge(data Data, width, height, labelWidth, messageWidth, logoWidth int) []byte {
	template := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d" role="img" aria-label="%s">
  <title>%s</title>
  <g shape-rendering="crispEdges">
    <rect width="%d" height="%d" fill="#555"/>
    <rect x="%d" width="%d" height="%d" fill="%s"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="11">
    %s
    <text x="%d" y="15">%s</text>
    <text x="%d" y="15">%s</text>
  </g>
</svg>`

	labelX := logoWidth + labelWidth/2 + 6
	messageX := logoWidth + labelWidth + messageWidth/2 + 8
	logoSvg := ""

	if data.Logo != "" {
		logoSvg = fmt.Sprintf(`<image x="5" y="3" width="14" height="14" xlink:href="%s"/>`, data.Logo)
	}

	return []byte(fmt.Sprintf(template,
		width, height, data.AriaLabel, data.AriaLabel,
		logoWidth+labelWidth+8, height,
		logoWidth+labelWidth+8, messageWidth+8, height, data.Color,
		logoSvg,
		labelX, data.Label,
		messageX, data.Message,
	))
}

// renderForTheBadge generates a "for-the-badge" style badge
func (g *Generator) renderForTheBadge(data Data, width, height, labelWidth, messageWidth, logoWidth int) []byte {
	template := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d" role="img" aria-label="%s">
  <title>%s</title>
  <g shape-rendering="crispEdges">
    <rect width="%d" height="%d" fill="#555"/>
    <rect x="%d" width="%d" height="%d" fill="%s"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="11" font-weight="bold">
    %s
    <text x="%d" y="19">%s</text>
    <text x="%d" y="19">%s</text>
  </g>
</svg>`

	labelX := logoWidth + labelWidth/2 + 6
	messageX := logoWidth + labelWidth + messageWidth/2 + 8
	logoSvg := ""

	if data.Logo != "" {
		logoSvg = fmt.Sprintf(`<image x="5" y="6" width="16" height="16" xlink:href="%s"/>`, data.Logo)
	}

	// Convert to uppercase for "for-the-badge" style
	label := strings.ToUpper(data.Label)
	message := strings.ToUpper(data.Message)

	return []byte(fmt.Sprintf(template,
		width, height, data.AriaLabel, data.AriaLabel,
		logoWidth+labelWidth+8, height,
		logoWidth+labelWidth+8, messageWidth+8, height, data.Color,
		logoSvg,
		labelX, label,
		messageX, message,
	))
}

// calculateTextWidth estimates text width (simplified calculation)
func (g *Generator) calculateTextWidth(text string) int {
	// Rough estimation: average character width ~6.5px for Verdana 11px
	return int(math.Ceil(float64(len(text)) * 6.5))
}

// Options represents options for badge generation
type Options struct {
	Style     string
	Label     string
	Logo      string
	LogoColor string
}

// Option is a function type for configuring badge options
type Option func(*Options)

// WithStyle sets the badge style
func WithStyle(style string) Option {
	return func(opts *Options) {
		opts.Style = style
	}
}

// WithLabel sets the badge label
func WithLabel(label string) Option {
	return func(opts *Options) {
		opts.Label = label
	}
}

// WithLogo sets the badge logo
func WithLogo(logo string) Option {
	return func(opts *Options) {
		opts.Logo = logo
	}
}

// WithLogoColor sets the logo color
func WithLogoColor(color string) Option {
	return func(opts *Options) {
		opts.LogoColor = color
	}
}
