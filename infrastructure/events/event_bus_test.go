// Package events tests
package events_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/flext/flexcore/domain"
	"github.com/flext/flexcore/infrastructure/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test event types

type TestEvent struct {
	domain.BaseDomainEvent
	Message string
}

type AnotherTestEvent struct {
	domain.BaseDomainEvent
	Value int
}

// Test event handler

type TestEventHandler struct {
	mu           sync.Mutex
	handledEvents []domain.DomainEvent
	delay        time.Duration
	shouldError  bool
}

func NewTestEventHandler() *TestEventHandler {
	return &TestEventHandler{
		handledEvents: make([]domain.DomainEvent, 0),
	}
}

func (h *TestEventHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.delay > 0 {
		time.Sleep(h.delay)
	}

	if h.shouldError {
		return assert.AnError
	}

	h.handledEvents = append(h.handledEvents, event)
	return nil
}

func (h *TestEventHandler) GetHandledEvents() []domain.DomainEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]domain.DomainEvent{}, h.handledEvents...)
}

func (h *TestEventHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handledEvents = make([]domain.DomainEvent, 0)
}

// Tests

func TestEventBusSubscribeAndPublish(t *testing.T) {
	bus := events.NewInMemoryEventBus(4, 100)
	handler := NewTestEventHandler()
	
	// Start the event bus
	err := bus.Start(context.Background())
	require.NoError(t, err)
	defer bus.Stop()

	// Subscribe to TestEvent
	bus.Subscribe("TestEvent", handler.Handle)

	// Publish event
	event := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "Hello, World!",
	}

	err = bus.Publish(context.Background(), event)
	require.NoError(t, err)

	// Give some time for async processing
	time.Sleep(50 * time.Millisecond)

	// Check handler received event
	handled := handler.GetHandledEvents()
	assert.Len(t, handled, 1)
	assert.Equal(t, event, handled[0])
}

func TestEventBusMultipleHandlers(t *testing.T) {
	bus := events.NewInMemoryEventBus(4, 100)
	handler1 := NewTestEventHandler()
	handler2 := NewTestEventHandler()
	handler3 := NewTestEventHandler()

	// Start the event bus
	err := bus.Start(context.Background())
	require.NoError(t, err)
	defer bus.Stop()

	// Subscribe multiple handlers to same event
	bus.Subscribe("TestEvent", handler1.Handle)
	bus.Subscribe("TestEvent", handler2.Handle)
	bus.Subscribe("TestEvent", handler3.Handle)

	// Publish event
	event := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "Broadcast message",
	}

	err = bus.Publish(context.Background(), event)
	require.NoError(t, err)

	// Give time for all handlers
	time.Sleep(100 * time.Millisecond)

	// All handlers should receive the event
	assert.Len(t, handler1.GetHandledEvents(), 1)
	assert.Len(t, handler2.GetHandledEvents(), 1)
	assert.Len(t, handler3.GetHandledEvents(), 1)
}

func TestEventBusMultipleEventTypes(t *testing.T) {
	bus := events.NewInMemoryEventBus(4, 100)
	handler1 := NewTestEventHandler()
	handler2 := NewTestEventHandler()

	// Start the event bus
	err := bus.Start(context.Background())
	require.NoError(t, err)
	defer bus.Stop()

	// Subscribe to different event types
	bus.Subscribe("TestEvent", handler1.Handle)
	bus.Subscribe("AnotherTestEvent", handler2.Handle)

	// Publish both types
	event1 := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "Test message",
	}

	event2 := &AnotherTestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("AnotherTestEvent", "aggregate-2"),
		Value:           42,
	}

	err = bus.Publish(context.Background(), event1)
	require.NoError(t, err)

	err = bus.Publish(context.Background(), event2)
	require.NoError(t, err)

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	// Each handler should only receive its subscribed event type
	assert.Len(t, handler1.GetHandledEvents(), 1)
	assert.Equal(t, event1, handler1.GetHandledEvents()[0])

	assert.Len(t, handler2.GetHandledEvents(), 1)
	assert.Equal(t, event2, handler2.GetHandledEvents()[0])
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := events.NewInMemoryEventBus(4, 100)
	handler := NewTestEventHandler()

	// Start the event bus
	err := bus.Start(context.Background())
	require.NoError(t, err)
	defer bus.Stop()

	// Subscribe to event
	err = bus.Subscribe("TestEvent", handler.Handle)
	require.NoError(t, err)

	// Publish first event
	event1 := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "First message",
	}
	err = bus.Publish(context.Background(), event1)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.Len(t, handler.GetHandledEvents(), 1)

	// Unsubscribe
	bus.Unsubscribe("TestEvent", handler.Handle)
	handler.Reset()

	// Publish second event
	event2 := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "Second message",
	}
	err = bus.Publish(context.Background(), event2)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Handler should not receive the second event
	assert.Len(t, handler.GetHandledEvents(), 0)
}

