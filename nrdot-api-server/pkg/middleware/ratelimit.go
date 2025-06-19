package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*tokenBucket
	rate       int           // tokens per interval
	interval   time.Duration // refill interval
	bucketSize int           // max tokens in bucket
	logger     *zap.Logger
}

// tokenBucket represents a token bucket for rate limiting
type tokenBucket struct {
	tokens    int
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, interval time.Duration, bucketSize int, logger *zap.Logger) *RateLimiter {
	if bucketSize == 0 {
		bucketSize = rate // Default bucket size equals rate
	}
	
	rl := &RateLimiter{
		buckets:    make(map[string]*tokenBucket),
		rate:       rate,
		interval:   interval,
		bucketSize: bucketSize,
		logger:     logger,
	}
	
	// Start cleanup goroutine
	go rl.cleanup()
	
	return rl
}

// RateLimitMiddleware creates HTTP middleware for rate limiting
func (rl *RateLimiter) RateLimitMiddleware(keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if key == "" {
				key = "default"
			}
			
			if !rl.Allow(key) {
				rl.logger.Warn("Rate limit exceeded",
					zap.String("key", key),
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method))
				
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.interval.Seconds())))
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	bucket, exists := rl.buckets[key]
	if !exists {
		// Create new bucket
		bucket = &tokenBucket{
			tokens:     rl.bucketSize,
			lastRefill: time.Now(),
		}
		rl.buckets[key] = bucket
	}
	
	// Refill tokens based on elapsed time
	rl.refillBucket(bucket)
	
	// Check if we have tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	
	return false
}

// refillBucket adds tokens based on elapsed time
func (rl *RateLimiter) refillBucket(bucket *tokenBucket) {
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	
	// Calculate how many intervals have passed
	intervals := int(elapsed / rl.interval)
	if intervals > 0 {
		// Add tokens for elapsed intervals
		tokensToAdd := intervals * rl.rate
		bucket.tokens = min(bucket.tokens+tokensToAdd, rl.bucketSize)
		bucket.lastRefill = bucket.lastRefill.Add(time.Duration(intervals) * rl.interval)
	}
}

// cleanup removes old buckets periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		
		// Remove buckets that haven't been used for 10 minutes
		for key, bucket := range rl.buckets {
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(rl.buckets, key)
			}
		}
		
		rl.mu.Unlock()
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Common key extraction functions

// IPKeyFunc extracts client IP for rate limiting
func IPKeyFunc(r *http.Request) string {
	// Try X-Real-IP first (common behind proxies)
	ip := r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}
	
	// Try X-Forwarded-For
	ip = r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// Take the first IP if multiple
		if idx := len(ip) - 1; idx > 0 {
			if comma := byte(','); ip[idx] == comma {
				ip = ip[:idx]
			}
		}
		return ip
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// APIKeyFunc extracts API key for rate limiting
func APIKeyFunc(r *http.Request) string {
	// Check header first
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}
	
	// Check query parameter
	return r.URL.Query().Get("api_key")
}

// UserKeyFunc extracts authenticated user for rate limiting
func UserKeyFunc(r *http.Request) string {
	// This would typically extract from JWT claims or session
	// For now, return a placeholder
	return "user"
}

// EndpointKeyFunc creates a global rate limit per endpoint
func EndpointKeyFunc(r *http.Request) string {
	return fmt.Sprintf("%s:%s", r.Method, r.URL.Path)
}