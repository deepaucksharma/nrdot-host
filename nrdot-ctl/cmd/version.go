package cmd

import (
	"runtime"

	"github.com/spf13/cobra"
	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/output"
)

var (
	// Version is set at build time
	Version = "dev"
	// BuildTime is set at build time
	BuildTime = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version information for nrdot-ctl and its components.`,
	RunE:  runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	versionInfo := &output.VersionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	// Format output
	formatter := output.NewFormatter(GetOutputFormat())
	return formatter.FormatVersion(versionInfo)
}