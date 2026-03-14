package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// presetListResult is the JSON-serializable result for preset list
type presetListResult struct {
	ExternalID   string `json:"external_id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	LabelCount   int    `json:"label_count"`
	RulesetCount int    `json:"ruleset_count"`
}

// presetDetailResult is the JSON-serializable result for preset get
type presetDetailResult struct {
	ExternalID               string                `json:"external_id"`
	Name                     string                `json:"name"`
	Description              string                `json:"description,omitempty"`
	HasIssues                bool                  `json:"has_issues"`
	HasWiki                  bool                  `json:"has_wiki"`
	HasProjects              bool                  `json:"has_projects"`
	HasDiscussions           bool                  `json:"has_discussions"`
	AllowSquashMerge         bool                  `json:"allow_squash_merge"`
	AllowMergeCommit         bool                  `json:"allow_merge_commit"`
	AllowRebaseMerge         bool                  `json:"allow_rebase_merge"`
	DeleteBranchOnMerge      bool                  `json:"delete_branch_on_merge"`
	AllowAutoMerge           bool                  `json:"allow_auto_merge"`
	AllowUpdateBranch        bool                  `json:"allow_update_branch"`
	SquashMergeCommitTitle   string                `json:"squash_merge_commit_title,omitempty"`
	SquashMergeCommitMessage string                `json:"squash_merge_commit_message,omitempty"`
	Labels                   []presetLabelResult   `json:"labels,omitempty"`
	Rulesets                 []presetRulesetResult `json:"rulesets,omitempty"`
}

type presetLabelResult struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
}

type presetRulesetResult struct {
	Name        string   `json:"name"`
	Target      string   `json:"target"`
	Enforcement string   `json:"enforcement"`
	Include     []string `json:"include"`
	Rules       []string `json:"rules"`
}

// newDBPresetCmd creates the "db preset" command with subcommands
func newDBPresetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preset",
		Short: "Manage settings presets",
		Long:  `Create, read, update, and delete repository settings presets in the database.`,
	}

	cmd.AddCommand(
		newDBPresetListCmd(),
		newDBPresetShowCmd(),
		newDBPresetCreateCmd(),
		newDBPresetDeleteCmd(),
		newDBPresetAssignCmd(),
		newDBPresetImportCmd(),
	)

	return cmd
}

func newDBPresetListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all settings presets",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runPresetList(jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runPresetList(jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("preset", "listed", err.Error(),
			"run 'go-broadcast db init' to create a database", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	presetRepo := db.NewSettingsPresetRepository(database.DB())
	presets, err := presetRepo.List(ctx)
	if err != nil {
		return printErrorResponse("preset", "listed", err.Error(), "", jsonOutput)
	}

	results := make([]presetListResult, 0, len(presets))
	for _, p := range presets {
		results = append(results, presetListResult{
			ExternalID:   p.ExternalID,
			Name:         p.Name,
			Description:  p.Description,
			LabelCount:   len(p.Labels),
			RulesetCount: len(p.Rulesets),
		})
	}

	resp := CLIResponse{
		Success: true,
		Action:  "listed",
		Type:    "preset",
		Data:    results,
		Count:   len(results),
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Found %d preset(s):", len(results)))
	for _, r := range results {
		output.Info(fmt.Sprintf("  %s (%s) - %d labels, %d rulesets",
			r.ExternalID, r.Name, r.LabelCount, r.RulesetCount))
	}
	return nil
}

func newDBPresetShowCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show preset details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPresetShow(args[0], jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runPresetShow(externalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("preset", "show", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	presetRepo := db.NewSettingsPresetRepository(database.DB())
	preset, err := presetRepo.GetByExternalID(ctx, externalID)
	if err != nil {
		return printErrorResponse("preset", "show", err.Error(),
			"run 'go-broadcast db preset list' to see available presets", jsonOutput)
	}

	result := buildPresetDetailResult(preset)

	resp := CLIResponse{
		Success: true,
		Action:  "show",
		Type:    "preset",
		Data:    result,
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Preset: %s (%s)", result.ExternalID, result.Name))
	if result.Description != "" {
		output.Info(fmt.Sprintf("  Description: %s", result.Description))
	}
	output.Info(fmt.Sprintf("  Issues: %v, Wiki: %v, Projects: %v, Discussions: %v",
		result.HasIssues, result.HasWiki, result.HasProjects, result.HasDiscussions))
	output.Info(fmt.Sprintf("  Squash: %v, Merge: %v, Rebase: %v",
		result.AllowSquashMerge, result.AllowMergeCommit, result.AllowRebaseMerge))
	output.Info(fmt.Sprintf("  Delete branch: %v, Auto-merge: %v, Update branch: %v",
		result.DeleteBranchOnMerge, result.AllowAutoMerge, result.AllowUpdateBranch))
	output.Info(fmt.Sprintf("  Labels: %d, Rulesets: %d", len(result.Labels), len(result.Rulesets)))
	return nil
}

func buildPresetDetailResult(p *db.SettingsPreset) presetDetailResult {
	result := presetDetailResult{
		ExternalID:               p.ExternalID,
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
		result.Labels = append(result.Labels, presetLabelResult{
			Name:        l.Name,
			Color:       l.Color,
			Description: l.Description,
		})
	}

	for _, r := range p.Rulesets {
		result.Rulesets = append(result.Rulesets, presetRulesetResult{
			Name:        r.Name,
			Target:      r.Target,
			Enforcement: r.Enforcement,
			Include:     []string(r.Include),
			Rules:       []string(r.Rules),
		})
	}

	return result
}

func newDBPresetCreateCmd() *cobra.Command {
	var (
		jsonOutput  bool
		name        string
		description string
	)
	cmd := &cobra.Command{
		Use:   "create <id>",
		Short: "Create a new settings preset with defaults",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPresetCreate(args[0], name, description, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&name, "name", "", "Preset name (defaults to ID)")
	cmd.Flags().StringVar(&description, "description", "", "Preset description")
	return cmd
}

func runPresetCreate(id, name, description string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("preset", "created", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	if name == "" {
		name = id
	}

	ctx := context.Background()
	presetRepo := db.NewSettingsPresetRepository(database.DB())

	// Check for duplicate
	if _, lookupErr := presetRepo.GetByExternalID(ctx, id); lookupErr == nil {
		return printErrorResponse("preset", "created",
			fmt.Sprintf("preset %q already exists", id),
			"use 'go-broadcast db preset import' to update from config", jsonOutput)
	}

	// Use default preset values
	dflt := config.DefaultPreset()
	preset := &db.SettingsPreset{
		ExternalID:               id,
		Name:                     name,
		Description:              description,
		HasIssues:                dflt.HasIssues,
		HasWiki:                  dflt.HasWiki,
		HasProjects:              dflt.HasProjects,
		HasDiscussions:           dflt.HasDiscussions,
		AllowSquashMerge:         dflt.AllowSquashMerge,
		AllowMergeCommit:         dflt.AllowMergeCommit,
		AllowRebaseMerge:         dflt.AllowRebaseMerge,
		DeleteBranchOnMerge:      dflt.DeleteBranchOnMerge,
		AllowAutoMerge:           dflt.AllowAutoMerge,
		AllowUpdateBranch:        dflt.AllowUpdateBranch,
		SquashMergeCommitTitle:   dflt.SquashMergeCommitTitle,
		SquashMergeCommitMessage: dflt.SquashMergeCommitMessage,
	}

	if err = presetRepo.Create(ctx, preset); err != nil {
		return printErrorResponse("preset", "created", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "created",
		Type:    "preset",
		Data: presetListResult{
			ExternalID:  id,
			Name:        name,
			Description: description,
		},
	}, jsonOutput)
}

func newDBPresetDeleteCmd() *cobra.Command {
	var (
		jsonOutput bool
		hard       bool
	)
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a settings preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPresetDelete(args[0], hard, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&hard, "hard", false, "Hard delete (permanent)")
	return cmd
}

func runPresetDelete(externalID string, hard, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("preset", "deleted", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	presetRepo := db.NewSettingsPresetRepository(database.DB())
	preset, err := presetRepo.GetByExternalID(ctx, externalID)
	if err != nil {
		return printErrorResponse("preset", "deleted", err.Error(),
			"run 'go-broadcast db preset list' to see available presets", jsonOutput)
	}

	if err = presetRepo.Delete(ctx, preset.ID, hard); err != nil {
		return printErrorResponse("preset", "deleted", err.Error(), "", jsonOutput)
	}

	deleteType := "soft-deleted"
	if hard {
		deleteType = "hard-deleted"
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  deleteType,
		Type:    "preset",
		Data:    map[string]string{"external_id": externalID},
	}, jsonOutput)
}

func newDBPresetAssignCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "assign <preset-id> <repo>",
		Short: "Assign a preset to a repository",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPresetAssign(args[0], args[1], jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runPresetAssign(presetExternalID, repoFullName string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("preset", "assigned", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	presetRepo := db.NewSettingsPresetRepository(gormDB)
	preset, err := presetRepo.GetByExternalID(ctx, presetExternalID)
	if err != nil {
		return printErrorResponse("preset", "assigned",
			fmt.Sprintf("preset %q not found: %v", presetExternalID, err),
			"run 'go-broadcast db preset list' to see available presets", jsonOutput)
	}

	repoRepo := db.NewRepoRepository(gormDB)
	repo, err := repoRepo.FindOrCreateFromFullName(ctx, repoFullName, 0)
	if err != nil {
		return printErrorResponse("preset", "assigned",
			fmt.Sprintf("repo %q: %v", repoFullName, err), "", jsonOutput)
	}

	if err = presetRepo.AssignPresetToRepo(ctx, repo.ID, preset.ID); err != nil {
		return printErrorResponse("preset", "assigned", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "assigned",
		Type:    "preset",
		Data: map[string]string{
			"preset": presetExternalID,
			"repo":   repoFullName,
		},
	}, jsonOutput)
}

func newDBPresetImportCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "import [config-file]",
		Short: "Import presets from sync.yaml configuration",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			configFile := globalFlags.ConfigFile
			if len(args) > 0 {
				configFile = args[0]
			}
			return runPresetImport(configFile, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runPresetImport(configFile string, jsonOutput bool) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return printErrorResponse("preset", "imported", err.Error(),
			"check config file path", jsonOutput)
	}

	if len(cfg.SettingsPresets) == 0 {
		return printErrorResponse("preset", "imported",
			"no settings_presets found in configuration",
			"add settings_presets section to sync.yaml", jsonOutput)
	}

	database, dbErr := openDatabase()
	if dbErr != nil {
		return printErrorResponse("preset", "imported", dbErr.Error(),
			"run 'go-broadcast db init' to create a database", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	presetRepo := db.NewSettingsPresetRepository(database.DB())

	imported := 0
	for _, cp := range cfg.SettingsPresets {
		preset := configPresetToDBPreset(&cp)
		if importErr := presetRepo.ImportFromConfig(ctx, preset); importErr != nil {
			return printErrorResponse("preset", "imported",
				fmt.Sprintf("failed to import preset %q: %v", cp.ID, importErr), "", jsonOutput)
		}
		imported++
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "imported",
		Type:    "preset",
		Data:    map[string]int{"count": imported},
		Count:   imported,
	}, jsonOutput)
}

// configPresetToDBPreset converts a config.SettingsPreset to a db.SettingsPreset
func configPresetToDBPreset(cp *config.SettingsPreset) *db.SettingsPreset {
	preset := &db.SettingsPreset{
		ExternalID:               cp.ID,
		Name:                     cp.Name,
		Description:              cp.Description,
		HasIssues:                cp.HasIssues,
		HasWiki:                  cp.HasWiki,
		HasProjects:              cp.HasProjects,
		HasDiscussions:           cp.HasDiscussions,
		AllowSquashMerge:         cp.AllowSquashMerge,
		AllowMergeCommit:         cp.AllowMergeCommit,
		AllowRebaseMerge:         cp.AllowRebaseMerge,
		DeleteBranchOnMerge:      cp.DeleteBranchOnMerge,
		AllowAutoMerge:           cp.AllowAutoMerge,
		AllowUpdateBranch:        cp.AllowUpdateBranch,
		SquashMergeCommitTitle:   cp.SquashMergeCommitTitle,
		SquashMergeCommitMessage: cp.SquashMergeCommitMessage,
	}

	for _, l := range cp.Labels {
		preset.Labels = append(preset.Labels, db.SettingsPresetLabel{
			Name:        l.Name,
			Color:       l.Color,
			Description: l.Description,
		})
	}

	for _, r := range cp.Rulesets {
		enforcement := r.Enforcement
		if enforcement == "" {
			enforcement = "active"
		}
		preset.Rulesets = append(preset.Rulesets, db.SettingsPresetRuleset{
			Name:        r.Name,
			Target:      r.Target,
			Enforcement: enforcement,
			Include:     r.Include,
			Exclude:     r.Exclude,
			Rules:       r.Rules,
		})
	}

	return preset
}
