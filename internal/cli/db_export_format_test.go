package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// richExportConfig is a valid config that uses every kind of value an export
// carries: lists, refs, booleans, numbers, strings that look like other types,
// and characters JSON encoders like to escape.
func richExportConfig() *config.Config {
	cfg := validImportConfig()
	cfg.FileLists[0].Description = `Files for <html> & "quoted" text`
	cfg.DirectoryLists = []config.DirectoryList{
		{
			ID:   "test-dirs",
			Name: "Test Directories",
			Directories: []config.DirectoryMapping{
				{Src: ".github/workflows", Dest: ".github/workflows", Exclude: []string{"ci.yml", "*.tmp"}},
			},
		},
	}
	cfg.Groups[0].Priority = 7
	cfg.Groups[0].Description = "2026-10-22"
	cfg.Groups[0].Targets[0].DirectoryListRefs = []string{"test-dirs"}
	cfg.Groups[0].Targets = append(cfg.Groups[0].Targets, config.TargetConfig{
		Repo:   "mrz1836/target2",
		Branch: "123",
		Files:  []config.FileMapping{{Src: "LICENSE", Dest: "LICENSE"}},
	})
	return cfg
}

// seedExportDB imports cfg into a fresh database and points dbPath at it.
func seedExportDB(t *testing.T, cfg *config.Config) string {
	t.Helper()
	tmp := t.TempDir()
	withDBPath(t, filepath.Join(tmp, "export.db"))
	importFile(t, writeYAML(t, tmp, "sync.yaml", cfg))
	return tmp
}

// importFile runs db import on path, with its messages kept out of the test's output.
func importFile(t *testing.T, path string) {
	t.Helper()
	oldYAML, oldForce := dbImportYAML, dbImportForce
	dbImportYAML, dbImportForce = path, false
	t.Cleanup(func() { dbImportYAML, dbImportForce = oldYAML, oldForce })
	scope := output.CaptureOutput()
	defer scope.Restore()
	require.NoError(t, runDBImport(newTestCmd(), nil))
}

// withExportFlags sets db export's flags for one test and restores them after.
func withExportFlags(t *testing.T, out string, stdout bool, format string) {
	t.Helper()
	oldOut, oldGroup, oldStdout, oldFormat := dbExportOutput, dbExportGroup, dbExportStdout, dbExportFormat
	t.Cleanup(func() {
		dbExportOutput, dbExportGroup, dbExportStdout, dbExportFormat = oldOut, oldGroup, oldStdout, oldFormat
	})
	dbExportOutput, dbExportGroup, dbExportStdout, dbExportFormat = out, "", stdout, format
}

// exportToStdout runs db export --stdout in format and returns stdout and stderr.
func exportToStdout(t *testing.T, format string) (string, string) {
	t.Helper()
	withExportFlags(t, "", true, format)
	scope := output.CaptureOutput()
	defer scope.Restore()
	require.NoError(t, runDBExport(newTestCmd(), nil))
	return scope.Stdout.String(), scope.Stderr.String()
}

// decodeStrict reads a whole exported document the way db import does: every
// field must be one the configuration knows.
func decodeStrict(t *testing.T, doc string) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	dec := yaml.NewDecoder(strings.NewReader(doc))
	dec.KnownFields(true)
	require.NoError(t, dec.Decode(cfg))
	return cfg
}

// TestRunDBExportStdoutCarriesOnlyTheDocument holds that --stdout writes the
// document alone to stdout, in either format, and the progress to stderr.
func TestRunDBExportStdoutCarriesOnlyTheDocument(t *testing.T) { //nolint:paralleltest // mutates global flags/dbPath
	seedExportDB(t, richExportConfig())

	tests := []struct {
		name   string
		format string
		isJSON bool
	}{
		{name: "yaml by default", format: exportFormatYAML},
		{name: "json", format: exportFormatJSON, isJSON: true},
		{name: "format is case-insensitive", format: "JSON", isJSON: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := exportToStdout(t, tt.format)

			assert.NotContains(t, stdout, "Opening database")
			assert.NotContains(t, stdout, "Exporting configuration")
			assert.Contains(t, stderr, "Opening database")
			assert.Contains(t, stderr, "Exporting configuration: test-cfg")
			assert.Equal(t, tt.isJSON, json.Valid([]byte(stdout)), "stdout is JSON: %v", tt.isJSON)

			cfg := decodeStrict(t, stdout)
			assert.Equal(t, "test-cfg", cfg.ID)
			require.Len(t, cfg.Groups, 1)
			assert.Len(t, cfg.Groups[0].Targets, 2)
		})
	}
}

