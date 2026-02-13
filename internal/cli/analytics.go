package cli

import (
	"github.com/spf13/cobra"
)

// newAnalyticsCmd creates the analytics command group
func newAnalyticsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Manage repository analytics",
		Long:  `Collect and query analytics data for GitHub repositories across organizations.`,
	}

	cmd.AddCommand(
		newAnalyticsSyncCmd(),
		newAnalyticsStatusCmd(),
	)

	return cmd
}
