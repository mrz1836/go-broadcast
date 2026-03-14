package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// newDBRefCmd creates the "db ref" command for managing file-list and dir-list references on targets
func newDBRefCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ref",
		Short: "Manage file-list and dir-list references on targets",
		Long:  `Attach and detach reusable file lists and directory lists to/from targets.`,
	}

	cmd.AddCommand(
		newDBRefAddFileListCmd(),
		newDBRefRemoveFileListCmd(),
		newDBRefAddDirListCmd(),
		newDBRefRemoveDirListCmd(),
	)

	return cmd
}

func newDBRefAddFileListCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
		fileListID string
	)
	cmd := &cobra.Command{
		Use:   "add-file-list",
		Short: "Attach a file list to a target",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRefAddFileList(groupID, repo, fileListID, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().StringVar(&fileListID, "file-list", "", "File list external ID (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("file-list")
	return cmd
}

func runRefAddFileList(groupExternalID, repoFullName, fileListExternalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("ref", "attached", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("ref", "attached", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("ref", "attached", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json'", groupExternalID),
			jsonOutput)
	}

	fl, err := resolveFileList(ctx, gormDB, fileListExternalID)
	if err != nil {
		return printErrorResponse("ref", "attached", err.Error(),
			"run 'go-broadcast db file-list list --json' to see available file lists", jsonOutput)
	}

	// Check if ref already exists
	var existingRef db.TargetFileListRef
	if err = gormDB.WithContext(ctx).
		Where("target_id = ? AND file_list_id = ?", target.ID, fl.ID).
		First(&existingRef).Error; err == nil {
		// Already exists - idempotent success
		return printResponse(CLIResponse{
			Success: true,
			Action:  "already_attached",
			Type:    "file_list_ref",
			Data: map[string]string{
				"repo":      repoFullName,
				"file_list": fileListExternalID,
			},
			Hint: "file list is already attached to this target",
		}, jsonOutput)
	}

	// Find max position
	var maxPos int
	gormDB.WithContext(ctx).Model(&db.TargetFileListRef{}).
		Where("target_id = ?", target.ID).
		Select("COALESCE(MAX(position), -1)").Scan(&maxPos)

	targetRepo := db.NewTargetRepository(gormDB)
	if err = targetRepo.AddFileListRef(ctx, target.ID, fl.ID, maxPos+1); err != nil {
		return printErrorResponse("ref", "attached", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "attached",
		Type:    "file_list_ref",
		Data: map[string]string{
			"repo":      repoFullName,
			"file_list": fileListExternalID,
		},
	}, jsonOutput)
}

func newDBRefRemoveFileListCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
		fileListID string
	)
	cmd := &cobra.Command{
		Use:   "remove-file-list",
		Short: "Detach a file list from a target",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRefRemoveFileList(groupID, repo, fileListID, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().StringVar(&fileListID, "file-list", "", "File list external ID (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("file-list")
	return cmd
}

func runRefRemoveFileList(groupExternalID, repoFullName, fileListExternalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("ref", "detached", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("ref", "detached", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("ref", "detached", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json'", groupExternalID),
			jsonOutput)
	}

	fl, err := resolveFileList(ctx, gormDB, fileListExternalID)
	if err != nil {
		return printErrorResponse("ref", "detached", err.Error(),
			"run 'go-broadcast db file-list list --json' to see available file lists", jsonOutput)
	}

	targetRepo := db.NewTargetRepository(gormDB)
	if err = targetRepo.RemoveFileListRef(ctx, target.ID, fl.ID); err != nil {
		return printErrorResponse("ref", "detached", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "detached",
		Type:    "file_list_ref",
		Data: map[string]string{
			"repo":      repoFullName,
			"file_list": fileListExternalID,
		},
	}, jsonOutput)
}

func newDBRefAddDirListCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
		dirListID  string
	)
	cmd := &cobra.Command{
		Use:   "add-dir-list",
		Short: "Attach a directory list to a target",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRefAddDirList(groupID, repo, dirListID, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().StringVar(&dirListID, "dir-list", "", "Directory list external ID (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("dir-list")
	return cmd
}

func runRefAddDirList(groupExternalID, repoFullName, dirListExternalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("ref", "attached", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("ref", "attached", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("ref", "attached", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json'", groupExternalID),
			jsonOutput)
	}

	dl, err := resolveDirectoryList(ctx, gormDB, dirListExternalID)
	if err != nil {
		return printErrorResponse("ref", "attached", err.Error(),
			"run 'go-broadcast db dir-list list --json' to see available directory lists", jsonOutput)
	}

	// Check if ref already exists
	var existingRef db.TargetDirectoryListRef
	if err = gormDB.WithContext(ctx).
		Where("target_id = ? AND directory_list_id = ?", target.ID, dl.ID).
		First(&existingRef).Error; err == nil {
		return printResponse(CLIResponse{
			Success: true,
			Action:  "already_attached",
			Type:    "directory_list_ref",
			Data: map[string]string{
				"repo":     repoFullName,
				"dir_list": dirListExternalID,
			},
			Hint: "directory list is already attached to this target",
		}, jsonOutput)
	}

	var maxPos int
	gormDB.WithContext(ctx).Model(&db.TargetDirectoryListRef{}).
		Where("target_id = ?", target.ID).
		Select("COALESCE(MAX(position), -1)").Scan(&maxPos)

	targetRepo := db.NewTargetRepository(gormDB)
	if err = targetRepo.AddDirectoryListRef(ctx, target.ID, dl.ID, maxPos+1); err != nil {
		return printErrorResponse("ref", "attached", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "attached",
		Type:    "directory_list_ref",
		Data: map[string]string{
			"repo":     repoFullName,
			"dir_list": dirListExternalID,
		},
	}, jsonOutput)
}

func newDBRefRemoveDirListCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
		dirListID  string
	)
	cmd := &cobra.Command{
		Use:   "remove-dir-list",
		Short: "Detach a directory list from a target",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRefRemoveDirList(groupID, repo, dirListID, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().StringVar(&dirListID, "dir-list", "", "Directory list external ID (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("dir-list")
	return cmd
}

func runRefRemoveDirList(groupExternalID, repoFullName, dirListExternalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("ref", "detached", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("ref", "detached", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("ref", "detached", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json'", groupExternalID),
			jsonOutput)
	}

	dl, err := resolveDirectoryList(ctx, gormDB, dirListExternalID)
	if err != nil {
		return printErrorResponse("ref", "detached", err.Error(),
			"run 'go-broadcast db dir-list list --json' to see available directory lists", jsonOutput)
	}

	targetRepo := db.NewTargetRepository(gormDB)
	if err = targetRepo.RemoveDirectoryListRef(ctx, target.ID, dl.ID); err != nil {
		return printErrorResponse("ref", "detached", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "detached",
		Type:    "directory_list_ref",
		Data: map[string]string{
			"repo":     repoFullName,
			"dir_list": dirListExternalID,
		},
	}, jsonOutput)
}
