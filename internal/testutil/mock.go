// Package testutil provides shared testing utilities for file creation and mock handling.
package testutil

import (
	"fmt"

	"github.com/stretchr/testify/mock"
)

// ValidateArgs validates mock arguments count against expected count
func ValidateArgs(args mock.Arguments, expectedCount int) error {
	if len(args) != expectedCount {
		return fmt.Errorf("mock not properly configured: expected %d return values, got %d", expectedCount, len(args)) //nolint:err113 // defensive error for test mock
	}
	return nil
}

// ExtractResult extracts a typed result from mock arguments at the specified index.
// This is used for methods that return (result, error) where result is at index 0.
func ExtractResult[T any](args mock.Arguments, index int) (T, error) {
	var zero T

	if err := ValidateArgs(args, 2); err != nil {
		return zero, err
	}

	if args.Get(index) == nil {
		return zero, args.Error(1)
	}

	result, ok := args.Get(index).(T)
	if !ok {
		return zero, fmt.Errorf("mock result at index %d is not of expected type", index) //nolint:err113 // defensive error for test mock
	}

	return result, args.Error(1)
}

// ExtractError extracts error from mock arguments with single return value validation.
// This is used for methods that return only error.
func ExtractError(args mock.Arguments) error {
	if err := ValidateArgs(args, 1); err != nil {
		return err
	}

	// Handle nil return value (which is a valid error value)
	if args.Get(0) == nil {
		return nil
	}

	// Try to cast to error, fallback to generic error if not possible
	if err, ok := args.Get(0).(error); ok {
		return err
	}

	// If not an error type, return a generic error
	return fmt.Errorf("mock returned non-error type: %T", args.Get(0)) //nolint:err113 // defensive error for test mock
}

// ExtractStringResult extracts string result from mock arguments for methods returning (string, error).
// This handles the fallback pattern for incorrectly configured mocks.
func ExtractStringResult(args mock.Arguments) (string, error) {
	// Check if we have enough arguments to avoid panic
	if len(args) < 2 {
		// Fallback for incorrectly configured mocks
		if len(args) == 1 {
			if err, ok := args.Get(0).(error); ok {
				return "", err
			}
		}
		// Return an error instead of nil,nil to avoid nil pointer dereference
		return "", fmt.Errorf("mock not properly configured: expected 2 return values, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	return args.String(0), args.Error(1)
}

// HandleTwoValueReturn handles the common pattern for methods returning (result, error).
// It includes fallback handling for incorrectly configured mocks.
func HandleTwoValueReturn[T any](args mock.Arguments) (T, error) {
	var zero T

	// Check if we have enough arguments to avoid panic
	if len(args) < 2 {
		// Fallback for incorrectly configured mocks
		if len(args) == 1 {
			if err, ok := args.Get(0).(error); ok {
				return zero, err
			}
		}
		// Return an error instead of nil,nil to avoid nil pointer dereference
		return zero, fmt.Errorf("mock not properly configured: expected 2 return values, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	if args.Get(0) == nil {
		return zero, args.Error(1)
	}

	result, ok := args.Get(0).(T)
	if !ok {
		return zero, fmt.Errorf("mock result is not of expected type") //nolint:err113 // defensive error for test mock
	}

	return result, args.Error(1)
}
