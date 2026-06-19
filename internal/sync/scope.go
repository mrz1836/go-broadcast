package sync

import (
	"fmt"
	"strings"

	"github.com/mrz1836/go-broadcast/internal/config"
	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
)

// ResolvedScope is the single, up-front resolution of what a sync invocation
// will actually touch. It is produced by ResolveScope before the single- vs
// multi-group branch in Engine.Sync, so the target/group filters are honored in
// every config shape (previously the multi-group path silently ignored them).
//
// Config is an executable, scoped config: its Groups have been filtered by the
// group/skip/enabled rules and, when a target filter is supplied, each group's
// Targets are narrowed to the matching repositories. The counts and the flat
// Groups / Repos lists describe the resolved blast radius and feed both the
// pre-write summary (SC-5) and the blast-radius guard (added in a later phase).
type ResolvedScope struct {
	// Config is the scoped configuration to execute. When no filtering applies
	// it is the original config pointer; otherwise it is a narrowed copy.
	Config *config.Config

	// Groups is the list of in-scope group labels (name, falling back to ID).
	Groups []string

	// Repos is the flat list of in-scope target repositories across all groups.
	Repos []string

	// GroupCount is the number of in-scope groups.
	GroupCount int

	// RepoCount is the total number of in-scope target repositories (the
	// resolved blast radius / total repos that may be written).
	RepoCount int
}

// ResolveScope resolves the set of groups and target repositories a sync
// invocation will actually run, mirroring cli.FilterConfigByGroups (group /
// skip / enabled filtering) and additionally narrowing each group's targets to
// targetFilter when one is provided.
//
// Resolution order, matching the orchestrator and the rate-limit estimate:
//  1. options.SkipGroups removes matching groups (by name or ID).
//  2. options.GroupFilter, when set, retains only matching groups.
//  3. Disabled groups (Enabled explicitly false) are removed.
//  4. targetFilter, when set, keeps only targets whose Repo is listed; groups
//     left with zero matching targets are dropped.
//
// When targetFilter is non-empty and no target repository matches it across the
// surviving groups, it returns appErrors.ErrNoMatchingTargets, preserving the
// engine's prior behavior. When nothing is filtered away (no explicit group /
// skip / target filters and every group is enabled) the original config pointer
// is preserved so callers relying on pointer identity (e.g. state discovery)
// see the unchanged config.
func ResolveScope(cfg *config.Config, options *Options, targetFilter []string) (ResolvedScope, error) {
	if cfg == nil {
		return ResolvedScope{}, nil
	}

	// Steps 1-3: group/skip/enabled filtering (reuses the audited estimate
	// helpers so the resolved scope and the rate-limit estimate cannot drift).
	// References options.GroupFilter / options.SkipGroups for the change check.
	filtered := filterEstimateGroupsByOptions(cfg.Groups, options)
	filtered = filterEstimateEnabledGroups(filtered)

	// Step 4: narrow each surviving group's targets to the target filter.
	narrowed := make([]config.Group, 0, len(filtered))
	matchedAnyTarget := false
	for _, group := range filtered {
		if len(targetFilter) == 0 {
			narrowed = append(narrowed, group)
			continue
		}

		keptTargets := make([]config.TargetConfig, 0, len(group.Targets))
		for _, target := range group.Targets {
			if containsRepo(targetFilter, target.Repo) {
				keptTargets = append(keptTargets, target)
			}
		}
		if len(keptTargets) == 0 {
			continue // group has no targets matching the filter; drop it
		}

		matchedAnyTarget = true
		scopedGroup := group
		scopedGroup.Targets = keptTargets
		narrowed = append(narrowed, scopedGroup)
	}

	if len(targetFilter) > 0 && !matchedAnyTarget {
		return ResolvedScope{}, fmt.Errorf("%w: %v", appErrors.ErrNoMatchingTargets, targetFilter)
	}

	// Preserve the original config pointer when nothing was filtered away, so
	// state discovery and other consumers keying off the config pointer are
	// unaffected by the resolution step.
	filtersProvided := options != nil && (len(options.GroupFilter) > 0 || len(options.SkipGroups) > 0)
	var scopeConfig *config.Config
	if !filtersProvided && len(targetFilter) == 0 && len(narrowed) == len(cfg.Groups) {
		scopeConfig = cfg
	} else {
		scopeConfig = cloneConfigWithGroups(cfg, narrowed)
	}

	groups := make([]string, 0, len(narrowed))
	repos := make([]string, 0)
	for _, group := range narrowed {
		groups = append(groups, groupLabel(group))
		for _, target := range group.Targets {
			repos = append(repos, target.Repo)
		}
	}

	return ResolvedScope{
		Config:     scopeConfig,
		Groups:     groups,
		Repos:      repos,
		GroupCount: len(narrowed),
		RepoCount:  len(repos),
	}, nil
}

// Summary returns a human-readable description of the resolved scope. It is
// printed before any write on every invocation (SC-5) so an operator always
// sees exactly which groups and repositories a sync will touch.
func (s ResolvedScope) Summary() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Resolved sync scope: %d group(s), %d repo(s)\n", s.GroupCount, s.RepoCount)

	if len(s.Groups) > 0 {
		fmt.Fprintf(&b, "  Groups: %s\n", strings.Join(s.Groups, ", "))
	}
	if len(s.Repos) > 0 {
		fmt.Fprintf(&b, "  Repos: %s\n", strings.Join(s.Repos, ", "))
	}

	// One pull-request/content write per target (mirrors the preflight's
	// contentWritesPerTarget), so the estimated writes equal the repo count.
	fmt.Fprintf(&b, "  Estimated PRs/writes: %d", s.RepoCount)

	return b.String()
}

// cloneConfigWithGroups returns a shallow copy of cfg with its Groups replaced
// by the provided (filtered/narrowed) groups, mirroring the field set copied by
// cli.FilterConfigByGroups so the scoped config remains fully usable.
func cloneConfigWithGroups(cfg *config.Config, groups []config.Group) *config.Config {
	scoped := *cfg
	scoped.Groups = groups
	return &scoped
}

// groupLabel returns a display label for a group, preferring its friendly name
// and falling back to its ID.
func groupLabel(group config.Group) string {
	if group.Name != "" {
		return group.Name
	}
	return group.ID
}

// containsRepo reports whether repo is present in the filter list (exact match).
func containsRepo(filter []string, repo string) bool {
	for _, f := range filter {
		if f == repo {
			return true
		}
	}
	return false
}
