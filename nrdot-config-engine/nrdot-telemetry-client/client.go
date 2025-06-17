package telemetry

import (
	"time"
)

// Config holds telemetry client configuration
type Config struct {
	Endpoint        string
	ReportInterval  time.Duration
	BatchSize       int
	FlushTimeout    time.Duration
	MaxRetries      int
	RetryBackoff    time.Duration
	Headers         map[string]string
	CompressionType CompressionType
}

// CompressionType defines compression types
type CompressionType string

const (
	CompressionNone CompressionType = "none"
	CompressionGzip CompressionType = "gzip"
)

// DefaultConfig returns default telemetry configuration
func DefaultConfig() Config {
	return Config{
		Endpoint:        "http://localhost:4318/v1/metrics",
		ReportInterval:  10 * time.Second,
		BatchSize:       100,
		FlushTimeout:    5 * time.Second,
		MaxRetries:      3,
		RetryBackoff:    time.Second,
		Headers:         map[string]string{},
		CompressionType: CompressionGzip,
	}
}

// Metric represents a telemetry metric
type Metric struct {
	Name      string
	Value     float64
	Timestamp time.Time
	Tags      map[string]string
}

// Client is a telemetry client
type Client struct {
	config Config
}

// NewClient creates a new telemetry client
func NewClient(config Config) (*Client, error) {
	return &Client{config: config}, nil
}

// Start starts the telemetry client
func (c *Client) Start() error {
	// Mock implementation
	return nil
}

// Stop stops the telemetry client
func (c *Client) Stop() {
	// Mock implementation
}

// SendMetric sends a metric
func (c *Client) SendMetric(metric Metric) error {
	// Mock implementation
	return nil
}