# Migrating from Codecov to GoFortress Coverage

Complete step-by-step guide for migrating from Codecov (or other external coverage services) to the self-hosted GoFortress Internal Coverage System.

## Migration Overview

### Why Migrate?

The GoFortress Internal Coverage System provides significant advantages over external services:

| Aspect | Codecov | GoFortress Coverage |
|--------|---------|-------------------|
| **Privacy** | Data sent to third-party | Complete data privacy |
| **Cost** | $29-$300+/month | $0 (one-time setup) |
| **Performance** | API-dependent | Sub-2s badge generation |
| **Features** | Limited by plan | All features included |
| **Control** | External dependency | Full control |
| **Customization** | Limited themes | Unlimited customization |
| **Availability** | 99.9% (external) | 99.9%+ (GitHub Pages) |

### Migration Timeline

- **Planning**: 30 minutes
- **Implementation**: 2-4 hours
- **Testing**: 1-2 hours
- **Production Deployment**: 30 minutes
- **Team Training**: 1 hour

## Pre-Migration Checklist

### ✅ Prerequisites

Before starting the migration, ensure you have:

- [ ] **Repository Admin Access**: Required for GitHub Pages and workflow changes
- [ ] **Existing Coverage Setup**: Current test suite with coverage generation
- [ ] **Codecov Configuration**: Access to current `codecov.yml` for reference
- [ ] **Badge URLs**: List of current badge locations to update
- [ ] **Backup Plan**: Document current setup for potential rollback

### ✅ Environment Requirements

- [ ] **Go 1.19+**: Required for building the coverage tool
- [ ] **GitHub Actions**: Existing CI/CD pipeline
- [ ] **GitHub Pages**: Will be auto-enabled during setup
- [ ] **Repository Permissions**: `contents: write` and `pages: write`

### ✅ Data Gathering

Collect information about your current setup:

```bash
# Document current badge URLs
grep -r "codecov.io" README.md docs/ --include="*.md"

# Review current codecov.yml configuration
cat codecov.yml

# Check current workflow integration
grep -r "codecov" .github/workflows/ --include="*.yml"
```

## Step-by-Step Migration Process

### Step 1: Add GoFortress Coverage System

#### 1.1 Add Environment Variables

Add coverage configuration to `.github/.env.shared`:

```bash
# GoFortress Coverage System Configuration
ENABLE_INTERNAL_COVERAGE=true

# Basic coverage thresholds
COVERAGE_FAIL_UNDER=80
COVERAGE_THRESHOLD_EXCELLENT=90
COVERAGE_THRESHOLD_GOOD=80
COVERAGE_THRESHOLD_ACCEPTABLE=70
COVERAGE_THRESHOLD_LOW=60

# Badge configuration
COVERAGE_BADGE_STYLE=flat
COVERAGE_BADGE_LOGO=go
COVERAGE_BADGE_BRANCHES=main,develop

# GitHub Pages and PR integration
COVERAGE_PAGES_AUTO_CREATE=true
COVERAGE_PR_COMMENT_ENABLED=true
COVERAGE_PR_COMMENT_BEHAVIOR=update

# Analytics and history
COVERAGE_ENABLE_TREND_ANALYSIS=true
COVERAGE_ENABLE_PACKAGE_BREAKDOWN=true
COVERAGE_HISTORY_RETENTION_DAYS=90

# Cleanup and maintenance
COVERAGE_CLEANUP_PR_AFTER_DAYS=7
```

#### 1.2 Set Up Coverage Tool Structure

```bash
# Create coverage system directory
mkdir -p .github/coverage

# Copy the GoFortress coverage system
cp -r /path/to/gofortress-coverage/* .github/coverage/

# Verify the structure
ls -la .github/coverage/
```

Expected directory structure:
```
.github/coverage/
├── cmd/gofortress-coverage/
├── internal/
│   ├── analytics/
│   ├── badge/
│   ├── config/
│   ├── github/
│   ├── history/
│   ├── notify/
│   ├── pages/
│   ├── parser/
│   └── report/
├── go.mod
├── go.sum
└── README.md
```

#### 1.3 Build and Validate Coverage Tool

