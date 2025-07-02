// Package observability tests
package observability_test

import (
	"context"
	"testing"
	"time"

	"github.com/flext/flexcore/infrastructure/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Metrics Collector

func TestMetricsCollector(t *testing.T) {
	collector := observability.NewMetricsCollector()
	require.NotNil(t, collector)
	assert.True(t, collector.IsEnabled())

	// Test counter
	collector.IncrementCounter("test_counter", map[string]string{"env": "test"})
	collector.AddToCounter("test_counter", 5, map[string]string{"env": "test"})
	
	count := collector.GetCounter("test_counter", map[string]string{"env": "test"})
	assert.Equal(t, int64(6), count)

	// Test gauge
	collector.SetGauge("test_gauge", 42.5, map[string]string{"type": "temperature"})
	gauge := collector.GetGauge("test_gauge", map[string]string{"type": "temperature"})
	assert.Equal(t, 42.5, gauge)

	// Test histogram
	collector.RecordHistogram("test_histogram", 1.5, map[string]string{"operation": "test"})
	collector.RecordHistogram("test_histogram", 2.5, map[string]string{"operation": "test"})

	// Test timer
	timer := collector.StartTimer("test_timer")
	time.Sleep(10 * time.Millisecond)
	duration := timer.Stop()
	assert.True(t, duration > 10*time.Millisecond)

	// Test get all metrics
	metrics := collector.GetAllMetrics()
	assert.True(t, len(metrics) > 0)
	
	// Find counter metric
	foundCounter := false
	for _, metric := range metrics {
		if metric.Name == "test_counter" && metric.Type == "counter" {
			foundCounter = true
			assert.Equal(t, int64(6), metric.Value)
			break
		}
	}
	assert.True(t, foundCounter, "Counter metric should be found")

	// Test health check
	result := collector.HealthCheck(context.Background())
	assert.True(t, result.IsSuccess())
	assert.True(t, result.Value())

	// Test disable/enable
	collector.Disable()
	assert.False(t, collector.IsEnabled())
	
	collector.Enable()
	assert.True(t, collector.IsEnabled())
}

func TestMetricsCollectorDisabled(t *testing.T) {
	collector := observability.NewMetricsCollector()
	collector.Disable()

	// Operations should not affect metrics when disabled
	collector.IncrementCounter("disabled_counter", nil)
	count := collector.GetCounter("disabled_counter", nil)
	assert.Equal(t, int64(0), count)

	collector.SetGauge("disabled_gauge", 100.0, nil)
	gauge := collector.GetGauge("disabled_gauge", nil)
	assert.Equal(t, 0.0, gauge)
}

// Test Tracer

func TestTracer(t *testing.T) {
	tracer := observability.NewTracer("test-service", 100)
	require.NotNil(t, tracer)
	assert.True(t, tracer.IsEnabled())

	ctx := context.Background()

	// Test span creation
	span, spanCtx := tracer.StartSpan(ctx, "test_operation")
	require.NotNil(t, span)
	assert.NotEmpty(t, span.TraceID)
	assert.NotEmpty(t, span.SpanID)
	assert.Equal(t, "test_operation", span.OperationName)
	assert.Equal(t, "test-service", span.ServiceName)

	// Test span from context
	extractedSpan := observability.SpanFromContext(spanCtx)
	assert.Equal(t, span, extractedSpan)

	// Test trace ID from context
	traceID := observability.TraceFromContext(spanCtx)
	assert.Equal(t, span.TraceID, traceID)

	// Test span tags
	tracer.SetSpanTag(span, "user_id", "123")
	tracer.SetSpanTag(span, "operation_type", "read")
	
	assert.Equal(t, "123", span.Tags["user_id"])
	assert.Equal(t, "read", span.Tags["operation_type"])

	// Test span events
	tracer.AddSpanEvent(span, "cache_miss", map[string]interface{}{
		"cache_key": "user:123",
		"ttl":       300,
	})
	
	assert.Len(t, span.Events, 1)
	assert.Equal(t, "cache_miss", span.Events[0].Name)
	assert.Equal(t, "user:123", span.Events[0].Attributes["cache_key"])

	// Test span status
	tracer.SetSpanStatus(span, observability.SpanStatusOK, "Operation completed successfully")
	assert.Equal(t, observability.SpanStatusOK, span.Status.Code)
	assert.Equal(t, "Operation completed successfully", span.Status.Message)

	// Test active spans
	activeSpans := tracer.GetActiveSpans()
	assert.Len(t, activeSpans, 1)
	assert.Equal(t, span, activeSpans[0])

	// Test span finishing
	tracer.FinishSpan(span)
	assert.NotNil(t, span.EndTime)
	assert.NotNil(t, span.Duration)
	
	// Should no longer be active
	activeSpans = tracer.GetActiveSpans()
	assert.Len(t, activeSpans, 0)

	// Should still be in all spans
	allSpans := tracer.GetAllSpans()
	assert.Len(t, allSpans, 1)
}

func TestTracerNestedSpans(t *testing.T) {
	tracer := observability.NewTracer("test-service", 100)
	ctx := context.Background()

	// Create parent span
	parentSpan, parentCtx := tracer.StartSpan(ctx, "parent_operation")
	require.NotNil(t, parentSpan)

	// Create child span
	childSpan, childCtx := tracer.StartSpan(parentCtx, "child_operation")
	require.NotNil(t, childSpan)

	// Child should have same trace ID but different span ID
	assert.Equal(t, parentSpan.TraceID, childSpan.TraceID)
	assert.NotEqual(t, parentSpan.SpanID, childSpan.SpanID)
	assert.Equal(t, parentSpan.SpanID, childSpan.ParentSpanID)

	// Test context propagation
	extractedChild := observability.SpanFromContext(childCtx)
	assert.Equal(t, childSpan, extractedChild)

	// Finish spans
	tracer.FinishSpan(childSpan)
	tracer.FinishSpan(parentSpan)

	// Both should be finished
	assert.NotNil(t, childSpan.EndTime)
	assert.NotNil(t, parentSpan.EndTime)
}

func TestTracerExporter(t *testing.T) {
	tracer := observability.NewTracer("test-service", 100)
	exporter := observability.NewInMemoryExporter()
	tracer.AddExporter(exporter)

	ctx := context.Background()
	span, _ := tracer.StartSpan(ctx, "exported_operation")
	tracer.FinishSpan(span)

	// Give exporter time to receive span
	time.Sleep(10 * time.Millisecond)

	exportedSpans := exporter.GetSpans()
	assert.Len(t, exportedSpans, 1)
	assert.Equal(t, span.SpanID, exportedSpans[0].SpanID)
}

func TestTracerHealthCheck(t *testing.T) {
	tracer := observability.NewTracer("test-service", 100)
	
	result := tracer.HealthCheck(context.Background())
	assert.True(t, result.IsSuccess())
	assert.True(t, result.Value())

	// Test disabled tracer
	tracer.Disable()
	result = tracer.HealthCheck(context.Background())
	assert.True(t, result.IsFailure())
}

// Test Monitor

func TestMonitor(t *testing.T) {
	metrics := observability.NewMetricsCollector()
	tracer := observability.NewTracer("test-service", 100)
	
	config := observability.MonitorConfig{
		MaxAlerts: 10,
		AlertThresholds: observability.AlertThresholds{
			MemoryMB:      100.0,
			GoroutinesMax: 1000,
		},
	}
	
	monitor := observability.NewMonitor(metrics, tracer, config)
	require.NotNil(t, monitor)

	// Test health checkers
	monitor.AddHealthChecker(metrics)
	monitor.AddHealthChecker(tracer)

	// Test health status
	ctx := context.Background()
	status := monitor.GetHealthStatus(ctx)
	assert.Equal(t, observability.HealthStateHealthy, status.Overall)
	assert.Len(t, status.Services, 2)

	// Test system metrics
	sysMetrics := monitor.GetSystemMetrics()
	assert.True(t, sysMetrics.Goroutines > 0)
	assert.True(t, sysMetrics.MemoryMB > 0)

	// Test alert creation
	monitor.CreateAlert(
		observability.AlertTypeSystem,
		observability.AlertSeverityWarning,
		"Test Alert",
		"This is a test alert",
		map[string]interface{}{"test": true},
	)

	// Give time for alert processing
	time.Sleep(10 * time.Millisecond)

	alerts := monitor.GetAlerts()
	assert.Len(t, alerts, 1)
	assert.Equal(t, "Test Alert", alerts[0].Title)
	assert.Equal(t, observability.AlertSeverityWarning, alerts[0].Severity)

	// Test monitor start/stop
	assert.False(t, monitor.IsRunning())
	
	err := monitor.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, monitor.IsRunning())
	
	monitor.Stop()
	assert.False(t, monitor.IsRunning())
}

