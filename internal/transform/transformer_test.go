package transform

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Static errors for linting compliance
var (
	errTransformationFailed      = errors.New("transformation failed")
	errChainTransformationFailed = errors.New("chain transformation failed")
)

func TestMockTransformer(t *testing.T) {
	mockTransformer := new(MockTransformer)

	// Test Name method
	mockTransformer.On("Name").Return("test-transformer")
	name := mockTransformer.Name()
	assert.Equal(t, "test-transformer", name)

	// Test Transform method - success case
	input := []byte("original content")
	expected := []byte("transformed content")
	ctx := Context{
		SourceRepo: "org/source",
		TargetRepo: "org/target",
		FilePath:   "test.txt",
		Variables:  map[string]string{"VAR": "value"},
	}

	mockTransformer.On("Transform", input, ctx).Return(expected, nil)

	result, err := mockTransformer.Transform(input, ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, result)

	// Test Transform method - error case
	transformErr := errors.New("transform failed") //nolint:err113 // Test error doesn't need to be a package variable
	mockTransformer.On("Transform", []byte("error"), mock.Anything).Return(nil, transformErr)

	result, err = mockTransformer.Transform([]byte("error"), Context{})
	require.Error(t, err)
	assert.Nil(t, result)

	mockTransformer.AssertExpectations(t)
}

func TestMockChain(t *testing.T) {
	mockChain := new(MockChain)
	mockTransformer := new(MockTransformer)

	// Test Add method
	mockChain.On("Add", mockTransformer).Return(mockChain)
	chain := mockChain.Add(mockTransformer)
	assert.Equal(t, mockChain, chain)

	// Test Transform method
	ctx := context.Background()
	input := []byte("input")
	transformCtx := Context{SourceRepo: "org/source"}
	expected := []byte("output")

	mockChain.On("Transform", ctx, input, transformCtx).Return(expected, nil)

	result, err := mockChain.Transform(ctx, input, transformCtx)
	require.NoError(t, err)
	assert.Equal(t, expected, result)

	// Test Transformers method
	transformers := []Transformer{mockTransformer}
	mockChain.On("Transformers").Return(transformers)

	result2 := mockChain.Transformers()
	assert.Equal(t, transformers, result2)

	mockChain.AssertExpectations(t)
}

// TestContext tests the Context struct functionality
func TestContext(t *testing.T) {
	ctx := Context{
		SourceRepo: "org/template-repo",
		TargetRepo: "org/service-a",
		FilePath:   "README.md",
		Variables: map[string]string{
			"SERVICE_NAME": "service-a",
			"VERSION":      "1.0.0",
		},
		LogConfig: nil,
	}

	// Test field access
	assert.Equal(t, "org/template-repo", ctx.SourceRepo)
	assert.Equal(t, "org/service-a", ctx.TargetRepo)
	assert.Equal(t, "README.md", ctx.FilePath)
	assert.Equal(t, "service-a", ctx.Variables["SERVICE_NAME"])
	assert.Equal(t, "1.0.0", ctx.Variables["VERSION"])
	assert.Nil(t, ctx.LogConfig)
}

// TestContextWithEmptyVariables tests Context with empty variables
func TestContextWithEmptyVariables(t *testing.T) {
	ctx := Context{
		SourceRepo: "org/source",
		TargetRepo: "org/target",
		FilePath:   "file.txt",
		Variables:  map[string]string{},
	}

	assert.NotNil(t, ctx.Variables)
	assert.Empty(t, ctx.Variables)
}

// TestContextWithNilVariables tests Context with nil variables
func TestContextWithNilVariables(t *testing.T) {
	ctx := Context{
		SourceRepo: "org/source",
		TargetRepo: "org/target",
		FilePath:   "file.txt",
		Variables:  nil,
	}

	assert.Nil(t, ctx.Variables)
}

