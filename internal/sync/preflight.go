package sync

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// ErrRateLimitProbeUnavailable indicates the live GET /rate_limit probe could
// not be completed (network error, missing/invalid token, or unparseable
// response). The integration layer decides whether this fails open or closed.
var ErrRateLimitProbeUnavailable = errors.New("rate-limit probe unavailable")

// ErrRateLimitPreflight is the sentinel returned when the preflight gate halts a
// run because the estimated request budget exceeds what is available (after the
// safety margin). It is a package-level error so the CLI can present a clean,
// actionable message instead of a raw gh 403 dump, and callers can match it with
// errors.Is.
var ErrRateLimitPreflight = errors.New("sync halted by rate-limit preflight")

// PreflightConfig carries the resolved knobs the preflight decision needs. It is
// derived from the sync Options (which in turn merge config + CLI flags) so the
// decision logic stays free of config/flag plumbing.
type PreflightConfig struct {
	// Enabled toggles the gate. When false the engine skips the preflight.
	Enabled bool

	// PrimaryMarginPercent is the percentage of the live primary "remaining"
	// budget to keep as headroom (0-100).
	PrimaryMarginPercent int

	// SecondaryReserve is how many of the documented per-minute secondary
	// content-write slots to keep in reserve.
	SecondaryReserve int

	// FailClosed halts when the probe is unavailable (default fails open).
	FailClosed bool
}

// Decision is the pure outcome of the preflight gate: whether the run may
// proceed, why it would halt, and the summary fields needed to render a
// human-readable preflight message.
type Decision struct {
	// Proceed reports whether the run is within budget (after the safety margin).
	Proceed bool

	// Reason is a human-readable explanation of why the run would halt. Empty
	// when Proceed is true.
	Reason string

	// ResetAt is the time the primary budget resets (for the halt hint). Zero
	// when unknown.
	ResetAt time.Time

	// Summary fields (always populated, for the preflight message).
	EstimatedPrimaryRequests int
	PrimaryRemaining         int
	PrimaryMargin            int
	EstimatedContentWrites   int
	SecondaryPerMinuteCap    int
	SecondaryReserve         int
	SecondaryPerHourCap      int
}

// Preflight probes the live primary rate-limit budget and decides whether a run
// may proceed. The decision logic (decide) is a pure function separated from the
// I/O (probe) so it can be unit-tested without a network or gh client.
type Preflight struct {
	client gh.Client
	cfg    PreflightConfig
}

// NewPreflight constructs a Preflight gate.
func NewPreflight(client gh.Client, cfg PreflightConfig) *Preflight {
	return &Preflight{client: client, cfg: cfg}
}

// Config returns the resolved preflight configuration.
func (p *Preflight) Config() PreflightConfig {
	return p.cfg
}

// probe queries the live primary budget via the free GET /rate_limit endpoint.
// It returns the primary core remaining count and reset time. Any failure is
// wrapped in ErrRateLimitProbeUnavailable so the caller can apply its
// fail-open/closed policy.
func (p *Preflight) probe(ctx context.Context) (remaining int, reset time.Time, err error) {
	resp, err := p.client.GetRateLimit(ctx)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("%w: %w", ErrRateLimitProbeUnavailable, err)
	}
	if resp == nil {
		return 0, time.Time{}, fmt.Errorf("%w: empty rate-limit response", ErrRateLimitProbeUnavailable)
	}
	return resp.Resources.Core.Remaining, time.Unix(resp.Resources.Core.Reset, 0), nil
}

// Evaluate probes the live primary budget via GET /rate_limit and returns the
// preflight decision for the given whole-run estimate. On probe failure it
// returns an error wrapping ErrRateLimitProbeUnavailable; the integration layer
// applies its fail-open (default) / fail-closed policy. The pure decision logic
// (decide) is exercised only on a successful probe.
func (p *Preflight) Evaluate(ctx context.Context, estimate RunEstimate) (Decision, error) {
	remaining, reset, err := p.probe(ctx)
	if err != nil {
		return Decision{}, err
	}
	return decide(estimate, remaining, reset, p.cfg), nil
}

