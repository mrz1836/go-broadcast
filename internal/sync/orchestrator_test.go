package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Static errors for testing
var (
	ErrGroupFailed = errors.New("group failed")
	ErrTestError   = errors.New("test error")
)

// testGroupExecutor is a test helper to track group executions
type testGroupExecutor struct {
	executedGroups []string
	errorsToReturn map[string]error
}

func (e *testGroupExecutor) executeGroup(_ context.Context, group config.Group) error {
	e.executedGroups = append(e.executedGroups, group.ID)
	if err, exists := e.errorsToReturn[group.ID]; exists {
		return err
	}
	return nil
}

func TestNewGroupOrchestrator(t *testing.T) {
	cfg := &config.Config{Version: 1}
	engine := &Engine{config: cfg}
	logger := logrus.New()

	orch := NewGroupOrchestrator(cfg, engine, logger)

	assert.NotNil(t, orch)
	assert.Equal(t, cfg, orch.config)
	assert.Equal(t, engine, orch.engine)
	assert.Equal(t, logger, orch.logger)
	assert.NotNil(t, orch.groupStatus)
}

func TestGroupOrchestrator_ExecuteGroups_NoGroups(t *testing.T) {
	cfg := &config.Config{Version: 1}
	engine := &Engine{config: cfg}
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	orch := NewGroupOrchestrator(cfg, engine, logger)

	err := orch.ExecuteGroups(context.Background(), []config.Group{})
	assert.NoError(t, err)
}

func TestGroupOrchestrator_ExecuteGroups_SingleGroup(t *testing.T) {
	cfg := &config.Config{Version: 1}
	engine := &Engine{
		config: cfg,
		logger: logrus.New(),
	}

	orch := NewGroupOrchestrator(cfg, engine, logrus.New())

	// Use test executor to track execution
	executor := &testGroupExecutor{
		errorsToReturn: make(map[string]error),
	}

	// Replace executeGroup method for testing
	originalExecuteGroup := orch.executeGroup
	orch.executeGroup = executor.executeGroup
	defer func() { orch.executeGroup = originalExecuteGroup }()

	groups := []config.Group{
		{
			ID:       "test-group",
			Name:     "Test Group",
			Priority: 1,
			Enabled:  boolPtr(true),
			Source:   config.SourceConfig{Repo: "test/source"},
			Targets:  []config.TargetConfig{{Repo: "test/target"}},
		},
	}

	err := orch.ExecuteGroups(context.Background(), groups)
	require.NoError(t, err)
	assert.Contains(t, executor.executedGroups, "test-group")

	// Check status
	status, exists := orch.GetGroupStatusByID("test-group")
	assert.True(t, exists)
	assert.Equal(t, "success", status.State)
}

func TestGroupOrchestrator_ExecuteGroups_MultipleGroups(t *testing.T) {
	cfg := &config.Config{Version: 1}
	engine := &Engine{
		config: cfg,
		logger: logrus.New(),
	}

	orch := NewGroupOrchestrator(cfg, engine, logrus.New())

	// Use test executor to track execution
	executor := &testGroupExecutor{
		errorsToReturn: make(map[string]error),
	}
	orch.executeGroup = executor.executeGroup

	groups := []config.Group{
		{
			ID:       "group-2",
			Name:     "Group 2",
			Priority: 2, // Lower priority
			Enabled:  boolPtr(true),
			Source:   config.SourceConfig{Repo: "test/source"},
			Targets:  []config.TargetConfig{{Repo: "test/target"}},
		},
		{
			ID:       "group-1",
			Name:     "Group 1",
			Priority: 1, // Higher priority
			Enabled:  boolPtr(true),
			Source:   config.SourceConfig{Repo: "test/source"},
			Targets:  []config.TargetConfig{{Repo: "test/target"}},
		},
		{
			ID:       "group-3",
			Name:     "Group 3",
			Priority: 3, // Lowest priority
			Enabled:  boolPtr(true),
			Source:   config.SourceConfig{Repo: "test/source"},
			Targets:  []config.TargetConfig{{Repo: "test/target"}},
		},
	}

	err := orch.ExecuteGroups(context.Background(), groups)
	require.NoError(t, err)

	// Check execution order (should be by priority)
	assert.Equal(t, []string{"group-1", "group-2", "group-3"}, executor.executedGroups)

	// Check all groups succeeded
	for _, group := range groups {
		status, exists := orch.GetGroupStatusByID(group.ID)
		assert.True(t, exists)
		assert.Equal(t, "success", status.State)
	}
}

