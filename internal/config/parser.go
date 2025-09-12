package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/mrz1836/go-broadcast/internal/logging"
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

	// Resolve list references
	if err := resolveListReferences(config); err != nil {
		return nil, fmt.Errorf("failed to resolve list references: %w", err)
	}

	return config, nil
}

// applyDefaults sets default values for optional fields in group-based configuration
func applyDefaults(config *Config) {
	// Apply defaults to all groups
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

// resolveListReferences expands file and directory list references in targets
func resolveListReferences(config *Config) error {
	// Build lookup maps for efficient resolution
	fileLists := make(map[string]*FileList)
	for i := range config.FileLists {
		list := &config.FileLists[i]
		if _, exists := fileLists[list.ID]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicateListID, list.ID)
		}
		fileLists[list.ID] = list
	}

	directoryLists := make(map[string]*DirectoryList)
	for i := range config.DirectoryLists {
		list := &config.DirectoryLists[i]
		if _, exists := directoryLists[list.ID]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicateListID, list.ID)
		}
		directoryLists[list.ID] = list
	}

	// Resolve references for each target in each group
	for i := range config.Groups {
		group := &config.Groups[i]
		for j := range group.Targets {
			target := &group.Targets[j]

			// Resolve file list references
			if len(target.FileListRefs) > 0 {
				// Use a map to track files by destination (later entries override)
				fileMap := make(map[string]FileMapping)

				// Add files from referenced lists (later lists override earlier ones)
				for _, ref := range target.FileListRefs {
					list, exists := fileLists[ref]
					if !exists {
						return fmt.Errorf("%w: file list '%s' (group: %s, target: %s)",
							ErrListReferenceNotFound, ref, group.ID, target.Repo)
					}
					// Add or override file mappings from the list
					for _, file := range list.Files {
						fileMap[file.Dest] = FileMapping{
							Src:    file.Src,
							Dest:   file.Dest,
							Delete: file.Delete,
						}
					}
				}

				// Add inline files, which override list files with same destination
				for _, file := range target.Files {
					fileMap[file.Dest] = file
				}

				// Convert map back to slice
				resolvedFiles := make([]FileMapping, 0, len(fileMap))
				for _, file := range fileMap {
					resolvedFiles = append(resolvedFiles, file)
				}

				target.Files = resolvedFiles
			}

			// Resolve directory list references
			if len(target.DirectoryListRefs) > 0 {
				// Use a map to track directories by destination (later entries override)
				dirMap := make(map[string]DirectoryMapping)

				// Add directories from referenced lists (later lists override earlier ones)
				for _, ref := range target.DirectoryListRefs {
					list, exists := directoryLists[ref]
					if !exists {
						return fmt.Errorf("%w: directory list '%s' (group: %s, target: %s)",
							ErrListReferenceNotFound, ref, group.ID, target.Repo)
					}
					// Add or override directory mappings from the list
					for _, dir := range list.Directories {
						// Create a new directory mapping with deep copy of all fields
						dirCopy := DirectoryMapping{
							Src:               dir.Src,
							Dest:              dir.Dest,
							Exclude:           append([]string(nil), dir.Exclude...),
							IncludeOnly:       append([]string(nil), dir.IncludeOnly...),
							Transform:         dir.Transform,
							PreserveStructure: dir.PreserveStructure,
							IncludeHidden:     dir.IncludeHidden,
							Delete:            dir.Delete,
						}

						// Deep copy module config if present
						if dir.Module != nil {
							moduleCopy := *dir.Module
							dirCopy.Module = &moduleCopy
						}

						dirMap[dir.Dest] = dirCopy
					}
				}

				// Add inline directories, which override list directories with same destination
				for _, dir := range target.Directories {
					dirMap[dir.Dest] = dir
				}

				// Convert map back to slice
				resolvedDirs := make([]DirectoryMapping, 0, len(dirMap))
				for _, dir := range dirMap {
					resolvedDirs = append(resolvedDirs, dir)
				}

				target.Directories = resolvedDirs
			}
		}
	}

	return nil
}
