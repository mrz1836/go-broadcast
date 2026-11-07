package analytics

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

// ErrMetadataNotReturned is returned when metadata could not be collected for a
// repository, even after the repository was queried again on its own. It wraps
// the last attempt's error, so errors.Is and errors.As also see the cause.
var ErrMetadataNotReturned = errors.New("metadata not returned for repository")

// ErrRepoOmittedFromResponse is the cause recorded when a GraphQL query succeeded
// but its response left the repository out (alias absent or null), as opposed to
// the query itself failing.
var ErrRepoOmittedFromResponse = errors.New("repository omitted from GraphQL response")

// errOrgNotFound is returned when an organization is not found in the database
var errOrgNotFound = errors.New("organization not found in database")

// Pipeline orchestrates repo discovery and batched metadata collection
type Pipeline struct {
	ghClient gh.Client
	repo     db.AnalyticsRepo
	repoRepo db.RepoRepository
	orgRepo  db.OrganizationRepository
	logger   *logrus.Logger
	throttle *Throttle
}

// NewPipeline creates a new analytics pipeline.
// throttle may be nil for unthrottled operation.
func NewPipeline(ghClient gh.Client, repo db.AnalyticsRepo, repoRepo db.RepoRepository, orgRepo db.OrganizationRepository, logger *logrus.Logger, throttle *Throttle) *Pipeline {
	return &Pipeline{
		ghClient: ghClient,
		repo:     repo,
		repoRepo: repoRepo,
		orgRepo:  orgRepo,
		logger:   logger,
		throttle: throttle,
	}
}

// GetGHClient returns the GitHub client
func (p *Pipeline) GetGHClient() gh.Client {
	return p.ghClient
}

// GetLogger returns the logger
func (p *Pipeline) GetLogger() *logrus.Logger {
	return p.logger
}

// GetThrottle returns the throttle (may be nil)
func (p *Pipeline) GetThrottle() *Throttle {
	return p.throttle
}

// SyncOrganization syncs metadata for configured repos in an organization.
// Only syncs repos that are already in the go-broadcast database, not all repos owned by the org.
//
// Each configured repo is keyed by its full name in exactly one of the two maps:
// metadata for the repos that were collected, failures for the ones that were not
// (each failure wraps ErrMetadataNotReturned and the cause). The maps are only
// incomplete when err is non-nil.
func (p *Pipeline) SyncOrganization(ctx context.Context, org string) (map[string]*RepoMetadata, map[string]error, error) {
	// Step 1: Query configured repos from database for this org
	configuredRepos, err := p.getConfiguredReposForOrg(ctx, org)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get configured repos for owner %s: %w", org, err)
	}

	if p.logger != nil {
		p.logger.WithFields(logrus.Fields{
			"owner":      org,
			"repo_count": len(configuredRepos),
		}).Info("Found configured repositories")
	}

	if len(configuredRepos) == 0 {
		return make(map[string]*RepoMetadata), nil, nil
	}

	// Step 2: Batch repos and collect metadata
	return p.collectMetadata(ctx, configuredRepos)
}

// getConfiguredReposForOrg queries the database for repos configured in this org
func (p *Pipeline) getConfiguredReposForOrg(ctx context.Context, orgName string) ([]gh.RepoInfo, error) {
	// Get organization from database
	org, err := p.orgRepo.GetByName(ctx, orgName)
	if err != nil {
		return nil, fmt.Errorf("%w: %s (import config first)", errOrgNotFound, orgName)
	}

	// Get configured repos for this organization
	repos, err := p.repoRepo.List(ctx, org.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query repos: %w", err)
	}

	// Convert db.Repo to gh.RepoInfo
	repoInfos := make([]gh.RepoInfo, len(repos))
	for i, repo := range repos {
		repoInfos[i] = gh.RepoInfo{
			Name:     repo.Name,
			FullName: fmt.Sprintf("%s/%s", orgName, repo.Name),
			Owner: struct {
				Login string `json:"login"`
			}{Login: orgName},
		}
	}

	return repoInfos, nil
}

