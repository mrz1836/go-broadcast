package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// maxURLLength is the maximum allowed URL length to prevent ReDoS attacks
const maxURLLength = 2048

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
	// ErrNoDependabotPRs is returned when no Dependabot PRs assigned to the user are found
	ErrNoDependabotPRs = errors.New("no assigned Dependabot PRs found")
	// ErrDependabotRequiresAllAssigned is returned when --dependabot is used without --all-assigned-prs
	ErrDependabotRequiresAllAssigned = errors.New("--dependabot requires --all-assigned-prs")
	// ErrURLTooLong is returned when URL exceeds maximum allowed length
	ErrURLTooLong = errors.New("URL exceeds maximum length")
)

// dependabotAuthor is the GitHub search author filter used to match Dependabot-authored PRs.
const dependabotAuthor = "app/dependabot"

// ciGateDecision describes the outcome of a CI status check for a PR.
type ciGateDecision int

const (
	// ciGateProceed indicates all checks passed (or there are no checks) and merging may proceed.
	ciGateProceed ciGateDecision = iota
	// ciGateSkipRunning indicates one or more checks are still running.
	ciGateSkipRunning
	// ciGateSkipFailed indicates one or more checks have failed.
	ciGateSkipFailed
	// ciGateUnknown indicates we could not fetch check status (permission issue, network, etc.).
	// Caller may choose to proceed optimistically.
	ciGateUnknown
)

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var reviewPRCmd = createReviewPRCmd(globalFlags)

// newReviewPRClient constructs the GitHub client used by the review-pr command.
// It is a package-level variable so tests can inject a mock client without
// spinning up the real gh CLI.
//
//nolint:gochecknoglobals // test injection seam
var newReviewPRClient = func(ctx context.Context, logger *logrus.Logger) (gh.Client, error) {
	return gh.NewClient(ctx, logger, nil)
}

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

	// Validate URL length to prevent ReDoS attacks on regex
	if len(url) > maxURLLength {
		return nil, fmt.Errorf("%w: %d characters (max %d)", ErrURLTooLong, len(url), maxURLLength)
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
	PRInfo                  PRInfo   `json:"pr_info"`
	Reviewed                bool     `json:"reviewed"`
	AlreadyReviewed         bool     `json:"already_reviewed,omitempty"`           // True if already reviewed by current user
	SelfAuthored            bool     `json:"self_authored,omitempty"`              // True if PR is authored by current user
	CommentAdded            bool     `json:"comment_added,omitempty"`              // True if comment was added instead of review
	Merged                  bool     `json:"merged"`                               // True if merged immediately
	AutoMergeEnabled        bool     `json:"auto_merge_enabled"`                   // True if auto-merge was enabled
	AutoMergeAlreadyEnabled bool     `json:"auto_merge_already_enabled,omitempty"` // True if auto-merge was already enabled
	MergeSkippedNoLabel     bool     `json:"merge_skipped_no_label,omitempty"`     // True if merge skipped due to missing automerge label
	MergeMethod             string   `json:"merge_method,omitempty"`
	Error                   string   `json:"error,omitempty"`
	AlreadyMerged           bool     `json:"already_merged,omitempty"`
	ChecksSkippedRunning    bool     `json:"checks_skipped_running,omitempty"` // True if skipped due to running checks
	ChecksSkippedFailed     bool     `json:"checks_skipped_failed,omitempty"`  // True if skipped due to failed checks
	CheckSummary            string   `json:"check_summary,omitempty"`          // Human-readable check summary
	RunningCheckNames       []string `json:"running_check_names,omitempty"`    // Names of running checks
	FailedCheckNames        []string `json:"failed_check_names,omitempty"`     // Names of failed checks
}

// skippedPRInfo tracks details about PRs skipped due to check status
type skippedPRInfo struct {
	Repo       string
	Number     int
	Reason     string   // "running" or "failed"
	CheckNames []string // Names of running/failed checks
}

