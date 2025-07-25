# Logging Review and Enhancement Plan for go-broadcast

## 🎯 Executive Summary

This document outlines a comprehensive plan to review and enhance logging throughout the go-broadcast project, focusing on debuggability, observability, and troubleshooting capabilities. The plan addresses current gaps in verbose logging, implements context-specific debug flags, and ensures all critical operations have proper logging coverage.

## 📋 Objectives

| Objective                   | Description                                                      |
|-----------------------------|------------------------------------------------------------------|
| **Enhanced Debuggability**  | Implement verbose and trace-level logging for complex operations |
| **Operational Visibility**  | Ensure all critical paths have appropriate logging with context  |
| **Performance Monitoring**  | Add timing and resource metrics to identify bottlenecks          |
| **Security Auditing**       | Log all security-relevant operations with proper data redaction  |
| **Troubleshooting Support** | Provide context-specific debug flags and diagnostic tools        |

## 🛠️ Go Development Standards

**⚠️ IMPORTANT**: All implementation in this plan MUST follow the Go conventions and standards defined in `.github/AGENTS.md`. This includes:

- **Context-First Design**: Always pass `context.Context` as the first parameter for operations that can be canceled or timeout
- **Error Handling Excellence**: Use `fmt.Errorf` for wrapping errors with context, check all errors, return early on failures
- **Interface Design**: Accept interfaces, return concrete types; keep interfaces small and focused, allow for easy mocking and overriding
- **No Global State**: Use dependency injection instead of package-level variables
- **Module Hygiene**: Follow proper Go module management and dependency practices
- **Performance Conscious**: Write benchmarks, profile when needed, avoid premature optimization
- **Testing Standards**: Use testify suite, table-driven tests, proper error handling in tests

Refer to `.github/AGENTS.md` for complete details on naming conventions, commenting standards, commit message format, and all other development practices.

## 🔧 Technical Approach

### Current State Analysis
- Using logrus v1.9.3 with structured logging
- Log levels: debug, info, warn, error via `--log-level` flag
- Separate output system for user-facing messages
- Logs to stderr to keep stdout clean
- Good coverage in sync operations, but gaps in configuration and state management

### Enhancement Strategy
1. Add intuitive verbose flags (-v, -vv)
2. Implement context-specific debug modes
3. Standardize log field names across components
4. Add operation timing and metrics
5. Implement log format options (text, json)
6. Create diagnostic helper commands
7. **Follow all Go conventions from `.github/AGENTS.md`** throughout implementation

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass
- update plan-04-status.md with progress

## Implementation Phases

### Phase 1: Verbose Flag Implementation (Day 1)

#### 1.1 Add Verbose Flag Support
```go
// internal/cli/flags.go
package cli

// LogConfig holds all logging configuration - passed via dependency injection
type LogConfig struct {
    ConfigFile string
    DryRun     bool
    LogLevel   string
    Verbose    int    // -v, -vv, -vvv support
    Debug      DebugFlags
    LogFormat  string
}

type DebugFlags struct {
    Git       bool
    API       bool
    Transform bool
    Config    bool
    State     bool
}

// internal/cli/root.go updates
var logConfig LogConfig // Instance created during initialization

func init() {
    // Add verbose flag with counter
    rootCmd.PersistentFlags().CountVarP(&logConfig.Verbose, "verbose", "v", 
        "Increase verbosity (can be used multiple times)")
    
    // Add debug flags
    rootCmd.PersistentFlags().BoolVar(&logConfig.Debug.Git, "debug-git", false, 
        "Enable detailed git command debugging")
    rootCmd.PersistentFlags().BoolVar(&logConfig.Debug.API, "debug-api", false, 
        "Enable API request/response debugging")
    rootCmd.PersistentFlags().BoolVar(&logConfig.Debug.Transform, "debug-transform", false, 
        "Enable file transformation debugging")
    rootCmd.PersistentFlags().BoolVar(&logConfig.Debug.Config, "debug-config", false,
        "Enable configuration debugging")
    rootCmd.PersistentFlags().BoolVar(&logConfig.Debug.State, "debug-state", false,
        "Enable state discovery debugging")
}

// setupLogging configures the logging system based on the provided configuration.
//
// Parameters:
// - ctx: Context for cancellation and timeout control
// - cmd: Cobra command being executed
// - args: Command line arguments
// - config: Logging configuration
//
// Returns:
// - Error if configuration is invalid
func setupLogging(ctx context.Context, cmd *cobra.Command, args []string, config *LogConfig) error {
    // Map verbose count to log levels
    if config.Verbose > 0 {
        switch config.Verbose {
        case 1:
            config.LogLevel = "debug"
        case 2:
            config.LogLevel = "trace"
        default:
            config.LogLevel = "trace"
            logrus.SetReportCaller(true) // Show file:line for -vvv
        }
    }
    
    // Existing log level parsing...
    return nil
}
```

#### 1.2 Implement Trace Level Support
```go
// Package cli provides command-line interface functionality for go-broadcast.
//
// This package implements the CLI commands, flags, and logging configuration
// used throughout the application. It is designed to provide intuitive
// debugging capabilities and flexible logging output.
//
// Key features include:
// - Verbose flag support (-v, -vv, -vvv) for increasing log detail
// - Component-specific debug flags for targeted troubleshooting
// - Multiple output formats (text and JSON) for different use cases
// - Automatic sensitive data redaction for security
//
// The package follows dependency injection patterns and requires context
// to be passed through all operations for proper cancellation support.
package cli

import (
    "fmt"
    "path/filepath"
    "runtime"
    
    "github.com/sirupsen/logrus"
)

// LoggerService provides logging configuration and trace level support
type LoggerService struct {
    traceLevel logrus.Level
    config     *LogConfig
}

// NewLoggerService creates a new logger service with the given configuration
func NewLoggerService(config *LogConfig) *LoggerService {
    return &LoggerService{
        traceLevel: logrus.DebugLevel - 1,
        config:     config,
    }
}

type TraceHook struct {
    Enabled    bool
    TraceLevel logrus.Level
}

func (h *TraceHook) Levels() []logrus.Level {
    if h.Enabled {
        return []logrus.Level{h.TraceLevel}
    }
    return []logrus.Level{}
}

func (h *TraceHook) Fire(entry *logrus.Entry) error {
    if entry.Level <= h.TraceLevel {
        entry.Level = logrus.DebugLevel
        entry.Message = "[TRACE] " + entry.Message
    }
    return nil
}

// ConfigureLogger sets up the logger with the provided configuration.
//
// Parameters:
// - ctx: Context for cancellation control
//
// Returns:
// - Error if configuration fails
func (s *LoggerService) ConfigureLogger(ctx context.Context) error {
    // Check for context cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Add custom trace level support
    logrus.AddHook(&TraceHook{
        Enabled:    s.config.Verbose >= 2,
        TraceLevel: s.traceLevel,
    })
    
    if s.config.Verbose >= 3 {
        logrus.SetFormatter(&logrus.TextFormatter{
            DisableColors:    false,
            FullTimestamp:    true,
            TimestampFormat:  "15:04:05.000",
            PadLevelText:     true,
            CallerPrettyfier: func(f *runtime.Frame) (string, string) {
                return "", fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line)
            },
        })
    }
    
    return nil
}
```

#### Phase 1 Status Tracking
At the end of Phase 1, update `plan-04-status.md` with:
- **Completed**: Verbose flag implementation, trace level support
- **Successes**: Intuitive -v flag usage, backwards compatibility maintained
- **Challenges**: Integration with existing log level flag
- **Next Steps**: Component-specific debug logging

### Phase 2: Component-Specific Debug Logging (Days 2-3)

#### 2.1 Git Command Debug Logging
```go
// internal/git/git.go enhancements

// gitClient represents a git client with logging configuration
type gitClient struct {
    logger    *logrus.Entry
    logConfig *LogConfig
}

// runCommand executes a git command with debug logging support.
//
// Parameters:
// - ctx: Context for cancellation control
// - cmd: The command to execute
//
// Returns:
// - Error if command execution fails
func (g *gitClient) runCommand(ctx context.Context, cmd *exec.Cmd) error {
    // Check for context cancellation
    select {
    case <-ctx.Done():
        return fmt.Errorf("command cancelled: %w", ctx.Err())
    default:
    }
    
    logger := g.logger.WithField("component", "git")
    
    if g.logConfig.Debug.Git {
        logger.WithFields(logrus.Fields{
            "command": cmd.Path,
            "args":    cmd.Args[1:], // Skip command name
            "dir":     cmd.Dir,
            "env":     filterSensitiveEnv(cmd.Env),
        }).Debug("Executing git command")
        
        // Capture and log output in real-time
        cmd.Stdout = &debugWriter{logger: logger, prefix: "stdout"}
        cmd.Stderr = &debugWriter{logger: logger, prefix: "stderr"}
    }
    
    start := time.Now()
    err := cmd.Run()
    duration := time.Since(start)
    
    logger.WithFields(logrus.Fields{
        "duration_ms": duration.Milliseconds(),
        "exit_code":   cmd.ProcessState.ExitCode(),
    }).Debug("Git command completed")
    
    return err
}

type debugWriter struct {
    logger *logrus.Entry
    prefix string
}

func (w *debugWriter) Write(p []byte) (n int, err error) {
    w.logger.WithField("stream", w.prefix).Trace(string(p))
    return len(p), nil
}

func filterSensitiveEnv(env []string) []string {
    filtered := make([]string, 0, len(env))
    for _, e := range env {
        if strings.HasPrefix(e, "GH_TOKEN=") || 
           strings.HasPrefix(e, "GITHUB_TOKEN=") {
            filtered = append(filtered, strings.Split(e, "=")[0]+"=REDACTED")
        } else {
            filtered = append(filtered, e)
        }
    }
    return filtered
}
```

