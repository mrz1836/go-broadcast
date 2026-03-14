// Package logging provides logging configuration and utilities for go-broadcast.
package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStandardFields_Values(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{
			name:     "timestamp field",
			field:    StandardFields.Timestamp,
			expected: "@timestamp",
		},
		{
			name:     "component field",
			field:    StandardFields.Component,
			expected: "component",
		},
		{
			name:     "operation field",
			field:    StandardFields.Operation,
			expected: "operation",
		},
		{
			name:     "correlation_id field",
			field:    StandardFields.CorrelationID,
			expected: "correlation_id",
		},
		{
			name:     "duration_ms field",
			field:    StandardFields.DurationMs,
			expected: "duration_ms",
		},
		{
			name:     "error field",
			field:    StandardFields.Error,
			expected: "error",
		},
		{
			name:     "status field",
			field:    StandardFields.Status,
			expected: "status",
		},
		{
			name:     "repo_name field",
			field:    StandardFields.RepoName,
			expected: "repo_name",
		},
		{
			name:     "source_repo field",
			field:    StandardFields.SourceRepo,
			expected: "source_repo",
		},
		{
			name:     "target_repo field",
			field:    StandardFields.TargetRepo,
			expected: "target_repo",
		},
		{
			name:     "commit_sha field",
			field:    StandardFields.CommitSHA,
			expected: "commit_sha",
		},
		{
			name:     "branch_name field",
			field:    StandardFields.BranchName,
			expected: "branch_name",
		},
		{
			name:     "pr_number field",
			field:    StandardFields.PRNumber,
			expected: "pr_number",
		},
		{
			name:     "file_path field",
			field:    StandardFields.FilePath,
			expected: "file_path",
		},
		{
			name:     "start_time field",
			field:    StandardFields.StartTime,
			expected: "start_time",
		},
		{
			name:     "end_time field",
			field:    StandardFields.EndTime,
			expected: "end_time",
		},
		{
			name:     "error_type field",
			field:    StandardFields.ErrorType,
			expected: "error_type",
		},
		{
			name:     "error_code field",
			field:    StandardFields.ErrorCode,
			expected: "error_code",
		},
		{
			name:     "exit_code field",
			field:    StandardFields.ExitCode,
			expected: "exit_code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.field, "field value should match expected constant")
		})
	}
}

func TestComponentNames_Values(t *testing.T) {
	tests := []struct {
		name      string
		component string
		expected  string
	}{
		{
			name:      "cli component",
			component: ComponentNames.CLI,
			expected:  "cli",
		},
		{
			name:      "git component",
			component: ComponentNames.Git,
			expected:  "git",
		},
		{
			name:      "github-api component",
			component: ComponentNames.API,
			expected:  "github-api",
		},
		{
			name:      "transform component",
			component: ComponentNames.Transform,
			expected:  "transform",
		},
		{
			name:      "config component",
			component: ComponentNames.Config,
			expected:  "config",
		},
		{
			name:      "state-discovery component",
			component: ComponentNames.State,
			expected:  "state-discovery",
		},
		{
			name:      "sync-engine component",
			component: ComponentNames.Sync,
			expected:  "sync-engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.component, "component value should match expected constant")
		})
	}
}

func TestOperationTypes_Values(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		expected  string
	}{
		{
			name:      "git_command operation",
			operation: OperationTypes.GitCommand,
			expected:  "git_command",
		},
		{
			name:      "api_request operation",
			operation: OperationTypes.APIRequest,
			expected:  "api_request",
		},
		{
			name:      "file_transform operation",
			operation: OperationTypes.FileTransform,
			expected:  "file_transform",
		},
		{
			name:      "config_validate operation",
			operation: OperationTypes.ConfigValidate,
			expected:  "config_validate",
		},
		{
			name:      "state_discover operation",
			operation: OperationTypes.StateDiscover,
			expected:  "state_discover",
		},
		{
			name:      "sync_execute operation",
			operation: OperationTypes.SyncExecute,
			expected:  "sync_execute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.operation, "operation value should match expected constant")
		})
	}
}

