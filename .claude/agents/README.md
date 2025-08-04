# 🚀 go-broadcast Sub-Agent Team

This directory contains 26 specialized sub-agents designed to manage all aspects of the go-broadcast repository lifecycle. Each agent follows the single-responsibility principle and is optimized for specific tasks within the Go ecosystem.

> **📚 Complete Documentation**: For comprehensive information about all agents, collaboration patterns, usage examples, and performance metrics, see [**docs/sub-agents.md**](../../docs/sub-agents.md)

## 📋 Agent Categories

### 🔧 Core go-broadcast Operations (4 agents)
- **sync-orchestrator** - Manages sync operations, validates configurations, coordinates workflows
- **config-validator** - Validates YAML configurations, checks repository access, validates transformations
- **github-sync-api** - Optimizes GitHub API usage, manages rate limits, improves performance
- **directory-sync-specialist** - Handles complex directory synchronization with performance optimization

### 🧪 Testing & Quality Assurance (5 agents)
- **test-commander** - Runs test suites with race detection, maintains >85% coverage
- **benchmark-runner** - Executes benchmarks, tracks performance regression
- **fuzz-test-guardian** - Manages fuzz testing and corpus generation
- **integration-test-manager** - Handles phased integration testing
- **go-quality-enforcer** - Enforces 60+ linters and Go conventions

### 🔄 Dependency & Upgrade Management (3 agents)
- **dependabot-coordinator** - Reviews Dependabot PRs, manages auto-merge decisions
- **dependency-upgrader** - Proactively upgrades Go modules and tools
- **breaking-change-detector** - Analyzes updates for breaking changes

### 📊 Performance & Monitoring (3 agents)
- **performance-profiler** - CPU/memory profiling and optimization
- **benchmark-analyst** - Compares benchmarks, detects regressions
- **coverage-maintainer** - Manages GoFortress coverage system

### 🛡️ Security & Compliance (2 agents)
- **security-auditor** - Runs govulncheck, nancy, gitleaks, OSSAR
- **compliance-checker** - Ensures OpenSSF Scorecard compliance

### 🤖 GitHub Automation (3 agents)
- **workflow-optimizer** - Maintains GitHub Actions, optimizes CI
- **pr-automation-manager** - Handles PR labeling, auto-merge, assignments
- **issue-triage-bot** - Manages stale issues and PR cleanup

### 🔍 Diagnostics & Troubleshooting (2 agents)
- **diagnostic-specialist** - Analyzes failures, collects diagnostics
- **debugging-expert** - Deep-dive debugging with trace analysis

### 📚 Documentation & Release (3 agents)
- **documentation-maintainer** - Keeps docs synchronized and accurate
- **changelog-generator** - Generates changelogs from commits
- **release-manager** - Coordinates releases with goreleaser

### 🔨 Code Refactoring & Maintenance (3 agents)
- **code-deduplicator** - Identifies and refactors duplicate code
- **refactoring-specialist** - Improves code structure and patterns
- **tech-debt-tracker** - Identifies and prioritizes technical debt

## 🔄 Agent Collaboration Patterns

### Parallel Execution Groups
- **Quality Group**: test-commander + benchmark-runner + go-quality-enforcer
- **Security Group**: security-auditor + compliance-checker + dependabot-coordinator
- **Performance Group**: performance-profiler + benchmark-analyst + coverage-maintainer

### Sequential Workflows
1. **Release Flow**: changelog-generator → release-manager → documentation-maintainer
2. **PR Review**: pr-automation-manager → test-commander → dependabot-coordinator
3. **Debug Flow**: diagnostic-specialist → debugging-expert → refactoring-specialist

### Proactive Triggers
- **On code change**: test-commander, benchmark-runner, coverage-maintainer
- **On PR open**: pr-automation-manager, go-quality-enforcer
- **On dependency update**: dependabot-coordinator, breaking-change-detector
- **Weekly**: tech-debt-tracker, security-auditor, workflow-optimizer

## 🚀 Usage

These agents will be automatically invoked by Claude Code based on the task at hand. You can also explicitly request a specific agent:

```
"Use the test-commander agent to run all tests"
"Have the security-auditor check for vulnerabilities"
"Ask the release-manager to prepare version 1.2.0"
```

## 📈 Performance Targets

Key performance metrics monitored by agents:
- **Binary detection**: 587M+ ops/sec
- **Content comparison**: 239M+ ops/sec
- **Directory sync**: 1000 files in ~32ms
- **Cache operations**: 13.5M+ ops/sec
- **Test coverage**: >85%

## 🛠️ Maintenance

To add or modify agents:
1. Use the meta-agent: "Use the meta-agent to create a new sub-agent"
2. Edit agent files directly in this directory
3. Test the agent with specific tasks
4. Document any inter-agent dependencies

## 📝 Best Practices

1. **Single Responsibility**: Each agent should focus on one area
2. **Clear Triggers**: Define when agents should be proactive
3. **Tool Minimization**: Only grant necessary tools
4. **Collaboration**: Design agents to work together
5. **Documentation**: Keep agent descriptions clear and actionable

---

*Created for go-broadcast project management - optimized for Go development workflows*