#### 2.2 API Request/Response Logging
```go
// internal/gh/command.go enhancements

// runner represents a GitHub CLI runner with logging configuration
type runner struct {
    logConfig *LogConfig
}

// Execute runs a GitHub CLI command with debug logging.
//
// Parameters:
// - ctx: Context for cancellation and timeout control
// - args: Command arguments
//
// Returns:
// - Output bytes and error if command fails
func (r *runner) Execute(ctx context.Context, args ...string) ([]byte, error) {
    logger := logrus.WithField("component", "github-api")
    
    if r.logConfig.Debug.API {
        logger.WithFields(logrus.Fields{
            "args":      args,
            "timestamp": time.Now().Format(time.RFC3339),
        }).Debug("GitHub CLI request")
        
        // Log request body if present
        for i, arg := range args {
            if arg == "-f" && i+1 < len(args) {
                logger.WithField("field", args[i+1]).Trace("Request field")
            }
        }
    }
    
    start := time.Now()
    output, err := r.run(ctx, args...)
    duration := time.Since(start)
    
    if r.logConfig.Debug.API {
        logger.WithFields(logrus.Fields{
            "duration_ms":   duration.Milliseconds(),
            "response_size": len(output),
            "error":         err,
        }).Debug("GitHub CLI response")
        
        if err == nil && len(output) < 1024 { // Only log small responses
            logger.WithField("response", string(output)).Trace("Response body")
        }
    }
    
    return output, err
}
```

#### 2.3 Transform Operation Logging
```go
// internal/transform/text.go enhancements

// TextTransformer handles text transformations with logging
type TextTransformer struct {
    logConfig *LogConfig
}

// Transform applies variable substitutions to content.
//
// Parameters:
// - ctx: Context for cancellation control
// - content: The content to transform
// - vars: Variables to substitute
//
// Returns:
// - Transformed content and error if transformation fails
func (t *TextTransformer) Transform(ctx context.Context, content []byte, vars map[string]string) ([]byte, error) {
    // Check for context cancellation
    select {
    case <-ctx.Done():
        return nil, fmt.Errorf("transformation cancelled: %w", ctx.Err())
    default:
    }
    
    logger := logrus.WithField("component", "transform")
    
    if t.logConfig.Debug.Transform {
        logger.WithFields(logrus.Fields{
            "content_size": len(content),
            "variables":    len(vars),
            "type":         "text",
        }).Debug("Starting transformation")
        
        // Log variable replacements
        for key, value := range vars {
            logger.WithFields(logrus.Fields{
                "variable": key,
                "value":    truncate(value, 50),
            }).Trace("Variable replacement")
        }
    }
    
    result, err := t.transform(ctx, content, vars)
    if err != nil {
        return nil, fmt.Errorf("transform failed: %w", err)
    }
    
    if t.logConfig.Debug.Transform && len(result) != len(content) {
        logger.WithFields(logrus.Fields{
            "size_before": len(content),
            "size_after":  len(result),
            "diff":        len(result) - len(content),
        }).Debug("Transformation completed")
        
        // Show before/after for small files
        if len(content) < 500 && t.logConfig.Verbose >= 3 {
            logger.WithField("before", string(content)).Trace("Content before")
            logger.WithField("after", string(result)).Trace("Content after")
        }
    }
    
    return result, nil
}

func truncate(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}
```

#### Phase 2 Status Tracking
At the end of Phase 2, update `plan-04-status.md` with:
- **Completed**: Git, API, and Transform debug logging
- **Successes**: Detailed command execution visibility, performance metrics
- **Challenges**: Output volume management, sensitive data handling
- **Next Steps**: Cover missing logging areas

### Phase 3: Missing Coverage Areas (Days 4-5)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass

#### 3.1 Config Validation Logging
```go
// internal/config/config.go enhancements

// Config represents the application configuration
type Config struct {
    Version string
    Source  SourceConfig
    Targets []TargetConfig
    logConfig *LogConfig
}

// Validate checks the configuration for correctness.
//
// Parameters:
// - ctx: Context for cancellation control
//
// Returns:
// - Error if validation fails
//
// Side Effects:
// - Logs validation progress and warnings
func (c *Config) Validate(ctx context.Context) error {
    logger := logrus.WithField("component", "config")
    
    if c.logConfig.Debug.Config {
        logger.Debug("Starting config validation")
    }
    
    // Check for context cancellation
    select {
    case <-ctx.Done():
        return fmt.Errorf("validation cancelled: %w", ctx.Err())
    default:
    }
    
    // Version validation
    if c.logConfig.Debug.Config {
        logger.WithField("version", c.Version).Trace("Validating version")
    }
    if c.Version != CurrentVersion {
        logger.WithFields(logrus.Fields{
            "expected": CurrentVersion,
            "actual":   c.Version,
        }).Warn("Config version mismatch")
    }
    
    // Source validation
    if c.logConfig.Debug.Config {
        logger.WithFields(logrus.Fields{
            "repo":   c.Source.Repo,
            "branch": c.Source.Branch,
            "path":   c.Source.Path,
        }).Debug("Validating source configuration")
    }
    
    if err := validateRepoName(c.Source.Repo); err != nil {
        logger.WithError(err).Error("Invalid source repository name")
        return fmt.Errorf("invalid source repo: %w", err)
    }
    
    // Target validation
    logger.WithField("target_count", len(c.Targets)).Debug("Validating targets")
    
    for i, target := range c.Targets {
        targetLogger := logger.WithFields(logrus.Fields{
            "index": i,
            "repo":  target.Repo,
        })
        
        if c.logConfig.Debug.Config {
            targetLogger.WithField("file_count", len(target.Files)).Trace("Validating target")
        }
        
        // Validate file mappings
        for j, file := range target.Files {
            if c.logConfig.Debug.Config {
                targetLogger.WithFields(logrus.Fields{
                    "file_index": j,
                    "from":       file.From,
                    "to":         file.To,
                }).Trace("Validating file mapping")
            }
            
            if file.From == "" || file.To == "" {
                targetLogger.Error("Empty file mapping found")
                return fmt.Errorf("target %d, file %d: empty mapping", i, j)
            }
        }
    }
    
    logger.Debug("Config validation completed successfully")
    return nil
}
```

#### 3.2 State Discovery Logging
```go
// internal/state/discovery.go enhancements

// Discoverer handles state discovery with logging
type Discoverer struct {
    config    *Config
    logConfig *LogConfig
}

// Discover finds the current state of repositories.
//
// Parameters:
// - ctx: Context for cancellation and timeout control
//
// Returns:
// - State information and error if discovery fails
//
// Side Effects:
// - Makes network calls to discover repository states
func (d *Discoverer) Discover(ctx context.Context) (*State, error) {
    logger := logrus.WithField("component", "state-discovery")
    
    logger.Info("Starting state discovery")
    startTime := time.Now()
    
    // Check for context cancellation
    select {
    case <-ctx.Done():
        return nil, fmt.Errorf("discovery cancelled: %w", ctx.Err())
    default:
    }
    
    // Source state discovery
    logger.Debug("Discovering source repository state")
    sourceState, err := d.discoverSource(ctx)
    if err != nil {
        logger.WithError(err).Error("Failed to discover source state")
        return nil, fmt.Errorf("source discovery failed: %w", err)
    }
    
    if d.logConfig.Debug.State {
        logger.WithFields(logrus.Fields{
            "commit":     sourceState.LatestCommit,
            "branch":     sourceState.Branch,
            "file_count": len(sourceState.Files),
            "files":      sourceState.Files,
        }).Debug("Source state discovered")
    }
    
    // Target states discovery
    logger.WithField("target_count", len(d.config.Targets)).Debug("Discovering target states")
    
    targetStates := make(map[string]*TargetState)
    for _, target := range d.config.Targets {
        targetLogger := logger.WithField("target", target.Repo)
        
        if d.logConfig.Debug.State {
            targetLogger.Trace("Discovering target state")
        }
        
        state, err := d.discoverTarget(ctx, target)
        if err != nil {
            targetLogger.WithError(err).Warn("Failed to discover target state")
            continue
        }
        
        targetLogger.WithFields(logrus.Fields{
            "has_pr":      state.PullRequest != nil,
            "base_match":  state.BaseCommitMatches,
        }).Debug("Target state discovered")
        
        if d.logConfig.Debug.State && state.PullRequest != nil {
            targetLogger.WithFields(logrus.Fields{
                "pr_number":     state.PullRequest.Number,
                "pr_state":      state.PullRequest.State,
                "pr_branch":     state.PullRequest.Branch,
                "pr_mergeable":  state.PullRequest.Mergeable,
                "pr_commits":    state.PullRequest.Commits,
                "pr_changed":    state.PullRequest.ChangedFiles,
            }).Trace("Pull request details")
        }
        
        targetStates[target.Repo] = state
    }
    
    duration := time.Since(startTime)
    logger.WithFields(logrus.Fields{
        "duration_ms":    duration.Milliseconds(),
        "targets_found":  len(targetStates),
        "targets_failed": len(d.config.Targets) - len(targetStates),
    }).Info("State discovery completed")
    
    return &State{
        Source:  *sourceState,
        Targets: targetStates,
    }, nil
}

// discoverPullRequest searches for existing pull requests.
//
// Parameters:
// - ctx: Context for cancellation control
// - repo: Repository to search in
//
// Returns:
// - Pull request information and error if search fails
func (d *Discoverer) discoverPullRequest(ctx context.Context, repo string) (*PullRequest, error) {
    logger := logrus.WithFields(logrus.Fields{
        "component": "state-discovery",
        "operation": "pr-discovery",
        "repo":      repo,
    })
    
    if d.logConfig.Debug.State {
        logger.Debug("Searching for existing pull request")
    }
    
    // Search for PR logic...
    var pr *PullRequest
    // Implementation details...
    
    if pr != nil && d.logConfig.Debug.State {
        logger.WithFields(logrus.Fields{
            "pr_found":  true,
            "pr_number": pr.Number,
            "pr_state":  pr.State,
        }).Debug("Found existing pull request")
    }
    
    return pr, nil
}
```

