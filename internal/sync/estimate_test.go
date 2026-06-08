package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// makeGroup builds a group with n targets for estimator tests.
func makeGroup(name, id string, enabled *bool, n int) config.Group {
	targets := make([]config.TargetConfig, n)
	for i := 0; i < n; i++ {
		targets[i] = config.TargetConfig{Repo: name + "/target"}
	}
	return config.Group{
		Name:    name,
		ID:      id,
		Enabled: enabled,
		Targets: targets,
	}
}

// expectedEstimate computes the expected RunEstimate for a given resolved target
// count using the same conservative per-target constants as the implementation.
func expectedEstimate(targets int) RunEstimate {
	contentWrites := targets * contentWritesPerTarget
	reads := targets * readsPerTarget
	return RunEstimate{
		Targets:              targets,
		PrimaryRequests:      reads + contentWrites,
		ContentWriteRequests: contentWrites,
		PRCreatePoints:       contentWrites * PRCreatePointCost,
	}
}

func TestEstimateRun(t *testing.T) {
	tests := []struct {
		name            string
		cfg             *config.Config
		options         *Options
		expectedTargets int
	}{
		{
			name:            "nil config yields zero estimate",
			cfg:             nil,
			options:         nil,
			expectedTargets: 0,
		},
		{
			name:            "no groups yields zero estimate",
			cfg:             &config.Config{},
			options:         nil,
			expectedTargets: 0,
		},
		{
			name: "single group sums its targets",
			cfg: &config.Config{
				Groups: []config.Group{makeGroup("group1", "g1", nil, 3)},
			},
			options:         nil,
			expectedTargets: 3,
		},
		{
			name: "multi group sums across all groups",
			cfg: &config.Config{
				Groups: []config.Group{
					makeGroup("group1", "g1", nil, 3),
					makeGroup("group2", "g2", nil, 2),
					makeGroup("group3", "g3", nil, 5),
				},
			},
			options:         nil,
			expectedTargets: 10,
		},
		{
			name: "group filter retains only matching groups",
			cfg: &config.Config{
				Groups: []config.Group{
					makeGroup("group1", "g1", nil, 3),
					makeGroup("group2", "g2", nil, 4),
					makeGroup("group3", "g3", nil, 5),
				},
			},
			options:         (&Options{}).WithGroupFilter([]string{"group2", "g3"}),
			expectedTargets: 9, // group2 (4) + group3 (5)
		},
		{
			name: "skip groups excludes matching groups",
			cfg: &config.Config{
				Groups: []config.Group{
					makeGroup("group1", "g1", nil, 3),
					makeGroup("group2", "g2", nil, 4),
					makeGroup("group3", "g3", nil, 5),
				},
			},
			options:         (&Options{}).WithSkipGroups([]string{"g2"}),
			expectedTargets: 8, // group1 (3) + group3 (5)
		},
		{
			name: "disabled groups are excluded",
			cfg: &config.Config{
				Groups: []config.Group{
					makeGroup("group1", "g1", boolPtr(true), 3),
					makeGroup("group2", "g2", boolPtr(false), 4),
					makeGroup("group3", "g3", nil, 5),
				},
			},
			options:         nil,
			expectedTargets: 8, // group1 (3, enabled) + group3 (5, nil=enabled); group2 disabled
		},
		{
			name: "skip then enabled filter compose",
			cfg: &config.Config{
				Groups: []config.Group{
					makeGroup("group1", "g1", boolPtr(false), 3),
					makeGroup("group2", "g2", nil, 4),
					makeGroup("group3", "g3", nil, 5),
				},
			},
			options:         (&Options{}).WithSkipGroups([]string{"group3"}),
			expectedTargets: 4, // group1 disabled, group3 skipped → only group2 (4)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateRun(tc.cfg, tc.options)
			assert.Equal(t, expectedEstimate(tc.expectedTargets), got)
		})
	}
}

// TestEstimateRunDeterministic verifies the estimate is stable across repeated
// calls with identical inputs (no hidden state / ordering effects).
func TestEstimateRunDeterministic(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.Group{
			makeGroup("group1", "g1", nil, 3),
			makeGroup("group2", "g2", nil, 7),
		},
	}

	first := EstimateRun(cfg, nil)
	for i := 0; i < 5; i++ {
		assert.Equal(t, first, EstimateRun(cfg, nil))
	}
}

// TestEstimateRunConservative documents the conservative cost model: every
// resolved target contributes at least one content-creation write (the PR
// create/update), and content writes are a subset of primary requests.
func TestEstimateRunConservative(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.Group{makeGroup("group1", "g1", nil, 4)},
	}

	est := EstimateRun(cfg, nil)

	assert.Equal(t, 4, est.Targets)
	assert.GreaterOrEqual(t, est.ContentWriteRequests, est.Targets,
		"each target should contribute at least one content-creation write")
	assert.Greater(t, est.PrimaryRequests, est.ContentWriteRequests,
		"primary requests should include reads on top of content writes")
	assert.Equal(t, est.ContentWriteRequests*PRCreatePointCost, est.PRCreatePoints,
		"PR-create points should map content writes at the documented write point cost")
}

// TestRateLimitConstants is a guard that the documented GitHub limit constants
// keep their published values; a change here should only follow a docs revision.
func TestRateLimitConstants(t *testing.T) {
	assert.Equal(t, 5000, PrimaryHourlyLimit)
	assert.Equal(t, 80, SecondaryContentPerMinute)
	assert.Equal(t, 500, SecondaryContentPerHour)
	assert.Equal(t, 900, SecondaryPointsPerMinute)
	assert.Equal(t, 1, PointCostRead)
	assert.Equal(t, 5, PointCostWrite)
	assert.Equal(t, 5, PRCreatePointCost)
	assert.Equal(t, 90, SecondaryCPUSecondsPer60s)
	assert.Equal(t, 100, MaxConcurrentRequests)
}
