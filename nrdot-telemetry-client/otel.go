package telemetryclient

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// initTracer initializes the OpenTelemetry trace provider
func initTracer(config Config, res *resource.Resource) (*trace.TracerProvider, error) {
	ctx := context.Background()

	// Create gRPC connection with timeout
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	conn, err := grpc.DialContext(dialCtx, config.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	// Create trace exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithGRPCConn(conn),
		otlptracegrpc.WithHeaders(map[string]string{
			"api-key": config.APIKey,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter,
			trace.WithMaxExportBatchSize(512),
			trace.WithBatchTimeout(30*time.Second),
		),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()), // Sample all telemetry data
	)

	return tp, nil
}

// initMeter initializes the OpenTelemetry meter provider
func initMeter(config Config, res *resource.Resource) (*metric.MeterProvider, error) {
	// For now, use a simple in-memory metric reader
	// In production, you would configure an OTLP metric exporter here
	
	interval := config.Interval
	if interval == 0 {
		interval = 60 * time.Second
	}

	// Create meter provider with in-memory reader
	mp := metric.NewMeterProvider(
		metric.WithResource(res),
	)

	return mp, nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		ServiceName:    "nrdot-host",
		ServiceVersion: "unknown",
		Environment:    "production",
		Endpoint:       "otlp.nr-data.net:4317",
		Interval:       60 * time.Second,
		Enabled:        true,
	}
}