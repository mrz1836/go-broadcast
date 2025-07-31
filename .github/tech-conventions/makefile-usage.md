# Makefile Usage

> Standardized build automation using Make for consistent development workflows across projects.

<br><br>

## 🛠 Overview

The repository's `Makefile` includes reusable targets from `.make/common.mk` and
`.make/go.mk`. The root file exposes a few high-level commands while the files
under `.make` contain the bulk of the build logic.

`common.mk` provides utility tasks for releasing with GoReleaser, tagging
releases, and updating the releaser tool. It also offers the `diff` and `help`
commands used across projects.

`go.mk` supplies Go-specific helpers for linting, testing, generating code,
building binaries, and updating dependencies. Targets such as `lint`, `test`,
`test-ci`, and `coverage` are defined here and invoked by the root `Makefile`.

Use `make help` to view the full list of supported commands.

<br><br>

## 📋 Common Commands

### Essential Development Commands

```bash
# View all available commands
make help

# Install dependencies
make mod-download
make install-stdlib

# Run tests
make test           # Fast linting + unit tests
make test-race      # Unit tests with race detector
make test-ci        # Full CI test suite

# Code quality
make lint           # Run all linters
make fumpt          # Format Go code
make coverage       # Generate coverage report
```

<br><br>

## 🏗️ Makefile Structure

### Root Makefile
* Located at project root
* Imports shared logic from `.make/` directory
* Defines project-specific targets
* Sets default goals and variables

### Shared Makefiles
* **`.make/common.mk`** - Cross-language utilities
  * Release management
  * Tagging and versioning
  * Diff checking
  * Help generation
  
* **`.make/go.mk`** - Go-specific targets
  * Linting and formatting
  * Testing and coverage
  * Building and installation
  * Dependency management

<br><br>

## 🎯 Target Categories

### 🧪 Testing
```bash
make test              # Default: lint + unit tests
make test-race         # With race detector
make test-ci           # CI configuration
make test-cover        # With coverage
make test-short        # Skip slow tests
make bench             # Run benchmarks
```

### 🔧 Building
```bash
make build-go          # Build binary
make install           # Install locally
make install-go        # Install specific version
make release-snap      # Build snapshot
make release           # Full release
```

### 📊 Code Quality
```bash
make lint              # Run linters
make fumpt             # Format code
make vet               # Run go vet
make govulncheck       # Security scan
make coverage          # Coverage report
```

### 📦 Dependencies
```bash
make mod-download      # Download modules
make mod-tidy          # Clean go.mod/sum
make update            # Update all deps
make clean-mods        # Clear module cache
```

<br><br>

## 🔄 Make Patterns

### Conditional Execution
```makefile
# Run only if file exists
test: $(if $(wildcard go.mod),test-go,)

# Platform-specific commands
install: $(if $(filter Darwin,$(shell uname)),install-mac,install-linux)
```

### Dependency Chains
```makefile
# Ensure deps before test
test: mod-download lint test-go

# Multi-step builds
release: test build-go tag
```

### Variable Overrides
```bash
# Override default version
make install VERSION=1.20

# Custom build flags
make build-go LDFLAGS="-X main.version=dev"

# Parallel execution
make -j4 test
```

<br><br>

## 📝 Best Practices

### DO:
* **Use standard target names** (`test`, `build`, `clean`)
* **Document custom targets** with `## ` comments for help
* **Check prerequisites** before running commands
* **Provide sensible defaults** that work out of the box
* **Make targets idempotent** - safe to run multiple times

### DON'T:
* **Hide important commands** - keep them discoverable
* **Assume tool installation** - check or install as needed
* **Use complex shell scripts** - extract to separate files
* **Hardcode paths** - use variables for flexibility
* **Break backward compatibility** without notice

<br><br>

## 🔍 Debugging Make

### Useful Commands
```bash
# See what make will run (dry run)
make -n test

# Print make database
make -p | less

# Debug variable values
make print-VERSION

# Trace execution
make --trace test
```

### Common Issues

**Command not found**
```bash
# Check if tool is installed
which golangci-lint || make install-linter
```

**Nothing to be done**
```bash
# Force rebuild
make -B build-go

# Or use .PHONY
.PHONY: test build clean
```

**Circular dependencies**
```bash
# Check dependency graph
make -d test 2>&1 | grep "Circular"
```

<br><br>

## 🚀 Advanced Features

### Multi-Project Support
```makefile
# Include project type
include .make/$(PROJECT_TYPE).mk

# Override for specific needs
test: test-$(PROJECT_TYPE)
```

### CI Integration
```makefile
# CI-specific targets
ci: lint test-ci coverage

# Skip interactive prompts
CI=true make release
```

### Cross-Platform Builds
```makefile
# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 make build-go
	GOOS=darwin GOARCH=amd64 make build-go
	GOOS=windows GOARCH=amd64 make build-go
```

<br><br>

## 📚 Resources

### Make Documentation
* [GNU Make Manual](https://www.gnu.org/software/make/manual/)
* [Make Tutorial](https://makefiletutorial.com/)
* [Makefile Best Practices](https://clarkgrubb.com/makefile-style-guide)

### Project Patterns
* Use `make help` to discover commands
* Check `.make/` directory for shared logic
* Read Makefile comments for usage notes