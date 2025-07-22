# File Sync Orchestrator - Product Requirements Document

## Executive Summary

The File Sync Orchestrator is a stateless Go CLI tool with optional TUI mode that synchronizes files from a template repository to multiple GitHub repositories across organizations. It leverages GitHub as the single source of truth, integrates with Claude Code for intelligent commit messages and PR descriptions, and follows a progressive enhancement design philosophy.

## Problem Statement

Development teams managing multiple microservices or repositories often need to keep common files (configs, scripts, workflows) synchronized across projects. Current solutions require manual copying, are error-prone, and lack proper tracking. Teams need a tool that:

- Automatically syncs files from a template repository to multiple targets
- Works across multiple GitHub organizations
- Provides clear visibility into sync status
- Generates meaningful PR descriptions
- Handles file transformations (e.g., replacing repository names)
- Recovers gracefully from failures

## Solution Overview

### Core Philosophy

1. **Stateless Architecture**: All state is derived from GitHub, never stored locally
2. **Progressive Enhancement**: CLI-first design with optional TUI for complex operations
3. **Intelligent Automation**: Claude Code integration for smart commit messages and PR descriptions
4. **Full Testability**: Comprehensive mocking and test coverage
5. **Production-Ready**: Built on proven go-template foundation

### Key Concepts

#### 1. GitHub as Source of Truth
- No local database or state files
- State discovered from branch names, PR metadata, and commit history
- Any operation can be safely retried
- System can recover from any failure

#### 2. Multi-Organization Support
- Leverages `gh auth` for authentication
- Works with any repository the user has access to
- Handles SSO/SAML authentication
- Supports personal, organizational, and cross-org repositories

#### 3. Smart File Transformations
- Automatic repository name replacement
- Template variable substitution
- Context-aware transformations based on file type
- Preview changes before applying

#### 4. Claude Code Integration
- Generate commit messages from file changes
- Create rich PR descriptions with context
- Help resolve merge conflicts
- Batch PR description generation

## Functional Requirements

### Core Features

#### 1. Repository Synchronization
- **Sync Single Repository**: Update files in one target repository
- **Sync Multiple Repositories**: Batch sync across configured targets
- **Selective Sync**: Choose specific files or repositories
- **Dry Run Mode**: Preview changes without applying them

#### 2. State Discovery
- **Automatic State Recovery**: Rebuild complete state from GitHub
- **Branch Detection**: Find existing sync branches
- **PR Discovery**: Locate open sync PRs with metadata
- **Last Sync Tracking**: Determine what was previously synced

#### 3. File Management
- **Pattern Matching**: Support glob patterns for file selection
- **Path Mapping**: Map source paths to different destination paths
- **File Transformations**: Apply context-aware content changes
- **Conflict Detection**: Identify and help resolve conflicts

#### 4. User Interfaces

##### CLI Mode
- Command-line interface for automation and scripting
- Clear console output with progress indicators
- Structured logging for debugging
- Exit codes for CI/CD integration

##### TUI Mode
- Interactive terminal UI for complex operations
- Repository tree view with status indicators
- Real-time progress tracking
- Log viewer with search and filtering
- Keyboard navigation and shortcuts

### User Experience

#### CLI Workflow
```bash
# Basic sync
file-sync sync

# Force specific mode
file-sync sync --cli
file-sync sync --tui

# Other operations
file-sync status
file-sync state discover
file-sync pr list
file-sync logs tail
```

#### TUI Layout
```
┌─────────────────┬──────────────────────────┐
│  Repositories   │     Sync Status          │
│  (TreeView)     │     (Table)              │
│                 ├──────────────────────────┤
│                 │     Progress             │
│                 │     (Custom Widget)      │
├─────────────────┴──────────────────────────┤
│              Logs (TextView)                │
└─────────────────────────────────────────────┘
```

### Configuration

#### sync.yaml Structure
```yaml
version: 2
source:
  repo: "org/template-repo"
  branch: "main"

defaults:
  branch_prefix: "sync/template"
  pr_labels: ["automated-sync"]
  transforms:
    - type: "smart-repo"
    - type: "template-var"
  auto_transform:
    - "*.md"
    - "go.mod"
    - ".github/workflows/*.yml"

targets:
  - repo: "org1/service-a"
    files:
      - src: "shared/config.yaml"
        dest: "config.yaml"
```

## Technical Requirements

### Architecture Principles

1. **Interface-Driven Design**
	- All external dependencies behind interfaces
	- Mockable for testing
	- Swappable implementations

2. **Concurrent Operations**
	- Thread-safe UI updates
	- Parallel repository processing where possible
	- Proper synchronization primitives

