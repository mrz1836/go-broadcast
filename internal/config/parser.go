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

// applyDefaults sets default values for optional fields
func applyDefaults(config *Config) {
	// Set default branch prefix if not specified
	if config.Defaults.BranchPrefix == "" {
		config.Defaults.BranchPrefix = "chore/sync-files"
	}

	// Set default PR labels if not specified
	if len(config.Defaults.PRLabels) == 0 {
		config.Defaults.PRLabels = []string{"automated-sync"}
	}

	// Apply defaults to each mapping
	for i := range config.Mappings {
		mapping := &config.Mappings[i]

		// Set default source branch if not specified
		if mapping.Source.Branch == "" {
			mapping.Source.Branch = "main"
		}

		// Apply directory defaults for all targets in this mapping
		for j := range mapping.Targets {
			for k := range mapping.Targets[j].Directories {
				ApplyDirectoryDefaults(&mapping.Targets[j].Directories[k])
			}
		}
	}
}
