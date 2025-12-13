// Package logging provides logging configuration and utilities for go-broadcast.
package logging

import (
	"fmt"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestRedactionHook_ConcurrentFire tests that the RedactionHook is safe for concurrent use.
// Run with: go test -race -run TestRedactionHook_ConcurrentFire
func TestRedactionHook_ConcurrentFire(t *testing.T) {
	t.Parallel()

	service := NewRedactionService()
	hook := service.CreateHook()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()

			// Each goroutine creates its own entry to avoid shared state
			// Use valid GitHub token format (ghp_ followed by alphanumeric chars only)
			entry := &logrus.Entry{
				Message: fmt.Sprintf("Secret ghp_abcdef1234567890%d and secret password123", n),
				Data: logrus.Fields{
					"password": "secret",
					"id":       n,
					"token":    fmt.Sprintf("ghp_secretabcdef%d", n),
					"nested": map[string]interface{}{
						"api_key": "sk-1234567890",
						"normal":  "value",
					},
				},
			}

			err := hook.Fire(entry)
			assert.NoError(t, err)

			// Verify redaction occurred
			assert.Contains(t, entry.Message, "ghp_***REDACTED***")
			assert.Equal(t, "***REDACTED***", entry.Data["password"])
		}(i)
	}

	wg.Wait()
}

// TestRedactionService_ConcurrentRedactSensitive tests concurrent calls to RedactSensitive.
// Run with: go test -race -run TestRedactionService_ConcurrentRedactSensitive
func TestRedactionService_ConcurrentRedactSensitive(t *testing.T) {
	t.Parallel()

	service := NewRedactionService()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()

			// Use valid GitHub token format (ghp_ followed by alphanumeric chars only)
			input := fmt.Sprintf("Secret ghp_abcdef1234567890%d and Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", n)
			result := service.RedactSensitive(input)

			assert.Contains(t, result, "ghp_***REDACTED***")
			assert.Contains(t, result, "Bearer ***REDACTED***")
		}(i)
	}

	wg.Wait()
}

// TestRedactionService_ConcurrentIsSensitiveField tests concurrent calls to IsSensitiveField.
// Run with: go test -race -run TestRedactionService_ConcurrentIsSensitiveField
func TestRedactionService_ConcurrentIsSensitiveField(t *testing.T) {
	t.Parallel()

	service := NewRedactionService()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	fieldNames := []string{"password", "token", "api_key", "normal", "data", "secret"}

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()

			fieldName := fieldNames[n%len(fieldNames)]
			result := service.IsSensitiveField(fieldName)

			// Verify expected results for known fields
			switch fieldName {
			case "password", "token", "api_key", "secret":
				assert.True(t, result, "field %s should be sensitive", fieldName)
			case "normal", "data":
				assert.False(t, result, "field %s should not be sensitive", fieldName)
			}
		}(i)
	}

	wg.Wait()
}

// TestGenerateCorrelationID_Concurrent tests concurrent correlation ID generation.
// Run with: go test -race -run TestGenerateCorrelationID_Concurrent
func TestGenerateCorrelationID_Concurrent(t *testing.T) {
	t.Parallel()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	ids := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			id := GenerateCorrelationID()
			ids <- id
		}()
	}

	wg.Wait()
	close(ids)

	// Collect all IDs and verify uniqueness
	seen := make(map[string]bool)
	for id := range ids {
		assert.NotEmpty(t, id, "correlation ID should not be empty")
		assert.False(t, seen[id], "correlation ID should be unique: %s", id)
		seen[id] = true
	}
}

// TestRedactionHook_ConcurrentFireWithNestedMaps tests concurrent redaction of deeply nested data.
// Run with: go test -race -run TestRedactionHook_ConcurrentFireWithNestedMaps
func TestRedactionHook_ConcurrentFireWithNestedMaps(t *testing.T) {
	t.Parallel()

	service := NewRedactionService()
	hook := service.CreateHook()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()

			// Create deeply nested structure
			entry := &logrus.Entry{
				Message: fmt.Sprintf("Processing request %d", n),
				Data: logrus.Fields{
					"level1": map[string]interface{}{
						"level2": map[string]interface{}{
							"level3": map[string]interface{}{
								"password": "secret123",
								"normal":   fmt.Sprintf("value%d", n),
							},
						},
					},
					"items": []interface{}{
						fmt.Sprintf("ghp_abcdef123%d", n),
						"regular_string",
						map[string]interface{}{
							"secret": "hidden",
							"public": "visible",
						},
					},
				},
			}

			err := hook.Fire(entry)
			assert.NoError(t, err)

			// Verify nested redaction occurred
			level1, ok := entry.Data["level1"].(map[string]interface{})
			assert.True(t, ok)
			level2, ok := level1["level2"].(map[string]interface{})
			assert.True(t, ok)
			level3, ok := level2["level3"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "***REDACTED***", level3["password"])
		}(i)
	}

	wg.Wait()
}

// TestAuditLogger_ConcurrentLogging tests concurrent audit logging operations.
// Run with: go test -race -run TestAuditLogger_ConcurrentLogging
func TestAuditLogger_ConcurrentLogging(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 3) // 3 types of audit logs

	for i := 0; i < goroutines; i++ {
		// Test LogAuthentication concurrently
		go func(n int) {
			defer wg.Done()
			logger := NewAuditLogger()
			logger.LogAuthentication(fmt.Sprintf("user%d", n), "token", n%2 == 0)
		}(i)

		// Test LogConfigChange concurrently
		go func(n int) {
			defer wg.Done()
			logger := NewAuditLogger()
			logger.LogConfigChange(fmt.Sprintf("user%d", n), "update", nil)
		}(i)

		// Test LogRepositoryAccess concurrently
		go func(n int) {
			defer wg.Done()
			logger := NewAuditLogger()
			logger.LogRepositoryAccess(fmt.Sprintf("user%d", n), "owner/repo", "clone")
		}(i)
	}

	wg.Wait()
}

// TestRedactionHook_MaxDepthConcurrent tests that depth limiting works under concurrent load.
// Run with: go test -race -run TestRedactionHook_MaxDepthConcurrent
func TestRedactionHook_MaxDepthConcurrent(t *testing.T) {
	t.Parallel()

	service := NewRedactionService()
	hook := service.CreateHook()

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Create a structure that exceeds max depth
	createDeepMap := func(depth int) map[string]interface{} {
		result := map[string]interface{}{"password": "secret"}
		current := result
		for i := 0; i < depth; i++ {
			nested := map[string]interface{}{"password": "secret"}
			current["nested"] = nested
			current = nested
		}
		return result
	}

	for i := 0; i < goroutines; i++ {
		go func(_ int) {
			defer wg.Done()

			// Test with depth exceeding maxRedactDepth (10)
			entry := &logrus.Entry{
				Message: "Deep nesting test",
				Data: logrus.Fields{
					"deep": createDeepMap(15), // Exceeds maxRedactDepth
				},
			}

			err := hook.Fire(entry)
			assert.NoError(t, err, "should handle deep nesting without error")
		}(i)
	}

	wg.Wait()
}
