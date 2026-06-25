package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResolveClonedEmail exercises the full rebase/verbatim/fallback matrix for a
// single cloned contact email column. Each case is labeled with the column it
// represents (security_email or support_email) so both columns share — and are
// proven to follow — the same resolver behavior.
func TestResolveClonedEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		column       string // security_email or support_email — both columns use this resolver
		sourceEmail  string
		srcShortName string
		dstShortName string
		wantResolved string
		wantWarning  bool
	}{
		{
			name:         "per-repo rebase security_email",
			column:       "security_email",
			sourceEmail:  "service-a@example.com",
			srcShortName: "service-a",
			dstShortName: "service-b",
			wantResolved: "service-b@example.com",
			wantWarning:  false,
		},
		{
			name:         "per-repo rebase support_email",
			column:       "support_email",
			sourceEmail:  "service-a@example.com",
			srcShortName: "service-a",
			dstShortName: "service-c",
			wantResolved: "service-c@example.com",
			wantWarning:  false,
		},
		{
			name:         "shared org email copied verbatim security_email",
			column:       "security_email",
			sourceEmail:  "security@example.org",
			srcShortName: "service-a",
			dstShortName: "service-b",
			wantResolved: "security@example.org",
			wantWarning:  false,
		},
		{
			name:         "shared project email copied verbatim support_email",
			column:       "support_email",
			sourceEmail:  "contact@example.net",
			srcShortName: "service-a",
			dstShortName: "service-b",
			wantResolved: "contact@example.net",
			wantWarning:  false,
		},
		{
			name:         "cross-name email copied verbatim security_email",
			column:       "security_email",
			sourceEmail:  "team@example.com",
			srcShortName: "service-a",
			dstShortName: "service-b",
			wantResolved: "team@example.com",
			wantWarning:  false,
		},
		{
			name:         "empty source copied empty support_email",
			column:       "support_email",
			sourceEmail:  "",
			srcShortName: "service-a",
			dstShortName: "service-b",
			wantResolved: "",
			wantWarning:  false,
		},
		{
			name:         "rebase no-op when source and dest short names match security_email",
			column:       "security_email",
			sourceEmail:  "service-a@example.com",
			srcShortName: "service-a",
			dstShortName: "service-a",
			wantResolved: "service-a@example.com",
			wantWarning:  false,
		},
		{
			name:         "invalid rebase falls back to verbatim and warns support_email",
			column:       "support_email",
			sourceEmail:  "service-a@example.com",
			srcShortName: "service-a",
			dstShortName: ".github", // leading dot -> invalid local-part
			wantResolved: "service-a@example.com",
			wantWarning:  true,
		},
		{
			name:         "source without @ copied verbatim security_email",
			column:       "security_email",
			sourceEmail:  "not-an-email",
			srcShortName: "service-a",
			dstShortName: "service-b",
			wantResolved: "not-an-email",
			wantWarning:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotResolved, gotWarning := resolveClonedEmail(tt.sourceEmail, tt.srcShortName, tt.dstShortName)

			assert.Equalf(t, tt.wantResolved, gotResolved, "resolved %s mismatch", tt.column)
			if tt.wantWarning {
				assert.NotEmptyf(t, gotWarning, "expected a warning for %s", tt.column)
			} else {
				assert.Emptyf(t, gotWarning, "expected no warning for %s", tt.column)
			}
		})
	}
}

// TestRepoShortName verifies that the short name is the segment after the final "/".
func TestRepoShortName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fullName string
		want     string
	}{
		{name: "org/repo", fullName: "org/service-a", want: "service-a"},
		{name: "bare repo", fullName: "service-a", want: "service-a"},
		{name: "three segments", fullName: "a/b/c", want: "c"},
		{name: "empty", fullName: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, repoShortName(tt.fullName))
		})
	}
}
