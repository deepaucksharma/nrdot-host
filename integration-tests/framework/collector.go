package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CollectorConfig contains configuration for a collector instance
type CollectorConfig struct {
	Name           string
	Image          string
	ConfigPath     string
	Network        testcontainers.Network
	Logger         *zap.Logger
	HealthEndpoint string
	MetricsPort    int
	OTLPGRPCPort   int
	OTLPHTTPPort   int
	EnvVars        map[string]string
	Volumes        map[string]string
}

// CollectorInstance represents a running collector instance
type CollectorInstance struct {
	config        *CollectorConfig
	container     testcontainers.Container
	logger        *zap.Logger
	metricsURL    string
	otlpGRPCURL   string
	otlpHTTPURL   string
	healthURL     string
	grpcConn      *grpc.ClientConn
	dockerClient  *client.Client
}

// NewCollectorInstance creates a new collector instance
func NewCollectorInstance(ctx context.Context, config *CollectorConfig) (*CollectorInstance, error) {
	// Prepare container request
	req := testcontainers.ContainerRequest{
		Image: config.Image,
		ExposedPorts: []string{
			fmt.Sprintf("%d/tcp", config.MetricsPort),
			fmt.Sprintf("%d/tcp", config.OTLPGRPCPort),
			fmt.Sprintf("%d/tcp", config.OTLPHTTPPort),
		},
		Env: config.EnvVars,
		Networks: []string{},
		WaitingFor: wait.ForAll(
			wait.ForHTTP(config.HealthEndpoint).WithPort(nat.Port(fmt.Sprintf("%d/tcp", config.MetricsPort))),
			wait.ForListeningPort(nat.Port(fmt.Sprintf("%d/tcp", config.OTLPGRPCPort))),
		),
		Name: config.Name,
	}

	// Add network if provided
	if config.Network != nil {
		if dockerNetwork, ok := config.Network.(*testcontainers.DockerNetwork); ok {
			req.Networks = []string{dockerNetwork.Name}
		}
	}

	// Mount config file
	if config.ConfigPath != "" {
		req.Files = []testcontainers.ContainerFile{
			{
				HostFilePath:      config.ConfigPath,
				ContainerFilePath: "/etc/otelcol/config.yaml",
				FileMode:          0644,
			},
		}
		req.Cmd = []string{"--config", "/etc/otelcol/config.yaml"}
	}

	// Add volumes
	if len(config.Volumes) > 0 {
		req.Mounts = make([]testcontainers.ContainerMount, 0, len(config.Volumes))
		for host, container := range config.Volumes {
			req.Mounts = append(req.Mounts, testcontainers.BindMount(host, testcontainers.ContainerMountTarget(container)))
		}
	}

	// Create container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Get endpoints
	metricsURL, err := container.Endpoint(ctx, "http")
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics endpoint: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	grpcPort, err := container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d", config.OTLPGRPCPort)))
	if err != nil {
		return nil, fmt.Errorf("failed to get GRPC port: %w", err)
	}

	httpPort, err := container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d", config.OTLPHTTPPort)))
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP port: %w", err)
	}

	// Create Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	instance := &CollectorInstance{
		config:       config,
		container:    container,
		logger:       config.Logger,
		metricsURL:   fmt.Sprintf("http://%s:%s", host, grpcPort.Port()),
		otlpGRPCURL:  fmt.Sprintf("%s:%s", host, grpcPort.Port()),
		otlpHTTPURL:  fmt.Sprintf("http://%s:%s", host, httpPort.Port()),
		healthURL:    fmt.Sprintf("%s%s", metricsURL, config.HealthEndpoint),
		dockerClient: dockerClient,
	}

	return instance, nil
}

// Start starts the collector instance
func (c *CollectorInstance) Start(ctx context.Context) error {
	return c.container.Start(ctx)
}

// Stop stops the collector instance
func (c *CollectorInstance) Stop(ctx context.Context) error {
	if c.grpcConn != nil {
		c.grpcConn.Close()
	}
	return c.container.Terminate(ctx)
}

