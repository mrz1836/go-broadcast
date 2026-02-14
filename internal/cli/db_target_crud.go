package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// targetListResult is the JSON-serializable result for target list
type targetListResult struct {
	Repo     string `json:"repo"`
	Branch   string `json:"branch,omitempty"`
	Position int    `json:"position"`
	GroupID  string `json:"group_id"`
}

// targetDetailResult is the JSON-serializable result for target get
type targetDetailResult struct {
	Repo              string              `json:"repo"`
	Branch            string              `json:"branch,omitempty"`
	PRLabels          []string            `json:"pr_labels,omitempty"`
	PRAssignees       []string            `json:"pr_assignees,omitempty"`
	PRReviewers       []string            `json:"pr_reviewers,omitempty"`
	PRTeamReviewers   []string            `json:"pr_team_reviewers,omitempty"`
	Files             []fileMappingResult `json:"files,omitempty"`
	Directories       []dirMappingResult  `json:"directories,omitempty"`
	FileListRefs      []string            `json:"file_list_refs,omitempty"`
	DirectoryListRefs []string            `json:"directory_list_refs,omitempty"`
	Position          int                 `json:"position"`
}

type fileMappingResult struct {
	Src    string `json:"src,omitempty"`
	Dest   string `json:"dest"`
	Delete bool   `json:"delete,omitempty"`
}

type dirMappingResult struct {
	Src               string   `json:"src,omitempty"`
	Dest              string   `json:"dest"`
	Exclude           []string `json:"exclude,omitempty"`
	IncludeOnly       []string `json:"include_only,omitempty"`
	PreserveStructure *bool    `json:"preserve_structure,omitempty"`
	Delete            bool     `json:"delete,omitempty"`
}

// newDBTargetCmd creates the "db target" command with subcommands
func newDBTargetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "target",
		Short: "Manage sync targets",
		Long:  `Add, remove, update, and query targets within sync groups.`,
	}

	cmd.AddCommand(
		newDBTargetListCmd(),
		newDBTargetGetCmd(),
		newDBTargetAddCmd(),
		newDBTargetRemoveCmd(),
		newDBTargetUpdateCmd(),
	)

	return cmd
}

func newDBTargetListCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List targets in a group",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTargetList(groupID, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	_ = cmd.MarkFlagRequired("group")
	return cmd
}

func runTargetList(groupExternalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("target", "listed", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("target", "listed", err.Error(), groupHintList(), jsonOutput)
	}

	targetRepo := db.NewTargetRepository(gormDB)
	targets, err := targetRepo.ListWithAssociations(ctx, group.ID)
	if err != nil {
		return printErrorResponse("target", "listed", err.Error(), "", jsonOutput)
	}

	results := make([]targetListResult, 0, len(targets))
	for _, t := range targets {
		repoName := resolveRepoName(ctx, gormDB, t.RepoID)
		results = append(results, targetListResult{
			Repo:     repoName,
			Branch:   t.Branch,
			Position: t.Position,
			GroupID:  groupExternalID,
		})
	}

	resp := CLIResponse{
		Success: true,
		Action:  "listed",
		Type:    "target",
		Data:    results,
		Count:   len(results),
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Found %d target(s) in group %q:", len(results), groupExternalID))
	for _, r := range results {
		branch := r.Branch
		if branch == "" {
			branch = "(default)"
		}
		output.Info(fmt.Sprintf("  %s (branch: %s)", r.Repo, branch))
	}
	return nil
}

func newDBTargetGetCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
	)
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show target details",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTargetGet(groupID, repo, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func runTargetGet(groupExternalID, repoFullName string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("target", "get", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("target", "get", err.Error(), groupHintList(), jsonOutput)
	}

	targetRepo := db.NewTargetRepository(gormDB)
	// Get all targets with associations and find our target
	targets, err := targetRepo.ListWithAssociations(ctx, group.ID)
	if err != nil {
		return printErrorResponse("target", "get", err.Error(), "", jsonOutput)
	}

	var target *db.Target
	for _, t := range targets {
		name := resolveRepoName(ctx, gormDB, t.RepoID)
		if strings.EqualFold(name, repoFullName) {
			target = t
			break
		}
	}
	if target == nil {
		return printErrorResponse("target", "get",
			fmt.Sprintf("target %q not found in group %q", repoFullName, groupExternalID),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json' to see available targets", groupExternalID),
			jsonOutput)
	}

	result := buildTargetDetailResult(ctx, gormDB, target)

	return printResponse(CLIResponse{
		Success: true,
		Action:  "get",
		Type:    "target",
		Data:    result,
	}, jsonOutput)
}

