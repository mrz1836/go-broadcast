package sync

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// rateLimitWithCoreUsed returns a rate-limit snapshot within one fixed window.
func rateLimitWithCoreUsed(used int) *gh.RateLimitResponse {
	resp := &gh.RateLimitResponse{}
	resp.Resources.Core = gh.RateLimitResource{Limit: 5000, Remaining: 5000 - used, Used: used, Reset: 1_900_000_000}
	resp.Resources.GraphQL = gh.RateLimitResource{Limit: 5000, Remaining: 5000, Reset: 1_900_000_000}
	return resp
}

func TestEngineStartUsageMeter(t *testing.T) {
	t.Run("nil client is not measured", func(t *testing.T) {
		engine := &Engine{logger: logrus.New()}
		assert.Nil(t, engine.startUsageMeter(context.Background()))
	})

	t.Run("unreadable rate limit is not measured", func(t *testing.T) {
		client := &gh.MockClient{}
		client.On("GetRateLimit", mock.Anything).Return(nil, assert.AnError).Once()

		engine := &Engine{gh: client, logger: logrus.New()}
		assert.Nil(t, engine.startUsageMeter(context.Background()))
		client.AssertExpectations(t)
	})
}

func TestEngineMeasureAPIUsage(t *testing.T) {
	t.Run("nil meter is not measured", func(t *testing.T) {
		engine := &Engine{logger: logrus.New()}
		assert.Nil(t, engine.measureAPIUsage(context.Background(), nil))
	})

	t.Run("measures the requests since the meter started", func(t *testing.T) {
		client := &gh.MockClient{}
		client.On("GetRateLimit", mock.Anything).Return(rateLimitWithCoreUsed(100), nil).Once()
		client.On("GetRateLimit", mock.Anything).Return(rateLimitWithCoreUsed(137), nil).Once()

		engine := &Engine{gh: client, logger: logrus.New()}
		meter := engine.startUsageMeter(context.Background())
		require.NotNil(t, meter)

		usage := engine.measureAPIUsage(context.Background(), meter)
		require.NotNil(t, usage)
		assert.Equal(t, 37, usage.CoreRequests)
		assert.Equal(t, 4863, usage.CoreRemaining)
		client.AssertExpectations(t)
	})

	t.Run("still measures after the sync context is canceled", func(t *testing.T) {
		client := &gh.MockClient{}
		client.On("GetRateLimit", mock.Anything).Return(rateLimitWithCoreUsed(10), nil).Once()
		client.On("GetRateLimit", mock.MatchedBy(func(ctx context.Context) bool {
			return ctx.Err() == nil
		})).Return(rateLimitWithCoreUsed(15), nil).Once()

		engine := &Engine{gh: client, logger: logrus.New()}
		ctx, cancel := context.WithCancel(context.Background())
		meter := engine.startUsageMeter(ctx)
		require.NotNil(t, meter)
		cancel()

		usage := engine.measureAPIUsage(ctx, meter)
		require.NotNil(t, usage)
		assert.Equal(t, 5, usage.CoreRequests)
		client.AssertExpectations(t)
	})

	t.Run("unreadable closing snapshot is not measured", func(t *testing.T) {
		client := &gh.MockClient{}
		client.On("GetRateLimit", mock.Anything).Return(rateLimitWithCoreUsed(10), nil).Once()
		client.On("GetRateLimit", mock.Anything).Return(nil, assert.AnError).Once()

		engine := &Engine{gh: client, logger: logrus.New()}
		meter := engine.startUsageMeter(context.Background())
		require.NotNil(t, meter)

		assert.Nil(t, engine.measureAPIUsage(context.Background(), meter))
		client.AssertExpectations(t)
	})
}

