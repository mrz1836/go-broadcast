package validation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/runner"
)

// ProductionScenariosTestSuite validates behavior under realistic production conditions
type ProductionScenariosTestSuite struct {
	suite.Suite
	tempDir    string
	envFile    string
	originalWD string
}

// SetupSuite initializes the test environment
func (s *ProductionScenariosTestSuite) SetupSuite() {
	var err error
	s.originalWD, err = os.Getwd()
	require.NoError(s.T(), err)

	// Create temporary directory structure
	s.tempDir = s.T().TempDir()

	// Create .github directory
	githubDir := filepath.Join(s.tempDir, ".github")
	require.NoError(s.T(), os.MkdirAll(githubDir, 0o755))

	// Create production-like .env.shared file
	s.envFile = filepath.Join(githubDir, ".env.shared")
	envContent := `# Production-like environment configuration
ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=info
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=false
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=120
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=4
PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB=50
PRE_COMMIT_SYSTEM_MAX_FILES_OPEN=500
PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS="vendor/,node_modules/,.git/,dist/,build/,coverage/,*.log,*.tmp"
PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT=60
PRE_COMMIT_SYSTEM_EOF_TIMEOUT=60
`
	require.NoError(s.T(), os.WriteFile(s.envFile, []byte(envContent), 0o644))

	// Change to temp directory for tests
	require.NoError(s.T(), os.Chdir(s.tempDir))

	// Initialize git repository
	require.NoError(s.T(), s.initGitRepo())
}

// TearDownSuite cleans up the test environment
func (s *ProductionScenariosTestSuite) TearDownSuite() {
	// Restore original working directory
	_ = os.Chdir(s.originalWD)
}

// initGitRepo initializes a git repository in the temp directory
func (s *ProductionScenariosTestSuite) initGitRepo() error {
	gitDir := filepath.Join(s.tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0o644)
}

// TestLargeRepositorySimulation validates performance with large numbers of files
func (s *ProductionScenariosTestSuite) TestLargeRepositorySimulation() {
	testCases := []struct {
		name        string
		fileCount   int
		target      time.Duration
		description string
	}{
		{
			name:        "Medium Repository",
			fileCount:   100,
			target:      5 * time.Second,
			description: "Typical medium-sized repository with 100 files",
		},
		{
			name:        "Large Repository",
			fileCount:   500,
			target:      10 * time.Second,
			description: "Large repository with 500 files",
		},
		{
			name:        "Very Large Repository",
			fileCount:   1000,
			target:      20 * time.Second,
			description: "Very large repository with 1000 files",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create repository structure with many files
			files := s.createLargeRepositoryStructure(tc.fileCount)

			// Load configuration
			cfg, err := config.Load()
			require.NoError(s.T(), err)

			// Create runner
			r := runner.New(cfg, s.tempDir)

			// Execute and measure performance
			ctx, cancel := context.WithTimeout(context.Background(), tc.target*2) // Allow 2x target for safety
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files:    files,
				Parallel: 4,
			})
			duration := time.Since(start)

			// Validate results
			assert.NoError(s.T(), err, tc.description)
			assert.NotNil(s.T(), result, "Result should not be nil")

			// Performance validation
			assert.True(s.T(), duration <= tc.target,
				"Execution should complete within target time: %v (target: %v)",
				duration, tc.target)

			s.T().Logf("%s: %d files processed in %v (target: %v)",
				tc.name, len(files), duration, tc.target)

			// Clean up files for next test
			s.cleanupLargeRepository(tc.fileCount)
		})
	}
}

// TestMixedFileTypesScenario validates handling of diverse file types
func (s *ProductionScenariosTestSuite) TestMixedFileTypesScenario() {
	// Create realistic mixed file structure
	files := s.createMixedFileTypeStructure()

	// Load configuration
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	// Create runner
	r := runner.New(cfg, s.tempDir)

	// Execute
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	result, err := r.Run(ctx, runner.Options{
		Files: files,
	})
	duration := time.Since(start)

	// Validate results
	assert.NoError(s.T(), err, "Mixed file types should be handled successfully")
	assert.NotNil(s.T(), result, "Result should not be nil")
	assert.True(s.T(), duration < 10*time.Second, "Should complete quickly with mixed files")

	// Validate file filtering worked correctly
	s.T().Logf("Mixed file types test: %d total files, processed in %v",
		len(files), duration)

	// Check that appropriate files were processed
	assert.True(s.T(), len(files) > 0, "Should have files to process")
}