func buildTargetDetailResult(ctx context.Context, gormDB *gorm.DB, t *db.Target) targetDetailResult {
	result := targetDetailResult{
		Repo:            resolveRepoName(ctx, gormDB, t.RepoID),
		Branch:          t.Branch,
		PRLabels:        []string(t.PRLabels),
		PRAssignees:     []string(t.PRAssignees),
		PRReviewers:     []string(t.PRReviewers),
		PRTeamReviewers: []string(t.PRTeamReviewers),
		Position:        t.Position,
	}

	for _, fm := range t.FileMappings {
		result.Files = append(result.Files, fileMappingResult{
			Src:    fm.Src,
			Dest:   fm.Dest,
			Delete: fm.DeleteFlag,
		})
	}

	for _, dm := range t.DirectoryMappings {
		result.Directories = append(result.Directories, dirMappingResult{
			Src:               dm.Src,
			Dest:              dm.Dest,
			Exclude:           []string(dm.Exclude),
			IncludeOnly:       []string(dm.IncludeOnly),
			PreserveStructure: dm.PreserveStructure,
			Delete:            dm.DeleteFlag,
		})
	}

	for _, ref := range t.FileListRefs {
		if ref.FileList.ExternalID != "" {
			result.FileListRefs = append(result.FileListRefs, ref.FileList.ExternalID)
		}
	}

	for _, ref := range t.DirectoryListRefs {
		if ref.DirectoryList.ExternalID != "" {
			result.DirectoryListRefs = append(result.DirectoryListRefs, ref.DirectoryList.ExternalID)
		}
	}

	return result
}

func newDBTargetAddCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
		branch     string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a target to a group",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTargetAdd(groupID, repo, branch, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().StringVar(&branch, "branch", "", "Target branch")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func runTargetAdd(groupExternalID, repoFullName, branch string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("target", "created", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("target", "created", err.Error(), groupHintList(), jsonOutput)
	}

	// Check if target already exists (idempotent)
	targetRepo := db.NewTargetRepository(gormDB)
	existing, err := targetRepo.GetByRepoName(ctx, group.ID, repoFullName)
	if err == nil && existing != nil {
		// Already exists - return success (idempotent)
		result := targetListResult{
			Repo:     repoFullName,
			Branch:   existing.Branch,
			Position: existing.Position,
			GroupID:  groupExternalID,
		}
		return printResponse(CLIResponse{
			Success: true,
			Action:  "already_exists",
			Type:    "target",
			Data:    result,
			Hint:    "target already exists in this group",
		}, jsonOutput)
	}

	// Count existing targets for position
	existingTargets, err := targetRepo.List(ctx, group.ID)
	if err != nil {
		return printErrorResponse("target", "created", err.Error(), "", jsonOutput)
	}

	// Resolve repo (auto-creates client/org/repo)
	repoRepo := db.NewRepoRepository(gormDB)
	repoRecord, err := repoRepo.FindOrCreateFromFullName(ctx, repoFullName, 0)
	if err != nil {
		return printErrorResponse("target", "created", err.Error(), "", jsonOutput)
	}

	target := &db.Target{
		GroupID:  group.ID,
		RepoID:   repoRecord.ID,
		Branch:   branch,
		Position: len(existingTargets),
	}

	if err = targetRepo.Create(ctx, target); err != nil {
		return printErrorResponse("target", "created", err.Error(), "", jsonOutput)
	}

	result := targetListResult{
		Repo:     repoFullName,
		Branch:   branch,
		Position: target.Position,
		GroupID:  groupExternalID,
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "created",
		Type:    "target",
		Data:    result,
	}, jsonOutput)
}

func newDBTargetRemoveCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
		hard       bool
	)
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a target from a group",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTargetRemove(groupID, repo, hard, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().BoolVar(&hard, "hard", false, "Hard delete (permanent)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func runTargetRemove(groupExternalID, repoFullName string, hard, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("target", "deleted", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("target", "deleted", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("target", "deleted", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json' to see available targets", groupExternalID),
			jsonOutput)
	}

	targetRepo := db.NewTargetRepository(gormDB)
	if err = targetRepo.Delete(ctx, target.ID, hard); err != nil {
		return printErrorResponse("target", "deleted", err.Error(), "", jsonOutput)
	}

	deleteType := "soft-deleted"
	if hard {
		deleteType = "hard-deleted"
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  deleteType,
		Type:    "target",
		Data: map[string]string{
			"repo":     repoFullName,
			"group_id": groupExternalID,
		},
	}, jsonOutput)
}

func newDBTargetUpdateCmd() *cobra.Command {
	var (
		jsonOutput  bool
		groupID     string
		repo        string
		branch      string
		prLabels    string
		prAssignees string
		prReviewers string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update target settings",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTargetUpdate(cmd, groupID, repo, branch, prLabels, prAssignees, prReviewers, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().StringVar(&branch, "branch", "", "Target branch")
	cmd.Flags().StringVar(&prLabels, "pr-labels", "", "Comma-separated PR labels")
	cmd.Flags().StringVar(&prAssignees, "pr-assignees", "", "Comma-separated PR assignees")
	cmd.Flags().StringVar(&prReviewers, "pr-reviewers", "", "Comma-separated PR reviewers")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func runTargetUpdate(cmd *cobra.Command, groupExternalID, repoFullName, branch, prLabels, prAssignees, prReviewers string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("target", "updated", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("target", "updated", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("target", "updated", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json' to see available targets", groupExternalID),
			jsonOutput)
	}

	if cmd.Flags().Changed("branch") {
		target.Branch = branch
	}
	if cmd.Flags().Changed("pr-labels") {
		target.PRLabels = splitCSV(prLabels)
	}
	if cmd.Flags().Changed("pr-assignees") {
		target.PRAssignees = splitCSV(prAssignees)
	}
	if cmd.Flags().Changed("pr-reviewers") {
		target.PRReviewers = splitCSV(prReviewers)
	}

	targetRepo := db.NewTargetRepository(gormDB)
	if err = targetRepo.Update(ctx, target); err != nil {
		return printErrorResponse("target", "updated", err.Error(), "", jsonOutput)
	}

	result := targetListResult{
		Repo:     repoFullName,
		Branch:   target.Branch,
		Position: target.Position,
		GroupID:  groupExternalID,
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "updated",
		Type:    "target",
		Data:    result,
	}, jsonOutput)
}

// resolveRepoName resolves a repo ID to its "org/repo" full name
func resolveRepoName(ctx context.Context, gormDB *gorm.DB, repoID uint) string {
	var repo db.Repo
	if err := gormDB.WithContext(ctx).Preload("Organization").First(&repo, repoID).Error; err == nil {
		return repo.FullName()
	}
	return fmt.Sprintf("repo-id-%d", repoID)
}

// splitCSV splits a comma-separated string into a JSONStringSlice, trimming whitespace
func splitCSV(s string) db.JSONStringSlice {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make(db.JSONStringSlice, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
