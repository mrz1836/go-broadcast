# go-broadcast Logging Guide

## Overview
go-broadcast provides comprehensive logging capabilities designed for debugging, monitoring, and troubleshooting. The logging system is built on structured logging principles with automatic sensitive data redaction.

## Logging Architecture

### Components
1. **Core Logger**: Built on logrus with custom enhancements
2. **Debug Subsystems**: Component-specific debug flags
3. **Performance Metrics**: Automatic operation timing
4. **Security Layer**: Automatic sensitive data redaction
5. **Structured Output**: JSON format for log aggregation

### Log Destinations
- **stderr**: All logs (keeps stdout clean for program output)
- **stdout**: User-facing messages only
- **Files**: Via shell redirection or log rotation tools

## Verbose Flags (-v, -vv, -vvv)

The verbose flags provide intuitive control over logging detail:

### -v (Debug Level)
Shows detailed operation progress and debugging information:
```
15:04:05 DEBUG Starting sync operation component=sync-engine
15:04:05 DEBUG Discovering source repository state component=state-discovery
15:04:06 DEBUG Git command completed duration_ms=1023 exit_code=0
```

### -vv (Trace Level)
Shows very detailed debugging information including internal operations:
```
15:04:05 [TRACE] Validating config version version=1.0
15:04:05 [TRACE] Variable replacement variable=REPO_NAME value=go-broadcast
15:04:06 [TRACE] Response body response={"data":{"repository":{"name":"go-broadcast"}}}
```

### -vvv (Trace with Caller Info)
Adds file:line information for deep debugging:
```
15:04:05.123 [TRACE] git.go:142 Executing git command args=[status --porcelain]
15:04:05.234 [TRACE] transform.go:89 Content before size=1024
15:04:05.235 [TRACE] transform.go:92 Content after size=1048 diff=24
```

## Debug Flags

### --debug-git
Detailed git command execution logging:
- Full command lines with arguments
- Working directory and environment
- Real-time stdout/stderr output
- Exit codes and timing

Example output:
```
DEBUG Executing git command command=/usr/bin/git args=[clone --depth=1] dir=/tmp/broadcast-123
[TRACE] stdout: Cloning into 'repo'...
[TRACE] stderr: Receiving objects: 100% (547/547), done.
DEBUG Git command completed duration_ms=2341 exit_code=0
```

### --debug-api
GitHub API request/response logging:
- Request parameters and headers (redacted)
- Response sizes and timing
- Rate limit information
- Error details

Example output:
```
DEBUG GitHub CLI request args=[api repos/owner/repo] timestamp=2024-01-15T10:30:45Z
[TRACE] Request field field=state:open
DEBUG GitHub CLI response duration_ms=234 response_size=4096 error=<nil>
```

### --debug-transform
File transformation and template processing:
- Variable substitutions
- Content size changes
- Before/after comparisons (small files)
- Transform timing

Example output:
```
DEBUG Starting transformation content_size=1024 variables=5 type=text
[TRACE] Variable replacement variable={{REPO_NAME}} value=go-broadcast
DEBUG Transformation completed size_before=1024 size_after=1048 diff=24
```

### --debug-config
Configuration loading and validation:
- Schema validation steps
- Default value application
- Environment variable resolution
- Validation errors with context

Example output:
```
DEBUG Starting config validation component=config
[TRACE] Validating version version=1.0
DEBUG Validating source configuration repo=owner/source branch=main path=.broadcast
DEBUG Validating targets target_count=3
```

### --debug-state
Repository state discovery process:
- Source repository analysis
- Target repository scanning
- Pull request detection
- File mapping resolution

Example output:
```
DEBUG Discovering source repository state component=state-discovery
DEBUG Source state discovered commit=abc123 branch=main file_count=5
DEBUG Target state discovered target=owner/target has_pr=true base_match=false
[TRACE] Pull request details pr_number=42 pr_state=open pr_mergeable=true
```

## Log Formats

### Text Format (Default)
Human-readable format with colors (when terminal supports):
```
15:04:05 INFO  Starting broadcast sync     version=1.2.3
15:04:05 DEBUG Config loaded successfully  path=.broadcast.yaml targets=3
15:04:06 WARN  Rate limit approaching      remaining=100
```

### JSON Format (--log-format json)
Machine-readable format for log aggregation:
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

## Performance Monitoring

### Automatic Timing
All operations are automatically timed with results in `duration_ms`:
```json
{
  "message": "Repository sync completed",
  "operation": "repository_sync",
  "duration_ms": 5234,
  "duration_human": "5.234s",
  "repo": "owner/target"
}
```

### Finding Bottlenecks
```bash
# Find operations taking more than 5 seconds
go-broadcast sync --log-format json 2>&1 | \
  jq 'select(.duration_ms > 5000) | {operation, duration_ms, repo}'

# Summary of operation times
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms) | "\(.operation)"' | \
  sort | uniq -c | sort -rn
```

## Security and Redaction

### Automatic Redaction
The following patterns are automatically redacted:
- GitHub tokens (ghp_*, ghs_*, github_pat_*, ghr_*)
- Bearer tokens and API keys
- Password/secret/key parameters in URLs
- Base64 encoded secrets
- Environment variables containing TOKEN/SECRET/KEY

### Redaction Examples
Input:
```
Executing command with token=ghp_1234567890abcdef1234567890abcdef1234
```

Output:
```
Executing command with token=ghp_***REDACTED***
```

### Audit Logging
Security-relevant operations are logged with audit markers:
```json
{
  "event": "config_change",
  "user": "system",
  "action": "update",
  "component": "audit",
  "time": 1705330245
}
```

