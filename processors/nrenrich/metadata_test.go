package nrenrich

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSystemMetadataProvider(t *testing.T) {
	logger := zap.NewNop()
	provider := NewSystemMetadataProvider(logger)

	assert.Equal(t, "system", provider.Name())

	metadata, err := provider.GetMetadata(context.Background())
	require.NoError(t, err)

	// Check expected system metadata
	assert.Contains(t, metadata, "host.name")
	assert.Equal(t, runtime.GOARCH, metadata["host.arch"])
	assert.Equal(t, runtime.GOOS, metadata["host.os"])
	assert.Equal(t, runtime.NumCPU(), metadata["host.cpu.count"])
}

func TestKubernetesMetadataProvider(t *testing.T) {
	logger := zap.NewNop()

	// Test when not in Kubernetes
	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	provider := NewKubernetesMetadataProvider(logger)
	assert.Nil(t, provider)

	// Test when in Kubernetes
	os.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
	defer os.Unsetenv("KUBERNETES_SERVICE_HOST")

	provider = NewKubernetesMetadataProvider(logger)
	require.NotNil(t, provider)
	assert.Equal(t, "kubernetes", provider.Name())

	// Set some test environment variables
	os.Setenv("HOSTNAME", "test-pod")
	os.Setenv("NAMESPACE", "test-namespace")
	os.Setenv("NODE_NAME", "test-node")
	os.Setenv("POD_IP", "10.1.2.3")
	defer func() {
		os.Unsetenv("HOSTNAME")
		os.Unsetenv("NAMESPACE")
		os.Unsetenv("NODE_NAME")
		os.Unsetenv("POD_IP")
	}()

	metadata, err := provider.GetMetadata(context.Background())
	require.NoError(t, err)

	// Check metadata
	assert.Equal(t, "test-pod", metadata["k8s.pod.name"])
	assert.Equal(t, "test-namespace", metadata["k8s.namespace.name"])
	assert.Equal(t, "test-node", metadata["k8s.node.name"])
	assert.Equal(t, "10.1.2.3", metadata["k8s.pod.ip"])

	// Test caching
	metadata2, err := provider.GetMetadata(context.Background())
	require.NoError(t, err)
	assert.Equal(t, metadata, metadata2)
}

func TestAWSMetadataProvider(t *testing.T) {
	// This test is skipped in most environments as it requires AWS IMDS
	t.Skip("Skipping AWS metadata provider test - requires AWS environment")
}

func TestGCPMetadataProvider(t *testing.T) {
	// This test is skipped in most environments as it requires GCP metadata service
	t.Skip("Skipping GCP metadata provider test - requires GCP environment")
}

func TestAzureMetadataProvider(t *testing.T) {
	logger := zap.NewNop()

	// Test when not on Azure
	os.Unsetenv("AZURE_SUBSCRIPTION_ID")
	provider := NewAzureMetadataProvider(logger)
	assert.Nil(t, provider)

	// Test when on Azure
	os.Setenv("AZURE_SUBSCRIPTION_ID", "test-sub-id")
	os.Setenv("AZURE_RESOURCE_GROUP", "test-rg")
	defer func() {
		os.Unsetenv("AZURE_SUBSCRIPTION_ID")
		os.Unsetenv("AZURE_RESOURCE_GROUP")
	}()

	provider = NewAzureMetadataProvider(logger)
	require.NotNil(t, provider)
	assert.Equal(t, "azure", provider.Name())

	metadata, err := provider.GetMetadata(context.Background())
	require.NoError(t, err)

	// Check metadata
	assert.Equal(t, "azure", metadata["cloud.provider"])
	assert.Equal(t, "azure_vm", metadata["cloud.platform"])
	assert.Equal(t, "test-sub-id", metadata["cloud.account.id"])
	assert.Equal(t, "test-rg", metadata["cloud.resource_group"])
}