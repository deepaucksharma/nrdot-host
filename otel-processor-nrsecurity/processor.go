package nrsecurity

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// nrSecurityProcessor implements the OpenTelemetry processor interface
type nrSecurityProcessor struct {
	config   *Config
	logger   *zap.Logger
	redactor *Redactor
}

// newProcessor creates a new instance of the processor
func newProcessor(cfg component.Config, logger *zap.Logger) (*nrSecurityProcessor, error) {
	pCfg := cfg.(*Config)

	redactor, err := NewRedactor(pCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create redactor: %w", err)
	}

	return &nrSecurityProcessor{
		config:   pCfg,
		logger:   logger,
		redactor: redactor,
	}, nil
}

// Capabilities returns the processing capabilities
func (p *nrSecurityProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// Start initializes the processor
func (p *nrSecurityProcessor) Start(_ context.Context, _ component.Host) error {
	p.logger.Info("Starting NR Security processor")
	return nil
}

// Shutdown stops the processor
func (p *nrSecurityProcessor) Shutdown(_ context.Context) error {
	p.logger.Info("Shutting down NR Security processor")
	return nil
}

// processTraces processes trace data
func (p *nrSecurityProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if !p.config.Enabled {
		return td, nil
	}

	// Process each resource span
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		
		// Redact resource attributes
		p.redactor.RedactAttributes(rs.Resource().Attributes())

		// Process scope spans
		ilss := rs.ScopeSpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			
			// Redact scope attributes
			p.redactor.RedactAttributes(ils.Scope().Attributes())

			// Process spans
			spans := ils.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				
				// Redact span attributes
				p.redactor.RedactAttributes(span.Attributes())

				// Process events
				events := span.Events()
				for l := 0; l < events.Len(); l++ {
					event := events.At(l)
					p.redactor.RedactAttributes(event.Attributes())
				}

				// Process links
				links := span.Links()
				for l := 0; l < links.Len(); l++ {
					link := links.At(l)
					p.redactor.RedactAttributes(link.Attributes())
				}
			}
		}
	}

	return td, nil
}

// processMetrics processes metric data
func (p *nrSecurityProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if !p.config.Enabled {
		return md, nil
	}

	// Process each resource metric
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		
		// Redact resource attributes
		p.redactor.RedactAttributes(rm.Resource().Attributes())

		// Process scope metrics
		ilms := rm.ScopeMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			
			// Redact scope attributes
			p.redactor.RedactAttributes(ilm.Scope().Attributes())

			// Process metrics
			metrics := ilm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				
				// Process data points based on metric type
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					p.processGaugeDataPoints(metric.Gauge().DataPoints())
				case pmetric.MetricTypeSum:
					p.processNumberDataPoints(metric.Sum().DataPoints())
				case pmetric.MetricTypeHistogram:
					p.processHistogramDataPoints(metric.Histogram().DataPoints())
				case pmetric.MetricTypeExponentialHistogram:
					p.processExponentialHistogramDataPoints(metric.ExponentialHistogram().DataPoints())
				case pmetric.MetricTypeSummary:
					p.processSummaryDataPoints(metric.Summary().DataPoints())
				}
			}
		}
	}

	return md, nil
}

// processLogs processes log data
func (p *nrSecurityProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	if !p.config.Enabled {
		return ld, nil
	}

	// Process each resource log
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		
		// Redact resource attributes
		p.redactor.RedactAttributes(rl.Resource().Attributes())

		// Process scope logs
		ills := rl.ScopeLogs()
		for j := 0; j < ills.Len(); j++ {
			ill := ills.At(j)
			
			// Redact scope attributes
			p.redactor.RedactAttributes(ill.Scope().Attributes())

			// Process log records
			logs := ill.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)
				
				// Redact log attributes
				p.redactor.RedactAttributes(log.Attributes())

				// Redact log body if it's a string
				if log.Body().Type() == pcommon.ValueTypeStr {
					redacted := p.redactor.RedactString(log.Body().Str())
					log.Body().SetStr(redacted)
				}
			}
		}
	}

	return ld, nil
}

// processGaugeDataPoints redacts attributes in gauge data points
func (p *nrSecurityProcessor) processGaugeDataPoints(dps pmetric.NumberDataPointSlice) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		p.redactor.RedactAttributes(dp.Attributes())
	}
}

// processNumberDataPoints redacts attributes in number data points
func (p *nrSecurityProcessor) processNumberDataPoints(dps pmetric.NumberDataPointSlice) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		p.redactor.RedactAttributes(dp.Attributes())
	}
}

// processHistogramDataPoints redacts attributes in histogram data points
func (p *nrSecurityProcessor) processHistogramDataPoints(dps pmetric.HistogramDataPointSlice) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		p.redactor.RedactAttributes(dp.Attributes())
	}
}

// processExponentialHistogramDataPoints redacts attributes in exponential histogram data points
func (p *nrSecurityProcessor) processExponentialHistogramDataPoints(dps pmetric.ExponentialHistogramDataPointSlice) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		p.redactor.RedactAttributes(dp.Attributes())
	}
}

// processSummaryDataPoints redacts attributes in summary data points
func (p *nrSecurityProcessor) processSummaryDataPoints(dps pmetric.SummaryDataPointSlice) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		p.redactor.RedactAttributes(dp.Attributes())
	}
}