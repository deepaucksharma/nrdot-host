package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/client"
	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/output"
)

var (
	configFile string
	outputFile string
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Manage NRDOT configuration including validation, generation, and application.`,
}

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long:  `Validate NRDOT configuration file for correctness and completeness.`,
	RunE:  runValidate,
}

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate OTel config from NRDOT config",
	Long: `Generate OpenTelemetry Collector configuration from simplified NRDOT 
configuration. This uses the nrdot-template-lib to transform the configuration.`,
	RunE: runGenerate,
}

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply new configuration",
	Long: `Apply a new configuration to the running collector. This will validate
the configuration and then reload the collector with the new settings.`,
	RunE: runApply,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(validateCmd)
	configCmd.AddCommand(generateCmd)
	configCmd.AddCommand(applyCmd)

	// Flags for validate command
	validateCmd.Flags().StringVarP(&configFile, "file", "f", "", "Configuration file to validate (required)")
	validateCmd.MarkFlagRequired("file")

	// Flags for generate command
	generateCmd.Flags().StringVarP(&configFile, "file", "f", "", "NRDOT configuration file (required)")
	generateCmd.Flags().StringVar(&outputFile, "out", "", "Output file for generated config (default: stdout)")
	generateCmd.MarkFlagRequired("file")

	// Flags for apply command
	applyCmd.Flags().StringVarP(&configFile, "file", "f", "", "Configuration file to apply (required)")
	applyCmd.MarkFlagRequired("file")
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Read config file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Create API client
	c := client.New(GetAPIEndpoint())

	// Validate configuration
	result, err := c.ValidateConfig(configData)
	if err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}

	// Format output
	formatter := output.NewFormatter(GetOutputFormat())
	return formatter.FormatValidationResult(result)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Read config file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Create API client
	c := client.New(GetAPIEndpoint())

	// Generate configuration
	generatedConfig, err := c.GenerateConfig(configData)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Write output
	if outputFile != "" {
		err = os.WriteFile(outputFile, generatedConfig, 0644)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Generated configuration written to %s\n", outputFile)
	} else {
		fmt.Print(string(generatedConfig))
	}

	return nil
}

func runApply(cmd *cobra.Command, args []string) error {
	// Read config file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Create API client
	c := client.New(GetAPIEndpoint())

	// Apply configuration
	result, err := c.ApplyConfig(configData)
	if err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}

	// Format output
	formatter := output.NewFormatter(GetOutputFormat())
	return formatter.FormatApplyResult(result)
}