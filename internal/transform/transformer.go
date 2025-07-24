// Package transform provides file content transformation capabilities
package transform

import (
	"context"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

// Transformer defines the interface for content transformations
type Transformer interface {
	// Name returns the name of this transformer
	Name() string

	// Transform applies the transformation to the content
	Transform(content []byte, ctx Context) ([]byte, error)
}

// Context provides context for transformations
type Context struct {
	// SourceRepo is the source repository (e.g., "org/template-repo")
	SourceRepo string

	// TargetRepo is the target repository (e.g., "org/service-a")
	TargetRepo string

	// FilePath is the path of the file being transformed
	FilePath string

	// Variables contains custom variables for template substitution
	Variables map[string]string

	// LogConfig provides configuration for debug logging and verbose settings
	LogConfig *logging.LogConfig
}

// Chain defines the interface for composing multiple transformers
type Chain interface {
	// Add appends a transformer to the chain
	Add(transformer Transformer) Chain

	// Transform applies all transformers in sequence
	Transform(ctx context.Context, content []byte, transformCtx Context) ([]byte, error)

	// Transformers returns the list of transformers in the chain
	Transformers() []Transformer
}
