package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// defaultPreflightConfig returns the documented-default preflight knobs.
func defaultPreflightConfig() PreflightConfig {
	return PreflightConfig{
		Enabled:              true,
		PrimaryMarginPercent: 20,
		SecondaryReserve:     10,
		FailClosed:           false,
	}
}

func TestPreflightDecide(t *testing.T) {
	reset := time.Unix(1_900_000_000, 0)

	tests := []struct {
		name             string
		estimate         RunEstimate
		primaryRemaining int
		cfg              PreflightConfig
		wantProceed      bool
		wantReasonSubstr string
	}{
		{
			name:             "under budget proceeds",
			estimate:         RunEstimate{PrimaryRequests: 50, ContentWriteRequests: 5},
			primaryRemaining: 5000,
			cfg:              defaultPreflightConfig(),
			wantProceed:      true,
		},
		{
			name:             "over primary halts",
			estimate:         RunEstimate{PrimaryRequests: 4200, ContentWriteRequests: 5},
			primaryRemaining: 5000, // margin = ceil(5000*0.2)=1000 -> available 4000
			cfg:              defaultPreflightConfig(),
			wantProceed:      false,
			wantReasonSubstr: "primary budget",
		},
		{
			name:             "primary boundary exactly at available proceeds",
			estimate:         RunEstimate{PrimaryRequests: 4000, ContentWriteRequests: 5},
			primaryRemaining: 5000, // available = 5000-1000 = 4000; 4000 > 4000 is false
			cfg:              defaultPreflightConfig(),
			wantProceed:      true,
		},
		{
			name:             "primary one over available halts",
			estimate:         RunEstimate{PrimaryRequests: 4001, ContentWriteRequests: 5},
			primaryRemaining: 5000,
			cfg:              defaultPreflightConfig(),
			wantProceed:      false,
			wantReasonSubstr: "primary budget",
		},
		{
			name:             "over secondary per-minute halts",
			estimate:         RunEstimate{PrimaryRequests: 100, ContentWriteRequests: 71},
			primaryRemaining: 5000, // per-minute available = 80-10 = 70; 71 > 70
			cfg:              defaultPreflightConfig(),
			wantProceed:      false,
			wantReasonSubstr: "per-minute",
		},
		{
			name:             "secondary per-minute boundary proceeds",
			estimate:         RunEstimate{PrimaryRequests: 100, ContentWriteRequests: 70},
			primaryRemaining: 5000, // 70 > 70 is false
			cfg:              defaultPreflightConfig(),
			wantProceed:      true,
		},
		{
			name:             "over secondary per-hour halts",
			estimate:         RunEstimate{PrimaryRequests: 600, ContentWriteRequests: 501},
			primaryRemaining: 50000, // huge primary so only per-hour trips (501 > 500)
			cfg:              PreflightConfig{Enabled: true, PrimaryMarginPercent: 0, SecondaryReserve: 0},
			wantProceed:      false,
			wantReasonSubstr: "per-hour",
		},
		{
			name:             "zero reserve allows full per-minute cap",
			estimate:         RunEstimate{PrimaryRequests: 100, ContentWriteRequests: 80},
			primaryRemaining: 5000,
			cfg:              PreflightConfig{Enabled: true, PrimaryMarginPercent: 20, SecondaryReserve: 0},
			wantProceed:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := decide(tt.estimate, tt.primaryRemaining, reset, tt.cfg)
			assert.Equal(t, tt.wantProceed, d.Proceed)
			if tt.wantProceed {
				assert.Empty(t, d.Reason)
			} else {
				assert.NotEmpty(t, d.Reason)
				assert.Contains(t, d.Reason, tt.wantReasonSubstr)
			}
			// Summary fields are always populated.
			assert.Equal(t, tt.estimate.PrimaryRequests, d.EstimatedPrimaryRequests)
			assert.Equal(t, tt.primaryRemaining, d.PrimaryRemaining)
			assert.Equal(t, tt.estimate.ContentWriteRequests, d.EstimatedContentWrites)
			assert.Equal(t, SecondaryContentPerMinute, d.SecondaryPerMinuteCap)
			assert.Equal(t, SecondaryContentPerHour, d.SecondaryPerHourCap)
			assert.Equal(t, reset, d.ResetAt)
		})
	}
}

func TestPreflightDecideMultipleReasons(t *testing.T) {
	// Both primary and secondary over budget: reason mentions both.
	d := decide(
		RunEstimate{PrimaryRequests: 4500, ContentWriteRequests: 100},
		5000,
		time.Time{},
		defaultPreflightConfig(),
	)
	require.False(t, d.Proceed)
	assert.Contains(t, d.Reason, "primary budget")
	assert.Contains(t, d.Reason, "per-minute")
}

func TestDecisionSummary(t *testing.T) {
	reset := time.Unix(1_900_000_000, 0)
	d := decide(
		RunEstimate{PrimaryRequests: 50, ContentWriteRequests: 5},
		5000,
		reset,
		defaultPreflightConfig(),
	)
	summary := d.Summary()
	assert.Contains(t, summary, "estimated 50 primary requests")
	assert.Contains(t, summary, "5000 remaining")
	assert.Contains(t, summary, "5 content writes")
	assert.Contains(t, summary, "resets at")
}

func TestDecisionSummaryNoResetWhenZero(t *testing.T) {
	d := decide(
		RunEstimate{PrimaryRequests: 50, ContentWriteRequests: 5},
		5000,
		time.Time{},
		defaultPreflightConfig(),
	)
	assert.NotContains(t, d.Summary(), "resets at")
}

func TestPreflightProbeSuccess(t *testing.T) {
	mockClient := new(gh.MockClient)
	resp := &gh.RateLimitResponse{}
	resp.Resources.Core.Remaining = 4321
	resp.Resources.Core.Reset = 1_900_000_000
	mockClient.On("GetRateLimit", mock.Anything).Return(resp, nil)

	p := NewPreflight(mockClient, defaultPreflightConfig())
	remaining, reset, err := p.probe(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 4321, remaining)
	assert.Equal(t, time.Unix(1_900_000_000, 0), reset)
	mockClient.AssertExpectations(t)
}

func TestPreflightProbeError(t *testing.T) {
	mockClient := new(gh.MockClient)
	mockClient.On("GetRateLimit", mock.Anything).Return(nil, assert.AnError)

	p := NewPreflight(mockClient, defaultPreflightConfig())
	_, _, err := p.probe(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRateLimitProbeUnavailable)
}

func TestPreflightProbeNilResponse(t *testing.T) {
	mockClient := new(gh.MockClient)
	mockClient.On("GetRateLimit", mock.Anything).Return(nil, nil)

	p := NewPreflight(mockClient, defaultPreflightConfig())
	_, _, err := p.probe(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRateLimitProbeUnavailable)
}

func TestPreflightConfigAccessor(t *testing.T) {
	cfg := defaultPreflightConfig()
	p := NewPreflight(new(gh.MockClient), cfg)
	assert.Equal(t, cfg, p.Config())
}
