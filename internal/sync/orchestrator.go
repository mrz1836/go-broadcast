package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// Static errors for orchestration
var (
	ErrOrchestrationFailures = errors.New("group orchestration completed with failures")
)

// GroupStatus represents the execution status of a group
type GroupStatus struct {
	State     string // pending, running, success, failed, skipped
	StartTime time.Time
	EndTime   time.Time
	Error     error
	Message   string // Additional status message
}

// GroupOrchestrator manages execution of multiple sync groups
type GroupOrchestrator struct {
	config       *config.Config
	engine       *Engine
	logger       *logrus.Logger
	groupStatus  map[string]GroupStatus                              // Track group execution status by group ID
	executeGroup func(ctx context.Context, group config.Group) error // Function field for testing
}

// NewGroupOrchestrator creates a new group orchestrator
func NewGroupOrchestrator(cfg *config.Config, engine *Engine, logger *logrus.Logger) *GroupOrchestrator {
	if logger == nil {
		logger = logrus.New()
	}
	o := &GroupOrchestrator{
		config:      cfg,
		engine:      engine,
		logger:      logger,
		groupStatus: make(map[string]GroupStatus),
	}
	// Set the default executeGroup function
	o.executeGroup = o.executeGroupImpl
	return o
}

// ExecuteGroups runs all enabled groups respecting dependencies and priority
func (o *GroupOrchestrator) ExecuteGroups(ctx context.Context, groups []config.Group) error {
	if len(groups) == 0 {
		o.logger.Debug("No groups to execute")
		return nil
	}

	// Apply group filters from options if engine is available
	if o.engine != nil {
		groups = o.filterGroupsByOptions(groups, o.engine.options)
		if len(groups) == 0 {
			o.logger.Info("No groups match the specified filters")
			return nil
		}
		o.logger.WithField("filtered_group_count", len(groups)).Debug("Groups after filtering")
	}

	// Filter enabled groups
	enabledGroups := o.filterEnabledGroups(groups)
	if len(enabledGroups) == 0 {
		o.logger.Info("No enabled groups to execute")
		return nil
	}

	// Resolve dependencies and get execution order
	executionOrder, err := o.resolveDependencies(enabledGroups)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Initialize group status tracking
	o.initializeGroupStatus(enabledGroups)

	// Execute groups in resolved order
	var hasFailures bool
	for _, group := range executionOrder {
		// Check context cancellation
		select {
		case <-ctx.Done():
			o.logger.Info("Context canceled, stopping group execution")
			return ctx.Err()
		default:
		}

		// Check if dependencies completed successfully
		if !o.areDependenciesSatisfied(group) {
			o.logger.WithField("group_id", group.ID).Info("Skipping group due to failed dependencies")
			o.groupStatus[group.ID] = GroupStatus{
				State:   "skipped",
				Message: "Dependencies failed",
			}
			continue
		}

		// Enhanced group start message with visual separation
		o.logger.WithFields(logrus.Fields{
			"group_name": group.Name,
			"group_id":   group.ID,
			"priority":   group.Priority,
			"depends_on": group.DependsOn,
		}).Info("━━━ Starting group sync ━━━")

		o.groupStatus[group.ID] = GroupStatus{
			State:     "running",
			StartTime: time.Now(),
		}

		// Execute the group
		if err := o.executeGroup(ctx, group); err != nil {
			o.groupStatus[group.ID] = GroupStatus{
				State:     "failed",
				EndTime:   time.Now(),
				Error:     err,
				StartTime: o.groupStatus[group.ID].StartTime,
			}
			o.logger.WithError(err).WithFields(logrus.Fields{
				"group_id":   group.ID,
				"group_name": group.Name,
			}).Error("━━━ Group sync failed ━━━")
			hasFailures = true
			// Continue with groups that don't depend on this one
		} else {
			o.groupStatus[group.ID] = GroupStatus{
				State:     "success",
				EndTime:   time.Now(),
				StartTime: o.groupStatus[group.ID].StartTime,
			}
			o.logger.WithFields(logrus.Fields{
				"group_id":   group.ID,
				"group_name": group.Name,
				"duration":   time.Since(o.groupStatus[group.ID].StartTime),
			}).Info("━━━ Group sync completed successfully ━━━")
		}
	}

	// Report final status
	return o.reportFinalStatus(hasFailures)
}

// filterEnabledGroups returns only enabled groups
func (o *GroupOrchestrator) filterEnabledGroups(groups []config.Group) []config.Group {
	var enabled []config.Group
	for _, group := range groups {
		// If Enabled is nil, default to true
		if group.Enabled == nil || *group.Enabled {
			enabled = append(enabled, group)
		} else {
			o.logger.WithField("group_id", group.ID).Debug("Group is disabled, skipping")
		}
	}
	return enabled
}

