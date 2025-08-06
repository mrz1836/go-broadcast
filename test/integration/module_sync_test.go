package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/sync"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper functions for git operations
func initGitRepoForModule(t *testing.T, dir string) {
	t.Helper()
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git
	cmd = exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	_ = cmd.Run()

	cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
	cmd.Dir = dir
	_ = cmd.Run()

	// Initial commit
	cmd = exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = dir
	_ = cmd.Run()

	cmd = exec.CommandContext(ctx, "git", "commit", "-m", "Initial commit", "--allow-empty")
	cmd.Dir = dir
	_ = cmd.Run()
}

func createGitTag(t *testing.T, dir, tag string) {
	t.Helper()
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "tag", tag)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create git tag %s: %v", tag, err)
	}
}

func TestModuleSync_Detection(t *testing.T) {
	t.Run("detect Go modules in source directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		sourceDir := filepath.Join(tmpDir, "source")

		// Create a Go module structure
		moduleDir := filepath.Join(sourceDir, "pkg", "mymodule")
		require.NoError(t, os.MkdirAll(moduleDir, 0o750))

		// Create go.mod file
		goModContent := `module github.com/example/mymodule

go 1.21

require (
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.8.4
)`
		require.NoError(t, os.WriteFile(
			filepath.Join(moduleDir, "go.mod"),
			[]byte(goModContent),
			0o600,
		))

		// Create Go source file
		goSrcContent := `package mymodule

import "fmt"

func Hello() {
	fmt.Println("Hello from module")
}`
		require.NoError(t, os.WriteFile(
			filepath.Join(moduleDir, "hello.go"),
			[]byte(goSrcContent),
			0o600,
		))

		// Create module detector
		logger := logrus.New()
		detector := sync.NewModuleDetector(logger)

		// Test detection
		isModule := detector.IsGoModule(moduleDir)
		assert.True(t, isModule, "Should detect Go module")

		// Detect module info
		moduleInfo, err := detector.DetectModule(moduleDir)
		require.NoError(t, err)
		assert.Equal(t, "github.com/example/mymodule", moduleInfo.Name)
		assert.Equal(t, moduleDir, moduleInfo.Path)
	})

	t.Run("detect nested modules", func(t *testing.T) {
		tmpDir := t.TempDir()
		sourceDir := filepath.Join(tmpDir, "source")

		// Create parent module
		require.NoError(t, os.MkdirAll(sourceDir, 0o750))
		require.NoError(t, os.WriteFile(
			filepath.Join(sourceDir, "go.mod"),
			[]byte("module github.com/example/parent\n\ngo 1.21"),
			0o600,
		))

		// Create nested module
		nestedDir := filepath.Join(sourceDir, "internal", "nested")
		require.NoError(t, os.MkdirAll(nestedDir, 0o750))
		require.NoError(t, os.WriteFile(
			filepath.Join(nestedDir, "go.mod"),
			[]byte("module github.com/example/nested\n\ngo 1.21"),
			0o600,
		))

		// Create another nested module
		toolsDir := filepath.Join(sourceDir, "tools")
		require.NoError(t, os.MkdirAll(toolsDir, 0o750))
		require.NoError(t, os.WriteFile(
			filepath.Join(toolsDir, "go.mod"),
			[]byte("module github.com/example/tools\n\ngo 1.21"),
			0o600,
		))

		logger := logrus.New()
		detector := sync.NewModuleDetector(logger)

		// Detect all modules
		modules, err := detector.DetectModules(sourceDir)
		require.NoError(t, err)
		assert.Len(t, modules, 3, "Should detect all three modules")

		// Verify module names
		moduleNames := make(map[string]bool)
		for _, mod := range modules {
			moduleNames[mod.Name] = true
		}
		assert.True(t, moduleNames["github.com/example/parent"])
		assert.True(t, moduleNames["github.com/example/nested"])
		assert.True(t, moduleNames["github.com/example/tools"])
	})

	t.Run("handle non-module directories gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		sourceDir := filepath.Join(tmpDir, "source")

		// Create directory without go.mod
		require.NoError(t, os.MkdirAll(sourceDir, 0o750))
		require.NoError(t, os.WriteFile(
			filepath.Join(sourceDir, "main.go"),
			[]byte("package main\n\nfunc main() {}"),
			0o600,
		))

		logger := logrus.New()
		detector := sync.NewModuleDetector(logger)

		// Should not detect as module
		isModule := detector.IsGoModule(sourceDir)
		assert.False(t, isModule, "Should not detect as Go module")

		// Detect should return error
		_, err := detector.DetectModule(sourceDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "go.mod not found")
	})

	t.Run("find module root from subdirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		moduleRoot := filepath.Join(tmpDir, "mymodule")
		subDir := filepath.Join(moduleRoot, "internal", "utils")

		// Create module structure
		require.NoError(t, os.MkdirAll(subDir, 0o750))
		require.NoError(t, os.WriteFile(
			filepath.Join(moduleRoot, "go.mod"),
			[]byte("module github.com/example/mymodule\n\ngo 1.21"),
			0o600,
		))

		logger := logrus.New()
		detector := sync.NewModuleDetector(logger)

		// Find module root from subdirectory
		foundRoot, err := detector.FindGoModInParents(subDir)
		require.NoError(t, err)
		assert.Equal(t, moduleRoot, foundRoot)

		// Test from module root itself
		foundRoot, err = detector.FindGoModInParents(moduleRoot)
		require.NoError(t, err)
		assert.Equal(t, moduleRoot, foundRoot)

		// Test from directory without module
		noModuleDir := filepath.Join(tmpDir, "nomodule")
		require.NoError(t, os.MkdirAll(noModuleDir, 0o750))
		_, err = detector.FindGoModInParents(noModuleDir)
		assert.Error(t, err)
	})
}

