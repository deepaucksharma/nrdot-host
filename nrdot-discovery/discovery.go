package discovery

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-telemetry/process"
	"go.uber.org/zap"
)

// ServiceInfo represents discovered service information
type ServiceInfo struct {
	Type         string                   `json:"type"`
	Version      string                   `json:"version,omitempty"`
	Endpoints    []Endpoint               `json:"endpoints"`
	DiscoveredBy []string                 `json:"discovered_by"`
	Confidence   string                   `json:"confidence"`
	ProcessInfo  *process.ProcessInfo     `json:"process_info,omitempty"`
	ConfigPaths  []string                 `json:"config_paths,omitempty"`
	PackageInfo  *PackageInfo             `json:"package_info,omitempty"`
	Additional   map[string]interface{}   `json:"additional_info,omitempty"`
}

// Endpoint represents a service endpoint
type Endpoint struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

// PackageInfo represents package manager information
type PackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Manager string `json:"manager"`
}

// ServiceDiscovery orchestrates multiple discovery methods
type ServiceDiscovery struct {
	logger           *zap.Logger
	processScanner   *ProcessScanner
	portScanner      *PortScanner
	configLocator    *ConfigLocator
	packageDetector  *PackageDetector
	privilegedHelper string // Path to privileged helper binary
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery(logger *zap.Logger) *ServiceDiscovery {
	return &ServiceDiscovery{
		logger:           logger,
		processScanner:   NewProcessScanner(logger),
		portScanner:      NewPortScanner(logger),
		configLocator:    NewConfigLocator(logger),
		packageDetector:  NewPackageDetector(logger),
		privilegedHelper: "/usr/local/bin/nrdot-helper",
	}
}

// Discover performs comprehensive service discovery
func (sd *ServiceDiscovery) Discover(ctx context.Context) ([]ServiceInfo, error) {
	startTime := time.Now()
	sd.logger.Info("Starting service discovery")

	// Run all discovery methods in parallel
	var wg sync.WaitGroup
	results := make(chan []ServiceInfo, 4)
	errors := make(chan error, 4)

	// Process scanning
	wg.Add(1)
	go func() {
		defer wg.Done()
		services, err := sd.processScanner.Scan(ctx)
		if err != nil {
			errors <- fmt.Errorf("process scan failed: %w", err)
			return
		}
		results <- services
	}()

	// Port scanning
	wg.Add(1)
	go func() {
		defer wg.Done()
		services, err := sd.portScanner.Scan(ctx)
		if err != nil {
			errors <- fmt.Errorf("port scan failed: %w", err)
			return
		}
		results <- services
	}()

	// Config file detection
	wg.Add(1)
	go func() {
		defer wg.Done()
		services, err := sd.configLocator.Scan(ctx)
		if err != nil {
			errors <- fmt.Errorf("config scan failed: %w", err)
			return
		}
		results <- services
	}()

	// Package detection
	wg.Add(1)
	go func() {
		defer wg.Done()
		services, err := sd.packageDetector.Scan(ctx)
		if err != nil {
			errors <- fmt.Errorf("package scan failed: %w", err)
			return
		}
		results <- services
	}()

	// Wait for all scans to complete
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	allServices := make(map[string]*ServiceInfo)
	
	for services := range results {
		for _, svc := range services {
			key := fmt.Sprintf("%s:%v", svc.Type, svc.Endpoints)
			
			if existing, exists := allServices[key]; exists {
				// Merge discovery methods
				existing.DiscoveredBy = mergeStrings(existing.DiscoveredBy, svc.DiscoveredBy)
				// Update confidence based on multiple signals
				existing.Confidence = sd.calculateConfidence(existing.DiscoveredBy)
				// Merge additional info
				if svc.ProcessInfo != nil && existing.ProcessInfo == nil {
					existing.ProcessInfo = svc.ProcessInfo
				}
				if svc.PackageInfo != nil && existing.PackageInfo == nil {
					existing.PackageInfo = svc.PackageInfo
				}
				if len(svc.ConfigPaths) > 0 {
					existing.ConfigPaths = mergeStrings(existing.ConfigPaths, svc.ConfigPaths)
				}
			} else {
				svc.Confidence = sd.calculateConfidence(svc.DiscoveredBy)
				allServices[key] = &svc
			}
		}
	}

	// Convert map to slice
	var finalServices []ServiceInfo
	for _, svc := range allServices {
		finalServices = append(finalServices, *svc)
	}

	duration := time.Since(startTime)
	sd.logger.Info("Service discovery completed",
		zap.Int("services_found", len(finalServices)),
		zap.Duration("duration", duration))

	return finalServices, nil
}

// calculateConfidence determines confidence level based on discovery methods
func (sd *ServiceDiscovery) calculateConfidence(methods []string) string {
	count := len(methods)
	if count >= 3 {
		return "HIGH"
	} else if count == 2 {
		return "MEDIUM"
	}
	return "LOW"
}

// ProcessScanner scans running processes to identify services
type ProcessScanner struct {
	logger   *zap.Logger
	detector *process.ServiceDetector
}

func NewProcessScanner(logger *zap.Logger) *ProcessScanner {
	return &ProcessScanner{
		logger:   logger,
		detector: process.NewServiceDetector(),
	}
}

func (ps *ProcessScanner) Scan(ctx context.Context) ([]ServiceInfo, error) {
	collector := process.NewProcessCollector(ps.logger, "/proc", 1000, time.Minute)
	processes, err := collector.Collect(ctx)
	if err != nil {
		return nil, err
	}

	var services []ServiceInfo
	detectedServices := make(map[string]bool)

	for _, proc := range processes {
		service, confidence := ps.detector.DetectService(proc)
		if service != "" && !detectedServices[service] {
			detectedServices[service] = true
			
			// Get service metadata
			metadata := ps.detector.GetServiceMetadata(service)
			
			// Build service info
			svc := ServiceInfo{
				Type:         service,
				DiscoveredBy: []string{"process"},
				ProcessInfo:  proc,
			}

			// Add endpoints from metadata
			if port, ok := metadata["default_port"].(int); ok {
				svc.Endpoints = append(svc.Endpoints, Endpoint{
					Address:  "localhost",
					Port:     port,
					Protocol: "tcp",
				})
			} else if ports, ok := metadata["default_ports"].([]int); ok {
				for _, port := range ports {
					svc.Endpoints = append(svc.Endpoints, Endpoint{
						Address:  "0.0.0.0",
						Port:     port,
						Protocol: "tcp",
					})
				}
			}

			// Add config paths
			if paths, ok := metadata["config_paths"].([]string); ok {
				svc.ConfigPaths = paths
			}

			svc.Additional = metadata
			services = append(services, svc)
		}
	}

	return services, nil
}

// PortScanner scans network ports to identify services
type PortScanner struct {
	logger *zap.Logger
}

func NewPortScanner(logger *zap.Logger) *PortScanner {
	return &PortScanner{logger: logger}
}

// Well-known ports for services
var wellKnownPorts = map[int]string{
	3306:  "mysql",
	5432:  "postgresql",
	6379:  "redis",
	80:    "http",
	443:   "https",
	11211: "memcached",
	27017: "mongodb",
	9200:  "elasticsearch",
	5672:  "rabbitmq",
	9092:  "kafka",
	2181:  "zookeeper",
	9042:  "cassandra",
}

func (ps *PortScanner) Scan(ctx context.Context) ([]ServiceInfo, error) {
	// Parse /proc/net/tcp and /proc/net/tcp6
	tcpPorts, err := ps.parseProcNet("/proc/net/tcp")
	if err != nil {
		ps.logger.Warn("Failed to parse /proc/net/tcp", zap.Error(err))
	}

	tcp6Ports, err := ps.parseProcNet("/proc/net/tcp6")
	if err != nil {
		ps.logger.Warn("Failed to parse /proc/net/tcp6", zap.Error(err))
	}

	// Combine all ports
	allPorts := append(tcpPorts, tcp6Ports...)

	var services []ServiceInfo
	detectedServices := make(map[string]bool)

	for _, port := range allPorts {
		if service, exists := wellKnownPorts[port.Port]; exists && !detectedServices[service] {
			detectedServices[service] = true
			
			services = append(services, ServiceInfo{
				Type: service,
				Endpoints: []Endpoint{{
					Address:  port.Address,
					Port:     port.Port,
					Protocol: "tcp",
				}},
				DiscoveredBy: []string{"port"},
			})
		}
	}

	return services, nil
}

type ListeningPort struct {
	Address string
	Port    int
	State   string
}

func (ps *PortScanner) parseProcNet(path string) ([]ListeningPort, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ports []ListeningPort
	scanner := bufio.NewScanner(file)
	
	// Skip header
	scanner.Scan()
	
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}

		// Parse local address
		localAddr := fields[1]
		parts := strings.Split(localAddr, ":")
		if len(parts) != 2 {
			continue
		}

		// Parse hex port
		portHex := parts[1]
		port, err := strconv.ParseInt(portHex, 16, 32)
		if err != nil {
			continue
		}

		// Parse hex IP
		ipHex := parts[0]
		ip := ps.hexToIP(ipHex)

		// Parse state (0A = LISTEN)
		state := fields[3]
		if state == "0A" {
			ports = append(ports, ListeningPort{
				Address: ip,
				Port:    int(port),
				State:   "LISTEN",
			})
		}
	}

	return ports, scanner.Err()
}

