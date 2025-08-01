# 📡 go-broadcast
> Stateless File Sync Orchestrator for Repository Management

<table>
  <thead>
    <tr>
      <th>CI&nbsp;/&nbsp;CD</th>
      <th>Quality&nbsp;&amp;&nbsp;Security</th>
      <th>Docs&nbsp;&amp;&nbsp;Meta</th>
      <th>Community</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td valign="top" align="left">
        <a href="https://github.com/mrz1836/go-broadcast/releases" target="_blank">
          <img src="https://img.shields.io/github/release-pre/mrz1836/go-broadcast?logo=github&style=flat&v=2" alt="Latest Release">
        </a><br/>
        <a href="https://github.com/mrz1836/go-broadcast/actions" target="_blank">
          <img src="https://img.shields.io/github/actions/workflow/status/mrz1836/go-broadcast/fortress.yml?branch=master&logo=github&style=flat" alt="Build Status">
        </a><br/>
		<a href="https://github.com/mrz1836/go-broadcast/actions" target="_blank">
          <img src="https://github.com/mrz1836/go-broadcast/actions/workflows/codeql-analysis.yml/badge.svg?style=flat" alt="CodeQL">
        </a><br/>
		<a href="https://github.com/mrz1836/go-broadcast/actions" target="_blank">
          <img src="https://github.com/mrz1836/go-broadcast/actions/workflows/ossar.yml/badge.svg?style=flat" alt="Ossar">
        </a><br/>
        <a href="https://github.com/mrz1836/go-broadcast/commits/master" target="_blank">
		  <img src="https://img.shields.io/github/last-commit/mrz1836/go-broadcast?style=flat&logo=clockify&logoColor=white" alt="Last commit">
		</a>
      </td>
      <td valign="top" align="left">
        <a href="https://goreportcard.com/report/github.com/mrz1836/go-broadcast" target="_blank">
          <img src="https://goreportcard.com/badge/github.com/mrz1836/go-broadcast?style=flat" alt="Go Report Card">
        </a><br/>
		<a href="https://mrz1836.github.io/go-broadcast/" target="_blank">
          <img src="https://mrz1836.github.io/go-broadcast/coverage.svg" alt="Code Coverage">
        </a><br/>
		<a href="https://scorecard.dev/viewer/?uri=github.com/mrz1836/go-broadcast" target="_blank">
          <img src="https://api.scorecard.dev/projects/github.com/mrz1836/go-broadcast/badge?logo=springsecurity&logoColor=white" alt="OpenSSF Scorecard">
        </a><br/>
		<a href=".github/SECURITY.md" target="_blank">
          <img src="https://img.shields.io/badge/security-policy-blue?style=flat&logo=springsecurity&logoColor=white" alt="Security policy">
        </a><br/>
		<!--<a href="https://www.bestpractices.dev/projects/10822" target="_blank">
		  <img src="https://www.bestpractices.dev/projects/10822/badge?style=flat&logo=springsecurity&logoColor=white" alt="OpenSSF Best Practices">
		</a>-->
      </td>
      <td valign="top" align="left">
        <a href="https://golang.org/" target="_blank">
          <img src="https://img.shields.io/github/go-mod/go-version/mrz1836/go-broadcast?style=flat" alt="Go version">
        </a><br/>
        <a href="https://pkg.go.dev/github.com/mrz1836/go-broadcast?tab=doc" target="_blank">
          <img src="https://pkg.go.dev/badge/github.com/mrz1836/go-broadcast.svg?style=flat" alt="Go docs">
        </a><br/>
        <a href=".github/AGENTS.md" target="_blank">
          <img src="https://img.shields.io/badge/AGENTS.md-found-40b814?style=flat&logo=openai" alt="AGENTS.md rules">
        </a><br/>
        <a href="Makefile" target="_blank">
          <img src="https://img.shields.io/badge/Makefile-supported-brightgreen?style=flat&logo=probot&logoColor=white" alt="Makefile Supported">
        </a><br/>
		<a href=".github/dependabot.yml" target="_blank">
          <img src="https://img.shields.io/badge/dependencies-automatic-blue?logo=dependabot&style=flat" alt="Dependabot">
        </a>
      </td>
      <td valign="top" align="left">
        <a href="https://github.com/mrz1836/go-broadcast/graphs/contributors" target="_blank">
          <img src="https://img.shields.io/github/contributors/mrz1836/go-broadcast?style=flat&logo=contentful&logoColor=white" alt="Contributors">
        </a><br/>
        <a href="https://github.com/sponsors/mrz1836" target="_blank">
          <img src="https://img.shields.io/badge/sponsor-MrZ-181717.svg?logo=github&style=flat" alt="Sponsor">
        </a><br/>
        <a href="https://mrz1818.com/?tab=tips&utm_source=github&utm_medium=sponsor-link&utm_campaign=go-broadcast&utm_term=go-broadcast&utm_content=go-broadcast" target="_blank">
          <img src="https://img.shields.io/badge/donate-bitcoin-ff9900.svg?logo=bitcoin&style=flat" alt="Donate Bitcoin">
        </a>
      </td>
    </tr>
  </tbody>
