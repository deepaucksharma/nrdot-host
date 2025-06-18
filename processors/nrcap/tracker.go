package nrcap

import (
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// CardinalityTracker tracks metric cardinality
type CardinalityTracker struct {
	mu sync.RWMutex

	// Metric name -> label hash -> timestamp
	metrics map[string]map[uint64]time.Time

	// Global cardinality count
	globalCount int

	// Metric name -> cardinality count
	metricCounts map[string]int

	// Configuration
	windowSize time.Duration

	// Statistics
	stats CardinalityStats
}

// CardinalityStats holds cardinality statistics
type CardinalityStats struct {
	mu sync.RWMutex

	TotalMetrics      int64
	DroppedMetrics    int64
	AggregatedMetrics int64
	SampledMetrics    int64
	
	MetricCardinalities map[string]int
	HighCardinalityLabels map[string]int
	
	LastReset time.Time
}

// NewCardinalityTracker creates a new cardinality tracker
func NewCardinalityTracker(windowSize time.Duration) *CardinalityTracker {
	return &CardinalityTracker{
		metrics:      make(map[string]map[uint64]time.Time),
		metricCounts: make(map[string]int),
		windowSize:   windowSize,
		stats: CardinalityStats{
			MetricCardinalities:   make(map[string]int),
			HighCardinalityLabels: make(map[string]int),
			LastReset:            time.Now(),
		},
	}
}

// Track tracks a metric and all its data points, returns true if any are new
func (ct *CardinalityTracker) Track(metric pmetric.Metric) (bool, uint64) {
	metricName := metric.Name()
	
	// Get all unique label combinations for this metric
	labelHashes := ct.getAllLabelHashes(metric)
	if len(labelHashes) == 0 {
		return false, 0
	}
	
	anyNew := false
	var firstHash uint64
	
	// Track all data points
	for i, labelHash := range labelHashes {
		if i == 0 {
			firstHash = labelHash
		}
		
		ct.mu.Lock()
		// Initialize metric map if needed
		if _, exists := ct.metrics[metricName]; !exists {
			ct.metrics[metricName] = make(map[uint64]time.Time)
		}

		// Check if this label combination exists
		if _, exists := ct.metrics[metricName][labelHash]; exists {
			// Update timestamp
			ct.metrics[metricName][labelHash] = time.Now()
		} else {
			// New label combination
			ct.metrics[metricName][labelHash] = time.Now()
			ct.metricCounts[metricName]++
			ct.globalCount++
			anyNew = true
		}
		ct.mu.Unlock()
	}

	// Update stats
	ct.mu.Lock()
	ct.stats.MetricCardinalities[metricName] = ct.metricCounts[metricName]
	ct.mu.Unlock()

	return anyNew, firstHash
}

// GetCardinality returns the current cardinality for a metric
func (ct *CardinalityTracker) GetCardinality(metricName string) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	
	return ct.metricCounts[metricName]
}

// GetGlobalCardinality returns the total cardinality across all metrics
func (ct *CardinalityTracker) GetGlobalCardinality() int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	
	return ct.globalCount
}

// CleanupOldEntries removes entries older than the window size
func (ct *CardinalityTracker) CleanupOldEntries() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-ct.windowSize)

	for metricName, labelMap := range ct.metrics {
		for hash, timestamp := range labelMap {
			if timestamp.Before(cutoff) {
				delete(labelMap, hash)
				ct.metricCounts[metricName]--
				ct.globalCount--
			}
		}

		// Remove empty metric entries
		if len(labelMap) == 0 {
			delete(ct.metrics, metricName)
			delete(ct.metricCounts, metricName)
		}
	}
}

// Reset clears all tracking data
func (ct *CardinalityTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.metrics = make(map[string]map[uint64]time.Time)
	ct.metricCounts = make(map[string]int)
	ct.globalCount = 0

	ct.stats.LastReset = time.Now()
}

// GetStats returns current statistics
func (ct *CardinalityTracker) GetStats() CardinalityStats {
	ct.stats.mu.RLock()
	defer ct.stats.mu.RUnlock()

	// Create a copy of the stats
	statsCopy := CardinalityStats{
		TotalMetrics:      ct.stats.TotalMetrics,
		DroppedMetrics:    ct.stats.DroppedMetrics,
		AggregatedMetrics: ct.stats.AggregatedMetrics,
		SampledMetrics:    ct.stats.SampledMetrics,
		LastReset:         ct.stats.LastReset,
		MetricCardinalities:   make(map[string]int),
		HighCardinalityLabels: make(map[string]int),
	}

	for k, v := range ct.stats.MetricCardinalities {
		statsCopy.MetricCardinalities[k] = v
	}
	for k, v := range ct.stats.HighCardinalityLabels {
		statsCopy.HighCardinalityLabels[k] = v
	}

	return statsCopy
}