#### Phase 3 Status Tracking
At the end of Phase 3, update `plan-04-status.md` with:
- **Completed**: Config validation and state discovery logging
- **Successes**: Complete visibility into validation and discovery process
- **Challenges**: Balance between detail and noise
- **Next Steps**: Standardize structured logging

### Phase 4: Structured Logging Improvements (Days 6-7)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass

#### 4.1 Standardize Field Names
```go
// Package logging provides standardized logging field definitions and utilities.
//
// This package defines the standard field names used throughout go-broadcast
// to ensure consistent log structure for easier querying and analysis.
//
// Key features include:
// - Standardized field name constants for consistency
// - Helper functions for common logging patterns
// - Request correlation ID generation
// - Context-aware logging utilities
//
// Usage examples:
//
//   logger := logrus.WithField(logging.FieldComponent, "my-component")
//   logger = logging.WithOperation(logger, "process-data")
//   logger = logging.WithDuration(logger, startTime)
//
// Important notes:
// - Always use the defined constants instead of string literals
// - Field names follow a consistent naming pattern
// - This ensures logs can be properly parsed by aggregation systems
package logging

// Standard field names used across the application
const (
    FieldComponent   = "component"
    FieldOperation   = "operation"
    FieldRepository  = "repo"
    FieldBranch      = "branch"
    FieldPR          = "pr_number"
    FieldDuration    = "duration_ms"
    FieldError       = "error"
    FieldPath        = "path"
    FieldSize        = "size_bytes"
    FieldCount       = "count"
    FieldRequestID   = "request_id"
    FieldCorrelation = "correlation_id"
    FieldTarget      = "target"
    FieldSource      = "source"
)

// WithOperation adds operation context to logger
func WithOperation(logger *logrus.Entry, op string) *logrus.Entry {
    return logger.WithFields(logrus.Fields{
        FieldOperation: op,
        FieldRequestID: generateRequestID(),
    })
}

// WithRepo adds repository context
func WithRepo(logger *logrus.Entry, repo string) *logrus.Entry {
    return logger.WithField(FieldRepository, repo)
}

// WithDuration adds duration information
func WithDuration(logger *logrus.Entry, start time.Time) *logrus.Entry {
    return logger.WithField(FieldDuration, time.Since(start).Milliseconds())
}

func generateRequestID() string {
    b := make([]byte, 8)
    rand.Read(b)
    return fmt.Sprintf("%x", b)
}
```

#### 4.2 Add Request Correlation
```go
// internal/sync/engine.go enhancements
func (e *Engine) Execute(ctx context.Context) error {
    correlationID := uuid.New().String()
    ctx = context.WithValue(ctx, "correlation_id", correlationID)
    
    logger := e.logger.WithFields(logrus.Fields{
        logging.FieldComponent:   "sync-engine",
        logging.FieldCorrelation: correlationID,
    })
    
    logger.Info("Starting sync operation")
    
    // Create wait group for parallel execution
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, e.options.Concurrency)
    
    for _, target := range e.config.Targets {
        wg.Add(1)
        go func(target config.Target) {
            defer wg.Done()
            
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            targetCtx := context.WithValue(ctx, "target_repo", target.Repo)
            targetLogger := logger.WithField(logging.FieldTarget, target.Repo)
            
            start := time.Now()
            if err := e.syncRepository(targetCtx, target); err != nil {
                targetLogger.WithFields(logrus.Fields{
                    logging.FieldError:    err.Error(),
                    logging.FieldDuration: time.Since(start).Milliseconds(),
                }).Error("Repository sync failed")
            } else {
                targetLogger.WithField(
                    logging.FieldDuration, time.Since(start).Milliseconds(),
                ).Info("Repository sync completed")
            }
        }(target)
    }
    
    wg.Wait()
    
    logger.WithField(
        logging.FieldDuration, time.Since(startTime).Milliseconds(),
    ).Info("All repository syncs completed")
    
    return nil
}

// updateLoggingFields migrates existing log statements to use standardized field names.
//
// This function performs the following steps:
// - Identifies all existing log statements using old field names
// - Updates them to use the standardized field constants
// - Ensures consistency across the codebase
//
// Parameters:
// - ctx: Context for cancellation control
//
// Returns:
// - Error if migration fails
//
// Notes:
// - This is a migration helper function
// - Should be run once during the logging enhancement implementation
func updateLoggingFields(ctx context.Context) error {
    // Example migration for sync package
    // Before:
    // logger.WithField("repo", repo)
    
    // After:
    // logger.WithField(logging.FieldRepository, repo)
}
```

#### Phase 4 Status Tracking
At the end of Phase 4, update `plan-04-status.md` with:
- **Completed**: Standardized field names, correlation IDs implemented
- **Successes**: Consistent log structure, easier log querying
- **Challenges**: Updating all existing log statements
- **Next Steps**: Output formats and diagnostics

### Phase 5: Output Format and Diagnostics (Days 8-9)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass

#### 5.1 JSON Output Format
```go
// internal/cli/root.go additions
func init() {
    rootCmd.PersistentFlags().StringVar(&logConfig.LogFormat, "log-format", "text", 
        "Log format (text, json)")
}

// setupLogging configures the logging system.
//
// Parameters:
// - ctx: Context for cancellation control
// - cmd: Cobra command being executed
// - args: Command line arguments
// - config: Logging configuration
//
// Returns:
// - Error if setup fails
//
// Side Effects:
// - Configures global logrus settings
func setupLogging(ctx context.Context, cmd *cobra.Command, args []string, config *LogConfig) error {
    // Check for context cancellation
    select {
    case <-ctx.Done():
        return fmt.Errorf("setup cancelled: %w", ctx.Err())
    default:
    }
    
    // Parse log level first
    level, err := logrus.ParseLevel(strings.ToLower(config.LogLevel))
    if err != nil {
        return fmt.Errorf("invalid log level %q: %w", config.LogLevel, err)
    }
    
    // Apply verbose flag override
    if config.Verbose > 0 {
        loggerService := NewLoggerService(config)
        switch config.Verbose {
        case 1:
            level = logrus.DebugLevel
        case 2:
            level = loggerService.traceLevel
        default:
            level = loggerService.traceLevel
            logrus.SetReportCaller(true)
        }
    }
    
    logrus.SetLevel(level)
    
    // Set formatter based on format flag
    switch config.LogFormat {
    case "json":
        logrus.SetFormatter(&logrus.JSONFormatter{
            TimestampFormat: time.RFC3339Nano,
            FieldMap: logrus.FieldMap{
                logrus.FieldKeyTime:  "@timestamp",
                logrus.FieldKeyLevel: "level",
                logrus.FieldKeyMsg:   "message",
                logrus.FieldKeyFunc:  "function",
                logrus.FieldKeyFile:  "file",
            },
        })
    case "text":
        logrus.SetFormatter(&logrus.TextFormatter{
            DisableColors:    false,
            FullTimestamp:    true,
            TimestampFormat:  "15:04:05",
            PadLevelText:     true,
            QuoteEmptyFields: true,
        })
    default:
        return fmt.Errorf("invalid log format: %s", config.LogFormat)
    }
    
    // Log to stderr
    logrus.SetOutput(os.Stderr)
    
    // Add redaction hook
    redactionService := NewRedactionService()
    logrus.AddHook(redactionService.CreateHook())
    
    logrus.WithFields(logrus.Fields{
        "config":     config.ConfigFile,
        "dry_run":    config.DryRun,
        "log_level":  level.String(),
        "log_format": config.LogFormat,
        "verbose":    config.Verbose,
    }).Debug("CLI initialized")
    
    return nil
}
```

