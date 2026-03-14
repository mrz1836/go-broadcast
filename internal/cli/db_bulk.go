package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// bulkResult is the JSON-serializable result for bulk operations
type bulkResult struct {
	GroupID  string `json:"group_id"`
	ListID   string `json:"list_id"`
	Affected int    `json:"affected"`
}

// newDBBulkCmd creates the "db bulk" command with subcommands
func newDBBulkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk",
		Short: "Bulk operations across all targets in a group",
		Long:  `Add or remove file lists and directory lists across all targets in a group at once.`,
	}

	cmd.AddCommand(
		newDBBulkAddFileListCmd(),
		newDBBulkRemoveFileListCmd(),
		newDBBulkAddDirListCmd(),
		newDBBulkRemoveDirListCmd(),
	)

	return cmd
}

func newDBBulkAddFileListCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		fileListID string
	)
	cmd := &cobra.Command{
		Use:   "add-file-list",
		Short: "Add a file list to all targets in a group",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runBulkAddFileList(groupID, fileListID, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&fileListID, "file-list", "", "File list external ID (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("file-list")
	return cmd
}

func runBulkAddFileList(groupExternalID, fileListExternalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("bulk", "add-file-list", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("bulk", "add-file-list", err.Error(), groupHintList(), jsonOutput)
	}

	fl, err := resolveFileList(ctx, gormDB, fileListExternalID)
	if err != nil {
		return printErrorResponse("bulk", "add-file-list", err.Error(),
			"run 'go-broadcast db file-list list --json' to see available file lists", jsonOutput)
	}

	bulkRepo := db.NewBulkRepository(gormDB)
	affected, err := bulkRepo.AddFileListToAllTargets(ctx, group.ID, fl.ID)
	if err != nil {
		return printErrorResponse("bulk", "add-file-list", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "bulk-attached",
		Type:    "file_list_ref",
		Data: bulkResult{
			GroupID:  groupExternalID,
			ListID:   fileListExternalID,
			Affected: affected,
		},
		Count: affected,
	}, jsonOutput)
}

func newDBBulkRemoveFileListCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		fileListID string
	)
	cmd := &cobra.Command{
		Use:   "remove-file-list",
		Short: "Remove a file list from all targets in a group",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runBulkRemoveFileList(groupID, fileListID, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&fileListID, "file-list", "", "File list external ID (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("file-list")
	return cmd
}

func runBulkRemoveFileList(groupExternalID, fileListExternalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("bulk", "remove-file-list", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("bulk", "remove-file-list", err.Error(), groupHintList(), jsonOutput)
	}

	fl, err := resolveFileList(ctx, gormDB, fileListExternalID)
	if err != nil {
		return printErrorResponse("bulk", "remove-file-list", err.Error(),
			"run 'go-broadcast db file-list list --json' to see available file lists", jsonOutput)
	}

	bulkRepo := db.NewBulkRepository(gormDB)
	affected, err := bulkRepo.RemoveFileListFromAllTargets(ctx, group.ID, fl.ID)
	if err != nil {
		return printErrorResponse("bulk", "remove-file-list", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "bulk-detached",
		Type:    "file_list_ref",
		Data: bulkResult{
			GroupID:  groupExternalID,
			ListID:   fileListExternalID,
			Affected: affected,
		},
		Count: affected,
	}, jsonOutput)
}

func newDBBulkAddDirListCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		dirListID  string
	)
	cmd := &cobra.Command{
		Use:   "add-dir-list",
		Short: "Add a directory list to all targets in a group",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runBulkAddDirList(groupID, dirListID, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&dirListID, "dir-list", "", "Directory list external ID (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("dir-list")
	return cmd
}

func runBulkAddDirList(groupExternalID, dirListExternalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("bulk", "add-dir-list", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("bulk", "add-dir-list", err.Error(), groupHintList(), jsonOutput)
	}

	dl, err := resolveDirectoryList(ctx, gormDB, dirListExternalID)
	if err != nil {
		return printErrorResponse("bulk", "add-dir-list", err.Error(),
			"run 'go-broadcast db dir-list list --json' to see available directory lists", jsonOutput)
	}

	bulkRepo := db.NewBulkRepository(gormDB)
	affected, err := bulkRepo.AddDirectoryListToAllTargets(ctx, group.ID, dl.ID)
	if err != nil {
		return printErrorResponse("bulk", "add-dir-list", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "bulk-attached",
		Type:    "directory_list_ref",
		Data: bulkResult{
			GroupID:  groupExternalID,
			ListID:   dirListExternalID,
			Affected: affected,
		},
		Count: affected,
	}, jsonOutput)
}

func newDBBulkRemoveDirListCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		dirListID  string
	)
	cmd := &cobra.Command{
		Use:   "remove-dir-list",
		Short: "Remove a directory list from all targets in a group",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runBulkRemoveDirList(groupID, dirListID, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&dirListID, "dir-list", "", "Directory list external ID (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("dir-list")
	return cmd
}

func runBulkRemoveDirList(groupExternalID, dirListExternalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("bulk", "remove-dir-list", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("bulk", "remove-dir-list", err.Error(), groupHintList(), jsonOutput)
	}

	dl, err := resolveDirectoryList(ctx, gormDB, dirListExternalID)
	if err != nil {
		return printErrorResponse("bulk", "remove-dir-list", err.Error(),
			"run 'go-broadcast db dir-list list --json' to see available directory lists", jsonOutput)
	}

	bulkRepo := db.NewBulkRepository(gormDB)
	affected, err := bulkRepo.RemoveDirectoryListFromAllTargets(ctx, group.ID, dl.ID)
	if err != nil {
		return printErrorResponse("bulk", "remove-dir-list", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "bulk-detached",
		Type:    "directory_list_ref",
		Data: bulkResult{
			GroupID:  groupExternalID,
			ListID:   dirListExternalID,
			Affected: affected,
		},
		Count: affected,
	}, jsonOutput)
}
