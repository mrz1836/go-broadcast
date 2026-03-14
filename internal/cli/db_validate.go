package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

//nolint:gochecknoglobals // Cobra commands and flags are designed to be global variables
var (
	dbValidateJSON bool
	dbValidateCmd  = &cobra.Command{
		Use:   "validate",
		Short: "Validate database consistency",
		Long: `Validate database consistency and check for common issues.

Checks performed:
• Orphaned file list references (targets referencing non-existent file lists)
• Orphaned directory list references (targets referencing non-existent directory lists)
• Missing file lists (file list refs pointing to deleted lists)
• Missing directory lists (directory list refs pointing to deleted lists)
• Circular dependencies between groups
• Orphaned file mappings (mappings with invalid owner references)
• Orphaned directory mappings (mappings with invalid owner references)

Examples:
  # Validate database
  go-broadcast db validate

  # Validate with JSON output
  go-broadcast db validate --json`,
		RunE: runDBValidate,
	}
)

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	dbValidateCmd.Flags().BoolVar(&dbValidateJSON, "json", false, "Output as JSON")
}

// ValidationResult represents validation results
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors"`
	Checks []ValidationCheck `json:"checks"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ValidationCheck represents a successful check
type ValidationCheck struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Count   int    `json:"count,omitempty"`
}

// runDBValidate executes the database validation command
func runDBValidate(_ *cobra.Command, _ []string) error {
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

	gormDB := database.DB()
	ctx := context.Background()

	result := ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
		Checks: []ValidationCheck{},
	}

	// Get the config ID (assume single config for now)
	var config db.Config
	if err := gormDB.First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Checks = append(result.Checks, ValidationCheck{
				Type:    "config",
				Message: "No configuration found in database",
			})
			return printValidationResult(result)
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Run all validation checks
	checkOrphanedFileListRefs(ctx, gormDB, &result)
	checkOrphanedDirectoryListRefs(ctx, gormDB, &result)
	checkOrphanedFileMappings(ctx, gormDB, &result)
	checkOrphanedDirectoryMappings(ctx, gormDB, &result)
	checkCircularDependencies(ctx, gormDB, config.ID, &result)

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	return printValidationResult(result)
}

// checkOrphanedFileListRefs checks for target_file_list_refs pointing to non-existent file lists
func checkOrphanedFileListRefs(ctx context.Context, gormDB *gorm.DB, result *ValidationResult) {
	type OrphanRef struct {
		TargetID   uint
		FileListID uint
		TargetRepo string
	}

	var orphans []OrphanRef
	err := gormDB.WithContext(ctx).
		Table("target_file_list_refs").
		Select("target_file_list_refs.target_id, target_file_list_refs.file_list_id, COALESCE(organizations.name || '/' || repos.name, 'unknown') as target_repo").
		Joins("LEFT JOIN file_lists ON file_lists.id = target_file_list_refs.file_list_id").
		Joins("LEFT JOIN targets ON targets.id = target_file_list_refs.target_id").
		Joins("LEFT JOIN repos ON repos.id = targets.repo_id").
		Joins("LEFT JOIN organizations ON organizations.id = repos.organization_id").
		Where("file_lists.id IS NULL OR file_lists.deleted_at IS NOT NULL").
		Scan(&orphans).Error
	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "orphaned_file_list_refs",
			Message: "Failed to check orphaned file list refs",
			Details: err.Error(),
		})
		return
	}

	if len(orphans) > 0 {
		for _, orphan := range orphans {
			result.Errors = append(result.Errors, ValidationError{
				Type: "orphaned_file_list_ref",
				Message: fmt.Sprintf("Target '%s' references non-existent file list (ID: %d)",
					orphan.TargetRepo, orphan.FileListID),
				Details: fmt.Sprintf("target_id=%d, file_list_id=%d", orphan.TargetID, orphan.FileListID),
			})
		}
	} else {
		result.Checks = append(result.Checks, ValidationCheck{
			Type:    "file_list_refs",
			Message: "All file list references are valid",
		})
	}
}

// checkOrphanedDirectoryListRefs checks for target_directory_list_refs pointing to non-existent directory lists
func checkOrphanedDirectoryListRefs(ctx context.Context, gormDB *gorm.DB, result *ValidationResult) {
	type OrphanRef struct {
		TargetID        uint
		DirectoryListID uint
		TargetRepo      string
	}

	var orphans []OrphanRef
	err := gormDB.WithContext(ctx).
		Table("target_directory_list_refs").
		Select("target_directory_list_refs.target_id, target_directory_list_refs.directory_list_id, COALESCE(organizations.name || '/' || repos.name, 'unknown') as target_repo").
		Joins("LEFT JOIN directory_lists ON directory_lists.id = target_directory_list_refs.directory_list_id").
		Joins("LEFT JOIN targets ON targets.id = target_directory_list_refs.target_id").
		Joins("LEFT JOIN repos ON repos.id = targets.repo_id").
		Joins("LEFT JOIN organizations ON organizations.id = repos.organization_id").
		Where("directory_lists.id IS NULL OR directory_lists.deleted_at IS NOT NULL").
		Scan(&orphans).Error
	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "orphaned_directory_list_refs",
			Message: "Failed to check orphaned directory list refs",
			Details: err.Error(),
		})
		return
	}

	if len(orphans) > 0 {
		for _, orphan := range orphans {
			result.Errors = append(result.Errors, ValidationError{
				Type: "orphaned_directory_list_ref",
				Message: fmt.Sprintf("Target '%s' references non-existent directory list (ID: %d)",
					orphan.TargetRepo, orphan.DirectoryListID),
				Details: fmt.Sprintf("target_id=%d, directory_list_id=%d", orphan.TargetID, orphan.DirectoryListID),
			})
		}
	} else {
		result.Checks = append(result.Checks, ValidationCheck{
			Type:    "directory_list_refs",
			Message: "All directory list references are valid",
		})
	}
}

// checkOrphanedFileMappings checks for file_mappings with invalid owner references
func checkOrphanedFileMappings(ctx context.Context, gormDB *gorm.DB, result *ValidationResult) {
	type OrphanMapping struct {
		ID        uint
		OwnerType string
		OwnerID   uint
		Dest      string
	}

	// Check target-owned file mappings
	var targetOrphans []OrphanMapping
	err := gormDB.WithContext(ctx).
		Table("file_mappings").
		Select("file_mappings.id, file_mappings.owner_type, file_mappings.owner_id, file_mappings.dest").
		Joins("LEFT JOIN targets ON targets.id = file_mappings.owner_id").
		Where("file_mappings.owner_type = ? AND (targets.id IS NULL OR targets.deleted_at IS NOT NULL)", "target").
		Scan(&targetOrphans).Error

	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "orphaned_file_mappings_check",
			Message: "Failed to check orphaned file mappings (target)",
			Details: err.Error(),
		})
	} else if len(targetOrphans) > 0 {
		for _, orphan := range targetOrphans {
			result.Errors = append(result.Errors, ValidationError{
				Type: "orphaned_file_mapping",
				Message: fmt.Sprintf("File mapping '%s' references non-existent target (ID: %d)",
					orphan.Dest, orphan.OwnerID),
				Details: fmt.Sprintf("file_mapping_id=%d", orphan.ID),
			})
		}
	}

	// Check file_list-owned file mappings
	var listOrphans []OrphanMapping
	err = gormDB.WithContext(ctx).
		Table("file_mappings").
		Select("file_mappings.id, file_mappings.owner_type, file_mappings.owner_id, file_mappings.dest").
		Joins("LEFT JOIN file_lists ON file_lists.id = file_mappings.owner_id").
		Where("file_mappings.owner_type = ? AND (file_lists.id IS NULL OR file_lists.deleted_at IS NOT NULL)", "file_list").
		Scan(&listOrphans).Error

	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "orphaned_file_mappings_check",
			Message: "Failed to check orphaned file mappings (file_list)",
			Details: err.Error(),
		})
	} else if len(listOrphans) > 0 {
		for _, orphan := range listOrphans {
			result.Errors = append(result.Errors, ValidationError{
				Type: "orphaned_file_mapping",
				Message: fmt.Sprintf("File mapping '%s' references non-existent file list (ID: %d)",
					orphan.Dest, orphan.OwnerID),
				Details: fmt.Sprintf("file_mapping_id=%d", orphan.ID),
			})
		}
	}

	if len(targetOrphans) == 0 && len(listOrphans) == 0 {
		result.Checks = append(result.Checks, ValidationCheck{
			Type:    "file_mappings",
			Message: "All file mappings have valid owner references",
		})
	}
}

// checkOrphanedDirectoryMappings checks for directory_mappings with invalid owner references
func checkOrphanedDirectoryMappings(ctx context.Context, gormDB *gorm.DB, result *ValidationResult) {
	type OrphanMapping struct {
		ID        uint
		OwnerType string
		OwnerID   uint
		Dest      string
	}

	// Check target-owned directory mappings
	var targetOrphans []OrphanMapping
	err := gormDB.WithContext(ctx).
		Table("directory_mappings").
		Select("directory_mappings.id, directory_mappings.owner_type, directory_mappings.owner_id, directory_mappings.dest").
		Joins("LEFT JOIN targets ON targets.id = directory_mappings.owner_id").
		Where("directory_mappings.owner_type = ? AND (targets.id IS NULL OR targets.deleted_at IS NOT NULL)", "target").
		Scan(&targetOrphans).Error

	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "orphaned_directory_mappings_check",
			Message: "Failed to check orphaned directory mappings (target)",
			Details: err.Error(),
		})
	} else if len(targetOrphans) > 0 {
		for _, orphan := range targetOrphans {
			result.Errors = append(result.Errors, ValidationError{
				Type: "orphaned_directory_mapping",
				Message: fmt.Sprintf("Directory mapping '%s' references non-existent target (ID: %d)",
					orphan.Dest, orphan.OwnerID),
				Details: fmt.Sprintf("directory_mapping_id=%d", orphan.ID),
			})
		}
	}

	// Check directory_list-owned directory mappings
	var listOrphans []OrphanMapping
	err = gormDB.WithContext(ctx).
		Table("directory_mappings").
		Select("directory_mappings.id, directory_mappings.owner_type, directory_mappings.owner_id, directory_mappings.dest").
		Joins("LEFT JOIN directory_lists ON directory_lists.id = directory_mappings.owner_id").
		Where("directory_mappings.owner_type = ? AND (directory_lists.id IS NULL OR directory_lists.deleted_at IS NOT NULL)", "directory_list").
		Scan(&listOrphans).Error

	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "orphaned_directory_mappings_check",
			Message: "Failed to check orphaned directory mappings (directory_list)",
			Details: err.Error(),
		})
	} else if len(listOrphans) > 0 {
		for _, orphan := range listOrphans {
			result.Errors = append(result.Errors, ValidationError{
				Type: "orphaned_directory_mapping",
				Message: fmt.Sprintf("Directory mapping '%s' references non-existent directory list (ID: %d)",
					orphan.Dest, orphan.OwnerID),
				Details: fmt.Sprintf("directory_mapping_id=%d", orphan.ID),
			})
		}
	}

	if len(targetOrphans) == 0 && len(listOrphans) == 0 {
		result.Checks = append(result.Checks, ValidationCheck{
			Type:    "directory_mappings",
			Message: "All directory mappings have valid owner references",
		})
	}
}

// checkCircularDependencies checks for circular dependencies between groups
func checkCircularDependencies(ctx context.Context, gormDB *gorm.DB, configID uint, result *ValidationResult) {
	// Use the dependency.go validation function
	err := db.ValidateGroupDependencies(ctx, gormDB, configID)
	if err != nil {
		if errors.Is(err, db.ErrCircularDependency) {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "circular_dependency",
				Message: "Circular dependency detected between groups",
				Details: err.Error(),
			})
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "dependency_check",
				Message: "Failed to validate group dependencies",
				Details: err.Error(),
			})
		}
		return
	}

	// Count groups and dependencies
	var groupCount, depCount int64
	gormDB.WithContext(ctx).Model(&db.Group{}).Where("config_id = ?", configID).Count(&groupCount)
	gormDB.WithContext(ctx).Model(&db.GroupDependency{}).
		Joins("JOIN groups ON groups.id = group_dependencies.group_id").
		Where("groups.config_id = ?", configID).
		Count(&depCount)

	result.Checks = append(result.Checks, ValidationCheck{
		Type:    "dependencies",
		Message: fmt.Sprintf("No circular dependencies (checked %d groups, %d dependencies)", groupCount, depCount),
		Count:   int(depCount),
	})
}

// printValidationResult prints the validation result
func printValidationResult(result ValidationResult) error {
	if dbValidateJSON {
		return printJSON(result)
	}

	// Human-readable output
	output.Info("Database Validation Results")
	output.Info("===========================\n")

	// Print successful checks
	if len(result.Checks) > 0 {
		output.Info("Checks Passed:")
		for _, check := range result.Checks {
			output.Info(fmt.Sprintf("  ✓ %s", check.Message))
		}
		output.Info("")
	}

	// Print errors
	if len(result.Errors) > 0 {
		output.Error(fmt.Sprintf("Found %d error(s):", len(result.Errors)))
		for _, err := range result.Errors {
			output.Error(fmt.Sprintf("  ✗ %s", err.Message))
			if err.Details != "" {
				output.Error(fmt.Sprintf("    Details: %s", err.Details))
			}
		}
		output.Info("")
	}

	// Summary
	if result.Valid {
		output.Info("✓ Database is valid!")
		return nil
	}

	output.Error(fmt.Sprintf("✗ Database validation failed with %d error(s)", len(result.Errors)))
	return fmt.Errorf("validation failed") //nolint:err113 // validation summary error
}
