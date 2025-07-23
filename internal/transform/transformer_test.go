package transform

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	// Test Transform method - error case
	transformErr := errors.New("transform failed") //nolint:err113
	mockTransformer.On("Transform", []byte("error"), mock.Anything).Return(nil, transformErr)

	result, err = mockTransformer.Transform([]byte("error"), Context{})
	assert.Error(t, err)
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
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	// Test Transformers method
	transformers := []Transformer{mockTransformer}
	mockChain.On("Transformers").Return(transformers)

	result2 := mockChain.Transformers()
	assert.Equal(t, transformers, result2)

	mockChain.AssertExpectations(t)
}

// Verify interface compliance
var (
	_ Transformer = (*MockTransformer)(nil)
	_ Chain       = (*MockChain)(nil)
)