// TestRunDBExportJSONMatchesTheYAML holds that the JSON export is the YAML
// export's document: the same configuration, the same keys in the same order.
func TestRunDBExportJSONMatchesTheYAML(t *testing.T) { //nolint:paralleltest // mutates global flags/dbPath
	seedExportDB(t, richExportConfig())
	yamlOut, _ := exportToStdout(t, exportFormatYAML)
	jsonOut, _ := exportToStdout(t, exportFormatJSON)

	assert.Equal(t, decodeStrict(t, yamlOut), decodeStrict(t, jsonOut))
	assert.Equal(t, topLevelKeys(t, yamlOut), topLevelKeys(t, jsonOut))
	assert.True(t, strings.HasSuffix(jsonOut, "}\n"), "the JSON ends with a newline")

	// Strings that look like other types stay strings, and nothing is HTML-escaped.
	assert.Contains(t, jsonOut, `"description": "2026-10-22"`)
	assert.Contains(t, jsonOut, `"branch": "123"`)
	assert.Contains(t, jsonOut, `"priority": 7`)
	assert.Contains(t, jsonOut, `Files for <html> & \"quoted\" text`)
}

// topLevelKeys are a document's top-level keys, in the order it writes them.
func topLevelKeys(t *testing.T, doc string) []string {
	t.Helper()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(doc), &node))
	require.Len(t, node.Content, 1)
	keys := []string{}
	for i := 0; i < len(node.Content[0].Content); i += 2 {
		keys = append(keys, node.Content[0].Content[i].Value)
	}
	return keys
}

// TestRunDBExportJSONToFileImportsBack holds that --format json --output
// writes JSON that db import reads back into the same configuration.
func TestRunDBExportJSONToFileImportsBack(t *testing.T) { //nolint:paralleltest // mutates global flags/dbPath
	tmp := seedExportDB(t, richExportConfig())
	before, _ := exportToStdout(t, exportFormatYAML)

	jsonPath := filepath.Join(tmp, "backup.json")
	withExportFlags(t, jsonPath, false, exportFormatJSON)
	scope := output.CaptureOutput()
	require.NoError(t, runDBExport(newTestCmd(), nil))
	scope.Restore()
	assert.Contains(t, scope.Stdout.String(), "Exported to: "+jsonPath)

	data, err := os.ReadFile(jsonPath) //nolint:gosec // the test's own file
	require.NoError(t, err)
	require.True(t, json.Valid(data), "the file is JSON")

	// Import the JSON into a fresh database, and export it again.
	withDBPath(t, filepath.Join(t.TempDir(), "reimported.db"))
	importFile(t, jsonPath)
	after, _ := exportToStdout(t, exportFormatYAML)

	assert.Equal(t, before, after)
}

// TestRunDBExportRejectsAnUnknownFormat holds that a format other than yaml
// or json is refused before the database is opened.
func TestRunDBExportRejectsAnUnknownFormat(t *testing.T) { //nolint:paralleltest // mutates global flags/dbPath
	withDBPath(t, filepath.Join(t.TempDir(), "never-opened.db"))
	for _, format := range []string{"xml", "", "yml"} {
		t.Run(format, func(t *testing.T) {
			withExportFlags(t, "", true, format)
			scope := output.CaptureOutput()
			err := runDBExport(newTestCmd(), nil)
			scope.Restore()

			require.ErrorIs(t, err, ErrUnsupportedExportFormat)
			assert.Contains(t, err.Error(), "use yaml or json")
			assert.Empty(t, scope.Stdout.String())
			assert.Empty(t, scope.Stderr.String())
		})
	}
}

