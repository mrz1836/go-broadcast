package gh

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testCoreReset is the reset time of the core window in rateLimitSnapshot.
const testCoreReset = 1_700_000_000

// rateLimitSnapshot builds a RateLimitResponse with the given core and GraphQL usage,
// within the core window that resets at testCoreReset.
func rateLimitSnapshot(coreUsed, graphQLUsed int, graphQLReset int64) *RateLimitResponse {
	resp := &RateLimitResponse{}
	resp.Resources.Core = RateLimitResource{Limit: 5000, Remaining: 5000 - coreUsed, Reset: testCoreReset, Used: coreUsed}
	resp.Resources.GraphQL = RateLimitResource{Limit: 5000, Remaining: 5000 - graphQLUsed, Reset: graphQLReset, Used: graphQLUsed}
	return resp
}

// atCoreReset returns the time offset from testCoreReset, the end of the core window.
func atCoreReset(offset time.Duration) time.Time {
	return time.Unix(testCoreReset, 0).Add(offset)
}

func TestConsumedBetween(t *testing.T) {
	midWindow := atCoreReset(-30 * time.Minute)

	tests := []struct {
		name           string
		before         RateLimitResource
		after          RateLimitResource
		measuredAt     time.Time
		wantUsed       int
		wantLowerBound bool
	}{
		{
			name:       "same window reports the exact difference",
			before:     RateLimitResource{Used: 100, Reset: testCoreReset},
			after:      RateLimitResource{Used: 512, Reset: testCoreReset},
			measuredAt: midWindow,
			wantUsed:   412,
		},
		{
			name:       "same window with no usage reports zero",
			before:     RateLimitResource{Used: 42, Reset: testCoreReset},
			after:      RateLimitResource{Used: 42, Reset: testCoreReset},
			measuredAt: midWindow,
			wantUsed:   0,
		},
		{
			name:       "same window never reports negative usage",
			before:     RateLimitResource{Used: 50, Reset: testCoreReset},
			after:      RateLimitResource{Used: 40, Reset: testCoreReset},
			measuredAt: midWindow,
			wantUsed:   0,
		},
		{
			// Before any request, GitHub reports a reset a full window from "now".
			// A request two minutes later opened the window.
			name:       "window opened during the measurement is exact",
			before:     RateLimitResource{Used: 0, Reset: testCoreReset},
			after:      RateLimitResource{Used: 87, Reset: testCoreReset + 120},
			measuredAt: atCoreReset(-58 * time.Minute),
			wantUsed:   87,
		},
		{
			// The old window ended, and a request five minutes later opened the next.
			name:           "window reset during the measurement is a lower bound",
			before:         RateLimitResource{Used: 4200, Reset: testCoreReset},
			after:          RateLimitResource{Used: 30, Reset: testCoreReset + 300 + 3600},
			measuredAt:     atCoreReset(10 * time.Minute),
			wantUsed:       30,
			wantLowerBound: true,
		},
		{
			name:           "window reset within the clock tolerance is a lower bound",
			before:         RateLimitResource{Used: 4200, Reset: testCoreReset},
			after:          RateLimitResource{Used: 30, Reset: testCoreReset + 3600 - 30},
			measuredAt:     atCoreReset(-30 * time.Second),
			wantUsed:       30,
			wantLowerBound: true,
		},
		{
			// A 28-second sync in the middle of a window was once reported as "at
			// least" the whole window's usage because its reset time moved a second.
			name:       "reset time that moved mid-window is the same window",
			before:     RateLimitResource{Used: 598, Reset: testCoreReset - 1},
			after:      RateLimitResource{Used: 657, Reset: testCoreReset},
			measuredAt: atCoreReset(-27 * time.Minute),
			wantUsed:   59,
		},
		{
			name:       "reset time reported by another region is the same window",
			before:     RateLimitResource{Used: 598, Reset: testCoreReset},
			after:      RateLimitResource{Used: 657, Reset: testCoreReset + 2700},
			measuredAt: atCoreReset(-27 * time.Minute),
			wantUsed:   59,
		},
		{
			name:       "reset a full window later is the same window until the old one ends",
			before:     RateLimitResource{Used: 598, Reset: testCoreReset},
			after:      RateLimitResource{Used: 657, Reset: testCoreReset + 3600},
			measuredAt: atCoreReset(-27 * time.Minute),
			wantUsed:   59,
		},
		{
			name:       "reset too soon to follow the ended window is the same window",
			before:     RateLimitResource{Used: 598, Reset: testCoreReset},
			after:      RateLimitResource{Used: 657, Reset: testCoreReset + 600},
			measuredAt: atCoreReset(time.Minute),
			wantUsed:   59,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			used, lowerBound := consumedBetween(tc.before, tc.after, tc.measuredAt)
			assert.Equal(t, tc.wantUsed, used)
			assert.Equal(t, tc.wantLowerBound, lowerBound)
		})
	}
}

