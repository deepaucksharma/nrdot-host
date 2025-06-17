package framework

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// AssertMetricsEqual compares two metric sets for equality
func AssertMetricsEqual(t *testing.T, expected, actual pmetric.Metrics) {
	require.Equal(t, expected.ResourceMetrics().Len(), actual.ResourceMetrics().Len(),
		"Different number of resource metrics")

	for i := 0; i < expected.ResourceMetrics().Len(); i++ {
		expectedRM := expected.ResourceMetrics().At(i)
		actualRM := actual.ResourceMetrics().At(i)

		// Compare resource attributes
		AssertAttributesEqual(t, expectedRM.Resource().Attributes(), actualRM.Resource().Attributes())

		// Compare scope metrics
		require.Equal(t, expectedRM.ScopeMetrics().Len(), actualRM.ScopeMetrics().Len(),
			"Different number of scope metrics")

		for j := 0; j < expectedRM.ScopeMetrics().Len(); j++ {
			expectedSM := expectedRM.ScopeMetrics().At(j)
			actualSM := actualRM.ScopeMetrics().At(j)

			// Compare metrics
			require.Equal(t, expectedSM.Metrics().Len(), actualSM.Metrics().Len(),
				"Different number of metrics")

			for k := 0; k < expectedSM.Metrics().Len(); k++ {
				AssertMetricEqual(t, expectedSM.Metrics().At(k), actualSM.Metrics().At(k))
			}
		}
	}
}

// AssertMetricEqual compares two metrics for equality
func AssertMetricEqual(t *testing.T, expected, actual pmetric.Metric) {
	assert.Equal(t, expected.Name(), actual.Name(), "Metric names don't match")
	assert.Equal(t, expected.Description(), actual.Description(), "Metric descriptions don't match")
	assert.Equal(t, expected.Unit(), actual.Unit(), "Metric units don't match")
	assert.Equal(t, expected.Type(), actual.Type(), "Metric types don't match")

	switch expected.Type() {
	case pmetric.MetricTypeGauge:
		AssertGaugeEqual(t, expected.Gauge(), actual.Gauge())
	case pmetric.MetricTypeSum:
		AssertSumEqual(t, expected.Sum(), actual.Sum())
	case pmetric.MetricTypeHistogram:
		AssertHistogramEqual(t, expected.Histogram(), actual.Histogram())
	case pmetric.MetricTypeSummary:
		AssertSummaryEqual(t, expected.Summary(), actual.Summary())
	}
}

// AssertGaugeEqual compares two gauges for equality
func AssertGaugeEqual(t *testing.T, expected, actual pmetric.Gauge) {
	require.Equal(t, expected.DataPoints().Len(), actual.DataPoints().Len(),
		"Different number of gauge data points")

	for i := 0; i < expected.DataPoints().Len(); i++ {
		expectedDP := expected.DataPoints().At(i)
		actualDP := actual.DataPoints().At(i)

		AssertAttributesEqual(t, expectedDP.Attributes(), actualDP.Attributes())
		assert.Equal(t, expectedDP.Timestamp(), actualDP.Timestamp())
		assert.Equal(t, expectedDP.DoubleValue(), actualDP.DoubleValue())
	}
}

// AssertSumEqual compares two sums for equality
func AssertSumEqual(t *testing.T, expected, actual pmetric.Sum) {
	assert.Equal(t, expected.IsMonotonic(), actual.IsMonotonic())
	assert.Equal(t, expected.AggregationTemporality(), actual.AggregationTemporality())

	require.Equal(t, expected.DataPoints().Len(), actual.DataPoints().Len(),
		"Different number of sum data points")

	for i := 0; i < expected.DataPoints().Len(); i++ {
		expectedDP := expected.DataPoints().At(i)
		actualDP := actual.DataPoints().At(i)

		AssertAttributesEqual(t, expectedDP.Attributes(), actualDP.Attributes())
		assert.Equal(t, expectedDP.Timestamp(), actualDP.Timestamp())
		assert.Equal(t, expectedDP.DoubleValue(), actualDP.DoubleValue())
	}
}