// decide is the pure preflight decision. It has no gh/network dependency.
//
// Two independent halt conditions are evaluated; either trips the gate:
//   - Primary (live):   estimate.PrimaryRequests > (primaryRemaining - primaryMargin)
//     where primaryMargin = ceil(primaryRemaining * marginPct / 100).
//   - Secondary (static cap, not live): estimate.ContentWriteRequests >
//     (SecondaryContentPerMinute - secondaryReserve) OR > SecondaryContentPerHour.
func decide(estimate RunEstimate, primaryRemaining int, resetAt time.Time, cfg PreflightConfig) Decision {
	primaryMargin := int(math.Ceil(float64(primaryRemaining) * float64(cfg.PrimaryMarginPercent) / 100.0))

	d := Decision{
		ResetAt:                  resetAt,
		EstimatedPrimaryRequests: estimate.PrimaryRequests,
		PrimaryRemaining:         primaryRemaining,
		PrimaryMargin:            primaryMargin,
		EstimatedContentWrites:   estimate.ContentWriteRequests,
		SecondaryPerMinuteCap:    SecondaryContentPerMinute,
		SecondaryReserve:         cfg.SecondaryReserve,
		SecondaryPerHourCap:      SecondaryContentPerHour,
	}

	var reasons []string

	// Primary budget (live).
	if estimate.PrimaryRequests > primaryRemaining-primaryMargin {
		reasons = append(reasons, fmt.Sprintf(
			"primary budget: estimated %d requests exceed %d available (remaining %d minus %d%% margin = %d)",
			estimate.PrimaryRequests, primaryRemaining-primaryMargin, primaryRemaining, cfg.PrimaryMarginPercent, primaryMargin,
		))
	}

	// Secondary content-creation per-minute cap (static).
	perMinuteAvailable := SecondaryContentPerMinute - cfg.SecondaryReserve
	if estimate.ContentWriteRequests > perMinuteAvailable {
		reasons = append(reasons, fmt.Sprintf(
			"secondary per-minute content-write cap: estimated %d writes exceed %d available (cap %d minus reserve %d)",
			estimate.ContentWriteRequests, perMinuteAvailable, SecondaryContentPerMinute, cfg.SecondaryReserve,
		))
	}

	// Secondary content-creation per-hour cap (static).
	if estimate.ContentWriteRequests > SecondaryContentPerHour {
		reasons = append(reasons, fmt.Sprintf(
			"secondary per-hour content-write cap: estimated %d writes exceed %d",
			estimate.ContentWriteRequests, SecondaryContentPerHour,
		))
	}

	if len(reasons) == 0 {
		d.Proceed = true
		return d
	}

	d.Proceed = false
	d.Reason = strings.Join(reasons, "; ")
	return d
}

// Summary renders the human-readable preflight summary fields. It is pure and
// does not decide proceed/halt; the integration layer prints it before any write.
func (d Decision) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Rate-limit preflight: estimated %d primary requests vs %d remaining (reserving %d for safety margin)",
		d.EstimatedPrimaryRequests, d.PrimaryRemaining, d.PrimaryMargin)
	fmt.Fprintf(&b, "; estimated %d content writes vs %d/min cap (reserve %d) and %d/hr cap",
		d.EstimatedContentWrites, d.SecondaryPerMinuteCap, d.SecondaryReserve, d.SecondaryPerHourCap)
	if !d.ResetAt.IsZero() {
		fmt.Fprintf(&b, "; primary budget resets at %s", d.ResetAt.Format(time.RFC3339))
	}
	return b.String()
}

// HaltMessage renders the actionable over-budget message shown to the user when
// the gate hard-halts a run. It names why the run was halted, the primary reset
// time (when known), and how to proceed — wait for the reset or force through
// with --ignore-rate-limit-preflight. It is pure (no wall-clock dependency) so
// the rendered text is deterministic for tests.
func (d Decision) HaltMessage() string {
	var b strings.Builder
	b.WriteString("rate-limit preflight halted the sync before any write: ")
	b.WriteString(d.Reason)
	if !d.ResetAt.IsZero() {
		fmt.Fprintf(&b, "; primary budget resets at %s", d.ResetAt.Format(time.RFC3339))
	}
	b.WriteString("; wait for the budget to reset or re-run with --ignore-rate-limit-preflight to force the sync through")
	return b.String()
}