#### 5.2 Diagnostic Command
```go
// internal/cli/diagnose.go
package cli

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "runtime"
    "time"
    
    "github.com/spf13/cobra"
)

var diagnoseCmd = &cobra.Command{
    Use:   "diagnose",
    Short: "Collect diagnostic information",
    Long:  "Collects system information, configuration, and recent logs for troubleshooting",
    RunE:  runDiagnose,
}

type DiagnosticInfo struct {
    Timestamp   time.Time         `json:"timestamp"`
    Version     VersionInfo       `json:"version"`
    System      SystemInfo        `json:"system"`
    Environment map[string]string `json:"environment"`
    GitVersion  string            `json:"git_version"`
    GHVersion   string            `json:"gh_cli_version"`
    Config      ConfigInfo        `json:"config"`
}

type SystemInfo struct {
    OS          string `json:"os"`
    Arch        string `json:"arch"`
    NumCPU      int    `json:"num_cpu"`
    GoVersion   string `json:"go_version"`
    Hostname    string `json:"hostname"`
    UserHome    string `json:"user_home"`
}

type ConfigInfo struct {
    Path   string `json:"path"`
    Exists bool   `json:"exists"`
    Valid  bool   `json:"valid"`
    Error  string `json:"error,omitempty"`
}

// runDiagnose collects and outputs diagnostic information.
//
// Parameters:
// - cmd: Cobra command being executed
// - args: Command line arguments
//
// Returns:
// - Error if diagnosis fails
//
// Side Effects:
// - Writes JSON output to stdout
func runDiagnose(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    
    info := &DiagnosticInfo{
        Timestamp: time.Now(),
        Version:   getVersionInfo(),
        System:    getSystemInfo(),
        Environment: collectEnvironment(ctx),
        GitVersion:  getGitVersion(ctx),
        GHVersion:   getGHCLIVersion(ctx),
        Config:      getConfigInfo(ctx, &logConfig),
    }
    
    // Output as JSON
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(info)
}

func getSystemInfo() SystemInfo {
    hostname, _ := os.Hostname()
    homeDir, _ := os.UserHomeDir()
    
    return SystemInfo{
        OS:        runtime.GOOS,
        Arch:      runtime.GOARCH,
        NumCPU:    runtime.NumCPU(),
        GoVersion: runtime.Version(),
        Hostname:  hostname,
        UserHome:  homeDir,
    }
}

// collectEnvironment gathers relevant environment variables for diagnostics.
//
// This function performs the following steps:
// - Collects specific environment variables relevant to go-broadcast
// - Redacts sensitive values like tokens for security
// - Returns a map suitable for diagnostic output
//
// Parameters:
// - ctx: Context for cancellation control
//
// Returns:
// - Map of environment variable names to their values (sensitive data redacted)
//
// Side Effects:
// - Reads from os.Environ
//
// Notes:
// - TOKEN variables are automatically redacted to "REDACTED"
// - Only collects a predefined list of relevant variables
func collectEnvironment(ctx context.Context) map[string]string {
    env := make(map[string]string)
    
    // Collect relevant environment variables
    relevantVars := []string{
        "PATH",
        "HOME",
        "USER",
        "SHELL",
        "GH_TOKEN",
        "GITHUB_TOKEN",
        "GO_BROADCAST_CONFIG",
        "NO_COLOR",
        "TERM",
    }
    
    for _, key := range relevantVars {
        value := os.Getenv(key)
        if value != "" {
            // Redact sensitive values
            if strings.Contains(key, "TOKEN") {
                value = "REDACTED"
            }
            env[key] = value
        }
    }
    
    return env
}

func getGitVersion(ctx context.Context) string {
    cmd := exec.CommandContext(ctx, "git", "--version")
    output, err := cmd.Output()
    if err != nil {
        return fmt.Sprintf("error: %v", err)
    }
    return strings.TrimSpace(string(output))
}

func getGHCLIVersion(ctx context.Context) string {
    cmd := exec.CommandContext(ctx, "gh", "--version")
    output, err := cmd.Output()
    if err != nil {
        return fmt.Sprintf("error: %v", err)
    }
    lines := strings.Split(string(output), "\n")
    if len(lines) > 0 {
        return strings.TrimSpace(lines[0])
    }
    return "unknown"
}

func getConfigInfo(ctx context.Context, logConfig *LogConfig) ConfigInfo {
    info := ConfigInfo{
        Path:   logConfig.ConfigFile,
        Exists: false,
        Valid:  false,
    }
    
    if _, err := os.Stat(info.Path); err == nil {
        info.Exists = true
        
        // Try to load and validate config
        cfg, err := config.Load(info.Path)
        if err != nil {
            info.Error = err.Error()
        } else if err := cfg.Validate(ctx); err != nil {
            info.Error = err.Error()
        } else {
            info.Valid = true
        }
    }
    
    return info
}
```

#### Phase 5 Status Tracking
At the end of Phase 5, update `plan-04-status.md` with:
- **Completed**: JSON format support, diagnostic command
- **Successes**: Machine-readable logs, comprehensive diagnostics
- **Challenges**: Balancing diagnostic detail with security
- **Next Steps**: Performance and security logging

### Phase 6: Performance and Security Logging (Days 10-11)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass

#### 6.1 Operation Timing
```go
// internal/metrics/timing.go
package metrics

import (
    "time"
    
    "github.com/sirupsen/logrus"
    "github.com/mrz1836/go-broadcast/internal/logging"
)

type Timer struct {
    start     time.Time
    operation string
    logger    *logrus.Entry
    fields    logrus.Fields
}

// StartTimer creates a new timer for an operation.
//
// Parameters:
// - ctx: Context for cancellation control
// - logger: Logger entry for output
// - operation: Name of the operation being timed
//
// Returns:
// - Timer instance for tracking operation duration
func StartTimer(ctx context.Context, logger *logrus.Entry, operation string) *Timer {
    return &Timer{
        start:     time.Now(),
        operation: operation,
        logger:    logger.WithField(logging.FieldOperation, operation),
        fields:    make(logrus.Fields),
    }
}

// AddField adds a field to be logged when the timer stops
func (t *Timer) AddField(key string, value interface{}) *Timer {
    t.fields[key] = value
    return t
}

// Stop stops the timer and logs the duration.
//
// Returns:
// - Duration of the operation
//
// Side Effects:
// - Logs operation duration and warnings for slow operations
func (t *Timer) Stop() time.Duration {
    duration := time.Since(t.start)
    
    t.fields[logging.FieldDuration] = duration.Milliseconds()
    t.fields["duration_human"] = duration.String()
    
    if duration > 30*time.Second {
        t.logger.WithFields(t.fields).Warn("Operation took longer than expected")
    } else {
        t.logger.WithFields(t.fields).Debug("Operation completed")
    }
    
    return duration
}

// Usage example in sync package
func (r *RepositorySync) Execute(ctx context.Context) error {
    timer := metrics.StartTimer(ctx, r.logger, "repository_sync").
        AddField(logging.FieldRepository, r.target.Repo)
    defer timer.Stop()
    
    // Clone or update repository
    cloneTimer := metrics.StartTimer(ctx, r.logger, "git_clone")
    if err := r.cloneOrUpdate(ctx); err != nil {
        cloneTimer.AddField("error", err.Error()).Stop()
        return fmt.Errorf("clone failed: %w", err)
    }
    cloneTimer.Stop()
    
    // Process files
    processTimer := metrics.StartTimer(ctx, r.logger, "process_files").
        AddField("file_count", len(r.target.Files))
    if err := r.processFiles(ctx); err != nil {
        processTimer.AddField("error", err.Error()).Stop()
        return fmt.Errorf("process files failed: %w", err)
    }
    processTimer.Stop()
    
    // Create or update PR
    prTimer := metrics.StartTimer(ctx, r.logger, "pr_operations")
    if err := r.createOrUpdatePR(ctx); err != nil {
        prTimer.AddField("error", err.Error()).Stop()
        return fmt.Errorf("PR operations failed: %w", err)
    }
    prTimer.Stop()
    
    return nil
}
```

#### 6.2 Sensitive Data Redaction
```go
// internal/logging/redact.go
package logging

import (
    "regexp"
    "strings"
    
    "github.com/sirupsen/logrus"
)

// RedactionService handles sensitive data redaction
type RedactionService struct {
    sensitivePatterns []*regexp.Regexp
    sensitiveFields   []string
}

// NewRedactionService creates a new redaction service
func NewRedactionService() *RedactionService {
    return &RedactionService{
        sensitivePatterns: []*regexp.Regexp{
            regexp.MustCompile(`(ghp_[a-zA-Z0-9]{36})`),              // GitHub personal tokens
            regexp.MustCompile(`(ghs_[a-zA-Z0-9]{36})`),              // GitHub app tokens
            regexp.MustCompile(`(github_pat_[a-zA-Z0-9_]{82})`),      // New GitHub PAT format
            regexp.MustCompile(`(ghr_[a-zA-Z0-9]{36})`),              // GitHub refresh tokens
            regexp.MustCompile(`(password|token|secret|key)=([^\s&]+)`), // URL parameters
            regexp.MustCompile(`(Bearer|Token)\s+([^\s]+)`),          // Authorization headers
            regexp.MustCompile(`([a-zA-Z0-9+/]{40,}={0,2})`),        // Base64 encoded secrets
        },
        sensitiveFields: []string{
            "password",
            "token",
            "secret",
            "api_key",
            "private_key",
            "gh_token",
            "github_token",
            "authorization",
        },
    }
}

// RedactSensitive removes sensitive data from text.
//
// Parameters:
// - text: Text to redact
//
// Returns:
// - Redacted text with sensitive data replaced
func (r *RedactionService) RedactSensitive(text string) string {
    for _, pattern := range r.sensitivePatterns {
        text = pattern.ReplaceAllStringFunc(text, func(match string) string {
            // Keep some context for debugging
            if len(match) > 10 {
                return match[:4] + "***REDACTED***"
            }
            return "***REDACTED***"
        })
    }
    return text
}

// CreateHook creates a logrus hook for automatic redaction
func (r *RedactionService) CreateHook() logrus.Hook {
    return &RedactionHook{service: r}
}

// RedactionHook automatically redacts sensitive data in logs
type RedactionHook struct {
    service *RedactionService
}

func (h *RedactionHook) Levels() []logrus.Level {
    return logrus.AllLevels
}

func (h *RedactionHook) Fire(entry *logrus.Entry) error {
    // Redact message
    entry.Message = h.service.RedactSensitive(entry.Message)
    
    // Redact fields
    for key, value := range entry.Data {
        // Check if field name suggests sensitive data
        keyLower := strings.ToLower(key)
        for _, sensitive := range h.service.sensitiveFields {
            if strings.Contains(keyLower, sensitive) {
                entry.Data[key] = "***REDACTED***"
                break
            }
        }
        
        // Also redact string values that look sensitive
        if str, ok := value.(string); ok {
            entry.Data[key] = h.service.RedactSensitive(str)
        }
    }
    
    return nil
}

// AuditLogger provides security audit logging
type AuditLogger struct {
    logger *logrus.Entry
}

func NewAuditLogger() *AuditLogger {
    return &AuditLogger{
        logger: logrus.WithField("component", "audit"),
    }
}

func (a *AuditLogger) LogAuthentication(user, method string, success bool) {
    a.logger.WithFields(logrus.Fields{
        "event":   "authentication",
        "user":    user,
        "method":  method,
        "success": success,
        "time":    time.Now().Unix(),
    }).Info("Authentication attempt")
}

func (a *AuditLogger) LogConfigChange(user, action string, config interface{}) {
    a.logger.WithFields(logrus.Fields{
        "event":   "config_change",
        "user":    user,
        "action":  action,
        "time":    time.Now().Unix(),
    }).Info("Configuration changed")
}

func (a *AuditLogger) LogRepositoryAccess(user, repo, action string) {
    a.logger.WithFields(logrus.Fields{
        "event":  "repo_access",
        "user":   user,
        "repo":   repo,
        "action": action,
        "time":   time.Now().Unix(),
    }).Info("Repository accessed")
}
```