// AssertHistogramEqual compares two histograms for equality
func AssertHistogramEqual(t *testing.T, expected, actual pmetric.Histogram) {
	assert.Equal(t, expected.AggregationTemporality(), actual.AggregationTemporality())

	require.Equal(t, expected.DataPoints().Len(), actual.DataPoints().Len(),
		"Different number of histogram data points")

	for i := 0; i < expected.DataPoints().Len(); i++ {
		expectedDP := expected.DataPoints().At(i)
		actualDP := actual.DataPoints().At(i)

		AssertAttributesEqual(t, expectedDP.Attributes(), actualDP.Attributes())
		assert.Equal(t, expectedDP.Count(), actualDP.Count())
		assert.Equal(t, expectedDP.Sum(), actualDP.Sum())
		assert.Equal(t, expectedDP.BucketCounts().AsRaw(), actualDP.BucketCounts().AsRaw())
		assert.Equal(t, expectedDP.ExplicitBounds().AsRaw(), actualDP.ExplicitBounds().AsRaw())
	}
}

// AssertSummaryEqual compares two summaries for equality
func AssertSummaryEqual(t *testing.T, expected, actual pmetric.Summary) {
	require.Equal(t, expected.DataPoints().Len(), actual.DataPoints().Len(),
		"Different number of summary data points")

	for i := 0; i < expected.DataPoints().Len(); i++ {
		expectedDP := expected.DataPoints().At(i)
		actualDP := actual.DataPoints().At(i)

		AssertAttributesEqual(t, expectedDP.Attributes(), actualDP.Attributes())
		assert.Equal(t, expectedDP.Count(), actualDP.Count())
		assert.Equal(t, expectedDP.Sum(), actualDP.Sum())

		require.Equal(t, expectedDP.QuantileValues().Len(), actualDP.QuantileValues().Len())
		for j := 0; j < expectedDP.QuantileValues().Len(); j++ {
			expectedQV := expectedDP.QuantileValues().At(j)
			actualQV := actualDP.QuantileValues().At(j)
			assert.Equal(t, expectedQV.Quantile(), actualQV.Quantile())
			assert.Equal(t, expectedQV.Value(), actualQV.Value())
		}
	}
}

// AssertAttributesEqual compares two attribute maps for equality
func AssertAttributesEqual(t *testing.T, expected, actual pcommon.Map) {
	assert.Equal(t, expected.Len(), actual.Len(), "Different number of attributes")

	expected.Range(func(k string, v pcommon.Value) bool {
		actualValue, exists := actual.Get(k)
		assert.True(t, exists, "Missing attribute: %s", k)
		assert.Equal(t, v.AsString(), actualValue.AsString(), "Attribute %s values don't match", k)
		return true
	})
}

// AssertMetricsEnriched verifies that metrics contain expected enrichment attributes
func AssertMetricsEnriched(t *testing.T, metrics pmetric.Metrics, requiredAttributes ...string) {
	resourceMetrics := metrics.ResourceMetrics()
	require.Greater(t, resourceMetrics.Len(), 0, "No resource metrics found")

	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		attrs := rm.Resource().Attributes()

		for _, attr := range requiredAttributes {
			_, exists := attrs.Get(attr)
			assert.True(t, exists, "Missing required attribute: %s", attr)
		}
	}
}

// AssertSecretsRedacted verifies that sensitive data is redacted in logs
func AssertSecretsRedacted(t *testing.T, logs plog.Logs, secretPatterns []string) {
	logRecords := getAllLogRecords(logs)

	for _, record := range logRecords {
		body := record.Body().AsString()
		
		// Check each attribute
		record.Attributes().Range(func(k string, v pcommon.Value) bool {
			checkForSecrets(t, v.AsString(), secretPatterns, fmt.Sprintf("attribute %s", k))
			return true
		})

		// Check body
		checkForSecrets(t, body, secretPatterns, "log body")
	}
}