// SyncRepository collects metadata for a single repository
func (p *Pipeline) SyncRepository(ctx context.Context, owner, name string) (*RepoMetadata, error) {
	// Create a single-repo info for batching (batch of 1)
	repo := gh.RepoInfo{
		Name:     name,
		FullName: fmt.Sprintf("%s/%s", owner, name),
		Owner: struct {
			Login string `json:"login"`
		}{Login: owner},
	}

	// Use batch query with single repo
	metadata, failures, err := p.collectMetadata(ctx, []gh.RepoInfo{repo})
	if err != nil {
		return nil, err
	}

	if result, ok := metadata[repo.FullName]; ok {
		return result, nil
	}
	if failure, ok := failures[repo.FullName]; ok {
		return nil, failure
	}

	// Defensive: collectMetadata places every repo in one of the two maps
	return nil, fmt.Errorf("%w: %s", ErrMetadataNotReturned, repo.FullName)
}

// collectMetadata executes batched GraphQL queries to collect repo metadata.
//
// A repo its batch did not return (left out of a successful response, or covered
// by a failed query) is queried once more on its own before it counts as failed.
// Every requested repo lands, keyed by full name, in exactly one of metadata or
// failures; each failure wraps ErrMetadataNotReturned and the last attempt's
// error. err is non-nil only when ctx is done, and the maps are then incomplete.
func (p *Pipeline) collectMetadata(ctx context.Context, repos []gh.RepoInfo) (map[string]*RepoMetadata, map[string]error, error) {
	// Split repos into batches
	batches := ChunkRepos(repos, DefaultBatchSize)

	if p.logger != nil {
		p.logger.WithFields(logrus.Fields{
			"total_repos": len(repos),
			"batch_count": len(batches),
			"batch_size":  DefaultBatchSize,
		}).Info("Starting batched metadata collection")
	}

	allMetadata := make(map[string]*RepoMetadata)
	failures := make(map[string]error)

	// Process each batch
	for i, batch := range batches {
		batchNum := i + 1

		// Inter-batch delay to avoid rate-limit pressure (also after a failed batch)
		if i > 0 && p.throttle != nil {
			if err := p.throttle.WaitInterRepo(ctx); err != nil {
				return allMetadata, failures, err
			}
		}

		// Check for context cancellation between batches
		if err := ctx.Err(); err != nil {
			return allMetadata, failures, err
		}

		if p.logger != nil {
			p.logger.WithFields(logrus.Fields{
				"batch":      batchNum,
				"batch_size": len(batch),
			}).Debug("Processing batch")
		}

		queryErrs := p.fetchBatch(ctx, batchNum, batch, allMetadata)
		if err := p.retryMissing(ctx, batchNum, batch, queryErrs, allMetadata, failures); err != nil {
			return allMetadata, failures, err
		}
	}

	if p.logger != nil {
		p.logger.WithFields(logrus.Fields{
			"total_metadata": len(allMetadata),
			"failed":         len(failures),
			"expected":       len(repos),
		}).Info("Metadata collection complete")
	}

	return allMetadata, failures, nil
}

// queryBatch runs one aliased GraphQL query for batch and merges the repos its
// response returned into into. It returns how many repos were merged, and the
// query or parse error if the query failed.
func (p *Pipeline) queryBatch(ctx context.Context, batch []gh.RepoInfo, into map[string]*RepoMetadata) (int, error) {
	data, err := p.ghClient.ExecuteGraphQL(ctx, BuildBatchQuery(batch))
	if err != nil {
		return 0, err
	}

	metadata, err := ParseBatchResponse(data, batch)
	if err != nil {
		return 0, fmt.Errorf("failed to parse batch response: %w", err)
	}

	maps.Copy(into, metadata)
	return len(metadata), nil
}

