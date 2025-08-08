package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSyncBranchName(t *testing.T) {
	tests := []struct {
		name        string
		branchName  string
		expectNil   bool
		expectError bool
		expected    *BranchMetadata
	}{
		{
			name:       "valid sync branch with group ID",
			branchName: "chore/sync-files-default-20240115-120530-abc123def",
			expectNil:  false,
			expected: &BranchMetadata{
				Timestamp: time.Date(2024, 1, 15, 12, 5, 30, 0, time.UTC),
				CommitSHA: "abc123def",
				Prefix:    "chore/sync-files",
				GroupID:   "default",
			},
		},
		{
			name:       "valid sync branch with complex group ID",
			branchName: "chore/sync-files-ci-cd-20240115-120530-abc123def",
			expectNil:  false,
			expected: &BranchMetadata{
				Timestamp: time.Date(2024, 1, 15, 12, 5, 30, 0, time.UTC),
				CommitSHA: "abc123def",
				Prefix:    "chore/sync-files",
				GroupID:   "ci-cd",
			},
		},
		{
			name:       "valid sync branch with full SHA",
			branchName: "chore/sync-files-platform-20240115-235959-1234567890abcdef1234567890abcdef12345678",
			expectNil:  false,
			expected: &BranchMetadata{
				Timestamp: time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC),
				CommitSHA: "1234567890abcdef1234567890abcdef12345678",
				Prefix:    "chore/sync-files",
				GroupID:   "platform",
			},
		},
		{
			name:        "not a sync branch - different prefix",
			branchName:  "feature/new-feature",
			expectNil:   true,
			expectError: true,
		},
		{
			name:        "not a sync branch - main",
			branchName:  "master",
			expectNil:   true,
			expectError: true,
		},
		{
			name:        "invalid format - missing group ID",
			branchName:  "chore/sync-files-20240115-120530-abc123def",
			expectNil:   true,
			expectError: true,
		},
		{
			name:        "invalid format - wrong date format",
			branchName:  "chore/sync-files-default-2024-01-15-120530-abc123def",
			expectNil:   true,
			expectError: true,
		},
		{
			name:        "invalid format - missing commit",
			branchName:  "chore/sync-files-default-20240115-120530",
			expectNil:   true,
			expectError: true,
		},
		{
			name:        "invalid format - extra components",
			branchName:  "chore/sync-files-default-20240115-120530-abc123def-extra",
			expectNil:   true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSyncBranchName(tt.branchName)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expectNil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)
			assert.Equal(t, tt.expected.Timestamp, result.Timestamp)
			assert.Equal(t, tt.expected.CommitSHA, result.CommitSHA)
			assert.Equal(t, tt.expected.Prefix, result.Prefix)
			assert.Equal(t, tt.expected.GroupID, result.GroupID)
		})
	}
}

func TestFormatSyncBranchName(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		groupID   string
		timestamp time.Time
		commitSHA string
		expected  string
	}{
		{
			name:      "with group ID",
			prefix:    "chore/sync-files",
			groupID:   "default",
			timestamp: time.Date(2024, 1, 15, 12, 5, 30, 0, time.UTC),
			commitSHA: "abc123def",
			expected:  "chore/sync-files-default-20240115-120530-abc123def",
		},
		{
			name:      "with complex group ID",
			prefix:    "chore/sync-files",
			groupID:   "ci-cd",
			timestamp: time.Date(2024, 1, 15, 12, 5, 30, 0, time.UTC),
			commitSHA: "abc123def",
			expected:  "chore/sync-files-ci-cd-20240115-120530-abc123def",
		},
		{
			name:      "midnight timestamp",
			prefix:    "chore/sync-files",
			groupID:   "platform",
			timestamp: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			commitSHA: "xyz789",
			expected:  "chore/sync-files-platform-20241231-000000-xyz789",
		},
		{
			name:      "with group ID and full SHA",
			prefix:    "chore/sync-files",
			groupID:   "platform",
			timestamp: time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC),
			commitSHA: "1234567890abcdef1234567890abcdef12345678",
			expected:  "chore/sync-files-platform-20240615-143045-1234567890abcdef1234567890abcdef12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSyncBranchName(tt.prefix, tt.groupID, tt.timestamp, tt.commitSHA)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateBranchPrefix(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		expectError bool
	}{
		{
			name:        "valid prefix",
			prefix:      "chore/sync-files",
			expectError: false,
		},
		{
			name:        "valid with underscore",
			prefix:      "chore/sync-files_v2",
			expectError: false,
		},
		{
			name:        "valid with dash",
			prefix:      "chore/sync-files-prod",
			expectError: false,
		},
		{
			name:        "empty prefix",
			prefix:      "",
			expectError: true,
		},
		{
			name:        "invalid characters - space",
			prefix:      "chore/sync-files prod",
			expectError: true,
		},
		{
			name:        "invalid characters - special",
			prefix:      "chore/sync-files@prod",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranchPrefix(tt.prefix)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBranchParsingRoundTrip(t *testing.T) {
	// Test that formatting and parsing are inverse operations
	prefix := "chore/sync-files"
	groupID := "platform"
	timestamp := time.Date(2024, 3, 15, 9, 30, 0, 0, time.UTC)
	commitSHA := "deadbeef123"

	// Format a branch name with group ID
	branchName := FormatSyncBranchName(prefix, groupID, timestamp, commitSHA)

	// Parse it back
	metadata, err := parseSyncBranchName(branchName)
	require.NoError(t, err)
	assert.NotNil(t, metadata)

	// Verify we get the same values
	assert.Equal(t, prefix, metadata.Prefix)
	assert.Equal(t, groupID, metadata.GroupID)
	assert.Equal(t, timestamp, metadata.Timestamp)
	assert.Equal(t, commitSHA, metadata.CommitSHA)
}
