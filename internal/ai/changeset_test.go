package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractChangeset_KeyValueForms(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want []KeyChange
	}{
		{
			name: "env modified with v prefix",
			diff: "diff --git a/.github/env/10-mage-x.env b/.github/env/10-mage-x.env\n" +
				"--- a/.github/env/10-mage-x.env\n" +
				"+++ b/.github/env/10-mage-x.env\n" +
				"@@ -38,3 +38,3 @@\n" +
				" # MAGE-X version\n" +
				"-MAGE_X_VERSION=v1.26.1\n" +
				"+MAGE_X_VERSION=v1.26.3\n",
			want: []KeyChange{
				{File: ".github/env/10-mage-x.env", Key: "MAGE_X_VERSION", Old: "v1.26.1", New: "v1.26.3", Kind: ChangeModified},
			},
		},
		{
			name: "env modified bare numeric",
			diff: "--- a/.github/env/00-core.env\n+++ b/.github/env/00-core.env\n" +
				"@@ -32 +32 @@\n-GOVULNCHECK_GO_VERSION=1.26.6\n+GOVULNCHECK_GO_VERSION=1.27.0\n",
			want: []KeyChange{
				{File: ".github/env/00-core.env", Key: "GOVULNCHECK_GO_VERSION", Old: "1.26.6", New: "1.27.0", Kind: ChangeModified},
			},
		},
		{
			name: "properties colon form (config file)",
			diff: "--- a/app.properties\n+++ b/app.properties\n@@ -1 +1 @@\n-version: 2.8.4\n+version: 2.8.5\n",
			want: []KeyChange{
				{File: "app.properties", Key: "version", Old: "2.8.4", New: "2.8.5", Kind: ChangeModified},
			},
		},
		{
			name: "toml equals with quotes",
			diff: "--- a/config.toml\n+++ b/config.toml\n@@ -1 +1 @@\n-tool = \"v1.0.0\"\n+tool = \"v1.1.0\"\n",
			want: []KeyChange{
				{File: "config.toml", Key: "tool", Old: "\"v1.0.0\"", New: "\"v1.1.0\"", Kind: ChangeModified},
			},
		},
		{
			name: "added key only",
			diff: "--- a/.github/env/10-mage-x.env\n+++ b/.github/env/10-mage-x.env\n" +
				"@@ -40,0 +41 @@\n+MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO=1.26\n",
			want: []KeyChange{
				{File: ".github/env/10-mage-x.env", Key: "MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO", New: "1.26", Kind: ChangeAdded},
			},
		},
		{
			name: "removed key only",
			diff: "--- a/.github/env/10-security.env\n+++ b/.github/env/10-security.env\n" +
				"@@ -5 +4,0 @@\n-GO_SEC_VERSION=v2.21.0\n",
			want: []KeyChange{
				{File: ".github/env/10-security.env", Key: "GO_SEC_VERSION", Old: "v2.21.0", Kind: ChangeRemoved},
			},
		},
		{
			name: "no-op identical value yields nothing",
			diff: "--- a/x.env\n+++ b/x.env\n@@ -1 +1 @@\n-FOO=1.0.0\n+FOO=1.0.0\n",
			want: nil,
		},
		{
			name: "value containing equals sign",
			diff: "--- a/x.env\n+++ b/x.env\n@@ -1 +1 @@\n-LDFLAGS=-X main.version=1.0.0\n+LDFLAGS=-X main.version=2.0.0\n",
			want: []KeyChange{
				{File: "x.env", Key: "LDFLAGS", Old: "-X main.version=1.0.0", New: "-X main.version=2.0.0", Kind: ChangeModified},
			},
		},
		{
			name: "comment lines are ignored",
			diff: "--- a/x.env\n+++ b/x.env\n@@ -1,2 +1,2 @@\n-# old comment\n+# new comment\n-FOO=1\n+FOO=2\n",
			want: []KeyChange{
				{File: "x.env", Key: "FOO", Old: "1", New: "2", Kind: ChangeModified},
			},
		},
		{
			name: "deleted file path via --- a/ header",
			diff: "diff --git a/old.env b/old.env\ndeleted file mode 100644\n--- a/old.env\n+++ /dev/null\n@@ -1 +0,0 @@\n-KEY=v1.0.0\n",
			want: []KeyChange{
				{File: "old.env", Key: "KEY", Old: "v1.0.0", Kind: ChangeRemoved},
			},
		},
		{
			name: "url value not misparsed as colon key",
			diff: "--- a/x.env\n+++ b/x.env\n@@ -1 +1 @@\n-URL=https://example.com/v1\n+URL=https://example.com/v2\n",
			want: []KeyChange{
				{File: "x.env", Key: "URL", Old: "https://example.com/v1", New: "https://example.com/v2", Kind: ChangeModified},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := ExtractChangeset(tt.diff)
			require.NotNil(t, cs)
			assert.Equal(t, tt.want, cs.KeyChanges)
		})
	}
}

