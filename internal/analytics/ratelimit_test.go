package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
)

func TestNewRateLimitInfo(t *testing.T) {
	t.Parallel()

	t.Run("extracts the core bucket", func(t *testing.T) {
		t.Parallel()

		resp := &gh.RateLimitResponse{}
		resp.Resources.Core = gh.RateLimitResource{Limit: 5000, Remaining: 3173, Used: 1827, Reset: 1_792_534_769}
		resp.Resources.GraphQL = gh.RateLimitResource{Limit: 5000, Remaining: 4998, Used: 2, Reset: 1_792_531_803}

		info := NewRateLimitInfo(resp)
		require.NotNil(t, info)
		assert.Equal(t, 5000, info.Limit)
		assert.Equal(t, 3173, info.Remaining)
		assert.Equal(t, 1827, info.Used)
		assert.Equal(t, time.Unix(1_792_534_769, 0), info.ResetAt)
	})

	t.Run("nil response", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, NewRateLimitInfo(nil))
	})
}

func TestEstimateSyncCost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		repoCount       int
		expectedMin     int
		expectedMax     int
		expectedBatches int
	}{
		{
			name:            "zero repos",
			repoCount:       0,
			expectedMin:     0,
			expectedMax:     0,
			expectedBatches: 0,
		},
		{
			name:            "single repo",
			repoCount:       1,
			expectedMin:     4,
			expectedMax:     10,
			expectedBatches: 1,
		},
		{
			name:            "25 repos (one batch)",
			repoCount:       25,
			expectedMin:     100,
			expectedMax:     250,
			expectedBatches: 1,
		},
		{
			name:            "26 repos (two batches)",
			repoCount:       26,
			expectedMin:     104,
			expectedMax:     260,
			expectedBatches: 2,
		},
		{
			name:            "200 repos",
			repoCount:       200,
			expectedMin:     800,
			expectedMax:     2000,
			expectedBatches: 8,
		},
		{
			name:            "50 repos (two batches)",
			repoCount:       50,
			expectedMin:     200,
			expectedMax:     500,
			expectedBatches: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			estimate := EstimateSyncCost(tt.repoCount)
			assert.Equal(t, tt.expectedMin, estimate.MinCalls, "MinCalls")
			assert.Equal(t, tt.expectedMax, estimate.MaxCalls, "MaxCalls")
			assert.Equal(t, tt.expectedBatches, estimate.GraphQLBatches, "GraphQLBatches")
		})
	}
}

func TestWarnIfBudgetLow(t *testing.T) {
	// Not parallel: subtests use output.CaptureOutput() which swaps shared global state
	t.Run("nil info does not panic", func(t *testing.T) {
		estimate := EstimateSyncCost(10)
		// Should not panic
		WarnIfBudgetLow(nil, estimate)
	})

	t.Run("plenty of budget shows no warning", func(t *testing.T) {
		scope := output.CaptureOutput()
		defer scope.Restore()

		info := &RateLimitInfo{
			Limit:     5000,
			Remaining: 4500,
			ResetAt:   time.Now().Add(1 * time.Hour),
			Used:      500,
		}
		estimate := EstimateSyncCost(10) // MinCalls=40, MaxCalls=100

		WarnIfBudgetLow(info, estimate)

		assert.Empty(t, scope.Stderr.String(), "no warning when budget is plenty")
	})

	t.Run("tight budget shows warning", func(t *testing.T) {
		scope := output.CaptureOutput()
		defer scope.Restore()

		info := &RateLimitInfo{
			Limit:     5000,
			Remaining: 75, // Between MinCalls (40) and MaxCalls (100)
			ResetAt:   time.Now().Add(30 * time.Minute),
			Used:      4925,
		}
		estimate := EstimateSyncCost(10) // MinCalls=40, MaxCalls=100

		WarnIfBudgetLow(info, estimate)

		assert.Contains(t, scope.Stderr.String(), "tight")
	})

	t.Run("very low budget shows low warning", func(t *testing.T) {
		scope := output.CaptureOutput()
		defer scope.Restore()

		info := &RateLimitInfo{
			Limit:     5000,
			Remaining: 10, // Below MinCalls (40)
			ResetAt:   time.Now().Add(15 * time.Minute),
			Used:      4990,
		}
		estimate := EstimateSyncCost(10) // MinCalls=40, MaxCalls=100

		WarnIfBudgetLow(info, estimate)

		assert.Contains(t, scope.Stderr.String(), "low")
	})
}

func TestDisplayRateLimitInfo(t *testing.T) {
	// Not parallel: subtests use output.CaptureOutput() which swaps shared global state
	t.Run("nil info does not panic", func(t *testing.T) {
		// Should not panic
		DisplayRateLimitInfo(nil)
	})

	t.Run("displays rate limit info", func(t *testing.T) {
		scope := output.CaptureOutput()
		defer scope.Restore()

		info := &RateLimitInfo{
			Limit:     5000,
			Remaining: 4200,
			ResetAt:   time.Date(2026, 2, 17, 14, 30, 0, 0, time.UTC),
			Used:      800,
		}

		DisplayRateLimitInfo(info)

		out := scope.Stdout.String()
		assert.Contains(t, out, "4200/5000")
		assert.Contains(t, out, "14:30:00")
	})
}
