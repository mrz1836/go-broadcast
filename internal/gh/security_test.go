package gh

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDependabotAlerts(t *testing.T) {
	ctx := context.Background()

	t.Run("successful fetch with alerts", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		now := time.Now()
		alerts := []DependabotAlert{
			{
				Number: 1,
				State:  "open",
				SecurityVulnerability: struct {
					Package struct {
						Ecosystem string `json:"ecosystem"`
						Name      string `json:"name"`
					} `json:"package"`
					Severity               string `json:"severity"`
					VulnerableVersionRange string `json:"vulnerable_version_range"`
					FirstPatchedVersion    *struct {
						Identifier string `json:"identifier"`
					} `json:"first_patched_version"`
				}{
					Package: struct {
						Ecosystem string `json:"ecosystem"`
						Name      string `json:"name"`
					}{
						Ecosystem: "npm",
						Name:      "lodash",
					},
					Severity:               "high",
					VulnerableVersionRange: "<4.17.21",
					FirstPatchedVersion: &struct {
						Identifier string `json:"identifier"`
					}{Identifier: "4.17.21"},
				},
				Dependency: struct {
					Package struct {
						Name string `json:"name"`
					} `json:"package"`
					ManifestPath string `json:"manifest_path"`
				}{
					Package: struct {
						Name string `json:"name"`
					}{Name: "lodash"},
					ManifestPath: "package.json",
				},
				HTMLURL:   "https://github.com/test/repo/security/dependabot/1",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		output, err := json.Marshal(alerts)
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/dependabot/alerts",
			"-F", "state=open",
			"-F", "per_page=100",
			"--paginate",
		}).Return(output, nil)

		result, err := client.GetDependabotAlerts(ctx, "test/repo")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, 1, result[0].Number)
		assert.Equal(t, "open", result[0].State)
		assert.Equal(t, "high", result[0].SecurityVulnerability.Severity)
		assert.Equal(t, "lodash", result[0].DependencyPackage)
		assert.Equal(t, "package.json", result[0].DependencyManifest)

		mockRunner.AssertExpectations(t)
	})

	t.Run("dependabot not enabled (404)", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/dependabot/alerts",
			"-F", "state=open",
			"-F", "per_page=100",
			"--paginate",
		}).Return(nil, &CommandError{Stderr: "404 Not Found"})

		result, err := client.GetDependabotAlerts(ctx, "test/repo")
		require.NoError(t, err)
		assert.Empty(t, result)

		mockRunner.AssertExpectations(t)
	})
}

func TestGetCodeScanningAlerts(t *testing.T) {
	ctx := context.Background()

	t.Run("successful fetch with alerts", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		now := time.Now()
		alerts := []CodeScanningAlert{
			{
				Number: 1,
				State:  "open",
				Rule: struct {
					ID          string `json:"id"`
					Severity    string `json:"severity"`
					Description string `json:"description"`
				}{
					ID:          "go/sql-injection",
					Severity:    "error",
					Description: "SQL injection vulnerability",
				},
				Tool: struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}{
					Name:    "CodeQL",
					Version: "2.11.0",
				},
				MostRecentInstance: struct {
					Ref      string `json:"ref"`
					Location struct {
						Path      string `json:"path"`
						StartLine int    `json:"start_line"`
						EndLine   int    `json:"end_line"`
					} `json:"location"`
					Message struct {
						Text string `json:"text"`
					} `json:"message"`
				}{
					Ref: "refs/heads/main",
					Location: struct {
						Path      string `json:"path"`
						StartLine int    `json:"start_line"`
						EndLine   int    `json:"end_line"`
					}{
						Path:      "internal/db/query.go",
						StartLine: 42,
						EndLine:   45,
					},
					Message: struct {
						Text string `json:"text"`
					}{
						Text: "Unsanitized user input flows into SQL query",
					},
				},
				HTMLURL:   "https://github.com/test/repo/security/code-scanning/1",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		output, err := json.Marshal(alerts)
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/code-scanning/alerts",
			"-F", "state=open",
			"-F", "per_page=100",
			"--paginate",
		}).Return(output, nil)

		result, err := client.GetCodeScanningAlerts(ctx, "test/repo")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, 1, result[0].Number)
		assert.Equal(t, "open", result[0].State)
		assert.Equal(t, "go/sql-injection", result[0].Rule.ID)
		assert.Equal(t, "error", result[0].Rule.Severity)

		mockRunner.AssertExpectations(t)
	})

	t.Run("code scanning not enabled (404)", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/code-scanning/alerts",
			"-F", "state=open",
			"-F", "per_page=100",
			"--paginate",
		}).Return(nil, &CommandError{Stderr: "404 Not Found"})

		result, err := client.GetCodeScanningAlerts(ctx, "test/repo")
		require.NoError(t, err)
		assert.Empty(t, result)

		mockRunner.AssertExpectations(t)
	})
}

func TestGetSecretScanningAlerts(t *testing.T) {
	ctx := context.Background()

	t.Run("successful fetch with alerts", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		now := time.Now()
		alerts := []SecretScanningAlert{
			{
				Number:                1,
				State:                 "open",
				SecretType:            "github_personal_access_token",
				SecretTypeDisplayName: "GitHub Personal Access Token",
				Secret:                "ghp_1234567890abcdefghijklmnopqrstuvwxyz",
				HTMLURL:               "https://github.com/test/repo/security/secret-scanning/1",
				CreatedAt:             now,
			},
		}

		output, err := json.Marshal(alerts)
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/secret-scanning/alerts",
			"-F", "state=open",
			"-F", "per_page=100",
			"--paginate",
		}).Return(output, nil)

		result, err := client.GetSecretScanningAlerts(ctx, "test/repo")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, 1, result[0].Number)
		assert.Equal(t, "open", result[0].State)
		assert.Equal(t, "github_personal_access_token", result[0].SecretType)

		mockRunner.AssertExpectations(t)
	})

	t.Run("secret scanning not enabled (404)", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/secret-scanning/alerts",
			"-F", "state=open",
			"-F", "per_page=100",
			"--paginate",
		}).Return(nil, &CommandError{Stderr: "404 Not Found"})

		result, err := client.GetSecretScanningAlerts(ctx, "test/repo")
		require.NoError(t, err)
		assert.Empty(t, result)

		mockRunner.AssertExpectations(t)
	})
}