// TestContextEdgeCases tests edge cases for Context fields
func TestContextEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		context Context
	}{
		{
			name: "Empty strings",
			context: Context{
				SourceRepo: "",
				TargetRepo: "",
				FilePath:   "",
				Variables:  map[string]string{},
			},
		},
		{
			name: "Special characters in repo names",
			context: Context{
				SourceRepo: "org-name/repo-name",
				TargetRepo: "org_name/repo_name",
				FilePath:   "path/to/file.md",
				Variables:  map[string]string{},
			},
		},
		{
			name: "Unicode in variables",
			context: Context{
				SourceRepo: "org/repo",
				TargetRepo: "org/target",
				FilePath:   "README.md",
				Variables: map[string]string{
					"UNICODE_VAR": "测试变量", //nolint:gosmopolitan // Testing Unicode handling
					"EMOJI_VAR":   "🚀",
				},
			},
		},
		{
			name: "Long strings",
			context: Context{
				SourceRepo: "very-long-organization-name/very-long-repository-name-with-many-characters",
				TargetRepo: "another-very-long-organization-name/another-very-long-repository-name",
				FilePath:   "very/deep/directory/structure/with/many/levels/file.txt",
				Variables: map[string]string{
					"VERY_LONG_VARIABLE_NAME": "very long variable value with many characters and spaces",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.context

			// Basic field access should work
			assert.Equal(t, tt.context.SourceRepo, ctx.SourceRepo)
			assert.Equal(t, tt.context.TargetRepo, ctx.TargetRepo)
			assert.Equal(t, tt.context.FilePath, ctx.FilePath)

			if tt.context.Variables != nil {
				assert.Len(t, ctx.Variables, len(tt.context.Variables))
				for k, v := range tt.context.Variables {
					assert.Equal(t, v, ctx.Variables[k])
				}
			}
		})
	}
}

// TestTransformerInterface tests that the Transformer interface methods work as expected
func TestTransformerInterface(t *testing.T) {
	mockTransformer := new(MockTransformer)

	// Test Name with different values
	names := []string{"", "simple", "complex-name", "name_with_underscores", "123numeric"}
	for _, name := range names {
		t.Run("Name_"+name, func(t *testing.T) {
			mockTransformer.On("Name").Return(name).Once()
			result := mockTransformer.Name()
			assert.Equal(t, name, result)
		})
	}

	// Test Transform with various input sizes
	testCases := []struct {
		name    string
		content []byte
		ctx     Context
		output  []byte
		err     error
	}{
		{
			name:    "Empty content",
			content: []byte{},
			ctx:     Context{SourceRepo: "org/source", TargetRepo: "org/target"},
			output:  []byte{},
			err:     nil,
		},
		{
			name:    "Small content",
			content: []byte("hello"),
			ctx:     Context{SourceRepo: "org/source", TargetRepo: "org/target"},
			output:  []byte("HELLO"),
			err:     nil,
		},
		{
			name:    "Large content",
			content: make([]byte, 10000),
			ctx:     Context{SourceRepo: "org/source", TargetRepo: "org/target"},
			output:  make([]byte, 10000),
			err:     nil,
		},
		{
			name:    "Binary content",
			content: []byte{0x00, 0xFF, 0x7F, 0x80},
			ctx:     Context{SourceRepo: "org/source", TargetRepo: "org/target"},
			output:  []byte{0xFF, 0x00, 0x80, 0x7F},
			err:     nil,
		},
		{
			name:    "Transform error",
			content: []byte("error trigger"),
			ctx:     Context{SourceRepo: "error/repo", TargetRepo: "error/target"},
			output:  nil,
			err:     errTransformationFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockTransformer.On("Transform", tc.content, tc.ctx).Return(tc.output, tc.err).Once()

			result, err := mockTransformer.Transform(tc.content, tc.ctx)

			if tc.err != nil {
				require.Error(t, err)
				assert.Equal(t, tc.err.Error(), err.Error())
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.output, result)
			}
		})
	}

	mockTransformer.AssertExpectations(t)
}