```bash
# Build the coverage tool
cd .github/coverage
go build -o gofortress-coverage ./cmd/gofortress-coverage/

# Validate it works
./gofortress-coverage --version
./gofortress-coverage --help

# Test with existing coverage file (if available)
./gofortress-coverage parse --file ../../coverage.out --format table
```

### Step 2: Update Workflows

#### 2.1 Add Coverage Processing Workflow

Create `.github/workflows/fortress-coverage.yml`:

```yaml
name: Fortress Coverage Processing

on:
  workflow_call:
    inputs:
      coverage-file:
        description: 'Path to coverage file'
        required: true
        type: string
      branch-name:
        description: 'Git branch name'
        required: true
        type: string
      commit-sha:
        description: 'Git commit SHA'
        required: true
        type: string
      pr-number:
        description: 'Pull request number'
        required: false
        type: string

jobs:
  coverage:
    runs-on: ubuntu-latest
    if: env.ENABLE_INTERNAL_COVERAGE == 'true'
    
    permissions:
      contents: write
      pages: write
      pull-requests: write
      checks: write
      
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
          
      - name: Download coverage artifact
        uses: actions/download-artifact@v3
        with:
          name: coverage-report
          
      - name: Setup GitHub Pages
        uses: actions/configure-pages@v3
        
      - name: Build Coverage Tool
        run: |
          cd .github/coverage
          go build -o gofortress-coverage ./cmd/gofortress-coverage/
          
      - name: Process Coverage
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          cd .github/coverage
          ./gofortress-coverage complete \
            --input ../../${{ inputs.coverage-file }} \
            --branch ${{ inputs.branch-name }} \
            --commit ${{ inputs.commit-sha }} \
            ${{ inputs.pr-number && format('--pr {0}', inputs.pr-number) || '' }} \
            --verbose
            
      - name: Upload Pages artifact
        uses: actions/upload-pages-artifact@v2
        with:
          path: coverage-reports
          
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v2
```

#### 2.2 Update Test Workflow

Modify your existing test workflow (e.g., `.github/workflows/test.yml`):

```yaml
# Add coverage artifact upload after your test step
- name: Upload Coverage Artifact
  if: env.ENABLE_INTERNAL_COVERAGE == 'true'
  uses: actions/upload-artifact@v3
  with:
    name: coverage-report
    path: coverage.out
    retention-days: 1

# Add coverage processing call
- name: Process Coverage
  if: env.ENABLE_INTERNAL_COVERAGE == 'true'
  uses: ./.github/workflows/fortress-coverage.yml
  with:
    coverage-file: coverage.out
    branch-name: ${{ github.ref_name }}
    commit-sha: ${{ github.sha }}
    pr-number: ${{ github.event.number }}
```

### Step 3: Configuration Mapping

#### 3.1 Codecov Configuration Translation

Map your existing `codecov.yml` configuration to GoFortress environment variables:

| Codecov Setting | GoFortress Variable | Example |
|-----------------|-------------------|---------|
| `coverage.status.project.default.target` | `COVERAGE_FAIL_UNDER` | `80` |
| `coverage.status.project.default.threshold` | `COVERAGE_THRESHOLD_GOOD` | `80` |
| `ignore` paths | `COVERAGE_EXCLUDE_PATHS` | `vendor/,test/` |
| `fixes` paths | `COVERAGE_INCLUDE_ONLY_PATHS` | `internal/,pkg/` |
| `comment.behavior` | `COVERAGE_PR_COMMENT_BEHAVIOR` | `update` |
| `comment.layout` | Template selection | `comprehensive` |

#### 3.2 Common Codecov Configurations

**Basic Codecov setup:**
```yaml
# codecov.yml
coverage:
  status:
    project:
      default:
        target: 80%
        threshold: 1%
```

**Equivalent GoFortress setup:**
```bash
COVERAGE_FAIL_UNDER=80
COVERAGE_ENFORCE_THRESHOLD=true
```

**Advanced Codecov setup:**
```yaml
# codecov.yml
coverage:
  ignore:
    - "vendor/"
    - "test/"
  status:
    project:
      default:
        target: 85%
    patch:
      default:
        target: 90%
comment:
  behavior: update
  layout: "diff, files"
```

