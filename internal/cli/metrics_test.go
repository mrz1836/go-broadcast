package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// TestFormatMetricsStatus tests status formatting
func TestFormatMetricsStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"success", "success", "✓ success"},
		{"partial", "partial", "⚠ partial"},
		{"failed", "failed", "✗ failed"},
		{"running", "running", "⟳ running"},
		{"pending", "pending", "○ pending"},
		{"skipped", "skipped", "- skipped"},
		{"no changes", "no_changes", "- no changes"},
		{"unknown", "unknown_status", "unknown_status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMetricsStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFormatMetricsDuration tests duration formatting
func TestFormatMetricsDuration(t *testing.T) {
	tests := []struct {
		name     string
		ms       int64
		expected string
	}{
		{"zero", 0, "-"},
		{"milliseconds", 500, "500ms"},
		{"under second", 999, "999ms"},
		{"seconds", 1500, "1.5s"},
		{"under minute", 45000, "45.0s"},
		{"minutes", 90000, "1.5m"},
		{"hours", 3660000, "61.0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMetricsDuration(tt.ms)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFormatMetricsTime tests time formatting
func TestFormatMetricsTime(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	testTime := time.Date(2026, 2, 15, 14, 30, 0, 0, loc)

	result := formatMetricsTime(testTime)
	assert.Contains(t, result, "2026-02-15")
	assert.Contains(t, result, "14:30:00")
}

// TestFormatMetricsTimeShort tests short time formatting
func TestFormatMetricsTimeShort(t *testing.T) {
	testTime := time.Date(2026, 2, 15, 14, 30, 0, 0, time.UTC)

	result := formatMetricsTimeShort(testTime)
	assert.Equal(t, "02-15 14:30", result)
}

// TestParseDuration tests duration parsing
func TestParseDuration(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		input         string
		expectedDelta time.Duration
		tolerance     time.Duration
		expectError   bool
	}{
		{"hours", "24h", 24 * time.Hour, 1 * time.Second, false},
		{"days", "7d", 7 * 24 * time.Hour, 1 * time.Second, false},
		{"weeks", "2w", 14 * 24 * time.Hour, 1 * time.Second, false},
		{"months", "1m", now.Sub(now.AddDate(0, -1, 0)), 1 * time.Second, false},
		{"years", "1y", now.Sub(now.AddDate(-1, 0, 0)), 1 * time.Second, false},
		{"empty", "", 0, 0, true},
		{"invalid format", "abc", 0, 0, true},
		{"invalid unit", "5x", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDuration(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			delta := now.Sub(result)

			assert.InDelta(t, tt.expectedDelta.Seconds(), delta.Seconds(), tt.tolerance.Seconds())
		})
	}
}

// TestParseDurationErrors tests error cases
func TestParseDurationErrors(t *testing.T) {
	t.Run("empty duration", func(t *testing.T) {
		_, err := parseDuration("")
		assert.ErrorIs(t, err, ErrEmptyDuration)
	})

	t.Run("unknown unit", func(t *testing.T) {
		_, err := parseDuration("5q")
		assert.ErrorIs(t, err, ErrUnknownDurationUnit)
	})
}

// TestOutputJSONMetrics tests JSON output formatting
func TestOutputJSONMetrics(t *testing.T) {
	// This test would require capturing stdout which is complex
	// For now, we'll just test that the function doesn't panic
	t.Run("simple data", func(t *testing.T) {
		data := map[string]interface{}{
			"total": 10,
			"rate":  95.5,
		}

		// This writes to stdout, so we can't easily assert the output.
		// In a real test, we'd want to capture stdout or refactor to take io.Writer.
		err := outputJSONMetrics("test", data)
		assert.NoError(t, err)
	})
}

// TestMetricsFlagsThreadSafety tests thread-safe flag access
func TestMetricsFlagsThreadSafety(t *testing.T) {
	t.Run("concurrent flag access", func(t *testing.T) {
		// Set initial values
		metricsFlagsMu.Lock()
		metricsLast = "7d"
		metricsRepo = "test-repo"
		metricsRunID = "SR-123"
		metricsJSON = true
		metricsFlagsMu.Unlock()

		// Read concurrently
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				last, repo, runID, jsonOut := getMetricsFlags()
				assert.NotEmpty(t, last)
				assert.NotEmpty(t, repo)
				assert.NotEmpty(t, runID)
				assert.True(t, jsonOut)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestShowRepoHistory_InvalidFormat tests that invalid repo format gives a clear error
func TestShowRepoHistory_InvalidFormat(t *testing.T) {
	t.Run("no slash in name returns error", func(t *testing.T) {
		// A string with no slash and not a number should fail
		err := showRepoHistory(context.Background(), nil, nil, "not-a-repo", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid repo format")
		assert.Contains(t, err.Error(), "owner/name")
	})
}

// TestMetricsCommandFlags tests metrics command flag initialization
func TestMetricsCommandFlags(t *testing.T) {
	t.Run("command has expected flags", func(t *testing.T) {
		cmd := metricsCmd

		assert.NotNil(t, cmd.Flags().Lookup("last"))
		assert.NotNil(t, cmd.Flags().Lookup("repo"))
		assert.NotNil(t, cmd.Flags().Lookup("run"))
		assert.NotNil(t, cmd.Flags().Lookup("json"))
	})

	t.Run("command metadata", func(t *testing.T) {
		cmd := metricsCmd

		assert.Equal(t, "metrics", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})
}

// TestShowSummaryStats tests the showSummaryStats function
func TestShowSummaryStats(t *testing.T) {
	t.Parallel()

	t.Run("EmptyDB", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		syncRepo := db.NewBroadcastSyncRepo(gormDB)

		err := showSummaryStats(context.Background(), syncRepo, false)
		require.NoError(t, err)
	})

	t.Run("JSONOutput", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		syncRepo := db.NewBroadcastSyncRepo(gormDB)

		err := showSummaryStats(context.Background(), syncRepo, true)
		require.NoError(t, err)
	})
}

// TestShowRecentRuns tests the showRecentRuns function
func TestShowRecentRuns(t *testing.T) {
	t.Parallel()

	t.Run("InvalidPeriod", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		syncRepo := db.NewBroadcastSyncRepo(gormDB)

		err := showRecentRuns(context.Background(), syncRepo, "bad", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid period")
	})

	t.Run("EmptyResults", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		syncRepo := db.NewBroadcastSyncRepo(gormDB)

		err := showRecentRuns(context.Background(), syncRepo, "7d", false)
		require.NoError(t, err)
	})

	t.Run("JSONOutput", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		syncRepo := db.NewBroadcastSyncRepo(gormDB)

		err := showRecentRuns(context.Background(), syncRepo, "7d", true)
		require.NoError(t, err)
	})
}

// TestShowRunDetails tests the showRunDetails function
func TestShowRunDetails(t *testing.T) {
	t.Parallel()

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		syncRepo := db.NewBroadcastSyncRepo(gormDB)

		err := showRunDetails(context.Background(), syncRepo, "SR-nonexistent", false)
		require.Error(t, err)
	})
}
