package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/client"
	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/output"
	"github.com/briandowns/spinner"
	"time"
)

var (
	follow bool
	lines  int
)

// collectorCmd represents the collector command
var collectorCmd = &cobra.Command{
	Use:   "collector",
	Short: "Control collector operations",
	Long:  `Start, stop, restart, and monitor the NRDOT collector process.`,
}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the collector",
	Long:  `Start the NRDOT collector if it is not already running.`,
	RunE:  runStart,
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the collector",
	Long:  `Stop the running NRDOT collector.`,
	RunE:  runStop,
}

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the collector",
	Long:  `Restart the NRDOT collector by stopping and starting it.`,
	RunE:  runRestart,
}

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View collector logs",
	Long:  `Display logs from the NRDOT collector.`,
	RunE:  runLogs,
}

func init() {
	rootCmd.AddCommand(collectorCmd)
	collectorCmd.AddCommand(startCmd)
	collectorCmd.AddCommand(stopCmd)
	collectorCmd.AddCommand(restartCmd)
	collectorCmd.AddCommand(logsCmd)

	// Flags for logs command
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&lines, "lines", "n", 100, "Number of lines to show")
}

func runStart(cmd *cobra.Command, args []string) error {
	// Create spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Starting collector..."
	s.Start()
	defer s.Stop()

	// Create API client
	c := client.New(GetAPIEndpoint())

	// Start collector
	result, err := c.StartCollector()
	if err != nil {
		return fmt.Errorf("failed to start collector: %w", err)
	}

	s.Stop()

	// Format output
	formatter := output.NewFormatter(GetOutputFormat())
	return formatter.FormatOperationResult(result)
}

func runStop(cmd *cobra.Command, args []string) error {
	// Create spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Stopping collector..."
	s.Start()
	defer s.Stop()

	// Create API client
	c := client.New(GetAPIEndpoint())

	// Stop collector
	result, err := c.StopCollector()
	if err != nil {
		return fmt.Errorf("failed to stop collector: %w", err)
	}

	s.Stop()

	// Format output
	formatter := output.NewFormatter(GetOutputFormat())
	return formatter.FormatOperationResult(result)
}

func runRestart(cmd *cobra.Command, args []string) error {
	// Create spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Restarting collector..."
	s.Start()
	defer s.Stop()

	// Create API client
	c := client.New(GetAPIEndpoint())

	// Restart collector
	result, err := c.RestartCollector()
	if err != nil {
		return fmt.Errorf("failed to restart collector: %w", err)
	}

	s.Stop()

	// Format output
	formatter := output.NewFormatter(GetOutputFormat())
	return formatter.FormatOperationResult(result)
}

func runLogs(cmd *cobra.Command, args []string) error {
	// Create API client
	c := client.New(GetAPIEndpoint())

	// Get logs
	if follow {
		// Stream logs
		reader, err := c.StreamLogs()
		if err != nil {
			return fmt.Errorf("failed to stream logs: %w", err)
		}
		defer reader.Close()

		_, err = io.Copy(os.Stdout, reader)
		return err
	} else {
		// Get recent logs
		logs, err := c.GetLogs(lines)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}

		fmt.Print(logs)
		return nil
	}
}