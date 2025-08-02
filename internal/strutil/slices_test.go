package strutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmptySlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		expected bool
	}{
		{
			name:     "NilSlice",
			slice:    nil,
			expected: true,
		},
		{
			name:     "EmptySlice",
			slice:    []string{},
			expected: true,
		},
		{
			name:     "NonEmptySlice",
			slice:    []string{"item"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmptySlice(tt.slice)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsEmptySliceWithInts(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		expected bool
	}{
		{
			name:     "NilIntSlice",
			slice:    nil,
			expected: true,
		},
		{
			name:     "EmptyIntSlice",
			slice:    []int{},
			expected: true,
		},
		{
			name:     "NonEmptyIntSlice",
			slice:    []int{1, 2, 3},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmptySlice(tt.slice)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNotEmptySlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		expected bool
	}{
		{
			name:     "NilSlice",
			slice:    nil,
			expected: false,
		},
		{
			name:     "EmptySlice",
			slice:    []string{},
			expected: false,
		},
		{
			name:     "NonEmptySlice",
			slice:    []string{"item"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotEmptySlice(tt.slice)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeSliceAccess(t *testing.T) {
	tests := []struct {
		name          string
		slice         []string
		index         int
		expectedValue string
		expectedOK    bool
	}{
		{
			name:          "ValidIndex",
			slice:         []string{"a", "b", "c"},
			index:         1,
			expectedValue: "b",
			expectedOK:    true,
		},
		{
			name:          "NegativeIndex",
			slice:         []string{"a", "b", "c"},
			index:         -1,
			expectedValue: "",
			expectedOK:    false,
		},
		{
			name:          "IndexTooLarge",
			slice:         []string{"a", "b", "c"},
			index:         5,
			expectedValue: "",
			expectedOK:    false,
		},
		{
			name:          "EmptySlice",
			slice:         []string{},
			index:         0,
			expectedValue: "",
			expectedOK:    false,
		},
		{
			name:          "NilSlice",
			slice:         nil,
			index:         0,
			expectedValue: "",
			expectedOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := SafeSliceAccess(tt.slice, tt.index)
			assert.Equal(t, tt.expectedValue, value)
			assert.Equal(t, tt.expectedOK, ok)
		})
	}
}

func TestSafeSliceAccessWithInts(t *testing.T) {
	slice := []int{10, 20, 30}

	// Valid access
	value, ok := SafeSliceAccess(slice, 0)
	assert.Equal(t, 10, value)
	assert.True(t, ok)

	// Invalid access
	value, ok = SafeSliceAccess(slice, 5)
	assert.Equal(t, 0, value)
	assert.False(t, ok)
}

func TestFilterNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		expected []string
	}{
		{
			name:     "MixedEmptyAndNonEmpty",
			slice:    []string{"hello", "", "world", "   ", "test"},
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "AllEmpty",
			slice:    []string{"", "   ", "\t\n"},
			expected: []string{},
		},
		{
			name:     "AllNonEmpty",
			slice:    []string{"hello", "world", "test"},
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "EmptySlice",
			slice:    []string{},
			expected: nil,
		},
		{
			name:     "NilSlice",
			slice:    nil,
			expected: nil,
		},
		{
			name:     "WithWhitespace",
			slice:    []string{"  hello  ", "world", "  "},
			expected: []string{"hello", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterNonEmpty(tt.slice)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		expected []string
	}{
		{
			name:     "WithDuplicates",
			slice:    []string{"hello", "world", "hello", "test", "world"},
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "NoDuplicates",
			slice:    []string{"hello", "world", "test"},
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "AllSame",
			slice:    []string{"hello", "hello", "hello"},
			expected: []string{"hello"},
		},
		{
			name:     "EmptySlice",
			slice:    []string{},
			expected: nil,
		},
		{
			name:     "NilSlice",
			slice:    nil,
			expected: nil,
		},
		{
			name:     "SingleItem",
			slice:    []string{"hello"},
			expected: []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UniqueStrings(tt.slice)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestChunkSlice(t *testing.T) {
	tests := []struct {
		name      string
		slice     []int
		chunkSize int
		expected  [][]int
	}{
		{
			name:      "EvenChunks",
			slice:     []int{1, 2, 3, 4, 5, 6},
			chunkSize: 2,
			expected:  [][]int{{1, 2}, {3, 4}, {5, 6}},
		},
		{
			name:      "UnevenChunks",
			slice:     []int{1, 2, 3, 4, 5},
			chunkSize: 2,
			expected:  [][]int{{1, 2}, {3, 4}, {5}},
		},
		{
			name:      "ChunkSizeLargerThanSlice",
			slice:     []int{1, 2, 3},
			chunkSize: 5,
			expected:  [][]int{{1, 2, 3}},
		},
		{
			name:      "ChunkSizeOne",
			slice:     []int{1, 2, 3},
			chunkSize: 1,
			expected:  [][]int{{1}, {2}, {3}},
		},
		{
			name:      "EmptySlice",
			slice:     []int{},
			chunkSize: 2,
			expected:  nil,
		},
		{
			name:      "NilSlice",
			slice:     nil,
			chunkSize: 2,
			expected:  nil,
		},
		{
			name:      "ZeroChunkSize",
			slice:     []int{1, 2, 3},
			chunkSize: 0,
			expected:  nil,
		},
		{
			name:      "NegativeChunkSize",
			slice:     []int{1, 2, 3},
			chunkSize: -1,
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ChunkSlice(tt.slice, tt.chunkSize)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestChunkSliceWithStrings(t *testing.T) {
	slice := []string{"a", "b", "c", "d", "e"}
	result := ChunkSlice(slice, 2)
	expected := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
	assert.Equal(t, expected, result)
}
