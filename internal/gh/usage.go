package gh

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrRateLimitUnavailable indicates the GitHub rate-limit status could not be read,
// so API usage cannot be measured.
var ErrRateLimitUnavailable = errors.New("rate limit status unavailable")

// detachedMeasureTimeout bounds the rate-limit request made by MeasureDetached.
const detachedMeasureTimeout = 10 * time.Second

// APIUsage is the GitHub API quota consumed between two rate-limit snapshots.
//
// The counts are GitHub's own accounting for the authenticated identity, which
// makes them exact for pagination and multi-request gh commands, but also means
// requests made at the same time by other tools sharing the token are included.
type APIUsage struct {
	// CoreRequests is the number of REST requests charged to the primary rate limit.
	CoreRequests int

	// GraphQLPoints is the number of points charged to the GraphQL rate limit.
	GraphQLPoints int

	// CoreLowerBound and GraphQLLowerBound report that the bucket's rate-limit
	// window reset between the snapshots. Requests made in the old window after
	// the first snapshot cannot be recovered, so that count is a minimum.
	CoreLowerBound    bool
	GraphQLLowerBound bool

	// CoreRemaining, CoreLimit, and CoreResetAt describe the REST budget as of
	// the second snapshot.
	CoreRemaining int
	CoreLimit     int
	CoreResetAt   time.Time
}

// String formats the usage for display, e.g. "412 REST requests, 3 GraphQL points".
// GraphQL points are omitted when none were used, and a count that is only a
// minimum is prefixed with "at least".
func (u APIUsage) String() string {
	var sb strings.Builder
	writeCount(&sb, u.CoreRequests, u.CoreLowerBound, "REST request", "REST requests")
	if u.GraphQLPoints > 0 {
		sb.WriteString(", ")
		writeCount(&sb, u.GraphQLPoints, u.GraphQLLowerBound, "GraphQL point", "GraphQL points")
	}
	return sb.String()
}

// writeCount writes "<n> <unit>", prefixed with "at least " when n is a minimum.
func writeCount(sb *strings.Builder, n int, lowerBound bool, singular, plural string) {
	if lowerBound {
		sb.WriteString("at least ")
	}
	fmt.Fprintf(sb, "%d %s", n, pluralize(n, singular, plural))
}

// UsageMeter measures the GitHub API quota an operation consumes by comparing
// rate-limit snapshots taken before and after it.
//
// GET /rate_limit is not charged against any rate limit, so measuring costs no quota.
type UsageMeter struct {
	client   Client
	baseline *RateLimitResponse
}

// StartUsageMeter records the current rate-limit status as the baseline for
// measuring the API usage of the operation that follows.
//
// Parameters:
// - ctx: Context for the rate-limit request
// - client: GitHub client used for this and later rate-limit snapshots
//
// Returns:
// - A meter holding the baseline snapshot
// - An error wrapping ErrRateLimitUnavailable when the status cannot be read
//
// Side Effects:
// - Calls GET /rate_limit, which does not count against any rate limit
func StartUsageMeter(ctx context.Context, client Client) (*UsageMeter, error) {
	baseline, err := readRateLimit(ctx, client)
	if err != nil {
		return nil, err
	}
	return &UsageMeter{client: client, baseline: baseline}, nil
}

// Baseline returns the rate-limit status captured when the meter started.
func (m *UsageMeter) Baseline() *RateLimitResponse {
	return m.baseline
}

// Measure returns the API quota consumed since the meter started. It may be called
// more than once; each call measures from the original baseline.
//
// Parameters:
// - ctx: Context for the rate-limit request
//
// Returns:
// - The REST requests and GraphQL points consumed since the baseline
// - An error wrapping ErrRateLimitUnavailable when the status cannot be read
//
// Side Effects:
// - Calls GET /rate_limit, which does not count against any rate limit
func (m *UsageMeter) Measure(ctx context.Context) (APIUsage, error) {
	current, err := readRateLimit(ctx, m.client)
	if err != nil {
		return APIUsage{}, err
	}
	return usageBetween(m.baseline, current), nil
}

// MeasureDetached measures like Measure, but ignores the cancellation of ctx and
// is instead bounded by a short timeout, so the usage of an operation that was
// canceled or timed out can still be reported and recorded.
func (m *UsageMeter) MeasureDetached(ctx context.Context) (APIUsage, error) {
	measureCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), detachedMeasureTimeout)
	defer cancel()
	return m.Measure(measureCtx)
}

// readRateLimit fetches the rate-limit status, normalizing every failure to
// ErrRateLimitUnavailable.
func readRateLimit(ctx context.Context, client Client) (*RateLimitResponse, error) {
	resp, err := client.GetRateLimit(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRateLimitUnavailable, err)
	}
	if resp == nil {
		return nil, fmt.Errorf("%w: empty response", ErrRateLimitUnavailable)
	}
	return resp, nil
}

// usageBetween returns the quota consumed between the before and after snapshots.
func usageBetween(before, after *RateLimitResponse) APIUsage {
	core, coreLowerBound := consumedBetween(before.Resources.Core, after.Resources.Core)
	graphQL, graphQLLowerBound := consumedBetween(before.Resources.GraphQL, after.Resources.GraphQL)

	return APIUsage{
		CoreRequests:      core,
		GraphQLPoints:     graphQL,
		CoreLowerBound:    coreLowerBound,
		GraphQLLowerBound: graphQLLowerBound,
		CoreRemaining:     after.Resources.Core.Remaining,
		CoreLimit:         after.Resources.Core.Limit,
		CoreResetAt:       time.Unix(after.Resources.Core.Reset, 0),
	}
}

// consumedBetween returns how much of one rate-limit bucket was used between two
// snapshots, and whether that count is only a lower bound.
//
// GitHub reports a fixed reset time for the lifetime of a window, so an unchanged
// reset time means both snapshots belong to the same window and the difference in
// "used" is exact. A changed reset time means a new window began in between. If
// nothing had been used at the first snapshot, no window was open yet (GitHub
// then reports a reset time relative to "now"), so the new window's count is still
// exact. Otherwise requests made in the old window after the first snapshot are
// lost and the new window's count is a lower bound.
func consumedBetween(before, after RateLimitResource) (used int, lowerBound bool) {
	if after.Reset == before.Reset {
		return max(after.Used-before.Used, 0), false
	}
	return after.Used, before.Used > 0
}

// pluralize returns singular when n is 1 and plural otherwise.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
