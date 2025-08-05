// Package fixtures provides utilities for generating test data and scenarios
// for integration testing of the go-broadcast synchronization engine.
package fixtures

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// TestRepoGenerator provides utilities for creating test repository scenarios
type TestRepoGenerator struct {
	BaseDir string
	TempDir string
}

// NewTestRepoGenerator creates a new test repository generator
func NewTestRepoGenerator(baseDir string) *TestRepoGenerator {
	return &TestRepoGenerator{
		BaseDir: baseDir,
	}
}

// TestScenario represents a complete test scenario with multiple repositories
type TestScenario struct {
	Name        string
	Description string
	SourceRepo  *TestRepository
	TargetRepos []*TestRepository
	Config      *config.Config
	State       *state.State
}

// TestRepository represents a test repository with its content and metadata
type TestRepository struct {
	Name        string
	Owner       string
	Branch      string
	CommitSHA   string
	Files       map[string][]byte
	Path        string
	HasConflict bool
	Size        int64 // Total size in bytes
}

// FailureMode defines types of failures that can be injected into tests
type FailureMode int

const (
	// FailureNone indicates no failure should be injected
	FailureNone FailureMode = iota
	// FailureNetwork indicates network-related failures
	FailureNetwork
	// FailureAuth indicates authentication failures
	FailureAuth
	// FailureRateLimit indicates rate limiting failures
	FailureRateLimit
	// FailurePartial indicates partial operation failures
	FailurePartial
	// FailureCorruption indicates data corruption failures
	FailureCorruption
	// FailureTimeout indicates timeout failures
	FailureTimeout
)

// Static error definitions for test fixtures
var (
	ErrNetworkTimeout       = errors.New("network error: connection timeout")
	ErrAuthenticationFailed = errors.New("authentication failed: invalid credentials")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded: too many requests")
	ErrPartialFailure       = errors.New("partial failure: some operations failed")
	ErrDataCorruption       = errors.New("data corruption detected")
	ErrOperationTimeout     = errors.New("operation timeout")
	ErrUnknownFailure       = errors.New("unknown failure")
	ErrNetworkPartition     = errors.New("network partition: connection refused")
	ErrBranchProtection     = errors.New("branch protection: direct pushes not allowed")
	ErrPushRejected         = errors.New("push failed: remote rejected")
	ErrUnauthorized         = errors.New("401 Unauthorized: bad credentials")
	ErrForbidden            = errors.New("403 Forbidden: insufficient permissions")
	ErrFileNotFound         = errors.New("file not found")
	ErrNetworkFailure       = errors.New("simulated network failure")
	ErrGitCloneFailed       = errors.New("git clone failed")
	ErrNetworkRefused       = errors.New("network error: connection refused")
	ErrNetworkUnreachable   = errors.New("git clone failed: network unreachable")
	ErrGitPushTimeout       = errors.New("git push failed: connection timeout")
	ErrGitAuthRequired      = errors.New("git clone failed: authentication required")
	ErrGitPermissionDenied  = errors.New("git push failed: permission denied")
	ErrRequestTimeout       = errors.New("request timeout")
	ErrServiceDegradation   = errors.New("partial service degradation")
	ErrSSLCertificate       = errors.New("git clone failed: x509: certificate signed by unknown authority")
	ErrProxyConnection      = errors.New("git clone failed: proxy connection failed (407 Proxy Authentication Required)")
)

