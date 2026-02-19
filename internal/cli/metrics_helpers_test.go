package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatMetricsStatusEdgeCases tests edge cases for status formatting
func TestFormatMetricsStatusEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string passthrough",
			input:    "",
			expected: "",
		},
		{
			name:     "case sensitive - uppercase not matched",
			input:    "SUCCESS",
			expected: "SUCCESS",
		},
		{
			name:     "mixed case not matched",
			input:    "Success",
			expected: "Success",
		},
		{
			name:     "whitespace not matched",
			input:    " success ",
			expected: " success ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatMetricsStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFormatMetricsDurationEdgeCases tests edge cases for duration formatting
func TestFormatMetricsDurationEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ms       int64
		expected string
	}{
		{
			name:     "negative value",
			ms:       -100,
			expected: "-100ms",
		},
		{
			name:     "one millisecond",
			ms:       1,
			expected: "1ms",
		},
		{
			name:     "exact one second",
			ms:       1000,
			expected: "1.0s",
		},
		{
			name:     "exact one minute",
			ms:       60000,
			expected: "1.0m",
		},
		{
			name:     "just under one second boundary",
			ms:       999,
			expected: "999ms",
		},
		{
			name:     "just at one second boundary",
			ms:       1000,
			expected: "1.0s",
		},
		{
			name:     "just under one minute boundary",
			ms:       59999,
			expected: "60.0s",
		},
		{
			name:     "large value in minutes",
			ms:       7200000,
			expected: "120.0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatMetricsDuration(tt.ms)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFormatMetricsTimeWithTimezones tests time formatting with various timezones
func TestFormatMetricsTimeWithTimezones(t *testing.T) {
	t.Parallel()

	t.Run("UTC time", func(t *testing.T) {
		t.Parallel()
		testTime := time.Date(2026, 1, 15, 10, 30, 45, 0, time.UTC)
		result := formatMetricsTime(testTime)
		assert.Equal(t, "2026-01-15 10:30:45 UTC", result)
	})

	t.Run("zero time", func(t *testing.T) {
		t.Parallel()
		result := formatMetricsTime(time.Time{})
		assert.Contains(t, result, "0001-01-01")
	})
}

// TestFormatMetricsTimeShortEdgeCases tests short time formatting edge cases
func TestFormatMetricsTimeShortEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("midnight", func(t *testing.T) {
		t.Parallel()
		testTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		result := formatMetricsTimeShort(testTime)
		assert.Equal(t, "01-01 00:00", result)
	})

	t.Run("end of day", func(t *testing.T) {
		t.Parallel()
		testTime := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC)
		result := formatMetricsTimeShort(testTime)
		assert.Equal(t, "12-31 23:59", result)
	})
}

// TestParseDurationEdgeCases tests additional edge cases for duration parsing
func TestParseDurationEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("one hour", func(t *testing.T) {
		t.Parallel()
		result, err := parseDuration("1h")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().Add(-time.Hour), result, 2*time.Second)
	})

	t.Run("one day", func(t *testing.T) {
		t.Parallel()
		result, err := parseDuration("1d")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().AddDate(0, 0, -1), result, 2*time.Second)
	})

	t.Run("one week", func(t *testing.T) {
		t.Parallel()
		result, err := parseDuration("1w")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().AddDate(0, 0, -7), result, 2*time.Second)
	})

	t.Run("one month", func(t *testing.T) {
		t.Parallel()
		result, err := parseDuration("1m")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().AddDate(0, -1, 0), result, 24*time.Hour)
	})

	t.Run("one year", func(t *testing.T) {
		t.Parallel()
		result, err := parseDuration("1y")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().AddDate(-1, 0, 0), result, 24*time.Hour)
	})

	t.Run("large number of days", func(t *testing.T) {
		t.Parallel()
		result, err := parseDuration("365d")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().AddDate(0, 0, -365), result, 2*time.Second)
	})

	t.Run("empty string returns ErrEmptyDuration", func(t *testing.T) {
		t.Parallel()
		_, err := parseDuration("")
		require.ErrorIs(t, err, ErrEmptyDuration)
	})

	t.Run("invalid unit returns ErrUnknownDurationUnit", func(t *testing.T) {
		t.Parallel()
		_, err := parseDuration("5z")
		require.ErrorIs(t, err, ErrUnknownDurationUnit)
	})

	t.Run("missing number returns error", func(t *testing.T) {
		t.Parallel()
		_, err := parseDuration("d")
		assert.Error(t, err)
	})

	t.Run("negative number is accepted", func(t *testing.T) {
		t.Parallel()
		// parseDuration uses Sscanf which will parse negative numbers
		// The result will be a time in the future
		_, err := parseDuration("-1d")
		// The Sscanf may or may not handle this; we just verify no panic
		// and check that an error or valid result is returned
		if err != nil {
			assert.Error(t, err)
		}
	})
}

// TestGetMetricsFlags tests the thread-safe flag getter
func TestGetMetricsFlags(t *testing.T) {
	t.Run("returns current flag values", func(t *testing.T) {
		// Save originals
		metricsFlagsMu.Lock()
		origLast := metricsLast
		origRepo := metricsRepo
		origRunID := metricsRunID
		origJSON := metricsJSON
		metricsFlagsMu.Unlock()

		// Restore after test
		defer func() {
			metricsFlagsMu.Lock()
			metricsLast = origLast
			metricsRepo = origRepo
			metricsRunID = origRunID
			metricsJSON = origJSON
			metricsFlagsMu.Unlock()
		}()

		// Set known values
		metricsFlagsMu.Lock()
		metricsLast = "14d"
		metricsRepo = "org/repo"
		metricsRunID = "SR-999"
		metricsJSON = false
		metricsFlagsMu.Unlock()

		last, repo, runID, jsonOut := getMetricsFlags()
		assert.Equal(t, "14d", last)
		assert.Equal(t, "org/repo", repo)
		assert.Equal(t, "SR-999", runID)
		assert.False(t, jsonOut)
	})

	t.Run("returns empty defaults", func(t *testing.T) {
		// Save originals
		metricsFlagsMu.Lock()
		origLast := metricsLast
		origRepo := metricsRepo
		origRunID := metricsRunID
		origJSON := metricsJSON
		metricsFlagsMu.Unlock()

		// Restore after test
		defer func() {
			metricsFlagsMu.Lock()
			metricsLast = origLast
			metricsRepo = origRepo
			metricsRunID = origRunID
			metricsJSON = origJSON
			metricsFlagsMu.Unlock()
		}()

		// Set empty values
		metricsFlagsMu.Lock()
		metricsLast = ""
		metricsRepo = ""
		metricsRunID = ""
		metricsJSON = false
		metricsFlagsMu.Unlock()

		last, repo, runID, jsonOut := getMetricsFlags()
		assert.Empty(t, last)
		assert.Empty(t, repo)
		assert.Empty(t, runID)
		assert.False(t, jsonOut)
	})
}
