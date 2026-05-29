package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPrintValidationResult covers the JSON and human-readable branches of
// printValidationResult for both valid and invalid results.
func TestPrintValidationResult(t *testing.T) { //nolint:paralleltest // mutates global dbValidateJSON
	old := dbValidateJSON
	t.Cleanup(func() { dbValidateJSON = old })

	valid := ValidationResult{
		Valid:  true,
		Checks: []ValidationCheck{{Type: "config", Message: "ok"}},
	}
	invalid := ValidationResult{
		Valid:  false,
		Checks: []ValidationCheck{{Type: "config", Message: "ok"}},
		Errors: []ValidationError{
			{Type: "orphan", Message: "bad ref", Details: "target_id=1"},
			{Type: "orphan2", Message: "another"},
		},
	}

	t.Run("human valid", func(t *testing.T) {
		dbValidateJSON = false
		require.NoError(t, printValidationResult(valid))
	})

	t.Run("human invalid returns error", func(t *testing.T) {
		dbValidateJSON = false
		err := printValidationResult(invalid)
		require.Error(t, err)
	})

	t.Run("json valid", func(t *testing.T) {
		dbValidateJSON = true
		require.NoError(t, printValidationResult(valid))
	})

	t.Run("json invalid", func(t *testing.T) {
		dbValidateJSON = true
		// JSON mode returns nil even for invalid results (output is the payload).
		require.NoError(t, printValidationResult(invalid))
	})
}
