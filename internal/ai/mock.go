package ai

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// MockProvider implements Provider interface for testing.
// It uses testify/mock for call tracking and expectation verification.
type MockProvider struct {
	mock.Mock
}

// Ensure MockProvider implements Provider interface.
var _ Provider = (*MockProvider)(nil)

// Name returns the provider identifier.
func (m *MockProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

// GenerateText generates text based on the given prompt.
func (m *MockProvider) GenerateText(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	args := m.Called(ctx, req)
	return testutil.HandleTwoValueReturn[*GenerateResponse](args)
}

// IsAvailable checks if the provider is properly configured and ready.
func (m *MockProvider) IsAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}

// NewMockProvider creates a new MockProvider instance.
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// SetupAvailable configures the mock to return the given availability status.
func (m *MockProvider) SetupAvailable(available bool) *MockProvider {
	m.On("IsAvailable").Return(available)
	return m
}

// SetupName configures the mock to return the given provider name.
func (m *MockProvider) SetupName(name string) *MockProvider {
	m.On("Name").Return(name)
	return m
}

// SetupGenerateText configures the mock to return the given response and error.
func (m *MockProvider) SetupGenerateText(response *GenerateResponse, err error) *MockProvider {
	m.On("GenerateText", mock.Anything, mock.Anything).Return(response, err)
	return m
}

// SetupGenerateTextOnce configures the mock to return the given response and error once.
func (m *MockProvider) SetupGenerateTextOnce(response *GenerateResponse, err error) *MockProvider {
	m.On("GenerateText", mock.Anything, mock.Anything).Return(response, err).Once()
	return m
}

// SetupGenerateTextSequence configures the mock to return responses in sequence.
// Each call will return the next response in the slice.
func (m *MockProvider) SetupGenerateTextSequence(responses []struct {
	Response *GenerateResponse
	Err      error
},
) *MockProvider {
	for _, r := range responses {
		m.On("GenerateText", mock.Anything, mock.Anything).Return(r.Response, r.Err).Once()
	}
	return m
}

// ---- Pre-built mock scenarios for common testing patterns ----

// NewSuccessMock creates a mock provider that always returns a successful response.
func NewSuccessMock(content string) *MockProvider {
	m := NewMockProvider()
	m.SetupAvailable(true)
	m.SetupName("mock")
	m.SetupGenerateText(&GenerateResponse{
		Content:      content,
		TokensUsed:   len(content) / 4, // Approximate token count
		FinishReason: "stop",
	}, nil)
	return m
}

// NewUnavailableMock creates a mock provider that reports as unavailable.
func NewUnavailableMock() *MockProvider {
	m := NewMockProvider()
	m.SetupAvailable(false)
	m.SetupName("mock")
	return m
}

// NewErrorMock creates a mock provider that always returns an error.
func NewErrorMock(err error) *MockProvider {
	m := NewMockProvider()
	m.SetupAvailable(true)
	m.SetupName("mock")
	m.SetupGenerateText(nil, err)
	return m
}

// NewRateLimitMock creates a mock that simulates rate limiting.
// Returns rate limit error on first call, then succeeds.
func NewRateLimitMock(successContent string) *MockProvider {
	m := NewMockProvider()
	m.SetupAvailable(true)
	m.SetupName("mock")
	m.SetupGenerateTextSequence([]struct {
		Response *GenerateResponse
		Err      error
	}{
		{nil, RateLimitError("mock", "1s")},
		{&GenerateResponse{Content: successContent, TokensUsed: len(successContent) / 4, FinishReason: "stop"}, nil},
	})
	return m
}

// NewTimeoutMock creates a mock that simulates a timeout error.
func NewTimeoutMock() *MockProvider {
	m := NewMockProvider()
	m.SetupAvailable(true)
	m.SetupName("mock")
	m.SetupGenerateText(nil, ErrGenerationTimeout)
	return m
}

// NewEmptyResponseMock creates a mock that returns an empty response.
func NewEmptyResponseMock() *MockProvider {
	m := NewMockProvider()
	m.SetupAvailable(true)
	m.SetupName("mock")
	m.SetupGenerateText(&GenerateResponse{
		Content:      "",
		TokensUsed:   0,
		FinishReason: "stop",
	}, nil)
	return m
}
