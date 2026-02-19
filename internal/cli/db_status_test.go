package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterUserTables_NilInput(t *testing.T) {
	t.Parallel()

	result := filterUserTables(nil)
	assert.Nil(t, result)
}

func TestOrderTables_OnlyUnknown(t *testing.T) {
	t.Parallel()

	tableCounts := map[string]int64{
		"custom_table_b": 5,
		"custom_table_a": 3,
	}

	result := orderTables(tableCounts)
	assert.Equal(t, []string{"custom_table_a", "custom_table_b"}, result)
}

func TestOrderTables_Empty(t *testing.T) {
	t.Parallel()

	result := orderTables(map[string]int64{})
	assert.Empty(t, result)
}

func TestPrintStatus(t *testing.T) {
	t.Run("json mode outputs without error", func(t *testing.T) {
		origJSON := dbStatusJSON
		dbStatusJSON = true
		defer func() { dbStatusJSON = origJSON }()

		status := DBStatus{
			Path:        "/test/path.db",
			Exists:      true,
			Size:        1024,
			Version:     "v1.0.0",
			TableCounts: map[string]int64{"repos": 5},
		}

		err := printStatus(status)
		assert.NoError(t, err)
	})

	t.Run("non-existent database returns error", func(t *testing.T) {
		origJSON := dbStatusJSON
		dbStatusJSON = false
		defer func() { dbStatusJSON = origJSON }()

		status := DBStatus{
			Path:   "/nonexistent/path.db",
			Exists: false,
			Error:  "database does not exist (run 'go-broadcast db init' to create)",
		}

		err := printStatus(status)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database does not exist")
	})

	t.Run("database with error", func(t *testing.T) {
		origJSON := dbStatusJSON
		dbStatusJSON = false
		defer func() { dbStatusJSON = origJSON }()

		status := DBStatus{
			Path:   "/test/path.db",
			Exists: true,
			Error:  "failed to open database: corrupted",
		}

		err := printStatus(status)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})

	t.Run("normal status with tables", func(t *testing.T) {
		origJSON := dbStatusJSON
		dbStatusJSON = false
		defer func() { dbStatusJSON = origJSON }()

		modTime := time.Now()
		status := DBStatus{
			Path:         "/test/path.db",
			Exists:       true,
			Size:         2048,
			Version:      "v2.0.0",
			LastModified: &modTime,
			TableCounts: map[string]int64{
				"repos":         10,
				"organizations": 3,
				"clients":       1,
			},
		}

		err := printStatus(status)
		assert.NoError(t, err)
	})

	t.Run("json mode with non-existent db", func(t *testing.T) {
		origJSON := dbStatusJSON
		dbStatusJSON = true
		defer func() { dbStatusJSON = origJSON }()

		status := DBStatus{
			Path:   "/nonexistent/path.db",
			Exists: false,
			Error:  "database does not exist",
		}

		err := printStatus(status)
		assert.NoError(t, err) // JSON mode encodes and returns nil error
	})
}
