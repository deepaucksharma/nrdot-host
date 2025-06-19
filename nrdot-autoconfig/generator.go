package autoconfig

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-discovery"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ConfigGenerator generates OpenTelemetry configurations from discovered services
type ConfigGenerator struct {
	logger         *zap.Logger
	templateEngine *TemplateEngine
	validator      *ConfigValidator
	signer         *ConfigSigner
}

// NewConfigGenerator creates a new configuration generator
func NewConfigGenerator(logger *zap.Logger) *ConfigGenerator {
	return &ConfigGenerator{
		logger:         logger,
		templateEngine: NewTemplateEngine(logger),
		validator:      NewConfigValidator(logger),
		signer:         NewConfigSigner(logger),
	}
}

// GenerateConfig creates a complete configuration from discovered services
func (cg *ConfigGenerator) GenerateConfig(ctx context.Context, services []discovery.ServiceInfo) (*GeneratedConfig, error) {
	cg.logger.Info("Generating configuration", zap.Int("services", len(services)))

	// Build configuration sections
	config := make(map[string]interface{})
	
	// Always include base receivers
	receivers, err := cg.generateReceivers(services)
	if err != nil {
		return nil, fmt.Errorf("failed to generate receivers: %w", err)
	}
	config["receivers"] = receivers

	// Generate processors
	processors, err := cg.generateProcessors()
	if err != nil {
		return nil, fmt.Errorf("failed to generate processors: %w", err)
	}
	config["processors"] = processors

	// Generate exporters
	exporters, err := cg.generateExporters()
	if err != nil {
		return nil, fmt.Errorf("failed to generate exporters: %w", err)
	}
	config["exporters"] = exporters

	// Generate service pipelines
	service, err := cg.generateServicePipelines(services)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service pipelines: %w", err)
	}
	config["service"] = service

	// Generate extensions
	extensions, err := cg.generateExtensions()
	if err != nil {
		return nil, fmt.Errorf("failed to generate extensions: %w", err)
	}
	config["extensions"] = extensions

	// Convert to YAML
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(config); err != nil {
		return nil, fmt.Errorf("failed to encode config: %w", err)
	}

	configYAML := buf.String()

	// Add header comment
	header := cg.generateHeader(services)
	configYAML = header + "\n" + configYAML

	// Validate configuration
	if err := cg.validator.Validate(configYAML); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Sign configuration
	signature, err := cg.signer.Sign([]byte(configYAML))
	if err != nil {
		return nil, fmt.Errorf("failed to sign configuration: %w", err)
	}

	// Identify required variables
	variables := cg.identifyRequiredVariables(services)

	return &GeneratedConfig{
		Version:           fmt.Sprintf("%s-%03d", time.Now().Format("2006-01-02"), 1),
		Config:            configYAML,
		Signature:         signature,
		DiscoveredServices: services,
		RequiredVariables: variables,
		GeneratedAt:       time.Now(),
	}, nil
}

// generateReceivers creates receiver configurations
func (cg *ConfigGenerator) generateReceivers(services []discovery.ServiceInfo) (map[string]interface{}, error) {
	receivers := make(map[string]interface{})

	// Always include host metrics
	receivers["hostmetrics"] = cg.templateEngine.RenderHostMetrics()

	// Add service-specific receivers
	for _, svc := range services {
		receiverConfig, err := cg.templateEngine.RenderServiceReceiver(svc)
		if err != nil {
			cg.logger.Warn("Failed to render receiver", 
				zap.String("service", svc.Type), 
				zap.Error(err))
			continue
		}

		// Add receiver config
		receivers[svc.Type] = receiverConfig

		// Add log receivers if applicable
		logConfigs := cg.templateEngine.RenderLogReceivers(svc)
		for name, config := range logConfigs {
			receivers[name] = config
		}
	}

	// Add system logs
	receivers["filelog/system"] = cg.templateEngine.RenderSystemLogs()

	return receivers, nil
}

