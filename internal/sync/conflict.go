package sync

import (
	"fmt"
	"sort"

	"github.com/mrz1836/go-broadcast/internal/config"
	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/sirupsen/logrus"
)

// ConflictResolver handles conflicts when multiple sources target the same file
type ConflictResolver struct {
	strategy config.ConflictResolution
	logger   *logrus.Logger
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(strategy config.ConflictResolution, logger *logrus.Logger) *ConflictResolver {
	return &ConflictResolver{
		strategy: strategy,
		logger:   logger,
	}
}

// FileConflict represents a conflict where multiple sources want to sync the same file
type FileConflict struct {
	TargetFile string
	Sources    []SourceFileInfo
}

// SourceFileInfo contains information about a source file in a conflict
type SourceFileInfo struct {
	SourceRepo   string
	SourceFile   string
	SourceID     string
	MappingIndex int
	Content      []byte
}

// ResolveConflicts resolves conflicts for a set of file mappings
func (cr *ConflictResolver) ResolveConflicts(conflicts []FileConflict) (map[string]SourceFileInfo, error) {
	resolved := make(map[string]SourceFileInfo)

	for _, conflict := range conflicts {
		winner, err := cr.resolveFileConflict(conflict)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve conflict for %s: %w", conflict.TargetFile, err)
		}
		resolved[conflict.TargetFile] = winner
	}

	return resolved, nil
}

// resolveFileConflict resolves a conflict for a single file
func (cr *ConflictResolver) resolveFileConflict(conflict FileConflict) (SourceFileInfo, error) {
	if len(conflict.Sources) == 0 {
		return SourceFileInfo{}, appErrors.NoSourcesInConflictError()
	}

	if len(conflict.Sources) == 1 {
		// No conflict, only one source
		return conflict.Sources[0], nil
	}

	cr.logger.WithFields(logrus.Fields{
		"file":         conflict.TargetFile,
		"source_count": len(conflict.Sources),
		"strategy":     cr.strategy.Strategy,
	}).Warn("Resolving file conflict")

	switch cr.strategy.Strategy {
	case "last-wins":
		return cr.resolveLastWins(conflict)
	case "priority":
		return cr.resolvePriority(conflict)
	case "error":
		return SourceFileInfo{}, appErrors.ConflictDetectedError(conflict.TargetFile, len(conflict.Sources))
	default:
		// Default to last-wins
		cr.logger.Warn("Unknown conflict strategy, defaulting to last-wins")
		return cr.resolveLastWins(conflict)
	}
}

// resolveLastWins picks the source with the highest mapping index (last in config)
func (cr *ConflictResolver) resolveLastWins(conflict FileConflict) (SourceFileInfo, error) {
	// Sort by mapping index, highest last
	sort.Slice(conflict.Sources, func(i, j int) bool {
		return conflict.Sources[i].MappingIndex < conflict.Sources[j].MappingIndex
	})

	winner := conflict.Sources[len(conflict.Sources)-1]

	cr.logger.WithFields(logrus.Fields{
		"file":        conflict.TargetFile,
		"winner":      winner.SourceRepo,
		"source_file": winner.SourceFile,
	}).Info("Resolved conflict using last-wins strategy")

	return winner, nil
}

// resolvePriority picks the source based on configured priority order
func (cr *ConflictResolver) resolvePriority(conflict FileConflict) (SourceFileInfo, error) {
	if len(cr.strategy.Priority) == 0 {
		return SourceFileInfo{}, appErrors.PriorityStrategyNoPriorityError()
	}

	// Create priority map for O(1) lookup
	priorityMap := make(map[string]int)
	for i, id := range cr.strategy.Priority {
		priorityMap[id] = i
	}

	// Find source with highest priority (lowest index)
	var winner *SourceFileInfo
	lowestPriority := len(cr.strategy.Priority)

	for i := range conflict.Sources {
		source := &conflict.Sources[i]
		if priority, exists := priorityMap[source.SourceID]; exists {
			if priority < lowestPriority {
				lowestPriority = priority
				winner = source
			}
		}
	}

	if winner == nil {
		// No source matched priority list, fall back to last-wins
		cr.logger.Warn("No sources matched priority list, falling back to last-wins")
		return cr.resolveLastWins(conflict)
	}

	cr.logger.WithFields(logrus.Fields{
		"file":        conflict.TargetFile,
		"winner":      winner.SourceRepo,
		"source_file": winner.SourceFile,
		"priority":    lowestPriority,
	}).Info("Resolved conflict using priority strategy")

	return *winner, nil
}

// DetectConflicts analyzes sync tasks to find potential conflicts
func DetectConflicts(tasks []Task) map[string][]SourceFileInfo {
	// Map of target file path to sources that want to sync it
	fileToSources := make(map[string][]SourceFileInfo)

	for _, task := range tasks {
		// Check file mappings
		for _, fileMapping := range task.Target.Files {
			info := SourceFileInfo{
				SourceRepo:   task.Source.Repo,
				SourceFile:   fileMapping.Src,
				SourceID:     task.Source.ID,
				MappingIndex: task.MappingIdx,
			}
			fileToSources[fileMapping.Dest] = append(fileToSources[fileMapping.Dest], info)
		}

		// For directories, we'd need to expand the files first
		// This would be done during actual sync when we know what files exist
	}

	// Filter to only conflicting files
	conflicts := make(map[string][]SourceFileInfo)
	for file, sources := range fileToSources {
		if len(sources) > 1 {
			conflicts[file] = sources
		}
	}

	return conflicts
}
