package nrenrich

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"cloud.google.com/go/compute/metadata"
	// "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	// "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"go.uber.org/zap"
)

// MetadataProvider defines the interface for metadata providers
type MetadataProvider interface {
	// GetMetadata returns metadata as key-value pairs
	GetMetadata(ctx context.Context) (map[string]interface{}, error)

	// Name returns the provider name
	Name() string
}

// SystemMetadataProvider provides system information
type SystemMetadataProvider struct {
	logger *zap.Logger
}

func NewSystemMetadataProvider(logger *zap.Logger) *SystemMetadataProvider {
	return &SystemMetadataProvider{logger: logger}
}

func (s *SystemMetadataProvider) Name() string {
	return "system"
}

func (s *SystemMetadataProvider) GetMetadata(ctx context.Context) (map[string]interface{}, error) {
	hostname, err := os.Hostname()
	if err != nil {
		s.logger.Warn("Failed to get hostname", zap.Error(err))
		hostname = "unknown"
	}

	return map[string]interface{}{
		"host.name":     hostname,
		"host.arch":     runtime.GOARCH,
		"host.os":       runtime.GOOS,
		"host.cpu.count": runtime.NumCPU(),
	}, nil
}

// AWSMetadataProvider provides AWS EC2 metadata
type AWSMetadataProvider struct {
	logger *zap.Logger
	client *imds.Client
	mu     sync.RWMutex
	cache  map[string]interface{}
	cacheTime time.Time
}

func NewAWSMetadataProvider(logger *zap.Logger) *AWSMetadataProvider {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Warn("Failed to load AWS config", zap.Error(err))
		return nil
	}

	return &AWSMetadataProvider{
		logger: logger,
		client: imds.NewFromConfig(cfg),
		cache:  make(map[string]interface{}),
	}
}

func (a *AWSMetadataProvider) Name() string {
	return "aws"
}

func (a *AWSMetadataProvider) GetMetadata(ctx context.Context) (map[string]interface{}, error) {
	a.mu.RLock()
	if time.Since(a.cacheTime) < 5*time.Minute && len(a.cache) > 0 {
		cache := make(map[string]interface{})
		for k, v := range a.cache {
			cache[k] = v
		}
		a.mu.RUnlock()
		return cache, nil
	}
	a.mu.RUnlock()

	metadata := make(map[string]interface{})

	// Get instance ID
	if instanceID, err := a.client.GetMetadata(ctx, &imds.GetMetadataInput{
		Path: "instance-id",
	}); err == nil {
		metadata["cloud.provider"] = "aws"
		metadata["cloud.platform"] = "aws_ec2"
		metadata["cloud.instance.id"] = instanceID.Content
	}

	// Get region
	if region, err := a.client.GetRegion(ctx, &imds.GetRegionInput{}); err == nil {
		metadata["cloud.region"] = region.Region
	}

	// Get availability zone
	if az, err := a.client.GetMetadata(ctx, &imds.GetMetadataInput{
		Path: "placement/availability-zone",
	}); err == nil {
		metadata["cloud.availability_zone"] = az.Content
	}

	// Get instance type
	if instanceType, err := a.client.GetMetadata(ctx, &imds.GetMetadataInput{
		Path: "instance-type",
	}); err == nil {
		metadata["cloud.instance.type"] = instanceType.Content
	}

	a.mu.Lock()
	a.cache = metadata
	a.cacheTime = time.Now()
	a.mu.Unlock()

	return metadata, nil
}

// GCPMetadataProvider provides Google Cloud metadata
type GCPMetadataProvider struct {
	logger *zap.Logger
	mu     sync.RWMutex
	cache  map[string]interface{}
	cacheTime time.Time
}

func NewGCPMetadataProvider(logger *zap.Logger) *GCPMetadataProvider {
	if !metadata.OnGCE() {
		return nil
	}

	return &GCPMetadataProvider{
		logger: logger,
		cache:  make(map[string]interface{}),
	}
}

func (g *GCPMetadataProvider) Name() string {
	return "gcp"
}