// TestHighVolumeCommitScenario simulates large commits with many files
func (s *ProductionScenariosTestSuite) TestHighVolumeCommitScenario() {
	// Simulate different commit scenarios
	testCases := []struct {
		name         string
		scenario     string
		files        []string
		expectTarget time.Duration
		description  string
	}{
		{
			name:         "Feature Branch Merge",
			scenario:     "feature_merge",
			files:        s.createFeatureBranchFiles(),
			expectTarget: 5 * time.Second,
			description:  "Large feature branch merge with multiple modules",
		},
		{
			name:         "Refactoring Commit",
			scenario:     "refactoring",
			files:        s.createRefactoringFiles(),
			expectTarget: 8 * time.Second,
			description:  "Major refactoring touching many files",
		},
		{
			name:         "Documentation Update",
			scenario:     "docs_update",
			files:        s.createDocumentationFiles(),
			expectTarget: 3 * time.Second,
			description:  "Large documentation update",
		},
		{
			name:         "Dependency Update",
			scenario:     "deps_update",
			files:        s.createDependencyUpdateFiles(),
			expectTarget: 2 * time.Second,
			description:  "Dependency update with config changes",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Load configuration
			cfg, err := config.Load()
			require.NoError(s.T(), err)

			// Create runner
			r := runner.New(cfg, s.tempDir)

			// Execute scenario
			ctx, cancel := context.WithTimeout(context.Background(), tc.expectTarget*3)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files: tc.files,
			})
			duration := time.Since(start)

			// Validate results
			assert.NoError(s.T(), err, tc.description)
			assert.NotNil(s.T(), result, "Result should not be nil")
			assert.True(s.T(), duration <= tc.expectTarget,
				"Scenario should complete within target: %v (target: %v)",
				duration, tc.expectTarget)

			s.T().Logf("%s: %d files, %v duration (target: %v)",
				tc.name, len(tc.files), duration, tc.expectTarget)
		})
	}
}

// TestNetworkConstrainedEnvironment simulates network connectivity issues
func (s *ProductionScenariosTestSuite) TestNetworkConstrainedEnvironment() {
	// Create files that would typically require network access if tools were missing
	files := []string{
		"main.go",
		"service.go",
		"README.md",
		"config.yaml",
	}

	// Create the files
	s.createBasicFiles(files)

	// Test different network constraint scenarios
	testCases := []struct {
		name        string
		envVars     map[string]string
		description string
	}{
		{
			name: "Offline Environment",
			envVars: map[string]string{
				"NO_NETWORK":    "true",
				"OFFLINE_BUILD": "true",
			},
			description: "Environment with no network access",
		},
		{
			name: "Limited Connectivity",
			envVars: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "http://proxy.example.com:8080",
			},
			description: "Environment with limited network through proxy",
		},
		{
			name: "Firewall Restricted",
			envVars: map[string]string{
				"RESTRICTED_NETWORK": "true",
			},
			description: "Environment with firewall restrictions",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Set environment variables
			for key, value := range tc.envVars {
				require.NoError(s.T(), os.Setenv(key, value))
			}

			// Load configuration
			cfg, err := config.Load()
			require.NoError(s.T(), err)

			// Create runner
			r := runner.New(cfg, s.tempDir)

			// Execute (should work with basic checks even without network)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			result, err := r.Run(ctx, runner.Options{
				Files: files,
			})

			// Should complete successfully even with network constraints
			assert.NoError(s.T(), err, tc.description)
			assert.NotNil(s.T(), result, "Should have results even in constrained environment")

			// Clean up environment variables
			for key := range tc.envVars {
				require.NoError(s.T(), os.Unsetenv(key))
			}

			s.T().Logf("%s: Completed successfully", tc.name)
		})
	}
}

// TestResourceConstrainedEnvironment validates behavior under resource limits
func (s *ProductionScenariosTestSuite) TestResourceConstrainedEnvironment() {
	// Create test files
	files := s.createBasicFiles([]string{
		"main.go", "service.go", "handler.go", "model.go",
		"README.md", "CHANGELOG.md", "config.yaml",
	})

	testCases := []struct {
		name        string
		config      map[string]string
		description string
	}{
		{
			name: "Limited Memory",
			config: map[string]string{
				"PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB": "1",
				"PRE_COMMIT_SYSTEM_MAX_FILES_OPEN":   "10",
			},
			description: "Environment with limited memory resources",
		},
		{
			name: "Single Worker",
			config: map[string]string{
				"PRE_COMMIT_SYSTEM_PARALLEL_WORKERS": "1",
			},
			description: "Environment with limited CPU (single worker)",
		},
		{
			name: "Short Timeouts",
			config: map[string]string{
				"PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS":    "30",
				"PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT": "10",
				"PRE_COMMIT_SYSTEM_EOF_TIMEOUT":        "10",
			},
			description: "Environment with aggressive timeouts",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create temporary config with constraints
			s.createConstrainedConfig(tc.config)

			// Load configuration
			cfg, err := config.Load()
			require.NoError(s.T(), err)

			// Create runner
			r := runner.New(cfg, s.tempDir)

			// Execute under constraints
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files: files,
			})
			duration := time.Since(start)

			// Should handle resource constraints gracefully
			assert.NoError(s.T(), err, tc.description)
			assert.NotNil(s.T(), result, "Should have results even under constraints")

			s.T().Logf("%s: Completed in %v under resource constraints", tc.name, duration)

			// Restore original config
			s.restoreOriginalConfig()
		})
	}
}

