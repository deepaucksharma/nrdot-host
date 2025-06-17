package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/client"
	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/output"
)

// metricsCmd represents the metrics command
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show current metrics",
	Long: `Display current metrics from the NRDOT collector including:
- Metrics received/sent
- Processing rates
- Error counts
- Resource usage`,
	RunE: runMetrics,
}

func init() {
	rootCmd.AddCommand(metricsCmd)
}

func runMetrics(cmd *cobra.Command, args []string) error {
	// Create API client
	c := client.New(GetAPIEndpoint())

	// Get metrics
	metrics, err := c.GetMetrics()
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	// Format output
	formatter := output.NewFormatter(GetOutputFormat())
	return formatter.FormatMetrics(metrics)
}