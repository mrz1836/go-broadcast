package cli

import (
	"fmt"
	"strings"

	"github.com/mrz1836/go-broadcast/internal/validation"
)

// repoShortName returns the repository short name: the segment after the final
// "/" of an "org/repo" full name (e.g. "org/service-a" -> "service-a").
// A name without a "/" is returned unchanged ("service-a" -> "service-a"),
// and only the last segment is taken ("a/b/c" -> "c").
func repoShortName(fullName string) string {
	if idx := strings.LastIndex(fullName, "/"); idx >= 0 {
		return fullName[idx+1:]
	}
	return fullName
}

// resolveClonedEmail decides how a single contact email column is carried from a
// source target to a cloned destination target. It is pure (no DB, no cobra) so
// the rebase/verbatim/fallback matrix is fast and deterministic to test.
//
// The rule:
//   - An empty source email is copied empty (no rebase).
//   - If the source local-part exactly (case-sensitive, byte-for-byte) equals the
//     source repo's short name, the address follows the per-repo "<repo>@domain"
//     convention: the destination repo's short name is substituted as the new
//     local-part, preserving the domain (e.g. "service-a@example.com" cloned to
//     "service-b" -> "service-b@example.com").
//   - Otherwise the address is a shared/org email (e.g. "security@example.org")
//     and is copied verbatim — no rebasing.
//   - If a rebase would produce an address that fails email validation (an exotic
//     destination repo name), the source email is copied verbatim and a warning is
//     returned; the clone never hard-fails and never writes a malformed address.
//
// It returns the resolved email and an optional colleague-safe warning (empty when
// there is nothing to warn about).
func resolveClonedEmail(sourceEmail, srcShortName, dstShortName string) (resolved, warning string) {
	// Empty source email: copy empty, never rebase.
	if sourceEmail == "" {
		return "", ""
	}

	// Split on the last "@"; if there is none, leave pre-existing odd data untouched.
	at := strings.LastIndex(sourceEmail, "@")
	if at < 0 {
		return sourceEmail, "" // verbatim: not an address we understand
	}

	localPart := sourceEmail[:at]
	domain := sourceEmail[at+1:]

	// Shared/org email (local-part does not match the source repo short name):
	// copy verbatim so org-wide and intentionally shared contacts are preserved.
	if localPart != srcShortName {
		return sourceEmail, "" // verbatim: shared/org email, no rebase
	}

	// Per-repo convention matched: rebase the local-part to the destination short name.
	candidate := dstShortName + "@" + domain
	if validation.ValidateEmail(candidate, "security_email/support_email") != nil {
		// Invalid rebase (exotic destination repo name): copy source verbatim + warn.
		warning = fmt.Sprintf(
			"could not rebase contact email to %q (invalid address); copied source value %q verbatim — set it explicitly with --security-email/--support-email",
			candidate, sourceEmail,
		)
		return sourceEmail, warning
	}

	return candidate, ""
}