func TestFieldConstants_Uniqueness(t *testing.T) {
	// Test that all standard fields have unique values
	fields := []string{
		StandardFields.Timestamp,
		StandardFields.Component,
		StandardFields.Operation,
		StandardFields.CorrelationID,
		StandardFields.DurationMs,
		StandardFields.Error,
		StandardFields.Status,
		StandardFields.RepoName,
		StandardFields.SourceRepo,
		StandardFields.TargetRepo,
		StandardFields.CommitSHA,
		StandardFields.BranchName,
		StandardFields.PRNumber,
		StandardFields.FilePath,
		StandardFields.StartTime,
		StandardFields.EndTime,
		StandardFields.ErrorType,
		StandardFields.ErrorCode,
		StandardFields.ExitCode,
		StandardFields.Phase,
		StandardFields.ContentSize,
		StandardFields.FileCount,
		StandardFields.TargetCount,
		StandardFields.SizeChange,
		StandardFields.Variable,
		StandardFields.VariableValue,
		StandardFields.Replacements,
		StandardFields.VariableCount,
		StandardFields.Progress,
		StandardFields.SyncStatus,
	}

	seen := make(map[string]bool)
	for _, field := range fields {
		assert.False(t, seen[field], "field %s should be unique", field)
		seen[field] = true
	}

	assert.Len(t, seen, len(fields), "all fields should be unique")
}

func TestComponentNames_Uniqueness(t *testing.T) {
	// Test that all component names have unique values
	components := []string{
		ComponentNames.CLI,
		ComponentNames.Git,
		ComponentNames.API,
		ComponentNames.Transform,
		ComponentNames.Config,
		ComponentNames.State,
		ComponentNames.Sync,
	}

	seen := make(map[string]bool)
	for _, component := range components {
		assert.False(t, seen[component], "component %s should be unique", component)
		seen[component] = true
	}

	assert.Len(t, seen, len(components), "all components should be unique")
}

func TestOperationTypes_Uniqueness(t *testing.T) {
	// Test that all operation types have unique values
	operations := []string{
		OperationTypes.GitCommand,
		OperationTypes.APIRequest,
		OperationTypes.FileTransform,
		OperationTypes.ConfigValidate,
		OperationTypes.StateDiscover,
		OperationTypes.SyncExecute,
	}

	seen := make(map[string]bool)
	for _, operation := range operations {
		assert.False(t, seen[operation], "operation %s should be unique", operation)
		seen[operation] = true
	}

	assert.Len(t, seen, len(operations), "all operations should be unique")
}

func TestFieldConstants_NonEmpty(t *testing.T) {
	// Test that no field constants are empty
	fieldTests := []struct {
		name  string
		value string
	}{
		{"Timestamp", StandardFields.Timestamp},
		{"Component", StandardFields.Component},
		{"Operation", StandardFields.Operation},
		{"CorrelationID", StandardFields.CorrelationID},
		{"DurationMs", StandardFields.DurationMs},
		{"Error", StandardFields.Error},
		{"Status", StandardFields.Status},
		{"RepoName", StandardFields.RepoName},
		{"SourceRepo", StandardFields.SourceRepo},
		{"TargetRepo", StandardFields.TargetRepo},
		{"CommitSHA", StandardFields.CommitSHA},
		{"BranchName", StandardFields.BranchName},
		{"PRNumber", StandardFields.PRNumber},
		{"FilePath", StandardFields.FilePath},
		{"StartTime", StandardFields.StartTime},
		{"EndTime", StandardFields.EndTime},
		{"ErrorType", StandardFields.ErrorType},
		{"ErrorCode", StandardFields.ErrorCode},
		{"ExitCode", StandardFields.ExitCode},
	}

	for _, tt := range fieldTests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value, "field %s should not be empty", tt.name)
		})
	}
}

func TestComponentNames_NonEmpty(t *testing.T) {
	// Test that no component names are empty
	componentTests := []struct {
		name  string
		value string
	}{
		{"CLI", ComponentNames.CLI},
		{"Git", ComponentNames.Git},
		{"API", ComponentNames.API},
		{"Transform", ComponentNames.Transform},
		{"Config", ComponentNames.Config},
		{"State", ComponentNames.State},
		{"Sync", ComponentNames.Sync},
	}

	for _, tt := range componentTests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value, "component %s should not be empty", tt.name)
		})
	}
}