func TestGroupOrchestrator_ExecuteGroups_WithDependencies(t *testing.T) {
	cfg := &config.Config{Version: 1}
	engine := &Engine{
		config: cfg,
		logger: logrus.New(),
	}

	orch := NewGroupOrchestrator(cfg, engine, logrus.New())

	// Use test executor to track execution
	executor := &testGroupExecutor{
		errorsToReturn: make(map[string]error),
	}
	orch.executeGroup = executor.executeGroup

	groups := []config.Group{
		{
			ID:        "group-3",
			Name:      "Group 3",
			Priority:  1, // Highest priority but depends on group-2
			DependsOn: []string{"group-2"},
			Enabled:   boolPtr(true),
			Source:    config.SourceConfig{Repo: "test/source"},
			Targets:   []config.TargetConfig{{Repo: "test/target"}},
		},
		{
			ID:        "group-2",
			Name:      "Group 2",
			Priority:  2,
			DependsOn: []string{"group-1"},
			Enabled:   boolPtr(true),
			Source:    config.SourceConfig{Repo: "test/source"},
			Targets:   []config.TargetConfig{{Repo: "test/target"}},
		},
		{
			ID:       "group-1",
			Name:     "Group 1",
			Priority: 3, // Lowest priority but no dependencies
			Enabled:  boolPtr(true),
			Source:   config.SourceConfig{Repo: "test/source"},
			Targets:  []config.TargetConfig{{Repo: "test/target"}},
		},
	}

	err := orch.ExecuteGroups(context.Background(), groups)
	require.NoError(t, err)

	// Check execution order (should respect dependencies)
	assert.Equal(t, []string{"group-1", "group-2", "group-3"}, executor.executedGroups)
}

func TestGroupOrchestrator_ExecuteGroups_FailedDependency(t *testing.T) {
	cfg := &config.Config{Version: 1}
	engine := &Engine{
		config: cfg,
		logger: logrus.New(),
	}

	orch := NewGroupOrchestrator(cfg, engine, logrus.New())

	// Use test executor with error for group-1
	executor := &testGroupExecutor{
		errorsToReturn: map[string]error{
			"group-1": ErrGroupFailed,
		},
	}
	orch.executeGroup = executor.executeGroup

	groups := []config.Group{
		{
			ID:       "group-1",
			Name:     "Group 1",
			Priority: 1,
			Enabled:  boolPtr(true),
			Source:   config.SourceConfig{Repo: "test/source"},
			Targets:  []config.TargetConfig{{Repo: "test/target"}},
		},
		{
			ID:        "group-2",
			Name:      "Group 2",
			Priority:  2,
			DependsOn: []string{"group-1"}, // Depends on failing group
			Enabled:   boolPtr(true),
			Source:    config.SourceConfig{Repo: "test/source"},
			Targets:   []config.TargetConfig{{Repo: "test/target"}},
		},
		{
			ID:       "group-3",
			Name:     "Group 3",
			Priority: 3,
			Enabled:  boolPtr(true), // No dependencies, should execute
			Source:   config.SourceConfig{Repo: "test/source"},
			Targets:  []config.TargetConfig{{Repo: "test/target"}},
		},
	}

	err := orch.ExecuteGroups(context.Background(), groups)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 groups failed")

	// Check execution
	assert.Contains(t, executor.executedGroups, "group-1")
	assert.NotContains(t, executor.executedGroups, "group-2") // Should be skipped
	assert.Contains(t, executor.executedGroups, "group-3")    // Should execute (no deps)

	// Check statuses
	status1, _ := orch.GetGroupStatusByID("group-1")
	assert.Equal(t, "failed", status1.State)

	status2, _ := orch.GetGroupStatusByID("group-2")
	assert.Equal(t, "skipped", status2.State)

	status3, _ := orch.GetGroupStatusByID("group-3")
	assert.Equal(t, "success", status3.State)
}

