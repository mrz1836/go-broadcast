# validation Package

The `validation` package provides shared validation utilities and patterns for consistent input validation across the go-broadcast codebase. It centralizes common validation logic and ensures consistent error messages.

## Features

- **Repository validation** - Validates repository names and formats
- **Branch validation** - Validates Git branch names
- **Path validation** - Validates file paths and prevents path traversal
- **Generic validation** - Extensible validation framework
- **Validation results** - Structured validation result handling

## Core Types

### Result
The `Result` type represents the outcome of validation operations:
```go
type Result struct {
    Valid  bool     // Whether the validation passed
    Errors []error  // List of validation errors (if any)
}
```

## Validation Functions

### ValidateRepository
Validates repository name format (owner/repo):
```go
func ValidateRepository(repo string) error

// Usage
if err := validation.ValidateRepository("owner/repo-name"); err != nil {
    return err
}
```

**Rules:**
- Must not be empty
- Must follow `owner/repo` format
- Cannot contain path traversal sequences (`..`)
- Allows alphanumeric characters, hyphens, underscores, and dots

### ValidateBranch
Validates Git branch names:
```go
func ValidateBranch(branch string) error

// Usage
if err := validation.ValidateBranch("feature/new-feature"); err != nil {
    return err
}
```

**Rules:**
- Must not be empty
- Allows alphanumeric characters, hyphens, underscores, and forward slashes
- Cannot contain invalid Git characters

### ValidatePath
Validates file paths and prevents path traversal attacks:
```go
func ValidatePath(path string) error

// Usage
if err := validation.ValidatePath("src/main.go"); err != nil {
    return err
}
```

**Rules:**
- Must not be empty
- Cannot contain path traversal sequences (`..`)
- Cannot contain null bytes
- Must be a relative path (no leading `/`)

### ValidateEmail
Validates email address format:
```go
func ValidateEmail(email string) error

// Usage
if err := validation.ValidateEmail("user@example.com"); err != nil {
    return err
}
```

### ValidateURL
Validates URL format and scheme:
```go
func ValidateURL(url string) error

// Usage
if err := validation.ValidateURL("https://github.com/owner/repo"); err != nil {
    return err
}
```

## Batch Validation

### ValidateFields
Validates multiple fields at once and returns all errors:
```go
func ValidateFields(validations map[string]func() error) Result

// Usage
result := validation.ValidateFields(map[string]func() error{
    "repository": func() error { return validation.ValidateRepository(config.Repo) },
    "branch":     func() error { return validation.ValidateBranch(config.Branch) },
    "path":       func() error { return validation.ValidatePath(config.FilePath) },
})

if !result.Valid {
    for _, err := range result.Errors {
        log.Printf("Validation error: %v", err)
    }
    return fmt.Errorf("validation failed with %d errors", len(result.Errors))
}
```

### ValidateStruct
Validates struct fields using tags (if implemented):
```go
type Config struct {
    Repository string `validate:"repository"`
    Branch     string `validate:"branch"`
    FilePath   string `validate:"path"`
}

result := validation.ValidateStruct(config)
```

## Custom Validators

### Creating Custom Validators
You can create custom validation functions that integrate with the validation framework:
```go
func ValidateCustomField(value string) error {
    if len(value) < 3 {
        return errors.InvalidField("custom_field", "must be at least 3 characters")
    }
    return nil
}

// Use in batch validation
result := validation.ValidateFields(map[string]func() error{
    "custom": func() error { return ValidateCustomField(value) },
})
```

## Usage Examples

### Single Field Validation
```go
func processRepository(repo string) error {
    if err := validation.ValidateRepository(repo); err != nil {
        return fmt.Errorf("invalid repository: %w", err)
    }
    
    // Process the repository
    return nil
}
```

### Multiple Field Validation
```go
func validateConfig(config *Config) error {
    result := validation.ValidateFields(map[string]func() error{
        "source_repo":   func() error { return validation.ValidateRepository(config.Source.Repo) },
        "source_branch": func() error { return validation.ValidateBranch(config.Source.Branch) },
        "webhook_url":   func() error { return validation.ValidateURL(config.WebhookURL) },
    })
    
    if !result.Valid {
        return fmt.Errorf("configuration validation failed: %v", result.Errors)
    }
    
    return nil
}
```

### Request Validation
```go
func validateSyncRequest(req *SyncRequest) error {
    validations := map[string]func() error{}
    
    if req.Repository != "" {
        validations["repository"] = func() error {
            return validation.ValidateRepository(req.Repository)
        }
    }
    
    if req.Branch != "" {
        validations["branch"] = func() error {
            return validation.ValidateBranch(req.Branch)
        }
    }
    
    for _, file := range req.Files {
        field := fmt.Sprintf("file_%s", file.Src)
        validations[field] = func() error {
            return validation.ValidatePath(file.Src)
        }
    }
    
    result := validation.ValidateFields(validations)
    if !result.Valid {
        return &ValidationError{
            Message: "Invalid sync request",
            Fields:  result.Errors,
        }
    }
    
    return nil
}
```

## Integration with HTTP Handlers

### Request Validation Middleware
```go
func validateRequestMiddleware(validator func(interface{}) error) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req RequestData
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }
        
        if err := validator(&req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // Continue processing
    }
}
```

## Best Practices

1. **Validate early** - Validate input at the boundary of your application
2. **Use appropriate validators** - Choose the right validator for each field type
3. **Batch validation** - Use `ValidateFields` to collect all errors at once
4. **Consistent error messages** - The validators provide standardized error messages
5. **Security first** - Always validate paths to prevent security vulnerabilities

## Security Considerations

### Path Traversal Prevention
The `ValidatePath` function prevents path traversal attacks:
```go
// These will return errors:
validation.ValidatePath("../../../etc/passwd")  // Path traversal
validation.ValidatePath("/etc/passwd")          // Absolute path
validation.ValidatePath("file\x00.txt")         // Null byte injection
```

### Repository Name Validation
Repository validation prevents injection attacks:
```go
// These will return errors:
validation.ValidateRepository("repo; rm -rf /")  // Command injection
validation.ValidateRepository("../repo")         // Path traversal
```

## Error Handling

### Validation Error Types
The package returns specific error types that can be checked:
```go
err := validation.ValidateRepository(repo)
if err != nil {
    var validationErr *ValidationError
    if errors.As(err, &validationErr) {
        // Handle validation-specific error
    }
}
```

### Custom Error Messages
You can create custom validation functions with specific error messages:
```go
func validateProjectName(name string) error {
    if name == "" {
        return errors.EmptyField("project_name")
    }
    
    if len(name) > 50 {
        return errors.InvalidField("project_name", "must be 50 characters or less")
    }
    
    return nil
}
```

## Testing Validation

### Unit Tests
```go
func TestRepositoryValidation(t *testing.T) {
    tests := []testutil.TestCase[string, bool]{
        {
            Name:     "valid repository",
            Input:    "owner/repo",
            Expected: true,
            WantErr:  false,
        },
        {
            Name:    "empty repository",
            Input:   "",
            WantErr: true,
        },
        {
            Name:    "path traversal",
            Input:   "../malicious",
            WantErr: true,
        },
    }
    
    testutil.RunTableTests(t, tests, func(t *testing.T, tc testutil.TestCase[string, bool]) {
        err := validation.ValidateRepository(tc.Input)
        
        if tc.WantErr {
            testutil.AssertError(t, err)
        } else {
            testutil.AssertNoError(t, err)
        }
    })
}
```