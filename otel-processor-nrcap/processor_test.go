package nrcap

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.uber.org/zap"
)

func TestNewCapProcessor(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	logger := zap.NewNop()
	consumer := consumertest.NewNop()

	proc, err := newCapProcessor(cfg, logger, consumer)
	require.NoError(t, err)
	assert.NotNil(t, proc)
	assert.Equal(t, cfg, proc.config)
	assert.Equal(t, logger, proc.logger)
	assert.Equal(t, consumer, proc.nextConsumer)
}

func TestCapProcessorStart(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.ResetInterval = 100 * time.Millisecond
	cfg.EnableStats = true
	
	logger := zap.NewNop()
	consumer := consumertest.NewNop()

	proc, err := newCapProcessor(cfg, logger, consumer)
	require.NoError(t, err)

	err = proc.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// Let the tickers run
	time.Sleep(150 * time.Millisecond)

	err = proc.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestCapProcessorConsumeMetrics(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		inputMetrics   func() pmetric.Metrics
		expectedCount  int
		expectedDrops  int64
	}{
		{
			name: "under_limit",
			config: &Config{
				GlobalLimit:   100,
				DefaultLimit:  10,
				Strategy:      StrategyDrop,
				ResetInterval: time.Hour,
				WindowSize:    5 * time.Minute,
			},
			inputMetrics: func() pmetric.Metrics {
				return generateMetrics("test_metric", 5)
			},
			expectedCount: 5,
			expectedDrops: 0,
		},
		{
			name: "over_limit_drop",
			config: &Config{
				GlobalLimit:   100,
				DefaultLimit:  3,
				Strategy:      StrategyDrop,
				ResetInterval: time.Hour,
				WindowSize:    5 * time.Minute,
			},
			inputMetrics: func() pmetric.Metrics {
				return generateMetrics("test_metric", 5)
			},
			expectedCount: 3,
			expectedDrops: 2,
		},
		{
			name: "aggregate_strategy",
			config: &Config{
				GlobalLimit:       100,
				DefaultLimit:      3,
				Strategy:          StrategyAggregate,
				ResetInterval:     time.Hour,
				WindowSize:        5 * time.Minute,
				AggregationLabels: []string{"service"},
			},
			inputMetrics: func() pmetric.Metrics {
				return generateMetrics("test_metric", 5)
			},
			expectedCount: 5,
			expectedDrops: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := consumertest.NewNop()
			proc, err := newCapProcessor(tt.config, zap.NewNop(), consumer)
			require.NoError(t, err)

			err = proc.Start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)
			defer proc.Shutdown(context.Background())

			metrics := tt.inputMetrics()
			err = proc.ConsumeMetrics(context.Background(), metrics)
			require.NoError(t, err)

			// Check stats
			stats := proc.limiter.GetStats()
			assert.Equal(t, tt.expectedDrops, stats.DroppedMetrics)
		})
	}
}

func TestCapProcessorWithMetricLimits(t *testing.T) {
	cfg := &Config{
		GlobalLimit:  100,
		DefaultLimit: 10,
		MetricLimits: map[string]int{
			"limited_metric": 2,
		},
		Strategy:      StrategyDrop,
		ResetInterval: time.Hour,
		WindowSize:    5 * time.Minute,
	}

	consumer := consumertest.NewNop()
	proc, err := newCapProcessor(cfg, zap.NewNop(), consumer)
	require.NoError(t, err)

	err = proc.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	defer proc.Shutdown(context.Background())

	// Send metrics with different cardinalities
	metrics1 := generateMetrics("limited_metric", 3)
	err = proc.ConsumeMetrics(context.Background(), metrics1)
	require.NoError(t, err)

	metrics2 := generateMetrics("unlimited_metric", 5)
	err = proc.ConsumeMetrics(context.Background(), metrics2)
	require.NoError(t, err)

	stats := proc.limiter.GetStats()
	assert.Equal(t, int64(1), stats.DroppedMetrics) // One from limited_metric
}

func TestCapProcessorDenyLabels(t *testing.T) {
	cfg := &Config{
		GlobalLimit:  100,
		DefaultLimit: 10,
		DenyLabels:   []string{"trace_id", "session_id"},
		Strategy:     StrategyAggregate,
		ResetInterval: time.Hour,
		WindowSize:    5 * time.Minute,
	}

	consumer := consumertest.NewNop()
	proc, err := newCapProcessor(cfg, zap.NewNop(), consumer)
	require.NoError(t, err)

	err = proc.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	defer proc.Shutdown(context.Background())

	// Create metrics with deny labels
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("test_metric")
	metric.SetEmptyGauge()
	
	dp := metric.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetDoubleValue(1.0)
	dp.Attributes().PutStr("service", "test")
	dp.Attributes().PutStr("trace_id", "12345")
	dp.Attributes().PutStr("session_id", "abcde")

	err = proc.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)
}

// Helper function to generate metrics with specific cardinality
func generateMetrics(name string, cardinality int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetEmptyGauge()

	for i := 0; i < cardinality; i++ {
		dp := metric.Gauge().DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		dp.SetDoubleValue(float64(i))
		dp.Attributes().PutStr("label", string(rune('a'+i)))
		dp.Attributes().PutStr("service", "test")
	}

	return metrics
}

func TestFactoryCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg)
	
	capCfg, ok := cfg.(*Config)
	require.True(t, ok)
	assert.Equal(t, 100000, capCfg.GlobalLimit)
	assert.Equal(t, 1000, capCfg.DefaultLimit)
	assert.Equal(t, StrategyDrop, capCfg.Strategy)
}

func TestFactoryCreateMetricsProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	
	set := processortest.NewNopCreateSettings()
	consumer := consumertest.NewNop()
	
	processor, err := factory.CreateMetricsProcessor(context.Background(), set, cfg, consumer)
	require.NoError(t, err)
	assert.NotNil(t, processor)
}

func TestFactoryCreateMetricsProcessorInvalidConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GlobalLimit = -1 // Invalid
	
	set := processortest.NewNopCreateSettings()
	consumer := consumertest.NewNop()
	
	_, err := factory.CreateMetricsProcessor(context.Background(), set, cfg, consumer)
	require.Error(t, err)
}