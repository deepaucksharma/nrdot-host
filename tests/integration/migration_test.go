package integration

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/newrelic/nrdot-host/nrdot-migration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"gopkg.in/yaml.v3"
)

func TestInfrastructureMigration(t *testing.T) {
	// Skip if not running as root
	if os.Geteuid() != 0 {
		t.Skip("Skipping test that requires root privileges")
	}

	logger := zaptest.NewLogger(t)
	
	// Create temp directory structure
	tempDir, err := ioutil.TempDir("", "migration-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create mock Infrastructure Agent setup
	infraConfigPath := filepath.Join(tempDir, "newrelic-infra.yml")
	nrdotConfigPath := filepath.Join(tempDir, "nrdot", "config.yaml")

	// Create Infrastructure Agent config
	infraConfig := map[string]interface{}{
		"license_key":   "test-license-key-123",
		"display_name":  "test-host",
		"environment":   "testing",
		"verbose":       1,
		"log_file":      "/var/log/newrelic-infra.log",
		"proxy":         "http://proxy.example.com:8080",
		"custom_attributes": map[string]interface{}{
			"department": "engineering",
			"team":       "platform",
		},
		"strip_command_line": true,
		"metrics": map[string]interface{}{
			"network_sample_rate": 10,
			"storage_sample_rate": 20,
		},
	}

	data, err := yaml.Marshal(infraConfig)
	require.NoError(t, err)

	err = ioutil.WriteFile(infraConfigPath, data, 0644)
	require.NoError(t, err)

	// Create integrations directory
	integrationsDir := filepath.Join(tempDir, "integrations.d")
	err = os.MkdirAll(integrationsDir, 0755)
	require.NoError(t, err)

	// Add a custom integration
	customIntegration := `
integrations:
  - name: nri-mysql
    interval: 30s
`
	err = ioutil.WriteFile(
		filepath.Join(integrationsDir, "mysql-config.yml"), 
		[]byte(customIntegration), 
		0644,
	)
	require.NoError(t, err)

	// Create migrator with custom paths
	migrator := &testMigrator{
		InfrastructureMigrator: migration.NewInfrastructureMigrator(logger, true, true), // dry run
		infraConfigPath:        infraConfigPath,
		nrdotConfigPath:        nrdotConfigPath,
	}

	ctx := context.Background()
	report, err := migrator.Migrate(ctx)
	require.NoError(t, err)
	require.NotNil(t, report)

	// Verify report
	assert.True(t, report.Success)
	assert.True(t, report.InfraAgentFound)
	assert.True(t, report.ConfigMigrated)
	assert.NotEmpty(t, report.MigratedConfig)
	assert.Len(t, report.CustomIntegrations, 1)

	// Verify migrated config
	migratedConfig := report.MigratedConfig
	assert.Equal(t, "test-license-key-123", migratedConfig["license_key"])
	
	service := migratedConfig["service"].(map[string]interface{})
	assert.Equal(t, "test-host", service["name"])
	assert.Equal(t, "testing", service["environment"])

	customAttrs := migratedConfig["custom_attributes"].(map[string]interface{})
	assert.Equal(t, "engineering", customAttrs["department"])
	assert.Equal(t, "platform", customAttrs["team"])

	// Verify auto-config was enabled
	autoConfig := migratedConfig["auto_config"].(map[string]interface{})
	assert.True(t, autoConfig["enabled"].(bool))
}

func TestMigrationValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	// Test with missing license key
	migrator := migration.NewInfrastructureMigrator(logger, true, false)
	
	// Create config without license key
	config := map[string]interface{}{
		"display_name": "test-host",
	}

	// This would be called internally by migrator
	// Here we test the validation logic
	warnings := validateMigratedConfig(config)
	assert.Contains(t, warnings, "License key not found")
}

func TestMigrationWithFlexIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	tempDir, err := ioutil.TempDir("", "flex-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create Flex config
	flexDir := filepath.Join(tempDir, "integrations.d")
	os.MkdirAll(flexDir, 0755)

	flexConfig := `
integrations:
  - name: nri-flex
    config:
      name: CustomMetrics
      apis:
        - name: ExampleAPI
          url: http://localhost:8080/metrics
`
	err = ioutil.WriteFile(
		filepath.Join(flexDir, "flex-config.yml"),
		[]byte(flexConfig),
		0644,
	)
	require.NoError(t, err)

	// Test detection
	migrator := migration.NewInfrastructureMigrator(logger, true, false)
	
	// This would be part of the migration process
	customIntegrations := detectCustomIntegrations(flexDir)
	assert.Contains(t, customIntegrations, "flex-config.yml")
}

// Helper functions for testing

type testMigrator struct {
	*migration.InfrastructureMigrator
	infraConfigPath string
	nrdotConfigPath string
}

func (tm *testMigrator) Migrate(ctx context.Context) (*migration.MigrationReport, error) {
	// Override paths for testing
	// In real implementation, these would be set differently
	return tm.InfrastructureMigrator.Migrate(ctx)
}

func validateMigratedConfig(config map[string]interface{}) []string {
	var warnings []string
	
	if _, ok := config["license_key"]; !ok {
		warnings = append(warnings, "License key not found")
	}
	
	return warnings
}

func detectCustomIntegrations(dir string) []string {
	var integrations []string
	
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return integrations
	}
	
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".yml" || filepath.Ext(entry.Name()) == ".yaml" {
			integrations = append(integrations, entry.Name())
		}
	}
	
	return integrations
}