// Package nrcap provides cardinality protection for OpenTelemetry metrics.
//
// The processor tracks and limits the cardinality (unique label combinations) of metrics
// to prevent metric explosions that can cause performance issues and increased costs.
//
// Features:
//   - Per-metric cardinality limits
//   - Global cardinality limit enforcement
//   - Multiple limiting strategies (drop, aggregate, sample, oldest)
//   - High-cardinality label detection and filtering
//   - Time-based cardinality windows
//   - Memory-efficient tracking using xxhash
//   - Configurable reset intervals
//   - Cardinality statistics reporting
//
// Limiting Strategies:
//   - drop: Drop new metrics that exceed cardinality limit
//   - aggregate: Remove labels to reduce cardinality
//   - sample: Randomly sample metrics over the limit
//   - oldest: Drop oldest label combinations
//
// Example configuration:
//
//	processors:
//	  nrcap:
//	    global_limit: 100000
//	    metric_limits:
//	      http_requests_total: 10000
//	      db_connections: 5000
//	    default_limit: 1000
//	    strategy: drop
//	    deny_labels:
//	      - request_id
//	      - session_id
//	    reset_interval: 1h
package nrcap