#### Phase 6 Status Tracking
At the end of Phase 6, update `plan-04-status.md` with:
- **Completed**: Operation timing, sensitive data redaction, audit logging
- **Successes**: Performance visibility, security compliance
- **Challenges**: Balancing security with debuggability
- **Integration**: All phases complete, ready for testing

### Phase 7: Documentation Review and Enhancement (Days 12-13)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass
- update plan-04-status.md with progress

#### 7.1 Update README with Logging Guide
```markdown
# README.md additions

## Logging and Debugging

### Quick Start
```bash
# Basic verbose output (-v for debug, -vv for trace, -vvv for trace with line numbers)
go-broadcast sync -v                    # Debug level logging
go-broadcast sync -vv                   # Trace level logging  
go-broadcast sync -vvv                  # Trace with caller info

# Component-specific debugging
go-broadcast sync --debug-git           # Show git command details
go-broadcast sync --debug-api           # Show GitHub API requests/responses
go-broadcast sync --debug-transform     # Show file transformation details
go-broadcast sync --debug-config        # Show configuration validation
go-broadcast sync --debug-state         # Show state discovery process

# Combine for comprehensive debugging
go-broadcast sync -vv --debug-git --debug-api

# JSON output for log aggregation
go-broadcast sync --log-format json

# Collect diagnostic information
go-broadcast diagnose > diagnostics.json
```

### Log Levels
- **ERROR**: Critical failures that prevent operation
- **WARN**: Important issues that don't stop execution
- **INFO**: High-level operation progress (default)
- **DEBUG**: Detailed operation information (-v)
- **TRACE**: Very detailed debugging information (-vv)

### Advanced Logging Features

#### Performance Monitoring
All operations are timed automatically. Look for `duration_ms` in logs:
```bash
# Find slow operations
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms > 5000) | "\(.operation): \(.duration_ms)ms"'
```

#### Security and Compliance
- All tokens and secrets are automatically redacted
- Audit trail for configuration changes and repository access
- No sensitive data is ever logged

#### Troubleshooting Common Issues

##### Git Authentication Issues
```bash
# Debug git authentication problems
go-broadcast sync -v --debug-git

# Common indicators:
# - "Authentication failed" in git output
# - "Permission denied" errors
# - Check GH_TOKEN or GITHUB_TOKEN environment variables
```

##### API Rate Limiting
```bash
# Monitor API usage
go-broadcast sync --debug-api --log-format json 2>&1 | \
  jq 'select(.component=="github-api") | {operation, duration_ms, error}'
```

##### File Transformation Issues
```bash
# Debug variable replacements and transformations
go-broadcast sync -vv --debug-transform

# Shows:
# - Variables being replaced
# - File size changes
# - Before/after content for small files (with -vvv)
```

##### State Discovery Problems
```bash
# Understand what go-broadcast sees in repositories
go-broadcast sync --debug-state

# Shows:
# - Source repository state
# - Target repository states
# - Existing PR detection
# - File discovery process
```

### Log Management

#### Structured Logging
JSON format is ideal for log aggregation systems:
```bash
# Send to log aggregation
go-broadcast sync --log-format json 2>&1 | fluentd

# Parse with jq
go-broadcast sync --log-format json 2>&1 | jq '.level="error"'

# Save debug session
go-broadcast sync -vvv --debug-git 2> debug-$(date +%Y%m%d-%H%M%S).log
```

#### Performance Analysis
```bash
# Find slowest operations
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms) | "\(.duration_ms)ms \(.operation)"' | \
  sort -rn | head -20

# Monitor memory usage (if implemented)
go-broadcast sync --log-format json 2>&1 | \
  jq 'select(.memory_mb) | {operation, memory_mb}'
```

### Environment Variables

| Variable                  | Description            | Example |
|---------------------------|------------------------|---------|
| `GO_BROADCAST_LOG_LEVEL`  | Default log level      | `debug` |
| `GO_BROADCAST_LOG_FORMAT` | Default log format     | `json`  |
| `GO_BROADCAST_DEBUG`      | Enable all debug flags | `true`  |
| `NO_COLOR`                | Disable colored output | `1`     |


#### 7.2 Create Dedicated Logging Documentation
```markdown
# docs/logging.md

# go-broadcast Logging Guide

## Overview
go-broadcast provides comprehensive logging capabilities designed for debugging, monitoring, and troubleshooting. The logging system is built on structured logging principles with automatic sensitive data redaction.

## Logging Architecture

### Components
1. **Core Logger**: Built on logrus with custom enhancements
2. **Debug Subsystems**: Component-specific debug flags
3. **Performance Metrics**: Automatic operation timing
4. **Security Layer**: Automatic sensitive data redaction
5. **Structured Output**: JSON format for log aggregation

### Log Destinations
- **stderr**: All logs (keeps stdout clean for program output)
- **stdout**: User-facing messages only
- **Files**: Via shell redirection or log rotation tools

## Verbose Flags (-v, -vv, -vvv)

The verbose flags provide intuitive control over logging detail:

### -v (Debug Level)
Shows detailed operation progress and debugging information:
```
15:04:05 DEBUG Starting sync operation component=sync-engine
15:04:05 DEBUG Discovering source repository state component=state-discovery
15:04:06 DEBUG Git command completed duration_ms=1023 exit_code=0
```

### -vv (Trace Level)
Shows very detailed debugging information including internal operations:
```
15:04:05 [TRACE] Validating config version version=1.0
15:04:05 [TRACE] Variable replacement variable=REPO_NAME value=go-broadcast
15:04:06 [TRACE] Response body response={"data":{"repository":{"name":"go-broadcast"}}}
```

### -vvv (Trace with Caller Info)
Adds file:line information for deep debugging:
```
15:04:05.123 [TRACE] git.go:142 Executing git command args=[status --porcelain]
15:04:05.234 [TRACE] transform.go:89 Content before size=1024
15:04:05.235 [TRACE] transform.go:92 Content after size=1048 diff=24
```

## Debug Flags

### --debug-git
Detailed git command execution logging:
- Full command lines with arguments
- Working directory and environment
- Real-time stdout/stderr output
- Exit codes and timing

Example output:
```
DEBUG Executing git command command=/usr/bin/git args=[clone --depth=1] dir=/tmp/broadcast-123
[TRACE] stdout: Cloning into 'repo'...
[TRACE] stderr: Receiving objects: 100% (547/547), done.
DEBUG Git command completed duration_ms=2341 exit_code=0
```

### --debug-api
GitHub API request/response logging:
- Request parameters and headers (redacted)
- Response sizes and timing
- Rate limit information
- Error details

Example output:
```
DEBUG GitHub CLI request args=[api repos/owner/repo] timestamp=2024-01-15T10:30:45Z
[TRACE] Request field field=state:open
DEBUG GitHub CLI response duration_ms=234 response_size=4096 error=<nil>
```

### --debug-transform
File transformation and template processing:
- Variable substitutions
- Content size changes
- Before/after comparisons (small files)
- Transform timing

Example output:
```
DEBUG Starting transformation content_size=1024 variables=5 type=text
[TRACE] Variable replacement variable={{REPO_NAME}} value=go-broadcast
DEBUG Transformation completed size_before=1024 size_after=1048 diff=24
```

### --debug-config
Configuration loading and validation:
- Schema validation steps
- Default value application
- Environment variable resolution
- Validation errors with context

Example output:
```
DEBUG Starting config validation component=config
[TRACE] Validating version version=1.0
DEBUG Validating source configuration repo=owner/source branch=main path=.broadcast
DEBUG Validating targets target_count=3
```

### --debug-state
Repository state discovery process:
- Source repository analysis
- Target repository scanning
- Pull request detection
- File mapping resolution

Example output:
```
DEBUG Discovering source repository state component=state-discovery
DEBUG Source state discovered commit=abc123 branch=main file_count=5
DEBUG Target state discovered target=owner/target has_pr=true base_match=false
[TRACE] Pull request details pr_number=42 pr_state=open pr_mergeable=true
```

## Log Formats

