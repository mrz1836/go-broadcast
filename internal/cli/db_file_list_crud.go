package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// fileListResult is the JSON-serializable result for file list operations
type fileListResult struct {
	ExternalID  string              `json:"external_id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	FileCount   int                 `json:"file_count"`
	Files       []fileMappingResult `json:"files,omitempty"`
}

// newDBFileListCmd creates the "db file-list" command with subcommands
func newDBFileListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file-list",
		Short: "Manage reusable file lists",
		Long:  `Create, read, and delete reusable file lists and their file mappings.`,
	}

	cmd.AddCommand(
		newDBFileListListCmd(),
		newDBFileListGetCmd(),
		newDBFileListCreateCmd(),
		newDBFileListDeleteCmd(),
		newDBFileListAddFileCmd(),
		newDBFileListRemoveFileCmd(),
	)

	return cmd
}

func newDBFileListListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all file lists",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runFileListList(jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runFileListList(jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("file_list", "listed", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	cfg, err := getDefaultConfig(ctx, gormDB)
	if err != nil {
		return printErrorResponse("file_list", "listed", err.Error(),
			"run 'go-broadcast db import <file>' to import a configuration", jsonOutput)
	}

	flRepo := db.NewFileListRepository(gormDB)
	fileLists, err := flRepo.ListWithFiles(ctx, cfg.ID)
	if err != nil {
		return printErrorResponse("file_list", "listed", err.Error(), "", jsonOutput)
	}

	results := make([]fileListResult, 0, len(fileLists))
	for _, fl := range fileLists {
		results = append(results, fileListResult{
			ExternalID:  fl.ExternalID,
			Name:        fl.Name,
			Description: fl.Description,
			FileCount:   len(fl.Files),
		})
	}

	resp := CLIResponse{
		Success: true,
		Action:  "listed",
		Type:    "file_list",
		Data:    results,
		Count:   len(results),
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Found %d file list(s):", len(results)))
	for _, r := range results {
		output.Info(fmt.Sprintf("  %s (%s) - %d files", r.ExternalID, r.Name, r.FileCount))
	}
	return nil
}

func newDBFileListGetCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Show file list with all mappings",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFileListGet(args[0], jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runFileListGet(externalID string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("file_list", "get", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	fl, err := resolveFileList(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("file_list", "get", err.Error(),
			"run 'go-broadcast db file-list list --json' to see available file lists", jsonOutput)
	}

	// Load file mappings
	fmRepo := db.NewFileMappingRepository(gormDB)
	mappings, err := fmRepo.ListByOwner(ctx, "file_list", fl.ID)
	if err != nil {
		return printErrorResponse("file_list", "get", err.Error(), "", jsonOutput)
	}

	files := make([]fileMappingResult, 0, len(mappings))
	for _, m := range mappings {
		files = append(files, fileMappingResult{
			Src:    m.Src,
			Dest:   m.Dest,
			Delete: m.DeleteFlag,
		})
	}

	result := fileListResult{
		ExternalID:  fl.ExternalID,
		Name:        fl.Name,
		Description: fl.Description,
		FileCount:   len(files),
		Files:       files,
	}

	resp := CLIResponse{
		Success: true,
		Action:  "get",
		Type:    "file_list",
		Data:    result,
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("File List: %s (%s)", result.ExternalID, result.Name))
	if result.Description != "" {
		output.Info(fmt.Sprintf("  Description: %s", result.Description))
	}
	output.Info(fmt.Sprintf("  Files: %d", result.FileCount))
	for _, f := range result.Files {
		if f.Src != "" {
			output.Info(fmt.Sprintf("    %s -> %s", f.Src, f.Dest))
		} else {
			output.Info(fmt.Sprintf("    %s", f.Dest))
		}
	}
	return nil
}

func newDBFileListCreateCmd() *cobra.Command {
	var (
		jsonOutput  bool
		id          string
		name        string
		description string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an empty file list",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runFileListCreate(id, name, description, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&id, "id", "", "File list external ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "File list name (required)")
	cmd.Flags().StringVar(&description, "description", "", "File list description")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func runFileListCreate(id, name, description string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("file_list", "created", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	cfg, err := getDefaultConfig(ctx, gormDB)
	if err != nil {
		return printErrorResponse("file_list", "created", err.Error(),
			"run 'go-broadcast db import <file>' to import a configuration first", jsonOutput)
	}

	// Check for duplicate
	flRepo := db.NewFileListRepository(gormDB)
	if _, err = flRepo.GetByExternalID(ctx, id); err == nil {
		return printErrorResponse("file_list", "created",
			fmt.Sprintf("file list %q already exists", id),
			"use a different --id value", jsonOutput)
	}

	// Count existing for position
	existingLists, err := flRepo.List(ctx, cfg.ID)
	if err != nil {
		return printErrorResponse("file_list", "created", err.Error(), "", jsonOutput)
	}

	fl := &db.FileList{
		ConfigID:    cfg.ID,
		ExternalID:  id,
		Name:        name,
		Description: description,
		Position:    len(existingLists),
	}

	if err = flRepo.Create(ctx, fl); err != nil {
		return printErrorResponse("file_list", "created", err.Error(), "", jsonOutput)
	}

	result := fileListResult{
		ExternalID:  id,
		Name:        name,
		Description: description,
		FileCount:   0,
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "created",
		Type:    "file_list",
		Data:    result,
	}, jsonOutput)
}

func newDBFileListDeleteCmd() *cobra.Command {
	var (
		jsonOutput bool
		hard       bool
	)
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a file list",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFileListDelete(args[0], hard, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&hard, "hard", false, "Hard delete (permanent)")
	return cmd
}

func runFileListDelete(externalID string, hard, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("file_list", "deleted", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	fl, err := resolveFileList(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("file_list", "deleted", err.Error(),
			"run 'go-broadcast db file-list list --json' to see available file lists", jsonOutput)
	}

	flRepo := db.NewFileListRepository(gormDB)
	if err = flRepo.Delete(ctx, fl.ID, hard); err != nil {
		return printErrorResponse("file_list", "deleted", err.Error(), "", jsonOutput)
	}

	deleteType := "soft-deleted"
	if hard {
		deleteType = "hard-deleted"
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  deleteType,
		Type:    "file_list",
		Data:    map[string]string{"external_id": externalID},
	}, jsonOutput)
}

func newDBFileListAddFileCmd() *cobra.Command {
	var (
		jsonOutput bool
		src        string
		dest       string
		deleteFlag bool
	)
	cmd := &cobra.Command{
		Use:   "add-file <id>",
		Short: "Add a file mapping to a file list",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFileListAddFile(args[0], src, dest, deleteFlag, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&src, "src", "", "Source file path")
	cmd.Flags().StringVar(&dest, "dest", "", "Destination file path (required)")
	cmd.Flags().BoolVar(&deleteFlag, "delete", false, "Mark file for deletion")
	_ = cmd.MarkFlagRequired("dest")
	return cmd
}

func runFileListAddFile(externalID, src, dest string, deleteFlag, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("file_mapping", "created", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	fl, err := resolveFileList(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("file_mapping", "created", err.Error(),
			"run 'go-broadcast db file-list list --json' to see available file lists", jsonOutput)
	}

	fmRepo := db.NewFileMappingRepository(gormDB)

	// Check if mapping with same dest already exists
	if _, err = fmRepo.FindByDest(ctx, "file_list", fl.ID, dest); err == nil {
		return printErrorResponse("file_mapping", "created",
			fmt.Sprintf("file mapping with dest %q already exists in file list %q", dest, externalID),
			"use 'remove-file' first to replace it", jsonOutput)
	}

	// Get count for position
	existing, err := fmRepo.ListByOwner(ctx, "file_list", fl.ID)
	if err != nil {
		return printErrorResponse("file_mapping", "created", err.Error(), "", jsonOutput)
	}

	mapping := &db.FileMapping{
		OwnerType:  "file_list",
		OwnerID:    fl.ID,
		Src:        src,
		Dest:       dest,
		DeleteFlag: deleteFlag,
		Position:   len(existing),
	}

	if err = fmRepo.Create(ctx, mapping); err != nil {
		return printErrorResponse("file_mapping", "created", err.Error(), "", jsonOutput)
	}

	result := fileMappingResult{
		Src:    src,
		Dest:   dest,
		Delete: deleteFlag,
	}

	return printResponse(CLIResponse{
		Success: true,
		Action:  "created",
		Type:    "file_mapping",
		Data:    result,
	}, jsonOutput)
}

func newDBFileListRemoveFileCmd() *cobra.Command {
	var (
		jsonOutput bool
		dest       string
	)
	cmd := &cobra.Command{
		Use:   "remove-file <id>",
		Short: "Remove a file mapping from a file list",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFileListRemoveFile(args[0], dest, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&dest, "dest", "", "Destination file path to remove (required)")
	_ = cmd.MarkFlagRequired("dest")
	return cmd
}

func runFileListRemoveFile(externalID, dest string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("file_mapping", "deleted", err.Error(), "", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	gormDB := database.DB()

	fl, err := resolveFileList(ctx, gormDB, externalID)
	if err != nil {
		return printErrorResponse("file_mapping", "deleted", err.Error(),
			"run 'go-broadcast db file-list list --json' to see available file lists", jsonOutput)
	}

	fmRepo := db.NewFileMappingRepository(gormDB)
	mapping, err := fmRepo.FindByDest(ctx, "file_list", fl.ID, dest)
	if err != nil {
		return printErrorResponse("file_mapping", "deleted",
			fmt.Sprintf("file mapping with dest %q not found in file list %q", dest, externalID),
			fmt.Sprintf("run 'go-broadcast db file-list get %s --json' to see available mappings", externalID),
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
			"file_list": externalID,
			"dest":      dest,
		},
	}, jsonOutput)
}
