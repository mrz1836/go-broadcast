package badge

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	generator := New()
	assert.NotNil(t, generator)
	assert.NotNil(t, generator.config)
	assert.Equal(t, "flat", generator.config.Style)
	assert.Equal(t, "coverage", generator.config.Label)
	assert.InDelta(t, 90.0, generator.config.ThresholdConfig.Excellent, 0.001)
	assert.InDelta(t, 80.0, generator.config.ThresholdConfig.Good, 0.001)
	assert.InDelta(t, 70.0, generator.config.ThresholdConfig.Acceptable, 0.001)
	assert.InDelta(t, 60.0, generator.config.ThresholdConfig.Low, 0.001)
}

func TestNewWithConfig(t *testing.T) {
	config := &Config{
		Style: "flat-square",
		Label: "test",
		ThresholdConfig: ThresholdConfig{
			Excellent:  95.0,
			Good:       85.0,
			Acceptable: 75.0,
			Low:        65.0,
		},
	}

	generator := NewWithConfig(config)
	assert.Equal(t, config, generator.config)
}

func TestGenerate(t *testing.T) {
	generator := New()
	ctx := context.Background()

	svg, err := generator.Generate(ctx, 85.5)
	require.NoError(t, err)
	assert.NotEmpty(t, svg)

	svgStr := string(svg)
	assert.Contains(t, svgStr, "<svg")
	assert.Contains(t, svgStr, "85.5%")
	assert.Contains(t, svgStr, "coverage")
	assert.Contains(t, svgStr, "</svg>")
	assert.Contains(t, svgStr, `role="img"`)
	assert.Contains(t, svgStr, `aria-label="Code coverage: 85.5 percent"`)
}

func TestGenerateWithOptions(t *testing.T) {
	generator := New()
	ctx := context.Background()

	svg, err := generator.Generate(ctx, 92.3,
		WithStyle("flat-square"),
		WithLabel("test coverage"),
		WithLogo("https://example.com/logo.svg"),
	)
	require.NoError(t, err)

	svgStr := string(svg)
	assert.Contains(t, svgStr, "92.3%")
	assert.Contains(t, svgStr, "test coverage")
	assert.Contains(t, svgStr, "https://example.com/logo.svg")
	assert.Contains(t, svgStr, `shape-rendering="crispEdges"`) // flat-square style
}

func TestGenerateTrendBadge(t *testing.T) {
	generator := New()
	ctx := context.Background()

	tests := []struct {
		name     string
		current  float64
		previous float64
		expected string
	}{
		{
			name:     "upward trend",
			current:  85.0,
			previous: 80.0,
			expected: "↑ +5.0%",
		},
		{
			name:     "downward trend",
			current:  75.0,
			previous: 82.0,
			expected: "↓ -7.0%",
		},
		{
			name:     "stable trend",
			current:  80.0,
			previous: 80.0,
			expected: "→ stable",
		},
		{
			name:     "small increase",
			current:  80.05,
			previous: 80.0,
			expected: "→ stable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg, err := generator.GenerateTrendBadge(ctx, tt.current, tt.previous)
			require.NoError(t, err)

			svgStr := string(svg)
			assert.Contains(t, svgStr, tt.expected)
			assert.Contains(t, svgStr, "trend")
		})
	}
}

func TestGetColorForPercentage(t *testing.T) {
	generator := New()

	tests := []struct {
		percentage float64
		expected   string
	}{
		{95.0, "#3fb950"},  // excellent (green)
		{85.0, "#7c3aed"},  // good (purple)
		{75.0, "#d29922"},  // acceptable (yellow)
		{65.0, "#fb8500"},  // low (orange)
		{55.0, "#f85149"},  // poor (red)
		{100.0, "#3fb950"}, // perfect (green)
		{0.0, "#f85149"},   // zero (red)
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%.1f%%", tt.percentage), func(t *testing.T) {
			color := generator.getColorForPercentage(tt.percentage)
			assert.Equal(t, tt.expected, color)
		})
	}
}

func TestGetColorByName(t *testing.T) {
	generator := New()

	tests := []struct {
		name     string
		expected string
	}{
		{"excellent", "#3fb950"},
		{"good", "#7c3aed"},
		{"acceptable", "#d29922"},
		{"low", "#fb8500"},
		{"poor", "#f85149"},
		{"unknown", "#8b949e"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := generator.getColorByName(tt.name)
			assert.Equal(t, tt.expected, color)
		})
	}
}

