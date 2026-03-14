package cli

import (
	"github.com/spf13/cobra"
)

// newSettingsCmd creates the "settings" parent command
func newSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Manage repository settings",
		Long: `Apply and audit repository settings against presets.

Commands:
  apply    Apply preset settings to a repository
  audit    Audit repositories against their assigned preset`,
	}

	cmd.AddCommand(
		newSettingsApplyCmd(),
		newSettingsAuditCmd(),
	)

	return cmd
}
