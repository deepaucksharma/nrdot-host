package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

var (
	metricCount      int
	cardinalityLevel string
	meter           metric.Meter
)

func init() {
	// Get configuration from environment
	count := os.Getenv("METRIC_COUNT")
	if count == "" {
		metricCount = 1000
	} else {
		metricCount, _ = strconv.Atoi(count)
	}

	cardinalityLevel = os.Getenv("CARDINALITY_LEVEL")
	if cardinalityLevel == "" {
		cardinalityLevel = "medium"
	}
}

func initMeter() func() {
	ctx := context.Background()

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
	)
	if err != nil {
		log.Fatal("Failed to create metrics exporter:", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("metric-generator"),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("deployment.environment", "cardinality-test"),
		),
		resource.WithFromEnv(),
	)
	if err != nil {
		log.Fatal("Failed to create resource:", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second)),
		),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(provider)
	meter = provider.Meter("high-cardinality-generator")

	return func() {
		if err := provider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
		}
	}
}

func generateHighCardinalityMetrics(ctx context.Context) {
	// Create various metric types with high cardinality
	
	// Counter with high cardinality labels
	requestCounter, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Printf("Failed to create counter: %v", err)
		return
	}

	// Histogram with high cardinality
	latencyHistogram, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request latency"),
		metric.WithUnit("s"),
	)
	if err != nil {
		log.Printf("Failed to create histogram: %v", err)
		return
	}

	// Gauge with dynamic labels
	activeConnectionsGauge, err := meter.Int64UpDownCounter(
		"active_connections",
		metric.WithDescription("Number of active connections"),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Printf("Failed to create gauge: %v", err)
		return
	}

	// Generate metrics with varying cardinality
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	labelSets := generateLabelSets()
	iteration := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Pick a random label set
			labels := labelSets[rand.Intn(len(labelSets))]
			
			// Record metrics
			requestCounter.Add(ctx, 1, metric.WithAttributes(labels...))
			
			latency := rand.Float64() * 5.0 // 0-5 seconds
			latencyHistogram.Record(ctx, latency, metric.WithAttributes(labels...))
			
			// Simulate connection changes
			if rand.Float64() > 0.5 {
				activeConnectionsGauge.Add(ctx, 1, metric.WithAttributes(labels...))
			} else {
				activeConnectionsGauge.Add(ctx, -1, metric.WithAttributes(labels...))
			}
			
			// Create new metric every N iterations to test cardinality limits
			if iteration%100 == 0 {
				createDynamicMetric(ctx, iteration)
			}
			
			iteration++
			
			if iteration%1000 == 0 {
				log.Printf("Generated %d metric updates", iteration)
			}
		}
	}
}

func generateLabelSets() [][]attribute.KeyValue {
	var labelSets [][]attribute.KeyValue
	
	// Base labels that create cardinality
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	statusCodes := []int{200, 201, 204, 301, 302, 400, 401, 403, 404, 500, 502, 503}
	
	// Generate label combinations based on cardinality level
	var userCount, pathCount, hostCount int
	
	switch cardinalityLevel {
	case "low":
		userCount = 10
		pathCount = 20
		hostCount = 5
	case "high":
		userCount = 1000
		pathCount = 500
		hostCount = 100
	case "extreme":
		userCount = 10000
		pathCount = 5000
		hostCount = 1000
	default: // medium
		userCount = 100
		pathCount = 50
		hostCount = 20
	}
	
	// Create label sets
	for i := 0; i < metricCount; i++ {
		labels := []attribute.KeyValue{
			attribute.String("method", methods[rand.Intn(len(methods))]),
			attribute.Int("status_code", statusCodes[rand.Intn(len(statusCodes))]),
			attribute.String("user_id", fmt.Sprintf("user_%d", rand.Intn(userCount))),
			attribute.String("path", fmt.Sprintf("/api/v1/resource_%d", rand.Intn(pathCount))),
			attribute.String("host", fmt.Sprintf("host-%d.example.com", rand.Intn(hostCount))),
			attribute.String("region", []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}[rand.Intn(4)]),
			attribute.String("environment", []string{"prod", "staging", "dev"}[rand.Intn(3)]),
			attribute.String("version", fmt.Sprintf("v1.%d.%d", rand.Intn(10), rand.Intn(100))),
		}
		
		// Add dynamic labels for extreme cardinality
		if cardinalityLevel == "extreme" {
			labels = append(labels,
				attribute.String("trace_id", fmt.Sprintf("%x", rand.Int63())),
				attribute.String("session_id", fmt.Sprintf("sess_%x", rand.Int63())),
				attribute.String("request_id", fmt.Sprintf("req_%x", rand.Int63())),
			)
		}
		
		labelSets = append(labelSets, labels)
	}
	
	return labelSets
}

func createDynamicMetric(ctx context.Context, iteration int) {
	// Create a new metric with unique name to test metric creation limits
	metricName := fmt.Sprintf("dynamic_metric_%d", iteration/100)
	
	counter, err := meter.Int64Counter(
		metricName,
		metric.WithDescription("Dynamically created metric"),
		metric.WithUnit("1"),
	)
	if err != nil {
		log.Printf("Failed to create dynamic metric %s: %v", metricName, err)
		return
	}
	
	// Record some data
	for i := 0; i < 10; i++ {
		counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("dynamic_label", fmt.Sprintf("value_%d", i)),
		))
	}
}

func main() {
	shutdown := initMeter()
	defer shutdown()

	ctx := context.Background()

	log.Printf("Starting metric generator...")
	log.Printf("Configuration: metricCount=%d, cardinalityLevel=%s", metricCount, cardinalityLevel)

	// Start generating metrics
	generateHighCardinalityMetrics(ctx)
}