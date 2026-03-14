package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBoolPtr tests the boolPtr helper function
func TestBoolPtr(t *testing.T) {
	// Test true
	truePtr := boolPtr(true)
	require.NotNil(t, truePtr)
	assert.True(t, *truePtr)

	// Test false
	falsePtr := boolPtr(false)
	require.NotNil(t, falsePtr)
	assert.False(t, *falsePtr)
}

// TestBoolVal_NilPointer tests the nil case of boolVal
func TestBoolVal_NilPointer(t *testing.T) {
	// Test nil with true default
	result := boolVal(nil, true)
	assert.True(t, result)

	// Test nil with false default
	result = boolVal(nil, false)
	assert.False(t, result)
}
