package db

import "errors"

// Sentinel errors following internal/errors/errors.go conventions
var (
	// ErrRecordNotFound is returned when a requested record does not exist
	ErrRecordNotFound = errors.New("record not found")

	// ErrDuplicateExternalID is returned when attempting to create a record with an external_id that already exists
	ErrDuplicateExternalID = errors.New("duplicate external_id")

	// ErrCircularDependency is returned when a circular dependency is detected in group dependencies
	ErrCircularDependency = errors.New("circular dependency detected")

	// ErrImportFailed is returned when YAML import to database fails
	ErrImportFailed = errors.New("import failed")

	// ErrExportFailed is returned when database to YAML export fails
	ErrExportFailed = errors.New("export failed")

	// ErrInvalidMetadata is returned when metadata JSON is malformed
	ErrInvalidMetadata = errors.New("invalid metadata JSON")

	// ErrInvalidReference is returned when a reference (file_list_ref, directory_list_ref) cannot be resolved
	ErrInvalidReference = errors.New("invalid reference")

	// ErrReferenceNotFound is returned when a referenced external ID does not exist
	ErrReferenceNotFound = errors.New("reference not found")

	// ErrValidationFailed is returned when model validation fails
	ErrValidationFailed = errors.New("validation failed")

	// ErrInvalidType is returned when scanning a value of incorrect type
	ErrInvalidType = errors.New("invalid type")

	// ErrMigrationNotFound is returned when a migration definition cannot be found
	ErrMigrationNotFound = errors.New("migration definition not found")

	// ErrNoDownFunction is returned when attempting to rollback a migration without a down function
	ErrNoDownFunction = errors.New("migration has no down function")

	// ErrMigrationStateChanged is returned when the migration state changes during a rollback operation
	ErrMigrationStateChanged = errors.New("migration state changed during rollback")

	// ErrInvalidRepoFormat is returned when a repo string is not in "org/repo" format
	ErrInvalidRepoFormat = errors.New("invalid repo format")

	// ErrMissingExternalID is returned when attempting to create a sync run without an external_id
	ErrMissingExternalID = errors.New("external_id is required")

	// ErrMissingSyncRunID is returned when attempting to update a sync run with ID 0
	ErrMissingSyncRunID = errors.New("cannot update sync run with ID 0")

	// ErrMissingBroadcastSyncRunID is returned when creating a target result without a broadcast_sync_run_id
	ErrMissingBroadcastSyncRunID = errors.New("broadcast_sync_run_id is required")

	// ErrMissingTargetID is returned when creating a target result without a target_id
	ErrMissingTargetID = errors.New("target_id is required")

	// ErrMissingRepoID is returned when creating a target result without a repo_id
	ErrMissingRepoID = errors.New("repo_id is required")

	// ErrMissingTargetResultID is returned when attempting to update a target result with ID 0
	ErrMissingTargetResultID = errors.New("cannot update target result with ID 0")

	// ErrMissingBroadcastSyncTargetResultID is returned when creating a file change without a broadcast_sync_target_result_id
	ErrMissingBroadcastSyncTargetResultID = errors.New("broadcast_sync_target_result_id is required")

	// ErrMissingFilePath is returned when creating a file change without a file_path
	ErrMissingFilePath = errors.New("file_path is required")
)