// generateProcessors creates processor configurations
func (cg *ConfigGenerator) generateProcessors() (map[string]interface{}, error) {
	return map[string]interface{}{
		"nrsecurity": map[string]interface{}{
			"_comment": "Automatic secret redaction - no configuration needed",
		},
		"nrenrich": map[string]interface{}{
			"host_metadata":     true,
			"cloud_detection":   true,
			"service_detection": true,
		},
		"attributes/services": map[string]interface{}{
			"actions": []map[string]interface{}{
				{
					"key":    "discovered.services",
					"value":  "${DISCOVERED_SERVICES}",
					"action": "insert",
				},
				{
					"key":    "autoconfig.version",
					"value":  "${CONFIG_VERSION}",
					"action": "insert",
				},
			},
		},
		"resource": cg.templateEngine.RenderResourceProcessor(),
		"batch": map[string]interface{}{
			"timeout":             "10s",
			"send_batch_size":     1000,
			"send_batch_max_size": 1500,
		},
		"memory_limiter": map[string]interface{}{
			"check_interval":  "1s",
			"limit_mib":       512,
			"spike_limit_mib": 128,
		},
	}, nil
}

// generateExporters creates exporter configurations
func (cg *ConfigGenerator) generateExporters() (map[string]interface{}, error) {
	return map[string]interface{}{
		"otlp/newrelic": map[string]interface{}{
			"endpoint": "otlp.nr-data.net:4317",
			"headers": map[string]interface{}{
				"api-key": "${NEW_RELIC_LICENSE_KEY}",
			},
			"compression": "gzip",
			"sending_queue": map[string]interface{}{
				"enabled":       true,
				"num_consumers": 4,
				"queue_size":    1000,
			},
			"retry_on_failure": map[string]interface{}{
				"enabled":         true,
				"initial_interval": "5s",
				"max_interval":     "30s",
				"max_elapsed_time": "300s",
			},
			"timeout": "30s",
		},
	}, nil
}

// generateServicePipelines creates pipeline configurations
func (cg *ConfigGenerator) generateServicePipelines(services []discovery.ServiceInfo) (map[string]interface{}, error) {
	// Build receiver lists
	metricsReceivers := []string{"hostmetrics"}
	logsReceivers := []string{"filelog/system"}

	for _, svc := range services {
		metricsReceivers = append(metricsReceivers, svc.Type)
		
		// Add log receivers
		if svc.Type == "mysql" {
			logsReceivers = append(logsReceivers, "filelog/mysql_error", "filelog/mysql_slow")
		} else if svc.Type == "postgresql" {
			logsReceivers = append(logsReceivers, "filelog/postgresql")
		} else if svc.Type == "nginx" {
			logsReceivers = append(logsReceivers, "filelog/nginx_access", "filelog/nginx_error")
		}
		// Add more service-specific logs as needed
	}

	return map[string]interface{}{
		"telemetry": map[string]interface{}{
			"logs": map[string]interface{}{
				"level":        "${LOG_LEVEL:info}",
				"encoding":     "json",
				"output_paths": []string{"stdout", "/var/log/nrdot/collector.log"},
			},
			"metrics": map[string]interface{}{
				"level":   "detailed",
				"address": "127.0.0.1:8888",
			},
		},
		"extensions": []string{"health_check", "zpages"},
		"pipelines": map[string]interface{}{
			"metrics": map[string]interface{}{
				"receivers": metricsReceivers,
				"processors": []string{
					"nrenrich",
					"attributes/services",
					"resource",
					"batch",
					"memory_limiter",
				},
				"exporters": []string{"otlp/newrelic"},
			},
			"logs": map[string]interface{}{
				"receivers": logsReceivers,
				"processors": []string{
					"nrsecurity",
					"nrenrich",
					"attributes/services",
					"resource",
					"batch",
					"memory_limiter",
				},
				"exporters": []string{"otlp/newrelic"},
			},
		},
	}, nil
}

// generateExtensions creates extension configurations
func (cg *ConfigGenerator) generateExtensions() (map[string]interface{}, error) {
	return map[string]interface{}{
		"health_check": map[string]interface{}{
			"endpoint": "127.0.0.1:13133",
			"path":     "/health",
		},
		"zpages": map[string]interface{}{
			"endpoint": "127.0.0.1:55679",
		},
	}, nil
}

// generateHeader creates configuration header comment
func (cg *ConfigGenerator) generateHeader(services []discovery.ServiceInfo) string {
	serviceList := make([]string, 0, len(services))
	for _, svc := range services {
		info := fmt.Sprintf("%s", svc.Type)
		if svc.Version != "" {
			info += fmt.Sprintf(" (%s)", svc.Version)
		}
		if len(svc.Endpoints) > 0 {
			endpoints := make([]string, 0, len(svc.Endpoints))
			for _, ep := range svc.Endpoints {
				endpoints = append(endpoints, fmt.Sprintf("%s:%d", ep.Address, ep.Port))
			}
			info += fmt.Sprintf(" on %s", strings.Join(endpoints, ","))
		}
		serviceList = append(serviceList, "# - " + info)
	}

	return fmt.Sprintf(`# Auto-Generated Configuration
# This file was automatically generated by NRDOT-HOST auto-configuration
# Generated at: %s
# Config version: %s
# Discovered services:
%s
#
# DO NOT EDIT - This file will be regenerated
`, 
		time.Now().Format(time.RFC3339),
		fmt.Sprintf("%s-%03d", time.Now().Format("2006-01-02"), 1),
		strings.Join(serviceList, "\n"))
}