func TestOperationTypes_NonEmpty(t *testing.T) {
	// Test that no operation types are empty
	operationTests := []struct {
		name  string
		value string
	}{
		{"GitCommand", OperationTypes.GitCommand},
		{"APIRequest", OperationTypes.APIRequest},
		{"FileTransform", OperationTypes.FileTransform},
		{"ConfigValidate", OperationTypes.ConfigValidate},
		{"StateDiscover", OperationTypes.StateDiscover},
		{"SyncExecute", OperationTypes.SyncExecute},
	}

	for _, tt := range operationTests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value, "operation %s should not be empty", tt.name)
		})
	}
}

func TestFieldConstants_Format(t *testing.T) {
	// Test that field constants follow expected naming conventions
	tests := []struct {
		name     string
		field    string
		validate func(t *testing.T, field string)
	}{
		{
			name:  "timestamp field should use @ prefix",
			field: StandardFields.Timestamp,
			validate: func(t *testing.T, field string) {
				assert.Equal(t, "@timestamp", field, "timestamp should use @ prefix for ELK compatibility")
			},
		},
		{
			name:  "snake_case fields",
			field: StandardFields.CorrelationID,
			validate: func(t *testing.T, field string) {
				assert.Equal(t, "correlation_id", field, "should use snake_case format")
			},
		},
		{
			name:  "duration field format",
			field: StandardFields.DurationMs,
			validate: func(t *testing.T, field string) {
				assert.Equal(t, "duration_ms", field, "duration should specify units in field name")
			},
		},
		{
			name:  "repository fields use snake_case",
			field: StandardFields.SourceRepo,
			validate: func(t *testing.T, field string) {
				assert.Equal(t, "source_repo", field, "repo fields should use snake_case")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.field)
		})
	}
}

func TestComponentNames_Format(t *testing.T) {
	// Test that component names follow expected naming conventions
	tests := []struct {
		name      string
		component string
		validate  func(t *testing.T, component string)
	}{
		{
			name:      "kebab-case for multi-word components",
			component: ComponentNames.API,
			validate: func(t *testing.T, component string) {
				assert.Equal(t, "github-api", component, "multi-word components should use kebab-case")
			},
		},
		{
			name:      "state discovery uses kebab-case",
			component: ComponentNames.State,
			validate: func(t *testing.T, component string) {
				assert.Equal(t, "state-discovery", component, "should use kebab-case")
			},
		},
		{
			name:      "sync engine uses kebab-case",
			component: ComponentNames.Sync,
			validate: func(t *testing.T, component string) {
				assert.Equal(t, "sync-engine", component, "should use kebab-case")
			},
		},
		{
			name:      "single word components use lowercase",
			component: ComponentNames.Git,
			validate: func(t *testing.T, component string) {
				assert.Equal(t, "git", component, "single word components should be lowercase")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.component)
		})
	}
}

func TestOperationTypes_Format(t *testing.T) {
	// Test that operation types follow expected naming conventions
	tests := []struct {
		name      string
		operation string
		validate  func(t *testing.T, operation string)
	}{
		{
			name:      "snake_case for multi-word operations",
			operation: OperationTypes.GitCommand,
			validate: func(t *testing.T, operation string) {
				assert.Equal(t, "git_command", operation, "multi-word operations should use snake_case")
			},
		},
		{
			name:      "api request uses snake_case",
			operation: OperationTypes.APIRequest,
			validate: func(t *testing.T, operation string) {
				assert.Equal(t, "api_request", operation, "should use snake_case")
			},
		},
		{
			name:      "file transform uses snake_case",
			operation: OperationTypes.FileTransform,
			validate: func(t *testing.T, operation string) {
				assert.Equal(t, "file_transform", operation, "should use snake_case")
			},
		},
		{
			name:      "sync execute uses snake_case",
			operation: OperationTypes.SyncExecute,
			validate: func(t *testing.T, operation string) {
				assert.Equal(t, "sync_execute", operation, "should use snake_case")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.operation)
		})
	}
}
