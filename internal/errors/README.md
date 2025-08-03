# errors Package

The `errors` package provides standardized error handling utilities and predefined error types for the go-broadcast codebase. It ensures consistent error messages and proper error wrapping throughout the application.

## Features

- **Predefined error types** - Common errors used across the application
- **Error wrapping utilities** - Consistent error context and formatting
- **Validation errors** - Standardized validation error creation
- **Command errors** - Consistent command failure error patterns

## Predefined Errors

The package defines several sentinel errors that are used throughout the application:

```go
var (
    // Sync-related errors
    ErrNoFilesToCommit    = errors.New("no files to commit")
    ErrNoTargets          = errors.New("no targets configured")
    ErrInvalidConfig      = errors.New("invalid configuration")
    ErrSyncFailed         = errors.New("sync operation failed")
    ErrNoMatchingTargets  = errors.New("no targets match the specified filter")
    ErrFileNotFound       = errors.New("source file not found")

    // Git-related errors
    ErrNoChanges          = errors.New("no changes to commit")
    ErrMergeConflict      = errors.New("merge conflict detected")
    ErrNotRepository      = errors.New("not a git repository")
    ErrBranchExists       = errors.New("branch already exists")
    ErrInvalidBranch      = errors.New("invalid branch name")
    ErrUncommittedChanges = errors.New("uncommitted changes in working directory")

    // GitHub-related errors
    ErrPRAlreadyExists    = errors.New("pull request already exists")
    ErrInvalidRepository  = errors.New("invalid repository format")
    ErrAPIError           = errors.New("github api error")
    ErrRateLimited        = errors.New("rate limited by github api")
    ErrUnauthorized       = errors.New("unauthorized access")
)
```

## Error Creation Utilities

### WrapWithContext
Wraps an error with operation context for better error messages:
```go
func WrapWithContext(err error, operation string) error

// Usage
data, err := json.Marshal(obj)
if err != nil {
    return errors.WrapWithContext(err, "marshal user data")
}
// Returns: "failed to marshal user data: <original error>"
```

### InvalidField
Creates a standardized invalid field error:
```go
func InvalidField(field, value string) error

// Usage
if name == "" {
    return errors.InvalidField("username", "cannot be empty")
}
// Returns: "invalid field: username: cannot be empty"
```

### CommandFailed
Creates a standardized command failure error:
```go
func CommandFailed(cmd string, err error) error

// Usage
output, err := exec.Command("git", "status").Output()
if err != nil {
    return errors.CommandFailed("git status", err)
}
// Returns: "command failed: git status: <original error>"
```

### ValidationFailed
Creates a standardized validation error:
```go
func ValidationFailed(item, reason string) error

// Usage
if !isValid {
    return errors.ValidationFailed("config file", "missing required fields")
}
// Returns: "validation failed: config file: missing required fields"
```

### PathTraversal
Creates a path traversal security error:
```go
func PathTraversal(path string) error

// Usage
if strings.Contains(path, "..") {
    return errors.PathTraversal(path)
}
// Returns: "path traversal detected: <path>"
```

### EmptyField
Creates an empty field validation error:
```go
func EmptyField(field string) error

// Usage
if repo == "" {
    return errors.EmptyField("repository")
}
// Returns: "field cannot be empty: repository"
```

### RequiredField
Creates a required field validation error:
```go
func RequiredField(field string) error

// Usage
if config.Token == "" {
    return errors.RequiredField("token")
}
// Returns: "field is required: token"
```

### InvalidFormat
Creates an invalid format error:
```go
func InvalidFormat(field, format string) error

// Usage
if !emailRegex.MatchString(email) {
    return errors.InvalidFormat("email", "user@example.com")
}
// Returns: "invalid format: email: expected format: user@example.com"
```

## Usage Examples

### Checking for Specific Errors
```go
err := processSync()
if errors.Is(err, errors.ErrNoFilesToCommit) {
    // Handle case where there are no files to commit
    return nil
}
```

### Wrapping Errors with Context
```go
func loadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, errors.WrapWithContext(err, "read config file")
    }

    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, errors.WrapWithContext(err, "parse config JSON")
    }

    return &config, nil
}
```

### Creating Validation Errors
```go
func validateRepository(repo string) error {
    if repo == "" {
        return errors.EmptyField("repository")
    }

    if !strings.Contains(repo, "/") {
        return errors.InvalidFormat("repository", "owner/name")
    }

    if strings.Contains(repo, "..") {
        return errors.PathTraversal(repo)
    }

    return nil
}
```

## Best Practices

1. **Use predefined errors** - Check if a suitable error already exists before creating new ones
2. **Wrap with context** - Always provide context when propagating errors
3. **Check with errors.Is()** - Use `errors.Is()` to check for specific error types
4. **Consistent messages** - Use the utility functions for consistent error formatting
5. **Avoid dynamic errors** - Define static errors and wrap them instead of using `fmt.Errorf` directly

## Error Handling Patterns

### Repository Pattern
```go
func (r *Repository) GetFile(path string) ([]byte, error) {
    if path == "" {
        return nil, errors.EmptyField("file path")
    }

    data, err := r.readFile(path)
    if os.IsNotExist(err) {
        return nil, errors.ErrFileNotFound
    }
    if err != nil {
        return nil, errors.WrapWithContext(err, "read file")
    }

    return data, nil
}
```

### Command Execution Pattern
```go
func (g *GitClient) Status(repo string) (string, error) {
    if err := validateRepository(repo); err != nil {
        return "", err
    }

    output, err := g.run("git", "-C", repo, "status")
    if err != nil {
        return "", errors.CommandFailed("git status", err)
    }

    return string(output), nil
}
```

## Migration Guide

### From fmt.Errorf
Before:
```go
return fmt.Errorf("failed to read file: %w", err)
```

After:
```go
return errors.WrapWithContext(err, "read file")
```

### From Dynamic Validation Errors
Before:
```go
return fmt.Errorf("invalid repository: %s", repo)
```

After:
```go
return errors.InvalidField("repository", repo)
```

### From Command Errors
Before:
```go
return fmt.Errorf("command 'git status' failed: %w", err)
```

After:
```go
return errors.CommandFailed("git status", err)
``` errors.CommandFailed("git status", err)
```
