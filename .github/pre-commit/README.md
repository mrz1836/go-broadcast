# GoFortress Pre-commit System

A high-performance, Go-native pre-commit framework with zero Python dependencies.

## Features

- 🚀 **Lightning Fast**: Parallel execution with native Go performance
- 📦 **Zero Dependencies**: Single binary, no Python or Node.js required
- 🔧 **Make Integration**: Seamlessly wraps existing make commands
- ⚙️ **Environment Config**: Configure via `.github/.env.shared`
- 🎯 **MVP Focused**: 5 essential checks for Go projects

## Quick Start

### Build

```bash
cd .github/pre-commit
make build
```

### Install

```bash
# Install the binary
make install

# Install git hooks
gofortress-pre-commit install
```

### Usage

```bash
# Run all checks on staged files
gofortress-pre-commit run

# Run specific check
gofortress-pre-commit run lint

# Run on all files
gofortress-pre-commit run --all-files

# Show available checks
gofortress-pre-commit run --show-checks
```

### Uninstall

```bash
gofortress-pre-commit uninstall
```

## Available Checks

| Check | Description | Type |
|-------|-------------|------|
| `fumpt` | Format code with gofumpt | Make wrapper |
| `lint` | Run golangci-lint | Make wrapper |
| `mod-tidy` | Ensure go.mod and go.sum are tidy | Make wrapper |
| `whitespace` | Fix trailing whitespace | Built-in |
| `eof` | Ensure files end with newline | Built-in |

## Configuration

All configuration is done via environment variables in `.github/.env.shared`:

```bash
# Enable/disable the system
ENABLE_PRE_COMMIT_SYSTEM=true

# Enable/disable specific checks
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
PRE_COMMIT_SYSTEM_ENABLE_LINT=true
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true

# Performance settings
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=0  # 0 = auto (CPU count)
PRE_COMMIT_SYSTEM_FAIL_FAST=false
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=300

# File handling
PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB=10
PRE_COMMIT_SYSTEM_MAX_FILES_OPEN=100
PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS=vendor/,node_modules/,.git/
```

## Development

### Project Structure

```
.github/pre-commit/
├── cmd/gofortress-pre-commit/   # CLI application
├── internal/
│   ├── checks/                   # Check interface and registry
│   │   ├── builtin/             # Built-in checks (whitespace, EOF)
│   │   └── makewrap/            # Make command wrappers
│   ├── config/                  # Configuration loading
│   ├── git/                     # Git operations
│   └── runner/                  # Parallel execution engine
├── Makefile                     # Build and development tasks
├── go.mod                       # Go module definition
└── README.md                    # This file
```

### Building from Source

```bash
# Build binary
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt
```

### Testing

```bash
# Run all tests
make test

# Run with race detector
make test-race

# Generate coverage report
make test-coverage
```

## Architecture

The GoFortress Pre-commit System follows a modular architecture:

1. **CLI Layer**: Cobra-based commands (install, run, uninstall)
2. **Configuration**: Environment-based config from `.env.shared`
3. **Check Registry**: Pluggable check system with clean interfaces
4. **Runner**: Parallel execution engine with worker pools
5. **Git Integration**: Hook installation and file detection

### Adding New Checks

To add a new check, implement the `Check` interface:

```go
type Check interface {
    Name() string
    Description() string
    Run(ctx context.Context, files []string) error
    FilterFiles(files []string) []string
}
```

Then register it in the registry.

## Performance

- Parallel execution by default (configurable workers)
- Efficient file filtering before check execution
- Direct make command execution for consistency
- Built-in checks process files in-memory

## CI/CD Integration

The system integrates with CI/CD pipelines through the `fortress-pre-commit.yml` workflow:

```yaml
- name: Run pre-commit checks
  run: |
    cd .github/pre-commit
    make build
    ./gofortress-pre-commit run --all-files
```

## Troubleshooting

### Common Issues

**"gofortress-pre-commit not found"**
- Run `make install` from `.github/pre-commit/`
- Or add `$(go env GOPATH)/bin` to your PATH

**"make: gofumpt: No such file or directory"**
- Install gofumpt: `go install mvdan.cc/gofumpt@latest`
- Or disable the check: `PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false`

**"Hook already exists"**
- Use `--force` flag: `gofortress-pre-commit install --force`
- Or uninstall first: `gofortress-pre-commit uninstall`

### Debug Mode

Run with verbose output for debugging:

```bash
gofortress-pre-commit run --verbose
```

## License

Part of the go-broadcast project. See repository LICENSE.
