package ai

import (
	"fmt"
)

// DeterministicCommitSubject builds a specific, guaranteed-correct commit subject
// directly from the extracted changeset - no model involved, so it cannot be wrong.
//
// It targets the common sync case (a single version bump) with a precise subject
// like "sync: bump MAGE_X_VERSION to v1.26.3", and summarizes larger changesets by
// count. It returns an empty string when there are no significant key changes, in
// which case the caller should use a file-based fallback.
func DeterministicCommitSubject(cs *Changeset) string {
	sig := cs.SignificantChanges()
	if len(sig) == 0 {
		return ""
	}

	if len(sig) == 1 {
		kc := sig[0]
		switch kc.Kind {
		case ChangeAdded:
			return "sync: add " + kc.Key
		case ChangeRemoved:
			return "sync: remove " + kc.Key
		case ChangeModified:
			fallthrough
		default:
			full := "sync: bump " + kc.Key + " to " + kc.New
			if len(full) <= maxCommitMessageLength {
				return full
			}
			// Value too long to fit cleanly; keep the key, drop the value.
			return "sync: bump " + kc.Key
		}
	}

	return fmt.Sprintf("sync: update %d config versions", len(sig))
}

// GuardCommitSubject checks an AI-generated commit subject for hallucinated version
// numbers - tokens that appear nowhere in the diff. When one is found it returns a
// deterministic, verified subject (which may be empty if none can be built) and
// true; otherwise it returns the subject unchanged and false.
func GuardCommitSubject(subject string, cs *Changeset) (string, bool) {
	if cs == nil {
		return subject, false
	}
	for _, tok := range versionTokenRe.FindAllString(subject, -1) {
		if _, ok := cs.VersionTokens[normalizeVersionToken(tok)]; !ok {
			return DeterministicCommitSubject(cs), true
		}
	}
	return subject, false
}
