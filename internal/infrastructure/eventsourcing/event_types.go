// Package eventsourcing provides event sourcing types and interfaces
// SOLID SRP: Dedicated module for event types and domain interfaces
package eventsourcing

import (
	"time"
)

// DomainEvent represents a domain event
// SOLID ISP: Interface segregation principle - minimal required methods
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

// EventStream represents a stream of events for an aggregate
type EventStream struct {
	AggregateID string  `json:"aggregate_id"`
	Events      []Event `json:"events"`
	Version     int     `json:"version"`
}

// EventStoreConfig represents configuration for event store
type EventStoreConfig struct {
	DatabasePath string
	MaxRetries   int
	Timeout      time.Duration
}