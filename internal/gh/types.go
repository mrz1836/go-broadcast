// Package gh provides GitHub API client interfaces and types
package gh

import (
	"fmt"
	"time"
)

// Branch represents a GitHub branch
type Branch struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	Commit    struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
}

// PR represents a GitHub pull request
type PR struct {
	Number         int    `json:"number"`
	State          string `json:"state"` // open, closed
	Title          string `json:"title"`
	Body           string `json:"body"`
	Draft          bool   `json:"draft"`           // true if PR is a draft
	Mergeable      *bool  `json:"mergeable"`       // nil if unknown, true if mergeable, false if not
	MergeableState string `json:"mergeable_state"` // "clean", "blocked", "unstable", "behind", "draft", "unknown"
	Head           struct {
		Ref string `json:"ref"` // branch name
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"` // target branch
		SHA string `json:"sha"`
	} `json:"base"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	MergedAt  *time.Time `json:"merged_at"`
	AutoMerge *AutoMerge `json:"auto_merge"` // nil if auto-merge is not enabled
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	// Repo stores the repository in "owner/repo" format.
	// This is populated by SearchAssignedPRs for cross-repository operations.
	// Not serialized to JSON as it's derived from the PR URL.
	Repo string `json:"-"`
}

// PRRequest represents a request to create a pull request
type PRRequest struct {
	Title         string   `json:"title"`
	Body          string   `json:"body"`
	Head          string   `json:"head"`                     // source branch
	Base          string   `json:"base"`                     // target branch
	Labels        []string `json:"labels,omitempty"`         // Labels to apply to PR
	Assignees     []string `json:"assignees,omitempty"`      // GitHub usernames to assign
	Reviewers     []string `json:"reviewers,omitempty"`      // GitHub usernames to request reviews from
	TeamReviewers []string `json:"team_reviewers,omitempty"` // GitHub team slugs to request reviews from
}

// PRUpdate represents updates to an existing pull request
type PRUpdate struct {
	State *string `json:"state,omitempty"` // "open" or "closed"
	Body  *string `json:"body,omitempty"`  // Updated body content
}

// Commit represents a GitHub commit
type Commit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
	Parents []struct {
		SHA string `json:"sha"`
	} `json:"parents"`
}

// File represents a file in a GitHub repository
type File struct {
	Path     string `json:"path"`
	Mode     string `json:"mode"`
	Type     string `json:"type"`
	SHA      string `json:"sha"`
	Size     int    `json:"size"`
	URL      string `json:"url"`
	Content  string `json:"content"`  // base64 encoded
	Encoding string `json:"encoding"` // usually "base64"
}

// FileContent represents decoded file content
type FileContent struct {
	Path    string `json:"path"`
	Content []byte `json:"content"`
	SHA     string `json:"sha"`
}

// User represents a GitHub user
type User struct {
	Login string `json:"login"`
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GitTreeNode represents a node in the GitHub Git tree
type GitTreeNode struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"` // "blob", "tree", "commit"
	SHA  string `json:"sha"`
	Size *int   `json:"size,omitempty"`
	URL  string `json:"url,omitempty"`
}

// GitTree represents the GitHub Git tree response
type GitTree struct {
	SHA       string        `json:"sha"`
	URL       string        `json:"url"`
	Tree      []GitTreeNode `json:"tree"`
	Truncated bool          `json:"truncated"`
}

// Repository represents a GitHub repository with settings
type Repository struct {
	Name             string `json:"name"`
	FullName         string `json:"full_name"`
	DefaultBranch    string `json:"default_branch"`
	AllowSquashMerge bool   `json:"allow_squash_merge"`
	AllowMergeCommit bool   `json:"allow_merge_commit"`
	AllowRebaseMerge bool   `json:"allow_rebase_merge"`
}

// MergeMethod represents the type of merge to perform
type MergeMethod string

const (
	// MergeMethodMerge creates a merge commit
	MergeMethodMerge MergeMethod = "merge"
	// MergeMethodSquash squashes all commits into one
	MergeMethodSquash MergeMethod = "squash"
	// MergeMethodRebase rebases and merges
	MergeMethodRebase MergeMethod = "rebase"
)

