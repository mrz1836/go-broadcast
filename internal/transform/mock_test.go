package transform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestMockTransformer_Name(t *testing.T) {
	tests := []struct {
		name         string
		expectedName string
		setupMock    func(*MockTransformer)
	}{
		{
			name:         "returns transformer name",
			expectedName: "test-transformer",
			setupMock: func(m *MockTransformer) {
				m.On("Name").Return("test-transformer")
			},
		},
		{
			name:         "returns empty name",
			expectedName: "",
			setupMock: func(m *MockTransformer) {
				m.On("Name").Return("")
			},
		},
		{
			name:         "returns complex name",
			expectedName: "variable-substitution-transformer",
			setupMock: func(m *MockTransformer) {
				m.On("Name").Return("variable-substitution-transformer")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransformer := &MockTransformer{}
			tt.setupMock(mockTransformer)

			name := mockTransformer.Name()
			assert.Equal(t, tt.expectedName, name)

			mockTransformer.AssertExpectations(t)
		})
	}
}

func TestMockTransformer_Transform(t *testing.T) {
	tests := []struct {
		name           string
		inputContent   []byte
		transformCtx   Context
		expectedOutput []byte
		expectedError  error
		setupMock      func(*MockTransformer)
	}{
		{
			name:         "successful transformation",
			inputContent: []byte("hello world"),
			transformCtx: Context{
				SourceRepo: "org/template",
				TargetRepo: "org/service",
				FilePath:   "README.md",
				Variables: map[string]string{
					"service_name": "test-service",
				},
			},
			expectedOutput: []byte("hello test-service"),
			expectedError:  nil,
			setupMock: func(m *MockTransformer) {
				ctx := Context{
					SourceRepo: "org/template",
					TargetRepo: "org/service",
					FilePath:   "README.md",
					Variables: map[string]string{
						"service_name": "test-service",
					},
				}
				m.On("Transform", []byte("hello world"), ctx).Return([]byte("hello test-service"), nil)
			},
		},
		{
			name:         "transformation with no changes",
			inputContent: []byte("no changes needed"),
			transformCtx: Context{
				SourceRepo: "org/template",
				TargetRepo: "org/service",
				FilePath:   "config.yaml",
			},
			expectedOutput: []byte("no changes needed"),
			expectedError:  nil,
			setupMock: func(m *MockTransformer) {
				ctx := Context{
					SourceRepo: "org/template",
					TargetRepo: "org/service",
					FilePath:   "config.yaml",
				}
				m.On("Transform", []byte("no changes needed"), ctx).Return([]byte("no changes needed"), nil)
			},
		},
		{
			name:         "transformation error",
			inputContent: []byte("invalid content"),
			transformCtx: Context{
				SourceRepo: "org/template",
				TargetRepo: "org/service",
				FilePath:   "invalid.txt",
			},
			expectedOutput: nil,
			expectedError:  assert.AnError,
			setupMock: func(m *MockTransformer) {
				ctx := Context{
					SourceRepo: "org/template",
					TargetRepo: "org/service",
					FilePath:   "invalid.txt",
				}
				m.On("Transform", []byte("invalid content"), ctx).Return(nil, assert.AnError)
			},
		},
		{
			name:         "empty content transformation",
			inputContent: []byte{},
			transformCtx: Context{
				SourceRepo: "org/template",
				TargetRepo: "org/service",
				FilePath:   "empty.txt",
			},
			expectedOutput: []byte{},
			expectedError:  nil,
			setupMock: func(m *MockTransformer) {
				ctx := Context{
					SourceRepo: "org/template",
					TargetRepo: "org/service",
					FilePath:   "empty.txt",
				}
				m.On("Transform", []byte{}, ctx).Return([]byte{}, nil)
			},
		},
		{
			name:         "transformation with complex context",
			inputContent: []byte("{{.service}} deployment"),
			transformCtx: Context{
				SourceRepo: "org/template",
				TargetRepo: "org/microservice",
				FilePath:   "k8s/deployment.yaml",
				Variables: map[string]string{
					"service":   "user-service",
					"namespace": "production",
					"replicas":  "3",
				},
				LogConfig: &logging.LogConfig{
					Debug: logging.DebugFlags{
						Transform: true,
					},
				},
			},
			expectedOutput: []byte("user-service deployment"),
			expectedError:  nil,
			setupMock: func(m *MockTransformer) {
				ctx := Context{
					SourceRepo: "org/template",
					TargetRepo: "org/microservice",
					FilePath:   "k8s/deployment.yaml",
					Variables: map[string]string{
						"service":   "user-service",
						"namespace": "production",
						"replicas":  "3",
					},
					LogConfig: &logging.LogConfig{
						Debug: logging.DebugFlags{
							Transform: true,
						},
					},
				}
				m.On("Transform", []byte("{{.service}} deployment"), ctx).Return([]byte("user-service deployment"), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransformer := &MockTransformer{}
			tt.setupMock(mockTransformer)

			output, err := mockTransformer.Transform(tt.inputContent, tt.transformCtx)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, output)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, output)
			}

			mockTransformer.AssertExpectations(t)
		})
	}
}

