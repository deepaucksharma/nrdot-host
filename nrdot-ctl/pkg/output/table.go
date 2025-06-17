package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/client"
)

var (
	outputWriter io.Writer = os.Stdout
	successColor = color.New(color.FgGreen).SprintFunc()
	errorColor   = color.New(color.FgRed).SprintFunc()
	warningColor = color.New(color.FgYellow).SprintFunc()
	infoColor    = color.New(color.FgCyan).SprintFunc()
)

// SetOutput sets the output writer (for testing)
func SetOutput(w io.Writer) {
	outputWriter = w
}

func formatStatusTable(status *client.Status) error {
	table := tablewriter.NewWriter(outputWriter)
	table.SetHeader([]string{"Property", "Value"})
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	// Format state with color
	stateStr := status.State
	switch status.State {
	case "running":
		stateStr = successColor(status.State)
	case "stopped":
		stateStr = errorColor(status.State)
	default:
		stateStr = warningColor(status.State)
	}

	table.Append([]string{"State", stateStr})
	table.Append([]string{"Uptime", formatDuration(status.Uptime)})
	table.Append([]string{"Config Version", status.ConfigVersion})
	table.Append([]string{"Collector Version", status.CollectorVersion})
	
	// Format health status
	healthStr := status.Health.Status
	switch status.Health.Status {
	case "healthy":
		healthStr = successColor(status.Health.Status)
	case "unhealthy":
		healthStr = errorColor(status.Health.Status)
	case "degraded":
		healthStr = warningColor(status.Health.Status)
	}
	table.Append([]string{"Health", healthStr})

	if status.LastError != "" {
		table.Append([]string{"Last Error", errorColor(status.LastError)})
		table.Append([]string{"Error Time", status.LastErrorTime.Format(time.RFC3339)})
	}

	table.Render()

	// Show health checks if any are failing
	if status.Health.Status != "healthy" && len(status.Health.Checks) > 0 {
		fmt.Fprintln(outputWriter, "\nHealth Checks:")
		for name, check := range status.Health.Checks {
			checkStr := fmt.Sprintf("  %s: %s", name, check.Status)
			if check.Status != "healthy" {
				checkStr = errorColor(checkStr)
				if check.Message != "" {
					checkStr += fmt.Sprintf(" - %s", check.Message)
				}
			}
			fmt.Fprintln(outputWriter, checkStr)
		}
	}

	return nil
}

func formatValidationTable(result *client.ValidationResult) error {
	if result.Valid {
		fmt.Fprintln(outputWriter, successColor("✓ Configuration is valid"))
	} else {
		fmt.Fprintln(outputWriter, errorColor("✗ Configuration is invalid"))
	}

	if len(result.Errors) > 0 {
		fmt.Fprintln(outputWriter, "\nErrors:")
		for _, err := range result.Errors {
			fmt.Fprintf(outputWriter, "  %s %s: %s\n", errorColor("✗"), err.Field, err.Message)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Fprintln(outputWriter, "\nWarnings:")
		for _, warning := range result.Warnings {
			fmt.Fprintf(outputWriter, "  %s %s\n", warningColor("!"), warning)
		}
	}

	return nil
}

func formatMetricsTable(metrics *client.Metrics) error {
	// Overall metrics
	table := tablewriter.NewWriter(outputWriter)
	table.SetHeader([]string{"Metric", "Value"})
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	table.Append([]string{"Received Metrics", fmt.Sprintf("%d", metrics.ReceivedMetrics)})
	table.Append([]string{"Sent Metrics", fmt.Sprintf("%d", metrics.SentMetrics)})
	table.Append([]string{"Dropped Metrics", fmt.Sprintf("%d", metrics.DroppedMetrics)})
	table.Append([]string{"Processing Rate", fmt.Sprintf("%.2f/s", metrics.ProcessingRate)})
	table.Append([]string{"Error Rate", fmt.Sprintf("%.2f%%", metrics.ErrorRate)})
	
	table.Render()

	// Resource usage
	fmt.Fprintln(outputWriter, "\nResource Usage:")
	resourceTable := tablewriter.NewWriter(outputWriter)
	resourceTable.SetBorder(false)
	resourceTable.SetColumnSeparator("")
	resourceTable.SetHeaderLine(false)
	resourceTable.SetAlignment(tablewriter.ALIGN_LEFT)

	resourceTable.Append([]string{"  CPU", fmt.Sprintf("%.1f%%", metrics.ResourceUsage.CPUPercent)})
	resourceTable.Append([]string{"  Memory", fmt.Sprintf("%.1f MB", metrics.ResourceUsage.MemoryMB)})
	resourceTable.Append([]string{"  Goroutines", fmt.Sprintf("%d", metrics.ResourceUsage.GoroutineCount)})
	
	resourceTable.Render()

	// Pipeline metrics
	if len(metrics.PipelineMetrics) > 0 {
		fmt.Fprintln(outputWriter, "\nPipeline Metrics:")
		pipelineTable := tablewriter.NewWriter(outputWriter)
		pipelineTable.SetHeader([]string{"Pipeline", "Received", "Sent", "Dropped", "Errors"})
		pipelineTable.SetBorder(false)

		for name, pm := range metrics.PipelineMetrics {
			pipelineTable.Append([]string{
				name,
				fmt.Sprintf("%d", pm.Received),
				fmt.Sprintf("%d", pm.Sent),
				fmt.Sprintf("%d", pm.Dropped),
				fmt.Sprintf("%d", pm.Errors),
			})
		}

		pipelineTable.Render()
	}

	return nil
}

func formatVersionTable(info *VersionInfo) error {
	table := tablewriter.NewWriter(outputWriter)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	table.Append([]string{"Version:", info.Version})
	table.Append([]string{"Build Time:", info.BuildTime})
	table.Append([]string{"Go Version:", info.GoVersion})
	table.Append([]string{"OS/Arch:", fmt.Sprintf("%s/%s", info.OS, info.Arch)})

	table.Render()
	return nil
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	
	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	
	return strings.Join(parts, " ")
}