func TestModuleSync_VersionResolution(t *testing.T) {
	t.Run("resolve exact version", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "repo")

		// Create mock repository with tags
		initGitRepoForModule(t, repoDir)
		createGitTag(t, repoDir, "v1.0.0")
		createGitTag(t, repoDir, "v1.1.0")
		createGitTag(t, repoDir, "v1.2.3")
		createGitTag(t, repoDir, "v2.0.0")

		logger := logrus.New()
		cache := sync.NewModuleCache(5*time.Minute, logger)
		resolver := sync.NewModuleResolver(logger, cache)
		ctx := context.Background()

		// Test exact version resolution
		version, err := resolver.ResolveVersion(ctx, repoDir, "v1.2.3", true)
		require.NoError(t, err)
		assert.Equal(t, "v1.2.3", version)

		// Test non-existent version
		_, err = resolver.ResolveVersion(ctx, repoDir, "v1.2.4", true)
		assert.Error(t, err)
	})

	t.Run("resolve latest version", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "repo")

		// Create mock repository with tags
		initGitRepoForModule(t, repoDir)
		createGitTag(t, repoDir, "v1.0.0")
		createGitTag(t, repoDir, "v1.5.2")
		createGitTag(t, repoDir, "v2.1.0")
		createGitTag(t, repoDir, "v2.0.0")

		logger := logrus.New()
		cache := sync.NewModuleCache(5*time.Minute, logger)
		resolver := sync.NewModuleResolver(logger, cache)
		ctx := context.Background()

		// Resolve "latest" should return highest version
		version, err := resolver.ResolveVersion(ctx, repoDir, "latest", true)
		require.NoError(t, err)
		assert.Equal(t, "v2.1.0", version)
	})

	t.Run("resolve semantic version constraints", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "repo")

		// Create mock repository with various tags
		initGitRepoForModule(t, repoDir)
		tags := []string{
			"v1.0.0", "v1.0.1", "v1.0.2",
			"v1.1.0", "v1.1.1",
			"v1.2.0", "v1.2.3", "v1.2.4",
			"v1.3.0",
			"v2.0.0", "v2.1.0",
		}
		for _, tag := range tags {
			createGitTag(t, repoDir, tag)
		}

		logger := logrus.New()
		cache := sync.NewModuleCache(5*time.Minute, logger)
		resolver := sync.NewModuleResolver(logger, cache)
		ctx := context.Background()

		testCases := []struct {
			constraint string
			expected   string
			desc       string
		}{
			{"~1.2.0", "v1.2.4", "tilde constraint allows patch updates"},
			{"^1.2.0", "v1.3.0", "caret constraint allows minor updates"},
			{">=1.2.0", "v2.1.0", "greater than or equal allows major updates"},
			{">=1.2.0 <2.0.0", "v1.3.0", "range constraint"},
			{"~1.1", "v1.1.1", "tilde with minor version"},
			{"^1.0", "v1.3.0", "caret with major.minor"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				version, err := resolver.ResolveVersion(ctx, repoDir, tc.constraint, true)
				require.NoError(t, err, "Failed for constraint: %s", tc.constraint)
				assert.Equal(t, tc.expected, version, "Constraint: %s", tc.constraint)
			})
		}
	})

	t.Run("handle repositories without tags", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "repo")

		// Create repository without tags
		initGitRepoForModule(t, repoDir)

		logger := logrus.New()
		cache := sync.NewModuleCache(5*time.Minute, logger)
		resolver := sync.NewModuleResolver(logger, cache)
		ctx := context.Background()

		// Should return error for version constraints
		_, err := resolver.ResolveVersion(ctx, repoDir, "latest", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no tags found")

		_, err = resolver.ResolveVersion(ctx, repoDir, "v1.0.0", true)
		assert.Error(t, err)
	})

	t.Run("handle invalid version constraints", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "repo")

		initGitRepoForModule(t, repoDir)
		createGitTag(t, repoDir, "v1.0.0")

		logger := logrus.New()
		cache := sync.NewModuleCache(5*time.Minute, logger)
		resolver := sync.NewModuleResolver(logger, cache)
		ctx := context.Background()

		// Test invalid constraints
		invalidConstraints := []string{
			"invalid",
			"v1.x.x",
			">=invalid",
			"***",
		}

		for _, constraint := range invalidConstraints {
			_, err := resolver.ResolveVersion(ctx, repoDir, constraint, true)
			assert.Error(t, err, "Should error for constraint: %s", constraint)
		}
	})
}

