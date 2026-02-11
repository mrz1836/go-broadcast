package cli

import "errors"

// Common CLI errors
var (
	// ErrConfigFileNotFound indicates the configuration file was not found
	ErrConfigFileNotFound = errors.New("configuration file not found")

	// ErrNoMatchingTargets indicates no targets matched the filter
	ErrNoMatchingTargets = errors.New("no matching targets found")

	// ErrNilConfig indicates a nil configuration was passed to a function that requires a valid config
	ErrNilConfig = errors.New("config cannot be nil")

	// ErrDatabaseFileNotFound indicates the database file was not found
	ErrDatabaseFileNotFound = errors.New("database file not found")
)
