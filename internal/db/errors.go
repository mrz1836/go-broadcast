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
)
