package checks

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/checks/builtin"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/checks/makewrap"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/shared"
)

// Registry manages all available checks
type Registry struct {
	checks    map[string]Check
	mu        sync.RWMutex
	sharedCtx *shared.Context
}

// NewRegistry creates a new check registry with all built-in checks
func NewRegistry() *Registry {
	r := &Registry{
		checks:    make(map[string]Check),
		sharedCtx: shared.NewContext(),
	}

	// Register built-in checks
	r.Register(builtin.NewWhitespaceCheck())
	r.Register(builtin.NewEOFCheck())

	// Register make wrapper checks with shared context
	r.Register(makewrap.NewFumptCheckWithSharedContext(r.sharedCtx))
	r.Register(makewrap.NewLintCheckWithSharedContext(r.sharedCtx))
	r.Register(makewrap.NewModTidyCheckWithSharedContext(r.sharedCtx))

	return r
}

// NewRegistryWithConfig creates a new check registry with configuration-based timeouts
func NewRegistryWithConfig(cfg *config.Config) *Registry {
	if cfg == nil {
		// Return an empty registry for nil config instead of nil
		return &Registry{
			checks:    make(map[string]Check),
			sharedCtx: shared.NewContext(),
		}
	}
	r := &Registry{
		checks:    make(map[string]Check),
		sharedCtx: shared.NewContext(),
	}

	// Register built-in checks with full config
	r.Register(builtin.NewWhitespaceCheckWithConfig(cfg))
	r.Register(builtin.NewEOFCheckWithTimeout(time.Duration(cfg.CheckTimeouts.EOF) * time.Second))

	// Register make wrapper checks with shared context and timeouts
	r.Register(makewrap.NewFumptCheckWithConfig(r.sharedCtx, time.Duration(cfg.CheckTimeouts.Fumpt)*time.Second))
	r.Register(makewrap.NewLintCheckWithConfig(r.sharedCtx, time.Duration(cfg.CheckTimeouts.Lint)*time.Second))
	r.Register(makewrap.NewModTidyCheckWithConfig(r.sharedCtx, time.Duration(cfg.CheckTimeouts.ModTidy)*time.Second))

	return r
}

// Register adds a check to the registry
func (r *Registry) Register(check Check) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[check.Name()] = check
}

// Get returns a check by name
func (r *Registry) Get(name string) (Check, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	check, ok := r.checks[name]
	return check, ok
}

// GetChecks returns all registered checks
func (r *Registry) GetChecks() []Check {
	r.mu.RLock()
	defer r.mu.RUnlock()

	checks := make([]Check, 0, len(r.checks))
	for _, check := range r.checks {
		checks = append(checks, check)
	}
	return checks
}

// Names returns the names of all registered checks
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.checks))
	for name := range r.checks {
		names = append(names, name)
	}
	return names
}

// GetMetadata returns metadata for a specific check
func (r *Registry) GetMetadata(name string) (CheckMetadata, bool) {
	check, ok := r.Get(name)
	if !ok {
		return CheckMetadata{}, false
	}
	return r.convertMetadata(check.Metadata()), true
}

// GetAllMetadata returns metadata for all registered checks
func (r *Registry) GetAllMetadata() []CheckMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata := make([]CheckMetadata, 0, len(r.checks))
	for _, check := range r.checks {
		metadata = append(metadata, r.convertMetadata(check.Metadata()))
	}
	return metadata
}

// GetChecksByCategory returns all checks in a specific category
func (r *Registry) GetChecksByCategory(category string) []Check {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var checks []Check
	for _, check := range r.checks {
		metadata := r.convertMetadata(check.Metadata())
		if metadata.Category == category {
			checks = append(checks, check)
		}
	}
	return checks
}

// ValidateCheckDependencies validates that all check dependencies are satisfied
func (r *Registry) ValidateCheckDependencies(ctx context.Context) []error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errors []error
	for _, check := range r.checks {
		metadata := r.convertMetadata(check.Metadata())
		for _, dependency := range metadata.Dependencies {
			// Check if make target exists
			if !r.sharedCtx.HasMakeTarget(ctx, dependency) {
				errors = append(errors, &CheckDependencyError{
					CheckName:      metadata.Name,
					Dependency:     dependency,
					DependencyType: "make_target",
				})
			}
		}
	}
	return errors
}

// CheckDependencyError represents an error when a check dependency is not satisfied
type CheckDependencyError struct {
	CheckName      string
	Dependency     string
	DependencyType string
}

// Error implements the error interface
func (e *CheckDependencyError) Error() string {
	return fmt.Sprintf("check '%s' requires %s '%s' which is not available",
		e.CheckName, e.DependencyType, e.Dependency)
}

// GetEstimatedDuration returns the total estimated duration for running all checks
func (r *Registry) GetEstimatedDuration() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var total time.Duration
	for _, check := range r.checks {
		metadata := r.convertMetadata(check.Metadata())
		total += metadata.EstimatedDuration
	}
	return total
}

// convertMetadata converts from the specific metadata type to the generic CheckMetadata
// This is needed to handle different metadata types from different packages
func (r *Registry) convertMetadata(checkMetadata interface{}) CheckMetadata {
	// Use type assertion to handle different metadata types
	switch metadata := checkMetadata.(type) {
	case CheckMetadata:
		return metadata
	default:
		// Use reflection to convert the metadata
		return r.convertMetadataReflection(metadata)
	}
}

// convertMetadataReflection uses reflection to convert metadata types
func (r *Registry) convertMetadataReflection(checkMetadata interface{}) CheckMetadata {
	val := reflect.ValueOf(checkMetadata)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return CheckMetadata{
			Name:              "unknown",
			Description:       "Unknown check",
			FilePatterns:      []string{},
			EstimatedDuration: 30 * time.Second,
			Dependencies:      []string{},
			DefaultTimeout:    30 * time.Second,
			Category:          "unknown",
			RequiresFiles:     true,
		}
	}

	result := CheckMetadata{}

	// Extract fields using reflection
	if field := val.FieldByName("Name"); field.IsValid() && field.Kind() == reflect.String {
		result.Name = field.String()
	}

	if field := val.FieldByName("Description"); field.IsValid() && field.Kind() == reflect.String {
		result.Description = field.String()
	}

	if field := val.FieldByName("FilePatterns"); field.IsValid() && field.Kind() == reflect.Slice {
		patterns := make([]string, field.Len())
		for i := 0; i < field.Len(); i++ {
			if pattern := field.Index(i); pattern.Kind() == reflect.String {
				patterns[i] = pattern.String()
			}
		}
		result.FilePatterns = patterns
	}

	if field := val.FieldByName("EstimatedDuration"); field.IsValid() {
		if duration, ok := field.Interface().(time.Duration); ok {
			result.EstimatedDuration = duration
		}
	}

	if field := val.FieldByName("Dependencies"); field.IsValid() && field.Kind() == reflect.Slice {
		deps := make([]string, field.Len())
		for i := 0; i < field.Len(); i++ {
			if dep := field.Index(i); dep.Kind() == reflect.String {
				deps[i] = dep.String()
			}
		}
		result.Dependencies = deps
	}

	if field := val.FieldByName("DefaultTimeout"); field.IsValid() {
		if duration, ok := field.Interface().(time.Duration); ok {
			result.DefaultTimeout = duration
		}
	}

	if field := val.FieldByName("Category"); field.IsValid() && field.Kind() == reflect.String {
		result.Category = field.String()
	}

	if field := val.FieldByName("RequiresFiles"); field.IsValid() && field.Kind() == reflect.Bool {
		result.RequiresFiles = field.Bool()
	}

	return result
}
