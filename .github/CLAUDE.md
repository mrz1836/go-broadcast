# CLAUDE.md

## 🤖 Welcome, Claude

This repository uses **`AGENTS.md`** as the single source of truth for:

* Coding conventions (naming, formatting, commenting, testing)
* Contribution workflows (branch prefixes, commit message style, PR templates)
* Release, CI, and dependency‑management policies
* Security reporting and governance links

> **TL;DR:** **Read `AGENTS.md` first.**  
> All technical or procedural questions are answered there.

---

## 🚀 go-broadcast Developer Workflow Guide

This section provides go-broadcast specific workflows while maintaining `AGENTS.md` as the authoritative source for general standards.

### 🔧 Essential Development Commands

**Quick Setup:**
```bash
# Install dependencies and validate environment
make mod-download
make install-stdlib

# Build the application locally
make build-go
# Or build directly: go build -o go-broadcast ./cmd/go-broadcast
```

**Core Development Workflow:**
```bash
# Standard development cycle - run before every commit
make test           # Fast linting + unit tests
make test-race      # Unit tests with race detector (slower)
make bench          # Run performance benchmarks

# Quality assurance
make lint           # Run golangci-lint
make coverage       # Generate and view test coverage
make govulncheck    # Scan for security vulnerabilities
```

### ⚡ Testing and Validation

**Unit Testing:**
```bash
# Fast testing (recommended for development)
make test-no-lint   # Skip linting, run only tests
make test-short     # Skip integration tests

# Comprehensive testing
make test-ci        # Full CI test suite with race detection
make test-cover     # Unit tests with coverage report
```

**Integration Testing:**
```bash
# Phase-specific integration tests
make test-integration-complex    # Phase 1: Complex workflows
make test-integration-advanced   # Phase 2: Advanced scenarios  
make test-integration-network    # Phase 3: Network edge cases
make test-integration-all        # All integration test phases
```

**Configuration Validation:**
```bash
# Validate go-broadcast configurations
./go-broadcast validate --config examples/minimal.yaml
./go-broadcast validate --config examples/sync.yaml

# Test dry-run functionality
./go-broadcast sync --dry-run --config examples/minimal.yaml

# Check configuration status
./go-broadcast status --config examples/minimal.yaml
```

### 🧪 Fuzz Testing Workflow

go-broadcast includes comprehensive fuzz testing for critical components:

**Run Fuzz Tests:**
```bash
# Run all fuzz tests (limited iterations for speed)
make test-fuzz

# Run specific fuzz tests with more iterations
go test -fuzz=FuzzConfigParsing -fuzztime=30s ./internal/config
go test -fuzz=FuzzTransformChain -fuzztime=30s ./internal/transform
go test -fuzz=FuzzRegexReplacement -fuzztime=30s ./internal/transform

# Generate new corpus entries
go run ./cmd/generate-corpus
```

**Fuzz Test Coverage:**
- **Config parsing** (`internal/config`): YAML validation, repository name validation
- **Transformations** (`internal/transform`): Variable substitution, regex replacement chains
- **GitHub API** (`internal/gh`): Response parsing and error handling
- **Git operations** (`internal/git`): Command parsing and output handling

### 📊 Performance Testing and Benchmarking

**Benchmark Execution:**
```bash
# Run all benchmarks with memory profiling
make bench

# Run specific component benchmarks
go test -bench=. -benchmem ./internal/cache
go test -bench=. -benchmem ./internal/algorithms
go test -bench=. -benchmem ./internal/transform

# Performance profiling with built-in demo
go run ./cmd/profile_demo
# Results saved to: ./profiles/final_demo/
```

**Benchmark Categories:**
- **Core algorithms**: Binary detection, content comparison, batch processing
- **Cache operations**: TTL cache performance across different hit rates
- **API simulation**: GitHub API latency simulation with various response times
- **Concurrency**: Goroutine performance across different worker pool sizes
- **Memory usage**: Allocation patterns and memory efficiency

