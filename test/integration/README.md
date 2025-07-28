# Integration Tests

This directory contains comprehensive integration tests for go-broadcast that test the complete end-to-end functionality.

## Test Structure

### `sync_test.go`
Tests the core synchronization workflows:
- **TestEndToEndSync**: Complete sync workflow with mocked GitHub/Git clients
- **TestConfigurationLoading**: Configuration parsing and validation
- **TestStateDiscovery**: State discovery system including branch parsing
- **TestTransformEngine**: File content transformation functionality

### `cli_test.go`
Tests the CLI interface:
- **TestCLICommands**: All CLI commands (sync, status, validate, version)
- **TestConfigurationExamples**: Validates all example configuration files
- **TestCLIFlags**: Various CLI flag combinations
- **TestCLIAliases**: Command aliases functionality

## Running Integration Tests

### Run all integration tests:
```bash
go test ./test/integration/... -v
```

### Run specific test suites:
```bash
# Test sync functionality
go test ./test/integration/... -run TestEndToEndSync -v

# Test CLI commands
go test ./test/integration/... -run TestCLICommands -v

# Test configuration examples
go test ./test/integration/... -run TestConfigurationExamples -v
```

### Run with race detection:
```bash
go test ./test/integration/... -race -v
```

## Test Coverage

These integration tests provide coverage for:

1. **Complete Sync Workflows**
   - Configuration loading and validation
   - State discovery from GitHub
   - File transformations
   - Repository cloning and modifications
   - Branch creation and commits
   - Pull request creation
   - Error handling and rollback

2. **CLI Interface**
   - All commands: sync, status, validate, version
   - Global flags: --config, --dry-run, --log-level
   - Command aliases
   - Help text and usage
   - Error scenarios

3. **Configuration System**
   - YAML parsing and validation
   - Default value handling
   - File mapping configurations
   - Transform configurations
   - Example file validation

4. **Edge Cases**
   - Missing configuration files
   - Invalid configurations  
   - Network failures (mocked)
   - Authentication failures (mocked)
   - Concurrent processing
   - Up-to-date targets (no-op scenarios)

## Mock Strategy

The integration tests use a comprehensive mocking strategy:

- **GitHub Client**: Mocked to simulate API responses without network calls
- **Git Client**: Mocked to simulate git operations without actual repositories
- **State Discoverer**: Mocked to provide controlled state scenarios
- **Transform Chain**: Mocked to test transformation pipelines

This approach allows testing complete workflows while maintaining:
- Fast execution (no network I/O)
- Reliable results (no external dependencies)
- Comprehensive coverage (including error scenarios)

## Test Data

Tests use:
- Temporary directories for file operations
- In-memory configuration objects
- Realistic mock data that mirrors actual GitHub API responses
- Various repository and branch scenarios

## Continuous Integration

These tests are designed to run in CI environments:
- No external dependencies
- No network access required
- Deterministic results
- Comprehensive error scenario coverage

## Adding New Tests

When adding new integration tests:

1. Follow the existing naming conventions
2. Use table-driven tests for multiple scenarios
3. Ensure proper mock setup and cleanup
4. Test both success and failure paths
5. Add documentation for complex test scenarios