## Diagnostic Command

The `diagnose` command collects system information for troubleshooting:

```bash
go-broadcast diagnose
```

Output includes:
- System information (OS, architecture, CPU count)
- go-broadcast version and build info
- Git and GitHub CLI versions
- Environment variables (redacted)
- Configuration file status
- Basic connectivity tests

Example output:
```json
{
  "timestamp": "2024-01-15T15:04:05Z",
  "version": {
    "version": "1.2.3",
    "commit": "abc123def",
    "date": "2024-01-15"
  },
  "system": {
    "os": "darwin",
    "arch": "arm64",
    "num_cpu": 8,
    "go_version": "go1.21.5"
  },
  "git_version": "git version 2.43.0",
  "gh_cli_version": "gh version 2.40.0 (2024-01-10)",
  "config": {
    "path": ".broadcast.yaml",
    "exists": true,
    "valid": true
  }
}
```

## Troubleshooting Guide

### No Output / Silent Failures
```bash
# Enable verbose output to see what's happening
go-broadcast sync -v

# Check if logs are going to stderr
go-broadcast sync 2>&1 | less
```

### Authentication Issues
```bash
# Debug git authentication
go-broadcast sync --debug-git -v

# Check token is set
echo $GITHUB_TOKEN | cut -c1-10  # Should show first 10 chars

# Test GitHub CLI directly
gh auth status
```

### Slow Operations
```bash
# Find slow operations
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms > 1000) | "\(.duration_ms)ms \(.operation) \(.repo // "")"'

# Enable all debugging to see where time is spent
go-broadcast sync -vv --debug-git --debug-api
```

### Memory Issues
```bash
# Monitor memory during execution (if supported by OS)
/usr/bin/time -l go-broadcast sync 2>&1 | grep "maximum resident set size"

# Use JSON logs to track memory if implemented
go-broadcast sync --log-format json 2>&1 | \
  jq 'select(.memory_mb) | {time: .["@timestamp"], memory_mb, operation}'
```

### Debugging Specific Repositories
```bash
# Set up detailed logging for problem repository
export GO_BROADCAST_LOG_LEVEL=trace
go-broadcast sync --debug-state --debug-git -vv 2> debug-repo.log

# Analyze the log
grep "target=owner/problem-repo" debug-repo.log
```

## Best Practices

### Development
1. Use `-vv` during development for detailed feedback
2. Enable relevant debug flags for the area you're working on
3. Use JSON format when debugging with scripts
4. Save debug logs for complex issues: `2> debug-$(date +%Y%m%d-%H%M%S).log`

### Production
1. Default INFO level for normal operation
2. Use JSON format for log aggregation
3. Enable debug selectively for specific issues
4. Monitor performance metrics regularly
5. Set up log rotation to prevent disk fill

### Log Rotation Example
```bash
# Using rotatelogs
go-broadcast sync --log-format json 2>&1 | \
  rotatelogs -l /var/log/go-broadcast/app.log 86400

# Using system logrotate
cat > /etc/logrotate.d/go-broadcast << EOF
/var/log/go-broadcast/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
EOF
```

### Integration with Monitoring Systems

#### Prometheus
```bash
# Extract metrics from JSON logs
go-broadcast sync --log-format json 2>&1 | \
  go-broadcast-prometheus-exporter

# Metrics available:
# - go_broadcast_operation_duration_seconds
# - go_broadcast_operation_total
# - go_broadcast_errors_total
```

#### ELK Stack
```yaml
# Logstash configuration
input {
  pipe {
    command => "go-broadcast sync --log-format json"
    codec => json
  }
}

filter {
  if [component] {
    mutate {
      add_tag => [ "go-broadcast" ]
    }
  }
}

output {
  elasticsearch {
    hosts => ["localhost:9200"]
    index => "go-broadcast-%{+YYYY.MM.dd}"
  }
}
```

## Appendix: Field Reference

### Standard Fields

| Field            | Description            | Example                           |
|------------------|------------------------|-----------------------------------|
| `@timestamp`     | ISO 8601 timestamp     | `2024-01-15T15:04:05.123Z`        |
| `level`          | Log level              | `info`, `debug`, `trace`          |
| `message`        | Log message            | `Starting sync operation`         |
| `component`      | System component       | `sync-engine`, `git`, `transform` |
| `operation`      | Current operation      | `repository_sync`, `git_clone`    |
| `duration_ms`    | Operation duration     | `1234`                            |
| `error`          | Error message          | `connection timeout`              |
| `repo`           | Repository name        | `owner/repo`                      |
| `correlation_id` | Request correlation ID | `a1b2c3d4-e5f6-7890`              |

### Component-Specific Fields

#### Git Operations
- `command`: Git command path
- `args`: Command arguments
- `exit_code`: Process exit code
- `dir`: Working directory

#### API Operations
- `request_size`: Request body size
- `response_size`: Response body size
- `status_code`: HTTP status code
- `rate_limit_remaining`: API calls remaining

#### Transform Operations
- `content_size`: Content size in bytes
- `variables`: Number of variables
- `type`: Transform type
- `size_before`/`size_after`: Size comparison

## Getting Help

### Debug Checklist
1. ✓ Enable appropriate verbose level (-v, -vv, -vvv)
2. ✓ Enable relevant debug flags
3. ✓ Check stderr for log output
4. ✓ Use JSON format for detailed analysis
5. ✓ Run diagnose command
6. ✓ Save logs for support requests

### Support Resources
- GitHub Issues: Include `go-broadcast diagnose` output
- Community Forum: Share relevant log excerpts (redacted)
- Enterprise Support: Provide full debug logs via secure channel