// fetchBatch queries one batch and merges the repos it returned into into. When
// the query is too complex for a batch larger than FallbackBatchSize, it retries
// the batch as sub-batches of that size. The returned map holds, for each repo
// covered by a query that failed, that query's error.
func (p *Pipeline) fetchBatch(ctx context.Context, batchNum int, batch []gh.RepoInfo, into map[string]*RepoMetadata) map[string]error {
	parsed, err := p.queryBatch(ctx, batch, into)
	if err == nil {
		if p.logger != nil {
			p.logger.WithFields(logrus.Fields{
				"batch":        batchNum,
				"repos_parsed": parsed,
			}).Debug("Batch processed successfully")
		}
		return nil
	}

	if p.logger != nil {
		p.logger.WithError(err).WithField("batch", batchNum).Warn("Failed to execute batch query")
	}

	// A batch already at the fallback size would only re-run the same query;
	// its repos get queries of their own in retryMissing instead.
	if !isComplexityError(err) || len(batch) <= FallbackBatchSize {
		return queryErrorsFor(batch, err)
	}

	if p.logger != nil {
		p.logger.WithField("batch", batchNum).Info("Retrying with smaller batch size")
	}

	queryErrs := make(map[string]error)
	for j, subBatch := range ChunkRepos(batch, FallbackBatchSize) {
		if _, subErr := p.queryBatch(ctx, subBatch, into); subErr != nil {
			if p.logger != nil {
				p.logger.WithError(subErr).WithFields(logrus.Fields{
					"batch":     batchNum,
					"sub_batch": j + 1,
				}).Error("Failed to execute fallback batch query")
			}
			maps.Copy(queryErrs, queryErrorsFor(subBatch, subErr))
		}
	}

	return queryErrs
}

// queryErrorsFor maps the full name of each repo in repos to err
func queryErrorsFor(repos []gh.RepoInfo, err error) map[string]error {
	errs := make(map[string]error, len(repos))
	for _, repo := range repos {
		errs[repo.FullName] = err
	}
	return errs
}

// retryMissing gives each repo of batch that is not in into one query of its own,
// and records the ones still missing in failures. A repo's first cause is its
// query error from queryErrs, or ErrRepoOmittedFromResponse when the batch
// succeeded without it. A rate-limited repo is recorded without a retry: another
// request would only fail the same way and spend more of the budget.
// It returns an error only when ctx is done.
func (p *Pipeline) retryMissing(
	ctx context.Context,
	batchNum int,
	batch []gh.RepoInfo,
	queryErrs map[string]error,
	into map[string]*RepoMetadata,
	failures map[string]error,
) error {
	for _, repo := range batch {
		if _, ok := into[repo.FullName]; ok {
			continue
		}

		cause := queryErrs[repo.FullName]
		if cause == nil {
			cause = ErrRepoOmittedFromResponse
		}

		if isAPIRateLimitError(cause) {
			if p.logger != nil {
				p.logger.WithError(cause).WithFields(logrus.Fields{
					"repo":  repo.FullName,
					"batch": batchNum,
				}).Warn("Not retrying rate-limited repository")
			}
		} else {
			cause = p.retryRepo(ctx, batchNum, repo, cause, into)
			if err := ctx.Err(); err != nil {
				return err
			}
			if cause == nil {
				continue
			}
		}

		failures[repo.FullName] = fmt.Errorf("%w: %s: %w", ErrMetadataNotReturned, repo.FullName, cause)
	}

	return nil
}

