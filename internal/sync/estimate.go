package sync

import "github.com/mrz1836/go-broadcast/internal/config"

// Per-target request-cost constants for the preflight estimate.
//
// These are deliberately conservative (rounded up): the preflight gate prefers
// to over-estimate and halt safely rather than under-estimate and trip GitHub's
// rate limit mid-run. They are derived from the audited happy-path `gh` calls a
// single target sync performs in internal/sync/repository.go.
const (
	// baseReadsPerTarget is a conservative count of the fixed GET (read) requests
	// a single target sync performs regardless of how many files or directories
	// it syncs. Audited fixed read calls per target include branch discovery
	// (ListBranches), existing-PR discovery (ListPRs), base-branch verification
	// (GetBranch) and current-user resolution (GetCurrentUser). These are
	// primary-bucket requests costing PointCostRead (1 point) each; they do NOT
	// count against the secondary content-creation cap.
	//
	// Per-file and per-directory content reads are NOT included here — they scale
	// with the target's mapping and are added separately by EstimateRun. See the
	// estimateTargetReads helper.
	baseReadsPerTarget = 4

	// treeReadsPerTarget is the recursive git-tree fetch a target performs before
	// comparing any mapped file. One tree is shared across all of a target's file
	// and directory mappings, so it is counted once per target that has any
	// mappings (not once per mapping).
	treeReadsPerTarget = 1

	// estimatedFilesPerDirectory is a conservative over-estimate of how many
	// files a single directory mapping expands to. The real count is only known
	// after the repo tree is fetched at runtime, so the preflight assumes this
	// many Contents-API reads per directory mapping. Deliberately high so the gate
	// over-estimates and halts safely rather than under-counting and tripping the
	// live rate limit mid-run.
	estimatedFilesPerDirectory = 20

	// contentWritesPerTarget is a conservative count of content-generating
	// (mutating) gh API calls a single target sync performs. The dominant write
	// is the pull-request create/update (CreatePR / UpdatePR). PR labels,
	// assignees and reviewers are bundled into the single PRRequest, so they do
	// not add separate content-creation calls.
	//
	// File content is pushed via git (git.Push), NOT through the GitHub Contents
	// API, so synced files do not consume the secondary content-creation (write)
	// budget and are intentionally EXCLUDED from this count. Note they DO still
	// consume the primary read budget when their current content is fetched for
	// diffing — that read cost is accounted for separately in EstimateRun.
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
	reads := 0
	for i := range groups {
		for j := range groups[i].Targets {
			targets++
			reads += estimateTargetReads(&groups[i].Targets[j])
		}
	}

	contentWrites := targets * contentWritesPerTarget

	return RunEstimate{
		Targets:              targets,
		PrimaryRequests:      reads + contentWrites,
		ContentWriteRequests: contentWrites,
		PRCreatePoints:       contentWrites * PRCreatePointCost,
	}
}

// estimateTargetReads returns the conservative primary-bucket read count for a
// single target sync. It combines the fixed discovery reads with the git-tree
// fetch and the per-file Contents-API reads the sync may perform to diff mapped
// files against the target.
//
// The target tree settles unchanged and missing files without a read, so only
// files whose content differs are fetched. Which files differ is unknown before
// the run, so the estimate charges the worst case: every mapped file differs and
// costs one GetFile (Contents API GET). Since the real directory file count is
// also unknown until runtime, each directory mapping is charged a conservative
// estimatedFilesPerDirectory reads.
func estimateTargetReads(t *config.TargetConfig) int {
	reads := baseReadsPerTarget

	if len(t.Files) > 0 || len(t.Directories) > 0 {
		reads += treeReadsPerTarget
	}
	reads += len(t.Files)
	reads += len(t.Directories) * estimatedFilesPerDirectory

	return reads
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