func TestUsageBetweenCombinesBuckets(t *testing.T) {
	before := rateLimitSnapshot(1000, 10, 1_700_000_500)
	after := rateLimitSnapshot(1250, 13, 1_700_000_500)

	usage := usageBetween(before, after, atCoreReset(-30*time.Minute))

	assert.Equal(t, 250, usage.CoreRequests)
	assert.Equal(t, 3, usage.GraphQLPoints)
	assert.False(t, usage.CoreLowerBound)
	assert.False(t, usage.GraphQLLowerBound)
	assert.Equal(t, 3750, usage.CoreRemaining)
	assert.Equal(t, 5000, usage.CoreLimit)
	assert.Equal(t, time.Unix(testCoreReset, 0), usage.CoreResetAt)
}

func TestUsageBetweenFlagsLowerBoundPerBucket(t *testing.T) {
	// The GraphQL window ended 25 minutes before the core window, and a request a
	// minute later opened the next one, so only the GraphQL count is a minimum; the
	// core window is unchanged and the REST count stays exact.
	graphQLReset := int64(testCoreReset - 1500)
	before := rateLimitSnapshot(1000, 400, graphQLReset)
	after := rateLimitSnapshot(1010, 2, graphQLReset+60+3600)

	usage := usageBetween(before, after, atCoreReset(-20*time.Minute))

	assert.Equal(t, 10, usage.CoreRequests)
	assert.False(t, usage.CoreLowerBound)
	assert.Equal(t, 2, usage.GraphQLPoints)
	assert.True(t, usage.GraphQLLowerBound)
}

func TestAPIUsageString(t *testing.T) {
	tests := []struct {
		name  string
		usage APIUsage
		want  string
	}{
		{name: "rest only", usage: APIUsage{CoreRequests: 412}, want: "412 REST requests"},
		{name: "singular request", usage: APIUsage{CoreRequests: 1}, want: "1 REST request"},
		{name: "no requests", usage: APIUsage{}, want: "0 REST requests"},
		{name: "rest and graphql", usage: APIUsage{CoreRequests: 2, GraphQLPoints: 3}, want: "2 REST requests, 3 GraphQL points"},
		{name: "singular point", usage: APIUsage{CoreRequests: 2, GraphQLPoints: 1}, want: "2 REST requests, 1 GraphQL point"},
		{name: "rest lower bound", usage: APIUsage{CoreRequests: 30, CoreLowerBound: true}, want: "at least 30 REST requests"},
		{
			name:  "graphql lower bound",
			usage: APIUsage{CoreRequests: 30, GraphQLPoints: 3, GraphQLLowerBound: true},
			want:  "30 REST requests, at least 3 GraphQL points",
		},
		{
			name:  "unused graphql is omitted even after its window reset",
			usage: APIUsage{CoreRequests: 30, GraphQLLowerBound: true},
			want:  "30 REST requests",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.usage.String())
		})
	}
}

func TestUsageMeterMeasuresFromBaseline(t *testing.T) {
	ctx := context.Background()
	client := &MockClient{}
	client.On("GetRateLimit", mock.Anything).Return(rateLimitSnapshot(100, 0, 1_700_000_000), nil).Once()
	client.On("GetRateLimit", mock.Anything).Return(rateLimitSnapshot(140, 0, 1_700_000_000), nil).Once()
	client.On("GetRateLimit", mock.Anything).Return(rateLimitSnapshot(175, 0, 1_700_000_000), nil).Once()

	meter, err := StartUsageMeter(ctx, client)
	require.NoError(t, err)
	require.NotNil(t, meter.Baseline())
	assert.Equal(t, 100, meter.Baseline().Resources.Core.Used)

	first, err := meter.Measure(ctx)
	require.NoError(t, err)
	assert.Equal(t, 40, first.CoreRequests)

	// A second measurement is still relative to the original baseline.
	second, err := meter.Measure(ctx)
	require.NoError(t, err)
	assert.Equal(t, 75, second.CoreRequests)

	client.AssertExpectations(t)
}

