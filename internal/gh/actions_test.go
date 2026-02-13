package gh

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
)

func TestListWorkflows(t *testing.T) {
	ctx := context.Background()

	t.Run("successful listing", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		response := WorkflowsResponse{
			TotalCount: 2,
			Workflows: []Workflow{
				{ID: 1, Name: "GoFortress", Path: ".github/workflows/fortress.yml", State: "active"},
				{ID: 2, Name: "CI", Path: ".github/workflows/ci.yml", State: "active"},
			},
		}

		output, err := json.Marshal(response)
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/actions/workflows"}).
			Return(output, nil)

		workflows, err := client.ListWorkflows(ctx, "owner/repo")
		require.NoError(t, err)
		assert.Len(t, workflows, 2)
		assert.Equal(t, "GoFortress", workflows[0].Name)
		assert.Equal(t, int64(1), workflows[0].ID)
		assert.Equal(t, "CI", workflows[1].Name)

		mockRunner.AssertExpectations(t)
	})

	t.Run("repository not found", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/missing/actions/workflows"}).
			Return(nil, ErrRepositoryNotFound)

		_, err := client.ListWorkflows(ctx, "owner/missing")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRepositoryNotFound)

		mockRunner.AssertExpectations(t)
	})

	t.Run("API error", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/actions/workflows"}).
			Return(nil, internalerrors.ErrTest)

		_, err := client.ListWorkflows(ctx, "owner/repo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "list workflows")

		mockRunner.AssertExpectations(t)
	})
}

func TestGetWorkflowRuns(t *testing.T) {
	ctx := context.Background()

	t.Run("successful run retrieval", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		response := WorkflowRunsResponse{
			TotalCount: 1,
			WorkflowRuns: []WorkflowRun{
				{
					ID:         100,
					Name:       "GoFortress",
					Status:     "completed",
					Conclusion: "success",
					HeadBranch: "main",
					HeadSHA:    "abc123",
					RunNumber:  42,
				},
			},
		}

		output, err := json.Marshal(response)
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/actions/workflows/1/runs?status=success&per_page=3"}).
			Return(output, nil)

		runs, err := client.GetWorkflowRuns(ctx, "owner/repo", 1, 3)
		require.NoError(t, err)
		assert.Len(t, runs, 1)
		assert.Equal(t, int64(100), runs[0].ID)
		assert.Equal(t, "main", runs[0].HeadBranch)
		assert.Equal(t, "abc123", runs[0].HeadSHA)

		mockRunner.AssertExpectations(t)
	})

	t.Run("defaults count to 1 when zero", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		response := WorkflowRunsResponse{WorkflowRuns: []WorkflowRun{}}
		output, err := json.Marshal(response)
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/actions/workflows/5/runs?status=success&per_page=1"}).
			Return(output, nil)

		runs, err := client.GetWorkflowRuns(ctx, "owner/repo", 5, 0)
		require.NoError(t, err)
		assert.Empty(t, runs)

		mockRunner.AssertExpectations(t)
	})

	t.Run("API error", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/actions/workflows/1/runs?status=success&per_page=1"}).
			Return(nil, internalerrors.ErrTest)

		_, err := client.GetWorkflowRuns(ctx, "owner/repo", 1, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get workflow runs")

		mockRunner.AssertExpectations(t)
	})
}

func TestGetRunArtifacts(t *testing.T) {
	ctx := context.Background()

	t.Run("successful artifact listing", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		response := ArtifactsResponse{
			TotalCount: 3,
			Artifacts: []Artifact{
				{ID: 10, Name: "loc-stats", SizeInBytes: 256, Expired: false},
				{ID: 11, Name: "coverage-stats-codecov", SizeInBytes: 512, Expired: false},
				{ID: 12, Name: "bench-stats-unit", SizeInBytes: 128, Expired: false},
			},
		}

		output, err := json.Marshal(response)
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/actions/runs/100/artifacts"}).
			Return(output, nil)

		artifacts, err := client.GetRunArtifacts(ctx, "owner/repo", 100)
		require.NoError(t, err)
		assert.Len(t, artifacts, 3)
		assert.Equal(t, "loc-stats", artifacts[0].Name)
		assert.Equal(t, int64(256), artifacts[0].SizeInBytes)
		assert.False(t, artifacts[0].Expired)

		mockRunner.AssertExpectations(t)
	})

	t.Run("empty artifacts", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		response := ArtifactsResponse{TotalCount: 0, Artifacts: []Artifact{}}
		output, err := json.Marshal(response)
		require.NoError(t, err)

		mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/actions/runs/99/artifacts"}).
			Return(output, nil)

		artifacts, err := client.GetRunArtifacts(ctx, "owner/repo", 99)
		require.NoError(t, err)
		assert.Empty(t, artifacts)

		mockRunner.AssertExpectations(t)
	})

	t.Run("API error", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/actions/runs/100/artifacts"}).
			Return(nil, internalerrors.ErrTest)

		_, err := client.GetRunArtifacts(ctx, "owner/repo", 100)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get run artifacts")

		mockRunner.AssertExpectations(t)
	})
}

func TestDownloadRunArtifact(t *testing.T) {
	ctx := context.Background()

	t.Run("successful download", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{"run", "download", "100", "--repo", "owner/repo", "--name", "loc-stats", "--dir", "/tmp/artifacts"}).
			Return([]byte(""), nil)

		err := client.DownloadRunArtifact(ctx, "owner/repo", 100, "loc-stats", "/tmp/artifacts")
		require.NoError(t, err)

		mockRunner.AssertExpectations(t)
	})

	t.Run("download failure", func(t *testing.T) {
		mockRunner := NewMockCommandRunner()
		client := NewClientWithRunner(mockRunner, logrus.New())

		mockRunner.On("Run", ctx, "gh", []string{"run", "download", "100", "--repo", "owner/repo", "--name", "missing-artifact", "--dir", "/tmp/artifacts"}).
			Return(nil, internalerrors.ErrTest)

		err := client.DownloadRunArtifact(ctx, "owner/repo", 100, "missing-artifact", "/tmp/artifacts")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `download artifact "missing-artifact" from run 100`)

		mockRunner.AssertExpectations(t)
	})
}
