package benchmark

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/jsonutil"
)

// BaselineReport represents a complete performance baseline
type BaselineReport struct {
	Timestamp  time.Time          `json:"timestamp"`
	GoVersion  string             `json:"go_version"`
	GOOS       string             `json:"goos"`
	GOARCH     string             `json:"goarch"`
	Benchmarks map[string]Metrics `json:"benchmarks"`
}

// Metrics contains detailed performance metrics
type Metrics struct {
	Name        string  `json:"name"`
	NsPerOp     int64   `json:"ns_per_op"`
	AllocsPerOp int64   `json:"allocs_per_op"`
	BytesPerOp  int64   `json:"bytes_per_op"`
	MBPerSec    float64 `json:"mb_per_sec,omitempty"`
	Operations  int64   `json:"operations,omitempty"`
}

// ComparisonReport contains performance comparison data
type ComparisonReport struct {
	BaselineReport BaselineReport     `json:"baseline"`
	CurrentReport  BaselineReport     `json:"current"`
	Improvements   map[string]float64 `json:"improvements"`
	Regressions    map[string]float64 `json:"regressions"`
	Summary        ComparisonSummary  `json:"summary"`
}

// ComparisonSummary provides high-level comparison metrics
type ComparisonSummary struct {
	TotalBenchmarks     int     `json:"total_benchmarks"`
	Improved            int     `json:"improved"`
	Regressed           int     `json:"regressed"`
	Unchanged           int     `json:"unchanged"`
	AvgSpeedImprovement float64 `json:"avg_speed_improvement"`
	AvgMemoryReduction  float64 `json:"avg_memory_reduction"`
}

// SaveBaseline saves benchmark results to a JSON file
func SaveBaseline(filename string, report BaselineReport) error {
	// Use PrettyPrint to get formatted JSON
	formatted, err := jsonutil.PrettyPrint(report)
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}
	data := []byte(formatted)

	if err := os.WriteFile(filename, data, 0o600); err != nil {
		return fmt.Errorf("failed to write baseline file: %w", err)
	}

	return nil
}

// LoadBaseline loads benchmark results from a JSON file
func LoadBaseline(filename string) (*BaselineReport, error) {
	data, err := os.ReadFile(filename) //nolint:gosec // Reading benchmark baseline file
	if err != nil {
		return nil, fmt.Errorf("failed to read baseline file: %w", err)
	}

	report, err := jsonutil.UnmarshalJSON[BaselineReport](data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal baseline: %w", err)
	}

	return &report, nil
}

