// Package eventsourcing provides event storage functionality
// SOLID SRP: Dedicated module for event persistence operations
package eventsourcing

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// EventStorage handles all event persistence operations
// SOLID SRP: Single responsibility for event storage
type EventStorage struct {
	db *sql.DB
}

// NewEventStorage creates a new event storage instance
func NewEventStorage(dbPath string) (*EventStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	storage := &EventStorage{db: db}
	if err := storage.createEventTable(); err != nil {
		return nil, err
	}

	return storage, nil
}

// createEventTable creates the events table if it doesn't exist
func (es *EventStorage) createEventTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			aggregate_id TEXT NOT NULL,
			aggregate_type TEXT NOT NULL,
			version INTEGER NOT NULL,
			data TEXT NOT NULL,
			metadata TEXT,
			occurred_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			UNIQUE(aggregate_id, version)
		)`

	_, err := es.db.Exec(query)
	return err
}

// StoreEvent stores a single event in the database
// SOLID SRP: Single responsibility for event storage
func (es *EventStorage) StoreEvent(event DomainEvent) error {
	query := `
		INSERT INTO events 
		(id, type, aggregate_id, aggregate_type, version, data, metadata, occurred_at, created_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	dataBytes, err := json.Marshal(event.EventData())
	if err != nil {
		return err
	}

	// Default metadata
	metadata := map[string]interface{}{
		"stored_by": "event_storage",
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	_, err = es.db.Exec(query,
		event.EventID(),
		event.EventType(),
		event.AggregateID(),
		event.AggregateType(),
		event.EventVersion(),
		string(dataBytes),
		string(metadataBytes),
		event.OccurredAt(),
		time.Now(),
	)

	return err
}

// EventLoadingContext encapsulates event loading context and operations
// SOLID SRP: Single responsibility for event loading context management
type EventLoadingContext struct {
	storage     *EventStorage
	aggregateID string
	fromVersion int
	events      []Event
}

// NewEventLoadingContext creates a new event loading context
func NewEventLoadingContext(storage *EventStorage, aggregateID string, fromVersion int) *EventLoadingContext {
	return &EventLoadingContext{
		storage:     storage,
		aggregateID: aggregateID,
		fromVersion: fromVersion,
		events:      make([]Event, 0),
	}
}

// LoadEvents loads events for an aggregate from a specific version
// SOLID SRP: Single responsibility for event loading orchestration
func (es *EventStorage) LoadEvents(aggregateID string, fromVersion int) ([]Event, error) {
	context := NewEventLoadingContext(es, aggregateID, fromVersion)
	return context.executeEventLoading()
}

// executeEventLoading executes the complete event loading process
// SOLID SRP: Single responsibility for event loading execution
func (ctx *EventLoadingContext) executeEventLoading() ([]Event, error) {
	// Phase 1: Execute database query
	rows, err := ctx.executeQuery()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Phase 2: Process all rows
	err = ctx.processEventRows(rows)
	if err != nil {
		return nil, err
	}

	// Phase 3: Validate row iteration completion
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ctx.events, nil
}

// executeQuery executes the database query for event loading
// SOLID SRP: Single responsibility for database query execution
func (ctx *EventLoadingContext) executeQuery() (*sql.Rows, error) {
	query := `
		SELECT id, type, aggregate_id, aggregate_type, version, data, metadata, occurred_at, created_at 
		FROM events 
		WHERE aggregate_id = ? AND version >= ? 
		ORDER BY version ASC`

	return ctx.storage.db.Query(query, ctx.aggregateID, ctx.fromVersion)
}

// processEventRows processes all event rows from the database query
// SOLID SRP: Single responsibility for event row processing
func (ctx *EventLoadingContext) processEventRows(rows *sql.Rows) error {
	for rows.Next() {
		event, err := ctx.processSingleEventRow(rows)
		if err != nil {
			return err
		}

		ctx.events = append(ctx.events, event)
	}

	return nil
}

// processSingleEventRow processes a single event row from database
// SOLID SRP: Single responsibility for individual event processing
func (ctx *EventLoadingContext) processSingleEventRow(rows *sql.Rows) (Event, error) {
	var event Event
	var dataStr, metadataStr string

	// Scan row data
	err := rows.Scan(
		&event.ID,
		&event.Type,
		&event.AggregateID,
		&event.AggregateType,
		&event.Version,
		&dataStr,
		&metadataStr,
		&event.OccurredAt,
		&event.CreatedAt,
	)
	if err != nil {
		return Event{}, err
	}

	// Process JSON data and metadata
	return ctx.processEventJsonData(&event, dataStr, metadataStr)
}

// processEventJsonData processes JSON data and metadata for an event
// SOLID SRP: Single responsibility for JSON data processing
func (ctx *EventLoadingContext) processEventJsonData(event *Event, dataStr, metadataStr string) (Event, error) {
	// Unmarshal data
	if err := json.Unmarshal([]byte(dataStr), &event.Data); err != nil {
		return Event{}, err
	}

	// Unmarshal metadata
	if err := json.Unmarshal([]byte(metadataStr), &event.Metadata); err != nil {
		return Event{}, err
	}

	return *event, nil
}

// LoadAllEvents loads all events for an aggregate
func (es *EventStorage) LoadAllEvents(aggregateID string) ([]Event, error) {
	return es.LoadEvents(aggregateID, 0)
}

// GetLatestVersion gets the latest version number for an aggregate
// SOLID SRP: Single responsibility for version tracking
func (es *EventStorage) GetLatestVersion(aggregateID string) (int, error) {
	query := `
		SELECT COALESCE(MAX(version), 0) 
		FROM events 
		WHERE aggregate_id = ?`

	var version int
	err := es.db.QueryRow(query, aggregateID).Scan(&version)
	if err != nil {
		return 0, err
	}

	return version, nil
}

// Close closes the database connection
func (es *EventStorage) Close() error {
	return es.db.Close()
}

// GetDB returns the database connection for advanced operations
func (es *EventStorage) GetDB() *sql.DB {
	return es.db
}