### Text Format (Default)
Human-readable format with colors (when terminal supports):
```
15:04:05 INFO  Starting broadcast sync     version=1.2.3
15:04:05 DEBUG Config loaded successfully  path=.broadcast.yaml targets=3
15:04:06 WARN  Rate limit approaching      remaining=100
```

### JSON Format (--log-format json)
Machine-readable format for log aggregation:
```json
{
  "@timestamp": "2024-01-15T15:04:05.123Z",
  "level": "info",
  "message": "Starting broadcast sync",
  "component": "cli",
  "version": "1.2.3",
  "correlation_id": "a1b2c3d4"
}
```

## Performance Monitoring

### Automatic Timing
All operations are automatically timed with results in `duration_ms`:
```json
{
  "message": "Repository sync completed",
  "operation": "repository_sync",
  "duration_ms": 5234,
  "duration_human": "5.234s",
  "repo": "owner/target"
}
```

### Finding Bottlenecks
```bash
# Find operations taking more than 5 seconds
go-broadcast sync --log-format json 2>&1 | \
  jq 'select(.duration_ms > 5000) | {operation, duration_ms, repo}'

# Summary of operation times
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms) | "\(.operation)"' | \
  sort | uniq -c | sort -rn
```

## Security and Redaction

### Automatic Redaction
The following patterns are automatically redacted:
- GitHub tokens (ghp_*, ghs_*, github_pat_*, ghr_*)
- Bearer tokens and API keys
- Password/secret/key parameters in URLs
- Base64 encoded secrets
- Environment variables containing TOKEN/SECRET/KEY

### Redaction Examples
Input:
```
Executing command with token=ghp_1234567890abcdef1234567890abcdef1234
```

Output:
```
Executing command with token=ghp_***REDACTED***
```

### Audit Logging
Security-relevant operations are logged with audit markers:
```json
{
  "event": "config_change",
  "user": "system",
  "action": "update",
  "component": "audit",
  "time": 1705330245
}
```

## Diagnostic Command

The `diagnose` command collects system information for troubleshooting:

```bash
go-broadcast diagnose
```

Output includes:
- System information (OS, architecture, CPU count)
- go-broadcast version and build info
- Git and GitHub CLI versions
- Environment variables (redacted)
- Configuration file status
- Basic connectivity tests

Example output:
```json
{
  "timestamp": "2024-01-15T15:04:05Z",
  "version": {
    "version": "1.2.3",
    "commit": "abc123def",
    "date": "2024-01-15"
  },
  "system": {
    "os": "darwin",
    "arch": "arm64",
    "num_cpu": 8,
    "go_version": "go1.21.5"
  },
  "git_version": "git version 2.43.0",
  "gh_cli_version": "gh version 2.40.0 (2024-01-10)",
  "config": {
    "path": ".broadcast.yaml",
    "exists": true,
    "valid": true
  }
}
```

## Troubleshooting Guide

### No Output / Silent Failures
```bash
# Enable verbose output to see what's happening
go-broadcast sync -v

# Check if logs are going to stderr
go-broadcast sync 2>&1 | less
```

### Authentication Issues
```bash
# Debug git authentication
go-broadcast sync --debug-git -v

# Check token is set
echo $GITHUB_TOKEN | cut -c1-10  # Should show first 10 chars

# Test GitHub CLI directly
gh auth status
```

### Slow Operations
```bash
# Find slow operations
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms > 1000) | "\(.duration_ms)ms \(.operation) \(.repo // "")"'

# Enable all debugging to see where time is spent
go-broadcast sync -vv --debug-git --debug-api
```

### Memory Issues
```bash
# Monitor memory during execution (if supported by OS)
/usr/bin/time -l go-broadcast sync 2>&1 | grep "maximum resident set size"

# Use JSON logs to track memory if implemented
go-broadcast sync --log-format json 2>&1 | \
  jq 'select(.memory_mb) | {time: .["@timestamp"], memory_mb, operation}'
```

### Debugging Specific Repositories
```bash
# Set up detailed logging for problem repository
export GO_BROADCAST_LOG_LEVEL=trace
go-broadcast sync --debug-state --debug-git -vv 2> debug-repo.log

# Analyze the log
grep "target=owner/problem-repo" debug-repo.log
```

## Best Practices

### Development
1. Use `-vv` during development for detailed feedback
2. Enable relevant debug flags for the area you're working on
3. Use JSON format when debugging with scripts
4. Save debug logs for complex issues: `2> debug-$(date +%Y%m%d-%H%M%S).log`

### Production
1. Default INFO level for normal operation
2. Use JSON format for log aggregation
3. Enable debug selectively for specific issues
4. Monitor performance metrics regularly
5. Set up log rotation to prevent disk fill

### Log Rotation Example
```bash
# Using rotatelogs
go-broadcast sync --log-format json 2>&1 | \
  rotatelogs -l /var/log/go-broadcast/app.log 86400

# Using system logrotate
cat > /etc/logrotate.d/go-broadcast << EOF
/var/log/go-broadcast/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
EOF
```

### Integration with Monitoring Systems

#### Prometheus
```bash
# Extract metrics from JSON logs
go-broadcast sync --log-format json 2>&1 | \
  go-broadcast-prometheus-exporter

# Metrics available:
# - go_broadcast_operation_duration_seconds
# - go_broadcast_operation_total
# - go_broadcast_errors_total
```

#### ELK Stack
```yaml
# Logstash configuration
input {
  pipe {
    command => "go-broadcast sync --log-format json"
    codec => json
  }
}

filter {
  if [component] {
    mutate {
      add_tag => [ "go-broadcast" ]
    }
  }
}

output {
  elasticsearch {
    hosts => ["localhost:9200"]
    index => "go-broadcast-%{+YYYY.MM.dd}"
  }
}
```

## Appendix: Field Reference

### Standard Fields

| Field            | Description            | Example                           |
|------------------|------------------------|-----------------------------------|
| `@timestamp`     | ISO 8601 timestamp     | `2024-01-15T15:04:05.123Z`        |
| `level`          | Log level              | `info`, `debug`, `trace`          |
| `message`        | Log message            | `Starting sync operation`         |
| `component`      | System component       | `sync-engine`, `git`, `transform` |
| `operation`      | Current operation      | `repository_sync`, `git_clone`    |
| `duration_ms`    | Operation duration     | `1234`                            |
| `error`          | Error message          | `connection timeout`              |
| `repo`           | Repository name        | `owner/repo`                      |
| `correlation_id` | Request correlation ID | `a1b2c3d4-e5f6-7890`              |

### Component-Specific Fields

#### Git Operations
- `command`: Git command path
- `args`: Command arguments
- `exit_code`: Process exit code
- `dir`: Working directory

#### API Operations
- `request_size`: Request body size
- `response_size`: Response body size
- `status_code`: HTTP status code
- `rate_limit_remaining`: API calls remaining

#### Transform Operations
- `content_size`: Content size in bytes
- `variables`: Number of variables
- `type`: Transform type
- `size_before`/`size_after`: Size comparison

## Getting Help

### Debug Checklist
1. ✓ Enable appropriate verbose level (-v, -vv, -vvv)
2. ✓ Enable relevant debug flags
3. ✓ Check stderr for log output
4. ✓ Use JSON format for detailed analysis
5. ✓ Run diagnose command
6. ✓ Save logs for support requests

### Support Resources
- GitHub Issues: Include `go-broadcast diagnose` output
- Community Forum: Share relevant log excerpts (redacted)
- Enterprise Support: Provide full debug logs via secure channel
```

#### 7.3 Create Quick Reference Card
```markdown
# docs/logging-quick-ref.md

# go-broadcast Logging Quick Reference

## Common Commands

```bash
# Basic debugging
go-broadcast sync -v                    # Debug level
go-broadcast sync -vv                   # Trace level
go-broadcast sync -vvv                  # Trace + line numbers

# Specific debugging
go-broadcast sync --debug-git          # Git operations
go-broadcast sync --debug-api          # API calls
go-broadcast sync --debug-transform    # File transforms
go-broadcast sync --debug-config       # Config validation
go-broadcast sync --debug-state        # State discovery

# Full debugging
go-broadcast sync -vvv --debug-git --debug-api --debug-transform

# Machine-readable logs
go-broadcast sync --log-format json

# Diagnostics
go-broadcast diagnose > diagnostics.json
```

## Quick Troubleshooting

### Auth Issues
```bash
go-broadcast sync -v --debug-git
# Look for: "Authentication failed", "Permission denied"
# Check: $GITHUB_TOKEN or $GH_TOKEN
```

### Slow Performance
```bash
go-broadcast sync --log-format json 2>&1 | \
  jq 'select(.duration_ms > 5000)'
```

### Find Errors
```bash
go-broadcast sync --log-format json 2>&1 | \
  jq 'select(.level == "error")'
```

### Debug Specific Repo
```bash
go-broadcast sync -vv 2>&1 | grep "repo=owner/name"
```

## Log Levels

| Flag      | Level        | Shows               |
|-----------|--------------|---------------------|
| (default) | INFO         | Basic progress      |
| `-v`      | DEBUG        | Detailed operations |
| `-vv`     | TRACE        | Internal details    |
| `-vvv`    | TRACE+caller | With file:line      |

## Debug Flags

| Flag                | Shows                                |
|---------------------|--------------------------------------|
| `--debug-git`       | Git commands, output, timing         |
| `--debug-api`       | API requests, responses, rate limits |
| `--debug-transform` | Variable substitution, file changes  |
| `--debug-config`    | Config loading, validation           |
| `--debug-state`     | Repository discovery, PR detection   |

## Environment Variables

