package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/test/helpers"
)

// TestGitRepositoryIntegration creates actual git repositories for testing
// This test fixes the Git repository setup issues in integration tests
func TestGitRepositoryIntegration(t *testing.T) {
	// Skip if git is not available
	helpers.SkipIfNoGit(t)

	t.Run("repository setup with real git operations", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create source repository with proper initialization
		sourceDir := filepath.Join(tmpDir, "source")
		err := os.MkdirAll(sourceDir, 0o750)
		require.NoError(t, err)

		// Initialize git repository with proper setup
		err = helpers.InitGitRepo(sourceDir, "Initial commit")
		require.NoError(t, err, "Failed to init git repo")

		// Create target repository
		targetDir := filepath.Join(tmpDir, "target")
		err = os.MkdirAll(targetDir, 0o750)
		require.NoError(t, err)

		err = helpers.InitGitRepo(targetDir, "Target initial commit")
		require.NoError(t, err, "Failed to init target git repo")

		// Test comprehensive git operations
		t.Run("basic git operations", func(t *testing.T) {
			// Add multiple files to source
			testFiles := map[string]string{
				"README.md":                "# Test Repository\n\nThis is a test.",
				"docs/guide.md":            "# Guide\n\nUser guide.",
				".github/workflows/ci.yml": "name: CI\non: [push]\njobs:\n  test:\n    runs-on: ubuntu-latest",
			}

			for filePath, content := range testFiles {
				fullPath := filepath.Join(sourceDir, filePath)
				dir := filepath.Dir(fullPath)
				if mkdirErr := os.MkdirAll(dir, 0o750); mkdirErr != nil {
					require.NoError(t, mkdirErr)
				}
				if writeErr := os.WriteFile(fullPath, []byte(content), 0o600); writeErr != nil {
					require.NoError(t, writeErr)
				}
			}

			// Commit changes
			err = helpers.CommitChanges(sourceDir, "Add initial files")
			require.NoError(t, err)

			// Verify commit exists
			commitSHA, err := helpers.GetLatestCommit(sourceDir)
			require.NoError(t, err)
			assert.Len(t, commitSHA, 40, "Should have valid commit SHA")

			// Test branch operations
			err = helpers.CreateBranch(sourceDir, "feature/test")
			require.NoError(t, err, "Failed to create branch")

			// Test file existence
			for filePath := range testFiles {
				fullPath := filepath.Join(sourceDir, filePath)
				exists, err := helpers.FileExists(fullPath)
				require.NoError(t, err)
				assert.True(t, exists, "File "+filePath+" should exist")
			}
		})
	})
}

// TestAdvancedConfigurationParsing tests complex configuration scenarios
func TestAdvancedConfigurationParsing(t *testing.T) {
	t.Run("multi group dependency resolution", func(t *testing.T) {
		// Create configuration with complex dependencies
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "advanced.yaml")

		configContent := `version: 1
groups:
  - name: "Foundation"
    id: "foundation"
    priority: 1
    source:
      repo: "org/foundation"
      branch: "master"
    targets:
      - repo: "org/base-lib"
        files:
          - src: "templates/base.md"
            dest: "README.md"
  - name: "Services"
    id: "services"
    priority: 2
    depends_on: ["foundation"]
    source:
      repo: "org/service-templates"
      branch: "master"
    targets:
      - repo: "org/api-service"
        files:
          - src: "service.yaml"
            dest: "config/service.yaml"
        transform:
          repo_name: true
          variables:
            SERVICE_TYPE: "api"
`
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		// Load and validate configuration
		cfg, err := config.Load(configPath)
		require.NoError(t, err)
		err = cfg.Validate()
		require.NoError(t, err)

		// Test group ordering and dependencies
		assert.Len(t, cfg.Groups, 2, "Should have 2 groups")

		// Verify dependencies
		for _, group := range cfg.Groups {
			if group.ID == "services" {
				assert.Contains(t, group.DependsOn, "foundation", "Services should depend on foundation")
			}
		}
	})

	t.Run("rate limiting configuration with many targets", func(t *testing.T) {
		// Test configuration that would trigger rate limiting scenarios
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "rate-limit-config.yaml")

		// Create configuration with many targets (could trigger rate limits)
		var configBuilder strings.Builder
		configBuilder.WriteString(`version: 1
groups:
  - name: "Rate Limit Test Group"
    id: "rate-limit-test"
    source:
      repo: "org/template-repo"
      branch: "master"
    targets:
`)

		// Add multiple targets to potentially trigger rate limiting
		for i := 1; i <= 5; i++ {
			configBuilder.WriteString("      - repo: \"org/target-repo-")
			configBuilder.WriteString(strings.Repeat("x", i))
			configBuilder.WriteString("\"\n        files:\n          - src: \"README.md\"\n            dest: \"README.md\"\n")
		}

		err := os.WriteFile(configPath, []byte(configBuilder.String()), 0o600)
		require.NoError(t, err)

		// Load and validate configuration
		cfg, err := config.Load(configPath)
		require.NoError(t, err)
		err = cfg.Validate()
		require.NoError(t, err)

		// Validate many targets configuration
		assert.Len(t, cfg.Groups, 1, "Should have 1 group")
		assert.Len(t, cfg.Groups[0].Targets, 5, "Should have 5 targets for rate limiting test")
	})
}
