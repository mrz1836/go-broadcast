package cli

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
)

var (
	// ErrEmptyPRURL is returned when an empty URL is provided
	ErrEmptyPRURL = errors.New("empty URL provided")
	// ErrInvalidPRNumber is returned when PR number cannot be parsed
	ErrInvalidPRNumber = errors.New("invalid PR number")
	// ErrInvalidPRURLFormat is returned when URL format is invalid
	ErrInvalidPRURLFormat = errors.New("invalid PR URL format")
	// ErrNoValidPRURLs is returned when no valid PR URLs are provided
	ErrNoValidPRURLs = errors.New("no valid PR URLs provided")
	// ErrMutuallyExclusiveFlags is returned when both --all-assigned-prs flag and explicit URLs are provided
	ErrMutuallyExclusiveFlags = errors.New("cannot use --all-assigned-prs flag with explicit PR URLs")
	// ErrNoAssignedPRs is returned when no assigned PRs are found
	ErrNoAssignedPRs = errors.New("no assigned PRs found")
)

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var reviewPRCmd = createReviewPRCmd(globalFlags)

// PRInfo contains parsed information from a PR URL
type PRInfo struct {
	Owner  string
	Repo   string
	Number int
	URL    string
}

// parsePRURL parses a GitHub PR URL into owner, repo, and PR number
// Supported formats:
//   - https://github.com/owner/repo/pull/123
//   - http://github.com/owner/repo/pull/123
//   - github.com/owner/repo/pull/123
//   - owner/repo#123
func parsePRURL(url string) (*PRInfo, error) {
	// Remove any leading/trailing whitespace
	url = strings.TrimSpace(url)

	if url == "" {
		return nil, ErrEmptyPRURL
	}

	// Pattern for full GitHub URLs
	// Matches: https://github.com/owner/repo/pull/123
	fullURLPattern := regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/]+)/pull/(\d+)`)
	matches := fullURLPattern.FindStringSubmatch(url)
	if len(matches) == 4 {
		number, err := strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidPRNumber, matches[3])
		}
		return &PRInfo{
			Owner:  matches[1],
			Repo:   matches[2],
			Number: number,
			URL:    url,
		}, nil
	}

	// Pattern for short format: owner/repo#123
	shortPattern := regexp.MustCompile(`^([^/]+)/([^#]+)#(\d+)$`)
	matches = shortPattern.FindStringSubmatch(url)
	if len(matches) == 4 {
		number, err := strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidPRNumber, matches[3])
		}
		return &PRInfo{
			Owner:  matches[1],
			Repo:   matches[2],
			Number: number,
			URL:    url,
		}, nil
	}

	return nil, fmt.Errorf("%w: %s (expected formats: 'https://github.com/owner/repo/pull/123' or 'owner/repo#123')", ErrInvalidPRURLFormat, url)
}

// ReviewPRResult contains the result of a PR review and merge operation
type ReviewPRResult struct {
	PRInfo                  PRInfo `json:"pr_info"`
	Reviewed                bool   `json:"reviewed"`
	AlreadyReviewed         bool   `json:"already_reviewed,omitempty"`           // True if already reviewed by current user
	Merged                  bool   `json:"merged"`                               // True if merged immediately
	AutoMergeEnabled        bool   `json:"auto_merge_enabled"`                   // True if auto-merge was enabled
	AutoMergeAlreadyEnabled bool   `json:"auto_merge_already_enabled,omitempty"` // True if auto-merge was already enabled
	MergeMethod             string `json:"merge_method,omitempty"`
	Error                   string `json:"error,omitempty"`
	AlreadyMerged           bool   `json:"already_merged,omitempty"`
}

