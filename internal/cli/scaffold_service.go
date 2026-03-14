package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// ScaffoldOptions holds all parameters for the scaffold flow
type ScaffoldOptions struct {
	Name        string
	Description string
	Owner       string
	Preset      *config.SettingsPreset
	Topics      []string
	NoClone     bool
	NoFiles     bool
	DryRun      bool
}

// ScaffoldResult holds the outcome of a scaffold operation
type ScaffoldResult struct {
	RepoFullName string
	ClonePath    string
	Created      bool
}

// RunScaffold executes the shared scaffold flow:
// validate -> create repo -> apply settings -> set topics -> sync labels -> create rulesets -> clone
func RunScaffold(ctx context.Context, ghClient gh.Client, opts ScaffoldOptions) (*ScaffoldResult, error) {
	repoFullName := fmt.Sprintf("%s/%s", opts.Owner, opts.Name)

	if opts.DryRun {
		output.Info(fmt.Sprintf("[DRY RUN] Scaffold plan for %s:", repoFullName))
		output.Info(fmt.Sprintf("  Preset: %s", opts.Preset.ID))
		output.Info("  Visibility: private")
		output.Info("  Steps:")
		output.Info("    1. Create repository")
		output.Info("    2. Apply 12 settings from preset")
		if len(opts.Topics) > 0 {
			output.Info(fmt.Sprintf("    3. Set topics: %s", strings.Join(opts.Topics, ", ")))
		}
		output.Info(fmt.Sprintf("    4. Sync %d labels", len(opts.Preset.Labels)))
		output.Info(fmt.Sprintf("    5. Create %d rulesets", len(opts.Preset.Rulesets)))
		if !opts.NoClone {
			output.Info("    6. Clone repository locally")
		}
		return &ScaffoldResult{RepoFullName: repoFullName}, nil
	}

	// Step 1: Create repository (always private)
	output.Info(fmt.Sprintf("Creating repository %s...", repoFullName))
	_, err := ghClient.CreateRepository(ctx, gh.CreateRepoOptions{
		Name:        repoFullName,
		Description: opts.Description,
		Private:     true,
	})
	if err != nil {
		output.Warn(fmt.Sprintf("Partial failure cleanup: gh repo delete %s --yes", repoFullName))
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}
	output.Success(fmt.Sprintf("Repository created: %s", repoFullName))

	// Step 2: Apply settings
	output.Info("Applying settings from preset...")
	settings := presetToRepoSettings(opts.Preset)
	if err = ghClient.UpdateRepoSettings(ctx, repoFullName, settings); err != nil {
		output.Warn(fmt.Sprintf("Failed to apply settings: %v (continuing...)", err))
	} else {
		output.Success("Settings applied")
	}

	// Step 3: Set topics
	if len(opts.Topics) > 0 {
		output.Info("Setting topics...")
		if err = ghClient.SetTopics(ctx, repoFullName, opts.Topics); err != nil {
			output.Warn(fmt.Sprintf("Failed to set topics: %v (continuing...)", err))
		} else {
			output.Success(fmt.Sprintf("Topics set: %s", strings.Join(opts.Topics, ", ")))
		}
	}

	// Step 4: Sync labels
	if len(opts.Preset.Labels) > 0 {
		output.Info("Syncing labels...")
		labels := presetLabelsToGH(opts.Preset.Labels)
		if err = ghClient.SyncLabels(ctx, repoFullName, labels); err != nil {
			output.Warn(fmt.Sprintf("Failed to sync labels: %v (continuing...)", err))
		} else {
			output.Success(fmt.Sprintf("%d labels synced", len(labels)))
		}
	}

	// Step 5: Create rulesets
	for _, rc := range opts.Preset.Rulesets {
		output.Info(fmt.Sprintf("Creating ruleset: %s...", rc.Name))
		ruleset := configRulesetToGH(&rc)
		if err = ghClient.CreateOrUpdateRuleset(ctx, repoFullName, ruleset); err != nil {
			output.Warn(fmt.Sprintf("Ruleset %q skipped: %v", rc.Name, err))
		} else {
			output.Success(fmt.Sprintf("Ruleset %q created", rc.Name))
		}
	}

	return &ScaffoldResult{
		RepoFullName: repoFullName,
		Created:      true,
	}, nil
}

