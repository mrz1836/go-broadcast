package checks

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/shared"
)

// Advanced mock check with different metadata structures
type mockCheckWithBuiltinMetadata struct {
	name     string
	desc     string
	category string
	deps     []string
}

func (m *mockCheckWithBuiltinMetadata) Name() string        { return m.name }
func (m *mockCheckWithBuiltinMetadata) Description() string { return m.desc }
func (m *mockCheckWithBuiltinMetadata) Run(_ context.Context, _ []string) error {
	return nil
}
func (m *mockCheckWithBuiltinMetadata) FilterFiles(files []string) []string { return files }
func (m *mockCheckWithBuiltinMetadata) Metadata() interface{} {
	// Return a struct that mimics builtin.CheckMetadata
	type builtinMetadata struct {
		Name              string
		Description       string
		FilePatterns      []string
		EstimatedDuration time.Duration
		Dependencies      []string
		DefaultTimeout    time.Duration
		Category          string
		RequiresFiles     bool
	}
	return builtinMetadata{
		Name:              m.name,
		Description:       m.desc,
		FilePatterns:      []string{"*.go"},
		EstimatedDuration: 5 * time.Second,
		Dependencies:      m.deps,
		DefaultTimeout:    30 * time.Second,
		Category:          m.category,
		RequiresFiles:     true,
	}
}

// Mock check that returns pointer metadata
type mockCheckWithPointerMetadata struct {
	name string
}

func (m *mockCheckWithPointerMetadata) Name() string        { return m.name }
func (m *mockCheckWithPointerMetadata) Description() string { return "Pointer metadata check" }
func (m *mockCheckWithPointerMetadata) Run(_ context.Context, _ []string) error {
	return nil
}
func (m *mockCheckWithPointerMetadata) FilterFiles(files []string) []string { return files }
func (m *mockCheckWithPointerMetadata) Metadata() interface{} {
	metadata := &CheckMetadata{
		Name:              m.name,
		Description:       "Pointer metadata check",
		FilePatterns:      []string{"*.txt"},
		EstimatedDuration: 2 * time.Second,
		Dependencies:      []string{},
		DefaultTimeout:    15 * time.Second,
		Category:          "test",
		RequiresFiles:     false,
	}
	return metadata
}

// Mock check with invalid metadata
type mockCheckWithInvalidMetadata struct {
	name string
}

func (m *mockCheckWithInvalidMetadata) Name() string        { return m.name }
func (m *mockCheckWithInvalidMetadata) Description() string { return "Invalid metadata check" }
func (m *mockCheckWithInvalidMetadata) Run(_ context.Context, _ []string) error {
	return nil
}
func (m *mockCheckWithInvalidMetadata) FilterFiles(files []string) []string { return files }
func (m *mockCheckWithInvalidMetadata) Metadata() interface{} {
	// Return a non-struct type
	return "invalid metadata"
}

// Test NewRegistryWithConfig
func TestNewRegistryWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		verify func(t *testing.T, r *Registry)
	}{
		{
			name:   "nil config returns empty registry",
			config: nil,
			verify: func(t *testing.T, r *Registry) {
				assert.NotNil(t, r)
				assert.NotNil(t, r.checks)
				assert.NotNil(t, r.sharedCtx)
				assert.Empty(t, r.GetChecks())
			},
		},
		{
			name: "config with custom timeouts",
			config: &config.Config{
				CheckTimeouts: struct {
					Fumpt      int
					Lint       int
					ModTidy    int
					Whitespace int
					EOF        int
				}{
					Fumpt:      60,
					Lint:       120,
					ModTidy:    45,
					Whitespace: 30,
					EOF:        20,
				},
			},
			verify: func(t *testing.T, r *Registry) {
				assert.NotNil(t, r)
				checks := r.GetChecks()
				assert.Len(t, checks, 5)

				// Verify all expected checks are present
				checkNames := r.Names()
				assert.Contains(t, checkNames, "fumpt")
				assert.Contains(t, checkNames, "lint")
				assert.Contains(t, checkNames, "mod-tidy")
				assert.Contains(t, checkNames, "whitespace")
				assert.Contains(t, checkNames, "eof")
			},
		},
		{
			name: "config with zero timeouts uses defaults",
			config: &config.Config{
				CheckTimeouts: struct {
					Fumpt      int
					Lint       int
					ModTidy    int
					Whitespace int
					EOF        int
				}{
					Fumpt:      0,
					Lint:       0,
					ModTidy:    0,
					Whitespace: 0,
					EOF:        0,
				},
			},
			verify: func(t *testing.T, r *Registry) {
				assert.NotNil(t, r)
				checks := r.GetChecks()
				assert.Len(t, checks, 5)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistryWithConfig(tt.config)
			tt.verify(t, r)
		})
	}
}

