# Test Cleanup Utility

A Go-based utility for cleaning up temporary test artifacts created during test execution.

## What it cleans

The utility removes:
- `*.test` - Compiled Go test binaries
- `*.out` - Coverage output files
- `*.prof` - Profiling data files
- `@*.test` - Named test binaries (common pattern)

## Usage

### Via Makefile (Recommended)

```bash
# Clean up test artifacts
make clean-test-artifacts

# Dry run to see what would be cleaned
make clean-test-artifacts-dry

# Clean with verbose output
make clean-test-artifacts-verbose

# Clean everything (build artifacts + test files)
make clean-all
```

### Direct Usage

```bash
# Build the utility first
go build -o bin/test-cleanup ./cmd/test-cleanup

# Basic cleanup
./bin/test-cleanup

# Dry run
./bin/test-cleanup -dry-run -verbose

# Custom patterns
./bin/test-cleanup -patterns="*.test,*.out,*.prof" -exclude-dirs=".git,vendor"
```

## Command Line Options

- `-root`: Root directory to clean (default: current directory)
- `-dry-run`: Show what would be deleted without actually deleting
- `-verbose`: Enable verbose output
- `-patterns`: Comma-separated list of file patterns to clean (default: "*.test,*.out,*.prof")
- `-exclude-dirs`: Comma-separated list of directories to exclude (default: ".git,vendor,node_modules")

## Integration

This utility is automatically run after:
- `make test` and related test targets
- `make bench` and related benchmark targets
- Integration test suites
- CI/CD test runs

## Safety Features

- Excludes important directories (.git, vendor, node_modules) by default
- Supports dry-run mode to preview changes
- Provides detailed verbose output
- Only removes files matching specific patterns
- Shows total size and count of files being removed