func TestMockChain_Add(t *testing.T) {
	tests := []struct {
		name           string
		transformer    Transformer
		expectedReturn Chain
		setupMock      func(*MockChain, *MockTransformer)
	}{
		{
			name:        "successful add returns chain",
			transformer: &MockTransformer{},
			setupMock: func(mc *MockChain, mt *MockTransformer) {
				mc.On("Add", mt).Return(mc)
			},
		},
		{
			name:        "add returns nil chain",
			transformer: &MockTransformer{},
			setupMock: func(mc *MockChain, mt *MockTransformer) {
				mc.On("Add", mt).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChain := &MockChain{}
			mockTransformer := &MockTransformer{}
			tt.setupMock(mockChain, mockTransformer)

			result := mockChain.Add(mockTransformer)

			if tt.name == "add returns nil chain" {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, mockChain, result)
			}

			mockChain.AssertExpectations(t)
			mockTransformer.AssertExpectations(t)
		})
	}
}

func TestMockChain_Transform(t *testing.T) {
	tests := []struct {
		name           string
		inputContent   []byte
		transformCtx   Context
		expectedOutput []byte
		expectedError  error
		setupMock      func(*MockChain)
	}{
		{
			name:         "successful chain transformation",
			inputContent: []byte("original content"),
			transformCtx: Context{
				SourceRepo: "org/template",
				TargetRepo: "org/service",
				FilePath:   "README.md",
			},
			expectedOutput: []byte("transformed content"),
			expectedError:  nil,
			setupMock: func(mc *MockChain) {
				ctx := context.Background()
				transformCtx := Context{
					SourceRepo: "org/template",
					TargetRepo: "org/service",
					FilePath:   "README.md",
				}
				mc.On("Transform", ctx, []byte("original content"), transformCtx).Return([]byte("transformed content"), nil)
			},
		},
		{
			name:         "chain transformation error",
			inputContent: []byte("invalid content"),
			transformCtx: Context{
				SourceRepo: "org/template",
				TargetRepo: "org/service",
				FilePath:   "invalid.txt",
			},
			expectedOutput: nil,
			expectedError:  assert.AnError,
			setupMock: func(mc *MockChain) {
				ctx := context.Background()
				transformCtx := Context{
					SourceRepo: "org/template",
					TargetRepo: "org/service",
					FilePath:   "invalid.txt",
				}
				mc.On("Transform", ctx, []byte("invalid content"), transformCtx).Return(nil, assert.AnError)
			},
		},
		{
			name:         "empty content transformation",
			inputContent: []byte{},
			transformCtx: Context{
				SourceRepo: "org/template",
				TargetRepo: "org/service",
				FilePath:   "empty.txt",
			},
			expectedOutput: []byte{},
			expectedError:  nil,
			setupMock: func(mc *MockChain) {
				ctx := context.Background()
				transformCtx := Context{
					SourceRepo: "org/template",
					TargetRepo: "org/service",
					FilePath:   "empty.txt",
				}
				mc.On("Transform", ctx, []byte{}, transformCtx).Return([]byte{}, nil)
			},
		},
		{
			name:         "context cancellation",
			inputContent: []byte("test content"),
			transformCtx: Context{
				SourceRepo: "org/template",
				TargetRepo: "org/service",
				FilePath:   "test.txt",
			},
			expectedOutput: nil,
			expectedError:  context.Canceled,
			setupMock: func(mc *MockChain) {
				ctx := context.Background()
				transformCtx := Context{
					SourceRepo: "org/template",
					TargetRepo: "org/service",
					FilePath:   "test.txt",
				}
				mc.On("Transform", ctx, []byte("test content"), transformCtx).Return(nil, context.Canceled)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChain := &MockChain{}
			tt.setupMock(mockChain)

			output, err := mockChain.Transform(context.Background(), tt.inputContent, tt.transformCtx)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, output)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, output)
			}

			mockChain.AssertExpectations(t)
		})
	}
}

