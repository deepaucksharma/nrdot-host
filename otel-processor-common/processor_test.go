package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.uber.org/zap"
)

func TestBaseProcessor(t *testing.T) {
	logger := zap.NewNop()
	telemetry := componenttest.NewNopTelemetrySettings()
	
	tests := []struct {
		name   string
		config ProcessorConfig
	}{
		{
			name: "enabled processor",
			config: ProcessorConfig{
				Enabled:   true,
				ErrorMode: ErrorModePropagateError,
				Timeout:   time.Second,
			},
		},
		{
			name: "disabled processor",
			config: ProcessorConfig{
				Enabled:   false,
				ErrorMode: ErrorModeIgnore,
				Timeout:   0,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewBaseProcessor(logger, tt.config, telemetry)
			
			// Test Start
			err := processor.Start(context.Background(), nil)
			assert.NoError(t, err)
			
			// Test Shutdown
			err = processor.Shutdown(context.Background())
			assert.NoError(t, err)
		})
	}
}

func TestHandleError(t *testing.T) {
	logger := zap.NewNop()
	telemetry := componenttest.NewNopTelemetrySettings()
	testErr := errors.New("test error")
	
	tests := []struct {
		name      string
		errorMode ErrorMode
		err       error
		wantErr   bool
	}{
		{
			name:      "propagate error mode with error",
			errorMode: ErrorModePropagateError,
			err:       testErr,
			wantErr:   true,
		},
		{
			name:      "ignore error mode with error",
			errorMode: ErrorModeIgnore,
			err:       testErr,
			wantErr:   false,
		},
		{
			name:      "silent error mode with error",
			errorMode: ErrorModeSilent,
			err:       testErr,
			wantErr:   false,
		},
		{
			name:      "propagate error mode with nil",
			errorMode: ErrorModePropagateError,
			err:       nil,
			wantErr:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProcessorConfig{
				Enabled:   true,
				ErrorMode: tt.errorMode,
			}
			processor := NewBaseProcessor(logger, config, telemetry)
			
			err := processor.HandleError(tt.err, "test message")
			if tt.wantErr {
				assert.Equal(t, tt.err, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	logger := zap.NewNop()
	telemetry := componenttest.NewNopTelemetrySettings()
	
	tests := []struct {
		name    string
		timeout time.Duration
		fnDelay time.Duration
		wantErr bool
	}{
		{
			name:    "successful operation within timeout",
			timeout: 100 * time.Millisecond,
			fnDelay: 10 * time.Millisecond,
			wantErr: false,
		},
		{
			name:    "operation exceeds timeout",
			timeout: 10 * time.Millisecond,
			fnDelay: 100 * time.Millisecond,
			wantErr: true,
		},
		{
			name:    "no timeout configured",
			timeout: 0,
			fnDelay: 10 * time.Millisecond,
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProcessorConfig{
				Enabled: true,
				Timeout: tt.timeout,
			}
			processor := NewBaseProcessor(logger, config, telemetry)
			
			fn := func(ctx context.Context) error {
				select {
				case <-time.After(tt.fnDelay):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			
			err := processor.WithTimeout(context.Background(), fn)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "timeout")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}