// Restart restarts the collector instance
func (c *CollectorInstance) Restart(ctx context.Context) error {
	if err := c.Stop(ctx); err != nil {
		return err
	}
	return c.Start(ctx)
}

// WaitReady waits for the collector to be ready
func (c *CollectorInstance) WaitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(c.healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			c.logger.Info("Collector is ready")
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			// Continue waiting
		}
	}
	return fmt.Errorf("collector did not become ready within %v", timeout)
}

// UpdateConfig updates the collector configuration
func (c *CollectorInstance) UpdateConfig(ctx context.Context, configData []byte) error {
	// Write new config to a temporary file
	tempFile, err := os.CreateTemp("", "collector-config-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(configData); err != nil {
		return err
	}
	tempFile.Close()

	// Copy new config to container
	err = c.container.CopyFileToContainer(ctx, tempFile.Name(), "/etc/otelcol/config.yaml", 0644)
	if err != nil {
		return err
	}

	// Send SIGHUP to reload config
	containerID := c.container.GetContainerID()
	return c.dockerClient.ContainerKill(ctx, containerID, "SIGHUP")
}

// GetLogs returns the collector logs
func (c *CollectorInstance) GetLogs(ctx context.Context) (string, error) {
	reader, err := c.container.Logs(ctx)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	logs, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}

// GetMetrics returns collector internal metrics
func (c *CollectorInstance) GetMetrics(ctx context.Context) ([]byte, error) {
	resp, err := http.Get(c.metricsURL + "/metrics")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// SendMetrics sends metrics to the collector via OTLP
func (c *CollectorInstance) SendMetrics(metrics pmetric.Metrics) error {
	// TODO: Implement OTLP metric sending
	return nil
}

// SendTraces sends traces to the collector via OTLP
func (c *CollectorInstance) SendTraces(traces ptrace.Traces) error {
	// TODO: Implement OTLP trace sending
	return nil
}

// SendLogs sends logs to the collector via OTLP
func (c *CollectorInstance) SendLogs(logs plog.Logs) error {
	// TODO: Implement OTLP log sending
	return nil
}

// GetOTLPGRPCEndpoint returns the OTLP GRPC endpoint
func (c *CollectorInstance) GetOTLPGRPCEndpoint() string {
	return c.otlpGRPCURL
}

// GetOTLPHTTPEndpoint returns the OTLP HTTP endpoint
func (c *CollectorInstance) GetOTLPHTTPEndpoint() string {
	return c.otlpHTTPURL
}

// GetMetricsEndpoint returns the metrics endpoint
func (c *CollectorInstance) GetMetricsEndpoint() string {
	return c.metricsURL
}

// GetContainerStats returns container resource usage statistics
func (c *CollectorInstance) GetContainerStats(ctx context.Context) (*types.StatsJSON, error) {
	containerID := c.container.GetContainerID()
	
	stats, err := c.dockerClient.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var statsJSON types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
		return nil, err
	}

	return &statsJSON, nil
}

// GetGRPCConnection returns a GRPC connection to the collector
func (c *CollectorInstance) GetGRPCConnection() (*grpc.ClientConn, error) {
	if c.grpcConn != nil {
		return c.grpcConn, nil
	}

	conn, err := grpc.Dial(c.otlpGRPCURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	c.grpcConn = conn
	return conn, nil
}

// ExecCommand executes a command inside the collector container
func (c *CollectorInstance) ExecCommand(ctx context.Context, cmd []string) (string, error) {
	containerID := c.container.GetContainerID()

	execConfig := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := c.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", err
	}

	resp, err := c.dockerClient.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{})
	if err != nil {
		return "", err
	}
	defer resp.Close()

	output, err := io.ReadAll(resp.Reader)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// GetResourceUsage returns current CPU and memory usage
func (c *CollectorInstance) GetResourceUsage(ctx context.Context) (float64, uint64, error) {
	stats, err := c.GetContainerStats(ctx)
	if err != nil {
		return 0, 0, err
	}

	// Calculate CPU usage percentage
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	cpuPercent := (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0

	// Memory usage in bytes
	memoryUsage := stats.MemoryStats.Usage

	return cpuPercent, memoryUsage, nil
}