**Equivalent GoFortress setup:**
```bash
COVERAGE_FAIL_UNDER=85
COVERAGE_EXCLUDE_PATHS=vendor/,test/
COVERAGE_PR_COMMENT_ENABLED=true
COVERAGE_PR_COMMENT_BEHAVIOR=update
COVERAGE_PR_COMMENT_SHOW_TREE=true
COVERAGE_PR_COMMENT_SHOW_MISSING=true
```

### Step 4: Update Badge URLs

#### 4.1 Identify Current Badge URLs

Find all Codecov badge references:

```bash
# Search for Codecov badges
grep -r "codecov.io" . --include="*.md" --include="*.rst"

# Common Codecov badge patterns
grep -r "https://codecov.io/gh/" . --include="*.md"
grep -r "https://img.shields.io/codecov/" . --include="*.md"
```

#### 4.2 Replace with GoFortress Badge URLs

**Old Codecov badge:**
```markdown
![Coverage](https://codecov.io/gh/organization/repository/branch/main/graph/badge.svg)
```

**New GoFortress badge:**
```markdown
![Coverage](https://organization.github.io/repository/badges/main.svg)
```

#### 4.3 Badge URL Patterns

| Branch | Old Codecov URL | New GoFortress URL |
|--------|-----------------|-------------------|
| Main | `codecov.io/gh/org/repo/branch/main/graph/badge.svg` | `org.github.io/repo/badges/main.svg` |
| Develop | `codecov.io/gh/org/repo/branch/develop/graph/badge.svg` | `org.github.io/repo/badges/develop.svg` |
| PR | Not available | `org.github.io/repo/badges/pr/123.svg` |

#### 4.4 Update All References

Create a script to update badge URLs:

```bash
#!/bin/bash
# update-badges.sh

ORG="your-org"
REPO="your-repo"

# Replace in README.md
sed -i.bak "s|https://codecov.io/gh/$ORG/$REPO/branch/main/graph/badge.svg|https://$ORG.github.io/$REPO/badges/main.svg|g" README.md

# Replace in other markdown files
find . -name "*.md" -exec sed -i.bak "s|https://codecov.io/gh/$ORG/$REPO/branch/main/graph/badge.svg|https://$ORG.github.io/$REPO/badges/main.svg|g" {} \;

echo "Badge URLs updated. Check .bak files for originals."
```

### Step 5: Remove Codecov Integration

#### 5.1 Remove Codecov from Workflows

```bash
# Remove Codecov upload steps
sed -i.bak '/codecov/d' .github/workflows/*.yml

# Remove Codecov action usage
sed -i.bak '/codecov\/codecov-action/,+3d' .github/workflows/*.yml
```

#### 5.2 Remove Configuration Files

```bash
# Backup and remove codecov.yml
mv codecov.yml codecov.yml.bak

# Remove from .gitignore if present
sed -i.bak '/codecov/d' .gitignore
```

#### 5.3 Remove Dependencies

```bash
# Remove from package.json (if using codecov npm package)
npm uninstall codecov

# Remove from requirements.txt (if using codecov Python package)
sed -i.bak '/codecov/d' requirements.txt
```

### Step 6: Test Migration

#### 6.1 Local Testing

```bash
# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Test coverage processing locally
cd .github/coverage
./gofortress-coverage complete --input ../../coverage.out --verbose

# Verify badge generation
./gofortress-coverage badge --coverage 87.2 --output test-badge.svg
```

#### 6.2 Push and Validate

```bash
# Commit changes
git add .
git commit -m "Migrate from Codecov to GoFortress Coverage System

- Add GoFortress coverage configuration
- Update workflows for internal coverage processing
- Replace Codecov badge URLs with GitHub Pages URLs
- Remove Codecov dependencies and configuration"

# Push and monitor workflow
git push origin feature/migrate-to-gofortress-coverage
```

#### 6.3 Verify Deployment

After the workflow runs:

1. **Check GitHub Pages**: Visit `https://your-org.github.io/your-repo/`
2. **Verify Badges**: Ensure badge URLs work and display correctly
3. **Test PR Comments**: Create a test PR and verify coverage comments
4. **Validate Analytics**: Check dashboard and trend analysis

### Step 7: Production Deployment

