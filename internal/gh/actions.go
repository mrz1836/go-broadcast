package gh

import (
	"context"
	"fmt"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/jsonutil"
)

// ListWorkflows returns all workflows for a repository
func (g *githubClient) ListWorkflows(ctx context.Context, repo string) ([]Workflow, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/actions/workflows", repo))
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("%w: %s", ErrRepositoryNotFound, repo)
		}
		return nil, appErrors.WrapWithContext(err, "list workflows")
	}

	response, err := jsonutil.UnmarshalJSON[WorkflowsResponse](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse workflows response")
	}

	return response.Workflows, nil
}

// GetWorkflowRuns returns recent successful runs for a specific workflow
func (g *githubClient) GetWorkflowRuns(ctx context.Context, repo string, workflowID int64, count int) ([]WorkflowRun, error) {
	if count <= 0 {
		count = 1
	}

	apiURL := fmt.Sprintf("repos/%s/actions/workflows/%d/runs?status=success&per_page=%d", repo, workflowID, count)
	output, err := g.runner.Run(ctx, "gh", "api", apiURL)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "get workflow runs")
	}

	response, err := jsonutil.UnmarshalJSON[WorkflowRunsResponse](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse workflow runs response")
	}

	return response.WorkflowRuns, nil
}

// GetRunArtifacts returns all artifacts for a workflow run
func (g *githubClient) GetRunArtifacts(ctx context.Context, repo string, runID int64) ([]Artifact, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/actions/runs/%d/artifacts", repo, runID))
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "get run artifacts")
	}

	response, err := jsonutil.UnmarshalJSON[ArtifactsResponse](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse artifacts response")
	}

	return response.Artifacts, nil
}

// DownloadRunArtifact downloads a named artifact from a workflow run to the specified directory.
// Uses `gh run download` which handles zip extraction natively.
func (g *githubClient) DownloadRunArtifact(ctx context.Context, repo string, runID int64, artifactName, destDir string) error {
	_, err := g.runner.Run(ctx, "gh", "run", "download",
		fmt.Sprintf("%d", runID),
		"--repo", repo,
		"--name", artifactName,
		"--dir", destDir,
	)
	if err != nil {
		return appErrors.WrapWithContext(err, fmt.Sprintf("download artifact %q from run %d", artifactName, runID))
	}

	return nil
}
