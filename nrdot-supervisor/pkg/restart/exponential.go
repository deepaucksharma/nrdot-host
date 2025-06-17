package restart

import (
	"sync"
	"time"
)

// ExponentialBackoff implements exponential backoff restart strategy
type ExponentialBackoff struct {
	mu                sync.Mutex
	initialDelay      time.Duration
	maxDelay          time.Duration
	backoffMultiplier float64
	maxRetries        int
	currentDelay      time.Duration
	retryCount        int
	consecutiveSuccess int
}

// NewExponentialBackoff creates a new exponential backoff strategy
func NewExponentialBackoff(initialDelay, maxDelay time.Duration, backoffMultiplier float64, maxRetries int) *ExponentialBackoff {
	return &ExponentialBackoff{
		initialDelay:      initialDelay,
		maxDelay:          maxDelay,
		backoffMultiplier: backoffMultiplier,
		maxRetries:        maxRetries,
		currentDelay:      initialDelay,
	}
}

// NextDelay returns the next delay and whether to restart
func (e *ExponentialBackoff) NextDelay() (time.Duration, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.maxRetries > 0 && e.retryCount >= e.maxRetries {
		return 0, false
	}

	delay := e.currentDelay
	e.retryCount++

	// Calculate next delay
	nextDelay := time.Duration(float64(e.currentDelay) * e.backoffMultiplier)
	if nextDelay > e.maxDelay {
		nextDelay = e.maxDelay
	}
	e.currentDelay = nextDelay

	return delay, true
}

// Reset resets the backoff state
func (e *ExponentialBackoff) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.currentDelay = e.initialDelay
	e.retryCount = 0
	e.consecutiveSuccess = 0
}

// RecordFailure records a failure
func (e *ExponentialBackoff) RecordFailure() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.consecutiveSuccess = 0
}

// RecordSuccess records a successful restart
func (e *ExponentialBackoff) RecordSuccess() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.consecutiveSuccess++
	// Reset backoff after 3 consecutive successful runs
	if e.consecutiveSuccess >= 3 {
		e.currentDelay = e.initialDelay
		e.retryCount = 0
	}
}