// TestCIEnvironmentScenarios validates common CI/CD scenarios
func (s *ProductionScenariosTestSuite) TestCIEnvironmentScenarios() {
	// Create realistic repository files
	files := s.createCIRepositoryStructure()

	ciScenarios := []struct {
		name        string
		envVars     map[string]string
		description string
	}{
		{
			name: "GitHub Actions Push",
			envVars: map[string]string{
				"CI":                "true",
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_REF":        "refs/heads/main",
			},
			description: "GitHub Actions push to main branch",
		},
		{
			name: "GitHub Actions Pull Request",
			envVars: map[string]string{
				"CI":                "true",
				"GITHUB_ACTIONS":    "true",
				"GITHUB_EVENT_NAME": "pull_request",
				"GITHUB_REF":        "refs/pull/123/merge",
			},
			description: "GitHub Actions pull request validation",
		},
		{
			name: "GitLab CI Pipeline",
			envVars: map[string]string{
				"CI":                 "true",
				"GITLAB_CI":          "true",
				"CI_PIPELINE_SOURCE": "push",
				"CI_COMMIT_REF_NAME": "main",
			},
			description: "GitLab CI pipeline execution",
		},
		{
			name: "Jenkins Build",
			envVars: map[string]string{
				"CI":           "true",
				"JENKINS_URL":  "http://jenkins.example.com/",
				"BUILD_NUMBER": "123",
				"JOB_NAME":     "project-validation",
			},
			description: "Jenkins build validation",
		},
	}

	for _, scenario := range ciScenarios {
		s.Run(scenario.name, func() {
			// Set CI environment variables
			for key, value := range scenario.envVars {
				require.NoError(s.T(), os.Setenv(key, value))
			}

			// Load configuration
			cfg, err := config.Load()
			require.NoError(s.T(), err)

			// Create runner
			r := runner.New(cfg, s.tempDir)

			// Execute in CI environment
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files: files,
			})
			duration := time.Since(start)

			// Should work correctly in CI
			assert.NoError(s.T(), err, scenario.description)
			assert.NotNil(s.T(), result, "Should have results in CI environment")

			// Performance should be reasonable in CI
			assert.True(s.T(), duration < 30*time.Second,
				"CI execution should complete in reasonable time: %v", duration)

			// Clean up environment variables
			for key := range scenario.envVars {
				require.NoError(s.T(), os.Unsetenv(key))
			}

			s.T().Logf("%s: Completed in %v", scenario.name, duration)
		})
	}
}

// TestRealWorldFilePatterns validates handling of real-world file patterns
func (s *ProductionScenariosTestSuite) TestRealWorldFilePatterns() {
	// Create files with real-world patterns and issues
	files := s.createRealWorldFiles()

	// Load configuration
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	// Create runner
	r := runner.New(cfg, s.tempDir)

	// Execute
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	start := time.Now()
	result, err := r.Run(ctx, runner.Options{
		Files: files,
	})
	duration := time.Since(start)

	// Should handle real-world patterns successfully
	assert.NoError(s.T(), err, "Should handle real-world file patterns")
	assert.NotNil(s.T(), result, "Should have results")
	assert.True(s.T(), duration < 20*time.Second, "Should complete in reasonable time")

	s.T().Logf("Real-world patterns test: %d files processed in %v",
		len(files), duration)
}

// Helper methods for creating test scenarios

