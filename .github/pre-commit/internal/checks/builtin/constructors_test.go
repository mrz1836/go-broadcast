package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWhitespaceCheck(t *testing.T) {
	check := NewWhitespaceCheck()
	assert.NotNil(t, check)
	assert.IsType(t, &WhitespaceCheck{}, check)
}

func TestNewEOFCheck(t *testing.T) {
	check := NewEOFCheck()
	assert.NotNil(t, check)
	assert.IsType(t, &EOFCheck{}, check)
}
