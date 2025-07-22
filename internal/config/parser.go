package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a configuration file from the given path
func Load(path string) (*Config, error) {
	file, err := os.Open(path) //#nosec G304 -- Path is user-provided config file
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() { _ = file.Close() }()

	return LoadFromReader(file)
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
	// Set default source branch if not specified
	if config.Source.Branch == "" {
		config.Source.Branch = "master"
	}

	// Set default branch prefix if not specified
	if config.Defaults.BranchPrefix == "" {
		config.Defaults.BranchPrefix = "sync/template"
	}

	// Set default PR labels if not specified
	if len(config.Defaults.PRLabels) == 0 {
		config.Defaults.PRLabels = []string{"automated-sync"}
	}
}
