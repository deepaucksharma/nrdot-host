package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the CLI configuration
type Config struct {
	APIEndpoint  string            `yaml:"api_endpoint"`
	OutputFormat string            `yaml:"output_format"`
	NoColor      bool              `yaml:"no_color"`
	Verbose      bool              `yaml:"verbose"`
	Aliases      map[string]string `yaml:"aliases"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		APIEndpoint:  "http://localhost:8080",
		OutputFormat: "table",
		NoColor:      false,
		Verbose:      false,
		Aliases:      make(map[string]string),
	}
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// Save saves configuration to file
func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetConfigPath returns the default config path
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".nrdot-ctl.yaml"
	}
	return filepath.Join(home, ".nrdot-ctl.yaml")
}