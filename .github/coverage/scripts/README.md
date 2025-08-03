# GoFortress Coverage Scripts

This directory contains utility scripts specifically for the GoFortress coverage system setup and maintenance.

## 🚀 setup-github-pages-env.sh

Automatically configures GitHub Pages environment settings to enable GoFortress coverage system deployments.

### Purpose

The GoFortress coverage system generates coverage reports, badges, and dashboards that are deployed to GitHub Pages. This script configures the necessary environment protection rules to allow deployments from the appropriate branches.

### Usage

```bash
# From repository root - use current repository
./.github/coverage/scripts/setup-github-pages-env.sh

# Specify repository explicitly
./.github/coverage/scripts/setup-github-pages-env.sh owner/repo-name
```

### What it configures

1. **GitHub Pages Environment**: Creates/configures the `github-pages` environment
2. **Deployment Branch Policies**: Sets up branch rules for:
   - `master` - Main coverage deployments from the default branch
   - `gh-pages` - GitHub Pages default branch (if used)
   - `dependabot/*` - Coverage reports for dependency update PRs
3. **Environment Protection**: Configures protection rules for secure deployments
4. **Verification**: Confirms setup and provides deployment URLs

### When to use

- **New Repository Setup**: First time enabling GoFortress coverage system
- **Deployment Errors**: Resolving "Branch not allowed to deploy to github-pages" errors
- **Environment Issues**: After encountering GitHub Pages environment protection rule failures
- **Dependabot Support**: Enabling coverage reports for automated dependency updates

### Requirements

- **GitHub CLI**: `gh` command installed and authenticated (`gh auth login`)
- **Repository Access**: Admin permissions to the target repository
- **Token Permissions**: Personal Access Token with `repo` scope for private repositories

### Troubleshooting

| Issue | Solution |
|-------|----------|
| `gh` command not found | Install GitHub CLI: https://cli.github.com/ |
| Authentication failed | Run `gh auth login` and follow prompts |
| Permission denied | Ensure you have admin access to the repository |
| Environment already exists | Script will update existing environment settings |
| Branch rules failed | May already exist - check repository Settings → Environments |

### Output Example

```
🏰 GoFortress Coverage - GitHub Pages Environment Setup
=======================================================

ℹ️  Checking GitHub CLI authentication...
✅ GitHub CLI is properly authenticated
ℹ️  Detected repository: owner/repo
ℹ️  Checking repository access: owner/repo
✅ Repository access confirmed
ℹ️  Setting up GitHub Pages environment for owner/repo...
ℹ️  Creating/updating github-pages environment...
✅ GitHub Pages environment configured
ℹ️  Configuring deployment branch rules...
ℹ️  Adding master branch deployment rule...
✅ Master branch deployment rule added
ℹ️  Adding gh-pages branch deployment rule...
✅ gh-pages branch deployment rule added
ℹ️  Adding dependabot/* branch pattern deployment rule...
✅ dependabot/* branch pattern deployment rule added
ℹ️  Verifying GitHub Pages environment configuration...
✅ GitHub Pages environment exists
✅ Found 3 deployment branch policies
ℹ️  Configured branches:
  - master
  - gh-pages
  - dependabot/*

✅ GitHub Pages environment setup completed successfully!

Next steps:
  1. Your repository is now configured to deploy to GitHub Pages from:
     - master branch (main deployments)
     - gh-pages branch (GitHub Pages default)
     - dependabot/* branches (automated dependency updates)
  2. The GoFortress coverage workflow should now deploy successfully
  3. Coverage reports will be available at: https://owner.github.io/repo/
```

### Integration with Coverage Workflow

This script configures the environment that the GoFortress coverage workflow (`.github/workflows/fortress-coverage.yml`) uses for deployment. The workflow includes:

- Coverage report generation (`dashboard.html`, `coverage.html`)
- Badge creation (`coverage.svg`)
- GitHub Pages deployment
- PR comment generation with coverage analysis

### Manual Alternative

If the script fails, you can manually configure the environment:

1. Go to repository **Settings** → **Environments** → **github-pages**
2. Under **Deployment branches**, select "Selected branches and tags"
3. Add deployment branch rules for: `master`, `gh-pages`, `dependabot/*`
4. Save changes and verify in the workflow runs

## Adding Coverage Scripts

When adding new coverage-related scripts:

1. **Focus on coverage**: Scripts should be specific to coverage system functionality
2. **Follow naming**: Use descriptive names with `.sh` extension
3. **Include documentation**: Add usage and purpose in script header
4. **Update this README**: Document the new script
5. **Test thoroughly**: Ensure scripts work across different repository configurations
6. **Handle errors**: Include comprehensive error handling and user feedbackck
