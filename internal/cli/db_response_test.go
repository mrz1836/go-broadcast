package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintResponse(t *testing.T) {
	t.Parallel()

	t.Run("json mode outputs without error", func(t *testing.T) {
		t.Parallel()

		resp := CLIResponse{
			Success: true,
			Action:  "created",
			Type:    "group",
			Data:    map[string]string{"id": "test-group"},
			Count:   1,
		}

		err := printResponse(resp, true)
		assert.NoError(t, err)
	})

	t.Run("human readable success", func(t *testing.T) {
		t.Parallel()

		resp := CLIResponse{
			Success: true,
			Action:  "created",
			Type:    "group",
		}

		err := printResponse(resp, false)
		assert.NoError(t, err)
	})

	t.Run("human readable non-success", func(t *testing.T) {
		t.Parallel()

		resp := CLIResponse{
			Success: false,
			Action:  "created",
			Type:    "group",
			Error:   "something failed",
		}

		err := printResponse(resp, false)
		assert.NoError(t, err)
	})

	t.Run("json mode with list data", func(t *testing.T) {
		t.Parallel()

		resp := CLIResponse{
			Success: true,
			Action:  "listed",
			Type:    "target",
			Data:    []string{"a", "b", "c"},
			Count:   3,
		}

		err := printResponse(resp, true)
		assert.NoError(t, err)
	})
}

func TestPrintErrorResponse(t *testing.T) {
	t.Parallel()

	t.Run("json mode outputs structured error", func(t *testing.T) {
		t.Parallel()

		err := printErrorResponse("group", "create", "duplicate name", "use a unique name", true)
		assert.NoError(t, err) // JSON mode writes to stdout, returns nil
	})

	t.Run("human readable without hint", func(t *testing.T) {
		t.Parallel()

		err := printErrorResponse("group", "create", "not found", "", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create: not found")
		assert.NotContains(t, err.Error(), "hint")
	})

	t.Run("human readable with hint", func(t *testing.T) {
		t.Parallel()

		err := printErrorResponse("target", "delete", "not found", "check the target name", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete: not found")
		assert.Contains(t, err.Error(), "hint: check the target name")
	})

	t.Run("json mode with empty hint", func(t *testing.T) {
		t.Parallel()

		err := printErrorResponse("file_list", "attach", "already attached", "", true)
		assert.NoError(t, err)
	})
}
