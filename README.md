<div align="center">

# 📡 go-broadcast

**Stateless File Sync Orchestrator for Repository Management**

<br/>

<a href="https://github.com/mrz1836/go-broadcast/releases"><img src="https://img.shields.io/github/release-pre/mrz1836/go-broadcast?include_prereleases&style=flat-square&logo=github&color=black" alt="Release"></a>
<a href="https://golang.org/"><img src="https://img.shields.io/github/go-mod/go-version/mrz1836/go-broadcast?style=flat-square&logo=go&color=00ADD8" alt="Go Version"></a>
<a href="https://github.com/mrz1836/go-broadcast/blob/master/LICENSE"><img src="https://img.shields.io/github/license/mrz1836/go-broadcast?style=flat-square&color=blue" alt="License"></a>

<br/>

<table align="center" border="0">
  <tr>
    <td align="right">
       <code>CI / CD</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://github.com/mrz1836/go-broadcast/actions"><img src="https://img.shields.io/github/actions/workflow/status/mrz1836/go-broadcast/fortress.yml?branch=master&label=build&logo=github&style=flat-square" alt="Build"></a>
       <a href="https://github.com/mrz1836/go-broadcast/actions"><img src="https://img.shields.io/github/last-commit/mrz1836/go-broadcast?style=flat-square&logo=git&logoColor=white&label=last%20update" alt="Last Commit"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Quality</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://goreportcard.com/report/github.com/mrz1836/go-broadcast"><img src="https://goreportcard.com/badge/github.com/mrz1836/go-broadcast?style=flat-square" alt="Go Report"></a>
       <a href="https://mrz1836.github.io/go-broadcast/"><img src="https://img.shields.io/badge/coverage-78.5%25-yellow?style=flat-square&logo=codecov&logoColor=white" alt="Coverage"></a>
    </td>
  </tr>

  <tr>
    <td align="right">
       <code>Security</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://scorecard.dev/viewer/?uri=github.com/mrz1836/go-broadcast"><img src="https://api.scorecard.dev/projects/github.com/mrz1836/go-broadcast/badge?style=flat-square" alt="Scorecard"></a>
       <a href=".github/SECURITY.md"><img src="https://img.shields.io/badge/policy-active-success?style=flat-square&logo=security&logoColor=white" alt="Security"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Community</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://github.com/mrz1836/go-broadcast/graphs/contributors"><img src="https://img.shields.io/github/contributors/mrz1836/go-broadcast?style=flat-square&color=orange" alt="Contributors"></a>
       <a href="https://mrz1818.com/"><img src="https://img.shields.io/badge/donate-bitcoin-ff9900?style=flat-square&logo=bitcoin" alt="Bitcoin"></a>
    </td>
  </tr>
</table>

</div>

<br/>

## 🗂️ Project Navigation

<div align="center">

