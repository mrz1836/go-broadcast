# go-broadcast Troubleshooting Runbook

## 🎯 Purpose

This runbook provides systematic troubleshooting procedures for go-broadcast issues. It's designed for operators, support teams, and developers to quickly identify and resolve common problems.

## 📋 General Troubleshooting Process

### 1. Information Gathering
```bash
# Collect system diagnostics
go-broadcast diagnose > diagnostics-$(date +%Y%m%d-%H%M%S).json

# Check basic connectivity
gh auth status
git --version
echo $GITHUB_TOKEN | cut -c1-10
```

### 2. Initial Assessment
- What operation was being performed?
- What error message was displayed?
- When did the issue start occurring?
- Has this worked before?

### 3. Log Collection
```bash
# Enable comprehensive logging
go-broadcast sync -vvv --debug-git --debug-api --debug-transform --debug-config --debug-state 2> full-debug.log

# Save with timestamp
DEBUG_LOG="debug-$(date +%Y%m%d-%H%M%S).log"
go-broadcast sync --json -vv --debug-git --debug-api 2> "$DEBUG_LOG"
```

## 🚨 Common Issues

### Authentication Failures

**Symptoms:**
- "Authentication failed" errors
- "Permission denied" messages
- "Invalid credentials" responses

**Diagnosis:**
```bash
# Check GitHub CLI authentication
gh auth status

# Test API access
gh api user

# Verify token format
echo $GITHUB_TOKEN | cut -c1-10  # Should show "ghp_" or "ghs_"

# Debug git authentication
go-broadcast sync --debug-git -v
```

**Resolution:**
1. **Re-authenticate GitHub CLI:**
   ```bash
   gh auth logout
   gh auth login
   ```

2. **Set token manually:**
   ```bash
   export GITHUB_TOKEN="your_token_here"
   ```

3. **Check token permissions:**
   - Repo access
   - Read/write permissions
   - Organization access

**Escalation:** If token has correct permissions but authentication fails, check enterprise SSO or IP restrictions.

---

### Configuration Validation Errors

**Symptoms:**
- "Invalid configuration" errors
- Schema validation failures
- Missing required fields

**Diagnosis:**
```bash
# Validate configuration with detailed output
go-broadcast validate --debug-config -v

# Check configuration syntax
cat sync.yaml | yq '.'  # or yaml validator
```

**Common Issues:**
1. **YAML syntax errors:**
   ```bash
   # Check YAML validity
   python -c "import yaml; yaml.safe_load(open('sync.yaml'))"
   ```

2. **Repository format issues:**
   ```yaml
   # Correct format
   repo: "owner/repository"
   
   # Incorrect formats
   repo: "github.com/owner/repository"  # ❌
   repo: "https://github.com/owner/repo" # ❌
   ```

3. **File path issues:**
   ```bash
   # Check for path traversal attempts
   grep -E "\.\./|\.\.\\\\|~/" sync.yaml
   ```

**Resolution:**
- Fix YAML syntax errors
- Ensure repository names follow "owner/repo" format
- Verify file mappings use relative paths
- Check branch names don't contain special characters

---

### State Discovery Issues

**Symptoms:**
- "Repository not found" errors
- Empty sync results
- Incorrect sync status

**Diagnosis:**
```bash
# Debug state discovery
go-broadcast sync --debug-state -vv

# Check repository access
gh api "repos/owner/repo"

# List branches
gh api "repos/owner/repo/branches"
```

**Common Causes:**
1. **Repository access permissions**
2. **Branch name mismatches**
3. **Sync branch pattern conflicts**

**Resolution:**
1. **Verify repository access:**
   ```bash
   gh repo view owner/repo
   ```

2. **Check branch configuration:**
   ```bash
   # List all branches
   gh api repos/owner/repo/branches --jq '.[].name'
   
   # Check specific branch
   gh api repos/owner/repo/branches/main
   ```

3. **Clean up sync branches if needed:**
   ```bash
   # List sync branches
   gh api repos/owner/repo/branches --jq '.[] | select(.name | startswith("sync/")) | .name'
   ```

---

### Performance Issues

**Symptoms:**
- Slow operation execution
- Timeouts
- High memory usage