type testSubscriber struct {
	alerts       []observability.Alert
	metricsCount int
	healthCount  int
}

func (s *testSubscriber) OnAlert(alert observability.Alert) {
	s.alerts = append(s.alerts, alert)
}

func (s *testSubscriber) OnMetricsUpdate(metrics []observability.MetricPoint) {
	s.metricsCount++
}

func (s *testSubscriber) OnHealthUpdate(status observability.HealthStatus) {
	s.healthCount++
}

func TestMonitorSubscriber(t *testing.T) {
	metrics := observability.NewMetricsCollector()
	tracer := observability.NewTracer("test-service", 100)
	
	config := observability.MonitorConfig{
		MetricsInterval:  50 * time.Millisecond,
		HealthInterval:   50 * time.Millisecond,
		ResourceInterval: 50 * time.Millisecond,
		MaxAlerts:        10,
	}
	
	monitor := observability.NewMonitor(metrics, tracer, config)
	subscriber := &testSubscriber{alerts: make([]observability.Alert, 0)}
	
	monitor.Subscribe(subscriber)
	monitor.AddHealthChecker(metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := monitor.Start(ctx)
	assert.NoError(t, err)

	// Create an alert
	monitor.CreateAlert(
		observability.AlertTypeCustom,
		observability.AlertSeverityInfo,
		"Subscriber Test",
		"Testing subscriber functionality",
		nil,
	)

	// Wait for monitoring cycles
	time.Sleep(150 * time.Millisecond)
	monitor.Stop()

	// Check subscriber received events
	assert.Len(t, subscriber.alerts, 1)
	assert.Equal(t, "Subscriber Test", subscriber.alerts[0].Title)
	
	// Should have received at least one metrics and health update
	assert.True(t, subscriber.metricsCount > 0)
	assert.True(t, subscriber.healthCount > 0)
}

// Benchmark tests

func BenchmarkMetricsCounterIncrement(b *testing.B) {
	collector := observability.NewMetricsCollector()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.IncrementCounter("benchmark_counter", nil)
	}
}

func BenchmarkTracerSpanCreation(b *testing.B) {
	tracer := observability.NewTracer("benchmark-service", 10000)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span, _ := tracer.StartSpan(ctx, "benchmark_operation")
		tracer.FinishSpan(span)
	}
}

func BenchmarkMetricsHistogramRecord(b *testing.B) {
	collector := observability.NewMetricsCollector()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordHistogram("benchmark_histogram", float64(i%100), nil)
	}
}