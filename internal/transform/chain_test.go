package transform

import (
	"context"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestChain_Add(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	chain := NewChain(logger)

	// Add transformers
	transformer1 := &MockTransformer{}
	transformer1.On("Name").Return("transformer1")

	transformer2 := &MockTransformer{}
	transformer2.On("Name").Return("transformer2")

	chain.Add(transformer1).Add(transformer2)

	// Verify transformers were added
	transformers := chain.Transformers()
	assert.Len(t, transformers, 2)
	assert.Equal(t, "transformer1", transformers[0].Name())
	assert.Equal(t, "transformer2", transformers[1].Name())
}

func TestChain_Transform(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (Chain, []byte, Context)
		wantContent string
		wantError   bool
	}{
		{
			name: "successful chain execution",
			setup: func() (Chain, []byte, Context) {
				logger := logrus.New()
				chain := NewChain(logger)

				// First transformer: uppercase
				t1 := &MockTransformer{}
				t1.On("Name").Return("uppercase")
				t1.On("Transform", []byte("hello world"), mock.Anything).
					Return([]byte("HELLO WORLD"), nil)

				// Second transformer: add suffix
				t2 := &MockTransformer{}
				t2.On("Name").Return("suffix")
				t2.On("Transform", []byte("HELLO WORLD"), mock.Anything).
					Return([]byte("HELLO WORLD!"), nil)

				chain.Add(t1).Add(t2)

				ctx := Context{
					SourceRepo: "org/source",
					TargetRepo: "org/target",
					FilePath:   "test.txt",
				}

				return chain, []byte("hello world"), ctx
			},
			wantContent: "HELLO WORLD!",
			wantError:   false,
		},
		{
			name: "empty chain returns original content",
			setup: func() (Chain, []byte, Context) {
				logger := logrus.New()
				chain := NewChain(logger)

				ctx := Context{
					SourceRepo: "org/source",
					TargetRepo: "org/target",
					FilePath:   "test.txt",
				}

				return chain, []byte("unchanged"), ctx
			},
			wantContent: "unchanged",
			wantError:   false,
		},
		{
			name: "transformer error stops chain",
			setup: func() (Chain, []byte, Context) {
				logger := logrus.New()
				chain := NewChain(logger)

				// First transformer: succeeds
				t1 := &MockTransformer{}
				t1.On("Name").Return("success")
				t1.On("Transform", mock.Anything, mock.Anything).
					Return([]byte("modified"), nil)

				// Second transformer: fails
				t2 := &MockTransformer{}
				t2.On("Name").Return("failure")
				t2.On("Transform", mock.Anything, mock.Anything).
					Return(nil, errors.ErrTest)

				// Third transformer: should not be called
				t3 := &MockTransformer{}
				t3.On("Name").Return("never-called")

				chain.Add(t1).Add(t2).Add(t3)

				ctx := Context{
					SourceRepo: "org/source",
					TargetRepo: "org/target",
					FilePath:   "test.txt",
				}

				return chain, []byte("original"), ctx
			},
			wantContent: "",
			wantError:   true,
		},
		{
			name: "context cancellation stops chain",
			setup: func() (Chain, []byte, Context) {
				logger := logrus.New()
				chain := NewChain(logger)

				// Add a transformer that won't be called
				t1 := &MockTransformer{}
				t1.On("Name").Return("never-called")

				chain.Add(t1)

				ctx := Context{
					SourceRepo: "org/source",
					TargetRepo: "org/target",
					FilePath:   "test.txt",
				}

				return chain, []byte("original"), ctx
			},
			wantContent: "",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, content, transformCtx := tt.setup()

			ctx := context.Background()
			if tt.name == "context cancellation stops chain" {
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel() // Cancel immediately

				ctx = cancelCtx
			}

			result, err := chain.Transform(ctx, content, transformCtx)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantContent, string(result))
			}
		})
	}
}

func TestChain_Transformers(t *testing.T) {
	logger := logrus.New()
	chain := NewChain(logger).(*chain)

	// Add transformers
	t1 := &MockTransformer{}
	t2 := &MockTransformer{}
	chain.transformers = []Transformer{t1, t2}

	// Get transformers
	transformers := chain.Transformers()
	assert.Len(t, transformers, 2)

	// Verify it's a copy (modifying returned slice doesn't affect chain)
	transformers[0] = nil
	assert.NotNil(t, chain.transformers[0])
}