// IncrementStats increments various statistics
func (ct *CardinalityTracker) IncrementStats(statType string) {
	ct.stats.mu.Lock()
	defer ct.stats.mu.Unlock()

	switch statType {
	case "total":
		ct.stats.TotalMetrics++
	case "dropped":
		ct.stats.DroppedMetrics++
	case "aggregated":
		ct.stats.AggregatedMetrics++
	case "sampled":
		ct.stats.SampledMetrics++
	}
}

// getAllLabelHashes returns all unique label hashes for a metric
func (ct *CardinalityTracker) getAllLabelHashes(metric pmetric.Metric) []uint64 {
	var hashes []uint64
	
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			hash := ct.hashDataPointLabels(metric.Name(), dps.At(i).Attributes())
			hashes = append(hashes, hash)
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			hash := ct.hashDataPointLabels(metric.Name(), dps.At(i).Attributes())
			hashes = append(hashes, hash)
		}
	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			hash := ct.hashDataPointLabels(metric.Name(), dps.At(i).Attributes())
			hashes = append(hashes, hash)
		}
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			hash := ct.hashDataPointLabels(metric.Name(), dps.At(i).Attributes())
			hashes = append(hashes, hash)
		}
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			hash := ct.hashDataPointLabels(metric.Name(), dps.At(i).Attributes())
			hashes = append(hashes, hash)
		}
	}
	
	return hashes
}

// hashDataPointLabels creates a hash from metric name and attributes
func (ct *CardinalityTracker) hashDataPointLabels(metricName string, attrs pcommon.Map) uint64 {
	h := xxhash.New()
	
	// Include metric name in hash
	h.WriteString(metricName)
	h.WriteString("|")
	
	// Collect and sort attribute keys for consistent hashing
	keys := make([]string, 0, attrs.Len())
	attrs.Range(func(k string, v pcommon.Value) bool {
		keys = append(keys, k)
		return true
	})
	
	// Simple sort
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	
	// Hash sorted attributes
	for _, k := range keys {
		v, _ := attrs.Get(k)
		h.WriteString(k)
		h.WriteString("=")
		h.WriteString(v.AsString())
		h.WriteString("|")
	}
	
	return h.Sum64()
}

// hashLabels creates a hash of metric labels (for backward compatibility)
func (ct *CardinalityTracker) hashLabels(metric pmetric.Metric) uint64 {
	hashes := ct.getAllLabelHashes(metric)
	if len(hashes) > 0 {
		return hashes[0]
	}
	return 0
}

// TrackLabelCardinality tracks cardinality of individual labels
func (ct *CardinalityTracker) TrackLabelCardinality(labelName string, uniqueValues int) {
	ct.stats.mu.Lock()
	defer ct.stats.mu.Unlock()

	ct.stats.HighCardinalityLabels[labelName] = uniqueValues
}

// GetOldestEntries returns the oldest entries for a metric
func (ct *CardinalityTracker) GetOldestEntries(metricName string, count int) []uint64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	labelMap, exists := ct.metrics[metricName]
	if !exists {
		return nil
	}

	// Create slice of hash-timestamp pairs
	type entry struct {
		hash      uint64
		timestamp time.Time
	}
	
	entries := make([]entry, 0, len(labelMap))
	for hash, timestamp := range labelMap {
		entries = append(entries, entry{hash, timestamp})
	}

	// Sort by timestamp (oldest first)
	// Simple bubble sort for now (can optimize later)
	for i := 0; i < len(entries)-1; i++ {
		for j := 0; j < len(entries)-i-1; j++ {
			if entries[j].timestamp.After(entries[j+1].timestamp) {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}

	// Return oldest hashes
	result := make([]uint64, 0, count)
	for i := 0; i < count && i < len(entries); i++ {
		result = append(result, entries[i].hash)
	}

	return result
}

// RemoveEntry removes a specific entry
func (ct *CardinalityTracker) RemoveEntry(metricName string, labelHash uint64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if labelMap, exists := ct.metrics[metricName]; exists {
		if _, exists := labelMap[labelHash]; exists {
			delete(labelMap, labelHash)
			ct.metricCounts[metricName]--
			ct.globalCount--
		}
	}
}

// TrackDataPoint tracks a single data point by metric name and attributes
func (ct *CardinalityTracker) TrackDataPoint(metricName string, attrs pcommon.Map) (bool, uint64) {
	labelHash := ct.hashDataPointLabels(metricName, attrs)
	
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Initialize metric map if needed
	if _, exists := ct.metrics[metricName]; !exists {
		ct.metrics[metricName] = make(map[uint64]time.Time)
	}

	// Check if this label combination exists
	if _, exists := ct.metrics[metricName][labelHash]; exists {
		// Update timestamp
		ct.metrics[metricName][labelHash] = time.Now()
		return false, labelHash
	}

	// New label combination
	ct.metrics[metricName][labelHash] = time.Now()
	ct.metricCounts[metricName]++
	ct.globalCount++

	// Update stats
	ct.stats.MetricCardinalities[metricName] = ct.metricCounts[metricName]

	return true, labelHash
}