package analytics

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// Pipeline orchestrates repo discovery and batched metadata collection
type Pipeline struct {
	ghClient gh.Client
	logger   *logrus.Logger
}

// NewPipeline creates a new analytics pipeline
func NewPipeline(ghClient gh.Client, logger *logrus.Logger) *Pipeline {
	return &Pipeline{
		ghClient: ghClient,
		logger:   logger,
	}
}

// SyncOrganization discovers repos for an organization and collects metadata
func (p *Pipeline) SyncOrganization(ctx context.Context, org string) (map[string]*RepoMetadata, error) {
	// Step 1: Discover all repos for the organization
	repos, err := p.ghClient.DiscoverOrgRepos(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("failed to discover repos for org %s: %w", org, err)
	}

	if p.logger != nil {
		p.logger.WithFields(logrus.Fields{
			"org":        org,
			"repo_count": len(repos),
		}).Info("Discovered repositories")
	}

	if len(repos) == 0 {
		return make(map[string]*RepoMetadata), nil
	}

	// Step 2: Batch repos and collect metadata
	return p.collectMetadata(ctx, repos)
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
	metadata, err := p.collectMetadata(ctx, []gh.RepoInfo{repo})
	if err != nil {
		return nil, err
	}

	result, ok := metadata[repo.FullName]
	if !ok {
		return nil, fmt.Errorf("no metadata returned for %s", repo.FullName)
	}

	return result, nil
}

// collectMetadata executes batched GraphQL queries to collect repo metadata
func (p *Pipeline) collectMetadata(ctx context.Context, repos []gh.RepoInfo) (map[string]*RepoMetadata, error) {
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

	// Process each batch
	for i, batch := range batches {
		if p.logger != nil {
			p.logger.WithFields(logrus.Fields{
				"batch":      i + 1,
				"batch_size": len(batch),
			}).Debug("Processing batch")
		}

		// Build GraphQL query for this batch
		query := BuildBatchQuery(batch)
		if query == "" {
			continue
		}

		// Execute GraphQL query
		data, err := p.ghClient.ExecuteGraphQL(ctx, query)
		if err != nil {
			// Log error and continue with next batch (or fall back to smaller batches)
			if p.logger != nil {
				p.logger.WithError(err).WithField("batch", i+1).Warn("Failed to execute batch query")
			}

			// Try fallback with smaller batch size if this is a complexity error
			if isComplexityError(err) {
				if p.logger != nil {
					p.logger.WithField("batch", i+1).Info("Retrying with smaller batch size")
				}

				// Retry with smaller batches
				smallerBatches := ChunkRepos(batch, FallbackBatchSize)
				for j, smallBatch := range smallerBatches {
					smallQuery := BuildBatchQuery(smallBatch)
					smallData, smallErr := p.ghClient.ExecuteGraphQL(ctx, smallQuery)
					if smallErr != nil {
						if p.logger != nil {
							p.logger.WithError(smallErr).WithFields(logrus.Fields{
								"batch":     i + 1,
								"sub_batch": j + 1,
							}).Error("Failed to execute fallback batch query")
						}
						continue
					}

					// Parse and merge results
					metadata, parseErr := ParseBatchResponse(smallData, smallBatch)
					if parseErr != nil {
						if p.logger != nil {
							p.logger.WithError(parseErr).Warn("Failed to parse fallback batch response")
						}
						continue
					}

					for k, v := range metadata {
						allMetadata[k] = v
					}
				}
			}

			continue
		}

		// Parse response
		metadata, err := ParseBatchResponse(data, batch)
		if err != nil {
			if p.logger != nil {
				p.logger.WithError(err).WithField("batch", i+1).Warn("Failed to parse batch response")
			}
			continue
		}

		// Merge into results
		for k, v := range metadata {
			allMetadata[k] = v
		}

		if p.logger != nil {
			p.logger.WithFields(logrus.Fields{
				"batch":        i + 1,
				"repos_parsed": len(metadata),
			}).Debug("Batch processed successfully")
		}
	}

	if p.logger != nil {
		p.logger.WithFields(logrus.Fields{
			"total_metadata": len(allMetadata),
			"expected":       len(repos),
		}).Info("Metadata collection complete")
	}

	return allMetadata, nil
}

// isComplexityError checks if the error is due to GraphQL complexity limits
func isComplexityError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "complexity") || contains(errStr, "too complex") || contains(errStr, "query cost")
}

// contains checks if a string contains a substring (avoiding strings import)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