func (ps *PortScanner) hexToIP(hexStr string) string {
	if len(hexStr) == 8 {
		// IPv4
		bytes, _ := hex.DecodeString(hexStr)
		if len(bytes) == 4 {
			return fmt.Sprintf("%d.%d.%d.%d", bytes[3], bytes[2], bytes[1], bytes[0])
		}
	}
	// Return as-is for IPv6 or errors
	return hexStr
}

// ConfigLocator finds services by configuration files
type ConfigLocator struct {
	logger *zap.Logger
}

func NewConfigLocator(logger *zap.Logger) *ConfigLocator {
	return &ConfigLocator{logger: logger}
}

// Config paths that indicate service presence
var configIndicators = map[string]string{
	"/etc/mysql":         "mysql",
	"/etc/postgresql":    "postgresql",
	"/etc/redis":         "redis",
	"/etc/nginx":         "nginx",
	"/etc/apache2":       "apache",
	"/etc/httpd":         "apache",
	"/etc/mongodb.conf":  "mongodb",
	"/etc/mongod.conf":   "mongodb",
	"/etc/elasticsearch": "elasticsearch",
	"/etc/rabbitmq":      "rabbitmq",
	"/etc/kafka":         "kafka",
	"/etc/cassandra":     "cassandra",
}

func (cl *ConfigLocator) Scan(ctx context.Context) ([]ServiceInfo, error) {
	var services []ServiceInfo
	detectedServices := make(map[string]bool)

	for path, service := range configIndicators {
		if _, err := os.Stat(path); err == nil && !detectedServices[service] {
			detectedServices[service] = true
			
			services = append(services, ServiceInfo{
				Type:         service,
				DiscoveredBy: []string{"config_file"},
				ConfigPaths:  []string{path},
			})
		}
	}

	return services, nil
}

