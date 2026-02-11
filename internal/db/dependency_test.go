package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyGraph_DetectCycles_NoCycle(t *testing.T) {
	// Create a simple linear dependency: A -> B -> C
	groups := []*Group{
		{ExternalID: "group-a", Dependencies: []GroupDependency{}},
		{ExternalID: "group-b", Dependencies: []GroupDependency{{DependsOnID: "group-a"}}},
		{ExternalID: "group-c", Dependencies: []GroupDependency{{DependsOnID: "group-b"}}},
	}

	dg := NewDependencyGraph(groups)
	err := dg.DetectCycles()
	require.NoError(t, err)
}

func TestDependencyGraph_DetectCycles_SimpleCycle(t *testing.T) {
	// Create a simple cycle: A -> B -> A
	groups := []*Group{
		{ExternalID: "group-a", Dependencies: []GroupDependency{{DependsOnID: "group-b"}}},
		{ExternalID: "group-b", Dependencies: []GroupDependency{{DependsOnID: "group-a"}}},
	}

	dg := NewDependencyGraph(groups)
	err := dg.DetectCycles()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrCircularDependency)
	assert.Contains(t, err.Error(), "group-a")
	assert.Contains(t, err.Error(), "group-b")
}

func TestDependencyGraph_DetectCycles_ComplexCycle(t *testing.T) {
	// Create a complex cycle: A -> B -> C -> D -> B
	groups := []*Group{
		{ExternalID: "group-a", Dependencies: []GroupDependency{{DependsOnID: "group-b"}}},
		{ExternalID: "group-b", Dependencies: []GroupDependency{{DependsOnID: "group-c"}}},
		{ExternalID: "group-c", Dependencies: []GroupDependency{{DependsOnID: "group-d"}}},
		{ExternalID: "group-d", Dependencies: []GroupDependency{{DependsOnID: "group-b"}}},
	}

	dg := NewDependencyGraph(groups)
	err := dg.DetectCycles()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircularDependency)
}

func TestDependencyGraph_DetectCycles_SelfReference(t *testing.T) {
	// Create a self-referencing group: A -> A
	groups := []*Group{
		{ExternalID: "group-a", Dependencies: []GroupDependency{{DependsOnID: "group-a"}}},
	}

	dg := NewDependencyGraph(groups)
	err := dg.DetectCycles()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrCircularDependency)
	assert.Contains(t, err.Error(), "group-a")
}

func TestDependencyGraph_DetectCycles_MultipleDependencies(t *testing.T) {
	// Create a diamond pattern: A -> B, A -> C, B -> D, C -> D (no cycle)
	groups := []*Group{
		{ExternalID: "group-a", Dependencies: []GroupDependency{
			{DependsOnID: "group-b"},
			{DependsOnID: "group-c"},
		}},
		{ExternalID: "group-b", Dependencies: []GroupDependency{{DependsOnID: "group-d"}}},
		{ExternalID: "group-c", Dependencies: []GroupDependency{{DependsOnID: "group-d"}}},
		{ExternalID: "group-d", Dependencies: []GroupDependency{}},
	}

	dg := NewDependencyGraph(groups)
	err := dg.DetectCycles()
	require.NoError(t, err)
}

func TestDependencyGraph_DetectCycles_Empty(t *testing.T) {
	// Empty graph
	groups := []*Group{}

	dg := NewDependencyGraph(groups)
	err := dg.DetectCycles()
	require.NoError(t, err)
}

func TestDependencyGraph_DetectCycles_NoDependencies(t *testing.T) {
	// Groups with no dependencies
	groups := []*Group{
		{ExternalID: "group-a", Dependencies: []GroupDependency{}},
		{ExternalID: "group-b", Dependencies: []GroupDependency{}},
		{ExternalID: "group-c", Dependencies: []GroupDependency{}},
	}

	dg := NewDependencyGraph(groups)
	err := dg.DetectCycles()
	require.NoError(t, err)
}

