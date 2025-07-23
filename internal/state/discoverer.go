package state

import (
	"context"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// Discoverer defines the interface for discovering sync state from GitHub
type Discoverer interface {
	// DiscoverState discovers the complete sync state by examining GitHub
	// This includes checking branches, PRs, and commits across all repositories
	DiscoverState(ctx context.Context, cfg *config.Config) (*State, error)

	// DiscoverTargetState discovers the state of a specific target repository
	DiscoverTargetState(ctx context.Context, repo string) (*TargetState, error)

	// ParseBranchName parses a branch name to extract sync metadata
	// Returns nil if the branch is not a sync branch
	ParseBranchName(name string) (*BranchMetadata, error)
}