func TestMockChain_Transformers(t *testing.T) {
	tests := []struct {
		name                 string
		expectedTransformers []Transformer
		setupMock            func(*MockChain)
	}{
		{
			name: "returns list of transformers",
			expectedTransformers: []Transformer{
				&MockTransformer{},
				&MockTransformer{},
			},
			setupMock: func(mc *MockChain) {
				transformers := []Transformer{
					&MockTransformer{},
					&MockTransformer{},
				}
				mc.On("Transformers").Return(transformers)
			},
		},
		{
			name:                 "returns empty transformer list",
			expectedTransformers: []Transformer{},
			setupMock: func(mc *MockChain) {
				mc.On("Transformers").Return([]Transformer{})
			},
		},
		{
			name:                 "returns nil transformer list",
			expectedTransformers: nil,
			setupMock: func(mc *MockChain) {
				mc.On("Transformers").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChain := &MockChain{}
			tt.setupMock(mockChain)

			transformers := mockChain.Transformers()
			assert.Len(t, transformers, len(tt.expectedTransformers))

			mockChain.AssertExpectations(t)
		})
	}
}

func TestMockTransformer_ImplementsInterface(t *testing.T) {
	// Test that MockTransformer implements Transformer interface
	var _ Transformer = (*MockTransformer)(nil)

	// Test instantiation
	mockTransformer := &MockTransformer{}
	require.NotNil(t, mockTransformer)

	// Test that methods exist and can be called
	ctx := Context{
		SourceRepo: "test/repo",
		TargetRepo: "test/target",
		FilePath:   "test.txt",
	}

	mockTransformer.On("Name").Return("test-transformer")
	mockTransformer.On("Transform", []byte("test"), ctx).Return([]byte("transformed"), nil)

	// Verify methods work
	name := mockTransformer.Name()
	assert.Equal(t, "test-transformer", name)

	output, err := mockTransformer.Transform([]byte("test"), ctx)
	require.NoError(t, err)
	assert.Equal(t, []byte("transformed"), output)

	mockTransformer.AssertExpectations(t)
}

func TestMockChain_ImplementsInterface(t *testing.T) {
	// Test that MockChain implements Chain interface
	var _ Chain = (*MockChain)(nil)

	// Test instantiation
	mockChain := &MockChain{}
	require.NotNil(t, mockChain)

	// Test that methods exist and can be called
	ctx := context.Background()
	transformCtx := Context{
		SourceRepo: "test/repo",
		TargetRepo: "test/target",
		FilePath:   "test.txt",
	}
	transformer := &MockTransformer{}

	mockChain.On("Add", transformer).Return(mockChain)
	mockChain.On("Transform", ctx, []byte("test"), transformCtx).Return([]byte("result"), nil)
	mockChain.On("Transformers").Return([]Transformer{transformer})

	// Verify methods work
	result := mockChain.Add(transformer)
	assert.Equal(t, mockChain, result)

	output, err := mockChain.Transform(ctx, []byte("test"), transformCtx)
	require.NoError(t, err)
	assert.Equal(t, []byte("result"), output)

	transformers := mockChain.Transformers()
	assert.Len(t, transformers, 1)
	assert.Equal(t, transformer, transformers[0])

	mockChain.AssertExpectations(t)
}

func TestMockTransformer_NilHandling(t *testing.T) {
	t.Run("Transform method handles nil output correctly", func(t *testing.T) {
		mockTransformer := &MockTransformer{}
		ctx := Context{
			SourceRepo: "test/repo",
			TargetRepo: "test/target",
			FilePath:   "test.txt",
		}

		mockTransformer.On("Transform", []byte("test"), ctx).Return(nil, assert.AnError)

		output, err := mockTransformer.Transform([]byte("test"), ctx)
		require.Error(t, err)
		assert.Nil(t, output)
		mockTransformer.AssertExpectations(t)
	})
}

func TestMockChain_NilHandling(t *testing.T) {
	t.Run("Transform method handles nil output correctly", func(t *testing.T) {
		mockChain := &MockChain{}
		ctx := context.Background()
		transformCtx := Context{
			SourceRepo: "test/repo",
			TargetRepo: "test/target",
			FilePath:   "test.txt",
		}

		mockChain.On("Transform", ctx, []byte("test"), transformCtx).Return(nil, assert.AnError)

		output, err := mockChain.Transform(ctx, []byte("test"), transformCtx)
		require.Error(t, err)
		assert.Nil(t, output)
		mockChain.AssertExpectations(t)
	})

	t.Run("Add method handles nil return correctly", func(t *testing.T) {
		mockChain := &MockChain{}
		transformer := &MockTransformer{}

		mockChain.On("Add", transformer).Return(nil)

		result := mockChain.Add(transformer)
		assert.Nil(t, result)
		mockChain.AssertExpectations(t)
	})
}
