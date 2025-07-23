package transform

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// chain implements the Chain interface
type chain struct {
	transformers []Transformer
	logger       *logrus.Logger
}

// NewChain creates a new transformer chain
func NewChain(logger *logrus.Logger) Chain {
	return &chain{
		transformers: []Transformer{},
		logger:       logger,
	}
}

// Add appends a transformer to the chain
func (c *chain) Add(transformer Transformer) Chain {
	c.transformers = append(c.transformers, transformer)
	c.logger.WithField("transformer", transformer.Name()).Debug("Added transformer to chain")
	return c
}

// Transform applies all transformers in sequence
func (c *chain) Transform(ctx context.Context, content []byte, transformCtx Context) ([]byte, error) {
	result := content

	c.logger.WithFields(logrus.Fields{
		"source_repo":  transformCtx.SourceRepo,
		"target_repo":  transformCtx.TargetRepo,
		"file_path":    transformCtx.FilePath,
		"transformers": len(c.transformers),
	}).Debug("Starting transform chain")

	for _, transformer := range c.transformers {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
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

		if len(transformed) != len(result) || string(transformed) != string(result) {
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

// Transformers returns the list of transformers in the chain
func (c *chain) Transformers() []Transformer {
	// Return a copy to prevent external modification
	result := make([]Transformer, len(c.transformers))
	copy(result, c.transformers)
	return result
}