**Diagnosis:**
```bash
# Performance monitoring
go-broadcast sync --json 2>&1 | \
  jq -r 'select(.duration_ms > 5000) | "\(.duration_ms)ms \(.operation) \(.repo // "")"'

# Memory monitoring (macOS)
/usr/bin/time -l go-broadcast sync 2>&1 | grep "maximum resident set size"

# API rate limit monitoring
go-broadcast sync --debug-api --json 2>&1 | \
  jq 'select(.rate_limit_remaining) | {remaining: .rate_limit_remaining, reset: .rate_limit_reset}'
```

**Common Bottlenecks:**
1. **Git operations (cloning large repos)**
2. **API rate limiting**
3. **File transformation (large files)**

**Resolution:**
1. **Git optimization:**
   ```bash
   # Check repository size
   gh api repos/owner/repo --jq '.size'
   
   # Use shallow clones for large repos
   # (configure in source repository settings)
   ```

2. **API rate limit management:**
   ```bash
   # Check current rate limit
   gh api rate_limit
   
   # Implement delays between operations
   # (built into go-broadcast automatically)
   ```

3. **File size optimization:**
   ```bash
   # Find large files in transformations
   go-broadcast sync --debug-transform -vv 2>&1 | \
     grep -E "content_size=[0-9]+" | sort -t= -k2 -n
   ```

---

### File Transformation Issues

**Symptoms:**
- Variables not replaced
- Encoding problems
- Binary file corruption

**Diagnosis:**
```bash
# Debug transformations
go-broadcast sync --debug-transform -vv

# Check file encodings
file path/to/template/file

# Test variable replacement manually
echo "{{REPO_NAME}}" | sed 's/{{REPO_NAME}}/test-repo/'
```

**Common Issues:**
1. **Variable format mismatches:**
   ```yaml
   # Supported formats
   variables:
     REPO_NAME: "my-repo"      # {{REPO_NAME}} or ${REPO_NAME}
   
   repo_name: true              # Automatic Go module path replacement
   ```

2. **Binary file handling:**
   ```bash
   # Check if file is binary
   go-broadcast sync --debug-transform -vvv 2>&1 | grep "Binary file detected"
   ```

**Resolution:**
- Verify variable names match configuration
- Check file encoding (UTF-8 recommended)
- Exclude binary files from transformation
- Test transformations on small files first

---

### Network and Connectivity Issues

**Symptoms:**
- Connection timeouts
- DNS resolution failures
- Proxy-related errors

**Diagnosis:**
```bash
# Test GitHub connectivity
curl -I https://api.github.com

# Test GitHub CLI connectivity
gh auth status

# Check proxy settings
env | grep -i proxy
```

**Resolution:**
1. **Network connectivity:**
   ```bash
   # Test DNS resolution
   nslookup api.github.com
   
   # Test HTTPS connectivity
   curl -v https://api.github.com/user
   ```

2. **Proxy configuration:**
   ```bash
   # Set proxy for GitHub CLI
   gh config set http_proxy http://proxy.company.com:8080
   
   # Set proxy for git operations
   git config --global http.proxy http://proxy.company.com:8080
   ```

## 🔍 Log Analysis Workflows

### Error Pattern Analysis
```bash
# Extract all errors with context
go-broadcast sync --json 2>&1 | \
  jq 'select(.level=="error") | {time: ."@timestamp", component, operation, message, error}' | \
  jq -s 'sort_by(.time)'

# Find error patterns
go-broadcast sync --json 2>&1 | \
  jq -r 'select(.level=="error") | .error' | \
  sort | uniq -c | sort -rn
```

### Performance Analysis
```bash
# Operation timing analysis
go-broadcast sync --json 2>&1 | \
  jq 'select(.duration_ms) | {operation, duration_ms, repo}' | \
  jq -s 'sort_by(.duration_ms) | reverse'

# Component performance breakdown
go-broadcast sync --json 2>&1 | \
  jq 'select(.duration_ms) | {component, operation, duration_ms}' | \
  jq -s 'group_by(.component) | map({component: .[0].component, total_ms: map(.duration_ms) | add, operations: length})'
```

### Security Event Monitoring
```bash
# Authentication events
go-broadcast sync --json 2>&1 | \
  jq 'select(.component=="audit" and .event=="authentication") | {time: ."@timestamp", success, method}'

# Repository access tracking
go-broadcast sync --json 2>&1 | \
  jq 'select(.event=="repo_access") | {time: ."@timestamp", repo, action, user}'
```

## 📊 Health Check Procedures

