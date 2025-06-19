package autoconfig

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-common/config"
	"github.com/newrelic/nrdot-host/nrdot-discovery"
	"github.com/newrelic/nrdot-host/nrdot-supervisor"
	"go.uber.org/zap"
)

// AutoConfigOrchestrator manages the auto-configuration lifecycle
type AutoConfigOrchestrator struct {
	logger             *zap.Logger
	enabled            bool
	scanInterval       time.Duration
	discovery          *discovery.ServiceDiscovery
	generator          *ConfigGenerator
	remoteClient       *RemoteConfigClient
	cache              *ConfigCache
	supervisor         *supervisor.UnifiedSupervisor
	configPath         string
	lastDiscovery      []discovery.ServiceInfo
	lastConfigVersion  string
	mu                 sync.RWMutex
	stopCh             chan struct{}
}

// NewAutoConfigOrchestrator creates a new auto-configuration orchestrator
func NewAutoConfigOrchestrator(logger *zap.Logger, cfg *config.Config, supervisor *supervisor.UnifiedSupervisor) *AutoConfigOrchestrator {
	hostID := getHostID()
	
	return &AutoConfigOrchestrator{
		logger:       logger,
		enabled:      cfg.AutoConfig.Enabled,
		scanInterval: cfg.AutoConfig.ScanInterval,
		discovery:    discovery.NewServiceDiscovery(logger),
		generator:    NewConfigGenerator(logger),
		remoteClient: NewRemoteConfigClient(logger, cfg.LicenseKey, hostID),
		cache:        NewConfigCache(logger, filepath.Join(cfg.DataDir, "config_cache.json")),
		supervisor:   supervisor,
		configPath:   cfg.ConfigPath,
		stopCh:       make(chan struct{}),
	}
}

// Start begins the auto-configuration process
func (aco *AutoConfigOrchestrator) Start(ctx context.Context) error {
	if !aco.enabled {
		aco.logger.Info("Auto-configuration is disabled")
		return nil
	}

	aco.logger.Info("Starting auto-configuration orchestrator",
		zap.Duration("scan_interval", aco.scanInterval))

	// Initial scan
	if err := aco.runDiscoveryAndConfig(ctx); err != nil {
		aco.logger.Error("Initial discovery failed", zap.Error(err))
		// Don't fail startup - continue with existing config
	}

	// Start periodic scanning
	go aco.runPeriodic(ctx)

	return nil
}

// Stop stops the auto-configuration process
func (aco *AutoConfigOrchestrator) Stop() {
	close(aco.stopCh)
}

// runPeriodic runs periodic discovery scans
func (aco *AutoConfigOrchestrator) runPeriodic(ctx context.Context) {
	ticker := time.NewTicker(aco.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := aco.runDiscoveryAndConfig(ctx); err != nil {
				aco.logger.Error("Periodic discovery failed", zap.Error(err))
			}
		case <-aco.stopCh:
			aco.logger.Info("Stopping auto-configuration orchestrator")
			return
		case <-ctx.Done():
			aco.logger.Info("Context cancelled, stopping orchestrator")
			return
		}
	}
}

// runDiscoveryAndConfig performs discovery and configuration update
func (aco *AutoConfigOrchestrator) runDiscoveryAndConfig(ctx context.Context) error {
	startTime := time.Now()

	// Run discovery
	services, err := aco.discovery.Discover(ctx)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	duration := time.Since(startTime)
	aco.logger.Info("Discovery completed",
		zap.Int("services_found", len(services)),
		zap.Duration("duration", duration))

	// Check if services changed
	if !aco.servicesChanged(services) {
		aco.logger.Debug("No service changes detected")
		return nil
	}

	// Update last discovery
	aco.mu.Lock()
	aco.lastDiscovery = services
	aco.mu.Unlock()

	// Send baseline to New Relic
	if err := aco.remoteClient.SendBaseline(ctx, services); err != nil {
		aco.logger.Warn("Failed to send baseline report", zap.Error(err))
		// Continue with local generation
	}

	// Fetch remote configuration
	remoteConfig, err := aco.remoteClient.FetchConfig(ctx)
	if err != nil {
		aco.logger.Warn("Failed to fetch remote configuration", zap.Error(err))
		// Fall back to local generation
		return aco.generateAndApplyLocal(ctx, services)
	}

	if remoteConfig == nil {
		// No new configuration
		return nil
	}

	// Apply remote configuration
	return aco.applyRemoteConfig(ctx, remoteConfig, services)
}

// servicesChanged checks if discovered services have changed
func (aco *AutoConfigOrchestrator) servicesChanged(services []discovery.ServiceInfo) bool {
	aco.mu.RLock()
	defer aco.mu.RUnlock()

	if len(services) != len(aco.lastDiscovery) {
		return true
	}

	// Create maps for comparison
	oldMap := make(map[string]bool)
	for _, svc := range aco.lastDiscovery {
		key := fmt.Sprintf("%s:%v", svc.Type, svc.Endpoints)
		oldMap[key] = true
	}

	for _, svc := range services {
		key := fmt.Sprintf("%s:%v", svc.Type, svc.Endpoints)
		if !oldMap[key] {
			return true
		}
	}

	return false
}