#### 7.1 Merge to Main Branch

```bash
# Create pull request
gh pr create --title "Migrate to GoFortress Coverage System" \
  --body "Replaces Codecov with self-hosted coverage system"

# After review and approval, merge
gh pr merge --squash
```

#### 7.2 Monitor Initial Deployment

```bash
# Watch the deployment
gh run watch

# Check GitHub Pages deployment
curl -I https://your-org.github.io/your-repo/badges/main.svg

# Verify API endpoints
curl https://your-org.github.io/your-repo/api/coverage.json
```

#### 7.3 Team Communication

Send announcement to team:

```markdown
## 📊 Coverage System Migration Complete!

We've successfully migrated from Codecov to our new self-hosted GoFortress Coverage System.

### What's New:
- ✅ Complete data privacy (no external services)
- ✅ Faster badge generation (<2 seconds)
- ✅ Enhanced PR comments with trend analysis
- ✅ Interactive coverage dashboard
- ✅ Zero ongoing costs

### What's Changed:
- Badge URLs updated (automatically in README)
- Coverage reports now at: https://your-org.github.io/your-repo/
- Enhanced PR coverage comments with more insights

### Dashboard:
Visit https://your-org.github.io/your-repo/ for the new interactive dashboard!
```

## Feature Comparison & Migration

### Feature Mapping

| Codecov Feature | GoFortress Equivalent | Status |
|-----------------|---------------------|---------|
| Coverage badges | Professional SVG badges with themes | ✅ Enhanced |
| PR comments | Intelligent PR comments (5 templates) | ✅ Enhanced |
| Coverage reports | Interactive HTML reports | ✅ Enhanced |
| Historical data | Trend analysis with predictions | ✅ Enhanced |
| Team analytics | Comprehensive team insights | ✅ New |
| API access | JSON API endpoints | ✅ Enhanced |
| Notifications | Multi-channel notifications | ✅ New |
| Custom exclusions | Advanced exclusion system | ✅ Enhanced |

### Enhanced Features

**New capabilities not available in Codecov:**

1. **Predictive Analytics**: Machine learning-powered coverage predictions
2. **Team Collaboration Metrics**: Developer impact analysis
3. **Real-time Dashboard**: Interactive coverage visualization
4. **PR Impact Analysis**: Risk assessment for pull requests
5. **Multi-channel Notifications**: Slack, Teams, Discord, Email
6. **Advanced Exclusions**: 7 types of file/folder exclusions
7. **Quality Scoring**: Comprehensive code quality assessment

### Migration Gotchas

#### Common Issues and Solutions

1. **Badge URLs not updating immediately**
   ```bash
   # Solution: Clear browser cache or use curl to test
   curl -I https://your-org.github.io/your-repo/badges/main.svg
   ```

2. **GitHub Pages not enabled**
   ```bash
   # Solution: Enable in repository settings or use CLI
   gh api repos/:owner/:repo --method PATCH \
     --field pages[source][branch]=gh-pages
   ```

3. **Coverage thresholds too strict**
   ```bash
   # Solution: Adjust thresholds gradually
   COVERAGE_FAIL_UNDER=70  # Start lower, increase over time
   ```

4. **Large repository performance**
   ```bash
   # Solution: Optimize exclusions
   COVERAGE_EXCLUDE_PATHS=vendor/,test/,docs/,scripts/
   COVERAGE_MIN_FILE_LINES=10
   ```

## Rollback Plan

If you need to revert to Codecov:

### Step 1: Disable GoFortress Coverage

```bash
# Temporarily disable internal coverage
echo "ENABLE_INTERNAL_COVERAGE=false" >> .github/.env.shared
```

### Step 2: Restore Codecov Configuration

```bash
# Restore codecov.yml
mv codecov.yml.bak codecov.yml

# Restore workflow files
git checkout HEAD~1 -- .github/workflows/
```

### Step 3: Update Badge URLs Back

```bash
# Restore original badge URLs
mv README.md.bak README.md
```

### Step 4: Re-add Codecov Action

```yaml
# Add back to workflow
- name: Upload to Codecov
  uses: codecov/codecov-action@v3
  with:
    file: coverage.out
```

