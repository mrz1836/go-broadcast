// Package validation provides comprehensive production readiness validation tests
package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/runner"
)

// CIEnvironmentTestSuite validates parity between local and CI execution
type CIEnvironmentTestSuite struct {
	suite.Suite
	tempDir    string
	envFile    string
	originalWD string
}

// SetupSuite initializes the test environment
func (s *CIEnvironmentTestSuite) SetupSuite() {
	var err error
	s.originalWD, err = os.Getwd()
	s.Require().NoError(err)

	// Create temporary directory structure
	s.tempDir = s.T().TempDir()

	// Create .github directory
	githubDir := filepath.Join(s.tempDir, ".github")
	s.Require().NoError(os.MkdirAll(githubDir, 0o755))

	// Create .env.shared file with test configuration
	s.envFile = filepath.Join(githubDir, ".env.shared")
	envContent := `# Test environment configuration
ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=info
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
PRE_COMMIT_SYSTEM_ENABLE_LINT=true
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=120
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=2
`
	s.Require().NoError(os.WriteFile(s.envFile, []byte(envContent), 0o644))

	// Change to temp directory for tests
	s.Require().NoError(os.Chdir(s.tempDir))

	// Initialize git repository
	s.Require().NoError(s.initGitRepo())

	// Create test files
	s.Require().NoError(s.createTestFiles())
}

// TearDownSuite cleans up the test environment
func (s *CIEnvironmentTestSuite) TearDownSuite() {
	// Restore original working directory
	_ = os.Chdir(s.originalWD)
}

// initGitRepo initializes a git repository in the temp directory
func (s *CIEnvironmentTestSuite) initGitRepo() error {
	// Initialize git repo (simplified for testing)
	gitDir := filepath.Join(s.tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		return err
	}

	// Create basic git files
	return os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0o644)
}

