package db

import (
	"context"
	"fmt"
	"sort"

	"gorm.io/gorm"
)

// DependencyGraph represents a directed graph of group dependencies
type DependencyGraph struct {
	groups map[string]*Group           // external_id -> Group
	edges  map[string][]string          // external_id -> []depends_on_external_id
	inDegree map[string]int             // external_id -> number of incoming edges
}

// NewDependencyGraph creates a dependency graph from a list of groups
func NewDependencyGraph(groups []*Group) *DependencyGraph {
	dg := &DependencyGraph{
		groups:   make(map[string]*Group),
		edges:    make(map[string][]string),
		inDegree: make(map[string]int),
	}

	// Build the graph
	for _, group := range groups {
		dg.groups[group.ExternalID] = group
		dg.inDegree[group.ExternalID] = 0
		dg.edges[group.ExternalID] = []string{}
	}

	// Add edges
	// If group-b depends on group-a, create edge: group-a -> group-b
	// This ensures group-a is processed before group-b
	for _, group := range groups {
		for _, dep := range group.Dependencies {
			dg.edges[dep.DependsOnID] = append(dg.edges[dep.DependsOnID], group.ExternalID)
			dg.inDegree[group.ExternalID]++
		}
	}

	return dg
}

// DetectCycles uses Kahn's algorithm to detect circular dependencies
// Returns nil if no cycles, or an error with the cycle description
func (dg *DependencyGraph) DetectCycles() error {
	// Create a copy of inDegree for processing
	inDegree := make(map[string]int)
	for k, v := range dg.inDegree {
		inDegree[k] = v
	}

	// Queue for nodes with no incoming edges
	queue := []string{}
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	processed := 0
	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]
		processed++

		// Process all outgoing edges
		for _, neighbor := range dg.edges[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If we processed all nodes, there are no cycles
	if processed == len(dg.groups) {
		return nil
	}

	// Otherwise, find the cycle for a better error message
	cycle := dg.findCycle()
	if len(cycle) > 0 {
		return fmt.Errorf("%w: %v", ErrCircularDependency, cycle)
	}

	return ErrCircularDependency
}

// findCycle uses DFS to find a cycle in the graph
func (dg *DependencyGraph) findCycle() []string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	parent := make(map[string]string)

	var dfs func(string) []string
	dfs = func(node string) []string {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range dg.edges[node] {
			if !visited[neighbor] {
				parent[neighbor] = node
				if cycle := dfs(neighbor); cycle != nil {
					return cycle
				}
			} else if recStack[neighbor] {
				// Found a cycle - reconstruct it
				cycle := []string{neighbor}
				current := node
				for current != neighbor {
					cycle = append([]string{current}, cycle...)
					current = parent[current]
				}
				cycle = append([]string{neighbor}, cycle...)
				return cycle
			}
		}

		recStack[node] = false
		return nil
	}

	// Try DFS from each unvisited node
	for id := range dg.groups {
		if !visited[id] {
			if cycle := dfs(id); cycle != nil {
				return cycle
			}
		}
	}

	return nil
}

// TopologicalSort returns groups in dependency order (dependencies first)
// Returns an error if a circular dependency is detected
func (dg *DependencyGraph) TopologicalSort() ([]*Group, error) {
	// First check for cycles
	if err := dg.DetectCycles(); err != nil {
		return nil, err
	}

	// Create a copy of inDegree for processing
	inDegree := make(map[string]int)
	for k, v := range dg.inDegree {
		inDegree[k] = v
	}

	// Queue for nodes with no incoming edges
	queue := []string{}
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	result := []*Group{}
	for len(queue) > 0 {
		// Sort queue for deterministic ordering
		sort.Strings(queue)

		// Dequeue
		current := queue[0]
		queue = queue[1:]
		result = append(result, dg.groups[current])

		// Process all outgoing edges
		for _, neighbor := range dg.edges[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return result, nil
}

// ValidateGroupDependencies validates all group dependencies in the database
// Checks that all referenced groups exist and there are no circular dependencies
func ValidateGroupDependencies(ctx context.Context, db *gorm.DB, configID uint) error {
	// Load all groups for this config
	var groups []*Group
	if err := db.WithContext(ctx).
		Preload("Dependencies").
		Where("config_id = ?", configID).
		Find(&groups).Error; err != nil {
		return fmt.Errorf("failed to load groups: %w", err)
	}

	// Build a map of existing external IDs
	existingIDs := make(map[string]bool)
	for _, group := range groups {
		existingIDs[group.ExternalID] = true
	}

	// Validate all dependencies exist
	for _, group := range groups {
		for _, dep := range group.Dependencies {
			if !existingIDs[dep.DependsOnID] {
				return fmt.Errorf("group %s depends on non-existent group %s",
					group.ExternalID, dep.DependsOnID)
			}
		}
	}

	// Check for circular dependencies
	dg := NewDependencyGraph(groups)
	if err := dg.DetectCycles(); err != nil {
		return err
	}

	return nil
}