// generateAndApplyLocal generates and applies configuration locally
func (aco *AutoConfigOrchestrator) generateAndApplyLocal(ctx context.Context, services []discovery.ServiceInfo) error {
	aco.logger.Info("Generating configuration locally")

	// Generate configuration
	generatedConfig, err := aco.generator.GenerateConfig(ctx, services)
	if err != nil {
		return fmt.Errorf("failed to generate configuration: %w", err)
	}

	// Check for required environment variables
	missing := aco.checkRequiredVariables(generatedConfig.RequiredVariables)
	if len(missing) > 0 {
		aco.logger.Warn("Missing required environment variables",
			zap.Strings("variables", missing))
		// Could provide instructions or partial config
	}

	// Apply configuration
	return aco.applyGeneratedConfig(ctx, generatedConfig)
}

// applyRemoteConfig applies configuration from remote source
func (aco *AutoConfigOrchestrator) applyRemoteConfig(ctx context.Context, remoteConfig *RemoteConfig, services []discovery.ServiceInfo) error {
	aco.logger.Info("Applying remote configuration",
		zap.String("version", remoteConfig.Version))

	// Convert remote config to local format
	generatedConfig, err := aco.convertRemoteConfig(remoteConfig, services)
	if err != nil {
		return fmt.Errorf("failed to convert remote config: %w", err)
	}

	// Cache the configuration
	if err := aco.cache.Save(remoteConfig); err != nil {
		aco.logger.Warn("Failed to cache configuration", zap.Error(err))
	}

	// Apply configuration
	return aco.applyGeneratedConfig(ctx, generatedConfig)
}

// convertRemoteConfig converts remote configuration to generated config format
func (aco *AutoConfigOrchestrator) convertRemoteConfig(remote *RemoteConfig, services []discovery.ServiceInfo) (*GeneratedConfig, error) {
	// Build configuration from remote integrations
	// This is a simplified version - production would be more sophisticated
	
	config := &GeneratedConfig{
		Version:            remote.Version,
		DiscoveredServices: services,
		GeneratedAt:        time.Now(),
	}

	// TODO: Convert remote.Integrations to YAML config
	// For now, generate locally
	return aco.generator.GenerateConfig(context.Background(), services)
}

// applyGeneratedConfig applies the generated configuration
func (aco *AutoConfigOrchestrator) applyGeneratedConfig(ctx context.Context, config *GeneratedConfig) error {
	// Write to temporary file
	tempFile := filepath.Join(filepath.Dir(aco.configPath), fmt.Sprintf(".config-%s.yaml", config.Version))
	
	if err := ioutil.WriteFile(tempFile, []byte(config.Config), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Validate the configuration file
	if err := aco.validateConfigFile(tempFile); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Use supervisor's blue-green reload
	if err := aco.supervisor.ReloadConfig(tempFile); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// Success - move temp file to actual config
	if err := os.Rename(tempFile, aco.configPath); err != nil {
		aco.logger.Warn("Failed to move config file", zap.Error(err))
	}

	// Update version
	aco.mu.Lock()
	aco.lastConfigVersion = config.Version
	aco.mu.Unlock()

	aco.logger.Info("Successfully applied configuration",
		zap.String("version", config.Version),
		zap.Int("services", len(config.DiscoveredServices)))

	return nil
}

// validateConfigFile validates a configuration file
func (aco *AutoConfigOrchestrator) validateConfigFile(path string) error {
	// Use the config engine to validate
	// This is a placeholder - would use actual validation
	return nil
}

// checkRequiredVariables checks if required environment variables are set
func (aco *AutoConfigOrchestrator) checkRequiredVariables(required []string) []string {
	var missing []string
	for _, v := range required {
		if os.Getenv(v) == "" {
			missing = append(missing, v)
		}
	}
	return missing
}

// GetStatus returns the current auto-configuration status
func (aco *AutoConfigOrchestrator) GetStatus() AutoConfigStatus {
	aco.mu.RLock()
	defer aco.mu.RUnlock()

	return AutoConfigStatus{
		Enabled:           aco.enabled,
		LastScan:          aco.getLastScanTime(),
		NextScan:          aco.getNextScanTime(),
		ActiveServices:    aco.getActiveServices(),
		ConfigVersion:     aco.lastConfigVersion,
		DiscoveredServices: len(aco.lastDiscovery),
	}
}

// getLastScanTime returns the last scan time
func (aco *AutoConfigOrchestrator) getLastScanTime() *time.Time {
	// This would be tracked properly in production
	t := time.Now().Add(-time.Minute)
	return &t
}

// getNextScanTime returns the next scan time
func (aco *AutoConfigOrchestrator) getNextScanTime() *time.Time {
	t := time.Now().Add(aco.scanInterval)
	return &t
}

// getActiveServices returns list of active service types
func (aco *AutoConfigOrchestrator) getActiveServices() []string {
	seen := make(map[string]bool)
	var services []string
	
	for _, svc := range aco.lastDiscovery {
		if !seen[svc.Type] {
			seen[svc.Type] = true
			services = append(services, svc.Type)
		}
	}
	
	return services
}

// AutoConfigStatus represents the current status
type AutoConfigStatus struct {
	Enabled            bool       `json:"enabled"`
	LastScan           *time.Time `json:"last_scan,omitempty"`
	NextScan           *time.Time `json:"next_scan,omitempty"`
	ActiveServices     []string   `json:"active_services"`
	ConfigVersion      string     `json:"config_version"`
	DiscoveredServices int        `json:"discovered_services"`
}

// getHostID generates or retrieves a persistent host ID
func getHostID() string {
	// In production, this would:
	// 1. Check for existing ID in /var/lib/nrdot/host_id
	// 2. Use cloud instance ID if available
	// 3. Generate and persist a new UUID
	
	// For now, use hostname
	hostname, _ := os.Hostname()
	return hostname
}