```bash
export GO_BROADCAST_LOG_LEVEL=debug    # Default log level
export GO_BROADCAST_LOG_FORMAT=json    # Default format
export GO_BROADCAST_DEBUG=true         # Enable all debug flags
export NO_COLOR=1                      # Disable colors
```

## Save Debug Session

```bash
# Timestamped debug log
go-broadcast sync -vvv --debug-git 2> debug-$(date +%Y%m%d-%H%M%S).log

# Full session with script
script -c "go-broadcast sync -vvv" full-session.log
```

## Performance Analysis

```bash
# Find slow operations
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms) | "\(.duration_ms)ms \(.operation)"' | \
  sort -rn

# Operation summary
go-broadcast sync --log-format json 2>&1 | \
  jq -r '.operation // empty' | sort | uniq -c
```

## Log Filtering

```bash
# Errors only
... 2>&1 | jq 'select(.level == "error")'

# Specific component
... 2>&1 | jq 'select(.component == "git")'

# Specific repo
... 2>&1 | jq 'select(.repo == "owner/repo")'

# Long operations
... 2>&1 | jq 'select(.duration_ms > 1000)'
```


#### 7.4 Update CLI Help Text
```go
// internal/cli/root.go - Enhanced help text
func init() {
    rootCmd.PersistentFlags().CountVarP(&logConfig.Verbose, "verbose", "v", 
        `Increase verbosity. Use multiple times for more detail:
  -v    Debug level (detailed operations)
  -vv   Trace level (internal operations)  
  -vvv  Trace level with file locations`)
    
    rootCmd.PersistentFlags().BoolVar(&logConfig.Debug.Git, "debug-git", false, 
        "Enable git debugging (shows commands, output, timing)")
    rootCmd.PersistentFlags().BoolVar(&logConfig.Debug.API, "debug-api", false, 
        "Enable API debugging (shows requests, responses)")
    rootCmd.PersistentFlags().BoolVar(&logConfig.Debug.Transform, "debug-transform", false, 
        "Enable transform debugging (shows file modifications)")
    rootCmd.PersistentFlags().BoolVar(&logConfig.Debug.Config, "debug-config", false,
        "Enable config debugging (shows validation details)")
    rootCmd.PersistentFlags().BoolVar(&logConfig.Debug.State, "debug-state", false,
        "Enable state debugging (shows discovery process)")
    
    rootCmd.PersistentFlags().StringVar(&logConfig.LogFormat, "log-format", "text", 
        "Log output format: text (human-readable) or json (machine-readable)")
}

// Add examples to root command
rootCmd.Example = `  # Basic sync with debug output
  go-broadcast sync -v
  
  # Debug git operations
  go-broadcast sync --debug-git
  
  # Full debugging with trace
  go-broadcast sync -vv --debug-git --debug-api
  
  # Machine-readable JSON logs
  go-broadcast sync --log-format json
  
  # Save debug session
  go-broadcast sync -vvv 2> debug.log`
```

#### 7.5 Create Troubleshooting Runbook
```markdown
# docs/troubleshooting-runbook.md

# go-broadcast Troubleshooting Runbook

## Common Issues and Solutions

### 1. Authentication Failures

#### Symptoms
- "Authentication failed" errors
- "Permission denied (publickey)" messages
- 403/401 HTTP errors

#### Diagnosis
```bash
# Check git authentication
go-broadcast sync -v --debug-git 2>&1 | grep -E "(Authentication|Permission denied|401|403)"

# Verify GitHub token
gh auth status
echo $GITHUB_TOKEN | cut -c1-10  # Should show first 10 chars
```

#### Solutions
1. Check token is set: `export GITHUB_TOKEN=ghp_...`
2. Verify token permissions (repo, write access)
3. Test with `gh repo view owner/repo`
4. Check git SSH keys: `ssh -T git@github.com`

### 2. Slow Performance

#### Symptoms
- Operations taking minutes instead of seconds
- Timeouts
- High CPU/memory usage

#### Diagnosis
```bash
# Find slow operations
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms > 5000) | "\(.duration_ms)ms \(.operation) \(.repo // "")"' | \
  sort -rn

# Check for rate limiting
go-broadcast sync --debug-api --log-format json 2>&1 | \
  jq 'select(.component == "github-api") | {duration_ms, error}'
```

#### Solutions
1. Check network connectivity
2. Reduce concurrency: `go-broadcast sync --concurrency 1`
3. Check for large files in repositories
4. Verify not hitting API rate limits
5. Use `--dry-run` to test without making changes

### 3. File Not Found / Transform Errors

#### Symptoms
- "file not found" errors
- "failed to transform" messages
- Empty or corrupted files

#### Diagnosis
```bash
# Debug file operations
go-broadcast sync -vv --debug-transform 2>&1 | grep -E "(Transform|not found|failed)"

# Check file mappings
go-broadcast sync --debug-config -v
```

#### Solutions
1. Verify source files exist in source repo
2. Check file paths are correct (case sensitive!)
3. Validate `.broadcast.yaml` configuration
4. Check for special characters in file names
5. Verify branch names are correct

### 4. Pull Request Issues

#### Symptoms
- PRs not created
- PRs created but not updated
- Duplicate PRs
- Merge conflicts

#### Diagnosis
```bash
# Debug PR operations
go-broadcast sync --debug-state --debug-api -v 2>&1 | grep -E "(pr_|pull request)"

# Check existing PRs
gh pr list --repo owner/repo --search "broadcast in:title"
```

#### Solutions
1. Check PR already exists (won't create duplicates)
2. Verify branch protection rules allow PR creation
3. Check for merge conflicts in existing PRs
4. Ensure bot has write permissions
5. Look for `pr_template` configuration issues

### 5. Configuration Problems

#### Symptoms
- "invalid configuration" errors
- Unexpected behavior
- Missing targets

#### Diagnosis
```bash
# Validate configuration
go-broadcast validate

# Debug config loading
go-broadcast sync --debug-config -v

# Check config file
cat .broadcast.yaml | go-broadcast validate -
```

#### Solutions
1. Check YAML syntax (proper indentation)
2. Verify schema version matches
3. Ensure all required fields present
4. Check for typos in repository names
5. Validate file paths exist

### 6. State Discovery Failures

#### Symptoms
- "failed to discover state" errors
- Missing repositories
- Incorrect PR detection

#### Diagnosis
```bash
# Debug state discovery
go-broadcast sync --debug-state -vv

# Check specific repository
go-broadcast sync --debug-state 2>&1 | grep "target=owner/repo"
```

#### Solutions
1. Verify repository access permissions
2. Check repository actually exists
3. Ensure default branch is correct
4. Verify network connectivity to GitHub
5. Check for GitHub service issues

## Escalation Procedures

### Level 1: Self-Service
1. Enable verbose logging: `-vv`
2. Enable relevant debug flags
3. Check this runbook
4. Search existing issues

### Level 2: Community Support
1. Run `go-broadcast diagnose > diagnose.json`
2. Collect debug logs: `go-broadcast sync -vvv 2> debug.log`
3. Create minimal reproduction case
4. Post to discussions/issues with:
   - Diagnosis output
   - Relevant log excerpts (redacted)
   - Steps to reproduce

### Level 3: Advanced Debugging
1. Enable all debugging:
   ```bash
   export GO_BROADCAST_DEBUG=true
   go-broadcast sync -vvv 2> full-debug.log
   ```

2. Capture network traffic:
   ```bash
   # If needed for API issues
   export HTTPS_PROXY=http://localhost:8080
   # Use mitmproxy or similar
   ```

3. Profile performance:
   ```bash
   # If available in build
   go-broadcast sync --cpuprofile=cpu.prof
   go tool pprof cpu.prof
   ```

## Emergency Procedures

### Rollback Failed Sync
```bash
# List recent PRs
gh pr list --repo owner/repo --limit 10

# Close PRs created by broadcast
gh pr close --repo owner/repo PR_NUMBER

# Revert commits if needed
git revert COMMIT_SHA
```

### Stop All Operations
```bash
# Kill all go-broadcast processes
pkill -f go-broadcast

# Cancel running GitHub Actions
gh run list --repo owner/repo | grep "in_progress" | \
  awk '{print $1}' | xargs -I {} gh run cancel {}
```

### Recovery Checklist
- [ ] Stop any running operations
- [ ] Collect all logs
- [ ] Document what was running
- [ ] Check for partial updates
- [ ] Verify repository states
- [ ] Clean up temporary files
- [ ] Report issue with logs

## Monitoring and Alerts

### Log Patterns to Monitor
```bash
# Critical errors
"level":"error"
"failed to sync"
"authentication failed"
"rate limit exceeded"

# Performance issues  
"duration_ms":10000  # Operations over 10s
"timeout"
"deadline exceeded"

# Security concerns
"permission denied"
"403"
"unauthorized"
```

### Health Check Script
```bash
#!/bin/bash
# health-check.sh

# Test basic functionality
if ! go-broadcast version > /dev/null 2>&1; then
    echo "ERROR: go-broadcast not responding"
    exit 1
fi

# Test config
if ! go-broadcast validate > /dev/null 2>&1; then
    echo "ERROR: Invalid configuration"
    exit 1
fi

# Test GitHub connectivity
if ! gh auth status > /dev/null 2>&1; then
    echo "ERROR: GitHub authentication failed"
    exit 1
fi

