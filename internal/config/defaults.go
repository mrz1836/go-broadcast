// Package config provides configuration defaults and helpers for go-broadcast.
package config

// DefaultBlobSizeLimit is the default maximum blob size for partial clone.
// Blobs larger than this are excluded during git clone operations.
// Use "0" to disable filtering and clone all blobs.
const DefaultBlobSizeLimit = "10m"

// Rate-limit preflight defaults (see RateLimitPreflightConfig). These match the
// conservative defaults agreed for the sync preflight gate: keep 20% of the
// live primary budget as headroom, and reserve 10 of the documented 80/min
// secondary content-write slots.
const (
	// DefaultRateLimitPrimaryMarginPercent is the default primary-budget headroom.
	DefaultRateLimitPrimaryMarginPercent = 20

	// DefaultRateLimitSecondaryReserve is the default per-minute secondary reserve.
	DefaultRateLimitSecondaryReserve = 10
)

// ResolveRateLimitPreflight returns the effective preflight settings for cfg,
// applying the documented defaults for any unset/zero field. It is defensive:
// it works correctly even on a Config that has not been run through
// applyDefaults (e.g. a hand-built config in tests) or a nil cfg.
func ResolveRateLimitPreflight(cfg *Config) (enabled bool, primaryMarginPercent, secondaryReserve int, failClosed bool) {
	enabled = true
	primaryMarginPercent = DefaultRateLimitPrimaryMarginPercent
	secondaryReserve = DefaultRateLimitSecondaryReserve
	failClosed = false

	if cfg == nil {
		return enabled, primaryMarginPercent, secondaryReserve, failClosed
	}

	rl := cfg.RateLimitPreflight
	if rl.Enabled != nil {
		enabled = *rl.Enabled
	}
	if rl.PrimaryMarginPercent != 0 {
		primaryMarginPercent = rl.PrimaryMarginPercent
	}
	if rl.SecondaryReserve != 0 {
		secondaryReserve = rl.SecondaryReserve
	}
	failClosed = rl.FailClosed

	return enabled, primaryMarginPercent, secondaryReserve, failClosed
}

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