func TestGroupOrchestrator_ExecuteGroups_DisabledGroup(t *testing.T) {
	cfg := &config.Config{Version: 1}
	engine := &Engine{
		config: cfg,
		logger: logrus.New(),
	}

	orch := NewGroupOrchestrator(cfg, engine, logrus.New())

	executor := &testGroupExecutor{
		errorsToReturn: make(map[string]error),
	}
	orch.executeGroup = executor.executeGroup

	groups := []config.Group{
		{
			ID:      "group-1",
			Name:    "Group 1",
			Enabled: boolPtr(false), // Disabled
			Source:  config.SourceConfig{Repo: "test/source"},
			Targets: []config.TargetConfig{{Repo: "test/target"}},
		},
		{
			ID:      "group-2",
			Name:    "Group 2",
			Enabled: boolPtr(true), // Enabled
			Source:  config.SourceConfig{Repo: "test/source"},
			Targets: []config.TargetConfig{{Repo: "test/target"}},
		},
	}

	err := orch.ExecuteGroups(context.Background(), groups)
	require.NoError(t, err)

	// Only group-2 should have executed
	assert.NotContains(t, executor.executedGroups, "group-1")
	assert.Contains(t, executor.executedGroups, "group-2")
}

func TestGroupOrchestrator_ExecuteGroups_ContextCancellation(t *testing.T) {
	cfg := &config.Config{Version: 1}
	engine := &Engine{
		config: cfg,
		logger: logrus.New(),
	}

	orch := NewGroupOrchestrator(cfg, engine, logrus.New())

	// Cancel context after first group
	ctx, cancel := context.WithCancel(context.Background())

	executor := &testGroupExecutor{
		errorsToReturn: make(map[string]error),
	}
	// Create a wrapper function that cancels after first execution
	orch.executeGroup = func(ctx context.Context, group config.Group) error {
		err := executor.executeGroup(ctx, group)
		if len(executor.executedGroups) == 1 {
			cancel() // Cancel after first group
		}
		return err
	}

	groups := []config.Group{
		{
			ID:      "group-1",
			Name:    "Group 1",
			Enabled: boolPtr(true),
			Source:  config.SourceConfig{Repo: "test/source"},
			Targets: []config.TargetConfig{{Repo: "test/target"}},
		},
		{
			ID:      "group-2",
			Name:    "Group 2",
			Enabled: boolPtr(true),
			Source:  config.SourceConfig{Repo: "test/source"},
			Targets: []config.TargetConfig{{Repo: "test/target"}},
		},
	}

	err := orch.ExecuteGroups(ctx, groups)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Len(t, executor.executedGroups, 1) // Only first group should have executed
}

func TestGroupOrchestrator_GetGroupStatus(t *testing.T) {
	cfg := &config.Config{Version: 1}
	engine := &Engine{
		config: cfg,
		logger: logrus.New(),
	}

	orch := NewGroupOrchestrator(cfg, engine, logrus.New())

	// Set some statuses
	orch.groupStatus["group-1"] = GroupStatus{
		State:     "success",
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}
	orch.groupStatus["group-2"] = GroupStatus{
		State: "failed",
		Error: ErrTestError,
	}

	// Get all statuses
	allStatus := orch.GetGroupStatus()
	assert.Len(t, allStatus, 2)
	assert.Equal(t, "success", allStatus["group-1"].State)
	assert.Equal(t, "failed", allStatus["group-2"].State)

	// Get individual status
	status1, exists1 := orch.GetGroupStatusByID("group-1")
	assert.True(t, exists1)
	assert.Equal(t, "success", status1.State)

	status2, exists2 := orch.GetGroupStatusByID("group-2")
	assert.True(t, exists2)
	assert.Equal(t, "failed", status2.State)

	// Non-existent group
	_, exists3 := orch.GetGroupStatusByID("group-3")
	assert.False(t, exists3)
}

func TestGroupOrchestrator_FilterEnabledGroups(t *testing.T) {
	cfg := &config.Config{Version: 1}
	engine := &Engine{config: cfg}
	orch := NewGroupOrchestrator(cfg, engine, logrus.New())

	groups := []config.Group{
		{ID: "group-1", Enabled: boolPtr(true)},
		{ID: "group-2", Enabled: boolPtr(false)},
		{ID: "group-3", Enabled: nil}, // nil defaults to true
		{ID: "group-4", Enabled: boolPtr(true)},
		{ID: "group-5", Enabled: boolPtr(false)},
	}

	enabled := orch.filterEnabledGroups(groups)

	require.Len(t, enabled, 3)
	assert.Equal(t, "group-1", enabled[0].ID)
	assert.Equal(t, "group-3", enabled[1].ID)
	assert.Equal(t, "group-4", enabled[2].ID)
}

// Helper function for tests
func boolPtr(b bool) *bool {
	return &b
}
