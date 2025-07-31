# go-broadcast Logging Quick Reference

## 🚀 Essential Commands

```bash
# Basic logging levels
go-broadcast sync --log-level debug     # Debug level logging
go-broadcast sync --log-level info      # Info level logging (default)
go-broadcast sync --log-level warn      # Warning level logging
go-broadcast sync --log-level error     # Error level logging

# System diagnostics
go-broadcast diagnose                   # Collect system info

# Configuration validation and testing
go-broadcast validate --config sync.yaml      # Validate configuration
go-broadcast sync --dry-run --config sync.yaml   # Preview changes

# Note: Verbose flags (-v, -vv, -vvv) and component-specific debug flags
# (--debug-git, --debug-api, etc.) are planned features not yet implemented.
# Current implementation supports --log-level for basic debugging.
```

## 📋 Flag Reference

| Flag                | Description                       | Output Level |
|---------------------|-----------------------------------|--------------|
| `--log-level debug` | Debug level logging               | DEBUG        |
| `--log-level info`  | Info level logging (default)      | INFO         |
| `--log-level warn`  | Warning level logging             | WARN         |
| `--log-level error` | Error level logging               | ERROR        |
| `--dry-run`         | Preview changes without executing | All levels   |
| `--config <file>`   | Specify configuration file        | All levels   |

**Note**: Advanced verbose flags (`-v`, `-vv`, `-vvv`) and component-specific debug flags (`--debug-git`, `--debug-api`, etc.) are planned features not yet implemented in the current version.

## 🔧 Common Troubleshooting

```bash
# Authentication issues
go-broadcast sync --log-level debug     # Enable debug logging
echo $GITHUB_TOKEN | cut -c1-10         # Check token (first 10 chars)
gh auth status                           # Test GitHub CLI

# Configuration validation
go-broadcast validate --config sync.yaml

# Test configuration without changes
go-broadcast sync --dry-run --config sync.yaml

# Save debug session
go-broadcast sync --log-level debug 2> debug-$(date +%Y%m%d-%H%M%S).log

# System diagnostics
go-broadcast diagnose > diagnostics-$(date +%Y%m%d-%H%M%S).json
```

## 🌍 Environment Variables

| Variable       | Default | Description                       |
|----------------|---------|-----------------------------------|
| `GITHUB_TOKEN` | -       | GitHub authentication token       |
| `GH_TOKEN`     | -       | Alternative GitHub token variable |
| `NO_COLOR`     | -       | Disable colored output            |

**Note**: Environment variables for log level and format are planned features not yet implemented.

## 📄 Output Examples

### Text Format (Default)
```
time="13:32:17" level=info msg="Configuration changed" action=config_loaded component=audit
time="13:32:17" level=info msg="Syncing all configured targets" command=sync
time="13:32:17" level=debug msg="CLI initialized" config=examples/minimal.yaml dry_run=true
```

### System Diagnostics (JSON)
```json
{
  "timestamp": "2025-07-26T13:31:10.010068-04:00",
  "version": {
    "version": "dev",
    "go_version": "go1.24.5",
    "built_by": "source"
  },
  "system": {
    "os": "darwin",
    "arch": "arm64",
    "num_cpu": 10
  }
}
```

## 🚨 Quick Diagnostics

```bash
# System info for support
go-broadcast diagnose > diagnostics.json

# Check configuration
go-broadcast validate --config sync.yaml

# Test connectivity
gh api user                             # GitHub API access
git --version                           # Git availability
gh auth status                          # GitHub CLI authentication

# Test configuration without changes
go-broadcast sync --dry-run --config examples/minimal.yaml
```

## 🎯 Current Debugging Capabilities

```bash
# Basic debugging
go-broadcast sync --log-level debug --config sync.yaml

# Preview changes without executing
go-broadcast sync --dry-run --config sync.yaml

# Validate configuration syntax
go-broadcast validate --config sync.yaml

# Collect comprehensive diagnostics
go-broadcast diagnose

# Monitor authentication
gh auth status
```

## 🔍 Basic Log Analysis

### Debug Log Review
```bash
# Save debug session for analysis
go-broadcast sync --log-level debug 2> debug.log

# Review errors in debug log
grep -i error debug.log

# Check authentication events
grep -i auth debug.log

# Review configuration loading
grep -i config debug.log
```

### System Diagnostics Analysis
```bash
# Save diagnostics for support
go-broadcast diagnose > diagnostics.json

# Extract version information
jq '.version' diagnostics.json

# Check system environment
jq '.system' diagnostics.json

# Review tool versions
jq '.git_version, .gh_cli_version' diagnostics.json
```

## 📚 Developer Workflow Integration

For comprehensive go-broadcast development workflows, see [CLAUDE.md](../.github/CLAUDE.md) which includes:
- **Debugging go-broadcast procedures** with verbose logging examples
- **Component-specific debugging workflows** for targeted investigation  
- **Environment troubleshooting steps** for common setup issues
- **Development workflow integration** for effective debugging

## Related Documentation

- **Comprehensive Guide**: [logging.md](logging.md) - Complete logging system documentation
- **Troubleshooting Runbook**: [troubleshooting-runbook.md](troubleshooting-runbook.md) - Operational procedures
- **General Troubleshooting**: [troubleshooting.md](troubleshooting.md) - Common issue resolution
- **Main Documentation**: [README.md](../README.md) - Overview and quick start
- **Developer Workflows**: [CLAUDE.md](../.github/CLAUDE.md) - Complete development workflow integration

---

**💡 Pro Tip**: Start with `-v` for basic debugging, then add specific `--debug-*` flags for targeted investigation. Use `--json` when piping to analysis tools. For complete workflows, see [CLAUDE.md](../.github/CLAUDE.md).
