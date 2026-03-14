package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// dirListResult is the JSON-serializable result for directory list operations
type dirListResult struct {
	ExternalID  string             `json:"external_id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	DirCount    int                `json:"dir_count"`
	Directories []dirMappingResult `json:"directories,omitempty"`
}

// newDBDirListCmd creates the "db dir-list" command with subcommands
func newDBDirListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dir-list",
		Short: "Manage reusable directory lists",
		Long:  `Create, read, and delete reusable directory lists and their directory mappings.`,
	}

	cmd.AddCommand(
		newDBDirListListCmd(),
		newDBDirListGetCmd(),
		newDBDirListCreateCmd(),
		newDBDirListDeleteCmd(),
		newDBDirListAddDirCmd(),
		newDBDirListRemoveDirCmd(),
	)

	return cmd
}

func newDBDirListListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all directory lists",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDirListList(jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runDirListList(jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("directory_list", "listed", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	cfg, err := getDefaultConfig(ctx, gormDB)
	if err != nil {
		return printErrorResponse("directory_list", "listed", err.Error(),
			"run 'go-broadcast db import <file>' to import a configuration", jsonOutput)
	}

	dlRepo := db.NewDirectoryListRepository(gormDB)
	dirLists, err := dlRepo.ListWithDirectories(ctx, cfg.ID)
	if err != nil {
		return printErrorResponse("directory_list", "listed", err.Error(), "", jsonOutput)
	}

	results := make([]dirListResult, 0, len(dirLists))
	for _, dl := range dirLists {
		results = append(results, dirListResult{
			ExternalID:  dl.ExternalID,
			Name:        dl.Name,
			Description: dl.Description,
			DirCount:    len(dl.Directories),
		})
	}

	resp := CLIResponse{
		Success: true,
		Action:  "listed",
		Type:    "directory_list",
		Data:    results,
		Count:   len(results),
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Found %d directory list(s):", len(results)))
	for _, r := range results {
		output.Info(fmt.Sprintf("  %s (%s) - %d directories", r.ExternalID, r.Name, r.DirCount))
	}
	return nil
}

func newDBDirListGetCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Show directory list with all mappings",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDirListGet(args[0], jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runDirListGet(externalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("directory_list", "get", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	dl, err := resolveDirectoryList(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("directory_list", "get", err.Error(),
			"run 'go-broadcast db dir-list list --json' to see available directory lists", jsonOutput)
	}

	dmRepo := db.NewDirectoryMappingRepository(gormDB)
	mappings, err := dmRepo.ListByOwner(ctx, "directory_list", dl.ID)
	if err != nil {
		return printErrorResponse("directory_list", "get", err.Error(), "", jsonOutput)
	}

	dirs := make([]dirMappingResult, 0, len(mappings))
	for _, m := range mappings {
		dirs = append(dirs, dirMappingResult{
			Src:               m.Src,
			Dest:              m.Dest,
			Exclude:           []string(m.Exclude),
			IncludeOnly:       []string(m.IncludeOnly),
			PreserveStructure: m.PreserveStructure,
			Delete:            m.DeleteFlag,
		})
	}

	result := dirListResult{
		ExternalID:  dl.ExternalID,
		Name:        dl.Name,
		Description: dl.Description,
		DirCount:    len(dirs),
		Directories: dirs,
	}

	resp := CLIResponse{
		Success: true,
		Action:  "get",
		Type:    "directory_list",
		Data:    result,
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Directory List: %s (%s)", result.ExternalID, result.Name))
	if result.Description != "" {
		output.Info(fmt.Sprintf("  Description: %s", result.Description))
	}
	output.Info(fmt.Sprintf("  Directories: %d", result.DirCount))
	for _, d := range result.Directories {
		if d.Src != "" {
			output.Info(fmt.Sprintf("    %s -> %s", d.Src, d.Dest))
		} else {
			output.Info(fmt.Sprintf("    %s", d.Dest))
		}
	}
	return nil
}

func newDBDirListCreateCmd() *cobra.Command {
	var (
		jsonOutput  bool
		id          string
		name        string
		description string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an empty directory list",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDirListCreate(id, name, description, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&id, "id", "", "Directory list external ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Directory list name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Directory list description")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func runDirListCreate(id, name, description string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("directory_list", "created", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	cfg, err := getDefaultConfig(ctx, gormDB)
	if err != nil {
		return printErrorResponse("directory_list", "created", err.Error(),
			"run 'go-broadcast db import <file>' to import a configuration first", jsonOutput)
	}

	dlRepo := db.NewDirectoryListRepository(gormDB)
	if _, err = dlRepo.GetByExternalID(ctx, id); err == nil {
		return printErrorResponse("directory_list", "created",
			fmt.Sprintf("directory list %q already exists", id),
			"use a different --id value", jsonOutput)
	}

	existingLists, err := dlRepo.List(ctx, cfg.ID)
	if err != nil {
		return printErrorResponse("directory_list", "created", err.Error(), "", jsonOutput)
	}

	dl := &db.DirectoryList{
		ConfigID:    cfg.ID,
		ExternalID:  id,
		Name:        name,
		Description: description,
		Position:    len(existingLists),
	}

	if err = dlRepo.Create(ctx, dl); err != nil {
		return printErrorResponse("directory_list", "created", err.Error(), "", jsonOutput)
	}

	result := dirListResult{
		ExternalID:  id,
		Name:        name,
		Description: description,
		DirCount:    0,
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "created",
		Type:    "directory_list",
		Data:    result,
	}, jsonOutput)
}

func newDBDirListDeleteCmd() *cobra.Command {
	var (
		jsonOutput bool
		hard       bool
	)
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a directory list",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDirListDelete(args[0], hard, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&hard, "hard", false, "Hard delete (permanent)")
	return cmd
}

func runDirListDelete(externalID string, hard, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("directory_list", "deleted", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	dl, err := resolveDirectoryList(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("directory_list", "deleted", err.Error(),
			"run 'go-broadcast db dir-list list --json' to see available directory lists", jsonOutput)
	}

	dlRepo := db.NewDirectoryListRepository(gormDB)
	if err = dlRepo.Delete(ctx, dl.ID, hard); err != nil {
		return printErrorResponse("directory_list", "deleted", err.Error(), "", jsonOutput)
	}

	deleteType := "soft-deleted"
	if hard {
		deleteType = "hard-deleted"
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  deleteType,
		Type:    "directory_list",
		Data:    map[string]string{"external_id": externalID},
	}, jsonOutput)
}

func newDBDirListAddDirCmd() *cobra.Command {
	var (
		jsonOutput        bool
		src               string
		dest              string
		exclude           string
		includeOnly       string
		preserveStructure bool
		deleteFlag        bool
	)
	cmd := &cobra.Command{
		Use:   "add-dir <id>",
		Short: "Add a directory mapping to a directory list",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDirListAddDir(args[0], src, dest, exclude, includeOnly, preserveStructure, deleteFlag, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&src, "src", "", "Source directory path")
	cmd.Flags().StringVar(&dest, "dest", "", "Destination directory path (required)")
	cmd.Flags().StringVar(&exclude, "exclude", "", "Comma-separated exclude patterns")
	cmd.Flags().StringVar(&includeOnly, "include-only", "", "Comma-separated include-only patterns")
	cmd.Flags().BoolVar(&preserveStructure, "preserve-structure", true, "Preserve directory structure")
	cmd.Flags().BoolVar(&deleteFlag, "delete", false, "Mark directory for deletion")
	_ = cmd.MarkFlagRequired("dest")
	return cmd
}

func runDirListAddDir(externalID, src, dest, exclude, includeOnly string, preserveStructure, deleteFlag, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("directory_mapping", "created", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	dl, err := resolveDirectoryList(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("directory_mapping", "created", err.Error(),
			"run 'go-broadcast db dir-list list --json' to see available directory lists", jsonOutput)
	}

	dmRepo := db.NewDirectoryMappingRepository(gormDB)

	if _, err = dmRepo.FindByDest(ctx, "directory_list", dl.ID, dest); err == nil {
		return printErrorResponse("directory_mapping", "created",
			fmt.Sprintf("directory mapping with dest %q already exists in directory list %q", dest, externalID),
			"use 'remove-dir' first to replace it", jsonOutput)
	}

	existing, err := dmRepo.ListByOwner(ctx, "directory_list", dl.ID)
	if err != nil {
		return printErrorResponse("directory_mapping", "created", err.Error(), "", jsonOutput)
	}

	mapping := &db.DirectoryMapping{
		OwnerType:         "directory_list",
		OwnerID:           dl.ID,
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

	result := dirMappingResult{
		Src:               src,
		Dest:              dest,
		Exclude:           []string(mapping.Exclude),
		IncludeOnly:       []string(mapping.IncludeOnly),
		PreserveStructure: &preserveStructure,
		Delete:            deleteFlag,
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "created",
		Type:    "directory_mapping",
		Data:    result,
	}, jsonOutput)
}

func newDBDirListRemoveDirCmd() *cobra.Command {
	var (
		jsonOutput bool
		dest       string
	)
	cmd := &cobra.Command{
		Use:   "remove-dir <id>",
		Short: "Remove a directory mapping from a directory list",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDirListRemoveDir(args[0], dest, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&dest, "dest", "", "Destination directory path to remove (required)")
	_ = cmd.MarkFlagRequired("dest")
	return cmd
}

func runDirListRemoveDir(externalID, dest string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("directory_mapping", "deleted", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	dl, err := resolveDirectoryList(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("directory_mapping", "deleted", err.Error(),
			"run 'go-broadcast db dir-list list --json' to see available directory lists", jsonOutput)
	}

	dmRepo := db.NewDirectoryMappingRepository(gormDB)
	mapping, err := dmRepo.FindByDest(ctx, "directory_list", dl.ID, dest)
	if err != nil {
		return printErrorResponse("directory_mapping", "deleted",
			fmt.Sprintf("directory mapping with dest %q not found in directory list %q", dest, externalID),
			fmt.Sprintf("run 'go-broadcast db dir-list get %s --json' to see available mappings", externalID),
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
			"directory_list": externalID,
			"dest":           dest,
		},
	}, jsonOutput)
}