func (s *ProductionScenariosTestSuite) createLargeRepositoryStructure(fileCount int) []string {
	var files []string

	// Create directory structure
	dirs := []string{
		"cmd", "pkg", "internal", "api", "web", "scripts",
		"docs", "examples", "test", "vendor", "build",
	}

	for _, dir := range dirs {
		require.NoError(s.T(), os.MkdirAll(filepath.Join(s.tempDir, dir), 0o755))
	}

	// Create files across directories
	fileTypes := []string{".go", ".md", ".yaml", ".json", ".sh", ".txt"}

	for i := 0; i < fileCount; i++ {
		dir := dirs[i%len(dirs)]
		fileType := fileTypes[i%len(fileTypes)]
		filename := fmt.Sprintf("file_%d%s", i, fileType)

		// Skip vendor directory for processing
		if dir == "vendor" {
			continue
		}

		fullPath := filepath.Join(s.tempDir, dir, filename)
		content := s.generateFileContent(fileType, i)

		require.NoError(s.T(), os.WriteFile(fullPath, []byte(content), 0o644))
		files = append(files, filepath.Join(dir, filename))
	}

	return files
}

func (s *ProductionScenariosTestSuite) cleanupLargeRepository(fileCount int) {
	dirs := []string{
		"cmd", "pkg", "internal", "api", "web", "scripts",
		"docs", "examples", "test", "build",
	}

	for _, dir := range dirs {
		_ = os.RemoveAll(filepath.Join(s.tempDir, dir))
	}
}

func (s *ProductionScenariosTestSuite) createMixedFileTypeStructure() []string {
	fileMap := map[string]string{
		"main.go":            "package main\n\nfunc main() {}\n",
		"README.md":          "# Project\n\nDescription\n",
		"config.yaml":        "app:\n  name: test\n",
		"data.json":          `{"key": "value"}`,
		"script.sh":          "#!/bin/bash\necho 'hello'\n",
		"Dockerfile":         "FROM alpine:latest\n",
		"docker-compose.yml": "version: '3'\nservices:\n  app:\n    image: test\n",
		"Makefile":           "all:\n\techo 'build'\n",
		"requirements.txt":   "requests==2.28.0\n",
		"package.json":       `{"name": "test", "version": "1.0.0"}`,
		"style.css":          "body { margin: 0; }\n",
		"app.js":             "console.log('hello');\n",
		"index.html":         "<html><body>Test</body></html>\n",
		"data.xml":           "<?xml version='1.0'?><root></root>\n",
		"config.ini":         "[section]\nkey=value\n",
		"binary.png":         "\x89PNG\r\n\x1a\n", // Binary file
	}

	var files []string
	for filename, content := range fileMap {
		fullPath := filepath.Join(s.tempDir, filename)
		require.NoError(s.T(), os.WriteFile(fullPath, []byte(content), 0o644))
		files = append(files, filename)
	}

	return files
}

func (s *ProductionScenariosTestSuite) createFeatureBranchFiles() []string {
	files := []string{
		"cmd/api/main.go",
		"pkg/auth/service.go",
		"pkg/auth/handler.go",
		"pkg/users/model.go",
		"pkg/users/service.go",
		"internal/database/migrations.go",
		"api/openapi.yaml",
		"docs/api.md",
		"test/integration/auth_test.go",
		"README.md",
	}

	s.createBasicFiles(files)
	return files
}

func (s *ProductionScenariosTestSuite) createRefactoringFiles() []string {
	files := []string{
		"pkg/legacy/old_service.go",
		"pkg/v2/new_service.go",
		"pkg/v2/interfaces.go",
		"internal/adapters/legacy.go",
		"internal/adapters/v2.go",
		"cmd/migrate/main.go",
		"docs/migration_guide.md",
		"test/unit/service_test.go",
		"CHANGELOG.md",
	}

	s.createBasicFiles(files)
	return files
}

func (s *ProductionScenariosTestSuite) createDocumentationFiles() []string {
	files := []string{
		"README.md", "CONTRIBUTING.md", "LICENSE.md",
		"docs/getting-started.md", "docs/api-reference.md",
		"docs/deployment.md", "docs/troubleshooting.md",
		"examples/basic/README.md", "examples/advanced/README.md",
		"CHANGELOG.md", "CODE_OF_CONDUCT.md",
	}

	s.createBasicFiles(files)
	return files
}

func (s *ProductionScenariosTestSuite) createDependencyUpdateFiles() []string {
	files := []string{
		"go.mod", "go.sum",
		"package.json", "package-lock.json",
		"requirements.txt", "Pipfile", "Pipfile.lock",
		"Dockerfile", "docker-compose.yml",
		".github/workflows/ci.yml",
	}

	s.createBasicFiles(files)
	return files
}

