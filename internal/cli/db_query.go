package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

//nolint:gochecknoglobals // Cobra commands and flags are designed to be global variables
var (
	dbQueryFile     string
	dbQueryRepo     string
	dbQueryFileList string
	dbQueryContains string
	dbQueryJSON     bool

	dbQueryCmd = &cobra.Command{
		Use:   "query",
		Short: "Query database configuration",
		Long: `Query the database to answer questions about configuration.

Available queries:
• Which repos sync a specific file? (--file)
• What files sync to a specific repo? (--repo)
• Which targets use a file list? (--file-list)
• Search file paths by pattern (--contains)

Examples:
  # Which repos sync .github/workflows/ci.yml?
  go-broadcast db query --file .github/workflows/ci.yml

  # What files sync to mrz1836/example-repo?
  go-broadcast db query --repo mrz1836/example-repo

  # Which targets use the "ai-files" file list?
  go-broadcast db query --file-list ai-files

  # Search for all workflows
  go-broadcast db query --contains workflows

  # JSON output
  go-broadcast db query --file .editorconfig --json`,
		RunE: runDBQuery,
	}
)

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	dbQueryCmd.Flags().StringVar(&dbQueryFile, "file", "", "Find repos that sync this file path")
	dbQueryCmd.Flags().StringVar(&dbQueryRepo, "repo", "", "Find files that sync to this repo")
	dbQueryCmd.Flags().StringVar(&dbQueryFileList, "file-list", "", "Find targets using this file list (external ID)")
	dbQueryCmd.Flags().StringVar(&dbQueryContains, "contains", "", "Search file paths containing this pattern")
	dbQueryCmd.Flags().BoolVar(&dbQueryJSON, "json", false, "Output as JSON")
}

// QueryResult represents a query result
type QueryResult struct {
	Query   string      `json:"query"`
	Results interface{} `json:"results"`
	Count   int         `json:"count"`
}

// runDBQuery executes the database query command
func runDBQuery(_ *cobra.Command, _ []string) error {
	// Validate exactly one query flag is specified
	flagCount := 0
	if dbQueryFile != "" {
		flagCount++
	}
	if dbQueryRepo != "" {
		flagCount++
	}
	if dbQueryFileList != "" {
		flagCount++
	}
	if dbQueryContains != "" {
		flagCount++
	}

	if flagCount == 0 {
		return fmt.Errorf("must specify one of: --file, --repo, --file-list, or --contains") //nolint:err113 // user-facing CLI validation error
	}
	if flagCount > 1 {
		return fmt.Errorf("only one query flag allowed at a time") //nolint:err113 // user-facing CLI validation error
	}

	path := getDBPath()

	// Check if database exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("database does not exist: %s (run 'go-broadcast db init' to create)", path) //nolint:err113 // user-facing CLI error
	}

	// Open database
	database, err := db.Open(db.OpenOptions{
		Path:     path,
		LogLevel: logger.Silent,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	queryRepo := db.NewQueryRepository(database.DB())
	ctx := context.Background()

	// Execute the appropriate query
	switch {
	case dbQueryFile != "":
		return queryByFile(ctx, queryRepo, dbQueryFile)
	case dbQueryRepo != "":
		return queryByRepo(ctx, queryRepo, dbQueryRepo)
	case dbQueryFileList != "":
		return queryByFileList(ctx, database, dbQueryFileList)
	case dbQueryContains != "":
		return queryByPattern(ctx, queryRepo, dbQueryContains)
	}

	return nil
}

// queryByFile finds all targets that sync a specific file
func queryByFile(ctx context.Context, repo db.QueryRepository, filePath string) error {
	targets, err := repo.FindByFile(ctx, filePath)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	result := QueryResult{
		Query:   fmt.Sprintf("file: %s", filePath),
		Results: targets,
		Count:   len(targets),
	}

	if dbQueryJSON {
		return printJSON(result)
	}

	// Human-readable output
	output.Info(fmt.Sprintf("File: %s", filePath))
	output.Info(fmt.Sprintf("Found %d target(s):\n", len(targets)))

	if len(targets) == 0 {
		output.Warn("No targets found syncing this file")
		return nil
	}

	for _, target := range targets {
		output.Info(fmt.Sprintf("  • %s", target.RepoRef.FullName()))

		// Show matching file mappings
		for _, fm := range target.FileMappings {
			if fm.Dest == filePath || fm.Src == filePath {
				if fm.Src == "" {
					output.Info(fmt.Sprintf("    → dest: %s", fm.Dest))
				} else {
					output.Info(fmt.Sprintf("    → src: %s, dest: %s", fm.Src, fm.Dest))
				}
			}
		}
	}

	return nil
}

// queryByRepo finds all files that sync to a specific repo
func queryByRepo(ctx context.Context, repo db.QueryRepository, repoName string) error {
	target, err := repo.FindByRepo(ctx, repoName)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			if dbQueryJSON {
				return printJSON(QueryResult{
					Query:   fmt.Sprintf("repo: %s", repoName),
					Results: nil,
					Count:   0,
				})
			}
			output.Warn(fmt.Sprintf("No configuration found for repo: %s", repoName))
			return nil
		}
		return fmt.Errorf("query failed: %w", err)
	}

	result := QueryResult{
		Query:   fmt.Sprintf("repo: %s", repoName),
		Results: target,
		Count:   len(target.FileMappings) + len(target.DirectoryMappings),
	}

	if dbQueryJSON {
		return printJSON(result)
	}

	// Human-readable output
	output.Info(fmt.Sprintf("Repository: %s", repoName))
	output.Info(fmt.Sprintf("Branch: %s", target.Branch))
	output.Info("")

	// Show file mappings
	if len(target.FileMappings) > 0 {
		output.Info("File Mappings:")
		for _, fm := range target.FileMappings {
			if fm.DeleteFlag {
				output.Info(fmt.Sprintf("  • DELETE: %s", fm.Dest))
			} else if fm.Src == "" {
				output.Info(fmt.Sprintf("  • %s", fm.Dest))
			} else {
				output.Info(fmt.Sprintf("  • %s → %s", fm.Src, fm.Dest))
			}
		}
		output.Info("")
	}

	// Show directory mappings
	if len(target.DirectoryMappings) > 0 {
		output.Info("Directory Mappings:")
		for _, dm := range target.DirectoryMappings {
			if dm.DeleteFlag {
				output.Info(fmt.Sprintf("  • DELETE: %s", dm.Dest))
			} else if dm.Src == "" {
				output.Info(fmt.Sprintf("  • %s", dm.Dest))
			} else {
				output.Info(fmt.Sprintf("  • %s → %s", dm.Src, dm.Dest))
			}
		}
		output.Info("")
	}

	// Show transform
	if target.Transform.ID != 0 {
		output.Info("Transform:")
		if target.Transform.RepoName {
			output.Info("  • repo_name: enabled")
		}
		if len(target.Transform.Variables) > 0 {
			output.Info(fmt.Sprintf("  • variables: %d defined", len(target.Transform.Variables)))
		}
		output.Info("")
	}

	// Show file list refs
	if len(target.FileListRefs) > 0 {
		output.Info("File List References:")
		for _, ref := range target.FileListRefs {
			if ref.FileList.ID != 0 {
				output.Info(fmt.Sprintf("  • %s (%s)", ref.FileList.Name, ref.FileList.ExternalID))
			}
		}
		output.Info("")
	}

	// Show directory list refs
	if len(target.DirectoryListRefs) > 0 {
		output.Info("Directory List References:")
		for _, ref := range target.DirectoryListRefs {
			if ref.DirectoryList.ID != 0 {
				output.Info(fmt.Sprintf("  • %s (%s)", ref.DirectoryList.Name, ref.DirectoryList.ExternalID))
			}
		}
		output.Info("")
	}

	totalMappings := len(target.FileMappings) + len(target.DirectoryMappings)
	output.Info(fmt.Sprintf("Total: %d mapping(s)", totalMappings))

	return nil
}