// TestUsageMeterJudgesWindowsAtMeasurementTime verifies Measure decides whether a
// new window began as of the time it takes the second snapshot.
func TestUsageMeterJudgesWindowsAtMeasurementTime(t *testing.T) {
	ctx := context.Background()
	after := rateLimitSnapshot(657, 0, testCoreReset)
	after.Resources.Core.Reset = testCoreReset + 3600 // A region reporting a later window

	client := &MockClient{}
	client.On("GetRateLimit", mock.Anything).Return(rateLimitSnapshot(598, 0, testCoreReset), nil).Once()
	client.On("GetRateLimit", mock.Anything).Return(after, nil).Once()

	meter, err := StartUsageMeter(ctx, client)
	require.NoError(t, err)
	meter.clock = func() time.Time { return atCoreReset(-27 * time.Minute) }

	usage, err := meter.Measure(ctx)
	require.NoError(t, err)

	// The baseline's window had 27 minutes left, so no new window can have begun.
	assert.Equal(t, 59, usage.CoreRequests)
	assert.False(t, usage.CoreLowerBound)
	client.AssertExpectations(t)
}

func TestStartUsageMeterRateLimitUnavailable(t *testing.T) {
	ctx := context.Background()

	t.Run("request error", func(t *testing.T) {
		client := &MockClient{}
		client.On("GetRateLimit", mock.Anything).Return(nil, assert.AnError)

		meter, err := StartUsageMeter(ctx, client)
		require.ErrorIs(t, err, ErrRateLimitUnavailable)
		require.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, meter)
	})

	t.Run("empty response", func(t *testing.T) {
		client := &MockClient{}
		client.On("GetRateLimit", mock.Anything).Return(nil, nil)

		meter, err := StartUsageMeter(ctx, client)
		require.ErrorIs(t, err, ErrRateLimitUnavailable)
		assert.Nil(t, meter)
	})
}

func TestUsageMeterMeasureDetachedIgnoresCancellation(t *testing.T) {
	client := &MockClient{}
	client.On("GetRateLimit", mock.Anything).Return(rateLimitSnapshot(10, 0, 1_700_000_000), nil).Once()
	client.On("GetRateLimit", mock.MatchedBy(func(ctx context.Context) bool {
		_, hasDeadline := ctx.Deadline()
		return ctx.Err() == nil && hasDeadline
	})).Return(rateLimitSnapshot(15, 0, 1_700_000_000), nil).Once()

	ctx, cancel := context.WithCancel(context.Background())
	meter, err := StartUsageMeter(ctx, client)
	require.NoError(t, err)
	cancel() // The measured operation was interrupted

	usage, err := meter.MeasureDetached(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, usage.CoreRequests)
	client.AssertExpectations(t)
}

func TestUsageMeterMeasureRateLimitUnavailable(t *testing.T) {
	ctx := context.Background()
	client := &MockClient{}
	client.On("GetRateLimit", mock.Anything).Return(rateLimitSnapshot(10, 0, 1_700_000_000), nil).Once()
	client.On("GetRateLimit", mock.Anything).Return(nil, assert.AnError).Once()

	meter, err := StartUsageMeter(ctx, client)
	require.NoError(t, err)

	usage, err := meter.Measure(ctx)
	require.ErrorIs(t, err, ErrRateLimitUnavailable)
	assert.Equal(t, APIUsage{}, usage)
}

// TestGetRateLimitParsesCoreAndGraphQL verifies both buckets are decoded from a
// GET /rate_limit response shaped like GitHub's.
func TestGetRateLimitParsesCoreAndGraphQL(t *testing.T) {
	ctx := context.Background()
	runner := NewMockCommandRunner()
	client := NewClientWithRunner(runner, logrus.New())

	body := []byte(`{
		"resources": {
			"core": {"limit": 5000, "used": 1827, "remaining": 3173, "reset": 1792534769},
			"graphql": {"limit": 5000, "used": 2, "remaining": 4998, "reset": 1792531803},
			"search": {"limit": 30, "used": 0, "remaining": 30, "reset": 1792531863}
		},
		"rate": {"limit": 5000, "used": 1827, "remaining": 3173, "reset": 1792534769}
	}`)
	runner.On("Run", ctx, "gh", []string{"api", "rate_limit"}).Return(body, nil)

	resp, err := client.GetRateLimit(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, RateLimitResource{Limit: 5000, Remaining: 3173, Reset: 1792534769, Used: 1827}, resp.Resources.Core)
	assert.Equal(t, RateLimitResource{Limit: 5000, Remaining: 4998, Reset: 1792531803, Used: 2}, resp.Resources.GraphQL)
	runner.AssertExpectations(t)
}
