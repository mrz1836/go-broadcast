package benchmark

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/testutil"
)

func TestCreateBaselineReport(t *testing.T) {
	tests := []struct {
		name       string
		benchmarks map[string]Metrics
		want       struct {
			hasBenchmarks bool
			hasSystemInfo bool
			hasTimestamp  bool
		}
	}{
		{
			name: "ValidBenchmarks",
			benchmarks: map[string]Metrics{
				"test1": {
					Name:        "test1",
					NsPerOp:     1000,
					AllocsPerOp: 5,
					BytesPerOp:  256,
					Operations:  100,
				},
				"test2": {
					Name:        "test2",
					NsPerOp:     2000,
					AllocsPerOp: 10,
					BytesPerOp:  512,
					Operations:  50,
				},
			},
			want: struct {
				hasBenchmarks bool
				hasSystemInfo bool
				hasTimestamp  bool
			}{
				hasBenchmarks: true,
				hasSystemInfo: true,
				hasTimestamp:  true,
			},
		},
		{
			name:       "EmptyBenchmarks",
			benchmarks: map[string]Metrics{},
			want: struct {
				hasBenchmarks bool
				hasSystemInfo bool
				hasTimestamp  bool
			}{
				hasBenchmarks: false,
				hasSystemInfo: true,
				hasTimestamp:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateBaselineReport(tt.benchmarks)

			if tt.want.hasBenchmarks {
				require.Len(t, result.Benchmarks, len(tt.benchmarks))
				for name, expected := range tt.benchmarks {
					actual, exists := result.Benchmarks[name]
					require.True(t, exists)
					require.Equal(t, expected, actual)
				}
			} else {
				require.Empty(t, result.Benchmarks)
			}

			if tt.want.hasSystemInfo {
				require.NotEmpty(t, result.GoVersion)
				require.NotEmpty(t, result.GOOS)
				require.NotEmpty(t, result.GOARCH)
			}

			if tt.want.hasTimestamp {
				require.False(t, result.Timestamp.IsZero())
				require.Less(t, time.Since(result.Timestamp), time.Minute)
			}
		})
	}
}

func TestSaveAndLoadBaseline(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)

	tests := []struct {
		name     string
		filename string
		report   BaselineReport
		want     struct {
			saveSuccess bool
			loadSuccess bool
		}
	}{
		{
			name:     "ValidBaseline",
			filename: "test_baseline.json",
			report: BaselineReport{
				Timestamp: time.Now(),
				GoVersion: "go1.21.0",
				GOOS:      "linux",
				GOARCH:    "amd64",
				Benchmarks: map[string]Metrics{
					"test1": {
						Name:        "test1",
						NsPerOp:     1000,
						AllocsPerOp: 5,
						BytesPerOp:  256,
					},
				},
			},
			want: struct {
				saveSuccess bool
				loadSuccess bool
			}{
				saveSuccess: true,
				loadSuccess: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)

			// Test saving
			err := SaveBaseline(filePath, tt.report)
			if tt.want.saveSuccess {
				require.NoError(t, err)

				// Verify file exists
				_, statErr := os.Stat(filePath)
				require.NoError(t, statErr)
			} else {
				require.Error(t, err)
			}

			// Test loading
			if tt.want.saveSuccess {
				loaded, loadErr := LoadBaseline(filePath)
				if tt.want.loadSuccess {
					require.NoError(t, loadErr)
					require.NotNil(t, loaded)

					// Verify data integrity
					require.Equal(t, tt.report.GoVersion, loaded.GoVersion)
					require.Equal(t, tt.report.GOOS, loaded.GOOS)
					require.Equal(t, tt.report.GOARCH, loaded.GOARCH)
					require.Len(t, loaded.Benchmarks, len(tt.report.Benchmarks))

					for name, expected := range tt.report.Benchmarks {
						actual, exists := loaded.Benchmarks[name]
						require.True(t, exists)
						require.Equal(t, expected.Name, actual.Name)
						require.Equal(t, expected.NsPerOp, actual.NsPerOp)
						require.Equal(t, expected.AllocsPerOp, actual.AllocsPerOp)
						require.Equal(t, expected.BytesPerOp, actual.BytesPerOp)
					}
				} else {
					require.Error(t, loadErr)
				}
			}
		})
	}
}