// Test Names function
func TestRegistry_Names(t *testing.T) {
	r := &Registry{
		checks: make(map[string]Check),
	}

	// Empty registry
	names := r.Names()
	assert.Empty(t, names)

	// Add checks
	check1 := &mockCheck{name: "check1"}
	check2 := &mockCheck{name: "check2"}
	check3 := &mockCheck{name: "check3"}

	r.Register(check1)
	r.Register(check2)
	r.Register(check3)

	names = r.Names()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "check1")
	assert.Contains(t, names, "check2")
	assert.Contains(t, names, "check3")
}

// Test GetMetadata function
func TestRegistry_GetMetadata(t *testing.T) {
	r := &Registry{
		checks: make(map[string]Check),
	}

	check := &mockCheck{
		name: "test-check",
		desc: "Test check description",
	}
	r.Register(check)

	// Get existing metadata
	metadata, ok := r.GetMetadata("test-check")
	assert.True(t, ok)
	assert.Equal(t, "test-check", metadata.Name)
	assert.Equal(t, "Test check description", metadata.Description)
	assert.Equal(t, "test", metadata.Category)

	// Get non-existent metadata
	metadata, ok = r.GetMetadata("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, CheckMetadata{}, metadata)
}

// Test GetAllMetadata function
func TestRegistry_GetAllMetadata(t *testing.T) {
	r := &Registry{
		checks: make(map[string]Check),
	}

	// Empty registry
	metadata := r.GetAllMetadata()
	assert.Empty(t, metadata)

	// Add checks
	check1 := &mockCheck{name: "check1", desc: "Check 1"}
	check2 := &mockCheckWithBuiltinMetadata{
		name:     "check2",
		desc:     "Check 2",
		category: "formatting",
	}
	check3 := &mockCheckWithPointerMetadata{name: "check3"}

	r.Register(check1)
	r.Register(check2)
	r.Register(check3)

	metadata = r.GetAllMetadata()
	assert.Len(t, metadata, 3)

	// Verify all metadata is present
	metadataMap := make(map[string]CheckMetadata)
	for _, m := range metadata {
		metadataMap[m.Name] = m
	}

	assert.Equal(t, "Check 1", metadataMap["check1"].Description)
	assert.Equal(t, "Check 2", metadataMap["check2"].Description)
	assert.Equal(t, "formatting", metadataMap["check2"].Category)
	assert.Equal(t, "Pointer metadata check", metadataMap["check3"].Description)
}

// Test GetChecksByCategory function
func TestRegistry_GetChecksByCategory(t *testing.T) {
	r := &Registry{
		checks: make(map[string]Check),
	}

	// Add checks with different categories
	check1 := &mockCheckWithBuiltinMetadata{
		name:     "format1",
		category: "formatting",
	}
	check2 := &mockCheckWithBuiltinMetadata{
		name:     "format2",
		category: "formatting",
	}
	check3 := &mockCheckWithBuiltinMetadata{
		name:     "lint1",
		category: "linting",
	}
	check4 := &mockCheck{name: "test1"} // category: "test"

	r.Register(check1)
	r.Register(check2)
	r.Register(check3)
	r.Register(check4)

	// Get formatting checks
	formattingChecks := r.GetChecksByCategory("formatting")
	assert.Len(t, formattingChecks, 2)
	checkNames := []string{}
	for _, c := range formattingChecks {
		checkNames = append(checkNames, c.Name())
	}
	assert.Contains(t, checkNames, "format1")
	assert.Contains(t, checkNames, "format2")

	// Get linting checks
	lintingChecks := r.GetChecksByCategory("linting")
	assert.Len(t, lintingChecks, 1)
	assert.Equal(t, "lint1", lintingChecks[0].Name())

	// Get test checks
	testChecks := r.GetChecksByCategory("test")
	assert.Len(t, testChecks, 1)
	assert.Equal(t, "test1", testChecks[0].Name())

	// Get non-existent category
	emptyChecks := r.GetChecksByCategory("nonexistent")
	assert.Empty(t, emptyChecks)
}

// Test ValidateCheckDependencies function
func TestRegistry_ValidateCheckDependencies(t *testing.T) {
	// For this test, we'll create checks with dependencies
	// The actual make targets won't exist in the test environment,
	// so we expect errors for all dependencies
	r := &Registry{
		checks:    make(map[string]Check),
		sharedCtx: shared.NewContext(),
	}

	// Add checks with dependencies
	check1 := &mockCheckWithBuiltinMetadata{
		name:     "check1",
		deps:     []string{"test-target-1"}, // Will not exist
		category: "test",
	}
	check2 := &mockCheckWithBuiltinMetadata{
		name:     "check2",
		deps:     []string{"test-target-2"}, // Will not exist
		category: "test",
	}
	check3 := &mockCheckWithBuiltinMetadata{
		name:     "check3",
		deps:     []string{"test-target-1", "test-target-3"}, // Multiple missing
		category: "test",
	}
	check4 := &mockCheckWithBuiltinMetadata{
		name:     "check4",
		deps:     []string{}, // No dependencies
		category: "test",
	}

	r.Register(check1)
	r.Register(check2)
	r.Register(check3)
	r.Register(check4)

	ctx := context.Background()
	errors := r.ValidateCheckDependencies(ctx)

	// Should have 4 errors (3 checks with missing dependencies, check4 has no deps)
	assert.Len(t, errors, 4)

	// Verify error messages
	errorMessages := []string{}
	for _, err := range errors {
		errorMessages = append(errorMessages, err.Error())
	}

	assert.Contains(t, errorMessages, "check 'check1' requires make_target 'test-target-1' which is not available")
	assert.Contains(t, errorMessages, "check 'check2' requires make_target 'test-target-2' which is not available")
	// Check3 should have 2 errors
	count := 0
	for _, msg := range errorMessages {
		if strings.Contains(msg, "check 'check3'") {
			count++
		}
	}
	assert.Equal(t, 2, count, "check3 should have 2 dependency errors")
}

// Test GetEstimatedDuration function
func TestRegistry_GetEstimatedDuration(t *testing.T) {
	r := &Registry{
		checks: make(map[string]Check),
	}

	// Empty registry
	duration := r.GetEstimatedDuration()
	assert.Equal(t, time.Duration(0), duration)

	// Add checks with different durations
	check1 := &mockCheck{name: "check1"} // 1 nanosecond from mock
	check2 := &mockCheckWithBuiltinMetadata{
		name:     "check2",
		category: "test",
	} // 5 seconds
	check3 := &mockCheckWithPointerMetadata{name: "check3"} // 2 seconds

	r.Register(check1)
	r.Register(check2)
	r.Register(check3)

	duration = r.GetEstimatedDuration()
	expectedDuration := time.Duration(1) + 5*time.Second + 2*time.Second
	assert.Equal(t, expectedDuration, duration)
}

// Test CheckDependencyError
func TestCheckDependencyError(t *testing.T) {
	err := &CheckDependencyError{
		CheckName:      "test-check",
		Dependency:     "make-target",
		DependencyType: "make_target",
	}

	assert.Equal(t, "check 'test-check' requires make_target 'make-target' which is not available", err.Error())
}

// Test metadata conversion with various types
func TestRegistry_ConvertMetadata(t *testing.T) {
	r := &Registry{
		checks: make(map[string]Check),
	}

	tests := []struct {
		name     string
		check    Check
		validate func(t *testing.T, metadata CheckMetadata)
	}{
		{
			name:  "direct CheckMetadata type",
			check: &mockCheck{name: "direct", desc: "Direct metadata"},
			validate: func(t *testing.T, metadata CheckMetadata) {
				assert.Equal(t, "direct", metadata.Name)
				assert.Equal(t, "Direct metadata", metadata.Description)
				assert.Equal(t, "test", metadata.Category)
			},
		},
		{
			name: "builtin-style metadata struct",
			check: &mockCheckWithBuiltinMetadata{
				name:     "builtin",
				desc:     "Builtin metadata",
				category: "formatting",
				deps:     []string{"make-target"},
			},
			validate: func(t *testing.T, metadata CheckMetadata) {
				assert.Equal(t, "builtin", metadata.Name)
				assert.Equal(t, "Builtin metadata", metadata.Description)
				assert.Equal(t, "formatting", metadata.Category)
				assert.Equal(t, []string{"make-target"}, metadata.Dependencies)
				assert.Equal(t, 5*time.Second, metadata.EstimatedDuration)
			},
		},
		{
			name:  "pointer metadata",
			check: &mockCheckWithPointerMetadata{name: "pointer"},
			validate: func(t *testing.T, metadata CheckMetadata) {
				assert.Equal(t, "pointer", metadata.Name)
				assert.Equal(t, "Pointer metadata check", metadata.Description)
				assert.Equal(t, "test", metadata.Category)
				assert.False(t, metadata.RequiresFiles)
			},
		},
		{
			name:  "invalid metadata type",
			check: &mockCheckWithInvalidMetadata{name: "invalid"},
			validate: func(t *testing.T, metadata CheckMetadata) {
				assert.Equal(t, "unknown", metadata.Name)
				assert.Equal(t, "Unknown check", metadata.Description)
				assert.Equal(t, "unknown", metadata.Category)
				assert.True(t, metadata.RequiresFiles)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r.Register(tt.check)
			metadata, ok := r.GetMetadata(tt.check.Name())
			require.True(t, ok)
			tt.validate(t, metadata)
		})
	}
}

// Test concurrent access to registry
func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()

	// Number of goroutines
	const numGoroutines = 100

	// Wait group to synchronize
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 4) // 4 operations per goroutine

	// Channel to collect errors
	errors := make(chan error, numGoroutines*4)

	for i := 0; i < numGoroutines; i++ {

		// Register checks
		go func() {
			defer wg.Done()
			check := &mockCheck{
				name: fmt.Sprintf("check-%d", i),
				desc: fmt.Sprintf("Check %d", i),
			}
			r.Register(check)
		}()

		// Get checks
		go func() {
			defer wg.Done()
			checkName := fmt.Sprintf("check-%d", i%10) // Try to get some that might not exist yet
			_, _ = r.Get(checkName)
		}()

		// Get all checks
		go func() {
			defer wg.Done()
			_ = r.GetChecks()
		}()

		// Get metadata
		go func() {
			defer wg.Done()
			_ = r.GetAllMetadata()
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}

	// Verify some checks were registered
	checks := r.GetChecks()
	assert.NotEmpty(t, checks)
}

