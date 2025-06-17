package restart

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNeverStrategy(t *testing.T) {
	strategy := &neverStrategy{}

	// Should always return false
	delay, shouldRestart := strategy.NextDelay()
	if shouldRestart {
		t.Error("Never strategy should not restart")
	}
	if delay != 0 {
		t.Error("Never strategy should return 0 delay")
	}

	// Test other methods don't panic
	strategy.Reset()
	strategy.RecordFailure()
	strategy.RecordSuccess()
}

func TestAlwaysStrategy(t *testing.T) {
	expectedDelay := 5 * time.Second
	strategy := &alwaysStrategy{delay: expectedDelay}

	// Should always return true with fixed delay
	for i := 0; i < 5; i++ {
		delay, shouldRestart := strategy.NextDelay()
		if !shouldRestart {
			t.Error("Always strategy should always restart")
		}
		if delay != expectedDelay {
			t.Errorf("Expected delay %v, got %v", expectedDelay, delay)
		}
	}

	// Test other methods don't panic
	strategy.Reset()
	strategy.RecordFailure()
	strategy.RecordSuccess()
}

func TestExponentialBackoff(t *testing.T) {
	initialDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second
	backoffMultiplier := 2.0
	maxRetries := 5

	strategy := NewExponentialBackoff(initialDelay, maxDelay, backoffMultiplier, maxRetries)

	// Test exponential progression
	expectedDelays := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1 * time.Second, // capped at max
	}

	for i, expected := range expectedDelays {
		delay, shouldRestart := strategy.NextDelay()
		if !shouldRestart {
			t.Errorf("Iteration %d: should restart", i)
		}
		if delay != expected {
			t.Errorf("Iteration %d: expected delay %v, got %v", i, expected, delay)
		}
		strategy.RecordFailure()
	}

	// Should stop after max retries
	delay, shouldRestart := strategy.NextDelay()
	if shouldRestart {
		t.Error("Should not restart after max retries")
	}
	if delay != 0 {
		t.Error("Should return 0 delay when not restarting")
	}
}

func TestExponentialBackoff_Reset(t *testing.T) {
	strategy := NewExponentialBackoff(100*time.Millisecond, 1*time.Second, 2.0, 5)

	// Use up some retries
	for i := 0; i < 3; i++ {
		strategy.NextDelay()
		strategy.RecordFailure()
	}

	// Reset
	strategy.Reset()

	// Should start from beginning
	delay, shouldRestart := strategy.NextDelay()
	if !shouldRestart {
		t.Error("Should restart after reset")
	}
	if delay != 100*time.Millisecond {
		t.Errorf("Expected initial delay after reset, got %v", delay)
	}
}

func TestExponentialBackoff_ConsecutiveSuccess(t *testing.T) {
	strategy := NewExponentialBackoff(100*time.Millisecond, 1*time.Second, 2.0, 10)

	// Use up some retries
	for i := 0; i < 3; i++ {
		strategy.NextDelay()
		strategy.RecordFailure()
	}

	// Record consecutive successes
	for i := 0; i < 3; i++ {
		strategy.RecordSuccess()
	}

	// Should reset after 3 consecutive successes
	delay, _ := strategy.NextDelay()
	if delay != 100*time.Millisecond {
		t.Errorf("Expected initial delay after consecutive successes, got %v", delay)
	}
}

func TestFactory(t *testing.T) {
	tests := []struct {
		name     string
		policy   Policy
		expected string
	}{
		{"Never", PolicyNever, "*restart.neverStrategy"},
		{"Always", PolicyAlways, "*restart.alwaysStrategy"},
		{"OnFailure", PolicyOnFailure, "*restart.ExponentialBackoff"},
		{"Invalid", Policy("invalid"), "*restart.ExponentialBackoff"}, // defaults to on-failure
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Policy = tt.policy
			factory := NewFactory(config)
			strategy := factory.Create()

			typeName := fmt.Sprintf("%T", strategy)
			if typeName != tt.expected {
				t.Errorf("Expected strategy type %s, got %s", tt.expected, typeName)
			}
		})
	}
}

func TestWaitForRestart(t *testing.T) {
	ctx := context.Background()

	// Test zero delay
	err := WaitForRestart(ctx, 0)
	if err != nil {
		t.Errorf("Zero delay should return immediately: %v", err)
	}

	// Test normal delay
	start := time.Now()
	err = WaitForRestart(ctx, 100*time.Millisecond)
	elapsed := time.Since(start)
	if err != nil {
		t.Errorf("Wait failed: %v", err)
	}
	if elapsed < 90*time.Millisecond || elapsed > 150*time.Millisecond {
		t.Errorf("Expected ~100ms delay, got %v", elapsed)
	}

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start = time.Now()
	err = WaitForRestart(ctx, 1*time.Second)
	elapsed = time.Since(start)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("Should have cancelled early, elapsed: %v", elapsed)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Policy != PolicyOnFailure {
		t.Errorf("Expected default policy %v, got %v", PolicyOnFailure, config.Policy)
	}
	if config.MaxRetries != 10 {
		t.Errorf("Expected default max retries 10, got %d", config.MaxRetries)
	}
	if config.InitialDelay != time.Second {
		t.Errorf("Expected default initial delay 1s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 5*time.Minute {
		t.Errorf("Expected default max delay 5m, got %v", config.MaxDelay)
	}
	if config.BackoffMultiplier != 2.0 {
		t.Errorf("Expected default backoff multiplier 2.0, got %f", config.BackoffMultiplier)
	}
}