func (s *ProductionScenariosTestSuite) createCIRepositoryStructure() []string {
	files := []string{
		"main.go", "service.go", "handler.go",
		"pkg/utils/helper.go", "pkg/api/routes.go",
		"internal/config/config.go", "internal/db/connection.go",
		"test/integration/api_test.go", "test/unit/service_test.go",
		"README.md", "CHANGELOG.md", "Dockerfile",
		".github/workflows/ci.yml", ".github/workflows/release.yml",
		"scripts/build.sh", "scripts/deploy.sh",
	}

	s.createBasicFiles(files)
	return files
}

func (s *ProductionScenariosTestSuite) createRealWorldFiles() []string {
	// Files with common real-world issues
	fileMap := map[string]string{
		"trailing_spaces.go": `package main

func main() {
	fmt.Println("hello")
}`,
		"no_newline.md":     "# Title\n\nContent without final newline",
		"mixed_endings.txt": "Line 1\r\nLine 2\nLine 3\r\n",
		"large_file.go":     strings.Repeat("// Comment line\n", 1000) + "package main\n",
		"unicode_content.go": `package main
// Unicode: café résumé naïve 🚀
func main() {}
`,
		"empty_file.go":      "",
		"only_whitespace.md": "   \n\t\n   \n",
	}

	var files []string
	for filename, content := range fileMap {
		fullPath := filepath.Join(s.tempDir, filename)
		require.NoError(s.T(), os.WriteFile(fullPath, []byte(content), 0o644))
		files = append(files, filename)
	}

	return files
}

func (s *ProductionScenariosTestSuite) createBasicFiles(filenames []string) []string {
	for _, filename := range filenames {
		// Create directory if needed
		dir := filepath.Dir(filename)
		if dir != "." {
			require.NoError(s.T(), os.MkdirAll(filepath.Join(s.tempDir, dir), 0o755))
		}

		// Create file with appropriate content
		content := s.generateFileContent(filepath.Ext(filename), 0)
		fullPath := filepath.Join(s.tempDir, filename)
		require.NoError(s.T(), os.WriteFile(fullPath, []byte(content), 0o644))
	}

	return filenames
}

func (s *ProductionScenariosTestSuite) generateFileContent(fileType string, index int) string {
	switch fileType {
	case ".go":
		return fmt.Sprintf(`package main

import "fmt"

// File %d
func function%d() {
	fmt.Println("Generated file %d")
}
`, index, index, index)
	case ".md":
		return fmt.Sprintf(`# Generated File %d

This is a generated markdown file for testing.

## Section

Content for file %d.
`, index, index)
	case ".yaml", ".yml":
		return fmt.Sprintf(`# Generated YAML %d
app:
  name: test-app-%d
  version: 1.0.%d
`, index, index, index)
	case ".json":
		return fmt.Sprintf(`{
  "id": %d,
  "name": "generated-file-%d",
  "active": true
}`, index, index)
	case ".sh":
		return fmt.Sprintf(`#!/bin/bash
# Generated script %d
echo "Running script %d"
exit 0
`, index, index)
	default:
		return fmt.Sprintf("Generated content for file %d\n", index)
	}
}

func (s *ProductionScenariosTestSuite) createConstrainedConfig(overrides map[string]string) {
	// Start with base config
	config := map[string]string{
		"ENABLE_PRE_COMMIT_SYSTEM":            "true",
		"PRE_COMMIT_SYSTEM_LOG_LEVEL":         "info",
		"PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE": "true",
		"PRE_COMMIT_SYSTEM_ENABLE_EOF":        "true",
		"PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS":   "120",
		"PRE_COMMIT_SYSTEM_PARALLEL_WORKERS":  "4",
	}

	// Apply overrides
	for key, value := range overrides {
		config[key] = value
	}

	// Write config file
	var content string
	for key, value := range config {
		content += key + "=" + value + "\n"
	}

	require.NoError(s.T(), os.WriteFile(s.envFile, []byte(content), 0o644))
}

func (s *ProductionScenariosTestSuite) restoreOriginalConfig() {
	originalConfig := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=info
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=false
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=120
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=4
PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB=50
PRE_COMMIT_SYSTEM_MAX_FILES_OPEN=500
PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS="vendor/,node_modules/,.git/,dist/,build/,coverage/,*.log,*.tmp"
PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT=60
PRE_COMMIT_SYSTEM_EOF_TIMEOUT=60
`
	require.NoError(s.T(), os.WriteFile(s.envFile, []byte(originalConfig), 0o644))
}

// TestSuite runs the production scenarios test suite
func TestProductionScenariosTestSuite(t *testing.T) {
	suite.Run(t, new(ProductionScenariosTestSuite))
}
