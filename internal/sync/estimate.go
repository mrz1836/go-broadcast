package sync

import "github.com/mrz1836/go-broadcast/internal/config"

// Per-target request-cost constants for the preflight estimate.
//
// These are deliberately conservative (rounded up): the preflight gate prefers
// to over-estimate and halt safely rather than under-estimate and trip GitHub's
// rate limit mid-run. They are derived from the audited happy-path `gh` calls a
// single target sync performs in internal/sync/repository.go.
const (
	// readsPerTarget is a conservative count of GET (read) requests a single
	// target sync performs against GitHub. Audited read calls per target include
	// branch discovery (ListBranches), existing-PR discovery (ListPRs),
	// base-branch verification (GetBranch) and existing-file/current-user reads
	// (GetFile / GetCurrentUser). These are primary-bucket requests costing
	// PointCostRead (1 point) each; they do NOT count against the secondary
	// content-creation cap.
	readsPerTarget = 4

	// contentWritesPerTarget is a conservative count of content-generating
	// (mutating) gh API calls a single target sync performs. The dominant write
	// is the pull-request create/update (CreatePR / UpdatePR). PR labels,
	// assignees and reviewers are bundled into the single PRRequest, so they do
	// not add separate content-creation calls.
	//
	// File content is pushed via git (git.Push), NOT through the GitHub Contents
	// API, so synced files are intentionally EXCLUDED from this count — they do
	// not consume the secondary content-creation budget.
	contentWritesPerTarget = 1
)

// RunEstimate is the whole-run GitHub API request estimate produced by the
// preflight, summed across every resolved (filtered + enabled) target across all
// groups. Counts are conservative over-estimates.
type RunEstimate struct {
	// Targets is the total number of resolved targets across all groups.
	Targets int

	// PrimaryRequests is the conservative total of GitHub requests (reads from
	// state discovery plus mutating writes) counted against the primary
	// PrimaryHourlyLimit bucket.
	PrimaryRequests int

	// ContentWriteRequests is the count of mutating (content-generating) gh API
	// calls counted against the documented secondary content-creation caps
	// (SecondaryContentPerMinute / SecondaryContentPerHour). File content is
	// git-pushed, not Contents-API, and is excluded.
	ContentWriteRequests int

	// PRCreatePoints is ContentWriteRequests mapped to the secondary REST points
	// budget (PRCreatePointCost per write), for the SecondaryPointsPerMinute
	// dimension.
	PRCreatePoints int
}

// EstimateRun computes the whole-run request estimate for a sync invocation.
//
// It resolves the set of targets that would actually run by applying the same
// filtering the orchestrator applies — group filter / skip-groups (from options)
// followed by the enabled-group filter — then sums the conservative per-target
// cost across every resolved target across all groups (whole-run, up front).
//
// Conservative over-estimation is intentional and acceptable: the estimate may
// count targets that are already up-to-date (and would be skipped after state
// discovery), because the preflight runs before any write and avoids a pre-write
// state-discovery round.
func EstimateRun(cfg *config.Config, options *Options) RunEstimate {
	groups := resolveEstimateGroups(cfg, options)

	targets := 0
	for i := range groups {
		targets += len(groups[i].Targets)
	}

	contentWrites := targets * contentWritesPerTarget
	reads := targets * readsPerTarget

	return RunEstimate{
		Targets:              targets,
		PrimaryRequests:      reads + contentWrites,
		ContentWriteRequests: contentWrites,
		PRCreatePoints:       contentWrites * PRCreatePointCost,
	}
}

// resolveEstimateGroups returns the groups that would actually be synced. It
// derives the group set from ResolveScope (with no target filter) so the
// rate-limit estimate and the resolved sync scope can never drift apart. The
// underlying group/skip/enabled filtering still flows through the same pure,
// logger-free helpers, so the estimate has no I/O dependency.
func resolveEstimateGroups(cfg *config.Config, options *Options) []config.Group {
	scope, err := ResolveScope(cfg, options, nil)
	if err != nil || scope.Config == nil {
		return nil
	}
	return scope.Config.Groups
}

// filterEstimateGroupsByOptions mirrors GroupOrchestrator.filterGroupsByOptions:
// a group is skipped if it matches any SkipGroups pattern (by name or ID), and,
// when a GroupFilter is set, only groups matching the filter (by name or ID) are
// retained.
func filterEstimateGroupsByOptions(groups []config.Group, options *Options) []config.Group {
	if options == nil || (len(options.GroupFilter) == 0 && len(options.SkipGroups) == 0) {
		return groups
	}

	filtered := make([]config.Group, 0, len(groups))
	for _, group := range groups {
		if matchesGroupPattern(group, options.SkipGroups) {
			continue
		}

		if len(options.GroupFilter) > 0 && !matchesGroupPattern(group, options.GroupFilter) {
			continue
		}

		filtered = append(filtered, group)
	}

	return filtered
}

// filterEstimateEnabledGroups mirrors GroupOrchestrator.filterEnabledGroups: a
// group is included when Enabled is nil (default true) or explicitly true.
func filterEstimateEnabledGroups(groups []config.Group) []config.Group {
	enabled := make([]config.Group, 0, len(groups))
	for _, group := range groups {
		if group.Enabled == nil || *group.Enabled {
			enabled = append(enabled, group)
		}
	}
	return enabled
}

// matchesGroupPattern reports whether the group's name or ID matches any of the
// given patterns (exact match), matching the orchestrator's filter comparison.
func matchesGroupPattern(group config.Group, patterns []string) bool {
	for _, pattern := range patterns {
		if group.Name == pattern || group.ID == pattern {
			return true
		}
	}
	return false
}
