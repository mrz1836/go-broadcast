package config

import (
	"fmt"
	"io"
	"os"

	"github.com/mrz1836/go-broadcast/internal/logging"
	"gopkg.in/yaml.v3"
)

// Load reads and parses a configuration file from the given path
func Load(path string) (*Config, error) {
	// Initialize audit logger for security event tracking
	auditLogger := logging.NewAuditLogger()

	file, err := os.Open(path) //#nosec G304 -- Path is user-provided config file
	if err != nil {
		// Log failed configuration access
		auditLogger.LogConfigChange("system", "config_load_failed", path)
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}

	defer func() { _ = file.Close() }()

	config, parseErr := LoadFromReader(file)
	if parseErr != nil {
		// Log failed configuration parsing
		auditLogger.LogConfigChange("system", "config_parse_failed", path)
		return nil, parseErr
	}

	// Log successful configuration loading
	auditLogger.LogConfigChange("system", "config_loaded", path)

	return config, nil
}

// LoadFromReader parses configuration from an io.Reader
func LoadFromReader(reader io.Reader) (*Config, error) {
	config := &Config{}

	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true) // Strict parsing - fail on unknown fields

	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply defaults
	applyDefaults(config)

	return config, nil
}

// applyDefaults sets default values for optional fields using compatibility layer
func applyDefaults(config *Config) {
	// Apply defaults to both old format and new format configurations
	// First, handle old format fields for backward compatibility
	if config.Source.Branch == "" {
		config.Source.Branch = "main"
	}

	if config.Defaults.BranchPrefix == "" {
		config.Defaults.BranchPrefix = "chore/sync-files"
	}

	if len(config.Defaults.PRLabels) == 0 {
		config.Defaults.PRLabels = []string{"automated-sync"}
	}

	// Apply directory defaults to old format targets
	for i := range config.Targets {
		for j := range config.Targets[i].Directories {
			ApplyDirectoryDefaults(&config.Targets[i].Directories[j])
		}
	}

	// Apply defaults to groups if they exist
	for i := range config.Groups {
		group := &config.Groups[i]

		// Set default source branch if not specified
		if group.Source.Branch == "" {
			group.Source.Branch = "main"
		}

		// Set default branch prefix if not specified
		if group.Defaults.BranchPrefix == "" {
			group.Defaults.BranchPrefix = "chore/sync-files"
		}

		// Set default PR labels if not specified
		if len(group.Defaults.PRLabels) == 0 {
			group.Defaults.PRLabels = []string{"automated-sync"}
		}

		// Set default enabled state if not specified
		if group.Enabled == nil {
			group.Enabled = boolPtr(true)
		}

		// Apply directory defaults to group targets
		for j := range group.Targets {
			for k := range group.Targets[j].Directories {
				ApplyDirectoryDefaults(&group.Targets[j].Directories[k])
			}
		}
	}
}