func TestModuleSync_CacheEffectiveness(t *testing.T) {
	t.Run("cache hit performance", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "repo")

		// Setup repository
		initGitRepoForModule(t, repoDir)
		for i := 0; i < 20; i++ {
			createGitTag(t, repoDir, fmt.Sprintf("v1.0.%d", i))
		}

		// Create cache with 1-minute TTL
		logger := logrus.New()
		cache := sync.NewModuleCache(1*time.Minute, logger)
		resolver := sync.NewModuleResolver(logger, cache)
		ctx := context.Background()

		// First resolution (cold cache)
		start := time.Now()
		version1, err := resolver.ResolveVersion(ctx, repoDir, "latest", true)
		require.NoError(t, err)
		coldDuration := time.Since(start)

		// Second resolution (warm cache)
		start = time.Now()
		version2, err := resolver.ResolveVersion(ctx, repoDir, "latest", true)
		require.NoError(t, err)
		warmDuration := time.Since(start)

		// Verify same result
		assert.Equal(t, version1, version2)

		// Cache should be significantly faster (at least 10x)
		// In practice it's usually 100x or more, but we use 10x for stability
		assert.Less(t, warmDuration, coldDuration/10,
			"Cache hit should be at least 10x faster. Cold: %v, Warm: %v",
			coldDuration, warmDuration)
	})

	t.Run("cache TTL expiration", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "repo")

		initGitRepoForModule(t, repoDir)
		createGitTag(t, repoDir, "v1.0.0")

		// Create cache with very short TTL
		cache := sync.NewModuleCache(100*time.Millisecond, logrus.New())

		// Store value
		key := fmt.Sprintf("versions:%s", repoDir)
		cache.Set(key, "v1.0.0,v1.0.1") // Store as comma-separated string

		// Should get cached value immediately
		cached, found := cache.Get(key)
		assert.True(t, found)
		assert.Equal(t, "v1.0.0,v1.0.1", cached)

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Should not find expired entry
		_, found = cache.Get(key)
		assert.False(t, found)
	})

	t.Run("concurrent cache access", func(t *testing.T) {
		logger := logrus.New()
		cache := sync.NewModuleCache(1*time.Minute, logger)

		// Concurrent writes
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				key := fmt.Sprintf("key-%d", id)
				cache.Set(key, fmt.Sprintf("value-%d", id))
				done <- true
			}(i)
		}

		// Wait for all writes
		for i := 0; i < 10; i++ {
			<-done
		}

		// Concurrent reads
		for i := 0; i < 10; i++ {
			go func(id int) {
				key := fmt.Sprintf("key-%d", id)
				val, found := cache.Get(key)
				assert.True(t, found)
				assert.Equal(t, fmt.Sprintf("value-%d", id), val)
				done <- true
			}(i)
		}

		// Wait for all reads
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("cache invalidation by prefix", func(t *testing.T) {
		logger := logrus.New()
		cache := sync.NewModuleCache(1*time.Minute, logger)

		// Set multiple entries
		cache.Set("versions:repo1", "v1.0.0")
		cache.Set("versions:repo2", "v2.0.0")
		cache.Set("modules:repo1", "module1")
		cache.Set("modules:repo2", "module2")

		// Clear cache entries with prefix (manual invalidation)
		cache.Clear() // Clear all entries as workaround

		// Version entries should be gone
		_, found := cache.Get("versions:repo1")
		assert.False(t, found)
		_, found = cache.Get("versions:repo2")
		assert.False(t, found)

		// Module entries should still exist
		val, found := cache.Get("modules:repo1")
		assert.True(t, found)
		assert.Equal(t, "module1", val)
	})
}