func (g *GCPMetadataProvider) GetMetadata(ctx context.Context) (map[string]interface{}, error) {
	g.mu.RLock()
	if time.Since(g.cacheTime) < 5*time.Minute && len(g.cache) > 0 {
		cache := make(map[string]interface{})
		for k, v := range g.cache {
			cache[k] = v
		}
		g.mu.RUnlock()
		return cache, nil
	}
	g.mu.RUnlock()

	gcpMetadata := map[string]interface{}{
		"cloud.provider": "gcp",
		"cloud.platform": "gcp_compute_engine",
	}

	// Get project ID
	if projectID, err := metadata.ProjectID(); err == nil {
		gcpMetadata["cloud.account.id"] = projectID
	}

	// Get instance ID
	if instanceID, err := metadata.InstanceID(); err == nil {
		gcpMetadata["cloud.instance.id"] = instanceID
	}

	// Get zone
	if zone, err := metadata.Zone(); err == nil {
		gcpMetadata["cloud.availability_zone"] = zone
	}

	// Get machine type
	if machineType, err := metadata.Get("instance/machine-type"); err == nil {
		gcpMetadata["cloud.instance.type"] = machineType
	}

	g.mu.Lock()
	g.cache = gcpMetadata
	g.cacheTime = time.Now()
	g.mu.Unlock()

	return gcpMetadata, nil
}

// AzureMetadataProvider provides Azure metadata
type AzureMetadataProvider struct {
	logger *zap.Logger
	mu     sync.RWMutex
	cache  map[string]interface{}
	cacheTime time.Time
}

func NewAzureMetadataProvider(logger *zap.Logger) *AzureMetadataProvider {
	// Check if running on Azure by looking for Azure-specific environment
	if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
		return nil
	}

	return &AzureMetadataProvider{
		logger: logger,
		cache:  make(map[string]interface{}),
	}
}

func (a *AzureMetadataProvider) Name() string {
	return "azure"
}

func (a *AzureMetadataProvider) GetMetadata(ctx context.Context) (map[string]interface{}, error) {
	a.mu.RLock()
	if time.Since(a.cacheTime) < 5*time.Minute && len(a.cache) > 0 {
		cache := make(map[string]interface{})
		for k, v := range a.cache {
			cache[k] = v
		}
		a.mu.RUnlock()
		return cache, nil
	}
	a.mu.RUnlock()

	metadata := map[string]interface{}{
		"cloud.provider": "azure",
		"cloud.platform": "azure_vm",
	}

	// Get basic metadata from environment
	if subID := os.Getenv("AZURE_SUBSCRIPTION_ID"); subID != "" {
		metadata["cloud.account.id"] = subID
	}

	if resourceGroup := os.Getenv("AZURE_RESOURCE_GROUP"); resourceGroup != "" {
		metadata["cloud.resource_group"] = resourceGroup
	}

	a.mu.Lock()
	a.cache = metadata
	a.cacheTime = time.Now()
	a.mu.Unlock()

	return metadata, nil
}

// KubernetesMetadataProvider provides Kubernetes metadata
type KubernetesMetadataProvider struct {
	logger *zap.Logger
	mu     sync.RWMutex
	cache  map[string]interface{}
	cacheTime time.Time
}

func NewKubernetesMetadataProvider(logger *zap.Logger) *KubernetesMetadataProvider {
	// Check if running in Kubernetes
	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		return nil
	}

	return &KubernetesMetadataProvider{
		logger: logger,
		cache:  make(map[string]interface{}),
	}
}

func (k *KubernetesMetadataProvider) Name() string {
	return "kubernetes"
}

func (k *KubernetesMetadataProvider) GetMetadata(ctx context.Context) (map[string]interface{}, error) {
	k.mu.RLock()
	if time.Since(k.cacheTime) < 5*time.Minute && len(k.cache) > 0 {
		cache := make(map[string]interface{})
		for key, v := range k.cache {
			cache[key] = v
		}
		k.mu.RUnlock()
		return cache, nil
	}
	k.mu.RUnlock()

	metadata := make(map[string]interface{})

	// Get metadata from environment variables (downward API)
	if podName := os.Getenv("HOSTNAME"); podName != "" {
		metadata["k8s.pod.name"] = podName
	}

	if namespace := os.Getenv("NAMESPACE"); namespace != "" {
		metadata["k8s.namespace.name"] = namespace
	}

	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		metadata["k8s.node.name"] = nodeName
	}

	if podIP := os.Getenv("POD_IP"); podIP != "" {
		metadata["k8s.pod.ip"] = podIP
	}

	k.mu.Lock()
	k.cache = metadata
	k.cacheTime = time.Now()
	k.mu.Unlock()

	return metadata, nil
}