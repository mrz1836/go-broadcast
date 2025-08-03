package checks

import (
	"sync"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/checks/builtin"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/checks/makewrap"
)

// Registry manages all available checks
type Registry struct {
	checks map[string]Check
	mu     sync.RWMutex
}

// NewRegistry creates a new check registry with all built-in checks
func NewRegistry() *Registry {
	r := &Registry{
		checks: make(map[string]Check),
	}

	// Register built-in checks
	r.Register(builtin.NewWhitespaceCheck())
	r.Register(builtin.NewEOFCheck())

	// Register make wrapper checks
	r.Register(makewrap.NewFumptCheck())
	r.Register(makewrap.NewLintCheck())
	r.Register(makewrap.NewModTidyCheck())

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
