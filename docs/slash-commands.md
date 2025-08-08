# 🚀 go-broadcast Slash Commands Reference

> Powerful Claude Code commands that leverage 26 specialized AI agents for automated workflows

## 📋 Table of Contents

- [Overview](#-overview)
- [Quick Start](#-quick-start)
- [Command Categories](#-command-categories)
  - [Testing & Quality](#-testing--quality)
  - [Security & Compliance](#️-security--compliance)
  - [Dependency Management](#-dependency-management)
  - [Debugging & Diagnostics](#-debugging--diagnostics)
  - [Documentation & Release](#-documentation--release)
  - [Code Improvement](#-code-improvement)
  - [Sync & Operations](#-sync--operations)
  - [Automation](#-automation)
- [Usage Examples](#-usage-examples)
- [Command Details](#-command-details)
- [Best Practices](#-best-practices)

## 🌟 Overview

The go-broadcast slash command system provides 20 powerful commands that coordinate multiple AI agents to automate complex development workflows. Each command is designed to save significant time by handling multi-step processes automatically.

### Key Features

- **🚀 Intelligent Agent Coordination** - Commands automatically select and coordinate the right agents
- **⚡ Parallel Execution** - Multiple agents work simultaneously for maximum efficiency
- **🎯 Smart Defaults** - Commands work with minimal input but accept specific targets
- **📊 Comprehensive Reporting** - Detailed feedback on actions taken and results achieved
- **🛡️ Safety First** - Automatic validation and testing to prevent breaking changes

## ⚡ Quick Start

```bash
# Run comprehensive tests with auto-fix
/test

# Perform security audit
/security

# Fix all issues in current PR
/pr-fix

# Prepare a release
/release v1.2.0
```

## 📚 Command Categories

### 🧪 Testing & Quality

| Command | Description | Agents Used |
|---------|-------------|-------------|
| `/test` | Run comprehensive tests with automatic fixes | test-commander, coverage-maintainer |
| `/fuzz` | Security-focused fuzz testing | fuzz-test-guardian |
| `/quality` | Enforce code quality with 60+ linters | go-quality-enforcer, test-commander |
| `/bench` | Run performance benchmarks | benchmark-runner, benchmark-analyst |

### 🛡️ Security & Compliance

| Command | Description | Agents Used |
|---------|-------------|-------------|
| `/security` | Comprehensive security audit | security-auditor, compliance-checker, dependabot-coordinator |
| `/audit` | Quick vulnerability scan for CI/CD | security-auditor |

### 📦 Dependency Management

| Command | Description | Agents Used |
|---------|-------------|-------------|
| `/deps` | Review and update all dependencies | dependabot-coordinator, dependency-upgrader, breaking-change-detector |
| `/breaking` | Analyze updates for breaking changes | breaking-change-detector |

### 🐛 Debugging & Diagnostics

| Command | Description | Agents Used |
|---------|-------------|-------------|
| `/debug` | Deep debug analysis for issues | diagnostic-specialist, debugging-expert |
| `/diagnose` | Collect diagnostic information | diagnostic-specialist |
| `/profile` | Performance profiling and optimization | performance-profiler, benchmark-analyst |

### 📚 Documentation & Release

| Command | Description | Agents Used |
|---------|-------------|-------------|
| `/docs` | Update documentation for changes | documentation-maintainer |
| `/changelog` | Generate changelog from commits | changelog-generator |
| `/release` | Complete release workflow | test-commander, security-auditor, changelog-generator, release-manager, documentation-maintainer |

### 🔧 Code Improvement

| Command | Description | Agents Used |
|---------|-------------|-------------|
| `/dedupe` | Find and remove duplicate code | code-deduplicator, refactoring-specialist |
| `/refactor` | Refactor code for better structure | refactoring-specialist |
| `/debt` | Technical debt analysis and tracking | tech-debt-tracker |

### 🔄 Sync & Operations

| Command | Description | Agents Used |
|---------|-------------|-------------|
| `/sync` | Validate and optimize sync operations | sync-orchestrator, config-validator |
| `/workflow` | Fix and optimize GitHub Actions | workflow-optimizer |

### 🤖 Automation

| Command | Description | Agents Used |
|---------|-------------|-------------|
| `/pr-fix` | Fix all issues in current PR | pr-automation-manager, test-commander, go-quality-enforcer |
| `/integrate` | Run phased integration tests | integration-test-manager |
| `/triage` | Manage and organize GitHub issues | issue-triage-bot |

## 💡 Usage Examples

### Basic Usage

```bash
# Run all tests
/test

# Test specific package
/test ./internal/sync

# Quick security check
/audit

# Full security scan
/security
```

### With Arguments

```bash
# Analyze specific dependency
/breaking github.com/some/package@v2.0.0

# Debug specific issue
/debug "sync operation failing with timeout"

# Profile specific operation
/profile memory

# Prepare specific release
/release v1.2.0
```

### Advanced Workflows

```bash
# Fix PR with all checks
/pr-fix 123

# Run integration tests phase
/integrate network

# Triage new issues
/triage new

# Refactor specific file
/refactor @internal/sync/handler.go
```

## 📖 Command Details

### /test - Comprehensive Testing

**Purpose**: Ensures all tests pass with proper coverage

**Features**:
- Runs tests with race detection
- Automatically fixes failing tests
- Maintains >85% coverage
- Updates coverage reports

**Usage**:
```bash
/test                    # All tests
/test ./pkg/...          # Specific package
/test integration        # Integration tests only
```

### /security - Security Audit

**Purpose**: Comprehensive security analysis

**Features**:
- Runs multiple security scanners in parallel
- Checks dependencies for vulnerabilities
- Scans for exposed secrets
- Verifies compliance standards

**Usage**:
```bash
/security                # Full audit
/security critical       # Critical issues only
```

### /release - Release Management

**Purpose**: Complete release workflow

**Features**:
- Runs all tests and security checks
- Generates changelog
- Creates tags and GitHub release
- Updates documentation
- Publishes artifacts

**Usage**:
```bash
/release v1.2.0         # Create release
/release v1.2.0-beta.1  # Pre-release
```

### /pr-fix - PR Auto-Fix

**Purpose**: Resolve all PR blockers

**Features**:
- Fixes failing tests
- Resolves linting issues
- Updates PR metadata
- Handles merge conflicts
- Ensures CI passes

**Usage**:
```bash
/pr-fix                 # Current PR
/pr-fix 456            # Specific PR
```

## 🎯 Best Practices

### 1. **Start Simple**
Begin with basic commands like `/test` and `/quality` before using complex workflows

### 2. **Use in CI/CD**
Commands like `/audit` and `/test` are designed for automation in CI/CD pipelines.

### 3. **Combine Commands**
Use commands sequentially for complete workflows by running them one after another.

### 4. **Leverage Arguments**
Be specific when needed:
```bash
/refactor @internal/complex-module.go
/debug "specific error message"
```

### 5. **Monitor Agent Output**
Watch for agent recommendations and warnings in the output

### 6. **Regular Maintenance**
Use these commands regularly:
- `/deps` - Weekly dependency review
- `/debt analyze` - Monthly tech debt review
- `/security` - Before each release

## 🔍 Troubleshooting

### Command Not Found
```bash
# List all available commands
/help

# Ensure commands directory exists
ls .claude/commands/
```

### Agent Errors
- Check agent output for specific error messages
- Verify required tools are available
- Ensure proper GitHub authentication

### Performance Issues
- Use `/profile` to identify bottlenecks
- Run `/workflow` to optimize CI/CD
- Consider using specific targets instead of full scans

## 🚀 Advanced Features

### Parallel Execution
Many commands use parallel agents for speed:
- `/security` runs 3 security agents simultaneously
- `/pr-fix` coordinates 3 fix agents in parallel
- `/deps` analyzes updates with 3 specialized agents

### Sequential Workflows
Some commands use sequential processing:
- `/release` ensures each step completes before the next
- `/debug` gathers diagnostics before deep analysis
- `/dedupe` finds duplicates before refactoring

### Smart Context
Commands understand context:
- File references with `@filename`
- Issue/PR numbers
- Version tags
- Error messages

## 📈 Command Performance

| Command | Typical Duration | Agents | Parallel |
|---------|-----------------|---------|----------|
| `/test` | 30-60s | 2 | ✅ |
| `/security` | 45-90s | 3 | ✅ |
| `/release` | 2-5min | 5 | ❌ |
| `/pr-fix` | 60-120s | 3 | ✅ |
| `/deps` | 30-60s | 3 | ✅ |

## 🔗 Related Documentation

- [Sub-Agents Guide](sub-agents.md) - Detailed information about each agent
- [Enhanced Troubleshooting Guide](troubleshooting.md) - Common issues and solutions
- [Performance Guide](performance-guide.md) - Performance tuning and optimization
- [Enhanced Logging Guide](logging.md) - Debugging and monitoring
- [AGENTS.md](../.github/AGENTS.md) - AI compliance and conventions
- [Claude Code Docs](https://docs.anthropic.com/en/docs/claude-code) - Official documentation

---

<div align="center">

**[⬆ back to top](#-table-of-contents)**

</div>
