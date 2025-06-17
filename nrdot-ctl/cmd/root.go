package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	apiEndpoint string
	outputFormat string
	noColor     bool
	verbose     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nrdot-ctl",
	Short: "Command-line interface for managing NRDOT system",
	Long: `nrdot-ctl is the command-line interface for managing the NRDOT 
(New Relic Data on Tap) system. It provides commands for configuration
management, collector control, and system monitoring.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.nrdot-ctl.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiEndpoint, "api-endpoint", "http://localhost:8080", "API server endpoint")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table|json|yaml)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	// Bind flags to viper
	viper.BindPFlag("api_endpoint", rootCmd.PersistentFlags().Lookup("api-endpoint"))
	viper.BindPFlag("output_format", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("no_color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Disable color if requested
	if noColor {
		color.NoColor = true
	}

	// Add completion command
	rootCmd.AddCommand(newCompletionCmd())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".nrdot-ctl" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath("/etc/nrdot")
		viper.SetConfigName(".nrdot-ctl")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("NRDOT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// GetAPIEndpoint returns the configured API endpoint
func GetAPIEndpoint() string {
	return viper.GetString("api_endpoint")
}

// GetOutputFormat returns the configured output format
func GetOutputFormat() string {
	return viper.GetString("output_format")
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return viper.GetBool("verbose")
}

// newCompletionCmd returns the completion command
func newCompletionCmd() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:

  $ source <(nrdot-ctl completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ nrdot-ctl completion bash > /etc/bash_completion.d/nrdot-ctl
  # macOS:
  $ nrdot-ctl completion bash > $(brew --prefix)/etc/bash_completion.d/nrdot-ctl

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ nrdot-ctl completion zsh > "${fpath[1]}/_nrdot-ctl"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ nrdot-ctl completion fish | source

  # To load completions for each session, execute once:
  $ nrdot-ctl completion fish > ~/.config/fish/completions/nrdot-ctl.fish

PowerShell:

  PS> nrdot-ctl completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> nrdot-ctl completion powershell > nrdot-ctl.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}

	return completionCmd
}