package transform

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/errors"
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

func TestChain_EmailBeforeRepoTransformer(t *testing.T) {
	// Regression test: Email transformer must run BEFORE repo transformer
	// to prevent repo name in email addresses from being corrupted.
	//
	// Bug scenario:
	// 1. Source file has: "go-broadcast@mrz1818.com"
	// 2. If repo transformer runs first, it replaces "go-broadcast" with target repo name
	//    resulting in: "go-lockfree-queue@mrz1818.com" (WRONG!)
	// 3. Email transformer then can't find "go-broadcast@mrz1818.com" to replace
	//
	// Correct behavior:
	// 1. Email transformer runs first, replaces "go-broadcast@mrz1818.com"
	//    with "security@bsvassociation.org"
	// 2. Repo transformer then runs and doesn't affect the email
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs for test

	chain := NewChain(logger)

	// Add transformers in the correct order: Email FIRST, then Repo
	chain.Add(NewEmailTransformer()).Add(NewRepoTransformer())

	input := `# Security Policy

If you've found a security issue, send a private email to:
📧 [go-broadcast@mrz1818.com](mailto:go-broadcast@mrz1818.com)

This is the go-broadcast repository.
`

	expected := `# Security Policy

If you've found a security issue, send a private email to:
📧 [security@bsvassociation.org](mailto:security@bsvassociation.org)

This is the go-lockfree-queue repository.
`

	ctx := Context{
		SourceRepo:          "mrz1836/go-broadcast",
		TargetRepo:          "bsv-blockchain/go-lockfree-queue",
		FilePath:            ".github/SECURITY.md",
		SourceSecurityEmail: "go-broadcast@mrz1818.com",
		TargetSecurityEmail: "security@bsvassociation.org",
	}

	result, err := chain.Transform(context.Background(), []byte(input), ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, string(result))

	// Verify the email was transformed correctly (not corrupted by repo transformer)
	assert.Contains(t, string(result), "security@bsvassociation.org")
	assert.NotContains(t, string(result), "go-broadcast@mrz1818.com")
	assert.NotContains(t, string(result), "go-lockfree-queue@mrz1818.com") // Should NOT contain corrupted email
}

func TestChain_RepoBeforeEmailTransformer_BreaksEmail(t *testing.T) {
	// Anti-pattern test: Demonstrates what happens when transformers are in WRONG order
	// This test documents the bug behavior to ensure we don't regress
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs for test

	chain := NewChain(logger)

	// Add transformers in the WRONG order: Repo FIRST, then Email
	chain.Add(NewRepoTransformer()).Add(NewEmailTransformer())

	input := `# Security Policy

If you've found a security issue, send a private email to:
📧 [go-broadcast@mrz1818.com](mailto:go-broadcast@mrz1818.com)

This is the go-broadcast repository.
`

	// With WRONG order, the email gets corrupted:
	// 1. Repo transformer replaces "go-broadcast" -> "go-lockfree-queue"
	//    Email becomes: "go-lockfree-queue@mrz1818.com"
	// 2. Email transformer can't find "go-broadcast@mrz1818.com" to replace
	wrongResult := `# Security Policy

If you've found a security issue, send a private email to:
📧 [go-lockfree-queue@mrz1818.com](mailto:go-lockfree-queue@mrz1818.com)

This is the go-lockfree-queue repository.
`

	ctx := Context{
		SourceRepo:          "mrz1836/go-broadcast",
		TargetRepo:          "bsv-blockchain/go-lockfree-queue",
		FilePath:            ".github/SECURITY.md",
		SourceSecurityEmail: "go-broadcast@mrz1818.com",
		TargetSecurityEmail: "security@bsvassociation.org",
	}

	result, err := chain.Transform(context.Background(), []byte(input), ctx)
	require.NoError(t, err)

	// This test verifies the WRONG behavior happens with wrong order
	assert.Equal(t, wrongResult, string(result))
	assert.Contains(t, string(result), "go-lockfree-queue@mrz1818.com")  // Corrupted email
	assert.NotContains(t, string(result), "security@bsvassociation.org") // Email transform failed
}