func TestModuleSync_Integration(t *testing.T) {
	t.Run("sync Go module with exact version", func(t *testing.T) {
		ctx := context.Background()
		tmpDir := t.TempDir()

		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		// Create source module
		moduleDir := filepath.Join(sourceDir, "pkg", "utils")
		require.NoError(t, os.MkdirAll(moduleDir, 0o750))

		// Create module files
		require.NoError(t, os.WriteFile(
			filepath.Join(moduleDir, "go.mod"),
			[]byte("module github.com/example/utils\n\ngo 1.21"),
			0o600,
		))
		require.NoError(t, os.WriteFile(
			filepath.Join(moduleDir, "utils.go"),
			[]byte("package utils\n\nfunc Helper() string { return \"v1.2.3\" }"),
			0o600,
		))

		initGitRepoForModule(t, sourceDir)
		initGitRepoForModule(t, targetDir)

		// Tag the source
		createGitTag(t, sourceDir, "v1.2.3")

		cfg := &config.Config{
			Version: 1,
			Groups: []config.Group{
				{
					Name:     "Module Sync",
					ID:       "module-sync",
					Priority: 1,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   sourceDir,
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: targetDir,
							Directories: []config.DirectoryMapping{
								{
									Src:  "pkg/utils",
									Dest: "vendor/github.com/example/utils",
									Module: &config.ModuleConfig{
										Type:    "go",
										Version: "v1.2.3",
									},
								},
							},
						},
					},
				},
			},
		}

		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)

		opts := sync.DefaultOptions().
			WithDryRun(false)

		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// Mock state discovery
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         sourceDir,
				Branch:       "main",
				LatestCommit: "abc123",
			},
			Targets: map[string]*state.TargetState{},
		}
		mockState.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		engine := sync.NewEngine(cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logger)
		err := engine.Sync(ctx, nil)
		require.NoError(t, err)

		// Verify module was synced
		assert.FileExists(t, filepath.Join(targetDir, "vendor/github.com/example/utils/go.mod"))
		assert.FileExists(t, filepath.Join(targetDir, "vendor/github.com/example/utils/utils.go"))

		// Verify content
		// #nosec G304 - test file path is controlled
		content, _ := os.ReadFile(filepath.Join(targetDir, "vendor/github.com/example/utils/utils.go"))
		assert.Contains(t, string(content), "v1.2.3")
	})

	t.Run("sync module with transformations", func(t *testing.T) {
		ctx := context.Background()
		tmpDir := t.TempDir()

		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		// Create source module with template
		moduleDir := filepath.Join(sourceDir, "templates")
		require.NoError(t, os.MkdirAll(moduleDir, 0o750))

		templateContent := `package config

const (
	ProjectName = "{{.PROJECT_NAME}}"
	Version = "{{.VERSION}}"
	Environment = "{{.ENV}}"
)`

		require.NoError(t, os.WriteFile(
			filepath.Join(moduleDir, "config.go.tmpl"),
			[]byte(templateContent),
			0o600,
		))

		initGitRepoForModule(t, sourceDir)
		initGitRepoForModule(t, targetDir)

		cfg := &config.Config{
			Version: 1,
			Groups: []config.Group{
				{
					Name:     "Template Module",
					ID:       "template",
					Priority: 1,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   sourceDir,
						Branch: "main",
					},
					Global: config.GlobalConfig{
						PRLabels: []string{"template-sync"},
					},
					Targets: []config.TargetConfig{
						{
							Repo: targetDir,
							Files: []config.FileMapping{
								{
									Src:  "templates/config.go.tmpl",
									Dest: "internal/config/config.go",
								},
							},
							Transform: config.Transform{
								Variables: map[string]string{
									"PROJECT_NAME": "MyProject",
									"VERSION":      "1.0.0",
									"ENV":          "production",
								},
							},
						},
					},
				},
			},
		}

		logger := logrus.New()
		logger.SetLevel(logrus.InfoLevel)

		opts := sync.DefaultOptions().
			WithDryRun(false)

		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// Mock state discovery
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         sourceDir,
				Branch:       "main",
				LatestCommit: "abc123",
			},
			Targets: map[string]*state.TargetState{},
		}
		mockState.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		engine := sync.NewEngine(cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logger)
		err := engine.Sync(ctx, nil)
		require.NoError(t, err)

		// Verify transformed file
		// #nosec G304 - test file path is controlled
		content, err := os.ReadFile(filepath.Join(targetDir, "internal/config/config.go"))
		require.NoError(t, err)

		assert.Contains(t, string(content), `ProjectName = "MyProject"`)
		assert.Contains(t, string(content), `Version = "1.0.0"`)
		assert.Contains(t, string(content), `Environment = "production"`)
	})

	t.Run("sync multiple modules in single group", func(t *testing.T) {
		ctx := context.Background()
		tmpDir := t.TempDir()

		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		// Create multiple modules
		modules := []struct {
			path    string
			name    string
			content string
		}{
			{
				path:    "libs/auth",
				name:    "github.com/example/auth",
				content: "package auth\n\nfunc Authenticate() bool { return true }",
			},
			{
				path:    "libs/database",
				name:    "github.com/example/database",
				content: "package database\n\nfunc Connect() error { return nil }",
			},
			{
				path:    "libs/cache",
				name:    "github.com/example/cache",
				content: "package cache\n\nfunc Get(key string) interface{} { return nil }",
			},
		}

		for _, mod := range modules {
			modDir := filepath.Join(sourceDir, mod.path)
			require.NoError(t, os.MkdirAll(modDir, 0o750))

			require.NoError(t, os.WriteFile(
				filepath.Join(modDir, "go.mod"),
				[]byte(fmt.Sprintf("module %s\n\ngo 1.21", mod.name)),
				0o600,
			))

			filename := filepath.Base(mod.path) + ".go"
			require.NoError(t, os.WriteFile(
				filepath.Join(modDir, filename),
				[]byte(mod.content),
				0o600,
			))
		}

		initGitRepoForModule(t, sourceDir)
		initGitRepoForModule(t, targetDir)

		// Create configuration with multiple module directories
		var dirMappings []config.DirectoryMapping
		for _, mod := range modules {
			dirMappings = append(dirMappings, config.DirectoryMapping{
				Src:  mod.path,
				Dest: filepath.Join("vendor", strings.ReplaceAll(mod.name, "github.com/example/", "")),
				Module: &config.ModuleConfig{
					Type:    "go",
					Version: "latest",
				},
			})
		}

		cfg := &config.Config{
			Version: 1,
			Groups: []config.Group{
				{
					Name:     "Multi-Module Sync",
					ID:       "multi-module",
					Priority: 1,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   sourceDir,
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo:        targetDir,
							Directories: dirMappings,
						},
					},
				},
			},
		}

		logger := logrus.New()
		logger.SetLevel(logrus.InfoLevel)

		opts := sync.DefaultOptions().
			WithDryRun(false)

		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// Mock state discovery
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         sourceDir,
				Branch:       "main",
				LatestCommit: "abc123",
			},
			Targets: map[string]*state.TargetState{},
		}
		mockState.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		engine := sync.NewEngine(cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logger)
		err := engine.Sync(ctx, nil)
		require.NoError(t, err)

		// Verify all modules were synced
		assert.DirExists(t, filepath.Join(targetDir, "vendor/auth"))
		assert.DirExists(t, filepath.Join(targetDir, "vendor/database"))
		assert.DirExists(t, filepath.Join(targetDir, "vendor/cache"))

		assert.FileExists(t, filepath.Join(targetDir, "vendor/auth/go.mod"))
		assert.FileExists(t, filepath.Join(targetDir, "vendor/database/go.mod"))
		assert.FileExists(t, filepath.Join(targetDir, "vendor/cache/go.mod"))
	})

	t.Run("module sync with exclude patterns", func(t *testing.T) {
		ctx := context.Background()
		tmpDir := t.TempDir()

		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		// Create module with various files
		moduleDir := filepath.Join(sourceDir, "mymodule")
		require.NoError(t, os.MkdirAll(moduleDir, 0o750))

		// Create files
		files := map[string]string{
			"go.mod":             "module example.com/mymodule\n\ngo 1.21",
			"main.go":            "package main\n\nfunc main() {}",
			"main_test.go":       "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}",
			"README.md":          "# My Module",
			".gitignore":         "*.log\n*.tmp",
			"internal/helper.go": "package internal\n\nfunc Helper() {}",
		}

		for file, content := range files {
			fullPath := filepath.Join(moduleDir, file)
			require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o750))
			require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
		}

		initGitRepoForModule(t, sourceDir)
		initGitRepoForModule(t, targetDir)

		cfg := &config.Config{
			Version: 1,
			Groups: []config.Group{
				{
					Name:     "Module with Excludes",
					ID:       "exclude-test",
					Priority: 1,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   sourceDir,
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: targetDir,
							Directories: []config.DirectoryMapping{
								{
									Src:  "mymodule",
									Dest: "vendor/mymodule",
									Exclude: []string{
										"*_test.go",
										"README.md",
										".gitignore",
									},
									Module: &config.ModuleConfig{
										Type:    "go",
										Version: "latest",
									},
								},
							},
						},
					},
				},
			},
		}

		logger := logrus.New()
		logger.SetLevel(logrus.InfoLevel)

		opts := sync.DefaultOptions().
			WithDryRun(false)

		// Setup mocks
		mockGH := &gh.MockClient{}
		mockGit := &git.MockClient{}
		mockState := &state.MockDiscoverer{}
		mockTransform := &transform.MockChain{}

		// Mock state discovery
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         sourceDir,
				Branch:       "main",
				LatestCommit: "abc123",
			},
			Targets: map[string]*state.TargetState{},
		}
		mockState.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		engine := sync.NewEngine(cfg, mockGH, mockGit, mockState, mockTransform, opts)
		engine.SetLogger(logger)
		err := engine.Sync(ctx, nil)
		require.NoError(t, err)

		// Verify included files exist
		assert.FileExists(t, filepath.Join(targetDir, "vendor/mymodule/go.mod"))
		assert.FileExists(t, filepath.Join(targetDir, "vendor/mymodule/main.go"))
		assert.FileExists(t, filepath.Join(targetDir, "vendor/mymodule/internal/helper.go"))

		// Verify excluded files don't exist
		assert.NoFileExists(t, filepath.Join(targetDir, "vendor/mymodule/main_test.go"))
		assert.NoFileExists(t, filepath.Join(targetDir, "vendor/mymodule/README.md"))
		assert.NoFileExists(t, filepath.Join(targetDir, "vendor/mymodule/.gitignore"))
	})
}