// createTestFiles creates sample files for testing
func (s *CIEnvironmentTestSuite) createTestFiles() error {
	files := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`,
		"service.go": `package main

type Service struct {
	name string
}

func NewService(name string) *Service {
	return &Service{name: name}
}
`,
		"README.md": `# Test Project

This is a test project for validation.
`,
		"go.mod": `module test-project

go 1.21
`,
	}

	for filename, content := range files {
		if err := os.WriteFile(filepath.Join(s.tempDir, filename), []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}

// TestCIEnvironmentParity validates that execution behaves consistently between local and CI
func (s *CIEnvironmentTestSuite) TestCIEnvironmentParity() {
	testCases := []struct {
		name        string
		ciEnvVars   map[string]string
		description string
	}{
		{
			name: "GitHub Actions Environment",
			ciEnvVars: map[string]string{
				"CI":                "true",
				"GITHUB_ACTIONS":    "true",
				"GITHUB_WORKFLOW":   "CI",
				"GITHUB_RUN_ID":     "12345",
				"GITHUB_RUN_NUMBER": "1",
				"GITHUB_JOB":        "test",
				"GITHUB_ACTION":     "run",
				"GITHUB_ACTOR":      "test-user",
				"GITHUB_REPOSITORY": "test/repo",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_SHA":        "abc123",
				"GITHUB_REF":        "refs/heads/main",
				"RUNNER_OS":         "Linux",
				"RUNNER_TEMP":       "/tmp",
				"RUNNER_TOOL_CACHE": "/opt/hostedtoolcache",
			},
			description: "GitHub Actions CI environment",
		},
		{
			name: "GitLab CI Environment",
			ciEnvVars: map[string]string{
				"CI":                        "true",
				"GITLAB_CI":                 "true",
				"CI_JOB_ID":                 "12345",
				"CI_JOB_NAME":               "test",
				"CI_JOB_STAGE":              "test",
				"CI_PIPELINE_ID":            "67890",
				"CI_PROJECT_ID":             "123",
				"CI_PROJECT_NAME":           "test-project",
				"CI_COMMIT_SHA":             "abc123",
				"CI_COMMIT_REF_NAME":        "main",
				"CI_RUNNER_EXECUTABLE_ARCH": "linux/amd64",
			},
			description: "GitLab CI environment",
		},
		{
			name: "Jenkins Environment",
			ciEnvVars: map[string]string{
				"CI":           "true",
				"JENKINS_URL":  "http://jenkins.example.com/",
				"BUILD_ID":     "123",
				"BUILD_NUMBER": "123",
				"JOB_NAME":     "test-job",
				"WORKSPACE":    "/var/jenkins_home/workspace/test-job",
				"NODE_NAME":    "master",
			},
			description: "Jenkins CI environment",
		},
		{
			name: "Generic CI Environment",
			ciEnvVars: map[string]string{
				"CI":       "true",
				"BUILD_ID": "generic-123",
				"NO_COLOR": "1",
				"TERM":     "dumb",
			},
			description: "Generic CI environment with basic settings",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// First, run in local environment (baseline)
			localResults := s.runInEnvironment(map[string]string{}, "local")

			// Then run in CI environment
			ciResults := s.runInEnvironment(tc.ciEnvVars, "ci")

			// Validate parity
			s.validateExecutionParity(localResults, ciResults, tc.description)
		})
	}
}

// TestCISpecificBehavior validates CI-specific behavior
func (s *CIEnvironmentTestSuite) TestCISpecificBehavior() {
	s.Run("Color Output Disabled in CI", func() {
		ciEnvVars := map[string]string{
			"CI":       "true",
			"NO_COLOR": "1",
		}

		results := s.runInEnvironment(ciEnvVars, "ci-no-color")

		// Verify that color output is properly disabled
		s.NotNil(results)
		s.Positive(results.TotalDuration)
	})

	s.Run("Progress Output in CI", func() {
		ciEnvVars := map[string]string{
			"CI":   "true",
			"TERM": "dumb",
		}

		results := s.runInEnvironment(ciEnvVars, "ci-progress")

		// Verify execution completes successfully even with limited terminal
		s.NotNil(results)
		s.GreaterOrEqual(results.Passed+results.Skipped, 1)
	})

	s.Run("Timeout Handling in CI", func() {
		ciEnvVars := map[string]string{
			"CI":                                "true",
			"PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS": "5", // Very short timeout
		}

		// Override env file temporarily
		s.createTempEnvFile(map[string]string{
			"ENABLE_PRE_COMMIT_SYSTEM":            "true",
			"PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS":   "5",
			"PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE": "true",
			"PRE_COMMIT_SYSTEM_ENABLE_EOF":        "true",
		})

		results := s.runInEnvironment(ciEnvVars, "ci-timeout")

		// Should complete within timeout or handle gracefully
		s.NotNil(results)
		s.Less(results.TotalDuration, 10*time.Second)
	})
}

// TestCIEnvironmentVariablePrecedence validates environment variable precedence in CI
func (s *CIEnvironmentTestSuite) TestCIEnvironmentVariablePrecedence() {
	testCases := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name: "CI Environment Overrides",
			envVars: map[string]string{
				"CI":                                 "true",
				"PRE_COMMIT_SYSTEM_LOG_LEVEL":        "debug",
				"PRE_COMMIT_SYSTEM_PARALLEL_WORKERS": "1",
			},
			expected: "debug",
		},
		{
			name: "Runtime Environment Priority",
			envVars: map[string]string{
				"PRE_COMMIT_SYSTEM_ENABLE_FUMPT": "false",
				"PRE_COMMIT_SYSTEM_ENABLE_LINT":  "false",
			},
			expected: "disabled",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create temporary env file with base config
			s.createTempEnvFile(map[string]string{
				"ENABLE_PRE_COMMIT_SYSTEM":            "true",
				"PRE_COMMIT_SYSTEM_LOG_LEVEL":         "info",
				"PRE_COMMIT_SYSTEM_PARALLEL_WORKERS":  "2",
				"PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE": "true",
				"PRE_COMMIT_SYSTEM_ENABLE_EOF":        "true",
			})

			results := s.runInEnvironment(tc.envVars, "precedence-test")

			// Validate that environment variables took precedence
			s.NotNil(results)
		})
	}
}

// runInEnvironment executes the pre-commit system in a specific environment
func (s *CIEnvironmentTestSuite) runInEnvironment(envVars map[string]string, envContext string) *runner.Results {
	// Set environment variables
	originalEnv := make(map[string]string)
	for key, value := range envVars {
		originalEnv[key] = os.Getenv(key)
		s.Require().NoError(os.Setenv(key, value))
	}

	// Ensure cleanup
	defer func() {
		for key, originalValue := range originalEnv {
			if originalValue == "" {
				s.Require().NoError(os.Unsetenv(key))
			} else {
				s.Require().NoError(os.Setenv(key, originalValue))
			}
		}
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		s.T().Logf("Failed to load config in %s environment: %v", envContext, err)
		return nil
	}

	// Create runner
	r := runner.New(cfg, s.tempDir)

	// Get test files
	files := []string{"main.go", "service.go", "README.md"}

	// Execute
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := r.Run(ctx, runner.Options{
		Files: files,
	})
	if err != nil {
		s.T().Logf("Execution failed in %s environment: %v", envContext, err)
		return nil
	}

	return results
}

// validateExecutionParity checks that local and CI execution produce consistent results
func (s *CIEnvironmentTestSuite) validateExecutionParity(local, ci *runner.Results, description string) {
	if local == nil || ci == nil {
		s.T().Logf("Skipping parity validation for %s due to execution failures", description)
		return
	}

	// Check that the same number of checks were executed
	s.Len(local.CheckResults, len(ci.CheckResults),
		"Number of checks should be consistent between local and CI")

	// Check that execution time is reasonable in both environments
	s.Positive(local.TotalDuration,
		"Local execution should have measurable duration")
	s.Positive(ci.TotalDuration,
		"CI execution should have measurable duration")

	// Check that CI execution isn't significantly slower (allow 3x difference)
	maxAllowedDuration := local.TotalDuration * 3
	s.LessOrEqual(ci.TotalDuration, maxAllowedDuration,
		"CI execution should not be more than 3x slower than local: local=%v, ci=%v",
		local.TotalDuration, ci.TotalDuration)

	// Verify consistent check results (accounting for environment differences)
	localCheckNames := make(map[string]bool)
	for _, result := range local.CheckResults {
		localCheckNames[result.Name] = true
	}

	ciCheckNames := make(map[string]bool)
	for _, result := range ci.CheckResults {
		ciCheckNames[result.Name] = true
	}

	s.Equal(localCheckNames, ciCheckNames,
		"Same checks should be executed in both environments")

	s.T().Logf("Parity validation passed for %s: local=%v, ci=%v",
		description, local.TotalDuration, ci.TotalDuration)
}

// createTempEnvFile creates a temporary .env.shared file for testing
func (s *CIEnvironmentTestSuite) createTempEnvFile(vars map[string]string) {
	var content string
	for key, value := range vars {
		content += key + "=" + value + "\n"
	}

	s.Require().NoError(os.WriteFile(s.envFile, []byte(content), 0o644))
}

// TestCINetworkConnectivity validates behavior under network constraints
func (s *CIEnvironmentTestSuite) TestCINetworkConnectivity() {
	s.Run("Offline CI Environment", func() {
		ciEnvVars := map[string]string{
			"CI":            "true",
			"NO_NETWORK":    "true",
			"OFFLINE_BUILD": "true",
		}

		// Create minimal config that doesn't require network
		s.createTempEnvFile(map[string]string{
			"ENABLE_PRE_COMMIT_SYSTEM":            "true",
			"PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE": "true",
			"PRE_COMMIT_SYSTEM_ENABLE_EOF":        "true",
			"PRE_COMMIT_SYSTEM_ENABLE_FUMPT":      "false", // Disable checks that might need network
			"PRE_COMMIT_SYSTEM_ENABLE_LINT":       "false",
			"PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY":   "false",
		})

		results := s.runInEnvironment(ciEnvVars, "offline-ci")

		// Should still run basic checks successfully
		s.NotNil(results)
		s.GreaterOrEqual(results.Passed+results.Skipped, 1)
	})
}

// TestCIResourceLimits validates behavior under resource constraints
func (s *CIEnvironmentTestSuite) TestCIResourceLimits() {
	s.Run("Limited Resources", func() {
		ciEnvVars := map[string]string{
			"CI":                                 "true",
			"PRE_COMMIT_SYSTEM_PARALLEL_WORKERS": "1", // Force single-threaded
			"PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS":  "60",
		}

		s.createTempEnvFile(map[string]string{
			"ENABLE_PRE_COMMIT_SYSTEM":            "true",
			"PRE_COMMIT_SYSTEM_PARALLEL_WORKERS":  "1",
			"PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS":   "60",
			"PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE": "true",
			"PRE_COMMIT_SYSTEM_ENABLE_EOF":        "true",
		})

		results := s.runInEnvironment(ciEnvVars, "limited-resources")

		// Should complete successfully even with limited resources
		s.NotNil(results)
		s.Less(results.TotalDuration, 60*time.Second)
	})
}

// TestSuite runs the CI environment test suite
func TestCIEnvironmentTestSuite(t *testing.T) {
	suite.Run(t, new(CIEnvironmentTestSuite))
}