// createReviewPRCmd creates the review-pr command for isolated testing
func createReviewPRCmd(flags *Flags) *cobra.Command {
	var message string
	var allAssignedPRs bool
	var bypass bool
	var ignoreChecks bool
	var dependabot bool

	cmd := &cobra.Command{
		Use:   "review-pr [<pr-url> [pr-url...]]",
		Short: "Review and merge pull requests",
		Long: `Review and merge one or more pull requests from GitHub URLs or all assigned PRs.

This command will:
1. Parse the PR URL(s) to extract owner/repo/number (or fetch all assigned PRs if --all-assigned-prs is used)
2. Submit an approving review with the specified message
3. Detect the repository's preferred merge method
4. Merge the PR using the detected method

The --dependabot flag (combined with --all-assigned-prs) narrows the search to
Dependabot-authored PRs and always enforces a CI gate: PRs with all checks
passing are approved and merged immediately, PRs with checks still running are
approved and handed to GitHub auto-merge, and PRs with failed checks are
skipped. The GO_BROADCAST_AUTOMERGE_LABELS requirement is bypassed for
Dependabot PRs.

The command supports both single and batch operations, processing multiple PRs in sequence.`,
		Example: `  # Review and merge a single PR
  go-broadcast review-pr https://github.com/owner/repo/pull/123

  # Review and merge multiple PRs
  go-broadcast review-pr https://github.com/owner/repo/pull/123 https://github.com/owner/repo/pull/124

  # Use short format
  go-broadcast review-pr owner/repo#123

  # Review and merge all PRs assigned to you
  go-broadcast review-pr --all-assigned-prs

  # Review and merge all assigned Dependabot PRs (CI gate enforced)
  go-broadcast review-pr --all-assigned-prs --dependabot

  # Preview what --dependabot would do without making changes
  go-broadcast review-pr --all-assigned-prs --dependabot --dry-run

  # Customize the review message
  go-broadcast review-pr --message "Approved after testing" https://github.com/owner/repo/pull/123

  # Preview without executing
  go-broadcast review-pr --dry-run https://github.com/owner/repo/pull/123

  # Bypass branch protection with admin privileges
  go-broadcast review-pr --bypass https://github.com/owner/repo/pull/123

  # Bypass and ignore status checks (dangerous)
  go-broadcast review-pr --bypass --ignore-checks https://github.com/owner/repo/pull/123

  # Review all assigned PRs with custom message
  go-broadcast review-pr --all-assigned-prs --message "LGTM" --dry-run`,
		Args: cobra.ArbitraryArgs, // Allow 0 or more args since --all-assigned-prs doesn't need URLs
		RunE: createRunReviewPR(flags, &message, &allAssignedPRs, &bypass, &ignoreChecks, &dependabot),
	}

	cmd.Flags().StringVarP(&message, "message", "m", "LGTM", "Review approval message")
	cmd.Flags().BoolVar(&allAssignedPRs, "all-assigned-prs", false, "Review and merge all open PRs assigned to you (excludes drafts)")
	cmd.Flags().BoolVar(&bypass, "bypass", false, "Use admin privileges to bypass branch protection rules")
	cmd.Flags().BoolVar(&ignoreChecks, "ignore-checks", false, "Skip waiting for status checks to pass (use with --bypass)")
	cmd.Flags().BoolVar(&dependabot, "dependabot", false, "Filter --all-assigned-prs to only Dependabot PRs; enforces CI gate and skips the GO_BROADCAST_AUTOMERGE_LABELS requirement")

	return cmd
}

