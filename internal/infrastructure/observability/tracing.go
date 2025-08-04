// Package observability provides distributed tracing capabilities
package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/flext-sh/flexcore/pkg/errors"
	"github.com/flext-sh/flexcore/pkg/result"
)

const (
	// Tracing configuration constants
	exportTimeoutSeconds = 5
	traceIDBytes         = 16
	spanIDBytes          = 8
)

// Tracer provides distributed tracing functionality
type Tracer struct {
	mu          sync.RWMutex
	serviceName string
	spans       map[string]*Span
	activeSpans map[string]*Span
	maxSpans    int
	enabled     bool
	exporters   []TraceExporter
}

// Span represents a single trace span
type Span struct {
	TraceID       string                 `json:"trace_id"`
	SpanID        string                 `json:"span_id"`
	ParentSpanID  string                 `json:"parent_span_id,omitempty"`
	OperationName string                 `json:"operation_name"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Duration      *time.Duration         `json:"duration,omitempty"`
	Tags          map[string]interface{} `json:"tags"`
	Status        SpanStatus             `json:"status"`
	Events        []SpanEvent            `json:"events"`
	ServiceName   string                 `json:"service_name"`
	mu            sync.RWMutex
}

// SpanStatus represents the status of a span
type SpanStatus struct {
	Code    SpanStatusCode `json:"code"`
	Message string         `json:"message"`
}

// SpanStatusCode represents status codes
type SpanStatusCode int

const (
	SpanStatusOK SpanStatusCode = iota
	SpanStatusError
	SpanStatusTimeout
	SpanStatusCanceled
)

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes"`
}

// TraceExporter exports traces to external systems
type TraceExporter interface {
	Export(ctx context.Context, spans []*Span) error
	Name() string
}

// SpanContext represents the context for a span
type SpanContext struct {
	TraceID    string
	SpanID     string
	Sampled    bool
	TraceState map[string]string
}

// NewTracer creates a new tracer
func NewTracer(serviceName string, maxSpans int) *Tracer {
	if maxSpans <= 0 {
		maxSpans = 10000 // Default max spans
	}

	return &Tracer{
		serviceName: serviceName,
		spans:       make(map[string]*Span),
		activeSpans: make(map[string]*Span),
		maxSpans:    maxSpans,
		enabled:     true,
		exporters:   make([]TraceExporter, 0),
	}
}

// Enable enables tracing
func (t *Tracer) Enable() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = true
}

// Disable disables tracing
func (t *Tracer) Disable() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = false
}

// IsEnabled returns whether tracing is enabled
func (t *Tracer) IsEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.enabled
}

// AddExporter adds a trace exporter
func (t *Tracer) AddExporter(exporter TraceExporter) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.exporters = append(t.exporters, exporter)
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, operationName string) (*Span, context.Context) {
	if !t.IsEnabled() {
		return nil, ctx
	}

	// Extract parent span from context if exists
	var parentSpanID string
	var traceID string

	if parentSpan := SpanFromContext(ctx); parentSpan != nil {
		parentSpanID = parentSpan.SpanID
		traceID = parentSpan.TraceID
	} else {
		traceID = t.generateTraceID()
	}

	spanID := t.generateSpanID()

	span := &Span{
		TraceID:       traceID,
		SpanID:        spanID,
		ParentSpanID:  parentSpanID,
		OperationName: operationName,
		StartTime:     time.Now(),
		Tags:          make(map[string]interface{}),
		Status:        SpanStatus{Code: SpanStatusOK},
		Events:        make([]SpanEvent, 0),
		ServiceName:   t.serviceName,
	}

	t.mu.Lock()
	// Prevent memory leak by limiting spans
	if len(t.spans) >= t.maxSpans {
		// Remove oldest spans (simple cleanup)
		for id := range t.spans {
			delete(t.spans, id)
			if len(t.spans) < t.maxSpans/2 {
				break
			}
		}
	}

	t.spans[spanID] = span
	t.activeSpans[spanID] = span
	t.mu.Unlock()

	// Add span to context
	newCtx := ContextWithSpan(ctx, span)

	return span, newCtx
}

// FinishSpan finishes a span
func (t *Tracer) FinishSpan(span *Span) {
	if span == nil || !t.IsEnabled() {
		return
	}

	span.mu.Lock()
	endTime := time.Now()
	span.EndTime = &endTime
	duration := endTime.Sub(span.StartTime)
	span.Duration = &duration
	span.mu.Unlock()

	t.mu.Lock()
	delete(t.activeSpans, span.SpanID)
	t.mu.Unlock()

	// Export to all exporters
	go t.exportSpan(span)
}

