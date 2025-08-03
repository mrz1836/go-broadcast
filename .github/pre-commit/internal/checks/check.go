// Package checks provides the check interface and registry for pre-commit checks
package checks

import (
	"context"
)

// Check is the interface that all pre-commit checks must implement
type Check interface {
	// Name returns the name of the check
	Name() string

	// Description returns a brief description of what the check does
	Description() string

	// Run executes the check on the given files
	Run(ctx context.Context, files []string) error

	// FilterFiles filters the list of files to only those this check should process
	FilterFiles(files []string) []string
}
