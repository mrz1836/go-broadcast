package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// ErrNotImplemented is returned for features not yet implemented
var ErrNotImplemented = errors.New("feature not yet implemented")

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
