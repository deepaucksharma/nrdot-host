package restart

import (
	"context"
	"time"
)

// Policy defines when the supervisor should restart the collector
type Policy string

const (
	PolicyNever     Policy = "never"
	PolicyOnFailure Policy = "on-failure"
	PolicyAlways    Policy = "always"
)

// Strategy defines the interface for restart strategies
type Strategy interface {
	// NextDelay returns the next delay before restart and whether to restart
	NextDelay() (time.Duration, bool)
	// Reset resets the strategy state
	Reset()
	// RecordFailure records a failure
	RecordFailure()
	// RecordSuccess records a successful restart
	RecordSuccess()
}

// Config holds restart strategy configuration
type Config struct {
	Policy             Policy
	MaxRetries         int
	InitialDelay       time.Duration
	MaxDelay           time.Duration
	BackoffMultiplier  float64
}

// DefaultConfig returns default restart configuration
func DefaultConfig() Config {
	return Config{
		Policy:            PolicyOnFailure,
		MaxRetries:        10,
		InitialDelay:      time.Second,
		MaxDelay:          5 * time.Minute,
		BackoffMultiplier: 2.0,
	}
}

// Factory creates a restart strategy based on configuration
type Factory struct {
	config Config
}

// NewFactory creates a new restart strategy factory
func NewFactory(config Config) *Factory {
	return &Factory{config: config}
}

// Create creates a new restart strategy based on the policy
func (f *Factory) Create() Strategy {
	switch f.config.Policy {
	case PolicyNever:
		return &neverStrategy{}
	case PolicyAlways:
		return &alwaysStrategy{delay: f.config.InitialDelay}
	case PolicyOnFailure:
		return NewExponentialBackoff(
			f.config.InitialDelay,
			f.config.MaxDelay,
			f.config.BackoffMultiplier,
			f.config.MaxRetries,
		)
	default:
		// Default to on-failure
		return NewExponentialBackoff(
			f.config.InitialDelay,
			f.config.MaxDelay,
			f.config.BackoffMultiplier,
			f.config.MaxRetries,
		)
	}
}

// neverStrategy never restarts
type neverStrategy struct{}

func (s *neverStrategy) NextDelay() (time.Duration, bool) {
	return 0, false
}

func (s *neverStrategy) Reset() {}

func (s *neverStrategy) RecordFailure() {}

func (s *neverStrategy) RecordSuccess() {}

// alwaysStrategy always restarts with a fixed delay
type alwaysStrategy struct {
	delay time.Duration
}

func (s *alwaysStrategy) NextDelay() (time.Duration, bool) {
	return s.delay, true
}

func (s *alwaysStrategy) Reset() {}

func (s *alwaysStrategy) RecordFailure() {}

func (s *alwaysStrategy) RecordSuccess() {}

// WaitForRestart waits for the specified delay with context cancellation support
func WaitForRestart(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	
	timer := time.NewTimer(delay)
	defer timer.Stop()
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}