// Benchmark registry operations
func BenchmarkRegistry_Register(b *testing.B) {
	r := NewRegistry()
	check := &mockCheck{name: "bench-check", desc: "Benchmark check"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Register(check)
	}
}

func BenchmarkRegistry_Get(b *testing.B) {
	r := NewRegistry()
	// Pre-populate with checks
	for i := 0; i < 100; i++ {
		check := &mockCheck{
			name: fmt.Sprintf("check-%d", i),
			desc: fmt.Sprintf("Check %d", i),
		}
		r.Register(check)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Get(fmt.Sprintf("check-%d", i%100))
	}
}

func BenchmarkRegistry_GetAllMetadata(b *testing.B) {
	r := NewRegistry()
	// Pre-populate with checks
	for i := 0; i < 50; i++ {
		check := &mockCheck{
			name: fmt.Sprintf("check-%d", i),
			desc: fmt.Sprintf("Check %d", i),
		}
		r.Register(check)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.GetAllMetadata()
	}
}

// Example usage of registry
func ExampleRegistry_GetChecksByCategory() {
	r := NewRegistry()

	// Get all formatting checks
	formattingChecks := r.GetChecksByCategory("formatting")
	for _, check := range formattingChecks {
		fmt.Printf("Formatting check: %s\n", check.Name())
	}

	// Get all linting checks
	lintingChecks := r.GetChecksByCategory("linting")
	for _, check := range lintingChecks {
		fmt.Printf("Linting check: %s\n", check.Name())
	}
}

func ExampleRegistry_ValidateCheckDependencies() {
	r := NewRegistry()
	ctx := context.Background()

	// Validate all check dependencies
	errors := r.ValidateCheckDependencies(ctx)
	if len(errors) > 0 {
		fmt.Println("Dependency validation errors:")
		for _, err := range errors {
			fmt.Printf("- %v\n", err)
		}
	} else {
		fmt.Println("All check dependencies are satisfied")
	}
}
