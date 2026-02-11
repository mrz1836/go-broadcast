package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONModuleConfigScan_Error tests Scan error paths
func TestJSONModuleConfigScan_Error(t *testing.T) {
	var jmc JSONModuleConfig

	// Test nil value
	err := jmc.Scan(nil)
	require.NoError(t, err)

	// Test invalid JSON
	err = jmc.Scan([]byte(`{invalid json`))
	assert.Error(t, err)
}

// TestJSONStringSliceScan_Error tests Scan error paths
func TestJSONStringSliceScan_Error(t *testing.T) {
	var jss JSONStringSlice

	// Test nil value
	err := jss.Scan(nil)
	require.NoError(t, err)

	// Test invalid JSON
	err = jss.Scan([]byte(`[invalid json`))
	assert.Error(t, err)
}

// TestJSONStringMapScan_Error tests Scan error paths
func TestJSONStringMapScan_Error(t *testing.T) {
	var jsm JSONStringMap

	// Test nil value
	err := jsm.Scan(nil)
	require.NoError(t, err)

	// Test invalid JSON
	err = jsm.Scan([]byte(`{invalid: json}`))
	assert.Error(t, err)
}

// TestMetadataScan_Error tests Scan error paths
func TestMetadataScan_Error(t *testing.T) {
	var meta Metadata

	// Test nil value
	err := meta.Scan(nil)
	require.NoError(t, err)

	// Test invalid JSON
	err = meta.Scan([]byte(`{bad json`))
	assert.Error(t, err)
}

// TestModelsScan_StringBranch tests Scan functions with string input
func TestModelsScan_StringBranch(t *testing.T) {
	t.Run("JSONStringSlice from string", func(t *testing.T) {
		var jss JSONStringSlice
		err := jss.Scan(`["item1","item2"]`)
		require.NoError(t, err)
		assert.Len(t, jss, 2)
		assert.Equal(t, "item1", jss[0])
	})

	t.Run("JSONStringMap from string", func(t *testing.T) {
		var jsm JSONStringMap
		err := jsm.Scan(`{"key":"value"}`)
		require.NoError(t, err)
		assert.Equal(t, "value", jsm["key"])
	})

	t.Run("Metadata from string", func(t *testing.T) {
		var meta Metadata
		err := meta.Scan(`{"key":"value"}`)
		require.NoError(t, err)
		assert.Equal(t, "value", meta["key"])
	})

	t.Run("JSONModuleConfig from string", func(t *testing.T) {
		var jmc JSONModuleConfig
		err := jmc.Scan(`{"type":"go","version":"v1.0.0"}`)
		require.NoError(t, err)
		assert.Equal(t, "go", jmc.Type)
		assert.Equal(t, "v1.0.0", jmc.Version)
	})
}

// TestModelsValue tests Value() methods for JSON types
func TestModelsValue(t *testing.T) {
	t.Run("JSONStringSlice Value", func(t *testing.T) {
		jss := JSONStringSlice{"a", "b", "c"}
		val, err := jss.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)
	})

	t.Run("JSONStringMap Value", func(t *testing.T) {
		jsm := JSONStringMap{"key": "value"}
		val, err := jsm.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)
	})

	t.Run("Metadata Value", func(t *testing.T) {
		meta := Metadata{"key": "value"}
		val, err := meta.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)
	})

	t.Run("JSONModuleConfig Value", func(t *testing.T) {
		jmc := &JSONModuleConfig{
			Type:    "go",
			Version: "v1.0.0",
		}
		val, err := jmc.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)
	})

	t.Run("Nil JSONModuleConfig Value", func(t *testing.T) {
		var jmc *JSONModuleConfig
		val, err := jmc.Value()
		require.NoError(t, err)
		assert.Equal(t, []byte("null"), val)
	})
}
