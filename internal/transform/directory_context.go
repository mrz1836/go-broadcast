// Package transform provides file content transformation capabilities
package transform

import (
	"fmt"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// DirectoryTransformContext extends the base Context with directory-specific metadata
// for tracking file transformations within directory sync operations.
type DirectoryTransformContext struct {
	Context

	// IsFromDirectory indicates if this transformation is part of a directory sync operation
	IsFromDirectory bool

	// DirectoryMapping contains the configuration for the directory being synced
	DirectoryMapping *config.DirectoryMapping

	// RelativePath represents the file's position within the source directory structure
	RelativePath string

	// FileIndex is the current file number being processed (0-based)
	FileIndex int

	// TotalFiles is the total number of files in the directory sync operation
	TotalFiles int

	// TransformStartTime records when the transformation began for performance metrics
	TransformStartTime time.Time
}

// NewDirectoryTransformContext creates a new DirectoryTransformContext with the provided parameters.
// The TransformStartTime is automatically set to the current time.
func NewDirectoryTransformContext(
	baseCtx Context,
	dirMapping *config.DirectoryMapping,
	relativePath string,
	fileIndex, totalFiles int,
) *DirectoryTransformContext {
	return &DirectoryTransformContext{
		Context:            baseCtx,
		IsFromDirectory:    true,
		DirectoryMapping:   dirMapping,
		RelativePath:       relativePath,
		FileIndex:          fileIndex,
		TotalFiles:         totalFiles,
		TransformStartTime: time.Now(),
	}
}

// GetTransformDuration returns the elapsed time since the transformation started.
// This is useful for performance monitoring and debugging slow transformations.
func (ctx *DirectoryTransformContext) GetTransformDuration() time.Duration {
	return time.Since(ctx.TransformStartTime)
}

// String returns a human-readable representation of the DirectoryTransformContext
// for debugging and logging purposes.
func (ctx *DirectoryTransformContext) String() string {
	if !ctx.IsFromDirectory {
		return fmt.Sprintf("DirectoryTransformContext{FilePath: %s, IsFromDirectory: false}",
			ctx.FilePath)
	}

	return fmt.Sprintf(
		"DirectoryTransformContext{"+
			"SourceRepo: %s, "+
			"TargetRepo: %s, "+
			"FilePath: %s, "+
			"RelativePath: %s, "+
			"Progress: %d/%d, "+
			"DirectoryMapping: %s->%s, "+
			"Duration: %v"+
			"}",
		ctx.SourceRepo,
		ctx.TargetRepo,
		ctx.FilePath,
		ctx.RelativePath,
		ctx.FileIndex+1, // Display as 1-based for human readability
		ctx.TotalFiles,
		ctx.DirectoryMapping.Src,
		ctx.DirectoryMapping.Dest,
		ctx.GetTransformDuration(),
	)
}
