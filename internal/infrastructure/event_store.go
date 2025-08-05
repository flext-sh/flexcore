package infrastructure

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flext-sh/flexcore/internal/domain/services"
	"github.com/flext-sh/flexcore/pkg/logging"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// EventEntry represents a stored event in the event store
type EventEntry struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Version   int         `json:"version"`
}

// MemoryEventStore implements an in-memory event store for Event Sourcing exactly as specified in FLEXT_SERVICE_ARCHITECTURE.md
type MemoryEventStore struct {
	events map[string][]EventEntry
	mu     sync.RWMutex
	logger logging.LoggerInterface
}

// NewMemoryEventStore creates a new in-memory event store
func NewMemoryEventStore(logger logging.LoggerInterface) *MemoryEventStore {
	return &MemoryEventStore{
		events: make(map[string][]EventEntry),
		logger: logger,
	}
}

// SaveEvent saves an event to the store exactly as specified in the architecture document
func (es *MemoryEventStore) SaveEvent(ctx context.Context, event interface{}) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	// Generate event ID
	eventID := uuid.New().String()

	// Determine event type from the event struct
	eventType := fmt.Sprintf("%T", event)

	// Create event entry
	entry := EventEntry{
		ID:        eventID,
		Type:      eventType,
		Data:      event,
		Timestamp: time.Now().UTC(),
		Version:   1,
	}

	// Store event (using event type as stream key for simplicity)
	streamKey := eventType
	es.events[streamKey] = append(es.events[streamKey], entry)

	es.logger.Debug("Event saved to store",
		zap.String("event_id", eventID),
		zap.String("event_type", eventType),
		zap.String("stream", streamKey))

	return nil
}

// SaveEvents implements EventStore interface - saves multiple domain events
func (es *MemoryEventStore) SaveEvents(ctx context.Context, aggregateID string, events []services.DomainEvent, expectedVersion int) error {
	// TODO: Implement proper version checking
	for _, event := range events {
		// Convert DomainEvent to interface{} for SaveEvent
		if err := es.SaveEvent(ctx, interface{}(event)); err != nil {
			return err
		}
	}
	return nil
}

// SaveSnapshot implements EventStore interface - placeholder
func (es *MemoryEventStore) SaveSnapshot(ctx context.Context, snapshot services.AggregateSnapshot) error {
	// TODO: Implement snapshot storage
	return nil
}

// GetSnapshot implements EventStore interface - placeholder
func (es *MemoryEventStore) GetSnapshot(ctx context.Context, aggregateID string) (services.AggregateSnapshot, error) {
	// TODO: Implement snapshot retrieval
	return services.AggregateSnapshot{}, nil
}

// GetEventsByType implements EventStore interface - placeholder
func (es *MemoryEventStore) GetEventsByType(ctx context.Context, eventType string, from time.Time) ([]services.DomainEvent, error) {
	// TODO: Implement type-based event retrieval
	return []services.DomainEvent{}, nil
}

// GetEvents retrieves events from the store (interface compatibility)
func (es *MemoryEventStore) GetEvents(ctx context.Context, aggregateID string, fromVersion int) ([]services.DomainEvent, error) {
	// TODO: Implement proper domain event conversion
	// For now, return empty slice to satisfy interface
	return []services.DomainEvent{}, nil
}

// getEventsLegacy - legacy method for backward compatibility  
func (es *MemoryEventStore) getEventsLegacy(ctx context.Context, streamKey string) ([]EventEntry, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events, exists := es.events[streamKey]
	if !exists {
		return []EventEntry{}, nil
	}

	// Return a copy to prevent external modification
	result := make([]EventEntry, len(events))
	copy(result, events)

	return result, nil
}

// GetAllEvents retrieves all events from the store
func (es *MemoryEventStore) GetAllEvents(ctx context.Context) ([]EventEntry, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var allEvents []EventEntry
	for _, events := range es.events {
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

// InMemoryEventBus implements an in-memory event bus for Event Sourcing + CQRS exactly as specified in FLEXT_SERVICE_ARCHITECTURE.md
type InMemoryEventBus struct {
	handlers map[string][]func(context.Context, services.DomainEvent) error
	mu       sync.RWMutex
	logger   logging.LoggerInterface
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus(logger logging.LoggerInterface) *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]func(context.Context, services.DomainEvent) error),
		logger:   logger,
	}
}

// Publish publishes an event to all registered handlers exactly as specified in the architecture document
func (eb *InMemoryEventBus) Publish(ctx context.Context, event services.DomainEvent) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	eventType := fmt.Sprintf("%T", event)
	handlers, exists := eb.handlers[eventType]

	if !exists {
		eb.logger.Debug("No handlers registered for event type", zap.String("event_type", eventType))
		return nil
	}

	eb.logger.Debug("Publishing event to handlers",
		zap.String("event_type", eventType),
		zap.Int("handler_count", len(handlers)))

	// Execute all handlers
	for i, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			eb.logger.Error("Event handler failed",
				zap.String("event_type", eventType),
				zap.Int("handler_index", i),
				zap.Error(err))
			return fmt.Errorf("handler %d failed for event %s: %w", i, eventType, err)
		}
	}

	return nil
}

// Subscribe registers a handler for a specific event type
func (eb *InMemoryEventBus) Subscribe(eventType string, handler services.EventHandler) error {
	// Wrap the EventHandler interface into our internal function type
	internalHandler := func(ctx context.Context, event services.DomainEvent) error {
		return handler.Handle(ctx, event)
	}
	eb.subscribeInternal(eventType, internalHandler)
	return nil
}

// subscribeInternal is the internal subscribe method
func (eb *InMemoryEventBus) subscribeInternal(eventType string, handler func(context.Context, services.DomainEvent) error) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	eb.logger.Debug("Handler subscribed", zap.String("event_type", eventType))
}

// Unsubscribe implements EventBus interface
func (eb *InMemoryEventBus) Unsubscribe(eventType string, handler services.EventHandler) error {
	// TODO: Implement proper handler removal
	return nil
}

// Close implements the EventBus interface Close method
// Cleans up resources and stops the event bus
func (eb *InMemoryEventBus) Close() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Clear all handlers
	eb.handlers = make(map[string][]func(context.Context, services.DomainEvent) error)
	eb.logger.Debug("EventBus closed and handlers cleared")
	return nil
}
