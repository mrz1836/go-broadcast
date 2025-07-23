package cli

// Flags contains all global flags for the CLI
type Flags struct {
	ConfigFile string
	DryRun     bool
	LogLevel   string
}

// globalFlags is the singleton instance of flags
var globalFlags = &Flags{
	ConfigFile: "sync.yaml",
	LogLevel:   "info",
}

// GetConfigFile returns the config file path
func GetConfigFile() string {
	return globalFlags.ConfigFile
}

// IsDryRun returns whether dry-run mode is enabled
func IsDryRun() bool {
	return globalFlags.DryRun
}

// SetFlags updates the global flags
func SetFlags(f *Flags) {
	globalFlags = f
}