echo "OK: All checks passed"
```

## Appendix: Error Code Reference

| Error | Meaning | Solution |
|-------|---------|----------|
| `config: invalid version` | Config version mismatch | Update config schema |
| `git: exit status 128` | Git command failed | Check git error message |
| `api: 403` | GitHub API forbidden | Check token permissions |
| `api: 404` | Repository not found | Verify repo exists and accessible |
| `transform: pattern not found` | Variable not replaced | Check variable syntax |
| `state: no PR found` | Expected PR missing | Check PR wasn't closed |
```

#### Phase 7 Status Tracking
At the end of Phase 7, update `plan-04-status.md` with:
- **Completed**: Comprehensive documentation review and enhancement
- **Deliverables**: 
  - Updated README with logging guide
  - Dedicated logging documentation (docs/logging.md)
  - Quick reference card for common commands
  - Troubleshooting runbook for operations
  - Enhanced CLI help text
- **Successes**: Clear, actionable documentation for all user levels
- **Improvements**: Examples for every feature, troubleshooting guides
- **Next Steps**: Gather user feedback and iterate on documentation

## Testing Infrastructure

### Unit Tests for Logging
```go
// Package logging provides comprehensive logging capabilities for go-broadcast.
//
// This package implements structured logging with automatic sensitive data
// redaction and is designed to support debugging, monitoring, and troubleshooting.
// It builds upon logrus to provide enhanced functionality specific to go-broadcast's needs.
//
// Key features include:
// - Trace level support beyond standard log levels
// - Automatic operation timing and performance metrics
// - Sensitive data redaction for security compliance
// - Standardized field names for consistent log structure
// - Context correlation for request tracking
//
// Usage examples:
// 
//   // Create a logger with configuration
//   logConfig := &LogConfig{Verbose: 2, Debug: DebugFlags{Git: true}}
//   logger := NewLoggerService(logConfig)
//   
//   // Time an operation
//   timer := StartTimer(ctx, logger, "my-operation")
//   defer timer.Stop()
//
// Important notes:
// - All operations must accept context.Context as first parameter
// - Configuration is passed via dependency injection, not globals
// - Sensitive data is automatically redacted in all log output
package logging

import (
    "bytes"
    "encoding/json"
    "testing"
    
    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestVerboseFlagMapping(t *testing.T) {
    tests := []struct {
        name     string
        verbose  int
        expected logrus.Level
    }{
        {"no verbose", 0, logrus.InfoLevel},
        {"verbose 1", 1, logrus.DebugLevel},
        {"verbose 2", 2, TraceLevel},
        {"verbose 3", 3, TraceLevel},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            level := mapVerboseToLevel(tt.verbose)
            assert.Equal(t, tt.expected, level)
        })
    }
}

func TestSensitiveDataRedaction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "github token",
            input:    "token=ghp_1234567890abcdef1234567890abcdef1234",
            expected: "token=ghp_***REDACTED***",
        },
        {
            name:     "url with password",
            input:    "https://user:secretpass@github.com/repo",
            expected: "https://user:***REDACTED***@github.com/repo",
        },
        {
            name:     "no sensitive data",
            input:    "This is a normal log message",
            expected: "This is a normal log message",
        },
        {
            name:     "bearer token",
            input:    "Authorization: Bearer abc123def456",
            expected: "Authorization: Bear***REDACTED***",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := RedactSensitive(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestJSONFormatOutput(t *testing.T) {
    var buf bytes.Buffer
    logger := logrus.New()
    logger.SetOutput(&buf)
    logger.SetFormatter(&logrus.JSONFormatter{})
    
    logger.WithFields(logrus.Fields{
        "component": "test",
        "operation": "test_op",
    }).Info("Test message")
    
    var result map[string]interface{}
    err := json.Unmarshal(buf.Bytes(), &result)
    require.NoError(t, err)
    
    assert.Equal(t, "Test message", result["message"])
    assert.Equal(t, "test", result["component"])
    assert.Equal(t, "test_op", result["operation"])
    assert.Equal(t, "info", result["level"])
}

func TestTimerMetrics(t *testing.T) {
    var buf bytes.Buffer
    logger := logrus.New()
    logger.SetOutput(&buf)
    logger.SetLevel(logrus.DebugLevel)
    
    timer := StartTimer(logrus.NewEntry(logger), "test_operation")
    time.Sleep(100 * time.Millisecond)
    duration := timer.Stop()
    
    assert.True(t, duration >= 100*time.Millisecond)
    assert.Contains(t, buf.String(), "test_operation")
    assert.Contains(t, buf.String(), "duration_ms")
}
```

### Integration Tests
```bash
#!/bin/bash
# test/integration/logging_test.sh

# Test verbose flags
echo "Testing verbose flags..."
go-broadcast sync -v 2>&1 | grep -q "DEBUG" || echo "FAIL: -v should show debug logs"
go-broadcast sync -vv 2>&1 | grep -q "\[TRACE\]" || echo "FAIL: -vv should show trace logs"
go-broadcast sync -vvv 2>&1 | grep -q ":[0-9]" || echo "FAIL: -vvv should show line numbers"

# Test debug flags
echo "Testing debug flags..."
go-broadcast sync --debug-git 2>&1 | grep -q "git command" || echo "FAIL: --debug-git"
go-broadcast sync --debug-api 2>&1 | grep -q "GitHub CLI" || echo "FAIL: --debug-api"
go-broadcast sync --debug-transform 2>&1 | grep -q "transform" || echo "FAIL: --debug-transform"

# Test log formats
echo "Testing log formats..."
go-broadcast sync --log-format json 2>&1 | jq . > /dev/null || echo "FAIL: JSON format invalid"

# Test diagnostics
echo "Testing diagnostics..."
go-broadcast diagnose | jq '.system.os' > /dev/null || echo "FAIL: diagnose command"

# Test sensitive data redaction
echo "Testing redaction..."
export GITHUB_TOKEN="ghp_1234567890abcdef1234567890abcdef1234"
go-broadcast sync --debug-api 2>&1 | grep -q "ghp_1234" && echo "FAIL: Token not redacted"

echo "All tests completed"
```

## Implementation Timeline

### Week 1 (Days 1-5)
- **Day 1**: Implement verbose flag support
  - Add -v, -vv, -vvv flags
  - Implement trace level support
  - Update CLI documentation
  
- **Days 2-3**: Add component-specific debug logging
  - Git command debugging
  - API request/response logging
  - Transform operation logging
  
- **Days 4-5**: Cover missing logging areas
  - Config validation logging
  - State discovery logging
  - Error context enhancement

### Week 2 (Days 6-11)
- **Days 6-7**: Standardize structured logging
  - Define standard field names
  - Add correlation IDs
  - Update all components
  
- **Days 8-9**: Add output formats and diagnostics
  - JSON format support
  - Diagnostic command
  - Environment collection
  
- **Days 10-11**: Implement performance and security logging
  - Operation timing metrics
  - Sensitive data redaction
  - Audit logging

### Week 3 (Days 12-13)
- **Days 12-13**: Documentation review and enhancement
  - Update README with comprehensive logging guide
  - Create dedicated logging documentation
  - Create quick reference card
  - Create troubleshooting runbook
  - Update CLI help text with examples

## Success Criteria

### Coverage Metrics
- **100%** of error paths have contextual logging
- **100%** of public API methods have entry/exit logging at debug level
- **90%+** of operations have timing metrics
- **100%** of security-sensitive operations are logged

### Usability Metrics
- Verbose flag reduces support ticket resolution time by **50%**
- Debug logs contain sufficient context to reproduce issues
- Log output remains under **10MB** for typical operations
- JSON format is compatible with common log aggregation tools

### Performance Metrics
- Logging overhead < **5%** of operation time
- Debug logging overhead < **10%** of operation time
- Log write performance > **10,000** entries/second
- Memory overhead < **50MB** for typical operations

### Security Metrics
- **0** sensitive tokens in logs
- **100%** of auth operations logged
- **100%** of configuration changes logged
- Audit trail complete for all state-changing operations

## Maintenance and Evolution

### Ongoing Tasks
1. **Weekly log review**: Analyze log patterns for improvement opportunities
2. **Monthly metrics review**: Evaluate log volume and performance impact
3. **Quarterly field audit**: Ensure field naming consistency
4. **Annual security audit**: Verify no sensitive data leakage

### Log Management
```bash
# Log rotation example
go-broadcast sync --log-format json 2> >(rotatelogs -l /var/log/go-broadcast.log 86400)

# Log aggregation example
go-broadcast sync --log-format json | fluentd -c fluent.conf

# Debug session capture
script -c "go-broadcast sync -vvv --debug-git" debug-session.log

# Performance analysis
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms != null) | "\(.operation): \(.duration_ms)ms"' | \
  sort -n -k2
```

### Future Enhancements
1. **OpenTelemetry Integration**: Add distributed tracing support
   - Trace IDs across operations
   - Span relationships
   - Metrics export
   
2. **Metrics Export**: Prometheus/StatsD integration
   - Operation counters
   - Duration histograms
   - Error rates
   
3. **Log Sampling**: Reduce volume in production
   - Configurable sampling rates
   - Always log errors
   - Intelligent sampling
   
4. **Custom Debug Profiles**: Save and load debug configurations
   - Named profiles (e.g., "git-debug", "full-debug")
   - Environment variable support
   - Config file integration

## Conclusion

This comprehensive logging enhancement plan will transform go-broadcast's observability and debuggability. By implementing intuitive verbose flags, context-specific debugging, and comprehensive coverage of all operations, we'll significantly improve the troubleshooting experience for both users and developers. The structured approach ensures consistent, performant, and secure logging throughout the application, making it easier to diagnose issues, monitor performance, and maintain the system over time.