**Performance Analysis:**
```bash
# Generate performance reports
go test -bench=. -benchmem -cpuprofile=cpu.prof ./internal/worker
go test -bench=. -benchmem -memprofile=mem.prof ./internal/git

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

### 🛠️ Troubleshooting Quick Reference

**Common Development Issues:**

1. **Build Failures:**
   ```bash
   # Clean and rebuild
   make clean-mods
   make mod-download
   make build-go
   ```

2. **Test Failures:**
   ```bash
   # Run tests with verbose output
   go test -v ./...
   
   # Run specific failing test
   go test -v -run TestSpecificFunction ./internal/package
   ```

3. **Linting Errors:**
   ```bash
   # Fix formatting issues
   make fumpt          # Apply gofumpt formatting
   goimports -w .      # Fix import statements
   
   # Check linting rules
   make lint-version   # Show linter version
   make lint           # Run all linters
   ```

4. **Performance Issues:**
   ```bash
   # Profile memory usage
   go test -bench=. -memprofile=mem.prof ./internal/component
   go tool pprof mem.prof
   
   # Monitor goroutines
   go test -bench=. -trace=trace.out ./internal/worker
   go tool trace trace.out
   ```

**Debugging go-broadcast:**
```bash
# Enable debug logging
./go-broadcast sync --log-level debug --config examples/minimal.yaml

# Collect diagnostic information
./go-broadcast diagnose > diagnostics.json
```

**Note**: Component-specific debug flags (`--debug-git`, `--debug-api`, etc.) are planned features not yet implemented.

**Environment Issues:**
```bash
# Check GitHub authentication
gh auth status

# Verify Go environment
go version
go env GOPATH GOROOT

# Validate dependencies
go mod verify
govulncheck ./...
```

### 📚 Documentation Navigation

**Core Documentation:**
- [`README.md`](../README.md) - Project overview and quick start
- [`AGENTS.md`](AGENTS.md) - **Primary authority** for all development standards
- [`CONTRIBUTING.md`](CONTRIBUTING.md) - Contribution guidelines
- [`CODE_STANDARDS.md`](CODE_STANDARDS.md) - Detailed coding standards

**Technical Documentation:**
- [`docs/logging.md`](../docs/logging.md) - Comprehensive logging guide
- [`docs/logging-quick-ref.md`](../docs/logging-quick-ref.md) - Quick logging reference
- [`docs/troubleshooting.md`](../docs/troubleshooting.md) - General troubleshooting
- [`docs/troubleshooting-runbook.md`](../docs/troubleshooting-runbook.md) - Operational runbook

**Performance and Optimization:**
- [`docs/benchmarking-profiling.md`](../docs/benchmarking-profiling.md) - Complete benchmarking guide
- [`docs/profiling-guide.md`](../docs/profiling-guide.md) - Advanced profiling techniques
- [`docs/performance-optimization.md`](../docs/performance-optimization.md) - Optimization best practices

**Configuration Examples:**
- [`examples/`](../examples/) - Complete configuration examples directory
- [`examples/README.md`](../examples/README.md) - Example configurations overview
- [`examples/minimal.yaml`](../examples/minimal.yaml) - Simplest configuration
- [`examples/sync.yaml`](../examples/sync.yaml) - Comprehensive example

---

## 🔧 Linter Knowledge & Go Best Practices

### Common Linter Issues and How to Avoid Them

**Security Issues (gosec):**
- Use `0600` for file permissions, not `0644` for sensitive files
- Use `0750` for directory permissions, not `0755`
- Always set `MinVersion: tls.VersionTLS12` for TLS configs
- Be careful with file path variables - add `//nolint:gosec // reason` if controlled

**Error Handling (errcheck):**
- Always check error returns: `if err := foo(); err != nil { ... }`
- For intentionally ignored errors, use `_ = foo()` 
- For deferred cleanup, use `defer func() { _ = file.Close() }()`
- Never ignore errors from `os.Setenv`, `os.Unsetenv`, or resource cleanup

**Error Wrapping (err113):**
- Create static error variables: `var ErrNotFound = errors.New("not found")`
- Wrap with context: `fmt.Errorf("%w: additional context", ErrNotFound, details)`
- Avoid dynamic error messages: use static errors + wrapping instead