// String returns the string representation of MergeMethod
func (m MergeMethod) String() string {
	return string(m)
}

// IsValid returns true if the MergeMethod is a valid, recognized value
func (m MergeMethod) IsValid() bool {
	switch m {
	case MergeMethodMerge, MergeMethodSquash, MergeMethodRebase:
		return true
	default:
		return false
	}
}

// Review represents a GitHub pull request review
type Review struct {
	ID          int        `json:"id"`
	User        User       `json:"user"`
	State       string     `json:"state"` // "APPROVED", "CHANGES_REQUESTED", "COMMENTED", "DISMISSED", "PENDING"
	Body        string     `json:"body"`
	SubmittedAt *time.Time `json:"submitted_at"`
}

// AutoMerge represents auto-merge configuration for a pull request
type AutoMerge struct {
	EnabledBy   User        `json:"enabled_by"`
	MergeMethod MergeMethod `json:"merge_method"`
	CommitTitle string      `json:"commit_title,omitempty"`
}

// CheckRun represents a GitHub check run
type CheckRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`     // "queued", "in_progress", "completed"
	Conclusion string `json:"conclusion"` // "success", "failure", "neutral", "canceled", "skipped", "timed_out", "action_required"
}

// CheckRunsResponse represents the response from GitHub's check-runs API
type CheckRunsResponse struct {
	TotalCount int        `json:"total_count"`
	CheckRuns  []CheckRun `json:"check_runs"`
}

// CheckStatusSummary provides a summary of all check runs for a commit
type CheckStatusSummary struct {
	Total     int        // Total number of check runs
	Completed int        // Checks that have completed (any conclusion)
	Passed    int        // success + neutral
	Skipped   int        // skipped (CI-skipped checks)
	Failed    int        // failure + canceled + timed_out + action_required
	Running   int        // queued + in_progress
	Checks    []CheckRun // All check runs for detailed output
}

// HasRunningChecks returns true if any checks are still running
func (s *CheckStatusSummary) HasRunningChecks() bool {
	return s.Running > 0
}

// HasFailedChecks returns true if any checks have failed
func (s *CheckStatusSummary) HasFailedChecks() bool {
	return s.Failed > 0
}

// AllPassed returns true if all checks are complete and none failed
func (s *CheckStatusSummary) AllPassed() bool {
	return s.Total > 0 && s.Running == 0 && s.Failed == 0
}

// NoChecks returns true if there are no check runs configured
func (s *CheckStatusSummary) NoChecks() bool {
	return s.Total == 0
}

// RunningCheckNames returns the names of all running checks
func (s *CheckStatusSummary) RunningCheckNames() []string {
	var names []string
	for _, check := range s.Checks {
		if check.Status == "queued" || check.Status == "in_progress" {
			names = append(names, check.Name)
		}
	}
	return names
}

// FailedCheckNames returns the names of all failed checks
func (s *CheckStatusSummary) FailedCheckNames() []string {
	var names []string
	for _, check := range s.Checks {
		if check.Status == "completed" {
			switch check.Conclusion {
			case "failure", "canceled", "timed_out", "action_required":
				names = append(names, check.Name)
			}
		}
	}
	return names
}

// Summary returns a human-readable summary string
func (s *CheckStatusSummary) Summary() string {
	if s.Total == 0 {
		return "no checks configured"
	}

	parts := []string{}
	if s.Passed > 0 {
		parts = append(parts, fmt.Sprintf("%d passed", s.Passed))
	}
	if s.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", s.Skipped))
	}
	if s.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", s.Failed))
	}
	if s.Running > 0 {
		parts = append(parts, fmt.Sprintf("%d running", s.Running))
	}

	detail := ""
	if len(parts) > 0 {
		detail = " (" + joinStrings(parts, ", ") + ")"
	}

	return fmt.Sprintf("%d/%d checks complete%s", s.Completed, s.Total, detail)
}

// joinStrings joins strings with a separator (avoiding strings import in types)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// RepoInfo represents basic repository information from discovery
type RepoInfo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Description   *string   `json:"description"`
	Language      *string   `json:"language"`
	Private       bool      `json:"private"`
	Fork          bool      `json:"fork"`
	Archived      bool      `json:"archived"`
	DefaultBranch string    `json:"default_branch"`
	HTMLURL       string    `json:"html_url"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DependabotAlert represents a Dependabot security alert