func TestEventBusPublishAll(t *testing.T) {
	bus := events.NewInMemoryEventBus(4, 100)
	handler := NewTestEventHandler()

	// Start the event bus
	err := bus.Start(context.Background())
	require.NoError(t, err)
	defer bus.Stop()

	bus.Subscribe("TestEvent", handler.Handle)
	bus.Subscribe("AnotherTestEvent", handler.Handle)

	// Create multiple events
	events := []domain.DomainEvent{
		&TestEvent{
			BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
			Message:         "Event 1",
		},
		&AnotherTestEvent{
			BaseDomainEvent: domain.NewBaseDomainEvent("AnotherTestEvent", "aggregate-2"),
			Value:           100,
		},
		&TestEvent{
			BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-3"),
			Message:         "Event 3",
		},
	}

	// Publish all events
	for _, event := range events {
		err = bus.Publish(context.Background(), event)
		require.NoError(t, err)
	}

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Handler should receive all events
	handled := handler.GetHandledEvents()
	assert.Len(t, handled, 3)
}

func TestEventBusConcurrentPublish(t *testing.T) {
	bus := events.NewInMemoryEventBus(4, 100)
	handler := NewTestEventHandler()

	// Start the event bus
	err := bus.Start(context.Background())
	require.NoError(t, err)
	defer bus.Stop()

	bus.Subscribe("TestEvent", handler.Handle)

	// Publish events concurrently
	wg := sync.WaitGroup{}
	eventCount := 100

	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			event := &TestEvent{
				BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
				Message:         "Concurrent event",
			}
			err = bus.Publish(context.Background(), event)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Give time for all events to be processed
	time.Sleep(200 * time.Millisecond)

	// All events should be handled
	assert.Len(t, handler.GetHandledEvents(), eventCount)
}

func TestEventBusHandlerError(t *testing.T) {
	bus := events.NewInMemoryEventBus(4, 100)
	handler := NewTestEventHandler()

	// Start the event bus
	err := bus.Start(context.Background())
	require.NoError(t, err)
	defer bus.Stop()
	handler.shouldError = true

	bus.Subscribe("TestEvent", handler.Handle)

	event := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "This will cause error",
	}

	// Publish should not return error even if handler fails
	// (async processing, errors are logged)
	err = bus.Publish(context.Background(), event)
	assert.NoError(t, err)
}

func TestEventBusContextCancellation(t *testing.T) {
	bus := events.NewInMemoryEventBus(4, 100)
	handler := NewTestEventHandler()

	// Start the event bus
	err := bus.Start(context.Background())
	require.NoError(t, err)
	defer bus.Stop()
	handler.delay = 100 * time.Millisecond // Slow handler

	bus.Subscribe("TestEvent", handler.Handle)

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	event := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "Event with context",
	}

	// Publish event
	err = bus.Publish(ctx, event)
	require.NoError(t, err)

	// Cancel context immediately
	cancel()

	// The handler might still process the event since it's async
	// This test just ensures the bus handles context properly
	time.Sleep(50 * time.Millisecond)
}

func TestEventBusNoHandlers(t *testing.T) {
	bus := events.NewInMemoryEventBus(4, 100)

	// Start the event bus
	err := bus.Start(context.Background())
	require.NoError(t, err)
	defer bus.Stop()

	event := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "No one is listening",
	}

	// Publishing to an event with no handlers should not error
	err = bus.Publish(context.Background(), event)
	assert.NoError(t, err)
}

func TestEventBusHandlerPanic(t *testing.T) {
	bus := events.NewInMemoryEventBus(4, 100)

	// Start the event bus
	err := bus.Start(context.Background())
	require.NoError(t, err)
	defer bus.Stop()

	// Create a handler that panics
	panicHandler := events.EventHandler(func(ctx context.Context, event domain.DomainEvent) error {
		panic("handler panic!")
	})

	bus.Subscribe("TestEvent", panicHandler)

	event := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "This will cause panic",
	}

	// The bus should recover from panics and not crash
	assert.NotPanics(t, func() {
		err = bus.Publish(context.Background(), event)
		assert.NoError(t, err)
	})
}

// Benchmarks

func BenchmarkEventBusPublish(b *testing.B) {
	bus := events.NewInMemoryEventBus(4, 100)
	handler := NewTestEventHandler()
	bus.Subscribe("TestEvent", handler.Handle)

	event := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "Benchmark event",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBusPublishNoHandlers(b *testing.B) {
	bus := events.NewInMemoryEventBus(4, 100)

	event := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "Benchmark event",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}

func BenchmarkEventBusPublishMultipleHandlers(b *testing.B) {
	bus := events.NewInMemoryEventBus(4, 100)

	// Add 10 handlers
	for i := 0; i < 10; i++ {
		handler := NewTestEventHandler()
		bus.Subscribe("TestEvent", handler.Handle)
	}

	event := &TestEvent{
		BaseDomainEvent: domain.NewBaseDomainEvent("TestEvent", "aggregate-1"),
		Message:         "Benchmark event",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bus.Publish(context.Background(), event)
	}
}