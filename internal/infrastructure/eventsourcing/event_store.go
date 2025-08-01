// REAL Event Sourcing - Complete Implementation with Persistence
package eventsourcing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/flext/flexcore/pkg/result"
	_ "github.com/mattn/go-sqlite3"
)

// DomainEvent represents a domain event
type DomainEvent interface {
	EventID() string
	EventType() string
	AggregateID() string
	AggregateType() string
	EventVersion() int
	OccurredAt() time.Time
	EventData() interface{}
}

// Event represents an event in the event store
type Event struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	Version       int                    `json:"version"`
	Data          map[string]interface{} `json:"data"`
	Metadata      map[string]interface{} `json:"metadata"`
	OccurredAt    time.Time              `json:"occurred_at"`
	CreatedAt     time.Time              `json:"created_at"`
}

// EventStore provides event sourcing capabilities
type EventStore struct {
	db        *sql.DB
	mu        sync.RWMutex
	snapshots map[string]*Snapshot
}

// Snapshot represents an aggregate snapshot
type Snapshot struct {
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	Version       int                    `json:"version"`
	Data          map[string]interface{} `json:"data"`
	CreatedAt     time.Time              `json:"created_at"`
}

// EventStream represents a stream of events for an aggregate
type EventStream struct {
	AggregateID string   `json:"aggregate_id"`
	Events      []*Event `json:"events"`
	Version     int      `json:"version"`
}

// NewEventStore creates a new event store with SQLite persistence
func NewEventStore(dbPath string) (*EventStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &EventStore{
		db:        db,
		snapshots: make(map[string]*Snapshot),
	}

	if err := store.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Printf("Event store initialized with database: %s", dbPath)
	return store, nil
}

