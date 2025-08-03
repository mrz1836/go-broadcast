package checks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock check for testing
type mockCheck struct {
	name string
	desc string
}

func (m *mockCheck) Name() string {
	return m.name
}

func (m *mockCheck) Description() string {
	return m.desc
}

func (m *mockCheck) Run(_ context.Context, _ []string) error {
	return nil
}

func (m *mockCheck) FilterFiles(files []string) []string {
	// Simple implementation - just return all files
	return files
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)
	assert.NotNil(t, r.checks)

	// Should have built-in checks registered
	checks := r.GetChecks()
	assert.NotEmpty(t, checks)

	// Check for specific built-in checks
	whitespace, ok := r.Get("whitespace")
	assert.True(t, ok)
	assert.NotNil(t, whitespace)

	eof, ok := r.Get("eof")
	assert.True(t, ok)
	assert.NotNil(t, eof)

	fumpt, ok := r.Get("fumpt")
	assert.True(t, ok)
	assert.NotNil(t, fumpt)

	lint, ok := r.Get("lint")
	assert.True(t, ok)
	assert.NotNil(t, lint)

	modTidy, ok := r.Get("mod-tidy")
	assert.True(t, ok)
	assert.NotNil(t, modTidy)
}

func TestRegistry_Register(t *testing.T) {
	r := &Registry{
		checks: make(map[string]Check),
	}

	check := &mockCheck{
		name: "test-check",
		desc: "Test check",
	}

	r.Register(check)

	// Verify check was registered
	got, ok := r.Get("test-check")
	assert.True(t, ok)
	assert.Equal(t, check, got)
}

func TestRegistry_Get(t *testing.T) {
	r := &Registry{
		checks: make(map[string]Check),
	}

	check1 := &mockCheck{name: "check1"}
	check2 := &mockCheck{name: "check2"}

	r.Register(check1)
	r.Register(check2)

	// Test existing checks
	got, ok := r.Get("check1")
	assert.True(t, ok)
	assert.Equal(t, check1, got)

	got, ok = r.Get("check2")
	assert.True(t, ok)
	assert.Equal(t, check2, got)

	// Test non-existent check
	got, ok = r.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistry_GetChecks(t *testing.T) {
	r := &Registry{
		checks: make(map[string]Check),
	}

	check1 := &mockCheck{name: "check1"}
	check2 := &mockCheck{name: "check2"}
	check3 := &mockCheck{name: "check3"}

	r.Register(check1)
	r.Register(check2)
	r.Register(check3)

	checks := r.GetChecks()
	assert.Len(t, checks, 3)

	// Verify all checks are returned (order not guaranteed)
	checkMap := make(map[string]Check)
	for _, c := range checks {
		checkMap[c.Name()] = c
	}

	assert.Equal(t, check1, checkMap["check1"])
	assert.Equal(t, check2, checkMap["check2"])
	assert.Equal(t, check3, checkMap["check3"])
}

func TestCheckInterface(_ *testing.T) {
	// Ensure our mock implements the Check interface
	var _ Check = (*mockCheck)(nil)
}