3. **Error Handling**
	- Graceful degradation
	- Clear error messages
	- Recovery strategies
	- Detailed logging

### Component Architecture

#### Core Components
- **Sync Engine**: Orchestrates sync operations
- **State Discovery**: Rebuilds state from GitHub
- **Transform Engine**: Applies file transformations
- **GitHub Client**: Wraps gh CLI commands
- **Git Client**: Handles git operations
- **Claude Client**: Integrates with Claude Code

#### UI Components
- **CLI Engine**: Command-line interface
- **TUI Engine**: Terminal UI with tview
- **UI Updater**: Abstraction for progress updates

### External Dependencies

#### Required Tools
- `gh` CLI for GitHub operations
- `git` for repository operations
- `claude-code` (optional) for intelligent messages

#### Go Dependencies
- `cobra` - CLI framework
- `tview` - TUI framework
- `testify` - Testing framework
- `logrus` - Structured logging
- `yaml.v3` - Configuration parsing

### File Structure
```
file-sync/
├── cmd/file-sync/       # Main application
├── internal/
│   ├── cli/            # CLI commands
│   ├── config/         # Configuration
│   ├── sync/           # Sync engine
│   ├── gh/             # GitHub wrapper
│   ├── git/            # Git operations
│   ├── state/          # State discovery
│   ├── transform/      # Transformers
│   ├── claude/         # Claude integration
│   └── tui/            # TUI components
├── test/
│   ├── mocks/          # Mock implementations
│   └── fixtures/       # Test data
└── examples/           # Usage examples
```

## Implementation Guidelines

### Development Approach

1. **Start Simple**
	- Build core sync engine with basic CLI
	- Add state discovery from GitHub
	- Implement file transformations
	- Layer on TUI and Claude integration

2. **Test-Driven Development**
	- Write interfaces first
	- Create mocks for external dependencies
	- Build comprehensive test suite
	- Add integration tests last

3. **Progressive Enhancement**
	- CLI must work standalone
	- TUI adds visual feedback
	- Claude enhances but isn't required
	- Each layer is optional

### Key Implementation Notes

#### State Discovery Logic
- Parse branch names for metadata (timestamp, source commit)
- Extract PR metadata from description
- Compare source and target commits
- Build complete sync state

#### Transformation System
- Chain multiple transformers
- Pass context variables (repo name, org, etc.)
- Support custom patterns
- Preview before applying

#### Claude Integration
- Prepare structured context for Claude
- Fall back to templates if unavailable
- Cache responses where appropriate
- Allow regeneration and editing

### Testing Strategy

1. **Unit Tests**
	- Mock all external dependencies
	- Test each component in isolation
	- Table-driven tests for transformations
	- High coverage target (>80%)

2. **Integration Tests**
	- Optional with environment flag
	- Test against real GitHub API
	- Use dedicated test organization
	- Clean up after tests

3. **Benchmarks**
	- State discovery performance
	- Transformation speed
	- Concurrent operations

4. **Fuzz Tests**
	- Transformation edge cases
	- Input validation
	- Parser robustness

## Success Criteria

1. **Functionality**
	- Successfully sync files across multiple repositories
	- Recover state from GitHub without local storage
	- Generate meaningful PR descriptions with Claude
	- Handle conflicts gracefully

2. **Performance**
	- Sync 10 repositories in under 30 seconds
	- State discovery under 5 seconds for 50 repos
	- Responsive TUI with <100ms updates

3. **Reliability**
	- 100% idempotent operations
	- Graceful handling of API rate limits
	- Recovery from network failures
	- No data loss scenarios

4. **User Experience**
	- Clear feedback in both CLI and TUI modes
	- Intuitive keyboard shortcuts
	- Helpful error messages
	- Comprehensive documentation

## Future Enhancements

- **Watch Mode**: Auto-sync on template changes
- **GitHub App**: Organization-wide deployment
- **Metrics Dashboard**: Sync statistics and history
- **Plugin System**: Custom hooks and transformations
- **Conflict Resolution UI**: Interactive merge tools
- **Template Library**: Reusable Claude prompts

## Appendix: Key Design Decisions

### Why Stateless?
- Simplifies disaster recovery
- Enables running from anywhere
- Reduces complexity
- Improves reliability

### Why Progressive Enhancement?
- Maintains scriptability
- Supports CI/CD pipelines
- Enhances developer experience
- Allows gradual adoption

### Why Claude Integration?
- Meaningful commit messages
- Rich PR descriptions save review time
- Context-aware suggestions
- Reduces manual documentation

### Why Based on go-template?
- Production-ready foundation
- Established patterns
- Comprehensive tooling
- AI-friendly structure
