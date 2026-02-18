package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// RateLimitInfo holds the current GitHub API rate limit status
type RateLimitInfo struct {
	Limit     int       // Maximum requests per hour
	Remaining int       // Requests remaining in current window
	ResetAt   time.Time // When the rate limit window resets
	Used      int       // Requests already used in current window
}

// SyncEstimate holds estimated API call counts for a sync operation
type SyncEstimate struct {
	MinCalls       int // Minimum expected API calls (repos * 4: metadata + 3 security)
	MaxCalls       int // Maximum expected API calls (repos * 10: security + CI + artifacts)
	GraphQLBatches int // Number of GraphQL batches for metadata
}

// CheckRateLimit queries the GitHub API rate limit and returns structured info
func CheckRateLimit(ctx context.Context, ghClient gh.Client) (*RateLimitInfo, error) {
	resp, err := ghClient.GetRateLimit(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	return &RateLimitInfo{
		Limit:     resp.Resources.Core.Limit,
		Remaining: resp.Resources.Core.Remaining,
		ResetAt:   time.Unix(resp.Resources.Core.Reset, 0),
		Used:      resp.Resources.Core.Used,
	}, nil
}

// EstimateSyncCost estimates the number of API calls needed for a sync operation
func EstimateSyncCost(repoCount int) SyncEstimate {
	return SyncEstimate{
		MinCalls:       repoCount * 4,         // 1 GraphQL batch share + 3 security REST calls per repo
		MaxCalls:       repoCount * 10,        // security + CI workflows + runs + artifacts + downloads
		GraphQLBatches: (repoCount + 24) / 25, // ceil(repoCount / 25)
	}
}

// WarnIfBudgetLow prints a warning if the remaining rate limit budget is lower than
// the estimated sync cost
func WarnIfBudgetLow(info *RateLimitInfo, estimate SyncEstimate) {
	if info == nil {
		return
	}

	if info.Remaining < estimate.MinCalls {
		output.Warn(fmt.Sprintf(
			"Rate limit budget low: %d remaining, need at least %d-%d calls. "+
				"Resets at %s. Sync may fail or be rate-limited.",
			info.Remaining, estimate.MinCalls, estimate.MaxCalls,
			info.ResetAt.Format("15:04:05"),
		))
	} else if info.Remaining < estimate.MaxCalls {
		output.Warn(fmt.Sprintf(
			"Rate limit budget tight: %d remaining, may need up to %d calls. "+
				"Resets at %s.",
			info.Remaining, estimate.MaxCalls,
			info.ResetAt.Format("15:04:05"),
		))
	}
}

// DisplayRateLimitInfo prints the current rate limit status
func DisplayRateLimitInfo(info *RateLimitInfo) {
	if info == nil {
		return
	}

	output.Info(fmt.Sprintf(
		"GitHub API Rate Limit: %d/%d remaining (resets at %s)",
		info.Remaining, info.Limit,
		info.ResetAt.Format("15:04:05"),
	))
}