func TestLoadBaselineFileNotFound(t *testing.T) {
	_, err := LoadBaseline("/nonexistent/path/baseline.json")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read baseline file")
}

func TestLoadBaselineInvalidJSON(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	invalidFile := filepath.Join(tempDir, "invalid.json")

	// Create file with invalid JSON
	testutil.WriteTestFile(t, invalidFile, "invalid json content")

	_, err := LoadBaseline(invalidFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal baseline")
}

func TestCompareWithBaseline(t *testing.T) {
	tests := []struct {
		name     string
		current  BaselineReport
		baseline BaselineReport
		want     struct {
			hasImprovements bool
			hasRegressions  bool
			hasSummary      bool
		}
	}{
		{
			name: "ImprovementScenario",
			current: BaselineReport{
				Benchmarks: map[string]Metrics{
					"test1": {Name: "test1", NsPerOp: 800, BytesPerOp: 200}, // Improved
				},
			},
			baseline: BaselineReport{
				Benchmarks: map[string]Metrics{
					"test1": {Name: "test1", NsPerOp: 1000, BytesPerOp: 250}, // Baseline
				},
			},
			want: struct {
				hasImprovements bool
				hasRegressions  bool
				hasSummary      bool
			}{
				hasImprovements: true,
				hasRegressions:  false,
				hasSummary:      true,
			},
		},
		{
			name: "RegressionScenario",
			current: BaselineReport{
				Benchmarks: map[string]Metrics{
					"test1": {Name: "test1", NsPerOp: 1200, BytesPerOp: 300}, // Regressed
				},
			},
			baseline: BaselineReport{
				Benchmarks: map[string]Metrics{
					"test1": {Name: "test1", NsPerOp: 1000, BytesPerOp: 250}, // Baseline
				},
			},
			want: struct {
				hasImprovements bool
				hasRegressions  bool
				hasSummary      bool
			}{
				hasImprovements: false,
				hasRegressions:  true,
				hasSummary:      true,
			},
		},
		{
			name: "MixedScenario",
			current: BaselineReport{
				Benchmarks: map[string]Metrics{
					"test1": {Name: "test1", NsPerOp: 800, BytesPerOp: 300},  // Speed improved, memory regressed
					"test2": {Name: "test2", NsPerOp: 1200, BytesPerOp: 200}, // Speed regressed, memory improved
				},
			},
			baseline: BaselineReport{
				Benchmarks: map[string]Metrics{
					"test1": {Name: "test1", NsPerOp: 1000, BytesPerOp: 250},
					"test2": {Name: "test2", NsPerOp: 1000, BytesPerOp: 250},
				},
			},
			want: struct {
				hasImprovements bool
				hasRegressions  bool
				hasSummary      bool
			}{
				hasImprovements: true,
				hasRegressions:  true,
				hasSummary:      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareWithBaseline(tt.current, tt.baseline)

			require.Equal(t, tt.baseline, result.BaselineReport)
			require.Equal(t, tt.current, result.CurrentReport)

			if tt.want.hasImprovements {
				require.NotEmpty(t, result.Improvements)
			} else {
				require.Empty(t, result.Improvements)
			}

			if tt.want.hasRegressions {
				require.NotEmpty(t, result.Regressions)
			} else {
				require.Empty(t, result.Regressions)
			}

			if tt.want.hasSummary {
				require.Equal(t, len(tt.current.Benchmarks), result.Summary.TotalBenchmarks)
				require.GreaterOrEqual(t, result.Summary.Improved, 0)
				require.GreaterOrEqual(t, result.Summary.Regressed, 0)
				require.GreaterOrEqual(t, result.Summary.Unchanged, 0, "Unchanged count should never be negative")
			}
		})
	}
}

func TestGenerateTextReport(t *testing.T) {
	tests := []struct {
		name       string
		comparison ComparisonReport
		want       struct {
			hasHeader       bool
			hasSummary      bool
			hasImprovements bool
			hasRegressions  bool
			hasDetailedComp bool
		}
	}{
		{
			name: "CompleteReport",
			comparison: ComparisonReport{
				BaselineReport: BaselineReport{
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					GOOS:      "linux",
					GOARCH:    "amd64",
					Benchmarks: map[string]Metrics{
						"test1": {Name: "test1", NsPerOp: 1000, BytesPerOp: 250, AllocsPerOp: 5},
					},
				},
				CurrentReport: BaselineReport{
					Timestamp: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
					GOOS:      "linux",
					GOARCH:    "amd64",
					Benchmarks: map[string]Metrics{
						"test1": {Name: "test1", NsPerOp: 800, BytesPerOp: 300, AllocsPerOp: 3},
					},
				},
				Improvements: map[string]float64{
					"test1_speed": 20.0,
				},
				Regressions: map[string]float64{
					"test1_memory": 20.0,
				},
				Summary: ComparisonSummary{
					TotalBenchmarks:     1,
					Improved:            1,
					Regressed:           1,
					Unchanged:           0,
					AvgSpeedImprovement: 10.0,
					AvgMemoryReduction:  -10.0,
				},
			},
			want: struct {
				hasHeader       bool
				hasSummary      bool
				hasImprovements bool
				hasRegressions  bool
				hasDetailedComp bool
			}{
				hasHeader:       true,
				hasSummary:      true,
				hasImprovements: true,
				hasRegressions:  true,
				hasDetailedComp: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateTextReport(tt.comparison)

			require.NotEmpty(t, result)

			if tt.want.hasHeader {
				require.Contains(t, result, "Performance Comparison Report")
				require.Contains(t, result, "=============================")
			}

			if tt.want.hasSummary {
				require.Contains(t, result, "Summary:")
				require.Contains(t, result, "Total Benchmarks:")
			}

			if tt.want.hasImprovements {
				require.Contains(t, result, "Improvements:")
			}

			if tt.want.hasRegressions {
				require.Contains(t, result, "Regressions:")
			}

			if tt.want.hasDetailedComp {
				require.Contains(t, result, "Detailed Comparison:")
			}
		})
	}
}

func TestReporterHelperFunctions(t *testing.T) {
	t.Run("SortMetrics", func(t *testing.T) {
		metrics := map[string]float64{
			"low":    5.0,
			"high":   25.0,
			"medium": 15.0,
		}

		sorted := sortMetrics(metrics)

		require.Len(t, sorted, 3)
		require.Equal(t, "high", sorted[0].Name)
		require.InDelta(t, 25.0, sorted[0].Value, 0.001)
		require.Equal(t, "medium", sorted[1].Name)
		require.InDelta(t, 15.0, sorted[1].Value, 0.001)
		require.Equal(t, "low", sorted[2].Name)
		require.InDelta(t, 5.0, sorted[2].Value, 0.001)
	})

	t.Run("CalculateImprovement", func(t *testing.T) {
		tests := []struct {
			baseline float64
			current  float64
			expected float64
		}{
			{1000, 800, 20.0},   // 20% improvement (lower is better)
			{1000, 1200, -20.0}, // 20% regression (higher is worse)
			{1000, 1000, 0.0},   // No change
			{0, 100, 0.0},       // Avoid division by zero
		}

		for _, test := range tests {
			result := calculateImprovement(test.baseline, test.current)
			require.InDelta(t, test.expected, result, 0.1)
		}
	})

	t.Run("GetImprovementEmoji", func(t *testing.T) {
		tests := []struct {
			improvement float64
			expected    string
		}{
			{60.0, "🚀"},
			{35.0, "⚡"},
			{15.0, "✅"},
			{7.0, "👍"},
			{2.0, "📈"},
		}

		for _, test := range tests {
			result := getImprovementEmoji(test.improvement)
			require.Equal(t, test.expected, result)
		}
	})
}

func TestSaveBaselinePermissions(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	filename := filepath.Join(tempDir, "permissions_test.json")

	report := BaselineReport{
		Timestamp:  time.Now(),
		GoVersion:  "go1.21.0",
		GOOS:       "linux",
		GOARCH:     "amd64",
		Benchmarks: map[string]Metrics{},
	}

	err := SaveBaseline(filename, report)
	require.NoError(t, err)

	// Check file permissions
	info, err := os.Stat(filename)
	require.NoError(t, err)

	// Should be readable and writable by owner only (0600)
	expectedMode := os.FileMode(0o600)
	require.Equal(t, expectedMode, info.Mode().Perm())
}

func TestComparisonReportJSONSerialization(t *testing.T) {
	report := ComparisonReport{
		BaselineReport: BaselineReport{
			Timestamp: time.Now(),
			GoVersion: "go1.21.0",
			GOOS:      "linux",
			GOARCH:    "amd64",
			Benchmarks: map[string]Metrics{
				"test1": {Name: "test1", NsPerOp: 1000, BytesPerOp: 250},
			},
		},
		CurrentReport: BaselineReport{
			Timestamp: time.Now(),
			GoVersion: "go1.21.0",
			GOOS:      "linux",
			GOARCH:    "amd64",
			Benchmarks: map[string]Metrics{
				"test1": {Name: "test1", NsPerOp: 800, BytesPerOp: 250},
			},
		},
		Improvements: map[string]float64{"test1_speed": 20.0},
		Regressions:  map[string]float64{},
		Summary: ComparisonSummary{
			TotalBenchmarks:     1,
			Improved:            1,
			Regressed:           0,
			Unchanged:           0,
			AvgSpeedImprovement: 20.0,
			AvgMemoryReduction:  0.0,
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(report)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Test JSON unmarshaling
	var unmarshaled ComparisonReport
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify key fields
	require.Equal(t, report.Summary.TotalBenchmarks, unmarshaled.Summary.TotalBenchmarks)
	require.Equal(t, report.Summary.Improved, unmarshaled.Summary.Improved)
	require.Len(t, unmarshaled.Improvements, len(report.Improvements))
}

// TestCompareWithBaselineNilMaps tests handling of nil benchmark maps
func TestCompareWithBaselineNilMaps(t *testing.T) {
	tests := []struct {
		name     string
		current  BaselineReport
		baseline BaselineReport
	}{
		{
			name:     "NilCurrentBenchmarks",
			current:  BaselineReport{Benchmarks: nil},
			baseline: BaselineReport{Benchmarks: map[string]Metrics{"test": {}}},
		},
		{
			name:     "NilBaselineBenchmarks",
			current:  BaselineReport{Benchmarks: map[string]Metrics{"test": {}}},
			baseline: BaselineReport{Benchmarks: nil},
		},
		{
			name:     "BothNil",
			current:  BaselineReport{Benchmarks: nil},
			baseline: BaselineReport{Benchmarks: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			require.NotPanics(t, func() {
				result := CompareWithBaseline(tt.current, tt.baseline)
				require.NotNil(t, result.Improvements)
				require.NotNil(t, result.Regressions)
			})
		})
	}
}

// TestCompareWithBaselineNoOverlap tests when benchmarks have no overlap
func TestCompareWithBaselineNoOverlap(t *testing.T) {
	current := BaselineReport{
		Benchmarks: map[string]Metrics{
			"newBenchmark": {Name: "newBenchmark", NsPerOp: 1000},
		},
	}
	baseline := BaselineReport{
		Benchmarks: map[string]Metrics{
			"oldBenchmark": {Name: "oldBenchmark", NsPerOp: 1000},
		},
	}

	result := CompareWithBaseline(current, baseline)

	// Should not cause division by zero
	require.InDelta(t, float64(0), result.Summary.AvgSpeedImprovement, 0.001)
	require.InDelta(t, float64(0), result.Summary.AvgMemoryReduction, 0.001)
	require.Empty(t, result.Improvements)
	require.Empty(t, result.Regressions)
	require.Equal(t, 1, result.Summary.TotalBenchmarks)
	require.Equal(t, 1, result.Summary.Unchanged)
}

// TestSaveBaselineEmptyFilename tests validation of empty filename
func TestSaveBaselineEmptyFilename(t *testing.T) {
	report := BaselineReport{}
	err := SaveBaseline("", report)
	require.Error(t, err)
	require.Contains(t, err.Error(), "filename cannot be empty")
}

// TestLoadBaselineEmptyFilename tests validation of empty filename
func TestLoadBaselineEmptyFilename(t *testing.T) {
	_, err := LoadBaseline("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "filename cannot be empty")
}

// TestGenerateTextReportDeterministic tests that report output is deterministic
func TestGenerateTextReportDeterministic(t *testing.T) {
	comparison := ComparisonReport{
		BaselineReport: BaselineReport{
			Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			GOOS:      "linux",
			GOARCH:    "amd64",
			Benchmarks: map[string]Metrics{
				"zzzBenchmark": {Name: "zzzBenchmark", NsPerOp: 1000, BytesPerOp: 250, AllocsPerOp: 5},
				"aaaBenchmark": {Name: "aaaBenchmark", NsPerOp: 2000, BytesPerOp: 500, AllocsPerOp: 10},
				"mmmBenchmark": {Name: "mmmBenchmark", NsPerOp: 1500, BytesPerOp: 375, AllocsPerOp: 7},
			},
		},
		CurrentReport: BaselineReport{
			Timestamp: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
			GOOS:      "linux",
			GOARCH:    "amd64",
			Benchmarks: map[string]Metrics{
				"zzzBenchmark": {Name: "zzzBenchmark", NsPerOp: 800, BytesPerOp: 200, AllocsPerOp: 4},
				"aaaBenchmark": {Name: "aaaBenchmark", NsPerOp: 1600, BytesPerOp: 400, AllocsPerOp: 8},
				"mmmBenchmark": {Name: "mmmBenchmark", NsPerOp: 1200, BytesPerOp: 300, AllocsPerOp: 6},
			},
		},
		Summary: ComparisonSummary{TotalBenchmarks: 3},
	}

	// Generate report multiple times and verify consistency
	report1 := GenerateTextReport(comparison)
	report2 := GenerateTextReport(comparison)
	report3 := GenerateTextReport(comparison)

	require.Equal(t, report1, report2, "Report should be deterministic")
	require.Equal(t, report2, report3, "Report should be deterministic")

	// Verify alphabetical ordering in detailed comparison
	require.Contains(t, report1, "aaaBenchmark")
	require.Contains(t, report1, "mmmBenchmark")
	require.Contains(t, report1, "zzzBenchmark")

	// aaaBenchmark should appear before mmmBenchmark which should appear before zzzBenchmark
	aaaIdx := len(report1) - len(report1[findIndex(report1, "aaaBenchmark"):])
	mmmIdx := len(report1) - len(report1[findIndex(report1, "mmmBenchmark"):])
	zzzIdx := len(report1) - len(report1[findIndex(report1, "zzzBenchmark"):])

	require.Less(t, aaaIdx, mmmIdx, "aaaBenchmark should appear before mmmBenchmark")
	require.Less(t, mmmIdx, zzzIdx, "mmmBenchmark should appear before zzzBenchmark")
}

// findIndex returns the starting index of substr in s, or -1 if not found
func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
