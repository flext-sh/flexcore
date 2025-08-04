// Package eventsourcing provides event sourcing capabilities with modular architecture
// SOLID SRP: Main event store orchestration using specialized modules
package eventsourcing

import (
	"fmt"
	"log"
	"sync"

	"github.com/flext-sh/flexcore/pkg/result"
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
// SOLID SRP: Single responsibility for event appending with Result pattern
func (es *EventStore) AppendEvent(event DomainEvent) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	result := es.performEventAppend(event)
	if result.IsFailure() {
		return result.Error()
	}
	return nil
}

// performEventAppend performs the actual event append operation
// SOLID SRP: Single responsibility for event append with Result pattern
func (es *EventStore) performEventAppend(event DomainEvent) result.Result[bool] {
	// Validate concurrency
	if concurrencyResult := es.validateConcurrency(event); concurrencyResult.IsFailure() {
		return concurrencyResult
	}

	// Store event using storage module
	return es.storage.StoreEvent(event)
}

// validateConcurrency validates that the event version is sequential
// SOLID SRP: Single responsibility for concurrency validation
func (es *EventStore) validateConcurrency(event DomainEvent) result.Result[bool] {
	latestVersionResult := es.storage.GetLatestVersion(event.AggregateID())
	if latestVersionResult.IsFailure() {
		return result.Failure[bool](latestVersionResult.Error())
	}

	expectedVersion := latestVersionResult.Value() + 1
	if event.EventVersion() != expectedVersion {
		return result.Failure[bool](
			fmt.Errorf("concurrency conflict: expected version %d, got %d",
				expectedVersion, event.EventVersion()))
	}

	return result.Success(true)
}

// LoadEventStream loads events for an aggregate from a specific version
// SOLID SRP: Single responsibility for event stream loading
func (es *EventStore) LoadEventStream(aggregateID string, fromVersion int) (*EventStream, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	eventsResult := es.storage.LoadEvents(aggregateID, fromVersion)
	if eventsResult.IsFailure() {
		return nil, eventsResult.Error()
	}

	events := eventsResult.Value()
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

	result := es.snapshotService.SaveSnapshot(snapshot)
	if result.IsFailure() {
		return result.Error()
	}
	return nil
}

// LoadSnapshot loads a snapshot for an aggregate
// SOLID SRP: Delegates snapshot operations to specialized service
func (es *EventStore) LoadSnapshot(aggregateID string) (*Snapshot, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	result := es.snapshotService.LoadSnapshot(aggregateID)
	if result.IsFailure() {
		// Snapshot not found is not an error
		if result.Error() == nil {
			return nil, nil
		}
		return nil, result.Error()
	}
	return result.Value(), nil
}

// GetEventCount returns the total number of events for an aggregate
// SOLID SRP: Single responsibility for event counting
func (es *EventStore) GetEventCount(aggregateID string) (int, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	eventsResult := es.storage.LoadAllEvents(aggregateID)
	if eventsResult.IsFailure() {
		return 0, eventsResult.Error()
	}

	return len(eventsResult.Value()), nil
}

// GetLatestVersion gets the latest version number for an aggregate
func (es *EventStore) GetLatestVersion(aggregateID string) (int, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	result := es.storage.GetLatestVersion(aggregateID)
	if result.IsFailure() {
		return 0, result.Error()
	}
	return result.Value(), nil
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
