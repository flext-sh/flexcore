package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flext/flexcore/pkg/logging"
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
func (es *PostgreSQLEventStore) GetEvents(ctx context.Context, streamID string) ([]EventEntry, error) {
	var events []Event
	
	if err := es.db.WithContext(ctx).
		Where("stream_id = ?", streamID).
		Order("timestamp ASC").
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve events: %w", err)
	}
	
	// Convert to EventEntry format
	var eventEntries []EventEntry
	for _, event := range events {
		var eventData interface{}
		if err := json.Unmarshal([]byte(event.Data), &eventData); err != nil {
			es.logger.Warn("Failed to unmarshal event data",
				zap.String("event_id", event.ID),
				zap.Error(err))
			continue
		}
		
		eventEntries = append(eventEntries, EventEntry{
			ID:        event.ID,
			Type:      event.Type,
			Data:      eventData,
			Timestamp: event.Timestamp,
			Version:   event.Version,
		})
	}
	
	return eventEntries, nil
}

// GetAllEvents retrieves all events from PostgreSQL
func (es *PostgreSQLEventStore) GetAllEvents(ctx context.Context) ([]EventEntry, error) {
	var events []Event
	
	if err := es.db.WithContext(ctx).
		Order("timestamp ASC").
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve all events: %w", err)
	}
	
	// Convert to EventEntry format
	var eventEntries []EventEntry
	for _, event := range events {
		var eventData interface{}
		if err := json.Unmarshal([]byte(event.Data), &eventData); err != nil {
			es.logger.Warn("Failed to unmarshal event data",
				zap.String("event_id", event.ID),
				zap.Error(err))
			continue
		}
		
		eventEntries = append(eventEntries, EventEntry{
			ID:        event.ID,
			Type:      event.Type,
			Data:      eventData,
			Timestamp: event.Timestamp,
			Version:   event.Version,
		})
	}
	
	return eventEntries, nil
}

// GetEventsByType retrieves events by type from PostgreSQL
func (es *PostgreSQLEventStore) GetEventsByType(ctx context.Context, eventType string) ([]EventEntry, error) {
	var events []Event
	
	if err := es.db.WithContext(ctx).
		Where("type = ?", eventType).
		Order("timestamp ASC").
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve events by type: %w", err)
	}
	
	// Convert to EventEntry format
	var eventEntries []EventEntry
	for _, event := range events {
		var eventData interface{}
		if err := json.Unmarshal([]byte(event.Data), &eventData); err != nil {
			es.logger.Warn("Failed to unmarshal event data",
				zap.String("event_id", event.ID),
				zap.Error(err))
			continue
		}
		
		eventEntries = append(eventEntries, EventEntry{
			ID:        event.ID,
			Type:      event.Type,
			Data:      eventData,
			Timestamp: event.Timestamp,
			Version:   event.Version,
		})
	}
	
	return eventEntries, nil
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

// Close closes the database connection
func (es *PostgreSQLEventStore) Close() error {
	if db, err := es.db.DB(); err != nil {
		return err
	} else {
		return db.Close()
	}
}