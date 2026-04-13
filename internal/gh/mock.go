package gh

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

// NewMockClient creates a new MockClient
func NewMockClient() *MockClient {
	return &MockClient{}
}

// ListBranches mock implementation
func (m *MockClient) ListBranches(ctx context.Context, repo string) ([]Branch, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[[]Branch](args)
}

// GetBranch mock implementation
func (m *MockClient) GetBranch(ctx context.Context, repo, branch string) (*Branch, error) {
	args := m.Called(ctx, repo, branch)
	return testutil.HandleTwoValueReturn[*Branch](args)
}

// CreatePR mock implementation
func (m *MockClient) CreatePR(ctx context.Context, repo string, req PRRequest) (*PR, error) {
	args := m.Called(ctx, repo, req)
	return testutil.HandleTwoValueReturn[*PR](args)
}

// GetPR mock implementation
func (m *MockClient) GetPR(ctx context.Context, repo string, number int) (*PR, error) {
	args := m.Called(ctx, repo, number)
	return testutil.HandleTwoValueReturn[*PR](args)
}

// ListPRs mock implementation
func (m *MockClient) ListPRs(ctx context.Context, repo, state string) ([]PR, error) {
	args := m.Called(ctx, repo, state)
	return testutil.HandleTwoValueReturn[[]PR](args)
}

// GetFile mock implementation
func (m *MockClient) GetFile(ctx context.Context, repo, path, ref string) (*FileContent, error) {
	args := m.Called(ctx, repo, path, ref)
	return testutil.HandleTwoValueReturn[*FileContent](args)
}

// GetCommit mock implementation
func (m *MockClient) GetCommit(ctx context.Context, repo, sha string) (*Commit, error) {
	args := m.Called(ctx, repo, sha)
	return testutil.HandleTwoValueReturn[*Commit](args)
}

// ClosePR mock implementation
func (m *MockClient) ClosePR(ctx context.Context, repo string, number int, comment string) error {
	args := m.Called(ctx, repo, number, comment)
	return args.Error(0)
}

// DeleteBranch mock implementation
func (m *MockClient) DeleteBranch(ctx context.Context, repo, branch string) error {
	args := m.Called(ctx, repo, branch)
	return args.Error(0)
}

// UpdatePR mock implementation
func (m *MockClient) UpdatePR(ctx context.Context, repo string, number int, updates PRUpdate) error {
	args := m.Called(ctx, repo, number, updates)
	return args.Error(0)
}

// GetCurrentUser mock implementation
func (m *MockClient) GetCurrentUser(ctx context.Context) (*User, error) {
	args := m.Called(ctx)
	return testutil.HandleTwoValueReturn[*User](args)
}

// GetGitTree mock implementation
func (m *MockClient) GetGitTree(ctx context.Context, repo, treeSHA string, recursive bool) (*GitTree, error) {
	args := m.Called(ctx, repo, treeSHA, recursive)
	return testutil.HandleTwoValueReturn[*GitTree](args)
}

// GetRepository mock implementation
func (m *MockClient) GetRepository(ctx context.Context, repo string) (*Repository, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[*Repository](args)
}

// ReviewPR mock implementation
func (m *MockClient) ReviewPR(ctx context.Context, repo string, number int, message string) error {
	args := m.Called(ctx, repo, number, message)
	return args.Error(0)
}

// MergePR mock implementation
func (m *MockClient) MergePR(ctx context.Context, repo string, number int, method MergeMethod) error {
	args := m.Called(ctx, repo, number, method)
	return args.Error(0)
}

// BypassMergePR mock implementation
func (m *MockClient) BypassMergePR(ctx context.Context, repo string, number int, method MergeMethod) error {
	args := m.Called(ctx, repo, number, method)
	return args.Error(0)
}

// EnableAutoMergePR mock implementation
func (m *MockClient) EnableAutoMergePR(ctx context.Context, repo string, number int, method MergeMethod) error {
	args := m.Called(ctx, repo, number, method)
	return args.Error(0)
}

// SearchAssignedPRs mock implementation
func (m *MockClient) SearchAssignedPRs(ctx context.Context) ([]PR, error) {
	args := m.Called(ctx)
	return testutil.HandleTwoValueReturn[[]PR](args)
}

// GetPRReviews mock implementation
func (m *MockClient) GetPRReviews(ctx context.Context, repo string, number int) ([]Review, error) {
	args := m.Called(ctx, repo, number)
	return testutil.HandleTwoValueReturn[[]Review](args)
}

// HasApprovedReview mock implementation
func (m *MockClient) HasApprovedReview(ctx context.Context, repo string, number int, username string) (bool, error) {
	args := m.Called(ctx, repo, number, username)
	return args.Bool(0), args.Error(1)
}

// AddPRComment mock implementation
func (m *MockClient) AddPRComment(ctx context.Context, repo string, number int, comment string) error {
	args := m.Called(ctx, repo, number, comment)
	return args.Error(0)
}

// GetPRCheckStatus mock implementation
func (m *MockClient) GetPRCheckStatus(ctx context.Context, repo string, number int) (*CheckStatusSummary, error) {
	args := m.Called(ctx, repo, number)
	return testutil.HandleTwoValueReturn[*CheckStatusSummary](args)
}