</table>

<br/>

## 🗂️ Table of Contents
* [Quick Start](#-quick-start)
* [Key Features](#-key-features)
* [How It Works](#-how-it-works)
* [Usage Examples](#-usage-examples)
* [Coverage System](#-coverage-system)
* [Performance](#-performance)
* [Documentation](#-documentation)
* [Examples & Tests](#-examples--tests)
* [Code Standards](#-code-standards)
* [AI Compliance](#-ai-compliance)
* [Maintainers](#-maintainers)
* [Contributing](#-contributing)
* [License](#-license)

<br/>

## ⚡ Quick Start

Get up and running with go-broadcast in under 5 minutes!

### Prerequisites
- [Go 1.24+](https://golang.org/doc/install) ([supported release](https://golang.org/doc/devel/release.html#policy)) and [GitHub CLI](https://cli.github.com/) installed
- GitHub authentication: `gh auth login`

### Installation

```bash
go install github.com/mrz1836/go-broadcast/cmd/go-broadcast@latest
```

### Create Configuration

Create a `sync.yaml` file:

```yaml
version: 1
source:
  repo: "mrz1836/template-repo"
  branch: "master"
targets:
  - repo: "mrz1836/target-repo"
    files:
      - src: ".github/workflows/ci.yml"
        dest: ".github/workflows/ci.yml"
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
│ README.md, Makefile
└─────────────────────────────────────────────────────────────────

📄 FILE CHANGES:
   ✨ README.md (added) (+1249 bytes)
   📝 Makefile (modified) (+45 bytes)

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
│ - `Makefile` (modified)
└─────────────────────────────────────────────────────────────────

✅ DRY-RUN SUMMARY: Repository sync preview completed successfully
   📁 Repository: company/service-name
   🌿 Branch: chore/sync-files-20250130-143052-abc123f
   📝 Files: 2 would be changed
   🔗 Commit: dry-run-commit-sha
   💡 Run without --dry-run to execute these changes
```

**That's it!** 🎉 go-broadcast automatically:
- Clones your template repository
- Applies configured transformations  
- Creates a branch in each target repository
- Commits synchronized files
- Opens a pull request for review

> **💡 Pro tip:** go-broadcast includes a [built-in coverage system](#-coverage-system), [enterprise performance](#-performance), and comprehensive logging & debugging - explore the features below!

<br/>

## ✨ Key Features

**go-broadcast** is more than just file sync - it's a complete repository management platform:

### 🔄 **Intelligent Sync Engine**
- **Stateless architecture** - No databases, all state tracked via GitHub
- **Smart diff detection** - Only syncs files that actually changed
- **Zero-downtime operations** - Works at any scale without conflicts
- **Full audit trail** - Every sync tracked in branches and PRs
- **Automated PR management** - Auto-assign reviewers, assignees, and labels

### ⚡ **Enterprise Performance**
- **587M+ ops/sec** - Binary detection with zero allocations
- **239M+ ops/sec** - Content comparison for identical files  
- **13.5M+ ops/sec** - Cache operations with minimal memory
- **Concurrent sync** - Multiple repositories in parallel

### 🛡️ **Security & Compliance**
- **60+ linters** - Zero tolerance policy for code issues
- **Vulnerability scanning** - govulncheck, OSSAR, CodeQL integration
- **OpenSSF Scorecard** - Supply chain security assessment
- **Secret detection** - gitleaks integration prevents leaks

### 📊 **Built-in Coverage System**
- **Third-party replacement** - Zero external dependencies, complete data privacy
- **Professional badges** - GitHub-style badges with real-time updates
- **Interactive dashboard** - Modern UI with analytics and trends
- **[🔗 Live Dashboard](https://mrz1836.github.io/go-broadcast/)**

<br/>


## 🔍 How It Works

**go-broadcast** uses a **stateless architecture** that tracks synchronization state through GitHub itself - no databases or state files needed!

### State Tracking Through Branch Names

Every sync operation creates a branch with encoded metadata:

```
chore/sync-files-20250123-143052-abc123f
│    │         │                │
│    │         │                └─── Source commit SHA (7 chars)
│    │         └──────────────────── Timestamp (YYYYMMDD-HHMMSS)
│    └────────────────────────────── Template identifier
└─────────────────────────────────── Configurable prefix
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
   - Fetches current file from target
   - Applies transformations to source
   - Compares content byte-by-byte
   - Skips unchanged files

### Pull Request Metadata

Each PR includes structured metadata for complete traceability:

```text
<!-- go-broadcast metadata
source:
  repo: company/template-repo
  branch: master
  commit: abc123f7890
files:
  - src: .github/workflows/ci.yml
    dest: .github/workflows/ci.yml
timestamp: 2025-01-23T14:30:52Z
-->
```

### Why This Approach is Powerful

✅ **No State Files** - Everything lives in GitHub  
✅ **Atomic Operations** - Each sync is self-contained  
✅ **Full Audit Trail** - Branch and PR history shows all syncs  
✅ **Disaster Recovery** - State can be reconstructed from GitHub  
✅ **Works at Scale** - No state corruption with concurrent syncs

<br/>


## 💡 Usage Examples

### Common Use Cases

**Sync CI/CD workflows across microservices:**
```yaml
source:
  repo: "company/ci-templates"
targets:
  - repo: "company/user-service"
    files:
      - src: "workflows/ci.yml"
        dest: ".github/workflows/ci.yml"
    transform:
      variables:
        SERVICE_NAME: "user-service"
```

**Maintain documentation standards:**
```yaml
source:
  repo: "company/doc-templates"
targets:
  - repo: "company/backend-api"
    files:
      - src: "README.md"
        dest: "README.md"
    transform:
      repo_name: true
```

**Automated PR management with assignees, reviewers, and labels:**
```yaml
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

### Essential Commands

```bash
# Validate and preview
go-broadcast validate --config sync.yaml
go-broadcast sync --dry-run --config sync.yaml

# Execute sync
go-broadcast sync --config sync.yaml
go-broadcast sync org/specific-repo --config sync.yaml

# Monitor status
go-broadcast status --config sync.yaml

# Troubleshooting and diagnostics
go-broadcast diagnose                    # Collect system diagnostic information
go-broadcast diagnose > diagnostics.json # Save diagnostics to file

# Cancel active syncs
go-broadcast cancel                      # Cancel all active sync operations
go-broadcast cancel org/repo1            # Cancel syncs for specific repository
go-broadcast cancel --dry-run            # Preview what would be cancelled
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
<summary><strong>📁 File Mapping Options</strong></summary>

```yaml
files:
  - src: "Makefile"         # Copy to same location
    dest: "Makefile"
  - src: "template.md"      # Rename during sync  
    dest: "README.md"
  - src: "config/app.yml"   # Move to different directory
    dest: "configs/app.yml"
```
</details>

<details>
<summary><strong>⚙️ Advanced Configuration</strong></summary>

```yaml
version: 1
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
    files:
      - src: ".github/workflows/ci.yml"
        dest: ".github/workflows/ci.yml"
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
<summary><strong>❌ Cancel Sync Operations</strong></summary>

When issues arise, you can cancel active sync operations to prevent unwanted changes.

**Cancel sync operations when issues arise:**
```bash
# Cancel all active syncs (closes PRs and deletes branches)
go-broadcast cancel --config sync.yaml

# Cancel syncs for specific repositories only
go-broadcast cancel company/service1 company/service2

# Preview what would be cancelled without making changes
go-broadcast cancel --dry-run --config sync.yaml

# Close PRs but keep sync branches for later cleanup
go-broadcast cancel --keep-branches --config sync.yaml

# Add custom comment when closing PRs
go-broadcast cancel --comment "Cancelling due to template update" --config sync.yaml
```

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
- **Usage Examples** – Real-world scenarios in the [Usage Examples section](#-usage-examples)
- **Configuration Reference** – Comprehensive configuration options including [global PR assignment merging](#configuration-reference)
- **Configuration Examples** – Browse practical patterns in the [examples directory](examples)
- **Troubleshooting** – Solve common issues with the [troubleshooting guide](docs/troubleshooting.md)
- **API Reference** – Dive into the godocs at [pkg.go.dev/github.com/mrz1836/go-broadcast](https://pkg.go.dev/github.com/mrz1836/go-broadcast)
- **Integration Tests** – End-to-end testing examples in [test/integration](test/integration)
- **Internal Utilities** – Shared testing and validation utilities in [internal](internal) packages
- **Performance** – Check the latest numbers in the [Performance section](#-performance)

<br/>

<details>
<summary><strong>📦 Repository Features</strong></summary>
<br/>

* **Continuous Integration on Autopilot** with [GitHub Actions](https://github.com/features/actions) – every push is built, tested, and reported in minutes.
* **Pull‑Request Flow That Merges Itself** thanks to [auto‑merge](.github/workflows/auto-merge-on-approval.yml) and hands‑free [Dependabot auto‑merge](.github/workflows/dependabot-auto-merge.yml).
* **One‑Command Builds** powered by battle‑tested [Make](https://www.gnu.org/software/make) targets for linting, testing, releases, and more.
* **First‑Class Dependency Management** using native [Go Modules](https://github.com/golang/go/wiki/Modules).
* **Uniform Code Style** via [gofumpt](https://github.com/mvdan/gofumpt) plus zero‑noise linting with [golangci‑lint](https://github.com/golangci/golangci-lint).
* **Confidence‑Boosting Tests** with [testify](https://github.com/stretchr/testify), the Go [race detector](https://blog.golang.org/race-detector), crystal‑clear [HTML coverage](https://blog.golang.org/cover) snapshots, and automatic reporting via internal coverage system.
* **Hands‑Free Releases** delivered by [GoReleaser](https://github.com/goreleaser/goreleaser) whenever you create a [new Tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging).
* **Relentless Dependency & Vulnerability Scans** via [Dependabot](https://dependabot.com), [Nancy](https://github.com/sonatype-nexus-community/nancy), [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck), and [OSSAR](https://github.com/github/ossar-action).
* **Security Posture by Default** with [CodeQL](https://docs.github.com/en/github/finding-security-vulnerabilities-and-errors-in-your-code/about-code-scanning), [OpenSSF Scorecard](https://openssf.org), [OSSAR](https://github.com/github/ossar-action), and secret‑leak detection via [gitleaks](https://github.com/gitleaks/gitleaks).
* **Automatic Syndication** to [pkg.go.dev](https://pkg.go.dev/) on every release for instant godoc visibility.
* **Polished Community Experience** using rich templates for [Issues & PRs](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-go-broadcastsitory).
* **All the Right Meta Files** (`LICENSE`, `CITATION.cff`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SUPPORT.md`, `SECURITY.md`) pre‑filled and ready.
* **Code Ownership** clarified through a [CODEOWNERS](.github/CODEOWNERS) file, keeping reviews fast and focused.
* **Zero‑Noise Dev Environments** with tuned editor settings (`.editorconfig`) plus curated *ignore* files for [VS Code](.editorconfig), [Docker](.dockerignore), and [Git](.gitignore).
* **Label Sync Magic**: your repo labels stay in lock‑step with [.github/labels.yml](.github/labels.yml).
* **Friendly First PR Workflow** – newcomers get a warm welcome thanks to a dedicated [workflow](.github/workflows/pull-request-management.yml).
* **Standards‑Compliant Docs** adhering to the [standard‑readme](https://github.com/RichardLitt/standard-readme/blob/master/spec.md) spec.
* **Instant Cloud Workspaces** via [Gitpod](https://gitpod.io/) – spin up a fully configured dev environment with automatic linting and tests.
* **Out‑of‑the‑Box VS Code Happiness** with a preconfigured [Go](https://code.visualstudio.com/docs/languages/go) workspace and [`.vscode`](.vscode) folder with all the right settings.
* **Optional Release Broadcasts** to your community via [Slack](https://slack.com), [Discord](https://discord.com), or [Twitter](https://twitter.com) – plug in your webhook.
* **AI Compliance Playbook** – machine‑readable guidelines ([AGENTS.md](.github/AGENTS.md), [CLAUDE.md](.github/CLAUDE.md), [.cursorrules](.cursorrules), [sweep.yaml](.github/sweep.yaml)) keep ChatGPT, Claude, Cursor & Sweep aligned with your repo's rules.
* **Pre-commit Hooks for Consistency** powered by [pre-commit](https://pre-commit.com) and the [.pre-commit-config.yaml](.pre-commit-config.yaml) file—run the same formatting, linting, and tests before every commit, just like CI.
* **Automated Hook Updates** keep the [.pre-commit-config.yaml](.pre-commit-config.yaml) current via a weekly [workflow](.github/workflows/update-pre-commit-hooks.yml).
* **DevContainers for Instant Onboarding** – Launch a ready-to-code environment in seconds with [VS Code DevContainers](https://containers.dev/) and the included [.devcontainer.json](.devcontainer.json) config.

</details>

<details>
<summary><strong>🚀 Library Deployment</strong></summary>
<br/>

This project uses [goreleaser](https://github.com/goreleaser/goreleaser) for streamlined binary and library deployment to GitHub. To get started, install it via:

```bash
brew install goreleaser
```

The release process is defined in the [.goreleaser.yml](.goreleaser.yml) configuration file.

To generate a snapshot (non-versioned) release for testing purposes, run:

```bash
make release-snap
```

Before tagging a new version, update the release metadata (version) in the `CITATION.cff` file:

```bash
make citation version=0.2.1
```

Then create and push a new Git tag using:

```bash
make tag version=x.y.z
```

This process ensures consistent, repeatable releases with properly versioned artifacts and citation metadata.

</details>

<details>
<summary><strong>🔨 Makefile Commands</strong></summary>
<br/>

View all `makefile` commands

```bash script
make help
```

List of all current commands:

<!-- make-help-start -->
```text
bench                 ## Run all benchmarks in the Go application
bench-compare         ## Run benchmarks and save results for comparison
bench-cpu             ## Run benchmarks with CPU profiling
bench-full            ## Run comprehensive benchmarks with multiple iterations
bench-save            ## Save current benchmark results as baseline
build-go              ## Build the Go application (locally)
citation              ## Update version in CITATION.cff (use version=X.Y.Z)
clean-mods            ## Remove all the Go mod cache
coverage              ## Show test coverage
diff                  ## Show git diff and fail if uncommitted changes exist
fumpt                 ## Run fumpt to format Go code
generate              ## Run go generate in the base of the repo
godocs                ## Trigger GoDocs tag sync
govulncheck-install   ## Install govulncheck (pass VERSION= to override)
govulncheck           ## Scan for vulnerabilities
help                  ## Display this help message
install-go            ## Install using go install with specific version
install-releaser      ## Install GoReleaser
install-stdlib        ## Install the Go standard library for the host platform
install               ## Install the application binary
lint-version          ## Show the golangci-lint version
lint                  ## Run the golangci-lint application (install if not found)
loc                   ## Total lines of code table
mod-download          ## Download Go module dependencies
mod-tidy              ## Clean up go.mod and go.sum
pre-build             ## Pre-build all packages to warm cache
release-snap          ## Build snapshot binaries
release-test          ## Run release dry-run (no publish)
release               ## Run production release (requires github_token)
tag-remove            ## Remove local and remote tag (use version=X.Y.Z)
tag-update            ## Force-update tag to current commit (use version=X.Y.Z)
tag                   ## Create and push a new tag (use version=X.Y.Z)
test-ci-no-race       ## CI test suite without race detector
test-ci               ## CI test runs tests with race detection and coverage (no lint - handled separately)
test-cover-race       ## Runs unit tests with race detector and outputs coverage
test-cover            ## Unit tests with coverage (no race)
test-fuzz             ## Run fuzz tests only (no unit tests)
test-no-lint          ## Run only tests (no lint)
test-parallel         ## Run tests in parallel (faster for large repos)
test-race             ## Unit tests with race detector (no coverage)
test-short            ## Run tests excluding integration tests (no lint)
test                  ## Default testing uses lint + unit tests (fast)
uninstall             ## Uninstall the Go binary
update-linter         ## Upgrade golangci-lint (macOS only)
update-releaser       ## Reinstall GoReleaser
update                ## Update dependencies
vet-parallel          ## Run go vet in parallel (faster for large repos)
vet                   ## Run go vet only on your module packages
```
<!-- make-help-end -->

</details>

<details>
<summary><strong>⚡ GitHub Workflows</strong></summary>
<br/>


### 🎛️ The Workflow Control Center

All GitHub Actions workflows in this repository are powered by a single configuration file: [**.env.shared**](.github/.env.shared) – your one-stop shop for tweaking CI/CD behavior without touching a single YAML file! 🎯

This magical file controls everything from:
- **⚙️ Go version matrix** (test on multiple versions or just one)
- **🏃 Runner selection** (Ubuntu or macOS, your wallet decides)
- **🔬 Feature toggles** (coverage, fuzzing, linting, race detection, benchmarks)
- **🛡️ Security tool versions** (gitleaks, nancy, govulncheck)
- **🤖 Auto-merge behaviors** (how aggressive should the bots be?)
- **🏷️ PR management rules** (size labels, auto-assignment, welcome messages)

> **Pro tip:** Want to disable code coverage? Just flip `ENABLE_CODE_COVERAGE=false` in [.env.shared](.github/.env.shared) and push. No YAML archaeology required! 

<br/>

| Workflow Name                                                                      | Description                                                                                                            |
|------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------|
| [auto-merge-on-approval.yml](.github/workflows/auto-merge-on-approval.yml)         | Automatically merges PRs after approval and all required checks, following strict rules.                               |
| [codeql-analysis.yml](.github/workflows/codeql-analysis.yml)                       | Analyzes code for security vulnerabilities using [GitHub CodeQL](https://codeql.github.com/).                          |
| [dependabot-auto-merge.yml](.github/workflows/dependabot-auto-merge.yml)           | Automatically merges [Dependabot](https://github.com/dependabot) PRs that meet all requirements.                       |
| [fortress.yml](.github/workflows/fortress.yml)                                     | Runs the GoFortress security and testing workflow, including linting, testing, releasing, and vulnerability checks.    |
| [ossar.yml](.github/workflows/ossar.yml)                                           | Runs [OSSAR](https://github.com/github/ossar-action) static analysis workflow                                          |
| [pull-request-management.yml](.github/workflows/pull-request-management.yml)       | Labels PRs by branch prefix, assigns a default user if none is assigned, and welcomes new contributors with a comment. |
| [scorecard.yml](.github/workflows/scorecard.yml)                                   | Runs [OpenSSF](https://openssf.org/) Scorecard to assess supply chain security.                                        |
| [stale.yml](.github/workflows/stale-check.yml)                                     | Warns about (and optionally closes) inactive issues and PRs on a schedule or manual trigger.                           |
| [sync-labels.yml](.github/workflows/sync-labels.yml)                               | Keeps GitHub labels in sync with the declarative manifest at [`.github/labels.yml`](./.github/labels.yml).             |
| [update-python-dependencies.yml](.github/workflows/update-python-dependencies.yml) | Updates Python dependencies for pre-commit hooks in the repository.                                                    |
| [update-pre-commit-hooks.yml](.github/workflows/update-pre-commit-hooks.yml)       | Automatically update versions for [pre-commit](https://pre-commit.com/) hooks                                          |

</details>

<details>
<summary><strong>📦 Updating Dependencies</strong></summary>
<br/>

To update all dependencies (Go modules, linters, and related tools), run:

```bash
make update
```

This command ensures all dependencies are brought up to date in a single step, including Go modules and any tools managed by the Makefile. It is the recommended way to keep your development environment and CI in sync with the latest versions.

</details>

<details>
<summary><strong>🔧 Pre-commit Hooks</strong></summary>
<br/>

Set up the optional [pre-commit](https://pre-commit.com) hooks to run the same formatting, linting, and tests defined in [AGENTS.md](.github/AGENTS.md) before every commit:

```bash
pip install pre-commit
pre-commit install
```

The hooks are configured in [.pre-commit-config.yaml](.pre-commit-config.yaml) and mirror the CI pipeline.

</details>

<details>
<summary><strong>🐛 Logging and Debugging</strong></summary>

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
# - go-broadcast version and build information
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

For more detailed information, see the [comprehensive logging guide](docs/logging.md) and [troubleshooting runbook](docs/troubleshooting-runbook.md).

</details>

<details>
<summary><strong>📊 Coverage System</strong></summary>

## 🚀 Coverage System

**Self-hosted coverage solution** - Complete data privacy, zero external dependencies, and enterprise-grade features.

<table>
  <tr>
    <td><a href="https://mrz1836.github.io/go-broadcast/" target="_blank"><img src="https://mrz1836.github.io/go-broadcast/coverage.svg" alt="Main Branch Coverage" /></a></td>
    <td><a href="https://mrz1836.github.io/go-broadcast/" target="_blank"><img src="https://img.shields.io/badge/coverage-87.2%25-brightgreen?style=flat-square" alt="Flat Square Style" /></a></td>
    <td><a href="https://mrz1836.github.io/go-broadcast/" target="_blank"><img src="https://img.shields.io/badge/trend-%E2%86%97%20improving-green?style=for-the-badge" alt="Trend Badge" /></a></td>
  </tr>
</table>

🔗 **[View Dashboard](https://mrz1836.github.io/go-broadcast/?v=1)**


<details>
<summary><strong>📊 Quick Setup</strong></summary>

### ⚡ Quick Setup

Enable in 2 steps:

```bash
# 1. Enable in .github/.env.shared
ENABLE_INTERNAL_COVERAGE=true
COVERAGE_FAIL_UNDER=80

# 2. Set GitHub Pages source to "GitHub Actions"
# Repository Settings → Pages → Source → "GitHub Actions"
```

That's it! Push any commit and get:
- ✅ Professional coverage badges
- ✅ Interactive dashboard
- ✅ PR comments with analysis
- ✅ GitHub Pages deployment

</details>

<details>
<summary><strong>🎯 Complete Feature List & Advanced Configuration</strong></summary>

### Core Features

#### Professional Coverage Badges
- **GitHub-style badges** with multiple themes (flat, flat-square, for-the-badge)
- **Real-time updates** on every push and pull request
- **Branch-specific badges** for `master` and PR branches
- **PR-specific badges** for pull request analysis

#### Interactive Coverage Dashboard
- **Modern, responsive UI** with dark/light theme support
- **Real-time metrics** with animated progress indicators
- **Historical trend** showing trend from last push
- **Responsive design** that works on desktop and mobile
- **Zero external dependencies** - fully self-contained

#### Intelligent PR Coverage Comments
- **Coverage analysis** comparing base vs PR branches
- **File-level breakdown** showing coverage changes
- **Smart anti-spam logic** to prevent comment noise on multiple pushes
- **Comprehensive PR comments** with detailed coverage analysis

#### Analytics & Insights
- **Google Analytics integration** for detailed usage tracking
- **Historical trend tracking** with basic trend analysis
- **Coverage history** stored in JSON format
- **Retention policies** for automatic data cleanup

#### GitHub Pages Deployment
- **Automatic GitHub Pages integration** with organized storage
- **PR-specific deployments** with isolated coverage reports
- **Automatic cleanup** of expired PR data
- **Simple CLI** with 3 core commands (complete, comment, history)

### Advanced Configuration

The coverage system includes 45+ configuration options for complete customization:

#### 🎨 Badge & Theme Configuration
```bash
COVERAGE_BADGE_STYLE=flat                # flat, flat-square, for-the-badge
COVERAGE_BADGE_LOGO=                     # Logo: go, github, custom URL (empty for no logo)
COVERAGE_REPORT_THEME=github-dark        # Dashboard theme
COVERAGE_THRESHOLD_EXCELLENT=90          # Green badge threshold
COVERAGE_THRESHOLD_GOOD=80               # Yellow-green threshold
```

#### 📊 Analytics & Reporting
```bash
COVERAGE_ENABLE_TREND_ANALYSIS=true      # Historical trend tracking
COVERAGE_ENABLE_PACKAGE_BREAKDOWN=true   # Package-level coverage
COVERAGE_HISTORY_RETENTION_DAYS=90       # Data retention period
COVERAGE_CLEANUP_PR_AFTER_DAYS=7         # PR cleanup schedule
```

#### 🔔 PR Comment Configuration
```bash
COVERAGE_PR_COMMENT_ENABLED=true         # Enable PR comments
COVERAGE_PR_COMMENT_SHOW_TREE=true       # Show file tree in PR comments
COVERAGE_PR_COMMENT_SHOW_MISSING=true    # Highlight uncovered lines
COVERAGE_PR_COMMENT_BEHAVIOR=update      # Comment behavior: new, update, delete-and-new
```

### GitHub Pages URLs

#### Main Branch Coverage
- **Coverage Badge**: `https://mrz1836.github.io/go-broadcast/coverage.svg`
- **Coverage Dashboard**: `https://mrz1836.github.io/go-broadcast/`
- **Coverage Report**: `https://mrz1836.github.io/go-broadcast/coverage.html`

#### Branch-Specific Coverage
- **Branch Badge**: `https://mrz1836.github.io/go-broadcast/coverage/branch/{branch-name}/coverage.svg`
- **Branch Dashboard**: `https://mrz1836.github.io/go-broadcast/coverage/branch/{branch-name}/`
- **Branch Report**: `https://mrz1836.github.io/go-broadcast/coverage/branch/{branch-name}/coverage.html`

#### Pull Request Coverage
- **PR Badge**: `https://mrz1836.github.io/go-broadcast/coverage/pr/{pr-number}/coverage.svg`
- **PR Coverage Report**: `https://mrz1836.github.io/go-broadcast/coverage/pr/{pr-number}/`
- **All Branches Index**: `https://mrz1836.github.io/go-broadcast/branches.html` (when deployed from main)

📚 **[Complete Configuration Guide](.github/coverage/docs/coverage-configuration.md)** | 📊 **[API Documentation](.github/coverage/docs/coverage-api.md)** | 🎯 **[Feature Guide](.github/coverage/docs/coverage-features.md)**

</details>

</details>



<br/>

## 🧪 Examples & Tests

All unit tests and [examples](examples) run via [GitHub Actions](https://github.com/mrz1836/go-broadcast/actions) and use [Go version 1.24.x](https://go.dev/doc/go1.24). View the [configuration file](.github/workflows/fortress.yml).

Run all tests (fast):

```bash script
make test
```

Run all tests with race detector (slower):
```bash script
make test-race
```

<br/>

## ⚡ Performance

**Enterprise-grade performance** - Designed for high-scale repository management with zero-allocation critical paths.

### Performance Highlights

| Operation              | Performance    | Memory           |
|------------------------|----------------|------------------|
| **Binary Detection**   | 587M+ ops/sec  | Zero allocations |
| **Content Comparison** | 239M+ ops/sec  | Zero allocations |
| **Cache Operations**   | 13.5M+ ops/sec | Minimal memory   |
| **Batch Processing**   | 23.8M+ ops/sec | Concurrent safe  |

### Quick Benchmarks

```bash
# Run all benchmarks
make bench

# Benchmark specific components
go test -bench=. -benchmem ./internal/git
go test -bench=. -benchmem ./internal/worker

# Try the profiling demo
go run ./cmd/profile_demo
```

<details>
<summary><strong>📊 Complete Benchmark Results & Profiling Tools</strong></summary>

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

📚 **Complete Guides:**
- [Benchmarking Guide](docs/benchmarking-profiling.md) - Complete benchmarking reference
- [Profiling Guide](docs/profiling-guide.md) - Advanced profiling techniques
- [Performance Optimization](docs/performance-optimization.md) - Best practices and tips

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
