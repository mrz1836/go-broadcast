package badge

import (
	"context"
	"testing"
)

// BenchmarkGenerate benchmarks badge generation performance
func BenchmarkGenerate(b *testing.B) {
	generator := New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, 85.5)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateFlat benchmarks flat badge generation
func BenchmarkGenerateFlat(b *testing.B) {
	generator := New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, 85.5, WithStyle("flat"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateFlatSquare benchmarks flat-square badge generation
func BenchmarkGenerateFlatSquare(b *testing.B) {
	generator := New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, 85.5, WithStyle("flat-square"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateForTheBadge benchmarks for-the-badge style generation
func BenchmarkGenerateForTheBadge(b *testing.B) {
	generator := New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, 85.5, WithStyle("for-the-badge"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateWithLogo benchmarks badge generation with logo
func BenchmarkGenerateWithLogo(b *testing.B) {
	generator := New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, 85.5, WithLogo("https://example.com/logo.svg"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateTrendBadge benchmarks trend badge generation
func BenchmarkGenerateTrendBadge(b *testing.B) {
	generator := New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.GenerateTrendBadge(ctx, 85.5, 80.0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetColorForPercentage benchmarks color calculation
func BenchmarkGetColorForPercentage(b *testing.B) { //nolint:revive // function naming
	generator := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.getColorForPercentage(85.5)
	}
}

// BenchmarkCalculateTextWidth benchmarks text width calculation
func BenchmarkCalculateTextWidth(b *testing.B) { //nolint:revive // function naming
	generator := New()
	text := "coverage 85.5%"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.calculateTextWidth(text)
	}
}

// BenchmarkRenderSVG benchmarks SVG rendering
func BenchmarkRenderSVG(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()

	data := Data{
		Label:     "coverage",
		Message:   "85.5%",
		Color:     "#3fb950",
		Style:     "flat",
		AriaLabel: "Code coverage: 85.5 percent",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.renderSVG(ctx, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRenderFlatBadge benchmarks flat badge rendering
func BenchmarkRenderFlatBadge(b *testing.B) { //nolint:revive // function naming
	generator := New()

	data := Data{
		Label:     "coverage",
		Message:   "85.5%",
		Color:     "#3fb950",
		Style:     "flat",
		AriaLabel: "Code coverage: 85.5 percent",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.renderFlatBadge(data, 100, 50, 40, 0)
	}
}

// BenchmarkRenderFlatSquareBadge benchmarks flat-square badge rendering
func BenchmarkRenderFlatSquareBadge(b *testing.B) { //nolint:revive // function naming
	generator := New()

	data := Data{
		Label:     "coverage",
		Message:   "85.5%",
		Color:     "#3fb950",
		Style:     "flat-square",
		AriaLabel: "Code coverage: 85.5 percent",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.renderFlatSquareBadge(data, 100, 20, 50, 40, 0)
	}
}

// BenchmarkRenderForTheBadge benchmarks for-the-badge style rendering
func BenchmarkRenderForTheBadge(b *testing.B) { //nolint:revive // function naming
	generator := New()

	data := Data{
		Label:     "coverage",
		Message:   "85.5%",
		Color:     "#3fb950",
		Style:     "for-the-badge",
		AriaLabel: "Code coverage: 85.5 percent",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.renderForTheBadge(data, 100, 28, 50, 40, 0)
	}
}

// BenchmarkMemoryAllocation benchmarks memory allocation during badge generation
func BenchmarkMemoryAllocation(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		badge, err := generator.Generate(ctx, 85.5)
		if err != nil {
			b.Fatal(err)
		}
		_ = badge // Prevent optimization
	}
}

// BenchmarkConcurrentGeneration benchmarks concurrent badge generation
func BenchmarkConcurrentGeneration(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := generator.Generate(ctx, 85.5)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMultipleStyles benchmarks generation of different badge styles
func BenchmarkMultipleStyles(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()

	styles := []string{"flat", "flat-square", "for-the-badge"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		style := styles[i%len(styles)]
		_, err := generator.Generate(ctx, 85.5, WithStyle(style))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVariousCoverages benchmarks generation with different coverage values
func BenchmarkVariousCoverages(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()

	coverages := []float64{0.0, 25.5, 50.0, 75.3, 100.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coverage := coverages[i%len(coverages)]
		_, err := generator.Generate(ctx, coverage)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWithOptions benchmarks generation with various options
func BenchmarkWithOptions(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, 85.5,
			WithStyle("flat"),
			WithLabel("test coverage"),
			WithLogo("https://example.com/logo.svg"),
			WithLogoColor("blue"),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}