// retryRepo queries repo on its own, after the inter-repo delay, and merges it
// into into when the response returns it. It returns nil once repo is recovered,
// otherwise the reason it is still missing: the retry's error, or
// ErrRepoOmittedFromResponse when the retry succeeded without it. cause is why
// the batch did not return repo; it is only logged.
func (p *Pipeline) retryRepo(ctx context.Context, batchNum int, repo gh.RepoInfo, cause error, into map[string]*RepoMetadata) error {
	if p.throttle != nil {
		if err := p.throttle.WaitInterRepo(ctx); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if p.logger != nil {
		p.logger.WithError(cause).WithFields(logrus.Fields{
			"repo":  repo.FullName,
			"batch": batchNum,
		}).Info("Retrying repository missing from batch")
	}

	_, err := p.queryBatch(ctx, []gh.RepoInfo{repo}, into)
	if err == nil {
		if _, ok := into[repo.FullName]; ok {
			return nil
		}
		err = ErrRepoOmittedFromResponse
	}

	if p.logger != nil {
		p.logger.WithError(err).WithFields(logrus.Fields{
			"repo":  repo.FullName,
			"batch": batchNum,
		}).Warn("Repository metadata still missing after retry")
	}

	return err
}

// isComplexityError checks if the error is due to GraphQL complexity limits
func isComplexityError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "complexity") || strings.Contains(errStr, "too complex") || strings.Contains(errStr, "query cost")
}

// ============================================================
// SyncRun Tracking
// ============================================================

// StartSyncRun creates a new SyncRun record at the start of sync
func (p *Pipeline) StartSyncRun(ctx context.Context, syncType, orgFilter, repoFilter string) (*db.SyncRun, error) {
	run := &db.SyncRun{
		StartedAt:  time.Now().UTC(),
		Status:     "running",
		SyncType:   syncType,
		OrgFilter:  orgFilter,
		RepoFilter: repoFilter,
	}

	if err := p.repo.CreateSyncRun(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to create sync run: %w", err)
	}

	if p.logger != nil {
		p.logger.WithFields(logrus.Fields{
			"sync_run_id": run.ID,
			"sync_type":   syncType,
			"org_filter":  orgFilter,
			"repo_filter": repoFilter,
		}).Info("Started sync run")
	}

	return run, nil
}

// UpdateSyncRunCounters updates the counters for a running SyncRun
func (p *Pipeline) UpdateSyncRunCounters(ctx context.Context, run *db.SyncRun) error {
	if err := p.repo.UpdateSyncRun(ctx, run); err != nil {
		return fmt.Errorf("failed to update sync run: %w", err)
	}
	return nil
}

// CompleteSyncRun marks a SyncRun as completed and calculates final metrics
func (p *Pipeline) CompleteSyncRun(ctx context.Context, run *db.SyncRun, status string) error {
	now := time.Now().UTC()
	run.CompletedAt = &now
	run.Status = status
	run.DurationMs = now.Sub(run.StartedAt).Milliseconds()

	if err := p.repo.UpdateSyncRun(ctx, run); err != nil {
		return fmt.Errorf("failed to complete sync run: %w", err)
	}

	if p.logger != nil {
		p.logger.WithFields(logrus.Fields{
			"sync_run_id":       run.ID,
			"status":            status,
			"duration_ms":       run.DurationMs,
			"repos_processed":   run.ReposProcessed,
			"repos_skipped":     run.ReposSkipped,
			"snapshots_created": run.SnapshotsCreated,
		}).Info("Completed sync run")
	}

	return nil
}

// RecordSyncRunError adds an error to the SyncRun's error log
func (p *Pipeline) RecordSyncRunError(ctx context.Context, run *db.SyncRun, repo string, err error) {
	errorEntry := map[string]any{
		"repo":  repo,
		"error": err.Error(),
		"time":  time.Now().UTC().Format(time.RFC3339),
	}

	// Append to errors array (always use []interface{} for JSON round-trip consistency)
	if run.Errors == nil {
		run.Errors = db.Metadata{
			"errors": []any{errorEntry},
		}
	} else {
		errorsArray, ok := run.Errors["errors"].([]any)
		if !ok {
			errorsArray = []any{}
		}
		errorsArray = append(errorsArray, errorEntry)
		run.Errors["errors"] = errorsArray
	}

	run.ReposFailed++
	run.LastProcessedRepo = repo

	if p.logger != nil {
		p.logger.WithFields(logrus.Fields{
			"sync_run_id": run.ID,
			"repo":        repo,
			"error":       err.Error(),
		}).Warn("Recorded sync error")
	}
}
