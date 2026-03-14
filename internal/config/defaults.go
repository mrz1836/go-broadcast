// Package config provides configuration defaults and helpers for go-broadcast.
package config

// DefaultBlobSizeLimit is the default maximum blob size for partial clone.
// Blobs larger than this are excluded during git clone operations.
// Use "0" to disable filtering and clone all blobs.
const DefaultBlobSizeLimit = "10m"

// DefaultExclusions returns smart default exclusions for development artifacts
func DefaultExclusions() []string {
	return []string{
		"*.out",        // Coverage outputs
		"*.test",       // Go test binaries
		"*.exe",        // Windows executables
		"**/.DS_Store", // macOS files
		"**/tmp/*",     // Temporary files
		"**/.git",      // Git directories
	}
}

// ApplyDirectoryDefaults applies default values to directory mappings.
// If dm is nil, the function returns immediately without panic.
func ApplyDirectoryDefaults(dm *DirectoryMapping) {
	if dm == nil {
		return
	}
	if dm.Exclude == nil {
		dm.Exclude = DefaultExclusions()
	}
	// Set default values for boolean pointers if not explicitly set
	if dm.PreserveStructure == nil {
		preserveStructure := true
		dm.PreserveStructure = &preserveStructure
	}
	if dm.IncludeHidden == nil {
		includeHidden := true
		dm.IncludeHidden = &includeHidden
	}
}
