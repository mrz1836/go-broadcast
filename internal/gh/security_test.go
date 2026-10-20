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

// genericSecretAlertsEndpoint is the single request that fetches every generic
// secret type for test/repo.
const genericSecretAlertsEndpoint = "repos/test/repo/secret-scanning/alerts?per_page=100" +
	"&secret_type=password,generic_api_key,generic_high_entropy_secret"

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
			"repos/test/repo/dependabot/alerts?per_page=100",
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

	t.Run("dependabot not enabled (404) returns ErrSecurityNotAvailable", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/dependabot/alerts?per_page=100",
			"--paginate",
		}).Return(nil, &CommandError{Stderr: "404 Not Found"})

		result, err := client.GetDependabotAlerts(ctx, "test/repo")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrSecurityNotAvailable)
		assert.Nil(t, result)

		mockRunner.AssertExpectations(t)
	})

	t.Run("empty alerts returns empty slice", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		output, err := json.Marshal([]DependabotAlert{})
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/dependabot/alerts?per_page=100",
			"--paginate",
		}).Return(output, nil)

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
			"repos/test/repo/code-scanning/alerts?per_page=100",
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

	t.Run("code scanning not enabled (404) returns ErrSecurityNotAvailable", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/code-scanning/alerts?per_page=100",
			"--paginate",
		}).Return(nil, &CommandError{Stderr: "404 Not Found"})

		result, err := client.GetCodeScanningAlerts(ctx, "test/repo")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrSecurityNotAvailable)
		assert.Nil(t, result)

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

		output, err := json.Marshal(alerts) //nolint:gosec // G117: test struct contains fake secret value for testing alert parsing
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/secret-scanning/alerts?per_page=100",
			"--paginate",
		}).Return(output, nil)

		// All generic secret types are requested together and return empty
		emptyJSON, _ := json.Marshal([]SecretScanningAlert{}) //nolint:gosec // G117: marshaling struct type with Secret field for test purposes
		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			genericSecretAlertsEndpoint,
			"--paginate",
		}).Return(emptyJSON, nil).Once()

		result, err := client.GetSecretScanningAlerts(ctx, "test/repo")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, 1, result[0].Number)
		assert.Equal(t, "open", result[0].State)
		assert.Equal(t, "github_personal_access_token", result[0].SecretType)

		mockRunner.AssertExpectations(t)
	})

	t.Run("generic alerts are merged without duplicates", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		defaultAlerts := []SecretScanningAlert{
			{Number: 1, State: "open", SecretType: "github_personal_access_token"},
			{Number: 2, State: "resolved", SecretType: "aws_access_key_id"},
		}
		genericAlerts := []SecretScanningAlert{
			{Number: 2, State: "resolved", SecretType: "aws_access_key_id"},
			{Number: 3, State: "open", SecretType: "password"},
			{Number: 4, State: "open", SecretType: "generic_high_entropy_secret"}, //nolint:gosec // G101: secret type name, not a credential
		}

		defaultJSON, err := json.Marshal(defaultAlerts) //nolint:gosec // G117: test struct has an empty Secret field
		require.NoError(t, err)
		genericJSON, err := json.Marshal(genericAlerts) //nolint:gosec // G117: test struct has an empty Secret field
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/secret-scanning/alerts?per_page=100",
			"--paginate",
		}).Return(defaultJSON, nil).Once()
		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			genericSecretAlertsEndpoint,
			"--paginate",
		}).Return(genericJSON, nil).Once()

		result, err := client.GetSecretScanningAlerts(ctx, "test/repo")
		require.NoError(t, err)

		numbers := make([]int, 0, len(result))
		for _, alert := range result {
			numbers = append(numbers, alert.Number)
		}
		assert.Equal(t, []int{1, 2, 3, 4}, numbers)

		mockRunner.AssertExpectations(t)
	})

	t.Run("combined generic request failure falls back to one request per type", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		mockRunner.Test(t)
		client := NewClientWithRunner(mockRunner, logrus.New())

		defaultJSON, err := json.Marshal([]SecretScanningAlert{{Number: 7, State: "open"}}) //nolint:gosec // G117: test struct has an empty Secret field
		require.NoError(t, err)
		passwordJSON, err := json.Marshal([]SecretScanningAlert{{Number: 8, State: "open", SecretType: "password"}}) //nolint:gosec // G117: test struct has an empty Secret field
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/secret-scanning/alerts?per_page=100",
			"--paginate",
		}).Return(defaultJSON, nil).Once()
		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			genericSecretAlertsEndpoint,
			"--paginate",
		}).Return(nil, &CommandError{Stderr: "HTTP 422: Validation Failed"}).Once()

		// Each type is then requested on its own; one succeeds, one fails, one is unavailable.
		perType := map[string]struct {
			output []byte
			err    error
		}{
			"password":                    {output: passwordJSON},
			"generic_api_key":             {err: &CommandError{Stderr: "HTTP 500: Server Error"}},
			"generic_high_entropy_secret": {err: &CommandError{Stderr: "HTTP 404: Not Found"}},
		}
		for secretType, resp := range perType {
			mockRunner.On("Run", ctx, "gh", []string{
				"api",
				"repos/test/repo/secret-scanning/alerts?per_page=100&secret_type=" + secretType,
				"--paginate",
			}).Return(resp.output, resp.err).Once()
		}

		result, err := client.GetSecretScanningAlerts(ctx, "test/repo")
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, 7, result[0].Number)
		assert.Equal(t, 8, result[1].Number)

		mockRunner.AssertExpectations(t)
	})

	t.Run("unavailable generic alerts are not retried per type", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		mockRunner.Test(t) // Any per-type request fails the test
		client := NewClientWithRunner(mockRunner, logrus.New())

		defaultJSON, err := json.Marshal([]SecretScanningAlert{{Number: 7, State: "open"}}) //nolint:gosec // G117: test struct has an empty Secret field
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/secret-scanning/alerts?per_page=100",
			"--paginate",
		}).Return(defaultJSON, nil).Once()
		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			genericSecretAlertsEndpoint,
			"--paginate",
		}).Return(nil, &CommandError{Stderr: "HTTP 404: Not Found"}).Once()

		result, err := client.GetSecretScanningAlerts(ctx, "test/repo")
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, 7, result[0].Number)

		mockRunner.AssertExpectations(t)
	})

	t.Run("secret scanning not enabled (404) returns ErrSecurityNotAvailable", func(t *testing.T) {
		mockRunner := new(MockCommandRunner)
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{
			"api",
			"repos/test/repo/secret-scanning/alerts?per_page=100",
			"--paginate",
		}).Return(nil, &CommandError{Stderr: "404 Not Found"})

		result, err := client.GetSecretScanningAlerts(ctx, "test/repo")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrSecurityNotAvailable)
		assert.Nil(t, result)

		mockRunner.AssertExpectations(t)
	})
}

func TestErrSecurityNotAvailable(t *testing.T) {
	t.Run("sentinel error is distinguishable", func(t *testing.T) {
		err := ErrSecurityNotAvailable
		require.ErrorIs(t, err, ErrSecurityNotAvailable)
		assert.Contains(t, err.Error(), "security feature not available")
	})
}
