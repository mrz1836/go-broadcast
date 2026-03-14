// Package logging provides logging configuration types and utilities.
package logging

// StandardFields defines the standardized field names for structured logging
// across all components to ensure consistency and enable better log analysis.
//
// This ensures that all components use the same field names for similar data,
// making it easier to query, filter, and analyze logs in aggregation systems.
//
//nolint:gochecknoglobals // Intentional global constants for standardized field names
var StandardFields = struct {
	// Repository Identifiers
	SourceRepo string
	TargetRepo string
	RepoName   string

	// Timing and Performance
	DurationMs string
	StartTime  string
	EndTime    string
	Timestamp  string

	// Operation Context
	Component     string
	Operation     string
	Phase         string
	CorrelationID string

	// Resource Identifiers
	CommitSHA  string
	BranchName string
	PRNumber   string
	FilePath   string

	// Content and Size Metrics
	ContentSize string
	FileCount   string
	TargetCount string
	SizeChange  string

	// Transform Context
	Variable      string
	VariableValue string
	Replacements  string
	VariableCount string

	// Error Information
	Error     string
	ErrorType string
	ErrorCode string
	ExitCode  string

	// Status and Progress
	Status     string
	Progress   string
	SyncStatus string
}{
	// Repository Identifiers
	SourceRepo: "source_repo",
	TargetRepo: "target_repo",
	RepoName:   "repo_name",

	// Timing and Performance
	DurationMs: "duration_ms",
	StartTime:  "start_time",
	EndTime:    "end_time",
	Timestamp:  "@timestamp",

	// Operation Context
	Component:     "component",
	Operation:     "operation",
	Phase:         "phase",
	CorrelationID: "correlation_id",

	// Resource Identifiers
	CommitSHA:  "commit_sha",
	BranchName: "branch_name",
	PRNumber:   "pr_number",
	FilePath:   "file_path",

	// Content and Size Metrics
	ContentSize: "content_size",
	FileCount:   "file_count",
	TargetCount: "target_count",
	SizeChange:  "size_change",

	// Transform Context
	Variable:      "variable",
	VariableValue: "variable_value",
	Replacements:  "replacements",
	VariableCount: "variable_count",

	// Error Information
	Error:     "error",
	ErrorType: "error_type",
	ErrorCode: "error_code",
	ExitCode:  "exit_code",

	// Status and Progress
	Status:     "status",
	Progress:   "progress",
	SyncStatus: "sync_status",
}

// ComponentNames defines standardized component names for logging consistency
//
//nolint:gochecknoglobals // Intentional global constants for standardized component names
var ComponentNames = struct {
	Git       string
	API       string
	Transform string
	Config    string
	State     string
	CLI       string
	Sync      string
}{
	Git:       "git",
	API:       "github-api",
	Transform: "transform",
	Config:    "config",
	State:     "state-discovery",
	CLI:       "cli",
	Sync:      "sync-engine",
}

// OperationTypes defines standardized operation type names
//
//nolint:gochecknoglobals // Intentional global constants for standardized operation types
var OperationTypes = struct {
	GitCommand     string
	APIRequest     string
	FileTransform  string
	ConfigValidate string
	StateDiscover  string
	SyncExecute    string
}{
	GitCommand:     "git_command",
	APIRequest:     "api_request",
	FileTransform:  "file_transform",
	ConfigValidate: "config_validate",
	StateDiscover:  "state_discover",
	SyncExecute:    "sync_execute",
}
