package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/client"
	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/output"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show collector status",
	Long: `Display the current status of the NRDOT collector including:
- Collector state (running/stopped)
- Uptime
- Configuration version
- Health status
- Recent errors`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Create API client
	c := client.New(GetAPIEndpoint())

	// Get status
	status, err := c.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Format output
	formatter := output.NewFormatter(GetOutputFormat())
	return formatter.FormatStatus(status)
}