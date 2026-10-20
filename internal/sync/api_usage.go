package sync

import (
	"context"
	"fmt"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// startUsageMeter snapshots the GitHub rate limit so the API usage of the group
// sync that follows can be measured. Measurement is best effort: when the rate
// limit cannot be read it returns nil, and the sync proceeds unmeasured.
func (e *Engine) startUsageMeter(ctx context.Context) *gh.UsageMeter {
	if e.gh == nil {
		return nil
	}

	meter, err := gh.StartUsageMeter(ctx, e.gh)
	if err != nil {
		e.logger.WithError(err).Debug("GitHub API usage will not be measured for this group")
		return nil
	}
	return meter
}

// measureAPIUsage returns the API usage since meter started, or nil when the
// usage cannot be measured. It measures even after ctx is canceled, so an
// interrupted group sync still reports what it used.
func (e *Engine) measureAPIUsage(ctx context.Context, meter *gh.UsageMeter) *gh.APIUsage {
	if meter == nil {
		return nil
	}

	usage, err := meter.MeasureDetached(ctx)
	if err != nil {
		e.logger.WithError(err).Debug("GitHub API usage could not be measured for this group")
		return nil
	}
	return &usage
}

// recordRunAPIUsage stores measured API usage on the current sync run, to be
// persisted by finalizeSyncRun. It is a no-op without a run or a measurement.
func (e *Engine) recordRunAPIUsage(usage *gh.APIUsage) {
	if usage == nil {
		return
	}

	e.currentRunMu.Lock()
	defer e.currentRunMu.Unlock()
	if e.currentRun != nil {
		e.currentRun.APICallsMade = usage.CoreRequests
		e.currentRun.APIUsageLowerBound = usage.CoreLowerBound
	}
}

// printAPIUsage reports a group sync's GitHub API usage and the REST budget left.
func printAPIUsage(groupName string, usage *gh.APIUsage) {
	if usage == nil {
		return
	}

	output.Info(fmt.Sprintf("GitHub API usage for group %q: %s (%d/%d REST requests remaining, resets at %s)",
		groupName, usage, usage.CoreRemaining, usage.CoreLimit, usage.CoreResetAt.Format("15:04:05")))
}