// identifyRequiredVariables identifies environment variables needed
func (cg *ConfigGenerator) identifyRequiredVariables(services []discovery.ServiceInfo) []string {
	required := []string{"NEW_RELIC_LICENSE_KEY"}

	for _, svc := range services {
		switch svc.Type {
		case "mysql":
			required = append(required, "MYSQL_MONITOR_USER", "MYSQL_MONITOR_PASS")
		case "postgresql":
			required = append(required, "POSTGRES_MONITOR_USER", "POSTGRES_MONITOR_PASS")
		case "mongodb":
			required = append(required, "MONGODB_MONITOR_USER", "MONGODB_MONITOR_PASS")
		case "redis":
			if requiresAuth(svc) {
				required = append(required, "REDIS_PASSWORD")
			}
		case "elasticsearch":
			required = append(required, "ELASTICSEARCH_USER", "ELASTICSEARCH_PASS")
		case "rabbitmq":
			required = append(required, "RABBITMQ_USER", "RABBITMQ_PASS")
		}
	}

	return required
}

func requiresAuth(svc discovery.ServiceInfo) bool {
	// Check if service requires authentication based on config or other signals
	// This is a simplified check - in production would check actual config files
	return false
}

// GeneratedConfig represents a generated configuration
type GeneratedConfig struct {
	Version            string                   `json:"version"`
	Config             string                   `json:"config"`
	Signature          string                   `json:"signature"`
	DiscoveredServices []discovery.ServiceInfo  `json:"discovered_services"`
	RequiredVariables  []string                 `json:"required_variables"`
	GeneratedAt        time.Time                `json:"generated_at"`
}

// ConfigValidator validates generated configurations
type ConfigValidator struct {
	logger *zap.Logger
}

func NewConfigValidator(logger *zap.Logger) *ConfigValidator {
	return &ConfigValidator{logger: logger}
}

func (cv *ConfigValidator) Validate(config string) error {
	// Parse YAML to validate structure
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &parsed); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Check required sections
	requiredSections := []string{"receivers", "processors", "exporters", "service"}
	for _, section := range requiredSections {
		if _, exists := parsed[section]; !exists {
			return fmt.Errorf("missing required section: %s", section)
		}
	}

	// Validate service section
	service, ok := parsed["service"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid service section")
	}

	pipelines, ok := service["pipelines"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing pipelines in service section")
	}

	// Check for at least one pipeline
	if len(pipelines) == 0 {
		return fmt.Errorf("no pipelines defined")
	}

	return nil
}

// ConfigSigner signs configurations using ECDSA
type ConfigSigner struct {
	logger     *zap.Logger
	privateKey *ecdsa.PrivateKey
}

func NewConfigSigner(logger *zap.Logger) *ConfigSigner {
	// In production, load from secure storage
	// For now, generate a key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		logger.Fatal("Failed to generate signing key", zap.Error(err))
	}

	return &ConfigSigner{
		logger:     logger,
		privateKey: privateKey,
	}
}

func (cs *ConfigSigner) Sign(data []byte) (string, error) {
	hash := sha256.Sum256(data)
	
	r, s, err := ecdsa.Sign(rand.Reader, cs.privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	// Encode signature
	signature := append(r.Bytes(), s.Bytes()...)
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (cs *ConfigSigner) GetPublicKeyPEM() string {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&cs.privateKey.PublicKey)
	if err != nil {
		return ""
	}

	pubKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	return string(pem.EncodeToMemory(pubKeyPEM))
}

// VerifySignature verifies a configuration signature
func VerifySignature(data []byte, signature string, publicKeyPEM string) error {
	// Decode public key
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return fmt.Errorf("failed to parse PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("not an ECDSA public key")
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	if len(sigBytes) != 64 {
		return fmt.Errorf("invalid signature length")
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	// Verify
	hash := sha256.Sum256(data)
	if !ecdsa.Verify(publicKey, hash[:], r, s) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}