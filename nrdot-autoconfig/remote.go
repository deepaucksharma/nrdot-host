package autoconfig

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-discovery"
	"go.uber.org/zap"
)

// RemoteConfigClient handles communication with New Relic Configuration Service
type RemoteConfigClient struct {
	logger      *zap.Logger
	httpClient  *http.Client
	baseURL     string
	licenseKey  string
	hostID      string
}

// NewRemoteConfigClient creates a new remote configuration client
func NewRemoteConfigClient(logger *zap.Logger, licenseKey, hostID string) *RemoteConfigClient {
	return &RemoteConfigClient{
		logger:     logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		baseURL:    "https://api.newrelic.com/v1/nrdot",
		licenseKey: licenseKey,
		hostID:     hostID,
	}
}

// BaselineReport represents the discovery report sent to New Relic
type BaselineReport struct {
	SchemaVersion      string                  `json:"schema_version"`
	HostID             string                  `json:"host_id"`
	Hostname           string                  `json:"hostname"`
	Timestamp          time.Time               `json:"timestamp"`
	DiscoveredServices []discovery.ServiceInfo `json:"discovered_services"`
	HostMetadata       HostMetadata            `json:"host_metadata"`
	DiscoveryMetadata  DiscoveryMetadata       `json:"discovery_metadata"`
}

// HostMetadata contains host information
type HostMetadata struct {
	OS               string            `json:"os"`
	OSFamily         string            `json:"os_family"`
	Kernel           string            `json:"kernel"`
	Architecture     string            `json:"architecture"`
	CPUCores         int               `json:"cpu_cores"`
	CPUModel         string            `json:"cpu_model"`
	MemoryGB         float64           `json:"memory_gb"`
	SwapGB           float64           `json:"swap_gb,omitempty"`
	CloudProvider    string            `json:"cloud_provider,omitempty"`
	CloudMetadata    map[string]string `json:"cloud_metadata,omitempty"`
	Virtualization   string            `json:"virtualization,omitempty"`
	BootTime         time.Time         `json:"boot_time"`
	AgentVersion     string            `json:"agent_version"`
	AgentCapabilities []string         `json:"agent_capabilities"`
}

// DiscoveryMetadata contains information about the discovery scan
type DiscoveryMetadata struct {
	DiscoveryID    string        `json:"discovery_id"`
	ScanDurationMS int64         `json:"scan_duration_ms"`
	Errors         []ScanError   `json:"errors,omitempty"`
	ConfigVersion  string        `json:"config_version,omitempty"`
	NextScan       *time.Time    `json:"next_scan,omitempty"`
}

