package migration

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// InfrastructureMigrator handles migration from New Relic Infrastructure Agent
type InfrastructureMigrator struct {
	logger          *zap.Logger
	dryRun          bool
	preserveOriginal bool
	infraConfigPath string
	nrdotConfigPath string
}

// NewInfrastructureMigrator creates a new migrator
func NewInfrastructureMigrator(logger *zap.Logger, dryRun, preserveOriginal bool) *InfrastructureMigrator {
	return &InfrastructureMigrator{
		logger:           logger,
		dryRun:           dryRun,
		preserveOriginal: preserveOriginal,
		infraConfigPath:  "/etc/newrelic-infra.yml",
		nrdotConfigPath:  "/etc/nrdot/config.yaml",
	}
}

// MigrationReport contains the migration results
type MigrationReport struct {
	Success            bool                     `json:"success"`
	InfraAgentFound    bool                     `json:"infra_agent_found"`
	ConfigMigrated     bool                     `json:"config_migrated"`
	ServicesStopped    bool                     `json:"services_stopped"`
	DataPreserved      bool                     `json:"data_preserved"`
	Errors             []string                 `json:"errors,omitempty"`
	Warnings           []string                 `json:"warnings,omitempty"`
	MigratedConfig     map[string]interface{}   `json:"migrated_config,omitempty"`
	CustomIntegrations []string                 `json:"custom_integrations,omitempty"`
	StartTime          time.Time                `json:"start_time"`
	EndTime            time.Time                `json:"end_time"`
}

// Migrate performs the migration from Infrastructure Agent to NRDOT-HOST
func (im *InfrastructureMigrator) Migrate(ctx context.Context) (*MigrationReport, error) {
	report := &MigrationReport{
		StartTime: time.Now(),
		Errors:    []string{},
		Warnings:  []string{},
	}

	im.logger.Info("Starting Infrastructure Agent migration",
		zap.Bool("dry_run", im.dryRun))

	// Step 1: Detect Infrastructure Agent
	if !im.detectInfrastructureAgent() {
		report.InfraAgentFound = false
		report.Errors = append(report.Errors, "New Relic Infrastructure Agent not found")
		report.EndTime = time.Now()
		return report, fmt.Errorf("Infrastructure Agent not detected")
	}
	report.InfraAgentFound = true

	// Step 2: Read and convert configuration
	migratedConfig, err := im.migrateConfiguration()
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Configuration migration failed: %v", err))
		report.EndTime = time.Now()
		return report, err
	}
	report.ConfigMigrated = true
	report.MigratedConfig = migratedConfig

	// Step 3: Detect custom integrations
	customIntegrations := im.detectCustomIntegrations()
	if len(customIntegrations) > 0 {
		report.CustomIntegrations = customIntegrations
		report.Warnings = append(report.Warnings, 
			fmt.Sprintf("Found %d custom integrations that may need manual migration", len(customIntegrations)))
	}

	// Step 4: Create NRDOT configuration
	if !im.dryRun {
		if err := im.writeNRDOTConfig(migratedConfig); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Failed to write NRDOT config: %v", err))
			report.EndTime = time.Now()
			return report, err
		}
	}

	// Step 5: Stop Infrastructure Agent
	if !im.dryRun {
		if err := im.stopInfrastructureAgent(); err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Failed to stop Infrastructure Agent: %v", err))
		} else {
			report.ServicesStopped = true
		}
	}

	// Step 6: Preserve data if requested
	if im.preserveOriginal {
		if err := im.preserveInfraData(); err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Failed to preserve data: %v", err))
		} else {
			report.DataPreserved = true
		}
	}

	// Step 7: Validate migration
	if err := im.validateMigration(migratedConfig); err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Validation warnings: %v", err))
	}

	report.Success = len(report.Errors) == 0
	report.EndTime = time.Now()

	im.logger.Info("Migration completed",
		zap.Bool("success", report.Success),
		zap.Duration("duration", report.EndTime.Sub(report.StartTime)))

	return report, nil
}

