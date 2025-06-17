package nrenrich

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	// "github.com/newrelic/nrdot-host/nrdot-privileged-helper/pkg/client"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// Enricher is the main enrichment engine
type Enricher struct {
	config    *Config
	logger    *zap.Logger
	providers []MetadataProvider
	// helperClient *client.PrivilegedHelperClient
	mu        sync.RWMutex
	cache     map[string]cachedMetadata
}

type cachedMetadata struct {
	data      map[string]interface{}
	timestamp time.Time
}

// NewEnricher creates a new enricher instance
func NewEnricher(config *Config, logger *zap.Logger) (*Enricher, error) {
	e := &Enricher{
		config: config,
		logger: logger,
		cache:  make(map[string]cachedMetadata),
	}

	// Initialize metadata providers based on configuration
	if config.Environment.Enabled {
		if config.Environment.System {
			e.providers = append(e.providers, NewSystemMetadataProvider(logger))
		}

		if config.Environment.CloudProvider {
			// Try to detect cloud provider
			if provider := NewAWSMetadataProvider(logger); provider != nil {
				e.providers = append(e.providers, provider)
			} else if provider := NewGCPMetadataProvider(logger); provider != nil {
				e.providers = append(e.providers, provider)
			} else if provider := NewAzureMetadataProvider(logger); provider != nil {
				e.providers = append(e.providers, provider)
			}
		}

		if config.Environment.Kubernetes {
			if provider := NewKubernetesMetadataProvider(logger); provider != nil {
				e.providers = append(e.providers, provider)
			}
		}
	}

	// Initialize privileged helper client if process enrichment is enabled
	// TODO: Enable when privileged helper is available
	// if config.Process.Enabled {
	// 	helperClient, err := client.NewPrivilegedHelperClient(config.Process.HelperEndpoint)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create privileged helper client: %w", err)
	// 	}
	// 	e.helperClient = helperClient
	// }

	// Sort rules by priority
	sort.Slice(config.Rules, func(i, j int) bool {
		return config.Rules[i].Priority > config.Rules[j].Priority
	})

	return e, nil
}

// EnrichTraces enriches trace spans with metadata
func (e *Enricher) EnrichTraces(ctx context.Context, td ptrace.Traces) error {
	// Collect metadata once for all spans
	metadata, err := e.collectMetadata(ctx)
	if err != nil {
		e.logger.Warn("Failed to collect metadata", zap.Error(err))
	}

	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		
		// Enrich resource attributes
		e.enrichResource(rs.Resource(), metadata)

		// Enrich spans
		sss := rs.ScopeSpans()
		for j := 0; j < sss.Len(); j++ {
			ss := sss.At(j)
			spans := ss.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				e.enrichAttributes(span.Attributes(), metadata)
				
				// Apply conditional rules
				e.applyRules(span.Attributes(), rs.Resource().Attributes())
				
				// Apply dynamic attributes
				e.applyDynamicAttributes(span.Attributes())
			}
		}
	}

	return nil
}

// EnrichMetrics enriches metrics with metadata
func (e *Enricher) EnrichMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Collect metadata once for all metrics
	metadata, err := e.collectMetadata(ctx)
	if err != nil {
		e.logger.Warn("Failed to collect metadata", zap.Error(err))
	}

	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		
		// Enrich resource attributes
		e.enrichResource(rm.Resource(), metadata)

		// Enrich metric data points
		sms := rm.ScopeMetrics()
		for j := 0; j < sms.Len(); j++ {
			sm := sms.At(j)
			metrics := sm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				e.enrichMetricDataPoints(metric, metadata, rm.Resource().Attributes())
			}
		}
	}

	return nil
}

// EnrichLogs enriches logs with metadata
func (e *Enricher) EnrichLogs(ctx context.Context, ld plog.Logs) error {
	// Collect metadata once for all logs
	metadata, err := e.collectMetadata(ctx)
	if err != nil {
		e.logger.Warn("Failed to collect metadata", zap.Error(err))
	}

	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		
		// Enrich resource attributes
		e.enrichResource(rl.Resource(), metadata)

		// Enrich log records
		sls := rl.ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			sl := sls.At(j)
			logs := sl.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)
				e.enrichAttributes(log.Attributes(), metadata)
				
				// Apply conditional rules
				e.applyRules(log.Attributes(), rl.Resource().Attributes())
				
				// Apply dynamic attributes
				e.applyDynamicAttributes(log.Attributes())
			}
		}
	}

	return nil
}

