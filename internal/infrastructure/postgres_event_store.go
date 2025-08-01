package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flext/flexcore/pkg/logging"
	"github.com/flext/flexcore/pkg/result"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Event represents a database event record
type Event struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Type      string    `gorm:"not null;index"`
	Data      string    `gorm:"type:jsonb;not null"`
	StreamID  string    `gorm:"not null;index"`
	Version   int       `gorm:"not null"`
	Timestamp time.Time `gorm:"not null;default:now()"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// TableName returns the table name for GORM
func (Event) TableName() string {
	return "events"
}

// PostgreSQLEventStore implements a PostgreSQL-based event store for Event Sourcing
type PostgreSQLEventStore struct {
	db     *gorm.DB
	logger logging.LoggerInterface
}

// NewPostgreSQLEventStore creates a new PostgreSQL event store
func NewPostgreSQLEventStore(db *gorm.DB, logger logging.LoggerInterface) *PostgreSQLEventStore {
	return &PostgreSQLEventStore{
		db:     db,
		logger: logger,
	}
}

// Initialize creates the events table if it doesn't exist
func (es *PostgreSQLEventStore) Initialize() error {
	if err := es.db.AutoMigrate(&Event{}); err != nil {
		return fmt.Errorf("failed to migrate events table: %w", err)
	}

	es.logger.Info("PostgreSQL event store initialized successfully")
	return nil
}

// SaveEvent saves an event to PostgreSQL exactly as specified in the architecture document
func (es *PostgreSQLEventStore) SaveEvent(ctx context.Context, event interface{}) error {
	// Generate event ID
	eventID := uuid.New().String()

	// Determine event type from the event struct
	eventType := fmt.Sprintf("%T", event)

	// Serialize event data to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Create event record
	eventRecord := Event{
		ID:        eventID,
		Type:      eventType,
		Data:      string(eventData),
		StreamID:  eventType, // Using event type as stream for simplicity
		Version:   1,
		Timestamp: time.Now().UTC(),
	}

	// Save to database
	if err := es.db.WithContext(ctx).Create(&eventRecord).Error; err != nil {
		es.logger.Error("Failed to save event to PostgreSQL",
			zap.String("event_id", eventID),
			zap.String("event_type", eventType),
			zap.Error(err))
		return fmt.Errorf("failed to save event: %w", err)
	}

	es.logger.Debug("Event saved to PostgreSQL",
		zap.String("event_id", eventID),
		zap.String("event_type", eventType),
		zap.String("stream_id", eventRecord.StreamID))

	return nil
}

// GetEvents retrieves events from PostgreSQL by stream ID
// DRY PRINCIPLE: Uses shared query execution eliminating 32-line duplication (mass=186)
func (es *PostgreSQLEventStore) GetEvents(ctx context.Context, streamID string) ([]EventEntry, error) {
	queryBuilder := es.createEventQueryBuilder(ctx)
	queryResult := queryBuilder.WithStreamFilter(streamID).ExecuteQuery()
	
	if queryResult.IsFailure() {
		return nil, queryResult.Error()
	}

	return queryResult.Value(), nil
}

// GetAllEvents retrieves all events from PostgreSQL
// DRY PRINCIPLE: Uses shared query execution for consistency with other methods
func (es *PostgreSQLEventStore) GetAllEvents(ctx context.Context) ([]EventEntry, error) {
	queryBuilder := es.createEventQueryBuilder(ctx)
	queryResult := queryBuilder.WithoutFilters().ExecuteQuery()
	
	if queryResult.IsFailure() {
		return nil, queryResult.Error()
	}

	return queryResult.Value(), nil
}

// GetEventsByType retrieves events by type from PostgreSQL
// DRY PRINCIPLE: Uses shared query execution eliminating 32-line duplication (mass=186)
func (es *PostgreSQLEventStore) GetEventsByType(ctx context.Context, eventType string) ([]EventEntry, error) {
	queryBuilder := es.createEventQueryBuilder(ctx)
	queryResult := queryBuilder.WithTypeFilter(eventType).ExecuteQuery()
	
	if queryResult.IsFailure() {
		return nil, queryResult.Error()
	}

	return queryResult.Value(), nil
}

// GetEventCount returns the total number of events in the store
func (es *PostgreSQLEventStore) GetEventCount(ctx context.Context) (int64, error) {
	var count int64

	if err := es.db.WithContext(ctx).Model(&Event{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}

	return count, nil
}

// DeleteEventsByStream deletes all events for a specific stream
func (es *PostgreSQLEventStore) DeleteEventsByStream(ctx context.Context, streamID string) error {
	result := es.db.WithContext(ctx).Where("stream_id = ?", streamID).Delete(&Event{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete events for stream %s: %w", streamID, result.Error)
	}

	es.logger.Info("Deleted events for stream",
		zap.String("stream_id", streamID),
		zap.Int64("deleted_count", result.RowsAffected))

	return nil
}

// EventQueryBuilder handles event querying with Result pattern
// SOLID SRP: Single responsibility for event query building and execution
type EventQueryBuilder struct {
	es       *PostgreSQLEventStore
	ctx      context.Context
	query    *gorm.DB
	errorMsg string
}

// createEventQueryBuilder creates a specialized event query builder
// SOLID SRP: Factory method for creating specialized query builders
func (es *PostgreSQLEventStore) createEventQueryBuilder(ctx context.Context) *EventQueryBuilder {
	return &EventQueryBuilder{
		es:    es,
		ctx:   ctx,
		query: es.db.WithContext(ctx).Order("timestamp ASC"),
	}
}

// WithStreamFilter adds stream ID filter to the query
// SOLID SRP: Single responsibility for stream filtering
func (qb *EventQueryBuilder) WithStreamFilter(streamID string) *EventQueryBuilder {
	qb.query = qb.query.Where("stream_id = ?", streamID)
	qb.errorMsg = "failed to retrieve events"
	return qb
}

// WithTypeFilter adds event type filter to the query
// SOLID SRP: Single responsibility for type filtering
func (qb *EventQueryBuilder) WithTypeFilter(eventType string) *EventQueryBuilder {
	qb.query = qb.query.Where("type = ?", eventType)
	qb.errorMsg = "failed to retrieve events by type"
	return qb
}

// WithoutFilters configures query builder to retrieve all events
// SOLID SRP: Single responsibility for unfiltered querying
func (qb *EventQueryBuilder) WithoutFilters() *EventQueryBuilder {
	qb.errorMsg = "failed to retrieve all events"
	return qb
}

// ExecuteQuery executes the built query and converts results
// SOLID SRP: Single responsibility for complete query execution and conversion
func (qb *EventQueryBuilder) ExecuteQuery() result.Result[[]EventEntry] {
	var events []Event

	// Execute database query
	if err := qb.query.Find(&events).Error; err != nil {
		return result.Failure[[]EventEntry](fmt.Errorf("%s: %w", qb.errorMsg, err))
	}

	// Convert events using specialized converter
	converter := qb.createEventConverter()
	conversionResult := converter.ConvertEventsToEntries(events)
	
	if conversionResult.IsFailure() {
		return result.Failure[[]EventEntry](conversionResult.Error())
	}

	return result.Success(conversionResult.Value())
}

// EventConverter handles event conversion with Result pattern
// SOLID SRP: Single responsibility for event conversion
type EventConverter struct {
	logger logging.LoggerInterface
}

// createEventConverter creates a specialized event converter
// SOLID SRP: Factory method for creating specialized converters
func (qb *EventQueryBuilder) createEventConverter() *EventConverter {
	return &EventConverter{
		logger: qb.es.logger,
	}
}

// ConvertEventsToEntries converts database events to EventEntry format
// SOLID SRP: Single responsibility for complete event conversion
func (converter *EventConverter) ConvertEventsToEntries(events []Event) result.Result[[]EventEntry] {
	var eventEntries []EventEntry

	for _, event := range events {
		conversionResult := converter.convertSingleEvent(event)
		if conversionResult.IsFailure() {
			// Log warning but continue processing other events
			converter.logger.Warn("Failed to convert event",
				zap.String("event_id", event.ID),
				zap.Error(conversionResult.Error()))
			continue
		}
		
		eventEntries = append(eventEntries, conversionResult.Value())
	}

	return result.Success(eventEntries)
}

// convertSingleEvent converts a single database event to EventEntry
// SOLID SRP: Single responsibility for single event conversion
func (converter *EventConverter) convertSingleEvent(event Event) result.Result[EventEntry] {
	var eventData interface{}
	if err := json.Unmarshal([]byte(event.Data), &eventData); err != nil {
		return result.Failure[EventEntry](fmt.Errorf("failed to unmarshal event data: %w", err))
	}

	eventEntry := EventEntry{
		ID:        event.ID,
		Type:      event.Type,
		Data:      eventData,
		Timestamp: event.Timestamp,
		Version:   event.Version,
	}

	return result.Success(eventEntry)
}

// Close closes the database connection
func (es *PostgreSQLEventStore) Close() error {
	if db, err := es.db.DB(); err != nil {
		return err
	} else {
		return db.Close()
	}
}