// detectInfrastructureAgent checks if Infrastructure Agent is installed
func (im *InfrastructureMigrator) detectInfrastructureAgent() bool {
	// Check for config file
	if _, err := os.Stat(im.infraConfigPath); err != nil {
		// Try alternate locations
		alternates := []string{
			"/etc/newrelic-infra/newrelic-infra.yml",
			"/usr/local/etc/newrelic-infra/newrelic-infra.yml",
		}
		
		found := false
		for _, alt := range alternates {
			if _, err := os.Stat(alt); err == nil {
				im.infraConfigPath = alt
				found = true
				break
			}
		}
		
		if !found {
			return false
		}
	}

	// Check if service exists
	cmd := exec.Command("systemctl", "status", "newrelic-infra")
	if err := cmd.Run(); err != nil {
		// Try alternative service names
		cmd = exec.Command("service", "newrelic-infra", "status")
		if err := cmd.Run(); err != nil {
			return false
		}
	}

	im.logger.Info("Infrastructure Agent detected",
		zap.String("config_path", im.infraConfigPath))

	return true
}

// migrateConfiguration converts Infrastructure Agent config to NRDOT format
func (im *InfrastructureMigrator) migrateConfiguration() (map[string]interface{}, error) {
	// Read Infrastructure Agent config
	data, err := ioutil.ReadFile(im.infraConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var infraConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &infraConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Convert to NRDOT format
	nrdotConfig := make(map[string]interface{})

	// License key
	if licenseKey, ok := infraConfig["license_key"].(string); ok {
		nrdotConfig["license_key"] = licenseKey
	}

	// Basic settings
	nrdotConfig["service"] = map[string]interface{}{
		"name": getStringOrDefault(infraConfig, "display_name", "migrated-host"),
		"environment": getStringOrDefault(infraConfig, "environment", "production"),
	}

	// Proxy settings
	if proxy, ok := infraConfig["proxy"].(string); ok {
		nrdotConfig["proxy"] = map[string]interface{}{
			"http_proxy": proxy,
		}
	}

	// Custom attributes
	if customAttrs, ok := infraConfig["custom_attributes"].(map[string]interface{}); ok {
		nrdotConfig["custom_attributes"] = customAttrs
	}

	// Log configuration
	logConfig := map[string]interface{}{
		"level": "info",
	}
	
	if logFile, ok := infraConfig["log_file"].(string); ok {
		logConfig["file"] = logFile
	}
	
	if verbose, ok := infraConfig["verbose"].(int); ok && verbose > 0 {
		logConfig["level"] = "debug"
	}
	
	nrdotConfig["logging"] = logConfig

	// Feature flags
	features := map[string]interface{}{}
	
	if stripCommandLine, ok := infraConfig["strip_command_line"].(bool); ok {
		features["strip_command_line"] = stripCommandLine
	}
	
	if len(features) > 0 {
		nrdotConfig["features"] = features
	}

	// Metrics
	if metrics, ok := infraConfig["metrics"].(map[string]interface{}); ok {
		// Convert network and storage sample rates
		metricsConfig := map[string]interface{}{}
		
		if networkRate, ok := metrics["network_sample_rate"].(int); ok {
			metricsConfig["network_interval"] = fmt.Sprintf("%ds", networkRate)
		}
		
		if storageRate, ok := metrics["storage_sample_rate"].(int); ok {
			metricsConfig["storage_interval"] = fmt.Sprintf("%ds", storageRate)
		}
		
		if len(metricsConfig) > 0 {
			nrdotConfig["metrics"] = metricsConfig
		}
	}

	// Add auto-configuration
	nrdotConfig["auto_config"] = map[string]interface{}{
		"enabled": true,
		"scan_interval": "5m",
	}

	im.logger.Info("Configuration migrated",
		zap.Int("settings", len(nrdotConfig)))

	return nrdotConfig, nil
}

// detectCustomIntegrations finds custom integrations
func (im *InfrastructureMigrator) detectCustomIntegrations() []string {
	var integrations []string

	// Check integrations directory
	integrationsDir := "/etc/newrelic-infra/integrations.d"
	entries, err := ioutil.ReadDir(integrationsDir)
	if err != nil {
		im.logger.Debug("No integrations directory found", zap.Error(err))
		return integrations
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".yaml") {
			integrations = append(integrations, entry.Name())
		}
	}

	// Check for OHI integrations
	ohiPath := "/var/db/newrelic-infra/custom-integrations"
	if entries, err := ioutil.ReadDir(ohiPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				integrations = append(integrations, "ohi:"+entry.Name())
			}
		}
	}

	return integrations
}