func TestExtractChangeset_ExcludesNonConfigFiles(t *testing.T) {
	// YAML (workflows/actions) and code contain shell script and templating that
	// pattern-match as assignments but are NOT config changes. None must be extracted.
	tests := []struct {
		name string
		diff string
	}{
		{
			name: "github actions env block and shell run lines",
			diff: "diff --git a/.github/actions/x/action.yml b/.github/actions/x/action.yml\n" +
				"--- a/.github/actions/x/action.yml\n+++ b/.github/actions/x/action.yml\n" +
				"@@ -1,4 +1,6 @@\n" +
				"+        BENCHSTAT_VERSION: ${{ inputs.benchstat-version }}\n" +
				"+        MATRIX_GO_VERSION: ${{ matrix.go-version }}\n" +
				"+          MIN_MAJOR=${BASH_REMATCH[1]}\n" +
				"+          WINDOWS=$(jq -r '\n",
		},
		{
			name: "workflow yaml",
			diff: "diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml\n" +
				"--- a/.github/workflows/ci.yml\n+++ b/.github/workflows/ci.yml\n" +
				"@@ -1 +1 @@\n-      RACE: ${{ inputs.race || 'false' }}\n+      RACE: ${RACE_ENABLED:-false}\n",
		},
		{
			name: "go source",
			diff: "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n" +
				"@@ -1 +1 @@\n-\tVersion = \"1.0.0\"\n+\tVersion = \"1.1.0\"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := ExtractChangeset(tt.diff)
			assert.Empty(t, cs.KeyChanges, "non-config file must yield no key changes")
		})
	}
}

func TestExtractChangeset_RejectsTemplateAndShellValues(t *testing.T) {
	// Even inside a config file, template/shell expressions are not real values.
	diff := "--- a/.github/env/x.env\n+++ b/.github/env/x.env\n@@ -1,3 +1,3 @@\n" +
		"-A=${{ inputs.a }}\n+A=${{ inputs.b }}\n" +
		"-B=$(compute)\n+B=$(compute-other)\n" +
		"-REAL_VERSION=1.0.0\n+REAL_VERSION=1.1.0\n"
	cs := ExtractChangeset(diff)
	require.Len(t, cs.KeyChanges, 1, "only the plain-valued change should survive")
	assert.Equal(t, "REAL_VERSION", cs.KeyChanges[0].Key)
	assert.Equal(t, "1.1.0", cs.KeyChanges[0].New)
}

func TestIsConfigValueFile(t *testing.T) {
	yes := []string{".github/env/10-mage-x.env", "config.toml", "app.ini", "x.properties", "settings.cfg", "redis.conf"}
	no := []string{".github/workflows/ci.yml", ".github/actions/x/action.yaml", "main.go", "README.md", "config.json", "Dockerfile"}
	for _, p := range yes {
		assert.Truef(t, isConfigValueFile(p), "%s should be a config file", p)
	}
	for _, p := range no {
		assert.Falsef(t, isConfigValueFile(p), "%s should NOT be a config file", p)
	}
}

func TestIsPlainConfigValue(t *testing.T) {
	assert.True(t, isPlainConfigValue("v1.2.3"))
	assert.True(t, isPlainConfigValue("1.26"))
	assert.False(t, isPlainConfigValue("${{ inputs.x }}"))
	assert.False(t, isPlainConfigValue("$(jq -r '"))
	assert.False(t, isPlainConfigValue("`backtick`"))
}

func TestExtractChangeset_MultiFileOrdering(t *testing.T) {
	diff := "diff --git a/a.env b/a.env\n--- a/a.env\n+++ b/a.env\n@@ -1 +1 @@\n-A=1.0.0\n+A=1.1.0\n" +
		"diff --git a/b.env b/b.env\n--- a/b.env\n+++ b/b.env\n@@ -1 +1 @@\n-B=2.0.0\n+B=2.1.0\n"
	cs := ExtractChangeset(diff)
	require.Len(t, cs.KeyChanges, 2)
	assert.Equal(t, "A", cs.KeyChanges[0].Key)
	assert.Equal(t, "a.env", cs.KeyChanges[0].File)
	assert.Equal(t, "B", cs.KeyChanges[1].Key)
	assert.Equal(t, "b.env", cs.KeyChanges[1].File)
}

func TestExtractChangeset_VersionTokenSet(t *testing.T) {
	diff := "--- a/x.env\n+++ b/x.env\n@@ -1,3 +1,3 @@\n" +
		" CONTEXT_VERSION=1.24.0\n" + // context line - still collected
		"-FOO=v1.26.1\n" +
		"+FOO=v1.26.3\n"
	cs := ExtractChangeset(diff)

	for _, tok := range []string{"1.24.0", "1.26.1", "1.26.3"} {
		_, ok := cs.VersionTokens[tok]
		assert.Truef(t, ok, "expected token %q in set", tok)
	}
	// A version not present anywhere must be absent.
	_, ok := cs.VersionTokens["9.9.9"]
	assert.False(t, ok)
}