// SetSpanTag sets a tag on a span
func (t *Tracer) SetSpanTag(span *Span, key string, value interface{}) {
	if span == nil {
		return
	}

	span.mu.Lock()
	defer span.mu.Unlock()
	span.Tags[key] = value
}

// SetSpanStatus sets the status of a span
func (t *Tracer) SetSpanStatus(span *Span, code SpanStatusCode, message string) {
	if span == nil {
		return
	}

	span.mu.Lock()
	defer span.mu.Unlock()
	span.Status = SpanStatus{
		Code:    code,
		Message: message,
	}
}

// AddSpanEvent adds an event to a span
func (t *Tracer) AddSpanEvent(span *Span, name string, attributes map[string]interface{}) {
	if span == nil {
		return
	}

	event := SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attributes,
	}

	span.mu.Lock()
	defer span.mu.Unlock()
	span.Events = append(span.Events, event)
}

// GetSpan retrieves a span by ID
func (t *Tracer) GetSpan(spanID string) *Span {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.spans[spanID]
}

// GetActiveSpans returns all active spans
func (t *Tracer) GetActiveSpans() []*Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	spans := make([]*Span, 0, len(t.activeSpans))
	for _, span := range t.activeSpans {
		spans = append(spans, span)
	}
	return spans
}

// GetAllSpans returns all spans
func (t *Tracer) GetAllSpans() []*Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	spans := make([]*Span, 0, len(t.spans))
	for _, span := range t.spans {
		spans = append(spans, span)
	}
	return spans
}

// exportSpan exports a span to all configured exporters
func (t *Tracer) exportSpan(span *Span) {
	ctx, cancel := context.WithTimeout(context.Background(), exportTimeoutSeconds*time.Second)
	defer cancel()

	t.mu.RLock()
	exporters := make([]TraceExporter, len(t.exporters))
	copy(exporters, t.exporters)
	t.mu.RUnlock()

	for _, exporter := range exporters {
		if err := exporter.Export(ctx, []*Span{span}); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to export span to %s: %v\n", exporter.Name(), err)
		}
	}
}

// generateTraceID generates a new trace ID
func (t *Tracer) generateTraceID() string {
	bytes := make([]byte, traceIDBytes)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateSpanID generates a new span ID
func (t *Tracer) generateSpanID() string {
	bytes := make([]byte, spanIDBytes)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Context utilities for span propagation

type spanContextKey struct{}

// ContextWithSpan returns a new context with the span attached
func ContextWithSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, spanContextKey{}, span)
}

// SpanFromContext extracts a span from the context
func SpanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanContextKey{}).(*Span); ok {
		return span
	}
	return nil
}

// TraceFromContext extracts the trace ID from the context
func TraceFromContext(ctx context.Context) string {
	if span := SpanFromContext(ctx); span != nil {
		return span.TraceID
	}
	return ""
}

// InMemoryExporter exports traces to memory (for testing/debugging)
type InMemoryExporter struct {
	mu    sync.RWMutex
	spans []*Span
}

// NewInMemoryExporter creates a new in-memory exporter
func NewInMemoryExporter() *InMemoryExporter {
	return &InMemoryExporter{
		spans: make([]*Span, 0),
	}
}

// Export exports spans to memory
func (e *InMemoryExporter) Export(ctx context.Context, spans []*Span) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, spans...)
	return nil
}

// Name returns the exporter name
func (e *InMemoryExporter) Name() string {
	return "in-memory"
}

// GetSpans returns all exported spans
func (e *InMemoryExporter) GetSpans() []*Span {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*Span, len(e.spans))
	copy(result, e.spans)
	return result
}

// Clear clears all stored spans
func (e *InMemoryExporter) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = e.spans[:0]
}

// Name returns the name of the tracer for health checking
func (t *Tracer) Name() string {
	return "distributed-tracer"
}

// Health check for tracer
func (t *Tracer) HealthCheck(ctx context.Context) result.Result[bool] {
	if !t.IsEnabled() {
		return result.Failure[bool](errors.ValidationError("tracer is disabled"))
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	// Check if tracer is functional
	activeCount := len(t.activeSpans)
	totalCount := len(t.spans)

	// Tracer is healthy if it's enabled and not overloaded
	healthy := totalCount < t.maxSpans && activeCount < t.maxSpans/2

	if !healthy {
		return result.Failure[bool](errors.InternalError(
			fmt.Sprintf("tracer overloaded: %d/%d spans, %d active",
				totalCount, t.maxSpans, activeCount)))
	}

	return result.Success(true)
}