## Post-Migration Validation

### ✅ Verification Checklist

After migration, verify:

- [ ] **Badges Work**: All badge URLs display correctly
- [ ] **Dashboard Accessible**: `https://your-org.github.io/your-repo/` loads
- [ ] **PR Comments**: Test PR shows coverage comment
- [ ] **Workflow Success**: CI/CD pipeline completes without errors
- [ ] **API Endpoints**: JSON endpoints return valid data
- [ ] **Team Access**: All team members can access reports
- [ ] **Performance**: Badge generation is fast (<5 seconds)

### Monitoring Setup

Set up monitoring for the new system:

```bash
# Add health check to workflow
- name: Health Check
  run: |
    curl -f https://your-org.github.io/your-repo/api/health.json
```

### Success Metrics

Track these metrics post-migration:

- **Badge Response Time**: <2 seconds (vs Codecov's 5-10 seconds)
- **Coverage Processing**: <30 seconds end-to-end
- **Team Adoption**: Usage of new dashboard features
- **Cost Savings**: $29-300/month saved from Codecov subscription

---

## Advanced Migration Scenarios

### Multi-Repository Migration

For organizations with multiple repositories:

#### 1. Template Repository Approach

```bash
# Create template with GoFortress coverage
gh repo create coverage-template --template
cd coverage-template

# Set up complete GoFortress system
# ... (follow standard migration steps)

# Use as template for other repos
gh repo create new-repo --template coverage-template
```

#### 2. Automated Migration Script

```bash
#!/bin/bash
# migrate-multiple-repos.sh

REPOS=("repo1" "repo2" "repo3")
ORG="your-org"

for repo in "${REPOS[@]}"; do
  echo "Migrating $repo..."
  
  # Clone repository
  gh repo clone "$ORG/$repo"
  cd "$repo"
  
  # Copy coverage system
  cp -r ../coverage-template/.github/coverage .github/
  cp ../coverage-template/.github/.env.shared .github/
  
  # Update workflows
  cp ../coverage-template/.github/workflows/fortress-coverage.yml .github/workflows/
  
  # Update badge URLs
  sed -i "s/codecov\.io\/gh\/$ORG\/\w\+/codecov.io\/gh\/$ORG\/$repo/g" README.md
  sed -i "s/codecov\.io\/gh\/$ORG\/$repo/$ORG.github.io\/$repo\/badges\/main.svg/g" README.md
  
  # Commit and push
  git add .
  git commit -m "Migrate to GoFortress Coverage System"
  git push
  
  cd ..
done
```

### Legacy System Integration

For repositories with complex coverage setups:

#### 1. Gradual Migration

```bash
# Phase 1: Run both systems in parallel
ENABLE_INTERNAL_COVERAGE=true
CODECOV_UPLOAD=true  # Keep uploading to Codecov temporarily

# Phase 2: Switch badge URLs
# Update badges to point to GoFortress

# Phase 3: Disable Codecov
CODECOV_UPLOAD=false
```

#### 2. Historical Data Import

```bash
# Export Codecov historical data (if available via API)
# Import into GoFortress history system
gofortress-coverage history import --file codecov-history.json
```

---

## Support & Troubleshooting

### Getting Help

- **Documentation**: Complete guides in `/docs/` directory
- **CLI Help**: `gofortress-coverage --help` for command-specific help
- **Debug Mode**: Use `--debug` flag for detailed troubleshooting
- **GitHub Issues**: Report migration issues and get support

### Migration Support

For migration assistance:

1. **Pre-migration Review**: Share your current setup for customized guidance
2. **Live Migration Support**: Schedule migration call for complex repositories
3. **Post-migration Validation**: Verify setup and optimize configuration

### Community Resources

- **Migration Examples**: Real-world migration examples and case studies
- **Best Practices**: Community-contributed optimization tips
- **Feature Requests**: Suggest improvements based on migration experience

---

## Related Documentation

- [📖 System Overview](coverage-system.md) - Architecture and components
- [🎯 Feature Showcase](coverage-features.md) - Explore all available features
- [⚙️ Configuration Guide](coverage-configuration.md) - Complete configuration reference
- [🛠️ API Documentation](coverage-api.md) - CLI commands and automation