// AssertCardinalityWithinLimits verifies that metric cardinality is within limits
func AssertCardinalityWithinLimits(t *testing.T, metrics pmetric.Metrics, maxCardinality int) {
	uniqueCombinations := make(map[string]struct{})

	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				combinations := getMetricLabelCombinations(metric)
				for _, combo := range combinations {
					uniqueCombinations[combo] = struct{}{}
				}
			}
		}
	}

	actualCardinality := len(uniqueCombinations)
	assert.LessOrEqual(t, actualCardinality, maxCardinality,
		"Cardinality %d exceeds limit %d", actualCardinality, maxCardinality)
}

// AssertMetricTransformed verifies that metrics have been transformed correctly
func AssertMetricTransformed(t *testing.T, original, transformed pmetric.Metrics, transformations map[string]MetricTransformation) {
	transformedMap := metricsToMap(transformed)

	resourceMetrics := original.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				
				if transform, exists := transformations[metric.Name()]; exists {
					// Check if metric was renamed
					expectedName := metric.Name()
					if transform.NewName != "" {
						expectedName = transform.NewName
					}

					transformedMetric, exists := transformedMap[expectedName]
					assert.True(t, exists, "Transformed metric %s not found", expectedName)

					if exists {
						// Verify transformation
						verifyTransformation(t, metric, transformedMetric, transform)
					}
				}
			}
		}
	}
}

// AssertTracesProcessed verifies that traces have been processed correctly
func AssertTracesProcessed(t *testing.T, traces ptrace.Traces, minSpans int) {
	spanCount := 0
	resourceSpans := traces.ResourceSpans()
	
	for i := 0; i < resourceSpans.Len(); i++ {
		rs := resourceSpans.At(i)
		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)
			spanCount += ss.Spans().Len()
		}
	}

	assert.GreaterOrEqual(t, spanCount, minSpans,
		"Expected at least %d spans, got %d", minSpans, spanCount)
}

// AssertLogsProcessed verifies that logs have been processed correctly
func AssertLogsProcessed(t *testing.T, logs plog.Logs, minRecords int) {
	recordCount := 0
	resourceLogs := logs.ResourceLogs()
	
	for i := 0; i < resourceLogs.Len(); i++ {
		rl := resourceLogs.At(i)
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			recordCount += sl.LogRecords().Len()
		}
	}

	assert.GreaterOrEqual(t, recordCount, minRecords,
		"Expected at least %d log records, got %d", minRecords, recordCount)
}

// AssertEventuallyTrue waits for a condition to become true
func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		<-ticker.C
	}

	t.Fatalf("Timeout waiting for condition: %s", message)
}

// AssertPrometheusMetricsExist verifies that expected metrics exist in Prometheus format
func AssertPrometheusMetricsExist(t *testing.T, prometheusData string, expectedMetrics []string) {
	for _, metric := range expectedMetrics {
		assert.Contains(t, prometheusData, metric,
			"Expected metric %s not found in Prometheus data", metric)
	}
}

// Helper types and functions

// MetricTransformation describes how a metric should be transformed
type MetricTransformation struct {
	NewName      string
	ScaleFactor  float64
	AddLabels    map[string]string
	RemoveLabels []string
}

func checkForSecrets(t *testing.T, content string, patterns []string, location string) {
	for _, pattern := range patterns {
		// Check for exact match
		assert.NotContains(t, strings.ToLower(content), strings.ToLower(pattern),
			"Found unredacted secret pattern '%s' in %s", pattern, location)

		// Check for common secret patterns
		secretRegexes := []string{
			fmt.Sprintf(`%s\s*[:=]\s*\S+`, pattern),
			fmt.Sprintf(`"%s"\s*:\s*"[^"]+`, pattern),
			fmt.Sprintf(`'%s'\s*:\s*'[^']+`, pattern),
		}

		for _, regex := range secretRegexes {
			re := regexp.MustCompile(regex)
			matches := re.FindAllString(content, -1)
			for _, match := range matches {
				// Check if the value part is redacted
				assert.Contains(t, match, "***", "Unredacted secret found: %s in %s", match, location)
			}
		}
	}
}