// createRunReviewPR creates the run function for the review-pr command
func createRunReviewPR(flags *Flags, message *string, allAssignedPRs, bypass, ignoreChecks, dependabot *bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		log := logrus.WithField("command", "review-pr")

		var prInfos []*PRInfo

		// Validate arguments BEFORE creating GitHub client (fail fast)
		// --dependabot is a filter on --all-assigned-prs; it cannot be used alone.
		if *dependabot && !*allAssignedPRs {
			return ErrDependabotRequiresAllAssigned
		}

		// Check for mutually exclusive options
		if *allAssignedPRs && len(args) > 0 {
			return ErrMutuallyExclusiveFlags
		}

		// If not using --all-assigned-prs, validate and parse PR URLs from arguments
		if !*allAssignedPRs {
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

		// Initialize GitHub client (only after validation passes)
		client, err := newReviewPRClient(ctx, log.Logger)
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}

		// Load automerge labels from environment for bypass validation
		automergeLabels := parseAutomergeLabels(os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"))

		// If using --all-assigned-prs, fetch PRs from GitHub
		if *allAssignedPRs {
			var prs []gh.PR
			var err error

			if *dependabot {
				output.Info("Fetching assigned Dependabot PRs...")
				prs, err = client.SearchAssignedPRsByAuthor(ctx, dependabotAuthor)
			} else {
				output.Info("Fetching all PRs assigned to you...")
				prs, err = client.SearchAssignedPRs(ctx)
			}
			if err != nil {
				return fmt.Errorf("failed to search assigned PRs: %w", err)
			}

			if len(prs) == 0 {
				if *dependabot {
					output.Warn("No assigned Dependabot PRs found")
					return ErrNoDependabotPRs
				}
				output.Warn("No assigned PRs found")
				return ErrNoAssignedPRs
			}

			if *dependabot {
				output.Info(fmt.Sprintf("Found %d assigned Dependabot PR(s) (draft PRs filtered out)", len(prs)))
			} else {
				output.Info(fmt.Sprintf("Found %d assigned PR(s) (draft PRs filtered out)", len(prs)))
			}

			// Convert PRs to PRInfo structs
			for _, pr := range prs {
				// Extract repository from pr.Repo field (populated by SearchAssignedPRs)
				repo := pr.Repo
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
		}

		if len(prInfos) == 0 {
			return ErrNoValidPRURLs
		}

		// Process each PR
		results := make([]ReviewPRResult, 0, len(prInfos))
		successCount := 0
		failureCount := 0
		immediatelyMergedCount := 0
		bypassMergedCount := 0
		autoMergeCount := 0
		selfAuthoredCount := 0
		reviewOnlyCount := 0
		checksRunningSkipCount := 0
		checksFailedSkipCount := 0
		skippedPRs := make([]skippedPRInfo, 0)

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

			// Check if this is a self-authored PR
			isSelfAuthored := pr.User.Login == currentUser.Login

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

			// Get repository settings to determine merge method (needed for dry-run output)
			repoFullName := fmt.Sprintf("%s/%s", prInfo.Owner, prInfo.Repo)
			repo, err := client.GetRepository(ctx, repoFullName)
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

			// Dry-run mode: show what would happen based on investigation
			if flags.DryRun {
				// Show review action
				if isSelfAuthored {
					output.Info("DRY RUN: You are the PR author - would add comment instead of review")
					output.Info(fmt.Sprintf("DRY RUN: Would add comment with message: %s", *message))
				} else if alreadyApproved {
					output.Info("DRY RUN: Already approved by you - would skip review")
				} else {
					output.Info(fmt.Sprintf("DRY RUN: Would submit approval with message: %s", *message))
				}

				// Check if PR has automerge label - required for any merge operation
				hasAutoLabel := hasAutomergeLabel(pr.Labels, automergeLabels)

				// Dependabot mode bypasses the automerge-label requirement entirely.
				if *dependabot {
					output.Info("DRY RUN: Dependabot mode - GO_BROADCAST_AUTOMERGE_LABELS gate is skipped")
				} else if len(automergeLabels) > 0 && !hasAutoLabel {
					// If automerge labels are configured but PR lacks the label, show review-only behavior
					output.Info("DRY RUN: PR lacks automerge label - would review only, NO merge attempt")
					output.Info(fmt.Sprintf("DRY RUN: Required labels for merge: %s", strings.Join(automergeLabels, ", ")))
					result.Reviewed = false
					result.Merged = false
					result.MergeSkippedNoLabel = true
					results = append(results, result) //nolint:staticcheck // results used in summary
					successCount++
					continue
				}

				// Show merge strategy
				output.Info(fmt.Sprintf("DRY RUN: Merge method: %s", mergeMethod))

				// Show merge approach based on state
				switch {
				case pr.Mergeable != nil && !*pr.Mergeable:
					output.Warn("DRY RUN: PR has merge conflicts - would enable auto-merge")
				case *dependabot:
					output.Info("DRY RUN: Dependabot mode - CI gate enforced")
					output.Info("DRY RUN: - All checks passed → approve + immediate merge")
					output.Info("DRY RUN: - Checks still running → approve + enable auto-merge")
					output.Info("DRY RUN: - Checks failed → skip PR")
				case *bypass:
					// Bypass is allowed since we already confirmed hasAutoLabel above (or no labels configured)
					output.Info("DRY RUN: PR has automerge label - bypass allowed if needed")
					if *ignoreChecks {
						output.Warn("DRY RUN: Would ignore status checks (--ignore-checks) and force merge")
					} else {
						output.Info("DRY RUN: Would check CI status before bypass merge")
						output.Info("DRY RUN: - Running checks → skip PR")
						output.Info("DRY RUN: - Failed checks → skip PR")
						output.Info("DRY RUN: - All passed → proceed with bypass merge")
						output.Info("DRY RUN: Tip: Use --ignore-checks to bypass even with running/failed checks")
					}
				default:
					output.Info("DRY RUN: Would attempt immediate merge (fallback to auto-merge if blocked)")
					output.Info("DRY RUN: Tip: Use --bypass to merge with admin privileges")
				}

				result.Reviewed = false
				result.Merged = false
				results = append(results, result) //nolint:staticcheck // results used in summary
				successCount++
				continue
			}

			// Check if PR has automerge label - required for any merge operation.
			// We check this BEFORE adding review/comment to avoid duplicates when PR is skipped.
			hasAutoLabel := hasAutomergeLabel(pr.Labels, automergeLabels)

			// If automerge labels are configured but PR lacks the label, skip all merge operations.
			// Dependabot mode bypasses this gate so users don't have to configure dependabot.yml
			// to auto-apply a label.
			if !*dependabot && len(automergeLabels) > 0 && !hasAutoLabel {
				result.MergeSkippedNoLabel = true
				reviewOnlyCount++
				output.Info(fmt.Sprintf("✓ PR #%d reviewed - no automerge label, skipping merge", prInfo.Number))
				output.Info(fmt.Sprintf("Required labels for merge: %s", strings.Join(automergeLabels, ", ")))
				results = append(results, result) //nolint:staticcheck // results used in summary
				successCount++
				continue
			}

			output.Info(fmt.Sprintf("Using merge method: %s", mergeMethod))

			// Smart merge strategy: try-and-fallback approach

			// If PR has merge conflicts, skip straight to auto-merge
			if pr.Mergeable != nil && !*pr.Mergeable {
				// Submit review/comment for merge conflict case (auto-merge will complete the merge)
				if alreadyApproved {
					result.AlreadyReviewed = true
					output.Info(fmt.Sprintf("✓ PR #%d already reviewed by you", prInfo.Number))
				} else if isSelfAuthored {
					result.SelfAuthored = true
					output.Info(fmt.Sprintf("Adding comment to self-authored PR #%d...", prInfo.Number))
					err = client.AddPRComment(ctx, repoFullName, prInfo.Number, *message)
					if err != nil {
						result.Error = fmt.Sprintf("Failed to add comment: %v", err)
						output.Error(result.Error)
						results = append(results, result) //nolint:staticcheck // results used in summary
						failureCount++
						continue
					}
					result.CommentAdded = true
					selfAuthoredCount++
					output.Success(fmt.Sprintf("✓ Comment added to PR #%d (self-authored)", prInfo.Number))
				} else {
					output.Info(fmt.Sprintf("Submitting approval for PR #%d...", prInfo.Number))
					err = client.ReviewPR(ctx, repoFullName, prInfo.Number, *message)
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
				// Determine if bypass is allowed (requires automerge label)
				bypassAllowed := *bypass && hasAutoLabel

				// Warn if bypass was requested but not allowed (only happens when no labels configured)
				if *bypass && !bypassAllowed && !*dependabot {
					output.Warn(fmt.Sprintf("⚠️  PR #%d - bypass not allowed (no automerge labels configured)", prInfo.Number))
					output.Info("Configure GO_BROADCAST_AUTOMERGE_LABELS to enable --bypass functionality")
				}

				// Dependabot mode allows bypass regardless of label gate (label gate is skipped above).
				if *dependabot && *bypass {
					bypassAllowed = true
				}

				// Run the CI gate when either:
				//   - dependabot mode is active (always gate on CI), OR
				//   - bypass is allowed AND --ignore-checks was not set.
				//
				// forceAutoMerge is set to true when dependabot mode detects checks still
				// running: we still approve, but skip straight to EnableAutoMergePR rather
				// than racing CI with an immediate MergePR call.
				var forceAutoMerge bool
				runCheckGate := *dependabot || (bypassAllowed && !*ignoreChecks)
				if runCheckGate {
					decision := runCIGate(ctx, client, repoFullName, prInfo, &result)
					switch decision {
					case ciGateSkipRunning:
						if *dependabot {
							// Dependabot: approve + enable auto-merge instead of skipping.
							output.Info("   Checks still running - will approve and enable auto-merge")
							forceAutoMerge = true
						} else {
							output.Warn("   Skipping - checks still running")
							result.ChecksSkippedRunning = true
							checksRunningSkipCount++
							skippedPRs = append(skippedPRs, skippedPRInfo{
								Repo:       repoFullName,
								Number:     prInfo.Number,
								Reason:     "running",
								CheckNames: result.RunningCheckNames,
							})
							results = append(results, result) //nolint:staticcheck // results used in summary
							successCount++                    // Count as success since we processed it correctly
							continue
						}
					case ciGateSkipFailed:
						output.Error("   Skipping - checks failed")
						result.ChecksSkippedFailed = true
						checksFailedSkipCount++
						skippedPRs = append(skippedPRs, skippedPRInfo{
							Repo:       repoFullName,
							Number:     prInfo.Number,
							Reason:     "failed",
							CheckNames: result.FailedCheckNames,
						})
						results = append(results, result) //nolint:staticcheck // results used in summary
						successCount++                    // Count as success since we processed it correctly
						continue
					case ciGateProceed, ciGateUnknown:
						// proceed with review + merge
					}
				}

				// Submit review/comment NOW (after all skip conditions passed)
				// We do this here to avoid duplicate comments when PR is skipped due to running/failed checks
				if alreadyApproved {
					result.AlreadyReviewed = true
					output.Info(fmt.Sprintf("✓ PR #%d already reviewed by you", prInfo.Number))
				} else if isSelfAuthored {
					// Can't approve own PR - add comment instead
					result.SelfAuthored = true
					output.Info(fmt.Sprintf("Adding comment to self-authored PR #%d...", prInfo.Number))
					err = client.AddPRComment(ctx, repoFullName, prInfo.Number, *message)
					if err != nil {
						result.Error = fmt.Sprintf("Failed to add comment: %v", err)
						output.Error(result.Error)
						results = append(results, result) //nolint:staticcheck // results used in summary
						failureCount++
						continue
					}
					result.CommentAdded = true
					selfAuthoredCount++
					output.Success(fmt.Sprintf("✓ Comment added to PR #%d (self-authored)", prInfo.Number))
				} else {
					output.Info(fmt.Sprintf("Submitting approval for PR #%d...", prInfo.Number))
					err = client.ReviewPR(ctx, repoFullName, prInfo.Number, *message)
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

				// Dependabot + CI running: skip immediate merge and enable auto-merge instead.
				if forceAutoMerge {
					if autoMergeAlreadyEnabled {
						result.AutoMergeAlreadyEnabled = true
						output.Info(fmt.Sprintf("✓ Auto-merge already enabled for PR #%d", prInfo.Number))
					} else {
						output.Info(fmt.Sprintf("Enabling auto-merge for PR #%d (waiting for CI)...", prInfo.Number))
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
						output.Success(fmt.Sprintf("✓ Auto-merge enabled for PR #%d - will merge when checks pass", prInfo.Number))
					}
					results = append(results, result) //nolint:staticcheck // results used in summary
					successCount++
					continue
				}

				// Try immediate merge first (optimistic approach)
				if bypassAllowed {
					if *ignoreChecks {
						output.Info(fmt.Sprintf("Merging PR #%d (bypass available, ignoring checks)...", prInfo.Number))
					} else {
						output.Info(fmt.Sprintf("Merging PR #%d (bypass available if needed)...", prInfo.Number))
					}
				} else {
					output.Info(fmt.Sprintf("Merging PR #%d...", prInfo.Number))
				}
				err = client.MergePR(ctx, repoFullName, prInfo.Number, mergeMethod)
				if err != nil {
					// Check if error is due to branch protection policies
					if gh.IsBranchProtectionError(err) {
						if bypassAllowed {
							// Use admin bypass as last resort
							output.Info(fmt.Sprintf("Branch protection blocking - using admin bypass for PR #%d...", prInfo.Number))
							err = client.BypassMergePR(ctx, repoFullName, prInfo.Number, mergeMethod)
							if err != nil {
								result.Error = fmt.Sprintf("Failed to bypass merge PR: %v", err)
								output.Error(result.Error)
								results = append(results, result) //nolint:staticcheck // results used in summary
								failureCount++
								continue
							}
							result.Merged = true
							bypassMergedCount++
							output.Success(fmt.Sprintf("✓ PR #%d merged using admin bypass with %s method", prInfo.Number, mergeMethod))
						} else {
							// Fallback to auto-merge (bypass not allowed)
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
			if selfAuthoredCount > 0 {
				output.Info(fmt.Sprintf("Self-authored (comment added): %d", selfAuthoredCount))
			}
			if immediatelyMergedCount > 0 {
				output.Success(fmt.Sprintf("Merged immediately: %d", immediatelyMergedCount))
			}
			if bypassMergedCount > 0 {
				output.Success(fmt.Sprintf("Merged via admin bypass: %d", bypassMergedCount))
			}
			if autoMergeCount > 0 {
				output.Success(fmt.Sprintf("Auto-merge enabled: %d", autoMergeCount))
			}
			if checksRunningSkipCount > 0 {
				output.Warn(fmt.Sprintf("Skipped (checks running): %d", checksRunningSkipCount))
				for _, skipped := range skippedPRs {
					if skipped.Reason == "running" {
						checkList := strings.Join(skipped.CheckNames, ", ")
						output.Info(fmt.Sprintf("  - %s#%d: %d check(s) still running (%s)",
							skipped.Repo, skipped.Number, len(skipped.CheckNames), checkList))
					}
				}
			}
			if checksFailedSkipCount > 0 {
				output.Error(fmt.Sprintf("Skipped (checks failed): %d", checksFailedSkipCount))
				for _, skipped := range skippedPRs {
					if skipped.Reason == "failed" {
						checkList := strings.Join(skipped.CheckNames, ", ")
						output.Info(fmt.Sprintf("  - %s#%d: %d check(s) failed (%s)",
							skipped.Repo, skipped.Number, len(skipped.CheckNames), checkList))
					}
				}
			}
			if reviewOnlyCount > 0 {
				output.Info(fmt.Sprintf("Review only (no automerge label): %d", reviewOnlyCount))
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

// runCIGate fetches CI check status for a PR, emits progress output, mutates
// the provided result with any discovered check summary / names, and returns a
// decision that tells the caller whether to proceed with the merge, skip the
// PR, or treat the status as unknown (network/permission error).
//
// The caller is responsible for updating counters, appending to skippedPRs,
// and deciding whether a skipRunning result should become a skip or a fallback
// to enabling auto-merge (the dependabot path does the latter).
func runCIGate(ctx context.Context, client gh.Client, repoFullName string, prInfo *PRInfo, result *ReviewPRResult) ciGateDecision {
	checkStatus, checkErr := client.GetPRCheckStatus(ctx, repoFullName, prInfo.Number)
	if checkErr != nil {
		output.Warn(fmt.Sprintf("⚠️  Could not fetch check status for PR #%d: %v", prInfo.Number, checkErr))
		return ciGateUnknown
	}

	result.CheckSummary = checkStatus.Summary()

	switch {
	case checkStatus.NoChecks():
		output.Info(fmt.Sprintf("PR #%d: %s", prInfo.Number, checkStatus.Summary()))
		return ciGateProceed
	case checkStatus.HasRunningChecks():
		output.Warn(fmt.Sprintf("⏳ PR #%d: %s", prInfo.Number, checkStatus.Summary()))
		runningNames := checkStatus.RunningCheckNames()
		result.RunningCheckNames = runningNames
		output.Info("   Running:")
		for _, name := range runningNames {
			output.Info(fmt.Sprintf("     - %s", name))
		}
		return ciGateSkipRunning
	case checkStatus.HasFailedChecks():
		output.Error(fmt.Sprintf("❌ PR #%d: %s", prInfo.Number, checkStatus.Summary()))
		failedNames := checkStatus.FailedCheckNames()
		result.FailedCheckNames = failedNames
		output.Info("   Failed:")
		for _, name := range failedNames {
			output.Info(fmt.Sprintf("     - %s", name))
		}
		return ciGateSkipFailed
	default:
		// All checks passed
		output.Success(fmt.Sprintf("✓ PR #%d: %s", prInfo.Number, checkStatus.Summary()))
		return ciGateProceed
	}
}

// parseAutomergeLabels parses comma-separated automerge labels from environment variable
func parseAutomergeLabels(envValue string) []string {
	if envValue == "" {
		return nil
	}
	var labels []string
	for _, label := range strings.Split(envValue, ",") {
		if trimmed := strings.TrimSpace(label); trimmed != "" {
			labels = append(labels, trimmed)
		}
	}
	return labels
}

// hasAutomergeLabel checks if PR has any of the configured automerge labels
func hasAutomergeLabel(prLabels []struct {
	Name string `json:"name"`
}, automergeLabels []string,
) bool {
	if len(automergeLabels) == 0 {
		return false // No labels configured = bypass not allowed
	}
	for _, prLabel := range prLabels {
		for _, autoLabel := range automergeLabels {
			if strings.EqualFold(prLabel.Name, autoLabel) {
				return true
			}
		}
	}
	return false
}
