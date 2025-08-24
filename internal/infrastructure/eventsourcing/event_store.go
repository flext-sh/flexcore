// Package eventsourcing provides event sourcing capabilities with modular architecture
// SOLID SRP: Main event store orchestration using specialized modules
package eventsourcing

import (
	"fmt"
	"log"
	"sync"
)

// EventStore provides event sourcing capabilities using composition
// SOLID SRP: Orchestrates specialized services for event sourcing operations
// DRY PRINCIPLE: Eliminates 200+ lines by delegating to specialized modules
type EventStore struct {
	storage         *EventStorage
	snapshotService *SnapshotService
	mu              sync.RWMutex
}

// NewEventStore creates a new event store with specialized modules
// SOLID DIP: Depends on abstractions, not concrete implementations
func NewEventStore(dbPath string) (*EventStore, error) {
	// Initialize storage layer
	storage, err := NewEventStorage(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event storage: %w", err)
	}

	// Initialize snapshot service
	snapshotService := NewSnapshotService(storage.GetDB())
	if err := snapshotService.CreateSnapshotTable(); err != nil {
		return nil, fmt.Errorf("failed to initialize snapshot service: %w", err)
	}

	return &EventStore{
		storage:         storage,
		snapshotService: snapshotService,
	}, nil
}

// AppendEvent appends an event to the event store
// SOLID SRP: Single responsibility for event appending
func (es *EventStore) AppendEvent(event DomainEvent) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	return es.performEventAppend(event)
}

// performEventAppend performs the actual event append operation
// SOLID SRP: Single responsibility for event append
func (es *EventStore) performEventAppend(event DomainEvent) error {
	// Validate concurrency
	if err := es.validateConcurrency(event); err != nil {
		return err
	}

	// Store event using storage module
	return es.storage.StoreEvent(event)
}

// validateConcurrency validates that the event version is sequential
// SOLID SRP: Single responsibility for concurrency validation
func (es *EventStore) validateConcurrency(event DomainEvent) error {
	latestVersion, err := es.storage.GetLatestVersion(event.AggregateID())
	if err != nil {
		return err
	}

	expectedVersion := latestVersion + 1
	if event.EventVersion() != expectedVersion {
		return fmt.Errorf("concurrency conflict: expected version %d, got %d",
			expectedVersion, event.EventVersion())
	}

	return nil
}

// LoadEventStream loads events for an aggregate from a specific version
// SOLID SRP: Single responsibility for event stream loading
func (es *EventStore) LoadEventStream(aggregateID string, fromVersion int) (*EventStream, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events, err := es.storage.LoadEvents(aggregateID, fromVersion)
	if err != nil {
		return nil, err
	}

	version := 0
	if len(events) > 0 {
		version = events[len(events)-1].Version
	}

	return &EventStream{
		AggregateID: aggregateID,
		Events:      events,
		Version:     version,
	}, nil
}

// LoadAllEvents loads all events for an aggregate
func (es *EventStore) LoadAllEvents(aggregateID string) (*EventStream, error) {
	return es.LoadEventStream(aggregateID, 0)
}

// SaveSnapshot saves a snapshot for an aggregate
// SOLID SRP: Delegates snapshot operations to specialized service
func (es *EventStore) SaveSnapshot(snapshot *Snapshot) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	return es.snapshotService.SaveSnapshot(snapshot)
}

// LoadSnapshot loads a snapshot for an aggregate
// SOLID SRP: Delegates snapshot operations to specialized service
func (es *EventStore) LoadSnapshot(aggregateID string) (*Snapshot, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return es.snapshotService.LoadSnapshot(aggregateID)
}

// GetEventCount returns the total number of events for an aggregate
// SOLID SRP: Single responsibility for event counting
func (es *EventStore) GetEventCount(aggregateID string) (int, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events, err := es.storage.LoadAllEvents(aggregateID)
	if err != nil {
		return 0, err
	}

	return len(events), nil
}

// GetLatestVersion gets the latest version number for an aggregate
func (es *EventStore) GetLatestVersion(aggregateID string) (int, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return es.storage.GetLatestVersion(aggregateID)
}

// Close closes the event store and releases resources
func (es *EventStore) Close() error {
	log.Println("Closing EventStore...")
	return es.storage.Close()
}

// HealthCheck performs a health check on the event store
// SOLID SRP: Single responsibility for health checking
func (es *EventStore) HealthCheck() error {
	// Simple health check - try to get latest version of a test aggregate
	_, err := es.GetLatestVersion("health_check_aggregate")
	return err
}
