package cli

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/output"
)

func newSettingsAuditCmd() *cobra.Command {
	var (
		presetID   string
		org        string
		all        bool
		save       bool
		format     string
		dryRun     bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "audit [owner/repo...]",
		Short: "Audit repositories against their assigned preset",
		Long: `Audit one or more repositories against a settings preset.

Checks all 12 managed settings, rulesets, and labels. Outputs a score and
detailed results. Exit code 1 if any checks fail (CI-friendly).`,
		Example: `  # Audit a single repo
  go-broadcast settings audit owner/my-repo

  # Audit with specific preset
  go-broadcast settings audit owner/my-repo --preset go-lib

  # Audit all repos in database
  go-broadcast settings audit --all

  # Audit and save results to database
  go-broadcast settings audit owner/my-repo --save

  # JSON output for CI
  go-broadcast settings audit owner/my-repo --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSettingsAudit(cmd.Context(), args, presetID, org, all, save, format, dryRun, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&presetID, "preset", "", "Settings preset ID (default: repo's assigned preset or mvp)")
	cmd.Flags().StringVar(&org, "org", "", "Audit all repos in an organization")
	cmd.Flags().BoolVar(&all, "all", false, "Audit all repos in database")
	cmd.Flags().BoolVar(&save, "save", false, "Save audit results to database")
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be audited")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// auditResult holds the audit outcome for a single repo
type auditResult struct {
	Repo   string             `json:"repo"`
	Preset string             `json:"preset"`
	Score  int                `json:"score"`
	Total  int                `json:"total"`
	Passed int                `json:"passed"`
	Failed int                `json:"failed"`
	Checks []auditCheckResult `json:"checks"`
	Error  string             `json:"error,omitempty"`
}

type auditCheckResult struct {
	Setting  string `json:"setting"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Pass     bool   `json:"pass"`
}

func runSettingsAudit(ctx context.Context, repos []string, presetID, org string, all, save bool, format string, dryRun, jsonOutput bool) error {
	// Resolve repos to audit
	auditRepos, err := resolveAuditRepos(ctx, repos, org, all)
	if err != nil {
		return err
	}

	if len(auditRepos) == 0 {
		return fmt.Errorf("no repositories to audit") //nolint:err113 // user-facing
	}

	if dryRun {
		output.Info(fmt.Sprintf("[DRY RUN] Would audit %d repository(ies):", len(auditRepos)))
		for _, r := range auditRepos {
			output.Info(fmt.Sprintf("  %s", r))
		}
		return nil
	}

	// Initialize GitHub client
	logger := logrus.StandardLogger()
	ghClient, err := gh.NewClient(ctx, logger, &logging.LogConfig{})
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Audit repos with bounded concurrency
	const maxConcurrency = 5
	sem := make(chan struct{}, maxConcurrency)
	var mu sync.Mutex
	results := make([]auditResult, 0, len(auditRepos))

	var wg sync.WaitGroup
	for _, repo := range auditRepos {
		wg.Add(1)
		go func(repoName string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := auditSingleRepo(ctx, ghClient, repoName, presetID)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(repo)
	}
	wg.Wait()

	// Save to DB if requested
	if save {
		saveAuditResults(ctx, results)
	}

	// Output results
	hasFailures := false
	if jsonOutput || format == "json" {
		resp := CLIResponse{
			Success: true,
			Action:  "audited",
			Type:    "settings_audit",
			Data:    results,
			Count:   len(results),
		}
		if err := printResponse(resp, true); err != nil {
			return err
		}
	} else {
		for _, r := range results {
			if r.Error != "" {
				output.Error(fmt.Sprintf("%s: %s", r.Repo, r.Error))
				hasFailures = true
				continue
			}

			scoreLabel := "Perfect"
			if r.Score < 100 {
				scoreLabel = "Needs attention"
			}
			if r.Score >= 70 && r.Score < 100 {
				scoreLabel = "Good"
			}

			output.Info(fmt.Sprintf("%s (preset: %s) — Score: %d/%d (%d%%) %s",
				r.Repo, r.Preset, r.Passed, r.Total, r.Score, scoreLabel))

			for _, c := range r.Checks {
				if c.Pass {
					output.Success(fmt.Sprintf("  %s: %s", c.Setting, c.Actual))
				} else {
					output.Error(fmt.Sprintf("  %s: %s (expected: %s)", c.Setting, c.Actual, c.Expected))
				}
			}
		}
	}

	// Check for failures
	for _, r := range results {
		if r.Score < 100 || r.Error != "" {
			hasFailures = true
			break
		}
	}

	if hasFailures {
		return fmt.Errorf("audit completed with failures") //nolint:err113 // CI exit code
	}
	return nil
}

func auditSingleRepo(ctx context.Context, ghClient gh.Client, repo, presetID string) auditResult {
	// Resolve preset for this repo. An empty --preset flag falls back to the
	// generic mvp default to match the documented "default: repo's assigned
	// preset or mvp" behavior without re-introducing silent forging for
	// arbitrary unknown ids.
	resolveID := presetID
	if resolveID == "" {
		resolveID = "mvp"
	}
	preset, err := resolvePreset(ctx, resolveID)
	if err != nil {
		return auditResult{Repo: repo, Preset: presetID, Error: err.Error()}
	}
	if presetID == "" {
		presetID = preset.ID
	}

	// Get current settings
	current, err := ghClient.GetRepoSettings(ctx, repo)
	if err != nil {
		return auditResult{Repo: repo, Preset: presetID, Error: err.Error()}
	}

	// Run checks
	checks := runAuditChecks(current, preset)

	passed := 0
	for _, c := range checks {
		if c.Pass {
			passed++
		}
	}

	total := len(checks)
	score := 0
	if total > 0 {
		score = passed * 100 / total
	}

	return auditResult{
		Repo:   repo,
		Preset: presetID,
		Score:  score,
		Total:  total,
		Passed: passed,
		Failed: total - passed,
		Checks: checks,
	}
}

func runAuditChecks(current *gh.RepoSettings, preset *config.SettingsPreset) []auditCheckResult {
	var checks []auditCheckResult

	checkBool := func(name string, cur, exp bool) {
		checks = append(checks, auditCheckResult{
			Setting:  name,
			Expected: fmt.Sprintf("%v", exp),
			Actual:   fmt.Sprintf("%v", cur),
			Pass:     cur == exp,
		})
	}
	checkStr := func(name, cur, exp string) {
		if exp == "" {
			return // Skip checks for empty expected values
		}
		checks = append(checks, auditCheckResult{
			Setting:  name,
			Expected: exp,
			Actual:   cur,
			Pass:     cur == exp,
		})
	}

	checkBool("has_issues", current.HasIssues, preset.HasIssues)
	checkBool("has_wiki", current.HasWiki, preset.HasWiki)
	checkBool("has_projects", current.HasProjects, preset.HasProjects)
	checkBool("has_discussions", current.HasDiscussions, preset.HasDiscussions)
	checkBool("allow_squash_merge", current.AllowSquashMerge, preset.AllowSquashMerge)
	checkBool("allow_merge_commit", current.AllowMergeCommit, preset.AllowMergeCommit)
	checkBool("allow_rebase_merge", current.AllowRebaseMerge, preset.AllowRebaseMerge)
	checkBool("delete_branch_on_merge", current.DeleteBranchOnMerge, preset.DeleteBranchOnMerge)
	checkBool("allow_auto_merge", current.AllowAutoMerge, preset.AllowAutoMerge)
	checkBool("allow_update_branch", current.AllowUpdateBranch, preset.AllowUpdateBranch)
	checkStr("squash_merge_commit_title", current.SquashMergeCommitTitle, preset.SquashMergeCommitTitle)
	checkStr("squash_merge_commit_message", current.SquashMergeCommitMessage, preset.SquashMergeCommitMessage)

	return checks
}

// resolveAuditRepos determines which repos to audit based on flags
func resolveAuditRepos(ctx context.Context, repos []string, org string, all bool) ([]string, error) {
	if len(repos) > 0 {
		return repos, nil
	}

	if org != "" {
		// Discover repos from org via DB
		return resolveOrgRepos(ctx, org)
	}

	if all {
		return resolveAllDBRepos(ctx)
	}

	return nil, fmt.Errorf("specify repos, --org, or --all") //nolint:err113 // user-facing
}

func resolveOrgRepos(ctx context.Context, org string) ([]string, error) {
	database, err := openDatabase()
	if err != nil {
		return nil, fmt.Errorf("database required for --org flag: %w", err)
	}
	defer func() { _ = database.Close() }()

	var repos []struct {
		FullName string `gorm:"column:full_name"`
	}
	if dbErr := database.DB().WithContext(ctx).Raw(`
		SELECT r.full_name FROM repos r
		JOIN organizations o ON r.organization_id = o.id
		WHERE o.name = ? AND r.deleted_at IS NULL
		ORDER BY r.name
	`, org).Scan(&repos).Error; dbErr != nil {
		return nil, fmt.Errorf("failed to query repos for org %q: %w", org, dbErr)
	}

	result := make([]string, 0, len(repos))
	for _, r := range repos {
		if r.FullName != "" {
			result = append(result, r.FullName)
		}
	}
	return result, nil
}

func resolveAllDBRepos(ctx context.Context) ([]string, error) {
	database, err := openDatabase()
	if err != nil {
		return nil, fmt.Errorf("database required for --all flag: %w", err)
	}
	defer func() { _ = database.Close() }()

	var repos []struct {
		FullName string `gorm:"column:full_name"`
	}
	if dbErr := database.DB().WithContext(ctx).Raw(`
		SELECT full_name FROM repos
		WHERE deleted_at IS NULL AND full_name != ''
		ORDER BY full_name
	`).Scan(&repos).Error; dbErr != nil {
		return nil, fmt.Errorf("failed to query repos: %w", dbErr)
	}

	result := make([]string, 0, len(repos))
	for _, r := range repos {
		if r.FullName != "" {
			result = append(result, r.FullName)
		}
	}
	return result, nil
}

// saveAuditResults saves audit results to the database
func saveAuditResults(ctx context.Context, results []auditResult) {
	database, err := openDatabase()
	if err != nil {
		output.Warn("Cannot save audit results: database not available")
		return
	}
	defer func() { _ = database.Close() }()

	gormDB := database.DB()

	for _, r := range results {
		if r.Error != "" {
			continue
		}

		// Find repo
		parts := strings.Split(r.Repo, "/")
		if len(parts) != 2 {
			continue
		}
		repoRepo := db.NewRepoRepository(gormDB)
		repo, repoErr := repoRepo.GetByFullName(ctx, parts[0], parts[1])
		if repoErr != nil {
			continue
		}

		// Find preset
		presetRepo := db.NewSettingsPresetRepository(gormDB)
		preset, presetErr := presetRepo.GetByExternalID(ctx, r.Preset)
		if presetErr != nil {
			continue
		}

		// Convert checks to JSONAuditResults
		auditResults := make(db.JSONAuditResults, 0, len(r.Checks))
		for _, c := range r.Checks {
			auditResults = append(auditResults, db.AuditCheckResult{
				Setting:  c.Setting,
				Expected: c.Expected,
				Actual:   c.Actual,
				Pass:     c.Pass,
			})
		}

		audit := &db.RepoSettingsAudit{
			RepoID:           repo.ID,
			SettingsPresetID: preset.ID,
			Score:            r.Score,
			Total:            r.Total,
			Passed:           r.Passed,
			Results:          auditResults,
		}

		if createErr := gormDB.WithContext(ctx).Create(audit).Error; createErr != nil {
			output.Warn(fmt.Sprintf("Failed to save audit for %s: %v", r.Repo, createErr))
		}
	}
}