// ScanError represents a non-fatal error during discovery
type ScanError struct {
	Method  string `json:"method"`
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// RemoteConfig represents configuration received from New Relic
type RemoteConfig struct {
	Version      string                `json:"version"`
	Integrations []IntegrationConfig   `json:"integrations"`
	Signature    string                `json:"signature"`
	PublicKey    string                `json:"public_key"`
	ValidUntil   time.Time             `json:"valid_until"`
}

// IntegrationConfig represents configuration for a specific service
type IntegrationConfig struct {
	Type    string                 `json:"type"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// SendBaseline sends a discovery baseline report to New Relic
func (rc *RemoteConfigClient) SendBaseline(ctx context.Context, services []discovery.ServiceInfo) error {
	report := BaselineReport{
		SchemaVersion:      "1.0",
		HostID:             rc.hostID,
		Hostname:           getHostname(),
		Timestamp:          time.Now(),
		DiscoveredServices: services,
		HostMetadata:       rc.collectHostMetadata(),
		DiscoveryMetadata: DiscoveryMetadata{
			DiscoveryID:    generateUUID(),
			ScanDurationMS: 0, // Will be set by caller
			ConfigVersion:  getCurrentConfigVersion(),
		},
	}

	// Serialize report
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal baseline report: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/hosts/%s/baseline", rc.baseURL, rc.hostID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rc.licenseKey))
	req.Header.Set("User-Agent", "NRDOT-HOST/2.0")

	// Send request
	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send baseline report: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("baseline report failed with status %d: %s", resp.StatusCode, string(body))
	}

	rc.logger.Info("Successfully sent baseline report",
		zap.String("host_id", rc.hostID),
		zap.Int("services", len(services)))

	return nil
}

// FetchConfig retrieves configuration from New Relic
func (rc *RemoteConfigClient) FetchConfig(ctx context.Context) (*RemoteConfig, error) {
	// Create request
	url := fmt.Sprintf("%s/hosts/%s/config", rc.baseURL, rc.hostID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rc.licenseKey))
	req.Header.Set("User-Agent", "NRDOT-HOST/2.0")
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode == http.StatusNotModified {
		// Config hasn't changed
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("config fetch failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var config RemoteConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config response: %w", err)
	}

	// Verify signature
	if err := rc.verifyConfigSignature(&config); err != nil {
		return nil, fmt.Errorf("config signature verification failed: %w", err)
	}

	rc.logger.Info("Successfully fetched remote configuration",
		zap.String("version", config.Version),
		zap.Int("integrations", len(config.Integrations)))

	return &config, nil
}

// verifyConfigSignature verifies the configuration signature
func (rc *RemoteConfigClient) verifyConfigSignature(config *RemoteConfig) error {
	// Reconstruct the signed data
	configData := struct {
		Version      string              `json:"version"`
		Integrations []IntegrationConfig `json:"integrations"`
		ValidUntil   time.Time           `json:"valid_until"`
	}{
		Version:      config.Version,
		Integrations: config.Integrations,
		ValidUntil:   config.ValidUntil,
	}

	data, err := json.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %w", err)
	}

	// Verify signature
	return VerifySignature(data, config.Signature, config.PublicKey)
}

// collectHostMetadata gathers host information
func (rc *RemoteConfigClient) collectHostMetadata() HostMetadata {
	return HostMetadata{
		OS:               getOSInfo(),
		OSFamily:         getOSFamily(),
		Kernel:           getKernelVersion(),
		Architecture:     getArchitecture(),
		CPUCores:         getCPUCores(),
		CPUModel:         getCPUModel(),
		MemoryGB:         getMemoryGB(),
		SwapGB:           getSwapGB(),
		CloudProvider:    detectCloudProvider(),
		CloudMetadata:    getCloudMetadata(),
		Virtualization:   detectVirtualization(),
		BootTime:         getBootTime(),
		AgentVersion:     "2.0.0",
		AgentCapabilities: []string{"auto_config", "process_monitoring", "log_forwarding"},
	}
}

// ConfigCache manages local caching of configurations
type ConfigCache struct {
	logger    *zap.Logger
	cacheFile string
}

// NewConfigCache creates a new configuration cache
func NewConfigCache(logger *zap.Logger, cacheFile string) *ConfigCache {
	return &ConfigCache{
		logger:    logger,
		cacheFile: cacheFile,
	}
}

// Load loads cached configuration
func (cc *ConfigCache) Load() (*RemoteConfig, error) {
	data, err := ioutil.ReadFile(cc.cacheFile)
	if err != nil {
		return nil, err
	}

	var config RemoteConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached config: %w", err)
	}

	// Check if still valid
	if time.Now().After(config.ValidUntil) {
		return nil, fmt.Errorf("cached config has expired")
	}

	return &config, nil
}

// Save saves configuration to cache
func (cc *ConfigCache) Save(config *RemoteConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(cc.cacheFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	cc.logger.Debug("Saved configuration to cache",
		zap.String("file", cc.cacheFile),
		zap.String("version", config.Version))

	return nil
}

// Helper functions (implementations would use actual system calls)

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

func generateUUID() string {
	// Simple UUID v4 generation
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func getCurrentConfigVersion() string {
	// Read from current config file
	return "none"
}

func getOSInfo() string {
	// Read from /etc/os-release
	return "Ubuntu 22.04.3 LTS"
}

func getOSFamily() string {
	// Determine from /etc/os-release
	return "debian"
}

func getKernelVersion() string {
	// uname -r
	return "5.15.0-88-generic"
}

func getArchitecture() string {
	// uname -m
	return "x86_64"
}

func getCPUCores() int {
	// Read from /proc/cpuinfo
	return 4
}

func getCPUModel() string {
	// Read from /proc/cpuinfo
	return "Intel(R) Xeon(R) CPU E5-2686 v4 @ 2.30GHz"
}

func getMemoryGB() float64 {
	// Read from /proc/meminfo
	return 16.0
}

func getSwapGB() float64 {
	// Read from /proc/meminfo
	return 4.0
}

func detectCloudProvider() string {
	// Check various indicators
	if _, err := os.Stat("/sys/hypervisor/uuid"); err == nil {
		// Read UUID and check prefix
		return "aws"
	}
	return "none"
}

func getCloudMetadata() map[string]string {
	// Query cloud metadata service
	return map[string]string{
		"instance_id":   "i-1234567890abcdef0",
		"instance_type": "t3.large",
		"region":        "us-east-1",
		"availability_zone": "us-east-1a",
	}
}

func detectVirtualization() string {
	// Check /sys/hypervisor/type or systemd-detect-virt
	return "kvm"
}

func getBootTime() time.Time {
	// Read from /proc/stat
	return time.Now().Add(-24 * time.Hour)
}