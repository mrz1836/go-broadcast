package config

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

// ApplyDirectoryDefaults applies default values to directory mappings
func ApplyDirectoryDefaults(dm *DirectoryMapping) {
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