// TestChainInterface tests that the Chain interface methods work as expected
func TestChainInterface(t *testing.T) {
	mockChain := new(MockChain)

	// Test Add method with multiple transformers
	transformers := []*MockTransformer{
		new(MockTransformer),
		new(MockTransformer),
		new(MockTransformer),
	}

	// Test adding transformers in sequence
	currentChain := mockChain
	for i, transformer := range transformers {
		t.Run("Add_transformer_"+string(rune('0'+i)), func(t *testing.T) {
			mockChain.On("Add", transformer).Return(mockChain).Once()
			result := currentChain.Add(transformer)
			assert.Equal(t, mockChain, result)
		})
	}

	// Test Transformers method
	t.Run("Transformers", func(t *testing.T) {
		expectedTransformers := make([]Transformer, len(transformers))
		for i, t := range transformers {
			expectedTransformers[i] = t
		}

		mockChain.On("Transformers").Return(expectedTransformers).Once()
		result := mockChain.Transformers()
		assert.Equal(t, expectedTransformers, result)
		assert.Len(t, result, len(transformers))
	})

	// Test Transform method with various contexts
	transformTests := []struct {
		name         string
		ctx          context.Context //nolint:containedctx // Test struct needs context for testing purposes
		content      []byte
		transformCtx Context
		output       []byte
		err          error
	}{
		{
			name:    "Simple transform",
			ctx:     context.Background(),
			content: []byte("input"),
			transformCtx: Context{
				SourceRepo: "org/source",
				TargetRepo: "org/target",
				FilePath:   "file.txt",
			},
			output: []byte("output"),
			err:    nil,
		},
		{
			name:    "Transform with variables",
			ctx:     context.Background(),
			content: []byte("template content"),
			transformCtx: Context{
				SourceRepo: "org/template",
				TargetRepo: "org/service",
				FilePath:   "template.md",
				Variables: map[string]string{
					"SERVICE_NAME": "my-service",
					"VERSION":      "2.0.0",
				},
			},
			output: []byte("processed content"),
			err:    nil,
		},
		{
			name:    "Transform with error",
			ctx:     context.Background(),
			content: []byte("bad input"),
			transformCtx: Context{
				SourceRepo: "error/source",
				TargetRepo: "error/target",
			},
			output: nil,
			err:    errChainTransformationFailed,
		},
		{
			name:    "Transform with canceled context",
			ctx:     func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			content: []byte("content"),
			transformCtx: Context{
				SourceRepo: "org/source",
				TargetRepo: "org/target",
			},
			output: nil,
			err:    context.Canceled,
		},
	}

	for _, tc := range transformTests {
		t.Run(tc.name, func(t *testing.T) {
			mockChain.On("Transform", tc.ctx, tc.content, tc.transformCtx).Return(tc.output, tc.err).Once()

			result, err := mockChain.Transform(tc.ctx, tc.content, tc.transformCtx)

			if tc.err != nil {
				require.Error(t, err)
				if errors.Is(tc.err, context.Canceled) {
					assert.Equal(t, context.Canceled, err)
				} else {
					assert.Equal(t, tc.err.Error(), err.Error())
				}
				assert.Equal(t, tc.output, result) // Should return nil in error cases
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.output, result)
			}
		})
	}

	mockChain.AssertExpectations(t)
}

// TestTransformerChaining tests behavior when transformers are chained together
func TestTransformerChaining(t *testing.T) {
	// This test validates the expected behavior when multiple transformers work together
	mockChain := new(MockChain)

	// Create mock transformers
	transformer1 := new(MockTransformer)
	transformer2 := new(MockTransformer)
	transformer3 := new(MockTransformer)

	// Test that adding transformers returns the chain for method chaining
	mockChain.On("Add", transformer1).Return(mockChain).Once()
	mockChain.On("Add", transformer2).Return(mockChain).Once()
	mockChain.On("Add", transformer3).Return(mockChain).Once()

	// Simulate method chaining
	result := mockChain.Add(transformer1).Add(transformer2).Add(transformer3)
	assert.Equal(t, mockChain, result)

	// Test that all transformers are in the chain
	expectedTransformers := []Transformer{transformer1, transformer2, transformer3}
	mockChain.On("Transformers").Return(expectedTransformers).Once()

	transformers := mockChain.Transformers()
	assert.Len(t, transformers, 3)
	assert.Equal(t, expectedTransformers, transformers)

	// Test transformation through the chain
	initialContent := []byte("initial content")
	finalContent := []byte("final transformed content")
	ctx := context.Background()
	transformCtx := Context{
		SourceRepo: "org/source",
		TargetRepo: "org/target",
		FilePath:   "test.md",
		Variables:  map[string]string{"VAR": "value"},
	}

	mockChain.On("Transform", ctx, initialContent, transformCtx).Return(finalContent, nil).Once()

	result2, err := mockChain.Transform(ctx, initialContent, transformCtx)
	require.NoError(t, err)
	assert.Equal(t, finalContent, result2)

	mockChain.AssertExpectations(t)
}

// Verify interface compliance
var (
	_ Transformer = (*MockTransformer)(nil)
	_ Chain       = (*MockChain)(nil)
)
