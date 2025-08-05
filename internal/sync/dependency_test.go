package sync

import (
	"strings"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDependencyResolver(t *testing.T) {
	resolver := NewDependencyResolver(nil)
	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.groups)
	assert.NotNil(t, resolver.dependencies)
	assert.NotNil(t, resolver.logger)
}

func TestDependencyResolver_AddGroup(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	group := config.Group{
		ID:        "test-group",
		Name:      "Test Group",
		DependsOn: []string{"dep-1", "dep-2"},
	}

	resolver.AddGroup(group)

	assert.Contains(t, resolver.groups, "test-group")
	assert.Equal(t, group, resolver.groups["test-group"])
	assert.Equal(t, []string{"dep-1", "dep-2"}, resolver.dependencies["test-group"])
}

func TestDependencyResolver_Resolve_NoDependencies(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	groups := []config.Group{
		{ID: "group-1", Name: "Group 1", Priority: 2},
		{ID: "group-2", Name: "Group 2", Priority: 1},
		{ID: "group-3", Name: "Group 3", Priority: 3},
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	result, err := resolver.Resolve()
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Should be sorted by priority
	assert.Equal(t, "group-2", result[0].ID) // Priority 1
	assert.Equal(t, "group-1", result[1].ID) // Priority 2
	assert.Equal(t, "group-3", result[2].ID) // Priority 3
}

func TestDependencyResolver_Resolve_LinearDependencies(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	// Linear chain: group-3 -> group-2 -> group-1
	groups := []config.Group{
		{
			ID:        "group-3",
			Name:      "Group 3",
			Priority:  1, // Highest priority but depends on others
			DependsOn: []string{"group-2"},
		},
		{
			ID:        "group-2",
			Name:      "Group 2",
			Priority:  2,
			DependsOn: []string{"group-1"},
		},
		{
			ID:       "group-1",
			Name:     "Group 1",
			Priority: 3, // Lowest priority but no dependencies
		},
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	result, err := resolver.Resolve()
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Should respect dependencies regardless of priority
	assert.Equal(t, "group-1", result[0].ID)
	assert.Equal(t, "group-2", result[1].ID)
	assert.Equal(t, "group-3", result[2].ID)
}

func TestDependencyResolver_Resolve_ComplexDependencies(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	// Complex dependency graph:
	//   group-1 (no deps)
	//   group-2 (no deps)
	//   group-3 -> group-1
	//   group-4 -> group-2
	//   group-5 -> group-3, group-4
	groups := []config.Group{
		{ID: "group-1", Name: "Group 1", Priority: 2},
		{ID: "group-2", Name: "Group 2", Priority: 1},
		{ID: "group-3", Name: "Group 3", Priority: 3, DependsOn: []string{"group-1"}},
		{ID: "group-4", Name: "Group 4", Priority: 2, DependsOn: []string{"group-2"}},
		{ID: "group-5", Name: "Group 5", Priority: 1, DependsOn: []string{"group-3", "group-4"}},
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	result, err := resolver.Resolve()
	require.NoError(t, err)
	require.Len(t, result, 5)

	// Create a map of positions for easier validation
	positions := make(map[string]int)
	for i, g := range result {
		positions[g.ID] = i
	}

	// Verify dependencies are respected
	assert.Less(t, positions["group-1"], positions["group-3"]) // group-1 before group-3
	assert.Less(t, positions["group-2"], positions["group-4"]) // group-2 before group-4
	assert.Less(t, positions["group-3"], positions["group-5"]) // group-3 before group-5
	assert.Less(t, positions["group-4"], positions["group-5"]) // group-4 before group-5

	// Among groups with no dependencies, priority should be respected
	if positions["group-1"] < positions["group-3"] && positions["group-2"] < positions["group-4"] {
		// group-1 and group-2 have no dependencies, so group-2 (priority 1) should come before group-1 (priority 2)
		assert.Less(t, positions["group-2"], positions["group-1"])
	}
}

func TestDependencyResolver_ValidateDependencies_MissingDependency(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	groups := []config.Group{
		{ID: "group-1", Name: "Group 1"},
		{ID: "group-2", Name: "Group 2", DependsOn: []string{"group-3"}}, // group-3 doesn't exist
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	_, err := resolver.Resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent group")
	assert.Contains(t, err.Error(), "group-3")
}

func TestDependencyResolver_DetectCircularDependencies_SimpleCycle(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	// Simple cycle: group-1 -> group-2 -> group-1
	groups := []config.Group{
		{ID: "group-1", Name: "Group 1", DependsOn: []string{"group-2"}},
		{ID: "group-2", Name: "Group 2", DependsOn: []string{"group-1"}},
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	_, err := resolver.Resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestDependencyResolver_DetectCircularDependencies_ComplexCycle(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	// Complex cycle: group-1 -> group-2 -> group-3 -> group-1
	groups := []config.Group{
		{ID: "group-1", Name: "Group 1", DependsOn: []string{"group-2"}},
		{ID: "group-2", Name: "Group 2", DependsOn: []string{"group-3"}},
		{ID: "group-3", Name: "Group 3", DependsOn: []string{"group-1"}},
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	_, err := resolver.Resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestDependencyResolver_DetectCircularDependencies_SelfDependency(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	// Self-dependency
	groups := []config.Group{
		{ID: "group-1", Name: "Group 1", DependsOn: []string{"group-1"}},
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	_, err := resolver.Resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestDependencyResolver_TopologicalSort_DeterministicOrder(t *testing.T) {
	// Test that groups with the same priority level are sorted deterministically
	resolver := NewDependencyResolver(logrus.New())

	groups := []config.Group{
		{ID: "group-c", Name: "Group C", Priority: 1},
		{ID: "group-a", Name: "Group A", Priority: 1},
		{ID: "group-b", Name: "Group B", Priority: 1},
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	result, err := resolver.Resolve()
	require.NoError(t, err)
	require.Len(t, result, 3)

	// With same priority, should be sorted by ID alphabetically
	assert.Equal(t, "group-a", result[0].ID)
	assert.Equal(t, "group-b", result[1].ID)
	assert.Equal(t, "group-c", result[2].ID)
}

func TestDependencyResolver_GetDependencyGraph(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	groups := []config.Group{
		{ID: "group-1", Name: "Group 1", Priority: 1},
		{ID: "group-2", Name: "Group 2", Priority: 2, DependsOn: []string{"group-1"}},
		{ID: "group-3", Name: "Group 3", Priority: 3, DependsOn: []string{"group-1", "group-2"}},
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	graph := resolver.GetDependencyGraph()
	assert.Contains(t, graph, "Dependency Graph:")
	assert.Contains(t, graph, "group-1 (priority=1)")
	assert.Contains(t, graph, "group-2 (priority=2) → [group-1]")
	assert.Contains(t, graph, "group-3 (priority=3) → [group-1 group-2]")
}

func TestDependencyResolver_Resolve_EmptyGroups(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	result, err := resolver.Resolve()
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDependencyResolver_Resolve_SingleGroup(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	group := config.Group{
		ID:       "single-group",
		Name:     "Single Group",
		Priority: 1,
	}

	resolver.AddGroup(group)

	result, err := resolver.Resolve()
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "single-group", result[0].ID)
}

func TestDependencyResolver_Resolve_DiamondDependency(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	// Diamond dependency pattern:
	//      group-1
	//      /     \
	//   group-2  group-3
	//      \     /
	//      group-4
	groups := []config.Group{
		{ID: "group-1", Name: "Group 1", Priority: 1},
		{ID: "group-2", Name: "Group 2", Priority: 2, DependsOn: []string{"group-1"}},
		{ID: "group-3", Name: "Group 3", Priority: 2, DependsOn: []string{"group-1"}},
		{ID: "group-4", Name: "Group 4", Priority: 3, DependsOn: []string{"group-2", "group-3"}},
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	result, err := resolver.Resolve()
	require.NoError(t, err)
	require.Len(t, result, 4)

	// Create a map of positions for easier validation
	positions := make(map[string]int)
	for i, g := range result {
		positions[g.ID] = i
	}

	// Verify the diamond pattern is respected
	assert.Less(t, positions["group-1"], positions["group-2"])
	assert.Less(t, positions["group-1"], positions["group-3"])
	assert.Less(t, positions["group-2"], positions["group-4"])
	assert.Less(t, positions["group-3"], positions["group-4"])
}

func TestDependencyResolver_Resolve_PartialCycle(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	// Partial graph with a cycle in one part
	groups := []config.Group{
		{ID: "group-1", Name: "Group 1"},                                 // Independent
		{ID: "group-2", Name: "Group 2", DependsOn: []string{"group-3"}}, // Part of cycle
		{ID: "group-3", Name: "Group 3", DependsOn: []string{"group-2"}}, // Part of cycle
		{ID: "group-4", Name: "Group 4", DependsOn: []string{"group-1"}}, // Depends on independent
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	_, err := resolver.Resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
	// Even though group-1 and group-4 are not part of the cycle,
	// the presence of any cycle should fail the entire resolution
}

func TestDependencyResolver_MultipleDependenciesOnSameGroup(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	// Multiple groups depend on the same base group
	groups := []config.Group{
		{ID: "base", Name: "Base", Priority: 1},
		{ID: "app-1", Name: "App 1", Priority: 2, DependsOn: []string{"base"}},
		{ID: "app-2", Name: "App 2", Priority: 2, DependsOn: []string{"base"}},
		{ID: "app-3", Name: "App 3", Priority: 2, DependsOn: []string{"base"}},
	}

	for _, g := range groups {
		resolver.AddGroup(g)
	}

	result, err := resolver.Resolve()
	require.NoError(t, err)
	require.Len(t, result, 4)

	// Base should come first
	assert.Equal(t, "base", result[0].ID)

	// All apps should come after base
	appNames := []string{result[1].ID, result[2].ID, result[3].ID}
	assert.Contains(t, appNames, "app-1")
	assert.Contains(t, appNames, "app-2")
	assert.Contains(t, appNames, "app-3")
}

func TestDependencyResolver_DeepDependencyChain(t *testing.T) {
	resolver := NewDependencyResolver(logrus.New())

	// Create a deep chain of dependencies
	chainLength := 10
	groups := make([]config.Group, chainLength)

	for i := 0; i < chainLength; i++ {
		groups[i] = config.Group{
			ID:       strings.Join([]string{"group", string(rune('0' + i))}, "-"),
			Name:     strings.Join([]string{"Group", string(rune('0' + i))}, " "),
			Priority: i + 1,
		}
		if i > 0 {
			groups[i].DependsOn = []string{groups[i-1].ID}
		}
	}

	// Add in reverse order to test that topological sort works regardless of input order
	for i := len(groups) - 1; i >= 0; i-- {
		resolver.AddGroup(groups[i])
	}

	result, err := resolver.Resolve()
	require.NoError(t, err)
	require.Len(t, result, chainLength)

	// Verify the chain is in correct order
	for i := 0; i < chainLength; i++ {
		assert.Equal(t, groups[i].ID, result[i].ID)
	}
}