func getAllLogRecords(logs plog.Logs) []plog.LogRecord {
	var records []plog.LogRecord
	
	resourceLogs := logs.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		rl := resourceLogs.At(i)
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			for k := 0; k < sl.LogRecords().Len(); k++ {
				records = append(records, sl.LogRecords().At(k))
			}
		}
	}
	
	return records
}

func getMetricLabelCombinations(metric pmetric.Metric) []string {
	var combinations []string
	
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		gauge := metric.Gauge()
		for i := 0; i < gauge.DataPoints().Len(); i++ {
			dp := gauge.DataPoints().At(i)
			combinations = append(combinations, attributesToString(dp.Attributes()))
		}
	case pmetric.MetricTypeSum:
		sum := metric.Sum()
		for i := 0; i < sum.DataPoints().Len(); i++ {
			dp := sum.DataPoints().At(i)
			combinations = append(combinations, attributesToString(dp.Attributes()))
		}
	case pmetric.MetricTypeHistogram:
		histogram := metric.Histogram()
		for i := 0; i < histogram.DataPoints().Len(); i++ {
			dp := histogram.DataPoints().At(i)
			combinations = append(combinations, attributesToString(dp.Attributes()))
		}
	}
	
	return combinations
}

func attributesToString(attrs pcommon.Map) string {
	var parts []string
	attrs.Range(func(k string, v pcommon.Value) bool {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v.AsString()))
		return true
	})
	return strings.Join(parts, ",")
}

func metricsToMap(metrics pmetric.Metrics) map[string]pmetric.Metric {
	result := make(map[string]pmetric.Metric)
	
	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				result[metric.Name()] = metric
			}
		}
	}
	
	return result
}

func verifyTransformation(t *testing.T, original, transformed pmetric.Metric, transform MetricTransformation) {
	// Verify scale factor
	if transform.ScaleFactor != 0 && transform.ScaleFactor != 1 {
		// Compare values after scaling
		originalValue := getMetricValue(original)
		transformedValue := getMetricValue(transformed)
		expectedValue := originalValue * transform.ScaleFactor
		assert.InDelta(t, expectedValue, transformedValue, 0.001,
			"Metric value not scaled correctly")
	}

	// Verify added labels
	if len(transform.AddLabels) > 0 {
		attrs := getMetricAttributes(transformed)
		for k, v := range transform.AddLabels {
			actualValue, exists := attrs.Get(k)
			assert.True(t, exists, "Expected label %s not found", k)
			if exists {
				assert.Equal(t, v, actualValue.AsString(),
					"Label %s has unexpected value", k)
			}
		}
	}

	// Verify removed labels
	if len(transform.RemoveLabels) > 0 {
		attrs := getMetricAttributes(transformed)
		for _, label := range transform.RemoveLabels {
			_, exists := attrs.Get(label)
			assert.False(t, exists, "Label %s should have been removed", label)
		}
	}
}

func getMetricValue(metric pmetric.Metric) float64 {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		if metric.Gauge().DataPoints().Len() > 0 {
			return metric.Gauge().DataPoints().At(0).DoubleValue()
		}
	case pmetric.MetricTypeSum:
		if metric.Sum().DataPoints().Len() > 0 {
			return metric.Sum().DataPoints().At(0).DoubleValue()
		}
	}
	return 0
}

func getMetricAttributes(metric pmetric.Metric) pcommon.Map {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		if metric.Gauge().DataPoints().Len() > 0 {
			return metric.Gauge().DataPoints().At(0).Attributes()
		}
	case pmetric.MetricTypeSum:
		if metric.Sum().DataPoints().Len() > 0 {
			return metric.Sum().DataPoints().At(0).Attributes()
		}
	case pmetric.MetricTypeHistogram:
		if metric.Histogram().DataPoints().Len() > 0 {
			return metric.Histogram().DataPoints().At(0).Attributes()
		}
	}
	return pcommon.NewMap()
}