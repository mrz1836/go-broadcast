package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
)

func TestCheckRateLimit(t *testing.T) {
	t.Parallel()

	t.Run("successful rate limit check", func(t *testing.T) {
		t.Parallel()

		mockClient := gh.NewMockClient()
		resetTime := time.Now().Add(1 * time.Hour).Unix()

		resp := &gh.RateLimitResponse{}
		resp.Resources.Core.Limit = 5000
		resp.Resources.Core.Remaining = 4200
		resp.Resources.Core.Reset = resetTime
		resp.Resources.Core.Used = 800

		mockClient.On("GetRateLimit", mock.Anything).Return(resp, nil)

		info, err := CheckRateLimit(context.Background(), mockClient)
		require.NoError(t, err)
		require.NotNil(t, info)

		assert.Equal(t, 5000, info.Limit)
		assert.Equal(t, 4200, info.Remaining)
		assert.Equal(t, 800, info.Used)
		assert.Equal(t, time.Unix(resetTime, 0), info.ResetAt)

		mockClient.AssertExpectations(t)
	})

	t.Run("API error returns error", func(t *testing.T) {
		t.Parallel()

		mockClient := gh.NewMockClient()
		mockClient.On("GetRateLimit", mock.Anything).Return(nil, assert.AnError)

		info, err := CheckRateLimit(context.Background(), mockClient)
		require.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "failed to check rate limit")

		mockClient.AssertExpectations(t)
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
	t.Parallel()

	t.Run("nil info does not panic", func(t *testing.T) {
		t.Parallel()

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
	t.Parallel()

	t.Run("nil info does not panic", func(t *testing.T) {
		t.Parallel()

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