// CreateBaselineReport creates a new baseline report with system information
func CreateBaselineReport(benchmarks map[string]Metrics) BaselineReport {
	return BaselineReport{
		Timestamp:  time.Now(),
		GoVersion:  runtime.Version(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		Benchmarks: benchmarks,
	}
}

// CompareWithBaseline compares current results with baseline
func CompareWithBaseline(current, baseline BaselineReport) ComparisonReport {
	improvements := make(map[string]float64)
	regressions := make(map[string]float64)

	var totalSpeedChange, totalMemoryChange float64
	var changedCount int

	for name, currentMetric := range current.Benchmarks {
		baselineMetric, exists := baseline.Benchmarks[name]
		if !exists {
			continue
		}

		// Calculate speed improvement (lower ns/op is better)
		speedChange := calculateImprovement(float64(baselineMetric.NsPerOp), float64(currentMetric.NsPerOp))

		// Calculate memory improvement (fewer bytes/op is better)
		memoryChange := calculateImprovement(float64(baselineMetric.BytesPerOp), float64(currentMetric.BytesPerOp))

		if speedChange > 0 {
			improvements[name+"_speed"] = speedChange
		} else if speedChange < 0 {
			regressions[name+"_speed"] = -speedChange
		}

		if memoryChange > 0 {
			improvements[name+"_memory"] = memoryChange
		} else if memoryChange < 0 {
			regressions[name+"_memory"] = -memoryChange
		}

		totalSpeedChange += speedChange
		totalMemoryChange += memoryChange
		changedCount++
	}

	summary := ComparisonSummary{
		TotalBenchmarks:     len(current.Benchmarks),
		Improved:            len(improvements),
		Regressed:           len(regressions),
		Unchanged:           len(current.Benchmarks) - len(improvements) - len(regressions),
		AvgSpeedImprovement: totalSpeedChange / float64(changedCount),
		AvgMemoryReduction:  totalMemoryChange / float64(changedCount),
	}

	return ComparisonReport{
		BaselineReport: baseline,
		CurrentReport:  current,
		Improvements:   improvements,
		Regressions:    regressions,
		Summary:        summary,
	}
}

// GenerateTextReport creates a human-readable performance report
func GenerateTextReport(comparison ComparisonReport) string {
	var report strings.Builder

	report.WriteString("Performance Comparison Report\n")
	report.WriteString("=============================\n\n")

	report.WriteString(fmt.Sprintf("Baseline: %s (%s %s)\n",
		comparison.BaselineReport.Timestamp.Format("2006-01-02 15:04:05"),
		comparison.BaselineReport.GOOS, comparison.BaselineReport.GOARCH))
	report.WriteString(fmt.Sprintf("Current:  %s (%s %s)\n\n",
		comparison.CurrentReport.Timestamp.Format("2006-01-02 15:04:05"),
		comparison.CurrentReport.GOOS, comparison.CurrentReport.GOARCH))

	// Summary
	summary := comparison.Summary
	report.WriteString("Summary:\n")
	report.WriteString(fmt.Sprintf("  Total Benchmarks: %d\n", summary.TotalBenchmarks))
	report.WriteString(fmt.Sprintf("  Improved:         %d\n", summary.Improved))
	report.WriteString(fmt.Sprintf("  Regressed:        %d\n", summary.Regressed))
	report.WriteString(fmt.Sprintf("  Unchanged:        %d\n", summary.Unchanged))
	report.WriteString(fmt.Sprintf("  Avg Speed Improvement: %.1f%%\n", summary.AvgSpeedImprovement))
	report.WriteString(fmt.Sprintf("  Avg Memory Reduction:  %.1f%%\n\n", summary.AvgMemoryReduction))

	// Improvements
	if len(comparison.Improvements) > 0 {
		report.WriteString("Improvements:\n")
		improvements := sortMetrics(comparison.Improvements)
		for _, metric := range improvements {
			report.WriteString(fmt.Sprintf("  %s: %.1f%% %s\n",
				metric.Name, metric.Value, getImprovementEmoji(metric.Value)))
		}
		report.WriteString("\n")
	}

	// Regressions
	if len(comparison.Regressions) > 0 {
		report.WriteString("Regressions:\n")
		regressions := sortMetrics(comparison.Regressions)
		for _, metric := range regressions {
			report.WriteString(fmt.Sprintf("  %s: %.1f%% ⚠️\n", metric.Name, metric.Value))
		}
		report.WriteString("\n")
	}

	// Detailed comparison
	report.WriteString("Detailed Comparison:\n")
	for name, currentMetric := range comparison.CurrentReport.Benchmarks {
		baselineMetric, exists := comparison.BaselineReport.Benchmarks[name]
		if !exists {
			continue
		}

		report.WriteString(fmt.Sprintf("\n%s:\n", name))
		report.WriteString(fmt.Sprintf("  Speed:  %d ns/op → %d ns/op (%.1f%%)\n",
			baselineMetric.NsPerOp, currentMetric.NsPerOp,
			calculateImprovement(float64(baselineMetric.NsPerOp), float64(currentMetric.NsPerOp))))
		report.WriteString(fmt.Sprintf("  Memory: %d B/op → %d B/op (%.1f%%)\n",
			baselineMetric.BytesPerOp, currentMetric.BytesPerOp,
			calculateImprovement(float64(baselineMetric.BytesPerOp), float64(currentMetric.BytesPerOp))))
		report.WriteString(fmt.Sprintf("  Allocs: %d allocs/op → %d allocs/op (%.1f%%)\n",
			baselineMetric.AllocsPerOp, currentMetric.AllocsPerOp,
			calculateImprovement(float64(baselineMetric.AllocsPerOp), float64(currentMetric.AllocsPerOp))))
	}

	return report.String()
}

// Helper functions

type metricPair struct {
	Name  string
	Value float64
}

func sortMetrics(metrics map[string]float64) []metricPair {
	pairs := make([]metricPair, 0, len(metrics))
	for name, value := range metrics {
		pairs = append(pairs, metricPair{Name: name, Value: value})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	return pairs
}

func calculateImprovement(baseline, current float64) float64 {
	if baseline == 0 {
		return 0
	}
	return ((baseline - current) / baseline) * 100
}

func getImprovementEmoji(improvement float64) string {
	switch {
	case improvement >= 50:
		return "🚀"
	case improvement >= 25:
		return "⚡"
	case improvement >= 10:
		return "✅"
	case improvement >= 5:
		return "👍"
	default:
		return "📈"
	}
}