type DependabotAlert struct {
	Number                int    `json:"number"`
	State                 string `json:"state"` // open, dismissed, fixed
	DependencyPackage     string `json:"-"`     // Extracted from dependency.package.name
	DependencyManifest    string `json:"-"`     // Extracted from dependency.manifest_path
	SecurityVulnerability struct {
		Package struct {
			Ecosystem string `json:"ecosystem"`
			Name      string `json:"name"`
		} `json:"package"`
		Severity               string `json:"severity"` // low, medium, high, critical
		VulnerableVersionRange string `json:"vulnerable_version_range"`
		FirstPatchedVersion    *struct {
			Identifier string `json:"identifier"`
		} `json:"first_patched_version"`
	} `json:"security_vulnerability"`
	Dependency struct {
		Package struct {
			Name string `json:"name"`
		} `json:"package"`
		ManifestPath string `json:"manifest_path"`
	} `json:"dependency"`
	HTMLURL     string     `json:"html_url"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DismissedAt *time.Time `json:"dismissed_at"`
	FixedAt     *time.Time `json:"fixed_at"`
}

// CodeScanningAlert represents a code scanning security alert
type CodeScanningAlert struct {
	Number int    `json:"number"`
	State  string `json:"state"` // open, dismissed, fixed
	Rule   struct {
		ID          string `json:"id"`
		Severity    string `json:"severity"` // Severity levels: info, warning, error
		Description string `json:"description"`
	} `json:"rule"`
	Tool struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"tool"`
	MostRecentInstance struct {
		Ref      string `json:"ref"`
		Location struct {
			Path      string `json:"path"`
			StartLine int    `json:"start_line"`
			EndLine   int    `json:"end_line"`
		} `json:"location"`
		Message struct {
			Text string `json:"text"`
		} `json:"message"`
	} `json:"most_recent_instance"`
	HTMLURL     string     `json:"html_url"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DismissedAt *time.Time `json:"dismissed_at"`
	FixedAt     *time.Time `json:"fixed_at"`
}

// Workflow represents a GitHub Actions workflow
type Workflow struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"` // active, disabled_manually, etc.
}

// WorkflowsResponse represents the response from the workflows API
type WorkflowsResponse struct {
	TotalCount int        `json:"total_count"`
	Workflows  []Workflow `json:"workflows"`
}

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`     // completed, in_progress, queued
	Conclusion string `json:"conclusion"` // success, failure, canceled, skipped
	HeadBranch string `json:"head_branch"`
	HeadSHA    string `json:"head_sha"`
	RunNumber  int    `json:"run_number"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// WorkflowRunsResponse represents the response from the workflow runs API
type WorkflowRunsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

// Artifact represents a GitHub Actions workflow artifact
type Artifact struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	SizeInBytes        int64  `json:"size_in_bytes"`
	ArchiveDownloadURL string `json:"archive_download_url"`
	Expired            bool   `json:"expired"`
	CreatedAt          string `json:"created_at"`
	ExpiresAt          string `json:"expires_at"`
}

// ArtifactsResponse represents the response from the artifacts API
type ArtifactsResponse struct {
	TotalCount int        `json:"total_count"`
	Artifacts  []Artifact `json:"artifacts"`
}

// SecretScanningAlert represents a secret scanning alert
type SecretScanningAlert struct {
	Number                int        `json:"number"`
	State                 string     `json:"state"` // open, resolved
	SecretType            string     `json:"secret_type"`
	SecretTypeDisplayName string     `json:"secret_type_display_name"`
	Secret                string     `json:"secret"`
	Resolution            *string    `json:"resolution"` // false_positive, wont_fix, revoked, used_in_tests
	ResolvedBy            *User      `json:"resolved_by"`
	ResolvedAt            *time.Time `json:"resolved_at"`
	HTMLURL               string     `json:"html_url"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             *time.Time `json:"updated_at"`
}
