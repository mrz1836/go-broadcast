package cli

import "errors"

// Common CLI errors
var (
	// ErrConfigFileNotFound indicates the configuration file was not found
	ErrConfigFileNotFound = errors.New("configuration file not found")
	
	// ErrNoMatchingTargets indicates no targets matched the filter
	ErrNoMatchingTargets = errors.New("no matching targets found")
)