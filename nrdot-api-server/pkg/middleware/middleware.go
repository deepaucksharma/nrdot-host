package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log request
			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Int("status", wrapped.statusCode),
				zap.Duration("duration", time.Since(start)),
				zap.String("user_agent", r.UserAgent()),
			)
		})
	}
}

// LocalhostOnlyMiddleware restricts access to localhost only
func LocalhostOnlyMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP from remote address
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				logger.Warn("Failed to parse remote address",
					zap.String("remote_addr", r.RemoteAddr),
					zap.Error(err),
				)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Check if IP is localhost
			if !isLocalhost(ip) {
				logger.Warn("Rejected non-localhost connection",
					zap.String("remote_addr", r.RemoteAddr),
				)
				http.Error(w, "Forbidden - localhost only", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered",
						zap.Any("error", err),
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
					)

					// Return 500 error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware adds CORS headers for localhost origins
func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Allow localhost origins
			if isLocalhostOrigin(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "3600")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ContentTypeMiddleware sets the Content-Type header
func ContentTypeMiddleware(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// isLocalhost checks if an IP address is localhost
func isLocalhost(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check for IPv4 localhost
	if parsedIP.IsLoopback() {
		return true
	}

	// Check for IPv4 localhost range
	if parsedIP.To4() != nil {
		return parsedIP.Equal(net.IPv4(127, 0, 0, 1))
	}

	// Check for IPv6 localhost
	return parsedIP.Equal(net.IPv6loopback)
}

// isLocalhostOrigin checks if an origin is from localhost
func isLocalhostOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	// Parse origin URL
	if !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
		return false
	}

	// Extract host
	parts := strings.Split(origin, "://")
	if len(parts) != 2 {
		return false
	}

	host := strings.Split(parts[1], "/")[0]
	hostParts := strings.Split(host, ":")
	hostname := hostParts[0]

	// Check if hostname is localhost
	return hostname == "localhost" || 
		hostname == "127.0.0.1" || 
		hostname == "[::1]" || 
		hostname == "::1"
}

// RequestIDMiddleware adds a request ID to each request
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), r.Context().Value("request_counter"))
			
			// Add to response header
			w.Header().Set("X-Request-ID", requestID)
			
			// Pass to next handler
			next.ServeHTTP(w, r)
		})
	}
}