package sync

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
)

// Blast-radius guard thresholds. A resolved sync scope that crosses either
// threshold trips the confirmation guard (SC-3): more than one group, OR more
// than five repositories in the resolved scope. The repo threshold closes the
// single-group-many-repos hole — a config where dozens of repos live in one
// group could otherwise blast widely without ever crossing the ">1 group" line.
const (
	// BlastRadiusMaxGroups is the largest group count that passes the guard
	// silently. The operating rule is one group at a time, so any scope with more
	// than one group always requires explicit confirmation.
	BlastRadiusMaxGroups = 1

	// BlastRadiusMaxRepos is the largest repository count that passes the guard
	// silently. A sync touching more than this many repos requires explicit
	// confirmation even when it stays within a single group.
	BlastRadiusMaxRepos = 5
)

// ErrScopeConfirmationRequired is the sentinel returned when a sync's resolved
// scope trips the blast-radius guard and the operator has not explicitly
// confirmed the resolved repository count. It is matchable with errors.Is.
//
// Non-interactive contexts (agents, CI, no TTY) return this unless a matching
// --confirm-scope=<N> is supplied (default-deny); interactive contexts return it
// when the typed value does not match the resolved repo count.
var ErrScopeConfirmationRequired = errors.New("blast-radius scope confirmation required")

// GuardTriggered reports whether the resolved scope crosses the blast-radius
// thresholds and therefore requires explicit confirmation before any write.
func (s ResolvedScope) GuardTriggered() bool {
	return s.GroupCount > BlastRadiusMaxGroups || s.RepoCount > BlastRadiusMaxRepos
}

// ScopeGuardDecision is the pure outcome of the blast-radius guard for a
// resolved scope and the supplied confirmation state. It carries no I/O.
type ScopeGuardDecision struct {
	// Proceed reports whether the sync may proceed without (further) prompting.
	Proceed bool

	// NeedPrompt reports that an interactive type-the-count prompt is required
	// before proceeding. Only set in interactive contexts when no matching
	// --confirm-scope was supplied.
	NeedPrompt bool

	// ExpectedN is the resolved repository count the operator must confirm. It is
	// always the repo count (never trivially 1 on a guarded single-group sync).
	ExpectedN int

	// Reason explains a refusal. Empty when Proceed or NeedPrompt is set.
	Reason string
}

// decideScopeGuard is the pure blast-radius decision. It has no TTY or stdin
// dependency, so every branch is unit-testable without a terminal.
//
// The confirmation token is always the resolved repository count (the real blast
// radius), so a guard tripped by the repo threshold inside a single group can
// never be satisfied with --confirm-scope=1.
//
// Branches:
//   - scope not triggered                       → Proceed.
//   - confirmScope supplied and == RepoCount     → Proceed.
//   - confirmScope supplied and != RepoCount     → refuse (mismatch).
//   - interactive (no confirmScope)              → NeedPrompt (ExpectedN = RepoCount).
//   - non-interactive (no confirmScope)          → refuse (default-deny).
func decideScopeGuard(scope ResolvedScope, confirmScope *int, interactive bool) ScopeGuardDecision {
	if !scope.GuardTriggered() {
		return ScopeGuardDecision{Proceed: true}
	}

	expected := scope.RepoCount

	// Explicit count-bearing opt-in works in any context (TTY or not).
	if confirmScope != nil {
		if *confirmScope == expected {
			return ScopeGuardDecision{Proceed: true, ExpectedN: expected}
		}
		return ScopeGuardDecision{
			ExpectedN: expected,
			Reason: fmt.Sprintf(
				"--confirm-scope=%d does not match the resolved repo count %d (blast radius: %d repo(s) across %d group(s))",
				*confirmScope, expected, scope.RepoCount, scope.GroupCount,
			),
		}
	}

	// Interactive: ask the operator to type the exact resolved repo count.
	if interactive {
		return ScopeGuardDecision{NeedPrompt: true, ExpectedN: expected}
	}

	// Non-interactive default-deny: an agent/CI literally cannot trigger a
	// guarded sync without stating the real blast radius.
	return ScopeGuardDecision{
		ExpectedN: expected,
		Reason: fmt.Sprintf(
			"refusing to sync %d repo(s) across %d group(s) without confirmation; re-run with --confirm-scope=%d to proceed",
			scope.RepoCount, scope.GroupCount, expected,
		),
	}
}

// ScopeConfirmer abstracts interactive confirmation of a guarded blast radius so
// TTY detection and stdin reads stay out of the pure decision logic and tests
// can inject a fake.
type ScopeConfirmer interface {
	// Interactive reports whether an interactive confirmation prompt is possible
	// (a real TTY is attached).
	Interactive() bool

	// Prompt displays the resolved scope and reads the operator's typed
	// confirmation, returning the integer they entered.
	Prompt(scope ResolvedScope) (int, error)
}

// terminalScopeConfirmer is the default ScopeConfirmer. It detects a TTY on its
// input stream and reads a single line as the confirmation count.
type terminalScopeConfirmer struct {
	in  *os.File
	out io.Writer
}

// newTerminalScopeConfirmer returns the default confirmer wired to stdin/stderr.
func newTerminalScopeConfirmer() *terminalScopeConfirmer {
	return &terminalScopeConfirmer{in: os.Stdin, out: os.Stderr}
}

// Interactive reports whether stdin is attached to a real terminal.
func (c *terminalScopeConfirmer) Interactive() bool {
	if c.in == nil {
		return false
	}
	fd := c.in.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// Prompt lists every in-scope group and repository, then reads one line of input
// and parses it as the confirmation count. Non-numeric input parses to 0, which
// never equals a guarded scope's repo count, so the caller fails closed.
func (c *terminalScopeConfirmer) Prompt(scope ResolvedScope) (int, error) {
	w := c.out
	if w == nil {
		w = os.Stderr
	}

	_, _ = fmt.Fprintln(w, scope.Summary())
	_, _ = fmt.Fprintf(w, "\nThis sync will write to %d repositor(ies) across %d group(s):\n", scope.RepoCount, scope.GroupCount)
	if len(scope.Groups) > 0 {
		_, _ = fmt.Fprintf(w, "  Groups: %s\n", strings.Join(scope.Groups, ", "))
	}
	for _, repo := range scope.Repos {
		_, _ = fmt.Fprintf(w, "  - %s\n", repo)
	}
	_, _ = fmt.Fprintf(w, "\nType the exact number of repositories (%d) to confirm, or anything else to abort: ", scope.RepoCount)

	reader := bufio.NewReader(c.in)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}

	// Non-numeric input parses to 0, which never matches a guarded scope's repo
	// count, so the caller fails closed.
	n, _ := strconv.Atoi(strings.TrimSpace(line))
	return n, nil
}
