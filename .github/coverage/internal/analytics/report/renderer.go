package report

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"math"
	"strings"
)

// Renderer handles template rendering for coverage reports
type Renderer struct {
	templates map[string]*template.Template
}

// NewRenderer creates a new template renderer
func NewRenderer() *Renderer {
	return &Renderer{
		templates: make(map[string]*template.Template),
	}
}

// RenderReport renders the coverage report template
func (r *Renderer) RenderReport(_ context.Context, data interface{}) ([]byte, error) {
	// Create template functions
	funcMap := template.FuncMap{
		"multiply": func(a float64, b float64) float64 {
			return a * b
		},
		"printf": fmt.Sprintf,
		"commas": func(n int) string {
			return addCommas(n)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length]
		},
		"ge": func(a, b float64) bool {
			return a >= b
		},
		"sub": func(a, b float64) float64 {
			return a - b
		},
		"round": func(f float64) float64 {
			return math.Round(f)
		},
	}

	// Parse template with functions
	tmpl, err := template.New("report").Funcs(funcMap).Parse(getReportTemplate())
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// addCommas adds a thousand separators to a number
func addCommas(n int) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	var result strings.Builder
	for i, ch := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(ch)
	}
	return result.String()
}
