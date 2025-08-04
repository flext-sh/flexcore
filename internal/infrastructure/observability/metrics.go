// Package observability provides comprehensive monitoring, metrics, and tracing
package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flext-sh/flexcore/pkg/errors"
	"github.com/flext-sh/flexcore/pkg/result"
)

const (
	// Histogram configuration constants
	histogramBucketsCount = 10 // 9 buckets + inf
	maxTimerSamples       = 1000
)

// MetricsCollector provides real-time metrics collection
type MetricsCollector struct {
	mu         sync.RWMutex
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	timers     map[string]*Timer
	enabled    bool
}

// Counter represents a monotonically increasing counter
type Counter struct {
	mu    sync.RWMutex
	name  string
	tags  map[string]string
	value int64
}

// GetName returns the counter name
func (c *Counter) GetName() string {
	return c.name
}

// GetTags returns the counter tags
func (c *Counter) GetTags() map[string]string {
	return c.tags
}

// SetMetadata sets the counter metadata
func (c *Counter) SetMetadata(name string, tags map[string]string) {
	c.name = name
	c.tags = tags
}

// Gauge represents a value that can go up and down
type Gauge struct {
	mu    sync.RWMutex
	name  string
	tags  map[string]string
	value float64
}

// GetName returns the gauge name
func (g *Gauge) GetName() string {
	return g.name
}

// GetTags returns the gauge tags
func (g *Gauge) GetTags() map[string]string {
	return g.tags
}

// SetMetadata sets the gauge metadata
func (g *Gauge) SetMetadata(name string, tags map[string]string) {
	g.name = name
	g.tags = tags
}

// Histogram tracks distribution of values
type Histogram struct {
	mu      sync.RWMutex
	name    string
	tags    map[string]string
	buckets []float64
	counts  []int64
	sum     float64
	count   int64
}

// Timer tracks timing information
type Timer struct {
	name      string
	durations []time.Duration
	mu        sync.RWMutex
}