// TestConfigToJSONKeepsYAMLsOrder holds that the JSON keeps every key in the
// order the YAML encoding writes it, nested ones included.
func TestConfigToJSONKeepsYAMLsOrder(t *testing.T) {
	cfg := richExportConfig()
	yamlData, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	jsonData, err := configToJSON(cfg)
	require.NoError(t, err)

	var yamlNode, jsonNode yaml.Node
	require.NoError(t, yaml.Unmarshal(yamlData, &yamlNode))
	require.NoError(t, yaml.Unmarshal(jsonData, &jsonNode))
	assert.Equal(t, keyPaths(&yamlNode, ""), keyPaths(&jsonNode, ""))
}

// keyPaths are every mapping key under n, as dotted paths, in document order.
func keyPaths(n *yaml.Node, prefix string) []string {
	var out []string
	switch n.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, c := range n.Content {
			out = append(out, keyPaths(c, prefix)...)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			path := prefix + "." + n.Content[i].Value
			out = append(out, path)
			out = append(out, keyPaths(n.Content[i+1], path)...)
		}
	case yaml.ScalarNode, yaml.AliasNode:
	}
	return out
}

// TestWriteJSONNode covers each kind of YAML node the JSON export meets, and
// the ones it refuses because JSON has no form for them.
func TestWriteJSONNode(t *testing.T) {
	scalar := func(tag, value string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
	}
	mapping := func(content ...*yaml.Node) *yaml.Node {
		return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: content}
	}

	tests := []struct {
		name    string
		node    *yaml.Node
		want    string
		wantErr bool
	}{
		{name: "string", node: scalar("!!str", "a<b>&c"), want: `"a<b>&c"`},
		{name: "null", node: scalar("!!null", "null"), want: `null`},
		{name: "bool", node: scalar("!!bool", "true"), want: `true`},
		{name: "int", node: scalar("!!int", "-42"), want: `-42`},
		{name: "float", node: scalar("!!float", "1.5e3"), want: `1.5e3`},
		{name: "empty sequence", node: &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}, want: `[]`},
		{
			name: "sequence",
			node: &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{scalar("!!int", "1"), scalar("!!str", "2")}},
			want: `[1,"2"]`,
		},
		{name: "mapping keeps order", node: mapping(scalar("!!str", "z"), scalar("!!int", "1"), scalar("!!str", "a"), scalar("!!int", "2")), want: `{"z":1,"a":2}`},
		{name: "document", node: &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{scalar("!!str", "x")}}, want: `"x"`},
		{name: "alias", node: &yaml.Node{Kind: yaml.AliasNode, Alias: scalar("!!str", "x")}, want: `"x"`},
		{name: "infinity", node: scalar("!!float", ".inf"), wantErr: true},
		{name: "not a number", node: scalar("!!float", ".nan"), wantErr: true},
		{name: "hex int", node: scalar("!!int", "0x1F"), wantErr: true},
		{name: "yaml 1.1 bool", node: scalar("!!bool", "yes"), wantErr: true},
		{name: "binary", node: scalar("!!binary", "aGk="), wantErr: true},
		{name: "empty document", node: &yaml.Node{Kind: yaml.DocumentNode}, wantErr: true},
		{name: "non-scalar key", node: mapping(mapping(), scalar("!!int", "1")), wantErr: true},
		{name: "refused value in a mapping", node: mapping(scalar("!!str", "a"), scalar("!!float", ".inf")), wantErr: true},
		{name: "refused value in a sequence", node: &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{scalar("!!float", ".inf")}}, wantErr: true},
		{name: "unknown kind", node: &yaml.Node{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := writeJSONNode(&buf, tt.node)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrUnsupportedYAMLNode)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

// failingWriter refuses every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, os.ErrClosed }

// TestRunDBExportReportsAFailedStdoutWrite holds that a document that can't
// be written to stdout fails the export.
func TestRunDBExportReportsAFailedStdoutWrite(t *testing.T) { //nolint:paralleltest // mutates global flags/dbPath and output
	seedExportDB(t, validImportConfig())
	withExportFlags(t, "", true, exportFormatJSON)
	scope := output.CaptureOutput()
	defer scope.Restore()
	output.SetStdout(failingWriter{})

	err := runDBExport(newTestCmd(), nil)
	require.ErrorIs(t, err, os.ErrClosed)
	assert.Contains(t, err.Error(), "failed to write to stdout")
}
