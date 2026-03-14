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

// newScaffoldCmd creates the "scaffold" command
func newScaffoldCmd() *cobra.Command {
	var (
		presetID    string
		topics      string
		description string
		noClone     bool
		noFiles     bool
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "scaffold <owner/name> <description>",
		Short: "Create a new GitHub repository with standard settings",
		Long: `Create a new private GitHub repository with settings from a preset.

Applies repository settings, labels, and rulesets from the specified preset.
Repositories are always created as private — make public manually when ready.`,
		Example: `  # Create with default (mvp) preset
  go-broadcast scaffold mrz1836/my-new-repo "A cool project"

  # Create with specific preset
  go-broadcast scaffold mrz1836/my-lib "Go library" --preset go-lib

  # Preview what would be created
  go-broadcast scaffold mrz1836/my-repo "Test" --dry-run

  # Create with topics
  go-broadcast scaffold mrz1836/my-repo "Test" --topics "go,library,tools"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScaffold(cmd.Context(), args[0], args[1], presetID, topics, description, noClone, noFiles, dryRun)
		},
	}

	cmd.Flags().StringVar(&presetID, "preset", "mvp", "Settings preset ID")
	cmd.Flags().StringVar(&topics, "topics", "", "Comma-separated topics")
	cmd.Flags().StringVar(&description, "description", "", "Override description (uses positional arg by default)")
	cmd.Flags().BoolVar(&noClone, "no-clone", false, "Don't clone repository after creation")
	cmd.Flags().BoolVar(&noFiles, "no-files", false, "Skip initial file creation")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without creating")

	return cmd
}

func runScaffold(ctx context.Context, repoName, desc, presetID, topics string, _ string, noClone, noFiles, dryRun bool) error {
	// Parse owner/name
	parts := strings.Split(repoName, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid repo format %q, expected owner/name", repoName) //nolint:err113 // user-facing
	}

	// Resolve preset: try DB first, fall back to config, then hardcoded default
	preset := resolvePreset(ctx, presetID)

	// Parse topics
	var topicList []string
	if topics != "" {
		topicList = strings.Split(topics, ",")
		for i := range topicList {
			topicList[i] = strings.TrimSpace(topicList[i])
		}
	}

	opts := ScaffoldOptions{
		Name:        parts[1],
		Description: desc,
		Owner:       parts[0],
		Preset:      preset,
		Topics:      topicList,
		NoClone:     noClone,
		NoFiles:     noFiles,
		DryRun:      dryRun,
	}

	if dryRun {
		// No need for a real GH client in dry-run mode
		_, err := RunScaffold(ctx, nil, opts)
		return err
	}

	// Initialize GitHub client
	logger := logrus.StandardLogger()
	ghClient, err := gh.NewClient(ctx, logger, &logging.LogConfig{})
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	result, err := RunScaffold(ctx, ghClient, opts)
	if err != nil {
		return err
	}

	output.Success(fmt.Sprintf("Repository scaffolded: %s", result.RepoFullName))
	return nil
}

// resolvePreset looks up a preset from DB first, then config file, then falls back to default
func resolvePreset(ctx context.Context, presetID string) *config.SettingsPreset {
	// Try DB lookup
	database, err := openDatabase()
	if err == nil {
		defer func() { _ = database.Close() }()
		presetRepo := db.NewSettingsPresetRepository(database.DB())
		dbPreset, dbErr := presetRepo.GetByExternalID(ctx, presetID)
		if dbErr == nil {
			// Convert DB preset to config preset
			compat := &dbSettingsPresetCompat{
				ExternalID:               dbPreset.ExternalID,
				Name:                     dbPreset.Name,
				Description:              dbPreset.Description,
				HasIssues:                dbPreset.HasIssues,
				HasWiki:                  dbPreset.HasWiki,
				HasProjects:              dbPreset.HasProjects,
				HasDiscussions:           dbPreset.HasDiscussions,
				AllowSquashMerge:         dbPreset.AllowSquashMerge,
				AllowMergeCommit:         dbPreset.AllowMergeCommit,
				AllowRebaseMerge:         dbPreset.AllowRebaseMerge,
				DeleteBranchOnMerge:      dbPreset.DeleteBranchOnMerge,
				AllowAutoMerge:           dbPreset.AllowAutoMerge,
				AllowUpdateBranch:        dbPreset.AllowUpdateBranch,
				SquashMergeCommitTitle:   dbPreset.SquashMergeCommitTitle,
				SquashMergeCommitMessage: dbPreset.SquashMergeCommitMessage,
			}
			for _, l := range dbPreset.Labels {
				compat.Labels = append(compat.Labels, dbLabelCompat{
					Name: l.Name, Color: l.Color, Description: l.Description,
				})
			}
			for _, r := range dbPreset.Rulesets {
				compat.Rulesets = append(compat.Rulesets, dbRulesetCompat{
					Name: r.Name, Target: r.Target, Enforcement: r.Enforcement,
					Include: []string(r.Include), Exclude: []string(r.Exclude), Rules: []string(r.Rules),
				})
			}
			return dbPresetToConfigPreset(compat)
		}
	}

	// Try config file
	cfg, cfgErr := config.Load(globalFlags.ConfigFile)
	if cfgErr == nil {
		if p := cfg.GetPreset(presetID); p != nil {
			return p
		}
	}

	// Fall back to hardcoded default
	dflt := config.DefaultPreset()
	dflt.ID = presetID
	return &dflt
}
