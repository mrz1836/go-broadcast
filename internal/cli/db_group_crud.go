package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// groupListResult is the JSON-serializable result for group list
type groupListResult struct {
	ExternalID  string `json:"external_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
	TargetCount int    `json:"target_count"`
}

// groupDetailResult is the JSON-serializable result for group get
type groupDetailResult struct {
	ExternalID   string               `json:"external_id"`
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	Priority     int                  `json:"priority"`
	Enabled      bool                 `json:"enabled"`
	Source       *groupSourceResult   `json:"source,omitempty"`
	Global       *groupGlobalResult   `json:"global,omitempty"`
	Defaults     *groupDefaultResult  `json:"defaults,omitempty"`
	Targets      []groupTargetSummary `json:"targets,omitempty"`
	Dependencies []string             `json:"dependencies,omitempty"`
}

type groupSourceResult struct {
	Repo          string `json:"repo"`
	Branch        string `json:"branch"`
	BlobSizeLimit string `json:"blob_size_limit,omitempty"`
	SecurityEmail string `json:"security_email,omitempty"`
	SupportEmail  string `json:"support_email,omitempty"`
}

type groupGlobalResult struct {
	PRLabels        []string `json:"pr_labels,omitempty"`
	PRAssignees     []string `json:"pr_assignees,omitempty"`
	PRReviewers     []string `json:"pr_reviewers,omitempty"`
	PRTeamReviewers []string `json:"pr_team_reviewers,omitempty"`
}

type groupDefaultResult struct {
	BranchPrefix    string   `json:"branch_prefix,omitempty"`
	PRLabels        []string `json:"pr_labels,omitempty"`
	PRAssignees     []string `json:"pr_assignees,omitempty"`
	PRReviewers     []string `json:"pr_reviewers,omitempty"`
	PRTeamReviewers []string `json:"pr_team_reviewers,omitempty"`
}

type groupTargetSummary struct {
	Repo     string `json:"repo"`
	Branch   string `json:"branch,omitempty"`
	Position int    `json:"position"`
}

// newDBGroupCmd creates the "db group" command with subcommands
func newDBGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manage sync groups",
		Long:  `Create, read, update, and delete sync groups in the database.`,
	}

	cmd.AddCommand(
		newDBGroupListCmd(),
		newDBGroupGetCmd(),
		newDBGroupCreateCmd(),
		newDBGroupUpdateCmd(),
		newDBGroupDeleteCmd(),
		newDBGroupEnableCmd(),
		newDBGroupDisableCmd(),
	)

	return cmd
}

func newDBGroupListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all groups",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runGroupList(jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runGroupList(jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("group", "listed", err.Error(),
			"run 'go-broadcast db init' to create a database", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	cfg, err := getDefaultConfig(ctx, gormDB)
	if err != nil {
		return printErrorResponse("group", "listed", err.Error(),
			"run 'go-broadcast db import <file>' to import a configuration", jsonOutput)
	}

	groupRepo := db.NewGroupRepository(gormDB)
	groups, err := groupRepo.ListWithAssociations(ctx, cfg.ID)
	if err != nil {
		return printErrorResponse("group", "listed", err.Error(), "", jsonOutput)
	}

	results := make([]groupListResult, 0, len(groups))
	for _, g := range groups {
		results = append(results, groupListResult{
			ExternalID:  g.ExternalID,
			Name:        g.Name,
			Description: g.Description,
			Priority:    g.Priority,
			Enabled:     g.Enabled == nil || *g.Enabled,
			TargetCount: len(g.Targets),
		})
	}

	resp := CLIResponse{
		Success: true,
		Action:  "listed",
		Type:    "group",
		Data:    results,
		Count:   len(results),
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Found %d group(s):", len(results)))
	for _, r := range results {
		enabled := "enabled"
		if !r.Enabled {
			enabled = "disabled"
		}
		output.Info(fmt.Sprintf("  %s (%s) - %s, %d targets, priority %d",
			r.ExternalID, r.Name, enabled, r.TargetCount, r.Priority))
	}
	return nil
}

func newDBGroupGetCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Show full group details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runGroupGet(args[0], jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runGroupGet(externalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("group", "get", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("group", "get", err.Error(),
			"run 'go-broadcast db group list --json' to see available groups", jsonOutput)
	}

	// Load full associations
	groupRepo := db.NewGroupRepository(gormDB)
	groups, err := groupRepo.ListWithAssociations(ctx, group.ConfigID)
	if err != nil {
		return printErrorResponse("group", "get", err.Error(), "", jsonOutput)
	}

	var fullGroup *db.Group
	for _, g := range groups {
		if g.ExternalID == externalID {
			fullGroup = g
			break
		}
	}
	if fullGroup == nil {
		return printErrorResponse("group", "get", "group not found after load", "", jsonOutput)
	}

	result := buildGroupDetailResult(ctx, gormDB, fullGroup)

	resp := CLIResponse{
		Success: true,
		Action:  "get",
		Type:    "group",
		Data:    result,
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Group: %s (%s)", result.ExternalID, result.Name))
	if result.Description != "" {
		output.Info(fmt.Sprintf("  Description: %s", result.Description))
	}
	output.Info(fmt.Sprintf("  Priority: %d, Enabled: %v", result.Priority, result.Enabled))
	if result.Source != nil {
		output.Info(fmt.Sprintf("  Source: %s (branch: %s)", result.Source.Repo, result.Source.Branch))
	}
	output.Info(fmt.Sprintf("  Targets: %d", len(result.Targets)))
	for _, t := range result.Targets {
		output.Info(fmt.Sprintf("    - %s", t.Repo))
	}
	return nil
}

func buildGroupDetailResult(ctx context.Context, gormDB *gorm.DB, g *db.Group) groupDetailResult {
	result := groupDetailResult{
		ExternalID:  g.ExternalID,
		Name:        g.Name,
		Description: g.Description,
		Priority:    g.Priority,
		Enabled:     g.Enabled == nil || *g.Enabled,
	}

	// Source
	if g.Source.ID != 0 {
		// Load repo ref for source
		var repo db.Repo
		if err := gormDB.WithContext(ctx).Preload("Organization").First(&repo, g.Source.RepoID).Error; err == nil {
			result.Source = &groupSourceResult{
				Repo:          repo.FullName(),
				Branch:        g.Source.Branch,
				BlobSizeLimit: g.Source.BlobSizeLimit,
				SecurityEmail: g.Source.SecurityEmail,
				SupportEmail:  g.Source.SupportEmail,
			}
		}
	}

	// Global
	if g.GroupGlobal.ID != 0 {
		result.Global = &groupGlobalResult{
			PRLabels:        []string(g.GroupGlobal.PRLabels),
			PRAssignees:     []string(g.GroupGlobal.PRAssignees),
			PRReviewers:     []string(g.GroupGlobal.PRReviewers),
			PRTeamReviewers: []string(g.GroupGlobal.PRTeamReviewers),
		}
	}

	// Defaults
	if g.GroupDefault.ID != 0 {
		result.Defaults = &groupDefaultResult{
			BranchPrefix:    g.GroupDefault.BranchPrefix,
			PRLabels:        []string(g.GroupDefault.PRLabels),
			PRAssignees:     []string(g.GroupDefault.PRAssignees),
			PRReviewers:     []string(g.GroupDefault.PRReviewers),
			PRTeamReviewers: []string(g.GroupDefault.PRTeamReviewers),
		}
	}

	// Targets
	for _, t := range g.Targets {
		var repo db.Repo
		repoName := fmt.Sprintf("repo-id-%d", t.RepoID)
		if err := gormDB.WithContext(ctx).Preload("Organization").First(&repo, t.RepoID).Error; err == nil {
			repoName = repo.FullName()
		}
		result.Targets = append(result.Targets, groupTargetSummary{
			Repo:     repoName,
			Branch:   t.Branch,
			Position: t.Position,
		})
	}

	// Dependencies
	for _, dep := range g.Dependencies {
		result.Dependencies = append(result.Dependencies, dep.DependsOnID)
	}

	return result
}

func newDBGroupCreateCmd() *cobra.Command {
	var (
		jsonOutput   bool
		id           string
		name         string
		sourceRepo   string
		sourceBranch string
		description  string
		priority     int
		disabled     bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new sync group",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runGroupCreate(id, name, sourceRepo, sourceBranch, description, priority, disabled, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&id, "id", "", "Group external ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Group name (required)")
	cmd.Flags().StringVar(&sourceRepo, "source-repo", "", "Source repository (org/repo) (required)")
	cmd.Flags().StringVar(&sourceBranch, "source-branch", "", "Source branch (required)")
	cmd.Flags().StringVar(&description, "description", "", "Group description")
	cmd.Flags().IntVar(&priority, "priority", 0, "Group priority")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Create group as disabled")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("source-repo")
	_ = cmd.MarkFlagRequired("source-branch")
	return cmd
}

func runGroupCreate(id, name, sourceRepo, sourceBranch, description string, priority int, disabled, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("group", "created", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	cfg, err := getDefaultConfig(ctx, gormDB)
	if err != nil {
		return printErrorResponse("group", "created", err.Error(),
			"run 'go-broadcast db import <file>' to import a configuration first", jsonOutput)
	}

	// Check for duplicate external ID
	groupRepo := db.NewGroupRepository(gormDB)
	if _, err = groupRepo.GetByExternalID(ctx, id); err == nil {
		return printErrorResponse("group", "created",
			fmt.Sprintf("group %q already exists", id),
			"use 'go-broadcast db group update' to modify existing groups", jsonOutput)
	}

	// Count existing groups for position
	existingGroups, err := groupRepo.List(ctx, cfg.ID)
	if err != nil {
		return printErrorResponse("group", "created", err.Error(), "", jsonOutput)
	}

	enabled := !disabled

	// Create group + source + global + default in transaction
	err = gormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create group
		group := &db.Group{
			ConfigID:    cfg.ID,
			ExternalID:  id,
			Name:        name,
			Description: description,
			Priority:    priority,
			Enabled:     &enabled,
			Position:    len(existingGroups),
		}
		if txErr := tx.Create(group).Error; txErr != nil {
			return fmt.Errorf("failed to create group: %w", txErr)
		}

		// Resolve source repo (auto-creates client/org/repo)
		repoRepo := db.NewRepoRepository(tx)
		repo, txErr := repoRepo.FindOrCreateFromFullName(ctx, sourceRepo, 0)
		if txErr != nil {
			return fmt.Errorf("failed to resolve source repo: %w", txErr)
		}

		// Create source
		source := &db.Source{
			GroupID: group.ID,
			RepoID:  repo.ID,
			Branch:  sourceBranch,
		}
		if txErr = tx.Create(source).Error; txErr != nil {
			return fmt.Errorf("failed to create source: %w", txErr)
		}

		// Create empty GroupGlobal
		global := &db.GroupGlobal{GroupID: group.ID}
		if txErr = tx.Create(global).Error; txErr != nil {
			return fmt.Errorf("failed to create group global: %w", txErr)
		}

		// Create empty GroupDefault
		defaults := &db.GroupDefault{GroupID: group.ID}
		if txErr = tx.Create(defaults).Error; txErr != nil {
			return fmt.Errorf("failed to create group defaults: %w", txErr)
		}

		return nil
	})
	if err != nil {
		return printErrorResponse("group", "created", err.Error(), "", jsonOutput)
	}

	result := groupListResult{
		ExternalID:  id,
		Name:        name,
		Description: description,
		Priority:    priority,
		Enabled:     enabled,
		TargetCount: 0,
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "created",
		Type:    "group",
		Data:    result,
	}, jsonOutput)
}

func newDBGroupUpdateCmd() *cobra.Command {
	var (
		jsonOutput  bool
		name        string
		description string
		priority    int
		enabled     bool
		disabled    bool
	)
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update group fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGroupUpdate(cmd, args[0], name, description, priority, enabled, disabled, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&name, "name", "", "New group name")
	cmd.Flags().StringVar(&description, "description", "", "New group description")
	cmd.Flags().IntVar(&priority, "priority", 0, "New group priority")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Enable the group")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Disable the group")
	return cmd
}

func runGroupUpdate(cmd *cobra.Command, externalID, name, description string, priority int, enabled, disabled, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("group", "updated", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("group", "updated", err.Error(),
			"run 'go-broadcast db group list --json' to see available groups", jsonOutput)
	}

	if cmd.Flags().Changed("name") {
		group.Name = name
	}
	if cmd.Flags().Changed("description") {
		group.Description = description
	}
	if cmd.Flags().Changed("priority") {
		group.Priority = priority
	}
	if enabled {
		e := true
		group.Enabled = &e
	}
	if disabled {
		e := false
		group.Enabled = &e
	}

	groupRepo := db.NewGroupRepository(gormDB)
	if err = groupRepo.Update(ctx, group); err != nil {
		return printErrorResponse("group", "updated", err.Error(), "", jsonOutput)
	}

	result := groupListResult{
		ExternalID:  group.ExternalID,
		Name:        group.Name,
		Description: group.Description,
		Priority:    group.Priority,
		Enabled:     group.Enabled == nil || *group.Enabled,
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "updated",
		Type:    "group",
		Data:    result,
	}, jsonOutput)
}

func newDBGroupDeleteCmd() *cobra.Command {
	var (
		jsonOutput bool
		hard       bool
	)
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runGroupDelete(args[0], hard, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&hard, "hard", false, "Hard delete (permanent)")
	return cmd
}

func runGroupDelete(externalID string, hard, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("group", "deleted", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("group", "deleted", err.Error(),
			"run 'go-broadcast db group list --json' to see available groups", jsonOutput)
	}

	groupRepo := db.NewGroupRepository(gormDB)
	if err = groupRepo.Delete(ctx, group.ID, hard); err != nil {
		return printErrorResponse("group", "deleted", err.Error(), "", jsonOutput)
	}

	deleteType := "soft-deleted"
	if hard {
		deleteType = "hard-deleted"
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  deleteType,
		Type:    "group",
		Data:    map[string]string{"external_id": externalID},
	}, jsonOutput)
}

func newDBGroupEnableCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "enable <id>",
		Short: "Enable a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runGroupSetEnabled(args[0], true, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func newDBGroupDisableCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "disable <id>",
		Short: "Disable a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runGroupSetEnabled(args[0], false, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runGroupSetEnabled(externalID string, enabled, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("group", "updated", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("group", "updated", err.Error(),
			"run 'go-broadcast db group list --json' to see available groups", jsonOutput)
	}

	group.Enabled = &enabled

	groupRepo := db.NewGroupRepository(gormDB)
	if err = groupRepo.Update(ctx, group); err != nil {
		return printErrorResponse("group", "updated", err.Error(), "", jsonOutput)
	}

	action := "enabled"
	if !enabled {
		action = "disabled"
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  action,
		Type:    "group",
		Data: map[string]interface{}{
			"external_id": externalID,
			"enabled":     enabled,
		},
	}, jsonOutput)
}

// groupHintList returns a hint for listing groups
func groupHintList() string {
	return "run 'go-broadcast db group list --json' to see available groups"
}
