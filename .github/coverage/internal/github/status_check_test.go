package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusCheckManager_parseCoverageOverrideFromLabels(t *testing.T) {
	tests := []struct {
		name                     string
		labels                   []Label
		config                   *StatusCheckConfig
		expectedThreshold        float64
		expectedHasOverride      bool
		expectedOverrideDetected bool
	}{
		{
			name:   "no labels",
			labels: []Label{},
			config: &StatusCheckConfig{
				AllowLabelOverride:   true,
				MinOverrideThreshold: 50.0,
				MaxOverrideThreshold: 95.0,
			},
			expectedThreshold:   0,
			expectedHasOverride: false,
		},
		{
			name: "generic coverage-override label",
			labels: []Label{
				{Name: "coverage-override", Color: "ff8c00"},
			},
			config: &StatusCheckConfig{
				AllowLabelOverride: true,
			},
			expectedThreshold:   0.0, // Completely ignores coverage
			expectedHasOverride: true,
		},
		{
			name: "label override disabled",
			labels: []Label{
				{Name: "coverage-override", Color: "ff8c00"},
			},
			config: &StatusCheckConfig{
				AllowLabelOverride: false,
			},
			expectedThreshold:   0,
			expectedHasOverride: false,
		},
		{
			name: "mixed labels with valid override",
			labels: []Label{
				{Name: "bug", Color: "d73a4a"},
				{Name: "coverage-override", Color: "ff8c00"},
				{Name: "documentation", Color: "0075ca"},
			},
			config: &StatusCheckConfig{
				AllowLabelOverride: true,
			},
			expectedThreshold:   0.0, // Completely ignores coverage
			expectedHasOverride: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &StatusCheckManager{
				config: tt.config,
			}

			threshold, hasOverride := manager.parseCoverageOverrideFromLabels(tt.labels)

			assert.Equal(t, tt.expectedHasOverride, hasOverride, "hasOverride mismatch")
			if tt.expectedHasOverride {
				assert.InDelta(t, tt.expectedThreshold, threshold, 0.01, "threshold mismatch")
			}
		})
	}
}

func TestStatusCheckManager_buildMainCoverageStatus_withoutPR(t *testing.T) {
	tests := []struct {
		name             string
		config           *StatusCheckConfig
		request          *StatusCheckRequest
		expectedState    string
		expectedContains []string
	}{
		{
			name: "coverage above threshold - no override",
			config: &StatusCheckConfig{
				CoverageThreshold:  80.0,
				AllowLabelOverride: false,
				BlockOnFailure:     true,
				MainContext:        "coverage/total",
				IncludeTargetURLs:  false,
			},
			request: &StatusCheckRequest{
				Coverage: CoverageStatusData{
					Percentage: 85.0,
					Change:     2.5,
				},
				PRNumber: 0, // No PR context
			},
			expectedState:    StatusStateSuccess,
			expectedContains: []string{"85.0%", "✅", "80.0%"},
		},
		{
			name: "coverage below threshold - no override",
			config: &StatusCheckConfig{
				CoverageThreshold:  80.0,
				AllowLabelOverride: false,
				BlockOnFailure:     true,
				MainContext:        "coverage/total",
				IncludeTargetURLs:  false,
			},
			request: &StatusCheckRequest{
				Coverage: CoverageStatusData{
					Percentage: 75.0,
					Change:     -1.2,
				},
				PRNumber: 0,
			},
			expectedState:    StatusStateFailure,
			expectedContains: []string{"75.0%", "⚠️", "80.0%"},
		},
		{
			name: "coverage with trend information",
			config: &StatusCheckConfig{
				CoverageThreshold:  80.0,
				AllowLabelOverride: false,
				BlockOnFailure:     false,
				MainContext:        "coverage/total",
				IncludeTargetURLs:  false,
			},
			request: &StatusCheckRequest{
				Coverage: CoverageStatusData{
					Percentage: 75.0,
					Change:     -2.5,
				},
				PRNumber: 0,
			},
			expectedState:    StatusStateSuccess, // Not blocking on failure
			expectedContains: []string{"75.0%", "⚠️", "80.0%", "-2.5%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a manager with minimal client (not used for these tests)
			manager := &StatusCheckManager{
				client: nil, // Not needed for these tests since PRNumber is 0
				config: tt.config,
			}

			status := manager.buildMainCoverageStatus(context.Background(), tt.request)

			assert.Equal(t, tt.expectedState, status.State, "state mismatch")
			assert.Equal(t, tt.config.MainContext, status.Context, "context mismatch")

			for _, expected := range tt.expectedContains {
				assert.Contains(t, status.Description, expected, "description should contain: %s", expected)
			}
		})
	}
}
