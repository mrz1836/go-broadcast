# Troubleshooting Guide

This guide helps resolve common issues when using go-broadcast.

## Table of Contents

- [Authentication Issues](#authentication-issues)
- [Configuration Problems](#configuration-problems)
- [GitHub API Issues](#github-api-issues)
- [Git Operations](#git-operations)
- [File Synchronization](#file-synchronization)
- [Performance Issues](#performance-issues)
- [Debugging](#debugging)

## Authentication Issues

### "gh: To get started with GitHub CLI, please run: gh auth login"

**Problem**: GitHub CLI is not authenticated.

**Solution**:
```bash
# Authenticate with GitHub CLI
gh auth login

# Verify authentication
gh auth status
```

**Alternative**: Set a GitHub token directly:
```bash
# Use GH_PAT_TOKEN as the preferred token (checked first)
export GH_PAT_TOKEN="your_personal_access_token"
# Or use GITHUB_TOKEN as a fallback
export GITHUB_TOKEN="your_personal_access_token"
```

### "gh: Not Found (HTTP 404)"

**Problem**: Repository doesn't exist or you don't have access.

**Solutions**:
1. **Check repository name format**: Must be `owner/repository`, not just `repository`
2. **Verify access**: Ensure you have read access to source repo and write access to target repos
3. **Check repository existence**: Use `gh repo view owner/repo` to verify the repository exists
4. **Private repositories**: Ensure your GitHub token has appropriate scopes (`repo` for private repos)

### "Permission denied (publickey)"

**Problem**: Git authentication failure when pushing changes.

**Solutions**:
1. **Check SSH keys**: Ensure your SSH key is added to GitHub
   ```bash
   ssh -T git@github.com
   ```
2. **Use HTTPS with token**: Configure git to use HTTPS with your personal access token
   ```bash
   git config --global credential.helper store
   ```

## Configuration Problems

### "Configuration file not found"

**Problem**: go-broadcast can't find the sync.yaml file.

**Solutions**:
1. **Specify config path explicitly**:
   ```bash
   go-broadcast sync --config /path/to/sync.yaml
   ```
2. **Check current directory**: Default looks for `sync.yaml` in current directory
3. **Create example config**:
   ```bash
   go-broadcast validate --help  # Shows example configuration
   ```

### "unsupported config version"

**Problem**: Configuration file uses wrong version format.

**Solution**: Update your configuration to use version 1:
```yaml
version: 1  # Must be version 1
source:
  repo: "org/template-repo"
  branch: "master"
targets:
  - repo: "org/target-repo"
    files:
      - src: "file.txt"
        dest: "file.txt"
```

### "Configuration validation failed"

**Common validation errors and fixes**:

1. **Empty source repository**:
   ```yaml
   source:
     repo: "org/template-repo"  # Must not be empty
     branch: "master"           # Must not be empty
   ```

2. **No targets specified**:
   ```yaml
   targets:  # Must have at least one target
     - repo: "org/target-repo"
       files:
         - src: "file.txt"
           dest: "file.txt"
   ```

3. **Invalid repository format**:
   ```yaml
   source:
     repo: "org/repo"  # Correct: owner/repository
     # repo: "repo"    # Wrong: missing owner
   ```

## GitHub API Issues

### "API rate limit exceeded"

**Problem**: Too many GitHub API requests.

**Solutions**:
1. **Wait for rate limit reset**: GitHub resets limits hourly
2. **Use authenticated requests**: Higher rate limits with authentication
3. **Reduce concurrent operations**:
   ```bash
   go-broadcast sync --config sync.yaml  # Uses default concurrency
   ```
4. **Check rate limit status**:
   ```bash
   gh api rate_limit
   ```

### "Commit not found"

**Problem**: Specified commit SHA doesn't exist in repository.

**Solutions**:
1. **Check commit exists**: `gh api repos/owner/repo/commits/SHA`
2. **Use branch name instead**: Ensure branch exists and is accessible
3. **Check repository sync**: Source repository might be ahead of your local state

### "Branch already exists"

**Problem**: Sync branch name conflicts with existing branch.

**Solutions**:
1. **Delete existing branch** (if safe):
   ```bash
   git push origin --delete chore/sync-files-YYYYMMDD-HHMMSS-commit
   ```
2. **Wait for branch name to change**: Branch names include timestamps
3. **Check for stuck sync processes**: Multiple sync processes might conflict

## Git Operations

### "fatal: could not read Username for 'https://github.com'"

**Problem**: Git credentials not configured for HTTPS.

**Solutions**:
1. **Configure git credentials**:
   ```bash
   git config --global credential.helper store
   echo "https://username:token@github.com" > ~/.git-credentials
   ```
2. **Use SSH instead**: Configure repositories to use SSH URLs
3. **Set GitHub token**:
   ```bash
   # Use GH_PAT_TOKEN as the preferred token (checked first)
   export GH_PAT_TOKEN="your_token"
   # Or use GITHUB_TOKEN as a fallback
   export GITHUB_TOKEN="your_token"
   ```

### "fatal: destination path already exists"

**Problem**: Target directory already exists during clone.

**Solutions**:
1. **Clean up temporary directories**: go-broadcast should clean up automatically
2. **Check disk space**: Insufficient space might prevent cleanup
3. **Kill stuck processes**: Check for other go-broadcast instances
   ```bash
   ps aux | grep go-broadcast
   ```

### "Your branch is ahead of 'origin/master'"

**Problem**: Local repository state is inconsistent.

**Solutions**:
1. **This is normal**: go-broadcast creates commits as part of sync process
2. **Check sync completed**: Ensure pull request was created successfully
3. **Manual cleanup** (if needed):
   ```bash
   git reset --hard origin/master
   ```

## File Synchronization

### "No changes detected"

**Problem**: Files appear unchanged but should be different.

**Solutions**:
1. **Check file transformations**: Ensure transform rules are configured correctly
2. **Verify source files**: Check files exist in source repository
3. **Review file mappings**: Ensure src/dest paths are correct
   ```yaml
   files:
     - src: ".github/workflows/ci.yml"    # Must exist in source
       dest: ".github/workflows/ci.yml"   # Destination path
   ```
4. **Use debug logging**:
   ```bash
   go-broadcast sync --log-level debug
   ```

### "Binary file detected, skipping transformation"

**Problem**: Binary files not being transformed (this is correct behavior).

**Solutions**:
1. **This is expected**: Binary files are copied without transformation
2. **Check file type**: Use `file` command to verify file type
3. **For text files misidentified as binary**: Check for null bytes or unusual characters

### "Transform failed"

**Problem**: File transformation encountered an error.

**Solutions**:
1. **Check regex patterns**: Ensure transformation patterns are valid
2. **Review template variables**: Ensure all required variables are defined
   ```yaml
   transform:
     variables:
       SERVICE_NAME: "my-service"  # Must be defined if used in templates
   ```
3. **Test transformations locally**: Create small test files to debug transforms

## Performance Issues

### "Sync is very slow"

**Problem**: Synchronization takes longer than expected.

**Solutions**:
1. **Reduce concurrency** (if system is overloaded):
   ```bash
   # System handles concurrency automatically, but you can limit targets
   go-broadcast sync org/specific-repo  # Sync only specific repositories
   ```
2. **Check network connectivity**: Slow GitHub API responses
3. **Reduce file count**: Large numbers of files take longer to process
4. **Use dry-run first**: Test configuration before full sync
   ```bash
   go-broadcast sync --dry-run
   ```

### "Out of memory errors"

**Problem**: go-broadcast consuming too much memory.

**Solutions**:
1. **Reduce concurrent operations**: Sync fewer repositories at once
2. **Check file sizes**: Large files consume more memory during transformation
3. **Split large configurations**: Sync subsets of repositories separately
4. **Monitor system resources**: Ensure adequate RAM available

## Debugging

### Enable Debug Logging

```bash
go-broadcast sync --log-level debug
```

This shows detailed information about:
- Configuration loading and validation
- GitHub API requests and responses
- Git operations and their results
- File transformations and their effects
- State discovery process

### Check Sync State

```bash
go-broadcast status --config sync.yaml
```

This shows:
- Current state of all target repositories
- Last sync times and commit SHAs
- Open pull requests
- Sync status for each repository

### Validate Configuration

```bash
go-broadcast validate --config sync.yaml
```

This checks:
- Configuration file syntax
- Required fields
- Repository name formats
- File path validity
- Transform configuration

### Dry Run Mode

```bash
go-broadcast sync --dry-run --config sync.yaml
```

This shows what would happen without making changes:
- Which files would be synchronized
- What transformations would be applied
- Which pull requests would be created

### Manual State Check

Check GitHub state manually:

```bash
# List branches in target repository
gh api repos/owner/target-repo/branches

# Check for existing sync PRs
gh pr list --repo owner/target-repo --label automated-sync

# View specific branch
gh api repos/owner/target-repo/branches/chore/sync-files-YYYYMMDD-HHMMSS-commit
```

## Getting Help

If you're still experiencing issues:

1. **Check the logs**: Always run with `--log-level debug` first
2. **Create minimal reproduction**: Simplify your configuration to isolate the problem
3. **Check GitHub status**: Visit [GitHub Status](https://githubstatus.com) for API issues
4. **Review configuration**: Double-check repository names, file paths, and permissions
5. **Test components individually**:
   - Test GitHub authentication: `gh auth status`
   - Test repository access: `gh repo view owner/repo`
   - Test configuration: `go-broadcast validate`

## Common Patterns

### Testing New Configuration

```bash
# 1. Validate configuration
go-broadcast validate --config new-sync.yaml

# 2. Test with dry-run
go-broadcast sync --dry-run --config new-sync.yaml

# 3. Test with single repository
go-broadcast sync org/test-repo --config new-sync.yaml

# 4. Full sync
go-broadcast sync --config new-sync.yaml
```

### Recovering from Failed Sync

```bash
# 1. Check current status
go-broadcast status --config sync.yaml

# 2. Check for stuck branches/PRs in GitHub web interface

# 3. Clean up if needed (be careful!)
gh pr close 123 --repo org/target-repo  # Close stuck PR
git push origin --delete branch-name     # Delete stuck branch

# 4. Retry sync
go-broadcast sync --config sync.yaml
```

### Monitoring Ongoing Sync

```bash
# Check status periodically
watch -n 30 "go-broadcast status --config sync.yaml"

# Monitor logs in another terminal
go-broadcast sync --log-level info --config sync.yaml 2>&1 | tee sync.log

## Developer Workflow Integration

For comprehensive go-broadcast development workflows, see [CLAUDE.md](../.github/CLAUDE.md#️-troubleshooting-quick-reference) which includes:
- **Troubleshooting development issues** with build failures, test failures, and linting errors
- **go-broadcast debugging procedures** with verbose logging and component-specific debugging
- **Environment troubleshooting steps** for GitHub authentication and Go environment issues
- **Development workflow integration** for effective problem-solving

## Related Documentation

1. [Troubleshooting Runbook](troubleshooting-runbook.md) - Systematic operational troubleshooting procedures
2. [Logging Guide](logging.md) - Comprehensive logging and debugging documentation
3. [Logging Quick Reference](logging-quick-ref.md) - Essential logging commands and flags
4. [CLAUDE.md Developer Workflows](../.github/CLAUDE.md) - Complete development workflow integration
```
