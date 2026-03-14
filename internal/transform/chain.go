package transform

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// chain implements the Chain interface
type chain struct {
	transformers []Transformer
	logger       *logrus.Logger
	mu           sync.RWMutex
}

// NewChain creates a new transformer chain.
// If logger is nil, a no-op logger is used to prevent nil pointer panics.
func NewChain(logger *logrus.Logger) Chain {
	if logger == nil {
		logger = logrus.New()
		logger.SetOutput(nil) // Discard output for no-op logger
	}
	return &chain{
		transformers: []Transformer{},
		logger:       logger,
	}
}

// Add appends a transformer to the chain.
// This method is thread-safe and can be called concurrently.
func (c *chain) Add(transformer Transformer) Chain {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.transformers = append(c.transformers, transformer)
	c.logger.WithField("transformer", transformer.Name()).Debug("Added transformer to chain")
	return c
}

// Transform applies all transformers in sequence.
// This method is thread-safe and can be called concurrently with Add().
func (c *chain) Transform(ctx context.Context, content []byte, transformCtx Context) ([]byte, error) {
	result := content

	// Take a snapshot of transformers under read lock to allow concurrent transforms
	c.mu.RLock()
	transformers := make([]Transformer, len(c.transformers))
	copy(transformers, c.transformers)
	c.mu.RUnlock()

	c.logger.WithFields(logrus.Fields{
		"source_repo":  transformCtx.SourceRepo,
		"target_repo":  transformCtx.TargetRepo,
		"file_path":    transformCtx.FilePath,
		"transformers": len(transformers),
	}).Debug("Starting transform chain")

	for _, transformer := range transformers {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("transform chain canceled: %w", ctx.Err())
		default:
		}

		c.logger.WithFields(logrus.Fields{
			"transformer": transformer.Name(),
			"file_path":   transformCtx.FilePath,
		}).Debug("Applying transformer")

		transformed, err := transformer.Transform(result, transformCtx)
		if err != nil {
			return nil, fmt.Errorf("transform %s failed: %w", transformer.Name(), err)
		}

		// Use bytes.Equal for efficient comparison without string allocation
		if !bytes.Equal(transformed, result) {
			c.logger.WithFields(logrus.Fields{
				"transformer": transformer.Name(),
				"file_path":   transformCtx.FilePath,
				"size_before": len(result),
				"size_after":  len(transformed),
			}).Debug("Content transformed")
		}

		result = transformed
	}

	return result, nil
}

// Transformers returns a copy of the list of transformers in the chain.
// This method is thread-safe and can be called concurrently.
func (c *chain) Transformers() []Transformer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a copy to prevent external modification
	result := make([]Transformer, len(c.transformers))
	copy(result, c.transformers)
	return result
}