// MetricPoint represents a single metric measurement
type MetricPoint struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Tags      map[string]string `json:"tags"`
	Value     interface{}       `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
}

// Metric represents a generic metric interface
// SOLID SRP: Common interface for all metric types
type Metric interface {
	GetName() string
	GetTags() map[string]string
	SetMetadata(name string, tags map[string]string)
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		timers:     make(map[string]*Timer),
		enabled:    true,
	}
}

// Enable enables metrics collection
func (mc *MetricsCollector) Enable() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.enabled = true
}

// Disable disables metrics collection
func (mc *MetricsCollector) Disable() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.enabled = false
}

// IsEnabled returns whether metrics collection is enabled
func (mc *MetricsCollector) IsEnabled() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.enabled
}

// Counter operations

// IncrementCounter increments a counter by 1
func (mc *MetricsCollector) IncrementCounter(name string, tags map[string]string) {
	mc.AddToCounter(name, 1, tags)
}

// AddToCounter adds a value to a counter
// DRY PRINCIPLE: Uses shared metric management logic
func (mc *MetricsCollector) AddToCounter(name string, value int64, tags map[string]string) {
	mc.manageCounterMetric(name, tags, func(counter *Counter) {
		counter.mu.Lock()
		counter.value += value
		counter.mu.Unlock()
	})
}

// GetCounter returns the current value of a counter
func (mc *MetricsCollector) GetCounter(name string, tags map[string]string) int64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := mc.getMetricKey(name, tags)
	if counter, exists := mc.counters[key]; exists {
		counter.mu.RLock()
		defer counter.mu.RUnlock()
		return counter.value
	}
	return 0
}

// Gauge operations

// SetGauge sets a gauge value
// DRY PRINCIPLE: Uses shared metric management logic
func (mc *MetricsCollector) SetGauge(name string, value float64, tags map[string]string) {
	mc.manageGaugeMetric(name, tags, func(gauge *Gauge) {
		gauge.mu.Lock()
		gauge.value = value
		gauge.mu.Unlock()
	})
}

// GetGauge returns the current value of a gauge
func (mc *MetricsCollector) GetGauge(name string, tags map[string]string) float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := mc.getMetricKey(name, tags)
	if gauge, exists := mc.gauges[key]; exists {
		gauge.mu.RLock()
		defer gauge.mu.RUnlock()
		return gauge.value
	}
	return 0
}

// Histogram operations

// RecordHistogram records a value in a histogram
func (mc *MetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	if !mc.IsEnabled() {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.getMetricKey(name, tags)
	hist, exists := mc.histograms[key]
	if !exists {
		// Default buckets: 0.1, 0.5, 1, 2.5, 5, 10, 25, 50, 100
		hist = &Histogram{
			name:    name,
			buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10, 25, 50, 100},
			counts:  make([]int64, histogramBucketsCount), // 9 buckets + inf
			tags:    tags,
		}
		mc.histograms[key] = hist
	}

	hist.mu.Lock()
	defer hist.mu.Unlock()

	hist.sum += value
	hist.count++

	// Find appropriate bucket
	for i, bucket := range hist.buckets {
		if value <= bucket {
			hist.counts[i]++
			break
		}
	}
	// If value is larger than all buckets, increment +Inf bucket
	if value > hist.buckets[len(hist.buckets)-1] {
		hist.counts[len(hist.counts)-1]++
	}
}

// Timer operations

// StartTimer starts a new timer
func (mc *MetricsCollector) StartTimer(name string) *TimerContext {
	return &TimerContext{
		collector: mc,
		name:      name,
		startTime: time.Now(),
	}
}

// TimerContext represents an active timer
type TimerContext struct {
	collector *MetricsCollector
	startTime time.Time
	name      string
}

// Stop stops the timer and records the duration
func (tc *TimerContext) Stop() time.Duration {
	duration := time.Since(tc.startTime)
	tc.collector.RecordTimer(tc.name, duration)
	return duration
}

// RecordTimer records a timer duration
func (mc *MetricsCollector) RecordTimer(name string, duration time.Duration) {
	if !mc.IsEnabled() {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	timer, exists := mc.timers[name]
	if !exists {
		timer = &Timer{
			name: name,
		}
		mc.timers[name] = timer
	}

	timer.mu.Lock()
	timer.durations = append(timer.durations, duration)
	// Keep only last 1000 measurements to prevent memory leak
	if len(timer.durations) > maxTimerSamples {
		timer.durations = timer.durations[len(timer.durations)-1000:]
	}
	timer.mu.Unlock()

	// Also record as histogram in seconds
	mc.RecordHistogram(name+"_duration_seconds", duration.Seconds(), nil)
}

// GetAllMetrics returns all current metrics
func (mc *MetricsCollector) GetAllMetrics() []MetricPoint {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var metrics []MetricPoint
	now := time.Now()

	// Counters
	for _, counter := range mc.counters {
		counter.mu.RLock()
		metrics = append(metrics, MetricPoint{
			Name:      counter.name,
			Type:      "counter",
			Value:     counter.value,
			Tags:      counter.tags,
			Timestamp: now,
		})
		counter.mu.RUnlock()
	}

	// Gauges
	for _, gauge := range mc.gauges {
		gauge.mu.RLock()
		metrics = append(metrics, MetricPoint{
			Name:      gauge.name,
			Type:      "gauge",
			Value:     gauge.value,
			Tags:      gauge.tags,
			Timestamp: now,
		})
		gauge.mu.RUnlock()
	}

	// Histograms
	for _, hist := range mc.histograms {
		hist.mu.RLock()
		metrics = append(metrics, MetricPoint{
			Name:      hist.name + "_sum",
			Type:      "histogram_sum",
			Value:     hist.sum,
			Tags:      hist.tags,
			Timestamp: now,
		})
		metrics = append(metrics, MetricPoint{
			Name:      hist.name + "_count",
			Type:      "histogram_count",
			Value:     hist.count,
			Tags:      hist.tags,
			Timestamp: now,
		})
		hist.mu.RUnlock()
	}

	return metrics
}

// Reset clears all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.counters = make(map[string]*Counter)
	mc.gauges = make(map[string]*Gauge)
	mc.histograms = make(map[string]*Histogram)
	mc.timers = make(map[string]*Timer)
}

// getMetricKey generates a unique key for a metric with tags
func (mc *MetricsCollector) getMetricKey(name string, tags map[string]string) string {
	key := name
	if tags != nil {
		for k, v := range tags {
			key += fmt.Sprintf("_%s_%s", k, v)
		}
	}
	return key
}

// MetricCreator interface for creating metrics with unified structure
// SOLID SRP: Single responsibility for metric creation
type MetricCreator[T Metric] interface {
	CreateMetric(name string, tags map[string]string) T
	GetMetricMap() map[string]T
}

// CounterCreator implements MetricCreator for Counter metrics
// SOLID SRP: Single responsibility for counter creation
type CounterCreator struct {
	mc *MetricsCollector
}

// CreateMetric creates a new counter metric
// SOLID SRP: Factory method for counter creation
func (cc *CounterCreator) CreateMetric(name string, tags map[string]string) *Counter {
	return &Counter{
		name: name,
		tags: tags,
	}
}

// GetMetricMap returns the counter metrics map
// SOLID SRP: Accessor for counter map
func (cc *CounterCreator) GetMetricMap() map[string]*Counter {
	return cc.mc.counters
}

// GaugeCreator implements MetricCreator for Gauge metrics
// SOLID SRP: Single responsibility for gauge creation
type GaugeCreator struct {
	mc *MetricsCollector
}

// CreateMetric creates a new gauge metric
// SOLID SRP: Factory method for gauge creation
func (gc *GaugeCreator) CreateMetric(name string, tags map[string]string) *Gauge {
	return &Gauge{
		name: name,
		tags: tags,
	}
}

// GetMetricMap returns the gauge metrics map
// SOLID SRP: Accessor for gauge map
func (gc *GaugeCreator) GetMetricMap() map[string]*Gauge {
	return gc.mc.gauges
}

// manageMetric manages metric creation and operation using Go generics
// SOLID SRP: Single responsibility for unified metric management
// DRY PRINCIPLE: Eliminates 20-line duplication (mass=106) between manageCounterMetric and manageGaugeMetric
func manageMetric[T Metric](mc *MetricsCollector, creator MetricCreator[T], name string, tags map[string]string, operation func(T)) {
	if !mc.IsEnabled() {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.getMetricKey(name, tags)
	metricMap := creator.GetMetricMap()
	metric, exists := metricMap[key]
	if !exists {
		metric = creator.CreateMetric(name, tags)
		metricMap[key] = metric
	}

	operation(metric)
}

// manageCounterMetric manages counter metric creation and operation
// SOLID SRP: Single responsibility for counter metric management
// DRY PRINCIPLE: Uses unified manageMetric eliminating 20-line duplication (mass=106)
func (mc *MetricsCollector) manageCounterMetric(name string, tags map[string]string, operation func(*Counter)) {
	creator := &CounterCreator{mc: mc}
	manageMetric(mc, creator, name, tags, operation)
}

// manageGaugeMetric manages gauge metric creation and operation
// SOLID SRP: Single responsibility for gauge metric management
// DRY PRINCIPLE: Uses unified manageMetric eliminating 20-line duplication (mass=106)
func (mc *MetricsCollector) manageGaugeMetric(name string, tags map[string]string, operation func(*Gauge)) {
	creator := &GaugeCreator{mc: mc}
	manageMetric(mc, creator, name, tags, operation)
}

// Name returns the name of the metrics collector for health checking
func (mc *MetricsCollector) Name() string {
	return "metrics-collector"
}

// Health check for metrics collector
func (mc *MetricsCollector) HealthCheck(ctx context.Context) result.Result[bool] {
	if !mc.IsEnabled() {
		return result.Failure[bool](errors.ValidationError("metrics collector is disabled"))
	}

	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Check if we have any metrics (indicates system is working)
	totalMetrics := len(mc.counters) + len(mc.gauges) + len(mc.histograms) + len(mc.timers)

	return result.Success(totalMetrics >= 0) // Always healthy if enabled
}
