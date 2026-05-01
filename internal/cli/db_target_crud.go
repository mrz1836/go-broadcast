package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
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
		newDBTargetCloneCmd(),
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

	if IsDryRun() {
		return printDryRunResponse(CLIResponse{
			Action: "created",
			Type:   "target",
			Data: targetListResult{
				Repo:     repoFullName,
				Branch:   branch,
				Position: len(existingTargets),
				GroupID:  groupExternalID,
			},
		}, fmt.Sprintf("create target %q in group %q", repoFullName, groupExternalID), jsonOutput)
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

	if IsDryRun() {
		return printDryRunResponse(CLIResponse{
			Action: deleteAction(hard),
			Type:   "target",
			Data: map[string]string{
				"repo":     repoFullName,
				"group_id": groupExternalID,
			},
		}, fmt.Sprintf("%s target %q from group %q", dryRunDeleteVerb(hard), repoFullName, groupExternalID), jsonOutput)
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

	result := targetListResult{
		Repo:     repoFullName,
		Branch:   target.Branch,
		Position: target.Position,
		GroupID:  groupExternalID,
	}

	if IsDryRun() {
		return printDryRunResponse(CLIResponse{
			Action: "updated",
			Type:   "target",
			Data:   result,
		}, fmt.Sprintf("update target %q in group %q", repoFullName, groupExternalID), jsonOutput)
	}

	targetRepo := db.NewTargetRepository(gormDB)
	if err = targetRepo.Update(ctx, target); err != nil {
		return printErrorResponse("target", "updated", err.Error(), "", jsonOutput)
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

func newDBTargetCloneCmd() *cobra.Command {
	var (
		jsonOutput      bool
		groupID         string
		from            string
		to              string
		branch          string
		prLabels        string
		prAssignees     string
		prReviewers     string
		prTeamReviewers string
		createRepo      bool
		presetID        string
		topicsStr       string
		noFiles         bool
		noCloneRepo     bool
	)
	cmd := &cobra.Command{
		Use:   "clone",
		Short: "Clone a target with all its mappings",
		Long: `Deep-copy a target and all its child records (file mappings, directory mappings, transforms, list refs) to a new repository.

With --create-repo, also creates the GitHub repository using the scaffold flow before cloning the target.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTargetClone(cmd, groupID, from, to, branch, prLabels, prAssignees, prReviewers, prTeamReviewers, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&from, "from", "", "Source target repository (org/repo) (required)")
	cmd.Flags().StringVar(&to, "to", "", "New target repository (org/repo) (required)")
	cmd.Flags().StringVar(&branch, "branch", "", "Override branch for new target (defaults to source)")
	cmd.Flags().StringVar(&prLabels, "pr-labels", "", "Override comma-separated PR labels")
	cmd.Flags().StringVar(&prAssignees, "pr-assignees", "", "Override comma-separated PR assignees")
	cmd.Flags().StringVar(&prReviewers, "pr-reviewers", "", "Override comma-separated PR reviewers")
	cmd.Flags().StringVar(&prTeamReviewers, "pr-team-reviewers", "", "Override comma-separated PR team reviewers")
	cmd.Flags().BoolVar(&createRepo, "create-repo", false, "Create the GitHub repository before cloning target")
	cmd.Flags().StringVar(&presetID, "preset", "mvp", "Settings preset ID (used with --create-repo)")
	cmd.Flags().StringVar(&topicsStr, "topics", "", "Comma-separated topics (used with --create-repo)")
	cmd.Flags().BoolVar(&noFiles, "no-files", false, "Skip initial file creation (used with --create-repo)")
	cmd.Flags().BoolVar(&noCloneRepo, "no-clone-repo", false, "Don't clone repository locally (used with --create-repo)")
	// Keep unused vars referenced to avoid compile error
	_ = createRepo
	_ = presetID
	_ = topicsStr
	_ = noFiles
	_ = noCloneRepo
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func runTargetClone(cmd *cobra.Command, groupExternalID, fromRepo, toRepo, branch, prLabels, prAssignees, prReviewers, prTeamReviewers string, jsonOutput bool) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// If --create-repo is set, run scaffold flow first
	if cmd.Flags().Changed("create-repo") {
		presetID, _ := cmd.Flags().GetString("preset")
		topicsStr, _ := cmd.Flags().GetString("topics")
		noFiles, _ := cmd.Flags().GetBool("no-files")
		noCloneRepo, _ := cmd.Flags().GetBool("no-clone-repo")

		var topicList []string
		if topicsStr != "" {
			topicList = strings.Split(topicsStr, ",")
			for i := range topicList {
				topicList[i] = strings.TrimSpace(topicList[i])
			}
		}

		preset, presetErr := resolvePreset(ctx, presetID)
		if presetErr != nil {
			return printErrorResponse("target", "cloned", presetErr.Error(),
				"run 'go-broadcast presets list' to see available presets", jsonOutput)
		}

		repoParts := strings.Split(toRepo, "/")
		if len(repoParts) != 2 {
			return printErrorResponse("target", "cloned",
				fmt.Sprintf("invalid repo format %q", toRepo), "", jsonOutput)
		}

		if !IsDryRun() {
			logger := logrus.StandardLogger()
			ghClient, ghErr := gh.NewClient(ctx, logger, &logging.LogConfig{})
			if ghErr != nil {
				return printErrorResponse("target", "cloned",
					fmt.Sprintf("GitHub client: %v", ghErr), "", jsonOutput)
			}

			_, scaffoldErr := RunScaffold(ctx, ghClient, ScaffoldOptions{
				Name:    repoParts[1],
				Owner:   repoParts[0],
				Preset:  preset,
				Topics:  topicList,
				NoClone: noCloneRepo,
				NoFiles: noFiles,
			})
			if scaffoldErr != nil {
				return printErrorResponse("target", "cloned",
					fmt.Sprintf("scaffold failed: %v", scaffoldErr),
					fmt.Sprintf("partial cleanup: gh repo delete %s --yes", toRepo),
					jsonOutput)
			}

			output.Success(fmt.Sprintf("Repository %s created, now cloning target...", toRepo))
		}
	}

	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("target", "cloned", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	gormDB := database.DB()

	// Resolve group
	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("target", "cloned", err.Error(), groupHintList(), jsonOutput)
	}

	// Check destination doesn't already exist
	targetRepo := db.NewTargetRepository(gormDB)
	existing, err := targetRepo.GetByRepoName(ctx, group.ID, toRepo)
	if err == nil && existing != nil {
		return printErrorResponse("target", "cloned",
			fmt.Sprintf("target %q already exists in group %q", toRepo, groupExternalID),
			fmt.Sprintf("run 'go-broadcast db target get --group %s --repo %s --json' to inspect it", groupExternalID, toRepo),
			jsonOutput)
	}

	// Find source target with all associations
	targets, err := targetRepo.ListWithAssociations(ctx, group.ID)
	if err != nil {
		return printErrorResponse("target", "cloned", err.Error(), "", jsonOutput)
	}

	var source *db.Target
	for _, t := range targets {
		name := resolveRepoName(ctx, gormDB, t.RepoID)
		if strings.EqualFold(name, fromRepo) {
			source = t
			break
		}
	}
	if source == nil {
		return printErrorResponse("target", "cloned",
			fmt.Sprintf("source target %q not found in group %q", fromRepo, groupExternalID),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json' to see available targets", groupExternalID),
			jsonOutput)
	}

	previewBranch := source.Branch
	if cmd.Flags().Changed("branch") {
		previewBranch = branch
	}

	if IsDryRun() {
		return printDryRunResponse(CLIResponse{
			Action: "cloned",
			Type:   "target",
			Data: targetListResult{
				Repo:     toRepo,
				Branch:   previewBranch,
				Position: len(targets),
				GroupID:  groupExternalID,
			},
			Hint: fmt.Sprintf("cloned from %s", fromRepo),
		}, fmt.Sprintf("clone target %q to %q in group %q", fromRepo, toRepo, groupExternalID), jsonOutput)
	}

	// Execute deep clone in a transaction
	var newTarget *db.Target
	err = gormDB.Transaction(func(tx *gorm.DB) error {
		newTarget, err = cloneTargetInTx(ctx, cmd, tx, source, group.ID, toRepo, len(targets),
			branch, prLabels, prAssignees, prReviewers, prTeamReviewers)
		return err
	})
	if err != nil {
		return printErrorResponse("target", "cloned", err.Error(), "", jsonOutput)
	}

	result := targetListResult{
		Repo:     toRepo,
		Branch:   newTarget.Branch,
		Position: newTarget.Position,
		GroupID:  groupExternalID,
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "cloned",
		Type:    "target",
		Data:    result,
		Hint:    fmt.Sprintf("cloned from %s", fromRepo),
	}, jsonOutput)
}

func cloneTargetInTx(
	ctx context.Context,
	cmd *cobra.Command,
	tx *gorm.DB,
	source *db.Target,
	groupID uint,
	toRepo string,
	position int,
	branch, prLabels, prAssignees, prReviewers, prTeamReviewers string,
) (*db.Target, error) {
	// Resolve destination repo (auto-creates client/org/repo)
	repoRepo := db.NewRepoRepository(tx)
	destRepo, err := repoRepo.FindOrCreateFromFullName(ctx, toRepo, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve destination repo: %w", err)
	}

	// Create new Target record - copy scalar fields from source
	newTarget := &db.Target{
		GroupID:         groupID,
		RepoID:          destRepo.ID,
		Branch:          source.Branch,
		BlobSizeLimit:   source.BlobSizeLimit,
		SecurityEmail:   source.SecurityEmail,
		SupportEmail:    source.SupportEmail,
		PRLabels:        copyJSONStringSlice(source.PRLabels),
		PRAssignees:     copyJSONStringSlice(source.PRAssignees),
		PRReviewers:     copyJSONStringSlice(source.PRReviewers),
		PRTeamReviewers: copyJSONStringSlice(source.PRTeamReviewers),
		Position:        position,
	}

	// Apply overrides (only if flag was explicitly provided)
	if cmd.Flags().Changed("branch") {
		newTarget.Branch = branch
	}
	if cmd.Flags().Changed("pr-labels") {
		newTarget.PRLabels = splitCSV(prLabels)
	}
	if cmd.Flags().Changed("pr-assignees") {
		newTarget.PRAssignees = splitCSV(prAssignees)
	}
	if cmd.Flags().Changed("pr-reviewers") {
		newTarget.PRReviewers = splitCSV(prReviewers)
	}
	if cmd.Flags().Changed("pr-team-reviewers") {
		newTarget.PRTeamReviewers = splitCSV(prTeamReviewers)
	}

	if err = tx.WithContext(ctx).Create(newTarget).Error; err != nil {
		return nil, fmt.Errorf("failed to create cloned target: %w", err)
	}

	// Clone FileMappings (polymorphic: OwnerType="target")
	for _, fm := range source.FileMappings {
		clone := db.FileMapping{
			OwnerType:  "target",
			OwnerID:    newTarget.ID,
			Src:        fm.Src,
			Dest:       fm.Dest,
			DeleteFlag: fm.DeleteFlag,
			Position:   fm.Position,
		}
		if err = tx.WithContext(ctx).Create(&clone).Error; err != nil {
			return nil, fmt.Errorf("failed to clone file mapping %q: %w", fm.Dest, err)
		}
	}

	// Clone DirectoryMappings with nested Transforms
	for _, dm := range source.DirectoryMappings {
		clone := db.DirectoryMapping{
			OwnerType:         "target",
			OwnerID:           newTarget.ID,
			Src:               dm.Src,
			Dest:              dm.Dest,
			Exclude:           copyJSONStringSlice(dm.Exclude),
			IncludeOnly:       copyJSONStringSlice(dm.IncludeOnly),
			PreserveStructure: copyBoolPtr(dm.PreserveStructure),
			IncludeHidden:     copyBoolPtr(dm.IncludeHidden),
			DeleteFlag:        dm.DeleteFlag,
			ModuleConfig:      copyModuleConfig(dm.ModuleConfig),
			Position:          dm.Position,
		}
		if err = tx.WithContext(ctx).Create(&clone).Error; err != nil {
			return nil, fmt.Errorf("failed to clone directory mapping %q: %w", dm.Dest, err)
		}

		// Clone directory-level transform (OwnerType="directory_mapping")
		if dm.Transform.ID != 0 {
			tmClone := db.Transform{
				OwnerType: "directory_mapping",
				OwnerID:   clone.ID,
				RepoName:  dm.Transform.RepoName,
				Variables: copyJSONStringMap(dm.Transform.Variables),
			}
			if err = tx.WithContext(ctx).Create(&tmClone).Error; err != nil {
				return nil, fmt.Errorf("failed to clone transform for directory %q: %w", dm.Dest, err)
			}
		}
	}

	// Clone target-level Transform (OwnerType="target")
	if source.Transform.ID != 0 {
		tmClone := db.Transform{
			OwnerType: "target",
			OwnerID:   newTarget.ID,
			RepoName:  source.Transform.RepoName,
			Variables: copyJSONStringMap(source.Transform.Variables),
		}
		if err = tx.WithContext(ctx).Create(&tmClone).Error; err != nil {
			return nil, fmt.Errorf("failed to clone target transform: %w", err)
		}
	}

	// Clone TargetFileListRefs (point to same FileList, just new join records)
	for _, ref := range source.FileListRefs {
		clone := db.TargetFileListRef{
			TargetID:   newTarget.ID,
			FileListID: ref.FileListID,
			Position:   ref.Position,
		}
		if err = tx.WithContext(ctx).Create(&clone).Error; err != nil {
			return nil, fmt.Errorf("failed to clone file list ref: %w", err)
		}
	}

	// Clone TargetDirectoryListRefs (point to same DirectoryList)
	for _, ref := range source.DirectoryListRefs {
		clone := db.TargetDirectoryListRef{
			TargetID:        newTarget.ID,
			DirectoryListID: ref.DirectoryListID,
			Position:        ref.Position,
		}
		if err = tx.WithContext(ctx).Create(&clone).Error; err != nil {
			return nil, fmt.Errorf("failed to clone directory list ref: %w", err)
		}
	}

	return newTarget, nil
}

// copyJSONStringSlice creates a deep copy of a JSONStringSlice
func copyJSONStringSlice(s db.JSONStringSlice) db.JSONStringSlice {
	if s == nil {
		return nil
	}
	c := make(db.JSONStringSlice, len(s))
	copy(c, s)
	return c
}

// copyJSONStringMap creates a deep copy of a JSONStringMap
func copyJSONStringMap(m db.JSONStringMap) db.JSONStringMap {
	if m == nil {
		return nil
	}
	c := make(db.JSONStringMap, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

// copyBoolPtr creates a copy of a *bool
func copyBoolPtr(b *bool) *bool {
	if b == nil {
		return nil
	}
	v := *b
	return &v
}

// copyModuleConfig creates a deep copy of a *JSONModuleConfig
func copyModuleConfig(mc *db.JSONModuleConfig) *db.JSONModuleConfig {
	if mc == nil {
		return nil
	}
	c := *mc
	return &c
}