// collectMetadata collects metadata from all providers
func (e *Enricher) collectMetadata(ctx context.Context) (map[string]interface{}, error) {
	e.mu.RLock()
	if cached, ok := e.cache["metadata"]; ok && time.Since(cached.timestamp) < e.config.Cache.TTL {
		e.mu.RUnlock()
		return cached.data, nil
	}
	e.mu.RUnlock()

	metadata := make(map[string]interface{})

	// Add static attributes
	for k, v := range e.config.StaticAttributes {
		metadata[k] = v
	}

	// Collect from metadata providers
	for _, provider := range e.providers {
		providerMetadata, err := provider.GetMetadata(ctx)
		if err != nil {
			e.logger.Warn("Failed to get metadata from provider",
				zap.String("provider", provider.Name()),
				zap.Error(err))
			continue
		}
		for k, v := range providerMetadata {
			metadata[k] = v
		}
	}

	// Cache the metadata
	e.mu.Lock()
	e.cache["metadata"] = cachedMetadata{
		data:      metadata,
		timestamp: time.Now(),
	}
	e.mu.Unlock()

	return metadata, nil
}

// enrichResource enriches resource attributes
func (e *Enricher) enrichResource(resource pcommon.Resource, metadata map[string]interface{}) {
	attrs := resource.Attributes()
	for k, v := range metadata {
		if _, exists := attrs.Get(k); !exists {
			setAttributeValue(attrs, k, v)
		}
	}
}

// enrichAttributes enriches attributes map
func (e *Enricher) enrichAttributes(attrs pcommon.Map, metadata map[string]interface{}) {
	for k, v := range metadata {
		if _, exists := attrs.Get(k); !exists {
			setAttributeValue(attrs, k, v)
		}
	}
}

// enrichMetricDataPoints enriches all data points in a metric
func (e *Enricher) enrichMetricDataPoints(metric pmetric.Metric, metadata map[string]interface{}, resourceAttrs pcommon.Map) {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			e.enrichAttributes(dp.Attributes(), metadata)
			e.applyRules(dp.Attributes(), resourceAttrs)
			e.applyDynamicAttributes(dp.Attributes())
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			e.enrichAttributes(dp.Attributes(), metadata)
			e.applyRules(dp.Attributes(), resourceAttrs)
			e.applyDynamicAttributes(dp.Attributes())
		}
	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			e.enrichAttributes(dp.Attributes(), metadata)
			e.applyRules(dp.Attributes(), resourceAttrs)
			e.applyDynamicAttributes(dp.Attributes())
		}
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			e.enrichAttributes(dp.Attributes(), metadata)
			e.applyRules(dp.Attributes(), resourceAttrs)
			e.applyDynamicAttributes(dp.Attributes())
		}
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			e.enrichAttributes(dp.Attributes(), metadata)
			e.applyRules(dp.Attributes(), resourceAttrs)
			e.applyDynamicAttributes(dp.Attributes())
		}
	}
}

// applyRules applies conditional enrichment rules
func (e *Enricher) applyRules(attrs pcommon.Map, resourceAttrs pcommon.Map) {
	// TODO: Implement CEL expression evaluation for rules
	// For now, we'll skip rule evaluation
}

// applyDynamicAttributes applies dynamic attribute computation
func (e *Enricher) applyDynamicAttributes(attrs pcommon.Map) {
	// TODO: Implement dynamic attribute computation
	// For now, we'll skip dynamic attributes
}

// setAttributeValue sets an attribute value based on its type
func setAttributeValue(attrs pcommon.Map, key string, value interface{}) {
	switch v := value.(type) {
	case string:
		attrs.PutStr(key, v)
	case int:
		attrs.PutInt(key, int64(v))
	case int64:
		attrs.PutInt(key, v)
	case float64:
		attrs.PutDouble(key, v)
	case bool:
		attrs.PutBool(key, v)
	case []string:
		slice := attrs.PutEmptySlice(key)
		for _, s := range v {
			slice.AppendEmpty().SetStr(s)
		}
	default:
		// Convert to string as fallback
		attrs.PutStr(key, fmt.Sprintf("%v", v))
	}
}

// Close shuts down the enricher
func (e *Enricher) Close() error {
	// TODO: Close helper client when available
	// if e.helperClient != nil {
	// 	return e.helperClient.Close()
	// }
	return nil
}