func TestDependencyGraph_TopologicalSort_Simple(t *testing.T) {
	// Create a simple dependency chain: C -> B -> A
	groups := []*Group{
		{ExternalID: "group-a", Dependencies: []GroupDependency{}},
		{ExternalID: "group-b", Dependencies: []GroupDependency{{DependsOnID: "group-a"}}},
		{ExternalID: "group-c", Dependencies: []GroupDependency{{DependsOnID: "group-b"}}},
	}

	dg := NewDependencyGraph(groups)
	sorted, err := dg.TopologicalSort()
	require.NoError(t, err)
	require.Len(t, sorted, 3)

	// A should come before B, B should come before C
	var aIdx, bIdx, cIdx int
	for i, g := range sorted {
		switch g.ExternalID {
		case "group-a":
			aIdx = i
		case "group-b":
			bIdx = i
		case "group-c":
			cIdx = i
		}
	}
	assert.Less(t, aIdx, bIdx, "group-a should come before group-b")
	assert.Less(t, bIdx, cIdx, "group-b should come before group-c")
}

func TestDependencyGraph_TopologicalSort_Diamond(t *testing.T) {
	// Diamond pattern: D is root, B and C depend on D, A depends on both B and C
	groups := []*Group{
		{ExternalID: "group-a", Dependencies: []GroupDependency{
			{DependsOnID: "group-b"},
			{DependsOnID: "group-c"},
		}},
		{ExternalID: "group-b", Dependencies: []GroupDependency{{DependsOnID: "group-d"}}},
		{ExternalID: "group-c", Dependencies: []GroupDependency{{DependsOnID: "group-d"}}},
		{ExternalID: "group-d", Dependencies: []GroupDependency{}},
	}

	dg := NewDependencyGraph(groups)
	sorted, err := dg.TopologicalSort()
	require.NoError(t, err)
	require.Len(t, sorted, 4)

	// D should come first (or near first), A should come last
	var aIdx, dIdx int
	for i, g := range sorted {
		switch g.ExternalID {
		case "group-a":
			aIdx = i
		case "group-d":
			dIdx = i
		}
	}
	assert.Less(t, dIdx, aIdx, "group-d should come before group-a")
}

func TestDependencyGraph_TopologicalSort_WithCycle(t *testing.T) {
	// Create a cycle: A -> B -> A
	groups := []*Group{
		{ExternalID: "group-a", Dependencies: []GroupDependency{{DependsOnID: "group-b"}}},
		{ExternalID: "group-b", Dependencies: []GroupDependency{{DependsOnID: "group-a"}}},
	}

	dg := NewDependencyGraph(groups)
	_, err := dg.TopologicalSort()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircularDependency)
}

func TestValidateGroupDependencies_Valid(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create a config
	config := &Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	// Create groups with valid dependencies
	groupA := &Group{
		ConfigID:   config.ID,
		ExternalID: "group-a",
		Name:       "Group A",
	}
	err = db.Create(groupA).Error
	require.NoError(t, err)

	groupB := &Group{
		ConfigID:     config.ID,
		ExternalID:   "group-b",
		Name:         "Group B",
		Dependencies: []GroupDependency{{DependsOnID: "group-a"}},
	}
	err = db.Create(groupB).Error
	require.NoError(t, err)

	// Validate should pass
	err = ValidateGroupDependencies(ctx, db, config.ID)
	require.NoError(t, err)
}

func TestValidateGroupDependencies_NonExistentDependency(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create a config
	config := &Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	// Create group with non-existent dependency
	group := &Group{
		ConfigID:     config.ID,
		ExternalID:   "group-a",
		Name:         "Group A",
		Dependencies: []GroupDependency{{DependsOnID: "non-existent"}},
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	// Validate should fail
	err = ValidateGroupDependencies(ctx, db, config.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent")
}

func TestValidateGroupDependencies_CircularDependency(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create a config
	config := &Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	err := db.Create(config).Error
	require.NoError(t, err)

	// Create groups with circular dependency
	groupA := &Group{
		ConfigID:     config.ID,
		ExternalID:   "group-a",
		Name:         "Group A",
		Dependencies: []GroupDependency{{DependsOnID: "group-b"}},
	}
	err = db.Create(groupA).Error
	require.NoError(t, err)

	groupB := &Group{
		ConfigID:     config.ID,
		ExternalID:   "group-b",
		Name:         "Group B",
		Dependencies: []GroupDependency{{DependsOnID: "group-a"}},
	}
	err = db.Create(groupB).Error
	require.NoError(t, err)

	// Validate should fail
	err = ValidateGroupDependencies(ctx, db, config.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircularDependency)
}
