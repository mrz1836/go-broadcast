package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportConfigCallsiteAudit is a forward-protection static check that enumerates
// every production caller of converter.ExportConfig in the internal/cli package and
// asserts that each is in an explicit allowlist (AC-14).
//
// The allowlist has three entries, each with a documented classification:
//
//	sync.go      — loadConfigFromDB calls ExportConfig and immediately follows with
//	               config.ApplyDefaultsAndResolve (needs-resolve site).
//	db_diff.go   — calls ExportConfig then ApplyDefaultsAndResolve so DB and YAML
//	               sides are both resolved before compareConfigs (needs-resolve site).
//	db_export.go — deliberate exception: preserves file_list_refs and
//	               directory_list_refs un-inlined for round-trip authoring
//	               (must-stay-unresolved; locked in by TestDBExport_PreservesListRefs_Unresolved).
//
// A new production caller of ExportConfig will fail this test until it is explicitly
// added here with its classification, preventing silent regression to the unresolved
// --from-db behavior that this PR fixes.
func TestExportConfigCallsiteAudit(t *testing.T) {
	allowlist := map[string]string{
		"sync.go":      "needs-resolve: loadConfigFromDB calls ExportConfig + ApplyDefaultsAndResolve",
		"db_diff.go":   "needs-resolve: db_diff calls ExportConfig + ApplyDefaultsAndResolve",
		"db_export.go": "must-stay-unresolved: deliberate exception for round-trip authoring shape",
	}

	// The working directory for tests is the package directory (internal/cli/),
	// so os.ReadDir(".") enumerates the production source files.
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	fset := token.NewFileSet()
	callerFiles := map[string]bool{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		f, parseErr := parser.ParseFile(fset, name, nil, 0)
		require.NoError(t, parseErr, "failed to parse production file %s", name)

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name == "ExportConfig" {
				callerFiles[name] = true
			}
			return true
		})
	}

	// Every discovered caller must be in the allowlist.
	for file := range callerFiles {
		classification, ok := allowlist[file]
		assert.True(t, ok,
			"unexpected production caller of converter.ExportConfig in %s — "+
				"add it to the allowlist in TestExportConfigCallsiteAudit with an explicit "+
				"classification (needs-resolve or must-stay-unresolved)", file)
		if ok {
			assert.NotEmpty(t, classification)
		}
	}

	// Every allowlist entry must actually exist as a caller (guards against stale entries).
	for file, classification := range allowlist {
		assert.True(t, callerFiles[file],
			"allowlist entry %q (%s) is stale — no call to ExportConfig found in that file; "+
				"remove the entry if the call was refactored away", file, classification)
	}
}