// CreateRepo generates a test repository with the specified files
func (g *TestRepoGenerator) CreateRepo(name, owner, branch string, files map[string]string) (*TestRepository, error) {
	repoPath := filepath.Join(g.BaseDir, fmt.Sprintf("%s-%s", owner, name))

	// Create directory structure
	err := os.MkdirAll(repoPath, 0o750)
	if err != nil {
		return nil, fmt.Errorf("failed to create repo directory: %w", err)
	}

	// Convert string files to byte files and calculate total size
	byteFiles := make(map[string][]byte)
	var totalSize int64

	for path, content := range files {
		bytes := []byte(content)
		byteFiles[path] = bytes
		totalSize += int64(len(bytes))

		// Create file and its directory structure
		fullPath := filepath.Join(repoPath, path)
		dir := filepath.Dir(fullPath)

		err := os.MkdirAll(dir, 0o750)
		if err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		err = os.WriteFile(fullPath, bytes, 0o600)
		if err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return &TestRepository{
		Name:      name,
		Owner:     owner,
		Branch:    branch,
		CommitSHA: generateCommitSHA(),
		Files:     byteFiles,
		Path:      repoPath,
		Size:      totalSize,
	}, nil
}

// CreateLargeFileRepo creates a repository with large files for testing memory usage
func (g *TestRepoGenerator) CreateLargeFileRepo(name, owner string, fileSizeMB int) (*TestRepository, error) {
	files := make(map[string]string)

	// Create a large file
	largeContent := make([]byte, fileSizeMB*1024*1024)
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
	}

	files[fmt.Sprintf("large_file_%dmb.txt", fileSizeMB)] = string(largeContent)
	files["README.md"] = fmt.Sprintf("# Large File Test Repository\n\nThis repository contains a %dMB test file.", fileSizeMB)
	files[".github/workflows/ci.yml"] = getStandardCIWorkflow()

	return g.CreateRepo(name, owner, "master", files)
}

// CreateConflictingRepo creates a repository that will have conflicts when synced
func (g *TestRepoGenerator) CreateConflictingRepo(name, owner string) (*TestRepository, error) {
	files := map[string]string{
		"README.md": fmt.Sprintf("# %s\n\nThis file has been modified in the target repo and will conflict.", name),
		".github/workflows/ci.yml": `name: CI (Modified)
on:
  push:
    branches: [ master, development ]
  pull_request:
    branches: [ master ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Run custom tests
      run: echo "Custom test for target repo"`,
		"Makefile": `# Modified Makefile for target repo
.PHONY: test build
test:
	go test -v ./...
build:
	go build -o bin/app ./cmd/app`,
	}

	repo, err := g.CreateRepo(name, owner, "master", files)
	if err != nil {
		return nil, err
	}

	repo.HasConflict = true
	return repo, nil
}

