package sync

import (
	"errors"
	"fmt"
	"sort"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/sirupsen/logrus"
)

// Static errors for dependency resolution
var (
	ErrNonExistentDependency = errors.New("group depends on non-existent group")
	ErrCircularDependency    = errors.New("circular dependency detected")
	ErrTopologicalSort       = errors.New("topological sort failed")
)

// DependencyResolver resolves group dependencies and determines execution order
type DependencyResolver struct {
	groups       map[string]config.Group // Map of group ID to group
	dependencies map[string][]string     // Map of group ID to its dependencies
	logger       *logrus.Logger
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(logger *logrus.Logger) *DependencyResolver {
	if logger == nil {
		logger = logrus.New()
	}
	return &DependencyResolver{
		groups:       make(map[string]config.Group),
		dependencies: make(map[string][]string),
		logger:       logger,
	}
}

// AddGroup adds a group to the resolver
func (r *DependencyResolver) AddGroup(group config.Group) {
	r.groups[group.ID] = group
	r.dependencies[group.ID] = group.DependsOn
}

// Resolve performs dependency resolution and returns execution order
func (r *DependencyResolver) Resolve() ([]config.Group, error) {
	// First, validate all dependencies exist
	if err := r.validateDependencies(); err != nil {
		return nil, err
	}

	// Check for circular dependencies
	if err := r.detectCircularDependencies(); err != nil {
		return nil, err
	}

	// Perform topological sort to get execution order
	executionOrder, err := r.topologicalSort()
	if err != nil {
		return nil, err
	}

	// Within groups at the same dependency level, sort by priority
	r.sortByPriority(executionOrder)

	return executionOrder, nil
}

// validateDependencies ensures all referenced dependencies exist
func (r *DependencyResolver) validateDependencies() error {
	for groupID, deps := range r.dependencies {
		for _, depID := range deps {
			if _, exists := r.groups[depID]; !exists {
				return fmt.Errorf("%w: group %q depends on non-existent group %q", ErrNonExistentDependency, groupID, depID)
			}
		}
	}
	return nil
}

// detectCircularDependencies uses DFS to detect cycles
func (r *DependencyResolver) detectCircularDependencies() error {
	// Track visit states: 0 = unvisited, 1 = visiting, 2 = visited
	visitState := make(map[string]int)

	// Track the path for better error reporting
	var path []string

	// DFS function to detect cycles
	var dfs func(groupID string) error
	dfs = func(groupID string) error {
		visitState[groupID] = 1 // Mark as visiting
		path = append(path, groupID)

		for _, depID := range r.dependencies[groupID] {
			switch visitState[depID] {
			case 1: // Currently visiting - cycle detected!
				// Find where the cycle starts
				cycleStart := -1
				for i, id := range path {
					if id == depID {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cyclePath := append(path[cycleStart:], depID)
					return fmt.Errorf("%w: %v", ErrCircularDependency, cyclePath)
				}
				return fmt.Errorf("%w involving group %q", ErrCircularDependency, depID)
			case 0: // Unvisited
				if err := dfs(depID); err != nil {
					return err
				}
			}
			// case 2: Already visited, skip
		}

		visitState[groupID] = 2   // Mark as visited
		path = path[:len(path)-1] // Remove from path
		return nil
	}

	// Check all groups
	for groupID := range r.groups {
		if visitState[groupID] == 0 {
			if err := dfs(groupID); err != nil {
				return err
			}
		}
	}

	return nil
}

// topologicalSort performs Kahn's algorithm for topological sorting
func (r *DependencyResolver) topologicalSort() ([]config.Group, error) {
	// Calculate in-degree for each node
	// In-degree is the number of groups that depend on this group
	inDegree := make(map[string]int)
	for groupID := range r.groups {
		inDegree[groupID] = 0
	}

	// For each group that has dependencies, increase the in-degree of the current group
	// (not the dependencies themselves)
	for groupID, deps := range r.dependencies {
		// This group depends on deps, so increase this group's in-degree by the number of its dependencies
		inDegree[groupID] = len(deps)
	}

	// Find all nodes with in-degree 0
	var queue []string
	for groupID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, groupID)
		}
	}

	// Sort initial queue by priority for deterministic ordering
	sort.Slice(queue, func(i, j int) bool {
		gi := r.groups[queue[i]]
		gj := r.groups[queue[j]]
		if gi.Priority != gj.Priority {
			return gi.Priority < gj.Priority
		}
		// If same priority, sort by ID for deterministic order
		return gi.ID < gj.ID
	})

	// Process queue
	var result []config.Group
	processedCount := 0

	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]

		// Add to result
		result = append(result, r.groups[current])
		processedCount++

		// For each group that depends on the current group, reduce its in-degree
		for groupID, deps := range r.dependencies {
			for _, depID := range deps {
				if depID == current {
					// groupID depends on current, and current is now processed
					inDegree[groupID]--
					if inDegree[groupID] == 0 {
						// All dependencies of groupID are now processed
						queue = append(queue, groupID)
					}
				}
			}
		}

		// Sort queue by priority for next iteration
		if len(queue) > 1 {
			sort.Slice(queue, func(i, j int) bool {
				gi := r.groups[queue[i]]
				gj := r.groups[queue[j]]
				if gi.Priority != gj.Priority {
					return gi.Priority < gj.Priority
				}
				return gi.ID < gj.ID
			})
		}
	}

	// Check if all nodes were processed (should be true if no cycles)
	if processedCount != len(r.groups) {
		return nil, fmt.Errorf("%w: processed %d groups out of %d",
			ErrTopologicalSort, processedCount, len(r.groups))
	}

	return result, nil
}

// sortByPriority sorts groups by priority within dependency levels
func (r *DependencyResolver) sortByPriority(groups []config.Group) {
	// Groups are already sorted by priority within dependency levels
	// due to the priority-aware queue processing in topologicalSort
	// This method is kept for clarity and potential future enhancements

	r.logger.WithField("group_count", len(groups)).Debug("Execution order determined")
	for i, group := range groups {
		r.logger.WithFields(logrus.Fields{
			"order":      i + 1,
			"group_id":   group.ID,
			"group_name": group.Name,
			"priority":   group.Priority,
			"depends_on": group.DependsOn,
		}).Debug("Group execution order")
	}
}

// GetDependencyGraph returns a string representation of the dependency graph (for debugging)
func (r *DependencyResolver) GetDependencyGraph() string {
	result := "Dependency Graph:\n"
	for groupID, deps := range r.dependencies {
		group := r.groups[groupID]
		result += fmt.Sprintf("  %s (priority=%d)", groupID, group.Priority)
		if len(deps) > 0 {
			result += fmt.Sprintf(" → %v", deps)
		}
		result += "\n"
	}
	return result
}