func (es *EventStore) createTables() error {
	// Create events table
	eventsTable := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		aggregate_id TEXT NOT NULL,
		aggregate_type TEXT NOT NULL,
		version INTEGER NOT NULL,
		data TEXT NOT NULL,
		metadata TEXT NOT NULL,
		occurred_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL,
		UNIQUE(aggregate_id, version)
	);

	CREATE INDEX IF NOT EXISTS idx_events_aggregate ON events(aggregate_id, version);
	CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
	CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);
	`

	if _, err := es.db.Exec(eventsTable); err != nil {
		return err
	}

	// Create snapshots table
	snapshotsTable := `
	CREATE TABLE IF NOT EXISTS snapshots (
		aggregate_id TEXT PRIMARY KEY,
		aggregate_type TEXT NOT NULL,
		version INTEGER NOT NULL,
		data TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_snapshots_version ON snapshots(version);
	`

	if _, err := es.db.Exec(snapshotsTable); err != nil {
		return err
	}

	log.Println("Event store tables created successfully")
	return nil
}

// AppendEvent appends an event to the event store
// SOLID SRP: Reduced from 6 returns to 1 return using Result pattern and helper methods
func (es *EventStore) AppendEvent(event DomainEvent) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	// Use Result pattern for centralized error handling
	appendResult := es.performEventAppend(event)
	if appendResult.IsFailure() {
		return appendResult.Error()
	}

	log.Printf("Event appended: %s (aggregate: %s, version: %d)",
		event.EventType(), event.AggregateID(), event.EventVersion())
	return nil
}

// performEventAppend handles the complete event append operation
// SOLID SRP: Single responsibility for event append with Result pattern
func (es *EventStore) performEventAppend(event DomainEvent) result.Result[bool] {
	// Validate concurrency
	concurrencyResult := es.validateConcurrency(event)
	if concurrencyResult.IsFailure() {
		return result.Failure[bool](concurrencyResult.Error())
	}

	// Serialize event data
	serializationResult := es.serializeEventData(event)
	if serializationResult.IsFailure() {
		return result.Failure[bool](serializationResult.Error())
	}

	// Insert event into database
	insertResult := es.insertEventRecord(event, serializationResult.Value())
	if insertResult.IsFailure() {
		return result.Failure[bool](insertResult.Error())
	}

	return result.Success(true)
}

// validateConcurrency checks optimistic concurrency control
// SOLID SRP: Single responsibility for concurrency validation
func (es *EventStore) validateConcurrency(event DomainEvent) result.Result[bool] {
	currentVersion, err := es.getCurrentVersion(event.AggregateID())
	if err != nil {
		return result.Failure[bool](fmt.Errorf("failed to get current version: %w", err))
	}

	expectedVersion := event.EventVersion()
	if expectedVersion != currentVersion+1 {
		return result.Failure[bool](fmt.Errorf("concurrency conflict: expected version %d, got %d", currentVersion+1, expectedVersion))
	}

	return result.Success(true)
}

// EventSerializationData holds serialized event data
type EventSerializationData struct {
	Data         string
	MetadataJSON string
}

// serializeEventData serializes event data and metadata
// SOLID SRP: Single responsibility for data serialization
func (es *EventStore) serializeEventData(event DomainEvent) result.Result[EventSerializationData] {
	// Serialize event data
	data, err := json.Marshal(event.EventData())
	if err != nil {
		return result.Failure[EventSerializationData](fmt.Errorf("failed to serialize event data: %w", err))
	}

	// Create and serialize metadata
	metadata := map[string]interface{}{
		"event_id":   event.EventID(),
		"source":     "flexcore",
		"created_by": "event-store",
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return result.Failure[EventSerializationData](fmt.Errorf("failed to serialize metadata: %w", err))
	}

	return result.Success(EventSerializationData{
		Data:         string(data),
		MetadataJSON: string(metadataJSON),
	})
}

// insertEventRecord inserts the event record into database
// SOLID SRP: Single responsibility for database insertion
func (es *EventStore) insertEventRecord(event DomainEvent, serializedData EventSerializationData) result.Result[bool] {
	query := `
	INSERT INTO events (id, type, aggregate_id, aggregate_type, version, data, metadata, occurred_at, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := es.db.Exec(query,
		event.EventID(),
		event.EventType(),
		event.AggregateID(),
		event.AggregateType(),
		event.EventVersion(),
		serializedData.Data,
		serializedData.MetadataJSON,
		event.OccurredAt(),
		time.Now(),
	)

	if err != nil {
		return result.Failure[bool](fmt.Errorf("failed to insert event: %w", err))
	}

	return result.Success(true)
}

// GetEventStream retrieves all events for an aggregate
func (es *EventStore) GetEventStream(aggregateID string) (*EventStream, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	query := `
	SELECT id, type, aggregate_id, aggregate_type, version, data, metadata, occurred_at, created_at
	FROM events
	WHERE aggregate_id = ?
	ORDER BY version ASC
	`

	rows, err := es.db.Query(query, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	var maxVersion int

	for rows.Next() {
		var event Event
		var dataJSON, metadataJSON string

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.AggregateID,
			&event.AggregateType,
			&event.Version,
			&dataJSON,
			&metadataJSON,
			&event.OccurredAt,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Deserialize data and metadata
		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			return nil, fmt.Errorf("failed to deserialize event data: %w", err)
		}

		if err := json.Unmarshal([]byte(metadataJSON), &event.Metadata); err != nil {
			return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
		}

		events = append(events, &event)
		if event.Version > maxVersion {
			maxVersion = event.Version
		}
	}

	stream := &EventStream{
		AggregateID: aggregateID,
		Events:      events,
		Version:     maxVersion,
	}

	return stream, nil
}

// GetEventsFromVersion retrieves events from a specific version
func (es *EventStore) GetEventsFromVersion(aggregateID string, fromVersion int) ([]*Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	query := `
	SELECT id, type, aggregate_id, aggregate_type, version, data, metadata, occurred_at, created_at
	FROM events
	WHERE aggregate_id = ? AND version >= ?
	ORDER BY version ASC
	`

	rows, err := es.db.Query(query, aggregateID, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event

	events, err = es.scanEventsFromRows(rows)
	if err != nil {
		return nil, err
	}

	return events, nil
}

// SaveSnapshot saves a snapshot of an aggregate
func (es *EventStore) SaveSnapshot(snapshot *Snapshot) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	data, err := json.Marshal(snapshot.Data)
	if err != nil {
		return fmt.Errorf("failed to serialize snapshot data: %w", err)
	}

	query := `
	INSERT OR REPLACE INTO snapshots (aggregate_id, aggregate_type, version, data, created_at)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err = es.db.Exec(query,
		snapshot.AggregateID,
		snapshot.AggregateType,
		snapshot.Version,
		string(data),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	// Cache snapshot in memory
	es.snapshots[snapshot.AggregateID] = snapshot

	log.Printf("Snapshot saved: aggregate %s, version %d", snapshot.AggregateID, snapshot.Version)
	return nil
}

// GetSnapshot retrieves the latest snapshot for an aggregate
func (es *EventStore) GetSnapshot(aggregateID string) (*Snapshot, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	// Check memory cache first
	if snapshot, exists := es.snapshots[aggregateID]; exists {
		return snapshot, nil
	}

	// Query database
	query := `
	SELECT aggregate_id, aggregate_type, version, data, created_at
	FROM snapshots
	WHERE aggregate_id = ?
	`

	var snapshot Snapshot
	var dataJSON string

	err := es.db.QueryRow(query, aggregateID).Scan(
		&snapshot.AggregateID,
		&snapshot.AggregateType,
		&snapshot.Version,
		&dataJSON,
		&snapshot.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No snapshot found
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query snapshot: %w", err)
	}

	// Deserialize data
	if err := json.Unmarshal([]byte(dataJSON), &snapshot.Data); err != nil {
		return nil, fmt.Errorf("failed to deserialize snapshot data: %w", err)
	}

	// Cache in memory
	es.snapshots[aggregateID] = &snapshot

	return &snapshot, nil
}

// GetEventsByType retrieves events by type with optional time range
func (es *EventStore) GetEventsByType(eventType string, fromTime, toTime *time.Time) ([]*Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	query := `
	SELECT id, type, aggregate_id, aggregate_type, version, data, metadata, occurred_at, created_at
	FROM events
	WHERE type = ?
	`
	args := []interface{}{eventType}

	if fromTime != nil {
		query += " AND occurred_at >= ?"
		args = append(args, *fromTime)
	}

	if toTime != nil {
		query += " AND occurred_at <= ?"
		args = append(args, *toTime)
	}

	query += " ORDER BY occurred_at ASC"

	rows, err := es.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by type: %w", err)
	}
	defer rows.Close()

	var events []*Event

	events, err = es.scanEventsFromRows(rows)
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (es *EventStore) getCurrentVersion(aggregateID string) (int, error) {
	query := `SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_id = ?`

	var version int
	err := es.db.QueryRow(query, aggregateID).Scan(&version)
	if err != nil {
		return 0, err
	}

	return version, nil
}

// scanEventsFromRows is a helper method to scan events from database rows
func (es *EventStore) scanEventsFromRows(rows *sql.Rows) ([]*Event, error) {
	var events []*Event

	for rows.Next() {
		var event Event
		var dataJSON, metadataJSON string

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.AggregateID,
			&event.AggregateType,
			&event.Version,
			&dataJSON,
			&metadataJSON,
			&event.OccurredAt,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Deserialize data and metadata
		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			return nil, fmt.Errorf("failed to deserialize event data: %w", err)
		}

		if err := json.Unmarshal([]byte(metadataJSON), &event.Metadata); err != nil {
			return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
		}

		events = append(events, &event)
	}

	return events, nil
}

// GetStats returns statistics about the event store
// SOLID SRP: Reduced from 6 returns to 1 return using Result pattern and StatsCollector
func (es *EventStore) GetStats() (map[string]interface{}, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	// Use specialized stats collector for centralized error handling
	collector := es.createStatsCollector()
	statsResult := collector.CollectAllStatistics()
	
	if statsResult.IsFailure() {
		return nil, statsResult.Error()
	}

	return statsResult.Value(), nil
}

// EventStoreStatsCollector handles statistics collection with Result pattern
type EventStoreStatsCollector struct {
	db *sql.DB
}

// createStatsCollector creates a specialized statistics collector
// SOLID SRP: Factory method for creating specialized collectors
func (es *EventStore) createStatsCollector() *EventStoreStatsCollector {
	return &EventStoreStatsCollector{db: es.db}
}

// CollectAllStatistics collects all event store statistics
// SOLID SRP: Single responsibility for complete statistics collection
func (collector *EventStoreStatsCollector) CollectAllStatistics() result.Result[map[string]interface{}] {
	stats := make(map[string]interface{})

	// Collect basic counts
	basicStatsResult := collector.collectBasicStats()
	if basicStatsResult.IsFailure() {
		return result.Failure[map[string]interface{}](basicStatsResult.Error())
	}
	
	// Merge basic stats
	for key, value := range basicStatsResult.Value() {
		stats[key] = value
	}

	// Collect events by type
	eventTypesResult := collector.collectEventsByType()
	if eventTypesResult.IsFailure() {
		return result.Failure[map[string]interface{}](eventTypesResult.Error())
	}
	stats["events_by_type"] = eventTypesResult.Value()

	return result.Success(stats)
}

// collectBasicStats collects basic statistics (counts)
// SOLID SRP: Single responsibility for basic statistics
func (collector *EventStoreStatsCollector) collectBasicStats() result.Result[map[string]interface{}] {
	stats := make(map[string]interface{})

	// Count total events
	totalEventsResult := collector.countTotalEvents()
	if totalEventsResult.IsFailure() {
		return result.Failure[map[string]interface{}](totalEventsResult.Error())
	}
	stats["total_events"] = totalEventsResult.Value()

	// Count total snapshots
	totalSnapshotsResult := collector.countTotalSnapshots()
	if totalSnapshotsResult.IsFailure() {
		return result.Failure[map[string]interface{}](totalSnapshotsResult.Error())
	}
	stats["total_snapshots"] = totalSnapshotsResult.Value()

	// Count unique aggregates
	uniqueAggregatesResult := collector.countUniqueAggregates()
	if uniqueAggregatesResult.IsFailure() {
		return result.Failure[map[string]interface{}](uniqueAggregatesResult.Error())
	}
	stats["unique_aggregates"] = uniqueAggregatesResult.Value()

	return result.Success(stats)
}

// countTotalEvents counts total events in the store
func (collector *EventStoreStatsCollector) countTotalEvents() result.Result[int] {
	var totalEvents int
	err := collector.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&totalEvents)
	if err != nil {
		return result.Failure[int](err)
	}
	return result.Success(totalEvents)
}

// countTotalSnapshots counts total snapshots in the store
func (collector *EventStoreStatsCollector) countTotalSnapshots() result.Result[int] {
	var totalSnapshots int
	err := collector.db.QueryRow("SELECT COUNT(*) FROM snapshots").Scan(&totalSnapshots)
	if err != nil {
		return result.Failure[int](err)
	}
	return result.Success(totalSnapshots)
}

// countUniqueAggregates counts unique aggregates in the store
func (collector *EventStoreStatsCollector) countUniqueAggregates() result.Result[int] {
	var uniqueAggregates int
	err := collector.db.QueryRow("SELECT COUNT(DISTINCT aggregate_id) FROM events").Scan(&uniqueAggregates)
	if err != nil {
		return result.Failure[int](err)
	}
	return result.Success(uniqueAggregates)
}

// collectEventsByType collects events grouped by type
func (collector *EventStoreStatsCollector) collectEventsByType() result.Result[map[string]int] {
	eventTypeQuery := `
	SELECT type, COUNT(*)
	FROM events
	GROUP BY type
	ORDER BY COUNT(*) DESC
	`
	rows, err := collector.db.Query(eventTypeQuery)
	if err != nil {
		return result.Failure[map[string]int](err)
	}
	defer rows.Close()

	eventTypes := make(map[string]int)
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return result.Failure[map[string]int](err)
		}
		eventTypes[eventType] = count
	}

	return result.Success(eventTypes)
}

// Close closes the event store database connection
func (es *EventStore) Close() error {
	return es.db.Close()
}