// CreateComplexScenario generates a multi-repository test scenario
func (g *TestRepoGenerator) CreateComplexScenario() (*TestScenario, error) {
	// Create source repository
	sourceFiles := map[string]string{
		"README.md":                     "# Template Repository\n\nThis is the source template for synchronization.",
		".github/workflows/ci.yml":      getStandardCIWorkflow(),
		".github/workflows/release.yml": getStandardReleaseWorkflow(),
		"Makefile":                      getStandardMakefile(),
		"docker-compose.yml":            getStandardDockerCompose(),
		"scripts/setup.sh":              getStandardSetupScript(),
		"docs/API.md":                   "# API Documentation\n\nTemplate API documentation.",
	}

	sourceRepo, err := g.CreateRepo("template-repo", "org", "master", sourceFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to create source repo: %w", err)
	}

	// Create target repositories with varying scenarios
	var targetRepos []*TestRepository

	// Normal target repo
	normalRepo, err := g.CreateRepo("service-a", "org", "master", map[string]string{
		"README.md": "# Service A\n\nA microservice.",
		"main.go":   "package main\n\nfunc main() {\n\tprintln(\"Service A\")\n}",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create normal target repo: %w", err)
	}
	targetRepos = append(targetRepos, normalRepo)

	// Conflicting target repo
	conflictRepo, err := g.CreateConflictingRepo("service-b", "org")
	if err != nil {
		return nil, fmt.Errorf("failed to create conflicting target repo: %w", err)
	}
	targetRepos = append(targetRepos, conflictRepo)

	// Large file target repo
	largeRepo, err := g.CreateLargeFileRepo("service-c", "org", 50) // 50MB file
	if err != nil {
		return nil, fmt.Errorf("failed to create large file target repo: %w", err)
	}
	targetRepos = append(targetRepos, largeRepo)

	// Create configuration
	cfg := &config.Config{
		Version: 1,
		Defaults: config.DefaultConfig{
			BranchPrefix: "chore/sync-files",
			PRLabels:     []string{"automated-sync", "integration-test"},
		},
		Mappings: []config.SourceMapping{
			{
				Source: config.SourceConfig{
					Repo:   "org/template-repo",
					Branch: "master",
					ID:     "template",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "org/service-a",
						Files: []config.FileMapping{
							{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
							{Src: "Makefile", Dest: "Makefile"},
						},
						Transform: config.Transform{
							RepoName:  true,
							Variables: map[string]string{"SERVICE_NAME": "service-a"},
						},
					},
					{
						Repo: "org/service-b",
						Files: []config.FileMapping{
							{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
							{Src: "Makefile", Dest: "Makefile"},
							{Src: "README.md", Dest: "README.md"},
						},
						Transform: config.Transform{
							RepoName:  true,
							Variables: map[string]string{"SERVICE_NAME": "service-b"},
						},
					},
					{
						Repo: "org/service-c",
						Files: []config.FileMapping{
							{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
							{Src: "docker-compose.yml", Dest: "docker-compose.yml"},
						},
						Transform: config.Transform{
							RepoName:  true,
							Variables: map[string]string{"SERVICE_NAME": "service-c"},
						},
					},
				},
			},
		},
	}

	// Create state
	currentState := &state.State{
		Source: state.SourceState{
			Repo:         "org/template-repo",
			Branch:       "master",
			LatestCommit: sourceRepo.CommitSHA,
			LastChecked:  time.Now(),
		},
		Sources:         make(map[string]state.SourceState),
		Targets:         make(map[string]*state.TargetState),
		SourceTargetMap: make(map[string]map[string]*state.SourceTargetSyncInfo),
	}

	// Populate the Sources map for v2 compatibility
	currentState.Sources["org/template-repo"] = state.SourceState{
		Repo:         "org/template-repo",
		Branch:       "master",
		LatestCommit: sourceRepo.CommitSHA,
		LastChecked:  time.Now(),
	}

	// Set target states (some outdated, some up-to-date)
	for i, target := range targetRepos {
		status := state.StatusBehind
		lastSyncCommit := "old" + generateCommitSHA()[:10]

		// Make first repo up-to-date for variety
		if i == 0 {
			status = state.StatusUpToDate
			lastSyncCommit = sourceRepo.CommitSHA
		}

		currentState.Targets[fmt.Sprintf("org/%s", target.Name)] = &state.TargetState{
			Repo:           fmt.Sprintf("org/%s", target.Name),
			LastSyncCommit: lastSyncCommit,
			Status:         status,
			LastSyncTime:   &[]time.Time{time.Now().Add(-time.Duration(i+1) * time.Hour)}[0],
		}
	}

	return &TestScenario{
		Name:        "Complex Multi-Repository Sync",
		Description: "Multi-repo scenario with normal, conflicting, and large file repositories",
		SourceRepo:  sourceRepo,
		TargetRepos: targetRepos,
		Config:      cfg,
		State:       currentState,
	}, nil
}

// CreatePartialFailureScenario creates a scenario where some repos succeed and others fail
func (g *TestRepoGenerator) CreatePartialFailureScenario() (*TestScenario, error) {
	scenario, err := g.CreateComplexScenario()
	if err != nil {
		return nil, err
	}

	scenario.Name = "Partial Failure Recovery"
	scenario.Description = "Scenario where sync partially fails and requires recovery"

	// Mark some targets as having conflicts
	for i, target := range scenario.TargetRepos {
		if i == 1 { // Second repo will have conflicts
			scenario.State.Targets[fmt.Sprintf("org/%s", target.Name)].Status = state.StatusConflict
		}
	}

	return scenario, nil
}

// CreateNetworkFailureScenario creates a scenario for testing network resilience
func (g *TestRepoGenerator) CreateNetworkFailureScenario() (*TestScenario, error) {
	scenario, err := g.CreateComplexScenario()
	if err != nil {
		return nil, err
	}

	scenario.Name = "Network Failure Resilience"
	scenario.Description = "Tests network interruption and rate limiting scenarios"

	return scenario, nil
}

// CreateBranchProtectionScenario creates a scenario for testing branch protection
func (g *TestRepoGenerator) CreateBranchProtectionScenario() (*TestScenario, error) {
	scenario, err := g.CreateComplexScenario()
	if err != nil {
		return nil, err
	}

	scenario.Name = "Branch Protection Testing"
	scenario.Description = "Tests sync behavior with protected branches and required reviews"

	// Branch protection is handled at the repository level, not in config
	// The sync engine will detect protected branches via GitHub API calls

	return scenario, nil
}

// MockFailureInjector provides controlled failure injection for testing
type MockFailureInjector struct {
	FailureMode   FailureMode
	FailureRate   float64 // 0.0 to 1.0
	FailureCount  int
	FailureRepos  []string
	RecoveryDelay time.Duration
}

// ShouldFail determines if an operation should fail based on the injector configuration
func (m *MockFailureInjector) ShouldFail(_ context.Context, repo, _ string) bool {
	if m.FailureMode == FailureNone {
		return false
	}

	// Check if this repo is in the failure list
	if len(m.FailureRepos) > 0 {
		found := false
		for _, failRepo := range m.FailureRepos {
			if strings.Contains(repo, failRepo) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check failure count
	if m.FailureCount > 0 {
		m.FailureCount--
		return true
	}

	// Check failure rate
	if m.FailureRate > 0 {
		return (float64(time.Now().UnixNano()%1000) / 1000.0) < m.FailureRate
	}

	return false
}

// GetFailureError returns an appropriate error for the failure mode
func (m *MockFailureInjector) GetFailureError(operation string) error {
	switch m.FailureMode {
	case FailureNetwork:
		return fmt.Errorf("%w during %s", ErrNetworkTimeout, operation)
	case FailureAuth:
		return fmt.Errorf("%w for %s", ErrAuthenticationFailed, operation)
	case FailureRateLimit:
		return fmt.Errorf("%w for %s", ErrRateLimitExceeded, operation)
	case FailurePartial:
		return fmt.Errorf("%w during %s", ErrPartialFailure, operation)
	case FailureCorruption:
		return fmt.Errorf("%w during %s", ErrDataCorruption, operation)
	case FailureTimeout:
		return fmt.Errorf("%w: %s took too long", ErrOperationTimeout, operation)
	default:
		return fmt.Errorf("%w during %s", ErrUnknownFailure, operation)
	}
}

// Cleanup removes all generated test data
func (g *TestRepoGenerator) Cleanup() error {
	if g.BaseDir != "" {
		return os.RemoveAll(g.BaseDir)
	}
	return nil
}

// generateCommitSHA creates a realistic-looking commit SHA for testing
func generateCommitSHA() string {
	bytes := make([]byte, 20)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to time-based generation
		return fmt.Sprintf("%x", time.Now().UnixNano())[:40]
	}
	return fmt.Sprintf("%x", bytes)
}

// Standard file templates for test repositories

func getStandardCIWorkflow() string {
	return `name: CI
on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
    - name: Run tests
      run: make test
    - name: Run linter
      run: make lint`
}

func getStandardReleaseWorkflow() string {
	return `name: Release
on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Create Release
      uses: actions/create-release@v1
      env:
        GITHUB_TOKEN: ${{ secrets.GH_PAT_TOKEN || secrets.GITHUB_TOKEN }}
      with:
        tag_name: ${{ github.ref }}
        release_name: Release ${{ github.ref }}`
}

func getStandardMakefile() string {
	return `# Template Makefile
.PHONY: test build lint clean

test:
	go test -v ./...

build:
	go build -o bin/app ./cmd/app

lint:
	golangci-lint run

clean:
	rm -rf bin/`
}

func getStandardDockerCompose() string {
	return `version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ENV=development

  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: appdb
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"`
}

func getStandardSetupScript() string {
	return `#!/bin/bash
# Setup script for development environment

set -e

echo "Setting up development environment..."

# Install dependencies
go mod download

# Run initial tests
make test

echo "Setup complete!"`
}