func TestRenderSVGStyles(t *testing.T) {
	generator := New()
	ctx := context.Background()

	badgeData := Data{
		Label:     "coverage",
		Message:   "85.5%",
		Color:     "#3fb950",
		AriaLabel: "Code coverage: 85.5 percent",
	}

	tests := []struct {
		style        string
		expectedText string
	}{
		{"flat", `rx="3"`}, // rounded corners
		{"flat-square", `shape-rendering="crispEdges"`}, // crisp edges
		{"for-the-badge", `font-weight="bold"`},         // bold text
	}

	for _, tt := range tests {
		t.Run(tt.style, func(t *testing.T) {
			badgeData.Style = tt.style
			svg, err := generator.renderSVG(ctx, badgeData)
			require.NoError(t, err)

			svgStr := string(svg)
			assert.Contains(t, svgStr, tt.expectedText)
		})
	}
}

func TestCalculateTextWidth(t *testing.T) {
	generator := New()

	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"a", 7},         // ceil(1 * 6.5) = 7
		{"abc", 20},      // ceil(3 * 6.5) = 20
		{"coverage", 52}, // ceil(8 * 6.5) = 52
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			width := generator.calculateTextWidth(tt.text)
			assert.Equal(t, tt.expected, width)
		})
	}
}

func TestOptions(t *testing.T) {
	opts := &Options{}

	WithStyle("flat-square")(opts)
	assert.Equal(t, "flat-square", opts.Style)

	WithLabel("test")(opts)
	assert.Equal(t, "test", opts.Label)

	WithLogo("logo.svg")(opts)
	assert.Equal(t, "logo.svg", opts.Logo)

	WithLogoColor("blue")(opts)
	assert.Equal(t, "blue", opts.LogoColor)
}

func TestRenderFlatBadge(t *testing.T) {
	generator := New()

	data := Data{
		Label:     "coverage",
		Message:   "85.5%",
		Color:     "#3fb950",
		Style:     "flat",
		AriaLabel: "Code coverage: 85.5 percent",
	}

	svg := generator.renderFlatBadge(data, 100, 50, 40, 0)
	svgStr := string(svg)

	assert.Contains(t, svgStr, `xmlns="http://www.w3.org/2000/svg"`)
	assert.Contains(t, svgStr, `width="100"`)
	assert.Contains(t, svgStr, `height="20"`)
	assert.Contains(t, svgStr, "coverage")
	assert.Contains(t, svgStr, "85.5%")
	assert.Contains(t, svgStr, "#3fb950")
	assert.Contains(t, svgStr, `rx="3"`)         // rounded corners
	assert.Contains(t, svgStr, `linearGradient`) // gradient effect
}

func TestRenderFlatSquareBadge(t *testing.T) {
	generator := New()

	data := Data{
		Label:     "coverage",
		Message:   "85.5%",
		Color:     "#3fb950",
		Style:     "flat-square",
		AriaLabel: "Code coverage: 85.5 percent",
	}

	svg := generator.renderFlatSquareBadge(data, 100, 20, 50, 40, 0)
	svgStr := string(svg)

	assert.Contains(t, svgStr, `xmlns="http://www.w3.org/2000/svg"`)
	assert.Contains(t, svgStr, `shape-rendering="crispEdges"`)
	assert.Contains(t, svgStr, "coverage")
	assert.Contains(t, svgStr, "85.5%")
	assert.Contains(t, svgStr, "#3fb950")
	assert.NotContains(t, svgStr, `rx="3"`) // no rounded corners
}

func TestRenderForTheBadge(t *testing.T) {
	generator := New()

	data := Data{
		Label:     "coverage",
		Message:   "85.5%",
		Color:     "#3fb950",
		Style:     "for-the-badge",
		AriaLabel: "Code coverage: 85.5 percent",
	}

	svg := generator.renderForTheBadge(data, 100, 28, 50, 40, 0)
	svgStr := string(svg)

	assert.Contains(t, svgStr, `xmlns="http://www.w3.org/2000/svg"`)
	assert.Contains(t, svgStr, `font-weight="bold"`)
	assert.Contains(t, svgStr, "COVERAGE") // uppercase
	assert.Contains(t, svgStr, "85.5%")
	assert.Contains(t, svgStr, "#3fb950")
}

func TestRenderWithLogo(t *testing.T) {
	generator := New()

	data := Data{
		Label:     "coverage",
		Message:   "85.5%",
		Color:     "#3fb950",
		Style:     "flat",
		Logo:      "https://example.com/logo.svg",
		AriaLabel: "Code coverage: 85.5 percent",
	}

	svg := generator.renderFlatBadge(data, 120, 50, 40, 16)
	svgStr := string(svg)

	assert.Contains(t, svgStr, `<image`)
	assert.Contains(t, svgStr, `xlink:href="https://example.com/logo.svg"`)
	assert.Contains(t, svgStr, `width="14"`)
	assert.Contains(t, svgStr, `height="14"`)
}