**Code Quality (revive):**
- Add comments to exported variables: `// ErrFoo indicates that foo failed`
- Add comments to exported constants: `// FormatJSON represents JSON format`
- Remove empty blocks or add meaningful implementation
- Remove unused parameters or rename to `_` (e.g., `func foo(_ context.Context, data string)`)
- Use descriptive parameter names, avoid redefining builtins like `max`
- For "stuttering" types like `export.ExportFormat`, either rename or use `//nolint:revive // reason`
- CLI flag variables in cmd packages are legitimate globals requiring `//nolint:gochecknoglobals`

**Global Variables (gochecknoglobals):**
- Minimize global state - prefer dependency injection
- For legitimate globals (CLI commands, constants), add `//nolint:gochecknoglobals // reason`
- Consider using `sync.Once` for initialization

**Code Efficiency (staticcheck):**
- Use `fmt.Fprintf(w, "text %s", arg)` instead of `w.WriteString(fmt.Sprintf("text %s", arg))`
- Use tagged switch statements for enum-like comparisons
- Expand `math.Pow(x, 2)` to `x * x`

**Memory Optimization (prealloc):**
- Pre-allocate slices when size is known: `make([]Type, 0, knownSize)`
- Use capacity hints for better performance

### Linter Configuration Insights

This project uses 60+ linters via golangci-lint with strict standards. Key linters:
- **Security**: gosec, forbidigo
- **Correctness**: errcheck, govet, staticcheck
- **Style**: revive, gofmt, gofumpt
- **Performance**: prealloc, ineffassign
- **Maintainability**: gochecknoglobals, unused

**Major Linter Categories Fixed:**
1. **Security (gosec)**: Fixed 9 issues - file permissions (0600/0750), controlled file paths
2. **Error Handling (errcheck + err113)**: Fixed 50 issues - proper error checking, static error wrapping
3. **Global Variables (gochecknoglobals)**: Fixed 40 issues - mostly CLI flags with proper justification
4. **Code Quality (revive)**: 50 issues - export comments, naming, unused parameters
5. **Performance (prealloc)**: Pre-allocate slices when size is known
6. **Formatting (gofmt/gofumpt)**: Automatic formatting fixes

The configuration prioritizes security and correctness over convenience. When adding `//nolint` comments, always include a specific reason.

**Common Fixes Applied:**
- Changed `0644` → `0600` for file permissions
- Changed `0755` → `0750` for directory permissions  
- Added static error variables with `fmt.Errorf("%w", staticErr)`
- Renamed unused `ctx` parameters to `_`
- Added export comments to public APIs
- Used `//nolint:revive // reason` for contextually clear names

---

## ✅ Pre-Development Checklist

Before starting any development work:

1. **Read `AGENTS.md` thoroughly** - Understand all conventions and standards
2. **Set up development environment:**
   ```bash
   make mod-download
   make install-stdlib
   pre-commit install  # Optional but recommended
   ```
3. **Validate environment:**
   ```bash
   make test          # Ensure all tests pass
   gh auth status     # Verify GitHub authentication
   ```
4. **Review relevant documentation** based on your planned changes

## 🎯 Development Workflow Summary

1. **Study `AGENTS.md`** - Make sure every change respects established standards
2. **Follow branch‑prefix and commit‑message standards** - Required for CI automation
3. **Never tag releases** - Only repository code‑owners handle releases
4. **Run comprehensive testing:**
   ```bash
   make test         # Fast linting + unit tests
   make test-race    # Race condition detection
   make govulncheck  # Security vulnerability scan
   ```
5. **Validate go-broadcast functionality:**
   ```bash
   ./go-broadcast validate --config examples/minimal.yaml
   ./go-broadcast sync --dry-run --config examples/minimal.yaml
   ```

## 🚨 Important Reminders

- **`AGENTS.md` is the ultimate authority** - When in doubt, refer to it first
- **Test thoroughly** - Use both unit tests and go-broadcast validation commands
- **Follow Go conventions** - Context-first design, interface composition, no global state
- **Security first** - Run `govulncheck` and validate all external dependencies
- **Performance matters** - Use benchmarks to validate optimizations

If you encounter conflicting guidance elsewhere, `AGENTS.md` wins.  
Questions or ambiguities? Open a discussion or ping a maintainer instead of guessing.

Happy hacking! 🚀