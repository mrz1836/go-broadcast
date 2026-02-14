package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// newDBFileCmd creates the "db file" command for inline file mappings on targets
func newDBFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "Manage inline file mappings on targets",
		Long:  `Add, remove, and list inline file mappings directly on targets.`,
	}

	cmd.AddCommand(
		newDBFileAddCmd(),
		newDBFileRemoveCmd(),
		newDBFileListMappingsCmd(),
	)

	return cmd
}

func newDBFileAddCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
		src        string
		dest       string
		deleteFlag bool
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a file mapping to a target",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runFileAdd(groupID, repo, src, dest, deleteFlag, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().StringVar(&src, "src", "", "Source file path")
	cmd.Flags().StringVar(&dest, "dest", "", "Destination file path (required)")
	cmd.Flags().BoolVar(&deleteFlag, "delete", false, "Mark file for deletion")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("dest")
	return cmd
}

func runFileAdd(groupExternalID, repoFullName, src, dest string, deleteFlag, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("file_mapping", "created", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("file_mapping", "created", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("file_mapping", "created", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json' to see available targets", groupExternalID),
			jsonOutput)
	}

	fmRepo := db.NewFileMappingRepository(gormDB)

	// Check for duplicate
	if _, err = fmRepo.FindByDest(ctx, "target", target.ID, dest); err == nil {
		return printErrorResponse("file_mapping", "created",
			fmt.Sprintf("file mapping with dest %q already exists on target %q", dest, repoFullName),
			"use 'go-broadcast db file remove' first to replace it", jsonOutput)
	}

	existing, err := fmRepo.ListByOwner(ctx, "target", target.ID)
	if err != nil {
		return printErrorResponse("file_mapping", "created", err.Error(), "", jsonOutput)
	}

	mapping := &db.FileMapping{
		OwnerType:  "target",
		OwnerID:    target.ID,
		Src:        src,
		Dest:       dest,
		DeleteFlag: deleteFlag,
		Position:   len(existing),
	}

	if err = fmRepo.Create(ctx, mapping); err != nil {
		return printErrorResponse("file_mapping", "created", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "created",
		Type:    "file_mapping",
		Data: fileMappingResult{
			Src:    src,
			Dest:   dest,
			Delete: deleteFlag,
		},
	}, jsonOutput)
}

func newDBFileRemoveCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
		dest       string
	)
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a file mapping from a target",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runFileRemove(groupID, repo, dest, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().StringVar(&dest, "dest", "", "Destination file path to remove (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("dest")
	return cmd
}

func runFileRemove(groupExternalID, repoFullName, dest string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("file_mapping", "deleted", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("file_mapping", "deleted", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("file_mapping", "deleted", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json'", groupExternalID),
			jsonOutput)
	}

	fmRepo := db.NewFileMappingRepository(gormDB)
	mapping, err := fmRepo.FindByDest(ctx, "target", target.ID, dest)
	if err != nil {
		return printErrorResponse("file_mapping", "deleted",
			fmt.Sprintf("file mapping with dest %q not found on target %q", dest, repoFullName),
			fmt.Sprintf("run 'go-broadcast db file list --group %s --repo %s --json'", groupExternalID, repoFullName),
			jsonOutput)
	}

	if err = fmRepo.Delete(ctx, mapping.ID, true); err != nil {
		return printErrorResponse("file_mapping", "deleted", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "deleted",
		Type:    "file_mapping",
		Data: map[string]string{
			"repo": repoFullName,
			"dest": dest,
		},
	}, jsonOutput)
}

func newDBFileListMappingsCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List file mappings on a target",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runFileListMappings(groupID, repo, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func runFileListMappings(groupExternalID, repoFullName string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("file_mapping", "listed", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("file_mapping", "listed", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("file_mapping", "listed", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json'", groupExternalID),
			jsonOutput)
	}

	fmRepo := db.NewFileMappingRepository(gormDB)
	mappings, err := fmRepo.ListByOwner(ctx, "target", target.ID)
	if err != nil {
		return printErrorResponse("file_mapping", "listed", err.Error(), "", jsonOutput)
	}

	results := make([]fileMappingResult, 0, len(mappings))
	for _, m := range mappings {
		results = append(results, fileMappingResult{
			Src:    m.Src,
			Dest:   m.Dest,
			Delete: m.DeleteFlag,
		})
	}

	resp := CLIResponse{
		Success: true,
		Action:  "listed",
		Type:    "file_mapping",
		Data:    results,
		Count:   len(results),
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Found %d file mapping(s) on target %q:", len(results), repoFullName))
	for _, r := range results {
		if r.Src != "" {
			output.Info(fmt.Sprintf("  %s -> %s", r.Src, r.Dest))
		} else {
			output.Info(fmt.Sprintf("  %s", r.Dest))
		}
	}
	return nil
}