func TestGenerateContextCancellation(t *testing.T) {
	generator := New()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	_, err := generator.Generate(ctx, 85.5)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestGenerateValidSVG(t *testing.T) {
	generator := New()
	ctx := context.Background()

	// Test various coverage percentages
	coverages := []float64{0.0, 25.5, 50.0, 75.3, 100.0}

	for _, coverage := range coverages {
		t.Run(fmt.Sprintf("coverage_%.1f", coverage), func(t *testing.T) {
			svg, err := generator.Generate(ctx, coverage)
			require.NoError(t, err)

			svgStr := string(svg)
			// Basic SVG validation
			assert.True(t, strings.HasPrefix(svgStr, "<svg"))
			assert.True(t, strings.HasSuffix(svgStr, "</svg>"))
			assert.Contains(t, svgStr, fmt.Sprintf("%.1f%%", coverage))

			// Accessibility checks
			assert.Contains(t, svgStr, `role="img"`)
			assert.Contains(t, svgStr, `aria-label=`)
			assert.Contains(t, svgStr, "<title>")
		})
	}
}

func TestGenerateCustomThresholds(t *testing.T) {
	config := &Config{
		Style: "flat",
		Label: "coverage",
		ThresholdConfig: ThresholdConfig{
			Excellent:  95.0,
			Good:       85.0,
			Acceptable: 75.0,
			Low:        65.0,
		},
	}

	generator := NewWithConfig(config)
	ctx := context.Background()

	tests := []struct {
		percentage float64
		expected   string
	}{
		{96.0, "#3fb950"}, // excellent
		{87.0, "#7c3aed"}, // good
		{77.0, "#d29922"}, // acceptable
		{67.0, "#fb8500"}, // low
		{55.0, "#f85149"}, // poor
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%.1f%%", tt.percentage), func(t *testing.T) {
			svg, err := generator.Generate(ctx, tt.percentage)
			require.NoError(t, err)

			svgStr := string(svg)
			assert.Contains(t, svgStr, tt.expected)
		})
	}
}

func TestGenerateEdgeCases(t *testing.T) {
	generator := New()
	ctx := context.Background()

	tests := []struct {
		name       string
		percentage float64
	}{
		{"zero coverage", 0.0},
		{"perfect coverage", 100.0},
		{"high precision", 87.123456},
		{"very low", 0.1},
		{"very high", 99.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg, err := generator.Generate(ctx, tt.percentage)
			require.NoError(t, err)
			assert.NotEmpty(t, svg)

			svgStr := string(svg)
			// Should format to 1 decimal place
			expected := fmt.Sprintf("%.1f%%", tt.percentage)
			assert.Contains(t, svgStr, expected)
		})
	}
}

func TestResolveLogo(t *testing.T) {
	generator := New()

	tests := []struct {
		name     string
		input    string
		expected string
		contains string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "go logo",
			input:    "go",
			contains: "data:image/svg+xml;base64,",
		},
		{
			name:     "GitHub logo",
			input:    "github",
			contains: "data:image/svg+xml;base64,",
		},
		{
			name:     "case insensitive go",
			input:    "GO",
			contains: "data:image/svg+xml;base64,",
		},
		{
			name:     "valid HTTP URL",
			input:    "https://example.com/logo.svg",
			expected: "https://example.com/logo.svg",
		},
		{
			name:     "valid data URI",
			input:    "data:image/svg+xml;base64,PHN2Zw==",
			expected: "data:image/svg+xml;base64,PHN2Zw==",
		},
		{
			name:     "invalid logo name",
			input:    "invalid-logo",
			expected: "",
		},
		{
			name:     "relative path - invalid",
			input:    "./logo.svg",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.resolveLogo(tt.input)
			
			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			} else if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestGenerateWithResolvedLogos(t *testing.T) {
	generator := New()
	ctx := context.Background()

	tests := []struct {
		name     string
		logo     string
		hasImage bool
	}{
		{"no logo", "", false},
		{"go logo", "go", true},
		{"github logo", "github", true},
		{"invalid logo", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg, err := generator.Generate(ctx, 85.5, WithLogo(tt.logo))
			require.NoError(t, err)
			assert.NotEmpty(t, svg)

			svgStr := string(svg)
			if tt.hasImage {
				assert.Contains(t, svgStr, "<image")
				assert.Contains(t, svgStr, "xlink:href=\"data:")
			} else {
				assert.NotContains(t, svgStr, "<image")
			}
		})
	}
}
