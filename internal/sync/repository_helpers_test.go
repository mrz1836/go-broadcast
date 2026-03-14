package sync

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRepositorySync_IsFileInDirectory(t *testing.T) {
	t.Parallel()

	rs := &RepositorySync{
		logger: logrus.NewEntry(logrus.New()),
	}

	tests := []struct {
		name          string
		filePath      string
		directoryPath string
		expected      bool
	}{
		{"direct match", "src/file.go", "src", true},
		{"subdirectory", "src/foo/bar.go", "src", true},
		{"not in dir", "other/file.go", "src", false},
		{"exact dir name (no slash)", "src", "src", true},
		{"prefix only, no slash", "srcbak/file.go", "src", false},
		{"nested deep", "src/a/b/c/d.go", "src", true},
		{"empty paths", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := rs.isFileInDirectory(tt.filePath, tt.directoryPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRepositorySync_GenerateSyntheticDiff(t *testing.T) {
	t.Parallel()

	rs := &RepositorySync{
		logger: logrus.NewEntry(logrus.New()),
	}

	t.Run("empty slice returns empty string", func(t *testing.T) {
		t.Parallel()

		result := rs.generateSyntheticDiff([]FileChange{})
		assert.Empty(t, result)
	})

	t.Run("new file diff starts with /dev/null header", func(t *testing.T) {
		t.Parallel()

		changes := []FileChange{
			{
				Path:    "newfile.go",
				Content: []byte("package main\n\nfunc main() {}\n"),
				IsNew:   true,
			},
		}
		result := rs.generateSyntheticDiff(changes)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "--- /dev/null")
		assert.Contains(t, result, "newfile.go")
	})

	t.Run("deleted file diff ends with /dev/null header", func(t *testing.T) {
		t.Parallel()

		changes := []FileChange{
			{
				Path:            "oldfile.go",
				OriginalContent: []byte("package main\n\nfunc old() {}\n"),
				IsDeleted:       true,
			},
		}
		result := rs.generateSyntheticDiff(changes)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "+++ /dev/null")
		assert.Contains(t, result, "oldfile.go")
	})

	t.Run("modified file produces non-empty diff", func(t *testing.T) {
		t.Parallel()

		changes := []FileChange{
			{
				Path:            "modified.go",
				OriginalContent: []byte("package main\n\nfunc old() {}\n"),
				Content:         []byte("package main\n\nfunc newFunc() {}\n"),
			},
		}
		result := rs.generateSyntheticDiff(changes)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "modified.go")
	})

	t.Run("identical content produces empty diff", func(t *testing.T) {
		t.Parallel()

		same := []byte("package main\n\nfunc same() {}\n")
		changes := []FileChange{
			{
				Path:            "same.go",
				OriginalContent: same,
				Content:         same,
			},
		}
		result := rs.generateSyntheticDiff(changes)
		assert.Empty(t, result)
	})
}
