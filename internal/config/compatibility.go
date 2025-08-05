package config

// Compatibility layer for seamless transition between old and new configuration formats.
// This file provides methods to work with both single-source and group-based configurations
// transparently during the transition period.

// GetGroups returns groups, converting from old format if needed.
// This method enables consuming code to work with groups regardless of the config format.
func (c *Config) GetGroups() []Group {
	// If groups are already defined, return them directly
	if len(c.Groups) > 0 {
		return c.Groups
	}

	// Convert old format to group format for compatibility
	if c.Source.Repo != "" {
		return []Group{{
			Name:     "default",
			ID:       "default",
			Priority: 0,
			Enabled:  boolPtr(true),
			Source:   c.Source,
			Global:   c.Global,
			Defaults: c.Defaults,
			Targets:  c.Targets,
		}}
	}

	// No valid configuration found
	return nil
}

// IsGroupBased returns true if using new group format.
// This method helps consuming code detect which configuration format is being used.
func (c *Config) IsGroupBased() bool {
	return len(c.Groups) > 0
}

// boolPtr is a helper function to create a pointer to a boolean value.
// This is used for optional boolean fields with default values.
func boolPtr(b bool) *bool {
	return &b
}