// createReviewPRCmd creates the review-pr command for isolated testing
func createReviewPRCmd(flags *Flags) *cobra.Command {
	var message string
	var allAssignedPRs bool

	cmd := &cobra.Command{
		Use:   "review-pr [<pr-url> [pr-url...]]",
		Short: "Review and merge pull requests",
		Long: `Review and merge one or more pull requests from GitHub URLs or all assigned PRs.

This command will:
1. Parse the PR URL(s) to extract owner/repo/number (or fetch all assigned PRs if --all-assigned-prs is used)
2. Submit an approving review with the specified message
3. Detect the repository's preferred merge method
4. Merge the PR using the detected method

The command supports both single and batch operations, processing multiple PRs in sequence.`,
		Example: `  # Review and merge a single PR
  go-broadcast review-pr https://github.com/owner/repo/pull/123

  # Review and merge multiple PRs
  go-broadcast review-pr https://github.com/owner/repo/pull/123 https://github.com/owner/repo/pull/124

  # Use short format
  go-broadcast review-pr owner/repo#123

  # Review and merge all PRs assigned to you
  go-broadcast review-pr --all-assigned-prs

  # Customize the review message
  go-broadcast review-pr --message "Approved after testing" https://github.com/owner/repo/pull/123

  # Preview without executing
  go-broadcast review-pr --dry-run https://github.com/owner/repo/pull/123

  # Review all assigned PRs with custom message
  go-broadcast review-pr --all-assigned-prs --message "LGTM" --dry-run`,
		Args: cobra.ArbitraryArgs, // Allow 0 or more args since --all-assigned-prs doesn't need URLs
		RunE: createRunReviewPR(flags, &message, &allAssignedPRs),
	}

	cmd.Flags().StringVarP(&message, "message", "m", "LGTM", "Review approval message")
	cmd.Flags().BoolVar(&allAssignedPRs, "all-assigned-prs", false, "Review and merge all open PRs assigned to you (excludes drafts)")

	return cmd
}

