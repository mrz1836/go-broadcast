package testutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// recordingTB is a stub implementation of testing.TB that records calls to
// Fatal/Fatalf/Error/Errorf instead of aborting the test. It embeds
// testing.TB so it satisfies the interface (including the unexported method),
// while overriding the methods exercised by the Assert* helpers.
//
// The embedded testing.TB is intentionally left nil; any method that is not
// overridden here and gets called will panic, which surfaces unexpected
// behavior during testing rather than silently passing.
type recordingTB struct {
	testing.TB

	helperCalled bool
	fatalCalled  bool
	errorCalled  bool
	lastMessage  string
}

// fatalSentinel is panicked by Fatal/Fatalf to abort execution of the function
// under test, mimicking the runtime.Goexit() behavior of the real *testing.T.
// runAssert recovers it so the calling test continues normally.
type fatalSentinel struct{}

func (r *recordingTB) Helper() { r.helperCalled = true }

func (r *recordingTB) Fatal(args ...interface{}) {
	r.fatalCalled = true
	r.lastMessage = fmt.Sprint(args...)
	panic(fatalSentinel{})
}

func (r *recordingTB) Fatalf(format string, args ...interface{}) {
	r.fatalCalled = true
	r.lastMessage = fmt.Sprintf(format, args...)
	panic(fatalSentinel{})
}

func (r *recordingTB) Error(args ...interface{}) {
	r.errorCalled = true
	r.lastMessage = fmt.Sprint(args...)
}

func (r *recordingTB) Errorf(format string, args ...interface{}) {
	r.errorCalled = true
	r.lastMessage = fmt.Sprintf(format, args...)
}

// runAssert invokes fn against a fresh recordingTB, recovering the fatal
// sentinel so a Fatal/Fatalf call aborts fn (as it would with a real
// *testing.T via runtime.Goexit) without aborting the caller's test.
func runAssert(fn func(t testing.TB)) (stub *recordingTB) {
	stub = &recordingTB{}
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(fatalSentinel); !ok {
				panic(r) // re-panic anything that is not our sentinel
			}
		}
	}()
	fn(stub)
	return stub
}

func TestAssertNoError_Failure(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		msgAndArgs  []interface{}
		wantContain string
	}{
		{
			name:        "error without message",
			err:         errors.New("boom"), //nolint:err113 // test-only error
			wantContain: "unexpected error: boom",
		},
		{
			name:        "error with message",
			err:         errors.New("boom"), //nolint:err113 // test-only error
			msgAndArgs:  []interface{}{"context info"},
			wantContain: "unexpected error: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := runAssert(func(stub testing.TB) {
				AssertNoError(stub, tt.err, tt.msgAndArgs...)
			})

			assert.True(t, stub.helperCalled, "Helper should be called")
			assert.True(t, stub.fatalCalled, "Fatalf should be called on failure")
			assert.Contains(t, stub.lastMessage, tt.wantContain)
		})
	}
}

func TestAssertError_Failure(t *testing.T) {
	tests := []struct {
		name        string
		msgAndArgs  []interface{}
		wantContain string
	}{
		{
			name:        "nil error without message",
			wantContain: "expected error but got nil",
		},
		{
			name:        "nil error with message",
			msgAndArgs:  []interface{}{"context info"},
			wantContain: "expected error but got nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := runAssert(func(stub testing.TB) {
				AssertError(stub, nil, tt.msgAndArgs...)
			})

			assert.True(t, stub.helperCalled, "Helper should be called")
			assert.True(t, stub.fatalCalled, "Fatal/Fatalf should be called on failure")
			assert.Contains(t, stub.lastMessage, tt.wantContain)
		})
	}
}

func TestAssertEqual_Failure(t *testing.T) {
	t.Run("without message", func(t *testing.T) {
		stub := runAssert(func(stub testing.TB) {
			AssertEqual(stub, 42, 43)
		})

		assert.True(t, stub.helperCalled)
		assert.True(t, stub.fatalCalled)
		assert.Contains(t, stub.lastMessage, "expected 42 but got 43")
	})

	t.Run("with message", func(t *testing.T) {
		stub := runAssert(func(stub testing.TB) {
			AssertEqual(stub, "foo", "bar", "extra context")
		})

		assert.True(t, stub.fatalCalled)
		assert.Contains(t, stub.lastMessage, "expected foo but got bar")
		assert.Contains(t, stub.lastMessage, "extra context")
	})
}

func TestAssertNotEqual_Failure(t *testing.T) {
	t.Run("without message", func(t *testing.T) {
		stub := runAssert(func(stub testing.TB) {
			AssertNotEqual(stub, 42, 42)
		})

		assert.True(t, stub.helperCalled)
		assert.True(t, stub.fatalCalled)
		assert.Contains(t, stub.lastMessage, "expected value to not be 42")
	})

	t.Run("with message", func(t *testing.T) {
		stub := runAssert(func(stub testing.TB) {
			AssertNotEqual(stub, "same", "same", "extra context")
		})

		assert.True(t, stub.fatalCalled)
		assert.Contains(t, stub.lastMessage, "expected value to not be same")
		assert.Contains(t, stub.lastMessage, "extra context")
	})
}

func TestAssertErrorContains_Failure(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		stub := runAssert(func(stub testing.TB) {
			AssertErrorContains(stub, nil, "needle")
		})

		assert.True(t, stub.helperCalled)
		assert.True(t, stub.fatalCalled)
		assert.Contains(t, stub.lastMessage, "expected error containing 'needle' but got nil")
	})

	t.Run("message not contained", func(t *testing.T) {
		stub := runAssert(func(stub testing.TB) {
			AssertErrorContains(stub, errors.New("haystack"), "needle") //nolint:err113 // test-only error
		})

		assert.True(t, stub.fatalCalled)
		assert.Contains(t, stub.lastMessage, "expected error to contain 'needle'")
	})
}