func TestExtractChangeset_Empty(t *testing.T) {
	for _, d := range []string{"", "   ", "\n\n"} {
		cs := ExtractChangeset(d)
		require.NotNil(t, cs)
		assert.Empty(t, cs.KeyChanges)
		assert.False(t, cs.HasKeyChanges())
	}
}

func TestKeyChange_Bullet(t *testing.T) {
	tests := []struct {
		name string
		kc   KeyChange
		want string
	}{
		{
			name: "modified",
			kc:   KeyChange{File: "a.env", Key: "K", Old: "v1", New: "v2", Kind: ChangeModified},
			want: "* `K`: `v1` → `v2` (`a.env`)",
		},
		{
			name: "added",
			kc:   KeyChange{File: "a.env", Key: "K", New: "v2", Kind: ChangeAdded},
			want: "* Added `K` = `v2` (`a.env`)",
		},
		{
			name: "removed",
			kc:   KeyChange{File: "a.env", Key: "K", Old: "v1", Kind: ChangeRemoved},
			want: "* Removed `K` (`a.env`)",
		},
		{
			name: "no file",
			kc:   KeyChange{Key: "K", Old: "v1", New: "v2", Kind: ChangeModified},
			want: "* `K`: `v1` → `v2`",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.kc.Bullet())
		})
	}
}

func TestRenderVerifiedChanges(t *testing.T) {
	t.Run("empty changeset renders empty", func(t *testing.T) {
		assert.Empty(t, RenderVerifiedChanges(&Changeset{}))
		assert.Empty(t, RenderVerifiedChanges(nil))
	})

	t.Run("renders each change", func(t *testing.T) {
		cs := &Changeset{KeyChanges: []KeyChange{
			{File: "a.env", Key: "MAGE_X_VERSION", Old: "v1.26.1", New: "v1.26.3", Kind: ChangeModified},
		}}
		got := RenderVerifiedChanges(cs)
		assert.Contains(t, got, "MAGE_X_VERSION")
		assert.Contains(t, got, "v1.26.1")
		assert.Contains(t, got, "v1.26.3")
	})

	t.Run("caps at maxVerifiedBullets", func(t *testing.T) {
		var changes []KeyChange
		for i := 0; i < maxVerifiedBullets+10; i++ {
			changes = append(changes, KeyChange{Key: "K", Old: "1.0.0", New: "1.0.1", Kind: ChangeModified})
		}
		got := RenderVerifiedChanges(&Changeset{KeyChanges: changes})
		assert.Contains(t, got, "and 10 more")
	})
}

func TestTruncateValue(t *testing.T) {
	assert.Equal(t, "abc", truncateValue("abc"))
	assert.Equal(t, "abc", truncateValue(`"abc"`))
	assert.Equal(t, "abc", truncateValue("'abc'"))
	long := make([]byte, maxValueDisplayLen+20)
	for i := range long {
		long[i] = 'x'
	}
	got := truncateValue(string(long))
	assert.Len(t, []rune(got), maxValueDisplayLen+1) // +1 for the ellipsis rune
}

func TestKeyChange_IsSignificant(t *testing.T) {
	tests := []struct {
		name string
		kc   KeyChange
		want bool
	}{
		{"env version constant", KeyChange{Key: "MAGE_X_VERSION", Old: "v1.26.1", New: "v1.26.3"}, true},
		{"env constant non-version value", KeyChange{Key: "MAGE_X_BENCHSTAT_VERSION_LATEST", New: "true", Kind: ChangeAdded}, true},
		{"lowercase yaml version value", KeyChange{Key: "version", Old: "2.8.4", New: "2.8.5"}, true},
		{"yaml structural using", KeyChange{Key: "using", New: "composite", Kind: ChangeAdded}, false},
		{"yaml structural shell", KeyChange{Key: "shell", New: "bash", Kind: ChangeAdded}, false},
		{"shell var assignment", KeyChange{Key: "go_minor", New: "$(go env GOVERSION)", Kind: ChangeAdded}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.kc.IsSignificant())
		})
	}
}

func TestRenderVerifiedChanges_FiltersNoise(t *testing.T) {
	cs := &Changeset{KeyChanges: []KeyChange{
		{File: "a.yml", Key: "using", New: "composite", Kind: ChangeAdded},
		{File: "a.yml", Key: "shell", New: "bash", Kind: ChangeAdded},
		{File: ".github/env/10-mage-x.env", Key: "MAGE_X_VERSION", Old: "v1.26.1", New: "v1.26.3", Kind: ChangeModified},
	}}
	got := RenderVerifiedChanges(cs)
	assert.Contains(t, got, "MAGE_X_VERSION")
	assert.NotContains(t, got, "using")
	assert.NotContains(t, got, "shell")
}

func TestNormalizeVersionToken(t *testing.T) {
	assert.Equal(t, "1.26.3", normalizeVersionToken("v1.26.3"))
	assert.Equal(t, "1.26.3", normalizeVersionToken("V1.26.3"))
	assert.Equal(t, "1.26.3", normalizeVersionToken("1.26.3"))
	assert.Equal(t, "1.26.3", normalizeVersionToken(" v1.26.3 "))
}
