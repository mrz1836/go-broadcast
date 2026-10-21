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

// rateLimitWindow is the length of a GitHub primary rate-limit window.
const rateLimitWindow = time.Hour

// resetTolerance absorbs disagreement about when a rate-limit window resets, both
// between this machine's clock and GitHub's and between GitHub regions.
const resetTolerance = time.Minute

// APIUsage is the GitHub API quota consumed between two rate-limit snapshots.
//
// The counts are GitHub's own accounting for the authenticated identity: they
// include every page of a paginated request and every request a gh command makes,
// but also requests made at the same time by other tools sharing the token. GitHub
// serves requests from several regions, so its counts can vary slightly from one
// response to the next.
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
	clock    func() time.Time
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
	return &UsageMeter{client: client, baseline: baseline, clock: time.Now}, nil
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
	return usageBetween(m.baseline, current, m.clock()), nil
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

// usageBetween returns the quota consumed between the before and after snapshots,
// where measuredAt is when the after snapshot was taken.
func usageBetween(before, after *RateLimitResponse, measuredAt time.Time) APIUsage {
	core, coreLowerBound := consumedBetween(before.Resources.Core, after.Resources.Core, measuredAt)
	graphQL, graphQLLowerBound := consumedBetween(before.Resources.GraphQL, after.Resources.GraphQL, measuredAt)

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
// snapshots, and whether that count is only a lower bound. measuredAt is when the
// after snapshot was taken.
//
// Within one window, the difference in "used" is the usage between the snapshots.
// When a new window began in between, requests made in the old window after the
// first snapshot are lost, so the new window's count is a lower bound. It is still
// exact if nothing had been used at the first snapshot.
func consumedBetween(before, after RateLimitResource, measuredAt time.Time) (used int, lowerBound bool) {
	if windowRolledOver(before, after, measuredAt) {
		return after.Used, before.Used > 0
	}
	return max(after.Used-before.Used, 0), false
}

// windowRolledOver reports whether a new rate-limit window began between two
// snapshots of a bucket, where measuredAt is when the second one was taken.
//
// A different reset time alone is not proof: GitHub serves requests from several
// regions, and the rate-limit values they report can differ from one response to
// the next. A new window can only open after the old one ends, so the old reset
// time must have passed by the second snapshot, and the new window then resets at
// least a full window after the old one.
//
// Before any request in a window, GitHub reports a reset time a full window from
// "now". A window that opens during the measurement then resets shortly after that
// time, not a full window later, so it is not counted as a rollover.
func windowRolledOver(before, after RateLimitResource, measuredAt time.Time) bool {
	oldReset := time.Unix(before.Reset, 0)
	newReset := time.Unix(after.Reset, 0)
	oldWindowEnded := !measuredAt.Before(oldReset.Add(-resetTolerance))
	newWindowFollows := !newReset.Before(oldReset.Add(rateLimitWindow - resetTolerance))
	return oldWindowEnded && newWindowFollows
}

// pluralize returns singular when n is 1 and plural otherwise.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