⚡ [**Quick Start**](#-quick-start) &nbsp;┃&nbsp;
✨ [**Key Features**](#-key-features) &nbsp;┃&nbsp;
🔍 [**How It Works**](#-how-it-works) &nbsp;┃&nbsp;
💡 [**Usage Examples**](#-usage-examples)

<br/>

📚 [**Documentation**](#-documentation) &nbsp;┃&nbsp;
🧪 [**Examples & Tests**](#-examples--tests) &nbsp;┃&nbsp;
🛠️ [**Code Standards**](#-code-standards) &nbsp;┃&nbsp;
🤖 [**AI Compliance**](#-ai-compliance)

<br/>

⚡ [**Performance**](#-performance) &nbsp;┃&nbsp;
👥 [**Maintainers**](#-maintainers) &nbsp;┃&nbsp;
🤝 [**Contributing**](#-contributing) &nbsp;┃&nbsp;
📝 [**License**](#-license)

</div>

<br>

## ⚡ Quick Start

Get up and running with go-broadcast in under 5 minutes!

### Prerequisites
- [Go 1.24+](https://golang.org/doc/install) ([supported release](https://golang.org/doc/devel/release.html#policy)) and [GitHub CLI](https://cli.github.com/) installed
- GitHub authentication: `gh auth login`
- [MAGE-X](https://github.com/mrz1836/mage-x) (optional, for building from source)

### Installation

```bash
# Install the go-broadcast CLI tool via master branch
go install github.com/mrz1836/go-broadcast/cmd/go-broadcast@latest

# Upgrade to the latest stable version
go-broadcast upgrade --force
```

### Create Configuration

Create a `sync.yaml` file:

```yaml
version: 1
groups:
  - name: "Default Sync"
    id: "default"
    description: "Basic file and directory synchronization"
    priority: 1
    enabled: true
    source:
      repo: "mrz1836/template-repo"
      branch: "master"
    targets:
      - repo: "mrz1836/target-repo"
        files:
          - src: ".github/workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
        directories:
          - src: ".github/actions"
            dest: ".github/actions"
            exclude: ["*.out", "*.test"]
        transform:
          repo_name: true
```

### Run Sync

```bash
# Validate configuration
go-broadcast validate --config sync.yaml

# Preview changes (dry run)
go-broadcast sync --dry-run --config sync.yaml

# Execute sync
go-broadcast sync --config sync.yaml
```

#### Example Dry-Run Output

When using `--dry-run`, go-broadcast provides clean, readable output showing exactly what would happen:

```
🔍 📋 COMMIT PREVIEW
┌─────────────────────────────────────────────────────────────────
│ Message: sync: update 2 files from source repository
│ Files: 2 changed
│ README.md, .mage.yaml
└─────────────────────────────────────────────────────────────────

📄 FILE CHANGES:
   ✨ README.md (added) (+1249 bytes)
   📝 .mage.yaml (modified) (+45 bytes)

🔍 DRY-RUN: Pull Request Preview
┌─────────────────────────────────────────────────────────────────
│ Repository: company/service-name
│ Branch: chore/sync-files-20250130-143052-abc123f
├─────────────────────────────────────────────────────────────────
│ Title: [Sync] Update project files from source repository (abc123f)
├─────────────────────────────────────────────────────────────────
│ ## What Changed
│ * Updated project files to synchronize with the latest ch...
│
│ ## Changed Files
│
│ - `README.md` (added)
│ - `.mage.yaml` (modified)
└─────────────────────────────────────────────────────────────────

✅ DRY-RUN SUMMARY: Repository sync preview completed successfully
   📁 Repository: company/service-name
   🌿 Branch: chore/sync-files-20250130-143052-abc123f
   📝 Files: 2 would be changed
   🔗 Commit: dry-run-commit-sha
   💡 Run without --dry-run to execute these changes
```

**That's it!** 🎉 go-broadcast automatically:
- Executes each group in priority order
- Clones your template repository
- Applies configured transformations
- Creates a branch in each target repository
- Commits synchronized files
- Opens a pull request for review

> **💡 Pro tip:** go-broadcast includes a [built-in coverage system](https://github.com/mrz1836/go-coverage), [enterprise performance](#-performance), and comprehensive logging & debugging - explore the features below!

<br/>

### Install [MAGE-X](https://github.com/mrz1836/mage-x) build tool
Want to contribute to go-broadcast? Use MAGE-X for building, testing, linting, and more.

```bash
# Install MAGE-X for development and building
go install github.com/mrz1836/mage-x/cmd/magex@latest
magex update:install
```

<br/>

## ✨ Key Features

**go-broadcast** is a production-grade repository synchronization platform with enterprise performance:

### 🚀 **Core Synchronization Engine**
- **Stateless architecture** - All state derived from GitHub (branches, PRs, commits)
- **File & directory sync** - Individual files or entire directories with intelligent filtering
- **Mixed sync operations** - Combine files and directories in single configurations
- **Smart diff detection** - Only syncs files that actually changed (content-based)
- **Zero-downtime operations** - Works at any scale without state corruption
- **Full audit trail** - Every operation tracked in Git history with metadata

### ⚡ **Enterprise Performance**
- **587M+ ops/sec** - Binary detection with zero memory allocations
- **239M+ ops/sec** - Content comparison for identical files
- **32ms/1000 files** - Directory processing with concurrent workers
- **90%+ API reduction** - GitHub Tree API optimization
- **Worker pools** - Concurrent task execution with panic recovery
- **TTL caching** - High-performance cache with 13.5M+ ops/sec

### 🎯 **Intelligent Configuration**
- **Group-based organization** - Logical grouping with names, IDs, and descriptions
- **Priority execution** - Groups execute in order (lower number = higher priority)
- **Dependency management** - Groups can depend on successful completion of others
- **Enable/disable control** - Toggle groups without removing configuration
- **Reusable lists** - Define file/directory lists once, use everywhere
- **Module-aware sync** - Version management for Go modules with semantic versioning

### 🔄 **Advanced Transformations**
- **Variable substitution** - Template variables ({{VAR}} and ${VAR} syntax)
- **Go module updates** - Automatic repository name transformation
- **Pattern-based transforms** - Apply to all files in directories
- **Context-aware** - Different transforms per target repository

### 🤖 **Automation & CI/CD**
- **Automatic PR creation** - Creates pull requests with rich metadata
- **PR management** - Auto-assign reviewers, assignees, and labels
- **Automerge labels** - Add configurable automerge labels to PRs with `--automerge` flag
- **Global settings** - Organization-wide PR assignments
- **Branch naming** - Encoded metadata for state tracking
- **Cancel operations** - Abort active syncs with cleanup
- **Self-updating** - Built-in upgrade command with version management

### 📊 **Developer Experience**
- **Rich dry-run mode** - Beautiful previews with exact change details
- **Component debugging** - Targeted debugging (--debug-git, --debug-api)
- **Verbose logging** - Multi-level verbosity (-v, -vv, -vvv)
- **Progress tracking** - Real-time sync progress with statistics
- **Diagnostic tool** - Comprehensive system information collection
- **Status command** - Repository synchronization status overview

### 🛡️ **Production Ready**
- **85%+ test coverage** - Comprehensive unit and integration tests
- **Fuzz testing** - Security-critical components fuzz tested
- **60+ linters** - Zero tolerance for code issues via golangci-lint
- **Vulnerability scanning** - govulncheck, nancy, CodeQL, OSSAR
- **OpenSSF Scorecard** - Supply chain security assessment
- **Secret detection** - gitleaks integration prevents leaks

### 🎨 **Smart Defaults & Exclusions**
- **Automatic exclusions** - Filters out *.out, *.test, *.exe, .DS_Store, tmp/
- **Custom patterns** - Add your own exclusion patterns per directory
- **Hidden file control** - Include or exclude dotfiles as needed
- **Binary detection** - Intelligent binary file detection prevents corruption
- **Preserve structure** - Maintains nested directory hierarchies

<br/>


## 🔍 How It Works

**go-broadcast** uses a **stateless architecture** that tracks synchronization state through GitHub itself - no databases or state files needed!

### State Tracking Through Branch Names

Every sync operation creates a branch with encoded metadata:

```
[chore/sync-files]-[group1]-[20250123-143052]-[abc123f]
        │             │              │            │
        │             │              │            └─── Source branch commit SHA (7 chars)
        │ 		 	  │              └──────────────── Timestamp (YYYYMMDD-HHMMSS)
        │ 			  └─────────────────────────────── Group ID (from config)
        └───────────────────────────────────────────── Configurable prefix
```

### How go-broadcast Determines What to Sync

1. **State Discovery** - Queries GitHub to find:
   - Latest commit in source repository
   - All sync branches in target repositories
   - Open sync pull requests

2. **Smart Comparison** - For each target:
   ```
   Source commit: abc123f (latest)
   Target's last sync: def456g (from branch name)
   Status: Behind → Needs sync ✓
   ```

3. **Content-Based Sync** - Only syncs files that actually changed:
   - Fetches current files from target (individual files or directory contents)
   - Applies transformations to source files
   - Compares content byte-by-byte with smart exclusion filtering
   - Skips unchanged files and processes directories concurrently

### Pull Request Metadata

Each PR includes structured metadata for complete traceability:

```text
<!-- go-broadcast metadata
group:
  id: default
  name: Default Sync
source:
  repo: company/template-repo
  branch: master
  commit: abc123f7890
files:
  - src: .github/workflows/ci.yml
    dest: .github/workflows/ci.yml
directories:
  - src: .github/actions
    dest: .github/actions
    excluded: ["*.out", "*.test"]
    files_synced: 87
    processing_time_ms: 4
performance:
  total_files: 88
  api_calls_saved: 79
timestamp: 2025-01-23T14:30:52Z
-->
```

### Why This Approach is Powerful

- **No State Files** - Everything lives in GitHub
- **Atomic Operations** - Each sync is self-contained
- **Full Audit Trail** - Branch and PR history shows all syncs
- **Disaster Recovery** - State can be reconstructed from GitHub
- **Works at Scale** - No state corruption with concurrent syncs

<br/>


## 💡 Usage Examples

### Common Use Cases

<details>
<summary><strong>Sync CI/CD workflows across microservices</strong></summary>

```yaml
version: 1
groups:
  - name: "CI/CD Templates"
    id: "ci-cd"
    priority: 1
    enabled: true
    source:
      repo: "company/ci-templates"
      branch: "main"
    targets:
      - repo: "company/user-service"
        files:
          - src: "workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
        transform:
          variables:
            SERVICE_NAME: "user-service"
```
</details>

<details>
<summary><strong>Sync entire directories with smart exclusions</strong></summary>

```yaml
version: 1
groups:
  - name: "GitHub Configuration"
    id: "github-config"
    priority: 1
    enabled: true
    source:
      repo: "company/ci-templates"
      branch: "main"
    targets:
      - repo: "company/microservice-a"
        directories:
          - src: ".github/workflows"
            dest: ".github/workflows"
            exclude: ["*-local.yml", "*.disabled"]
          - src: ".github/actions"
            dest: ".github/actions"
            # Smart defaults automatically exclude: *.out, *.test, *.exe, **/.DS_Store
        transform:
          repo_name: true
```
</details>

<details>
<summary><strong>Mixed file and directory synchronization</strong></summary>

```yaml
version: 1
groups:
  - name: "Mixed Content Sync"
    id: "mixed-sync"
    priority: 1
    enabled: true
    source:
      repo: "company/template-repo"
      branch: "main"
    targets:
      - repo: "company/service"
        files:
          - src: "Makefile"
            dest: "Makefile"
        directories:
          - src: "configs"
            dest: "configs"
            exclude: ["*.local", "*.secret"]
        transform:
          variables:
            SERVICE_NAME: "user-service"
```
</details>

<details>
<summary><strong>Automated PR management with assignees, reviewers, and labels</strong></summary>

```yaml
version: 1
groups:
  - name: "Security Policies"
    id: "security-sync"
    priority: 1
    enabled: true
    source:
      repo: "company/security-templates"
      branch: "main"
    defaults:
      pr_labels: ["automated-sync", "chore"]
      pr_assignees: ["tech-lead", "platform-team"]
      pr_reviewers: ["senior-dev1", "senior-dev2"]
      pr_team_reviewers: ["architecture-team"]
    targets:
      - repo: "company/critical-service"
        files:
          - src: "security/policies.yml"
            dest: "security/policies.yml"
        # Critical service needs security team review
        pr_labels: ["security-update", "high-priority"]
        pr_assignees: ["security-lead"]
        pr_reviewers: ["security-engineer"]
        pr_team_reviewers: ["security-team"]
```
</details>

<details>
<summary><strong>File and directory cleanup with deletions</strong></summary>

```yaml
version: 1
groups:
  - name: "Repository Cleanup"
    id: "cleanup-sync"
    description: "Clean up obsolete files while syncing new ones"
    priority: 1
    enabled: true
    source:
      repo: "company/template-repo"
      branch: "main"
    targets:
      - repo: "company/service-a"
        # Regular file syncing
        files:
          - src: ".github/workflows/new-ci.yml"
            dest: ".github/workflows/ci.yml"

        # File deletions - remove obsolete files
        # Note: src can be empty or omitted when delete: true
        files:
          - dest: ".github/workflows/old-ci.yml"
            delete: true
          - dest: "deprecated-config.json"
            delete: true

        # Directory operations
        directories:
          - src: ".github/actions"
            dest: ".github/actions"

        # Directory deletions - remove entire directories
        directories:
          - dest: "old-docs"
            delete: true
          - dest: ".github/legacy-workflows"
            delete: true

        transform:
          repo_name: true
```
</details>

<details>
<summary><strong>Target different branches for development workflows</strong></summary>

```yaml
version: 1
groups:
  - name: "Development Sync"
    id: "dev-sync"
    priority: 1
    enabled: true
    source:
      repo: "company/template-repo"
      branch: "main"
    targets:
      # Sync to main branch (production)
      - repo: "company/service-a"
        branch: "main"
        files:
          - src: ".github/workflows/production.yml"
            dest: ".github/workflows/ci.yml"

      # Sync to develop branch (staging)
      - repo: "company/service-a"
        branch: "develop"
        files:
          - src: ".github/workflows/staging.yml"
            dest: ".github/workflows/ci.yml"
        transform:
          variables:
            ENVIRONMENT: "staging"

      # Sync to feature branch
      - repo: "company/service-b"
        branch: "feature/new-deployment"
        files:
          - src: ".github/workflows/feature.yml"
            dest: ".github/workflows/ci.yml"
        transform:
          variables:
            ENVIRONMENT: "development"
```
</details>

<details>
<summary><strong>Review and merge pull requests</strong></summary>

```bash
# Review and merge a single PR
go-broadcast review-pr https://github.com/owner/repo/pull/123

# Review and merge multiple PRs in batch
go-broadcast review-pr \
  https://github.com/owner/repo/pull/123 \
  https://github.com/owner/repo/pull/124 \
  https://github.com/owner/repo/pull/125

# Use short format
go-broadcast review-pr owner/repo#123

# Review all PRs assigned to you (excludes drafts)
go-broadcast review-pr --all-assigned-prs

# Customize the review approval message
go-broadcast review-pr --message "Approved after testing" \
  https://github.com/owner/repo/pull/123

# Preview all assigned PRs without executing (dry run)
go-broadcast review-pr --all-assigned-prs --dry-run

# Review all assigned PRs with custom message
go-broadcast review-pr --all-assigned-prs --message "LGTM"
```

The `review-pr` command will:
1. Parse the PR URL(s) to extract owner, repo, and PR number (or fetch all assigned PRs with `--all-assigned-prs`)
2. Check if the PR is already merged or closed
3. Submit an approving review with your message (default: "LGTM")
4. Detect the repository's preferred merge method (squash, merge, or rebase)
5. Intelligently merge the PR using a try-and-fallback approach:
   - Tries to merge immediately first (optimistic)
   - If branch protection blocks merge, automatically enables auto-merge
   - Handles merge conflicts, pending checks, and required reviews gracefully

**All Assigned PRs Mode (`--all-assigned-prs`):**
- Automatically fetches all open pull requests assigned to you
- Excludes draft PRs from processing (only ready-for-review PRs are included)
- Cannot be used together with explicit PR URLs (mutually exclusive)
- Perfect for bulk reviewing and merging your assigned PRs
- Works with `--dry-run` to preview what would be processed
- Processes each PR sequentially with the same smart merge behavior

**Smart Merge Behavior (Try-and-Fallback):**
- The command uses an intelligent try-first approach for maximum efficiency
- **Merge conflicts detected**: If the PR has merge conflicts, it enables auto-merge immediately
  - Warning: "⚠️  PR has merge conflicts - enabling auto-merge for when conflicts are resolved"
- **Optimistic merge attempt**: For all other PRs, it tries to merge immediately first
  - **Success**: If merge succeeds, the PR is merged right away ✓
  - **Branch protection detected**: If merge fails due to branch protection policies:
    - Automatically falls back to enabling auto-merge
    - Warning: "⚠️  Branch protection blocking merge - enabling auto-merge"
    - Success: "✓ Auto-merge enabled - will merge when requirements are met"
    - Handles: pending status checks, required reviews, or other protection rules
  - **Real errors**: Other errors (permissions, PR not found, etc.) fail as expected
- This ensures PRs get merged without manual intervention once all requirements are satisfied

**Merge Method Detection:**
- The command automatically queries the repository settings via GitHub API
- Uses the first allowed method in this priority: squash → merge → rebase
- Handles all three merge strategies supported by GitHub

**Batch Operations:**
- Process multiple PRs sequentially
- Continue on error (doesn't fail fast)
- Display summary showing:
  - How many PRs were merged immediately
  - How many have auto-merge enabled
  - How many failed
- All PRs are processed even if some fail
- Non-zero exit code if any PR fails

**Error Handling:**
- Skips PRs that are already merged (with warning)
- Reports PRs that are closed but not merged
- Provides clear error messages for failures
- Dry-run mode shows what would happen without making changes

</details>

### Essential Commands

```bash
# Validate and preview
go-broadcast validate --config sync.yaml
go-broadcast sync --dry-run --config sync.yaml

# Execute sync
go-broadcast sync --config sync.yaml
go-broadcast sync org/specific-repo --config sync.yaml

# Group-based sync operations
go-broadcast sync --groups "Default Sync" --config sync.yaml        # Sync only one group by name
go-broadcast sync --groups "default" --config sync.yaml             # Sync by group ID
go-broadcast sync --groups "core,security" --config sync.yaml       # Sync multiple groups
go-broadcast sync --skip-groups "experimental" --config sync.yaml   # Skip specific groups
go-broadcast sync --groups "core" org/repo1 --config sync.yaml      # Combine with target filtering

# Automerge configuration
go-broadcast sync --automerge --config sync.yaml                    # Add automerge labels to created PRs
go-broadcast sync --automerge --groups "core" --config sync.yaml    # Automerge with group filtering (adds labels)

# Monitor status
go-broadcast status --config sync.yaml

# Troubleshooting and diagnostics
go-broadcast diagnose                    # Collect system diagnostic information
go-broadcast diagnose > diagnostics.json # Save diagnostics to file

# Cancel active syncs
go-broadcast cancel                                        # Cancel all active sync operations
go-broadcast cancel org/repo1                              # Cancel syncs for specific repository
go-broadcast cancel --groups "core"                        # Cancel syncs for specific group (much faster!)
go-broadcast cancel --groups "core,security"               # Cancel syncs for multiple groups
go-broadcast cancel --groups "core" org/repo1              # Cancel specific repo in a group
go-broadcast cancel --skip-groups "experimental"           # Cancel all except experimental group
go-broadcast cancel --dry-run                              # Preview what would be cancelled

# Review and merge pull requests
go-broadcast review-pr <pr-url>                                      # Review and merge single PR
go-broadcast review-pr <url1> <url2> <url3>                         # Batch review and merge multiple PRs
go-broadcast review-pr --message "Looks good!" <pr-url>             # Custom review message
go-broadcast review-pr --dry-run <pr-url>                           # Preview without executing
go-broadcast review-pr owner/repo#123                               # Use short format
go-broadcast review-pr https://github.com/owner/repo/pull/123       # Use full URL
go-broadcast review-pr --all-assigned-prs                           # Review all PRs assigned to you
go-broadcast review-pr --all-assigned-prs --dry-run                 # Preview all assigned PRs
go-broadcast review-pr --all-assigned-prs --message "LGTM"          # Custom message for all assigned PRs

# Upgrade go-broadcast
go-broadcast upgrade                     # Upgrade to latest version
go-broadcast upgrade --check             # Check for updates without upgrading
go-broadcast upgrade --force             # Force upgrade even if already on latest
go-broadcast upgrade --verbose           # Show release notes after upgrade
```

### Configuration Reference

<details>
<summary><strong>🔄 File Transformations</strong></summary>

```yaml
transform:
  repo_name: true  # Updates Go module paths
  variables:
    SERVICE_NAME: "my-service"    # {{SERVICE_NAME}} → my-service
    ENVIRONMENT: "production"     # ${ENVIRONMENT} → production
```
</details>

<details>
<summary><strong>📁 File & Directory Mapping Options</strong></summary>

**File Mapping:**
```yaml
files:
  - src: "Makefile"         # Copy to same location
    dest: "Makefile"
  - src: "template.md"      # Rename during sync
    dest: "README.md"
  - src: "config/app.yml"   # Move to different directory
    dest: "configs/app.yml"
  - dest: "old-config.json" # Delete file (src can be omitted)
    delete: true
  - dest: "deprecated.yml"  # Delete file with explicit empty src
    src: ""
    delete: true
```

**Directory Mapping:**
```yaml
directories:
  - src: ".github/workflows"           # Basic directory sync
    dest: ".github/workflows"
  - src: ".github/actions"            # Directory with exclusions
    dest: ".github/actions"
    exclude: ["*.out", "*.test", "go-coverage"]
  - src: "docs"                        # Advanced directory options
    dest: "documentation"
    exclude: ["*.tmp", "**/draft/*"]
    preserve_structure: true           # Keep nested structure (default: true)
    include_hidden: true               # Include hidden files (default: true)
    transform:                         # Apply transforms to all files
      variables:
        VERSION: "v2.0"
  - dest: "legacy-docs"                # Delete entire directory
    delete: true
  - dest: "old-scripts"                # Delete directory (src can be omitted)
    src: ""
    delete: true
```

**Smart Default Exclusions:**
Automatically applied to all directories: `*.out`, `*.test`, `*.exe`, `**/.DS_Store`, `**/tmp/*`, `**/.git`

**File and Directory Deletions:**
- Set `delete: true` to remove files or directories from target repositories
- When deleting, the `src` field can be omitted or set to an empty string
- Deletions are processed alongside regular sync operations
- Perfect for cleaning up deprecated files, old CI workflows, or restructuring projects

</details>

<details>
<summary><strong>⚙️ Advanced Configuration</strong></summary>

```yaml
version: 1
groups:
  - name: "Platform Configuration"
    id: "platform-config"
    priority: 1
    enabled: true
    source:
      repo: "org/template-repo"
      branch: "master"
    # Global PR settings applied to ALL targets (merged with target-specific settings)
    global:
      pr_labels: ["automated-sync", "chore"]
      pr_assignees: ["platform-team"]
      pr_reviewers: ["platform-lead"]
      pr_team_reviewers: ["infrastructure-team"]
    # Default settings (fallback when no global or target settings)
    defaults:
      branch_prefix: "chore/sync-files"
      pr_labels: ["maintenance"]
      pr_assignees: ["maintainer1", "maintainer2"]
      pr_reviewers: ["reviewer1", "reviewer2"]
      pr_team_reviewers: ["platform-team"]
    targets:
      - repo: "org/target-repo"
        branch: "main"                        # Target branch for PRs (defaults to repo's default branch)
        files:
          - src: ".github/workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
        directories:
          - src: ".github/actions"
            dest: ".github/actions"
            exclude: ["*.out", "go-coverage"]
        transform:
          repo_name: true
          variables:
            ENVIRONMENT: "production"
        # Additional PR settings merged with global settings
        # Final labels: ["automated-sync", "chore", "service-specific"]
        pr_labels: ["service-specific"]
        # Final assignees: ["platform-team", "service-owner"]
        pr_assignees: ["service-owner"]
        # Final reviewers: ["platform-lead", "service-reviewer"]
        pr_reviewers: ["service-reviewer"]
```
</details>

<details>
<summary><strong>📋 Reusable File & Directory Lists</strong></summary>

Define reusable file and directory lists to reduce configuration repetition when syncing the same files to multiple repositories.

```yaml
version: 1

# Define reusable file lists
file_lists:
  - id: "common-github-files"
    name: "Common GitHub Files"
    description: "Standard GitHub configuration files"
    files:
      - src: ".github/CODE_OF_CONDUCT.md"
        dest: ".github/CODE_OF_CONDUCT.md"
      - src: ".github/SECURITY.md"
        dest: ".github/SECURITY.md"
      - src: ".github/SUPPORT.md"
        dest: ".github/SUPPORT.md"

  - id: "editor-config"
    name: "Editor Configuration"
    description: "Editor and code formatting files"
    files:
      - src: ".editorconfig"
        dest: ".editorconfig"
      - src: ".gitattributes"
        dest: ".gitattributes"

# Define reusable directory lists
directory_lists:
  - id: "github-workflows"
    name: "GitHub Actions Workflows"
    description: "Standard CI/CD workflows"
    directories:
      - src: ".github/workflows"
        dest: ".github/workflows"
        exclude: ["*.tmp", "*.local"]

groups:
  - name: "standard-sync"
    id: "standard-sync"
    source:
      repo: "org/template-repo"
      branch: "master"
    targets:
      # Use lists for multiple repos
      - repo: "org/service-a"
        file_list_refs: ["common-github-files", "editor-config"]
        directory_list_refs: ["github-workflows"]
        # Can still add inline files
        files:
          - src: "LICENSE"
            dest: "LICENSE"

      - repo: "org/service-b"
        file_list_refs: ["common-github-files"]
        directory_list_refs: ["github-workflows"]
```

**Benefits:**
- Define file/directory lists once, use many times
- Easy updates - change lists in one place
- Mix references with inline files/directories
- Clear organization and reduced YAML duplication

</details>

<details>
<summary><strong>❌ Cancel Sync Operations</strong></summary>

When issues arise, you can cancel active sync operations to prevent unwanted changes. Group filtering makes cancellation much faster for multi-group configurations.

**Cancel sync operations when issues arise:**
```bash
# Cancel all active syncs (closes PRs and deletes branches)
go-broadcast cancel --config sync.yaml

# Cancel syncs for specific repositories only
go-broadcast cancel company/service1 company/service2

# Cancel syncs for specific groups (much faster for multi-group configs!)
go-broadcast cancel --groups "core"
go-broadcast cancel --groups "core,security"

# Cancel specific repository in a specific group
go-broadcast cancel --groups "third-party-libraries" company/repo1

# Cancel all except experimental group
go-broadcast cancel --skip-groups "experimental"

# Preview what would be cancelled without making changes
go-broadcast cancel --dry-run --config sync.yaml

# Close PRs but keep sync branches for later cleanup
go-broadcast cancel --keep-branches --config sync.yaml

# Add custom comment when closing PRs
go-broadcast cancel --comment "Cancelling due to template update" --config sync.yaml
```

**Performance Benefits:**
- **Without group filtering**: Scans all groups in your config (100+ API calls for 5 groups)
- **With `--groups "core"`**: Scans only the core group (20 API calls)
- **Result**: 5x faster cancellation for multi-group configurations

</details>

<details>
<summary><strong>🌐 Global PR Assignment Configuration</strong></summary>

The `global` section allows you to define PR assignments (labels, assignees, reviewers, team reviewers) that are **merged** with target-specific assignments rather than overridden. This provides powerful control over PR workflows across all repositories.

#### How It Works

**Merge Priority**: `global` + `target` → `defaults` (fallback)

- **Global settings** apply to ALL target repositories
- **Target settings** are merged with global settings (duplicates removed)
- **Default settings** are used only when neither global nor target settings exist

#### Example Configuration

```yaml
version: 1
groups:
  - name: "Workflow Distribution"
    id: "workflow-dist"
    priority: 1
    enabled: true
    source:
      repo: "org/template-repo"
      branch: "master"
    # Applied to ALL PRs across all targets
    global:
      pr_labels: ["automated-sync", "chore"]
      pr_assignees: ["platform-team"]
      pr_reviewers: ["platform-lead"]
      pr_team_reviewers: ["infrastructure-team"]
    # Fallback settings (used only if no global/target assignments)
    defaults:
      branch_prefix: "chore/sync-files"
      pr_labels: ["maintenance"]
    targets:
      # This repo gets ONLY global settings
      - repo: "org/service-a"
        files:
          - src: ".github/workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
        # Effective PR settings:
        # Labels: ["automated-sync", "chore"]
        # Assignees: ["platform-team"]
        # Reviewers: ["platform-lead"]
        # Team reviewers: ["infrastructure-team"]
      # This repo gets global + target merged
      - repo: "org/service-b"
        files:
          - src: ".github/workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
        pr_labels: ["critical", "service-b"]
        pr_assignees: ["service-b-owner"]
        # Effective PR settings (merged):
        # Labels: ["automated-sync", "chore", "critical", "service-b"]
        # Assignees: ["platform-team", "service-b-owner"]
        # Reviewers: ["platform-lead"] (from global)
        # Team reviewers: ["infrastructure-team"] (from global)
```

#### Use Cases

- **Organization-wide standards**: Apply consistent labels and assignees across all repositories
- **Platform team oversight**: Ensure platform team is always assigned to infrastructure changes
- **Security requirements**: Add security team reviewers to all template updates
- **Compliance labeling**: Automatically tag all PRs with audit/compliance labels
</details>

<br/>

## 📚 Documentation

- **Quick Start** – Get up and running in 5 minutes with the [Quick Start guide](#-quick-start)
- **Configuration Guide** – Complete guide to group-based configuration at [docs/configuration-guide.md](docs/configuration-guide.md)
- **Module-Aware Sync** – Smart module versioning and synchronization at [docs/module-sync.md](docs/module-sync.md)
- **Group Examples** – Practical configuration patterns at [docs/group-examples.md](docs/group-examples.md)
- **Usage Examples** – Real-world scenarios in the [Usage Examples section](#-usage-examples)
- **AI Sub-Agents Guide** – Comprehensive guide to [26 specialized AI agents](docs/sub-agents.md) for repository management
- **Slash Commands Reference** – 20+ powerful [Claude Code commands](docs/slash-commands.md) for automated workflows
- **Directory Sync Guide** – Complete guide to directory synchronization at [docs/directory-sync.md](docs/directory-sync.md)
- **Configuration Examples** – Browse practical patterns in the [examples directory](examples)
- **Troubleshooting** – Solve common issues with the [troubleshooting guide](docs/troubleshooting.md)
- **API Reference** – Dive into the godocs at [pkg.go.dev/github.com/mrz1836/go-broadcast](https://pkg.go.dev/github.com/mrz1836/go-broadcast)
- **Integration Tests** – End-to-end testing examples in [test/integration](test/integration)
- **Internal Utilities** – Shared testing and validation utilities in [internal](internal) packages
- **Performance** – Check the latest numbers in the [Performance section](#-performance)

<br/>

<details>
<summary><strong>Repository Features</strong></summary>
<br/>

* **Continuous Integration on Autopilot** with [GitHub Actions](https://github.com/features/actions) – every push is built, tested, and reported in minutes.
* **Pull‑Request Flow That Merges Itself** thanks to [auto‑merge](.github/workflows/auto-merge-on-approval.yml) and hands‑free [Dependabot auto‑merge](.github/workflows/dependabot-auto-merge.yml).
* **One‑Command Builds** powered by battle‑tested [MAGE-X](https://github.com/mrz1836/mage-x) targets for linting, testing, releases, and more.
* **First‑Class Dependency Management** using native [Go Modules](https://github.com/golang/go/wiki/Modules).
* **Uniform Code Style** via [gofumpt](https://github.com/mvdan/gofumpt) plus zero‑noise linting with [golangci‑lint](https://github.com/golangci/golangci-lint).
* **Confidence‑Boosting Tests** with [testify](https://github.com/stretchr/testify), the Go [race detector](https://blog.golang.org/race-detector), crystal‑clear [HTML coverage](https://blog.golang.org/cover) snapshots, and automatic reporting via internal coverage system.
* **Hands‑Free Releases** delivered by [GoReleaser](https://github.com/goreleaser/goreleaser) whenever you create a [new Tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging).
* **Relentless Dependency & Vulnerability Scans** via [Dependabot](https://dependabot.com) (runs daily at 8am to ensure broadcast dependencies are always current), [Nancy](https://github.com/sonatype-nexus-community/nancy), and [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck).
* **Security Posture by Default** with [CodeQL](https://docs.github.com/en/github/finding-security-vulnerabilities-and-errors-in-your-code/about-code-scanning), [OpenSSF Scorecard](https://openssf.org), and secret‑leak detection via [gitleaks](https://github.com/gitleaks/gitleaks).
* **Automatic Syndication** to [pkg.go.dev](https://pkg.go.dev/) on every release for instant godoc visibility.
* **Polished Community Experience** using rich templates for [Issues & PRs](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-go-broadcastsitory).
* **All the Right Meta Files** (`LICENSE`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SUPPORT.md`, `SECURITY.md`) pre‑filled and ready.
* **Code Ownership** clarified through a [CODEOWNERS](.github/CODEOWNERS) file, keeping reviews fast and focused.
* **Zero‑Noise Dev Environments** with tuned editor settings (`.editorconfig`) plus curated *ignore* files for [VS Code](.editorconfig), [Docker](.dockerignore), and [Git](.gitignore).
* **Label Sync Magic**: your repo labels stay in lock‑step with [.github/labels.yml](.github/labels.yml).
* **Friendly First PR Workflow** – newcomers get a warm welcome thanks to a dedicated [workflow](.github/workflows/pull-request-management.yml).
* **Standards‑Compliant Docs** adhering to the [standard‑readme](https://github.com/RichardLitt/standard-readme/blob/master/spec.md) spec.
* **Instant Cloud Workspaces** via [Gitpod](https://gitpod.io/) – spin up a fully configured dev environment with automatic linting and tests.
* **Out‑of‑the‑Box VS Code Happiness** with a preconfigured [Go](https://code.visualstudio.com/docs/languages/go) workspace and [`.vscode`](.vscode) folder with all the right settings.
* **Optional Release Broadcasts** to your community via [Slack](https://slack.com), [Discord](https://discord.com), or [Twitter](https://twitter.com) – plug in your webhook.
* **AI Compliance Playbook** – machine‑readable guidelines ([AGENTS.md](.github/AGENTS.md), [CLAUDE.md](.github/CLAUDE.md), [.cursorrules](.cursorrules), [sweep.yaml](.github/sweep.yaml)) keep ChatGPT, Claude, Cursor & Sweep aligned with your repo's rules.
* **20+ Powerful Slash Commands** – Claude Code commands that coordinate 26 specialized AI agents for automated workflows like `/test`, `/security`, `/release`, and more. See [docs/slash-commands.md](docs/slash-commands.md).
* **Go-Pre-commit System** - [High-performance Go-native pre-commit hooks](https://github.com/mrz1836/go-pre-commit) with 17x faster execution—run the same formatting, linting, and tests before every commit, just like CI.
* **Zero Python Dependencies** - Pure Go implementation with environment-based configuration via [.env.base](.github/.env.base).
* **DevContainers for Instant Onboarding** – Launch a ready-to-code environment in seconds with [VS Code DevContainers](https://containers.dev/) and the included [.devcontainer.json](.devcontainer.json) config.

</details>

<details>
<summary><strong>Library Deployment</strong></summary>
<br/>

This project uses [goreleaser](https://github.com/goreleaser/goreleaser) for streamlined binary and library deployment to GitHub. To get started, install it via:

```bash
brew install goreleaser
```

The release process is defined in the [.goreleaser.yml](.goreleaser.yml) configuration file.


Then create and push a new Git tag using:

```bash
magex version:bump push=true bump=patch
```

This process ensures consistent, repeatable releases with properly versioned artifacts and citation metadata.

</details>

<details>
<summary><strong>Build Commands</strong></summary>
<br/>

View all build commands

```bash script
magex help
```

</details>

<details>
<summary><strong>GitHub Workflows</strong></summary>
<br/>


### 🎛️ The Workflow Control Center

All GitHub Actions workflows in this repository are powered by a single configuration files – your one-stop shop for tweaking CI/CD behavior without touching a single YAML file! 🎯

**Configuration Files:**
- **[.env.base](.github/.env.base)** – Default configuration that works for most Go projects
- **[.env.custom](.github/.env.custom)** – Optional project-specific overrides

This magical file controls everything from:
- **⚙️ Go version matrix** (test on multiple versions or just one)
- **🏃 Runner selection** (Ubuntu or macOS, your wallet decides)
- **🔬 Feature toggles** (coverage, fuzzing, linting, race detection, benchmarks)
- **🛡️ Security tool versions** (gitleaks, nancy, govulncheck)
- **🤖 Auto-merge behaviors** (how aggressive should the bots be?)
- **🏷️ PR management rules** (size labels, auto-assignment, welcome messages)

<br/>

| Workflow Name                                                                      | Description                                                                                                            |
|------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------|
| [auto-merge-on-approval.yml](.github/workflows/auto-merge-on-approval.yml)         | Automatically merges PRs after approval and all required checks, following strict rules.                               |
| [codeql-analysis.yml](.github/workflows/codeql-analysis.yml)                       | Analyzes code for security vulnerabilities using [GitHub CodeQL](https://codeql.github.com/).                          |
| [dependabot-auto-merge.yml](.github/workflows/dependabot-auto-merge.yml)           | Automatically merges [Dependabot](https://github.com/dependabot) PRs that meet all requirements.                       |
| [fortress.yml](.github/workflows/fortress.yml)                                     | Runs the GoFortress security and testing workflow, including linting, testing, releasing, and vulnerability checks.    |
| [pull-request-management.yml](.github/workflows/pull-request-management.yml)       | Labels PRs by branch prefix, assigns a default user if none is assigned, and welcomes new contributors with a comment. |
| [scorecard.yml](.github/workflows/scorecard.yml)                                   | Runs [OpenSSF](https://openssf.org/) Scorecard to assess supply chain security.                                        |
| [stale.yml](.github/workflows/stale-check.yml)                                     | Warns about (and optionally closes) inactive issues and PRs on a schedule or manual trigger.                           |
| [sync-labels.yml](.github/workflows/sync-labels.yml)                               | Keeps GitHub labels in sync with the declarative manifest at [`.github/labels.yml`](./.github/labels.yml).             |

</details>

<details>
<summary><strong>Updating Dependencies</strong></summary>
<br/>

To update all dependencies (Go modules, linters, and related tools), run:

```bash
magex deps:update
```

This command ensures all dependencies are brought up to date in a single step, including Go modules and any tools managed by [MAGE-X](https://github.com/mrz1836/mage-x). It is the recommended way to keep your development environment and CI in sync with the latest versions.

</details>

<details>
<summary><strong>Pre-commit Hooks</strong></summary>
<br/>

Set up the Go-Pre-commit System to run the same formatting, linting, and tests defined in [AGENTS.md](.github/AGENTS.md) before every commit:

```bash
go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@latest
go-pre-commit install
```

The system is configured via [.env.base](.github/.env.base) and can be customized using also using [.env.custom](.github/.env.custom) and provides 17x faster execution than traditional Python-based pre-commit hooks. See the [complete documentation](http://github.com/mrz1836/go-pre-commit) for details.

</details>

<details>
<summary><strong>Logging and Debugging</strong></summary>

go-broadcast provides comprehensive logging capabilities designed for debugging, monitoring, and troubleshooting. The logging system features intuitive verbose flags, component-specific debug modes, and automatic sensitive data redaction.

### Quick Start

```bash
# Basic logging levels
go-broadcast sync --log-level debug     # Debug level logging
go-broadcast sync --log-level info      # Info level logging (default)
go-broadcast sync --log-level warn      # Warning level logging
go-broadcast sync --log-level error     # Error level logging

# Collect comprehensive diagnostic information
go-broadcast diagnose                    # Display diagnostic info to stdout
go-broadcast diagnose > diagnostics.json # Save diagnostics to file for support
```

**Note**: Advanced verbose flags (`-v`, `-vv`, `-vvv`) and component-specific debug flags (`--debug-git`, `--debug-api`, etc.) are planned features not yet implemented. The current implementation supports `--log-level` for basic debugging.

### Log Levels

- **ERROR**: Critical failures that prevent operation
- **WARN**: Important issues that don't stop execution
- **INFO**: High-level operation progress (default)
- **DEBUG**: Detailed operation information (`--log-level debug`)

### Advanced Logging Features

#### Performance Monitoring
All operations are timed automatically. Look for `duration_ms` in logs:
```bash
# Find slow operations
go-broadcast sync --log-format json 2>&1 | \
  jq -r 'select(.duration_ms > 5000) | "\(.operation): \(.duration_ms)ms"'
```

#### Security and Compliance
- All tokens and secrets are automatically redacted
- Audit trail for configuration changes and repository access
- No sensitive data is ever logged

#### Troubleshooting Common Issues

**Git Authentication Issues**
```bash
# Enable debug logging to see git operations
go-broadcast sync --log-level debug

# Common indicators:
# - "Authentication failed" in git output
# - "Permission denied" errors
# - Check GH_TOKEN or GITHUB_TOKEN environment variables
```

**API Rate Limiting**
```bash
# Monitor operations with debug logging
go-broadcast sync --log-level debug 2>&1 | grep -i "rate"
```

**File Transformation Issues**
```bash
# Use debug logging to see operation details
go-broadcast sync --log-level debug

# Debug output shows:
# - File operations
# - Configuration processing
# - Error details
```

**State Discovery Problems**
```bash
# Enable debug logging for repository state information
go-broadcast sync --log-level debug

# Debug output includes:
# - Repository access attempts
# - Branch discovery
# - File discovery process
```

### Log Management

#### Debug Sessions
```bash
# Save debug session
go-broadcast sync --log-level debug 2> debug-$(date +%Y%m%d-%H%M%S).log

# Review debug logs
go-broadcast sync --log-level debug 2>&1 | tee sync-debug.log
```

#### Diagnostic Information Collection

The `diagnose` command collects comprehensive system information for troubleshooting:

```bash
# Collect all diagnostic information (JSON format)
go-broadcast diagnose

# Information collected includes:
# - System details (OS, architecture, CPU count, hostname)
# - go-broadcast --version and build information
# - Git and GitHub CLI versions
# - Environment variables (sensitive data automatically redacted)
# - Configuration file status and validation results
# - Timestamp and runtime information
```

**Note**: JSON log format (`--log-format json`) is a planned feature. The `diagnose` command provides JSON output for system information.

### Environment Variables

| Variable                  | Description            | Example |
|---------------------------|------------------------|---------|
| `NO_COLOR`                | Disable colored output | `1`     |

**Note**: Environment variables for log level and format are planned features not yet implemented.

For more detailed information, see the [comprehensive logging guide](docs/logging.md) and [enhanced troubleshooting guide](docs/troubleshooting.md).

</details>

<br/>

## 🧪 Examples & Tests

All unit tests and [examples](examples) run via [GitHub Actions](https://github.com/mrz1836/go-broadcast/actions) and use [Go version 1.24.x](https://go.dev/doc/go1.24). View the [configuration file](.github/workflows/fortress.yml).

Run all tests (fast):

```bash script
magex test
```

Run all tests with race detector (slower):
```bash script
magex test:race
```

<br/>

## ⚡ Performance

**Enterprise-grade performance** - Designed for high-scale repository management with zero-allocation critical paths.

### Performance Highlights

| Operation              | Performance     | Memory           |
|------------------------|-----------------|------------------|
| **Binary Detection**   | 587M+ ops/sec   | Zero allocations |
| **Content Comparison** | 239M+ ops/sec   | Zero allocations |
| **Cache Operations**   | 13.5M+ ops/sec  | Minimal memory   |
| **Batch Processing**   | 23.8M+ ops/sec  | Concurrent safe  |
| **Directory Sync**     | 32ms/1000 files | Linear scaling   |
| **Exclusion Engine**   | 107ns/op        | Zero allocations |

### Quick Benchmarks

<details>
<summary><strong>Commands for Benchmarking</strong></summary>

```bash
# Run quick benchmarks (CI default, <5 minutes)
magex bench

# Run heavy benchmarks manually (10-30 minutes)
# Includes worker pools, large datasets, real-world scenarios
mage benchHeavy

# Run all benchmarks (30-60 minutes)
mage benchAll

# Benchmark specific components
go test -bench=. -benchmem ./internal/algorithms
go test -bench=. -benchmem ./internal/cache

# Run heavy benchmarks with custom settings
go test -bench=. -benchmem -tags=bench_heavy -benchtime=1s ./...

# Try the profiling demo
go run ./cmd/profile_demo
```

**Note:** Heavy benchmarks (worker pools with 1000+ tasks, large directory syncs, memory efficiency tests) are excluded from CI to prevent timeouts. Use `mage benchHeavy` for comprehensive performance testing during development.

</details>

<details>
<summary><strong>Complete Benchmark Results & Profiling Tools</strong></summary>

### Performance Analysis Tools

- **🔬 100+ Benchmarks** covering all major components
- **📊 CPU & Memory Profiling** with detailed analysis
- **📈 Performance Reports** in HTML, JSON, and Markdown
- **🔍 Goroutine Analysis** for concurrency debugging
- **⚡ Zero-Allocation** operations in critical paths

### Complete Performance Results

The following benchmarks were run on Apple M1 Max (updated January 2025):

| Benchmark                      | Operations  | ns/op   | B/op  | allocs/op |
|--------------------------------|-------------|---------|-------|-----------|
| **Core Algorithms**            |
| BinaryDetection (Small Text)   | 5,852,616   | 204.5   | 0     | 0         |
| BinaryDetection (Large Text)   | 179,217     | 6,606   | 0     | 0         |
| BinaryDetection (Small Binary) | 335,143,730 | 3.6     | 0     | 0         |
| BinaryDetection (Large Binary) | 587,204,924 | 2.0     | 0     | 0         |
| DiffOptimized (Identical)      | 239,319,295 | 5.0     | 0     | 0         |
| DiffOptimized (Different)      | 4,035,818   | 297.2   | 240   | 10        |
| DiffOptimized (Large Similar)  | 250,452     | 4,711   | 5,492 | 7         |
| BatchProcessor                 | 23,842,558  | 54.3    | 25    | 1         |
| **Cache Operations**           |
| Cache Set                      | 6,067,380   | 177.4   | 48    | 4         |
| Cache Get (Hit)                | 11,481,175  | 103.8   | 7     | 1         |
| Cache Get (Miss)               | 13,565,466  | 89.4    | 32    | 2         |
| Cache GetOrLoad                | 11,330,936  | 106.2   | 16    | 1         |
| **Performance Profiling**      |
| CaptureMemStats                | 58,352      | 20,476  | 0     | 0         |
| CaptureMemoryStats             | 3,475       | 302,402 | 107   | 4         |
| MeasureOperation               | 4,032       | 316,467 | 107   | 4         |

### Performance Characteristics

go-broadcast is designed for efficiency:

- **Binary detection** executes 587M+ operations/second with zero allocations for binary files
- **Content comparison** performs 239M+ operations/second for identical files with zero allocations
- **Cache operations** handle 13.5M+ get operations/second with minimal memory usage
- **Batch processing** manages 23.8M+ operations/second for concurrent tasks
- **Memory profiling** captures detailed statistics at 58K+ operations/second
- **Performance monitoring** measures operations at 3K+ captures/second with comprehensive metrics
- **Zero-allocation paths** optimized algorithms avoid memory allocation in critical operations
- **Concurrent operations** sync multiple repositories simultaneously (configurable concurrency)
- **GitHub API optimization** reduces API calls through intelligent state discovery
- **Memory efficiency** most core operations use minimal allocations
- **Test coverage** maintained at >85% across core packages with comprehensive error handling

> Performance varies based on GitHub API rate limits, network conditions, and repository sizes.

### Profiling Documentation

📚 **Complete Performance Guide:**
- [Performance Guide](docs/performance-guide.md) - Complete benchmarking, profiling, and optimization reference

</details>

<br/>

## 🛠️ Code Standards
Read more about this Go project's [code standards](.github/CODE_STANDARDS.md).

<br/>

## 🤖 AI Compliance
This project documents expectations for AI assistants using a few dedicated files:

- [AGENTS.md](.github/AGENTS.md) — canonical rules for coding style, workflows, and pull requests used by [Codex](https://chatgpt.com/codex).
- [CLAUDE.md](.github/CLAUDE.md) — quick checklist for the [Claude](https://www.anthropic.com/product) agent.
- [.cursorrules](.cursorrules) — machine-readable subset of the policies for [Cursor](https://www.cursor.so/) and similar tools.
- [sweep.yaml](.github/sweep.yaml) — rules for [Sweep](https://github.com/sweepai/sweep), a tool for code review and pull request management.

Edit `AGENTS.md` first when adjusting these policies, and keep the other files in sync within the same pull request.

<br/>

## 👥 Maintainers
| [<img src="https://github.com/mrz1836.png" height="50" width="50" alt="MrZ" />](https://github.com/mrz1836) |
|:-----------------------------------------------------------------------------------------------------------:|
|                                      [MrZ](https://github.com/mrz1836)                                      |

<br/>

## 🤝 Contributing
View the [contributing guidelines](.github/CONTRIBUTING.md) and please follow the [code of conduct](.github/CODE_OF_CONDUCT.md).

### How can I help?
All kinds of contributions are welcome :raised_hands:!
The most basic way to show your support is to star :star2: the project, or to raise issues :speech_balloon:.
You can also support this project by [becoming a sponsor on GitHub](https://github.com/sponsors/mrz1836) :clap:
or by making a [**bitcoin donation**](https://mrz1818.com/?tab=tips&utm_source=github&utm_medium=sponsor-link&utm_campaign=go-broadcast&utm_term=go-broadcast&utm_content=go-broadcast) to ensure this journey continues indefinitely! :rocket:

[![Stars](https://img.shields.io/github/stars/mrz1836/go-broadcast?label=Please%20like%20us&style=social&v=1)](https://github.com/mrz1836/go-broadcast/stargazers)

<br/>

## 📝 License

[![License](https://img.shields.io/github/license/mrz1836/go-broadcast.svg?style=flat&v=1)](LICENSE)