### Basic Health Check
```bash
#!/bin/bash
# go-broadcast-health-check.sh

echo "=== go-broadcast Health Check ==="

# Check binary exists and is executable
if ! command -v go-broadcast &> /dev/null; then
    echo "❌ go-broadcast not found in PATH"
    exit 1
fi

# Check version
echo "✅ Version: $(go-broadcast version 2>/dev/null || echo 'Unknown')"

# Check dependencies
echo "✅ Git: $(git --version)"
echo "✅ GitHub CLI: $(gh --version | head -1)"

# Check authentication
if gh auth status &> /dev/null; then
    echo "✅ GitHub authentication: OK"
else
    echo "❌ GitHub authentication: Failed"
fi

# Test basic functionality
if go-broadcast diagnose > /dev/null 2>&1; then
    echo "✅ Basic functionality: OK"
else
    echo "❌ Basic functionality: Failed"
fi

echo "=== Health Check Complete ==="
```

### Performance Baseline
```bash
#!/bin/bash
# performance-baseline.sh

echo "=== Performance Baseline Test ==="

# Small repository test
time go-broadcast sync --config test-config.yaml --dry-run

# Memory usage test
/usr/bin/time -l go-broadcast sync --config test-config.yaml --dry-run 2>&1 | \
  grep "maximum resident set size"

# API rate limit check
go-broadcast sync --debug-api --json --config test-config.yaml --dry-run 2>&1 | \
  jq 'select(.rate_limit_remaining) | .rate_limit_remaining' | tail -1
```

## 🆘 Escalation Procedures

### Level 1: Self-Service
- Review this runbook
- Check logs with basic verbose flags (`-v`)
- Verify configuration and authentication
- Test with `--dry-run` flag

### Level 2: Advanced Debugging
- Collect comprehensive logs (`-vvv` with all debug flags)
- Run `go-broadcast diagnose`
- Performance analysis with JSON output
- Search existing GitHub issues

### Level 3: Support Engagement
Collect the following information:

```bash
# Support information package
SUPPORT_DIR="go-broadcast-support-$(date +%Y%m%d-%H%M%S)"
mkdir "$SUPPORT_DIR"

# System diagnostics
go-broadcast diagnose > "$SUPPORT_DIR/diagnostics.json"

# Configuration (with sensitive data redacted)
cp sync.yaml "$SUPPORT_DIR/config.yaml"
sed -i.bak 's/ghp_[a-zA-Z0-9_]*/ghp_***REDACTED***/g' "$SUPPORT_DIR/config.yaml"

# Full debug logs
go-broadcast sync --json -vvv --debug-git --debug-api --debug-transform --debug-config --debug-state 2> "$SUPPORT_DIR/debug.log"

# Error extraction
jq 'select(.level=="error")' < "$SUPPORT_DIR/debug.log" > "$SUPPORT_DIR/errors.json"

# Create support archive
tar -czf "${SUPPORT_DIR}.tar.gz" "$SUPPORT_DIR"
echo "Support package created: ${SUPPORT_DIR}.tar.gz"
```

## Developer Workflow Integration

For comprehensive go-broadcast development workflows, see [CLAUDE.md](../.github/CLAUDE.md) which includes:
- **Development environment troubleshooting** for build failures and test issues
- **go-broadcast debugging workflows** with component-specific procedures
- **Environment validation procedures** for GitHub authentication and Go setup
- **Performance troubleshooting integration** with benchmarking and profiling

## Reference Links

- **General Troubleshooting**: [troubleshooting.md](troubleshooting.md) - Common issues and solutions
- **Logging Quick Reference**: [logging-quick-ref.md](logging-quick-ref.md) - Essential logging commands
- **Detailed Logging Guide**: [logging.md](logging.md) - Comprehensive logging documentation
- **Developer Workflows**: [CLAUDE.md](../.github/CLAUDE.md) - Complete development workflow integration
- **Main Documentation**: [README.md](../README.md) - Project overview and quick start
- **GitHub Issues**: [Report a bug](https://github.com/mrz1836/go-broadcast/issues/new)
- **Security Issues**: [Security Policy](../.github/SECURITY.md)

## 🔄 Runbook Maintenance

This runbook should be updated when:
- New common issues are identified
- Troubleshooting procedures change
- New diagnostic tools are added
- Support escalation procedures change

**Last Updated**: 2025-01-15  
**Version**: 1.0  
**Maintainer**: go-broadcast team