// presetToRepoSettings converts a config preset to gh.RepoSettings
func presetToRepoSettings(p *config.SettingsPreset) gh.RepoSettings {
	return gh.RepoSettings{
		HasIssues:                p.HasIssues,
		HasWiki:                  p.HasWiki,
		HasProjects:              p.HasProjects,
		HasDiscussions:           p.HasDiscussions,
		AllowSquashMerge:         p.AllowSquashMerge,
		AllowMergeCommit:         p.AllowMergeCommit,
		AllowRebaseMerge:         p.AllowRebaseMerge,
		DeleteBranchOnMerge:      p.DeleteBranchOnMerge,
		AllowAutoMerge:           p.AllowAutoMerge,
		AllowUpdateBranch:        p.AllowUpdateBranch,
		SquashMergeCommitTitle:   p.SquashMergeCommitTitle,
		SquashMergeCommitMessage: p.SquashMergeCommitMessage,
	}
}

// presetLabelsToGH converts config label specs to gh.Label slice
func presetLabelsToGH(labels []config.LabelSpec) []gh.Label {
	result := make([]gh.Label, 0, len(labels))
	for _, l := range labels {
		result = append(result, gh.Label{
			Name:        l.Name,
			Color:       l.Color,
			Description: l.Description,
		})
	}
	return result
}

// configRulesetToGH converts a config ruleset to gh.Ruleset
func configRulesetToGH(rc *config.RulesetConfig) gh.Ruleset {
	if rc.Target == "tag" {
		return gh.BuildTagRuleset(rc.Name, rc.Include, rc.Exclude, rc.Rules)
	}
	return gh.BuildBranchRuleset(rc.Name, rc.Include, rc.Exclude, rc.Rules)
}

// dbPresetToConfigPreset converts a db.SettingsPreset to config.SettingsPreset for shared use
func dbPresetToConfigPreset(p *dbSettingsPresetCompat) *config.SettingsPreset {
	preset := &config.SettingsPreset{
		ID:                       p.ExternalID,
		Name:                     p.Name,
		Description:              p.Description,
		HasIssues:                p.HasIssues,
		HasWiki:                  p.HasWiki,
		HasProjects:              p.HasProjects,
		HasDiscussions:           p.HasDiscussions,
		AllowSquashMerge:         p.AllowSquashMerge,
		AllowMergeCommit:         p.AllowMergeCommit,
		AllowRebaseMerge:         p.AllowRebaseMerge,
		DeleteBranchOnMerge:      p.DeleteBranchOnMerge,
		AllowAutoMerge:           p.AllowAutoMerge,
		AllowUpdateBranch:        p.AllowUpdateBranch,
		SquashMergeCommitTitle:   p.SquashMergeCommitTitle,
		SquashMergeCommitMessage: p.SquashMergeCommitMessage,
	}

	for _, l := range p.Labels {
		preset.Labels = append(preset.Labels, config.LabelSpec{
			Name:        l.Name,
			Color:       l.Color,
			Description: l.Description,
		})
	}

	for _, r := range p.Rulesets {
		preset.Rulesets = append(preset.Rulesets, config.RulesetConfig{
			Name:        r.Name,
			Target:      r.Target,
			Enforcement: r.Enforcement,
			Include:     r.Include,
			Exclude:     r.Exclude,
			Rules:       r.Rules,
		})
	}

	return preset
}

// dbSettingsPresetCompat is a bridge type to avoid importing db in scaffold service signatures
type dbSettingsPresetCompat struct {
	ExternalID               string
	Name                     string
	Description              string
	HasIssues                bool
	HasWiki                  bool
	HasProjects              bool
	HasDiscussions           bool
	AllowSquashMerge         bool
	AllowMergeCommit         bool
	AllowRebaseMerge         bool
	DeleteBranchOnMerge      bool
	AllowAutoMerge           bool
	AllowUpdateBranch        bool
	SquashMergeCommitTitle   string
	SquashMergeCommitMessage string
	Labels                   []dbLabelCompat
	Rulesets                 []dbRulesetCompat
}

type dbLabelCompat struct {
	Name        string
	Color       string
	Description string
}

type dbRulesetCompat struct {
	Name        string
	Target      string
	Enforcement string
	Include     []string
	Exclude     []string
	Rules       []string
}