// DiscoverOrgRepos mock implementation
func (m *MockClient) DiscoverOrgRepos(ctx context.Context, org string) ([]RepoInfo, error) {
	args := m.Called(ctx, org)
	return testutil.HandleTwoValueReturn[[]RepoInfo](args)
}

// ExecuteGraphQL mock implementation
func (m *MockClient) ExecuteGraphQL(ctx context.Context, query string) (map[string]interface{}, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// GetDependabotAlerts mock implementation
func (m *MockClient) GetDependabotAlerts(ctx context.Context, repo string) ([]DependabotAlert, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[[]DependabotAlert](args)
}

// GetCodeScanningAlerts mock implementation
func (m *MockClient) GetCodeScanningAlerts(ctx context.Context, repo string) ([]CodeScanningAlert, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[[]CodeScanningAlert](args)
}

// GetSecretScanningAlerts mock implementation
func (m *MockClient) GetSecretScanningAlerts(ctx context.Context, repo string) ([]SecretScanningAlert, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[[]SecretScanningAlert](args)
}

// GetVulnerabilityAlertsGraphQL mock implementation
func (m *MockClient) GetVulnerabilityAlertsGraphQL(ctx context.Context, repo string) ([]VulnerabilityAlert, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[[]VulnerabilityAlert](args)
}

// ListWorkflows mock implementation
func (m *MockClient) ListWorkflows(ctx context.Context, repo string) ([]Workflow, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[[]Workflow](args)
}

// GetWorkflowRuns mock implementation
func (m *MockClient) GetWorkflowRuns(ctx context.Context, repo string, workflowID int64, count int) ([]WorkflowRun, error) {
	args := m.Called(ctx, repo, workflowID, count)
	return testutil.HandleTwoValueReturn[[]WorkflowRun](args)
}

// GetRunArtifacts mock implementation
func (m *MockClient) GetRunArtifacts(ctx context.Context, repo string, runID int64) ([]Artifact, error) {
	args := m.Called(ctx, repo, runID)
	return testutil.HandleTwoValueReturn[[]Artifact](args)
}

// DownloadRunArtifact mock implementation
func (m *MockClient) DownloadRunArtifact(ctx context.Context, repo string, runID int64, artifactName, destDir string) error {
	args := m.Called(ctx, repo, runID, artifactName, destDir)
	return args.Error(0)
}

// GetRateLimit mock implementation
func (m *MockClient) GetRateLimit(ctx context.Context) (*RateLimitResponse, error) {
	args := m.Called(ctx)
	return testutil.HandleTwoValueReturn[*RateLimitResponse](args)
}

// GetContributorCount mock implementation
func (m *MockClient) GetContributorCount(ctx context.Context, repo string) (int, error) {
	args := m.Called(ctx, repo)
	return args.Int(0), args.Error(1)
}

// CreateRepository mock implementation
func (m *MockClient) CreateRepository(ctx context.Context, opts CreateRepoOptions) (*Repository, error) {
	args := m.Called(ctx, opts)
	return testutil.HandleTwoValueReturn[*Repository](args)
}

// UpdateRepoSettings mock implementation
func (m *MockClient) UpdateRepoSettings(ctx context.Context, repo string, settings RepoSettings) error {
	args := m.Called(ctx, repo, settings)
	return args.Error(0)
}

// GetRepoSettings mock implementation
func (m *MockClient) GetRepoSettings(ctx context.Context, repo string) (*RepoSettings, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[*RepoSettings](args)
}

// CreateOrUpdateRuleset mock implementation
func (m *MockClient) CreateOrUpdateRuleset(ctx context.Context, repo string, ruleset Ruleset) error {
	args := m.Called(ctx, repo, ruleset)
	return args.Error(0)
}

// ListRulesets mock implementation
func (m *MockClient) ListRulesets(ctx context.Context, repo string) ([]Ruleset, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[[]Ruleset](args)
}

// SyncLabels mock implementation
func (m *MockClient) SyncLabels(ctx context.Context, repo string, labels []Label) error {
	args := m.Called(ctx, repo, labels)
	return args.Error(0)
}

// ListLabels mock implementation
func (m *MockClient) ListLabels(ctx context.Context, repo string) ([]Label, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[[]Label](args)
}

// SetTopics mock implementation
func (m *MockClient) SetTopics(ctx context.Context, repo string, topics []string) error {
	args := m.Called(ctx, repo, topics)
	return args.Error(0)
}

// CloneRepository mock implementation
func (m *MockClient) CloneRepository(ctx context.Context, repo string, destPath string) error {
	args := m.Called(ctx, repo, destPath)
	return args.Error(0)
}

// CreateFileCommit mock implementation
func (m *MockClient) CreateFileCommit(ctx context.Context, repo, path, message string, content []byte, branch string) error {
	args := m.Called(ctx, repo, path, message, content, branch)
	return args.Error(0)
}

// RenameBranch mock implementation
func (m *MockClient) RenameBranch(ctx context.Context, repo, oldName, newName string) error {
	args := m.Called(ctx, repo, oldName, newName)
	return args.Error(0)
}
