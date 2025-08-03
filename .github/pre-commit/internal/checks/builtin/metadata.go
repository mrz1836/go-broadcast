package builtin

import "time"

// CheckMetadata contains metadata about a check
type CheckMetadata struct {
	Name              string
	Description       string
	FilePatterns      []string
	EstimatedDuration time.Duration
	Dependencies      []string
	DefaultTimeout    time.Duration
	Category          string
	RequiresFiles     bool
}
