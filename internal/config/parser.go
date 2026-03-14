package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

const (
	// configLoadMaxRetries is the maximum number of attempts to load the config file
	configLoadMaxRetries = 2
	// configLoadRetryDelay is the delay between retry attempts
	configLoadRetryDelay = 100 * time.Millisecond
)

// Load reads and parses a configuration file from the given path.
// It includes retry logic for transient I/O errors (e.g., file being
// modified by an editor during read).
func Load(path string) (*Config, error) {
	// Initialize audit logger for security event tracking
	auditLogger := logging.NewAuditLogger()

	var lastErr error
	for attempt := 1; attempt <= configLoadMaxRetries; attempt++ {
		cfg, err := loadOnce(path, auditLogger)
		if err == nil {
			if attempt > 1 {
				auditLogger.LogConfigChange("system", "config_loaded_after_retry", path)
			}
			return cfg, nil
		}

		lastErr = err

		// Don't retry semantic/validation errors - only I/O and parsing errors
		if !isTransientConfigError(err) {
			return nil, err
		}

		// Don't wait after last attempt
		if attempt < configLoadMaxRetries {
			time.Sleep(configLoadRetryDelay)
		}
	}

	return nil, lastErr
}

// loadOnce performs a single attempt to load and parse the config file
func loadOnce(path string, auditLogger *logging.AuditLogger) (*Config, error) {
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

// isTransientConfigError determines if an error is likely transient and worth retrying.
// Semantic errors (like missing list references) are not retried as they require config changes.
func isTransientConfigError(err error) bool {
	if err == nil {
		return false
	}

	// Don't retry semantic/validation errors - these require config file changes
	if errors.Is(err, ErrListReferenceNotFound) ||
		errors.Is(err, ErrDuplicateListID) ||
		errors.Is(err, ErrDuplicateTarget) ||
		errors.Is(err, ErrNoTargets) ||
		errors.Is(err, ErrNoMappings) ||
		errors.Is(err, ErrPathTraversal) {
		return false
	}

	// Retry I/O and parsing errors (could be transient file access issues)
	return true
}

// formatListNotFoundError creates a detailed error message when a list reference is not found.
// It includes available lists and hints if the reference exists in the wrong list type.
func formatListNotFoundError(listType, ref, groupID, targetRepo string, fileLists map[string]*FileList, directoryLists map[string]*DirectoryList) string {
	var hint string

	// Check if the reference exists in the wrong list type
	if listType == "file" {
		if _, existsInDir := directoryLists[ref]; existsInDir {
			hint = fmt.Sprintf(" (note: '%s' exists as a directory_list, did you mean to use directory_list_refs?)", ref)
		}
	} else {
		if _, existsInFile := fileLists[ref]; existsInFile {
			hint = fmt.Sprintf(" (note: '%s' exists as a file_list, did you mean to use file_list_refs?)", ref)
		}
	}

	// Collect available list IDs
	var availableIDs []string
	if listType == "file" {
		availableIDs = make([]string, 0, len(fileLists))
		for id := range fileLists {
			availableIDs = append(availableIDs, id)
		}
	} else {
		availableIDs = make([]string, 0, len(directoryLists))
		for id := range directoryLists {
			availableIDs = append(availableIDs, id)
		}
	}
	sort.Strings(availableIDs)

	return fmt.Sprintf("%s_list '%s' not found (group: %s, target: %s)%s; available %s_lists: %v",
		listType, ref, groupID, targetRepo, hint, listType, availableIDs)
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

		// Set default blob size limit for source
		if group.Source.BlobSizeLimit == "" {
			group.Source.BlobSizeLimit = DefaultBlobSizeLimit
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
			// Set default blob size limit for targets (inherits from source if not set)
			if group.Targets[j].BlobSizeLimit == "" {
				group.Targets[j].BlobSizeLimit = group.Source.BlobSizeLimit
			}

			for k := range group.Targets[j].Directories {
				ApplyDirectoryDefaults(&group.Targets[j].Directories[k])
			}
		}
	}
}

// deepCopyTransform creates a deep copy of a Transform struct,
// including the Variables map to avoid shared mutable state.
func deepCopyTransform(t Transform) Transform {
	result := Transform{
		RepoName: t.RepoName,
	}
	if t.Variables != nil {
		result.Variables = make(map[string]string, len(t.Variables))
		for k, v := range t.Variables {
			result.Variables[k] = v
		}
	}
	return result
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
						return fmt.Errorf("%w: %s", ErrListReferenceNotFound,
							formatListNotFoundError("file", ref, group.ID, target.Repo, fileLists, directoryLists))
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

				// Sort for deterministic order (map iteration is non-deterministic)
				sort.Slice(resolvedFiles, func(i, j int) bool {
					return resolvedFiles[i].Dest < resolvedFiles[j].Dest
				})

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
						return fmt.Errorf("%w: %s", ErrListReferenceNotFound,
							formatListNotFoundError("directory", ref, group.ID, target.Repo, fileLists, directoryLists))
					}
					// Add or override directory mappings from the list
					for _, dir := range list.Directories {
						// Create a new directory mapping with deep copy of all fields
						dirCopy := DirectoryMapping{
							Src:               dir.Src,
							Dest:              dir.Dest,
							Exclude:           append([]string(nil), dir.Exclude...),
							IncludeOnly:       append([]string(nil), dir.IncludeOnly...),
							Transform:         deepCopyTransform(dir.Transform),
							PreserveStructure: dir.PreserveStructure,
							IncludeHidden:     dir.IncludeHidden,
							Delete:            dir.Delete,
						}

						// Deep copy module config if present, including CheckTags pointer
						if dir.Module != nil {
							moduleCopy := *dir.Module
							if dir.Module.CheckTags != nil {
								checkTags := *dir.Module.CheckTags
								moduleCopy.CheckTags = &checkTags
							}
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

				// Sort for deterministic order (map iteration is non-deterministic)
				sort.Slice(resolvedDirs, func(i, j int) bool {
					return resolvedDirs[i].Dest < resolvedDirs[j].Dest
				})

				target.Directories = resolvedDirs
			}
		}
	}

	return nil
}
