# go-broadcast Logging Quick Reference

## 🚀 Essential Commands

```bash
# Basic verbose output
go-broadcast sync -v                    # Debug level
go-broadcast sync -vv                   # Trace level  
go-broadcast sync -vvv                  # Trace with caller info

# Component debugging
go-broadcast sync --debug-git           # Git commands
go-broadcast sync --debug-api           # GitHub API
go-broadcast sync --debug-transform     # File transformations
go-broadcast sync --debug-config        # Configuration validation
go-broadcast sync --debug-state         # State discovery

# JSON output
go-broadcast sync --json -v             # Structured JSON output
go-broadcast sync --log-format json     # Alternative syntax

# System diagnostics
go-broadcast diagnose                   # Collect system info
```

## 📋 Flag Reference

| Flag | Description | Output Level |
|------|-------------|--------------|
| `-v` | Debug level logging | DEBUG |
| `-vv` | Trace level logging | TRACE |
| `-vvv` | Trace with file:line info | TRACE + caller |
| `--debug-git` | Git command details | DEBUG/TRACE |
| `--debug-api` | GitHub API requests/responses | DEBUG/TRACE |
| `--debug-transform` | File transformation details | DEBUG/TRACE |
| `--debug-config` | Configuration validation | DEBUG/TRACE |
| `--debug-state` | State discovery process | DEBUG/TRACE |
| `--json` | JSON structured output | All levels |
| `--log-format json` | JSON output (alternative) | All levels |

## 🔧 Common Troubleshooting

```bash
# Authentication issues
go-broadcast sync --debug-git -v
echo $GITHUB_TOKEN | cut -c1-10        # Check token
gh auth status                          # Test GitHub CLI

# Performance analysis
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms > 1000) | "\(.duration_ms)ms \(.operation)"'

# Find errors in logs
go-broadcast sync --json 2>&1 | jq 'select(.level=="error")'

# Save debug session
go-broadcast sync -vvv --debug-git 2> debug-$(date +%Y%m%d-%H%M%S).log

# Memory/performance monitoring
go-broadcast sync --json --debug-api 2>&1 | \
  jq 'select(.component=="github-api") | {operation, duration_ms, error}'
```

## 🌍 Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GO_BROADCAST_LOG_LEVEL` | `info` | Default log level |
| `GO_BROADCAST_LOG_FORMAT` | `text` | Default output format |
| `GO_BROADCAST_DEBUG` | `false` | Enable all debug flags |
| `NO_COLOR` | - | Disable colored output |
| `GITHUB_TOKEN` | - | GitHub authentication token |

## 📄 Output Examples

### Text Format (Default)
```
15:04:05 INFO  Starting broadcast sync     version=1.2.3
15:04:05 DEBUG Config loaded successfully  path=.broadcast.yaml targets=3
15:04:06 WARN  Rate limit approaching      remaining=100
```

### JSON Format
```json
{
  "@timestamp": "2024-01-15T15:04:05.123Z",
  "level": "info",
  "message": "Starting broadcast sync",
  "component": "cli",
  "version": "1.2.3",
  "correlation_id": "a1b2c3d4"
}
```

## 🚨 Quick Diagnostics

```bash
# System info for support
go-broadcast diagnose > diagnostics.json

# Check configuration
go-broadcast validate --debug-config -v

# Test connectivity
gh api user                             # GitHub API access
git --version                           # Git availability

# Log analysis pipeline
go-broadcast sync --json 2>&1 | \
  jq 'select(.duration_ms) | {operation, duration_ms}' | \
  sort_by(.duration_ms) | reverse
```

## 🎯 Debug Combinations

```bash
# Full debugging (maximum verbosity)
go-broadcast sync -vvv --debug-git --debug-api --debug-transform --debug-config --debug-state

# Performance focus
go-broadcast sync --json --debug-api 2>&1 | jq 'select(.duration_ms > 5000)'

# Security audit
go-broadcast sync --json 2>&1 | jq 'select(.component=="audit")'

# Git troubleshooting
go-broadcast sync --debug-git -vv 2>&1 | grep -E "(git|clone|push|branch)"

# API rate limiting
go-broadcast sync --debug-api --json 2>&1 | jq 'select(.rate_limit_remaining)'
```

## 🔍 Log Analysis Tips

### Find Bottlenecks
```bash
# Operations taking >5 seconds
jq 'select(.duration_ms > 5000) | {operation, duration_ms, repo}' < logs.json

# Most common operations
jq -r 'select(.duration_ms) | .operation' < logs.json | sort | uniq -c | sort -rn
```

### Error Analysis  
```bash
# All errors with context
jq 'select(.level=="error") | {message, component, operation, error}' < logs.json

# Failed operations with timing
jq 'select(.status=="failed") | {operation, duration_ms, error}' < logs.json
```

### Security Monitoring
```bash
# Authentication events
jq 'select(.component=="audit" and .event=="authentication")' < logs.json

# Repository access
jq 'select(.event=="repo_access") | {repo, action, user}' < logs.json
```

## 📚 See Also

- **Comprehensive Guide**: [docs/logging.md](logging.md)
- **Troubleshooting**: [docs/troubleshooting-runbook.md](troubleshooting-runbook.md)
- **Main Documentation**: [README.md](../README.md#-logging-and-debugging)

---

**💡 Pro Tip**: Start with `-v` for basic debugging, then add specific `--debug-*` flags for targeted investigation. Use `--json` when piping to analysis tools.