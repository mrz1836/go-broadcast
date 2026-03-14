package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// newDBDirCmd creates the "db dir" command for inline directory mappings on targets
func newDBDirCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dir",
		Short: "Manage inline directory mappings on targets",
		Long:  `Add, remove, and list inline directory mappings directly on targets.`,
	}

	cmd.AddCommand(
		newDBDirAddCmd(),
		newDBDirRemoveCmd(),
		newDBDirListMappingsCmd(),
	)

	return cmd
}

func newDBDirAddCmd() *cobra.Command {
	var (
		jsonOutput        bool
		groupID           string
		repo              string
		src               string
		dest              string
		exclude           string
		includeOnly       string
		preserveStructure bool
		deleteFlag        bool
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a directory mapping to a target",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDirAdd(groupID, repo, src, dest, exclude, includeOnly, preserveStructure, deleteFlag, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().StringVar(&src, "src", "", "Source directory path")
	cmd.Flags().StringVar(&dest, "dest", "", "Destination directory path (required)")
	cmd.Flags().StringVar(&exclude, "exclude", "", "Comma-separated exclude patterns")
	cmd.Flags().StringVar(&includeOnly, "include-only", "", "Comma-separated include-only patterns")
	cmd.Flags().BoolVar(&preserveStructure, "preserve-structure", true, "Preserve directory structure")
	cmd.Flags().BoolVar(&deleteFlag, "delete", false, "Mark directory for deletion")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("dest")
	return cmd
}

func runDirAdd(groupExternalID, repoFullName, src, dest, exclude, includeOnly string, preserveStructure, deleteFlag, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("directory_mapping", "created", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("directory_mapping", "created", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("directory_mapping", "created", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json'", groupExternalID),
			jsonOutput)
	}

	dmRepo := db.NewDirectoryMappingRepository(gormDB)

	if _, err = dmRepo.FindByDest(ctx, "target", target.ID, dest); err == nil {
		return printErrorResponse("directory_mapping", "created",
			fmt.Sprintf("directory mapping with dest %q already exists on target %q", dest, repoFullName),
			"use 'go-broadcast db dir remove' first to replace it", jsonOutput)
	}

	existing, err := dmRepo.ListByOwner(ctx, "target", target.ID)
	if err != nil {
		return printErrorResponse("directory_mapping", "created", err.Error(), "", jsonOutput)
	}

	mapping := &db.DirectoryMapping{
		OwnerType:         "target",
		OwnerID:           target.ID,
		Src:               src,
		Dest:              dest,
		Exclude:           splitCSV(exclude),
		IncludeOnly:       splitCSV(includeOnly),
		PreserveStructure: &preserveStructure,
		DeleteFlag:        deleteFlag,
		Position:          len(existing),
	}

	if err = dmRepo.Create(ctx, mapping); err != nil {
		return printErrorResponse("directory_mapping", "created", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "created",
		Type:    "directory_mapping",
		Data: dirMappingResult{
			Src:               src,
			Dest:              dest,
			Exclude:           []string(mapping.Exclude),
			IncludeOnly:       []string(mapping.IncludeOnly),
			PreserveStructure: &preserveStructure,
			Delete:            deleteFlag,
		},
	}, jsonOutput)
}

func newDBDirRemoveCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
		dest       string
	)
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a directory mapping from a target",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDirRemove(groupID, repo, dest, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	cmd.Flags().StringVar(&dest, "dest", "", "Destination directory path to remove (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("dest")
	return cmd
}

func runDirRemove(groupExternalID, repoFullName, dest string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("directory_mapping", "deleted", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("directory_mapping", "deleted", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("directory_mapping", "deleted", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json'", groupExternalID),
			jsonOutput)
	}

	dmRepo := db.NewDirectoryMappingRepository(gormDB)
	mapping, err := dmRepo.FindByDest(ctx, "target", target.ID, dest)
	if err != nil {
		return printErrorResponse("directory_mapping", "deleted",
			fmt.Sprintf("directory mapping with dest %q not found on target %q", dest, repoFullName),
			fmt.Sprintf("run 'go-broadcast db dir list --group %s --repo %s --json'", groupExternalID, repoFullName),
			jsonOutput)
	}

	if err = dmRepo.Delete(ctx, mapping.ID, true); err != nil {
		return printErrorResponse("directory_mapping", "deleted", err.Error(), "", jsonOutput)
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "deleted",
		Type:    "directory_mapping",
		Data: map[string]string{
			"repo": repoFullName,
			"dest": dest,
		},
	}, jsonOutput)
}

func newDBDirListMappingsCmd() *cobra.Command {
	var (
		jsonOutput bool
		groupID    string
		repo       string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List directory mappings on a target",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDirListMappings(groupID, repo, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&groupID, "group", "", "Group external ID (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "Target repository (org/repo) (required)")
	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func runDirListMappings(groupExternalID, repoFullName string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("directory_mapping", "listed", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	group, err := resolveGroup(ctx, gormDB, groupExternalID)
	if err != nil {
		return printErrorResponse("directory_mapping", "listed", err.Error(), groupHintList(), jsonOutput)
	}

	target, err := resolveTarget(ctx, gormDB, group.ID, repoFullName)
	if err != nil {
		return printErrorResponse("directory_mapping", "listed", err.Error(),
			fmt.Sprintf("run 'go-broadcast db target list --group %s --json'", groupExternalID),
			jsonOutput)
	}

	dmRepo := db.NewDirectoryMappingRepository(gormDB)
	mappings, err := dmRepo.ListByOwner(ctx, "target", target.ID)
	if err != nil {
		return printErrorResponse("directory_mapping", "listed", err.Error(), "", jsonOutput)
	}

	results := make([]dirMappingResult, 0, len(mappings))
	for _, m := range mappings {
		results = append(results, dirMappingResult{
			Src:               m.Src,
			Dest:              m.Dest,
			Exclude:           []string(m.Exclude),
			IncludeOnly:       []string(m.IncludeOnly),
			PreserveStructure: m.PreserveStructure,
			Delete:            m.DeleteFlag,
		})
	}

	resp := CLIResponse{
		Success: true,
		Action:  "listed",
		Type:    "directory_mapping",
		Data:    results,
		Count:   len(results),
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Found %d directory mapping(s) on target %q:", len(results), repoFullName))
	for _, r := range results {
		if r.Src != "" {
			output.Info(fmt.Sprintf("  %s -> %s", r.Src, r.Dest))
		} else {
			output.Info(fmt.Sprintf("  %s", r.Dest))
		}
	}
	return nil
}
