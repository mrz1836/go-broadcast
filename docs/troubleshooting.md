# Troubleshooting Guide

This guide helps resolve common issues when using go-broadcast.

## Table of Contents

- [Authentication Issues](#authentication-issues)
- [Configuration Problems](#configuration-problems)
- [GitHub API Issues](#github-api-issues)
- [Git Operations](#git-operations)
- [File Synchronization](#file-synchronization)
- [Directory Synchronization](#directory-synchronization)
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

**Problem**: Repository doesn't exist, or you don't have access.

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

## Directory Synchronization

### "Directory sync completed but no files were synced"

**Problem**: Directory mapping completes successfully but results in 0 files synchronized.

**Common Causes & Solutions**:

1. **All files excluded by patterns**:
   ```bash
   # Use debug logging to see exclusion details
   go-broadcast sync --log-level debug --config sync.yaml 2>&1 | grep -i "excluded"
   ```

   Check smart defaults (automatically applied):
   ```yaml
   # These are ALWAYS excluded for safety:
   # *.out, *.test, *.exe, **/.DS_Store, **/tmp/*, **/.git

   # Review your additional exclusions
   directories:
     - src: "src"
       dest: "src"
       exclude:
         - "*.go"  # This would exclude all Go files!
   ```

2. **Source directory doesn't exist**:
   ```bash
   # Verify source directory exists in template repo
   gh api repos/company/template-repo/contents/source-directory
   ```

3. **Empty source directory**:
   ```bash
   # Check if source directory is empty
   gh api repos/company/template-repo/git/trees/main --recursive | \
   jq '.tree[] | select(.path | startswith("source-directory/"))'
   ```

**Fix**: Review exclusion patterns and verify source directory content.

### "Directory sync is extremely slow"

**Problem**: Directory synchronization takes much longer than expected performance targets.

**Expected Performance**:
- <50 files: <3ms
- 50-150 files: 1-7ms
- 500+ files: 16-32ms
- 1000+ files: ~32ms

**Diagnosis**:
```bash
# Check actual processing time
go-broadcast sync --log-level debug --config sync.yaml 2>&1 | \
grep -E "(processing_time_ms|duration)"
```

**Performance Issues & Solutions**:

1. **Large individual files**:
   ```yaml
   # Exclude large files that don't need sync
   directories:
     - src: "assets"
       dest: "assets"
       exclude:
         - "**/*.zip"        # Large archives
         - "**/*.tar.gz"     # Compressed files
         - "**/*.iso"        # Disk images
         - "**/*.dmg"        # macOS images
   ```

2. **Too many excluded files** (paradoxically slow):
   ```yaml
   # Instead of excluding many files, sync specific subdirectories
   directories:
     # Instead of this (slow):
     # - src: "large-dir"
     #   exclude: ["sub1/**", "sub2/**", "sub3/**", ...]

     # Do this (fast):
     - src: "large-dir/wanted-subdir"
       dest: "large-dir/wanted-subdir"
   ```

3. **Deep directory nesting**:
   ```yaml
   # Avoid very deep recursion in exclusion patterns
   exclude:
     - "**/deeply/nested/path/**"  # Can be slow
   # Prefer specific patterns:
     - "deeply/nested/path/**"     # Faster
   ```

### "Exclusion patterns not working as expected"

**Problem**: Files that should be excluded are still being synchronized.

**Common Pattern Issues**:

1. **Case sensitivity**:
   ```yaml
   # Patterns are case-sensitive
   exclude:
     - "*.TMP"    # Won't match *.tmp
     - "*.tmp"    # Won't match *.TMP
     - "*.[Tt][Mm][Pp]"  # Matches both
   ```

2. **Directory vs file patterns**:
   ```yaml
   exclude:
     - "temp"      # Matches files named 'temp'
     - "temp/"     # Matches directories named 'temp'
     - "temp/**"   # Matches everything under temp directories
     - "**/temp/**" # Matches temp directories at any depth
   ```

3. **Relative path confusion**:
   ```yaml
   # Patterns match against paths relative to src directory
   directories:
     - src: ".github"
       dest: ".github"
       exclude:
         - "workflows/local-*"      # Matches .github/workflows/local-*
         - "**/workflows/local-*"   # Also matches nested workflows
   ```

**Debugging Exclusions**:
```bash
# See detailed pattern matching
go-broadcast sync --log-level debug --config sync.yaml 2>&1 | \
grep -E "(excluded|pattern|matching)"

# Test specific patterns with dry-run
go-broadcast sync --dry-run --config sync.yaml
```

### "Transform errors on directory files"

**Problem**: Some files in directory fail transformation but sync continues.

**Expected Behavior**: Transform errors on individual files don't fail entire directory sync (by design for robustness).

**Common Issues**:

1. **Binary files processed as text**:
   ```bash
   # Check logs for binary detection
   go-broadcast sync --log-level debug --config sync.yaml 2>&1 | \
   grep -i "binary"
   ```

   **Solution**: Binary files are automatically detected and skipped for transforms. This is correct behavior.

2. **Invalid variable substitution**:
   ```yaml
   directories:
     - src: "configs"
       dest: "configs"
       transform:
         variables:
           SERVICE_NAME: "my-service"
           # Missing variable will cause transform errors
           # Check logs for "undefined variable" messages
   ```

3. **Malformed Go module paths** (for repo_name transform):
   ```yaml
   # repo_name transform only works on text files with valid Go syntax
   transform:
     repo_name: true  # May fail on config files, docs, etc.
   ```

**Solutions**:
```yaml
# Exclude problematic files from transforms
directories:
  - src: "mixed-content"
    dest: "mixed-content"
    exclude:
      - "**/*.bin"      # Skip binary files
      - "**/*.jpg"      # Skip images
      - "**/*.pdf"      # Skip PDFs
    transform:
      repo_name: true   # Only applies to remaining text files
```

### "API rate limiting during large directory sync"

**Problem**: GitHub API rate limits exceeded during directory operations.

**Understanding the Issue**:
- go-broadcast uses GitHub tree API for 90%+ call reduction
- Rate limits should rarely be hit with proper optimization
- If you're hitting limits, there may be a configuration issue

**Diagnosis**:
```bash
# Check API usage in logs
go-broadcast sync --log-level debug --config sync.yaml 2>&1 | \
grep -E "(api|rate|tree|calls)"

# Check current rate limit status
gh api rate_limit
```

**Solutions**:

1. **Verify tree API is being used**:
   ```bash
   # Should see "using tree API" in debug logs
   # Should see very few individual file API calls
   ```

2. **Split very large directories**:
   ```yaml
   # Instead of syncing huge directory
   directories:
     # Split into logical chunks
     - src: "large-dir/core"
       dest: "large-dir/core"
     - src: "large-dir/modules"
       dest: "large-dir/modules"
   ```

3. **Optimize concurrent repositories**:
   ```bash
   # Sync specific repositories to reduce load
   go-broadcast sync company/repo1 company/repo2 --config sync.yaml
   ```

### "Memory usage too high during directory sync"

**Problem**: go-broadcast consuming excessive memory during large directory operations.

**Expected Memory Usage**:
- ~1.2MB per 1000 files processed
- Linear scaling with file count
- Memory released immediately after processing

**Diagnosis**:
```bash
# Monitor memory during sync
go-broadcast sync --config sync.yaml &
PID=$!
while kill -0 $PID 2>/dev/null; do
  ps -o pid,vsz,rss,comm -p $PID
  sleep 2
done
```

**Solutions**:

1. **Segment large directories**:
   ```yaml
   # Process in smaller chunks
   directories:
     - src: "docs/section1"
       dest: "docs/section1"
     - src: "docs/section2"
       dest: "docs/section2"
   ```

2. **Exclude large files**:
   ```yaml
   exclude:
     - "**/*.zip"     # Large archives
     - "**/*.tar.*"   # Compressed files
     - "**/*.iso"     # Disk images
     - "data/**"      # Large data files
   ```

3. **Reduce concurrent operations**:
   ```bash
   # Process fewer repositories simultaneously
   go-broadcast sync specific/repo --config sync.yaml
   ```

### "Directory structure not preserved correctly"

**Problem**: Nested directory structure is flattened or modified unexpectedly.

**Understanding preserve_structure**:
```yaml
directories:
  - src: "docs/api/v1"
    dest: "api-docs"
    preserve_structure: true   # Results in: api-docs/nested/file.md
    preserve_structure: false  # Results in: api-docs/file.md (flattened)
```

**Common Issues**:

1. **Unexpected flattening**:
   ```yaml
   # Default is preserve_structure: true
   # If you're seeing flattening, check for:
   directories:
     - src: "nested/content"
       dest: "content"
       preserve_structure: false  # This causes flattening
   ```

2. **Path conflicts** (when flattening):
   ```yaml
   # Flattening can cause conflicts with same filenames
   # src/dir1/config.yml and src/dir2/config.yml both → dest/config.yml
   # Solution: Use preserve_structure: true or rename during sync
   ```

### "Hidden files not syncing"

**Problem**: Files starting with `.` are not being synchronized.

**Understanding include_hidden**:
```yaml
directories:
  - src: "configs"
    dest: "configs"
    include_hidden: true   # Syncs .env, .gitignore, etc. (default)
    include_hidden: false  # Skips hidden files
```

**Solutions**:
```yaml
# Ensure include_hidden is not set to false
directories:
  - src: "project-root"
    dest: "project-root"
    include_hidden: true   # Explicit enable
    exclude:
      - "**/.DS_Store"     # Still excluded by smart defaults
      - "**/.git/**"       # Still excluded by smart defaults
```

**Note**: Smart defaults always exclude certain hidden files (`.DS_Store`, `.git`) for safety, even with `include_hidden: true`.

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
