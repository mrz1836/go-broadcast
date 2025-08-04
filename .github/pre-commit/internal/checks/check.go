// Package checks provides the check interface and registry for pre-commit checks
package checks

import (
	"context"
	"time"
)

// CheckMetadata contains metadata about a check
type CheckMetadata struct {
	// Name is the unique identifier for the check
	Name string

	// Description explains what the check does
	Description string

	// FilePatterns defines which file patterns this check processes
	FilePatterns []string

	// EstimatedDuration is the expected execution time for this check
	EstimatedDuration time.Duration

	// Dependencies lists make targets or tools this check requires
	Dependencies []string

	// DefaultTimeout is the default timeout for this check
	DefaultTimeout time.Duration

	// Category groups related checks together (e.g., "formatting", "linting")
	Category string

	// RequiresFiles indicates if the check needs at least one file to run
	RequiresFiles bool
}

// Check is the interface that all pre-commit checks must implement
type Check interface {
	// Name returns the name of the check
	Name() string

	// Description returns a brief description of what the check does
	Description() string

	// Metadata returns comprehensive metadata about the check
	Metadata() interface{}

	// Run executes the check on the given files
	Run(ctx context.Context, files []string) error

	// FilterFiles filters the list of files to only those this check should process
	FilterFiles(files []string) []string
}