// writeNRDOTConfig writes the migrated configuration
func (im *InfrastructureMigrator) writeNRDOTConfig(config map[string]interface{}) error {
	// Ensure directory exists
	configDir := filepath.Dir(im.nrdotConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Add migration metadata
	config["_migration"] = map[string]interface{}{
		"migrated_from": "infrastructure-agent",
		"migrated_at":   time.Now().Format(time.RFC3339),
		"version":       "1.0",
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add header comment
	header := `# NRDOT-HOST Configuration
# Migrated from New Relic Infrastructure Agent
# Migration Date: ` + time.Now().Format(time.RFC3339) + `
#
# This configuration was automatically generated. Please review and adjust as needed.
# For more information, see: https://docs.newrelic.com/docs/nrdot-host

`
	
	fullConfig := header + string(data)

	// Write config file
	if err := ioutil.WriteFile(im.nrdotConfigPath, []byte(fullConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	im.logger.Info("NRDOT configuration written",
		zap.String("path", im.nrdotConfigPath))

	return nil
}

// stopInfrastructureAgent stops the Infrastructure Agent service
func (im *InfrastructureMigrator) stopInfrastructureAgent() error {
	// Stop service
	cmd := exec.Command("systemctl", "stop", "newrelic-infra")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try alternative
		cmd = exec.Command("service", "newrelic-infra", "stop")
		if output2, err2 := cmd.CombinedOutput(); err2 != nil {
			return fmt.Errorf("failed to stop service: %v (output: %s %s)", err2, output, output2)
		}
	}

	// Disable service
	cmd = exec.Command("systemctl", "disable", "newrelic-infra")
	cmd.Run() // Ignore errors for disable

	im.logger.Info("Infrastructure Agent service stopped")
	return nil
}

// preserveInfraData backs up Infrastructure Agent data
func (im *InfrastructureMigrator) preserveInfraData() error {
	backupDir := "/var/lib/nrdot/migration-backup"
	
	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Backup paths
	backupPaths := []struct {
		src  string
		dest string
	}{
		{im.infraConfigPath, filepath.Join(backupDir, "newrelic-infra.yml")},
		{"/etc/newrelic-infra/integrations.d", filepath.Join(backupDir, "integrations.d")},
		{"/var/db/newrelic-infra", filepath.Join(backupDir, "db")},
		{"/var/log/newrelic-infra", filepath.Join(backupDir, "logs")},
	}

	for _, bp := range backupPaths {
		if _, err := os.Stat(bp.src); err == nil {
			// Use cp -r for directories
			cmd := exec.Command("cp", "-r", bp.src, bp.dest)
			if err := cmd.Run(); err != nil {
				im.logger.Warn("Failed to backup",
					zap.String("src", bp.src),
					zap.Error(err))
			}
		}
	}

	// Create backup info file
	info := fmt.Sprintf(`Infrastructure Agent Migration Backup
=====================================
Date: %s
Original Config: %s
NRDOT Config: %s

This backup contains:
- Original configuration files
- Custom integrations
- Database files
- Recent log files

To restore Infrastructure Agent:
1. Copy files back to original locations
2. Run: systemctl enable newrelic-infra
3. Run: systemctl start newrelic-infra
`, time.Now().Format(time.RFC3339), im.infraConfigPath, im.nrdotConfigPath)

	infoPath := filepath.Join(backupDir, "README.txt")
	if err := ioutil.WriteFile(infoPath, []byte(info), 0644); err != nil {
		im.logger.Warn("Failed to write backup info", zap.Error(err))
	}

	im.logger.Info("Infrastructure Agent data preserved",
		zap.String("backup_dir", backupDir))

	return nil
}

// validateMigration performs post-migration validation
func (im *InfrastructureMigrator) validateMigration(config map[string]interface{}) error {
	var warnings []string

	// Check license key
	if _, ok := config["license_key"]; !ok {
		warnings = append(warnings, "License key not found in configuration")
	}

	// Check for custom integrations
	integrationsDir := "/etc/newrelic-infra/integrations.d"
	if entries, err := ioutil.ReadDir(integrationsDir); err == nil && len(entries) > 0 {
		warnings = append(warnings, fmt.Sprintf("Found %d custom integrations that need manual review", len(entries)))
	}

	// Check for flex integrations
	flexConfig := "/etc/newrelic-infra/integrations.d/flex-config.yml"
	if _, err := os.Stat(flexConfig); err == nil {
		warnings = append(warnings, "Flex integration detected - manual migration required")
	}

	// Check for logging integrations
	loggingConfig := "/etc/newrelic-infra/logging.d"
	if _, err := os.Stat(loggingConfig); err == nil {
		warnings = append(warnings, "Log forwarding configuration detected - review NRDOT log configuration")
	}

	if len(warnings) > 0 {
		return fmt.Errorf("validation warnings: %s", strings.Join(warnings, "; "))
	}

	return nil
}

// PrintReport prints a human-readable migration report
func PrintReport(report *MigrationReport) {
	fmt.Println("\n=== Infrastructure Agent Migration Report ===")
	fmt.Printf("Duration: %s\n", report.EndTime.Sub(report.StartTime))
	fmt.Printf("Success: %v\n", report.Success)
	
	if report.InfraAgentFound {
		fmt.Println("✓ Infrastructure Agent detected")
	} else {
		fmt.Println("✗ Infrastructure Agent not found")
	}
	
	if report.ConfigMigrated {
		fmt.Println("✓ Configuration migrated")
	} else {
		fmt.Println("✗ Configuration migration failed")
	}
	
	if report.ServicesStopped {
		fmt.Println("✓ Infrastructure Agent stopped")
	}
	
	if report.DataPreserved {
		fmt.Println("✓ Original data preserved")
	}
	
	if len(report.CustomIntegrations) > 0 {
		fmt.Printf("\nCustom Integrations Found (%d):\n", len(report.CustomIntegrations))
		for _, integration := range report.CustomIntegrations {
			fmt.Printf("  - %s\n", integration)
		}
	}
	
	if len(report.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(report.Warnings))
		for _, warning := range report.Warnings {
			fmt.Printf("  ⚠ %s\n", warning)
		}
	}
	
	if len(report.Errors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(report.Errors))
		for _, err := range report.Errors {
			fmt.Printf("  ✗ %s\n", err)
		}
	}
	
	if report.Success {
		fmt.Println("\n✅ Migration completed successfully!")
		fmt.Println("\nNext steps:")
		fmt.Println("1. Review the generated configuration at /etc/nrdot/config.yaml")
		fmt.Println("2. Set any required environment variables for service credentials")
		fmt.Println("3. Start NRDOT-HOST: sudo systemctl start nrdot-host")
		fmt.Println("4. Verify metrics in New Relic")
	} else {
		fmt.Println("\n❌ Migration failed. Please review errors above.")
	}
}

// Helper functions

func getStringOrDefault(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}