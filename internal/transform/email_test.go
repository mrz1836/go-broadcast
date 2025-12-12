package transform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailTransformer_Name(t *testing.T) {
	transformer := NewEmailTransformer()
	assert.Equal(t, "email-address-replacer", transformer.Name())
}

func TestEmailTransformer_SkipWhenNoEmails(t *testing.T) {
	transformer := NewEmailTransformer()
	content := []byte("security@example.com")

	ctx := Context{
		FilePath: "test.md",
		// No email configuration
	}

	result, err := transformer.Transform(content, ctx)
	require.NoError(t, err)
	assert.Equal(t, content, result, "Should not transform when emails not configured")
}

func TestEmailTransformer_SkipWhenEmailsMatch(t *testing.T) {
	transformer := NewEmailTransformer()
	content := []byte("security@example.com")

	ctx := Context{
		FilePath:            "test.md",
		SourceSecurityEmail: "security@example.com",
		TargetSecurityEmail: "security@example.com", // Same as source
	}

	result, err := transformer.Transform(content, ctx)
	require.NoError(t, err)
	assert.Equal(t, content, result, "Should not transform when source and target emails are identical")
}

func TestEmailTransformer_Markdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain email",
			input:    "Contact: security@example.com",
			expected: "Contact: security@company.com",
		},
		{
			name:     "mailto link",
			input:    "[Email us](mailto:security@example.com)",
			expected: "[Email us](mailto:security@company.com)",
		},
		{
			name:     "email as link text",
			input:    "[security@example.com](mailto:security@example.com)",
			expected: "[security@company.com](mailto:security@company.com)",
		},
		{
			name:     "multiple occurrences",
			input:    "Email security@example.com or use mailto:security@example.com",
			expected: "Email security@company.com or use mailto:security@company.com",
		},
		{
			name:     "email in text",
			input:    "Please contact security@example.com for issues.",
			expected: "Please contact security@company.com for issues.",
		},
	}

	transformer := NewEmailTransformer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Context{
				FilePath:            "SECURITY.md",
				SourceSecurityEmail: "security@example.com",
				TargetSecurityEmail: "security@company.com",
			}

			result, err := transformer.Transform([]byte(tt.input), ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestEmailTransformer_YAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "double quoted",
			input:    `email: "security@example.com"`,
			expected: `email: "security@company.com"`,
		},
		{
			name:     "single quoted",
			input:    `email: 'security@example.com'`,
			expected: `email: 'security@company.com'`,
		},
		{
			name:     "unquoted",
			input:    `email: security@example.com`,
			expected: `email: security@company.com`,
		},
		{
			name:     "in list",
			input:    "emails:\n  - \"security@example.com\"\n  - \"support@example.com\"",
			expected: "emails:\n  - \"security@company.com\"\n  - \"support@company.com\"",
		},
	}

	transformer := NewEmailTransformer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Context{
				FilePath:            "config.yml",
				SourceSecurityEmail: "security@example.com",
				TargetSecurityEmail: "security@company.com",
				SourceSupportEmail:  "support@example.com",
				TargetSupportEmail:  "support@company.com",
			}

			result, err := transformer.Transform([]byte(tt.input), ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestEmailTransformer_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple field",
			input:    `{"email": "security@example.com"}`,
			expected: `{"email": "security@company.com"}`,
		},
		{
			name:     "nested field",
			input:    `{"contact": {"email": "security@example.com"}}`,
			expected: `{"contact": {"email": "security@company.com"}}`,
		},
		{
			name:     "array",
			input:    `{"emails": ["security@example.com", "support@example.com"]}`,
			expected: `{"emails": ["security@company.com", "support@company.com"]}`,
		},
	}

	transformer := NewEmailTransformer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Context{
				FilePath:            "config.json",
				SourceSecurityEmail: "security@example.com",
				TargetSecurityEmail: "security@company.com",
				SourceSupportEmail:  "support@example.com",
				TargetSupportEmail:  "support@company.com",
			}

			result, err := transformer.Transform([]byte(tt.input), ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestEmailTransformer_HTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "mailto link",
			input:    `<a href="mailto:security@example.com">Contact</a>`,
			expected: `<a href="mailto:security@company.com">Contact</a>`,
		},
		{
			name:     "email in text",
			input:    `<p>Contact: security@example.com</p>`,
			expected: `<p>Contact: security@company.com</p>`,
		},
		{
			name:     "email as link text",
			input:    `<a href="mailto:security@example.com">security@example.com</a>`,
			expected: `<a href="mailto:security@company.com">security@company.com</a>`,
		},
	}

	transformer := NewEmailTransformer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Context{
				FilePath:            "index.html",
				SourceSecurityEmail: "security@example.com",
				TargetSecurityEmail: "security@company.com",
			}

			result, err := transformer.Transform([]byte(tt.input), ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestEmailTransformer_BothSecurityAndSupport(t *testing.T) {
	input := `
# Security Policy

For security issues, contact security@example.com

For general support, email support@example.com
`

	expected := `
# Security Policy

For security issues, contact security@company.com

For general support, email support@company.com
`

	transformer := NewEmailTransformer()
	ctx := Context{
		FilePath:            "SECURITY.md",
		SourceSecurityEmail: "security@example.com",
		TargetSecurityEmail: "security@company.com",
		SourceSupportEmail:  "support@example.com",
		TargetSupportEmail:  "support@company.com",
	}

	result, err := transformer.Transform([]byte(input), ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

func TestEmailTransformer_OnlyTransformConfiguredEmails(t *testing.T) {
	input := "Contact security@example.com or support@example.com"

	transformer := NewEmailTransformer()

	// Only security email configured
	ctx := Context{
		FilePath:            "test.md",
		SourceSecurityEmail: "security@example.com",
		TargetSecurityEmail: "security@company.com",
		// Support email not configured
	}

	result, err := transformer.Transform([]byte(input), ctx)
	require.NoError(t, err)

	expected := "Contact security@company.com or support@example.com"
	assert.Equal(t, expected, string(result), "Should only transform security email")
}

func TestEmailTransformer_RealWorldSecurityMD(t *testing.T) {
	input := `# 🔐 Security Policy

## 📨 Reporting a Vulnerability

If you've found a security issue, **please don't open a public issue or PR**.

Instead, send a private email to:
📧 [go-broadcast@mrz1818.com](mailto:go-broadcast@mrz1818.com)

Include the following:

* 🕵️ Description of the issue and its impact
* 🧪 Steps to reproduce or a working PoC
* 🔧 Any known workarounds or mitigations
`

	expected := `# 🔐 Security Policy

## 📨 Reporting a Vulnerability

If you've found a security issue, **please don't open a public issue or PR**.

Instead, send a private email to:
📧 [my-service@company.com](mailto:my-service@company.com)

Include the following:

* 🕵️ Description of the issue and its impact
* 🧪 Steps to reproduce or a working PoC
* 🔧 Any known workarounds or mitigations
`

	transformer := NewEmailTransformer()
	ctx := Context{
		FilePath:            ".github/SECURITY.md",
		SourceSecurityEmail: "go-broadcast@mrz1818.com",
		TargetSecurityEmail: "my-service@company.com",
	}

	result, err := transformer.Transform([]byte(input), ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

func TestEmailTransformer_RealWorldSupportMD(t *testing.T) {
	input := `# 🛟 Support Guide

## 📬 Private Contact

For sensitive or non-public concerns, reach out to:
📧 [go-broadcast@mrz1818.com](mailto:go-broadcast@mrz1818.com)
`

	expected := `# 🛟 Support Guide

## 📬 Private Contact

For sensitive or non-public concerns, reach out to:
📧 [my-service@company.com](mailto:my-service@company.com)
`

	transformer := NewEmailTransformer()
	ctx := Context{
		FilePath:           ".github/SUPPORT.md",
		SourceSupportEmail: "go-broadcast@mrz1818.com",
		TargetSupportEmail: "my-service@company.com",
	}

	result, err := transformer.Transform([]byte(input), ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

func TestEmailTransformer_PartialEmailNotReplaced(t *testing.T) {
	// Ensure we don't replace partial matches
	input := "Contact newsecurity@example.com not security@example.com"

	transformer := NewEmailTransformer()
	ctx := Context{
		FilePath:            "test.md",
		SourceSecurityEmail: "security@example.com",
		TargetSecurityEmail: "security@company.com",
	}

	result, err := transformer.Transform([]byte(input), ctx)
	require.NoError(t, err)

	// Only the exact match should be replaced
	expected := "Contact newsecurity@example.com not security@company.com"
	assert.Equal(t, expected, string(result))
}

func TestEmailTransformer_EmailWithRepoNameInAddress(t *testing.T) {
	// Regression test: Ensure email addresses containing repo names are transformed correctly
	// This test covers the bug where repo name transformer would corrupt emails like
	// "go-broadcast@mrz1818.com" by replacing "go-broadcast" before email transformer runs
	input := `# Security Policy

If you've found a security issue, **please don't open a public issue or PR**.

Instead, send a private email to:
📧 [go-broadcast@mrz1818.com](mailto:go-broadcast@mrz1818.com)

Include the following:
* Description of the issue
`

	expected := `# Security Policy

If you've found a security issue, **please don't open a public issue or PR**.

Instead, send a private email to:
📧 [security@bsvassociation.org](mailto:security@bsvassociation.org)

Include the following:
* Description of the issue
`

	transformer := NewEmailTransformer()
	ctx := Context{
		FilePath:            ".github/SECURITY.md",
		SourceSecurityEmail: "go-broadcast@mrz1818.com",
		TargetSecurityEmail: "security@bsvassociation.org",
	}

	result, err := transformer.Transform([]byte(input), ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

func TestEmailTransformer_EmailAndSupportWithRepoNameInAddress(t *testing.T) {
	// Regression test: Ensure both security and support emails are transformed correctly
	// when they contain repo names that might be transformed by repo transformer
	input := `# Support Guide

For sensitive or non-public concerns, reach out to:
📧 [go-broadcast@mrz1818.com](mailto:go-broadcast@mrz1818.com)

For general questions, contact:
📧 [go-broadcast@mrz1818.com](mailto:go-broadcast@mrz1818.com)
`

	expected := `# Support Guide

For sensitive or non-public concerns, reach out to:
📧 [security@bsvassociation.org](mailto:security@bsvassociation.org)

For general questions, contact:
📧 [security@bsvassociation.org](mailto:security@bsvassociation.org)
`

	transformer := NewEmailTransformer()
	ctx := Context{
		FilePath:            ".github/SUPPORT.md",
		SourceSecurityEmail: "go-broadcast@mrz1818.com",
		TargetSecurityEmail: "security@bsvassociation.org",
		SourceSupportEmail:  "go-broadcast@mrz1818.com",
		TargetSupportEmail:  "security@bsvassociation.org",
	}

	result, err := transformer.Transform([]byte(input), ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

// TestEmailTransformer_SpecialCharactersInTargetEmail tests that email addresses
// with special regex characters (like $) in the target email are handled correctly.
// This was a bug where $ would be interpreted as a regex backreference.
func TestEmailTransformer_SpecialCharactersInTargetEmail(t *testing.T) {
	tests := []struct {
		name        string
		sourceEmail string
		targetEmail string
		input       string
		expected    string
	}{
		{
			name:        "dollar sign in target email",
			sourceEmail: "security@example.com",
			targetEmail: "user$1@company.com",
			input:       "Contact security@example.com for help",
			expected:    "Contact user$1@company.com for help",
		},
		{
			name:        "multiple dollar signs in target email",
			sourceEmail: "security@example.com",
			targetEmail: "user$$test@company.com",
			input:       "Contact security@example.com for help",
			expected:    "Contact user$$test@company.com for help",
		},
		{
			name:        "dollar sign in markdown link",
			sourceEmail: "security@example.com",
			targetEmail: "user$1@company.com",
			input:       "[security@example.com](mailto:security@example.com)",
			expected:    "[user$1@company.com](mailto:user$1@company.com)",
		},
		{
			name:        "plus sign in target email",
			sourceEmail: "security@example.com",
			targetEmail: "user+tag@company.com",
			input:       "Contact security@example.com for help",
			expected:    "Contact user+tag@company.com for help",
		},
	}

	transformer := NewEmailTransformer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Context{
				FilePath:            "test.md",
				SourceSecurityEmail: tt.sourceEmail,
				TargetSecurityEmail: tt.targetEmail,
			}

			result, err := transformer.Transform([]byte(tt.input), ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}
