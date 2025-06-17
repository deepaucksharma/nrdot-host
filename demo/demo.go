package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// Demo NRDOT-HOST functionality
func main() {
	fmt.Println("=================================")
	fmt.Println("NRDOT-HOST Component Demo")
	fmt.Println("=================================")
	fmt.Println()

	// Demo 1: Security Processor
	demoSecurityProcessor()
	fmt.Println()

	// Demo 2: Enrichment Processor
	demoEnrichmentProcessor()
	fmt.Println()

	// Demo 3: Transform Processor
	demoTransformProcessor()
	fmt.Println()

	// Demo 4: Cardinality Processor
	demoCardinalityProcessor()
}

func demoSecurityProcessor() {
	fmt.Println("1. Security Processor Demo")
	fmt.Println("--------------------------")

	// Simulate sensitive data
	sensitiveData := map[string]string{
		"message":     "User login with password=secret123",
		"api_key":     "sk-1234567890abcdef",
		"credit_card": "4111-1111-1111-1111",
		"email":       "user@example.com",
	}

	fmt.Println("Before redaction:")
	for k, v := range sensitiveData {
		fmt.Printf("  %s: %s\n", k, v)
	}

	// Simulate redaction
	redacted := map[string]string{
		"message":     "User login with password=[REDACTED]",
		"api_key":     "[REDACTED]",
		"credit_card": "****-****-****-1111",
		"email":       "[REDACTED]",
	}

	fmt.Println("\nAfter redaction:")
	for k, v := range redacted {
		fmt.Printf("  %s: %s\n", k, v)
	}
}

func demoEnrichmentProcessor() {
	fmt.Println("2. Enrichment Processor Demo")
	fmt.Println("----------------------------")

	// Original metric
	fmt.Println("Original metric:")
	fmt.Println("  name: http.request.duration")
	fmt.Println("  value: 125.5")
	fmt.Println("  attributes:")
	fmt.Println("    http.method: GET")
	fmt.Println("    http.route: /api/users")

	// After enrichment
	fmt.Println("\nAfter enrichment:")
	fmt.Println("  name: http.request.duration")
	fmt.Println("  value: 125.5")
	fmt.Println("  attributes:")
	fmt.Println("    http.method: GET")
	fmt.Println("    http.route: /api/users")
	fmt.Println("    host.name: demo-host")
	fmt.Println("    cloud.provider: local")
	fmt.Println("    service.version: 1.0.0")
	fmt.Println("    environment: demo")
}

func demoTransformProcessor() {
	fmt.Println("3. Transform Processor Demo")
	fmt.Println("---------------------------")

	// Unit conversion
	bytes := 1073741824.0 // 1GB in bytes
	gb := bytes / (1024 * 1024 * 1024)
	
	fmt.Printf("Unit conversion:\n")
	fmt.Printf("  Original: %.0f bytes\n", bytes)
	fmt.Printf("  Converted: %.2f GB\n", gb)

	// Rate calculation
	fmt.Println("\nRate calculation:")
	requests := []float64{1000, 1050, 1100, 1150, 1200}
	fmt.Println("  Request counts: ", requests)
	fmt.Print("  Calculated rates: [")
	for i := 1; i < len(requests); i++ {
		rate := requests[i] - requests[i-1]
		fmt.Printf("%.0f", rate)
		if i < len(requests)-1 {
			fmt.Print(", ")
		}
	}
	fmt.Println("] requests/interval")

	// Error percentage
	totalReqs := 1000.0
	errorReqs := 50.0
	errorRate := (errorReqs / totalReqs) * 100
	fmt.Printf("\nError rate calculation:\n")
	fmt.Printf("  Total requests: %.0f\n", totalReqs)
	fmt.Printf("  Error requests: %.0f\n", errorReqs)
	fmt.Printf("  Error rate: %.1f%%\n", errorRate)
}

func demoCardinalityProcessor() {
	fmt.Println("4. Cardinality Processor Demo")
	fmt.Println("-----------------------------")

	// Simulate high cardinality scenario
	fmt.Println("Simulating cardinality growth:")
	
	cardinality := 0
	limit := 1000
	
	// Simulate metrics with different cardinalities
	metrics := []struct {
		name        string
		dimensions  int
		values      int
	}{
		{"http.request.duration", 4, 100},    // 400 series
		{"database.query.time", 3, 200},      // 600 series
		{"user.action.count", 5, 1000},       // 5000 series - will be limited
	}

	fmt.Printf("\nCardinality limit: %d\n\n", limit)

	for _, m := range metrics {
		series := m.dimensions * m.values
		fmt.Printf("Metric: %s\n", m.name)
		fmt.Printf("  Dimensions: %d, Unique values: %d\n", m.dimensions, m.values)
		fmt.Printf("  Potential series: %d\n", series)
		
		if cardinality+series > limit {
			dropped := (cardinality + series) - limit
			kept := series - dropped
			fmt.Printf("  Status: LIMITED (kept %d, dropped %d)\n", kept, dropped)
			cardinality = limit
		} else {
			fmt.Printf("  Status: OK\n")
			cardinality += series
		}
		fmt.Printf("  Current total cardinality: %d\n\n", cardinality)
	}
}

// Helper function to create sample metrics
func createSampleMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	
	// Set resource attributes
	rm.Resource().Attributes().PutStr("service.name", "demo-service")
	rm.Resource().Attributes().PutStr("host.name", "demo-host")
	
	// Add scope metrics
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("demo")
	
	// Add a gauge metric
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("system.memory.usage")
	metric.SetUnit("bytes")
	gauge := metric.SetEmptyGauge()
	
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetDoubleValue(float64(rand.Intn(1000000000))) // Random memory value
	
	return metrics
}