// queryByFileList finds all targets using a specific file list
func queryByFileList(ctx context.Context, database db.Database, externalID string) error {
	// First, find the file list by external ID
	fileListRepo := db.NewFileListRepository(database.DB())
	fileList, err := fileListRepo.GetByExternalID(ctx, externalID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			if dbQueryJSON {
				return printJSON(QueryResult{
					Query:   fmt.Sprintf("file-list: %s", externalID),
					Results: nil,
					Count:   0,
				})
			}
			output.Warn(fmt.Sprintf("File list not found: %s", externalID))
			return nil
		}
		return fmt.Errorf("failed to find file list: %w", err)
	}

	// Now find all targets using this file list
	queryRepo := db.NewQueryRepository(database.DB())
	targets, err := queryRepo.FindByFileList(ctx, fileList.ID)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	result := QueryResult{
		Query:   fmt.Sprintf("file-list: %s", externalID),
		Results: targets,
		Count:   len(targets),
	}

	if dbQueryJSON {
		return printJSON(result)
	}

	// Human-readable output
	output.Info(fmt.Sprintf("File List: %s (%s)", fileList.Name, fileList.ExternalID))
	if fileList.Description != "" {
		output.Info(fmt.Sprintf("Description: %s", fileList.Description))
	}
	output.Info(fmt.Sprintf("\nFound %d target(s) using this list:\n", len(targets)))

	if len(targets) == 0 {
		output.Warn("No targets found using this file list")
		return nil
	}

	for _, target := range targets {
		output.Info(fmt.Sprintf("  • %s", target.RepoRef.FullName()))
	}

	return nil
}

// queryByPattern searches file paths matching a pattern
func queryByPattern(ctx context.Context, repo db.QueryRepository, pattern string) error {
	fileMappings, err := repo.FindByPattern(ctx, pattern)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	result := QueryResult{
		Query:   fmt.Sprintf("contains: %s", pattern),
		Results: fileMappings,
		Count:   len(fileMappings),
	}

	if dbQueryJSON {
		return printJSON(result)
	}

	// Human-readable output
	output.Info(fmt.Sprintf("Pattern: %s", pattern))
	output.Info(fmt.Sprintf("Found %d file mapping(s):\n", len(fileMappings)))

	if len(fileMappings) == 0 {
		output.Warn("No file mappings found matching this pattern")
		return nil
	}

	for _, fm := range fileMappings {
		if fm.DeleteFlag {
			output.Info(fmt.Sprintf("  • DELETE: %s (type: %s, id: %d)", fm.Dest, fm.OwnerType, fm.OwnerID))
		} else if fm.Src == "" {
			output.Info(fmt.Sprintf("  • dest: %s (type: %s, id: %d)", fm.Dest, fm.OwnerType, fm.OwnerID))
		} else {
			output.Info(fmt.Sprintf("  • src: %s → dest: %s (type: %s, id: %d)",
				fm.Src, fm.Dest, fm.OwnerType, fm.OwnerID))
		}
	}

	return nil
}

// printJSON prints a result as JSON
func printJSON(result interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