func TestEngineRecordRunAPIUsage(t *testing.T) {
	usage := &gh.APIUsage{CoreRequests: 42, CoreLowerBound: true}

	t.Run("a GraphQL window reset does not mark the REST count as a minimum", func(t *testing.T) {
		engine := &Engine{}
		run := &BroadcastSyncRun{ExternalID: "SR-20260101-abc123"}
		engine.setCurrentRun(run)

		engine.recordRunAPIUsage(&gh.APIUsage{CoreRequests: 42, GraphQLLowerBound: true})

		assert.Equal(t, 42, run.APICallsMade)
		assert.False(t, run.APIUsageLowerBound)
	})

	t.Run("stores usage on the current run", func(t *testing.T) {
		engine := &Engine{}
		run := &BroadcastSyncRun{ExternalID: "SR-20260101-abc123"}
		engine.setCurrentRun(run)

		engine.recordRunAPIUsage(usage)

		assert.Equal(t, 42, run.APICallsMade)
		assert.True(t, run.APIUsageLowerBound)
	})

	t.Run("unmeasured usage leaves the run untouched", func(t *testing.T) {
		engine := &Engine{}
		run := &BroadcastSyncRun{ExternalID: "SR-20260101-abc123"}
		engine.setCurrentRun(run)

		engine.recordRunAPIUsage(nil)

		assert.Zero(t, run.APICallsMade)
		assert.False(t, run.APIUsageLowerBound)
	})

	t.Run("no current run is a no-op", func(t *testing.T) {
		engine := &Engine{}
		assert.NotPanics(t, func() { engine.recordRunAPIUsage(usage) })
	})
}

func TestPrintAPIUsage(t *testing.T) {
	t.Run("prints usage and remaining budget", func(t *testing.T) {
		scope := output.CaptureOutput()
		defer scope.Restore()

		resetAt := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)
		printAPIUsage("core", &gh.APIUsage{CoreRequests: 23, CoreRemaining: 4977, CoreLimit: 5000, CoreResetAt: resetAt})

		assert.Contains(t, scope.Stdout.String(),
			`GitHub API usage for group "core": 23 REST requests (4977/5000 REST requests remaining, resets at 15:04:05)`)
	})

	t.Run("unmeasured usage prints nothing", func(t *testing.T) {
		scope := output.CaptureOutput()
		defer scope.Restore()

		printAPIUsage("core", nil)

		assert.Empty(t, scope.Stdout.String())
	})
}

// TestExecuteSingleGroupRecordsAPIUsage verifies a group sync measures its API
// usage from before state discovery and persists it on the sync run record.
func TestExecuteSingleGroupRecordsAPIUsage(t *testing.T) {
	scope := output.CaptureOutput()
	defer scope.Restore()

	group := config.Group{
		ID:      "core",
		Name:    "core",
		Source:  config.SourceConfig{Repo: "org/template", Branch: "master"},
		Targets: []config.TargetConfig{{Repo: "org/target"}},
	}
	cfg := &config.Config{Groups: []config.Group{group}}

	currentState := &state.State{
		Source: state.SourceState{Repo: "org/template", Branch: "master", LatestCommit: "abc123"},
		Targets: map[string]*state.TargetState{
			"org/target": {Repo: "org/target", Status: state.StatusUpToDate, LastSyncCommit: "abc123"},
		},
	}

	ghClient := &gh.MockClient{}
	ghClient.On("GetRateLimit", mock.Anything).Return(rateLimitWithCoreUsed(1000), nil).Once()
	ghClient.On("GetRateLimit", mock.Anything).Return(rateLimitWithCoreUsed(1004), nil).Once()

	discoverer := &state.MockDiscoverer{}
	discoverer.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil).Once()

	recorder := &mockSyncRecorder{}
	recorder.On("LookupGroupID", mock.Anything, "core").Return(uint(0), errCoverageBoost)
	recorder.On("LookupRepoID", mock.Anything, "org/template").Return(uint(0), errCoverageBoost)
	recorder.On("CreateSyncRun", mock.Anything, mock.Anything).Return(nil).Once()
	recorder.On("UpdateSyncRun", mock.Anything, mock.MatchedBy(func(run *BroadcastSyncRun) bool {
		return run.APICallsMade == 4 && !run.APIUsageLowerBound
	})).Return(nil).Once()

	engine := &Engine{
		config:  cfg,
		gh:      ghClient,
		state:   discoverer,
		options: DefaultOptions(),
		logger:  logrus.New(),
	}
	engine.SetSyncMetricsRecorder(recorder)

	require.NoError(t, engine.executeSingleGroup(context.Background(), group, nil))

	assert.Contains(t, scope.Stdout.String(), `GitHub API usage for group "core": 4 REST requests`)
	ghClient.AssertExpectations(t)
	discoverer.AssertExpectations(t)
	recorder.AssertExpectations(t)
}