// createRunReviewPR creates the run function for the review-pr command
func createRunReviewPR(flags *Flags, message *string, allAssignedPRs *bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		log := logrus.WithField("command", "review-pr")

		// Initialize GitHub client
		client, err := gh.NewClient(ctx, log.Logger, nil)
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}

		var prInfos []*PRInfo

		// Check for mutually exclusive options
		if *allAssignedPRs && len(args) > 0 {
			return ErrMutuallyExclusiveFlags
		}

		if *allAssignedPRs {
			// Fetch all assigned PRs
			output.Info("Fetching all PRs assigned to you...")
			prs, err := client.SearchAssignedPRs(ctx)
			if err != nil {
				return fmt.Errorf("failed to search assigned PRs: %w", err)
			}

			if len(prs) == 0 {
				output.Warn("No assigned PRs found")
				return ErrNoAssignedPRs
			}

			output.Info(fmt.Sprintf("Found %d assigned PR(s) (draft PRs filtered out)", len(prs)))

			// Convert PRs to PRInfo structs
			for _, pr := range prs {
				// Extract repository from pr.Head.SHA (we stored "owner/repo" there)
				repo := pr.Head.SHA
				parts := strings.Split(repo, "/")
				if len(parts) != 2 {
					output.Warn(fmt.Sprintf("Skipping PR #%d: invalid repository format", pr.Number))
					continue
				}

				url := fmt.Sprintf("https://github.com/%s/pull/%d", repo, pr.Number)
				prInfos = append(prInfos, &PRInfo{
					Owner:  parts[0],
					Repo:   parts[1],
					Number: pr.Number,
					URL:    url,
				})
			}
		} else {
			// Parse all PR URLs from arguments
			if len(args) == 0 {
				return ErrNoValidPRURLs
			}

			for _, url := range args {
				info, err := parsePRURL(url)
				if err != nil {
					output.Error(fmt.Sprintf("Failed to parse URL '%s': %v", url, err))
					return fmt.Errorf("invalid PR URL: %w", err)
				}
				prInfos = append(prInfos, info)
			}
		}

		if len(prInfos) == 0 {
			return ErrNoValidPRURLs
		}

		// Process each PR
		results := make([]ReviewPRResult, 0, len(prInfos))
		successCount := 0
		failureCount := 0
		immediatelyMergedCount := 0
		autoMergeCount := 0

		for i, prInfo := range prInfos {
			if len(prInfos) > 1 {
				output.Info(fmt.Sprintf("\n[%d/%d] Processing PR #%d in %s/%s", i+1, len(prInfos), prInfo.Number, prInfo.Owner, prInfo.Repo))
			}

			result := ReviewPRResult{
				PRInfo: *prInfo,
			}

			// Get PR details to check if already merged
			pr, err := client.GetPR(ctx, fmt.Sprintf("%s/%s", prInfo.Owner, prInfo.Repo), prInfo.Number)
			if err != nil {
				result.Error = fmt.Sprintf("Failed to fetch PR: %v", err)
				output.Error(result.Error)
				results = append(results, result) //nolint:staticcheck // results used in summary
				failureCount++
				continue
			}

			// Check if already merged
			if pr.MergedAt != nil {
				result.AlreadyMerged = true
				result.Error = "PR is already merged"
				output.Warn(fmt.Sprintf("PR #%d is already merged", prInfo.Number))
				results = append(results, result) //nolint:staticcheck // results used in summary
				successCount++                    // Consider this a success since the PR is merged
				continue
			}

			// Check if PR is closed (but not merged)
			if pr.State == "closed" {
				result.Error = "PR is closed but not merged"
				output.Error(result.Error)
				results = append(results, result) //nolint:staticcheck // results used in summary
				failureCount++
				continue
			}

			// Get current user for review check
			currentUser, err := client.GetCurrentUser(ctx)
			if err != nil {
				result.Error = fmt.Sprintf("Failed to get current user: %v", err)
				output.Error(result.Error)
				results = append(results, result) //nolint:staticcheck // results used in summary
				failureCount++
				continue
			}

			// Check if user has already approved this PR
			alreadyApproved, err := client.HasApprovedReview(ctx, fmt.Sprintf("%s/%s", prInfo.Owner, prInfo.Repo), prInfo.Number, currentUser.Login)
			if err != nil {
				result.Error = fmt.Sprintf("Failed to check existing reviews: %v", err)
				output.Error(result.Error)
				results = append(results, result) //nolint:staticcheck // results used in summary
				failureCount++
				continue
			}

			// Check if auto-merge is already enabled
			autoMergeAlreadyEnabled := pr.AutoMerge != nil

			// If already reviewed and auto-merge is already enabled (or PR is already merged), skip
			if alreadyApproved && (autoMergeAlreadyEnabled || pr.MergedAt != nil) {
				result.AlreadyReviewed = true
				result.AutoMergeAlreadyEnabled = autoMergeAlreadyEnabled
				output.Info(fmt.Sprintf("✓ PR #%d already reviewed and %s", prInfo.Number,
					func() string {
						if autoMergeAlreadyEnabled {
							return "auto-merge enabled"
						}
						return "ready to merge"
					}()))
				results = append(results, result) //nolint:staticcheck // results used in summary
				successCount++
				continue
			}

			if flags.DryRun {
				output.Info(fmt.Sprintf("DRY RUN: Would review PR #%d with message: %s", prInfo.Number, *message))
				output.Info(fmt.Sprintf("DRY RUN: Would detect merge method for %s/%s", prInfo.Owner, prInfo.Repo))
				output.Info(fmt.Sprintf("DRY RUN: Would merge PR #%d", prInfo.Number))
				result.Reviewed = false
				result.Merged = false
				results = append(results, result) //nolint:staticcheck // results used in summary
				successCount++
				continue
			}

			// Submit review (skip if already approved)
			if alreadyApproved {
				result.AlreadyReviewed = true
				output.Info(fmt.Sprintf("✓ PR #%d already reviewed by you", prInfo.Number))
			} else {
				output.Info(fmt.Sprintf("Submitting approval for PR #%d...", prInfo.Number))
				err = client.ReviewPR(ctx, fmt.Sprintf("%s/%s", prInfo.Owner, prInfo.Repo), prInfo.Number, *message)
				if err != nil {
					result.Error = fmt.Sprintf("Failed to review PR: %v", err)
					output.Error(result.Error)
					results = append(results, result) //nolint:staticcheck // results used in summary
					failureCount++
					continue
				}
				result.Reviewed = true
				output.Success(fmt.Sprintf("✓ PR #%d approved", prInfo.Number))
			}

			// Get repository settings to determine merge method
			output.Info(fmt.Sprintf("Detecting merge method for %s/%s...", prInfo.Owner, prInfo.Repo))
			repo, err := client.GetRepository(ctx, fmt.Sprintf("%s/%s", prInfo.Owner, prInfo.Repo))
			if err != nil {
				result.Error = fmt.Sprintf("Failed to get repository settings: %v", err)
				output.Error(result.Error)
				results = append(results, result) //nolint:staticcheck // results used in summary
				failureCount++
				continue
			}

			// Determine merge method based on repository settings
			var mergeMethod gh.MergeMethod
			if repo.AllowSquashMerge {
				mergeMethod = gh.MergeMethodSquash
			} else if repo.AllowMergeCommit {
				mergeMethod = gh.MergeMethodMerge
			} else if repo.AllowRebaseMerge {
				mergeMethod = gh.MergeMethodRebase
			} else {
				// Default to squash if no method is explicitly allowed
				mergeMethod = gh.MergeMethodSquash
			}
			result.MergeMethod = mergeMethod.String()
			output.Info(fmt.Sprintf("Using merge method: %s", mergeMethod))

			// Smart merge strategy: try-and-fallback approach
			repoFullName := fmt.Sprintf("%s/%s", prInfo.Owner, prInfo.Repo)

			// If PR has merge conflicts, skip straight to auto-merge
			if pr.Mergeable != nil && !*pr.Mergeable {
				if autoMergeAlreadyEnabled {
					result.AutoMergeAlreadyEnabled = true
					output.Info(fmt.Sprintf("✓ Auto-merge already enabled for PR #%d", prInfo.Number))
				} else {
					output.Warn(fmt.Sprintf("⚠️  PR #%d has merge conflicts - enabling auto-merge for when conflicts are resolved", prInfo.Number))
					output.Info(fmt.Sprintf("Enabling auto-merge for PR #%d...", prInfo.Number))
					err = client.EnableAutoMergePR(ctx, repoFullName, prInfo.Number, mergeMethod)
					if err != nil {
						result.Error = fmt.Sprintf("Failed to enable auto-merge: %v", err)
						output.Error(result.Error)
						results = append(results, result) //nolint:staticcheck // results used in summary
						failureCount++
						continue
					}
					result.AutoMergeEnabled = true
					autoMergeCount++
					output.Success(fmt.Sprintf("✓ Auto-merge enabled for PR #%d - will merge when conflicts are resolved", prInfo.Number))
				}
			} else {
				// Try immediate merge first (optimistic approach)
				output.Info(fmt.Sprintf("Merging PR #%d...", prInfo.Number))
				err = client.MergePR(ctx, repoFullName, prInfo.Number, mergeMethod)
				if err != nil {
					// Check if error is due to branch protection policies
					if gh.IsBranchProtectionError(err) {
						// Fallback to auto-merge
						if autoMergeAlreadyEnabled {
							result.AutoMergeAlreadyEnabled = true
							output.Info(fmt.Sprintf("✓ Auto-merge already enabled for PR #%d", prInfo.Number))
						} else {
							output.Warn(fmt.Sprintf("⚠️  Branch protection blocking merge for PR #%d - enabling auto-merge", prInfo.Number))
							output.Info(fmt.Sprintf("Enabling auto-merge for PR #%d...", prInfo.Number))
							err = client.EnableAutoMergePR(ctx, repoFullName, prInfo.Number, mergeMethod)
							if err != nil {
								result.Error = fmt.Sprintf("Failed to enable auto-merge: %v", err)
								output.Error(result.Error)
								results = append(results, result) //nolint:staticcheck // results used in summary
								failureCount++
								continue
							}
							result.AutoMergeEnabled = true
							autoMergeCount++
							output.Success(fmt.Sprintf("✓ Auto-merge enabled for PR #%d - will merge when requirements are met", prInfo.Number))
						}
					} else {
						// Real error - fail
						result.Error = fmt.Sprintf("Failed to merge PR: %v", err)
						output.Error(result.Error)
						results = append(results, result) //nolint:staticcheck // results used in summary
						failureCount++
						continue
					}
				} else {
					// Merge succeeded immediately
					result.Merged = true
					immediatelyMergedCount++
					output.Success(fmt.Sprintf("✓ PR #%d merged immediately using %s method", prInfo.Number, mergeMethod))
				}
			}

			results = append(results, result) //nolint:staticcheck // results used in summary
			successCount++
		}

		// Print summary for batch operations
		if len(prInfos) > 1 {
			output.Info("\n=== Summary ===")
			output.Info(fmt.Sprintf("Total PRs: %d", len(prInfos)))
			if immediatelyMergedCount > 0 {
				output.Success(fmt.Sprintf("Merged immediately: %d", immediatelyMergedCount))
			}
			if autoMergeCount > 0 {
				output.Success(fmt.Sprintf("Auto-merge enabled: %d", autoMergeCount))
			}
			if failureCount > 0 {
				output.Error(fmt.Sprintf("Failed: %d", failureCount))
			}
		}

		// Return error if any PR failed
		if failureCount > 0 {
			return fmt.Errorf("%d PR(s) failed to process", failureCount) //nolint:err113 // Dynamic count in error message
		}

		return nil
	}
}
