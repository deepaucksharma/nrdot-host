package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/newrelic/nrdot-host/nrdot-schema"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config-file>\n", os.Args[0])
		os.Exit(1)
	}

	configFile := os.Args[1]
	
	// Read the file
	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Create validator
	validator, err := schema.NewValidator()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating validator: %v\n", err)
		os.Exit(1)
	}

	// Validate based on file extension
	var config *schema.Config
	ext := filepath.Ext(configFile)
	
	switch ext {
	case ".yaml", ".yml":
		config, err = validator.ValidateYAML(data)
	case ".json":
		config, err = validator.ValidateJSON(data)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported file type: %s\n", ext)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Validation failed:\n%v\n", err)
		os.Exit(1)
	}

	// Success - print the validated config as JSON
	fmt.Printf("✓ Configuration is valid!\n\n")
	
	// Pretty print the configuration
	jsonData, _ := json.MarshalIndent(config, "", "  ")
	fmt.Printf("Parsed configuration:\n%s\n", jsonData)
}