// PackageDetector finds services via package managers
type PackageDetector struct {
	logger *zap.Logger
}

func NewPackageDetector(logger *zap.Logger) *PackageDetector {
	return &PackageDetector{logger: logger}
}

func (pd *PackageDetector) Scan(ctx context.Context) ([]ServiceInfo, error) {
	var services []ServiceInfo

	// Detect package manager
	var manager string
	var cmd *exec.Cmd

	if _, err := exec.LookPath("dpkg"); err == nil {
		manager = "apt"
		cmd = exec.CommandContext(ctx, "dpkg", "-l")
	} else if _, err := exec.LookPath("rpm"); err == nil {
		manager = "yum"
		cmd = exec.CommandContext(ctx, "rpm", "-qa")
	} else {
		return services, nil // No supported package manager
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query packages: %w", err)
	}

	// Parse package list
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	detectedServices := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()
		
		// Check for service packages
		for _, pattern := range []struct {
			name    string
			service string
		}{
			{"mysql-server", "mysql"},
			{"mariadb-server", "mysql"},
			{"postgresql", "postgresql"},
			{"redis-server", "redis"},
			{"nginx", "nginx"},
			{"apache2", "apache"},
			{"httpd", "apache"},
			{"mongodb-server", "mongodb"},
			{"elasticsearch", "elasticsearch"},
			{"rabbitmq-server", "rabbitmq"},
			{"kafka", "kafka"},
		} {
			if strings.Contains(line, pattern.name) && !detectedServices[pattern.service] {
				detectedServices[pattern.service] = true
				
				// Extract version if possible
				version := pd.extractVersion(line, manager)
				
				services = append(services, ServiceInfo{
					Type:         pattern.service,
					DiscoveredBy: []string{"package"},
					PackageInfo: &PackageInfo{
						Name:    pattern.name,
						Version: version,
						Manager: manager,
					},
				})
			}
		}
	}

	return services, nil
}

func (pd *PackageDetector) extractVersion(line, manager string) string {
	if manager == "apt" {
		// dpkg format: ii  package-name  version  description
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "ii" {
			return fields[2]
		}
	} else if manager == "yum" {
		// rpm format: package-name-version-release.arch
		parts := strings.Split(line, "-")
		if len(parts) >= 2 {
			return parts[len(parts)-2]
		}
	}
	return ""
}

// Helper functions
func mergeStrings(a, b []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	
	return result
}