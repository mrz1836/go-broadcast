package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/output"
)

func newSettingsApplyCmd() *cobra.Command {
	var (
		presetID    string
		topics      string
		description string
		force       bool
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "apply <owner/repo>",
		Short: "Apply preset settings to a repository",
		Long: `Apply repository settings, labels, and rulesets from a preset.

Compares current settings against the preset and shows a diff before applying.
Idempotent: re-running with no changes produces no API calls.`,
		Example: `  # Apply with default preset
  go-broadcast settings apply mrz1836/go-broadcast

  # Apply specific preset
  go-broadcast settings apply mrz1836/my-repo --preset go-lib

  # Preview changes
  go-broadcast settings apply mrz1836/my-repo --dry-run

  # Apply with topics and description
  go-broadcast settings apply mrz1836/my-repo --topics "go,library" --description "A Go library"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSettingsApply(cmd.Context(), args[0], presetID, topics, description, force, dryRun)
		},
	}

	cmd.Flags().StringVar(&presetID, "preset", "mvp", "Settings preset ID")
	cmd.Flags().StringVar(&topics, "topics", "", "Comma-separated topics to set")
	cmd.Flags().StringVar(&description, "description", "", "Repository description to set")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")

	return cmd
}

// settingsDiff represents a single setting change
type settingsDiff struct {
	Name     string
	Current  string
	Expected string
}

func runSettingsApply(ctx context.Context, repo, presetID, topics, description string, force, dryRun bool) error {
	// Validate repo format
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid repo format %q, expected owner/repo", repo) //nolint:err113 // user-facing
	}

	// Resolve preset
	preset := resolvePreset(ctx, presetID)

	output.Info(fmt.Sprintf("Settings apply: %s (preset: %s)", repo, preset.ID))

	if dryRun {
		output.Info("[DRY RUN] Would apply the following settings:")
		printPresetSettings(preset)
		if topics != "" {
			output.Info(fmt.Sprintf("  Topics: %s", topics))
		}
		if description != "" {
			output.Info(fmt.Sprintf("  Description: %s", description))
		}
		output.Info(fmt.Sprintf("  Labels: %d", len(preset.Labels)))
		output.Info(fmt.Sprintf("  Rulesets: %d", len(preset.Rulesets)))
		return nil
	}

	// Initialize GitHub client
	logger := logrus.StandardLogger()
	ghClient, err := gh.NewClient(ctx, logger, &logging.LogConfig{})
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Get current settings
	output.Info("Fetching current settings...")
	current, err := ghClient.GetRepoSettings(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to get current settings: %w", err)
	}

	// Compute diff
	diffs := computeSettingsDiffs(current, preset)

	if len(diffs) == 0 {
		output.Success("Repository already matches preset — no changes needed")
	} else {
		output.Info(fmt.Sprintf("%d setting(s) to change:", len(diffs)))
		for _, d := range diffs {
			output.Info(fmt.Sprintf("  %s: %s -> %s", d.Name, d.Current, d.Expected))
		}

		if !force {
			output.Info("Use --force to apply without confirmation")
		}

		// Apply settings
		output.Info("Applying settings...")
		settings := presetToRepoSettings(preset)
		if err = ghClient.UpdateRepoSettings(ctx, repo, settings); err != nil {
			return fmt.Errorf("failed to apply settings: %w", err)
		}
		output.Success("Settings applied")
	}

	// Set topics if provided
	if topics != "" {
		topicList := strings.Split(topics, ",")
		for i := range topicList {
			topicList[i] = strings.TrimSpace(topicList[i])
		}
		output.Info("Setting topics...")
		if err = ghClient.SetTopics(ctx, repo, topicList); err != nil {
			output.Warn(fmt.Sprintf("Failed to set topics: %v", err))
		} else {
			output.Success(fmt.Sprintf("Topics set: %s", topics))
		}
	}

	// Sync labels
	if len(preset.Labels) > 0 {
		output.Info("Syncing labels...")
		labels := presetLabelsToGH(preset.Labels)
		if err = ghClient.SyncLabels(ctx, repo, labels); err != nil {
			output.Warn(fmt.Sprintf("Failed to sync labels: %v", err))
		} else {
			output.Success(fmt.Sprintf("%d labels synced", len(labels)))
		}
	}

	// Sync rulesets
	for _, rc := range preset.Rulesets {
		output.Info(fmt.Sprintf("Syncing ruleset: %s...", rc.Name))
		ruleset := configRulesetToGH(&rc)
		if err = ghClient.CreateOrUpdateRuleset(ctx, repo, ruleset); err != nil {
			output.Warn(fmt.Sprintf("Ruleset %q: %v", rc.Name, err))
		} else {
			output.Success(fmt.Sprintf("Ruleset %q synced", rc.Name))
		}
	}

	// Update DB if available
	updateRepoSettingsInDB(ctx, repo, preset)

	output.Success(fmt.Sprintf("Settings apply complete for %s", repo))
	return nil
}

func computeSettingsDiffs(current *gh.RepoSettings, preset *config.SettingsPreset) []settingsDiff {
	var diffs []settingsDiff

	check := func(name string, cur, exp bool) {
		if cur != exp {
			diffs = append(diffs, settingsDiff{name, fmt.Sprintf("%v", cur), fmt.Sprintf("%v", exp)})
		}
	}
	checkStr := func(name, cur, exp string) {
		if cur != exp && exp != "" {
			diffs = append(diffs, settingsDiff{name, cur, exp})
		}
	}

	check("has_issues", current.HasIssues, preset.HasIssues)
	check("has_wiki", current.HasWiki, preset.HasWiki)
	check("has_projects", current.HasProjects, preset.HasProjects)
	check("has_discussions", current.HasDiscussions, preset.HasDiscussions)
	check("allow_squash_merge", current.AllowSquashMerge, preset.AllowSquashMerge)
	check("allow_merge_commit", current.AllowMergeCommit, preset.AllowMergeCommit)
	check("allow_rebase_merge", current.AllowRebaseMerge, preset.AllowRebaseMerge)
	check("delete_branch_on_merge", current.DeleteBranchOnMerge, preset.DeleteBranchOnMerge)
	check("allow_auto_merge", current.AllowAutoMerge, preset.AllowAutoMerge)
	check("allow_update_branch", current.AllowUpdateBranch, preset.AllowUpdateBranch)
	checkStr("squash_merge_commit_title", current.SquashMergeCommitTitle, preset.SquashMergeCommitTitle)
	checkStr("squash_merge_commit_message", current.SquashMergeCommitMessage, preset.SquashMergeCommitMessage)

	return diffs
}

func printPresetSettings(p *config.SettingsPreset) {
	output.Info(fmt.Sprintf("  has_issues: %v", p.HasIssues))
	output.Info(fmt.Sprintf("  has_wiki: %v", p.HasWiki))
	output.Info(fmt.Sprintf("  has_projects: %v", p.HasProjects))
	output.Info(fmt.Sprintf("  has_discussions: %v", p.HasDiscussions))
	output.Info(fmt.Sprintf("  allow_squash_merge: %v", p.AllowSquashMerge))
	output.Info(fmt.Sprintf("  allow_merge_commit: %v", p.AllowMergeCommit))
	output.Info(fmt.Sprintf("  allow_rebase_merge: %v", p.AllowRebaseMerge))
	output.Info(fmt.Sprintf("  delete_branch_on_merge: %v", p.DeleteBranchOnMerge))
	output.Info(fmt.Sprintf("  allow_auto_merge: %v", p.AllowAutoMerge))
	output.Info(fmt.Sprintf("  allow_update_branch: %v", p.AllowUpdateBranch))
	if p.SquashMergeCommitTitle != "" {
		output.Info(fmt.Sprintf("  squash_merge_commit_title: %s", p.SquashMergeCommitTitle))
	}
	if p.SquashMergeCommitMessage != "" {
		output.Info(fmt.Sprintf("  squash_merge_commit_message: %s", p.SquashMergeCommitMessage))
	}
}

// updateRepoSettingsInDB tries to update the Repo record in the database with preset values
func updateRepoSettingsInDB(ctx context.Context, repoFullName string, preset *config.SettingsPreset) {
	database, err := openDatabase()
	if err != nil {
		return // DB not available, skip
	}
	defer func() { _ = database.Close() }()

	gormDB := database.DB()

	// Find repo by full name
	parts := strings.Split(repoFullName, "/")
	if len(parts) != 2 {
		return
	}

	repoRepo := db.NewRepoRepository(gormDB)
	repo, err := repoRepo.GetByFullName(ctx, parts[0], parts[1])
	if err != nil {
		return // Repo not in DB, skip
	}

	// Update merge settings on the Repo record
	repo.AllowSquashMerge = preset.AllowSquashMerge
	repo.AllowMergeCommit = preset.AllowMergeCommit
	repo.AllowRebaseMerge = preset.AllowRebaseMerge
	repo.DeleteBranchOnMerge = preset.DeleteBranchOnMerge
	repo.SquashMergeCommitTitle = preset.SquashMergeCommitTitle
	repo.SquashMergeCommitMessage = preset.SquashMergeCommitMessage

	_ = repoRepo.Update(ctx, repo)
}