// filterGroupsByOptions filters groups based on the sync options (GroupFilter and SkipGroups)
func (o *GroupOrchestrator) filterGroupsByOptions(groups []config.Group, options *Options) []config.Group {
	// If options is nil or no filters specified, return all groups
	if options == nil || (len(options.GroupFilter) == 0 && len(options.SkipGroups) == 0) {
		return groups
	}

	filtered := make([]config.Group, 0, len(groups))
	for _, group := range groups {
		// Check if group should be skipped
		shouldSkip := false
		for _, skipPattern := range options.SkipGroups {
			if group.Name == skipPattern || group.ID == skipPattern {
				o.logger.WithFields(logrus.Fields{
					"group_name": group.Name,
					"group_id":   group.ID,
				}).Debug("Group matches skip pattern, excluding from sync")
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		// Check if group matches filter (if filter is specified)
		if len(options.GroupFilter) > 0 {
			matchesFilter := false
			for _, filterPattern := range options.GroupFilter {
				if group.Name == filterPattern || group.ID == filterPattern {
					matchesFilter = true
					break
				}
			}
			if !matchesFilter {
				o.logger.WithFields(logrus.Fields{
					"group_name": group.Name,
					"group_id":   group.ID,
				}).Debug("Group doesn't match filter pattern, excluding from sync")
				continue
			}
		}

		filtered = append(filtered, group)
	}

	return filtered
}

// resolveDependencies resolves dependencies and returns execution order
func (o *GroupOrchestrator) resolveDependencies(groups []config.Group) ([]config.Group, error) {
	// Create dependency resolver
	resolver := NewDependencyResolver(o.logger)

	// Add all groups to resolver
	for _, group := range groups {
		resolver.AddGroup(group)
	}

	// Resolve and get execution order
	executionOrder, err := resolver.Resolve()
	if err != nil {
		return nil, err
	}

	return executionOrder, nil
}

// initializeGroupStatus initializes status tracking for all groups
func (o *GroupOrchestrator) initializeGroupStatus(groups []config.Group) {
	for _, group := range groups {
		o.groupStatus[group.ID] = GroupStatus{
			State: "pending",
		}
	}
}

// areDependenciesSatisfied checks if all dependencies of a group completed successfully
func (o *GroupOrchestrator) areDependenciesSatisfied(group config.Group) bool {
	for _, depID := range group.DependsOn {
		if status, exists := o.groupStatus[depID]; exists {
			if status.State != "success" {
				o.logger.WithFields(logrus.Fields{
					"group_id":      group.ID,
					"dependency_id": depID,
					"dep_state":     status.State,
				}).Debug("Dependency not satisfied")
				return false
			}
		}
		// If dependency doesn't exist in status map, it might be disabled
		// This is handled during dependency resolution
	}
	return true
}

// executeGroupImpl is the actual implementation of executing a single group's sync operations
func (o *GroupOrchestrator) executeGroupImpl(ctx context.Context, group config.Group) error {
	// Set the current group in the engine
	o.engine.currentGroup = &group
	defer func() {
		o.engine.currentGroup = nil
	}()

	// Create a temporary config for this group
	groupConfig := &config.Config{
		Version: o.config.Version,
		Name:    o.config.Name,
		ID:      o.config.ID,
		Groups:  []config.Group{group},
	}

	// Store original config and replace with group config
	originalConfig := o.engine.config
	o.engine.config = groupConfig
	defer func() {
		o.engine.config = originalConfig
	}()

	// Execute the sync for this group using the single group execution method
	// Pass empty target filter since the group already has its targets defined
	return o.engine.executeSingleGroup(ctx, group, []string{})
}

// reportFinalStatus reports the final execution status
func (o *GroupOrchestrator) reportFinalStatus(hasFailures bool) error {
	// Log summary
	var successCount, failedCount, skippedCount int
	var failedGroups []string

	for groupID, status := range o.groupStatus {
		switch status.State {
		case "success":
			successCount++
		case "failed":
			failedCount++
			failedGroups = append(failedGroups, groupID)
		case "skipped":
			skippedCount++
		}
	}

	// Enhanced final status reporting
	o.logger.WithFields(logrus.Fields{
		"success": successCount,
		"failed":  failedCount,
		"skipped": skippedCount,
		"total":   len(o.groupStatus),
	}).Info("═══ Group orchestration completed ═══")

	// Log individual group results for better visibility
	for groupID, status := range o.groupStatus {
		switch status.State {
		case "success":
			o.logger.WithFields(logrus.Fields{
				"group_id": groupID,
				"duration": status.EndTime.Sub(status.StartTime),
			}).Info("✓ Group completed successfully")
		case "failed":
			o.logger.WithFields(logrus.Fields{
				"group_id": groupID,
				"error":    status.Error,
			}).Error("✗ Group failed")
		case "skipped":
			o.logger.WithFields(logrus.Fields{
				"group_id": groupID,
				"reason":   status.Message,
			}).Warn("⚠ Group skipped")
		}
	}

	if hasFailures {
		return fmt.Errorf("%w: %d groups failed (%v)",
			ErrOrchestrationFailures, failedCount, failedGroups)
	}

	return nil
}

// GetGroupStatus returns the status of all groups (for testing and monitoring)
func (o *GroupOrchestrator) GetGroupStatus() map[string]GroupStatus {
	// Return a copy to prevent external modification
	statusCopy := make(map[string]GroupStatus)
	for k, v := range o.groupStatus {
		statusCopy[k] = v
	}
	return statusCopy
}

// GetGroupStatusByID returns the status of a specific group
func (o *GroupOrchestrator) GetGroupStatusByID(groupID string) (GroupStatus, bool) {
	status, exists := o.groupStatus[groupID]
	return status, exists
}
