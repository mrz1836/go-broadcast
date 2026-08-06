package cli

import (
	selfupdate "github.com/mrz1836/go-selfupdate"
	"github.com/mrz1836/go-selfupdate/cobracmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// attachUpdateCommand registers go-broadcast's self-update command on root and
// wires the passive "a new version is available" notice, both derived from a
// single self-update config. The running binary's compiled-in version is threaded
// in explicitly so a binary run from outside PATH still updates itself, and the
// registered command is returned so callers (and tests) can inspect the wiring.
func attachUpdateCommand(root *cobra.Command) *cobra.Command {
	// One call registers the `update` command (alias `upgrade`, flags
	// --check/--force/--verbose) and the passive "a new version is available"
	// banner, both derived from this single config. go-broadcast installs only
	// from its GitHub release archives — verifying the SHA-256 checksum and
	// atomically replacing the binary — and refuses to overwrite a binary owned
	// by `go install` or Homebrew. The environment-variable prefix derives from
	// the binary name, giving GO_BROADCAST_ (opt out with
	// GO_BROADCAST_NO_UPDATE_CHECK; the shared NO_UPDATE_CHECK and CI also
	// disable it). The deprecated --use-binary flag is kept, hidden and inert, so
	// old invocations do not error now that a release archive is the only install
	// route.
	cmd := cobracmd.Attach(root, selfupdate.Config{
		Owner:          "mrz1836",
		Repo:           "go-broadcast",
		BinaryName:     "go-broadcast",
		CurrentVersion: GetVersion(),
		TokenEnvVar:    "GO_BROADCAST_GITHUB_TOKEN",
	}, cobracmd.WithDeprecatedUseBinaryFlag())

	// go-broadcast's global --config already owns the -c shorthand; the library's
	// --check flag ships -c too. Strip --check's shorthand so cobra can merge the
	// inherited --config/-c without a pflag shorthand-redefine panic at run time.
	// The long --check form is unaffected.
	dropShorthand(cmd, flagCheck)

	// Match go-broadcast's help conventions with a rendered Examples section.
	cmd.Example = `  go-broadcast update            # download & install the latest release
  go-broadcast update --check    # report whether a newer version is available
  go-broadcast update --force    # reinstall the latest even if already current
  go-broadcast update --verbose  # narrate each step`

	return cmd
}

// flagCheck is the library's dry-run flag whose default -c shorthand collides
// with go-broadcast's global --config.
const flagCheck = "check"

// dropShorthand clears the single-letter shorthand on cmd's named flag by
// rebuilding the flag set without it. pflag exposes no shorthand-removal API,
// so every flag is re-added to a fresh set (ResetFlags) with the target's
// shorthand cleared; each flag's value binding and hidden state are preserved.
func dropShorthand(cmd *cobra.Command, name string) {
	var flags []*pflag.Flag
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Name == name {
			f.Shorthand = ""
		}
		flags = append(flags, f)
	})
	cmd.ResetFlags()
	for _, f := range flags {
		cmd.Flags().AddFlag(f)
	}
}
